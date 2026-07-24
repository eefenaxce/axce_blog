-- Idempotent migrations for existing databases

-- Add per-article comment toggle (defaults to enabled for existing posts)
ALTER TABLE articles ADD COLUMN IF NOT EXISTS comment_enabled BOOLEAN DEFAULT true;

-- Article upvotes (per-user or per-IP to support anonymous toggling)
CREATE TABLE IF NOT EXISTS article_upvotes (
    id SERIAL PRIMARY KEY,
    article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    ip_address VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(article_id, user_id),
    UNIQUE(article_id, ip_address)
);
CREATE INDEX IF NOT EXISTS idx_article_upvotes_article_id ON article_upvotes(article_id);
