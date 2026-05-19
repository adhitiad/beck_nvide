-- MIGRATION: 017_HIGH_PRIVACY
-- DESKRIPSI: Fitur Keamanan Transaksi, Enkripsi, & Fitur Privasi Tinggi (E2EE Key Registry, Disappearing Messages, Screenshot Detection, Incognito Mode, Profile Privacy, User Mutes)

-- 1. Tabel Registri Kunci E2EE Ujung-ke-Ujung (End-to-End Encryption Keys)
CREATE TABLE IF NOT EXISTS user_e2ee_keys (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    key_type VARCHAR(50) NOT NULL DEFAULT 'X25519',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Tambah kolom enkripsi dan disappearing messages pada tabel messages (private chat)
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_encrypted BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS disappear_mode VARCHAR(50) DEFAULT 'none'; -- 'none', 'view_once', '7s', '24h', etc.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS disappear_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS viewed_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_screenshot_detected BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_expired BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_forwarded BOOLEAN DEFAULT FALSE;

-- 3. Tambah kolom pengaturan privasi tambahan pada tabel users
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_private_profile BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_incognito BOOLEAN DEFAULT FALSE;

-- 4. Tambah kolom masa berlaku bisukan interaksi pada tabel user_mutes
CREATE TABLE IF NOT EXISTS user_mutes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    muter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    muted_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(muter_id, muted_id)
);
ALTER TABLE user_mutes ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
