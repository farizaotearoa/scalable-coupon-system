package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"scalable-coupon-system/internal/coupon"
	"scalable-coupon-system/internal/shared"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Error("failed to load environment variables", "err", err)
	}

	cfg := shared.NewConfig()

	log, closeLog, err := shared.NewLogger(*cfg)
	if err != nil {
		panic(err)
	}
	defer closeLog()

	db, err := shared.NewDatabase(cfg)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return
	}
	defer db.Close()

	couponHandler := coupon.NewHandler(db, log)
	router := NewRouter(couponHandler)

	srv := &http.Server{
		Addr:    cfg.AppPort,
		Handler: router,
	}

	go func() {
		log.Info("http server started", "addr", srv.Addr)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
		}
	}()

	waitForShutdown(srv, log)
}

func waitForShutdown(srv *http.Server, log *slog.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown failed", "err", err)
	} else {
		log.Info("server stopped gracefully")
	}
}
