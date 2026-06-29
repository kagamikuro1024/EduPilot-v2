-- STEP 1: Create security_events table to track blocked attempts
CREATE TABLE IF NOT EXISTS security_events (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    student_id        UUID,
    session_id        UUID,
    query_content     TEXT,
    violation_reason  VARCHAR(100),
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- STEP 2: Index for performance on student analytics
CREATE INDEX IF NOT EXISTS idx_security_events_student_id ON security_events(student_id);
CREATE INDEX IF NOT EXISTS idx_security_events_created_at ON security_events(created_at);
