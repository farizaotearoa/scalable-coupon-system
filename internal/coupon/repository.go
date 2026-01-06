package coupon

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

var (
	ErrCouponAlreadyExists  = errors.New("coupon already exists")
	ErrCouponNotFound       = errors.New("coupon not found")
	ErrCouponAlreadyClaimed = errors.New("coupon already claimed")
	ErrCouponOutOfStock     = errors.New("coupon out of stock")
)

func (r *Repository) CheckCouponExist(
	ctx context.Context,
	couponName string,
) (bool, error) {
	var count int
	query := `
		SELECT COUNT(1)
		FROM coupons
		WHERE name = $1`

	err := r.db.QueryRow(ctx, query, couponName).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repository) InsertCoupon(
	ctx context.Context,
	coupon Coupons,
) error {
	query := `
		INSERT INTO coupons (name, amount) 
		VALUES ($1, $2)
	`
	_, err := r.db.Exec(ctx, query, coupon.Name, coupon.Amount)
	return err
}

func (r *Repository) ClaimCoupon(
	ctx context.Context,
	req ClaimCouponRequest,
) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var alreadyClaimed bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 
			FROM claim_history 
			WHERE coupon_name = $1 AND user_id = $2
		)
	`, req.CouponName, req.UserId).Scan(&alreadyClaimed)
	if err != nil {
		return err
	}
	if alreadyClaimed {
		return ErrCouponAlreadyClaimed
	}

	var amount int
	var used int
	err = tx.QueryRow(ctx, `
		SELECT
			c.amount,
			(SELECT COUNT(*) FROM claim_history ch WHERE ch.coupon_name = c.name)
		FROM coupons c
		WHERE c.name = $1
		FOR UPDATE
	`, req.CouponName).Scan(&amount, &used)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCouponNotFound
		}
		return err
	}

	if amount-used <= 0 {
		return ErrCouponOutOfStock
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO claim_history (coupon_name, user_id)
		VALUES ($1, $2)
	`, req.CouponName, req.UserId)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetCouponDetails(
	ctx context.Context,
	couponName string,
) (*Details, error) {
	query := `
		SELECT
			c.name,
			c.amount,
			c.amount - COUNT(ch.user_id) AS remaining_amount,
			COALESCE(
				ARRAY_AGG(ch.user_id) FILTER (WHERE ch.user_id IS NOT NULL),
				'{}'::text[]
			) AS claimed_by
		FROM coupons c
		LEFT JOIN claim_history ch
			ON c.name = ch.coupon_name
		WHERE c.name = $1
		GROUP BY c.name, c.amount`

	var resp Details

	err := r.db.QueryRow(ctx, query, couponName).Scan(
		&resp.Name,
		&resp.Amount,
		&resp.RemainingAmount,
		&resp.ClaimedBy,
	)

	if err != nil {
		return nil, err
	}

	return &resp, nil

}
