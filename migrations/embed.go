// Package migrations holds the embedded SQL migration files.
// The embed directive must sit beside the .sql files, so this lives at repo root
// rather than under internal/store.
package migrations

import "embed"

// FS carries the up/down migration files for the golang-migrate iofs source.
//
//go:embed *.sql
var FS embed.FS
