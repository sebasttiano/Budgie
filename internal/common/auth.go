package common

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"strconv"
	"time"
)

var ErrTokenCreationFailed = errors.New("token creation failed")

// Claims — struct with standart and customs claims
type Claims struct {
	jwt.RegisteredClaims
	ID int `json:"id"`
}

const TokenExp = time.Hour * 3

// BuildJWTString creates token and returns it via string.
func BuildJWTString(id int, secretKey string) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
			Subject:   strconv.Itoa(id),
			Issuer:    "localhost:8080/api/user/login",
			Audience:  []string{"localhost:8080"},
		},
		ID: id,
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenCreationFailed, err)
	}

	return tokenString, nil
}
