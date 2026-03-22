// Package vpm provides a proxy provider adapter for the VPS Proxy Manager (VPM) API.
// VPM is an internal service that manages proxy allocation on physical nodes.
package vpm

import "encoding/json"

// CreateProxyRequest is the request body for POST /api/v1/proxies.
type CreateProxyRequest struct {
	Protocol         string         `json:"protocol"`                    // required: "socks5" | "http"
	IPRangeID        string         `json:"ip_range_id,omitempty"`       // VPM IP range UUID
	NodeID           string         `json:"node_id,omitempty"`           // specific node; VPM auto-selects if empty
	AuthUser         string         `json:"auth_user,omitempty"`         // 3-50 chars
	AuthPass         string         `json:"auth_pass,omitempty"`         // 8-100 chars
	BandwidthLimitMB int            `json:"bandwidth_limit_mb,omitempty"` // 0 = unlimited
	SpeedLimitMbps   int            `json:"speed_limit_mbps,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// ProxySummary is the response object returned by VPM for proxy operations.
type ProxySummary struct {
	ID               string `json:"id"`
	Host             string `json:"host"`         // listen IP (per API docs)
	Port             int    `json:"port"`
	Protocol         string `json:"protocol"`
	Status           string `json:"status"`       // running|stopped|error|creating
	AuthUser         string `json:"auth_user"`
	AuthPass         string `json:"auth_pass"`
	OutboundIP       string `json:"outbound_ip"`
	NodeID           string `json:"node_id"`
	NodeName         string `json:"node_name"`
	BandwidthUp      int    `json:"bandwidth_up"`
	BandwidthDown    int    `json:"bandwidth_down"`
	BandwidthLimitMB int    `json:"bandwidth_limit_mb"`
	SpeedLimitMbps   int    `json:"speed_limit_mbps"`
	CurrentSpeedBps  int    `json:"current_speed_bps"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// apiResponse wraps all VPM API responses.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
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
