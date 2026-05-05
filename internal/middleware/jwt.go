package middleware

import (
	"net/http"
	"strings"

	"saas/internal/iam"

	"github.com/gin-gonic/gin"
)

const ContextUserIDKey = "user_id"

func JWTAuth(jwt *iam.JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		// Expect: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization format",
			})
			return
		}

		userID, err := jwt.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		// Inject into context
		c.Set(ContextUserIDKey, userID)

		c.Next()
	}
}
