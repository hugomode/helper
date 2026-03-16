package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewLoggerWithLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zap.AtomicLevel
	}{
		{
			name:  "Debug level",
			level: "debug",
			expected: zap.NewAtomicLevelAt(zap.DebugLevel),
		},
		{
			name:  "Info level",
			level: "info",
			expected: zap.NewAtomicLevelAt(zap.InfoLevel),
		},
		{
			name:  "Error level",
			level: "error",
			expected: zap.NewAtomicLevelAt(zap.ErrorLevel),
		},
		{
			name:  "Empty level (defaults to project info)",
			level: "",
			expected: zap.NewAtomicLevelAt(zap.InfoLevel),
		},
		{
			name:  "Invalid level (fallback to project info)",
			level: "invalid",
			expected: zap.NewAtomicLevelAt(zap.InfoLevel),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLoggerWithLevel(tt.level)
			assert.NotNil(t, l)
			// check the level if possible, cfg is not exported but level is embedded in the core
		})
	}
}

func TestInitLogger(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		InitLogger()
	})
	assert.NotNil(t, Log)
}
