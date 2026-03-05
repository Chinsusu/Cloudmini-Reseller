// Package middleware provides shared HTTP middleware for all services.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ContextKey is a typed key for request-scoped values.
type ContextKey string

const (
	ContextKeyRequestID  ContextKey = "request_id"
	ContextKeyUserID     ContextKey = "user_id"
	ContextKeyUserRole   ContextKey = "user_role"
	ContextKeyResellerID ContextKey = "reseller_id"
)

// RequestID generates or forwards the X-Request-ID header and stores in context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ContextKeyRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recovery catches panics and returns a 500 response.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					requestID, _ := r.Context().Value(ContextKeyRequestID).(string)
					logger.ErrorContext(r.Context(), "panic recovered",
						slog.Any("error", rec),
						slog.String("request_id", requestID),
						slog.String("path", r.URL.Path),
					)
					http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS sets permissive CORS headers for local dev. Tighten in production.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, Idempotency-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Claims holds JWT claims for our access tokens.
type Claims struct {
	UserID     string `json:"sub"`
	Role       string `json:"role"`
	ResellerID string `json:"reseller_id,omitempty"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

// Auth validates the Bearer JWT and injects user claims into context and headers.
func Auth(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Authorization header required"}}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Invalid or expired token"}}`, http.StatusUnauthorized)
				return
			}

			// Inject into context and headers (for downstream services)
			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyUserRole, claims.Role)
			ctx = context.WithValue(ctx, ContextKeyResellerID, claims.ResellerID)

			r = r.WithContext(ctx)
			r.Header.Set("X-User-ID", claims.UserID)
			r.Header.Set("X-User-Role", claims.Role)
			r.Header.Set("X-Reseller-ID", claims.ResellerID)

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns 403 if the authenticated user's role is not in the allowed set.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ContextKeyUserRole).(string)
			if !allowed[role] {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"Insufficient permissions"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(ContextKeyRequestID).(string)
	return id
}

// GetUserID extracts the user ID from context.
func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(ContextKeyUserID).(string)
	return id
}

// GetUserRole extracts the user role from context.
func GetUserRole(ctx context.Context) string {
	role, _ := ctx.Value(ContextKeyUserRole).(string)
	return role
}
