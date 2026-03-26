package main

import (
	"context"
	"encoding/json"
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
	"github.com/pvp/proxy-service/internal/providers/vpm"
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

	// VPS Proxy Manager credentials (optional)
	vpmBaseURL := getEnv("VPM_BASE_URL", "")
	vpmAPIKey  := getEnv("VPM_API_KEY", "")

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
	orderRepo     := repopg.NewOrderRepository(db)
	orderEvtRepo  := repopg.NewOrderEventRepository(db)
	productRepo   := repopg.NewProductRepository(db)
	providerRepo  := repopg.NewProviderRepository(db)

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

	// VPM adapters — load all active VPM providers from DB.
	// Each provider row with adapter_type='vpm' gets its own adapter instance.
	vpmProviders, err := providerRepo.ListByAdapterType(context.Background(), "vpm")
	if err != nil {
		log.Error("failed to load VPM providers from DB", slog.String("error", err.Error()))
	}
	if len(vpmProviders) > 0 {
		for _, p := range vpmProviders {
			var cfg vpm.Config
			if jsonErr := json.Unmarshal(p.Config, &cfg); jsonErr != nil {
				log.Error("invalid VPM provider config", slog.String("provider_id", p.ID.String()), slog.String("error", jsonErr.Error()))
				continue
			}
			if cfg.BaseURL == "" {
				log.Warn("VPM provider has empty base_url, skipping", slog.String("provider_id", p.ID.String()))
				continue
			}
			adapter := vpm.NewAdapter(cfg)
			registry.Register(p.ID.String(), adapter)
			log.Info("vpm adapter registered",
				slog.String("provider_id", p.ID.String()),
				slog.String("name", p.Name),
				slog.String("display_name", p.DisplayName),
				slog.String("base_url", cfg.BaseURL),
			)
		}
	} else if vpmBaseURL != "" && vpmAPIKey != "" {
		// Fallback: register from env vars (legacy single-instance mode)
		vpmAdapter := vpm.NewAdapter(vpm.Config{
			BaseURL: vpmBaseURL,
			APIKey:  vpmAPIKey,
		})
		registry.Register("b2000000-0000-0000-0000-000000000002", vpmAdapter)
		log.Info("vpm adapter registered (env fallback)", slog.String("base_url", vpmBaseURL))
	} else {
		log.Warn("no VPM providers configured")
	}


	// ── Usecases ─────────────────────────────────────────────────────────────
	orderUC := usecase.NewOrderUsecase(
		orderRepo, productRepo, providerRepo,
		registry, billingClient, cipher, eventPub, orderEvtRepo, log,
	)

	// Expiry lifecycle: check expired orders every 15 minutes
	expiryUC := usecase.NewExpiryUsecase(
		orderRepo, registry, orderEvtRepo, usecase.DefaultGracePeriod, log,
	)

	// Admin lock/unlock
	lockUC := usecase.NewLockUsecase(orderRepo, orderEvtRepo, registry, log)

	webhookUC := usecase.NewWebhookUsecase(
		orderRepo, productRepo, billingClient, cipher, eventPub, orderEvtRepo, log,
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
	handler := httphandler.NewHandler(orderUC, lockUC, orderRepo, orderEvtRepo, productRepo, providerRepo, webhookHTTP, log)
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

	// Background expiry scheduler — runs every 15 minutes
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		log.Info("expiry scheduler started", slog.String("grace_period", usecase.DefaultGracePeriod.String()))
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				bgCtx := context.Background()
				if err := expiryUC.ProcessExpired(bgCtx); err != nil {
					log.Error("expiry scheduler: ProcessExpired", slog.String("error", err.Error()))
				}
				if err := expiryUC.ProcessGraceExpired(bgCtx); err != nil {
					log.Error("expiry scheduler: ProcessGraceExpired", slog.String("error", err.Error()))
				}
			}
		}
	}()

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
