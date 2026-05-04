package database

import (
	"Topicgram/config"
	"Topicgram/model"

	"gorm.io/gorm"
)

var DB func() *gorm.DB

func InitDB(config config.Database) error {
	dialector, err := config.Open()
	if err != nil {
		return err
	}

	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(model.Topic{}, model.Msg{})
	if err != nil {
		return err
	}

	DB = db.Unscoped
	return nil
}
