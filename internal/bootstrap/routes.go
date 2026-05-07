package bootstrap

import (
	"saas/internal/iam"
	"saas/internal/module/auth"
	"saas/internal/module/user"
	"saas/internal/router"
)

type Modules struct {
	Auth *auth.Module
	User *user.Module
}

func RegisterRoutes(r router.Router, m Modules, jwt *iam.JWT, bl iam.Blocklist) {
	r.GET("/health", func(c router.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	m.Auth.RegisterRoutes(r, jwt, bl)
	m.User.RegisterRoutes(r, jwt, bl)
}
