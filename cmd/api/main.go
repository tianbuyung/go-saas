package main

import (
	"context"
	"strings"

	"saas/internal/config"
	"saas/internal/db"
	"saas/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New()
	dbPoolConn := db.NewPool(cfg.DBPoolUrl)

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

	queries := db.New(dbPoolConn)

	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email: "test@mail.com",
		Password: pgtype.Text{
			String: "123", Valid: true,
		},
		Provider: pgtype.Text{
			Valid: false,
		},
		ProviderID: pgtype.Text{
			Valid: false,
		},
	})

	if err != nil {
		log.Fatal("failed to create user", zap.Error(err))
	}

	log.Info("user created", zap.Any("user", user))

	r.Run(":" + cfg.Port)
}
