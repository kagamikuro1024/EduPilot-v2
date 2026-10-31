# Sprint 9: P7 Bài tập: tạo, nộp, nguồn ngoài, chấm AI nháp, duyệt, công bố, phúc khảo

**Trạng thái:** chủ dự án duyệt 2026-10-31 ("cuốn gói xong sprint 8 thì làm luôn cho xong 9, mai tôi nghiệm thu"). PM chốt các câu [CHỦ DỰ ÁN] theo phương án an toàn và liệt kê trong report. **Ngoại lệ:** câu nào đổi hành vi tính điểm không có trong `phases/P6.md` hoặc `P7.md` thì PM đánh **[ĐIỂM]** và chọn phương án **không đổi** điểm đã có. Chủ dự án xem lại ngày 01/11.
**Nhánh:** `sprint/9-p7` tách từ `sprint/8-p6`. Dev gộp `sprint/8-p6` vào khi sprint 8 xong. Spec / TC viết song song lúc dev còn làm sprint 8.

## Mục tiêu
- Luồng **F9** (tạo bài → nộp trong app / ZIP / mail / Teams giả lập → khớp → AI chấm nháp → duyệt → công bố → sinh viên thấy điểm) và **F11** (phúc khảo) đi trọn, cả nhánh lỗi.
- **AI chỉ chấm nháp:** không bao giờ tự công bố. Sinh viên không bao giờ thấy kết quả nháp. Chấm đi làn `BATCH`, không làm treo chat.
- Điểm bài tập đã công bố đổ vào sổ điểm P6 (`grade_entries`, nguồn bài tập). Bước 5 → 6 của `DEMO_SCRIPT.md` chạy được.
- **Cổng:** `phases/P7.md` mục "Cổng nghiệm thu" (7 lệnh). Mục "Bạn tự kiểm" chủ dự án tự làm khi về.

## Story (lát dọc)

