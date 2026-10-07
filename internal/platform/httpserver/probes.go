package httpserver

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

// Check reports whether a dependency is healthy. Returns nil when ready.
type Check func(ctx context.Context) error

// Liveness reports that the process is up. It never depends on external state.
func Liveness() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, http.StatusOK, "alive", nil)
	}
}

// Readiness runs each named check against the request context. All pass -> 200;
// any fail -> 503 with the per-dependency status. An empty map is trivially ready;
// real dependency checks (postgres, valkey) are injected in later milestones.
func Readiness(checks map[string]Check) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := make(map[string]string, len(checks))
		ready := true
		for name, check := range checks {
			if err := check(c.Request.Context()); err != nil {
				results[name] = err.Error()
				ready = false
				continue
			}
			results[name] = "ok"
		}

		if ready {
			response.Success(c, http.StatusOK, "ready", gin.H{"checks": results})
			return
		}

		c.JSON(http.StatusServiceUnavailable, response.Envelope{
			Success:   false,
			Message:   "not ready",
			Data:      gin.H{"checks": results},
			Errors:    []response.ErrDetail{{Code: apperr.CodeServiceUnavail, Message: "not ready"}},
			RequestID: reqcontext.RequestIDFromContext(c.Request.Context()),
		})
	}
}
