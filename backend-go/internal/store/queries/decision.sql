-- Cấu hình decision (SRS FEAT-decision-model 5): ảnh chụp đọc MỘT lần mỗi lần làm mới, không đọc trên đường nóng (AC5 US-DM-02).

-- name: DecisionGatesAll :many
select gate, enabled, threshold, timeout_ms, version from decision_gates order by gate;

-- name: DecisionCourseGatesEnabled :many
-- Mọi cặp (lớp, cổng) đang bật; không có hàng = tắt.
select course_id, gate from course_decision_gates where enabled;
