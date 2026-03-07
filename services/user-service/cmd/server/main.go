package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	natspkg "github.com/pvp/pkg/nats"
	"github.com/pvp/pkg/logger"
	mw "github.com/pvp/pkg/middleware"
	pgpkg "github.com/pvp/pkg/postgres"
	"github.com/pvp/user-service/internal/config"
	"github.com/pvp/user-service/internal/events"
	httphandler "github.com/pvp/user-service/internal/handler/http"
	repopg "github.com/pvp/user-service/internal/repository/postgres"
	"github.com/pvp/user-service/internal/usecase"
)

func main() {
	// ─── Config ───────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	// ─── Database ─────────────────────────────────────────────────────────────
	db, err := pgpkg.Connect(pgpkg.Config{URL: cfg.DatabaseURL})
	if err != nil {
		log.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()
	log.Info("database connected")

	// ─── NATS ─────────────────────────────────────────────────────────────────
	natsClient, err := natspkg.Connect(cfg.NatsURL)
	if err != nil {
		log.Error("failed to connect to NATS", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer natsClient.Close()
	log.Info("NATS connected")

	// ─── Dependency Injection ─────────────────────────────────────────────────
	// Repositories
	accountRepo := repopg.NewAccountRepository(db)
	sessionRepo := repopg.NewSessionRepository(db)
	apiKeyRepo  := repopg.NewAPIKeyRepository(db)

	// Event publisher
	natsPub  := natspkg.NewPublisher(natsClient)
	eventPub := events.NewPublisher(natsPub)

	// Usecases
	authUC := usecase.NewAuthUsecase(
		accountRepo, sessionRepo, eventPub, log,
		cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, cfg.JWTAdminRefreshTTL,
		cfg.MaxSessionsPerUser, cfg.MaxLoginAttempts,
	)
	userUC   := usecase.NewUserUsecase(accountRepo, eventPub, log)
	apiKeyUC := usecase.NewAPIKeyUsecase(apiKeyRepo, log)

	// Audit logger
	auditLogger := mw.NewNATSAuditLogger(natsPub, "user-service")

	// Handler + Router
	handler := httphandler.NewHandler(authUC, userUC, apiKeyUC, log)
	router  := httphandler.NewRouter(handler, cfg.JWTSecret, auditLogger)

	// ─── HTTP Server ──────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("user-service starting", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down user-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", slog.String("error", err.Error()))
	}

	log.Info("user-service stopped")
}
