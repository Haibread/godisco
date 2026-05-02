package logging

import "go.uber.org/zap"

// InitLogger creates a new development sugared logger and returns it
// alongside a sync function. The caller is responsible for deferring
// the sync function so buffered log entries are flushed at shutdown.
func InitLogger() (*zap.SugaredLogger, func() error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	sugar := logger.Sugar()
	return sugar, logger.Sync
}
