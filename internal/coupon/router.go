package coupon

import "net/http"

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/coupons", h.CreateCoupon)
	mux.HandleFunc("POST /api/coupons/claim", h.ClaimCoupon)
	mux.HandleFunc("GET /api/coupons/{name}", h.GetCouponDetails)

	return mux
}
