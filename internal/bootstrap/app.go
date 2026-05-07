package bootstrap

import (
	"saas/internal/config"
	"saas/internal/db"
	"saas/internal/iam"
	"saas/internal/module/auth"
	"saas/internal/module/user"
	"saas/internal/repository"
	"saas/internal/router"
	"saas/pkg/cache"
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
	jwt := iam.NewJWT(iam.JWTConfig{
		Secret:    cfg.JWTSecret,
		Expire:    cfg.JWTExpire,
		Issuer:    cfg.JWTIssuer,
		Audience:  cfg.JWTAudience,
		Algorithm: cfg.JWTAlgorithm,
	})

	// Redis — single shared client; nil when REDIS_URL is unset
	var redisClient *cache.Redis
	if cfg.RedisURL != "" {
		redisClient = cache.NewRedis(cfg.RedisURL)
	}

	// SessionStore — toggle via SESSION_STORE=db|redis|hybrid
	// db:     DB only
	// redis:  Redis only (no DB fallback)
	// hybrid: Redis as write-through cache; miss → DB fallback → re-populate Redis
	var sessionStore iam.SessionStore
	switch cfg.SessionStore {
	case "redis":
		sessionStore = iam.NewRedisSessionStore(redisClient, log)
	case "hybrid":
		sessionStore = iam.NewHybridSessionStore(
			repository.NewDBSessionStore(queries),
			iam.NewRedisSessionStore(redisClient, log),
			log,
		)
	default:
		sessionStore = repository.NewDBSessionStore(queries)
	}

	// Blocklist — Redis-backed when available; no-op (fail-open) otherwise
	var blocklist iam.Blocklist
	if redisClient != nil {
		blocklist = iam.NewRedisBlocklist(redisClient)
	} else {
		blocklist = iam.NewNoopBlocklist()
		log.Warn("blocklist disabled: REDIS_URL not set; JWT revocation on logout is a no-op")
	}

	// Modules
	authModule := auth.New(queries, txManager, jwt, sessionStore, blocklist, log, cfg.RefreshExpireHours)
	userModule := user.New(queries)

	// Router
	r := router.NewGinEngine(&cfg)
	r.Use(router.ZapLogger(log))

	// Register routes
	RegisterRoutes(r, Modules{
		Auth: authModule,
		User: userModule,
	}, jwt, blocklist)

	return &App{router: r, port: cfg.Port}
}

func (a *App) Run() {
	a.router.Run(":" + a.port)
}
