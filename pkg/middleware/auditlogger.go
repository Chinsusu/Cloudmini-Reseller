// auditlogger.go provides an HTTP audit logging middleware and NATS-based implementation.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	natspkg "github.com/pvp/pkg/nats"
)

// HTTPLogEntry holds captured request/response data for audit logging.
type HTTPLogEntry struct {
	ServiceName string `json:"service_name"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	StatusCode  int    `json:"status_code"`
	DurationMS  int64  `json:"duration_ms"`
	UserID      string `json:"user_id,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
}

// AuditLogger is the interface for publishing HTTP audit events.
type AuditLogger interface {
	LogHTTP(ctx context.Context, entry HTTPLogEntry)
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// AuditLog returns an HTTP middleware that logs all mutating requests and 4xx/5xx responses.
// Log condition: method ∈ {POST, PUT, PATCH, DELETE} OR status >= 400.
func AuditLog(logger AuditLogger) func(http.Handler) http.Handler {
	mutating := map[string]bool{
		http.MethodPost:   true,
		http.MethodPut:    true,
		http.MethodPatch:  true,
		http.MethodDelete: true,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Only log mutating requests or errors
			if !mutating[r.Method] && rec.status < 400 {
				return
			}

			ctx := r.Context()
			entry := HTTPLogEntry{
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: rec.status,
				DurationMS: time.Since(start).Milliseconds(),
				UserID:     GetUserID(ctx),
				IPAddress:  realIP(r),
				RequestID:  GetRequestID(ctx),
			}
			go logger.LogHTTP(ctx, entry)
		})
	}
}

// realIP extracts the best-guess real IP from a request.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[:i]
	}
	return addr
}

// ─── NATS implementation ──────────────────────────────────────────────────────

const topicSysHTTP = "sys.http_request"

// NATSAuditLogger publishes HTTP log entries to NATS JetStream.
type NATSAuditLogger struct {
	pub         *natspkg.Publisher
	serviceName string
}

// NewNATSAuditLogger creates a NATSAuditLogger for a named service.
func NewNATSAuditLogger(pub *natspkg.Publisher, serviceName string) *NATSAuditLogger {
	return &NATSAuditLogger{pub: pub, serviceName: serviceName}
}

// LogHTTP publishes an HTTP audit entry to NATS. Runs in a goroutine — never blocks.
func (l *NATSAuditLogger) LogHTTP(ctx context.Context, entry HTTPLogEntry) {
	entry.ServiceName = l.serviceName
	if err := l.pub.Publish(context.Background(), topicSysHTTP, entry); err != nil {
		// Fire-and-forget; failure is silently dropped to not impact traffic
		_ = fmt.Sprintf("audit publish failed: %v", err)
	}
}
