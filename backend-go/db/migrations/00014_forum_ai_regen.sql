-- US-P4-06 AC5 (proposals #25): sửa thread khi bài AI còn PENDING → bài AI cũ REJECTED rồi AI sinh LẠI. Chỉ mục "một bài AI mỗi thread" (00007)
-- chỉ được tính trên bài chưa bị loại, nếu không việc `thread.answer` giao lại luôn gặp 23505.
-- Không CREATE bảng, không ALTER bảng: DROP + CREATE đúng một chỉ mục.

-- +goose Up
DROP INDEX forum_posts_one_ai_key;
CREATE UNIQUE INDEX forum_posts_one_ai_key ON forum_posts (thread_id) WHERE kind = 'AI' AND verification_state <> 'REJECTED';

-- +goose Down
-- Lùi về bản cũ chỉ được khi mọi thread còn tối đa MỘT bài AI (bài đã sinh lại sau khi bị loại làm CREATE INDEX báo 23505).
DROP INDEX forum_posts_one_ai_key;
CREATE UNIQUE INDEX forum_posts_one_ai_key ON forum_posts (thread_id) WHERE kind = 'AI';
