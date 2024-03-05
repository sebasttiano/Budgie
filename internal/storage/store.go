package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"github.com/sebasttiano/Budgie/internal/models"
	"go.uber.org/zap"
	"time"
)

type Store interface {
	UserExists(ctx context.Context, login string) (bool, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
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

func (d *DBStorage) GetUserByID(ctx context.Context, id int) (*models.User, error) {

	u := &models.User{}
	//sqlSelect := `SELECT id, login, password, registered_at FROM users WHERE id = $1`
	//
	//if err := p.conn.GetContext(ctx, metric, sqlSelect, metric.Name); err != nil {
	//	if errors.Is(err, sql.ErrNoRows) {
	//		return metric, p.Errors.ErrNoRows
	//	} else {
	//		return nil, err
	//	}
	//}
	return u, nil
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
