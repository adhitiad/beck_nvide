-- ============================================
-- FASE 1: FOUNDATION & AUTHENTICATION
-- Live Streaming Platform - Database Schema
-- ============================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- TABLE: roles
-- ============================================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- TABLE: permissions
-- ============================================
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource VARCHAR(100) NOT NULL,      -- e.g., 'stream', 'vod', 'story', 'comment', 'user'
    action VARCHAR(50) NOT NULL,         -- e.g., 'create', 'read', 'update', 'delete', 'manage'
    name VARCHAR(200) UNIQUE NOT NULL,   -- e.g., 'stream:create', 'user:delete'
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- TABLE: role_permissions (junction)
-- ============================================
CREATE TABLE role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);

-- ============================================
-- TABLE: users
-- ============================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id),
    avatar_url TEXT,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verification_token VARCHAR(255),
    reset_token VARCHAR(255),
    reset_token_expires_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index untuk performa
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role_id ON users(role_id);

-- ============================================
-- TABLE: refresh_tokens
-- ============================================
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

-- Index untuk lookup cepat
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id, revoked_at, expires_at)
WHERE revoked_at IS NULL AND expires_at > NOW();

-- ============================================
-- TRIGGER: Update updated_at timestamp
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- VIEW: user_with_role (untuk query yang efisien)
-- ============================================
CREATE VIEW user_with_role AS
SELECT
    u.id,
    u.username,
    u.email,
    u.role_id,
    r.name as role_name,
    u.avatar_url,
    u.is_verified,
    u.last_login_at,
    u.created_at,
    u.updated_at
FROM users u
JOIN roles r ON u.role_id = r.id;
-- ============================================
-- FASE 1: SEED DATA - RBAC & INITIAL USERS
-- ============================================

-- ============================================
-- INSERT ROLES (5 roles)
-- ============================================
INSERT INTO roles (id, name, description) VALUES
    (gen_random_uuid(), 'guest', 'Tamu - akses terbatas, hanya bisa melihat konten publik'),
    (gen_random_uuid(), 'user', 'Pengguna biasa - bisa mengakses fitur sosial dasar'),
    (gen_random_uuid(), 'host', 'Host - bisa live streaming dan mengelola konten'),
    (gen_random_uuid(), 'agency', 'Agency - mengelola multiple host, revenue sharing'),
    (gen_random_uuid(), 'admin', 'Admin - akses penuh ke semua fitur')
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- INSERT PERMISSIONS (comprehensive permission matrix)
-- ============================================

