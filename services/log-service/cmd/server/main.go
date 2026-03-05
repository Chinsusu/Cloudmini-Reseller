// Package main runs the log-service: consumes NATS events, persists them
// to PostgreSQL (partitioned), and streams via WebSocket.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	natspkg "github.com/pvp/pkg/nats"
	"github.com/pvp/pkg/logger"
	"github.com/pvp/pkg/middleware"
	"github.com/pvp/pkg/apierror"
	"github.com/pvp/pkg/pagination"
	pgpkg "github.com/pvp/pkg/postgres"
	_ "github.com/lib/pq"
)

// LogEntry mirrors logs.entries table columns.
type LogEntry struct {
	ID           string         `db:"id"             json:"id"`
	RequestID    *string        `db:"request_id"     json:"request_id,omitempty"`
	ServiceName  string         `db:"service_name"   json:"service_name"`
	UserID       *string        `db:"user_id"        json:"user_id,omitempty"`
	ResellerID   *string        `db:"reseller_id"    json:"reseller_id,omitempty"`
	ActorType    string         `db:"actor_type"     json:"actor_type"`
	Action       string         `db:"action"         json:"action"`
	Level        string         `db:"level"          json:"level"`
	ResourceType *string        `db:"resource_type"  json:"resource_type,omitempty"`
	ResourceID   *string        `db:"resource_id"    json:"resource_id,omitempty"`
	Message      string         `db:"message"        json:"message"`
	DurationMS   *int           `db:"duration_ms"    json:"duration_ms,omitempty"`
	IPAddress    *string        `db:"ip_address"     json:"ip_address,omitempty"`
	CreatedAt    time.Time      `db:"created_at"     json:"created_at"`
}

// ─── WebSocket Hub ────────────────────────────────────────────────────────────

// wsClient represents a connected WebSocket subscriber.
type wsClient struct {
	conn   *websocket.Conn
	send   chan []byte
	userID string
	role   string
}

// Hub manages all WebSocket connections.
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
}

func newHub() *Hub { return &Hub{clients: make(map[*wsClient]bool)} }

func (h *Hub) register(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) unregister(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	_ = c.conn.Close()
}

// Broadcast sends an event to all connected clients that are allowed to see it.
func (h *Hub) Broadcast(entry *LogEntry) {
	data, _ := json.Marshal(entry)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		// Admins see everything; users only see their own events
		if c.role == "admin" || c.role == "super_admin" {
			select {
			case c.send <- data:
			default:
			}
			continue
		}
		if entry.UserID != nil && *entry.UserID == c.userID {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // tighten in prod
}

// ─── Application ─────────────────────────────────────────────────────────────

func main() {
	port := getEnv("PORT", "8087")
	dbURL := requireEnv("DATABASE_URL")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	jwtSecret := []byte(getEnv("JWT_SECRET", ""))
	logLevel := getEnv("LOG_LEVEL", "info")

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

	hub := newHub()

	// Start NATS consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	go runConsumer(ctx, natsClient, db, hub, log)

	// HTTP router
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.CORS, middleware.Recovery(log))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierror.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "log-service"})
	})

	// WebSocket endpoint
	r.Get("/ws/events", func(w http.ResponseWriter, r *http.Request) {
		// JWT auth via query param (since WS doesn't support headers easily)
		// In production: validate the token here before upgrading
		userID := r.URL.Query().Get("user_id")
		role := r.URL.Query().Get("role")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Warn("ws upgrade failed", slog.String("error", err.Error()))
			return
		}

		client := &wsClient{
			conn:   conn,
			send:   make(chan []byte, 256),
			userID: userID,
			role:   role,
		}
		hub.register(client)
		defer hub.unregister(client)

		// Write pump
		go func() {
			for msg := range client.send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		// Read pump (keeps connection alive)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})

	// REST log query endpoint
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		r.Get("/api/v1/logs", func(w http.ResponseWriter, r *http.Request) {
			p := pagination.Parse(r)
			userID := middleware.GetUserID(r.Context())
			role := middleware.GetUserRole(r.Context())

			var entries []*LogEntry
			var total int
			var queryErr error

			if role == "admin" || role == "super_admin" {
				entries, total, queryErr = listAllLogs(r.Context(), db, p.Offset, p.Limit)
			} else {
				entries, total, queryErr = listUserLogs(r.Context(), db, userID, p.Offset, p.Limit)
			}
			if queryErr != nil {
				apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "failed to list logs")
				return
			}
			apierror.RespondJSONWithMeta(w, http.StatusOK, entries, pagination.NewMeta(p, total))
		})
	})

	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("log-service starting", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", slog.String("error", err.Error()))
		}
	}()

	<-quit
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	log.Info("log-service stopped")
}

