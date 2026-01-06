package coupon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	wd, _ := os.Getwd()
	var envPath string

	for {
		envPath = filepath.Join(wd, ".env")
		if _, err := os.Stat(envPath); err == nil {
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			envPath = ""
			break
		}
		wd = parent
	}

	if envPath != "" {
		if err := godotenv.Load(envPath); err != nil {
			t.Logf("Warning: Could not load .env file from %s: %v", envPath, err)
		}
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/coupon_test?sslmode=disable"
		t.Logf("Using default TEST_DATABASE_URL: %s", dsn)
	} else {
		t.Logf("Using TEST_DATABASE_URL from .env")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	createTables(t, pool)

	return pool
}

func createTables(t *testing.T, db *pgxpool.Pool) {
	ctx := context.Background()

	_, err := db.Exec(ctx, `
		DROP TABLE IF EXISTS claim_history CASCADE;
		DROP TABLE IF EXISTS coupons CASCADE;
	`)
	if err != nil {
		t.Fatalf("Failed to drop tables: %v", err)
	}

	_, err = db.Exec(ctx, `
		CREATE TABLE coupons (
			name VARCHAR(255) PRIMARY KEY,
			amount INTEGER NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create coupons table: %v", err)
	}

	_, err = db.Exec(ctx, `
		CREATE TABLE claim_history (
			user_id VARCHAR(255) NOT NULL,
			coupon_name VARCHAR(255) NOT NULL,
			CONSTRAINT claim_history_unique UNIQUE (user_id, coupon_name)
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create claim_history table: %v", err)
	}
}

func cleanupTestDB(t *testing.T, db *pgxpool.Pool) {
	ctx := context.Background()
	_, err := db.Exec(ctx, `
		TRUNCATE TABLE claim_history CASCADE;
		TRUNCATE TABLE coupons CASCADE;
	`)
	if err != nil {
		t.Logf("Failed to cleanup test database: %v", err)
	}
}

func TestFlashSaleAttack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := NewRepository(db, logger)
	service := NewService(repo, logger)

	ctx := context.Background()

	couponName := "PROMO_SUPER"
	stockAmount := 5
	concurrentRequests := 50

	err := service.CreateCoupon(ctx, CreateCouponRequest{
		Name:   couponName,
		Amount: stockAmount,
	})
	if err != nil {
		t.Fatalf("Failed to create coupon: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failureCount := 0
	errors := make(map[string]int)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			err := service.ClaimCoupon(ctx, ClaimCouponRequest{
				UserId:     fmt.Sprintf("user_%d", userID),
				CouponName: couponName,
			})

			mu.Lock()
			if err != nil {
				failureCount++
				errorType := err.Error()
				errors[errorType]++
			} else {
				successCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	details, err := service.GetCouponDetails(ctx, couponName)
	if err != nil {
		t.Fatalf("Failed to get coupon details: %v", err)
	}

	if successCount != stockAmount {
		t.Errorf("Expected exactly %d successful claims, got %d", stockAmount, successCount)
	}

	if details.RemainingAmount != 0 {
		t.Errorf("Expected 0 remaining stock, got %d", details.RemainingAmount)
	}

	if failureCount != concurrentRequests-stockAmount {
		t.Errorf("Expected %d failures, got %d", concurrentRequests-stockAmount, failureCount)
	}

	t.Logf("Flash Sale Attack Results:")
	t.Logf("  Total Requests: %d", concurrentRequests)
	t.Logf("  Successful Claims: %d", successCount)
	t.Logf("  Failed Claims: %d", failureCount)
	t.Logf("  Remaining Stock: %d", details.RemainingAmount)
	t.Logf("  Error Breakdown: %v", errors)
}

func TestDoubleDipAttack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestDB(t, db)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := NewRepository(db, logger)
	service := NewService(repo, logger)

	ctx := context.Background()

	couponName := "PROMO_SUPER"
	userID := "user_12345"
	concurrentRequests := 10

	err := service.CreateCoupon(ctx, CreateCouponRequest{
		Name:   couponName,
		Amount: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create coupon: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failureCount := 0
	errors := make(map[string]int)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := service.ClaimCoupon(ctx, ClaimCouponRequest{
				UserId:     userID,
				CouponName: couponName,
			})

			mu.Lock()
			if err != nil {
				failureCount++
				errorType := err.Error()
				errors[errorType]++
			} else {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	details, err := service.GetCouponDetails(ctx, couponName)
	if err != nil {
		t.Fatalf("Failed to get coupon details: %v", err)
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful claim, got %d", successCount)
	}

	if failureCount != 9 {
		t.Errorf("Expected exactly 9 failures, got %d", failureCount)
	}

	expectedRemaining := 9
	if details.RemainingAmount != expectedRemaining {
		t.Errorf("Expected %d remaining stock, got %d", expectedRemaining, details.RemainingAmount)
	}

	claimedByCount := 0
	for _, claimedUser := range details.ClaimedBy {
		if claimedUser == userID {
			claimedByCount++
		}
	}

	if claimedByCount != 1 {
		t.Errorf("Expected user %s to appear exactly once in claimed_by, but found %d times", userID, claimedByCount)
	}

	t.Logf("Double Dip Attack Results:")
	t.Logf("  Total Requests: %d", concurrentRequests)
	t.Logf("  Successful Claims: %d", successCount)
	t.Logf("  Failed Claims: %d", failureCount)
	t.Logf("  Remaining Stock: %d", details.RemainingAmount)
	t.Logf("  Error Breakdown: %v", errors)
}
