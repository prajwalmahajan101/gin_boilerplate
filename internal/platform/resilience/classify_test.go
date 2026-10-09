package resilience

import (
	"context"
	"errors"
	"testing"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

type fakeTimeout struct{ to bool }

func (fakeTimeout) Error() string   { return "fake net error" }
func (f fakeTimeout) Timeout() bool { return f.to }
func (fakeTimeout) Temporary() bool { return false }

func TestRetryable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"transient", apperr.TransientError("boom"), true},
		{"timeout", apperr.Timeout("slow"), true},
		{"external", apperr.ExternalError("5xx"), true},
		{"service unavailable (breaker open)", apperr.ServiceUnavailable("dep"), false},
		{"validation", apperr.ValidationError("bad"), false},
		{"not found", apperr.NotFound("missing"), false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, true},
		{"net timeout", fakeTimeout{to: true}, true},
		{"net non-timeout", fakeTimeout{to: false}, false},
		{"unknown error", errors.New("mystery"), false},
		{"wrapped transient", apperr.Wrap(apperr.CodeTransientError, "w", 502, errors.New("x")), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Retryable(c.err); got != c.want {
				t.Fatalf("Retryable(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

// guard: a timeout-marked error must stay retryable even when wrapped deeper.
func TestRetryable_WrappedDeadline(t *testing.T) {
	err := apperr.Wrap(apperr.CodeExternalError, "call failed", 502, context.DeadlineExceeded)
	if !Retryable(err) {
		t.Fatal("wrapped external+deadline must be retryable")
	}
}
