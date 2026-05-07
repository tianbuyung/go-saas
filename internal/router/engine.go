package router

import (
	"context"
	"net/http"
	"strings"

	"saas/internal/config"
	"saas/internal/iam"

	"github.com/gin-gonic/gin"
)

// ===== Context Adapter =====

type ginContext struct {
	ctx *gin.Context
}

func (c *ginContext) JSON(code int, data any) {
	c.ctx.JSON(code, data)
}

func (c *ginContext) Param(key string) string {
	return c.ctx.Param(key)
}

func (c *ginContext) Query(key string) string {
	return c.ctx.Query(key)
}

func (c *ginContext) BindJSON(obj any) error {
	return c.ctx.ShouldBindJSON(obj)
}

func (c *ginContext) Set(key string, value any) {
	c.ctx.Set(key, value)
}

func (c *ginContext) Get(key string) (any, bool) {
	return c.ctx.Get(key)
}

func (c *ginContext) GetHeader(key string) string {
	return c.ctx.GetHeader(key)
}

func (c *ginContext) Context() context.Context {
	return c.ctx.Request.Context()
}

// ===== Router Adapter =====

type ginRouter interface {
	GET(string, ...gin.HandlerFunc) gin.IRoutes
	POST(string, ...gin.HandlerFunc) gin.IRoutes
	Group(string, ...gin.HandlerFunc) *gin.RouterGroup
	Use(...gin.HandlerFunc) gin.IRoutes
}

type GinEngine struct {
	router ginRouter
	root   *gin.Engine
}

func NewGinEngine(cfg *config.Config) Router {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	e := gin.New()

	var proxies []string
	if cfg.TrustedProxies != "" {
		proxies = strings.Split(cfg.TrustedProxies, ",")
	}
	e.SetTrustedProxies(proxies)

	e.Use(gin.Recovery())

	return &GinEngine{router: e, root: e}
}

// ===== Router methods =====

func (g *GinEngine) GET(path string, h HandlerFunc) {
	g.router.GET(path, g.wrap(h))
}

func (g *GinEngine) POST(path string, h HandlerFunc) {
	g.router.POST(path, g.wrap(h))
}

func (g *GinEngine) Group(prefix string) Router {
	group := g.router.Group(prefix)
	return &GinEngine{router: group, root: g.root}
}

func (g *GinEngine) Use(mw ...MiddlewareFunc) {
	for _, m := range mw {
		g.router.Use(g.wrapMiddleware(m))
	}
}

func (g *GinEngine) Run(addr string) error {
	return g.root.Run(addr)
}

// ===== Helpers =====

func (g *GinEngine) wrap(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(&ginContext{ctx: c})
	}
}

func (g *GinEngine) wrapMiddleware(mw MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &ginContext{ctx: c}
		next := HandlerFunc(func(_ Context) { c.Next() })
		mw(next)(ctx)
	}
}

// ===== Middleware constructors =====

// NewJWTMiddleware returns a framework-agnostic JWT auth middleware.
// Validates Bearer token, checks the revocation blocklist, and injects
// user_id, jti, and token_expires_at into context on success.
func NewJWTMiddleware(j *iam.JWT, bl iam.Blocklist) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
				return
			}

			claims, err := j.ParseToken(parts[1])
			if err != nil {
				c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}

			revoked, err := bl.IsRevoked(c.Context(), claims.JTI)
			if err != nil || revoked {
				c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}

			c.Set(ContextUserIDKey, claims.PublicID)
			c.Set(ContextJTIKey, claims.JTI)
			c.Set(ContextTokenExpiresAtKey, claims.ExpiresAt)
			next(c)
		}
	}
}
