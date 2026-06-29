-- CORE
CREATE INDEX idx_submissions_user ON submissions(user_id);
CREATE INDEX idx_submissions_assignment ON submissions(assignment_id);

-- CHAT
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);

-- FORUM
CREATE INDEX idx_threads_author ON forum_threads(author_id);
CREATE INDEX idx_forum_posts_thread ON forum_posts(thread_id);

-- RAG
CREATE INDEX idx_chunks_document ON document_chunks(document_id);

-- VECTOR SEARCH (IMPORTANT)
CREATE INDEX idx_chunks_embedding ON document_chunks USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

-- GIN Index cho JSONB
CREATE INDEX idx_chunks_metadata ON document_chunks USING GIN (metadata);