-- Add columns if not exists
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_filename VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMP;

-- Create table if not exists
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(50) NOT NULL DEFAULT 'SYSTEM',
    font_size VARCHAR(50) NOT NULL DEFAULT 'DEFAULT',
    reduce_motion BOOLEAN NOT NULL DEFAULT FALSE,
    default_student_page VARCHAR(50) NOT NULL DEFAULT 'ASSIGNMENTS',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_theme CHECK (theme IN ('LIGHT', 'DARK', 'SYSTEM')),
    CONSTRAINT chk_font_size CHECK (font_size IN ('SMALL', 'DEFAULT', 'LARGE')),
    CONSTRAINT chk_default_page CHECK (default_student_page IN ('ASSIGNMENTS', 'CHAT'))
);
