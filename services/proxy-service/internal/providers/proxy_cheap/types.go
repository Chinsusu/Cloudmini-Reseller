// Package proxy_cheap provides a Proxy-Cheap provider adapter for proxy-service.
// API docs: https://docs.proxy-cheap.com/
// Base URL: https://api.proxy-cheap.com
package proxy_cheap

import (
	"fmt"
	"time"
)

// ─── Service / Catalog ────────────────────────────────────────────────────────

// ServicePlan represents a plan within a service (e.g. basic, standard, premium).
type ServicePlan struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Service represents a proxy service returned by GET /v2/order.
type Service struct {
	ID    string        `json:"id"`
	Label string        `json:"label"`
	Plans []ServicePlan `json:"plans,omitempty"`
}

// ServicesResponse is the response from GET /v2/order.
type ServicesResponse struct {
	Services []Service `json:"services"`
}

// ISP is an Internet Service Provider available in a country.
type ISP struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Package represents a fixed proxy package (e.g. 50/150/500 proxies for IPv6).
type Package struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ServiceOptionsRequest is the request body for POST /v2/order/:serviceId.
type ServiceOptionsRequest struct {
	PlanID string `json:"planId,omitempty"`
}

// ServiceOptionsResponse is the response from POST /v2/order/:serviceId.
type ServiceOptionsResponse struct {
	ServiceID string              `json:"serviceId"`
	Countries []string            `json:"countries,omitempty"`
	ISPs      map[string][]ISP    `json:"isps,omitempty"`
	Packages  []Package           `json:"packages,omitempty"`
	Periods   *PeriodOptions      `json:"periods,omitempty"`
}

// PeriodOptions holds available period values.
type PeriodOptions struct {
	Months []int `json:"months,omitempty"`
	Days   []int `json:"days,omitempty"`
}

// ─── Pricing ─────────────────────────────────────────────────────────────────

// PeriodSpec defines a time period for ordering.
type PeriodSpec struct {
	Unit  string `json:"unit"`  // "months" or "days"
	Value int    `json:"value"`
}

// AutoExtendSpec defines auto-extend settings.
type AutoExtendSpec struct {
	IsEnabled bool `json:"isEnabled"`
}

// PriceRequest is the body for POST /v2/order/:serviceId/price.
type PriceRequest struct {
	PlanID     string         `json:"planId,omitempty"`
	PackageID  string         `json:"packageId,omitempty"`
	Quantity   int            `json:"quantity,omitempty"`
	Country    string         `json:"country,omitempty"`
	ISPID      string         `json:"ispId,omitempty"`
	Period     *PeriodSpec    `json:"period,omitempty"`
	Traffic    int            `json:"traffic,omitempty"` // GB for rotating proxies
	CouponCode string         `json:"couponCode,omitempty"`
}

// PriceResponse is the response from POST /v2/order/:serviceId/price.
type PriceResponse struct {
	FinalPrice             float64 `json:"finalPrice"`
	PriceNoDiscounts       float64 `json:"priceNoDiscounts"`
	Discount               float64 `json:"discount"`
	UnitPrice              float64 `json:"unitPrice"`
	UnitPriceAfterDiscount float64 `json:"unitPriceAfterDiscount"`
	PaymentFee             float64 `json:"paymentFee"`
	Subtotal               float64 `json:"subtotal"`
	DiscountAmount         float64 `json:"discountAmount"`
	FinalPriceInCurrency   float64 `json:"finalPriceInCurrency"`
	Currency               string  `json:"currency"`
}

// ─── Order ────────────────────────────────────────────────────────────────────

// ExecuteRequest is the body for POST /v2/order/:serviceId/execute.
type ExecuteRequest struct {
	PlanID     string         `json:"planId,omitempty"`
	PackageID  string         `json:"packageId,omitempty"`
	Quantity   int            `json:"quantity,omitempty"`
	Country    string         `json:"country,omitempty"`
	ISPID      string         `json:"ispId,omitempty"`
	Period     *PeriodSpec    `json:"period,omitempty"`
	AutoExtend *AutoExtendSpec `json:"autoExtend,omitempty"`
	Traffic    int            `json:"traffic,omitempty"`
	CouponCode string         `json:"couponCode,omitempty"`
}

