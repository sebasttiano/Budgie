package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"github.com/sebasttiano/Budgie/internal/service"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func (s *ServerViews) UserRegister(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Error("got request with wrong header", zap.String("Content-Type", r.Header.Get("Content-Type")))
		makeResponse(w, http.StatusBadRequest, "error: Content-Type must be application/json")
		return
	}

	logger.Log.Debug("decoding incoming request")
	var user models.User
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&user); err != nil {
		logger.Log.Error("cannot decode request JSON body", zap.Error(err))
		makeResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request payload: %v", err))
		return
	}

	_, err := govalidator.ValidateStruct(user)
	if err != nil {
		logger.Log.Error("error validating user", zap.Error(err))
		makeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	token, err := s.serv.Register(ctx, &user)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			logger.Log.Info("user already exists")
			makeResponse(w, http.StatusConflict, "error: User already exists")
			return
		}
		logger.Log.Error("couldn`t register user. error: ", zap.Error(err))
		makeResponse(w, http.StatusInternalServerError, err.Error())
	}
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	makeResponse(w, http.StatusOK, "user registered and authenticated")
}

func (s *ServerViews) UserLogin(w http.ResponseWriter, r *http.Request) {

}

func (s *ServerViews) UserGetBalance(w http.ResponseWriter, r *http.Request) {

}
