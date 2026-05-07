package auth

import (
	"saas/internal/db"
	"saas/internal/handler"
	"saas/internal/iam"
	"saas/internal/repository"
	"saas/internal/router"
	"saas/internal/service"

	"go.uber.org/zap"
)

type Module struct {
	IamHandler *handler.IamHandler
}

func New(q *db.Queries, tx db.TxManager, jwt *iam.JWT, ss iam.SessionStore, bl iam.Blocklist, log *zap.Logger, refreshExpireHours int) *Module {
	userRepo := repository.NewUserRepository(q)
	accountRepo := repository.NewAccountRepository(q)
	iamService := service.NewIamService(userRepo, accountRepo, tx, jwt, ss, bl, log, refreshExpireHours)
	iamHandler := handler.NewIamHandler(iamService)

	return &Module{
		IamHandler: iamHandler,
	}
}

func (m *Module) RegisterRoutes(r router.Router, jwt *iam.JWT, bl iam.Blocklist) {
	authGroup := r.Group("/auth")
	authGroup.POST("/register", m.IamHandler.Register)
	authGroup.POST("/login", m.IamHandler.Login)
	authGroup.POST("/refresh", m.IamHandler.Refresh)

	protected := r.Group("/auth")
	protected.Use(router.NewJWTMiddleware(jwt, bl))
	protected.POST("/logout", m.IamHandler.Logout)
}
