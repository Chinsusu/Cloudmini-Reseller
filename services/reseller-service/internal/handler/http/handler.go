package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pvp/pkg/apierror"
	"github.com/pvp/pkg/middleware"
	"github.com/pvp/pkg/pagination"
	"github.com/pvp/reseller-service/internal/domain"
	"github.com/pvp/reseller-service/internal/usecase"
	"github.com/shopspring/decimal"
)

// Handler holds reseller-service usecase dependencies.
type Handler struct {
	resellerUC *usecase.ResellerUsecase
	apiKeyUC   *usecase.APIKeyUsecase
	webhookUC  *usecase.WebhookUsecase
	logger     *slog.Logger
}

func NewHandler(
	resellerUC *usecase.ResellerUsecase,
	apiKeyUC *usecase.APIKeyUsecase,
	webhookUC *usecase.WebhookUsecase,
	logger *slog.Logger,
) *Handler {
	return &Handler{resellerUC: resellerUC, apiKeyUC: apiKeyUC, webhookUC: webhookUC, logger: logger}
}

// ─── Admin Routes ─────────────────────────────────────────────────────────────

// GET /api/v1/admin/resellers
func (h *Handler) AdminListResellers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	p := pagination.Parse(r)
	resellers, total, err := h.resellerUC.ListResellers(r.Context(), status, p.Offset, p.Limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, resellers, pagination.NewMeta(p, total))
}

// POST /api/v1/admin/resellers
func (h *Handler) AdminCreateReseller(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID        string `json:"user_id"`
		CompanyName   string `json:"company_name"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		Address       string `json:"address"`
		TaxID         string `json:"tax_id"`
		CommissionPct string `json:"commission_pct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	userID, _ := uuid.Parse(req.UserID)
	commission, _ := decimal.NewFromString(req.CommissionPct)

	reseller, err := h.resellerUC.CreateReseller(r.Context(), usecase.CreateResellerRequest{
		UserID: userID, CompanyName: req.CompanyName, Email: req.Email,
		Phone: req.Phone, Address: req.Address, TaxID: req.TaxID, CommissionPct: commission,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusCreated, reseller)
}

// PUT /api/v1/admin/resellers/{id}/approve
func (h *Handler) AdminApproveReseller(w http.ResponseWriter, r *http.Request) {
	id := mustParseUUID(chi.URLParam(r, "id"))
	reseller, err := h.resellerUC.ApproveReseller(r.Context(), id)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, reseller)
}

