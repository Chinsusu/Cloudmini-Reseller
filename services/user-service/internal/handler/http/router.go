package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	mw "github.com/pvp/pkg/middleware"
)

// NewRouter builds and returns the chi router for user-service.
func NewRouter(h *Handler, jwtSecret []byte) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(mw.CORS)
	r.Use(mw.RequestID)
	r.Use(mw.Recovery(h.logger))
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(5))

	// Health check (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"user-service"}`))
	})

	// Auth routes (no JWT required)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.RefreshToken)
		r.Post("/logout", h.Logout)
	})

	// Authenticated user routes
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(jwtSecret))

		// Own profile
		r.Route("/api/v1/users", func(r chi.Router) {
			r.Get("/me", h.GetMe)
			r.Put("/me", h.UpdateMe)
			r.Put("/me/password", h.ChangePassword)
			r.Get("/me/api-keys", h.ListAPIKeys)
			r.Post("/me/api-keys", h.CreateAPIKey)
			r.Delete("/me/api-keys/{id}", h.RevokeAPIKey)
		})

		// Admin routes
		r.Route("/api/v1/admin/users", func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "super_admin"))
			r.Get("/", h.AdminListUsers)
			r.Get("/{id}", h.AdminGetUser)
			r.Put("/{id}/status", h.AdminUpdateStatus)
			r.Put("/{id}/role", h.AdminUpdateRole)
		})
	})

	return r
}
