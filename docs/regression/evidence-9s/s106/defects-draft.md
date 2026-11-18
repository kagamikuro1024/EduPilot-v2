
### DEF-R106S-01 — Công tắc «Lưu ý chấm» không tải lại hàng sau 409 (TC-G01-17)
- Mức: Thấp · Sprint gốc: 10.6 · Mã 4ac1638a · ep-qc, lớp QD9G
- Tái hiện: 2 tab teacher cùng `/class/settings`; tab 1 bật «Lưu ý chấm»; tab 2 (version cũ) bấm công tắc.
- Kỳ vọng: 409 → tải lại hàng + báo tại chỗ · Thực tế: báo «Có người vừa sửa mục này. Chọn giữ bản nào. Thử lại» đúng, nhưng công tắc tab 2 vẫn «Tắt» trong khi DB enabled=t (v4). `GuidanceFlagsSection.tsx` chỉ `invalidateQueries` sau khi thành công.
- Bằng chứng: /tmp/qc9s/s106-g0117.png

### DEF-R106S-02 — «Rút lưu ý» thất bại rồi bấm lại trong 1 giờ trả lại đúng job đã FAILED (TC-G03-14)
- Mức: Trung bình · Sprint gốc: 10.6 · Mã 4ac1638a
- Tái hiện: bài có ≥5 bài duyệt / ≥3 lệch; stub LLM trả JSON sai (`rf_fake_support`) → job REFLECT_FAILED «Chưa rút được lưu ý. Thử lại.»; sửa nguồn lỗi, POST `/guidance/reflect` lại (Idempotency-Key mới).
- Kỳ vọng: thử lại tạo việc mới · Thực tế: 202 với CÙNG job_id FAILED (khoá Redis `ep:guidance:reflect:<sha>` giữ 1 giờ, `guidance.go:366-370`; không bị xoá khi job FAILED) → giảng viên không thử lại được tới khi dữ liệu duyệt đổi. Chỉ xoá khoá Redis mới ra job mới.
