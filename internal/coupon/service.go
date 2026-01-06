package coupon

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
)

type Service struct {
	repo *Repository
	log  *slog.Logger
}

func NewService(repo *Repository,
	log *slog.Logger,
) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

func (s *Service) CreateCoupon(
	ctx context.Context,
	request CreateCouponRequest,
) error {
	couponExist, err := s.repo.CheckCouponExist(ctx, request.Name)
	if err != nil {
		return err
	}

	if couponExist {
		return ErrCouponAlreadyExists
	}

	coupon := Coupons{
		Name:   request.Name,
		Amount: request.Amount,
	}
	err = s.repo.InsertCoupon(ctx, coupon)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ClaimCoupon(
	ctx context.Context,
	req ClaimCouponRequest,
) error {
	err := s.repo.ClaimCoupon(ctx, req)
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrCouponNotFound):
		return ErrCouponNotFound
	case errors.Is(err, ErrCouponOutOfStock):
		return ErrCouponOutOfStock
	case errors.Is(err, ErrCouponAlreadyClaimed):
		return ErrCouponAlreadyClaimed
	default:
		return err
	}
}

func (s *Service) GetCouponDetails(
	ctx context.Context,
	couponName string,
) (GetCouponDetailsResponse, error) {
	var resp GetCouponDetailsResponse

	details, err := s.repo.GetCouponDetails(ctx, couponName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return resp, ErrCouponNotFound
		}
		return resp, err
	}

	resp.Name = details.Name
	resp.Amount = details.Amount
	resp.RemainingAmount = details.RemainingAmount
	resp.ClaimedBy = details.ClaimedBy

	return resp, nil
}
