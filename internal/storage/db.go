package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/sebasttiano/Budgie/internal/logger"
	"go.uber.org/zap"
	"time"
)

var pgError *pgconn.PgError

type DBError struct {
	Code string
	Err  error
}

func (e DBError) Error() string {
	return fmt.Sprintf("db failed with code: %s. message: %v", e.Code, e.Err)
}

func (e DBError) Unwrap() error {
	return e.Err
}

func NewDBError(err error, code string) *DBError {
	return &DBError{code, err}
}

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
	return db, nil
}

// Bootstrap creates tables in DB
func (d *DBStorage) Bootstrap(ctx context.Context) error {

	logger.Log.Debug("checking db tables")
	// create types
	if _, err := d.conn.ExecContext(ctx, `
		CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED', 'ERROR')
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

	// create table users
	if _, err := tx.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id serial PRIMARY KEY,
            login varchar(128) NOT NULL, 
			password varchar(128) NOT NULL,
			registered_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
	       	UNIQUE(login) 
        )
	`); err != nil {
		logger.Log.Error("failed to create users table", zap.Error(err))
		tx.Rollback()
		return err
	} else {
		logger.Log.Info("table users OK!")
	}

	// create table balance
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS balance (
		    user_id integer REFERENCES users (id) ON DELETE CASCADE,
		    balance numeric(10,2)
		) 
	`); err != nil {
		logger.Log.Error("failed to create balance table", zap.Error(err))
		tx.Rollback()
		return err
	} else {
		logger.Log.Info("table balance OK!")
	}

	// create table for orders
	if _, err := tx.ExecContext(ctx, `
	   CREATE TABLE IF NOT EXISTS orders (
	       id bigint PRIMARY KEY,
		   user_id bigint REFERENCES users(id) ON DELETE CASCADE,
	       status order_status NOT NULL,
	       action order_actions NOT NULL,
	       accrual numeric(10,2),
	       upload_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
	       processed_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
	   )
	`); err != nil {
		logger.Log.Error("failed to create orders table", zap.Error(err))
		tx.Rollback()
		return err
	} else {
		logger.Log.Info("table orders OK!")
	}

	// create table secret
	if _, err := tx.ExecContext(ctx, `
	   CREATE TABLE IF NOT EXISTS secrets (
	       secret varchar(128) NOT NULL
	   )
	`); err != nil {
		logger.Log.Error("failed to create secrets table", zap.Error(err))
		tx.Rollback()
		return err
	} else {
		logger.Log.Info("table secrets OK!")
	}

	// commit
	if err := tx.Commit(); err != nil {
		return err
	}

	// Set one row table secrets
	if _, err := d.conn.ExecContext(ctx, `CREATE UNIQUE INDEX one_row_only_uidx ON secrets (( true ))`); err != nil && errors.As(err, &pgError) {
		if pgError.Code == pgerrcode.DuplicateTable {
			logger.Log.Debug("index one_row_only_uidx already exists. going on")
		} else {
			logger.Log.Error("failed to index one_row_only_uidx", zap.Error(err))
			return err
		}
	}
	// Generate random secret key if doesn`t exist
	if _, err := d.conn.ExecContext(ctx, `
			INSERT INTO secrets (secret) VALUES ((SELECT MD5(random()::text))) ON CONFLICT DO NOTHING;
		`); err != nil {
		logger.Log.Error("failed to generate secret key in secrets table", zap.Error(err))
		return err
	}

	return nil
}
