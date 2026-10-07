package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/httpserver"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/logger"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("server exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.Setup()
	log.InfoContext(ctx, "starting server", slog.Any("config", cfg))

	r, err := httpserver.NewRouter(httpserver.RouterConfig{
		Cfg:    cfg,
		Logger: log,
		// Modules and ReadyChecks are wired as later milestones land
		// (items module M5; postgres M4 / valkey M6 readiness checks).
	})
	if err != nil {
		return err
	}

	srv := httpserver.New(cfg, r)
	return srv.RunWithGracefulShutdown(ctx)
}