-- User permissions
INSERT INTO permissions (id, resource, action, name, description) VALUES
    -- User: Self
    (gen_random_uuid(), 'user', 'create', 'user:create', 'Daftar akun baru'),
    (gen_random_uuid(), 'user', 'read', 'user:read:own', 'Baca profil sendiri'),
    (gen_random_uuid(), 'user', 'update', 'user:update:own', 'Update profil sendiri'),
    (gen_random_uuid(), 'user', 'delete', 'user:delete:own', 'Hapus akun sendiri'),
    -- User: Others (requires higher role)
    (gen_random_uuid(), 'user', 'read', 'user:read:any', 'Baca profil pengguna lain'),
    (gen_random_uuid(), 'user', 'update', 'user:update:any', 'Update profil pengguna lain'),
    (gen_random_uuid(), 'user', 'delete', 'user:delete:any', 'Hapus akun pengguna lain'),
    (gen_random_uuid(), 'user', 'ban', 'user:ban', 'Ban/toggle status user'),

    -- Authentication
    (gen_random_uuid(), 'auth', 'login', 'auth:login', 'Login ke sistem'),
    (gen_random_uuid(), 'auth', 'logout', 'auth:logout', 'Logout dari sistem'),
    (gen_random_uuid(), 'auth', 'refresh', 'auth:refresh', 'Refresh token'),
    (gen_random_uuid(), 'auth', 'verify', 'auth:verify', 'Verifikasi email/akun'),

    -- Stream / Live
    (gen_random_uuid(), 'stream', 'create', 'stream:create', 'Buat live streaming'),
    (gen_random_uuid(), 'stream', 'read', 'stream:read', 'Lihat live stream'),
    (gen_random_uuid(), 'stream', 'update', 'stream:update', 'Update setting stream'),
    (gen_random_uuid(), 'stream', 'delete', 'stream:delete', 'Hapus live stream'),
    (gen_random_uuid(), 'stream', 'manage', 'stream:manage', 'Kelola semua stream (moderator)'),

    -- VOD (Video On Demand)
    (gen_random_uuid(), 'vod', 'upload', 'vod:upload', 'Upload VOD'),
    (gen_random_uuid(), 'vod', 'read', 'vod:read', 'Lihat VOD'),
    (gen_random_uuid(), 'vod', 'update', 'vod:update', 'Update VOD'),
    (gen_random_uuid(), 'vod', 'delete', 'vod:delete', 'Hapus VOD'),
    (gen_random_uuid(), 'vod', 'moderate', 'vod:moderate', 'Moderasi VOD'),

    -- Story
    (gen_random_uuid(), 'story', 'create', 'story:create', 'Buat story (24h)'),
    (gen_random_uuid(), 'story', 'read', 'story:read', 'Baca story'),
    (gen_random_uuid(), 'story', 'delete', 'story:delete', 'Hapus story sendiri'),
    (gen_random_uuid(), 'story', 'moderate', 'story:moderate', 'Moderasi story'),

    -- Comment
    (gen_random_uuid(), 'comment', 'create', 'comment:create', 'Buat komentar'),
    (gen_random_uuid(), 'comment', 'read', 'comment:read', 'Baca komentar'),
    (gen_random_uuid(), 'comment', 'update', 'comment:update:own', 'Update komentar sendiri'),
    (gen_random_uuid(), 'comment', 'delete', 'comment:delete:own', 'Hapus komentar sendiri'),
    (gen_random_uuid(), 'comment', 'moderate', 'comment:moderate', 'Moderasi komentar (hapus semua)'),

    -- Like
    (gen_random_uuid(), 'like', 'create', 'like:create', 'Berlike'),
    (gen_random_uuid(), 'like', 'delete', 'like:delete', 'Unlike'),

    -- Gift (monetization)
    (gen_random_uuid(), 'gift', 'send', 'gift:send', 'Kirim gift'),
    (gen_random_uuid(), 'gift', 'read', 'gift:read', 'Lihat history gift'),
    (gen_random_uuid(), 'gift', 'manage', 'gift:manage', 'Kelola gift (admin)'),

    -- Wallet
    (gen_random_uuid(), 'wallet', 'read', 'wallet:read:own', 'Baca balance wallet sendiri'),
    (gen_random_uuid(), 'wallet', 'update', 'wallet:update', 'Update balance (admin/system)'),
    (gen_random_uuid(), 'wallet', 'transaction', 'wallet:transaction', 'Transaksi wallet'),

    -- Agency
    (gen_random_uuid(), 'agency', 'create', 'agency:create', 'Buat agency'),
    (gen_random_uuid(), 'agency', 'read', 'agency:read', 'Baca data agency'),
    (gen_random_uuid(), 'agency', 'update', 'agency:update', 'Update agency'),
    (gen_random_uuid(), 'agency', 'manage', 'agency:manage', 'Kelola agency (admin)'),
    (gen_random_uuid(), 'agency', 'host:manage', 'agency:host:manage', 'Kelola host di agency'),

    -- Host Application
    (gen_random_uuid(), 'host_application', 'create', 'host_application:create', 'Apply jadi host'),
    (gen_random_uuid(), 'host_application', 'read', 'host_application:read', 'Baca aplikasi host'),
    (gen_random_uuid(), 'host_application', 'approve', 'host_application:approve', 'Approve aplikasi host'),
    (gen_random_uuid(), 'host_application', 'reject', 'host_application:reject', 'Tolak aplikasi host'),

    -- Chat / Message
    (gen_random_uuid(), 'chat', 'send', 'chat:send', 'Kirim pesan'),
    (gen_random_uuid(), 'chat', 'read', 'chat:read', 'Baca pesan'),
    (gen_random_uuid(), 'chat', 'moderate', 'chat:moderate', 'Moderasi chat'),

    -- Payment
    (gen_random_uuid(), 'payment', 'create', 'payment:create', 'Buat payment (top-up)'),
    (gen_random_uuid(), 'payment', 'read', 'payment:read', 'Baca history payment'),
    (gen_random_uuid(), 'payment', 'callback', 'payment:callback', 'Payment callback (Duitku)'),
    (gen_random_uuid(), 'payment', 'refund', 'payment:refund', 'Refund payment'),

    -- System
    (gen_random_uuid(), 'system', 'config', 'system:config', 'Ubah system config'),
    (gen_random_uuid(), 'system', 'monitor', 'system:monitor', 'Akses monitoring/logs'),
    (gen_random_uuid(), 'system', 'backup', 'system:backup', 'Backup/restore database')
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- ASSIGN PERMISSIONS TO ROLES
-- ============================================

