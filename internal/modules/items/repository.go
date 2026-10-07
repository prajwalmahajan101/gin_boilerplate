package items

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store/db"
)

// Repository composes the generic reflection-based CRUD repository with sqlc-generated
// typed queries for item-specific lookups.
type Repository struct {
	*store.Repository[Item]
	q *db.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		Repository: store.NewRepository[Item](pool),
		q:          db.New(pool),
	}
}

// GetByCode fetches an item by its unique code. found is false when no row matches.
func (r *Repository) GetByCode(ctx context.Context, code string) (db.Item, bool, error) {
	row, err := r.q.GetItemByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Item{}, false, nil
	}
	if err != nil {
		return db.Item{}, false, fmt.Errorf("items: get by code: %w", err)
	}
	return row, true, nil
}
