package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(CodeNotFound, "item missing", http.StatusNotFound)
	if e.Code != CodeNotFound || e.Message != "item missing" || e.HTTPStatus != 404 {
		t.Fatalf("got %v", e)
	}
}

func TestNewf(t *testing.T) {
	e := Newf(CodeConflict, http.StatusConflict, "dup %s", "abc")
	if e.Message != "dup abc" || e.HTTPStatus != 409 {
		t.Fatalf("got %q status %d", e.Message, e.HTTPStatus)
	}
}

func TestWrapUnwrap(t *testing.T) {
	cause := errors.New("db down")
	e := Wrap(CodeInternalError, "oops", http.StatusInternalServerError, cause)
	if !errors.Is(e, cause) {
		t.Fatal("cause not reachable via errors.Is")
	}
	if e.Unwrap() != cause {
		t.Fatal("Unwrap mismatch")
	}
}

func TestErrorString(t *testing.T) {
	plain := NotFound("gone")
	if plain.Error() != "NOT_FOUND: gone" {
		t.Fatalf("got %q", plain.Error())
	}
	wrapped := Wrap(CodeInternalError, "fail", http.StatusInternalServerError, errors.New("io"))
	want := "INTERNAL_ERROR: fail: io"
	if wrapped.Error() != want {
		t.Fatalf("got %q, want %q", wrapped.Error(), want)
	}
}

func TestWithDetails(t *testing.T) {
	e := ValidationError("bad").WithDetails(map[string]any{"field": "name"})
	if e.Details["field"] != "name" {
		t.Fatalf("got %v", e.Details)
	}
}

func TestAsAppError(t *testing.T) {
	ae := NotFound("x")
	if AsAppError(ae) != ae {
		t.Fatal("should return same pointer")
	}
	plain := errors.New("boom")
	got := AsAppError(plain)
	if got.Code != CodeInternalError {
		t.Fatalf("got code %s", got.Code)
	}
	if got.Unwrap() != plain {
		t.Fatal("should wrap original")
	}
}

func TestAsAppErrorWrapped(t *testing.T) {
	inner := NotFound("x")
	outer := fmt.Errorf("wrap: %w", inner)
	got := AsAppError(outer)
	if got.Code != CodeNotFound {
		t.Fatalf("got code %s", got.Code)
	}
}

func TestHTTPStatusOnStruct(t *testing.T) {
	cases := []struct {
		err    *AppError
		status int
	}{
		{ValidationError("x"), http.StatusBadRequest},
		{InvalidInput("x"), http.StatusBadRequest},
		{Unauthorized("x"), http.StatusUnauthorized},
		{Forbidden("x"), http.StatusForbidden},
		{NotFound("x"), http.StatusNotFound},
		{Conflict("x"), http.StatusConflict},
		{PayloadTooLarge("x"), http.StatusRequestEntityTooLarge},
		{RateLimited("x"), http.StatusTooManyRequests},
		{InternalError(errors.New("x")), http.StatusInternalServerError},
		{ExternalError("x"), http.StatusBadGateway},
		{TransientError("x"), http.StatusBadGateway},
		{Timeout("x"), http.StatusBadGateway},
		{ServiceUnavailable("x"), http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		if tc.err.HTTPStatus != tc.status {
			t.Errorf("%s: got %d, want %d", tc.err.Code, tc.err.HTTPStatus, tc.status)
		}
	}
}

func TestTripsBreaker(t *testing.T) {
	if !TripsBreaker(TransientError("x")) {
		t.Fatal("TRANSIENT should trip")
	}
	if !TripsBreaker(Timeout("x")) {
		t.Fatal("TIMEOUT should trip")
	}
	if TripsBreaker(NotFound("x")) {
		t.Fatal("NOT_FOUND should not trip")
	}
	if TripsBreaker(errors.New("plain")) {
		t.Fatal("plain error should not trip")
	}
}

func TestIs(t *testing.T) {
	e := NotFound("x")
	if !Is(e, CodeNotFound) {
		t.Fatal("should match")
	}
	if Is(e, CodeConflict) {
		t.Fatal("should not match")
	}
	wrapped := fmt.Errorf("outer: %w", e)
	if !Is(wrapped, CodeNotFound) {
		t.Fatal("should match through wrapping")
	}
	if Is(errors.New("plain"), CodeNotFound) {
		t.Fatal("plain error should not match")
	}
}
