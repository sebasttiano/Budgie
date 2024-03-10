package models

import (
	"time"
)

const (
	OrderStatusNew        = "NEW"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusInvalid    = "INVALID"
	OrderStatusProcessed  = "PROCESSED"
	OrderStatusError      = "ERROR"

	OrderActionAdd      = "add"
	OrderActionWithdraw = "withdraw"
)

type User struct {
	ID           int
	Login        string `json:"login" valid:"required,type(string)"`
	Password     string `json:"password" valid:"required,type(string)"`
	RegisteredAT string
}

type Order struct {
	ID          int       `db:"id" json:"number,omitempty"`
	UserID      int       `db:"user_id,omitempty" json:"-"`
	Action      string    `db:"action,omitempty" json:"-"`
	Status      string    `db:"status,omitempty" json:"status"`
	Accrual     float32   `db:"accrual,omitempty" json:"accrual,omitempty"`
	UploadAt    time.Time `db:"upload_at,omitempty" json:"uploaded_at"`
	ProcessedAt time.Time `db:"processed_at,omitempty" json:"-"`
}

type UserBalance struct {
	UserID    int     `db:"user_id"`
	Balance   float32 `db:"balance,omitempty"`
	Withdrawn float32 `db:"withdrawn,omitempty"`
}

type Secret struct {
	Secret string
}
