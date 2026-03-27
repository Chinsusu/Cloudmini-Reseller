package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pvp/pkg/apierror"
	"github.com/pvp/pkg/middleware"
	"github.com/pvp/pkg/pagination"
	"github.com/pvp/proxy-service/internal/domain"
	proxycheap "github.com/pvp/proxy-service/internal/providers/proxy_cheap"
	"github.com/pvp/proxy-service/internal/providers/vpm"
	"github.com/pvp/proxy-service/internal/usecase"
	"github.com/shopspring/decimal"
)

// Handler holds proxy usecase dependencies.
type Handler struct {
	orderUC          *usecase.OrderUsecase
	lockUC           *usecase.LockUsecase
	orderRepo        domain.IOrderRepository
	orderEvtRepo     domain.IOrderEventRepository
	productRepo      domain.IProductRepository
	providerRepo     domain.IProviderRepository
	webhookHandler   http.Handler
	proxyCheapClient *proxycheap.Client
	logger           *slog.Logger
}

func NewHandler(orderUC *usecase.OrderUsecase, lockUC *usecase.LockUsecase, orderRepo domain.IOrderRepository, orderEvtRepo domain.IOrderEventRepository, productRepo domain.IProductRepository, providerRepo domain.IProviderRepository, webhookHandler http.Handler, logger *slog.Logger) *Handler {
	return &Handler{orderUC: orderUC, lockUC: lockUC, orderRepo: orderRepo, orderEvtRepo: orderEvtRepo, productRepo: productRepo, providerRepo: providerRepo, webhookHandler: webhookHandler, logger: logger}
}

// WithProxyCheapClient injects the proxy-cheap client into the handler (for service-options endpoint).
func (h *Handler) WithProxyCheapClient(c *proxycheap.Client) { h.proxyCheapClient = c }

// ─── Public / User Routes ──────────────────────────────────────────────────

// GET /api/v1/proxy/products
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	proxyType := r.URL.Query().Get("proxy_type")
	protocol := r.URL.Query().Get("protocol")
	location := r.URL.Query().Get("location")
	p := pagination.Parse(r)
	products, total, err := h.productRepo.List(r.Context(), proxyType, protocol, location, p.Offset, p.Limit)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "ListProducts error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, products, pagination.NewMeta(p, total))
}

// CreateOrder handles POST /api/v1/proxy/orders
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))

	var req struct {
		ProductID      string            `json:"product_id"`
		Quantity       int               `json:"quantity"`
		IdempotencyKey string            `json:"idempotency_key"`
		Metadata       map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid product_id")
		return
	}

	order, err := h.orderUC.CreateOrder(r.Context(), usecase.CreateOrderRequest{
		UserID:         userID,
		ProductID:      productID,
		Quantity:       req.Quantity,
		IdempotencyKey: req.IdempotencyKey,
		Metadata:       req.Metadata,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /api/v1/proxy/orders/{id}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	orderID := mustParseUUID(chi.URLParam(r, "id"))
	order, err := h.orderUC.GetOrder(r.Context(), orderID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, order)
}

// ListOrders handles GET /api/v1/proxy/orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	p := pagination.Parse(r)
	orders, total, err := h.orderUC.ListOrders(r.Context(), userID, p.Offset, p.Limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, orders, pagination.NewMeta(p, total))
}

// GetCredentials handles GET /api/v1/proxy/orders/{id}/credentials
func (h *Handler) GetCredentials(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	orderID := mustParseUUID(chi.URLParam(r, "id"))
	creds, err := h.orderUC.GetCredentials(r.Context(), orderID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, creds)
}

// CancelOrder handles DELETE /api/v1/proxy/orders/{id}
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	orderID := mustParseUUID(chi.URLParam(r, "id"))
	var req struct{ Reason string `json:"reason"` }
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.orderUC.CancelOrder(r.Context(), orderID, userID, req.Reason); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RenewOrder handles POST /api/v1/proxy/orders/{id}/renew
// Renews an expired order during its grace period.
// New expiry = COALESCE(custom_expires_at, expires_at) + product.DurationDays.
func (h *Handler) RenewOrder(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	orderID := mustParseUUID(chi.URLParam(r, "id"))

	order, err := h.orderUC.RenewOrder(r.Context(), orderID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, order)
}

