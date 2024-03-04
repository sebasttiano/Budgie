package application

import (
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sebasttiano/Budgie/internal/config"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/service"
	"github.com/sebasttiano/Budgie/internal/storage"
	"go.uber.org/zap"
	"net/http"
	"os"
)

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
	serv := service.NewService(store)

	logger.Log.Info("Running server", zap.String("address", cfg.ServerAddress))
	if err := http.ListenAndServe(cfg.ServerAddress, serv.InitRouter()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Log.Error("server error", zap.Error(err))
	}
}
