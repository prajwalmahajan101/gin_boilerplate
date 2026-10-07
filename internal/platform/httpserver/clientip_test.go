package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

func clientIPFor(t *testing.T, trusted []string, remoteAddr, xff string) string {
	t.Helper()
	r := gin.New()
	if err := configureTrustedProxies(r, &config.Config{TrustedProxies: trusted}); err != nil {
		t.Fatalf("configureTrustedProxies: %v", err)
	}

	var got string
	r.GET("/", func(c *gin.Context) { got = ClientIP(c) })

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestClientIP_TrustedProxyHonorsXFF(t *testing.T) {
	got := clientIPFor(t, []string{"10.0.0.0/8"}, "10.0.0.5:1234", "203.0.113.9")
	if got != "203.0.113.9" {
		t.Fatalf("ClientIP = %q, want 203.0.113.9", got)
	}
}

func TestClientIP_UntrustedProxyIgnoresXFF(t *testing.T) {
	// No trusted proxies: XFF must be ignored, RemoteAddr wins.
	got := clientIPFor(t, nil, "198.51.100.7:4321", "203.0.113.9")
	if got != "198.51.100.7" {
		t.Fatalf("ClientIP = %q, want 198.51.100.7", got)
	}
}
