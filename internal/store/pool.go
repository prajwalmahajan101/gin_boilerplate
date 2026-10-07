package store

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

var queryTimeout = 2 * time.Second

func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse DATABASE_URL: %w", err)
	}

	maxConns := cfg.DBMaxConns
	if maxConns <= 0 {
		maxConns = 4 * runtime.NumCPU()
	}
	if maxConns > math.MaxInt32 {
		maxConns = math.MaxInt32
	}
	poolCfg.MaxConns = int32(maxConns) //nolint:gosec // clamped above
	poolCfg.ConnConfig.ConnectTimeout = time.Duration(cfg.DBConnTimeoutMS) * time.Millisecond

	queryTimeout = time.Duration(cfg.DBQueryTimeoutMS) * time.Millisecond

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("store: create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, poolCfg.ConnConfig.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return pool, nil
}

func WithQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, queryTimeout)
}

func Atomic(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	txCtx, cancel := WithQueryTimeout(ctx)
	defer cancel()

	tx, err := pool.Begin(txCtx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(txCtx)
		return err
	}

	if err := tx.Commit(txCtx); err != nil {
		_ = tx.Rollback(txCtx)
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
