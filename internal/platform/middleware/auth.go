package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

type TokenParser func(tokenStr string) (userID int64, role string, err error)

func Auth(parse TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		hdr := c.GetHeader("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			response.Error(c, apperr.Unauthorized("missing bearer token"))
			c.Abort()
			return
		}
		uid, role, err := parse(hdr[7:])
		if err != nil {
			response.Error(c, apperr.Unauthorized("invalid token"))
			c.Abort()
			return
		}
		ctx := reqcontext.WithAuth(c.Request.Context(), reqcontext.AuthClaims{
			UserID: uid,
			Role:   role,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
