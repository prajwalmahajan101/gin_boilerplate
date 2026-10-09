package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	_ "github.com/prajwalmahajan101/gin_boilerplate/docs/swagger" // registers the generated OpenAPI spec
	awshelper "github.com/prajwalmahajan101/gin_boilerplate/internal/aws"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/auth"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/items"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/cache"
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

	tokenSvc := auth.NewTokenService(cfg)

	vk, vkErr := valkey.New(cfg)
	if vkErr != nil {
		return vkErr
	}
	if vk != nil {
		defer vk.Close()
	}

	blacklist := auth.NewBlacklist(vk)

	// Password-reset: Valkey holds the single-use tokens and the per-user
	// revocation epoch; SES (if configured) delivers the token email.
	var resetStore *auth.ResetStore
	if vk != nil {
		resetStore = auth.NewResetStore(vk)
	}
	var emailSender auth.EmailSender
	if cfg.SESFromEmail != "" {
		awsCfg, awsErr := awshelper.Config(ctx, cfg.AWSRegion)
		if awsErr != nil {
			return awsErr
		}
		emailSender = awshelper.NewSES(awsCfg, cfg.SESFromEmail)
	}
	resetDeps := auth.ResetDeps{
		Sender:  emailSender,
		TTL:     time.Duration(cfg.PasswordResetTTLMin) * time.Minute,
		URLBase: cfg.PasswordResetURLBase,
	}
	if resetStore != nil { // avoid a typed-nil interface (would defeat the nil check)
		resetDeps.Store = resetStore
	}

	readyChecks := make(map[string]httpserver.Check)
	if vk != nil {
		readyChecks["valkey"] = vk.Ping
	}

	var mods []modules.Module
	var apiKeyValidator middleware.APIKeyValidator
	if cfg.DatabaseURL != "" {
		pool, perr := store.NewPool(ctx, cfg)
		if perr != nil {
			return perr
		}
		defer store.Close(pool)
		readyChecks["postgres"] = pool.Ping
		apiKeySvc := auth.NewAPIKeyService(pool, cfg.APIKeyHashPepper)
		apiKeyValidator = apiKeySvc.Validate
		rbacSvc := auth.NewRBACService(pool)
		itemCache := cache.NewTiered("items", vk.Raw(), cache.TierConfig{
			L1Enabled:            cfg.CacheL1Enabled,
			L1Max:                cfg.CacheL1Max,
			L1TTL:                time.Duration(cfg.CacheL1TTLS) * time.Second,
			BreakerFailThreshold: cfg.CacheBreakerFailThreshold,
			BreakerRecovery:      time.Duration(cfg.CacheBreakerRecoveryS) * time.Second,
			TTLJitterPct:         cfg.CacheTTLJitterPct,
		})
		itemSvc := items.NewService(pool, itemCache,
			time.Duration(cfg.CacheItemTTLS)*time.Second,
			time.Duration(cfg.CacheNegTTLS)*time.Second)
		mods = append(mods,
			auth.NewHandler(auth.NewService(pool, tokenSvc, blacklist, resetDeps), apiKeySvc, rbacSvc, tokenSvc),
			items.NewHandler(itemSvc),
		)
	}

	r, err := httpserver.NewRouter(httpserver.RouterConfig{
		Cfg:             cfg,
		Logger:          log,
		ReadyChecks:     readyChecks,
		Modules:         mods,
		TokenParser:     tokenSvc.AccessParser(),
		APIKeyValidator: apiKeyValidator,
		Blacklist:       blacklist.IsBlacklisted,
		UserEpoch:       userEpochChecker(resetStore),
		Valkey:          vk,
	})
	if err != nil {
		return err
	}

	srv := httpserver.New(cfg, r)
	return srv.RunWithGracefulShutdown(ctx)
}

// userEpochChecker adapts the reset store's Epoch lookup to the middleware hook,
// returning nil when there is no store so session revocation is simply skipped.
func userEpochChecker(rs *auth.ResetStore) middleware.UserEpochChecker {
	if rs == nil {
		return nil
	}
	return rs.Epoch
}
