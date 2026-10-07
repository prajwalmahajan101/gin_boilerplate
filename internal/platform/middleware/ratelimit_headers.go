package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type RateLimitResult struct {
	Limit     int
	Remaining int
	Reset     int64
}

func SetRateLimitResult(c *gin.Context, r RateLimitResult) {
	c.Set(ctxRateLimit, r)
}

func WriteRateLimitHeaders(c *gin.Context, r RateLimitResult) {
	h := c.Writer.Header()
	h.Set("X-RateLimit-Limit", strconv.Itoa(r.Limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(r.Remaining))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(r.Reset, 10))
}

func RateLimitHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if v, ok := c.Get(ctxRateLimit); ok {
			if r, ok := v.(RateLimitResult); ok {
				WriteRateLimitHeaders(c, r)
			}
		}
	}
}
