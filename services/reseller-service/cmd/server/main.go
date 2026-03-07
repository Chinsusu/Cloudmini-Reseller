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
	"github.com/pvp/reseller-service/internal/events"
	httphandler "github.com/pvp/reseller-service/internal/handler/http"
	repopg "github.com/pvp/reseller-service/internal/repository/postgres"
	"github.com/pvp/reseller-service/internal/usecase"
)

func main() {
	port      := getEnv("PORT", "8084")
	dbURL     := requireEnv("DATABASE_URL")
	natsURL   := getEnv("NATS_URL", "nats://localhost:4222")
	jwtSecret := []byte(getEnv("JWT_SECRET", ""))
	logLevel  := getEnv("LOG_LEVEL", "info")

	log := logger.New(logLevel)

	db, err := pgpkg.Connect(pgpkg.Config{URL: dbURL})
	if err != nil {
		log.Error("db connect", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	natsClient, err := natspkg.Connect(natsURL)
	if err != nil {
		log.Error("nats connect", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer natsClient.Close()

	// Repos
	resellerRepo   := repopg.NewResellerRepository(db)
	pricingRepo    := repopg.NewPricingRepository(db)
	apiKeyRepo     := repopg.NewAPIKeyRepository(db)
	subAccountRepo := repopg.NewSubAccountRepository(db)
	webhookRepo    := repopg.NewWebhookRepository(db)

	// Events
	natsPub  := natspkg.NewPublisher(natsClient)
	eventPub := events.NewPublisher(natsPub)

	// Usecases
	resellerUC := usecase.NewResellerUsecase(resellerRepo, pricingRepo, subAccountRepo, eventPub, log)
	apiKeyUC   := usecase.NewAPIKeyUsecase(apiKeyRepo, log)
	webhookUC  := usecase.NewWebhookUsecase(webhookRepo, log)

	// HTTP
	auditLogger := mw.NewNATSAuditLogger(natsPub, "reseller-service")
	handler := httphandler.NewHandler(resellerUC, apiKeyUC, webhookUC, log)
	router  := httphandler.NewRouter(handler, jwtSecret, auditLogger)

	srv := &http.Server{Addr: ":" + port, Handler: router, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("reseller-service starting", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("reseller-service stopped")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var not set: %s", key))
	}
	return v
}
