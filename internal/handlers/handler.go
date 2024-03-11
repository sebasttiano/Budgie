package handlers

import (
	"errors"
	"fmt"
	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/service"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

var ErrGetTokenPayload = errors.New("failed to get token payload")

type TokenPayload struct {
	UserID int
}

type ServerViews struct {
	serv *service.Service
}

func NewServerViews(s *service.Service) *ServerViews {
	return &ServerViews{serv: s}
}

func GetTokenPayload(r *http.Request) (*TokenPayload, error) {
	claims, ok := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	if !ok {
		return nil, ErrGetTokenPayload
	}

	userID, err := strconv.Atoi(claims.RegisteredClaims.Subject)
	if err != nil {
		return nil, fmt.Errorf("error to convert user id in int. %w", ErrGetTokenPayload)
	}
	return &TokenPayload{UserID: userID}, nil
}

func CustomJWTErrorHandler(w http.ResponseWriter, r *http.Request, err error) {

	switch {
	case errors.Is(err, jwtmiddleware.ErrJWTMissing):
		logger.Log.Debug("auth without JWT rejected")
		makeResponse(w, http.StatusUnauthorized, "JWT is missing")
	case errors.Is(err, jwtmiddleware.ErrJWTInvalid):
		logger.Log.Debug("auth with invalid JWT rejected")
		makeResponse(w, http.StatusUnauthorized, "JWT is invalid.")
	default:
		logger.Log.Error("failed to parse JWT token", zap.Error(err))
		makeResponse(w, http.StatusInternalServerError, "Something went wrong while checking the JWT.")
	}
}
