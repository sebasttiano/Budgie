package handlers

import (
	"encoding/json"
	"github.com/sebasttiano/Budgie/internal/logger"
	"go.uber.org/zap"
	"net/http"
)

// marshalResponse to json
func makeResponse(w http.ResponseWriter, code int, v any) {

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)

	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		logger.Log.Error("error encoding response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
