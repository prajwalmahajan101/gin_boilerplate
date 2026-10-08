package main

import (
	"context"
	"log/slog"
	"os"

	_ "github.com/prajwalmahajan101/gin_boilerplate/docs/swagger" // registers the generated OpenAPI spec
	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/auth"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/items"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/httpserver"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/logger"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/middleware"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

// @title        gin_boilerplate API
// @version      0.1.0
// @description  Modular monolith boilerplate API.
// @BasePath     /api/v1
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
	tokenSvc := auth.NewTokenService(cfg)

	var readyChecks map[string]httpserver.Check
	var mods []modules.Module
	var apiKeyValidator middleware.APIKeyValidator
	if cfg.DatabaseURL != "" {
		pool, perr := store.NewPool(ctx, cfg)
		if perr != nil {
			return perr
		}
		defer store.Close(pool)
		readyChecks = map[string]httpserver.Check{"postgres": pool.Ping}
		apiKeySvc := auth.NewAPIKeyService(pool, cfg.APIKeyHashPepper)
		apiKeyValidator = apiKeySvc.Validate
		mods = append(mods,
			auth.NewHandler(auth.NewService(pool, tokenSvc), apiKeySvc),
			items.NewHandler(items.NewService(pool)),
		)
	}

	vk, vkErr := valkey.New(cfg)
	if vkErr != nil {
		return vkErr
	}
	if vk != nil {
		defer vk.Close()
		if readyChecks == nil {
			readyChecks = make(map[string]httpserver.Check)
		}
		readyChecks["valkey"] = vk.Ping
	}

	r, err := httpserver.NewRouter(httpserver.RouterConfig{
		Cfg:             cfg,
		Logger:          log,
		ReadyChecks:     readyChecks,
		Modules:         mods,
		TokenParser:     tokenSvc.AccessParser(),
		APIKeyValidator: apiKeyValidator,
		Valkey:          vk,
	})
	if err != nil {
		return err
	}

	srv := httpserver.New(cfg, r)
	return srv.RunWithGracefulShutdown(ctx)
}
