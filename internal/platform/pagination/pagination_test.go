package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func paramsFor(query string) (int, int) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+query, http.NoBody)
	return Params(c)
}

func TestParams(t *testing.T) {
	cases := []struct {
		query            string
		wantPage, wantSz int
	}{
		{"", 1, 20},
		{"page=3&page_size=10", 3, 10},
		{"page=0&page_size=0", 1, 20},
		{"page=-5&page_size=-1", 1, 20},
		{"page=abc&page_size=xyz", 1, 20},
		{"page_size=500", 1, 100},
	}
	for _, tc := range cases {
		p, s := paramsFor(tc.query)
		if p != tc.wantPage || s != tc.wantSz {
			t.Errorf("%q: got page=%d size=%d, want %d/%d", tc.query, p, s, tc.wantPage, tc.wantSz)
		}
	}
}

func TestNewMeta(t *testing.T) {
	cases := []struct {
		page, size, total  int
		wantPages          int
		wantNext, wantPrev bool
	}{
		{1, 10, 0, 0, false, false},
		{1, 10, 25, 3, true, false},
		{2, 10, 25, 3, true, true},
		{3, 10, 25, 3, false, true},
		{1, 10, 10, 1, false, false},
	}
	for _, tc := range cases {
		m := NewMeta(tc.page, tc.size, tc.total)
		if m.TotalPages != tc.wantPages || m.HasNext != tc.wantNext || m.HasPrev != tc.wantPrev {
			t.Errorf("total=%d page=%d: got pages=%d next=%v prev=%v, want %d/%v/%v",
				tc.total, tc.page, m.TotalPages, m.HasNext, m.HasPrev, tc.wantPages, tc.wantNext, tc.wantPrev)
		}
	}
}
