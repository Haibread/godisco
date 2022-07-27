package logging

import "go.uber.org/zap"

/* var (
	log *zap.SugaredLogger
) */

func InitLogger() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	log := logger.Sugar()
	return log
}

/* func GetLogger() *zap.SugaredLogger {
	return log
} */
