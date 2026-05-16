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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    latitude DECIMAL(10,8) NOT NULL,
    longitude DECIMAL(11,8) NOT NULL,
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_location_logs_booking ON booking_location_logs(booking_id);
