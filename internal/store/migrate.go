package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" database driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/prajwalmahajan101/gin_boilerplate/migrations"
)

// toPgx5URL rewrites a standard postgres URL to the scheme golang-migrate's
// pgx/v5 driver registers under ("pgx5").
func toPgx5URL(databaseURL string) string {
	for _, scheme := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(databaseURL, scheme) {
			return "pgx5://" + strings.TrimPrefix(databaseURL, scheme)
		}
	}
	return databaseURL
}

// newMigrator builds a migrate.Migrate over the embedded migration files and the
// target database. The pgx/v5 driver takes a pg_advisory_lock during apply, so
// concurrent boots serialize safely. Caller must Close the returned migrator.
func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("store: migrate source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5URL(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("store: migrate init: %w", err)
	}
	return m, nil
}

// Migrate applies all pending migrations. A no-op (already current) is success.
func Migrate(databaseURL string) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("store: migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back migrations. steps <= 0 rolls back everything; otherwise
// it rolls back exactly that many steps.
func MigrateDown(databaseURL string, steps int) error {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if steps <= 0 {
		err = m.Down()
	} else {
		err = m.Steps(-steps)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("store: migrate down: %w", err)
	}
	return nil
}

// MigrateVersion reports the current schema version and whether it is dirty.
func MigrateVersion(databaseURL string) (version uint, dirty bool, err error) {
	m, err := newMigrator(databaseURL)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("store: migrate version: %w", err)
	}
	return version, dirty, nil
}
