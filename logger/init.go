package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log *zap.Logger
)

func InitLogger() {
	// Configure Zap logger to output JSON format
	cfg := zap.NewProductionConfig()
	if os.Getenv("APP_ENV") == "development" {
		cfg = zap.NewDevelopmentConfig()
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.Encoding = "console"
	// 👇 fuerza el umbral del stacktrace a Error (no en Warn)
	l, err := cfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		panic(err)
	}
	Log = l

	if os.Getenv("APP_ENV") == "development" && os.Getenv("PRETTY") == "true" {
		tmpFile, err := os.Create("/tmp/log.tmp")
		if err != nil {
			panic(fmt.Sprintf("Failed to create temporary log file: %v", err))
		}

		fmt.Printf("Logging to temporary file: %s\n", tmpFile.Name())

		encoder := zapcore.NewJSONEncoder(cfg.EncoderConfig)
		consoleSyncer := zapcore.AddSync(os.Stdout)
		fileSyncer := zapcore.AddSync(tmpFile)
		multiSyncer := zapcore.NewMultiWriteSyncer(consoleSyncer, fileSyncer)
		newCore := zapcore.NewCore(encoder, multiSyncer, zapcore.InfoLevel)
		Log = zap.New(newCore)
	}

	defer Log.Sync()
}