// PUT /api/v1/admin/resellers/{id}/suspend
func (h *Handler) AdminSuspendReseller(w http.ResponseWriter, r *http.Request) {
	id := mustParseUUID(chi.URLParam(r, "id"))
	var req struct{ Reason string `json:"reason"` }
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := h.resellerUC.SuspendReseller(r.Context(), id, req.Reason); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/v1/admin/resellers/{id}/pricing — admin sets cost + floor price
func (h *Handler) AdminSetPricing(w http.ResponseWriter, r *http.Request) {
	resellerID := mustParseUUID(chi.URLParam(r, "id"))
	var req struct {
		ProductID   string `json:"product_id"`
		ProductType string `json:"product_type"`
		CostPrice   string `json:"cost_price"`
		FloorPrice  string `json:"floor_price"`
		SellPrice   string `json:"sell_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	productID, _ := uuid.Parse(req.ProductID)
	costPrice, _ := decimal.NewFromString(req.CostPrice)
	floorPrice, _ := decimal.NewFromString(req.FloorPrice)
	sellPrice, _ := decimal.NewFromString(req.SellPrice)

	pricing, err := h.resellerUC.SetPricing(r.Context(), usecase.SetPricingRequest{
		ResellerID: resellerID, ProductID: productID, ProductType: req.ProductType,
		CostPrice: costPrice, FloorPrice: floorPrice, SellPrice: sellPrice,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, pricing)
}

// ─── Reseller Self-Service Routes ─────────────────────────────────────────────

// GET /api/v1/reseller/dashboard
func (h *Handler) ResellerDashboard(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	reseller, err := h.resellerUC.GetResellerByUserID(r.Context(), userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	// Return basic reseller info; production would aggregate orders + revenue
	apierror.RespondJSON(w, http.StatusOK, map[string]any{
		"reseller_id":   reseller.ID,
		"company_name":  reseller.CompanyName,
		"status":        reseller.Status,
		"commission_pct": reseller.CommissionPct,
	})
}

// GET /api/v1/reseller/users
func (h *Handler) ListSubAccounts(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	p := pagination.Parse(r)
	subs, total, err := h.resellerUC.ListSubAccounts(r.Context(), resellerID, p.Offset, p.Limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, subs, pagination.NewMeta(p, total))
}

// POST /api/v1/reseller/users
func (h *Handler) CreateSubAccount(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	var req struct {
		UserID      string `json:"user_id"`
		CreditLimit string `json:"credit_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	userID, _ := uuid.Parse(req.UserID)
	creditLimit, _ := decimal.NewFromString(req.CreditLimit)
	sub, err := h.resellerUC.CreateSubAccount(r.Context(), usecase.CreateSubAccountRequest{
		ResellerID: resellerID, UserID: userID, CreditLimit: creditLimit,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusCreated, sub)
}

// PUT /api/v1/reseller/users/{user_id}/credit
func (h *Handler) SetUserCredit(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	userID := mustParseUUID(chi.URLParam(r, "user_id"))
	var req struct{ CreditLimit string `json:"credit_limit"` }
	_ = json.NewDecoder(r.Body).Decode(&req)
	limit, _ := decimal.NewFromString(req.CreditLimit)
	if err := h.resellerUC.SetCreditLimit(r.Context(), userID, resellerID, limit); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/reseller/pricing
func (h *Handler) ListPricing(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	pricing, err := h.resellerUC.ListPricing(r.Context(), resellerID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, pricing)
}

// PUT /api/v1/reseller/pricing/{product_id}
func (h *Handler) SetPricing(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	productID := mustParseUUID(chi.URLParam(r, "product_id"))
	var req struct {
		ProductType string `json:"product_type"`
		CostPrice   string `json:"cost_price"`
		FloorPrice  string `json:"floor_price"`
		SellPrice   string `json:"sell_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	costPrice, _ := decimal.NewFromString(req.CostPrice)
	floorPrice, _ := decimal.NewFromString(req.FloorPrice)
	sellPrice, _ := decimal.NewFromString(req.SellPrice)

	pricing, err := h.resellerUC.SetPricing(r.Context(), usecase.SetPricingRequest{
		ResellerID: resellerID, ProductID: productID, ProductType: req.ProductType,
		CostPrice: costPrice, FloorPrice: floorPrice, SellPrice: sellPrice,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, pricing)
}

// ─── API Key Routes ───────────────────────────────────────────────────────────

// GET /api/v1/reseller/api-keys
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	keys, err := h.apiKeyUC.ListAPIKeys(r.Context(), resellerID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	// Never return key_hash in API response
	apierror.RespondJSON(w, http.StatusOK, keys)
}

// POST /api/v1/reseller/api-keys
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	var req struct {
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		ExpiresAt *string  `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}
	result, err := h.apiKeyUC.CreateAPIKey(r.Context(), resellerID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	// Return plain key only on creation — never stored
	apierror.RespondJSON(w, http.StatusCreated, map[string]any{
		"api_key": result.APIKey,
		"key":     result.PlainKey, // shown once
		"warning": "Store this key securely. It will not be shown again.",
	})
}

// DELETE /api/v1/reseller/api-keys/{id}
func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	keyID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.apiKeyUC.RevokeAPIKey(r.Context(), keyID, resellerID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Webhook Routes ───────────────────────────────────────────────────────────

// GET /api/v1/reseller/webhooks
func (h *Handler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	webhooks, err := h.webhookUC.ListWebhooks(r.Context(), resellerID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, webhooks)
}

// POST /api/v1/reseller/webhooks
func (h *Handler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	var req struct {
		URL    string   `json:"url"`
		Secret string   `json:"secret"`
		Events []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	webhook, err := h.webhookUC.CreateWebhook(r.Context(), resellerID, req.URL, req.Secret, req.Events)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusCreated, webhook)
}

// DELETE /api/v1/reseller/webhooks/{id}
func (h *Handler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	resellerID, ok := h.mustResellerID(w, r)
	if !ok { return }
	id := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.webhookUC.DeleteWebhook(r.Context(), id, resellerID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Router ───────────────────────────────────────────────────────────────────

// NewRouter builds the chi router for reseller-service.
func NewRouter(h *Handler, jwtSecret []byte, auditLogger middleware.AuditLogger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.CORS, middleware.Recovery(h.logger))
	r.Use(middleware.AuditLog(auditLogger))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierror.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "reseller-service"})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		// Admin endpoints (admin + super_admin roles only)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "super_admin"))
			r.Route("/api/v1/admin/resellers", func(r chi.Router) {
				r.Get("/", h.AdminListResellers)
				r.Post("/", h.AdminCreateReseller)
				r.Put("/{id}/approve", h.AdminApproveReseller)
				r.Put("/{id}/suspend", h.AdminSuspendReseller)
				r.Put("/{id}/pricing", h.AdminSetPricing)
			})
		})

		// Reseller self-service (reseller role)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("reseller"))
			r.Route("/api/v1/reseller", func(r chi.Router) {
				r.Get("/dashboard", h.ResellerDashboard)

				// Sub-accounts
				r.Get("/users", h.ListSubAccounts)
				r.Post("/users", h.CreateSubAccount)
				r.Put("/users/{user_id}/credit", h.SetUserCredit)

				// Custom pricing
				r.Get("/pricing", h.ListPricing)
				r.Put("/pricing/{product_id}", h.SetPricing)

				// API keys
				r.Get("/api-keys", h.ListAPIKeys)
				r.Post("/api-keys", h.CreateAPIKey)
				r.Delete("/api-keys/{id}", h.RevokeAPIKey)

				// Webhooks
				r.Get("/webhooks", h.ListWebhooks)
				r.Post("/webhooks", h.CreateWebhook)
				r.Delete("/webhooks/{id}", h.DeleteWebhook)
			})
		})
	})
	return r
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrResellerNotFound), errors.Is(err, domain.ErrAPIKeyNotFound),
		errors.Is(err, domain.ErrSubAccountNotFound), errors.Is(err, domain.ErrWebhookNotFound):
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		apierror.Respond(w, r, http.StatusForbidden, apierror.CodeForbidden, err.Error())
	case errors.Is(err, domain.ErrResellerAlreadyExists):
		apierror.Respond(w, r, http.StatusConflict, apierror.CodeConflict, err.Error())
	case errors.Is(err, domain.ErrResellerNotApproved), errors.Is(err, domain.ErrResellerSuspended):
		apierror.Respond(w, r, http.StatusForbidden, apierror.CodeForbidden, err.Error())
	case errors.Is(err, domain.ErrPriceBelowCost), errors.Is(err, domain.ErrPriceBelowFloor), errors.Is(err, domain.ErrPriceAboveCap):
		apierror.Respond(w, r, http.StatusUnprocessableEntity, apierror.CodeValidationError, err.Error())
	case errors.Is(err, domain.ErrAPIKeyRevoked), errors.Is(err, domain.ErrAPIKeyExpired):
		apierror.Respond(w, r, http.StatusUnauthorized, apierror.CodeUnauthorized, err.Error())
	default:
		h.logger.ErrorContext(r.Context(), "reseller handler error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
	}
}

// mustResellerID resolves the reseller account for the authenticated user.
// It looks up the reseller by user_id from the JWT context.
func (h *Handler) mustResellerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userIDStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		apierror.Respond(w, r, http.StatusUnauthorized, apierror.CodeUnauthorized, "invalid user identity")
		return uuid.Nil, false
	}
	reseller, err := h.resellerUC.GetResellerByUserID(r.Context(), userID)
	if err != nil {
		h.handleError(w, r, err)
		return uuid.Nil, false
	}
	return reseller.ID, true
}

func mustParseUUID(s string) uuid.UUID { id, _ := uuid.Parse(s); return id }
