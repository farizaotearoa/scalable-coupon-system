-- Create coupons table
CREATE TABLE IF NOT EXISTS coupons (
    name VARCHAR(255) PRIMARY KEY,
    amount INTEGER NOT NULL
);

-- Create claim_history table
CREATE TABLE IF NOT EXISTS claim_history (
    user_id VARCHAR(255) NOT NULL,
    coupon_name VARCHAR(255) NOT NULL,
    CONSTRAINT claim_history_unique UNIQUE (user_id, coupon_name)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_claim_history_coupon_name ON claim_history(coupon_name);
CREATE INDEX IF NOT EXISTS idx_claim_history_user_id ON claim_history(user_id);

