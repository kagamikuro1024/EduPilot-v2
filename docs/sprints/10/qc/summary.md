# QC sprint 10 (FEAT-practice, P9) — kết luận pha 2

2026-11-14, nhánh `sprint/10-practice` ở `57f4cb25`, stack `ep-qc` thật (MacBook Apple Silicon, Colima 6 CPU / 10 GB, provider LLM `fake`, docling thật). k6 / Lighthouse / axe: **hoãn theo chủ dự án**. Chi tiết, bước tái hiện, bằng chứng: `report-stack.md`; trạng thái từng TC: `tc-status-pha2.md`; ảnh: `shots/`; script: `scripts/`; đề xuất: `../proposals.md` P-10-9.

## Kết luận theo story
«Một AC FAIL → story FAIL» (`QC.md`). Kết luận chỉ tính trên phạm vi ưu tiên đã chạy (AC chính, phân quyền, nhánh lỗi, API + màn 1440 / 375); TC chưa chạy ghi ở `tc-status-pha2.md` («CHƯA CHẠY»).

| Story | Kết luận | Lý do |
| --- | --- | --- |
| US-P9-01 Lược đồ, Quiz Engine, DTO | **PASS** | Bảng, enum, FK phức hợp, CHECK, UNIQUE từng phần, chỉ mục đúng; chấm `SHORT` 15 ca + 4 ca; `SHORT`/`ESSAY` bị chặn ở bài thi; DTO sinh viên không rò (Go + quét API); 403 / 404 đúng. Chưa chạy `goose down/up` (TC-01-04) |
| US-P9-02 Trích đề cũ | **FAIL** | **L1 Cao**: trích trên docling thật ra 0/24 câu (tái hiện cả trên stack sạch). Đạt: idempotency, 429, 403, `WRONG_DOCUMENT_TYPE`, tài liệu / bảng đáp án của lớp khác → 404 |
| US-P9-03 Sinh câu có trích dẫn | **FAIL** | L9 Thấp (`types:[]` → `202`). Đạt: câu `GENERATED` có `{document_id, chunk_id, page}` 5/5, `ANSWER_KEY` → 422, `count=11` → 422, 403, 429 ở việc thứ 6. `OVERLOADED` không kiểm được |
| US-P9-04 `/questions` + duyệt hàng loạt | **FAIL** | L11 Thấp (menu không có «Soạn tay»; dòng phụ trái luật chữ), L10 Thấp (mã lỗi bulk). Đạt: bulk 50 / 422 / không huỷ lô / 403, `VERSION_CONFLICT`, `CITATION_REQUIRED` ở `PUT review`, phân trang |
| US-P9-05 Dựng đề theo ma trận | **FAIL** | L6, L7, L8 (Thấp). Đạt: không lặp, thứ tự ổn định, hai SV khác thứ tự, p95 dựng đề 10 ms, loại câu chưa duyệt / lưu trữ / thuộc bài thi, 403 / 404 |
| US-P9-06 Luyện theo chủ đề | **FAIL** | AC4 «trích dẫn bấm được» không kiểm được: dữ liệu mẫu không có citations (L4 Trung bình); L15 Thấp (chữ trạng thái lớp rỗng khác TC). Đạt: phản hồi ngay, khoá đáp án, `explain` SSE (byte đầu 11 ms, 1 lời gọi LLM / nhiều lần), `NOT_ANSWERED`, quá tải → `EXPLAIN_FAILED` tại chỗ, submit idempotent, 404 / 403, `MCQ_MULTI` đủ tập; 375 px không tràn, 0 nút < 44 px |
| US-P9-07 Thi thử có giờ, tự lưu | **PASS** | `server_now`, `deadline` +30 phút, một lượt MOCK, `TAB_TAKEN_OVER`, `VERSION_CONFLICT`, dung sai 60 s (nhận / bỏ + notes / `ATTEMPT_CLOSED`), điểm `6.5`, `6.3`, `3.5`, `4.3` (decimal, nửa lên), worker đóng lượt quá hạn (`EXPIRED`), điểm không vào sổ điểm, không `exam_events`; e2e dev (ngoại tuyến, tải lại, đồng hồ) xanh. k6 hoãn; L13 Thấp ghi nhận |
| US-P9-08 Tự luận + nhận xét gợi ý | **PASS** | Nhận xét theo tiêu chí, không điểm, không vào sổ điểm; bài chèn lệnh không kéo kết quả; 4.001 ký tự → `422`; `429` ở lần thứ 9 trong lượt thử (trần 10 / giờ theo cửa sổ giờ); GV / TA / ADMIN → 403; 0 `ESSAY` ở MOCK. Quan sát L14 |
| US-P9-09 Lịch sử, điểm yếu, Hôm nay | **FAIL** | **L3 Trung bình**: «Ôn chủ đề» không biến mất ≤ 5 s ở `/courses/{id}/today`. Đạt: lịch sử phân trang, điểm yếu k ≥ 3, 403 cho staff, lọc theo lớp |
| US-P9-10 Liêm chính | **PASS** | Bài thi `OPEN` giữ câu → câu bị loại khỏi luyện đề; MOCK đang làm 0 khoá nhạy cảm trong mọi phản hồi; chat khoá khi có lượt bài thi PE (`409 EXAM_IN_PROGRESS`) và mở lại khi nộp; luyện đề không khoá chat; 0 `exam_events`; `NOT_SUBMITTED`; Go: `TestPracticeNoExamLeak`, `TestPracticeNoAnswerLeak`, `TestSearchLibrary*` xanh |
| US-P9-11 Nhập Forms | **FAIL** | L12 Thấp (ánh xạ theo tên cột trái ví dụ openapi; chữ lỗi tệp sai «PDF, DOCX hoặc PPTX», «50 MB»; ô chọn tệp mặc định). Đạt: ba bước chỉ TEACHER, `PREVIEW_REQUIRED`, khớp email đã xác minh / MSSV, `MISMATCH`, trùng lấy dòng mới nhất, điểm decimal, `replayed`, không đặt `published_at` |
| US-P9-12 Seed + cổng | **FAIL** | L4 Trung bình (seed không có citations / `EXTRACTED`; Quiz `SCHEDULED`; lớp 2 không có câu). L5 (`check-practice-seed.mjs`) dev đã sửa, đạt. k6 / Lighthouse / axe hoãn |

