package coupon

import "net/http"

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	return mux
}
