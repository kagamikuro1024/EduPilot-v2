\set ON_ERROR_STOP on

-- Idempotent migrations that must also run against an existing persistent
-- Hugging Face volume carrying the legacy completion marker.
\ir migration/V23__add_chat_message_created_at.sql
\ir migration/V24__add_forum_post_created_at.sql
