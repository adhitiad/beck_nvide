-- ============================================
-- FASE 4B: PAID INTERACTIONS (PAY-TO-CHAT & PAY-PER-CALL)
-- ============================================

-- Host call rate configuration
CREATE TABLE host_call_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
