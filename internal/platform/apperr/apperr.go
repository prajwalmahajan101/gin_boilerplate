package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Details    map[string]any
	RequestID  string
	cause      error
	trips      bool
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return e.Code + ": " + e.Message
}

func (e *AppError) Unwrap() error { return e.cause }

const (
	CodeValidationError = "VALIDATION_ERROR"
	CodeInvalidInput    = "INVALID_INPUT"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodePayloadTooLarge = "PAYLOAD_TOO_LARGE"
	CodeRateLimited     = "RATE_LIMITED"
	CodeInternalError   = "INTERNAL_ERROR"
	CodeExternalError   = "EXTERNAL_ERROR"
	CodeTransientError  = "TRANSIENT_ERROR"
	CodeTimeout         = "TIMEOUT"
	CodeServiceUnavail  = "SERVICE_UNAVAILABLE"
)

func New(code, msg string, status int) *AppError {
	return &AppError{Code: code, Message: msg, HTTPStatus: status}
}

func Newf(code string, status int, format string, args ...any) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), HTTPStatus: status}
}

func Wrap(code, msg string, status int, cause error) *AppError {
	return &AppError{Code: code, Message: msg, HTTPStatus: status, cause: cause}
}

func (e *AppError) WithDetails(d map[string]any) *AppError {
	e.Details = d
	return e
}

func (e *AppError) WithRequestID(id string) *AppError {
	e.RequestID = id
	return e
}

// Per-code convenience constructors.

func ValidationError(msg string) *AppError {
	return &AppError{Code: CodeValidationError, Message: msg, HTTPStatus: http.StatusBadRequest}
}

func InvalidInput(msg string) *AppError {
	return &AppError{Code: CodeInvalidInput, Message: msg, HTTPStatus: http.StatusBadRequest}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg, HTTPStatus: http.StatusUnauthorized}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg, HTTPStatus: http.StatusForbidden}
}

func NotFound(msg string) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg, HTTPStatus: http.StatusNotFound}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: CodeConflict, Message: msg, HTTPStatus: http.StatusConflict}
}

func PayloadTooLarge(msg string) *AppError {
	return &AppError{Code: CodePayloadTooLarge, Message: msg, HTTPStatus: http.StatusRequestEntityTooLarge}
}

func RateLimited(msg string) *AppError {
	return &AppError{Code: CodeRateLimited, Message: msg, HTTPStatus: http.StatusTooManyRequests}
}

func ExternalError(msg string) *AppError {
	return &AppError{Code: CodeExternalError, Message: msg, HTTPStatus: http.StatusBadGateway}
}

func TransientError(msg string) *AppError {
	return &AppError{Code: CodeTransientError, Message: msg, HTTPStatus: http.StatusBadGateway, trips: true}
}

func Timeout(msg string) *AppError {
	return &AppError{Code: CodeTimeout, Message: msg, HTTPStatus: http.StatusBadGateway, trips: true}
}

func ServiceUnavailable(msg string) *AppError {
	return &AppError{Code: CodeServiceUnavail, Message: msg, HTTPStatus: http.StatusServiceUnavailable}
}

func InternalError(cause error) *AppError {
	return &AppError{Code: CodeInternalError, Message: "internal error", HTTPStatus: http.StatusInternalServerError, cause: cause}
}

// AsAppError unwraps err to *AppError, or wraps it as INTERNAL_ERROR.
func AsAppError(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return InternalError(err)
}

// TripsBreaker reports whether err should count toward opening a circuit breaker.
func TripsBreaker(err error) bool {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.trips
	}
	return false
}

// Is checks whether err (or any wrapped error) has the given code.
func Is(err error, code string) bool {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code == code
	}
	return false
}
