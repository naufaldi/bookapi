ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN location VARCHAR(255);
ALTER TABLE users ADD COLUMN website VARCHAR(500);
ALTER TABLE users ADD COLUMN is_public BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN reading_preferences JSONB;
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMPTZ;

CREATE INDEX idx_users_public ON users(is_public) WHERE is_public = true;
