// Package apperrors defines unified error codes shared between the
// Go server and the Dart client.
package apperrors

import "net/http"

// ErrorCode is a numeric error code sent to clients in error messages.
type ErrorCode int

const (
	// ErrOK is the zero value, meaning no error.
	ErrOK ErrorCode = iota
	// ErrInvalidToken means the authentication token is missing, expired, or invalid.
	ErrInvalidToken
	// ErrUserNotFound means the user does not exist.
	ErrUserNotFound
	// ErrAlreadyInCall means the user is already in an active call.
	ErrAlreadyInCall
	// ErrNotInPool means the user is not in any matching pool.
	ErrNotInPool
	// ErrPoolTimeout means the user waited too long in the pool.
	ErrPoolTimeout
	// ErrCallNotFound means the call ID does not exist.
	ErrCallNotFound
	// ErrCallRejected means the callee rejected the call.
	ErrCallRejected
	// ErrNetwork means a network-level failure occurred.
	ErrNetwork
	// ErrInternal means an internal server error.
	ErrInternal
	// ErrRateLimited means the client sent too many requests.
	ErrRateLimited
	// ErrBlacklisted means the caller and callee have blocked each other.
	ErrBlacklisted
	// ErrInvalidParams means the request parameters are invalid.
	ErrInvalidParams
)

// codeMessages maps each ErrorCode to a human-readable message.
var codeMessages = map[ErrorCode]string{
	ErrOK:            "ok",
	ErrInvalidToken:  "invalid or expired token",
	ErrUserNotFound:  "user not found",
	ErrAlreadyInCall: "user is already in a call",
	ErrNotInPool:     "user is not in any matching pool",
	ErrPoolTimeout:   "matching pool wait timeout",
	ErrCallNotFound:  "call not found",
	ErrCallRejected:  "call was rejected",
	ErrNetwork:       "network error",
	ErrInternal:      "internal server error",
	ErrRateLimited:   "too many requests",
	ErrBlacklisted:   "user is blacklisted by peer",
	ErrInvalidParams: "invalid request parameters",
}

// codeHTTPStatus maps each ErrorCode to an HTTP status code.
var codeHTTPStatus = map[ErrorCode]int{
	ErrOK:            http.StatusOK,
	ErrInvalidToken:  http.StatusUnauthorized,
	ErrUserNotFound:  http.StatusNotFound,
	ErrAlreadyInCall: http.StatusConflict,
	ErrNotInPool:     http.StatusBadRequest,
	ErrPoolTimeout:   http.StatusRequestTimeout,
	ErrCallNotFound:  http.StatusNotFound,
	ErrCallRejected:  http.StatusConflict,
	ErrNetwork:       http.StatusBadGateway,
	ErrInternal:      http.StatusInternalServerError,
	ErrRateLimited:   http.StatusTooManyRequests,
	ErrBlacklisted:   http.StatusForbidden,
	ErrInvalidParams: http.StatusBadRequest,
}

// Message returns the human-readable message for the error code.
func (c ErrorCode) Message() string {
	if msg, ok := codeMessages[c]; ok {
		return msg
	}
	return "unknown error"
}

// HTTPStatus returns the recommended HTTP status code for the error.
func (c ErrorCode) HTTPStatus() int {
	if status, ok := codeHTTPStatus[c]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// Int returns the integer representation of the error code.
func (c ErrorCode) Int() int {
	return int(c)
}
