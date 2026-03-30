package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a production-grade structured logger.
// In "development" mode it uses a human-readable format; otherwise JSON.
func New(env string) (*zap.Logger, error) {
	var cfg zap.Config

	if env == "development" || env == "debug" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	return cfg.Build()
}