-- GUEST: Only read access to public content
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'guest'
AND p.name IN (
    'user:read:any',
    'stream:read',
    'vod:read',
    'story:read',
    'comment:read',
    'like:create',
    'auth:login',
    'auth:refresh'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- USER: Basic social features + self management
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'user'
AND p.name IN (
    -- Auth
    'auth:login',
    'auth:logout',
    'auth:refresh',
    'auth:verify',
    -- User self
    'user:create',
    'user:read:own',
    'user:update:own',
    'user:delete:own',
    'user:read:any',
    -- Stream (viewer only)
    'stream:read',
    -- VOD
    'vod:read',
    'vod:upload',
    'vod:update',
    'vod:delete',
    -- Story
    'story:create',
    'story:read',
    'story:delete',
    -- Comment
    'comment:create',
    'comment:read',
    'comment:update:own',
    'comment:delete:own',
    -- Like
    'like:create',
    'like:delete',
    -- Gift
    'gift:send',
    'gift:read',
    -- Wallet
    'wallet:read:own',
    'wallet:transaction',
    -- Chat
    'chat:send',
    'chat:read',
    -- Payment
    'payment:create',
    'payment:read'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- HOST: All user features + streaming + monetization
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'host'
AND p.name IN (
    -- All user permissions
    'auth:login', 'auth:logout', 'auth:refresh', 'auth:verify',
    'user:read:own', 'user:update:own', 'user:delete:own', 'user:read:any',
    'stream:read',
    'vod:read', 'vod:upload', 'vod:update', 'vod:delete',
    'story:create', 'story:read', 'story:delete',
    'comment:create', 'comment:read', 'comment:update:own', 'comment:delete:own',
    'like:create', 'like:delete',
    'gift:send', 'gift:read',
    'wallet:read:own', 'wallet:transaction',
    'chat:send', 'chat:read',
    'payment:create', 'payment:read',
    -- Host-specific
    'stream:create',
    'stream:update',
    'stream:delete',
    'host_application:create'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- AGENCY: Manage hosts + revenue sharing
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'agency'
AND p.name IN (
    -- All host permissions
    'auth:login', 'auth:logout', 'auth:refresh', 'auth:verify',
    'user:read:own', 'user:update:own', 'user:delete:own', 'user:read:any',
    'stream:create', 'stream:read', 'stream:update', 'stream:delete',
    'vod:read', 'vod:upload', 'vod:update', 'vod:delete',
    'story:create', 'story:read', 'story:delete',
    'comment:create', 'comment:read', 'comment:update:own', 'comment:delete:own',
    'like:create', 'like:delete',
    'gift:send', 'gift:read',
    'wallet:read:own', 'wallet:transaction',
    'chat:send', 'chat:read',
    'payment:create', 'payment:read',
    'host_application:create',
    -- Agency-specific
    'agency:read',
    'agency:update',
    'agency:host:manage',
    'user:read:any',
    'stream:manage'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ADMIN: Full system access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================
-- CREATE DEFAULT ADMIN USER (for testing)
-- ============================================
DO $$
DECLARE
    admin_role_id UUID;
    admin_user_id UUID;
BEGIN
    SELECT id INTO admin_role_id FROM roles WHERE name = 'admin';
    admin_user_id := gen_random_uuid();

    INSERT INTO users (id, username, email, password_hash, role_id, is_verified, last_login_at)
    VALUES (
        admin_user_id,
        'admin',
        'admin@nvide.live',
        '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj8xm5/Y5KQq', -- bcrypt hash of "Admin123!"
        admin_role_id,
        TRUE,
        NOW()
    ) ON CONFLICT (email) DO NOTHING;

    RAISE NOTICE 'Admin user created with email: admin@nvide.live, password: Admin123!';
END $$;

-- ============================================
-- CREATE TEST USER (for development)
-- ============================================
DO $$
DECLARE
    user_role_id UUID;
    test_user_id UUID;
BEGIN
    SELECT id INTO user_role_id FROM roles WHERE name = 'user';
    test_user_id := gen_random_uuid();

    INSERT INTO users (id, username, email, password_hash, role_id, is_verified)
    VALUES (
        test_user_id,
        'testuser',
        'test@nvide.live',
        '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj8xm5/Y5KQq', -- "Test123!"
        user_role_id,
        TRUE
    ) ON CONFLICT (email) DO NOTHING;

    RAISE NOTICE 'Test user created with email: test@nvide.live, password: Test123!';
END $$;

-- ============================================
-- VERIFICATION: Show inserted data
-- ============================================
SELECT 'Roles:' as type, name, description FROM roles ORDER BY name;
SELECT 'Permissions count:' as type, COUNT(*)::text as count FROM permissions;
SELECT 'Role-Permissions count:' as type, COUNT(*)::text as count FROM role_permissions;
SELECT 'Users count:' as type, COUNT(*)::text as count FROM users;
-- ============================================
-- FASE 2: SOCIAL FEATURES
-- ============================================

-- ============================================
-- TABLE: stories
-- ============================================
CREATE TABLE stories (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    media_type VARCHAR(50) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stories_user_id ON stories(user_id);
CREATE INDEX idx_stories_expires_at ON stories(expires_at);

-- ============================================
-- TABLE: story_views
-- ============================================
CREATE TABLE story_views (
    id UUID PRIMARY KEY,
    story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(story_id, user_id)
);

-- ============================================
-- TABLE: comments
-- ============================================
CREATE TABLE comments (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_id UUID NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    like_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_comments_content ON comments(content_id, content_type);
CREATE INDEX idx_comments_parent ON comments(parent_id);

-- ============================================
-- TABLE: comment_likes
-- ============================================
CREATE TABLE comment_likes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, comment_id)
);

-- ============================================
-- TABLE: likes
-- ============================================
CREATE TABLE likes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_id UUID NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, content_id, content_type)
);
CREATE INDEX idx_likes_content ON likes(content_id, content_type);

-- ============================================
-- TABLE: chat_rooms
-- ============================================
CREATE TABLE chat_rooms (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    target_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_chat_rooms_target ON chat_rooms(target_id);

-- ============================================
-- TABLE: room_participants
-- ============================================
CREATE TABLE room_participants (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(room_id, user_id)
);

-- ============================================
-- TABLE: messages
-- ============================================
CREATE TABLE messages (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_room ON messages(room_id);
-- Phase 3: Streaming Infrastructure Migrations

CREATE TABLE IF NOT EXISTS streams (
    id UUID PRIMARY KEY,
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    thumbnail_url VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'preparing', -- preparing, live, ended, archived
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    viewer_peak INT DEFAULT 0,
    total_duration INT DEFAULT 0,
    room_id UUID NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Unique index for host_id and status to enforce 1 live stream per host
CREATE UNIQUE INDEX idx_streams_host_live ON streams (host_id) WHERE status = 'live';

CREATE TABLE IF NOT EXISTS stream_sessions (
    id UUID PRIMARY KEY,
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    viewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    duration INT DEFAULT 0,
    ip_address VARCHAR(45)
);

CREATE TABLE IF NOT EXISTS vod_media (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    original_url VARCHAR(255) NOT NULL,
    hls_url VARCHAR(255),
    thumbnail_url VARCHAR(255),
    duration INT DEFAULT 0,
    file_size BIGINT DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'processing', -- processing, ready, failed
    visibility VARCHAR(50) NOT NULL DEFAULT 'public', -- public, followers, private
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS stream_signaling (
    id UUID PRIMARY KEY,
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    peer_id UUID NOT NULL,
    signal_type VARCHAR(50) NOT NULL, -- offer, answer, ice_candidate
    data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
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
-- Phase 5: Scaling & Optimization

-- Composite Indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_streams_status_started ON streams(status, started_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_gift_tx_stream_created ON gift_transactions(stream_id, created_at DESC);

-- Partitioning Example (Since we cannot easily convert an existing table to a partitioned table in postgres without recreating it, 
-- we will just note this as part of the schema design for future fresh deployments, or create a partitioned history table)

-- Create a partitioned table for transaction history to archive old transactions
CREATE TABLE IF NOT EXISTS transaction_history (
    id UUID,
    user_id UUID,
    type VARCHAR(50),
    amount BIGINT,
    currency VARCHAR(10),
    status VARCHAR(20),
    reference_id VARCHAR(255),
    payment_method VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE
) PARTITION BY RANGE (created_at);

-- Create partitions for the next few months
CREATE TABLE IF NOT EXISTS transaction_history_y2026m05 PARTITION OF transaction_history FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS transaction_history_y2026m06 PARTITION OF transaction_history FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Add indexes to the partitioned table
CREATE INDEX IF NOT EXISTS idx_tx_history_user_created ON transaction_history(user_id, created_at DESC);

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
-- ============================================
-- FASE 3: PRIVATE CHAT (DM) & REFACTORING
-- ============================================

-- 1. Refactor existing tables to avoid name collision
-- Rename old 'messages' (from Fase 2) to 'live_messages'
ALTER TABLE messages RENAME TO live_messages;
ALTER INDEX idx_messages_room RENAME TO idx_live_messages_room;

-- Rename old 'chat_rooms' to 'live_rooms'
ALTER TABLE chat_rooms RENAME TO live_rooms;
ALTER INDEX idx_chat_rooms_target RENAME TO idx_live_rooms_target;

-- Rename old 'room_participants' to 'live_room_participants'
ALTER TABLE room_participants RENAME TO live_room_participants;

-- 2. Create New Private Chat Tables
-- Conversations (1-on-1 chat room)
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL DEFAULT 'direct', -- 'direct', 'group'
    initiator_id UUID NOT NULL REFERENCES users(id),
    recipient_id UUID NOT NULL REFERENCES users(id),
    last_message_id UUID, -- Will be set after messages table exists
    last_message_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(initiator_id, recipient_id) -- Prevent duplicate DM
);

-- Conversation metadata per user (mute, archive, pin)
CREATE TABLE conversation_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    unread_count INTEGER DEFAULT 0,
    is_muted BOOLEAN DEFAULT false,
    is_archived BOOLEAN DEFAULT false,
    is_pinned BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false, -- soft delete for user
    last_read_message_id UUID,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(conversation_id, user_id)
);

-- Messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL DEFAULT 'text', -- 'text', 'image', 'voice', 'gift', 'system'
    content TEXT, -- text content or caption
    metadata JSONB DEFAULT '{}', -- {voice_duration: 15, gift_id: "...", image_width: 1920}
    reply_to_message_id UUID REFERENCES messages(id),
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false, -- soft delete
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Message status tracking (sent, delivered, read)
CREATE TABLE message_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL, -- 'delivered', 'read'
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id, status)
);

-- Attachments
CREATE TABLE message_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_name VARCHAR(255),
    file_url TEXT NOT NULL,
    file_type VARCHAR(50), -- 'image/jpeg', 'audio/mp4'
    file_size INTEGER, -- bytes
    width INTEGER,
    height INTEGER,
    duration INTEGER, -- untuk voice note (detik)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Block list (privacy)
CREATE TABLE user_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(blocker_id, blocked_id)
);

-- Add foreign key for last_message_id to conversations
ALTER TABLE conversations ADD CONSTRAINT fk_last_message FOREIGN KEY (last_message_id) REFERENCES messages(id) ON DELETE SET NULL;

-- Indexes for performance
CREATE INDEX idx_conversations_initiator ON conversations(initiator_id);
CREATE INDEX idx_conversations_recipient ON conversations(recipient_id);
CREATE INDEX idx_conversations_last_message_at ON conversations(last_message_at DESC);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

CREATE INDEX idx_conversation_participants_user_id ON conversation_participants(user_id);
CREATE INDEX idx_conversation_participants_conv_user ON conversation_participants(conversation_id, user_id);

CREATE INDEX idx_user_blocks_blocker ON user_blocks(blocker_id);
CREATE INDEX idx_user_blocks_blocked ON user_blocks(blocked_id);

-- Add preference to user table
ALTER TABLE users ADD COLUMN IF NOT EXISTS read_receipts_enabled BOOLEAN DEFAULT true;

-- Trigger to update updated_at
CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_conversation_participants_updated_at
    BEFORE UPDATE ON conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- ============================================
-- FASE 4B: PAID INTERACTIONS (PAY-TO-CHAT & PAY-PER-CALL)
-- ============================================

-- Host call rate configuration
CREATE TABLE host_call_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    voice_call_rate_idr DECIMAL(12,2) NOT NULL DEFAULT 1000, -- per minute
    video_call_rate_idr DECIMAL(12,2) NOT NULL DEFAULT 2000, -- per minute
    min_duration_seconds INTEGER DEFAULT 60,
    is_enabled BOOLEAN DEFAULT false,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(host_id)
);

-- Paid chat unlock (one-time payment per conversation)
CREATE TABLE paid_chat_unlocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    payer_id UUID NOT NULL REFERENCES users(id), -- user who pays
    recipient_id UUID NOT NULL REFERENCES users(id), -- recipient (host/user)
    amount_idr DECIMAL(12,2) NOT NULL DEFAULT 3500,
    status VARCHAR(20) DEFAULT 'active', -- active, refunded
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(conversation_id, payer_id)
);

-- Call sessions
CREATE TABLE call_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id), -- host being called
    caller_id UUID NOT NULL REFERENCES users(id), -- caller
    type VARCHAR(10) NOT NULL, -- 'voice', 'video'
    rate_idr DECIMAL(12,2) NOT NULL, -- rate per minute at time of call
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, rejected, active, ended, failed
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER DEFAULT 0,
    total_charge_idr DECIMAL(12,2) DEFAULT 0,
    platform_fee_idr DECIMAL(12,2) DEFAULT 0,
    host_earning_idr DECIMAL(12,2) DEFAULT 0,
    ended_reason VARCHAR(50), -- 'user_end', 'balance_insufficient', 'host_reject', 'timeout', 'network_error'
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Call billing ticks (per minute log)
CREATE TABLE call_billing_ticks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_session_id UUID NOT NULL REFERENCES call_sessions(id) ON DELETE CASCADE,
    tick_number INTEGER NOT NULL, -- minute 1, 2, etc.
    charge_idr DECIMAL(12,2) NOT NULL,
    deducted_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(call_session_id, tick_number)
);

