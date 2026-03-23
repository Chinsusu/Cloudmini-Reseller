package vpm

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
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
)

// Client handles HTTP communication with the VPM API.
// API v2: authentication uses ?access_code=<key> query parameter.
// API v1: authentication used Authorization: Bearer <key> header (legacy fallback).
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a VPM API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewClientWithHTTP creates a client with a custom http.Client (for testing).
func NewClientWithHTTP(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, httpClient: httpClient}
}

// do executes an authenticated HTTP request to VPM.
// v2 API: injects ?access_code= query parameter for auth.
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

	// Build URL with access_code query param (v2 auth)
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
	// Also send Bearer for v1 endpoints (backward compat)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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

// ─── API v2 Methods (current) ─────────────────────────────────────────────────

// CreateProxyV2 allocates proxies from a region pool.
// POST /api/v2/proxies?access_code=<key> — used when group_id/region is specified.
func (c *Client) CreateProxyV2(ctx context.Context, req CreateProxyV2Request) ([]ProxySummaryV2, error) {
	if req.Count == 0 {
		req.Count = 1
	}
	var out []ProxySummaryV2
	if err := c.do(ctx, http.MethodPost, "/api/v2/proxies", req, &out); err != nil {
		return nil, fmt.Errorf("vpm.CreateProxyV2: %w", err)
	}
	return out, nil
}

// CreateProxyByIPV4 allocates a proxy for a specific IP address (default endpoint).
// POST /api/v2/ipv4?access_code=<key>
// body: {"ipv4": "<ip>", "protocol": "default"|"http"|"socks5"}
func (c *Client) CreateProxyByIPV4(ctx context.Context, ipv4, protocol string) ([]ProxySummaryV2, error) {
	body := map[string]string{"ipv4": ipv4, "protocol": protocol}
	var out []ProxySummaryV2
	if err := c.do(ctx, http.MethodPost, "/api/v2/ipv4", body, &out); err != nil {
		return nil, fmt.Errorf("vpm.CreateProxyByIPV4: %w", err)
	}
	return out, nil
}

// GetProxyV2 returns details of a proxy by ID.
// GET /api/v2/ipv4/{id}?access_code=<key>
func (c *Client) GetProxyV2(ctx context.Context, proxyID string) (*ProxySummaryV2, error) {
	var out ProxySummaryV2
	if err := c.do(ctx, http.MethodGet, "/api/v2/ipv4/"+proxyID, nil, &out); err != nil {
		return nil, fmt.Errorf("vpm.GetProxyV2: %w", err)
	}
	return &out, nil
}

// DeleteProxyV2 permanently removes a proxy.
// DELETE /api/v2/ipv4/{id}?access_code=<key>  →  204 No Content
// For protocol="default", a SINGLE DELETE removes both HTTP and SOCKS5 inbounds.
// Only the first/any ID of the pair needs to be provided.
func (c *Client) DeleteProxyV2(ctx context.Context, proxyID string) error {
	if err := c.do(ctx, http.MethodDelete, "/api/v2/ipv4/"+proxyID, nil, nil); err != nil {
		return fmt.Errorf("vpm.DeleteProxyV2: %w", err)
	}
	return nil
}

// SuspendProxyV2 locks a proxy (equivalent to Stop/Suspend).
// PUT /api/v2/ipv4/{id}?access_code=<key>  body: {"action": "lock"}
func (c *Client) SuspendProxyV2(ctx context.Context, proxyID string) error {
	body := map[string]string{"action": "lock"}
	if err := c.do(ctx, http.MethodPut, "/api/v2/ipv4/"+proxyID, body, nil); err != nil {
		return fmt.Errorf("vpm.SuspendProxyV2: %w", err)
	}
	return nil
}

// ResumeProxyV2 unlocks a proxy (equivalent to Start/Resume).
// PUT /api/v2/ipv4/{id}?access_code=<key>  body: {"action": "unlock"}
func (c *Client) ResumeProxyV2(ctx context.Context, proxyID string) error {
	body := map[string]string{"action": "unlock"}
	if err := c.do(ctx, http.MethodPut, "/api/v2/ipv4/"+proxyID, body, nil); err != nil {
		return fmt.Errorf("vpm.ResumeProxyV2: %w", err)
	}
	return nil
}

// ListGroups returns all proxy regions/groups.
// GET /api/v1/groups?access_code=<key>
// Use the returned id as group_id when creating proxies.
func (c *Client) ListGroups(ctx context.Context) ([]ProxyGroup, error) {
	var out []ProxyGroup
	if err := c.do(ctx, http.MethodGet, "/api/v1/groups", nil, &out); err != nil {
		return nil, fmt.Errorf("vpm.ListGroups: %w", err)
	}
	return out, nil
}

// ─── API v1 Methods (legacy — kept for backward compat) ───────────────────────

// CreateProxy allocates a new proxy on VPM (v1, DEPRECATED).
func (c *Client) CreateProxy(ctx context.Context, req CreateProxyRequest) (*ProxySummary, error) {
	var out ProxySummary
	if err := c.do(ctx, http.MethodPost, "/api/v1/proxies", req, &out); err != nil {
		return nil, fmt.Errorf("vpm.CreateProxy: %w", err)
	}
	return &out, nil
}

// DeleteProxy permanently removes a proxy (v1, DEPRECATED).
func (c *Client) DeleteProxy(ctx context.Context, proxyID string) error {
	if err := c.do(ctx, http.MethodDelete, "/api/v1/proxies/"+proxyID, nil, nil); err != nil {
		return fmt.Errorf("vpm.DeleteProxy: %w", err)
	}
	return nil
}

// GetProxy returns details of a proxy (v1, DEPRECATED).
func (c *Client) GetProxy(ctx context.Context, proxyID string) (*ProxySummary, error) {
	var out ProxySummary
	if err := c.do(ctx, http.MethodGet, "/api/v1/proxies/"+proxyID, nil, &out); err != nil {
		return nil, fmt.Errorf("vpm.GetProxy: %w", err)
	}
	return &out, nil
}

// StopProxy suspends a proxy (v1, DEPRECATED — use SuspendProxyV2).
func (c *Client) StopProxy(ctx context.Context, proxyID string) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/proxies/"+proxyID+"/stop", nil, nil); err != nil {
		return fmt.Errorf("vpm.StopProxy: %w", err)
	}
	return nil
}

// StartProxy re-activates a stopped proxy (v1, DEPRECATED — use ResumeProxyV2).
func (c *Client) StartProxy(ctx context.Context, proxyID string) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/proxies/"+proxyID+"/start", nil, nil); err != nil {
		return fmt.Errorf("vpm.StartProxy: %w", err)
	}
	return nil
}
