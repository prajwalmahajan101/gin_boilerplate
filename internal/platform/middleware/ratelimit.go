package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/ratelimit"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

func RateLimit(limiter ratelimit.Limiter, rpm int, prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil || rpm <= 0 {
			c.Next()
			return
		}
		key := fmt.Sprintf("rl:%s:%s", prefix, c.ClientIP())
		ok, err := limiter.Allow(c.Request.Context(), key, rpm)
		if err != nil {
			c.Next()
			return
		}
		if !ok {
			response.Error(c, apperr.RateLimited("too many requests"))
			c.Abort()
			return
		}
		c.Next()
	}
}
