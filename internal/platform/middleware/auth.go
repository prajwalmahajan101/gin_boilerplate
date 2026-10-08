package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

type TokenParser func(tokenStr string) (userID int64, role string, err error)
type APIKeyValidator func(ctx context.Context, key string) (userID int64, role string, err error)

func Auth(parse TokenParser, apiKeyValidators ...APIKeyValidator) gin.HandlerFunc {
	var validateKey APIKeyValidator
	if len(apiKeyValidators) > 0 {
		validateKey = apiKeyValidators[0]
	}
	return func(c *gin.Context) {
		var uid int64
		var role string
		var err error

		hdr := c.GetHeader("Authorization")
		switch {
		case strings.HasPrefix(hdr, "Bearer "):
			uid, role, err = parse(hdr[7:])
		case validateKey != nil && c.GetHeader("X-API-Key") != "":
			uid, role, err = validateKey(c.Request.Context(), c.GetHeader("X-API-Key"))
		default:
			response.Error(c, apperr.Unauthorized("missing bearer token or api key"))
			c.Abort()
			return
		}

		if err != nil {
			response.Error(c, apperr.Unauthorized("invalid credentials"))
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
