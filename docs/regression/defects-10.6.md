### DEF-R10.6-01 — reflect / regrade-stale qua lớp có cờ tắt trả 409 GUIDANCE_DISABLED trước 404
- Mức: Thấp
- Sprint gốc: 10.6 · TC gốc: TC-G03-24, TC-G05-24 · Hồi quy do sprint sau?: KHÔNG (có sẵn; summary 10.6 «Ghi nhận (1)» đã nêu, PM chưa đổi)
- Môi trường: ep-r, be0c0d96, vai teacher (GV lớp 761987 và lớp 761988), 1440
- Bước tái hiện: 1. `teacher` đăng nhập; lấy `C2` = id lớp 761988, `A` = id «Bài tập 03» của lớp 761987 (không thuộc lớp 761988). 2. Cờ `GRADING_GUIDANCE` của lớp 761988 đang tắt. 3. `POST /courses/$C2/assignments/$A/guidance/reflect` (Idempotency-Key bất kỳ) và `POST /courses/$C2/assignments/$A/regrade-stale`. (Script: `docs/regression/evidence-10.6/scripts/gd_k.py`, `gd_d.py`)
- Kỳ vọng (theo TC): `404` (bài tập không thuộc lớp)  · Thực tế: `409 GUIDANCE_DISABLED` (kiểm cờ lớp chạy trước kiểm sở hữu bài tập); bật cờ lớp 761988 thì ra 404 (suy luận, chưa chạy)
- Bằng chứng: `docs/regression/evidence-10.5/batchB1.log` (G03-24), `docs/regression/evidence-10.6/gd_d.log` (G05-24)  · Nghi ngờ mã: `internal/httpapi/assignmenthttp/guidance.go` (thứ tự guard cờ → tải bài tập)
- Chạy lại lần 2: ổn định (2/2)

### Ghi nhận không phải lỗi sản phẩm (đã loại)
- TC-G03-22 (reflect `error` / `invalid` → FAILED sau 3 lần): hành vi đổi theo P-11-23 (`llm.AwaitOutage`: mất nhà cung cấp không tiêu lượt, hoãn tới OutageMax) → job FAILED sau ≈ 10 phút, vẫn 0 DRAFT, mã `REFLECT_FAILED`. Ghi `LỖI THỜI (P-11-23)`.
- TC-G01-05 «lớp lạ → 404»: GV/TA/SV không thuộc lớp nhận 403 từ CourseAccessGuard (không đổi từ 10-2026; tc-00 sprint 10: «403 hoặc 404»); ADMIN nhận 404.
- `go test -tags testroutes ./internal/contract` đỏ ở `TestNoAnswerLeak` khi chạy trong shell đã export `QC_*`/`SEED_DEFAULT_PASSWORD`; chạy trong môi trường sạch: PASS (cả gói 57 s). Do môi trường chạy của QC, không phải lỗi.

## Đã xoá (lỗi thời có nguồn quyết định; lệnh chủ dự án 2026-11-18)
- TC-G03-22: xoá khỏi smoke-10.6.md và khỏi tổng — nguồn P-11-23 (PM 2026-11-15 chấp nhận: lỗi LLM_UNAVAILABLE/circuit_open xếp lại không tiêu lượt thử, nên hành vi «FAILED sau 3 lần ≈30 s» đã đổi).

## Lượt 2026-11-18…21 (QC cặp 3, mã 4ac1638a → chạy lại 0bdef404, ep-qc, lớp QD9G)
| Mã | Mức | TC | Tóm tắt | Trạng thái |
| --- | --- | --- | --- | --- |
| DEF-R106S-01 | Thấp | TC-G01-17 | 409 khi lưu công tắc nhưng hàng không tải lại (tab 2 vẫn «Tắt» khi DB bật) | ĐÓNG (a4f0a8b3; chạy lại P) |
| DEF-R106S-02 | Trung bình | TC-G03-14 | «Rút lưu ý» FAILED, bấm lại trong 1 giờ trả lại đúng job FAILED dù Idempotency-Key khác (khoá Redis `ep:guidance:reflect:<sha>` không xoá khi FAILED) | ĐÓNG (ab89d82b; lần 2 ra job mới SUCCEEDED, lần 3 cùng dữ liệu → cùng job lần 2) |
| DEF-R106S-03 | Thấp | TC-G03-03 | reflect với bài tập QUIZ trả 404 ASSIGNMENT_NOT_FOUND | ĐÓNG (ff09501f; nay 409 TYPE_NOT_SUPPORTED, không tạo job) |
| DEF-R106S-04 | Thấp | TC-G04-01 | Khối «Lưu ý chấm» thiếu «M chờ duyệt» | ĐÓNG (87073119; «Bản v2 · 3 chờ duyệt») |
| DEF-R106S-05 | Thấp | TC-G04-02 | «Đang dùng · vN» thay «Đã áp dụng», «bài đổi điểm» thay «lần sửa», mục «Đã nghỉ» không hiện | ĐÓNG (c3632cdc, P-11-77) |
| DEF-R106S-06 | Thấp | TC-G04-15 | Không EmptyState; GET 500 làm khối biến mất không «Thử lại» | ĐÓNG (81e58ce6); chữ sửa còn sau tải lại: P |
| DEF-R106S-07 | Thấp | TC-G04-02 | Chữ chiều lưu ý: UI «Chấm chặt hơn / Chấm rộng hơn / Làm rõ tiêu chí» (`guidanceApi.ts:85`) ≠ US-GD-04 AC2 «Gắt hơn / Nhẹ hơn / Làm rõ» | ĐÓNG (1595bfa1; chạy lại P: «Gắt hơn»…) |
