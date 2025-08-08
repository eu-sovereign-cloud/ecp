package logger

import (
	"os"

	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func init() {
	logger := zap.Must(zap.NewProduction())
	if os.Getenv("APP_ENV") == "development" {
		logger = zap.Must(zap.NewDevelopment())
	}

	Log = logger.Sugar()
}
