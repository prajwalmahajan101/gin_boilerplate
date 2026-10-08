//go:build integration

package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/auth"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/modules/items"
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

type testEnv struct {
	router *gin.Engine
	pool   *pgxpool.Pool
}

func e2eSetup(t *testing.T) testEnv {
	t.Helper()
	cfg := testCfg()
	pool := dbtest.New(t)
	tokenSvc := auth.NewTokenService(cfg)
	apiKeySvc := auth.NewAPIKeyService(pool, "test-pepper")
	rbacSvc := auth.NewRBACService(pool)
	handler := auth.NewHandler(auth.NewService(pool, tokenSvc, nil), apiKeySvc, rbacSvc, tokenSvc)
	r, err := httpserver.NewRouter(httpserver.RouterConfig{
		Cfg:             cfg,
		Logger:          slog.Default(),
		Modules:         []modules.Module{handler, items.NewHandler(items.NewService(pool))},
		TokenParser:     tokenSvc.AccessParser(),
		APIKeyValidator: apiKeySvc.Validate,
	})
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	return testEnv{router: r, pool: pool}
}

func e2eRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return e2eSetup(t).router
}

func promoteToAdmin(t *testing.T, pool *pgxpool.Pool, email string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "UPDATE users SET role = 'admin' WHERE email = $1", email)
	if err != nil {
		t.Fatalf("promote to admin: %v", err)
	}
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

func TestAuth_AdminCRUD(t *testing.T) {
	te := e2eSetup(t)
	r := te.router

	// Register admin
	req(t, r, http.MethodPost, "/api/v1/auth/register",
		`{"email":"admin@test.com","password":"password123"}`)
	promoteToAdmin(t, te.pool, "admin@test.com")

	// Re-login to get admin-role token
	_, env := req(t, r, http.MethodPost, "/api/v1/auth/login",
		`{"email":"admin@test.com","password":"password123"}`)
	adminAccess, _ := tokenPairFromData(t, env)
	bearer := "Bearer " + adminAccess

	// Register a regular user
	req(t, r, http.MethodPost, "/api/v1/auth/register",
		`{"email":"user@test.com","password":"password123"}`)

	// List users
	w, env := req(t, r, http.MethodGet, "/api/v1/admin/users", "", "Authorization", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("list users status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// Get the regular user's ID from the list
	data := env.Data.(map[string]any)
	userList := data["items"].([]any)
	var userID float64
	for _, item := range userList {
		u := item.(map[string]any)
		if u["email"] == "user@test.com" {
			userID = u["id"].(float64)
		}
	}
	if userID == 0 {
		t.Fatal("user not found in list")
	}
	path := fmt.Sprintf("/api/v1/admin/users/%d", int64(userID))

	// Get user by ID
	w, _ = req(t, r, http.MethodGet, path, "", "Authorization", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("get user status = %d, want 200", w.Code)
	}

	// Update role
	w, _ = req(t, r, http.MethodPatch, path, `{"role":"admin"}`, "Authorization", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("update user status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// Soft-delete
	w, _ = req(t, r, http.MethodDelete, path, "", "Authorization", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("delete user status = %d, want 200", w.Code)
	}
}

func TestAuth_APIKey_SelfService(t *testing.T) {
	te := e2eSetup(t)
	r := te.router

	// Register
	_, env := req(t, r, http.MethodPost, "/api/v1/auth/register",
		`{"email":"keyuser@test.com","password":"password123"}`)
	access, _ := tokenPairFromData(t, env)
	bearer := "Bearer " + access

	// Create API key
	w, env := req(t, r, http.MethodPost, "/api/v1/auth/api-keys",
		`{"name":"test-key"}`, "Authorization", bearer)
	if w.Code != http.StatusCreated {
		t.Fatalf("create key status = %d, want 201; body = %s", w.Code, w.Body.String())
	}
	keyData := env.Data.(map[string]any)
	rawKey := keyData["key"].(string)
	keyID := int64(keyData["id"].(float64))
	if rawKey == "" {
		t.Fatal("key should be returned on creation")
	}

	// List keys
	w, _ = req(t, r, http.MethodGet, "/api/v1/auth/api-keys", "", "Authorization", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("list keys status = %d, want 200", w.Code)
	}

	// Use API key on protected route
	w, _ = req(t, r, http.MethodGet, "/api/v1/items", "", "X-API-Key", rawKey)
	if w.Code != http.StatusOK {
		t.Fatalf("api key auth status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// Revoke key
	w, _ = req(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/auth/api-keys/%d", keyID),
		"", "Authorization", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("revoke key status = %d, want 200", w.Code)
	}
}

func TestAuth_ChangePassword(t *testing.T) {
	r := e2eRouter(t)

	// Register
	_, env := req(t, r, http.MethodPost, "/api/v1/auth/register",
		`{"email":"chpw@test.com","password":"password123"}`)
	access, _ := tokenPairFromData(t, env)
	bearer := "Bearer " + access

	// Change password
	w, _ := req(t, r, http.MethodPost, "/api/v1/auth/change-password",
		`{"old_password":"password123","new_password":"newpass456"}`, "Authorization", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("change pw status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// Login with new password
	w, _ = req(t, r, http.MethodPost, "/api/v1/auth/login",
		`{"email":"chpw@test.com","password":"newpass456"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("login new pw status = %d, want 200", w.Code)
	}

	// Old password fails
	w, _ = req(t, r, http.MethodPost, "/api/v1/auth/login",
		`{"email":"chpw@test.com","password":"password123"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("login old pw status = %d, want 401", w.Code)
	}
}
