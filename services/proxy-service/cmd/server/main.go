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
	"github.com/pvp/pkg/crypto"
	"github.com/pvp/pkg/logger"
	pgpkg "github.com/pvp/pkg/postgres"
	"github.com/pvp/proxy-service/internal/events"
	httphandler "github.com/pvp/proxy-service/internal/handler/http"
	"github.com/pvp/proxy-service/internal/providers"
	repopg "github.com/pvp/proxy-service/internal/repository/postgres"
	"github.com/pvp/proxy-service/internal/usecase"
)

func main() {
	port := getEnv("PORT", "8082")
	dbURL := requireEnv("DATABASE_URL")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	encKey := getEnv("ENCRYPTION_KEY", "0000000000000000000000000000000000000000000000000000000000000000")
	jwtSecret := []byte(getEnv("JWT_SECRET", ""))
	logLevel := getEnv("LOG_LEVEL", "info")
	billingURL := getEnv("BILLING_SERVICE_URL", "http://localhost:8085")

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

	// Encryption cipher for credentials
	cipher, err := crypto.New(encKey)
	if err != nil {
		log.Error("cipher init", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Provider registry — register sandbox provider
	registry := providers.NewRegistry()
	registry.Register("sandbox-provider-uuid", providers.NewSandboxAdapter())

	// Repos
	orderRepo   := repopg.NewOrderRepository(db)
	productRepo := repopg.NewProductRepository(db)
	providerRepo := repopg.NewProviderRepository(db)

	// Events
	natsPub  := natspkg.NewPublisher(natsClient)
	eventPub := events.NewPublisher(natsPub)

	// Billing HTTP client
	billingClient := usecase.NewHTTPBillingClient(billingURL)

	// Usecases
	orderUC := usecase.NewOrderUsecase(
		orderRepo, productRepo, providerRepo,
		registry, billingClient, cipher, eventPub, log,
	)

	// HTTP
	handler := httphandler.NewHandler(orderUC, log)
	router  := httphandler.NewRouter(handler, jwtSecret)

	srv := &http.Server{Addr: ":" + port, Handler: router, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("proxy-service starting", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("proxy-service stopped")
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
