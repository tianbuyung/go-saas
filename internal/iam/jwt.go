package iam

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	Secret    string
	Expire    time.Duration
	Issuer    string
	Audience  string
	Algorithm string
}

type TokenClaims struct {
	PublicID  string
	JTI       string
	ExpiresAt time.Time
}

type JWT struct {
	secret    []byte
	expire    time.Duration
	issuer    string
	audience  string
	algorithm jwt.SigningMethod
}

func NewJWT(cfg JWTConfig) *JWT {
	return &JWT{
		secret:    []byte(cfg.Secret),
		expire:    cfg.Expire,
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
		algorithm: hmacMethod(cfg.Algorithm),
	}
}

// hmacMethod maps an algorithm name to its HMAC signing method.
// Defaults to HS256 for unrecognised values.
func hmacMethod(alg string) jwt.SigningMethod {
	switch alg {
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	default:
		return jwt.SigningMethodHS256
	}
}

func (j *JWT) GenerateToken(publicID string) (string, error) {
	jti, err := GenerateOpaqueToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub": publicID,
		"jti": jti,
		"iat": now.Unix(),
		"exp": now.Add(j.expire).Unix(),
	}
	if j.issuer != "" {
		claims["iss"] = j.issuer
	}
	if j.audience != "" {
		claims["aud"] = j.audience
	}

	token := jwt.NewWithClaims(j.algorithm, claims)
	return token.SignedString(j.secret)
}

func (j *JWT) ParseToken(tokenStr string) (*TokenClaims, error) {
	opts := []jwt.ParserOption{}
	if j.issuer != "" {
		opts = append(opts, jwt.WithIssuer(j.issuer))
	}
	if j.audience != "" {
		opts = append(opts, jwt.WithAudience(j.audience))
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if t.Method != j.algorithm {
			return nil, errors.New("invalid signing method")
		}
		return j.secret, nil
	}, opts...)

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	publicID, ok := claims["sub"].(string)
	if !ok || publicID == "" {
		return nil, errors.New("invalid sub claim")
	}

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return nil, errors.New("missing jti claim")
	}

	var expiresAt time.Time
	if exp, ok := claims["exp"].(float64); ok {
		expiresAt = time.Unix(int64(exp), 0)
	}

	return &TokenClaims{
		PublicID:  publicID,
		JTI:       jti,
		ExpiresAt: expiresAt,
	}, nil
}
