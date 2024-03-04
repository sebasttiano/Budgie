package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sebasttiano/Budgie/internal/config"
	"github.com/sebasttiano/Budgie/internal/handlers"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/service"
	"github.com/sebasttiano/Budgie/internal/storage"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	srv   *http.Server
	views *handlers.ServerViews
}

func NewServer(serverAddr string, store storage.Store) *Server {
	return &Server{
		srv:   &http.Server{Addr: serverAddr},
		views: handlers.NewServerViews(service.NewService(store)),
	}
}

func (s *Server) InitRouter() {

	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(handlers.WithLogging, handlers.GzipMiddleware)

	r.Route("/api/user/", func(r chi.Router) {
		r.Post("/register", s.views.UserRegister)
		r.Post("/login", s.views.UserLogin)
		r.Post("/orders/", s.views.LoadOrder)
		r.Get("/orders", s.views.GetOrders)
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", s.views.GetBalance)
			r.Post("/withdraw", s.views.WithdrawBalance)
		})
		r.Get("/withdrawals", s.views.WithdrawHistory)
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

func Run() {

	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Println("parsing config failed")
		return
	}

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		fmt.Println("logger initialization failed")
		return
	}

	var store storage.Store
	var conn *sqlx.DB
	if cfg.DatabaseURI != "" {
		conn, err = sqlx.Connect("pgx", cfg.DatabaseURI)
		if err != nil {
			logger.Log.Error("database openning failed", zap.Error(err))
			os.Exit(1)
		}
		defer conn.Close()
		logger.Log.Info("init database storage")
	} else {
		logger.Log.Error("error. you must specify database uri ")
		os.Exit(1)
	}

	store, err = storage.NewDBStorage(conn, true, 3, 1)
	server := NewServer(cfg.ServerAddress, store)

	server.InitRouter()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go server.Start(&cfg)
	go server.HandleShutdown(ctx, wg)

	wg.Wait()
}
