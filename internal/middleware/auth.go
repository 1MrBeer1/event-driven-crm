package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/1MrBeer1/event-driven-crm/internal/auth"
)

const (
	ContextUserID = "user_id"
	ContextEmail  = "email"
	ContextRole   = "role"
)

func AuthRequired(manager *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || token == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims, err := manager.Validate(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRole, claims.Role)
		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Next()
	}
}