// ExecuteResponse is the response from POST /v2/order/:serviceId/execute.
type ExecuteResponse struct {
	ID             string `json:"id"`
	PeriodInMonths string `json:"periodInMonths"`
	Bandwidth      string `json:"bandwidth"`
	TotalPrice     string `json:"totalPrice"`
}

// OrderDetailsResponse is the response from GET /orders/:id.
type OrderDetailsResponse struct {
	ID             string `json:"id"`
	PeriodInMonths string `json:"periodInMonths"`
	Bandwidth      string `json:"bandwidth"`
	TotalPrice     string `json:"totalPrice"`
}

// ─── Proxy ────────────────────────────────────────────────────────────────────

// ProxyStatus constants from Proxy-Cheap.
const (
	ProxyStatusPending    = "PENDING"
	ProxyStatusInitiating = "INITIATING"
	ProxyStatusActive     = "ACTIVE"
	ProxyStatusExpired    = "EXPIRED"
	ProxyStatusCanceled   = "CANCELED"
)

// ProxyAuthentication holds auth details for a proxy.
type ProxyAuthentication struct {
	WhitelistedIPs []string `json:"whitelistedIps"`
	Username       string   `json:"username"`
	Password       string   `json:"password"`
}

// ProxyConnection holds connection details for a proxy.
type ProxyConnection struct {
	PublicIP   string `json:"publicIp"`
	ConnectIP  string `json:"connectIp"`
	IPVersion  string `json:"ipVersion,omitempty"`
	HTTPPort   int    `json:"httpPort"`
	HTTPSPort  int    `json:"httpsPort"`
	SOCKS5Port int    `json:"socks5Port"`
}

// ProxyBandwidth holds bandwidth usage info.
type ProxyBandwidth struct {
	Total int `json:"total"` // GB
	Used  int `json:"used"`  // GB
}

// ProxyMetadata holds additional proxy metadata.
type ProxyMetadata struct {
	ISPName string `json:"ispName"`
}

// Proxy is a single proxy resource from Proxy-Cheap.
type Proxy struct {
	ID             int64               `json:"id"`
	Status         string              `json:"status"`
	NetworkType    string              `json:"networkType"`
	CountryCode    string              `json:"countryCode"`
	Authentication ProxyAuthentication `json:"authentication"`
	Connection     ProxyConnection     `json:"connection"`
	ProxyType      string              `json:"proxyType"` // HTTP or SOCKS5
	CreatedAt      time.Time           `json:"createdAt"`
	ExpiresAt      time.Time           `json:"expiresAt"`
	Metadata       ProxyMetadata       `json:"metadata"`
	Bandwidth      *ProxyBandwidth     `json:"bandwidth,omitempty"`
}

// ─── Account ─────────────────────────────────────────────────────────────────

// AccountBalance is the response from GET /account/balance.
type AccountBalance struct {
	Balance float64 `json:"balance"`
}

// ─── Webhooks ─────────────────────────────────────────────────────────────────

// Webhook event name constants.
const (
	WebhookEventStatusChanged              = "proxy.status.changed"
	WebhookEventBandwidthAdded             = "proxy.bandwidth.added"
	WebhookEventMaintenanceWindowCreated   = "proxy.maintenance_window.created"
	WebhookEventMaintenanceWindowCancelled = "proxy.maintenance_window.cancelled"
)

// WebhookStatusChanged is the payload for proxy.status.changed.
type WebhookStatusChanged struct {
	ProxyID   int64  `json:"proxyId"`
	OldStatus string `json:"oldStatus"`
	Status    string `json:"status"`
}

// WebhookBandwidthAdded is the payload for proxy.bandwidth.added.
type WebhookBandwidthAdded struct {
	ProxyID     int64 `json:"proxyId"`
	TrafficInGB int   `json:"trafficInGb"`
}

// WebhookMaintenanceWindow is the payload for maintenance window events.
type WebhookMaintenanceWindow struct {
	ProxyID              int64     `json:"proxyId"`
	MaintenanceWindowID  string    `json:"maintenanceWindowId"`
	StartsAt             time.Time `json:"startsAt,omitempty"`
	EndsAt               time.Time `json:"endsAt,omitempty"`
}

// ─── API Error ────────────────────────────────────────────────────────────────

// APIError represents a Proxy-Cheap API error response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("proxy-cheap API error: status=%d body=%s", e.StatusCode, e.Body)
}
