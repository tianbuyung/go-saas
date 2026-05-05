package iam

import (
	"errors"
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

func (j *JWT) GenerateToken(publicID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": publicID,
		"exp": time.Now().Add(j.expire).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *JWT) ParseToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return j.secret, nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}

	publicID, ok := claims["sub"].(string)
	if !ok || publicID == "" {
		return "", errors.New("invalid sub claim")
	}

	return publicID, nil
}