-- Indexes for performance
CREATE INDEX idx_host_call_rates_host ON host_call_rates(host_id);
CREATE INDEX idx_paid_chat_unlocks_conv ON paid_chat_unlocks(conversation_id);
CREATE INDEX idx_call_sessions_caller ON call_sessions(caller_id);
CREATE INDEX idx_call_sessions_host ON call_sessions(host_id);
CREATE INDEX idx_call_sessions_status ON call_sessions(status);

-- Trigger to update updated_at
CREATE TRIGGER update_host_call_rates_updated_at
    BEFORE UPDATE ON host_call_rates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- ============================================
-- FASE 2C & 4C: DISAPPEARING MESSAGES, REACTIONS & WITHDRAWAL FEE
-- ============================================

-- 1. DISAPPEARING MESSAGES EXTENSION
ALTER TABLE messages ADD COLUMN disappear_mode VARCHAR(20) DEFAULT 'none'; 
ALTER TABLE messages ADD COLUMN disappear_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN viewed_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN is_screenshot_detected BOOLEAN DEFAULT false;
ALTER TABLE messages ADD COLUMN is_forwarded BOOLEAN DEFAULT false;
ALTER TABLE messages ADD COLUMN is_expired BOOLEAN DEFAULT false;

-- View tracking for view_once
CREATE TABLE message_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    viewer_id UUID NOT NULL REFERENCES users(id),
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, viewer_id)
);

