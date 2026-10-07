package config

import (
	"context"
	"log/slog"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const redacted = "***REDACTED***"

type Config struct {
	// Server
	Port                string   `env:"PORT" envDefault:"8080"`
	Env                 string   `env:"ENV" envDefault:"local"`
	ServerReadTimeoutS  int      `env:"SERVER_READ_TIMEOUT_S" envDefault:"15"`
	ServerWriteTimeoutS int      `env:"SERVER_WRITE_TIMEOUT_S" envDefault:"15"`
	ServerIdleTimeoutS  int      `env:"SERVER_IDLE_TIMEOUT_S" envDefault:"60"`
	ShutdownTimeoutS    int      `env:"SHUTDOWN_TIMEOUT_S" envDefault:"10"`
	TrustedProxies      []string `env:"TRUSTED_PROXIES" envSeparator:","`

	// Database
	DatabaseURL      string `env:"DATABASE_URL" envDefault:""`
	DBMaxConns       int    `env:"DB_MAX_CONNS" envDefault:"0"`
	DBConnTimeoutMS  int    `env:"DB_CONN_TIMEOUT_MS" envDefault:"5000"`
	DBQueryTimeoutMS int    `env:"DB_QUERY_TIMEOUT_MS" envDefault:"2000"`

	// Valkey
	ValkeyURL string `env:"VALKEY_URL" envDefault:""`

	// Logging
	LogLevel       string `env:"LOG_LEVEL" envDefault:"INFO"`
	LogJSON        bool   `env:"LOG_JSON" envDefault:"true"`
	LogFile        string `env:"LOG_FILE" envDefault:""`
	LogMaxMB       int    `env:"LOG_MAX_MB" envDefault:"10"`
	LogBackupCount int    `env:"LOG_BACKUP_COUNT" envDefault:"5"`

	// HTTP Edge
	MaxBodyBytes int64    `env:"MAX_BODY_BYTES" envDefault:"1048576"`
	CORSOrigins  []string `env:"CORS_ORIGINS" envSeparator:","`

	// Auth
	AuthTokenSecret  string `env:"AUTH_TOKEN_SECRET" envDefault:""`
	AuthTokenTTLMin  int    `env:"JWT_ACCESS_TTL_M" envDefault:"15"`
	RefreshTokenTTLH int    `env:"JWT_REFRESH_TTL_H" envDefault:"168"`
	APIKeyHashPepper string `env:"API_KEY_HASH_PEPPER" envDefault:""`

	// Resilience
	RetryMax             int `env:"RETRY_MAX" envDefault:"3"`
	RetryBaseMS          int `env:"RETRY_BASE_MS" envDefault:"200"`
	BreakerFailThreshold int `env:"BREAKER_FAIL_THRESHOLD" envDefault:"5"`
	BreakerRecoveryS     int `env:"BREAKER_RECOVERY_S" envDefault:"30"`

	// Rate Limit
	RateLimitRPM      int `env:"RATE_LIMIT_RPM" envDefault:"120"`
	LoginRateLimitRPM int `env:"LOGIN_RATE_LIMIT_RPM" envDefault:"10"`

	// AWS
	SecretsManagerSecretID string `env:"SECRETS_MANAGER_SECRET_ID" envDefault:""`
}

// IsProd reports whether the server runs in a production environment.
func (c Config) IsProd() bool {
	return c.Env == "production"
}

func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("port", c.Port),
		slog.String("env", c.Env),
		slog.String("database_url", redacted),
		slog.Int("db_max_conns", c.DBMaxConns),
		slog.String("valkey_url", redacted),
		slog.String("log_level", c.LogLevel),
		slog.Bool("log_json", c.LogJSON),
		slog.Int64("max_body_bytes", c.MaxBodyBytes),
		slog.Any("cors_origins", c.CORSOrigins),
		slog.String("auth_token_secret", redacted),
		slog.Int("jwt_access_ttl_m", c.AuthTokenTTLMin),
		slog.Int("jwt_refresh_ttl_h", c.RefreshTokenTTLH),
		slog.String("api_key_hash_pepper", redacted),
		slog.Int("retry_max", c.RetryMax),
		slog.Int("breaker_fail_threshold", c.BreakerFailThreshold),
		slog.Int("rate_limit_rpm", c.RateLimitRPM),
	)
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	if err := loadCloudSecrets(context.Background()); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
