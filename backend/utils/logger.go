package utils

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Log adalah instance global Zap SugaredLogger dengan fallback Nop logger untuk pengujian
var Log *zap.SugaredLogger = zap.NewNop().Sugar()

// parseLogLevel mengubah nilai string level log (dari env LOG_LEVEL) menjadi
// zapcore.Level. Nilai yang valid: debug, info, warn, error, dpanic, panic, fatal.
// Bersifat case-insensitive. Jika nilai kosong atau tidak dikenal, kembalikan InfoLevel.
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// InitLogger menginisialisasi logger dengan output ganda (Console & File).
// Level log ditentukan dari env LOG_LEVEL (default: info) dan diterapkan
// secara konsisten pada kedua core (file & console). File log ditulis dengan
// rotasi otomatis menggunakan lumberjack.
func InitLogger() {
	// Buat direktori logs jika belum ada
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("Gagal membuat direktori logs: " + err.Error())
	}

	// Baca level log dari env LOG_LEVEL, default ke info
	logLevel := parseLogLevel(os.Getenv("LOG_LEVEL"))

	// Konfigurasi lumberjack untuk rotasi file log:
	// - MaxSize 100 MB per file
	// - MaxBackups 5 file lama
	// - MaxAge 30 hari
	// - Compress true (gzip file lama)
	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "server.log"),
		MaxSize:    100,  // megabytes per file
		MaxBackups: 5,    // simpan 5 file lama
		MaxAge:     30,   // hari
		Compress:   true, // gzip file lama
	}

	// 1. Encoder Config untuk File (JSON format, efisien dan permanen)
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	// 2. Encoder Config untuk Console (Console/Text format, berwarna, mudah dibaca)
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	// Buat cores. Kedua core menggunakan level log yang sama (dari LOG_LEVEL)
	// agar filtering konsisten antara output console dan file.
	fileWriteSyncer := zapcore.AddSync(logFile)
	consoleWriteSyncer := zapcore.AddSync(os.Stdout)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileWriteSyncer, logLevel),
		zapcore.NewCore(consoleEncoder, consoleWriteSyncer, logLevel),
	)

	// Inisialisasi Logger
	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)

	Log = logger.Sugar()
}
