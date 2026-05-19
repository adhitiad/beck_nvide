-- =========================================================================
-- MIGRATION: 018_KYC_AND_SUBSCRIPTIONS
-- DESKRIPSI: Sistem KYC Ketat, Banned Users (LGBT Policy), Onboarding Progress, dan VIP AI Clip Subscription
-- =========================================================================

-- 1. TABEL: banned_users
CREATE TABLE IF NOT EXISTS banned_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    reason VARCHAR(100) NOT NULL, -- 'lgbt_policy', 'content_violation', 'harassment', 'other'
    banned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_permanent BOOLEAN NOT NULL DEFAULT TRUE,
    can_appeal BOOLEAN NOT NULL DEFAULT FALSE,
    device_fingerprint VARCHAR(255),
    ip_address VARCHAR(50)
);

-- Index untuk search cepat
CREATE INDEX IF NOT EXISTS idx_banned_users_user ON banned_users(user_id);
CREATE INDEX IF NOT EXISTS idx_banned_users_fingerprint ON banned_users(device_fingerprint);

-- 2. TABEL: kyc_verifications
CREATE TABLE IF NOT EXISTS kyc_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    id_card_number VARCHAR(100) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    gender VARCHAR(20) NOT NULL, -- e.g. 'male', 'female', 'non-binary', dll
    country VARCHAR(100) NOT NULL,
    document_url TEXT NOT NULL,
    selfie_url TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    rejection_reason TEXT,
    verified_at TIMESTAMPTZ,
    verified_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk search cepat
CREATE INDEX IF NOT EXISTS idx_kyc_user ON kyc_verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_kyc_status ON kyc_verifications(status);

-- 3. TABEL: agency_verifications
CREATE TABLE IF NOT EXISTS agency_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    company_name VARCHAR(255) NOT NULL,
    registration_number VARCHAR(100) NOT NULL, -- Akta Pendirian / SIUP / NIB
    tax_number VARCHAR(100) NOT NULL,          -- NPWP Badan
    phone_number VARCHAR(50) NOT NULL,
    office_address TEXT NOT NULL,
    document_url TEXT NOT NULL,                -- PDF of Akta/SIUP/NPWP
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    rejection_reason TEXT,
    verified_at TIMESTAMPTZ,
    verified_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk search cepat
CREATE INDEX IF NOT EXISTS idx_agency_ver_user ON agency_verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_agency_ver_status ON agency_verifications(status);

-- 4. TABEL: onboarding_progress
CREATE TABLE IF NOT EXISTS onboarding_progress (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role_type VARCHAR(20) NOT NULL,               -- 'user', 'host', 'agency'
    steps_completed JSONB NOT NULL DEFAULT '[]',   -- e.g. ["profile", "email_verified"]
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. TABEL: clip_subscription_plans
CREATE TABLE IF NOT EXISTS clip_subscription_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(50) NOT NULL UNIQUE,
    price BIGINT NOT NULL,           -- dalam IDR
    quota INT NOT NULL,              -- jumlah clip per bulan
    duration_days INT NOT NULL DEFAULT 30,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed data for plans
INSERT INTO clip_subscription_plans (id, name, price, quota, duration_days) VALUES
    ('018e0000-0000-0000-0000-000000000001', 'VIP1', 49999, 10, 30),
    ('018e0000-0000-0000-0000-000000000002', 'VIP2', 89999, 45, 30),
    ('018e0000-0000-0000-0000-000000000003', 'VIP3', 178999, 100, 30)
ON CONFLICT (name) DO NOTHING;

-- 6. TABEL: clip_subscriptions
CREATE TABLE IF NOT EXISTS clip_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES clip_subscription_plans(id) ON DELETE CASCADE,
    start_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    end_date TIMESTAMPTZ NOT NULL,
    quota_used INT NOT NULL DEFAULT 0,
    quota_total INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'expired'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clip_subs_user ON clip_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_clip_subs_status ON clip_subscriptions(user_id, status);

-- 7. TABEL: clip_generation_logs
CREATE TABLE IF NOT EXISTS clip_generation_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES clip_subscriptions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clip_logs_user ON clip_generation_logs(user_id);
