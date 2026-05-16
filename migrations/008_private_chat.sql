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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL, -- 'delivered', 'read'
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(message_id, user_id, status)
);

-- Attachments
CREATE TABLE message_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
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
