package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
)

func init() { gin.SetMode(gin.TestMode) }

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", http.NoBody)

	Success(c, http.StatusOK, "done", gin.H{"id": 1})

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.Success || env.Message != "done" {
		t.Fatalf("envelope: %+v", env)
	}
}

func TestErrorAppError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", http.NoBody)

	Error(c, apperr.NotFound("item"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Success {
		t.Fatal("should be false")
	}
	if len(env.Errors) != 1 || env.Errors[0].Code != "NOT_FOUND" {
		t.Fatalf("errors: %+v", env.Errors)
	}
}

func TestErrorPlain(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", http.NoBody)

	Error(c, errors.New("secret db info"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if containsSubstring(body, "secret db info") {
		t.Fatal("internal detail leaked")
	}
}

func TestPaginated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", http.NoBody)

	meta := pagination.NewMeta(1, 10, 25)
	Paginated(c, []string{"a", "b"}, meta)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.Success || env.Message != "ok" {
		t.Fatalf("envelope: %+v", env)
	}
}

func TestErrorHandler(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(ErrorHandler())
	r.GET("/test", func(c *gin.Context) {
		_ = c.Error(apperr.Forbidden("nope"))
	})

	c.Request = httptest.NewRequest("GET", "/test", http.NoBody)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d", w.Code)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || s != "" && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
