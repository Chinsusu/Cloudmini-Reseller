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
	mw "github.com/pvp/pkg/middleware"
	pgpkg "github.com/pvp/pkg/postgres"
	"github.com/pvp/proxy-service/internal/events"
	httphandler "github.com/pvp/proxy-service/internal/handler/http"
	"github.com/pvp/proxy-service/internal/providers"
	proxycheap "github.com/pvp/proxy-service/internal/providers/proxy_cheap"
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

	// Proxy-Cheap credentials (optional — sandbox still works if not set)
	proxyCheapAPIKey    := getEnv("PROXY_CHEAP_API_KEY", "")
	proxyCheapAPISecret := getEnv("PROXY_CHEAP_API_SECRET", "")
	proxyCheapWHSecret  := getEnv("PROXY_CHEAP_WEBHOOK_SECRET", "")

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

	cipher, err := crypto.New(encKey)
	if err != nil {
		log.Error("cipher init", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Repos
	orderRepo    := repopg.NewOrderRepository(db)
	productRepo  := repopg.NewProductRepository(db)
	providerRepo := repopg.NewProviderRepository(db)

	// Events
	natsPub  := natspkg.NewPublisher(natsClient)
	eventPub := events.NewPublisher(natsPub)

	// Billing HTTP client
	billingClient := usecase.NewHTTPBillingClient(billingURL)

	// ── Provider registry ────────────────────────────────────────────────────
	registry := providers.NewRegistry()
	// Sandbox adapter — always registered for dev/test. UUID matches proxy.providers row.
	registry.Register("a1000000-0000-0000-0000-000000000001", providers.NewSandboxAdapter())
	log.Info("sandbox adapter registered")

	if proxyCheapAPIKey != "" && proxyCheapAPISecret != "" {
		pcAdapter := proxycheap.NewAdapter(proxycheap.Config{
			APIKey:        proxyCheapAPIKey,
			APISecret:     proxyCheapAPISecret,
			WebhookSecret: proxyCheapWHSecret,
		})
		registry.Register(proxycheap.ProviderName, pcAdapter)
		log.Info("proxy-cheap adapter registered")
	} else {
		log.Warn("PROXY_CHEAP_API_KEY/SECRET not set — proxy-cheap adapter disabled")
	}

	// ── Usecases ─────────────────────────────────────────────────────────────
	orderUC := usecase.NewOrderUsecase(
		orderRepo, productRepo, providerRepo,
		registry, billingClient, cipher, eventPub, log,
	)

	webhookUC := usecase.NewWebhookUsecase(
		orderRepo, productRepo, billingClient, cipher, eventPub, log,
	)

	// ── Webhook HTTP handler ─────────────────────────────────────────────────
	var webhookHTTP http.Handler
	if proxyCheapWHSecret != "" {
		pcClient := proxycheap.NewClient(proxyCheapAPIKey, proxyCheapAPISecret)
		pcWH := proxycheap.NewWebhookHandler(pcClient, proxyCheapWHSecret, webhookUC, log)
		webhookHTTP = http.HandlerFunc(pcWH.HandleEvent)
		log.Info("proxy-cheap webhook handler enabled")
	}

	// ── HTTP ──────────────────────────────────────────────────────────────────
	auditLogger := mw.NewNATSAuditLogger(natsPub, "proxy-service")
	handler := httphandler.NewHandler(orderUC, productRepo, providerRepo, webhookHTTP, log)
	if proxyCheapAPIKey != "" && proxyCheapAPISecret != "" {
		pcClient := proxycheap.NewClient(proxyCheapAPIKey, proxyCheapAPISecret)
		handler.WithProxyCheapClient(pcClient)
	}
	router  := httphandler.NewRouter(handler, jwtSecret, auditLogger)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

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
