-- Phase 4B: Crypto Payment Integration

-- Master wallet untuk platform (hot wallet)
CREATE TABLE IF NOT EXISTS crypto_master_wallets (
    id UUID PRIMARY KEY,
    chain VARCHAR(20) NOT NULL, -- 'SOL', 'BTC', 'USDT_ERC20', 'USDT_TRC20', 'USDT_BEP20'
    public_key VARCHAR(255) NOT NULL,
    encrypted_private_key TEXT NOT NULL, -- AES-256 encrypted
    derivation_path VARCHAR(50),
    balance DECIMAL(24,8) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active', -- active, frozen, drained
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User deposit addresses (HD wallet derived)
CREATE TABLE IF NOT EXISTS crypto_deposit_addresses (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    chain VARCHAR(20) NOT NULL,
    address VARCHAR(255) NOT NULL,
    derivation_index INTEGER,
    master_wallet_id UUID REFERENCES crypto_master_wallets(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, chain)
);

-- Crypto transactions (deposit & withdrawal)
CREATE TABLE IF NOT EXISTS crypto_transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL, -- 'deposit', 'withdrawal'
    chain VARCHAR(20) NOT NULL,
    asset VARCHAR(10) NOT NULL, -- 'SOL', 'BTC', 'USDT'
    amount_crypto DECIMAL(24,8) NOT NULL,
    amount_idr DECIMAL(24,2) NOT NULL, -- equivalent saat transaksi
    exchange_rate DECIMAL(24,2) NOT NULL, -- 1 crypto = X IDR
    tx_hash VARCHAR(255),
    from_address VARCHAR(255),
    to_address VARCHAR(255) NOT NULL,
    confirmations INTEGER DEFAULT 0,
    required_confirmations INTEGER NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- pending, confirming, success, failed, cancelled
    fee_crypto DECIMAL(24,8),
    fee_idr DECIMAL(24,2),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Exchange rates (cache dari CoinGecko/CoinMarketCap)
CREATE TABLE IF NOT EXISTS crypto_exchange_rates (
    id UUID PRIMARY KEY,
    asset VARCHAR(10) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    rate DECIMAL(24,2) NOT NULL,
    source VARCHAR(50), -- 'coingecko', 'binance'
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(asset, currency)
);

-- Withdrawal whitelist (security)
CREATE TABLE IF NOT EXISTS crypto_withdrawal_whitelist (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    chain VARCHAR(20) NOT NULL,
    address VARCHAR(255) NOT NULL,
    label VARCHAR(100),
    is_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, chain, address)
);

-- Audit logs for crypto operations
CREATE TABLE IF NOT EXISTS crypto_audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    tx_id UUID,
    amount DECIMAL(24,8),
    chain VARCHAR(20),
    ip_address VARCHAR(45),
    user_agent TEXT,
    metadata JSONB,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_crypto_tx_user ON crypto_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_crypto_tx_status ON crypto_transactions(status);
CREATE INDEX IF NOT EXISTS idx_crypto_tx_hash ON crypto_transactions(tx_hash);
CREATE INDEX IF NOT EXISTS idx_crypto_deposit_address ON crypto_deposit_addresses(address);
