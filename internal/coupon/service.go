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

var ErrCouponAlreadyExists = errors.New("coupon already exists")
var ErrCouponNotFound = errors.New("coupon not found")
var ErrCouponAlreadyClaimed = errors.New("coupon already claimed")
var ErrCouponNoStock = errors.New("coupon no stock")

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
