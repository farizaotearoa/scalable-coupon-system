package coupon

type CreateCouponRequest struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

type ClaimCouponRequest struct {
	UserId     string `json:"user_id"`
	CouponName string `json:"coupon_name"`
}
