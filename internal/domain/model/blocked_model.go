package model

import "time"

type TgUserModel struct {
	Uid         int64
	Username    string
	IsBlock     bool
	Price       float64
	IsFree      bool
	PaymentDate *time.Time
}
