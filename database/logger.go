package database

import (
	"time"

	"gitlab.com/CoiaPrant/clog"
	gorm_logger "gorm.io/gorm/logger"
)

var logger = gorm_logger.New(clog.Standard("Database", clog.LevelDebug), gorm_logger.Config{
	SlowThreshold:             200 * time.Millisecond,
	LogLevel:                  gorm_logger.Warn,
	IgnoreRecordNotFoundError: true,
})
