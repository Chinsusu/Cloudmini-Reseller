package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pvp/pkg/apierror"
	"github.com/pvp/pkg/middleware"
	"github.com/pvp/pkg/pagination"
	"github.com/pvp/proxy-service/internal/domain"
	"github.com/pvp/proxy-service/internal/usecase"
	"github.com/shopspring/decimal"
)

// Handler holds proxy usecase dependencies.
type Handler struct {
	orderUC     *usecase.OrderUsecase
	productRepo domain.IProductRepository
	logger      *slog.Logger
}

func NewHandler(orderUC *usecase.OrderUsecase, productRepo domain.IProductRepository, logger *slog.Logger) *Handler {
	return &Handler{orderUC: orderUC, productRepo: productRepo, logger: logger}
}

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
		ProductID      string `json:"product_id"`
		Quantity       int    `json:"quantity"`
		IdempotencyKey string `json:"idempotency_key"`
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

// ─── Admin Routes ─────────────────────────────────────────────────────────────

// GET /api/v1/admin/proxy/products
func (h *Handler) AdminListProducts(w http.ResponseWriter, r *http.Request) {
	p := pagination.Parse(r)
	products, total, err := h.productRepo.List(r.Context(), "", "", "", p.Offset, p.Limit)
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

// ─── Router ───────────────────────────────────────────────────────────────────

// NewRouter builds the chi router for proxy-service.
func NewRouter(h *Handler, jwtSecret []byte, auditLogger middleware.AuditLogger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.CORS, middleware.Recovery(h.logger))
	r.Use(middleware.AuditLog(auditLogger))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierror.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "proxy-service"})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		r.Route("/api/v1/proxy", func(r chi.Router) {
			r.Get("/products", h.ListProducts)
			r.Get("/orders", h.ListOrders)
			r.Post("/orders", h.CreateOrder)
			r.Get("/orders/{id}", h.GetOrder)
			r.Delete("/orders/{id}", h.CancelOrder)
			r.Get("/orders/{id}/credentials", h.GetCredentials)
		})

		// Admin endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "super_admin"))
			r.Route("/api/v1/admin/proxy", func(r chi.Router) {
				r.Get("/products", h.AdminListProducts)
				r.Post("/products", h.AdminCreateProduct)
				r.Put("/products/{id}/toggle", h.AdminToggleProduct)
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
