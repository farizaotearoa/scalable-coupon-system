package coupon

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

func NewHandler(db *pgxpool.Pool,
	log *slog.Logger,
) *Handler {
	repo := NewRepository(db, log)
	svc := NewService(repo, log)
	return &Handler{
		service: svc,
		log:     log,
	}
}

func (h *Handler) CreateCoupon(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.log.Info("create coupon request received")
	defer h.log.Info("create coupon request completed")

	var req CreateCouponRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.log.Warn("failed to decode request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.service.CreateCoupon(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCouponAlreadyExists) {
			http.Error(w,
				fmt.Sprintf("%s: %s", err.Error(), req.Name),
				http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) ClaimCoupon(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.log.Info("claim coupon request received")
	defer h.log.Info("claim coupon request completed")

	var req ClaimCouponRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.log.Warn("failed to decode request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.service.ClaimCoupon(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrCouponAlreadyClaimed):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case errors.Is(err, ErrCouponOutOfStock):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case errors.Is(err, ErrCouponNotFound):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) GetCouponDetails(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.log.Info("get coupon details request received")
	defer h.log.Info("get coupon details request completed")

	name := r.PathValue("name")
	if name == "" {
		h.log.Warn("coupon name missing in request")
		http.Error(w, "coupon name required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetCouponDetails(r.Context(), name)
	if err != nil {
		if errors.Is(err, ErrCouponNotFound) {
			http.Error(w,
				fmt.Sprintf("%s: %s", err.Error(), name),
				http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
