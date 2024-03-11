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
	ID          string    `db:"id" json:"number,omitempty,type(string)"`
	UserID      int       `db:"user_id,omitempty" json:"-"`
	Action      string    `db:"action,omitempty" json:"-"`
	Status      string    `db:"status,omitempty" json:"status"`
	Accrual     float32   `db:"accrual,omitempty" json:"accrual,omitempty"`
	UploadAt    time.Time `db:"upload_at,omitempty" json:"uploaded_at"`
	ProcessedAt time.Time `db:"processed_at,omitempty" json:"-"`
}

type UserBalance struct {
	UserID    int     `db:"user_id" json:"-"`
	Balance   float32 `db:"balance,omitempty" json:"current"`
	Withdrawn float32 `db:"withdrawn,omitempty" jsonn:"withdrawn"`
}

type Secret struct {
	Secret string
}

type WithdrawnRequest struct {
	Order string  `json:"order" valid:"required,type(string)"`
	Sum   float32 `json:"sum" valid:"required,type(float32)"`
}

type WithdrawnResponse struct {
	Order       string    `json:"order" db:"id"`
	Sum         float32   `json:"sum" db:"accrual"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}
