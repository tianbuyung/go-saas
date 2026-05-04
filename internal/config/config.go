package config

import "github.com/spf13/viper"

type Config struct {
	Env            string
	Port           string
	TrustedProxies string
	DBUrl          string
	DBPoolUrl      string
	JWTSecret      string
	JWTExpireHours int
}

func Load() Config {
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	return Config{
		Env:            viper.GetString("ENVIRONMENT"),
		Port:           viper.GetString("PORT"),
		TrustedProxies: viper.GetString("TRUSTED_PROXIES"),
		DBUrl:          viper.GetString("DATABASE_URL"),
		DBPoolUrl:      viper.GetString("DATABASE_POOL_URL"),
		JWTSecret:      viper.GetString("JWT_SECRET"),
		JWTExpireHours: viper.GetInt("JWT_EXPIRE_HOURS"),
	}
}
