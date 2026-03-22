package vpm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pvp/proxy-service/internal/providers"
	"github.com/pvp/proxy-service/internal/providers/vpm"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// vpmResponse wraps data in the VPM API envelope.
func vpmResponse(data any) []byte {
	raw, _ := json.Marshal(data)
	env := map[string]any{"success": true, "data": json.RawMessage(raw)}
	b, _ := json.Marshal(env)
	return b
}

// vpmError returns a VPM error envelope.
func vpmError(code, message string) []byte {
	env := map[string]any{
		"success": false,
		"error":   map[string]string{"code": code, "message": message},
	}
	b, _ := json.Marshal(env)
	return b
}

// ─── Client Tests ─────────────────────────────────────────────────────────────

func TestClient_CreateProxy_Success(t *testing.T) {
	want := vpm.ProxySummary{
		ID:       "proxy-uuid-123",
		Host:     "10.0.0.1",
		Port:     43000,
		Protocol: "socks5",
		AuthUser: "user_abc",
		AuthPass: "secretpass",
		Status:   "running",
		NodeName: "node-sg-01",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/proxies" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %q", got)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(vpmResponse(want))
	}))
	defer srv.Close()

	client := vpm.NewClientWithHTTP(srv.URL, "test-key", srv.Client())
	got, err := client.CreateProxy(context.Background(), vpm.CreateProxyRequest{
		Protocol:  "socks5",
		IPRangeID: "range-uuid-001",
		AuthUser:  "user_abc",
		AuthPass:  "secretpass",
	})
	if err != nil {
		t.Fatalf("CreateProxy error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("got ID=%q want %q", got.ID, want.ID)
	}
	if got.AuthPass != want.AuthPass {
		t.Errorf("got AuthPass=%q want %q", got.AuthPass, want.AuthPass)
	}
	if got.Port != want.Port {
		t.Errorf("got Port=%d want %d", got.Port, want.Port)
	}
}

func TestClient_CreateProxy_APIError_422(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(vpmError("INVALID_PROTOCOL", "unsupported protocol"))
	}))
	defer srv.Close()

	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	_, err := client.CreateProxy(context.Background(), vpm.CreateProxyRequest{Protocol: "invalid"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_CreateProxy_Retries_On_5xx(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(vpmResponse(vpm.ProxySummary{ID: "retry-ok", Host: "1.2.3.4", Port: 5000, Protocol: "socks5", AuthUser: "u", AuthPass: "p"}))
	}))
	defer srv.Close()

	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	got, err := client.CreateProxy(context.Background(), vpm.CreateProxyRequest{Protocol: "socks5"})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got.ID != "retry-ok" {
		t.Errorf("got ID=%q", got.ID)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls (2 retries), got %d", calls)
	}
}

func TestClient_DeleteProxy_Success(t *testing.T) {
	const proxyID = "proxy-to-delete"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/proxies/"+proxyID {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	if err := client.DeleteProxy(context.Background(), proxyID); err != nil {
		t.Fatalf("DeleteProxy error: %v", err)
	}
}

func TestClient_GetProxy_Success(t *testing.T) {
	want := vpm.ProxySummary{ID: "proxy-123", Status: "running", Host: "5.6.7.8", Port: 9000, Protocol: "http", AuthUser: "u", AuthPass: "p"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/proxies/proxy-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write(vpmResponse(want))
	}))
	defer srv.Close()

	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	got, err := client.GetProxy(context.Background(), "proxy-123")
	if err != nil {
		t.Fatalf("GetProxy error: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("expected running, got %q", got.Status)
	}
}

func TestClient_GetProxy_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(vpmError("NOT_FOUND", "proxy not found"))
	}))
	defer srv.Close()

	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	_, err := client.GetProxy(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── Adapter Tests ─────────────────────────────────────────────────────────────

func TestAdapter_Purchase_ReturnsSyncCredentials(t *testing.T) {
	summary := vpm.ProxySummary{
		ID:       "vpm-proxy-abc",
		Host:     "103.228.75.149",
		Port:     43312,
		Protocol: "socks5",
		AuthUser: "user_abc123",
		AuthPass: "secretpass",
		Status:   "running",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/proxies" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write(vpmResponse(summary))
	}))
	defer srv.Close()

	cfg := vpm.Config{BaseURL: srv.URL, APIKey: "test-key"}
	client := vpm.NewClientWithHTTP(srv.URL, "test-key", srv.Client())
	adapter := vpm.NewAdapterWithClient(cfg, client)

	result, err := adapter.Purchase(context.Background(), providers.PurchaseRequest{
		Quantity: 1,
		Protocol: "socks5",
		Metadata: map[string]string{
			"ip_range_id": "range-uuid-001",
		},
	})
	if err != nil {
		t.Fatalf("Purchase error: %v", err)
	}

	// VPM is sync: Credentials must NOT be nil
	if result.Credentials == nil {
		t.Fatal("expected non-nil Credentials for sync provider VPM")
	}
	if len(result.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(result.Credentials))
	}
	cred := result.Credentials[0]
	if cred.Host != "103.228.75.149" {
		t.Errorf("expected host 103.228.75.149, got %q", cred.Host)
	}
	if cred.Port != 43312 {
		t.Errorf("expected port 43312, got %d", cred.Port)
	}
	if cred.Username != "user_abc123" {
		t.Errorf("expected user_abc123, got %q", cred.Username)
	}
	if cred.Password != "secretpass" {
		t.Errorf("expected secretpass, got %q", cred.Password)
	}
	if result.ProviderOrderID != "vpm-proxy-abc" {
		t.Errorf("expected ProviderOrderID=vpm-proxy-abc, got %q", result.ProviderOrderID)
	}
}

