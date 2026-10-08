//go:build integration

package auth_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/auth"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/httpserver"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store/dbtest"
)

func testCfg() *config.Config {
	return &config.Config{
		Env:              "local",
		MaxBodyBytes:     1 << 20,
		AuthTokenSecret:  "integration-test-secret-32chars!",
		AuthTokenTTLMin:  15,
		RefreshTokenTTLH: 168,
	}
}

func e2eRouter(t *testing.T) *gin.Engine {
	t.Helper()
	cfg := testCfg()
	pool := dbtest.New(t)
	tokenSvc := auth.NewTokenService(cfg)
	handler := auth.NewHandler(auth.NewService(pool, tokenSvc))
	r, err := httpserver.NewRouter(httpserver.RouterConfig{
		Cfg:         cfg,
		Logger:      slog.Default(),
		Modules:     []modules.Module{handler},
		TokenParser: tokenSvc.AccessParser(),
	})
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	return r
}

func req(t *testing.T, r *gin.Engine, method, path, body string, headers ...string) (*httptest.ResponseRecorder, response.Envelope) {
	t.Helper()
	w := httptest.NewRecorder()
	hr := httptest.NewRequest(method, path, strings.NewReader(body))
	hr.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		hr.Header.Set(headers[i], headers[i+1])
	}
	r.ServeHTTP(w, hr)

	var env response.Envelope
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("unmarshal %s %s: %v (body=%s)", method, path, err, w.Body.String())
		}
	}
	return w, env
}

func tokenPairFromData(t *testing.T, env response.Envelope) (string, string) {
	t.Helper()
	m, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("data is not an object: %v", env.Data)
	}
	return m["access_token"].(string), m["refresh_token"].(string)
}

func TestAuth_RegisterLoginRefresh(t *testing.T) {
	r := e2eRouter(t)

	// Register
	w, env := req(t, r, http.MethodPost, "/api/v1/auth/register",
		`{"email":"test@example.com","password":"password123"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201; body = %s", w.Code, w.Body.String())
	}
	access, refresh := tokenPairFromData(t, env)
	if access == "" || refresh == "" {
		t.Fatal("register returned empty tokens")
	}

	// Login
	w, env = req(t, r, http.MethodPost, "/api/v1/auth/login",
		`{"email":"test@example.com","password":"password123"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", w.Code)
	}
	access, refresh = tokenPairFromData(t, env)

	// Refresh
	w, env = req(t, r, http.MethodPost, "/api/v1/auth/refresh",
		`{"refresh_token":"`+refresh+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	newAccess, _ := tokenPairFromData(t, env)
	if newAccess == access {
		t.Fatal("refresh should return a new access token")
	}
}

func TestAuth_DuplicateEmail(t *testing.T) {
	r := e2eRouter(t)
	body := `{"email":"dup@example.com","password":"password123"}`
	if w, _ := req(t, r, http.MethodPost, "/api/v1/auth/register", body); w.Code != http.StatusCreated {
		t.Fatalf("first register status = %d, want 201", w.Code)
	}
	w, env := req(t, r, http.MethodPost, "/api/v1/auth/register", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("dup status = %d, want 409", w.Code)
	}
	if env.Errors[0].Code != "CONFLICT" {
		t.Fatalf("code = %s, want CONFLICT", env.Errors[0].Code)
	}
}

func TestAuth_WrongPassword(t *testing.T) {
	r := e2eRouter(t)
	req(t, r, http.MethodPost, "/api/v1/auth/register",
		`{"email":"wrong@example.com","password":"password123"}`)
	w, env := req(t, r, http.MethodPost, "/api/v1/auth/login",
		`{"email":"wrong@example.com","password":"badpass"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong pw status = %d, want 401", w.Code)
	}
	if env.Errors[0].Code != "UNAUTHORIZED" {
		t.Fatalf("code = %s, want UNAUTHORIZED", env.Errors[0].Code)
	}
}

func TestAuth_ProtectedRouteRequiresToken(t *testing.T) {
	r := e2eRouter(t)

	// No token -> 401
	w, _ := req(t, r, http.MethodGet, "/api/v1/items", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no-token status = %d, want 401", w.Code)
	}

	// Register and use token -> protected route accessible
	_, env := req(t, r, http.MethodPost, "/api/v1/auth/register",
		`{"email":"prot@example.com","password":"password123"}`)
	access, _ := tokenPairFromData(t, env)

	w, _ = req(t, r, http.MethodGet, "/api/v1/items", "",
		"Authorization", "Bearer "+access)
	if w.Code != http.StatusOK {
		t.Fatalf("with-token status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}
