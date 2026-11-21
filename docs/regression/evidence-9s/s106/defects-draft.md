
### DEF-R106S-01 — Công tắc «Lưu ý chấm» không tải lại hàng sau 409 (TC-G01-17)
- Mức: Thấp · Sprint gốc: 10.6 · Mã 4ac1638a · ep-qc, lớp QD9G
- Tái hiện: 2 tab teacher cùng `/class/settings`; tab 1 bật «Lưu ý chấm»; tab 2 (version cũ) bấm công tắc.
- Kỳ vọng: 409 → tải lại hàng + báo tại chỗ · Thực tế: báo «Có người vừa sửa mục này. Chọn giữ bản nào. Thử lại» đúng, nhưng công tắc tab 2 vẫn «Tắt» trong khi DB enabled=t (v4). `GuidanceFlagsSection.tsx` chỉ `invalidateQueries` sau khi thành công.
- Bằng chứng: /tmp/qc9s/s106-g0117.png

### DEF-R106S-02 — «Rút lưu ý» FAILED: bấm lại trong 1 giờ trả lại đúng job FAILED dù Idempotency-Key mới (TC-G03-14)
- Mức: Trung bình · Sprint gốc: 10.6 · Mã 4ac1638a
- Tái hiện: bài có ≥5 bài duyệt / ≥3 lệch; stub LLM trả JSON sai (`rf_fake_support`) → job FAILED `REFLECT_FAILED` «Chưa rút được lưu ý. Thử lại.» (3 lần thử); sửa stub; POST `/guidance/reflect` lại với **Idempotency-Key khác** (UUID mới) và dữ liệu duyệt không đổi.
- Kỳ vọng: việc mới (retry) · Thực tế: 202 với CÙNG `job_id` FAILED; không có job mới (bảng `jobs` chỉ 1 dòng), LLM không được gọi. Khoá chống trùng là `ep:guidance:reflect:sha256(assignment|max(reviewed_at)|guidance_version)` giữ 1 giờ (`guidance.go:340-370`), không phụ thuộc khoá client, và không bị xoá khi job FAILED → nút «Thử lại» vô tác dụng tới khi giảng viên duyệt thêm bài. Chỉ xoá khoá Redis mới ra job mới. (Chống trùng khi cùng dữ liệu là chủ ý của SRS cho job còn chạy / đã xong; lỗi là áp cho cả job FAILED.)

### DEF-R106S-03 — `reflect` với bài tập không phải ESSAY trả 404 «Không tìm thấy bài tập» (TC-G03-03)
- Mức: Thấp · Mã 4ac1638a
- Tái hiện: bài tập QD9G đổi `type='QUIZ'` (cờ bật), teacher POST `/courses/{c}/assignments/{a}/guidance/reflect`.
- Kỳ vọng (TC-G03-03 / AC1): từ chối 409 hoặc 422, ghi rõ lý do; không job · Thực tế: không tạo job (đúng) nhưng mã 404 `ASSIGNMENT_NOT_FOUND`; `guidance.go:346-348` gộp `cur.Type != "ESSAY"` với ErrNoRows → thông báo sai sự thật (bài tập có tồn tại).

### DEF-R106S-04 — Khối «Lưu ý chấm» thiếu dòng «N chờ duyệt» (TC-G04-01)
- Mức: Thấp · Mã 4ac1638a · ngăn bài tập (`/grading?tab=assignments&open=…`; TC ghi `/assignments/[id]` nhưng route đó chỉ dành cho sinh viên)
- Kỳ vọng AC1: dòng `bản v2 · 3 chờ duyệt` + `Rút lưu ý` phụ · Thực tế: tiêu đề «Lưu ý chấm · Bản v2», nút «Rút lưu ý» (phụ, đúng) và «Áp dụng (2)» (đếm mục chờ áp dụng/nghỉ); không có số mục «chờ duyệt» (DB có 3 DRAFT). `GuidancePanel.tsx` head chỉ in `Bản vN`.

### DEF-R106S-05 — Danh sách mục sai chữ + không hiện mục «Đã nghỉ» (TC-G04-02)
- Mức: Thấp · Mã 4ac1638a · US-GD-04 AC2 / SRS FEAT-grading-guidance 201, 410, 412, 416 (BA xác nhận spec rõ)
- Thực tế (fixture 1 mục mỗi trạng thái + 3 Chờ duyệt): (1) `GuidancePanel.tsx:21` «Đang dùng · v{n}» thay «Đã áp dụng»; (2) `:125` «{n} bài đổi điểm» thay «{n} lần sửa»; (3) mục RETIRED có trong DB (`retired_in_version=2`) nhưng không hiện: `groups` (`GuidancePanel.tsx` ~dòng 214) lọc `i.status === DRAFT|APPROVED|REJECTED` nên «Đã nghỉ» (`stateOf` dòng 22) không bao giờ xuất hiện.
- Bằng chứng: /tmp/qc9s/s106-g0402.png; evidence-9s/s106/specs/qc-real-106-g04.spec.ts

### DEF-R106S-06 — Khối «Lưu ý chấm»: không EmptyState khi chưa có lưu ý, GET lỗi 500 làm khối biến mất không «Thử lại» (TC-G04-15)
- Mức: Thấp · Mã 4ac1638a
- Tái hiện: (a) bài tập bật cờ, chưa có mục → khối chỉ có tiêu đề «Chưa có bản» + hai nút, không dòng EmptyState; (b) chặn `GET …/guidance` trả 500 → `GuidancePanel` `return null` (`!d` khi lỗi), ngăn bài tập không báo gì, không nút «Thử lại». Kỳ vọng AC9: EmptyState một dòng; `<PageState>` + «Thử lại». (Phần «chữ sửa còn sau tải lại» chưa kết luận: sau tải lại không mở lại được ô «Sửa» trong script.)
