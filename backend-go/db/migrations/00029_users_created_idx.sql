-- DEF-R4-04 (TC-P206-28, US-P2-06 AC8): danh sách người dùng của ADMIN phân trang (created_at, id) giảm dần — chỉ mục phục vụ đúng thứ tự đó,
-- không quét tuần tự + sắp xếp toàn bảng. Không sửa migration đã merge; chỉ thêm một chỉ mục.

-- +goose Up
CREATE INDEX users_created_idx ON users (created_at DESC, id DESC);

-- +goose Down
DROP INDEX users_created_idx;
