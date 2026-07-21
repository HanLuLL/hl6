// Package logger - DBSink 部分
//
// DBSink 把标准库 log 包输出和 Gin access log 多路写入到 SystemLog 表。
// 注意：本文件与 logger.go（slog 实现）共存，类型命名刻意避开 Config/Logger/Entry
// 以免与现有定义冲突。
//
// 设计要点：
//   - 实现 io.Writer 接口，解析标准库 log 包的输出（如 "2026/07/21 12:34:56 [email] msg..."）
//   - 异步 channel + 批量 flush，避免每条 log 同步 INSERT 拖慢请求
//   - DB 写入失败时静默丢弃（日志不能反向拖崩主流程）
//   - 敏感信息脱敏：邮箱、token、密码字段值入库前正则脱敏
//   - 配合 StartRetentionLoop 后台 goroutine 定期清理过期日志
package logger

import (
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/model"
)

// SinkEntry 是一条待入库的日志记录。
type SinkEntry struct {
	Level   string                 // DEBUG / INFO / WARN / ERROR
	Module  string                 // auth / dns / email / http / admin / migration 等
	Message string                 // 已脱敏的消息文本
	Fields  map[string]interface{} // 可选的结构化字段（已脱敏）
}

// SinkConfig 控制 DBSink 的行为。
type SinkConfig struct {
	BufferSize    int           // 内部 channel 缓冲长度，默认 1024
	FlushInterval time.Duration // 批量 flush 间隔，默认 100ms
	BatchSize     int           // 单次 INSERT 最大条数，默认 64
}

// DBSink 把标准库 log 输出异步入库到 SystemLog 表。
// 同时实现 io.Writer（给 log.SetOutput 用）和 Gin access log 钩子（给 middleware 用）。
type DBSink struct {
	db     *gorm.DB
	cfg    SinkConfig
	ch     chan SinkEntry
	wg     sync.WaitGroup
	closed chan struct{}
}

// NewDBSink 构造一个新的 DBSink 并启动后台 flush goroutine。
// 调用方应在退出时调用 Close() 确保剩余日志落库。
func NewDBSink(db *gorm.DB, cfg SinkConfig) *DBSink {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 100 * time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 64
	}
	w := &DBSink{
		db:     db,
		cfg:    cfg,
		ch:     make(chan SinkEntry, cfg.BufferSize),
		closed: make(chan struct{}),
	}
	w.wg.Add(1)
	go w.flushLoop()
	return w
}

// Write 实现 io.Writer。标准库 log 包每次调用会写入一行（以 \n 结尾）。
// 多行合并的情况按行拆分独立入库。
func (w *DBSink) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\n")
	if text == "" {
		return len(p), nil
	}
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		entry := parseStdLogLine(line)
		w.enqueue(entry)
	}
	return len(p), nil
}

// LogEntry 是给业务代码直接调用入库的接口（不经过 log 包）。
// 例如 Gin access log middleware 可以构造结构化 SinkEntry 后调用此方法。
func (w *DBSink) LogEntry(entry SinkEntry) {
	if entry.Message == "" {
		return
	}
	if entry.Level == "" {
		entry.Level = model.LogLevelInfo
	}
	if entry.Module == "" {
		entry.Module = "system"
	}
	entry.Message = maskSensitive(entry.Message)
	entry.Fields = maskFields(entry.Fields)
	w.enqueue(entry)
}

func (w *DBSink) enqueue(entry SinkEntry) {
	select {
	case w.ch <- entry:
	default:
		// channel 满时丢弃，避免反压拖崩调用方
	}
}

// flushLoop 后台批量 flush channel 中的日志到 DB。
func (w *DBSink) flushLoop() {
	defer w.wg.Done()

	batch := make([]SinkEntry, 0, w.cfg.BatchSize)
	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		w.insertBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case entry, ok := <-w.ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, entry)
			if len(batch) >= w.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// insertBatch 把一批 SinkEntry 转成 SystemLog 并批量 INSERT。
// 失败时静默丢弃，不能让日志写入失败反向影响主业务。
func (w *DBSink) insertBatch(batch []SinkEntry) {
	logs := make([]model.SystemLog, 0, len(batch))
	now := time.Now().UTC()
	for _, e := range batch {
		var fieldsJSON json.RawMessage
		if len(e.Fields) > 0 {
			if b, err := json.Marshal(e.Fields); err == nil {
				fieldsJSON = b
			}
		}
		logs = append(logs, model.SystemLog{
			Level:     e.Level,
			Module:    e.Module,
			Message:   truncate(e.Message, 8000),
			Fields:    fieldsJSON,
			CreatedAt: now,
		})
	}
	if len(logs) == 0 {
		return
	}
	// 失败时不重试，避免日志堆积反向拖崩系统
	_ = w.db.Create(&logs).Error
}

// Close 关闭 sink，等待后台 goroutine 把剩余日志落库。
// 应在 server 优雅关闭流程中调用。
func (w *DBSink) Close() {
	close(w.ch)
	w.wg.Wait()
}

