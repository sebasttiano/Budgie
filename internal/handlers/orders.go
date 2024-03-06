package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"github.com/sebasttiano/Budgie/internal/service"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrOrderValidationRequest = errors.New("invalid format of request")
	ErrOrderValidationNumber  = errors.New("invalid format of order number")
)

func (s *ServerViews) LoadOrder(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "text/plain" {
		logger.Log.Error("got request with wrong header", zap.String("Content-Type", r.Header.Get("Content-Type")))
		makeResponse(w, http.StatusBadRequest, "error: Content-Type must be text/plain")
		return
	}

	payload, err := GetTokenPayload(r)
	if err != nil {
		logger.Log.Error("load order error: ", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	number, err := s.ValidateAndCheckOrder(ctx, r, payload.UserID)

	if err != nil {
		switch {
		case errors.Is(err, ErrOrderValidationRequest):
			logger.Log.Error("load order error: ", zap.Error(err))
			makeResponse(w, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, ErrOrderValidationNumber):
			logger.Log.Error("load order error: ", zap.Error(err))
			makeResponse(w, http.StatusUnprocessableEntity, err.Error())
			return
		case errors.Is(err, service.ErrOrderAnotherUser):
			logger.Log.Error("load order error: ", zap.Error(err))
			makeResponse(w, http.StatusConflict, "order already passed by another user")
			return
		case errors.Is(err, service.ErrOrderAlreadyExist):
			logger.Log.Error("load order error: ", zap.Error(err))
			makeResponse(w, http.StatusOK, "order already downloaded")
			return
		default:
			logger.Log.Error("load order error: ", zap.Error(err))
			makeResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	order := &models.Order{
		ID:      number,
		UserID:  payload.UserID,
		Status:  models.OrderStatusRegistered,
		Action:  models.OrderActionAdd,
		Accrual: 0.00,
	}

	if err := s.serv.SaveOrder(ctx, order); err != nil {
		logger.Log.Error("load order error: ", zap.Error(err))
		makeResponse(w, http.StatusInternalServerError, service.ErrOrderSave.Error())
		return
	}

	message := fmt.Sprintf("Successfully load order %d !!!", number)
	logger.Log.Info(message)
	makeResponse(w, http.StatusOK, message)
}

func (s *ServerViews) GetOrders(w http.ResponseWriter, r *http.Request) {

}

func (s *ServerViews) ValidateAndCheckOrder(ctx context.Context, r *http.Request, user int) (int, error) {

	byteArray, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	r.Body.Close()

	orderStr := string(byteArray[:])

	order, err := strconv.Atoi(orderStr)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to convert order to int: %v", ErrOrderValidationRequest, err)
	}

	if err := common.ValidateLuhnSum(orderStr); err != nil {
		return 0, fmt.Errorf("%w: check you input: %v", ErrOrderValidationNumber, err)
	}

	if err := s.serv.CheckOrder(ctx, order, user); err != nil {
		return 0, err
	}
	return order, nil
}
