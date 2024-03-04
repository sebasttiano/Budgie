package service

import (
	"github.com/sebasttiano/Budgie/internal/storage"
)

type Service struct {
	Store storage.Store
}

func NewService(store storage.Store) *Service {
	return &Service{
		Store: store,
	}
}