PASS: 4 / 12 (P9-01, 07, 08, 10). FAIL: 8 / 12.

## Lỗi theo mức
| Mức | Số | Mã |
| --- | --- | --- |
| Cao | 1 | L1 trích đề docling thật 0/24 |
| Trung bình | 2 mở + 1 đã sửa | L3 Hôm nay không huỷ cache theo lớp; L4 dữ liệu mẫu sai spec; L5 script kiểm seed (dev sửa ở `5d9c54e3`, QC kiểm lại: PASS) |
| Thấp | 10 | L6–L15 |
| Nghiêm trọng | 0 | |

## Phát hiện về quy trình (cần PM)
1. Dev sửa test có sẵn / đổi nhãn để xanh CI (`ac2a7aef`: `panels.spec.ts` +4 dòng, nhãn menu trả về «Câu trắc nghiệm» làm TC-04-02 đỏ). Không thấy sửa contract test, `expected_final_grades.csv`, `t.Skip` hay nới assertion; `today/staff_test.go` 6 → 7 có P-10-8 F1. Xin PM quyết P-10-9.
2. Công thức `OVERLOADED` ở handoff không làm đầy hàng cho việc nền (TC-02-14, 03-08 không kiểm được) và `llm_audit` không có payload (6 TC kiểm PII ở lời gọi LLM không kiểm được): cần dev chỉ cách.
3. Môi trường: `.env.local` mặc định đặt `BLOB_PUBLIC_ENDPOINT=localhost:9000` nên stack có offset PUT tệp lên MinIO của stack khác (`ep-qc` ghi nhầm vào stack demo). Đã sửa cục bộ (`9500`, không commit). Stack demo `edupilot` có thể có vài đối tượng thừa ở bucket do lượt đầu của QC (UPLOAD_INCOMPLETE).

## Seed sạch (TC-12-01…06)
Dựng lại từ trống (`down -v` → `pnpm dev` với `SEED_ON_EMPTY_DB=true`, `57f4cb25`): `scripts/seed.mjs` **mã 0, 334 s** (kể cả docling); chạy lại **mã 0, 21 s**, `question_bank` 129 / `practice_attempts` 42 / `exams` 3 không đổi (idempotent, TC-12-06 PASS). Nội dung: 104 `APPROVED` (14 chủ đề; 60 một đáp án, 12 nhiều đáp án, 20 đúng–sai, 8 trả lời ngắn, 2 tự luận, 2 code; EASY 39, MEDIUM 51, HARD 14), 20 `PENDING` + 5 `AI_DRAFT`, 42 lượt của 13 SV (1 lượt dở). Sai spec: 0 câu có citations, không có `EXTRACTED`, Quiz `SCHEDULED`, lớp 2 không có câu (L4); `node scripts/check-practice-seed.mjs` ở `57f4cb25` lỗi `Không thấy lớp 761987` (L5), sau bản sửa `5d9c54e3` báo `OK: 107 APPROVED, 26 PENDING, Quiz SCHEDULED, 23 lượt của 6 sinh viên mẫu`.
