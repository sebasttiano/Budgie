package storage

import (
	"context"
	"errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/sebasttiano/Budgie/internal/logger"
	"go.uber.org/zap"
	"time"
)

var pgError *pgconn.PgError

// DBStorage Keeps data in database
type DBStorage struct {
	conn *sqlx.DB
}

// NewDBStorage returns new database storage
func NewDBStorage(conn *sqlx.DB, bootstrap bool, retries uint, backoffFactor uint) (*DBStorage, error) {
	db := &DBStorage{conn: conn}
	if bootstrap {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := db.Bootstrap(ctx); err != nil {
			if errors.As(err, &pgError) {
				if pgError.Code == pgerrcode.InFailedSQLTransaction {
					logger.Log.Debug("rollback in bootstrap occured!")
				} else {
					logger.Log.Error("db bootstrap failed", zap.Error(err))
				}
			}
			return nil, err
		}
	}
	return &DBStorage{conn: conn}, nil
}

// Bootstrap creates tables in DB
func (d *DBStorage) Bootstrap(ctx context.Context) error {

	logger.Log.Debug("checking db tables")
	// create types
	if _, err := d.conn.ExecContext(ctx, `
		CREATE TYPE order_status AS ENUM ('REGISTERED', 'INVALID', 'PROCESSING', 'PROCESSED')
	`); err != nil && errors.As(err, &pgError) {
		if pgError.Code == pgerrcode.DuplicateObject {
			logger.Log.Debug("type order_status already exists. going on")
		} else {
			logger.Log.Error("failed to create type order_status", zap.Error(err))
			return err
		}
	} else {
		logger.Log.Info("created type order_status")
	}

	if _, err := d.conn.ExecContext(ctx, `
		CREATE TYPE order_actions AS ENUM ('add', 'withdraw')
	`); err != nil && errors.As(err, &pgError) {
		if pgError.Code == pgerrcode.DuplicateObject {
			logger.Log.Debug("type order_actions already exists. going on")
		} else {
			logger.Log.Error("failed to create type order_actions", zap.Error(err))
			return err
		}
	} else {
		logger.Log.Info("created type order_actions")
	}

	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// create table for gauge metrics
	if _, err := tx.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id serial PRIMARY KEY,
            login varchar(128) NOT NULL, 
			password varchar(128) NOT NULL,
            balance numeric(10,2),
	       	UNIQUE(login) 
        )
	`); err != nil {
		logger.Log.Error("failed to create users table", zap.Error(err))
		tx.Rollback()
		return err
	} else {
		logger.Log.Info("created users table")
	}

	// create table for orders
	if _, err := tx.ExecContext(ctx, `
	   CREATE TABLE IF NOT EXISTS orders (
	       id integer PRIMARY KEY,
		   user_id integer REFERENCES users(id) ON DELETE CASCADE,
	       status order_status NOT NULL,
	       action order_actions NOT NULL,
	       accural numeric(10,2),
	       upload_at timestamp without time zone NOT NULL,
	       processed_at timestamp without time zone NOT NULL
	   )
	`); err != nil {
		logger.Log.Error("failed to create orders table", zap.Error(err))
		tx.Rollback()
		return err
	} else {
		logger.Log.Info("created orders table")
	}

	// commit
	return tx.Commit()
}
