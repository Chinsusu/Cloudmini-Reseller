package vpm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	// Polling for async proxy creation
	pollInterval = 2 * time.Second
	pollTimeout  = 60 * time.Second
)

// Client handles HTTP communication with the VPM Billing API V1.
// Docs: https://cz.resvn.net/billing-docs-v1
// Auth: X-API-Key header (preferred) + ?access_code= query param (fallback).
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a VPM API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewClientWithHTTP creates a client with a custom http.Client (for testing).
func NewClientWithHTTP(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, httpClient: httpClient}
}

// do executes an authenticated HTTP request to VPM.
// Auth: X-API-Key header + ?access_code= query parameter (dual for compat).
// body may be nil for requests without payload. out may be nil to discard response.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("vpm: marshal request: %w", err)
		}
		bodyBytes = b
		reqBody = bytes.NewReader(b)
	}

	// Build URL with access_code query param
	url := c.baseURL + path
	if c.apiKey != "" {
		sep := "?"
		for _, ch := range path {
			if ch == '?' {
				sep = "&"
				break
			}
		}
		url += sep + "access_code=" + c.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("vpm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Primary auth: X-API-Key header
	req.Header.Set("X-API-Key", c.apiKey)

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<uint(attempt-1)) * time.Second):
			}
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("vpm: http do: %w", err)
			continue
		}

		respBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Retry on 5xx
		if resp.StatusCode >= 500 {
			lastErr = &APIError{StatusCode: resp.StatusCode, Message: string(respBytes)}
			continue
		}

		// 204 No Content — success with no body (e.g. DELETE)
		if resp.StatusCode == 204 {
			return nil
		}

		// 4xx — parse and return error immediately (no retry)
		if resp.StatusCode >= 400 {
			var apiResp apiResponse
			if jsonErr := json.Unmarshal(respBytes, &apiResp); jsonErr == nil && apiResp.Error != nil {
				return &APIError{
					StatusCode: resp.StatusCode,
					Code:       apiResp.Error.Code,
					Message:    apiResp.Error.Message,
				}
			}
			return &APIError{StatusCode: resp.StatusCode, Message: string(respBytes)}
		}

		// 2xx — parse response body
		if out != nil && len(respBytes) > 0 {
			// VPM wraps responses in { success, data, error }
			var apiResp apiResponse
			if err := json.Unmarshal(respBytes, &apiResp); err != nil {
				return fmt.Errorf("vpm: unmarshal envelope: %w", err)
			}
			if !apiResp.Success && apiResp.Error != nil {
				return &APIError{
					StatusCode: resp.StatusCode,
					Code:       apiResp.Error.Code,
					Message:    apiResp.Error.Message,
				}
			}
			if len(apiResp.Data) > 0 {
				if err := json.Unmarshal(apiResp.Data, out); err != nil {
					return fmt.Errorf("vpm: unmarshal data: %w", err)
				}
			}
		}
		return nil
	}
	return lastErr
}

// ─── Proxy CRUD ───────────────────────────────────────────────────────────────

// CreateProxy allocates a new proxy.
// POST /api/v1/proxies
// Returns immediately with status="creating". Use WaitForRunning to poll.
func (c *Client) CreateProxy(ctx context.Context, req CreateProxyRequest) (*ProxySummary, error) {
	var out ProxySummary
	if err := c.do(ctx, http.MethodPost, "/api/v1/proxies", req, &out); err != nil {
		return nil, fmt.Errorf("vpm.CreateProxy: %w", err)
	}
	return &out, nil
}

// CreateProxyAndWait creates a proxy and polls until status != "creating".
// Returns the proxy once it reaches "running" (or other terminal status).
func (c *Client) CreateProxyAndWait(ctx context.Context, req CreateProxyRequest) (*ProxySummary, error) {
	proxy, err := c.CreateProxy(ctx, req)
	if err != nil {
		return nil, err
	}

	// If already running (unlikely but possible), return immediately
	if proxy.Status == "running" {
		return proxy, nil
	}

	// Poll until status changes from "creating"
	return c.WaitForRunning(ctx, proxy.ID)
}

// WaitForRunning polls GET /api/v1/proxies/:id until status != "creating".
func (c *Client) WaitForRunning(ctx context.Context, proxyID string) (*ProxySummary, error) {
	deadline := time.Now().Add(pollTimeout)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("vpm.WaitForRunning: timeout after %v waiting for proxy %s", pollTimeout, proxyID)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		proxy, err := c.GetProxy(ctx, proxyID)
		if err != nil {
			return nil, fmt.Errorf("vpm.WaitForRunning: %w", err)
		}

		switch proxy.Status {
		case "creating":
			continue // keep polling
		case "running":
			return proxy, nil
		default:
			// "error", "stopped", or unknown — return as-is
			return proxy, nil
		}
	}
}

// GetProxy returns details of a proxy by ID.
// GET /api/v1/proxies/:id
func (c *Client) GetProxy(ctx context.Context, proxyID string) (*ProxySummary, error) {
	var out ProxySummary
	if err := c.do(ctx, http.MethodGet, "/api/v1/proxies/"+proxyID, nil, &out); err != nil {
		return nil, fmt.Errorf("vpm.GetProxy: %w", err)
	}
	return &out, nil
}

// ListProxies returns all proxies.
// GET /api/v1/proxies
func (c *Client) ListProxies(ctx context.Context) ([]ProxySummary, error) {
	var out []ProxySummary
	if err := c.do(ctx, http.MethodGet, "/api/v1/proxies", nil, &out); err != nil {
		return nil, fmt.Errorf("vpm.ListProxies: %w", err)
	}
	return out, nil
}

// DeleteProxy permanently removes a proxy.
// DELETE /api/v1/proxies/:id → 204 No Content
func (c *Client) DeleteProxy(ctx context.Context, proxyID string) error {
	if err := c.do(ctx, http.MethodDelete, "/api/v1/proxies/"+proxyID, nil, nil); err != nil {
		return fmt.Errorf("vpm.DeleteProxy: %w", err)
	}
	return nil
}

// StopProxy temporarily stops a running proxy (traffic disabled).
// POST /api/v1/proxies/:id/stop
func (c *Client) StopProxy(ctx context.Context, proxyID string) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/proxies/"+proxyID+"/stop", nil, nil); err != nil {
		return fmt.Errorf("vpm.StopProxy: %w", err)
	}
	return nil
}

// StartProxy re-activates a stopped proxy.
// POST /api/v1/proxies/:id/start
func (c *Client) StartProxy(ctx context.Context, proxyID string) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/proxies/"+proxyID+"/start", nil, nil); err != nil {
		return fmt.Errorf("vpm.StartProxy: %w", err)
	}
	return nil
}

// CheckProxy verifies the proxy exit IP and returns geo info.
// GET /api/v1/proxies/:id/check
func (c *Client) CheckProxy(ctx context.Context, proxyID string) (*CheckResult, error) {
	var out CheckResult
	if err := c.do(ctx, http.MethodGet, "/api/v1/proxies/"+proxyID+"/check", nil, &out); err != nil {
		return nil, fmt.Errorf("vpm.CheckProxy: %w", err)
	}
	return &out, nil
}

// ListGroups returns all proxy regions/groups.
// GET /api/v1/groups
func (c *Client) ListGroups(ctx context.Context) ([]ProxyGroup, error) {
	var out []ProxyGroup
	if err := c.do(ctx, http.MethodGet, "/api/v1/groups", nil, &out); err != nil {
		return nil, fmt.Errorf("vpm.ListGroups: %w", err)
	}
	return out, nil
}
