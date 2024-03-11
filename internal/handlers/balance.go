package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"github.com/sebasttiano/Budgie/internal/service"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var (
	ErrBalanceValidation     = errors.New("balance validation error")
	ErrBalanceNotEnoughBonus = errors.New("not enough promotional point on your balance")
)

func (s *ServerViews) GetBalance(w http.ResponseWriter, r *http.Request) {

	payload, err := GetTokenPayload(r)
	if err != nil {
		logger.Log.Error("token payload error: ", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	balance, err := s.serv.GetBalance(ctx, payload.UserID)
	if err != nil {
		message := fmt.Sprintf("failed to get balance for user %d", payload.UserID)
		logger.Log.Error(message, zap.Error(err))
		makeResponse(w, http.StatusInternalServerError, message)
		return
	}
	makeResponse(w, http.StatusOK, balance)

}

func (s *ServerViews) WithdrawBalance(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Error("got request with wrong header", zap.String("Content-Type", r.Header.Get("Content-Type")))
		makeResponse(w, http.StatusBadRequest, "error: Content-Type must be application/json")
		return
	}

	payload, err := GetTokenPayload(r)
	if err != nil {
		logger.Log.Error("token payload error: ", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var withdrawn models.WithdrawnRequest
	if err := s.ValidateBalance(ctx, &withdrawn, r, payload.UserID); err != nil {
		switch {
		case errors.Is(err, ErrBalanceValidation):
			logger.Log.Error("withdrawn error: ", zap.Error(err))
			makeResponse(w, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, ErrOrderValidationNumber):
			logger.Log.Error("withdrawn error: ", zap.Error(err))
			makeResponse(w, http.StatusUnprocessableEntity, err.Error())
			return
		case errors.Is(err, ErrBalanceNotEnoughBonus):
			logger.Log.Error("withdrawn error: ", zap.Error(err))
			makeResponse(w, http.StatusPaymentRequired, err.Error())
			return
		case errors.Is(err, service.ErrOrderAnotherUser):
			logger.Log.Error("withdrawn error: ", zap.Error(err))
			makeResponse(w, http.StatusConflict, "order already passed by another user")
			return
		case errors.Is(err, service.ErrOrderAlreadyExist):
			logger.Log.Error("withdrawn error: ", zap.Error(err))
			makeResponse(w, http.StatusUnprocessableEntity, "order has been already process with accrual system. withdrawn unavailable")
			return
		default:
			logger.Log.Error("withdrawn error: ", zap.Error(err))
			makeResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	//orderNumber, _ := strconv.Atoi(withdrawn.Order)

	order := &models.Order{
		ID:      withdrawn.Order,
		UserID:  payload.UserID,
		Status:  models.OrderStatusProcessed,
		Action:  models.OrderActionWithdraw,
		Accrual: withdrawn.Sum,
	}

	if err := s.serv.SaveOrder(ctx, order); err != nil {
		logger.Log.Error("withdrawn error: failed to save order", zap.Error(err))
		makeResponse(w, http.StatusInternalServerError, service.ErrOrderSave.Error())
		return
	}

	balance := &models.UserBalance{UserID: payload.UserID, Balance: 0, Withdrawn: withdrawn.Sum}
	if err := s.serv.SetBalance(ctx, balance); err != nil {
		logger.Log.Error("withdrawn error: failed to save new balance", zap.Error(err))
		makeResponse(w, http.StatusInternalServerError, "failed to save new balance")
		return
	}
	message := fmt.Sprintf("successfully make withdraw for order %s !!!", withdrawn.Order)
	logger.Log.Info(message)
	makeResponse(w, http.StatusOK, message)
}

func (s *ServerViews) WithdrawHistory(w http.ResponseWriter, r *http.Request) {

	payload, err := GetTokenPayload(r)
	if err != nil {
		logger.Log.Error("token payload error: ", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userWithdrawals, err := s.serv.GetAllUserWithdrawals(ctx, payload.UserID)
	if err != nil {
		if errors.Is(err, service.ErrNoUserWithdrawals) {
			logger.Log.Debug(err.Error())
			makeResponse(w, http.StatusNoContent, nil)
			return
		}
		logger.Log.Error("error ", zap.Error(err))
		makeResponse(w, http.StatusInternalServerError, "load withdrawals history failed")
		return
	}
	makeResponse(w, http.StatusOK, userWithdrawals)

}

func (s *ServerViews) ValidateBalance(ctx context.Context, withdrawn *models.WithdrawnRequest, r *http.Request, userID int) error {

	logger.Log.Debug("decoding incoming request")
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(withdrawn); err != nil {
		return fmt.Errorf("%w: cannot decode JSON body: %s", ErrBalanceValidation, err)
	}

	_, err := govalidator.ValidateStruct(withdrawn)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBalanceValidation, err)
	}

	orderStr := withdrawn.Order

	if err := common.ValidateLuhnSum(orderStr); err != nil {
		return fmt.Errorf("%w: check you input: %v", ErrOrderValidationNumber, err)
	}

	if err := s.serv.CheckOrder(ctx, withdrawn.Order, userID); err != nil {
		return err
	}

	balance, err := s.serv.GetBalance(ctx, userID)
	if err != nil {
		return err
	}

	if balance.Balance < withdrawn.Sum {
		return fmt.Errorf("%w: validation failed", ErrBalanceNotEnoughBonus)
	}
	return nil
}
