package model

import (
	"database/sql/driver"

	"gorm.io/gorm/schema"
)

type Verification uint8

const (
	VerificationNotSent Verification = iota
	VerificationNotCompleted
	VerificationCompleted
)

func (Verification) GormDataType() string {
	return string(schema.Uint)
}

func (p Verification) Value() (driver.Value, error) {
	return int64(p), nil
}
