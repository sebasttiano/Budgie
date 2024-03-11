package tasks

import (
	"context"
	"errors"
	"fmt"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"github.com/sebasttiano/Budgie/internal/storage"
	"github.com/sebasttiano/Budgie/internal/worker"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var ErrInternalAccrualSystem = errors.New("accrual returned internal error")

const (
	AccrualStatusRegistered = "REGISTERED"
	AccrualStatusInvalid    = "INVALID"
	AccrualStatusProcessing = "PROCESSING"
	AccrualStatusProcessed  = "PROCESSED"
)

type AccrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual,omitempty"`
}

type ProcessOrder struct {
	c           *common.HTTPClient
	orderNumber string
	store       storage.Storer
	awaitPool   worker.Pool
}

func NewProcessOrder(baseURL string, retries int, order string, store storage.Storer, awaitPool worker.Pool) *ProcessOrder {
	return &ProcessOrder{c: common.NewHTTPClient(baseURL, retries), orderNumber: order, store: store, awaitPool: awaitPool}
}

func (p *ProcessOrder) Execute() error {

	var accrual AccrualResponse
	resp, err := p.c.Get("/api/orders/"+p.orderNumber, &accrual)
	if err != nil {
		logger.Log.Error("get request to accrual system failed. retrying...")
		p.awaitPool.AddWork(p)
		return nil
	}

	order := &models.Order{ID: p.orderNumber, Accrual: 0.00, Action: models.OrderActionAdd}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	switch resp.StatusCode() {
	case http.StatusNoContent:
		logger.Log.Info(fmt.Sprintf("order %s has not been uploaded yet to accrual system. awaiting...", p.orderNumber))
		p.awaitPool.AddWork(p)
		return nil
	case http.StatusOK:
		switch accrual.Status {
		case AccrualStatusProcessed:
			order.Status = models.OrderStatusProcessed
			if accrual.Accrual != 0 {
				order.Accrual = accrual.Accrual
			}
			logger.Log.Debug(fmt.Sprintf("order %s has been processed. accrual equal to %f", p.orderNumber, accrual.Accrual))
		case AccrualStatusInvalid:
			order.Status = models.OrderStatusInvalid
		case AccrualStatusProcessing:
			order.Status = models.OrderStatusProcessing
			logger.Log.Debug(fmt.Sprintf("order %s is processing in the accrual system at the monent. awaiting...", p.orderNumber))
			p.awaitPool.AddWork(p)
		case AccrualStatusRegistered:
			order.Status = models.OrderStatusProcessing
			logger.Log.Debug(fmt.Sprintf("order %s has been registered in the accrual system. awaiting...", p.orderNumber))
			p.awaitPool.AddWork(p)
		}

	case http.StatusInternalServerError:
		return fmt.Errorf("error: %w", ErrInternalAccrualSystem)
	}

	if err := p.store.SetOrder(ctx, order); err != nil {
		return err
	}

	return nil
}

func (p *ProcessOrder) OnFailure(err error) {

	order := &models.Order{ID: p.orderNumber, Accrual: 0.00, Action: models.OrderActionAdd}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if errors.Is(err, ErrInternalAccrualSystem) {
		order.Status = models.OrderStatusError
		if err := p.store.SetOrder(ctx, order); err != nil {
			logger.Log.Error("worker failed with", zap.Error(err))
		}
	}

}
