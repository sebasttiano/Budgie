package handlers

import (
	"github.com/sebasttiano/Budgie/internal/service"
)

type ServerViews struct {
	serv *service.Service
}

func NewServerViews(s *service.Service) *ServerViews {
	return &ServerViews{serv: s}
}
