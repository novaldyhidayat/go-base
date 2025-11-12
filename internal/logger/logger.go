package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go-base/internal/config"
)

var globalLogger *zap.Logger

// SetGlobalLogger configures the global logger from config settings.
func SetGlobalLogger(cfg config.LoggingConfig) {
	if globalLogger != nil {
		return
	}

	logger := buildLogger(cfg)
	globalLogger = logger
	zap.ReplaceGlobals(logger)
}

func buildLogger(cfg config.LoggingConfig) *zap.Logger {
	var zapCfg zap.Config

	switch cfg.Mode {
	case "development":
		zapCfg = zap.NewDevelopmentConfig()
	default:
		zapCfg = zap.NewProductionConfig()
	}

	if cfg.Level != "" {
		if level, err := zapcore.ParseLevel(cfg.Level); err == nil {
			zapCfg.Level = zap.NewAtomicLevelAt(level)
		}
	}

	logger, err := zapCfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build logger: %v\n", err)
		return zap.NewNop()
	}

	return logger
}

// L returns the global logger; if none configured returns zap.NewNop.
func L() *zap.Logger {
	if globalLogger == nil {
		return zap.NewNop()
	}

	return globalLogger
}