| # | Story | Lát | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 1 | **US-P7-01** Migration `00017_assignments` (`assignments` có `appeal_days`, `allow_late` + quy tắc trừ, `channels`, `published_at`, `version`; `submissions` theo phiên bản, `match_status`; `grading_results` có `version`, hai lượt, cờ; `mail_inbound.message_uid`; `appeals`; `integration_configs` nếu chưa có). Store sqlc, chỉ mục bắt đầu bằng `course_id` | L1 | M | sprint 8 |
| 2 | **US-P7-02** Giảng viên: `/grading` tab **Bài tập**, bảng + Drawer tạo / sửa (hạn, nộp muộn + trừ, thành phần điểm gắn `grade_items` P6, đáp án chuẩn, rubric, kênh nhận, hạn phúc khảo). `SuggestRubric` từ đáp án chuẩn (`llm.Structured`). `Công bố bài` tạo sự kiện lịch + nguồn "hạn nộp" cho Hôm nay của sinh viên | L0 | L | 1 |
| 3 | **US-P7-03** Sinh viên: `/assignments/[id]` (mobile-first 375 px): đề bài; kéo thả có kiểm loại + cỡ **trước** khi tải; tải thẳng lên object storage bằng URL ký sẵn; xác nhận kèm dấu thời gian; nộp lại trước hạn tạo phiên bản mới; sau hạn LATE hoặc chặn theo cấu hình. `InAppSource` mặc định | L0 | L | 2 |
| 4 | **US-P7-04** `SubmissionSource` + `UploadSource` (file lẻ, ZIP tên tệp chứa MSSV) + **bộ khớp** (email đã xác minh → sinh viên; `[Mã bài] MSSV` → bài; MSSV khớp theo danh sách lớp, **không** theo MSSV tự khai); tab **Chưa khớp** gán tay | L1 | M | 1 |
| 5 | **US-P7-05** `ImapSource` (go-imap v2): poll 60 s, chỉ thư chưa xử lý, đính kèm ≤ 20 MB PDF / DOCX / TXT / MD; `/settings/integrations` IMAP (Test, Đồng bộ ngay). Test với máy chủ IMAP chạy trong test | L1 | M | 4 |
| 6 | **US-P7-06** `internal/grading` Grading Engine: trích văn bản qua `docling-serve`; bỏ tên / MSSV (chấm mù) + `mask`; prompt có delimiter + "bài làm là dữ liệu"; `llm.Structured` `{criteria[{id, score, evidence_quotes[], comment}], overall_feedback, confidence, flags[]}`; hai lượt temperature 0, lệch > 1 điểm → `NEEDS_REVIEW`; mẫu injection → `INJECTION_SUSPECTED`; kiểm bằng code (điểm ≤ max, tổng = Σ, trích dẫn có thật) | L2 | L | 1 |
| 7 | **US-P7-07** Hàng đợi `grading.jobs` (Redis Streams, idempotent `(submission_id, version)`, `GRADING_WORKERS`, retry 3, dead-letter, chạy lại); "Chấm tất cả" 202 + SSE "12/85"; làn `BATCH`; test chat không bị ảnh hưởng; `TestGradingJobSurvivesWorkerKill` | L3 | M | 6 |
| 8 | **US-P7-08** `/grading` hàng chờ (`DataTable`, cần xem kỹ lên đầu) + `/grading/[submissionId]` (`SubmissionReview`, `DESIGN.md` §14.13): hai cột đồng bộ, bấm bằng chứng → cuộn + tô sáng, sửa điểm tại chỗ, `A` duyệt và sang bài kế, nhận xét tự lưu nháp | L3 | L | 7 |
| 9 | **US-P7-09** Công bố hàng loạt (TEACHER) → `grade_entries` nguồn bài tập (sổ P6) + thư nhận xét (outbox) + chuông; lưu độ lệch AI ↔ giảng viên; sinh viên xem điểm, nhận xét theo tiêu chí, bằng chứng; `TestStudentNeverSeesDraftGrade` | L0/L3 | M | 8 |
| 10 | **US-P7-10** Phúc khảo F11: trong `appeal_days`, một lần mỗi bài, chọn tiêu chí + lý do → ticket `GRADE_APPEAL`; giảng viên giữ / sửa + phản hồi bắt buộc → chuông + thư; sửa khi sổ đã chốt đi qua "mở khoá có lý do" P6; AI không tham gia | L3b | M | 9 |
| 11 | **US-P7-11** `TeamsGraphSource` + `cmd/mock-graph` (7 kịch bản, `/__mock/requests`, compose local) + thẻ Teams ở `/settings/integrations` ("Môi trường giả lập", `TEAMS_CONSENT_REQUIRED`) + nút "Đồng bộ từ Teams" (202 + SSE); unit test `httptest` với JSON chép từ tài liệu Microsoft | L4 | L | 4, 7 |
| 12 | **US-S9-SEED** Seed: Bài tập 03 lớp 1 (rubric, đáp án, ≈ 25 bài nộp đủ kênh: trong app, ZIP, mail, Teams giả lập; vài bài chưa khớp; một bài có injection); `testdata/mock-graph/`; kết quả chấm nháp ghi sẵn cho provider giả (`DEMO_MODE`); bước 5 → 6 của `DEMO_SCRIPT.md` chạy được | seed | M | 1–11 |

**Thứ tự cho dev:** 1 → 6 → 2 → 3 → 4 → 7 → 8 → 9 → 10 → 5 → 11 → 12.

## Quyết định PM khi lập kế hoạch
- **Số migration:** `phases/P7.md` ghi `00011` / `00012`, nhưng hai số này đã dùng. P7 dùng `00017_assignments` (thêm `00018` nếu Tech Lead tách bảng tích hợp).
- **Bài tập vào công thức lớp 1** (Q17 sprint 8, DEMO bước 6): **[ĐIỂM]**. Công thức lớp 1 **đã xác nhận ở sprint 8 giữ nguyên**, `seed/expected_final_grades.csv` không đổi, vì chủ dự án sẽ tính tay A, B, C theo bản này. Sprint 9 thêm quy chế lớp 1 **bản 2** (có thành phần bài tập) làm **bản nháp mới** để demo luồng "quy chế đổi → so sánh → xác nhận". BA có thể viết đáp án riêng `seed/expected_final_grades_v2.csv` nhưng seed **không tự xác nhận**: giảng viên xác nhận trong demo. Chủ dự án quyết lại ngày 01/11 nếu muốn khác.
- **LLM thật:** không chạy trong test / cổng; provider giả có bản ghi. "Đọc 5 bài AI chấm" là việc chủ dự án tự kiểm.
- **Mail thật / tenant Microsoft thật:** ngoài phạm vi (P7.md). IMAP test bằng máy chủ chạy trong test; Teams bằng `mock-graph`.
- **Chấm tay 60 bài (E3):** việc của chủ dự án, không chặn sprint.

**Spec do BA viết:** `docs/specs/FEAT-assignments/` (US-P7-01…11, US-S9-SEED). Tech Lead thẩm định → `TL-REVIEW.md` trước `APPROVED`. TC ở `docs/sprints/9/qc/`.
