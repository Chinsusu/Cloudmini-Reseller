package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pvp/billing-service/internal/domain"
	"github.com/pvp/billing-service/internal/usecase"
	"github.com/pvp/pkg/apierror"
	"github.com/pvp/pkg/middleware"
	"github.com/pvp/pkg/pagination"
	"github.com/shopspring/decimal"
)

// Handler holds billing usecase dependencies.
type Handler struct {
	walletUC   *usecase.WalletUsecase
	paymentUC  *usecase.PaymentUsecase
	pricingEng *usecase.PricingEngine
	logger     *slog.Logger
}

// NewHandler constructs the billing HTTP handler.
func NewHandler(w *usecase.WalletUsecase, p *usecase.PaymentUsecase, pr *usecase.PricingEngine, log *slog.Logger) *Handler {
	return &Handler{walletUC: w, paymentUC: p, pricingEng: pr, logger: log}
}

// ─── Wallet ───────────────────────────────────────────────────────────────────

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	wallet, err := h.walletUC.GetBalance(r.Context(), userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, wallet)
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	p := pagination.Parse(r)
	txns, total, err := h.walletUC.ListTransactions(r.Context(), userID, p.Offset, p.Limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, txns, pagination.NewMeta(p, total))
}

// ─── Payments ─────────────────────────────────────────────────────────────────

func (h *Handler) CreateDeposit(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))

	var req struct {
		Gateway  string  `json:"gateway"`
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	if req.Gateway == "" || req.Amount <= 0 {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "gateway and amount are required")
		return
	}

	// Get user wallet to find wallet_id
	wallet, err := h.walletUC.GetBalance(r.Context(), userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	result, err := h.paymentUC.CreateDeposit(r.Context(), usecase.CreateDepositRequest{
		UserID:   userID,
		WalletID: wallet.ID,
		Gateway:  req.Gateway,
		Amount:   decimal.NewFromFloat(req.Amount),
		Currency: req.Currency,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	apierror.RespondJSON(w, http.StatusCreated, map[string]any{
		"payment":      result.Payment,
		"checkout_url": result.CheckoutURL,
	})
}

func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	// In production: verify stripe signature with stripe.ConstructEvent()
	var payload struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID       string  `json:"id"`
				Amount   float64 `json:"amount_received"`
				Currency string  `json:"currency"`
				Metadata struct {
					UserID string `json:"user_id"`
				} `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Type != "payment_intent.succeeded" {
		w.WriteHeader(http.StatusOK) // acknowledge non-handled events
		return
	}

	amount := decimal.NewFromFloat(payload.Data.Object.Amount / 100) // Stripe uses cents
	if err := h.paymentUC.HandleWebhook(
		r.Context(), "stripe", payload.Data.Object.ID, amount,
	); err != nil {
		h.logger.ErrorContext(r.Context(), "stripe webhook error", slog.String("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ─── Admin ─────────────────────────────────────────────────────────────────────

func (h *Handler) AdminListWallets(w http.ResponseWriter, r *http.Request) {
	p := pagination.Parse(r)
	_ = p // pagination used when a proper ListWallets usecase is added
	// Admin wallet listing — placeholder response until ListWallets usecase is implemented
	apierror.RespondJSON(w, http.StatusOK, []any{})
}

func (h *Handler) AdminAdjustBalance(w http.ResponseWriter, r *http.Request) {
	adminID := mustParseUUID(middleware.GetUserID(r.Context()))
	var req struct {
		UserID      string  `json:"user_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	userID := mustParseUUID(req.UserID)
	amount := decimal.NewFromFloat(req.Amount)

	var txn *domain.Transaction
	var txErr error
	if amount.GreaterThan(decimal.Zero) {
		aid := adminID
		txn, txErr = h.walletUC.Credit(r.Context(), userID, amount.Abs(), "adjustment", &aid, req.Description)
	} else {
		aid := adminID
		txn, txErr = h.walletUC.Deduct(r.Context(), usecase.DeductRequest{
			UserID: userID, Amount: amount.Abs(),
			ReferenceType: "adjustment", ReferenceID: &aid, Description: req.Description,
		})
	}
	if txErr != nil {
		h.handleError(w, r, txErr)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, txn)
}

// ─── Router ───────────────────────────────────────────────────────────────────

// NewRouter returns the chi router for billing-service.
func NewRouter(h *Handler, jwtSecret []byte) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.CORS, middleware.Recovery(h.logger))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierror.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "billing-service"})
	})

	// Stripe webhook — no JWT (signature verified separately)
	r.Post("/webhooks/stripe", h.StripeWebhook)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		r.Route("/api/v1/billing", func(r chi.Router) {
			r.Get("/wallet", h.GetWallet)
			r.Get("/transactions", h.ListTransactions)
			r.Post("/deposit", h.CreateDeposit)
		})

		r.Route("/api/v1/admin/billing", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "super_admin"))
			r.Get("/wallets", h.AdminListWallets)
			r.Post("/adjustment", h.AdminAdjustBalance)
		})
	})

	return r
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrWalletNotFound), errors.Is(err, domain.ErrPaymentNotFound):
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, err.Error())
	case errors.Is(err, domain.ErrInsufficientFunds):
		apierror.Respond(w, r, http.StatusPaymentRequired, apierror.CodeInsufficientFunds, err.Error())
	case errors.Is(err, domain.ErrInvalidAmount):
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, err.Error())
	default:
		h.logger.ErrorContext(r.Context(), "billing handler error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
	}
}

func mustParseUUID(s string) uuid.UUID { id, _ := uuid.Parse(s); return id }
