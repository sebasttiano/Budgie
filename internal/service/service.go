package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"github.com/sebasttiano/Budgie/internal/storage"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserRegisrationFailed = errors.New("user registration failed")
	ErrOrderAnotherUser      = errors.New("order belongs to other user")
	ErrOrderAlreadyExist     = errors.New("order already exist")
	ErrOrderSave             = errors.New("order save failed")
)

type Authenticator interface {
	Register(ctx context.Context, u *models.User) (string, error)
	Login(ctx context.Context, u *models.User) (string, error)
}

type Service struct {
	Store     storage.Storer
	secretKey string
}

func NewService(store storage.Storer, secretKey string) *Service {
	return &Service{
		Store:     store,
		secretKey: secretKey,
	}
}

func (s *Service) Register(ctx context.Context, u *models.User) (string, error) {

	if err := s.Store.AddUser(ctx, u); err != nil {
		return "", fmt.Errorf("%w: user %s error: %v", ErrUserRegisrationFailed, u.Login, err)
	}

	token, err := common.BuildJWTString(u.ID, s.secretKey)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) UserExists(ctx context.Context, u *models.User) (bool, error) {

	exist, err := s.Store.UserExists(ctx, u.Login)
	if err != nil {
		return false, err
	}
	return exist, nil
}
func (s *Service) Login(ctx context.Context, u *models.User) (string, error) {

	passPassword := u.Password
	if err := s.Store.GetUser(ctx, u); err != nil {
		return "", fmt.Errorf("%w: user %s error: %v", ErrUserNotFound, u.Login, err)
	}

	if err := common.CheckPasswordHash(passPassword, u.Password); err != nil {
		return "", err
	}

	logger.Log.Info(fmt.Sprintf("user %s login succesfully", u.Login))

	return common.BuildJWTString(u.ID, s.secretKey)
}

func (s *Service) CheckOrder(ctx context.Context, number int, user int) error {

	var order models.Order
	if err := s.Store.GetOrder(ctx, &order, number); err != nil {
		if errors.Is(err, storage.ErrDBNoRows) {
			return nil
		}
	} else {
		return err
	}
	fmt.Println(order)
	if order.UserID != user {
		return fmt.Errorf("check order error: %w", ErrOrderAnotherUser)
	} else {
		return fmt.Errorf("check order error: %w", ErrOrderAlreadyExist)
	}
}

func (s *Service) SaveOrder(ctx context.Context, o *models.Order) error {

	if err := s.Store.SetOrder(ctx, o); err != nil {
		return fmt.Errorf("order with number %d error: %w. reason: %v", o.ID, ErrOrderSave, err)
	}

	return nil
}
