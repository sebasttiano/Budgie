package models

type User struct {
	ID           int
	Login        string `json:"login" valid:"required,type(string)"`
	Password     string `json:"password" valid:"required,type(string)"`
	RegisteredAT string
}

type Order struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id,omitempty"`
	Action      string `json:"action,omitempty"`
	Status      string
	Accrual     string
	UploadAt    string
	ProcessedAt string
}

type Secret struct {
	Secret string
}
