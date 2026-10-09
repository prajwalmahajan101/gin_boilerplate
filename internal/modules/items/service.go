package items

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/cache"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

// tombstone is the negative-cache sentinel: a confirmed-absent id is cached as
// this value so a repeated bad id is answered from cache, not the DB (cache
// penetration, T28). The leading 0x00 byte cannot start a marshalled Item (a
// JSON object begins with '{'), so a tombstone is unambiguously distinguishable
// from a real cached value.
var tombstone = []byte{0}

// ItemService adds item-specific behaviour on top of the generic BaseService: a
// cache-aside hot-read tier (L1 → L2 → DB) with singleflight stampede collapse
// and negative caching. The cache is fail-open, so a Valkey outage reports a
// miss and the read falls through to Postgres — never a 5xx.
type ItemService struct {
	*store.BaseService[Item]
	repo     *Repository
	cache    cache.Cache
	cacheTTL time.Duration
	negTTL   time.Duration      // negative-cache (tombstone) TTL for absent ids (T28)
	sf       singleflight.Group // collapses concurrent same-key cache misses (T26)
}

// NewService wires the generic CRUD service with item hooks and a cache-aside
// backend for the hot reads. A nil cache falls back to an in-process cache so
// tests and Valkey-disabled runs still work. The code field is immutable after
// creation, so uniqueness is enforced only on create.
func NewService(pool *pgxpool.Pool, c cache.Cache, cacheTTL, negTTL time.Duration) *ItemService {
	repo := NewRepository(pool)
	base := store.NewBaseService[Item](repo.Repository, pool, itemHooks{repo: repo})
	if c == nil {
		c = cache.NewTiered("items", nil, cache.TierConfig{}) // nil rdb → in-memory
	}
	return &ItemService{BaseService: base, repo: repo, cache: c, cacheTTL: cacheTTL, negTTL: negTTL}
}

func itemCacheKey(id int64) string       { return fmt.Sprintf("item:%d", id) }
func listCacheKey(page, size int) string { return fmt.Sprintf("items:list:%d:%d", page, size) }

// GetByIDOrFail serves the single-item hot read cache-aside: a hit skips the DB
// pool entirely (the scarce resource under load); a miss is collapsed by
// singleflight so exactly one goroutine reads the DB and repopulates (T26). A
// confirmed-absent id is tombstoned so a repeat does not re-hit the DB (T28).
func (s *ItemService) GetByIDOrFail(ctx context.Context, id int64) (Item, error) {
	key := itemCacheKey(id)
	if b, hit, _ := s.cache.Get(ctx, key); hit {
		if bytes.Equal(b, tombstone) {
			return Item{}, apperr.NotFound("record not found")
		}
		var item Item
		if json.Unmarshal(b, &item) == nil {
			return item, nil
		}
		_ = s.cache.Delete(ctx, key) // corrupt entry: drop it, fall through to DB
	}

	// ponytail: Do shares the leader's ctx — a cancelled leader fails its
	// followers; fine for a sub-ms PK read, revisit with DoChan if it grows slow.
	v, err, _ := s.sf.Do(key, func() (any, error) {
		item, derr := s.BaseService.GetByIDOrFail(ctx, id)
		if derr != nil {
			if apperr.Is(derr, apperr.CodeNotFound) {
				_ = s.cache.Set(ctx, key, tombstone, s.negTTL)
			}
			return Item{}, derr
		}
		if b, merr := json.Marshal(item); merr == nil {
			_ = s.cache.Set(ctx, key, b, s.cacheTTL)
		}
		return item, nil
	})
	if err != nil {
		return Item{}, err
	}
	return v.(Item), nil
}

type listResult struct {
	Items []Item          `json:"items"`
	Meta  pagination.Meta `json:"meta"`
}

// List serves a paginated read cache-aside, keyed by (page, size), collapsing
// concurrent misses with singleflight.
func (s *ItemService) List(ctx context.Context, page, pageSize int) ([]Item, pagination.Meta, error) {
	key := listCacheKey(page, pageSize)
	if b, hit, _ := s.cache.Get(ctx, key); hit {
		var lr listResult
		if json.Unmarshal(b, &lr) == nil {
			return lr.Items, lr.Meta, nil
		}
		_ = s.cache.Delete(ctx, key)
	}

	v, err, _ := s.sf.Do(key, func() (any, error) {
		items, meta, derr := s.BaseService.List(ctx, page, pageSize)
		if derr != nil {
			return nil, derr
		}
		lr := listResult{Items: items, Meta: meta}
		if b, merr := json.Marshal(lr); merr == nil {
			_ = s.cache.Set(ctx, key, b, s.cacheTTL)
		}
		return lr, nil
	})
	if err != nil {
		return nil, pagination.Meta{}, err
	}
	lr := v.(listResult)
	return lr.Items, lr.Meta, nil
}

// Update writes through the base service, then invalidates the item's cache entry.
func (s *ItemService) Update(ctx context.Context, m *Item) error {
	if err := s.BaseService.Update(ctx, m); err != nil {
		return err
	}
	_ = s.cache.Delete(ctx, itemCacheKey(m.ID))
	return nil
}

// Delete soft/hard-deletes via the base service, then drops the cache entry.
//
// ponytail: list cache keys are left to TTL expiry on any mutation (same as the
// source) — a write can serve a slightly stale list for up to cacheTTL. Add
// per-mutation list invalidation only if staleness shows up as a real problem.
func (s *ItemService) Delete(ctx context.Context, id int64, soft bool) error {
	if err := s.BaseService.Delete(ctx, id, soft); err != nil {
		return err
	}
	_ = s.cache.Delete(ctx, itemCacheKey(id))
	return nil
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
