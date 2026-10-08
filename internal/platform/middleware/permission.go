package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

type PermissionChecker func(ctx context.Context, userID int64, resource, action string) (bool, error)

func RequirePermission(check PermissionChecker, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := reqcontext.AuthFromContext(c.Request.Context())
		if !ok {
			response.Error(c, apperr.Unauthorized("not authenticated"))
			c.Abort()
			return
		}
		allowed, err := check(c.Request.Context(), claims.UserID, resource, action)
		if err != nil {
			response.Error(c, apperr.InternalError(err))
			c.Abort()
			return
		}
		if !allowed {
			response.Error(c, apperr.Forbidden("insufficient permissions"))
			c.Abort()
			return
		}
		c.Next()
	}
}
