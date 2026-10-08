package httpserver

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/middleware"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

// RouterConfig carries everything NewRouter needs to assemble the engine.
type RouterConfig struct {
	Cfg             *config.Config
	Logger          *slog.Logger
	Modules         []modules.Module
	ReadyChecks     map[string]Check
	TokenParser     middleware.TokenParser     // nil = no auth enforcement
	APIKeyValidator middleware.APIKeyValidator // nil = API key auth disabled
	Valkey          *valkey.Client             // nil = rate limiting disabled
}

// NewRouter builds the gin engine: middleware chain, health probes, and the
// three route gates (public / protected / admin) handed to each module.
// Auth and rate-limit gating attach to protected/admin when the auth module lands.
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

	public := r.Group("/api/v1")
	public.Use(middleware.RateLimit(rc.Valkey, rc.Cfg.LoginRateLimitRPM, "pub"))

	protected := r.Group("/api/v1")
	protected.Use(middleware.RateLimit(rc.Valkey, rc.Cfg.RateLimitRPM, "api"))

	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.RateLimit(rc.Valkey, rc.Cfg.RateLimitRPM, "api"))

	if rc.TokenParser != nil {
		var authOpts []middleware.APIKeyValidator
		if rc.APIKeyValidator != nil {
			authOpts = append(authOpts, rc.APIKeyValidator)
		}
		protected.Use(middleware.Auth(rc.TokenParser, authOpts...))
		admin.Use(middleware.Auth(rc.TokenParser, authOpts...), middleware.RequireRole("admin"))
	}

	for _, m := range rc.Modules {
		m.RegisterRoutes(public, protected, admin)
	}

	return r, nil
}
