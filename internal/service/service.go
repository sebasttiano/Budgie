package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"github.com/sebasttiano/Budgie/internal/storage"
	"github.com/sebasttiano/Budgie/internal/tasks"
	"github.com/sebasttiano/Budgie/internal/worker"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserRegisrationFailed = errors.New("user registration failed")
	ErrOrderAnotherUser      = errors.New("order belongs to other user")
	ErrOrderAlreadyExist     = errors.New("order already exist")
	ErrOrderSave             = errors.New("order save failed")
	ErrNoUserOrders          = errors.New("user has no any orders")
	ErrNoUserWithdrawals     = errors.New("user has no any withdrawals")
)

type Authenticator interface {
	Register(ctx context.Context, u *models.User) (string, error)
	Login(ctx context.Context, u *models.User) (string, error)
}

type ServiceSettings struct {
	Key         string
	HTTPRetries int
	AccuralURL  string
	workerPool,
	awaitPool worker.Pool
}

type ServicePools struct {
	MainPool,
	AwaitPool worker.Pool
}
type Service struct {
	Store    storage.Storer
	settings *ServiceSettings
	pools    *ServicePools
}

func NewService(store storage.Storer, settings *ServiceSettings, pools *ServicePools) *Service {
	return &Service{
		Store:    store,
		settings: settings,
		pools:    pools,
	}
}

func (s *Service) Register(ctx context.Context, u *models.User) (string, error) {

	if err := s.Store.AddUser(ctx, u); err != nil {
		return "", fmt.Errorf("%w: user %s error: %v", ErrUserRegisrationFailed, u.Login, err)
	}

	token, err := common.BuildJWTString(u.ID, s.settings.Key)
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

	return common.BuildJWTString(u.ID, s.settings.Key)
}

func (s *Service) CheckOrder(ctx context.Context, number string, user int) error {

	var order models.Order
	if err := s.Store.GetOrder(ctx, &order, number); err != nil {
		if errors.Is(err, storage.ErrDBNoRows) {
			return nil
		} else {
			return err
		}
	}
	if order.UserID != user {
		return fmt.Errorf("check order error: %w", ErrOrderAnotherUser)
	} else {
		return fmt.Errorf("check order error: %w", ErrOrderAlreadyExist)
	}
}

func (s *Service) SaveOrder(ctx context.Context, o *models.Order) error {

	if err := s.Store.SetOrder(ctx, o); err != nil {
		return fmt.Errorf("order with number %s error: %w. reason: %v", o.ID, ErrOrderSave, err)
	}

	return nil
}

func (s *Service) ProccessOrder(ctx context.Context, o *models.Order) error {

	task := tasks.NewProcessOrder(s.settings.AccuralURL, s.settings.HTTPRetries, o.ID, s.Store, s.pools.AwaitPool)
	s.pools.MainPool.AddWork(task)

	return nil
}

func (s *Service) GetAllUserOrders(ctx context.Context, userID int) ([]*models.Order, error) {

	userOrders, err := s.Store.GetAllUserOrders(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(userOrders) == 0 {
		return nil, fmt.Errorf("%w, user id: %d", ErrNoUserOrders, userID)

	}
	return userOrders, nil
}

func (s *Service) GetBalance(ctx context.Context, userID int) (*models.UserBalance, error) {

	balance := &models.UserBalance{UserID: userID}
	if err := s.Store.GetBalance(ctx, balance); err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *Service) SetBalance(ctx context.Context, balance *models.UserBalance) error {

	return s.Store.SetBalance(ctx, balance)
}

func (s *Service) GetAllUserWithdrawals(ctx context.Context, userID int) ([]*models.WithdrawnResponse, error) {
	userWithdrawals, err := s.Store.GetAllUserWithdrawals(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(userWithdrawals) == 0 {
		return nil, fmt.Errorf("%w, user id: %d", ErrNoUserWithdrawals, userID)

	}
	return userWithdrawals, nil
}
