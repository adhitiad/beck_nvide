-- ============================================
-- FASE 2: SOCIAL FEATURES OPTIMIZATION (SCHEMA ADJUSTMENTS)
-- ============================================

-- 1. Adjust Stories Table
ALTER TABLE stories ADD COLUMN IF NOT EXISTS media_url TEXT;
ALTER TABLE stories ADD COLUMN IF NOT EXISTS caption TEXT;
ALTER TABLE stories ADD COLUMN IF NOT EXISTS is_expired BOOLEAN DEFAULT false;

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_stories_is_expired ON stories(is_expired);

-- 2. Adjust Comments Table
ALTER TABLE comments ADD COLUMN IF NOT EXISTS content_id UUID;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS content_type VARCHAR(50);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS text TEXT;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS like_count INTEGER DEFAULT 0;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT false;

-- Migrate existing comments data
UPDATE comments SET content_id = story_id WHERE content_id IS NULL AND story_id IS NOT NULL;
UPDATE comments SET content_type = 'story' WHERE content_type IS NULL;
UPDATE comments SET text = content WHERE text IS NULL AND content IS NOT NULL;

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_comments_is_deleted ON comments(is_deleted);
CREATE INDEX IF NOT EXISTS idx_comments_content ON comments(content_id, content_type);

-- 3. Adjust Likes Table
ALTER TABLE likes ADD COLUMN IF NOT EXISTS content_id UUID;
ALTER TABLE likes ADD COLUMN IF NOT EXISTS content_type VARCHAR(50);

-- Migrate existing likes data
UPDATE likes SET content_id = story_id WHERE content_id IS NULL AND story_id IS NOT NULL;
UPDATE likes SET content_type = 'story' WHERE content_type IS NULL;

-- Add unique constraint for likes
ALTER TABLE likes DROP CONSTRAINT IF EXISTS likes_user_id_content_id_content_type_key;
ALTER TABLE likes ADD CONSTRAINT likes_user_id_content_id_content_type_key UNIQUE (user_id, content_id, content_type);

-- 4. Adjust Conversations Table
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS unread_count INTEGER DEFAULT 0;
