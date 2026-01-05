package coupon

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

func NewHandler(db *pgxpool.Pool,
	log *slog.Logger,
) *Handler {
	repo := NewRepository(db)
	svc := NewService(repo, log)
	return &Handler{
		service: svc,
		log:     log,
	}
}
