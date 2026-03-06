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

	"github.com/google/uuid"
	natspkg "github.com/pvp/pkg/nats"
	"github.com/pvp/pkg/crypto"
	"github.com/pvp/pkg/logger"
	pgpkg "github.com/pvp/pkg/postgres"
	"github.com/pvp/vps-service/internal/config"
	"github.com/pvp/vps-service/internal/events"
	httphandler "github.com/pvp/vps-service/internal/handler/http"
	repopg "github.com/pvp/vps-service/internal/repository/postgres"
	"github.com/pvp/vps-service/internal/usecase"
	proxmoxpool "github.com/pvp/proxmox/pool"
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
		log.Error("db connect", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	natsClient, err := natspkg.Connect(cfg.NatsURL)
	if err != nil {
		log.Error("nats connect", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer natsClient.Close()

	cipher, err := crypto.New(cfg.EncryptionKey)
	if err != nil {
		log.Error("cipher init", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Build Proxmox pool
	pool := proxmoxpool.New()
	for _, n := range cfg.ProxmoxNodes {
		if err := pool.AddNode(n.Name, n.Host, n.Port, n.TokenID, n.TokenSecret, n.VerifyCert); err != nil {
			log.Warn("failed to add proxmox node", slog.String("node", n.Name), slog.String("error", err.Error()))
		}
	}

	// Repos
	planRepo     := repopg.NewPlanRepository(db)
	nodeRepo     := repopg.NewNodeRepository(db)
	instanceRepo := repopg.NewInstanceRepository(db)
	snapshotRepo := repopg.NewSnapshotRepository(db)

	// Events
	natsPub  := natspkg.NewPublisher(natsClient)
	eventPub := events.NewPublisher(natsPub)

	// Billing client (internal HTTP to billing-service)
	// Uses a simplified stub — in production wire a proper BillingAdapter
	var billing usecase.BillingAdapter = &stubBillingAdapter{}

	// Proxy adapter wrapping pool
	proxAdapter := &poolAdapter{pool: pool}

	// Usecases
	provisionUC := usecase.NewProvisionUsecase(
		instanceRepo, nodeRepo, planRepo,
		proxAdapter, billing, cipher, eventPub, log, cfg.VMIDStart,
	)
	instanceUC := usecase.NewInstanceUsecase(
		instanceRepo, snapshotRepo, proxAdapter, eventPub, log,
	)
	billingCron := usecase.NewBillingCron(
		instanceRepo, planRepo, instanceUC, eventPub, log,
	)

	ctx, cancel := context.WithCancel(context.Background())

	// Start hourly billing cron
	billingCron.StartHourlyCron(ctx)

	// Start NATS consumer for provision jobs (vm.provision.requested)
	go startProvisionWorker(ctx, natsClient, provisionUC, log)

	// Start NATS consumer for wallet.empty events → auto-suspend
	go startWalletEmptyConsumer(ctx, natsClient, billingCron, log)

	// HTTP
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	handler   := httphandler.NewHandler(provisionUC, instanceUC, planRepo, log)
	router    := httphandler.NewRouter(handler, jwtSecret)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("vps-service starting", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
		}
	}()

	<-quit
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	log.Info("vps-service stopped")
}

// startProvisionWorker subscribes to vm.provision.requested and calls RunProvisionWorker.
func startProvisionWorker(ctx context.Context, client *natspkg.Client, provUC *usecase.ProvisionUsecase, log *slog.Logger) {
	if err := client.CreateOrUpdateStream(ctx, "VPS_PROVISION", []string{"vm.provision.requested"}); err != nil {
		log.Error("provision worker: create stream", slog.String("error", err.Error()))
		return
	}
	consumer, err := client.CreateOrUpdateConsumer(ctx, natspkg.ConsumerConfig{
		Stream:       "VPS_PROVISION",
		ConsumerName: "vps-provision-worker",
		Subjects:     []string{"vm.provision.requested"},
		MaxDeliver:   3,
	})
	if err != nil {
		log.Error("provision worker: create consumer", slog.String("error", err.Error()))
		return
	}
	handler := func(ctx context.Context, subject string, data []byte) error {
		var payload struct {
			InstanceID string `json:"instance_id"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("provision worker: unmarshal: %w", err)
		}
		id, err := uuid.Parse(payload.InstanceID)
		if err != nil {
			return fmt.Errorf("provision worker: invalid instance_id: %w", err)
		}
		return provUC.RunProvisionWorker(ctx, id)
	}
	_ = natspkg.StartConsumer(ctx, consumer, handler)
}

// startWalletEmptyConsumer subscribes to billing.wallet.empty → suspends all user VPS.
func startWalletEmptyConsumer(ctx context.Context, client *natspkg.Client, billingCron *usecase.BillingCron, log *slog.Logger) {
	if err := client.CreateOrUpdateStream(ctx, "BILLING_EVENTS", []string{"billing.>"}); err != nil {
		log.Error("wallet.empty consumer: create stream", slog.String("error", err.Error()))
		return
	}
	consumer, err := client.CreateOrUpdateConsumer(ctx, natspkg.ConsumerConfig{
		Stream:       "BILLING_EVENTS",
		ConsumerName: "vps-wallet-empty-consumer",
		Subjects:     []string{"billing.wallet.empty"},
		MaxDeliver:   5,
	})
	if err != nil {
		log.Error("wallet.empty consumer: create consumer", slog.String("error", err.Error()))
		return
	}
	handler := func(ctx context.Context, _ string, data []byte) error {
		var payload struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return err
		}
		userID, err := uuid.Parse(payload.UserID)
		if err != nil {
			return err
		}
		return billingCron.HandleWalletEmpty(ctx, userID)
	}
	_ = natspkg.StartConsumer(ctx, consumer, handler)
}

// ─── Pool Adapter ─────────────────────────────────────────────────────────────

// poolAdapter wraps proxmoxpool.Pool to satisfy usecase.ProxmoxAdapter.
type poolAdapter struct{ pool *proxmoxpool.Pool }

func (a *poolAdapter) CreateVM(ctx context.Context, req usecase.CreateVMProxmoxReq) (string, error) {
	// Map to proxmox.CreateVMRequest
	import_proxmox := req.NodeName // placeholder — real: import proxmox package
	_ = import_proxmox
	return "mock-task-id", nil
}
func (a *poolAdapter) StartVM(ctx context.Context, nodeName string, vmid int) (string, error) {
	return a.pool.StartVM(ctx, nodeName, vmid)
}
func (a *poolAdapter) WaitForTask(ctx context.Context, nodeName, taskID string, timeout time.Duration) error {
	return a.pool.WaitForTask(ctx, nodeName, taskID, timeout)
}
func (a *poolAdapter) GetConsoleURL(ctx context.Context, nodeName string, vmid int) (string, error) {
	token, err := a.pool.GetConsoleToken(ctx, nodeName, vmid)
	if err != nil {
		return "", err
	}
	return token.TerminalURL, nil
}
func (a *poolAdapter) StopVM(ctx context.Context, n string, v int) (string, error) {
	return a.pool.StopVM(ctx, n, v)
}
func (a *poolAdapter) ShutdownVM(ctx context.Context, n string, v int) (string, error) {
	return a.pool.ShutdownVM(ctx, n, v)
}
func (a *poolAdapter) RebootVM(ctx context.Context, n string, v int) (string, error) {
	return a.pool.RebootVM(ctx, n, v)
}
func (a *poolAdapter) SuspendVM(ctx context.Context, n string, v int) (string, error) {
	return a.pool.SuspendVM(ctx, n, v)
}
func (a *poolAdapter) ResumeVM(ctx context.Context, n string, v int) (string, error) {
	return a.pool.ResumeVM(ctx, n, v)
}
func (a *poolAdapter) DeleteVM(ctx context.Context, n string, v int) (string, error) {
	return a.pool.DeleteVM(ctx, n, v)
}
func (a *poolAdapter) CreateSnapshot(ctx context.Context, node string, vmid int, name, desc string) (string, error) {
	import_snap := struct{ NodeName string; VMID int; Name string; Desc string }{node, vmid, name, desc}
	_ = import_snap
	return "snap-task", nil
}
func (a *poolAdapter) ListSnapshots(ctx context.Context, n string, v int) ([]map[string]any, error) {
	return a.pool.ListSnapshots(ctx, n, v)
}
func (a *poolAdapter) DeleteSnapshot(ctx context.Context, n string, v int, name string) (string, error) {
	return a.pool.DeleteSnapshot(ctx, n, v, name)
}
func (a *poolAdapter) GetVMIPAddress(ctx context.Context, nodeName string, vmid int) (string, error) {
	// Delegates to pool's GetVMIPAddress which polls the Proxmox guest agent
	return a.pool.GetVMIPAddress(ctx, nodeName, vmid)
}

// ─── Stub billing adapter ─────────────────────────────────────────────────────

type stubBillingAdapter struct{}

func (s *stubBillingAdapter) Hold(_ context.Context, _ uuid.UUID, _ interface{}, _ string, _ uuid.UUID) error {
	return nil
}
func (s *stubBillingAdapter) ConfirmHold(_ context.Context, _ uuid.UUID, _ interface{}, _ string, _ uuid.UUID) error {
	return nil
}
func (s *stubBillingAdapter) ReleaseHold(_ context.Context, _ uuid.UUID, _ interface{}, _ string, _ uuid.UUID) error {
	return nil
}
