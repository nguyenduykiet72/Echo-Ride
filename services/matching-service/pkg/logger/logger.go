package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(mode string) *zap.Logger {
	var config zap.Config

	if mode == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	config.OutputPaths = []string{"stdout"}

	logger, err := config.Build()
	if err != nil {
		os.Exit(1)
	}

	zap.ReplaceGlobals(logger)

	return logger
}
