package proxy_cheap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL        = "https://api.proxy-cheap.com"
	defaultTimeoutSeconds = 30
)

// Client handles HTTP communication with the Proxy-Cheap API.
// Authentication is done via X-Api-Key and X-Api-Secret headers on every request.
type Client struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewClient creates a Proxy-Cheap API client.
func NewClient(apiKey, apiSecret string) *Client {
	return &Client{
		baseURL:   defaultBaseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: defaultTimeoutSeconds * time.Second,
		},
	}
}

// NewClientWithBase creates a client with a custom base URL (useful for tests).
func NewClientWithBase(apiKey, apiSecret, baseURL string) *Client {
	c := NewClient(apiKey, apiSecret)
	c.baseURL = baseURL
	return c
}

// do executes an authenticated HTTP request to the Proxy-Cheap API.
// body may be nil for GET requests. out may be nil if no response body is expected.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("proxy_cheap: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("proxy_cheap: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("X-Api-Secret", c.apiSecret)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<uint(attempt-1)) * time.Second):
			}
			// Re-clone body for retry (already read on first attempt)
			if body != nil {
				b, _ := json.Marshal(body)
				req.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("proxy_cheap: http do: %w", err)
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
			continue // retry on 5xx
		}

		if resp.StatusCode >= 400 {
			return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
		}

		if out != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("proxy_cheap: unmarshal response: %w", err)
			}
		}
		return nil
	}
	return lastErr
}

// ─── Service / Catalog ────────────────────────────────────────────────────────

// GetServices fetches available services and plans from GET /v2/order.
func (c *Client) GetServices(ctx context.Context) (*ServicesResponse, error) {
	var out ServicesResponse
	if err := c.do(ctx, http.MethodGet, "/v2/order", nil, &out); err != nil {
		return nil, fmt.Errorf("GetServices: %w", err)
	}
	return &out, nil
}

// GetServiceOptions fetches config options for a specific service.
// POST /v2/order/:serviceId
func (c *Client) GetServiceOptions(ctx context.Context, serviceID, planID string) (*ServiceOptionsResponse, error) {
	req := ServiceOptionsRequest{PlanID: planID}
	var out ServiceOptionsResponse
	if err := c.do(ctx, http.MethodPost, "/v2/order/"+serviceID, req, &out); err != nil {
		return nil, fmt.Errorf("GetServiceOptions: %w", err)
	}
	return &out, nil
}

// GetPrice calculates the price for a given configuration.
// POST /v2/order/:serviceId/price
func (c *Client) GetPrice(ctx context.Context, serviceID string, req PriceRequest) (*PriceResponse, error) {
	var out PriceResponse
	if err := c.do(ctx, http.MethodPost, "/v2/order/"+serviceID+"/price", req, &out); err != nil {
		return nil, fmt.Errorf("GetPrice: %w", err)
	}
	return &out, nil
}

// Execute places an order for a proxy service.
// POST /v2/order/:serviceId/execute
func (c *Client) Execute(ctx context.Context, serviceID string, req ExecuteRequest) (*ExecuteResponse, error) {
	var out ExecuteResponse
	if err := c.do(ctx, http.MethodPost, "/v2/order/"+serviceID+"/execute", req, &out); err != nil {
		return nil, fmt.Errorf("Execute: %w", err)
	}
	return &out, nil
}

// ─── Orders ───────────────────────────────────────────────────────────────────

// GetOrderDetails fetches details for a specific order.
// GET /orders/:id
func (c *Client) GetOrderDetails(ctx context.Context, orderID string) (*OrderDetailsResponse, error) {
	var out OrderDetailsResponse
	if err := c.do(ctx, http.MethodGet, "/orders/"+orderID, nil, &out); err != nil {
		return nil, fmt.Errorf("GetOrderDetails: %w", err)
	}
	return &out, nil
}

// GetOrderProxies returns the list of proxies for a given order.
// GET /orders/:id/proxies
func (c *Client) GetOrderProxies(ctx context.Context, orderID string) ([]Proxy, error) {
	var out []Proxy
	if err := c.do(ctx, http.MethodGet, "/orders/"+orderID+"/proxies", nil, &out); err != nil {
		return nil, fmt.Errorf("GetOrderProxies: %w", err)
	}
	return out, nil
}

// ─── Proxy Management ─────────────────────────────────────────────────────────

// GetProxy returns details of a single proxy by its numeric ID.
// GET /proxies/:id
func (c *Client) GetProxy(ctx context.Context, proxyID int64) (*Proxy, error) {
	var out Proxy
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/proxies/%d", proxyID), nil, &out); err != nil {
		return nil, fmt.Errorf("GetProxy: %w", err)
	}
	return &out, nil
}

// ─── Account ─────────────────────────────────────────────────────────────────

// GetBalance fetches the Proxy-Cheap account balance.
// GET /account/balance
func (c *Client) GetBalance(ctx context.Context) (*AccountBalance, error) {
	var out AccountBalance
	if err := c.do(ctx, http.MethodGet, "/account/balance", nil, &out); err != nil {
		return nil, fmt.Errorf("GetBalance: %w", err)
	}
	return &out, nil
}
