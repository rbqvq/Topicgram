package database

import (
	"os"
	"time"

	"gitlab.com/CoiaPrant/clog"
	"gorm.io/gorm/logger"
)

func newLogger() logger.Interface {
	logLevel := logger.Silent
	if clog.Level() == clog.LevelDebug {
		logLevel = logger.Warn
	}
	if os.Getenv("GORM_DEBUG") == "1" {
		logLevel = logger.Info
	}

	return logger.New(clog.Printer(2, "Database", clog.LevelDebug), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
		LogLevel:                  logLevel,
	})
}
