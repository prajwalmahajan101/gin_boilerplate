package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

const (
	msgRateLimitExceeded = "rate limit exceeded"
	msgInternalServer    = "internal server error"
	msgBodyTooLarge      = "request body too large"
)

const (
	requestIDHeader = "X-Request-ID"
	ctxRateLimit    = "rate_limit_result"
)

// Setup installs the global middleware chain in required order.
// Auth + rate-limit added when their modules land.
func Setup(r *gin.Engine, cfg *config.Config, logger *slog.Logger) {
	r.Use(Recovery())
	r.Use(BodyLimit(cfg.MaxBodyBytes))
	r.Use(CORS(cfg.CORSOrigins))
	r.Use(SecurityHeaders(cfg.Env))
	r.Use(RequestID())
	r.Use(RequestLogging(logger))
	r.Use(RateLimitHeaders())
	r.Use(response.ErrorHandler())
}
