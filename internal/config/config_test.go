package config_test

import (
	"os"
	"testing"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
	_ = os.Setenv(key, value)
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "local", cfg.Env)
	assert.Equal(t, "", cfg.DatabaseURL)
	assert.Equal(t, 0, cfg.DBMaxConns)
	assert.Equal(t, 5000, cfg.DBConnTimeoutMS)
	assert.Equal(t, 2000, cfg.DBQueryTimeoutMS)
	assert.Equal(t, "", cfg.ValkeyURL)
	assert.Equal(t, "INFO", cfg.LogLevel)
	assert.True(t, cfg.LogJSON)
	assert.Equal(t, int64(1048576), cfg.MaxBodyBytes)
	assert.Empty(t, cfg.CORSOrigins)
	assert.Equal(t, 3, cfg.RetryMax)
	assert.Equal(t, 200, cfg.RetryBaseMS)
	assert.Equal(t, 5, cfg.BreakerFailThreshold)
	assert.Equal(t, 30, cfg.BreakerRecoveryS)
	assert.Equal(t, 15, cfg.AuthTokenTTLMin)
	assert.Equal(t, 168, cfg.RefreshTokenTTLH)
	assert.Equal(t, 120, cfg.RateLimitRPM)
	assert.Equal(t, 10, cfg.LoginRateLimitRPM)
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	setEnv(t, "PORT", "9090")
	setEnv(t, "ENV", "prod")
	setEnv(t, "DATABASE_URL", "postgres://x")
	setEnv(t, "LOG_LEVEL", "DEBUG")
	setEnv(t, "LOG_JSON", "false")
	setEnv(t, "CORS_ORIGINS", "https://a.com,https://b.com")
	setEnv(t, "DB_MAX_CONNS", "16")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "prod", cfg.Env)
	assert.Equal(t, "postgres://x", cfg.DatabaseURL)
	assert.Equal(t, "DEBUG", cfg.LogLevel)
	assert.False(t, cfg.LogJSON)
	assert.Equal(t, []string{"https://a.com", "https://b.com"}, cfg.CORSOrigins)
	assert.Equal(t, 16, cfg.DBMaxConns)
}
