-- MIGRATION: 015_OBSERVABILITY_AND_SCALING
-- DESKRIPSI: Penerapan indeks komposit untuk query sering dan template partisi bulanan pada tabel transaksi.

-- 1. INDEKS KOMPOSIT UNTUK OPTIMALISASI Kueri Sering (Database Slow Query Prevention)

-- Indeks komposit pada transactions untuk pengecekan cepat berdasarkan user_id dan status
CREATE INDEX IF NOT EXISTS idx_transactions_user_status 
ON transactions(user_id, status);

-- Indeks komposit pada stories untuk feed aktif yang belum kedaluwarsa
CREATE INDEX IF NOT EXISTS idx_stories_user_expired 
ON stories(user_id, is_expired);

-- Indeks komposit pada comments untuk memuat komentar story diurutkan waktu terbaru
CREATE INDEX IF NOT EXISTS idx_comments_story_created 
ON comments(story_id, created_at DESC);

-- Indeks komposit pada likes untuk pencarian cepat isi konten/postingan
CREATE INDEX IF NOT EXISTS idx_likes_content_type_id 
ON likes(content_type, content_id);

-- Indeks komposit pada stream_moderators untuk otorisasi cepat WebSocket hub
CREATE INDEX IF NOT EXISTS idx_stream_moderators_ids 
ON stream_moderators(stream_id, user_id);

-- Indeks komposit pada stream_sessions untuk pencarian session aktif
CREATE INDEX IF NOT EXISTS idx_stream_sessions_stream_viewer 
ON stream_sessions(stream_id, viewer_id, status);


-- 2. PARTISI DATABASE: TRANSACTIONS TABLE BY MONTH (RANGE PARTITIONING)
-- Catatan: Di PostgreSQL, tabel yang sudah ada tidak dapat langsung diubah menjadi partisi tanpa re-kreasi.
-- Berikut adalah skema migrasi aman untuk menerapkan Range Partitioning bulanan pada tabel 'transactions'.

-- Langkah A: Membuat tabel penampung transaksi terpartisi
CREATE TABLE IF NOT EXISTS transactions_partitioned (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    wallet_id VARCHAR(36) NOT NULL,
    amount BIGINT NOT NULL, -- disimpan dalam sen (integer)
    type VARCHAR(50) NOT NULL,
    reference_id VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (id, created_at) -- Kolom partisi wajib menjadi bagian dari primary key komposit
) PARTITION BY RANGE (created_at);

-- Langkah B: Membuat partisi bulanan default untuk tahun 2026 (Staging / Production)
CREATE TABLE IF NOT EXISTS transactions_y2026m05 PARTITION OF transactions_partitioned
    FOR VALUES FROM ('2026-05-01 00:00:00+00') TO ('2026-06-01 00:00:00+00');

CREATE TABLE IF NOT EXISTS transactions_y2026m06 PARTITION OF transactions_partitioned
    FOR VALUES FROM ('2026-06-01 00:00:00+00') TO ('2026-07-01 00:00:00+00');

CREATE TABLE IF NOT EXISTS transactions_y2026m07 PARTITION OF transactions_partitioned
    FOR VALUES FROM ('2026-07-01 00:00:00+00') TO ('2026-08-01 00:00:00+00');

-- Langkah C: Prosedur pemindahan data (diaktifkan saat maintenance window)
-- INSERT INTO transactions_partitioned SELECT * FROM transactions ON CONFLICT DO NOTHING;
-- ALTER TABLE transactions RENAME TO transactions_old;
-- ALTER TABLE transactions_partitioned RENAME TO transactions;
