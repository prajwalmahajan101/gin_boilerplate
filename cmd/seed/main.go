package main

import (
	"context"
	"fmt"
	"os"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/auth"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := store.NewPool(ctx, cfg)
	if err != nil {
		return err
	}
	defer store.Close(pool)

	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if password == "" {
		return fmt.Errorf("SEED_ADMIN_PASSWORD env is required")
	}

	tokenSvc := auth.NewTokenService(cfg)
	svc := auth.NewService(pool, tokenSvc)

	email := "admin@example.com"
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	u := &auth.User{
		Email:        email,
		PasswordHash: hash,
		Role:         auth.RoleAdmin,
	}
	if err := svc.Create(ctx, u); err != nil {
		if apperr.Is(err, apperr.CodeConflict) {
			fmt.Println("seed: admin already exists, skipping")
			return nil
		}
		return err
	}
	fmt.Printf("seed: created admin user %s (id=%d)\n", email, u.ID)
	return nil
}
