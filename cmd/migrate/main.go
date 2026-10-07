// Command migrate applies, rolls back, or inspects database migrations.
//
// Usage:
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down [n]      # n steps, default 1
//	go run ./cmd/migrate version
//	go run ./cmd/migrate create <name>
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

const migrationsDir = "migrations"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: migrate up|down [n]|version|create <name>")
	}

	// create does not need a database connection.
	if args[0] == "create" {
		if len(args) < 2 {
			return fmt.Errorf("usage: migrate create <name>")
		}
		return createMigration(args[1])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	switch args[0] {
	case "up":
		if mErr := store.Migrate(cfg.DatabaseURL); mErr != nil {
			return mErr
		}
		fmt.Println("migrations applied")
		return nil
	case "down":
		steps := 1
		if len(args) > 1 {
			steps, err = strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid step count %q: %w", args[1], err)
			}
		}
		if err := store.MigrateDown(cfg.DatabaseURL, steps); err != nil {
			return err
		}
		fmt.Printf("rolled back %d step(s)\n", steps)
		return nil
	case "version":
		v, dirty, err := store.MigrateVersion(cfg.DatabaseURL)
		if err != nil {
			return err
		}
		fmt.Printf("version=%d dirty=%t\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// createMigration writes empty up/down files with the next sequence number.
func createMigration(name string) error {
	seq, err := nextSeq()
	if err != nil {
		return err
	}
	slug := slugRE.ReplaceAllString(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"), "")
	if slug == "" {
		return fmt.Errorf("migration name must contain [A-Za-z0-9_]")
	}
	base := fmt.Sprintf("%06d_%s", seq, slug)
	for _, suffix := range []string{"up", "down"} {
		// slug is sanitized to [A-Za-z0-9_], so no path traversal is possible.
		path := filepath.Join(migrationsDir, base+"."+suffix+".sql")
		if err := os.WriteFile(path, []byte("-- "+suffix+" migration\n"), 0o600); err != nil { //nolint:gosec // slug sanitized above
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Println("created", path)
	}
	return nil
}

var slugRE = regexp.MustCompile(`[^A-Za-z0-9_]`)

// nextSeq returns one past the highest numeric prefix among existing migrations.
func nextSeq() (int, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", migrationsDir, err)
	}
	maxSeq := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		prefix, _, ok := strings.Cut(e.Name(), "_")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(prefix)
		if err != nil {
			continue
		}
		if n > maxSeq {
			maxSeq = n
		}
	}
	return maxSeq + 1, nil
}
