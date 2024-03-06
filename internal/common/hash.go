package common

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

var ErrWrongPassword = errors.New("wrong password")

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf("%w", ErrWrongPassword)
	}
	return nil
}
