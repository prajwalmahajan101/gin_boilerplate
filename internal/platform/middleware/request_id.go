package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !requestIDPattern.MatchString(id) {
			id = uuid.NewString()
		}

		ctx := reqcontext.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}
