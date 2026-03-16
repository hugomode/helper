package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// ZapGormLogger is a custom GORM logger implementation using Uber's Zap.
type ZapGormLogger struct {
	logger *zap.Logger
}

// NewZapGormLogger creates a custom GORM logger that pipes outputs to Uber's Zap logger.
func NewZapGormLogger(zapLogger *zap.Logger) *ZapGormLogger {
	return &ZapGormLogger{
		logger: zapLogger,
	}
}

// LogMode sets the log level for GORM logger.
func (l *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l // For simplicity, return the same instance
}

// Info logs informational messages using Zap's SugaredLogger.
func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Infow(msg, data...)
}

// Warn logs warning messages using Zap's SugaredLogger.
func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Warnw(msg, data...)
}

// Error logs error messages using Zap's SugaredLogger.
func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Errorw(msg, data...)
}

// Trace logs SQL query details, timing, and errors.
func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	switch {
	case err != nil:
		l.logger.Sugar().Errorf("SQL Trace (ERROR): \n%s -> rows: %d elapsed: %s error: %v", sql, rows, elapsed, err)
	case elapsed > 500*time.Millisecond: // If it takes more than 500ms, mark as warning
		l.logger.Sugar().Warnw(fmt.Sprintf("SQL Trace (SLOW QUERY): \n%s -> rows: %d elapsed: %s", sql, rows, elapsed))
	default:
		l.logger.Sugar().Infof("SQL Trace: \n%s -> rows: %d elapsed: %s", sql, rows, elapsed)
	}
}
