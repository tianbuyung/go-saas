package bootstrap

import (
	"saas/internal/config"
	"saas/internal/db"
	"saas/internal/iam"
	"saas/internal/module/auth"
	"saas/internal/module/user"
	"saas/internal/router"
	"saas/pkg/logger"
)

type App struct {
	router router.Router
	port   string
}

func NewApp() *App {
	cfg := config.Load()
	log := logger.New()

	// DB
	dbPool := db.NewPool(cfg.DBPoolUrl)
	queries := db.New(dbPool)
	txManager := db.NewTxManager(dbPool)

	// IAM
	jwt := iam.NewJWT(cfg.JWTSecret, cfg.JWTExpireHours)

	// Modules
	authModule := auth.New(queries, txManager, jwt)
	userModule := user.New(queries)

	// Router
	r := router.NewGinEngine(&cfg, log)

	// Register routes
	RegisterRoutes(r, Modules{
		Auth: authModule,
		User: userModule,
	}, jwt)

	return &App{router: r, port: cfg.Port}
}

func (a *App) Run() {
	a.router.Run(":" + a.port)
}