-- 2. MESSAGE REACTIONS
CREATE TABLE message_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    emoji VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id, emoji)
);

-- 3. WITHDRAWAL SYSTEM
CREATE TABLE fee_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL,
    fee_type VARCHAR(20) NOT NULL, -- 'percentage', 'fixed'
    value DECIMAL(10,4) NOT NULL,
    applies_to VARCHAR(20) NOT NULL, -- 'all', 'host', 'host_with_agency', 'user'
    is_active BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed Default Fee Rules
INSERT INTO fee_rules (name, fee_type, value, applies_to, priority) VALUES
('Platform Fee', 'percentage', 0.1500, 'all', 1),
('Processing Fee', 'percentage', 0.0350, 'all', 2),
('Tax PPh', 'percentage', 0.1000, 'all', 3),
('Agency Fee', 'percentage', 0.0670, 'host_with_agency', 4);

CREATE TABLE withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    amount_requested DECIMAL(12,2) NOT NULL,
    gross_amount DECIMAL(12,2) NOT NULL,
    
    fee_platform DECIMAL(12,2) NOT NULL,
    fee_processing DECIMAL(12,2) NOT NULL,
    fee_tax DECIMAL(12,2) NOT NULL,
    fee_agency DECIMAL(12,2) DEFAULT 0,
    
    total_fee DECIMAL(12,2) NOT NULL,
    net_amount DECIMAL(12,2) NOT NULL,
    
    agency_id UUID REFERENCES agencies(id),
    
    status VARCHAR(20) DEFAULT 'pending', -- pending, approved, rejected, processing, completed, failed
    payment_method VARCHAR(20),
    bank_account_info JSONB,
    tx_reference VARCHAR(255),
    
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE withdrawal_fee_audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    withdrawal_id UUID NOT NULL REFERENCES withdrawals(id),
    fee_name VARCHAR(50) NOT NULL,
    fee_percentage DECIMAL(10,4) NOT NULL,
    fee_amount DECIMAL(12,2) NOT NULL,
    calculated_from DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_messages_disappear_at ON messages(disappear_at) WHERE disappear_at IS NOT NULL;
