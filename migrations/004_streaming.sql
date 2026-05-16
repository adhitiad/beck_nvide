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
