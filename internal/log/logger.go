package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger
type Logger struct {
	*zap.Logger
}

// New creates a new logger instance
func New(level string) (*Logger, error) {
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
	
	// Configure encoding
	config.Encoding = "json"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	
	return &Logger{Logger: logger}, nil
}

// NewDevelopment creates a development logger
func NewDevelopment() *Logger {
	logger, _ := zap.NewDevelopment()
	return &Logger{Logger: logger}
}

// NewProduction creates a production logger
func NewProduction() *Logger {
	logger, _ := zap.NewProduction()
	return &Logger{Logger: logger}
}

// NewNop creates a no-op logger
func NewNop() *Logger {
	return &Logger{Logger: zap.NewNop()}
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
