package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// parser that returns a fixed identity with the given iat.
func parserWithIAT(uid, iat int64) TokenParser {
	return func(string) (int64, string, string, int64, error) {
		return uid, "user", "jti-1", iat, nil
	}
}

func runAuth(t *testing.T, cfg AuthConfig, bearer string) int {
	t.Helper()
	r := setupRouter(Auth(cfg))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest("GET", "/x", http.NoBody)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestAuth_UserEpoch_RejectsTokenIssuedBeforeEpoch(t *testing.T) {
	cfg := AuthConfig{
		Parse:     parserWithIAT(7, 1000), // token issued at t=1000
		UserEpoch: func(context.Context, int64) int64 { return 2000 },
	}
	if code := runAuth(t, cfg, "tok"); code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401 (revoked)", code)
	}
}

func TestAuth_UserEpoch_AllowsTokenIssuedAfterEpoch(t *testing.T) {
	cfg := AuthConfig{
		Parse:     parserWithIAT(7, 3000),
		UserEpoch: func(context.Context, int64) int64 { return 2000 },
	}
	if code := runAuth(t, cfg, "tok"); code != http.StatusOK {
		t.Fatalf("code=%d, want 200", code)
	}
}

func TestAuth_UserEpoch_ZeroMeansNoRevocation(t *testing.T) {
	cfg := AuthConfig{
		Parse:     parserWithIAT(7, 1),
		UserEpoch: func(context.Context, int64) int64 { return 0 },
	}
	if code := runAuth(t, cfg, "tok"); code != http.StatusOK {
		t.Fatalf("code=%d, want 200 (epoch 0 = no revocation)", code)
	}
}

func TestAuth_APIKeyExemptFromEpoch(t *testing.T) {
	// API key path carries no iat; epoch must not apply.
	cfg := AuthConfig{
		Parse:     parserWithIAT(7, 1), // unused: no bearer header
		APIKey:    func(context.Context, string) (int64, string, error) { return 7, "user", nil },
		UserEpoch: func(context.Context, int64) int64 { return 9999 },
	}
	r := setupRouter(Auth(cfg))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest("GET", "/x", http.NoBody)
	req.Header.Set("X-API-Key", "k")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d, want 200 (api key exempt from epoch)", w.Code)
	}
}
