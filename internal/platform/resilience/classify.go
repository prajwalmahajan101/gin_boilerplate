package resilience

import (
	"context"
	"errors"
	"net"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

// retryableCodes are the apperr codes that represent a transient downstream
// failure worth retrying. Breaker-open (SERVICE_UNAVAILABLE) is deliberately
// excluded: once a breaker is OPEN, retrying only hammers a dependency that is
// already known to be down — the loop should stop and surface fast.
var retryableCodes = map[string]bool{
	apperr.CodeTransientError: true,
	apperr.CodeTimeout:        true,
	apperr.CodeExternalError:  true,
}

// Retryable reports whether err represents a transient failure that a retry
// could plausibly recover from. It is deliberately conservative: anything it
// does not recognise is treated as non-retryable (fail safe — never retry a
// validation error or a cancelled context).
func Retryable(err error) bool {
	if err == nil {
		return false
	}
	// A cancelled context is a caller decision, not a transient fault.
	if errors.Is(err, context.Canceled) {
		return false
	}
	// A deadline is transient from the dependency's point of view.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// Network timeouts (dial/read deadlines) are transient.
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		return retryableCodes[ae.Code]
	}
	return false
}
