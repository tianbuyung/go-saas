package main

import (
	"strings"

	"saas/internal/config"
	"saas/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log := logger.New()

	r := gin.New()
	proxies := strings.Split(cfg.TrustedProxies, ",")

	r.Use(gin.Recovery())
	r.SetTrustedProxies(proxies)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"port":   cfg.Port,
		})
	})

	log.Info("server running on :" + cfg.Port)

	r.Run(":" + cfg.Port)
}
