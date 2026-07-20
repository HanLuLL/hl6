package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level constants for log levels.
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// SensitiveFieldKeywords contains keywords that indicate sensitive fields requiring masking.
var SensitiveFieldKeywords = []string{
	"password", "passwd", "pwd", "secret", "token", "api_key", "apikey",
	"credential", "private_key", "privatekey", "auth", "key", "session",
}

// MaskConfig defines masking rules for sensitive fields.
type MaskConfig struct {
	// FullMask fields are replaced with "***".
	FullMask []string
	// PartialMask fields show partial content (e.g., "abc***xyz").
	PartialMask []string
	// PartialMaskLen determines how many characters to show before/after.
	PartialMaskLen int
}

// DefaultMaskConfig provides sensible defaults.
var DefaultMaskConfig = MaskConfig{
	FullMask: []string{
		"password", "passwd", "pwd", "secret", "credential",
		"private_key", "privatekey", "smtp_password", "session_secret",
	},
	PartialMask: []string{
		"token", "api_key", "apikey", "key", "authorization",
	},
	PartialMaskLen: 4,
}

// Config holds logger configuration.
type Config struct {
	// Level is the minimum log level to output.
	Level slog.Level
	// EnableFile enables file output.
	EnableFile bool
	// FilePath is the path to the log file.
	FilePath string
	// FileMaxSize is the maximum size in MB before rotation.
	FileMaxSize int64
	// FileMaxAge is the maximum age in days to keep log files.
	FileMaxAge int
	// MaskConfig defines masking rules for sensitive fields.
	MaskConfig MaskConfig
	// EnableConsole enables console output (Docker logs).
	EnableConsole bool
}

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	config Config
	mu     sync.Mutex
	file   *os.File
	writer io.Writer
}

// SystemLogEntry represents a structured log entry for database storage.
type SystemLogEntry struct {
	Level     string                 `json:"level"`
	Module    string                 `json:"module"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the default logger.
func Init(cfg Config) error {
	var err error
	once.Do(func() {
		defaultLogger, err = New(cfg)
	})
	return err
}

// New creates a new Logger with the given configuration.
func New(cfg Config) (*Logger, error) {
	if cfg.FileMaxSize <= 0 {
		cfg.FileMaxSize = 100 // 100 MB default
	}
	if cfg.FileMaxAge <= 0 {
		cfg.FileMaxAge = 30 // 30 days default
	}
	if cfg.MaskConfig.PartialMaskLen <= 0 {
		cfg.MaskConfig.PartialMaskLen = DefaultMaskConfig.PartialMaskLen
	}

	l := &Logger{
		config: cfg,
	}

	// Create writers
	var writers []io.Writer
	if cfg.EnableConsole {
		writers = append(writers, os.Stdout)
	}
	if cfg.EnableFile && cfg.FilePath != "" {
		if err := l.openFile(); err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writers = append(writers, l.file)
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	l.writer = io.MultiWriter(writers...)

	// Create handler with custom formatter
	handler := slog.NewJSONHandler(l.writer, &slog.HandlerOptions{
		Level: cfg.Level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Apply masking to sensitive fields
			return l.maskAttribute(a)
		},
	})

	l.Logger = slog.New(handler)
	slog.SetDefault(l.Logger)

	return l, nil
}

// openFile opens or creates the log file.
func (l *Logger) openFile() error {
	dir := filepath.Dir(l.config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(l.config.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	l.file = f
	return nil
}

// Close closes the log file if opened.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// maskAttribute applies masking rules to a log attribute.
func (l *Logger) maskAttribute(a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)

	// Check for full mask
	for _, field := range l.config.MaskConfig.FullMask {
		if strings.Contains(key, strings.ToLower(field)) {
			return slog.String(a.Key, "***")
		}
	}

	// Check for partial mask
	for _, field := range l.config.MaskConfig.PartialMask {
		if strings.Contains(key, strings.ToLower(field)) {
			strVal := fmt.Sprintf("%v", a.Value.Any())
			return slog.String(a.Key, l.partialMask(strVal))
		}
	}

	return a
}

// partialMask masks a string showing only first and last N characters.
func (l *Logger) partialMask(s string) string {
	n := l.config.MaskConfig.PartialMaskLen
	if len(s) <= n*2 {
		return "***"
	}
	return s[:n] + "***" + s[len(s)-n:]
}

// MaskEmail partially masks an email address.
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 1 {
		return "***@" + domain
	}
	return string(local[0]) + "***@" + domain
}

// MaskToken masks a token showing only first and last 4 characters.
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}

// MaskAPIKey masks an API key showing only first and last 4 characters.
func MaskAPIKey(key string) string {
	return MaskToken(key)
}

// Rotate rotates the log file if it exceeds the maximum size.
func (l *Logger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil
	}

	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	// Check if rotation needed (convert MB to bytes)
	maxBytes := l.config.FileMaxSize * 1024 * 1024
	if info.Size() < maxBytes {
		return nil
	}

	// Close current file
	if err := l.file.Close(); err != nil {
		return err
	}

	// Rename to timestamped backup
	backupPath := l.config.FilePath + "." + time.Now().Format("20060102-150405")
	if err := os.Rename(l.config.FilePath, backupPath); err != nil {
		return err
	}

	// Open new file
	return l.openFile()
}

// CleanupOldFiles removes log files older than MaxAge days.
func (l *Logger) CleanupOldFiles() error {
	if l.config.FilePath == "" || l.config.FileMaxAge <= 0 {
		return nil
	}

	dir := filepath.Dir(l.config.FilePath)
	base := filepath.Base(l.config.FilePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -l.config.FileMaxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check if it's a rotated log file
		if !strings.HasPrefix(name, base+".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}

// --- Module-level convenience functions ---

// Debug logs a message at debug level.
func Debug(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, args...)
	}
}

// Info logs a message at info level.
func Info(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, args...)
	}
}

// Warn logs a message at warn level.
func Warn(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, args...)
	}
}

// Error logs a message at error level.
func Error(msg string, args ...any) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, args...)
	}
}

// WithModule returns a logger with a module field.
func WithModule(module string) *slog.Logger {
	if defaultLogger == nil {
		return slog.Default()
	}
	return defaultLogger.With(slog.String("module", module))
}

// LogWithContext logs a message with context information.
func LogWithContext(ctx context.Context, level slog.Level, module, msg string, fields ...any) {
	if defaultLogger == nil {
		return
	}
	logger := defaultLogger.With(slog.String("module", module))
	logger.Log(ctx, level, msg, fields...)
}

// RecordDatabase records a log entry to the database via the provided callback.
func RecordDatabase(ctx context.Context, recordFunc func(entry SystemLogEntry) error, level, module, message string, fields map[string]interface{}) error {
	entry := SystemLogEntry{
		Level:     level,
		Module:    module,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
	}
	return recordFunc(entry)
}

// ToJSON converts a log entry to JSON bytes.
func (e *SystemLogEntry) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON parses a log entry from JSON bytes.
func FromJSON(data []byte) (*SystemLogEntry, error) {
	var entry SystemLogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetDefault returns the default logger instance.
func GetDefault() *Logger {
	return defaultLogger
}