// PatchOrder handles PATCH /api/v1/proxy/orders/{id}
// Allows users to set custom_price and custom_expires_at on their own orders.
func (h *Handler) PatchOrder(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	orderID := mustParseUUID(chi.URLParam(r, "id"))

	// Verify ownership
	order, err := h.orderUC.GetOrder(r.Context(), orderID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	_ = order // ownership verified

	var req struct {
		CustomPrice     *string `json:"custom_price"`
		CustomExpiresAt *string `json:"custom_expires_at"` // RFC3339
		AdminNote       string  `json:"admin_note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}

	var customPrice *decimal.Decimal
	if req.CustomPrice != nil && *req.CustomPrice != "" {
		v, _ := decimal.NewFromString(*req.CustomPrice)
		customPrice = &v
	}

	var customExpiresAt *time.Time
	if req.CustomExpiresAt != nil && *req.CustomExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.CustomExpiresAt)
		if err != nil {
			apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid custom_expires_at, use RFC3339")
			return
		}
		customExpiresAt = &t
	}

	if err := h.orderRepo.UpdateOrder(r.Context(), orderID, customPrice, customExpiresAt, req.AdminNote); err != nil {
		h.logger.ErrorContext(r.Context(), "PatchOrder error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	// Log order.patched event
	payload := map[string]any{}
	if req.CustomPrice != nil {
		payload["custom_price"] = *req.CustomPrice
	}
	if req.CustomExpiresAt != nil {
		payload["custom_expires_at"] = *req.CustomExpiresAt
	}
	if req.AdminNote != "" {
		payload["admin_note"] = req.AdminNote
	}
	_ = h.orderEvtRepo.Log(r.Context(), orderID, domain.EventOrderPatched, payload)
	w.WriteHeader(http.StatusNoContent)
}

// GetOrderEvents handles GET /api/v1/proxy/orders/{id}/events
func (h *Handler) GetOrderEvents(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	orderID := mustParseUUID(chi.URLParam(r, "id"))
	// Verify ownership
	if _, err := h.orderUC.GetOrder(r.Context(), orderID, userID); err != nil {
		h.handleError(w, r, err)
		return
	}
	events, err := h.orderEvtRepo.ListByOrder(r.Context(), orderID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "GetOrderEvents error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	apierror.RespondJSON(w, http.StatusOK, events)
}

// ─── Admin Routes ─────────────────────────────────────────────────────────────

// GET /api/v1/admin/proxy/products
func (h *Handler) AdminListProducts(w http.ResponseWriter, r *http.Request) {
	p := pagination.Parse(r)
	products, total, err := h.productRepo.AdminList(r.Context(), p.Offset, p.Limit)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "AdminListProducts error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, products, pagination.NewMeta(p, total))
}

// POST /api/v1/admin/proxy/products
func (h *Handler) AdminCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProviderID   string  `json:"provider_id"`
		Name         string  `json:"name"`
		ProxyType    string  `json:"proxy_type"`
		Protocol     string  `json:"protocol"`
		Location     string  `json:"location"`
		DurationDays *int    `json:"duration_days"`
		BandwidthGB  *string `json:"bandwidth_gb"`
		BaseCost     string  `json:"base_cost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	baseCost, _ := decimal.NewFromString(req.BaseCost)
	providerID, _ := uuid.Parse(req.ProviderID)

	product := &domain.Product{
		ID:           uuid.New(),
		ProviderID:   providerID,
		Name:         req.Name,
		ProxyType:    req.ProxyType,
		Protocol:     req.Protocol,
		Location:     req.Location,
		DurationDays: req.DurationDays,
		BaseCost:     baseCost,
		IsActive:     true,
	}
	if req.BandwidthGB != nil {
		v, _ := decimal.NewFromString(*req.BandwidthGB)
		product.BandwidthGB = &v
	}

	if err := h.productRepo.Create(r.Context(), product); err != nil {
		h.logger.ErrorContext(r.Context(), "AdminCreateProduct error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	apierror.RespondJSON(w, http.StatusCreated, product)
}

// PUT /api/v1/admin/proxy/products/{id}/toggle
func (h *Handler) AdminToggleProduct(w http.ResponseWriter, r *http.Request) {
	productID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.productRepo.ToggleActive(r.Context(), productID); err != nil {
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/v1/admin/proxy/products/{id}
func (h *Handler) AdminUpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID := mustParseUUID(chi.URLParam(r, "id"))
	var req struct {
		Name         string  `json:"name"`
		ProxyType    string  `json:"proxy_type"`
		Protocol     string  `json:"protocol"`
		Location     string  `json:"location"`
		DurationDays *int    `json:"duration_days"`
		BandwidthGB  *string `json:"bandwidth_gb"`
		BaseCost     string  `json:"base_cost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	baseCost, _ := decimal.NewFromString(req.BaseCost)
	p := &domain.Product{
		ID: productID, Name: req.Name, ProxyType: req.ProxyType,
		Protocol: req.Protocol, Location: req.Location,
		DurationDays: req.DurationDays, BaseCost: baseCost,
	}
	if req.BandwidthGB != nil {
		v, _ := decimal.NewFromString(*req.BandwidthGB)
		p.BandwidthGB = &v
	}
	if err := h.productRepo.Update(r.Context(), p); err != nil {
		h.logger.ErrorContext(r.Context(), "AdminUpdateProduct error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/admin/proxy/products/{id}
func (h *Handler) AdminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.productRepo.Delete(r.Context(), productID); err != nil {
		h.logger.ErrorContext(r.Context(), "AdminDeleteProduct error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/admin/proxy/providers — list ALL providers (active + inactive)
func (h *Handler) AdminListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.providerRepo.ListAll(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "AdminListProviders error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
		return
	}
	apierror.RespondJSON(w, http.StatusOK, providers)
}

// POST /api/v1/admin/proxy/providers — create a new provider
func (h *Handler) AdminCreateProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string          `json:"name"`
		DisplayName string          `json:"display_name"`
		AdapterType string          `json:"adapter_type"`
		Config      json.RawMessage `json:"config"`
		Priority    int             `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	if req.Name == "" || req.AdapterType == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "name and adapter_type are required")
		return
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Name
	}
	if len(req.Config) == 0 {
		req.Config = json.RawMessage(`{}`)
	}

	provider := &domain.Provider{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		AdapterType: req.AdapterType,
		Config:      req.Config,
		IsActive:    true,
		Priority:    req.Priority,
	}

	if err := h.providerRepo.Create(r.Context(), provider); err != nil {
		h.logger.ErrorContext(r.Context(), "AdminCreateProvider error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, err.Error())
		return
	}
	apierror.RespondJSON(w, http.StatusCreated, provider)
}

// PUT /api/v1/admin/proxy/providers/{id} — update a provider
func (h *Handler) AdminUpdateProvider(w http.ResponseWriter, r *http.Request) {
	providerID := mustParseUUID(chi.URLParam(r, "id"))

	var req struct {
		Name        string          `json:"name"`
		DisplayName string          `json:"display_name"`
		AdapterType string          `json:"adapter_type"`
		Config      json.RawMessage `json:"config"`
		Priority    int             `json:"priority"`
		IsActive    bool            `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	if len(req.Config) == 0 || string(req.Config) == "null" {
		req.Config = json.RawMessage(`{}`)
	}

	provider := &domain.Provider{
		ID:          providerID,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		AdapterType: req.AdapterType,
		Config:      req.Config,
		IsActive:    req.IsActive,
		Priority:    req.Priority,
	}

	if err := h.providerRepo.Update(r.Context(), provider); err != nil {
		h.logger.ErrorContext(r.Context(), "AdminUpdateProvider error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, err.Error())
		return
	}
	apierror.RespondJSON(w, http.StatusOK, provider)
}

// PUT /api/v1/admin/proxy/providers/{id}/toggle — toggle active state
func (h *Handler) AdminToggleProvider(w http.ResponseWriter, r *http.Request) {
	providerID := mustParseUUID(chi.URLParam(r, "id"))

	existing, err := h.providerRepo.GetByID(r.Context(), providerID)
	if err != nil {
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, "provider not found")
		return
	}

	newActive := !existing.IsActive
	if err := h.providerRepo.ToggleActive(r.Context(), providerID, newActive); err != nil {
		h.logger.ErrorContext(r.Context(), "AdminToggleProvider error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, err.Error())
		return
	}
	existing.IsActive = newActive
	apierror.RespondJSON(w, http.StatusOK, existing)
}

// DELETE /api/v1/admin/proxy/providers/{id} — delete a provider
func (h *Handler) AdminDeleteProvider(w http.ResponseWriter, r *http.Request) {
	providerID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.providerRepo.Delete(r.Context(), providerID); err != nil {
		h.logger.ErrorContext(r.Context(), "AdminDeleteProvider error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/admin/proxy/providers/{id}/groups — fetch groups from a VPM provider
func (h *Handler) AdminGetProviderGroups(w http.ResponseWriter, r *http.Request) {
	providerID := mustParseUUID(chi.URLParam(r, "id"))

	provider, err := h.providerRepo.GetByID(r.Context(), providerID)
	if err != nil {
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, "provider not found")
		return
	}
	if provider.AdapterType != "vpm" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "groups only available for VPM providers")
		return
	}

	var cfg struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := json.Unmarshal(provider.Config, &cfg); err != nil || cfg.BaseURL == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid provider config")
		return
	}

	client := vpm.NewClient(cfg.BaseURL, cfg.APIKey)
	groups, err := client.ListGroups(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "AdminGetProviderGroups error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusBadGateway, apierror.CodeInternalError, "failed to fetch groups from provider")
		return
	}
	apierror.RespondJSON(w, http.StatusOK, groups)
}

// GET /api/v1/proxy/products/{id}/groups — public: fetch groups for a product
func (h *Handler) GetProductGroups(w http.ResponseWriter, r *http.Request) {
	productID := mustParseUUID(chi.URLParam(r, "id"))

	product, err := h.productRepo.GetByID(r.Context(), productID)
	if err != nil {
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, "product not found")
		return
	}

	// Read group_ids from product metadata
	var pmeta map[string]string
	if product.Metadata != nil {
		_ = json.Unmarshal(product.Metadata, &pmeta)
	}
	allowedIDs := map[string]bool{}
	if gids, ok := pmeta["group_ids"]; ok && gids != "" {
		for _, id := range splitCSV(gids) {
			allowedIDs[id] = true
		}
	}

	// Fetch provider
	provider, err := h.providerRepo.GetByID(r.Context(), product.ProviderID)
	if err != nil || provider.AdapterType != "vpm" {
		apierror.RespondJSON(w, http.StatusOK, []any{}) // no groups for non-VPM
		return
	}

	var cfg struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := json.Unmarshal(provider.Config, &cfg); err != nil || cfg.BaseURL == "" {
		apierror.RespondJSON(w, http.StatusOK, []any{})
		return
	}

	client := vpm.NewClient(cfg.BaseURL, cfg.APIKey)
	groups, err := client.ListGroups(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "GetProductGroups error", slog.String("error", err.Error()))
		apierror.RespondJSON(w, http.StatusOK, []any{})
		return
	}

	// Filter to only allowed groups if product has group_ids set
	if len(allowedIDs) > 0 {
		filtered := make([]vpm.ProxyGroup, 0, len(allowedIDs))
		for _, g := range groups {
			if allowedIDs[g.ID] {
				filtered = append(filtered, g)
			}
		}
		groups = filtered
	}

	apierror.RespondJSON(w, http.StatusOK, groups)
}

// splitCSV splits a comma-separated string into trimmed non-empty parts.
func splitCSV(s string) []string {
	parts := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// GET /api/v1/admin/proxy/service-options?service_id=X&plan_id=Y
func (h *Handler) AdminGetServiceOptions(w http.ResponseWriter, r *http.Request) {
	if h.proxyCheapClient == nil {
		// Adapter not configured — return empty
		apierror.RespondJSON(w, http.StatusOK, map[string]any{
			"countries": []string{},
			"isps":      map[string]any{},
			"note":      "proxy-cheap adapter not configured",
		})
		return
	}
	serviceID := r.URL.Query().Get("service_id")
	planID := r.URL.Query().Get("plan_id")
	if serviceID == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "service_id is required")
		return
	}
	opts, err := h.proxyCheapClient.GetServiceOptions(r.Context(), serviceID, planID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "AdminGetServiceOptions error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "failed to fetch service options")
		return
	}
	apierror.RespondJSON(w, http.StatusOK, opts)
}

// GET /api/v1/admin/proxy/user-orders?user_id=xxx
func (h *Handler) AdminGetUserOrders(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "user_id is required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid user_id")
		return
	}
	p := pagination.Parse(r)
	orders, total, err := h.orderUC.ListOrders(r.Context(), userID, 0, p.Limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, orders, pagination.NewMeta(p, total))
}

// AdminOrderAction handles PUT /api/v1/admin/proxy/orders/{id}/action
// body: {"action": "lock"|"unlock", "reason": "optional note"}
// lock  → suspend proxy at provider + status "suspended"
// unlock → resume proxy at provider + status "active"
func (h *Handler) AdminOrderAction(w http.ResponseWriter, r *http.Request) {
	orderID := mustParseUUID(chi.URLParam(r, "id"))

	var req struct {
		Action string `json:"action"` // "lock" | "unlock"
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}

	lockReq := usecase.LockOrderRequest{OrderID: orderID, Reason: req.Reason}

	var err error
	switch req.Action {
	case "lock":
		err = h.lockUC.LockOrder(r.Context(), lockReq)
	case "unlock":
		err = h.lockUC.UnlockOrder(r.Context(), lockReq)
	default:
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "action must be 'lock' or 'unlock'")
		return
	}

	if err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Router ───────────────────────────────────────────────────────────────────

// NewRouter builds the chi router for proxy-service.
func NewRouter(h *Handler, jwtSecret []byte, auditLogger middleware.AuditLogger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.CORS, middleware.Recovery(h.logger))
	r.Use(middleware.AuditLog(auditLogger))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierror.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "proxy-service"})
	})

	// Public webhook endpoint — authentication via HMAC (inside handler)
	if h.webhookHandler != nil {
		r.Post("/webhooks/proxy-cheap", h.webhookHandler.ServeHTTP)
	}

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		r.Route("/api/v1/proxy", func(r chi.Router) {
			r.Get("/products", h.ListProducts)
			r.Get("/products/{id}/groups", h.GetProductGroups)
			r.Get("/orders", h.ListOrders)
				r.Post("/orders", h.CreateOrder)
				r.Get("/orders/{id}", h.GetOrder)
				r.Patch("/orders/{id}", h.PatchOrder)
			r.Delete("/orders/{id}", h.CancelOrder)
			r.Get("/orders/{id}/credentials", h.GetCredentials)
				r.Get("/orders/{id}/events", h.GetOrderEvents)
				r.Post("/orders/{id}/renew", h.RenewOrder)
		})

		// Admin endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "super_admin"))
			r.Route("/api/v1/admin/proxy", func(r chi.Router) {
				r.Get("/products", h.AdminListProducts)
				r.Post("/products", h.AdminCreateProduct)
				r.Put("/products/{id}", h.AdminUpdateProduct)
				r.Put("/products/{id}/toggle", h.AdminToggleProduct)
				r.Delete("/products/{id}", h.AdminDeleteProduct)
				r.Get("/providers", h.AdminListProviders)
				r.Post("/providers", h.AdminCreateProvider)
				r.Put("/providers/{id}", h.AdminUpdateProvider)
				r.Put("/providers/{id}/toggle", h.AdminToggleProvider)
				r.Delete("/providers/{id}", h.AdminDeleteProvider)
				r.Get("/providers/{id}/groups", h.AdminGetProviderGroups)
				r.Get("/service-options", h.AdminGetServiceOptions)
				r.Get("/user-orders", h.AdminGetUserOrders)
				// PUT /api/v1/admin/proxy/orders/{id}/action
				// body: {"action": "lock"|"unlock", "reason": "..."}
				r.Put("/orders/{id}/action", h.AdminOrderAction)
			})
		})
	})
	return r
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound), errors.Is(err, domain.ErrProductNotFound):
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, err.Error())
	case errors.Is(err, domain.ErrOrderNotCancellable):
		apierror.Respond(w, r, http.StatusConflict, apierror.CodeConflict, err.Error())
	case errors.Is(err, domain.ErrNoProviderAvailable):
		apierror.Respond(w, r, http.StatusServiceUnavailable, apierror.CodeProviderUnavailable, err.Error())
	default:
		h.logger.ErrorContext(r.Context(), "proxy handler error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
	}
}

func mustParseUUID(s string) uuid.UUID { id, _ := uuid.Parse(s); return id }
