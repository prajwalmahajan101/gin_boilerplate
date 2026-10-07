package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/response"
)

func init() { gin.SetMode(gin.TestMode) }

func setupRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	for _, m := range middlewares {
		r.Use(m)
	}
	return r
}

func envelope(t *testing.T, resp *httptest.ResponseRecorder) response.Envelope {
	t.Helper()
	var env response.Envelope
	if err := json.Unmarshal(resp.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	return env
}

// --- Recovery ---

func TestRecovery_PanicReturns500(t *testing.T) {
	r := setupRouter(RequestID(), Recovery(), response.ErrorHandler())
	r.GET("/boom", func(c *gin.Context) { panic("test panic") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/boom", http.NoBody))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", w.Code)
	}
	env := envelope(t, w)
	if env.Success {
		t.Fatal("expected success=false")
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID")
	}
}

// --- BodyLimit ---

func TestBodyLimit_OversizedContentLength(t *testing.T) {
	r := setupRouter(BodyLimit(100), response.ErrorHandler())
	r.POST("/upload", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest("POST", "/upload", strings.NewReader("x"))
	req.ContentLength = 200
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", w.Code)
	}
}

func TestBodyLimit_AllowsSmallBody(t *testing.T) {
	r := setupRouter(BodyLimit(100), response.ErrorHandler())
	r.POST("/upload", func(c *gin.Context) {
		io.ReadAll(c.Request.Body)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/upload", strings.NewReader("small"))
	req.ContentLength = 5
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

// --- CORS ---

func TestCORS_PreflightAllowed(t *testing.T) {
	r := setupRouter(CORS([]string{"https://app.example.com"}))
	r.OPTIONS("/api", func(c *gin.Context) {})

	req := httptest.NewRequest("OPTIONS", "/api", http.NoBody)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("got %d, want 204", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatal("missing ACAO header")
	}
}

func TestCORS_PreflightDisallowed(t *testing.T) {
	r := setupRouter(CORS([]string{"https://app.example.com"}))
	r.OPTIONS("/api", func(c *gin.Context) {})

	req := httptest.NewRequest("OPTIONS", "/api", http.NoBody)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", w.Code)
	}
}

func TestCORS_NoOriginPassesThrough(t *testing.T) {
	r := setupRouter(CORS([]string{"https://app.example.com"}))
	r.GET("/api", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api", http.NoBody))

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

// --- SecurityHeaders ---

func TestSecurityHeaders_AllPresent(t *testing.T) {
	r := setupRouter(SecurityHeaders("prod"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", http.NoBody))

	checks := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "no-referrer",
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains; preload",
	}
	for k, want := range checks {
		if got := w.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestSecurityHeaders_NoHSTSInDev(t *testing.T) {
	r := setupRouter(SecurityHeaders("local"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", http.NoBody))

	if got := w.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS should be empty in dev, got %q", got)
	}
}

// --- RequestID ---

func TestRequestID_EchoesValid(t *testing.T) {
	r := setupRouter(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("X-Request-ID", "my-trace-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "my-trace-123" {
		t.Fatalf("got %q, want my-trace-123", got)
	}
}

func TestRequestID_MintsUUIDForMissing(t *testing.T) {
	r := setupRouter(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", http.NoBody))

	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected minted request ID")
	}
	if len(id) != 36 {
		t.Fatalf("expected UUID format, got %q", id)
	}
}

func TestRequestID_RejectsInvalidAndMints(t *testing.T) {
	r := setupRouter(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("X-Request-ID", "has spaces bad!")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	id := w.Header().Get("X-Request-ID")
	if id == "has spaces bad!" {
		t.Fatal("should have rejected invalid ID")
	}
	if len(id) != 36 {
		t.Fatalf("expected minted UUID, got %q", id)
	}
}

// --- RequestLogging ---

func TestRequestLogging_ServerTimingHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := setupRouter(RequestLogging(logger))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", http.NoBody))

	st := w.Header().Get("Server-Timing")
	if st == "" {
		t.Fatal("missing Server-Timing header")
	}
	if !strings.Contains(st, "handler;dur=") {
		t.Fatalf("unexpected Server-Timing format: %s", st)
	}
}

// --- RateLimitHeaders ---

func TestRateLimitHeaders_EmitsWhenSet(t *testing.T) {
	r := setupRouter(RateLimitHeaders())
	r.GET("/", func(c *gin.Context) {
		SetRateLimitResult(c, RateLimitResult{Limit: 100, Remaining: 99, Reset: 1700000000})
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", http.NoBody))

	if got := w.Header().Get("X-RateLimit-Limit"); got != "100" {
		t.Fatalf("got %q, want 100", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "99" {
		t.Fatalf("got %q, want 99", got)
	}
}

func TestRateLimitHeaders_NoopWhenAbsent(t *testing.T) {
	r := setupRouter(RateLimitHeaders())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", http.NoBody))

	if got := w.Header().Get("X-RateLimit-Limit"); got != "" {
		t.Fatalf("expected no rate limit header, got %q", got)
	}
}
