package middleware

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}

			if isBrokenPipe(r) {
				slog.ErrorContext(c.Request.Context(), "panic: broken pipe",
					"error", r,
					"path", c.Request.URL.Path,
				)
				c.Abort()
				return
			}

			slog.ErrorContext(c.Request.Context(), "panic recovered",
				"error", r,
				"path", c.Request.URL.Path,
				"stack", string(debug.Stack()),
			)
			response.Error(c, apperr.New(apperr.CodeInternalError, msgInternalServer, http.StatusInternalServerError))
			c.Abort()
		}()
		c.Next()
	}
}

func isBrokenPipe(r any) bool {
	err, ok := r.(error)
	if !ok {
		return false
	}
	var ne *net.OpError
	if !errors.As(err, &ne) {
		return false
	}
	var se *os.SyscallError
	if !errors.As(ne, &se) {
		return false
	}
	msg := strings.ToLower(se.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
}
