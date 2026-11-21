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
| DEF-R106S-08 | Thấp | TC-G05-17 | Công bố lô có bài miễn: hộp xác nhận chỉ hiện «8 bài đã duyệt chấm theo bản lưu ý cũ, vẫn công bố được» (không tách a/b); dòng «k bài chấm theo bản lưu ý cũ: a đã sửa tay, b duyệt nguyên nháp» chỉ hiện SAU khi công bố (thông báo, `QueueTab.tsx:299`), chữ «bản lưu ý cũ» ≠ «bản cũ». US-GD-05 AC6a / SRS 208: dòng nằm TRONG hộp xác nhận, chữ «k bài chấm theo bản cũ: a đã sửa tay, b duyệt nguyên nháp» (số a/b chỉ có từ phản hồi `exempt`, nên cần tính trước: đếm bài đã chọn có `first_reviewed_at` & lệch bản) | MỞ |
| DEF-R106S-09 | Thấp | TC-G05-12 | «Chấm lại» hàng loạt: bấm → khối «N bài chấm theo bản cũ · Chấm lại» biến mất ngay (đếm stale loại QUEUED/RUNNING), thay bằng «Đã xếp 8 bài chấm lại»; không có nút khoá «Đang chấm lại …» và không hiện tiến độ n/N (8 bài, stub 12 s/lời gọi, ~100 s: chữ chỉ đổi ở cuối «Đã chấm xong 8 bài»); /events 200, kết quả đúng (8 DONE round 2). TC ghi nút khoá «Đang chấm lại …» + SSE `job.progress` | MỞ |
| (quan sát, không tính F) | Thấp | `/grading` | Bộ lọc «Bài tập» nạp 30 bài đầu (`useAssignments` limit 30, không «tải thêm»); lớp có > 30 bài tập (QD9G 78) thì bài thứ 31+ không chọn được và liên kết sâu `?tab=queue&assignment=<id>` bị bỏ khỏi URL; `?assignment=` thiếu `tab=queue` cũng bị bỏ | ghi nhận |
| DEF-R106S-10 | Thấp | TC-G06-06 | `/grading/[submissionId]` kết quả v1: một nút «bản v1 · 3 lưu ý» trên thanh công cụ (`GuidanceMeta.tsx`, tổng số lưu ý của cả bài) thay vì phụ đề `meta` «bản v1 · 3 lần sửa» trên NHÃN từng tiêu chí có lưu ý áp dụng (US-GD-06 AC3, SRS 412). Popover liệt kê chữ lưu ý nhóm theo tiêu chí (đúng); kết quả v0 không có phụ đề (đúng, TC-G06-07 P) | MỞ |
| DEF-R106S-11 | Thấp | TC-G06-11 | Chặn `GET …/guidance/stats` trả 500: khối «Lưu ý chấm» vẫn hiện «Bản v1 (đang dùng) · chưa đủ bài đã duyệt để đo» (nói sai: lỗi bị coi là chưa đủ dữ liệu), không «—», không nút «Thử lại» (`GuidancePanel.tsx`: `stats.data` rỗng khi lỗi). US-GD-06 AC7 | MỞ |
| DEF-R106S-12 | Thấp | TC-G07-07 | `cmd/regress run` chỉ nhận `--course`, báo cáo không có «số tốt lên» (US-GD-07 AC3: `--assignment <id>`, {n, MAE, lệch TB, xấu đi, tốt lên}) | MỞ |
| DEF-R106S-13 | Thấp | TC-G07-08 | `regress run` không kiểm ADMIN / môi trường đo (chỉ `export` có cổng REGRESSION_EXPORT_ALLOWED) | MỞ |
| DEF-R106S-14 | Thấp | TC-G07-09 | `regressed()` chỉ xét MAE và \|lệch TB\|; `worse` không vào mã thoát, baseline không theo tiêu chí → «tiêu chí vốn đúng nay sai 0,6» không bị bắt; thiếu `benchmarks/guidance/baseline.json` | MỞ |
| DEF-R106S-15 | Thấp | TC-G07-10 | Thiếu checklist PR ghi lệnh `regress run` (guide có, CI đúng không chạy) | MỞ |
| DEF-R106S-16 | Thấp | TC-G08-07 | Gợi ý khi soạn thread: debounce thực ≈ 800 ms (`usePostGate.tsx`, precheck 783–801 ms rồi similar-confirmed ~890–910 ms) thay vì 600 ms (US-GD-08 AC3); mục gợi ý chỉ có tiêu đề liên kết, không nhãn «Đã xác nhận» cạnh từng mục | MỞ |
| DEF-R106S-17 | Trung bình | TC-G09-09 | `GET /courses/{id}/questions/gen-stats?job_id=…` không bao giờ trả `hints` (US-GD-09 AC6): `PracticeGenHints` (queries/guidance.sql, guidance.sql.go:983) có `where … payload ->> 'course_id' = …` nhưng `jobs` không có cột `payload` (chỉ id, kind, status, progress, result, error, owner_id, created_at, updated_at, finished_at) → SQL lỗi, `GenStatsOf` nuốt lỗi (`herr == nil`). `jobs.result.hints` đã có đúng [MO_HO, LAC_DE, TRUNG_LAP]. Lọc theo lớp: dùng `owner_id`/khoá từ result, hoặc bỏ lọc payload | MỞ |
