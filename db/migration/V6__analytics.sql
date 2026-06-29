CREATE TABLE analytics_topic_difficulty (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_date DATE,
    topic_name VARCHAR(255),
    query_count INT,
    difficulty_score FLOAT
);

CREATE TABLE analytics_at_risk_students (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_date DATE,
    student_id UUID,
    risk_level risk_level_enum,
    reason TEXT
);