-- ============================================
-- FASE 1: SEED DATA - RBAC & INITIAL USERS
-- ============================================

-- ============================================
-- INSERT ROLES (5 roles)
-- ============================================
INSERT INTO roles (id, name, description) VALUES
    (uuid_generate_v7(), 'guest', 'Tamu - akses terbatas, hanya bisa melihat konten publik'),
    (uuid_generate_v7(), 'user', 'Pengguna biasa - bisa mengakses fitur sosial dasar'),
    (uuid_generate_v7(), 'host', 'Host - bisa live streaming dan mengelola konten'),
    (uuid_generate_v7(), 'agency', 'Agency - mengelola multiple host, revenue sharing'),
    (uuid_generate_v7(), 'admin', 'Admin - akses penuh ke semua fitur')
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- INSERT PERMISSIONS (comprehensive permission matrix)
-- ============================================

-- User permissions
INSERT INTO permissions (id, resource, action, name, description) VALUES
    -- User: Self
    (uuid_generate_v7(), 'user', 'create', 'user:create', 'Daftar akun baru'),
    (uuid_generate_v7(), 'user', 'read', 'user:read:own', 'Baca profil sendiri'),
    (uuid_generate_v7(), 'user', 'update', 'user:update:own', 'Update profil sendiri'),
    (uuid_generate_v7(), 'user', 'delete', 'user:delete:own', 'Hapus akun sendiri'),
    -- User: Others (requires higher role)
    (uuid_generate_v7(), 'user', 'read', 'user:read:any', 'Baca profil pengguna lain'),
    (uuid_generate_v7(), 'user', 'update', 'user:update:any', 'Update profil pengguna lain'),
    (uuid_generate_v7(), 'user', 'delete', 'user:delete:any', 'Hapus akun pengguna lain'),
    (uuid_generate_v7(), 'user', 'ban', 'user:ban', 'Ban/toggle status user'),

    -- Authentication
    (uuid_generate_v7(), 'auth', 'login', 'auth:login', 'Login ke sistem'),
    (uuid_generate_v7(), 'auth', 'logout', 'auth:logout', 'Logout dari sistem'),
    (uuid_generate_v7(), 'auth', 'refresh', 'auth:refresh', 'Refresh token'),
    (uuid_generate_v7(), 'auth', 'verify', 'auth:verify', 'Verifikasi email/akun'),

    -- Stream / Live
    (uuid_generate_v7(), 'stream', 'create', 'stream:create', 'Buat live streaming'),
    (uuid_generate_v7(), 'stream', 'read', 'stream:read', 'Lihat live stream'),
    (uuid_generate_v7(), 'stream', 'update', 'stream:update', 'Update setting stream'),
    (uuid_generate_v7(), 'stream', 'delete', 'stream:delete', 'Hapus live stream'),
    (uuid_generate_v7(), 'stream', 'manage', 'stream:manage', 'Kelola semua stream (moderator)'),

    -- VOD (Video On Demand)
    (uuid_generate_v7(), 'vod', 'upload', 'vod:upload', 'Upload VOD'),
    (uuid_generate_v7(), 'vod', 'read', 'vod:read', 'Lihat VOD'),
    (uuid_generate_v7(), 'vod', 'update', 'vod:update', 'Update VOD'),
    (uuid_generate_v7(), 'vod', 'delete', 'vod:delete', 'Hapus VOD'),
    (uuid_generate_v7(), 'vod', 'moderate', 'vod:moderate', 'Moderasi VOD'),

    -- Story
    (uuid_generate_v7(), 'story', 'create', 'story:create', 'Buat story (24h)'),
    (uuid_generate_v7(), 'story', 'read', 'story:read', 'Baca story'),
    (uuid_generate_v7(), 'story', 'delete', 'story:delete', 'Hapus story sendiri'),
    (uuid_generate_v7(), 'story', 'moderate', 'story:moderate', 'Moderasi story'),

    -- Comment
    (uuid_generate_v7(), 'comment', 'create', 'comment:create', 'Buat komentar'),
    (uuid_generate_v7(), 'comment', 'read', 'comment:read', 'Baca komentar'),
    (uuid_generate_v7(), 'comment', 'update', 'comment:update:own', 'Update komentar sendiri'),
    (uuid_generate_v7(), 'comment', 'delete', 'comment:delete:own', 'Hapus komentar sendiri'),
    (uuid_generate_v7(), 'comment', 'moderate', 'comment:moderate', 'Moderasi komentar (hapus semua)'),

    -- Like
    (uuid_generate_v7(), 'like', 'create', 'like:create', 'Berlike'),
    (uuid_generate_v7(), 'like', 'delete', 'like:delete', 'Unlike'),

    -- Gift (monetization)
    (uuid_generate_v7(), 'gift', 'send', 'gift:send', 'Kirim gift'),
    (uuid_generate_v7(), 'gift', 'read', 'gift:read', 'Lihat history gift'),
    (uuid_generate_v7(), 'gift', 'manage', 'gift:manage', 'Kelola gift (admin)'),

    -- Wallet
    (uuid_generate_v7(), 'wallet', 'read', 'wallet:read:own', 'Baca balance wallet sendiri'),
    (uuid_generate_v7(), 'wallet', 'update', 'wallet:update', 'Update balance (admin/system)'),
    (uuid_generate_v7(), 'wallet', 'transaction', 'wallet:transaction', 'Transaksi wallet'),

    -- Agency
    (uuid_generate_v7(), 'agency', 'create', 'agency:create', 'Buat agency'),
    (uuid_generate_v7(), 'agency', 'read', 'agency:read', 'Baca data agency'),
    (uuid_generate_v7(), 'agency', 'update', 'agency:update', 'Update agency'),
    (uuid_generate_v7(), 'agency', 'manage', 'agency:manage', 'Kelola agency (admin)'),
    (uuid_generate_v7(), 'agency', 'host:manage', 'agency:host:manage', 'Kelola host di agency'),

    -- Host Application
    (uuid_generate_v7(), 'host_application', 'create', 'host_application:create', 'Apply jadi host'),
    (uuid_generate_v7(), 'host_application', 'read', 'host_application:read', 'Baca aplikasi host'),
    (uuid_generate_v7(), 'host_application', 'approve', 'host_application:approve', 'Approve aplikasi host'),
    (uuid_generate_v7(), 'host_application', 'reject', 'host_application:reject', 'Tolak aplikasi host'),

    -- Chat / Message
    (uuid_generate_v7(), 'chat', 'send', 'chat:send', 'Kirim pesan'),
    (uuid_generate_v7(), 'chat', 'read', 'chat:read', 'Baca pesan'),
    (uuid_generate_v7(), 'chat', 'moderate', 'chat:moderate', 'Moderasi chat'),

    -- Payment
    (uuid_generate_v7(), 'payment', 'create', 'payment:create', 'Buat payment (top-up)'),
    (uuid_generate_v7(), 'payment', 'read', 'payment:read', 'Baca history payment'),
    (uuid_generate_v7(), 'payment', 'callback', 'payment:callback', 'Payment callback (Duitku)'),
    (uuid_generate_v7(), 'payment', 'refund', 'payment:refund', 'Refund payment'),

    -- System
    (uuid_generate_v7(), 'system', 'config', 'system:config', 'Ubah system config'),
    (uuid_generate_v7(), 'system', 'monitor', 'system:monitor', 'Akses monitoring/logs'),
    (uuid_generate_v7(), 'system', 'backup', 'system:backup', 'Backup/restore database')
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
    admin_user_id := uuid_generate_v7();

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
    test_user_id := uuid_generate_v7();

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