package application

import (
	"context"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sebasttiano/Budgie/internal/config"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/server"
	"github.com/sebasttiano/Budgie/internal/storage"
	"github.com/sebasttiano/Budgie/internal/worker"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

	var store storage.Storer
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
	if err != nil {
		logger.Log.Error("init storage failed", zap.Error(err))
		return
	}

	if cfg.SecretKey == "" {
		cfg.SecretKey, err = store.GetKey()
		if err != nil {
			logger.Log.Error("get secret key failed", zap.Error(err))
			return
		}
	}

	// Init pools
	pool, err := worker.NewWorkerPool(cfg.WorkerNumber, cfg.TaskChannelSize)
	if err != nil {
		logger.Log.Error("start worker pool failed", zap.Error(err))
		return
	}

	awaitPool, err := worker.NewWaitingPool(cfg.WorkerNumber, cfg.TaskChannelSize, 3*time.Second)
	if err != nil {
		logger.Log.Error("start await pool failed", zap.Error(err))
		return
	}

	pool.Start()
	defer pool.Stop()

	awaitPool.Start()
	defer awaitPool.Stop()

	srv := server.NewServer(cfg.ServerAddress, store, pool, awaitPool, cfg.SecretKey, cfg.AccrualAddress)

	srv.InitRouter()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go srv.Start(&cfg)
	go srv.HandleShutdown(ctx, wg)

	wg.Wait()
}