CREATE INDEX idx_message_reactions_msg ON message_reactions(message_id);
CREATE INDEX idx_withdrawals_user ON withdrawals(user_id);
CREATE INDEX idx_withdrawals_status ON withdrawals(status);
-- ============================================
-- FASE 4D: OPEN BOOKING HOST SYSTEM
-- ============================================

-- 1. HOST SCHEDULES (Recurring)
CREATE TABLE host_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL, -- 0=Sunday, 1=Monday, ..., 6=Saturday
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    slot_duration_minutes INTEGER DEFAULT 30,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(host_id, day_of_week)
);

-- 2. SCHEDULE EXCEPTIONS (Libur/Jam Khusus)
CREATE TABLE host_schedule_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exception_date DATE NOT NULL,
    type VARCHAR(20) NOT NULL, -- 'unavailable', 'special_hours'
    start_time TIME,
    end_time TIME,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. BOOKING TYPES
CREATE TABLE host_booking_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- 'chat_session', 'voice_call', 'video_call', 'custom_request'
    name VARCHAR(100) NOT NULL,
    description TEXT,
    duration_options INTEGER[] DEFAULT '{15,30,60}',
    price_per_minute DECIMAL(12,2) NOT NULL,
    min_duration INTEGER DEFAULT 15,
    max_duration INTEGER DEFAULT 60,
    is_active BOOLEAN DEFAULT true,
    requires_prepayment BOOLEAN DEFAULT true,
    allow_extend BOOLEAN DEFAULT true,
    extend_price_per_minute DECIMAL(12,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. BOOKINGS
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_code VARCHAR(30) UNIQUE NOT NULL,
    host_id UUID NOT NULL REFERENCES users(id),
    user_id UUID NOT NULL REFERENCES users(id),
    booking_type_id UUID NOT NULL REFERENCES host_booking_types(id),
    
    -- Schedule
    scheduled_at TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL, -- scheduled_at + duration
    
    -- Pricing (Follows multi-layer fee structure)
    base_price DECIMAL(12,2) NOT NULL,
    platform_fee DECIMAL(12,2) NOT NULL,
    processing_fee DECIMAL(12,2) NOT NULL,
    tax_fee DECIMAL(12,2) NOT NULL,
    agency_fee DECIMAL(12,2) DEFAULT 0,
    total_price DECIMAL(12,2) NOT NULL,
    host_earning DECIMAL(12,2) NOT NULL,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending', 
    payment_status VARCHAR(20) DEFAULT 'escrow', -- escrow, released, refunded
    escrow_released_at TIMESTAMPTZ,
    
    -- Execution
    actual_started_at TIMESTAMPTZ,
    actual_ended_at TIMESTAMPTZ,
    actual_duration_minutes INTEGER DEFAULT 0,
    
    -- Session info
    room_id UUID,
    join_token TEXT,
    
    -- Metadata
    user_notes TEXT,
    host_notes TEXT,
    cancelled_by UUID REFERENCES users(id),
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,
    refund_amount DECIMAL(12,2) DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 5. REMINDERS & DISPUTES
CREATE TABLE booking_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    reminder_type VARCHAR(20) NOT NULL, -- '24h', '1h', '15m', '5m'
    sent_at TIMESTAMPTZ,
    is_sent BOOLEAN DEFAULT false
);

