package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
)

type stubModule struct{ registered bool }

func (m *stubModule) RegisterRoutes(public, protected, admin *gin.RouterGroup) {
	m.registered = true
	public.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
}

func newTestRouter(t *testing.T, mods ...modules.Module) *gin.Engine {
	t.Helper()
	r, err := NewRouter(RouterConfig{
		Cfg:     &config.Config{Port: "8080", Env: "local"},
		Logger:  slog.Default(),
		Modules: mods,
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return r
}

func TestRouter_HealthzOKWithMiddlewareChain(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// Proves the M2 middleware chain is mounted (RequestID injects this header).
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID header; middleware chain not mounted")
	}
}

func TestRouter_RegistersModules(t *testing.T) {
	mod := &stubModule{}
	r := newTestRouter(t, mod)
	if !mod.registered {
		t.Fatal("module RegisterRoutes not called")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/ping", http.NoBody))
	if w.Code != http.StatusOK {
		t.Fatalf("module route status = %d, want 200", w.Code)
	}
}
