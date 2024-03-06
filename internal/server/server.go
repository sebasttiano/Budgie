package server

import (
	"context"
	"errors"
	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sebasttiano/Budgie/internal/config"
	"github.com/sebasttiano/Budgie/internal/handlers"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/service"
	"github.com/sebasttiano/Budgie/internal/storage"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	srv     *http.Server
	views   *handlers.ServerViews
	JWTWare *jwtmiddleware.JWTMiddleware
}

func NewServer(serverAddr string, store storage.Storer, secretKey string) *Server {
	keyFunc := func(ctx context.Context) (interface{}, error) {
		return []byte(secretKey), nil
	}
	// Set up the JWT validator.
	jwtValidator, err := validator.New(
		keyFunc,
		validator.HS256,
		serverAddr+"/api/user/login",
		[]string{serverAddr},
	)
	if err != nil {
		logger.Log.Error("failed to set up the validator: %v", zap.Error(err))
	}

	return &Server{
		srv:     &http.Server{Addr: serverAddr},
		views:   handlers.NewServerViews(service.NewService(store, secretKey)),
		JWTWare: jwtmiddleware.New(jwtValidator.ValidateToken),
	}
}

func (s *Server) InitRouter() {
	r := chi.NewRouter()

	r.Route("/api/user/", func(r chi.Router) {
		r.Use(middleware.RealIP)
		r.Use(handlers.WithLogging, handlers.GzipMiddleware)
		r.Method(http.MethodPost, "/register", http.HandlerFunc(s.views.UserRegister))
		r.Method(http.MethodPost, "/login", http.HandlerFunc(s.views.UserLogin))
		r.Method(http.MethodPost, "/orders", s.JWTWare.CheckJWT(http.HandlerFunc(s.views.LoadOrder)))
		r.Method(http.MethodGet, "/orders", s.JWTWare.CheckJWT(http.HandlerFunc(s.views.GetOrders)))
		r.Route("/balance/", func(r chi.Router) {
			r.Method(http.MethodGet, "/", s.JWTWare.CheckJWT(http.HandlerFunc(s.views.GetBalance)))
			r.Method(http.MethodPost, "/withdraw", s.JWTWare.CheckJWT(http.HandlerFunc(s.views.WithdrawBalance)))
		})
		r.Method(http.MethodGet, "/withdrawals", s.JWTWare.CheckJWT(http.HandlerFunc(s.views.WithdrawHistory)))

	})

	s.srv.Handler = r
}

func (s *Server) Start(cfg *config.Config) {
	logger.Log.Info("Running server", zap.String("address", cfg.ServerAddress))
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Log.Error("server error", zap.Error(err))
		return
	}
}

func (s *Server) HandleShutdown(ctx context.Context, wg *sync.WaitGroup) {

	defer wg.Done()

	<-ctx.Done()
	logger.Log.Info("shutdown signal caught. shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := s.srv.Shutdown(ctx)
	if err != nil {
		logger.Log.Error("server shutdown error", zap.Error(err))
		return
	}
	logger.Log.Info("server gracefully shutdown")
}
