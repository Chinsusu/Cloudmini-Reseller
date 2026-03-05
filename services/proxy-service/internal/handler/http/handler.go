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
)

// Handler holds proxy usecase dependencies.
type Handler struct {
	orderUC *usecase.OrderUsecase
	logger  *slog.Logger
}

func NewHandler(orderUC *usecase.OrderUsecase, logger *slog.Logger) *Handler {
	return &Handler{orderUC: orderUC, logger: logger}
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

// NewRouter builds the chi router for proxy-service.
func NewRouter(h *Handler, jwtSecret []byte) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.CORS, middleware.Recovery(h.logger))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierror.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "proxy-service"})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		r.Route("/api/v1/proxy", func(r chi.Router) {
			r.Get("/orders", h.ListOrders)
			r.Post("/orders", h.CreateOrder)
			r.Get("/orders/{id}", h.GetOrder)
			r.Delete("/orders/{id}", h.CancelOrder)
			r.Get("/orders/{id}/credentials", h.GetCredentials)
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
