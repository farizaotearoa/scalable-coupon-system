package coupon

type Coupons struct {
	Name   string
	Amount int
}

type ClaimHistory struct {
	UserID     string
	CouponName string
}

type Details struct {
	Name            string
	Amount          int
	RemainingAmount int
	ClaimedBy       []string
}
