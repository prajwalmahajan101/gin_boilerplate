package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

// Server wraps http.Server with lifecycle helpers.
type Server struct {
	*http.Server
	shutdownTimeout time.Duration
}

// New builds a Server bound to cfg.Port with the configured timeouts.
func New(cfg *config.Config, handler http.Handler) *Server {
	return &Server{
		Server: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      handler,
			ReadTimeout:  time.Duration(cfg.ServerReadTimeoutS) * time.Second,
			WriteTimeout: time.Duration(cfg.ServerWriteTimeoutS) * time.Second,
			IdleTimeout:  time.Duration(cfg.ServerIdleTimeoutS) * time.Second,
		},
		shutdownTimeout: time.Duration(cfg.ShutdownTimeoutS) * time.Second,
	}
}

// RunWithGracefulShutdown serves until ctx is cancelled or SIGINT/SIGTERM
// arrives, then drains in-flight requests within the shutdown timeout.
func (s *Server) RunWithGracefulShutdown(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "http server listening", slog.String("addr", s.Addr))
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining", slog.Duration("timeout", s.shutdownTimeout))
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		return s.Shutdown(shutdownCtx)
	}
}
