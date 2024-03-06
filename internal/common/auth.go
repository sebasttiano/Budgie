package common

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"strconv"
	"time"
)

var ErrTokenCreationFailed = errors.New("token creation failed")

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	ID int `json:"id"`
}

const TokenExp = time.Hour * 3

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(id int, secretKey string) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
			Subject:   strconv.Itoa(id),
			Issuer:    "localhost:8080/api/user/login",
			Audience:  []string{"localhost:8080"},
		},
		ID: id,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenCreationFailed, err)
	}

	// возвращаем строку токена
	return tokenString, nil
}
