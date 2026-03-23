// Package vpm provides a proxy provider adapter for the VPS Proxy Manager (VPM) API.
// VPM is an internal service that manages proxy allocation on physical nodes.
package vpm

import "encoding/json"

// ─── API v2 Types (current) ───────────────────────────────────────────────────

// CreateProxyV2Request is the request body for POST /api/v2/proxies.
// auth is via ?access_code= query parameter.
type CreateProxyV2Request struct {
	Protocol         string `json:"protocol"`                    // "default"|"socks5"|"http"
	GroupID          string `json:"group_id,omitempty"`          // region/group UUID (replaces ip_range_id)
	IPv4             string `json:"ipv4,omitempty"`              // specific IP (use /api/v2/ipv4 instead)
	Count            int    `json:"count,omitempty"`             // batch count (default 1)
	BandwidthLimitMB int    `json:"bandwidth_limit_mb,omitempty"`
	SpeedLimitMbps   int    `json:"speed_limit_mbps,omitempty"`
}

// ProxySummaryV2 is one element in the array returned by POST/GET /api/v2/proxies.
// When protocol="default", two entries are created sharing the same IP but different ports.
// The second entry's id is exposed as pair_id on the first, and vice versa.
type ProxySummaryV2 struct {
	ID               string `json:"id"`
	IPv4             string `json:"ipv4"`
	IPv6             string `json:"ipv6"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	PortHTTP         int    `json:"port_http"`  // non-null when protocol=http or default
	PortSOCKS        int    `json:"port_socks"` // non-null when protocol=socks5 or default
	Protocol         string `json:"protocol"`   // "http"|"socks5"|"default"
	ConnectionString string `json:"connection_string"`
	Status           string `json:"status"` // "completed"|"suspended"|"error"
	PairID           string `json:"pair_id,omitempty"` // ID of the paired proxy (protocol=default only)
}

// effectivePort returns the correct port for the given protocol.
// For protocol="default" we use PortHTTP; callers can override.
func (s *ProxySummaryV2) effectivePort() int {
	if s.PortHTTP > 0 {
		return s.PortHTTP
	}
	return s.PortSOCKS
}

// effectiveProtocol returns the human-readable protocol string for credentials.
func (s *ProxySummaryV2) effectiveProtocol() string {
	if s.PortHTTP > 0 {
		return "http"
	}
	return "socks5"
}

// ProxyGroup is one region/group returned by GET /api/v1/groups.
type ProxyGroup struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"` // nil for top-level regions
}

// ─── API v1 Types (kept for backward compat with existing provider_order_ids) ──

// CreateProxyRequest is the v1 request body for POST /api/v1/proxies (DEPRECATED).
type CreateProxyRequest struct {
	Protocol         string         `json:"protocol"`
	IPRangeID        string         `json:"ip_range_id,omitempty"`
	NodeID           string         `json:"node_id,omitempty"`
	AuthUser         string         `json:"auth_user,omitempty"`
	AuthPass         string         `json:"auth_pass,omitempty"`
	BandwidthLimitMB int            `json:"bandwidth_limit_mb,omitempty"`
	SpeedLimitMbps   int            `json:"speed_limit_mbps,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// ProxySummary is the v1 response object (DEPRECATED — kept for GetProxy fallback).
type ProxySummary struct {
	ID               string `json:"id"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Protocol         string `json:"protocol"`
	Status           string `json:"status"`
	AuthUser         string `json:"auth_user"`
	AuthPass         string `json:"auth_pass"`
	OutboundIP       string `json:"outbound_ip"`
	NodeID           string `json:"node_id"`
	NodeName         string `json:"node_name"`
	BandwidthLimitMB int    `json:"bandwidth_limit_mb"`
	SpeedLimitMbps   int    `json:"speed_limit_mbps"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// ─── Shared API envelope ──────────────────────────────────────────────────────

// apiResponse wraps all VPM API responses.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message,omitempty"`
	Error   *apiErrorDetail `json:"error"`
}

// apiErrorDetail is the error payload inside apiResponse.
type apiErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIError is returned when VPM responds with a non-2xx status or success=false.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return "vpm: " + e.Code + ": " + e.Message
	}
	return "vpm: http " + string(rune('0'+e.StatusCode/100)) + "xx: " + e.Message
}
