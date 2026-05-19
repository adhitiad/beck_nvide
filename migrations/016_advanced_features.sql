-- MIGRATION: 016_ADVANCED_FEATURES
-- DESKRIPSI: Migrasi database untuk 10 Fitur Baru Terintegrasi (Creator Tokens, Prediction Market, DRM, AI Recommendation, AI Clips, Dual Stream, Moderation, E2EE/High Privacy/Security, Paid Rooms, Toys Lovense, Request Show, AI Chat companion, Incognito, Private Profile, Disappearing Messages, Mute/Block)

-- ============================================
-- 1. FITUR 1: TOKENISASI KREATOR (CREATOR TOKENS)
-- ============================================
CREATE TABLE IF NOT EXISTS creator_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10) NOT NULL UNIQUE,
    total_supply BIGINT NOT NULL DEFAULT 0,
    max_supply BIGINT NOT NULL,
    base_price BIGINT NOT NULL DEFAULT 1000, -- harga dasar dalam IDR
    slope BIGINT NOT NULL DEFAULT 10,       -- slope untuk bonding curve
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_id UUID NOT NULL REFERENCES creator_tokens(id) ON DELETE CASCADE,
    balance BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, token_id)
);

-- ============================================
-- 2. FITUR 2: PASAR PREDIKSI (PREDICTION MARKET)
-- ============================================
CREATE TABLE IF NOT EXISTS predictions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'resolved', 'cancelled'
    resolved_outcome VARCHAR(10), -- 'yes', 'no'
    total_yes_pool BIGINT NOT NULL DEFAULT 0,
    total_no_pool BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS prediction_bets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    prediction_id UUID NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    outcome VARCHAR(10) NOT NULL, -- 'yes', 'no'
    amount BIGINT NOT NULL,       -- taruhan dalam IDR atau token
    currency_type VARCHAR(20) NOT NULL DEFAULT 'wallet', -- 'wallet', 'token'
    creator_token_id UUID REFERENCES creator_tokens(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 3. FITUR 3: MANAJEMEN HAK DIGITAL (DRM)
-- ============================================
CREATE TABLE IF NOT EXISTS vod_access_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    vod_id UUID NOT NULL REFERENCES vods(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 4. FITUR 4: PERSONALISASI BERBASIS AI
-- ============================================
CREATE TABLE IF NOT EXISTS user_interactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stream_id UUID REFERENCES streams(id) ON DELETE SET NULL,
    interaction_type VARCHAR(50) NOT NULL, -- 'watch', 'like', 'comment', 'gift'
    duration_seconds INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 5. FITUR 5: KLIP AI OTOMATIS
-- ============================================
CREATE TABLE IF NOT EXISTS stream_clips (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    clip_url TEXT NOT NULL,
    duration INT NOT NULL, -- dalam detik
    score FLOAT NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 6. FITUR 6: SIARAN FORMAT GANDA
-- ============================================
ALTER TABLE streams ADD COLUMN IF NOT EXISTS stream_key_landscape VARCHAR(255);
ALTER TABLE streams ADD COLUMN IF NOT EXISTS stream_key_portrait VARCHAR(255);
ALTER TABLE streams ADD COLUMN IF NOT EXISTS playback_id_landscape VARCHAR(255);
ALTER TABLE streams ADD COLUMN IF NOT EXISTS playback_id_portrait VARCHAR(255);

-- ============================================
-- 7. FITUR 7: SISTEM MODERASI & KEBIJAKAN KONTEN
-- ============================================
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reported_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    stream_id UUID REFERENCES streams(id) ON DELETE SET NULL,
    chat_message_id UUID,
    report_type VARCHAR(50) NOT NULL, -- 'explicit_content', 'violence', 'harassment', 'other'
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'reviewed', 'action_taken', 'ignored'
    action_taken VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS age_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS is_adult BOOLEAN DEFAULT TRUE;

-- ============================================
-- 8. FITUR 8: KEAMANAN & ENKRIPSI DATA
-- ============================================
CREATE TABLE IF NOT EXISTS user_public_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    nickname VARCHAR(100),
    bio TEXT,
    avatar_url TEXT,
    display_name_type VARCHAR(20) NOT NULL DEFAULT 'nickname', -- 'nickname', 'anonymous'
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 9. FITUR 9: MONETISASI TAMBAHAN
-- ============================================
-- Private Room Berbayar
CREATE TABLE IF NOT EXISTS paid_rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    entry_fee_idr BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Interactive Toys Integration (Lovense)
CREATE TABLE IF NOT EXISTS host_devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name VARCHAR(100) NOT NULL,
    device_id VARCHAR(100) NOT NULL,
    api_token VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(host_id, device_id)
);

-- Request Show
CREATE TABLE IF NOT EXISTS show_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    tips_amount BIGINT NOT NULL, -- dalam IDR
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'accepted', 'rejected', 'completed'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- AI Companion Chatbot
CREATE TABLE IF NOT EXISTS ai_chat_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    session_id UUID NOT NULL REFERENCES ai_chat_sessions(id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL, -- 'user', 'ai'
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- 10. FITUR 10: FITUR PRIVASI TINGGI
-- ============================================
ALTER TABLE messages ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_private_profile BOOLEAN DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS user_mutes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    muter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    muted_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(muter_id, muted_id)
);
