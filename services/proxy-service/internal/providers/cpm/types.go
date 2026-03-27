// Package cpm provides a proxy provider adapter for the CPM ipv4-res API.
// Current integration: Billing API V1 — https://cz.resvn.net/billing-docs-res-v1
package cpm

import "encoding/json"

// ─── API v1 Types ─────────────────────────────────────────────────────────────

// CreateProxyRequest is the request body for POST /api/v1/proxies.
type CreateProxyRequest struct {
	Protocol         string `json:"protocol"`                      // "default"|"socks5"|"http"|"vmess"|"vless"|"shadowsocks"|"trojan"
	GroupID          string `json:"group_id,omitempty"`             // region/group UUID
	IPRangeID        string `json:"ip_range_id,omitempty"`          // specific IP range
	IPAddressID      string `json:"ip_address_id,omitempty"`        // specific IP address UUID
	ListenIP         string `json:"listen_ip,omitempty"`            // bind to specific IP (e.g. "103.151.53.41")
	AuthUser         string `json:"auth_user,omitempty"`            // custom username (auto-generated if empty)
	AuthPass         string `json:"auth_pass,omitempty"`            // custom password (auto-generated if empty)
	BandwidthLimitMB int    `json:"bandwidth_limit_mb,omitempty"`
	SpeedLimitMbps   int    `json:"speed_limit_mbps,omitempty"`
	SSMethod         string `json:"ss_method,omitempty"`            // Shadowsocks cipher (e.g. "chacha20-ietf-poly1305")
	RealityDest      string `json:"reality_dest,omitempty"`         // VLESS Reality destination (e.g. "google.com:443")
}

// ProxySummary is the response object returned by POST/GET /api/v1/proxies.
type ProxySummary struct {
	ID               string          `json:"id"`
	Host             string          `json:"host"`              // GET endpoint uses this
	Ipv4             string          `json:"ipv4"`              // POST endpoint uses this
	Port             int             `json:"port"`              // primary port
	PortHTTP         int             `json:"port_http"`         // >0 for http/default protocols
	PortSOCKS        int             `json:"port_socks"`        // >0 for socks5/default protocols
	Protocol         string          `json:"protocol"`          // "default"|"http"|"socks5"|"vmess"|"vless"|"shadowsocks"|"trojan"
	OutboundIP       string          `json:"outbound_ip"`       // exit IP
	AuthUser         string          `json:"auth_user"`         // proxy username
	AuthPass         string          `json:"auth_pass"`         // proxy password
	Status           string          `json:"status"`            // "creating"|"running"|"stopped"|"error"
	BandwidthUp      int64           `json:"bandwidth_up"`      // bytes uploaded
	BandwidthDown    int64           `json:"bandwidth_down"`    // bytes downloaded
	BandwidthLimitMB int             `json:"bandwidth_limit_mb"`
	SpeedLimitMbps   int             `json:"speed_limit_mbps"`
	CurrentSpeedBps  int64           `json:"current_speed_bps"`
	ConnectURL       string          `json:"connect_url"`       // ready-to-use connection string
	ConnectionString string          `json:"connection_string"` // alternative connection string (VPN protocols)
	ProtocolConfig   json.RawMessage `json:"protocol_config"`   // protocol-specific config
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
}

// effectivePort returns the port for credential building.
func (s *ProxySummary) effectivePort() int {
	if s.PortHTTP > 0 {
		return s.PortHTTP
	}
	if s.PortSOCKS > 0 {
		return s.PortSOCKS
	}
	return s.Port
}

// effectiveProtocol returns the protocol string for credentials.
func (s *ProxySummary) effectiveProtocol() string {
	switch s.Protocol {
	case "default", "http":
		return "http"
	case "socks5":
		return "socks5"
	default:
		return s.Protocol // vmess, vless, shadowsocks, trojan
	}
}

// effectiveHost returns the host for credential building.
func (s *ProxySummary) effectiveHost() string {
	if s.Host != "" {
		return s.Host
	}
	if s.Ipv4 != "" {
		return s.Ipv4
	}
	return s.OutboundIP
}

// ProxyGroup is one region/group returned by GET /api/v1/groups.
type ProxyGroup struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	TotalIPs     int     `json:"total_ips"`
	AvailableIPs int     `json:"available_ips"`
	NodeCount    int     `json:"node_count"`
	ProxyCount   int     `json:"proxy_count"`
	ParentID     *string `json:"parent_id"` // nil for top-level regions
}

// CheckResult is the response from GET /api/v1/proxies/:id/check.
type CheckResult struct {
	IP         string `json:"ip"`
	Country    string `json:"country"`
	City       string `json:"city"`
	Org        string `json:"org"`
	LatencyMs  int    `json:"latency_ms"`
	CheckedVia string `json:"checked_via"`
}

// ─── Shared API envelope ──────────────────────────────────────────────────────

// apiResponse wraps all CPM API responses.
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

// APIError is returned when CPM responds with a non-2xx status or success=false.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return "cpm: " + e.Code + ": " + e.Message
	}
	return "cpm: http " + string(rune('0'+e.StatusCode/100)) + "xx: " + e.Message
}
