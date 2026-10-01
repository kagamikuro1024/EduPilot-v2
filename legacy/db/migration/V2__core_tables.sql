CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role user_role NOT NULL,
    student_code VARCHAR(50) UNIQUE,
    full_name VARCHAR(255),
    email VARCHAR(255) UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255),
    description TEXT,
    due_date TIMESTAMP,
    late_penalty_rule TEXT
);

CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID,
    assignment_id UUID,
    status submission_status,
    score FLOAT,
    updated_at TIMESTAMP
);