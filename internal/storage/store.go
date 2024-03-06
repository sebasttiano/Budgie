package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"go.uber.org/zap"
	"time"
)

var ErrDBNoRows = errors.New("sql: no rows in result set")

type Store interface {
	UserExists(ctx context.Context, login string) (bool, error)
	GetUser(ctx context.Context, user *models.User) error
	AddUser(ctx context.Context, user *models.User) error
	GetKey() (string, error)
}

func (d *DBStorage) GetKey() (string, error) {

	s := &models.Secret{}
	sqlSelect := "SELECT secret FROM secrets"

	if err := d.conn.Get(s, sqlSelect); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Log.Error("secret not found", zap.Error(err))
		}
		return "", err
	}
	return s.Secret, nil
}

func (d *DBStorage) UserExists(ctx context.Context, login string) (bool, error) {

	u := &models.User{}
	sqlSelect := `SELECT login FROM users WHERE login = $1`

	if err := d.conn.GetContext(ctx, u, sqlSelect, login); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		} else if errors.As(err, &pgError) {
			return false, NewDBError(err, pgError.Code)
		} else {
			return false, err
		}
	}
	return true, nil
}

func (d *DBStorage) GetUser(ctx context.Context, user *models.User) error {

	sqlSelect := `SELECT id, login, password FROM users WHERE login = $1`

	if err := d.conn.GetContext(ctx, user, sqlSelect, user.Login); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user not found, %w", ErrDBNoRows)
		}
		return err
	}
	return nil
}

func (d *DBStorage) AddUser(ctx context.Context, user *models.User) error {

	hashedPassword, err := common.HashPassword(user.Password)
	if err != nil {
		logger.Log.Error("failed to hash password", zap.Error(err))
		return err
	}

	tx, err := d.conn.Beginx()
	if err != nil {
		return err
	}

	sqlInsert := `INSERT INTO users (login, password, registered_at) VALUES ($1, $2, $3) RETURNING id`

	var id int
	if err := tx.GetContext(ctx, &id, sqlInsert, user.Login, hashedPassword, time.Now()); err != nil {
		tx.Rollback()
		return err
	}
	user.ID = id

	tx.Commit()
	return nil
}
