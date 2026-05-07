package logger

import "go.uber.org/zap"

func New() *zap.Logger {
	log, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	return log
}
