package domain

import "errors"

// Domain errors — all prefixed with Err.
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountSuspended    = errors.New("account is suspended or banned")
	ErrEmailNotVerified    = errors.New("email not verified")
	ErrTokenExpired        = errors.New("token has expired")
	ErrTokenInvalid        = errors.New("token is invalid")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionRevoked      = errors.New("session has been revoked")
	ErrMaxSessionsReached  = errors.New("maximum active sessions reached")
	ErrAPIKeyNotFound      = errors.New("api key not found")
	ErrAPIKeyRevoked       = errors.New("api key has been revoked")
	ErrForbidden           = errors.New("insufficient permissions")
	ErrWeakPassword        = errors.New("password does not meet complexity requirements")
)
