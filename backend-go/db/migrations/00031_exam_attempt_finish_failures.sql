-- DEF-R7S-EXAMTICK (hướng B): đếm số lần tick tự nộp thất bại trên một lượt (computeGrade lỗi, vd. bài 0 mục). Lượt đạt ngưỡng bị bỏ khỏi AttemptsDue,
-- không bị chọn lại mỗi 5 s vô hạn. Không đổi tính điểm. Cột có DEFAULT hằng nên không viết lại bảng.

-- +goose Up
ALTER TABLE exam_attempts ADD COLUMN finish_failures smallint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE exam_attempts DROP COLUMN finish_failures;
