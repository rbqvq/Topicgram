package config

import (
	"gorm.io/gorm"
)

type Database interface {
	Open() (gorm.Dialector, error)
}
