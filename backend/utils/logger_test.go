package utils

import (
	"os"
	"testing"

	"go.uber.org/zap/zapcore"
)

// TestParseLogLevel memverifikasi fungsi parseLogLevel untuk berbagai nilai input.
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected zapcore.Level
	}{
		{"level debug", "debug", zapcore.DebugLevel},
		{"level info", "info", zapcore.InfoLevel},
		{"level warn", "warn", zapcore.WarnLevel},
		{"level error", "error", zapcore.ErrorLevel},
		{"level dpanic", "dpanic", zapcore.DPanicLevel},
		{"level panic", "panic", zapcore.PanicLevel},
		{"level fatal", "fatal", zapcore.FatalLevel},
		{"case insensitive", "INFO", zapcore.InfoLevel},
		{"mixed case", "DeBuG", zapcore.DebugLevel},
		{"empty string defaults to info", "", zapcore.InfoLevel},
		{"invalid value defaults to info", "trace", zapcore.InfoLevel},
		{"whitespace defaults to info", "   ", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestInitLogger memastikan InitLogger berhasil dijalankan dan Log tidak nil.
// Rotasi file oleh lumberjack tidak diuji di sini karena bersifat integration-level.
func TestInitLogger(t *testing.T) {
	// Setel level log ke info agar konsisten (opsional)
	os.Setenv("LOG_LEVEL", "info")
	defer os.Unsetenv("LOG_LEVEL")

	InitLogger()

	if Log == nil {
		t.Fatal("InitLogger() menghasilkan Log yang nil")
	}
}
