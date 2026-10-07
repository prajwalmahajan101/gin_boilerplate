package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			abortTooLarge(c)
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()

		if c.Writer.Written() {
			return
		}

		for _, e := range c.Errors {
			var mbe *http.MaxBytesError
			if errors.As(e.Err, &mbe) {
				abortTooLarge(c)
				return
			}
		}
	}
}

func abortTooLarge(c *gin.Context) {
	response.Error(c, apperr.PayloadTooLarge(msgBodyTooLarge))
	c.Abort()
}
