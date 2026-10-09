package httpserver

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/middleware"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/ratelimit"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

// RouterConfig carries everything NewRouter needs to assemble the engine.
type RouterConfig struct {
	Cfg               *config.Config
	Logger            *slog.Logger
	Modules           []modules.Module
	ReadyChecks       map[string]Check
	TokenParser       middleware.TokenParser       // nil = no auth enforcement
	APIKeyValidator   middleware.APIKeyValidator   // nil = API key auth disabled
	Blacklist         middleware.BlacklistChecker  // nil = no token blacklist
	UserEpoch         middleware.UserEpochChecker  // nil = no user-level session revocation
	PermissionChecker middleware.PermissionChecker // nil = no permission enforcement
	Valkey            *valkey.Client               // nil = Valkey-based rate limiting disabled
}

func buildLimiter(vk *valkey.Client) ratelimit.Limiter {
	mem := ratelimit.NewMemoryLimiter()
	if vk != nil {
		return ratelimit.WithFallback(ratelimit.NewValkeyLimiter(vk), mem)
	}
	return mem
}

// NewRouter builds the gin engine: middleware chain, health probes, and the
// three route gates (public / protected / admin) handed to each module.
func NewRouter(rc RouterConfig) (*gin.Engine, error) {
	if rc.Cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	if err := configureTrustedProxies(r, rc.Cfg); err != nil {
		return nil, err
	}

	middleware.Setup(r, rc.Cfg, rc.Logger)

	// Health probes live outside /api so they never sit behind auth.
	r.GET("/healthz", Liveness())
	r.GET("/readyz", Readiness(rc.ReadyChecks))

	// Swagger UI (spec registered by the blank-imported docs/swagger package).
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	limiter := buildLimiter(rc.Valkey)

	public := r.Group("/api/v1")
	public.Use(middleware.RateLimit(limiter, rc.Cfg.LoginRateLimitRPM, "pub"))

	protected := r.Group("/api/v1")
	protected.Use(middleware.RateLimit(limiter, rc.Cfg.RateLimitRPM, "api"))

	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.RateLimit(limiter, rc.Cfg.RateLimitRPM, "api"))

	if rc.TokenParser != nil {
		authCfg := middleware.AuthConfig{
			Parse:     rc.TokenParser,
			APIKey:    rc.APIKeyValidator,
			Blacklist: rc.Blacklist,
			UserEpoch: rc.UserEpoch,
		}
		protected.Use(middleware.Auth(authCfg))
		admin.Use(middleware.Auth(authCfg), middleware.RequireRole("admin"))
	}

	for _, m := range rc.Modules {
		m.RegisterRoutes(public, protected, admin)
	}

	return r, nil
}
