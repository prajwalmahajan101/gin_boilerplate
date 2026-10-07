package store

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
)

type ServiceHooks[T Model] interface {
	PreCreate(ctx context.Context, m *T) error
	PostCreate(ctx context.Context, m *T)
	PreUpdate(ctx context.Context, m *T) error
	PostUpdate(ctx context.Context, m *T)
	PreDelete(ctx context.Context, m *T) error
	PostDelete(ctx context.Context, m *T)
}

type NoOpHooks[T Model] struct{}

func (NoOpHooks[T]) PreCreate(context.Context, *T) error { return nil }
func (NoOpHooks[T]) PostCreate(context.Context, *T)      {}
func (NoOpHooks[T]) PreUpdate(context.Context, *T) error { return nil }
func (NoOpHooks[T]) PostUpdate(context.Context, *T)      {}
func (NoOpHooks[T]) PreDelete(context.Context, *T) error { return nil }
func (NoOpHooks[T]) PostDelete(context.Context, *T)      {}

type BaseService[T Model] struct {
	Repo     *Repository[T]
	Pool     *pgxpool.Pool
	hooks    ServiceHooks[T]
	Cascades []CascadeTarget
}

func NewBaseService[T Model](repo *Repository[T], pool *pgxpool.Pool, hooks ServiceHooks[T]) *BaseService[T] {
	if hooks == nil {
		hooks = &NoOpHooks[T]{}
	}
	return &BaseService[T]{Repo: repo, Pool: pool, hooks: hooks}
}

func (s *BaseService[T]) Create(ctx context.Context, m *T) error {
	return Atomic(ctx, s.Pool, func(tx pgx.Tx) error {
		repo := s.Repo.WithTx(tx)
		if err := s.hooks.PreCreate(ctx, m); err != nil {
			return err
		}
		if err := repo.Create(ctx, m); err != nil {
			return err
		}
		s.hooks.PostCreate(ctx, m)
		return nil
	})
}

func (s *BaseService[T]) Update(ctx context.Context, m *T) error {
	return Atomic(ctx, s.Pool, func(tx pgx.Tx) error {
		repo := s.Repo.WithTx(tx)
		if err := s.hooks.PreUpdate(ctx, m); err != nil {
			return err
		}
		if err := repo.Update(ctx, m); err != nil {
			return err
		}
		s.hooks.PostUpdate(ctx, m)
		return nil
	})
}

func (s *BaseService[T]) Delete(ctx context.Context, id int64, soft bool) error {
	return Atomic(ctx, s.Pool, func(tx pgx.Tx) error {
		repo := s.Repo.WithTx(tx)

		rec, found, err := repo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return apperr.NotFound("record not found")
		}

		if err := s.hooks.PreDelete(ctx, &rec); err != nil {
			return err
		}

		if soft {
			if err := repo.SoftDelete(ctx, id); err != nil {
				return err
			}
			if len(s.Cascades) > 0 {
				if err := CascadeSoftDelete(ctx, id, s.Cascades); err != nil {
					return err
				}
			}
		} else {
			if err := repo.HardDelete(ctx, id); err != nil {
				return err
			}
		}

		s.hooks.PostDelete(ctx, &rec)
		return nil
	})
}

func (s *BaseService[T]) GetByID(ctx context.Context, id int64) (T, bool, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *BaseService[T]) GetActiveByID(ctx context.Context, id int64) (T, bool, error) {
	return s.Repo.GetActiveByID(ctx, id)
}

func (s *BaseService[T]) GetByIDOrFail(ctx context.Context, id int64) (T, error) {
	rec, found, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return rec, err
	}
	if !found {
		var zero T
		return zero, apperr.NotFound("record not found")
	}
	return rec, nil
}

func (s *BaseService[T]) GetActiveByIDOrFail(ctx context.Context, id int64) (T, error) {
	rec, found, err := s.Repo.GetActiveByID(ctx, id)
	if err != nil {
		return rec, err
	}
	if !found {
		var zero T
		return zero, apperr.NotFound("record not found")
	}
	return rec, nil
}

func (s *BaseService[T]) List(ctx context.Context, page, pageSize int) ([]T, pagination.Meta, error) {
	items, total, err := s.Repo.Paginate(ctx, page, pageSize)
	if err != nil {
		return nil, pagination.Meta{}, err
	}
	return items, pagination.NewMeta(page, pageSize, total), nil
}

// Cascade soft-delete.

const MaxCascadeDepth = 10

type CascadeTarget interface {
	// SoftDeleteByParent soft-deletes the rows owned by parentID and returns the
	// IDs of the affected rows so the cascade can descend into their own children.
	// Return an empty slice when a target has no further descendants.
	SoftDeleteByParent(ctx context.Context, parentID int64) (childIDs []int64, err error)
}

// CascadeSoftDelete walks the parent's descendants breadth-first, soft-deleting
// each target's rows level by level. The depth cap guards against deep or cyclic
// graphs: once the cap is reached, deletion stops descending and logs a warning.
func CascadeSoftDelete(ctx context.Context, parentID int64, targets []CascadeTarget) error {
	type entry struct {
		id    int64
		depth int
	}
	queue := []entry{{id: parentID, depth: 0}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, t := range targets {
			childIDs, err := t.SoftDeleteByParent(ctx, cur.id)
			if err != nil {
				return err
			}
			if len(childIDs) == 0 {
				continue
			}
			if cur.depth+1 >= MaxCascadeDepth {
				slog.WarnContext(ctx, "cascade soft-delete depth cap reached; not descending further",
					"parent_id", cur.id, "depth", cur.depth, "max", MaxCascadeDepth, "undescended", len(childIDs))
				continue
			}
			for _, cid := range childIDs {
				queue = append(queue, entry{id: cid, depth: cur.depth + 1})
			}
		}
	}
	return nil
}
