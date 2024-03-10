package models

import "time"

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
	ID          int       `db:"id"`
	UserID      int       `db:"user_id,omitempty"`
	Action      string    `db:"action,omitempty"`
	Status      string    `db:"status,omitempty"`
	Accrual     float32   `db:"accrual,omitempty"`
	UploadAt    time.Time `db:"upload_at,omitempty"`
	ProcessedAt time.Time `db:"processed_at,omitempty"`
}

type Secret struct {
	Secret string
}
