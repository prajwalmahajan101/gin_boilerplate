package items_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/items"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeService is an in-memory service double for handler unit tests.
type fakeService struct {
	createErr error
	getItem   items.Item
	getErr    error
}

func (s *fakeService) Create(_ context.Context, m *items.Item) error {
	if s.createErr != nil {
		return s.createErr
	}
	m.ID = 1
	return nil
}
func (s *fakeService) Update(_ context.Context, _ *items.Item) error   { return nil }
func (s *fakeService) Delete(_ context.Context, _ int64, _ bool) error { return nil }
func (s *fakeService) GetByIDOrFail(_ context.Context, _ int64) (items.Item, error) {
	return s.getItem, s.getErr
}
func (s *fakeService) List(_ context.Context, _, _ int) ([]items.Item, pagination.Meta, error) {
	return nil, pagination.Meta{}, nil
}

func newRouter(svc *fakeService) *gin.Engine {
	r := gin.New()
	g := r.Group("/api/v1")
	admin := r.Group("/api/v1/admin")
	items.NewHandler(svc).RegisterRoutes(g, g, admin)
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) response.Envelope {
	t.Helper()
	var env response.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return env
}

func TestCreate_MissingFieldsReturns400(t *testing.T) {
	r := newRouter(&fakeService{})
	w := do(r, http.MethodPost, "/api/v1/items", `{"name":"only name"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if env := decode(t, w); env.Errors[0].Code != apperr.CodeValidationError {
		t.Fatalf("code = %s, want VALIDATION_ERROR", env.Errors[0].Code)
	}
}

func TestCreate_Valid201(t *testing.T) {
	r := newRouter(&fakeService{})
	w := do(r, http.MethodPost, "/api/v1/items", `{"name":"Widget","code":"W-1"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
}

func TestCreate_DuplicateCodeReturns409(t *testing.T) {
	r := newRouter(&fakeService{createErr: apperr.Conflict("dup")})
	w := do(r, http.MethodPost, "/api/v1/items", `{"name":"Widget","code":"W-1"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if env := decode(t, w); env.Errors[0].Code != apperr.CodeConflict {
		t.Fatalf("code = %s, want CONFLICT", env.Errors[0].Code)
	}
}

func TestGet_NotFoundReturns404(t *testing.T) {
	r := newRouter(&fakeService{getErr: apperr.NotFound("missing")})
	w := do(r, http.MethodGet, "/api/v1/items/99", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGet_InvalidIDReturns400(t *testing.T) {
	r := newRouter(&fakeService{})
	w := do(r, http.MethodGet, "/api/v1/items/abc", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
