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

var ErrUserValidation = errors.New("user validation error")

func (s *ServerViews) UserRegister(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Error("got request with wrong header", zap.String("Content-Type", r.Header.Get("Content-Type")))
		makeResponse(w, http.StatusBadRequest, "error: Content-Type must be application/json")
		return
	}

	var user models.User

	if err := s.ValidateUser(&user, r); err != nil {
		logger.Log.Error("registration error: ", zap.Error(err))
		makeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	exist, err := s.serv.UserExists(ctx, &user)
	if err != nil {
		makeResponse(w, http.StatusInternalServerError, err.Error())
	}

	if !exist {
		token, err := s.serv.Register(ctx, &user)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrUserRegisrationFailed):
				logger.Log.Error("couldn`t register user. error: ", zap.Error(err))
				makeResponse(w, http.StatusInternalServerError, err.Error())
				return
			case errors.Is(err, common.ErrTokenCreationFailed):
				logger.Log.Error("error: ", zap.Error(err))
				makeResponse(w, http.StatusOK, "user registered. error: token creation failed, please sign in via /api/user/login.")
				return
			default:
				logger.Log.Error("error: ", zap.Error(err))
				makeResponse(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		makeResponse(w, http.StatusOK, "user registered and authenticated")
	} else {
		makeResponse(w, http.StatusConflict, "error: User already exists")
	}
}

func (s *ServerViews) UserLogin(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Error("got request with wrong header", zap.String("Content-Type", r.Header.Get("Content-Type")))
		makeResponse(w, http.StatusBadRequest, "error: Content-Type must be application/json")
		return
	}

	var user models.User

	if err := s.ValidateUser(&user, r); err != nil {
		logger.Log.Error("auth error: ", zap.Error(err))
		makeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	exist, err := s.serv.UserExists(ctx, &user)

	if err != nil {
		makeResponse(w, http.StatusInternalServerError, err.Error())
	}
	if exist {
		token, err := s.serv.Login(ctx, &user)
		if err != nil {
			switch {
			case errors.Is(err, common.ErrTokenCreationFailed):
				logger.Log.Error("error login user: ", zap.Error(err))
				makeResponse(w, http.StatusInternalServerError, "error: token creation failed, please try login later.")
				return
			case errors.Is(err, common.ErrWrongPassword) || errors.Is(err, service.ErrUserNotFound):
				logger.Log.Error("error login user: ", zap.Error(err))
				makeResponse(w, http.StatusUnauthorized, err.Error())
				return
			default:
				logger.Log.Error("error login user:", zap.Error(err))
				makeResponse(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		makeResponse(w, http.StatusOK, "user authenticated")
		return
	}
	makeResponse(w, http.StatusBadRequest, "user doesn`t exist")
}

func (s *ServerViews) ValidateUser(user *models.User, r *http.Request) error {

	logger.Log.Debug("decoding incoming request")
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&user); err != nil {
		return fmt.Errorf("%w: cannot decode JSON body: %s", ErrUserValidation, err)
	}

	_, err := govalidator.ValidateStruct(user)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUserValidation, err)
	}
	return nil
}
