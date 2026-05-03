package config

import "github.com/spf13/viper"

type Config struct {
	Port           string
	TrustedProxies string
}

func Load() Config {
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	return Config{
		Port:           viper.GetString("PORT"),
		TrustedProxies: viper.GetString("TRUSTED_PROXIES"),
	}
}