// ─── NATS Consumer ────────────────────────────────────────────────────────────

func runConsumer(ctx context.Context, client *natspkg.Client, db *sqlx.DB, hub *Hub, log *slog.Logger) {
	// Ensure stream exists for all service subjects
	if err := client.CreateOrUpdateStream(ctx, "PVP_EVENTS", []string{
		"user.>", "billing.>", "proxy.>", "vps.>", "vm.>",
	}); err != nil {
		log.Error("create stream", slog.String("error", err.Error()))
		return
	}

	consumer, err := client.CreateOrUpdateConsumer(ctx, natspkg.ConsumerConfig{
		Stream:       "PVP_EVENTS",
		ConsumerName: "log-service-consumer",
		Subjects:     []string{"user.>", "billing.>", "proxy.>", "vps.>", "vm.>"},
		MaxDeliver:   3,
	})
	if err != nil {
		log.Error("create consumer", slog.String("error", err.Error()))
		return
	}

	handler := func(ctx context.Context, subject string, data []byte) error {
		entry := &LogEntry{
			ServiceName: serviceFromSubject(subject),
			ActorType:   "system",
			Action:      subject,
			Level:       "INFO",
			Message:     fmt.Sprintf("Event: %s", subject),
			CreatedAt:   time.Now(),
		}

		// Try to extract user_id from payload
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err == nil {
			if uid, ok := payload["user_id"].(string); ok {
				entry.UserID = &uid
			}
		}

		if err := persistLog(ctx, db, entry); err != nil {
			log.Warn("persist log failed", slog.String("subject", subject), slog.String("error", err.Error()))
			return err // trigger Nak → retry
		}

		hub.Broadcast(entry)
		return nil
	}

	if err := natspkg.StartConsumer(ctx, consumer, handler); err != nil {
		if ctx.Err() == nil {
			log.Error("consumer stopped", slog.String("error", err.Error()))
		}
	}
}

func persistLog(ctx context.Context, db *sqlx.DB, e *LogEntry) error {
	q := `INSERT INTO logs.entries
		(service_name, user_id, actor_type, action, level, message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.ExecContext(ctx, q, e.ServiceName, e.UserID, e.ActorType, e.Action, e.Level, e.Message, e.CreatedAt)
	return err
}

func listAllLogs(ctx context.Context, db *sqlx.DB, offset, limit int) ([]*LogEntry, int, error) {
	var total int
	_ = db.GetContext(ctx, &total, `SELECT COUNT(*) FROM logs.entries`)
	var entries []*LogEntry
	err := db.SelectContext(ctx, &entries, `SELECT * FROM logs.entries ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	return entries, total, err
}

func listUserLogs(ctx context.Context, db *sqlx.DB, userID string, offset, limit int) ([]*LogEntry, int, error) {
	var total int
	_ = db.GetContext(ctx, &total, `SELECT COUNT(*) FROM logs.entries WHERE user_id=$1`, userID)
	var entries []*LogEntry
	err := db.SelectContext(ctx, &entries, `SELECT * FROM logs.entries WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	return entries, total, err
}

func serviceFromSubject(subject string) string {
	parts := []rune(subject)
	for i, c := range parts {
		if c == '.' {
			return string(parts[:i])
		}
	}
	return subject
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
		panic("required env var not set: " + key)
	}
	return v
}
