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
	GetAllUserOrders(ctx context.Context, userID int) ([]*models.Order, error)
	SetBalance(ctx context.Context, balance *models.UserBalance) error
	GetBalance(ctx context.Context, balance *models.UserBalance) error
	GetAllUserWithdrawals(ctx context.Context, userID int) ([]*models.WithdrawnResponse, error)
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

	// create new user
	sqlInsert := `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`

	var id int
	if err := tx.GetContext(ctx, &id, sqlInsert, user.Login, hashedPassword); err != nil {
		tx.Rollback()
		return err
	}
	user.ID = id

	// create balance for user
	slqInsertBalance := `INSERT INTO balance (user_id) VALUES ($1)`
	if _, err := tx.ExecContext(ctx, slqInsertBalance, user.ID); err != nil {
		tx.Rollback()
	}

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

	var userID int
	sqlInsert := `INSERT INTO orders (id, user_id, status, action, accrual)
                      VALUES ($1, $2, $3, $4, $5)
                      ON CONFLICT (id) DO UPDATE
                      SET status = excluded.status, action = excluded.action, accrual = excluded.accrual, processed_at = now()
                      RETURNING user_id;`

	if err := tx.GetContext(ctx, &userID, sqlInsert, order.ID, order.UserID, order.Status, order.Action, order.Accrual); err != nil {
		tx.Rollback()
		return err
	}

	// increase balance
	if order.Status == models.OrderStatusProcessed && order.Action == models.OrderActionAdd {
		balance := &models.UserBalance{UserID: userID, Balance: order.Accrual}
		if err := d.SetBalance(ctx, balance); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DBStorage) GetAllUserOrders(ctx context.Context, userID int) ([]*models.Order, error) {

	var allOrders []*models.Order
	sqlSelect := `SELECT id, user_id, status, action, accrual, upload_at
					FROM orders
					WHERE user_id = $1 AND status IN ($2, $3, $4, $5)
					ORDER BY upload_at`

	if err := d.conn.SelectContext(ctx, &allOrders, sqlSelect, userID, models.OrderStatusNew, models.OrderStatusProcessing, models.OrderStatusInvalid, models.OrderStatusProcessed); err != nil {
		return nil, err
	}
	return allOrders, nil

}

func (d *DBStorage) SetBalance(ctx context.Context, balance *models.UserBalance) error {

	tx, err := d.conn.Beginx()
	if err != nil {
		return err
	}

	sqlInsert := `INSERT INTO balance (user_id, balance, withdrawn)
					VALUES ($1, $2, $3)
					ON CONFLICT (user_id) DO UPDATE 
					SET balance = balance.balance + excluded.balance - excluded.withdrawn, withdrawn = balance.withdrawn + excluded.withdrawn`

	if _, err := tx.ExecContext(ctx, sqlInsert, balance.UserID, balance.Balance, balance.Withdrawn); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DBStorage) GetBalance(ctx context.Context, balance *models.UserBalance) error {

	sqlSelect := `SELECT balance.balance, balance.withdrawn FROM balance WHERE user_id = $1`
	if err := d.conn.GetContext(ctx, balance, sqlSelect, balance.UserID); err != nil {
		return err
	}
	return nil
}

func (d *DBStorage) GetAllUserWithdrawals(ctx context.Context, userID int) ([]*models.WithdrawnResponse, error) {

	var allWithdrawals []*models.WithdrawnResponse
	sqlSelect := `SELECT id, accrual, processed_at
					FROM orders
					WHERE user_id = $1 AND status = $2 AND action = $3 
					ORDER BY processed_at`

	if err := d.conn.SelectContext(ctx, &allWithdrawals, sqlSelect, userID, models.OrderStatusProcessed, models.OrderActionWithdraw); err != nil {
		return nil, err
	}
	return allWithdrawals, nil
}
