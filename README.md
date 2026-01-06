# Scalable Coupon System

This is my submission for Technical Assessment: Scalable Coupon System

## Prerequisites

- **Docker Desktop** (version 20.10 or later)
- **Docker Compose** (included with Docker Desktop, version 2.0+)

## How to Run

1. Clone the repository:
   ```bash
   git clone https://github.com/farizaotearoa/scalable-coupon-system.git
   cd scalable-coupon-system
   ```

2. Create a `.env` file (optional, defaults are provided):
   ```bash
   cp .env.example .env
   ```

3. Start the application:
   ```bash
   docker compose up --build
   ```

   This command will:
   - Build the Go application Docker image
   - Start PostgreSQL database container
   - Run database migrations automatically
   - Start the application server on port 8080

4. The application will be available at `http://localhost:8080`

**Note**: For newer versions of Docker Desktop, use `docker compose` (with a space) instead of `docker-compose` (with a hyphen).

## How to Test

### Running Unit Tests

The project includes unit tests that has 2 scenarios:

1. **Flash Sale Attack**: 50 concurrent requests for a coupon with only 5 items in stock
   - Expected: Exactly 5 successful claims, 0 remaining stock

2. **Double Dip Attack**: 10 concurrent requests from the same user for the same coupon
   - Expected: Exactly 1 success, 9 failures (409 Conflict)

```bash
# Run all tests
go test ./internal/coupon/... -v

# Run specific test scenarios
go test ./internal/coupon/... -v -run TestFlashSaleAttack
go test ./internal/coupon/... -v -run TestDoubleDipAttack
```

## Architecture Notes

### Database Design

#### `coupons` Table
- `name` (VARCHAR(255), PRIMARY KEY): Unique coupon identifier
- `amount` (INTEGER): Total stock available when creating coupons

#### `claim_history` Table
- `user_id` (VARCHAR(255)): User identifier
- `coupon_name` (VARCHAR(255)): Coupon identifier
- **Unique Constraint**: `(user_id, coupon_name)` - Prevents duplicate claims per user

#### Indexes
- `idx_claim_history_coupon_name`: Optimizes queries filtering by coupon name
- `idx_claim_history_user_id`: Optimizes queries filtering by user ID

### Locking Strategy

#### 1. Transaction-Based Atomicity
All claim operations are wrapped in a database transaction. The transaction ensures that checking stock, inserting claim, and committing happen atomically. If any step fails, the entire transaction is rolled back

#### 2. FOR UPDATE Lock


The `FOR UPDATE` clause locks the coupon row for the duration of the transaction. This prevents concurrent transactions from reading stale stock values. Other transactions must wait until the lock is released (on commit or rollback)

#### 3. Eligibility Check First
Before checking stock, the system first verifies if the user has already claimed the coupon. This prevents unnecessary stock checks and provides faster rejection for duplicate claims

#### 4. Stock Calculation
Stock availability is calculated as: `amount - COUNT(claim_history entries)`. The count is calculated in real-time, not stored, ensuring consistency

## Environment Variables

Key environment variables (see `.env.example` for defaults):

- `DB_USERNAME`: Database username (default: postgres)
- `DB_PASSWORD`: Database password (default: postgres)
- `DB_HOST`: Database host (default: db for Docker)
- `DB_PORT`: Database port (default: 5432)
- `DB_NAME`: Database name (default: coupon_db)
- `APP_PORT`: Application port (default: :8080)
- `LOG_PATH`: Log file path (default: ./logs/app.log)
- `TEST_DATABASE_URL`: Test database connection string

## Project Structure

```
scalable-coupon-system/
├── cmd/
│   └── server/
│       ├── main.go          # Application entry point
│       └── router.go        # HTTP router setup
├── internal/
│   ├── coupon/
│   │   ├── handler.go        # HTTP handlers
│   │   ├── service.go        # Business logic
│   │   ├── repository.go     # Database operations
│   │   ├── model.go          # Data models
│   │   ├── request.go        # Request DTOs
│   │   ├── response.go       # Response DTOs
│   │   └── router.go         # Route definitions
│   └── shared/
│       ├── config.go         # Configuration
│       ├── database.go       # Database connection
│       └── logger.go         # Logger setup
├── migration/
│   └── 001_init.sql         # Database schema
├── docker-compose.yml       # Docker Compose configuration
├── Dockerfile              # Application Docker image
└── README.md              # This file
```
