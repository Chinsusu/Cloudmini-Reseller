package proxy_cheap_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pvp/proxy-service/internal/providers"
	proxycheap "github.com/pvp/proxy-service/internal/providers/proxy_cheap"
)

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func bytesReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// computeHMAC computes the Proxy-Cheap webhook HMAC signature.
func computeHMAC(eventName, eventID string, body []byte, secret string) string {
	input := "sha256" + eventName + eventID + string(body) + secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// testFulfiller is a mock implementation of proxycheap.WebhookFulfiller.
type testFulfiller struct {
	onFulfill func(providerOrderID string, proxy *proxycheap.Proxy) error
}

func (f *testFulfiller) FulfillFromProxyCheap(ctx context.Context, providerOrderID string, proxy *proxycheap.Proxy) error {
	if f.onFulfill != nil {
		return f.onFulfill(providerOrderID, proxy)
	}
	return nil
}

// ─── Client Tests ─────────────────────────────────────────────────────────────

func TestClient_Execute_Success(t *testing.T) {
	want := proxycheap.ExecuteResponse{
		ID:             "order-abc-123",
		PeriodInMonths: "1",
		Bandwidth:      "0",
		TotalPrice:     "3.39",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v2/order/static-residential-ipv4/execute" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") == "" || r.Header.Get("X-Api-Secret") == "" {
			t.Error("missing auth headers")
		}
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := proxycheap.NewClientWithBase("test-key", "test-secret", srv.URL)
	got, err := client.Execute(context.Background(), "static-residential-ipv4", proxycheap.ExecuteRequest{
		PlanID:   "basic",
		Quantity: 1,
		Country:  "US",
		Period:   &proxycheap.PeriodSpec{Unit: "months", Value: 1},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("got ID=%q want %q", got.ID, want.ID)
	}
}

func TestClient_Execute_APIError_422(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message":"invalid country"}`))
	}))
	defer srv.Close()

	client := proxycheap.NewClientWithBase("k", "s", srv.URL)
	_, err := client.Execute(context.Background(), "static-residential-ipv4", proxycheap.ExecuteRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// client.do wraps APIError via fmt.Errorf, use errors.As to unwrap
	var apiErr *proxycheap.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError in chain, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("got status %d want 422", apiErr.StatusCode)
	}
}

func TestClient_Execute_Retries_On_5xx(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(proxycheap.ExecuteResponse{ID: "ok-after-retry"})
	}))
	defer srv.Close()

	client := proxycheap.NewClientWithBase("k", "s", srv.URL)
	got, err := client.Execute(context.Background(), "static-residential-ipv4", proxycheap.ExecuteRequest{})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got.ID != "ok-after-retry" {
		t.Errorf("got ID=%q", got.ID)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls (2 retries), got %d", calls)
	}
}

func TestClient_GetOrderProxies(t *testing.T) {
	wantProxies := []proxycheap.Proxy{
		{
			ID:     12345,
			Status: "ACTIVE",
			Connection: proxycheap.ProxyConnection{
				ConnectIP: "1.2.3.4",
				HTTPPort:  10000,
			},
			Authentication: proxycheap.ProxyAuthentication{
				Username: "user123",
				Password: "pass456",
			},
			ProxyType:   "HTTP",
			CountryCode: "US",
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orders/order-xyz/proxies" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(wantProxies)
	}))
	defer srv.Close()

	client := proxycheap.NewClientWithBase("k", "s", srv.URL)
	got, err := client.GetOrderProxies(context.Background(), "order-xyz")
	if err != nil {
		t.Fatalf("GetOrderProxies error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(got))
	}
	if got[0].Status != "ACTIVE" {
		t.Errorf("expected ACTIVE, got %s", got[0].Status)
	}
}

func TestClient_GetBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/balance" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(proxycheap.AccountBalance{Balance: 42.5})
	}))
	defer srv.Close()

	client := proxycheap.NewClientWithBase("k", "s", srv.URL)
	bal, err := client.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance error: %v", err)
	}
	if bal.Balance != 42.5 {
		t.Errorf("expected 42.5, got %v", bal.Balance)
	}
}

// ─── Adapter Tests ─────────────────────────────────────────────────────────────

