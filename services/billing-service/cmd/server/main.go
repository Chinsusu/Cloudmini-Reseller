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
	"github.com/pvp/billing-service/internal/config"
	"github.com/pvp/billing-service/internal/events"
	httphandler "github.com/pvp/billing-service/internal/handler/http"
	repopg "github.com/pvp/billing-service/internal/repository/postgres"
	"github.com/pvp/billing-service/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log := logger.New(cfg.LogLevel)

	db, err := pgpkg.Connect(pgpkg.Config{URL: cfg.DatabaseURL})
	if err != nil {
		log.Error("db connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	natsClient, err := natspkg.Connect(cfg.NatsURL)
	if err != nil {
		log.Error("nats connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer natsClient.Close()

	// Repos
	walletRepo  := repopg.NewWalletRepository(db)
	txnRepo     := repopg.NewTransactionRepository(db)
	paymentRepo := repopg.NewPaymentRepository(db)
	pricingRepo := repopg.NewPricingRepository(db)
	txRunner    := repopg.NewTxRunner(db)

	// Events
	natsPub  := natspkg.NewPublisher(natsClient)
	eventPub := events.NewPublisher(natsPub)

	// Usecases
	walletUC   := usecase.NewWalletUsecase(walletRepo, txnRepo, txRunner, eventPub, log)
	pricingEng := usecase.NewPricingEngine(pricingRepo, log)
	paymentUC  := usecase.NewPaymentUsecase(paymentRepo, walletUC, log, cfg.StripeSecretKey, cfg.FrontendBaseURL)

	// HTTP
	jwtSecret   := []byte(os.Getenv("JWT_SECRET"))
	auditLogger := mw.NewNATSAuditLogger(natsPub, "billing-service")
	handler     := httphandler.NewHandler(walletUC, paymentUC, pricingEng, log)
	router      := httphandler.NewRouter(handler, jwtSecret, auditLogger)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("billing-service starting", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("billing-service stopped")
}
