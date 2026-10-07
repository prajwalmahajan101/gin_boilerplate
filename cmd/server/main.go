package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/httpserver"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/logger"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
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

	// Postgres is optional: a set DATABASE_URL wires the pool and its readiness
	// check; an empty one keeps the server bootable without a database.
	var readyChecks map[string]httpserver.Check
	if cfg.DatabaseURL != "" {
		pool, perr := store.NewPool(ctx, cfg)
		if perr != nil {
			return perr
		}
		defer store.Close(pool)
		readyChecks = map[string]httpserver.Check{"postgres": pool.Ping}
	}

	r, err := httpserver.NewRouter(httpserver.RouterConfig{
		Cfg:         cfg,
		Logger:      log,
		ReadyChecks: readyChecks,
		// Modules are registered when the items module lands (M5).
	})
	if err != nil {
		return err
	}

	srv := httpserver.New(cfg, r)
	return srv.RunWithGracefulShutdown(ctx)
}
