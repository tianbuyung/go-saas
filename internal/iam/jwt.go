package iam

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	secret []byte
	expire time.Duration
}

func NewJWT(secret string, expireHours int) *JWT {
	return &JWT{
		secret: []byte(secret),
		expire: time.Duration(expireHours) * time.Hour,
	}
}

func (j *JWT) GenerateToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(j.expire).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}
