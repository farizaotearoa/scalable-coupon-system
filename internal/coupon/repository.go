package coupon

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func NewRepository(db *pgxpool.Pool, log *slog.Logger) *Repository {
	return &Repository{
		db:  db,
		log: log,
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
	r.log.Info("checking coupon existence", "coupon_name", couponName)
	defer r.log.Info("finished checking coupon existence", "coupon_name", couponName)

	var count int
	query := `
		SELECT COUNT(1)
		FROM coupons
		WHERE name = $1`

	err := r.db.QueryRow(ctx, query, couponName).Scan(&count)
	if err != nil {
		r.log.Error("failed to check coupon existence", "coupon_name", couponName, "error", err)
		return false, err
	}

	exists := count > 0
	r.log.Info("coupon existence checked", "coupon_name", couponName, "exists", exists)
	return exists, nil
}

func (r *Repository) InsertCoupon(
	ctx context.Context,
	coupon Coupons,
) error {
	r.log.Info("inserting coupon", "coupon_name", coupon.Name, "amount", coupon.Amount)
	defer r.log.Info("finished inserting coupon", "coupon_name", coupon.Name)

	query := `
		INSERT INTO coupons (name, amount) 
		VALUES ($1, $2)
	`
	_, err := r.db.Exec(ctx, query, coupon.Name, coupon.Amount)
	if err != nil {
		r.log.Error("failed to insert coupon", "coupon_name", coupon.Name, "error", err)
		return err
	}

	r.log.Info("coupon inserted successfully", "coupon_name", coupon.Name)
	return nil
}

func (r *Repository) ClaimCoupon(
	ctx context.Context,
	req ClaimCouponRequest,
) error {
	r.log.Info("starting coupon claim", "coupon_name", req.CouponName, "user_id", req.UserId)
	defer r.log.Info("finished coupon claim", "coupon_name", req.CouponName, "user_id", req.UserId)

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		r.log.Error("failed to begin transaction", "coupon_name", req.CouponName, "user_id", req.UserId, "error", err)
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
		r.log.Error("failed to check claim history", "coupon_name", req.CouponName, "user_id", req.UserId, "error", err)
		return err
	}
	if alreadyClaimed {
		r.log.Warn("coupon already claimed", "coupon_name", req.CouponName, "user_id", req.UserId)
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
			r.log.Warn("coupon not found", "coupon_name", req.CouponName)
			return ErrCouponNotFound
		}
		r.log.Error("failed to check stock", "coupon_name", req.CouponName, "error", err)
		return err
	}

	if amount-used <= 0 {
		r.log.Warn("coupon out of stock", "coupon_name", req.CouponName, "amount", amount, "used", used)
		return ErrCouponOutOfStock
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO claim_history (coupon_name, user_id)
		VALUES ($1, $2)
	`, req.CouponName, req.UserId)

	if err != nil {
		r.log.Error("failed to insert claim", "coupon_name", req.CouponName, "user_id", req.UserId, "error", err)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		r.log.Error("failed to commit transaction", "coupon_name", req.CouponName, "user_id", req.UserId, "error", err)
		return err
	}

	r.log.Info("coupon claimed successfully", "coupon_name", req.CouponName, "user_id", req.UserId)
	return nil
}

func (r *Repository) GetCouponDetails(
	ctx context.Context,
	couponName string,
) (*Details, error) {
	r.log.Info("getting coupon details", "coupon_name", couponName)
	defer r.log.Info("finished getting coupon details", "coupon_name", couponName)

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
		r.log.Error("failed to get coupon details", "coupon_name", couponName, "error", err)
		return nil, err
	}

	r.log.Info("coupon details retrieved", "coupon_name", couponName, "remaining", resp.RemainingAmount)
	return &resp, nil

}
