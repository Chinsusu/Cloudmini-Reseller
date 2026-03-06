// Package apierror provides standard HTTP API error types and response helpers.
package apierror

import (
	"encoding/json"
	"net/http"
)

// Standard error codes (uppercase snake_case).
const (
	CodeValidationError    = "VALIDATION_ERROR"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodeInsufficientFunds  = "INSUFFICIENT_FUNDS"
	CodeProviderUnavailable = "PROVIDER_UNAVAILABLE"
	CodeNodeUnavailable    = "NODE_UNAVAILABLE"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	CodeInternalError      = "INTERNAL_ERROR"
)

// APIError is the standard error response body.
type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

// Respond writes a JSON error response.
func Respond(w http.ResponseWriter, r *http.Request, statusCode int, code, message string, details ...any) {
	requestID := r.Header.Get("X-Request-ID")
	e := errorEnvelope{
		Error: APIError{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	}
	if len(details) > 0 && details[0] != nil {
		e.Error.Details = details[0]
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(e)
}

// RespondJSON writes a generic JSON success response wrapped in { "data": ... }.
func RespondJSON(w http.ResponseWriter, statusCode int, data any) {
	env := map[string]any{"data": data}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(env)
}

// RespondJSONWithMeta writes a success response with meta field.
func RespondJSONWithMeta(w http.ResponseWriter, statusCode int, data, meta any) {
	env := map[string]any{"data": data, "meta": meta}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(env)
}
