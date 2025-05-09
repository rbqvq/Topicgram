package config

import (
	"fmt"
	"net/url"

	sqlite "gitlab.com/CoiaPrant/gorm-sqlite"
	"gorm.io/gorm"
)

type SQLite3 struct {
	File string

	BusyTimeout uint64
	JournalMode string
}

func (c *SQLite3) Open() (gorm.Dialector, error) {
	configs := make(url.Values)

	if c.BusyTimeout <= 0 {
		c.BusyTimeout = 5000
	}
	configs.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", c.BusyTimeout))

	if c.JournalMode == "" {
		c.JournalMode = "WAL"
	}
	configs.Add("_pragma", fmt.Sprintf("journal_mode(%s)", c.JournalMode))

	dsn := c.File
	if args := configs.Encode(); args != "" {
		dsn += "?" + args
	}

	return sqlite.Open(dsn), nil
}
