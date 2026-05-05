package main

import (
	"strings"

	"saas/internal/config"
	"saas/internal/db"
	"saas/internal/handler"
	"saas/internal/iam"
	"saas/internal/middleware"
	"saas/internal/service"
	"saas/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New()

	dbPoolConn := db.NewPool(cfg.DBPoolUrl)
	defer dbPoolConn.Close()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	var proxies []string
	if cfg.TrustedProxies != "" {
		proxies = strings.Split(cfg.TrustedProxies, ",")
	}
	r.SetTrustedProxies(proxies)

	r.Use(gin.Recovery())
	r.Use(middleware.ZapLogger(log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"port":   cfg.Port,
		})
	})

	queries := db.New(dbPoolConn)
	jwt := iam.NewJWT(cfg.JWTSecret, cfg.JWTExpireHours)

	iamService := service.NewIamService(queries, jwt)
	iamHandler := handler.NewIamHandler(iamService)

	r.POST("/register", iamHandler.Register)
	r.POST("/login", iamHandler.Login)

	protected := r.Group("/api")
	protected.Use(middleware.JWTAuth(jwt))

	protected.GET("/me", func(c *gin.Context) {
		userID, exists := c.Get(middleware.ContextUserIDKey)
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(200, gin.H{
			"user_id": userID,
		})
	})

	log.Info("server running",
		zap.String("port", cfg.Port),
		zap.String("env", cfg.Env),
	)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}
}
