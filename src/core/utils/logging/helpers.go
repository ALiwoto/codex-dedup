package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func LoadLogger(options *LoggerOptions) (func(), error) {
	logDirectory := options.LogDirectory
	if logDirectory == "" {
		return nil, fmt.Errorf("log directory is empty")
	}
	if err := os.MkdirAll(filepath.Join(logDirectory, "errors"), 0o750); err != nil {
		return nil, fmt.Errorf("create error log directory: %w", err)
	}
	if options.Verbose {
		if err := os.MkdirAll(filepath.Join(logDirectory, "info"), 0o750); err != nil {
			return nil, fmt.Errorf("create info log directory: %w", err)
		}
	}

	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.EncodeTime = zapcore.RFC3339TimeEncoder
	config.EncodeLevel = zapcore.CapitalLevelEncoder
	consoleLevel := zap.InfoLevel
	if options.Debug {
		consoleLevel = zap.DebugLevel
	}

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config),
		zapcore.Lock(os.Stderr),
		consoleLevel,
	)
	errorCore := newFileCore(config, filepath.Join(
		logDirectory,
		"errors",
		fmt.Sprintf("error_%s.log", time.Now().Format("2006-01-02")),
	), zap.ErrorLevel, 10)

	cores := []zapcore.Core{consoleCore, errorCore}
	if options.Verbose {
		cores = append(cores, newFileCore(config, filepath.Join(
			logDirectory,
			"info",
			fmt.Sprintf("info_%s.log", time.Now().Format("2006-01-02")),
		), zap.InfoLevel, 25))
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1))
	appLogger = logger.Sugar()

	return func() {
		_ = logger.Sync()
		appLogger = nil
	}, nil
}

func newFileCore(config zapcore.EncoderConfig, fileName string, level zapcore.LevelEnabler, maxSize int) zapcore.Core {
	writer := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    maxSize,
		MaxBackups: 3,
		MaxAge:     10,
		LocalTime:  true,
	}

	return zapcore.NewCore(zapcore.NewJSONEncoder(config), zapcore.AddSync(writer), level)
}

func Infof(template string, args ...any) {
	appLogger.Infof(template, args...)
}

func Debugf(template string, args ...any) {
	appLogger.Debugf(template, args...)
}

func Error(args ...any) {
	appLogger.Error(args...)
}
