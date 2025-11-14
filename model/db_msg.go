package model

import "time"

type Msg struct {
	Id int64 `gorm:"column:id; primaryKey; not null"`

	TopicId int64 `gorm:"column:topic_id; not null"`

	UserMsgId  int `gorm:"column:user_msg_id; not null"`
	TopicMsgId int `gorm:"column:topic_msg_id; not null"`

	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

func (*Msg) TableName() string {
	return "messages"
}