// parseStdLogLine 解析标准库 log 包的输出行。
// 标准库格式："2026/07/21 12:34:56 message" 或带前缀 "2026/07/21 12:34:56 [email] message"
// 通过前缀 [xxx] 推断 module 和 level。
func parseStdLogLine(line string) SinkEntry {
	entry := SinkEntry{Level: model.LogLevelInfo, Module: "system"}

	// 跳过日期时间前缀 "2006/01/02 15:04:05 "（19 字符）
	rest := line
	if len(line) >= 19 && line[4] == '/' && line[7] == '/' && line[10] == ' ' && line[13] == ':' {
		rest = strings.TrimSpace(line[19:])
	}

	// 识别 [LEVEL] 和 [module] 前缀
	for {
		if !strings.HasPrefix(rest, "[") {
			break
		}
		end := strings.Index(rest, "]")
		if end <= 0 {
			break
		}
		tag := rest[1:end]
		rest = strings.TrimSpace(rest[end+1:])

		switch strings.ToUpper(tag) {
		case "DEBUG":
			entry.Level = model.LogLevelDebug
		case "INFO":
			entry.Level = model.LogLevelInfo
		case "WARN", "WARNING":
			entry.Level = model.LogLevelWarn
		case "ERROR", "ERR", "FATAL", "PANIC":
			entry.Level = model.LogLevelError
		default:
			// 非标准 level，按 module 处理
			entry.Module = strings.ToLower(tag)
		}
	}

	// 如果 message 本身包含 WARN/ERROR 等关键词，升级 level
	upper := strings.ToUpper(rest)
	if entry.Level == model.LogLevelInfo {
		if strings.Contains(upper, "WARN") {
			entry.Level = model.LogLevelWarn
		} else if strings.Contains(upper, "ERROR") || strings.Contains(upper, "FAILED") || strings.Contains(upper, "FAIL") {
			entry.Level = model.LogLevelError
		}
	}

	entry.Message = maskSensitive(rest)
	return entry
}

// StartRetentionLoop 启动后台 goroutine 定期清理超过 maxAge 的日志。
// 建议每天清理一次，每次最多删除 10000 条避免长事务锁表。
// 在 ctx.Done 时优雅退出。
func StartRetentionLoop(ctx context.Context, db *gorm.DB, maxAge time.Duration) {
	if maxAge <= 0 {
		maxAge = 30 * 24 * time.Hour
	}
	interval := 24 * time.Hour
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupOldLogs(db, maxAge)
			}
		}
	}()
}

func cleanupOldLogs(db *gorm.DB, maxAge time.Duration) {
	cutoff := time.Now().UTC().Add(-maxAge)
	const batchSize = 10000
	for {
		result := db.Where("created_at < ?", cutoff).
			Limit(batchSize).
			Delete(&model.SystemLog{})
		if result.Error != nil {
			return
		}
		if result.RowsAffected < batchSize {
			return
		}
	}
}

// truncate 截断字符串到最大长度，避免超长日志撑爆 text 列。
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// ============ 敏感信息脱敏 ============

var (
	// 邮箱地址：foo@bar.com -> f***@bar.com
	emailRegexp = regexp.MustCompile(`([a-zA-Z0-9._%+\-])[a-zA-Z0-9._%+\-]*@([a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})`)
	// 长十六进制 token（32+ 字符）：替换为 [REDACTED]
	tokenRegexp = regexp.MustCompile(`\b[a-fA-F0-9]{32,}\b`)
	// Bearer token
	bearerRegexp = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._\-]+`)
	// JWT（三段式）
	jwtRegexp = regexp.MustCompile(`\beyJ[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+\b`)
	// password=xxx / password":"xxx" 等
	passwordRegexp = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|api_key|apikey|token)["'\s:=]+["']?[^"'\s,}]+`)
)

// maskSensitive 对日志文本进行敏感信息脱敏。
func maskSensitive(s string) string {
	s = bearerRegexp.ReplaceAllString(s, "Bearer [REDACTED]")
	s = jwtRegexp.ReplaceAllString(s, "[REDACTED_JWT]")
	s = passwordRegexp.ReplaceAllStringFunc(s, func(m string) string {
		idx := -1
		for i := 0; i < len(m); i++ {
			ch := m[i]
			if ch == '=' || ch == ':' {
				idx = i
				break
			}
		}
		if idx < 0 {
			return "[REDACTED]"
		}
		j := idx + 1
		for j < len(m) && (m[j] == ' ' || m[j] == '\t' || m[j] == '"' || m[j] == '\'') {
			j++
		}
		return m[:j] + "[REDACTED]"
	})
	s = tokenRegexp.ReplaceAllString(s, "[REDACTED_TOKEN]")
	s = emailRegexp.ReplaceAllString(s, "$1***@$2")
	return s
}

// maskFields 对结构化字段做脱敏（递归处理 map）。
func maskFields(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		return nil
	}
	out := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		out[k] = maskFieldValue(k, v)
	}
	return out
}

func maskFieldValue(key string, value interface{}) interface{} {
	keyLower := strings.ToLower(key)
	if strings.Contains(keyLower, "password") || strings.Contains(keyLower, "passwd") ||
		strings.Contains(keyLower, "pwd") || strings.Contains(keyLower, "secret") ||
		strings.Contains(keyLower, "token") || strings.Contains(keyLower, "api_key") ||
		strings.Contains(keyLower, "apikey") || strings.Contains(keyLower, "authorization") {
		return "[REDACTED]"
	}
	switch v := value.(type) {
	case string:
		return maskSensitive(v)
	case map[string]interface{}:
		return maskFields(v)
	default:
		return v
	}
}

// 确保 DBSink 实现了 io.Writer 接口
var _ io.Writer = (*DBSink)(nil)
