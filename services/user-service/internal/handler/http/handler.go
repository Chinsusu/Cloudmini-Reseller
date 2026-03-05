package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pvp/pkg/apierror"
	"github.com/pvp/pkg/middleware"
	"github.com/pvp/pkg/pagination"
	"github.com/pvp/user-service/internal/domain"
	"github.com/pvp/user-service/internal/usecase"
)

// Handler holds all usecase dependencies for HTTP handling.
type Handler struct {
	authUC   *usecase.AuthUsecase
	userUC   *usecase.UserUsecase
	apiKeyUC *usecase.APIKeyUsecase
	logger   *slog.Logger
}

// NewHandler creates a Handler with injected usecases.
func NewHandler(
	authUC *usecase.AuthUsecase,
	userUC *usecase.UserUsecase,
	apiKeyUC *usecase.APIKeyUsecase,
	logger *slog.Logger,
) *Handler {
	return &Handler{authUC: authUC, userUC: userUC, apiKeyUC: apiKeyUC, logger: logger}
}

// ─── Auth Handlers ────────────────────────────────────────────────────────────

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		Phone    string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "Invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.FullName) == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "email, password, and full_name are required")
		return
	}

	result, err := h.authUC.Register(r.Context(), usecase.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	apierror.RespondJSON(w, http.StatusCreated, map[string]string{
		"user_id": result.UserID,
		"message": "Registration successful. Please verify your email.",
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "Invalid JSON body")
		return
	}

	result, err := h.authUC.Login(r.Context(), usecase.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	apierror.RespondJSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_at":    result.ExpiresAt,
		"user_id":       result.UserID,
		"role":          result.Role,
	})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "refresh_token is required")
		return
	}

	result, err := h.authUC.RefreshToken(r.Context(), usecase.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
		IPAddress:    r.RemoteAddr,
		UserAgent:    r.UserAgent(),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	apierror.RespondJSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_at":    result.ExpiresAt,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "refresh_token is required")
		return
	}

	if err := h.authUC.Logout(r.Context(), req.RefreshToken); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── User Handlers ────────────────────────────────────────────────────────────

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	acc, err := h.userUC.GetProfile(r.Context(), userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, toAccountResponse(acc))
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	var req struct {
		FullName string `json:"full_name"`
		Phone    string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "Invalid JSON body")
		return
	}

	acc, err := h.userUC.UpdateProfile(r.Context(), userID, usecase.UpdateProfileRequest{
		FullName: req.FullName,
		Phone:    req.Phone,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, toAccountResponse(acc))
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "Invalid JSON body")
		return
	}

	if err := h.userUC.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── API Key Handlers ─────────────────────────────────────────────────────────

func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	keys, err := h.apiKeyUC.ListAPIKeys(r.Context(), userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, keys)
}

func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "name is required")
		return
	}

	result, err := h.apiKeyUC.CreateAPIKey(r.Context(), usecase.CreateAPIKeyRequest{
		UserID: userID,
		Name:   req.Name,
		Scopes: req.Scopes,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	apierror.RespondJSON(w, http.StatusCreated, map[string]any{
		"key":    result.RawKey, // shown once
		"api_key": result.APIKey,
	})
}

func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	keyID := mustParseUUID(chi.URLParam(r, "id"))

	if err := h.apiKeyUC.RevokeAPIKey(r.Context(), userID, keyID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Admin Handlers ───────────────────────────────────────────────────────────

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	p := pagination.Parse(r)
	users, total, err := h.userUC.ListUsers(r.Context(), p.Offset, p.Limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	meta := pagination.NewMeta(p, total)
	apierror.RespondJSONWithMeta(w, http.StatusOK, users, meta)
}

func (h *Handler) AdminGetUser(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(chi.URLParam(r, "id"))
	acc, err := h.userUC.GetProfile(r.Context(), userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, toAccountResponse(acc))
}

func (h *Handler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	adminID := mustParseUUID(middleware.GetUserID(r.Context()))
	userID := mustParseUUID(chi.URLParam(r, "id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "Invalid JSON body")
		return
	}
	if err := h.userUC.UpdateStatus(r.Context(), adminID, userID, req.Status); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AdminUpdateRole(w http.ResponseWriter, r *http.Request) {
	adminID := mustParseUUID(middleware.GetUserID(r.Context()))
	userID := mustParseUUID(chi.URLParam(r, "id"))
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "Invalid JSON body")
		return
	}
	if err := h.userUC.UpdateRole(r.Context(), adminID, userID, req.Role); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrAPIKeyNotFound), errors.Is(err, domain.ErrSessionNotFound):
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrTokenInvalid), errors.Is(err, domain.ErrTokenExpired):
		apierror.Respond(w, r, http.StatusUnauthorized, apierror.CodeUnauthorized, err.Error())
	case errors.Is(err, domain.ErrAccountSuspended), errors.Is(err, domain.ErrEmailNotVerified):
		apierror.Respond(w, r, http.StatusForbidden, apierror.CodeForbidden, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		apierror.Respond(w, r, http.StatusConflict, apierror.CodeConflict, err.Error())
	case errors.Is(err, domain.ErrWeakPassword):
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, err.Error())
	default:
		requestID := middleware.GetRequestID(r.Context())
		h.logger.ErrorContext(r.Context(), "unhandled error",
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
		)
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "An internal error occurred")
	}
}

func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

type accountResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	FullName      string  `json:"full_name"`
	Phone         string  `json:"phone"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	EmailVerified bool    `json:"email_verified"`
	ResellerID    *string `json:"reseller_id,omitempty"`
}

func toAccountResponse(acc *domain.Account) accountResponse {
	r := accountResponse{
		ID:            acc.ID.String(),
		Email:         acc.Email,
		FullName:      acc.FullName,
		Phone:         acc.Phone,
		Role:          acc.Role,
		Status:        acc.Status,
		EmailVerified: acc.EmailVerified,
	}
	if acc.ResellerID != nil {
		s := acc.ResellerID.String()
		r.ResellerID = &s
	}
	return r
}
