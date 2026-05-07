package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env                string
	Port               string
	TrustedProxies     string
	DBUrl              string
	DBPoolUrl          string
	JWTSecret          string
	JWTExpire          time.Duration
	JWTIssuer          string
	JWTAudience        string
	JWTAlgorithm       string
	SessionStore       string
	RedisURL           string
	RefreshExpireHours int
}

func Load() Config {
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	return Config{
		Env:                viper.GetString("ENVIRONMENT"),
		Port:               viper.GetString("PORT"),
		TrustedProxies:     viper.GetString("TRUSTED_PROXIES"),
		DBUrl:              viper.GetString("DATABASE_URL"),
		DBPoolUrl:          viper.GetString("DATABASE_POOL_URL"),
		JWTSecret:          viper.GetString("JWT_SECRET"),
		JWTExpire:          viper.GetDuration("JWT_EXPIRE"),
		JWTIssuer:          viper.GetString("JWT_TOKEN_ISSUER"),
		JWTAudience:        viper.GetString("JWT_TOKEN_AUDIENCE"),
		JWTAlgorithm:       viper.GetString("JWT_ALGORITHM"),
		SessionStore:       viper.GetString("SESSION_STORE"),
		RedisURL:           viper.GetString("REDIS_URL"),
		RefreshExpireHours: viper.GetInt("REFRESH_EXPIRE_HOURS"),
	}
}
