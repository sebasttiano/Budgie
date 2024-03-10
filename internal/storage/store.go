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
)

var ErrDBNoRows = errors.New("sql: no rows in result set")

type Storer interface {
	UserExists(ctx context.Context, login string) (bool, error)
	GetUser(ctx context.Context, user *models.User) error
	AddUser(ctx context.Context, user *models.User) error
	GetKey() (string, error)
	GetOrder(ctx context.Context, order *models.Order, number int) error
	SetOrder(ctx context.Context, order *models.Order) error
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

	sqlInsert := `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`

	var id int
	if err := tx.GetContext(ctx, &id, sqlInsert, user.Login, hashedPassword); err != nil {
		tx.Rollback()
		return err
	}
	user.ID = id

	tx.Commit()
	return nil
}

func (d *DBStorage) GetOrder(ctx context.Context, order *models.Order, number int) error {

	sqlSelect := `SELECT id, user_id, status, action, accrual, upload_at FROM orders WHERE id = $1`

	if err := d.conn.GetContext(ctx, order, sqlSelect, number); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("order not found, %w", ErrDBNoRows)
		} else {
			return err
		}
	}
	return nil
}

func (d *DBStorage) SetOrder(ctx context.Context, order *models.Order) error {

	tx, err := d.conn.Beginx()
	if err != nil {
		return err
	}

	sqlInsert := `INSERT INTO orders (id, user_id, status, action, accrual)
                      VALUES ($1, $2, $3, $4, $5)
                      ON CONFLICT (id) DO UPDATE
                      SET status = excluded.status, action = excluded.action, accrual = excluded.accrual, processed_at = now();`

	if _, err := tx.ExecContext(ctx, sqlInsert, order.ID, order.UserID, order.Status, order.Action, order.Accrual); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