CREATE TABLE booking_disputes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id),
    raised_by UUID NOT NULL REFERENCES users(id),
    reason VARCHAR(50) NOT NULL,
    description TEXT,
    evidence_urls TEXT[],
    status VARCHAR(20) DEFAULT 'open', -- open, under_review, resolved
    resolution_amount DECIMAL(12,2),
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_bookings_host ON bookings(host_id);
CREATE INDEX idx_bookings_user ON bookings(user_id);
CREATE INDEX idx_bookings_scheduled ON bookings(scheduled_at);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_host_schedules_host ON host_schedules(host_id);
-- ============================================
-- FASE 4E: OB/BO (OFFER BOOK / BOOK OFFER) SYSTEM
-- ============================================

-- 1. HOST OFFERS (OB - Host Promotes Slots)
CREATE TABLE host_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    offer_code VARCHAR(30) UNIQUE NOT NULL,
    
    title VARCHAR(100) NOT NULL,
    description TEXT,
    booking_type_id UUID NOT NULL REFERENCES host_booking_types(id),
    
    offer_mode VARCHAR(20) NOT NULL DEFAULT 'specific', -- 'specific', 'recurring'
    specific_at TIMESTAMPTZ,
    recurring_days INTEGER[],
    recurring_start_time TIME,
    recurring_end_time TIME,
    slot_duration_minutes INTEGER DEFAULT 30,
    
    base_price_per_minute DECIMAL(12,2) NOT NULL,
    discount_percentage DECIMAL(5,2) DEFAULT 0,
    final_price_per_minute DECIMAL(12,2) NOT NULL,
    
    max_bookings INTEGER DEFAULT 1,
    bookings_made INTEGER DEFAULT 0,
    max_bookings_per_user INTEGER DEFAULT 1,
    
    status VARCHAR(20) DEFAULT 'active', -- active, paused, expired, fully_booked
    expires_at TIMESTAMPTZ,
    is_auto_confirm BOOLEAN DEFAULT true,
    
    tags VARCHAR(50)[],
    thumbnail_url TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. USER OFFERS (BO - User Negotiates with Host)
