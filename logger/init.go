package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log *zap.Logger
)

// getBaseConfig returns the common zap configuration for the project
func getBaseConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	if os.Getenv("APP_ENV") == "development" {
		cfg = zap.NewDevelopmentConfig()
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.Encoding = "console"
	return cfg
}

// InitLogger initializes the global logger based on environment configuration.
func InitLogger() {
	cfg := getBaseConfig()

	// 👇 force stacktrace threshold to Error (not Warn)
	var err error
	Log, err = cfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		panic(err)
	}

	defer Log.Sync()
}

// NewLoggerWithLevel creates a new logger instance with the project's base configuration and a specific level.
func NewLoggerWithLevel(level string) *zap.Logger {
	cfg := getBaseConfig()

	if level != "" {
		var zapLev zapcore.Level
		if err := zapLev.UnmarshalText([]byte(level)); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(zapLev)
		}
	}

	l, err := cfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		// Fallback to production if something fails
		l, _ = zap.NewProduction()
	}
	return l
}
