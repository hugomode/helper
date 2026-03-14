package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

type ZapGormLogger struct {
	logger *zap.Logger
}

func NewZapGormLogger(zapLogger *zap.Logger) *ZapGormLogger {
	return &ZapGormLogger{
		logger: zapLogger,
	}
}

func (l *ZapGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l // For simplicity, returning the same instance
}

func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Infow(msg, data...)
}

func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Warnw(msg, data...)
}

func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Sugar().Errorw(msg, data...)
}

func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	switch {
	case err != nil:
		l.logger.Sugar().Errorf("SQL Trace (ERROR): \n%s -> rows: %d elapsed: %s error: %v", sql, rows, elapsed, err)
	case elapsed > 500*time.Millisecond: // Si tarda más de 500ms, marcar como advertencia
		l.logger.Sugar().Warnw(fmt.Sprintf("SQL Trace (SLOW QUERY): \n%s -> rows: %d elapsed: %s", sql, rows, elapsed))
	default:
		l.logger.Sugar().Infof("SQL Trace: \n%s -> rows: %d elapsed: %s", sql, rows, elapsed)
	}
}
