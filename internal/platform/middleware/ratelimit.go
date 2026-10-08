package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

func RateLimit(vk *valkey.Client, rpm int, prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if vk == nil || rpm <= 0 {
			c.Next()
			return
		}
		key := fmt.Sprintf("rl:%s:%s", prefix, c.ClientIP())
		ctx := c.Request.Context()
		count, err := vk.Incr(ctx, key)
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			_ = vk.Expire(ctx, key, time.Minute)
		}
		if count > int64(rpm) {
			response.Error(c, apperr.RateLimited("too many requests"))
			c.Abort()
			return
		}
		c.Next()
	}
}
