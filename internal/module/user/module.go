package user

import (
	"saas/internal/db"
	"saas/internal/handler"
	"saas/internal/iam"
	"saas/internal/repository"
	"saas/internal/router"
	"saas/internal/service"
)

type Module struct {
	UserHandler *handler.UserHandler
}

func New(q *db.Queries) *Module {
	userRepo := repository.NewUserRepository(q)

	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	return &Module{
		UserHandler: userHandler,
	}
}

func (m *Module) RegisterRoutes(r router.Router, jwt *iam.JWT, bl iam.Blocklist) {
	api := r.Group("/api")
	api.Use(router.NewJWTMiddleware(jwt, bl))

	api.GET("/me", m.UserHandler.Me)
}
