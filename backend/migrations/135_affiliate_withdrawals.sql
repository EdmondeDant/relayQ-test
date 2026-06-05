-- Affiliate withdrawal requests for salesperson cashout workflow.
CREATE TABLE IF NOT EXISTS user_affiliate_withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20,8) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ NULL,
    paid_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    remark TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_user_affiliate_withdrawals_amount_positive CHECK (amount > 0),
    CONSTRAINT chk_user_affiliate_withdrawals_status CHECK (status IN ('pending', 'paid'))
);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_user_created
    ON user_affiliate_withdrawals(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_status_created
    ON user_affiliate_withdrawals(status, created_at DESC);

COMMENT ON TABLE user_affiliate_withdrawals IS 'Affiliate cashout withdrawal requests submitted by users and marked paid by admins after offline payout.';
COMMENT ON COLUMN user_affiliate_withdrawals.status IS 'pending|paid';
COMMENT ON COLUMN user_affiliate_ledger.action IS 'accrue|transfer|withdraw_request|withdraw_paid';
