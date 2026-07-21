package logger

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
)

// GinSinkMiddleware 返回一个 Gin middleware，把 access log 写入 DBSink。
// module 标为 "http"，fields 包含 method、path、status、latency_ms、ip_hash、user_agent。
// 敏感信息（Authorization 头、JWT、原始 IP）不入库。
func GinSinkMiddleware(sink *DBSink) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// 按状态码和延迟判定 level
		level := model.LogLevelInfo
		if status >= 500 {
			level = model.LogLevelError
		} else if status >= 400 {
			level = model.LogLevelWarn
		} else if latency < 500*time.Millisecond {
			// 2xx 且快请求降级为 DEBUG，减少噪音
			level = model.LogLevelDebug
		}

		// 跳过健康检查和静态资源，减少噪音
		if path == "/health" || path == "/favicon.ico" || strings.HasPrefix(path, "/assets/") {
			return
		}

		displayPath := path
		if raw != "" {
			displayPath = path + "?" + raw
		}
		displayPath = maskSensitive(displayPath)
		if len(displayPath) > 512 {
			displayPath = displayPath[:512]
		}

		fields := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       displayPath,
			"status":     status,
			"latency_ms": latency.Milliseconds(),
			"ip_hash":    hashIP(c.ClientIP()),
			"user_agent": truncateUA(c.Request.UserAgent()),
		}
		if uid, exists := c.Get("user_id"); exists {
			fields["user_id"] = uid
		}

		sink.LogEntry(SinkEntry{
			Level:   level,
			Module:  "http",
			Message: c.Request.Method + " " + displayPath + " " + http.StatusText(status),
			Fields:  fields,
		})
	}
}

// hashIP 对 IP 做简单 hash（与 AuthSecurityEvent.IPHash 风格一致，避免存原始 IP）。
func hashIP(ip string) string {
	if ip == "" {
		return ""
	}
	return simpleHash(ip)
}

// simpleHash 使用 FNV-1a 64 位，输出 16 位十六进制字符串。
func simpleHash(s string) string {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	const hex = "0123456789abcdef"
	out := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		out[i] = hex[h&0xf]
		h >>= 4
	}
	return string(out)
}

func truncateUA(ua string) string {
	if len(ua) > 256 {
		return ua[:256]
	}
	return ua
}
