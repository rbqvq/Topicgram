package model

type Topic struct {
	Id int64 `gorm:"column:id; primaryKey; not null"`

	UserId  int64 `gorm:"column:user_id; not null"`
	TopicId int   `gorm:"column:topic_id; not null"`

	Verification  Verification `gorm:"column:verification; not null; default: 0"`
	ChallangeId   uint64       `gorm:"column:challange_id; not null; default: 0"`
	ChallangeSent int64        `gorm:"column:challange_sent; not null; default: 0"`

	IsBan        bool   `gorm:"column:is_ban; not null"`
	LanguageCode string `gorm:"column:language_code; not null"`
}

func (*Topic) TableName() string {
	return "topics"
}
