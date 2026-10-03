package database

import (
	"Topicgram/config"
	"Topicgram/model"

	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB(config config.Database) error {
	dialector, err := config.Open()
	if err != nil {
		return err
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		AllowGlobalUpdate: true,
		Logger:            newLogger(),
	})
	if err != nil {
		return err
	}

	return db.AutoMigrate(model.Topic{}, model.Msg{})
}

func DB() *gorm.DB {
	return db.Unscoped()
}