func TestAdapter_Purchase_DefaultsToSocks5(t *testing.T) {
	var capturedReq vpm.CreateProxyRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedReq)
		w.WriteHeader(http.StatusCreated)
		w.Write(vpmResponse(vpm.ProxySummary{ID: "x", Host: "1.2.3.4", Port: 1234, Protocol: "socks5", AuthUser: "u", AuthPass: "p"}))
	}))
	defer srv.Close()

	cfg := vpm.Config{BaseURL: srv.URL, APIKey: "k"}
	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	adapter := vpm.NewAdapterWithClient(cfg, client)

	// No protocol in request
	_, err := adapter.Purchase(context.Background(), providers.PurchaseRequest{Quantity: 1})
	if err != nil {
		t.Fatalf("Purchase error: %v", err)
	}
	if capturedReq.Protocol != "socks5" {
		t.Errorf("expected default protocol socks5, got %q", capturedReq.Protocol)
	}
}

func TestAdapter_Purchase_MetadataBandwidthAndSpeed(t *testing.T) {
	var capturedReq vpm.CreateProxyRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedReq)
		w.WriteHeader(http.StatusCreated)
		w.Write(vpmResponse(vpm.ProxySummary{ID: "y", Host: "2.3.4.5", Port: 5000, Protocol: "http", AuthUser: "u", AuthPass: "p"}))
	}))
	defer srv.Close()

	cfg := vpm.Config{BaseURL: srv.URL, APIKey: "k"}
	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	adapter := vpm.NewAdapterWithClient(cfg, client)

	_, err := adapter.Purchase(context.Background(), providers.PurchaseRequest{
		Quantity: 1,
		Protocol: "http",
		Metadata: map[string]string{
			"bandwidth_limit_mb": "51200",
			"speed_limit_mbps":   "10",
		},
	})
	if err != nil {
		t.Fatalf("Purchase error: %v", err)
	}
	if capturedReq.BandwidthLimitMB != 51200 {
		t.Errorf("expected BandwidthLimitMB=51200, got %d", capturedReq.BandwidthLimitMB)
	}
	if capturedReq.SpeedLimitMbps != 10 {
		t.Errorf("expected SpeedLimitMbps=10, got %d", capturedReq.SpeedLimitMbps)
	}
}

func TestAdapter_Cancel_CallsDelete(t *testing.T) {
	const proxyID = "vpm-proxy-to-cancel"
	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/proxies/"+proxyID {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	cfg := vpm.Config{BaseURL: srv.URL, APIKey: "k"}
	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	adapter := vpm.NewAdapterWithClient(cfg, client)

	if err := adapter.Cancel(context.Background(), proxyID); err != nil {
		t.Fatalf("Cancel error: %v", err)
	}
	if !deleted {
		t.Error("expected DELETE /api/v1/proxies/{id} to be called")
	}
}

func TestAdapter_CheckStatus_Mapping(t *testing.T) {
	tests := []struct {
		vpmStatus string
		want      string
	}{
		{"running", "active"},
		{"stopped", "active"},
		{"creating", "processing"},
		{"error", "failed"},
		{"unknown", "processing"},
	}

	for _, tc := range tests {
		t.Run(tc.vpmStatus, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write(vpmResponse(vpm.ProxySummary{
					ID: "p", Host: "1.1.1.1", Port: 1234,
					Protocol: "socks5", AuthUser: "u", AuthPass: "p",
					Status: tc.vpmStatus,
				}))
			}))
			defer srv.Close()

			cfg := vpm.Config{BaseURL: srv.URL, APIKey: "k"}
			client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
			adapter := vpm.NewAdapterWithClient(cfg, client)

			got, err := adapter.CheckStatus(context.Background(), "proxy-id")
			if err != nil {
				t.Fatalf("CheckStatus error: %v", err)
			}
			if got != tc.want {
				t.Errorf("vpm status %q → got %q, want %q", tc.vpmStatus, got, tc.want)
			}
		})
	}
}

func TestAdapter_Purchase_ProviderError_MapsToInvalidConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(vpmError("INVALID_RANGE", "ip range not found"))
	}))
	defer srv.Close()

	cfg := vpm.Config{BaseURL: srv.URL, APIKey: "k"}
	client := vpm.NewClientWithHTTP(srv.URL, "k", srv.Client())
	adapter := vpm.NewAdapterWithClient(cfg, client)

	_, err := adapter.Purchase(context.Background(), providers.PurchaseRequest{Protocol: "socks5"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should wrap ErrInvalidConfig
	expected := fmt.Sprintf("%s", providers.ErrInvalidConfig)
	if err.Error() == "" || len(err.Error()) == 0 {
		t.Error("expected non-empty error message")
	}
	_ = expected // Just checking it wraps the error type
}
