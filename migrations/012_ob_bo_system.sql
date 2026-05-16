-- ============================================
-- FASE 4E: OB/BO (OFFER BOOK / BOOK OFFER) SYSTEM
-- ============================================

-- 1. HOST OFFERS (OB - Host Promotes Slots)
CREATE TABLE host_offers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
