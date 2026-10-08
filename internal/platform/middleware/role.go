package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		claims, ok := reqcontext.AuthFromContext(c.Request.Context())
		if !ok {
			response.Error(c, apperr.Unauthorized("not authenticated"))
			c.Abort()
			return
		}
		if _, ok := allowed[claims.Role]; !ok {
			response.Error(c, apperr.Forbidden("insufficient role"))
			c.Abort()
			return
		}
		c.Next()
	}
}
