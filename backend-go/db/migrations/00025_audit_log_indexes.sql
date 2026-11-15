-- Nhật ký kiểm toán cho ADMIN (US-P10-14, P-11-30): lọc theo khoảng thời gian và theo thực thể, phân trang (created_at, id) giảm dần.
-- Không sửa migration đã merge; chỉ thêm hai chỉ mục.

-- +goose Up
CREATE INDEX audit_log_created_idx ON audit_log (created_at DESC, id DESC);
CREATE INDEX audit_log_entity_created_idx ON audit_log (entity, created_at DESC, id DESC);

-- +goose Down
DROP INDEX audit_log_entity_created_idx;
DROP INDEX audit_log_created_idx;
