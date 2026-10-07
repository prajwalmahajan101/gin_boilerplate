package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

func init() { gin.SetMode(gin.TestMode) }

func doGET(h gin.HandlerFunc, path string) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET(path, h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	return w
}

func TestLiveness_OK(t *testing.T) {
	w := doGET(Liveness(), "/healthz")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestReadiness_EmptyIsReady(t *testing.T) {
	w := doGET(Readiness(nil), "/readyz")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestReadiness_AllPass(t *testing.T) {
	checks := map[string]Check{
		"postgres": func(context.Context) error { return nil },
	}
	w := doGET(Readiness(checks), "/readyz")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestReadiness_FailReturns503(t *testing.T) {
	checks := map[string]Check{
		"postgres": func(context.Context) error { return nil },
		"valkey":   func(context.Context) error { return errors.New("connection refused") },
	}
	w := doGET(Readiness(checks), "/readyz")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}

	var env response.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Success {
		t.Fatal("success = true, want false")
	}
	data, _ := env.Data.(map[string]any)
	checksOut, _ := data["checks"].(map[string]any)
	if checksOut["valkey"] == "ok" {
		t.Fatalf("failing check reported ok: %v", checksOut)
	}
	if checksOut["postgres"] != "ok" {
		t.Fatalf("passing check not ok: %v", checksOut)
	}
}
