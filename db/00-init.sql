\set ON_ERROR_STOP on

-- PostgreSQL's Docker entrypoint runs only top-level init files. Keep the
-- versioned migrations in their original folder and include them in order.
\ir migration/V1__init_extensions_and_enums.sql
\ir migration/V2__core_tables.sql
\ir migration/V3__chat_system.sql
\ir migration/V4__forum_system.sql
\ir migration/V5__rag_system.sql
\ir migration/V6__analytics.sql
\ir migration/V7__indexes.sql
\ir migration/V8__standardize_post_author.sql
\ir migration/V9__fix_author_constraints.sql
\ir migration/V10_MANUAL_SQL_ADD_DOCUMENT_FILE_SIZE_BYTES.sql
\ir migration/V11_MANUAL_SQL_ADD_DOCUMENT_STATUS_FAILED.sql
\ir migration/V12__align_documents_for_rag_pipeline.sql
\ir migration/V13__add_document_status_duplicate.sql
\ir migration/V14__analytics_logging.sql
\ir migration/V15__analytics_privacy_events.sql
\ir migration/V16__add_citations_jsonb.sql
\ir migration/V17__analytics_daily_summary.sql
\ir migration/V18__add_analytics_risk_unique_constraint.sql
\ir migration/V19__extend_verification_status_for_ta_review.sql
\ir migration/V20__add_escalated_to_thread_status.sql
\ir migration/V21__add_document_type_to_documents.sql
\ir migration/V22__add_global_content_hash_unique_idx.sql
\ir migration/V23__add_chat_message_created_at.sql
\ir migration/V24__add_forum_post_created_at.sql
\ir migration/V25__add_user_profile_and_preferences.sql

-- Seed default assignments for July 2026
DELETE FROM assignments WHERE title IN ('Lab 1: Introduction to AI', 'Assignment 1: Neural Networks', 'Lab 2: Search Algorithms', 'Assignment 2: Machine Learning');

INSERT INTO assignments (id, title, description, due_date, late_penalty_rule)
VALUES 
  (uuid_generate_v4(), 'Lab 1: Introduction to AI', 'Learn the basics of AI models and search algorithms.', '2026-07-06 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.'),
  (uuid_generate_v4(), 'Assignment 1: Neural Networks', 'Implement a simple feedforward neural network from scratch.', '2026-07-13 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.'),
  (uuid_generate_v4(), 'Lab 2: Search Algorithms', 'Solve maze problems using BFS, DFS, and A* search.', '2026-07-20 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.'),
  (uuid_generate_v4(), 'Assignment 2: Machine Learning', 'Train a classifier on the provided dataset and write a report.', '2026-07-27 23:59:00', 'Submissions within 24 hours after the deadline receive a 10% penalty unless prior approval exists.');

