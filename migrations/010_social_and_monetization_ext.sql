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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    viewer_id UUID NOT NULL REFERENCES users(id),
    viewed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, viewer_id)
);

-- 2. MESSAGE REACTIONS
CREATE TABLE message_reactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    emoji VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id, emoji)
);

-- 3. WITHDRAWAL SYSTEM
CREATE TABLE fee_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
