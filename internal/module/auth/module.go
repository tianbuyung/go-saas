package auth

import (
	"saas/internal/db"
	"saas/internal/handler"
	"saas/internal/iam"
	"saas/internal/router"
	"saas/internal/service"
)

type Module struct {
	IamHandler *handler.IamHandler
}

func New(q *db.Queries, jwt *iam.JWT) *Module {
	iamService := service.NewIamService(q, jwt)
	iamHandler := handler.NewIamHandler(iamService)

	return &Module{
		IamHandler: iamHandler,
	}
}

func (m *Module) RegisterRoutes(r router.Router) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", m.IamHandler.Register)
		authGroup.POST("/login", m.IamHandler.Login)
	}
}
