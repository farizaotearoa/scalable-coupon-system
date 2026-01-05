package coupon

import (
	"context"

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
