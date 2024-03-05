package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/models"
	"github.com/sebasttiano/Budgie/internal/storage"
)

var ErrUserExists = errors.New("user already exists")

type Authenticator interface {
	Register(ctx context.Context, u *models.User) (string, error)
	Login(ctx context.Context, u models.User) (string, error)
	//GenerateToken(ctx context.Context, u *models.User) (string, error)
}

type Service struct {
	Store     storage.Store
	secretKey string
}

func NewService(store storage.Store, secretKey string) *Service {
	return &Service{
		Store:     store,
		secretKey: secretKey,
	}
}

func (s *Service) Register(ctx context.Context, u *models.User) (string, error) {

	exist, err := s.Store.UserExists(ctx, u.Login)
	if err != nil {
		return "", err
	}
	if !exist {
		if err := s.Store.AddUser(ctx, u); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("user %s already exists: %w", u.Login, ErrUserExists)
	}

	token, err := common.BuildJWTString(u.ID, s.secretKey)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) Login(ctx context.Context, u models.User) (string, error) {
	return "", nil
}
