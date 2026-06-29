-- Migration to add unique constraint for at-risk analytics reporting
-- Prevents duplicate records for the same student on the same day

ALTER TABLE analytics_at_risk_students 
ADD CONSTRAINT unique_student_report UNIQUE (report_date, student_id);