func TestAdapter_Purchase_ReturnsAsyncResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(proxycheap.ExecuteResponse{
			ID:         "pc-order-999",
			TotalPrice: "3.39",
		})
	}))
	defer srv.Close()

	cfg := proxycheap.Config{APIKey: "k", APISecret: "s"}
	client := proxycheap.NewClientWithBase("k", "s", srv.URL)
	adapter := proxycheap.NewAdapterWithClient(cfg, client)

	result, err := adapter.Purchase(context.Background(), providers.PurchaseRequest{
		Quantity: 1,
		Country:  "US",
		Metadata: map[string]string{
			"service_id":    "static-residential-ipv4",
			"plan_id":       "basic",
			"period_months": "1",
		},
	})
	if err != nil {
		t.Fatalf("Purchase error: %v", err)
	}
	if result.ProviderOrderID != "pc-order-999" {
		t.Errorf("expected pc-order-999, got %q", result.ProviderOrderID)
	}
	// Async provider must return nil Credentials
	if result.Credentials != nil {
		t.Error("expected nil Credentials for async provider")
	}
}

func TestAdapter_Purchase_MissingServiceID_ReturnsError(t *testing.T) {
	cfg := proxycheap.Config{APIKey: "k", APISecret: "s"}
	client := proxycheap.NewClientWithBase("k", "s", "http://localhost:1")
	adapter := proxycheap.NewAdapterWithClient(cfg, client)

	_, err := adapter.Purchase(context.Background(), providers.PurchaseRequest{
		Metadata: map[string]string{}, // missing service_id
	})
	if err == nil {
		t.Error("expected error for missing service_id")
	}
}

func TestAdapter_CheckStatus_Active(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]proxycheap.Proxy{{ID: 1, Status: "ACTIVE"}})
	}))
	defer srv.Close()

	cfg := proxycheap.Config{APIKey: "k", APISecret: "s"}
	adapter := proxycheap.NewAdapterWithClient(cfg, proxycheap.NewClientWithBase("k", "s", srv.URL))

	status, err := adapter.CheckStatus(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("CheckStatus error: %v", err)
	}
	if status != "active" {
		t.Errorf("expected active, got %q", status)
	}
}

func TestAdapter_CheckStatus_Pending(t *testing.T) {
	for _, pcStatus := range []string{"PENDING", "INITIATING"} {
		t.Run(pcStatus, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode([]proxycheap.Proxy{{ID: 1, Status: pcStatus}})
			}))
			defer srv.Close()

			adapter := proxycheap.NewAdapterWithClient(
				proxycheap.Config{APIKey: "k", APISecret: "s"},
				proxycheap.NewClientWithBase("k", "s", srv.URL),
			)
			status, _ := adapter.CheckStatus(context.Background(), "order-1")
			if status != "processing" {
				t.Errorf("%s → expected processing, got %q", pcStatus, status)
			}
		})
	}
}

func TestAdapter_CheckStatus_EmptyProxies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]proxycheap.Proxy{})
	}))
	defer srv.Close()

	adapter := proxycheap.NewAdapterWithClient(
		proxycheap.Config{APIKey: "k", APISecret: "s"},
		proxycheap.NewClientWithBase("k", "s", srv.URL),
	)
	status, _ := adapter.CheckStatus(context.Background(), "order-1")
	if status != "processing" {
		t.Errorf("expected processing, got %q", status)
	}
}

// ─── Webhook Tests ────────────────────────────────────────────────────────────

