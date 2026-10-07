package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
)

type timingWriter struct {
	gin.ResponseWriter
	ctx     context.Context
	start   time.Time
	stamped bool
}

func (w *timingWriter) stamp() {
	if w.stamped {
		return
	}
	w.stamped = true
	serviceMS, repoMS := reqcontext.TimingFromContext(w.ctx)
	handlerMS := time.Since(w.start).Milliseconds()
	w.Header().Set("Server-Timing", fmt.Sprintf(
		"handler;dur=%d, service;dur=%d, repo;dur=%d",
		handlerMS, serviceMS, repoMS,
	))
}

func (w *timingWriter) WriteHeader(code int) {
	w.stamp()
	w.ResponseWriter.WriteHeader(code)
}

func (w *timingWriter) Write(b []byte) (int, error) {
	w.stamp()
	return w.ResponseWriter.Write(b)
}

func (w *timingWriter) WriteString(s string) (int, error) {
	w.stamp()
	return w.ResponseWriter.WriteString(s)
}

func RequestLogging(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := reqcontext.WithTiming(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Writer = &timingWriter{ResponseWriter: c.Writer, ctx: ctx, start: start}
		c.Next()
		handlerMS := time.Since(start).Milliseconds()

		serviceMS, repoMS := reqcontext.TimingFromContext(ctx)

		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"handler_ms", handlerMS,
			"service_ms", serviceMS,
			"repo_ms", repoMS,
			"request_id", reqcontext.RequestIDFromContext(ctx),
			"client_ip", c.ClientIP(),
		)
	}
}
