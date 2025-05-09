package model

type Topic struct {
	Id int64 `gorm:"column:id; primaryKey; not null"`

	UserId  int64 `gorm:"column:user_id; not null"`
	TopicId int   `gorm:"column:topic_id; not null"`

	IsBan        bool   `gorm:"column:is_ban; not null"`
	LanguageCode string `gorm:"column:language_code; not null"`
}

func (*Topic) TableName() string {
	return "topics"
}
