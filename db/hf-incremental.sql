\set ON_ERROR_STOP on

-- Idempotent migrations that must also run against an existing persistent
-- Hugging Face volume carrying the legacy completion marker.
\ir migration/V23__add_chat_message_created_at.sql
\ir migration/V24__add_forum_post_created_at.sql
\ir migration/V25__add_user_profile_and_preferences.sql
\ir migration/V26__add_grade_report_support.sql

-- Seed default assignments for July 2026
DELETE FROM assignments WHERE title IN ('Lab 1: Introduction to AI', 'Assignment 1: Neural Networks', 'Lab 2: Search Algorithms', 'Assignment 2: Machine Learning');

INSERT INTO assignments (id, title, description, due_date, late_penalty_rule)
VALUES 
  (uuid_generate_v4(), 'Lab 1: Introduction to AI', 'Learn the basics of AI models and search algorithms.', '2026-07-06 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.'),
  (uuid_generate_v4(), 'Assignment 1: Neural Networks', 'Implement a simple feedforward neural network from scratch.', '2026-07-13 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.'),
  (uuid_generate_v4(), 'Lab 2: Search Algorithms', 'Solve maze problems using BFS, DFS, and A* search.', '2026-07-20 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.'),
  (uuid_generate_v4(), 'Assignment 2: Machine Learning', 'Train a classifier on the provided dataset and write a report.', '2026-07-27 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.');
