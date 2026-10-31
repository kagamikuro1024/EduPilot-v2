# Handoff — US-P7-04 (`SubmissionSource`, `UploadSource`, khớp sinh viên, tab `Chưa khớp`)

Máy chủ + frontend nằm ở commit `e424a20b` (CI xanh) và commit kế tiếp (frontend tab `Chưa khớp`).

## Đã làm
- **Khớp (`internal/submission/match.go`):** `Match(Raw)` theo 4.4.2 — mã bài trong tên tệp / tiêu đề, MSSV từ tên tệp / nội dung / email đã xác minh, bảng `unmatched_reason` đủ 9 lý do; MSSV tự khai không bao giờ nối tài khoản (TLR-1). `TestMatchTable` ≥ 20 dòng.
- **`Ingest` (`ingest.go`):** một cửa ghi bài nộp cho mọi nguồn (`LoadEnv` + `Ingest`): sha256 chống trùng, `version` mới nhất của sinh viên, `UNMATCHED` khi không khớp.
- **`UploadSource` (`upload.go`):** `POST …/assignments/{aid}/submissions/upload` (`uploads/presign` purpose `submission_upload`, tệp lẻ hoặc ZIP ≤ 200 MiB). ZIP ≤ 5 mục hợp lệ → 200 đồng bộ; nhiều hơn → 202 + job `submission.zip` (chạy ở worker, `RegisterZip`). Chặn zip-slip (`ENTRY_UNSAFE`), loại tệp (`ENTRY_TYPE_NOT_ALLOWED`), bom (`ZIP_LIMIT`: 200 mục / 200 MiB giải nén, đếm khi đọc). Mục hỏng không làm hỏng cả lô.
- **Chưa khớp (`staff.go`):** `GET …/submissions?match_status=UNMATCHED`, `PUT …/{sid}/match`, `GET …/{sid}/file` (URL ký 5 phút, kiểm vai trước khi ký), `POST|DELETE …/{sid}/dismiss`. Dòng bỏ qua không còn trong danh sách và không tính vào `MISSING_SCORE`. TA bị chặn gán bài của bài tập đã có điểm công bố (Q5).
- **Sổ điểm (AC9 phần máy chủ):** `grade.Input.ItemClosed/ItemReason`; `LoadInputs` đọc kết quả đã công bố của bản MATCHED mới nhất; còn bài `UNMATCHED` thì sinh viên `INCOMPLETE` (`UNMATCHED_PENDING`), không thành 0 (TLR-9); bài tập chưa đóng → `ASSIGNMENT_OPEN`.
- **"Hôm nay":** `UNMATCHED_SUBMISSION` (bậc 60, một truy vấn tổng hợp cho nhân viên), `href /grading?tab=unmatched&course={id}`; topic `submission.resolved` dọn cache.
- **Frontend:** tab thứ ba `Chưa khớp` ở `/grading?tab=unmatched` (`features/grading/UnmatchedTab.tsx`, `submissionApi.ts`): bảng con trỏ, tải tệp qua URL ký khi bấm, hai ô chọn Bài tập + Sinh viên (tìm theo tên / MSSV) + `Gán`, `Bỏ qua` kèm lý do 1–200 + hoàn tác 5 giây, 409 `VERSION_CONFLICT` → InlineNotice + tải lại. Mã lỗi mới trong `ApiError.ts` / `vi.ts`.
- **OpenAPI:** +6 thao tác (257 → **263**); `contract_test.go` cập nhật số đếm.

## Ghi chú thiết kế
- `version` của dòng `UNMATCHED` = `updated_at` tính bằng micro-giây (không thêm cột; khoá lạc quan cho gán / bỏ qua). Hai nhân viên cùng thao tác → người sau `409 VERSION_CONFLICT`.
- Tệp tải lên và ZIP đi qua `uploads/presign` + `Complete` của `internal/document` (purpose `submission_upload`; `Complete` bây giờ chỉ nhận `purpose = document` cho đường tài liệu).

## Không làm / chuyển giao
- Câu "Sinh viên này đã có {n} phiên bản" ở hộp thoại gán: API chưa trả `n` (không có route nào cho nó). Cần quyết định BA: thêm trường vào `PUT …/match` (trả `409 ALREADY_REVIEWED` có sẵn) hay bỏ câu.
- Nút `Tải bài lên` (tệp lẻ / ZIP) ở tab `Hàng chờ` thuộc US-P7-08 (SRS 7.1 đặt ở `Hàng chờ`); route đã sẵn và có test.
- Thư `IGNORED` ở khối riêng + `mail_inbound` tính vào "còn chờ" của item ASSIGNMENT: làm cùng US-P7-05.
- Kiểm tra liên kết item `ASSIGNMENT` ở lược đồ điểm (`ASSIGNMENT_ALREADY_LINKED`, `ASSIGNMENT_SCALE_MISMATCH`, `ITEM_NOT_EDITABLE`): US-P7-09.
- Lệnh `Kiểm:` ghi `-tags integration`; repo không dùng tag đó (test chạy container thật mặc định).

## Test cũ đã sửa
- `today.Topics()` 27 → 28 và `TestTodayQueryBudget` (giảng viên 6 → 7 / 7 → 8): proposals sprint 9 #9 (PM đã chấp nhận).
- `internal/grade/me_test.go` `noStamps()`: chuỗi canary "7.77" trùng phần giây lẻ của `confirmed_at` → loại timestamp RFC3339 trước khi tìm canary (chặt hơn, không nới).
