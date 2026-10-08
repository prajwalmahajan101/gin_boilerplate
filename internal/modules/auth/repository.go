package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store/db"
)

type Repository struct {
	*store.Repository[User]
	q *db.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		Repository: store.NewRepository[User](pool),
		q:          db.New(pool),
	}
}

func (r *Repository) TouchLogin(ctx context.Context, id int64) error {
	_, err := r.q.TouchUserLogin(ctx, id)
	return err
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (db.User, bool, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, false, nil
	}
	if err != nil {
		return db.User{}, false, fmt.Errorf("auth: get by email: %w", err)
	}
	return row, true, nil
}
