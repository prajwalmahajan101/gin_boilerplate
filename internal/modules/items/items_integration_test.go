//go:build integration

package items_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/items"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/httpserver"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store/dbtest"
)

func e2eRouter(t *testing.T) *gin.Engine {
	t.Helper()
	pool := dbtest.New(t)
	handler := items.NewHandler(items.NewService(pool, nil, 5*time.Minute, 30*time.Second))
	r, err := httpserver.NewRouter(httpserver.RouterConfig{
		Cfg:     &config.Config{Env: "local", MaxBodyBytes: 1 << 20},
		Logger:  slog.Default(),
		Modules: []modules.Module{handler},
	})
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	return r
}

func req(t *testing.T, r *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, response.Envelope) {
	t.Helper()
	w := httptest.NewRecorder()
	hr := httptest.NewRequest(method, path, strings.NewReader(body))
	hr.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, hr)

	var env response.Envelope
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("unmarshal %s %s: %v (body=%s)", method, path, err, w.Body.String())
		}
	}
	return w, env
}

func dataID(t *testing.T, env response.Envelope) int64 {
	t.Helper()
	m, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("data is not an object: %v", env.Data)
	}
	return int64(m["id"].(float64))
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestItems_FullCRUD(t *testing.T) {
	r := e2eRouter(t)

	// Create -> 201
	w, env := req(t, r, http.MethodPost, "/api/v1/items", `{"name":"Widget","code":"W-1","notes":{"a":1}}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", w.Code)
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID")
	}
	id := dataID(t, env)
	path := "/api/v1/items/" + itoa(id)

	// Get by id -> 200
	if w, _ = req(t, r, http.MethodGet, path, ""); w.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", w.Code)
	}

	// List -> 200 paginated
	w, env = req(t, r, http.MethodGet, "/api/v1/items", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", w.Code)
	}
	if d, _ := env.Data.(map[string]any); d["pagination"] == nil {
		t.Fatalf("list missing pagination meta: %v", env.Data)
	}

	// Patch -> 200
	if w, _ = req(t, r, http.MethodPatch, path, `{"name":"Widget2"}`); w.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want 200", w.Code)
	}

	// Delete (soft) -> 200
	if w, _ = req(t, r, http.MethodDelete, path, ""); w.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", w.Code)
	}
}

func TestItems_DuplicateCodeConflict(t *testing.T) {
	r := e2eRouter(t)
	body := `{"name":"Dup","code":"DUP-1"}`
	if w, _ := req(t, r, http.MethodPost, "/api/v1/items", body); w.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want 201", w.Code)
	}
	w, env := req(t, r, http.MethodPost, "/api/v1/items", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", w.Code)
	}
	if env.Errors[0].Code != "CONFLICT" {
		t.Fatalf("code = %s, want CONFLICT", env.Errors[0].Code)
	}
}

func TestItems_ValidationAndNotFound(t *testing.T) {
	r := e2eRouter(t)

	w, env := req(t, r, http.MethodPost, "/api/v1/items", `{"name":"no code"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status = %d, want 400", w.Code)
	}
	if env.Errors[0].Code != "VALIDATION_ERROR" {
		t.Fatalf("code = %s, want VALIDATION_ERROR", env.Errors[0].Code)
	}

	w, env = req(t, r, http.MethodGet, "/api/v1/items/99999", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", w.Code)
	}
	if env.Errors[0].Code != "NOT_FOUND" {
		t.Fatalf("code = %s, want NOT_FOUND", env.Errors[0].Code)
	}
}

// countingCache is an in-memory Cache that records Get hits and Set calls so a
// test can prove the hot read is served from cache (not the DB) and that an
// absent id is tombstoned.
type countingCache struct {
	mu   sync.Mutex
	m    map[string][]byte
	hits int
	sets int
}

func newCountingCache() *countingCache { return &countingCache{m: map[string][]byte{}} }

func (c *countingCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	if ok {
		c.hits++
	}
	return v, ok, nil
}

func (c *countingCache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sets++
	c.m[key] = val
	return nil
}

func (c *countingCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
	return nil
}

// TestItems_CacheAside proves the roadmap M7 verify points at the service layer:
// a second GET is served from cache (hit), and a non-existent id is negatively
// cached (tombstone) so a repeat is answered from cache too.
func TestItems_CacheAside(t *testing.T) {
	pool := dbtest.New(t)
	cc := newCountingCache()
	svc := items.NewService(pool, cc, time.Minute, time.Minute)
	ctx := context.Background()

	it := &items.Item{Name: "Cached", Code: "C-1"}
	if err := svc.Create(ctx, it); err != nil {
		t.Fatalf("create: %v", err)
	}

	// First GET: cache miss → DB → populate (one Set, no hit yet).
	if _, err := svc.GetByIDOrFail(ctx, it.ID); err != nil {
		t.Fatalf("first get: %v", err)
	}
	if cc.sets != 1 {
		t.Fatalf("after first get: sets = %d, want 1 (cache populated)", cc.sets)
	}

	// Second GET: served from cache (a hit) — the DB path is not taken.
	got, err := svc.GetByIDOrFail(ctx, it.ID)
	if err != nil {
		t.Fatalf("second get: %v", err)
	}
	if got.ID != it.ID || got.Name != "Cached" {
		t.Fatalf("second get = %+v, want cached item %d", got, it.ID)
	}
	if cc.hits < 1 {
		t.Fatal("second get did not hit the cache")
	}

	// Non-existent id: first GET tombstones it, second is answered from cache.
	const absent = int64(987654321)
	if _, err := svc.GetByIDOrFail(ctx, absent); !apperr.Is(err, apperr.CodeNotFound) {
		t.Fatalf("absent first get err = %v, want NOT_FOUND", err)
	}
	hitsBefore := cc.hits
	if _, err := svc.GetByIDOrFail(ctx, absent); !apperr.Is(err, apperr.CodeNotFound) {
		t.Fatalf("absent second get err = %v, want NOT_FOUND", err)
	}
	if cc.hits <= hitsBefore {
		t.Fatal("absent second get did not hit the negative cache (tombstone)")
	}
}
