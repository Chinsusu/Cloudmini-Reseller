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
	"github.com/pvp/vps-service/internal/domain"
	"github.com/pvp/vps-service/internal/usecase"
)

// Handler holds vps-service usecase dependencies.
type Handler struct {
	provisionUC *usecase.ProvisionUsecase
	instanceUC  *usecase.InstanceUsecase
	planRepo    domain.IPlanRepository
	logger      *slog.Logger
}

func NewHandler(
	provisionUC *usecase.ProvisionUsecase,
	instanceUC *usecase.InstanceUsecase,
	planRepo domain.IPlanRepository,
	logger *slog.Logger,
) *Handler {
	return &Handler{provisionUC: provisionUC, instanceUC: instanceUC, planRepo: planRepo, logger: logger}
}

// GET /api/v1/vps/plans
func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.planRepo.List(r.Context())
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, plans)
}

// POST /api/v1/vps/orders — returns 202 Accepted
func (h *Handler) CreateVPS(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	var req struct {
		PlanID         string `json:"plan_id"`
		Hostname       string `json:"hostname"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid JSON")
		return
	}
	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		apierror.Respond(w, r, http.StatusBadRequest, apierror.CodeValidationError, "invalid plan_id")
		return
	}

	result, err := h.provisionUC.CreateVPS(r.Context(), usecase.OrderRequest{
		UserID:         userID,
		PlanID:         planID,
		Hostname:       req.Hostname,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusAccepted, result)
}

// GET /api/v1/vps/instances
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	p := pagination.Parse(r)
	insts, total, err := h.instanceUC.ListInstances(r.Context(), userID, p.Offset, p.Limit)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSONWithMeta(w, http.StatusOK, insts, pagination.NewMeta(p, total))
}

// GET /api/v1/vps/instances/{id}
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	inst, err := h.instanceUC.GetInstance(r.Context(), instID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, inst)
}

// POST /api/v1/vps/instances/{id}/start
func (h *Handler) StartInstance(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.instanceUC.StartInstance(r.Context(), instID, userID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/vps/instances/{id}/stop
func (h *Handler) StopInstance(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.instanceUC.StopInstance(r.Context(), instID, userID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/vps/instances/{id}/reboot
func (h *Handler) RebootInstance(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.instanceUC.RebootInstance(r.Context(), instID, userID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/vps/instances/{id}
func (h *Handler) TerminateInstance(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	if err := h.instanceUC.TerminateInstance(r.Context(), instID, userID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/vps/instances/{id}/console
func (h *Handler) GetConsole(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	url, err := h.instanceUC.GetConsoleURL(r.Context(), instID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, map[string]string{"console_url": url})
}

// POST /api/v1/vps/instances/{id}/snapshots
func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	var req struct {
		Name string `json:"name"`
		Desc string `json:"description"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	snap, err := h.instanceUC.CreateSnapshot(r.Context(), instID, userID, req.Name, req.Desc)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusCreated, snap)
}

// GET /api/v1/vps/instances/{id}/snapshots
func (h *Handler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	snaps, err := h.instanceUC.ListSnapshots(r.Context(), instID, userID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	apierror.RespondJSON(w, http.StatusOK, snaps)
}

// DELETE /api/v1/vps/instances/{id}/snapshots/{snap_id}
func (h *Handler) DeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(middleware.GetUserID(r.Context()))
	instID := mustParseUUID(chi.URLParam(r, "id"))
	snapID := mustParseUUID(chi.URLParam(r, "snap_id"))
	if err := h.instanceUC.DeleteSnapshot(r.Context(), instID, snapID, userID); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// NewRouter builds the chi router for vps-service.
func NewRouter(h *Handler, jwtSecret []byte) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.CORS, middleware.Recovery(h.logger))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		apierror.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "vps-service"})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		r.Route("/api/v1/vps", func(r chi.Router) {
			r.Get("/plans", h.ListPlans)
			r.Post("/orders", h.CreateVPS)

			r.Route("/instances", func(r chi.Router) {
				r.Get("/", h.ListInstances)
				r.Get("/{id}", h.GetInstance)
				r.Post("/{id}/start", h.StartInstance)
				r.Post("/{id}/stop", h.StopInstance)
				r.Post("/{id}/reboot", h.RebootInstance)
				r.Delete("/{id}", h.TerminateInstance)
				r.Get("/{id}/console", h.GetConsole)
				r.Post("/{id}/snapshots", h.CreateSnapshot)
				r.Get("/{id}/snapshots", h.ListSnapshots)
				r.Delete("/{id}/snapshots/{snap_id}", h.DeleteSnapshot)
			})
		})
	})
	return r
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrPlanNotFound), errors.Is(err, domain.ErrInstanceNotFound), errors.Is(err, domain.ErrSnapshotNotFound):
		apierror.Respond(w, r, http.StatusNotFound, apierror.CodeNotFound, err.Error())
	case errors.Is(err, domain.ErrInstanceNotOwned):
		apierror.Respond(w, r, http.StatusForbidden, apierror.CodeForbidden, err.Error())
	case errors.Is(err, domain.ErrNoAvailableNode):
		apierror.Respond(w, r, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable, err.Error())
	case errors.Is(err, domain.ErrInstanceTerminated), errors.Is(err, domain.ErrInstanceNotRunning):
		apierror.Respond(w, r, http.StatusConflict, apierror.CodeConflict, err.Error())
	default:
		h.logger.ErrorContext(r.Context(), "vps handler error", slog.String("error", err.Error()))
		apierror.Respond(w, r, http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
	}
}

func mustParseUUID(s string) uuid.UUID { id, _ := uuid.Parse(s); return id }