func TestWebhook_ValidSignature_StatusChanged_Active(t *testing.T) {
	const secret = "0qFbBP1zE5MuSMy"
	eventName := proxycheap.WebhookEventStatusChanged
	eventID := "test-event-id-001"

	payload := proxycheap.WebhookStatusChanged{ProxyID: 99999, OldStatus: "PENDING", Status: "ACTIVE"}
	body := mustMarshal(payload)
	sig := computeHMAC(eventName, eventID, body, secret)

	// Mock client: GET /proxies/99999 returns a proxy
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(proxycheap.Proxy{
			ID:             99999,
			Status:         "ACTIVE",
			Connection:     proxycheap.ProxyConnection{ConnectIP: "5.6.7.8", HTTPPort: 20000},
			Authentication: proxycheap.ProxyAuthentication{Username: "u", Password: "p"},
			ProxyType:      "HTTP",
			CountryCode:    "US",
		})
	}))
	defer mockSrv.Close()

	fulfilled := false
	fulfiller := &testFulfiller{
		onFulfill: func(providerOrderID string, _ *proxycheap.Proxy) error {
			fulfilled = true
			if providerOrderID != "99999" {
				t.Errorf("expected proxyId 99999, got %s", providerOrderID)
			}
			return nil
		},
	}

	client := proxycheap.NewClientWithBase("k", "s", mockSrv.URL)
	wh := proxycheap.NewWebhookHandler(client, secret, fulfiller, newTestLogger())

	req := httptest.NewRequest(http.MethodPost, "/webhooks/proxy-cheap", bytesReader(body))
	req.Header.Set("Webhook-Event", eventName)
	req.Header.Set("Webhook-Id", eventID)
	req.Header.Set("Webhook-Signature", sig)

	rr := httptest.NewRecorder()
	wh.HandleEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !fulfilled {
		t.Error("expected FulfillFromProxyCheap to be called")
	}
}

func TestWebhook_InvalidSignature_Returns401(t *testing.T) {
	body := mustMarshal(proxycheap.WebhookStatusChanged{ProxyID: 1, Status: "ACTIVE"})
	client := proxycheap.NewClientWithBase("k", "s", "http://localhost:1")
	wh := proxycheap.NewWebhookHandler(client, "real-secret", &testFulfiller{}, newTestLogger())

	req := httptest.NewRequest(http.MethodPost, "/webhooks/proxy-cheap", bytesReader(body))
	req.Header.Set("Webhook-Event", proxycheap.WebhookEventStatusChanged)
	req.Header.Set("Webhook-Id", "event-id")
	req.Header.Set("Webhook-Signature", "sha256=wrong_signature_here")

	rr := httptest.NewRecorder()
	wh.HandleEvent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestWebhook_NonActiveStatus_DoesNotFulfill(t *testing.T) {
	const secret = "test-secret"
	eventName := proxycheap.WebhookEventStatusChanged
	eventID := "ev-002"

	payload := proxycheap.WebhookStatusChanged{ProxyID: 111, OldStatus: "PENDING", Status: "INITIATING"}
	body := mustMarshal(payload)
	sig := computeHMAC(eventName, eventID, body, secret)

	called := false
	fulfiller := &testFulfiller{
		onFulfill: func(_ string, _ *proxycheap.Proxy) error {
			called = true
			return nil
		},
	}

	client := proxycheap.NewClientWithBase("k", "s", "http://localhost:1")
	wh := proxycheap.NewWebhookHandler(client, secret, fulfiller, newTestLogger())

	req := httptest.NewRequest(http.MethodPost, "/webhooks/proxy-cheap", bytesReader(body))
	req.Header.Set("Webhook-Event", eventName)
	req.Header.Set("Webhook-Id", eventID)
	req.Header.Set("Webhook-Signature", sig)

	rr := httptest.NewRecorder()
	wh.HandleEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if called {
		t.Error("FulfillFromProxyCheap should NOT be called for non-ACTIVE status")
	}
}

func TestWebhook_BandwidthAdded_ReturnsOK(t *testing.T) {
	const secret = "test-secret"
	eventName := proxycheap.WebhookEventBandwidthAdded
	eventID := "bw-001"

	payload := proxycheap.WebhookBandwidthAdded{ProxyID: 222, TrafficInGB: 5}
	body := mustMarshal(payload)
	sig := computeHMAC(eventName, eventID, body, secret)

	client := proxycheap.NewClientWithBase("k", "s", "http://localhost:1")
	wh := proxycheap.NewWebhookHandler(client, secret, &testFulfiller{}, newTestLogger())

	req := httptest.NewRequest(http.MethodPost, "/webhooks/proxy-cheap", bytesReader(body))
	req.Header.Set("Webhook-Event", eventName)
	req.Header.Set("Webhook-Id", eventID)
	req.Header.Set("Webhook-Signature", sig)

	rr := httptest.NewRecorder()
	wh.HandleEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}
