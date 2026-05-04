package config

import "github.com/spf13/viper"

type Config struct {
	Port           string
	TrustedProxies string
	DBUrl          string
	DBPoolUrl      string
}

func Load() Config {
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	return Config{
		Port:           viper.GetString("PORT"),
		TrustedProxies: viper.GetString("TRUSTED_PROXIES"),
		DBUrl:          viper.GetString("DATABASE_URL"),
		DBPoolUrl:      viper.GetString("DATABASE_POOL_URL"),
	}
}
