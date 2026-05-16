-- ============================================
-- FASE 4D: OPEN BOOKING HOST SYSTEM
-- ============================================

-- 1. HOST SCHEDULES (Recurring)
CREATE TABLE host_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    reminder_type VARCHAR(20) NOT NULL, -- '24h', '1h', '15m', '5m'
    sent_at TIMESTAMPTZ,
    is_sent BOOLEAN DEFAULT false
);

CREATE TABLE booking_disputes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
