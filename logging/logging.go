package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
)

// InitLogger creates a new sugared logger and returns it alongside a sync
// function. The caller is responsible for deferring the sync function so
// buffered log entries are flushed at shutdown.
//
// The logger is built using zap's production config by default, which emits
// JSON-formatted logs suitable for log aggregators. Setting GODISCO_LOG_MODE
// to "development" (or "dev") switches to the human-readable development
// config with stacktraces.
func InitLogger() (*zap.SugaredLogger, func() error) {
	logger, err := buildLogger(os.Getenv("GODISCO_LOG_MODE"))
	if err != nil {
		panic(err)
	}
	sugar := logger.Sugar()
	return sugar, logger.Sync
}

func buildLogger(mode string) (*zap.Logger, error) {
	switch strings.ToLower(mode) {
	case "development", "dev":
		return zap.NewDevelopment()
	default:
		return zap.NewProduction()
	}
}