CREATE TABLE user_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offer_code VARCHAR(30) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    host_id UUID NOT NULL REFERENCES users(id),
    
    booking_type_id UUID REFERENCES host_booking_types(id),
    offer_type VARCHAR(20) NOT NULL DEFAULT 'standard', -- 'standard', 'custom_price'
    
    proposed_at TIMESTAMPTZ NOT NULL,
    proposed_duration_minutes INTEGER NOT NULL,
    proposed_price_per_minute DECIMAL(12,2),
    total_offer_amount DECIMAL(12,2) NOT NULL,
    
    message TEXT,
    status VARCHAR(20) DEFAULT 'pending', 
    
    host_response_at TIMESTAMPTZ,
    host_message TEXT,
    host_counter_price DECIMAL(12,2),
    host_counter_at TIMESTAMPTZ,
    host_counter_expires_at TIMESTAMPTZ,
    
    converted_booking_id UUID, -- Will be set after accept
    is_prepaid BOOLEAN DEFAULT false,
    prepaid_amount DECIMAL(12,2) DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Update BOOKINGS to support OB/BO tracking
ALTER TABLE bookings ADD COLUMN source_type VARCHAR(20) DEFAULT 'direct'; -- 'ob', 'bo', 'direct'
ALTER TABLE bookings ADD COLUMN source_offer_id UUID;
ALTER TABLE bookings ADD COLUMN ob_discount_applied DECIMAL(12,2) DEFAULT 0;
ALTER TABLE bookings ADD COLUMN bo_negotiation_round INTEGER DEFAULT 0;

-- 4. ANALYTICS
CREATE TABLE host_offer_views (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_offer_id UUID NOT NULL REFERENCES host_offers(id) ON DELETE CASCADE,
    viewer_id UUID REFERENCES users(id),
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    converted_to_booking BOOLEAN DEFAULT false
);

-- Indexes
CREATE INDEX idx_host_offers_host ON host_offers(host_id);
CREATE INDEX idx_host_offers_status ON host_offers(status);
CREATE INDEX idx_user_offers_user ON user_offers(user_id);
CREATE INDEX idx_user_offers_host ON user_offers(host_id);
CREATE INDEX idx_user_offers_status ON user_offers(status);
-- ============================================
-- FASE 4F: REAL-TIME LOCATION & GPS TRACKING
-- ============================================

-- 1. Tambahkan kolom lokasi ke Host Offers (OB)
ALTER TABLE host_offers 
ADD COLUMN latitude DECIMAL(10,8),
ADD COLUMN longitude DECIMAL(11,8),
ADD COLUMN location_name VARCHAR(255),
ADD COLUMN share_location_type VARCHAR(20) DEFAULT 'none'; -- 'none', 'fixed', 'realtime'

-- 2. Tambahkan kolom lokasi ke User Offers (BO)
ALTER TABLE user_offers
ADD COLUMN latitude DECIMAL(10,8),
ADD COLUMN longitude DECIMAL(11,8),
ADD COLUMN location_name VARCHAR(255);

-- 3. Tambahkan kolom lokasi ke Bookings (Final Meeting Point)
ALTER TABLE bookings
ADD COLUMN meeting_latitude DECIMAL(10,8),
ADD COLUMN meeting_longitude DECIMAL(11,8),
ADD COLUMN meeting_location_name VARCHAR(255),
ADD COLUMN is_realtime_tracking_active BOOLEAN DEFAULT false;

-- 4. Tabel untuk log pergerakan realtime (optional, untuk history)
CREATE TABLE booking_location_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    latitude DECIMAL(10,8) NOT NULL,
    longitude DECIMAL(11,8) NOT NULL,
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_location_logs_booking ON booking_location_logs(booking_id);
