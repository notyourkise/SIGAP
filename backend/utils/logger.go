package utils

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log adalah instance global Zap SugaredLogger dengan fallback Nop logger untuk pengujian
var Log *zap.SugaredLogger = zap.NewNop().Sugar()

// InitLogger menginisialisasi logger dengan output ganda (Console & File)
func InitLogger() {
	// Buat direktori logs jika belum ada
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("Gagal membuat direktori logs: " + err.Error())
	}

	logFilePath := filepath.Join(logDir, "server.log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("Gagal membuka file log: " + err.Error())
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

	// Buat cores
	fileWriteSyncer := zapcore.AddSync(logFile)
	consoleWriteSyncer := zapcore.AddSync(os.Stdout)

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileWriteSyncer, zap.InfoLevel),
		zapcore.NewCore(consoleEncoder, consoleWriteSyncer, zap.DebugLevel),
	)

	// Inisialisasi Logger
	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)

	Log = logger.Sugar()
}
