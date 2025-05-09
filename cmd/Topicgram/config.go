package main

import (
	"Topicgram/config"
	"Topicgram/model"
)

type Config struct {
	Web struct {
		Type      string
		Listen    string
		Cert, Key string
	}

	Database struct {
		Type string // sqlite3, mysql, postgres, oracle

		SQLite3  *config.SQLite3
		MySQL    *config.MySQL
		Postgres *config.Postgres
		Oracle   *config.Oracle
	}

	Bot *model.BotConfig

	Security struct {
		InsecureSkipVerify bool
	}

	Proxy string
}
