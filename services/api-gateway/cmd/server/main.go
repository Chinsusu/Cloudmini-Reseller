package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/pvp/api-gateway/internal/config"
	"github.com/pvp/pkg/apierror"
	"github.com/pvp/pkg/logger"
	mw "github.com/pvp/pkg/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	r := chi.NewRouter()

	// ─── Middleware chain (follows doc 03-API-GATEWAY.md) ─────────────────────
	r.Use(mw.Recovery(log))
	r.Use(mw.RequestID)
	r.Use(mw.CORS)
	r.Use(chimw.RealIP)
	r.Use(chimw.Compress(5))

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"api-gateway"}`))
	})

	// Auth routes — forwarded without JWT validation
	r.Mount("/api/v1/auth", proxy(cfg.UserServiceURL))

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(cfg.JWTSecret))

		// User routes
		r.Mount("/api/v1/users", proxy(cfg.UserServiceURL))

		// Service routes
		r.Mount("/api/v1/proxy", proxy(cfg.ProxyServiceURL))
		r.Mount("/api/v1/vps", proxy(cfg.VPSServiceURL))
		r.Mount("/api/v1/billing", proxy(cfg.BillingServiceURL))
		r.Mount("/api/v1/logs", proxy(cfg.LogServiceURL))
		r.Mount("/api/v1/notifications", proxy(cfg.LogServiceURL))

		// Reseller routes
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("reseller", "admin", "super_admin"))
			r.Mount("/api/v1/reseller", proxy(cfg.ResellerServiceURL))
		})

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireRole("admin", "super_admin"))
			r.Mount("/api/v1/admin", adminRouter(cfg))
		})
	})

	// Access log for all routes
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.InfoContext(r.Context(), "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", mw.GetRequestID(r.Context())),
			)
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("api-gateway starting", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down api-gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("api-gateway stopped")
}

// proxy creates a reverse proxy to the target URL.
func proxy(target string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		panic("invalid proxy target: " + target)
	}

	p := httputil.NewSingleHostReverseProxy(u)
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		apierror.Respond(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE",
			"Upstream service is unavailable")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip mount prefix from path when forwarding
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "")
		p.ServeHTTP(w, r)
	})
}

// adminRouter routes admin sub-paths to respective services.
func adminRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Mount("/users", proxy(cfg.UserServiceURL))
	r.Mount("/proxy", proxy(cfg.ProxyServiceURL))
	r.Mount("/vps", proxy(cfg.VPSServiceURL))
	r.Mount("/billing", proxy(cfg.BillingServiceURL))
	r.Mount("/logs", proxy(cfg.LogServiceURL))
	r.Mount("/resellers", proxy(cfg.ResellerServiceURL))
	return r
}
