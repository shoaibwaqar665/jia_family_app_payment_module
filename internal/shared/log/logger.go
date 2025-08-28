package log

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Context keys for request-scoped fields
type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	RequestIDKey contextKey = "request_id"
	TraceIDKey   contextKey = "trace_id"
)

var (
	// globalLogger is the default logger instance
	globalLogger *zap.Logger
)

// Logger wraps zap logger
type Logger struct {
	*zap.Logger
}

// Init initializes the global logger with the specified level
func Init(level string) error {
	logger, err := NewProduction(level)
	if err != nil {
		return err
	}
	globalLogger = logger.Logger
	return nil
}

// NewProduction creates a production logger with the specified level
func NewProduction(level string) (*Logger, error) {
	config := zap.NewProductionConfig()

	// Parse log level
	logLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		logLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(logLevel)

	// Configure output
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Configure encoding for production
	config.Encoding = "json"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.NameKey = "logger"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Enable caller information for better debugging
	config.DisableCaller = false
	config.DisableStacktrace = false

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{Logger: logger}, nil
}

// NewDevelopment creates a development logger
func NewDevelopment() *Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := config.Build()
	return &Logger{Logger: logger}
}

// NewNop creates a no-op logger
func NewNop() *Logger {
	return &Logger{Logger: zap.NewNop()}
}

// L returns a logger with request-scoped fields from context
func L(ctx context.Context) *zap.Logger {
	if globalLogger == nil {
		// Fallback to a basic production logger if not initialized
		logger, _ := zap.NewProduction()
		globalLogger = logger
	}

	logger := globalLogger

	// Extract request-scoped fields from context
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		logger = logger.With(zap.String("user_id", userID))
	}

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		logger = logger.With(zap.String("request_id", requestID))
	}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		logger = logger.With(zap.String("trace_id", traceID))
	}

	return logger
}

// WithUserID adds user_id to the context for logging
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithRequestID adds request_id to the context for logging
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTraceID adds trace_id to the context for logging
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithFields adds fields to the logger
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.Logger.With(fields...)}
}

// WithError adds an error field to the logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{Logger: l.Logger.With(zap.Error(err))}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Global logger convenience functions that use the context-aware logger

// Info logs an info message with context
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Info(msg, fields...)
}

// Error logs an error message with context
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Error(msg, fields...)
}

// Warn logs a warning message with context
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Warn(msg, fields...)
}

// Debug logs a debug message with context
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	L(ctx).Debug(msg, fields...)
}
