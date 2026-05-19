-- 019_payout_methods_and_crypto_addresses.sql
-- Payment/Payout method management for host and agency withdrawals

CREATE TABLE IF NOT EXISTS payout_methods (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('bank_transfer', 'ewallet', 'crypto')),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,

    -- bank transfer
    bank_name TEXT,
    account_number TEXT,
    account_holder_name TEXT,

    -- e-wallet
    ewallet_provider VARCHAR(20) CHECK (ewallet_provider IN ('gopay', 'ovo', 'dana', 'shopeepay', 'linkaja')),
    ewallet_phone_number TEXT,

    -- verification controls
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    micro_deposit_required BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payout_methods_user_id ON payout_methods(user_id);
CREATE INDEX IF NOT EXISTS idx_payout_methods_user_type ON payout_methods(user_id, type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payout_methods_primary_per_user ON payout_methods(user_id) WHERE is_primary = TRUE;

CREATE TABLE IF NOT EXISTS crypto_payout_addresses (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    network VARCHAR(20) NOT NULL CHECK (network IN ('solana', 'bitcoin', 'bsc')),
    address TEXT NOT NULL, -- AES-256 encrypted in app layer
    label TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crypto_payout_addresses_user_id ON crypto_payout_addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_crypto_payout_addresses_user_network ON crypto_payout_addresses(user_id, network);

