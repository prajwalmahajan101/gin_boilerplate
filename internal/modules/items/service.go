package items

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

// ItemService adds item-specific behaviour on top of the generic BaseService.
type ItemService struct {
	*store.BaseService[Item]
	repo *Repository
}

// NewService wires the generic CRUD service with item hooks. The code field is
// immutable after creation, so uniqueness is enforced only on create.
func NewService(pool *pgxpool.Pool) *ItemService {
	repo := NewRepository(pool)
	base := store.NewBaseService[Item](repo.Repository, pool, itemHooks{repo: repo})
	return &ItemService{BaseService: base, repo: repo}
}

// itemHooks overrides PreCreate to reject duplicate codes with a clean 409.
// The DB UNIQUE constraint remains the hard guard against races.
type itemHooks struct {
	store.NoOpHooks[Item]
	repo *Repository
}

func (h itemHooks) PreCreate(ctx context.Context, m *Item) error {
	_, found, err := h.repo.GetByCode(ctx, m.Code)
	if err != nil {
		return err
	}
	if found {
		return apperr.Conflict("item with this code already exists")
	}
	return nil
}
