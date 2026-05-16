-- Phase 4: Monetization & Agency System

-- Host applications
CREATE TABLE IF NOT EXISTS host_applications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bio TEXT NOT NULL,
    id_card_url VARCHAR(500) NOT NULL,
    bank_account_name VARCHAR(255) NOT NULL,
    bank_account_number VARCHAR(50) NOT NULL,
    bank_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Agencies
CREATE TABLE IF NOT EXISTS agencies (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    logo_url VARCHAR(500),
    commission_rate INT NOT NULL DEFAULT 20, -- percentage 10-30
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, suspended
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agencies_owner ON agencies(owner_id);

-- Agency-Host relationships
CREATE TABLE IF NOT EXISTS agency_hosts (
    agency_id UUID NOT NULL REFERENCES agencies(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, invited, removed
    revenue_share INT NOT NULL DEFAULT 60, -- host share percentage
    total_earnings BIGINT NOT NULL DEFAULT 0, -- in IDR
    PRIMARY KEY (agency_id, host_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agency_hosts_host ON agency_hosts(host_id) WHERE status = 'active';

-- Wallets
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance BIGINT NOT NULL DEFAULT 0, -- in IDR
    frozen_balance BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT wallet_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT wallet_frozen_non_negative CHECK (frozen_balance >= 0)
);

-- Transactions (immutable ledger)
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- deposit, withdrawal, gift_sent, gift_received, agency_commission, host_earning, platform_fee
    amount BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, success, failed, cancelled, refunded
    reference_id VARCHAR(255), -- idempotency / external reference
    payment_method VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions(reference_id);

-- Gift catalog
CREATE TABLE IF NOT EXISTS gifts (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    icon_url VARCHAR(500) NOT NULL,
    price BIGINT NOT NULL, -- in IDR
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    animation_url VARCHAR(500),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Gift transactions
CREATE TABLE IF NOT EXISTS gift_transactions (
    id UUID PRIMARY KEY,
    stream_id UUID REFERENCES streams(id),
    sender_id UUID NOT NULL REFERENCES users(id),
    receiver_id UUID NOT NULL REFERENCES users(id),
    gift_id UUID NOT NULL REFERENCES gifts(id),
    quantity INT NOT NULL DEFAULT 1,
    total_price BIGINT NOT NULL,
    agency_id UUID REFERENCES agencies(id),
    agency_commission BIGINT NOT NULL DEFAULT 0,
    host_earning BIGINT NOT NULL DEFAULT 0,
    platform_fee BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_gift_tx_stream ON gift_transactions(stream_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gift_tx_sender ON gift_transactions(sender_id, created_at DESC);

-- Duitku payments
CREATE TABLE IF NOT EXISTS duitku_payments (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    merchant_order_id VARCHAR(255) NOT NULL UNIQUE,
    duitku_reference VARCHAR(255),
    payment_url TEXT,
    va_number VARCHAR(50),
    payment_method VARCHAR(50),
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, success, failed, expired
    amount BIGINT NOT NULL,
    expiry_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_duitku_merchant ON duitku_payments(merchant_order_id);
