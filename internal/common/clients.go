package common

import (
	"github.com/go-resty/resty/v2"
	"github.com/sebasttiano/Budgie/internal/logger"
	"go.uber.org/zap"
	"net/http"
	"time"
)

// HTTPClientError for serialize/deserialize errors
type HTTPClientError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// HTTPClient simple client
type HTTPClient struct {
	client *resty.Client
	Error  HTTPClientError
}

// NewHTTPClient constructor for HTTTPClient
func NewHTTPClient(baseURL string, retries int) *HTTPClient {

	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetRetryCount(retries)
	client.SetRetryWaitTime(1 * time.Second)
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		return err != nil || r.StatusCode() == http.StatusTooManyRequests
	})

	return &HTTPClient{client: client}
}

func (h *HTTPClient) Get(relativeURL string, object any) (*resty.Response, error) {

	resp, err := h.client.R().SetError(&h.Error).SetResult(object).Get(relativeURL)
	if err != nil {
		logger.Log.Error("method get error", zap.Error(err))
		return nil, err
	}
	return resp, nil
}
