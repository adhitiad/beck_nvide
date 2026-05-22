-- MIGRATION: 021_PLATFORM_OVERHAUL
-- DESKRIPSI: Overhaul fitur backend NVide Live setara platform established
-- (Love678, Bigo, Mango, Hot51)

-- ============================================
-- 1. HOST LEVEL SYSTEM (Newbie → Diamond)
-- ============================================
CREATE TABLE IF NOT EXISTS host_levels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(20) NOT NULL UNIQUE,       -- newbie, bronze, silver, gold, diamond
    display_name VARCHAR(50) NOT NULL,
    min_stream_hours INT NOT NULL DEFAULT 0,
    min_total_income BIGINT NOT NULL DEFAULT 0,
    commission_rate INT NOT NULL DEFAULT 50, -- host share percentage
    badge_url TEXT,
    perks JSONB DEFAULT '{}',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS host_tier VARCHAR(20) DEFAULT 'newbie';
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_stream_hours INT DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_income BIGINT DEFAULT 0;

-- Seed host levels
INSERT INTO host_levels (name, display_name, min_stream_hours, min_total_income, commission_rate, sort_order, perks) VALUES
    ('newbie', 'Newbie', 0, 0, 50, 0, '{"max_stream_quality":"720p"}'),
    ('bronze', 'Bronze', 50, 500000, 55, 1, '{"max_stream_quality":"720p","can_pk":true}'),
    ('silver', 'Silver', 200, 2000000, 60, 2, '{"max_stream_quality":"1080p","can_pk":true,"lucky_bag":true}'),
    ('gold', 'Gold', 500, 10000000, 65, 3, '{"max_stream_quality":"1080p","can_pk":true,"lucky_bag":true,"priority_listing":true}'),
    ('diamond', 'Diamond', 1000, 50000000, 70, 4, '{"max_stream_quality":"4k","can_pk":true,"lucky_bag":true,"priority_listing":true,"exclusive_effects":true}')
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- 2. VIP SYSTEM (SVIP, MVP, King)
-- ============================================
CREATE TABLE IF NOT EXISTS vip_levels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(20) NOT NULL UNIQUE,           -- svip, mvp, king
    display_name VARCHAR(50) NOT NULL,
    price BIGINT NOT NULL,                       -- harga per periode (IDR)
    duration_days INT NOT NULL DEFAULT 30,
    badge_url TEXT,
    chat_color VARCHAR(20) DEFAULT '#FFFFFF',
    name_glow_color VARCHAR(20),
    privileges JSONB DEFAULT '{}',               -- {"exclusive_chat":true,"entry_effect":true,...}
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_vip (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vip_level_id UUID NOT NULL REFERENCES vip_levels(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    auto_renew BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_vip_user_id ON user_vip(user_id);
CREATE INDEX IF NOT EXISTS idx_user_vip_expires ON user_vip(expires_at);

CREATE TABLE IF NOT EXISTS vip_emoticons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    vip_level_id UUID NOT NULL REFERENCES vip_levels(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    code VARCHAR(30) NOT NULL UNIQUE,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS entry_effects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    vip_level_id UUID NOT NULL REFERENCES vip_levels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    animation_url TEXT NOT NULL,
    sound_url TEXT,
    duration_ms INT NOT NULL DEFAULT 3000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed VIP levels
INSERT INTO vip_levels (name, display_name, price, duration_days, chat_color, sort_order, privileges) VALUES
    ('svip', 'SVIP', 500000, 30, '#FFD700', 1, '{"exclusive_chat":true,"special_emoticons":true,"entry_effect":true,"gift_discount":5}'),
    ('mvp', 'MVP', 1500000, 30, '#FF4500', 2, '{"exclusive_chat":true,"special_emoticons":true,"entry_effect":true,"gift_discount":10,"invisible_visit":true,"priority_dm":true}'),
    ('king', 'King', 5000000, 30, '#FF0000', 3, '{"exclusive_chat":true,"special_emoticons":true,"entry_effect":true,"gift_discount":15,"invisible_visit":true,"priority_dm":true,"custom_badge":true,"exclusive_gifts":true,"vip_room_access":true}')
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- 3. ROYAL FAMILY / CLAN
-- ============================================
CREATE TABLE IF NOT EXISTS royal_families (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    badge_url TEXT,
    level INT NOT NULL DEFAULT 1,
    total_contribution BIGINT NOT NULL DEFAULT 0,
    max_members INT NOT NULL DEFAULT 50,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS royal_family_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    family_id UUID NOT NULL REFERENCES royal_families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member', -- owner, elder, member
    total_contribution BIGINT NOT NULL DEFAULT 0,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(family_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_rfm_family ON royal_family_members(family_id);
CREATE INDEX IF NOT EXISTS idx_rfm_user ON royal_family_members(user_id);

CREATE TABLE IF NOT EXISTS royal_family_contributions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    family_id UUID NOT NULL REFERENCES royal_families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    source VARCHAR(30) NOT NULL DEFAULT 'direct', -- direct, gift, mission
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rfc_family ON royal_family_contributions(family_id);

-- ============================================
-- 4. SHORT VIDEO (TikTok-like)
-- ============================================
CREATE TABLE IF NOT EXISTS short_videos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_url TEXT NOT NULL,
    thumbnail_url TEXT DEFAULT '',
    caption TEXT DEFAULT '',
    duration INT NOT NULL DEFAULT 0,             -- seconds, max 60
    like_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,
    share_count INT NOT NULL DEFAULT 0,
    view_count INT NOT NULL DEFAULT 0,
    gift_value BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, hidden, deleted, processing
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sv_user ON short_videos(user_id);
CREATE INDEX IF NOT EXISTS idx_sv_status ON short_videos(status);
CREATE INDEX IF NOT EXISTS idx_sv_created ON short_videos(created_at DESC);

CREATE TABLE IF NOT EXISTS short_video_likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    video_id UUID NOT NULL REFERENCES short_videos(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(video_id, user_id)
);

CREATE TABLE IF NOT EXISTS short_video_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    video_id UUID NOT NULL REFERENCES short_videos(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    parent_id UUID REFERENCES short_video_comments(id) ON DELETE CASCADE,
    like_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_svc_video ON short_video_comments(video_id);

-- ============================================
-- 5. PK BATTLE ENHANCED (Multi-Round)
-- ============================================
ALTER TABLE pk_battles ADD COLUMN IF NOT EXISTS current_round INT DEFAULT 1;
ALTER TABLE pk_battles ADD COLUMN IF NOT EXISTS total_rounds INT DEFAULT 3;
ALTER TABLE pk_battles ADD COLUMN IF NOT EXISTS round_duration INT DEFAULT 180;
ALTER TABLE pk_battles ADD COLUMN IF NOT EXISTS winner_reward BIGINT DEFAULT 0;
ALTER TABLE pk_battles ADD COLUMN IF NOT EXISTS config JSONB DEFAULT '{}';

CREATE TABLE IF NOT EXISTS pk_battle_rounds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    pk_id UUID NOT NULL REFERENCES pk_battles(id) ON DELETE CASCADE,
    round_number INT NOT NULL,
    score_a BIGINT NOT NULL DEFAULT 0,
    score_b BIGINT NOT NULL DEFAULT 0,
    winner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(pk_id, round_number)
);
CREATE INDEX IF NOT EXISTS idx_pkr_pk ON pk_battle_rounds(pk_id);

-- ============================================
-- 6. BACKPACK / INVENTORY
-- ============================================
CREATE TABLE IF NOT EXISTS inventory_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(30) NOT NULL,           -- gift, voucher, effect, badge_frame, chat_bubble
    icon_url TEXT NOT NULL,
    description TEXT DEFAULT '',
    is_tradeable BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_inventory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 1,
    source VARCHAR(30) NOT NULL DEFAULT 'purchase', -- purchase, wheel, mission, gift, admin
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ui_user ON user_inventory(user_id);
CREATE INDEX IF NOT EXISTS idx_ui_expires ON user_inventory(expires_at);

-- ============================================
-- 7. WHEEL OF FORTUNE
-- ============================================
CREATE TABLE IF NOT EXISTS wheel_prizes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(30) NOT NULL,           -- coin, gift, voucher, entry_effect, nothing
    value BIGINT NOT NULL DEFAULT 0,     -- coin amount or item_id reference
    item_id UUID REFERENCES inventory_items(id) ON DELETE SET NULL,
    icon_url TEXT DEFAULT '',
    probability FLOAT NOT NULL DEFAULT 0.1,  -- 0.0 - 1.0
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wheel_spins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    prize_id UUID NOT NULL REFERENCES wheel_prizes(id) ON DELETE CASCADE,
    cost BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ws_user ON wheel_spins(user_id);

-- ============================================
-- 8. DAILY MISSION & GAMIFICATION
-- ============================================
CREATE TABLE IF NOT EXISTS daily_missions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    title VARCHAR(200) NOT NULL,
    description TEXT DEFAULT '',
    type VARCHAR(30) NOT NULL,                -- watch, send_gift, live_stream, login, share, comment
    target_value INT NOT NULL DEFAULT 1,      -- how many times/minutes
    reward_type VARCHAR(20) NOT NULL,          -- coin, exp, item
    reward_value BIGINT NOT NULL DEFAULT 0,
    reward_item_id UUID REFERENCES inventory_items(id) ON DELETE SET NULL,
    role_target VARCHAR(20) DEFAULT 'all',     -- all, user, host
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_missions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mission_id UUID NOT NULL REFERENCES daily_missions(id) ON DELETE CASCADE,
    progress INT NOT NULL DEFAULT 0,
    is_completed BOOLEAN DEFAULT FALSE,
    is_claimed BOOLEAN DEFAULT FALSE,
    mission_date DATE NOT NULL DEFAULT CURRENT_DATE,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, mission_id, mission_date)
);
CREATE INDEX IF NOT EXISTS idx_um_user_date ON user_missions(user_id, mission_date);

CREATE TABLE IF NOT EXISTS user_badges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_name VARCHAR(100) NOT NULL,
    badge_icon TEXT DEFAULT '',
    achievement_key VARCHAR(100) NOT NULL,     -- top_supporter, veteran_viewer, big_spender, etc.
    description TEXT DEFAULT '',
    earned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, achievement_key)
);
CREATE INDEX IF NOT EXISTS idx_ub_user ON user_badges(user_id);

CREATE TABLE IF NOT EXISTS leaderboard_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    type VARCHAR(20) NOT NULL,            -- host_income, user_gift, family_contribution
    period VARCHAR(10) NOT NULL,           -- daily, weekly, monthly
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score BIGINT NOT NULL DEFAULT 0,
    rank INT NOT NULL DEFAULT 0,
    snapshot_date DATE NOT NULL,
    reward_claimed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ls_type_period ON leaderboard_snapshots(type, period, snapshot_date);

-- Seed daily missions
INSERT INTO daily_missions (title, description, type, target_value, reward_type, reward_value, role_target) VALUES
    ('Tonton 10 Menit', 'Tonton live stream selama 10 menit', 'watch', 10, 'coin', 100, 'all'),
    ('Kirim 3 Gift', 'Kirim 3 gift ke host manapun', 'send_gift', 3, 'coin', 200, 'user'),
    ('Login Harian', 'Login ke aplikasi hari ini', 'login', 1, 'exp', 50, 'all'),
    ('Live 30 Menit', 'Streaming live selama 30 menit', 'live_stream', 30, 'coin', 500, 'host'),
    ('Komentar 5 Kali', 'Tulis 5 komentar di live chat', 'comment', 5, 'exp', 30, 'all'),
    ('Share Video', 'Share 1 video pendek', 'share', 1, 'coin', 50, 'all')
ON CONFLICT DO NOTHING;

-- ============================================
-- 9. AGENCY MLM (Multi-Level Commission)
-- ============================================
ALTER TABLE agency_hosts ADD COLUMN IF NOT EXISTS referrer_host_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE agency_hosts ADD COLUMN IF NOT EXISTS mlm_level INT DEFAULT 0;

CREATE TABLE IF NOT EXISTS agency_commissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    agency_id UUID NOT NULL REFERENCES agencies(id) ON DELETE CASCADE,
    from_host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level INT NOT NULL DEFAULT 1,           -- 1=direct, 2=indirect
    amount BIGINT NOT NULL,
    percentage INT NOT NULL,                 -- commission percentage applied
    source_tx_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ac_agency ON agency_commissions(agency_id);
CREATE INDEX IF NOT EXISTS idx_ac_to_host ON agency_commissions(to_host_id);

-- ============================================
-- 10. VOICE CHAT ROOM
-- ============================================
CREATE TABLE IF NOT EXISTS voice_rooms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT DEFAULT '',
    max_speakers INT NOT NULL DEFAULT 8,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, ended
    total_gift_value BIGINT NOT NULL DEFAULT 0,
    listener_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_vr_status ON voice_rooms(status);

CREATE TABLE IF NOT EXISTS voice_room_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    room_id UUID NOT NULL REFERENCES voice_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'listener', -- host, speaker, listener
    is_muted BOOLEAN DEFAULT FALSE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    UNIQUE(room_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_vrp_room ON voice_room_participants(room_id);

-- ============================================
-- 11. LUCKY BAG / RANDOM GIFT BOX
-- ============================================
CREATE TABLE IF NOT EXISTS lucky_bags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    min_value BIGINT NOT NULL DEFAULT 100,
    max_value BIGINT NOT NULL DEFAULT 10000,
    total_count INT NOT NULL DEFAULT 10,
    remaining INT NOT NULL DEFAULT 10,
    total_pool BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, depleted, expired
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_lb_stream ON lucky_bags(stream_id);

CREATE TABLE IF NOT EXISTS lucky_bag_claims (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    bag_id UUID NOT NULL REFERENCES lucky_bags(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(bag_id, user_id)
);

-- ============================================
-- 12. LODGING / STAY MODE
-- ============================================
ALTER TABLE streams ADD COLUMN IF NOT EXISTS is_lodging_mode BOOLEAN DEFAULT FALSE;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS lodging_alarm_price BIGINT DEFAULT 5000;

CREATE TABLE IF NOT EXISTS lodging_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(30) NOT NULL,       -- alarm, sound_effect, gift_wake
    gift_id UUID REFERENCES gifts(id) ON DELETE SET NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_le_stream ON lodging_events(stream_id);

-- ============================================
-- 13. STREAM AUTO-TAGS
-- ============================================
CREATE TABLE IF NOT EXISTS stream_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(50) NOT NULL UNIQUE,
    category VARCHAR(30) DEFAULT 'general',   -- general, mood, content, special
    keywords TEXT[] DEFAULT '{}',              -- keywords that trigger this tag
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stream_tag_map (
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES stream_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (stream_id, tag_id)
);

-- Seed stream tags
INSERT INTO stream_tags (name, category, keywords) VALUES
    ('dance', 'content', '{"dance","dancing","tarian","joget","goyang"}'),
    ('sing', 'content', '{"sing","singing","karaoke","nyanyi","lagu"}'),
    ('chat', 'content', '{"chat","ngobrol","curhat","talk","bincang"}'),
    ('gaming', 'content', '{"game","gaming","play","main","esport"}'),
    ('mukbang', 'content', '{"makan","mukbang","food","eating","kuliner"}'),
    ('asmr', 'content', '{"asmr","whisper","relax","santai"}'),
    ('workout', 'content', '{"workout","fitness","gym","olahraga","exercise"}'),
    ('cooking', 'content', '{"masak","cooking","cook","recipe","resep"}'),
    ('hot', 'mood', '{"hot","sexy","spicy","panas"}'),
    ('chill', 'mood', '{"chill","santai","relax","calm"}'),
    ('party', 'mood', '{"party","pesta","celebrate","happy"}'),
    ('late_night', 'special', '{"malam","night","midnight","begadang","late"}'),
    ('pk_battle', 'special', '{"pk","battle","versus","vs","lawan"}'),
    ('newbie', 'special', '{"newbie","baru","first","debut","perdana"}')
ON CONFLICT (name) DO NOTHING;
