package main

import (
	"net/http"
	"scalable-coupon-system/internal/coupon"
)

func NewRouter(handler *coupon.Handler) http.Handler {
	root := http.NewServeMux()
	root.Handle("/", handler.Routes())

	return root
}
