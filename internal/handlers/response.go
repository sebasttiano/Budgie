package handlers

import (
	"encoding/json"
	"github.com/sebasttiano/Budgie/internal/logger"
	"go.uber.org/zap"
	"net/http"
)

// marshalResponse to json
func marshalResponse(res http.ResponseWriter, code int, v any) {

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(code)

	enc := json.NewEncoder(res)
	if err := enc.Encode(v); err != nil {
		logger.Log.Error("error encoding response", zap.Error(err))
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
