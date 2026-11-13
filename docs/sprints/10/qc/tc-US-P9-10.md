# QC test case — US-P9-10 Liêm chính: luyện đề không phá bài thi tính điểm
Nguồn: US-P9-10 + SRS FR-21–24, Q-P9-2, 3, 4. Bài thi PE: GV tạo qua `POST …/exams` (FEAT-weekly-exam), trạng thái `DRAFT→SCHEDULED→OPEN→CLOSED→PUBLISHED`. Cách dựng nhanh: Q-S10-9.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-10-01 | AC1 | 6 bài thi ở 6 trạng thái (DRAFT, SCHEDULED, OPEN, CLOSED chưa công bố, PUBLISHED, xoá mềm), mỗi bài một câu duy nhất X1…X6 cùng chủ đề T | `go test ./internal/practice -run TestPracticeNoExamLeak -v` | PASS; log cho thấy đủ 6 trạng thái | GO |
| TC-10-02 | AC1 | Cùng | SV dựng ≥ 50 lượt TOPIC/MOCK chủ đề T; thu `question_id` mọi đề, phản hồi, `explain` | X1, X2, X3, X4, X6 **không bao giờ** xuất hiện (đề / phản hồi / `explain`); X5 chỉ xuất hiện (được phép) | API |
| TC-10-03 | AC2 | – | `go test ./internal/practice -run TestPracticeNoAnswerLeak -v` | PASS; log nêu các trạng thái: TOPIC chưa trả lời / đã trả lời / đã nộp; MOCK đang làm / đã nộp | GO |
| TC-10-04 | AC2 | `sv.kha` MOCK đang làm | Thu mọi phản hồi trong lượt: `start`, `GET attempt` (cả tải lại nhiều lần), `PUT answers` (cả lỗi `409/422/404`), `PATCH`, `explain` (409), lỗi 4xx bất kỳ; `grep -Ei 'answer_key\|"correct"\|explanation\|citations\|is_correct'` | 0 khớp trong **mọi** phản hồi (kể cả thân lỗi) | API |
| TC-10-05 | AC2 | TOPIC | Câu chưa trả lời: `GET attempt`; câu đã trả lời; lượt đã nộp | Chưa trả lời: 0 khoá nhạy cảm cho **câu đó**; đã trả lời: chỉ câu đó có; các câu chưa làm vẫn 0 | API |
| TC-10-06 | AC2 | – | So tên trường DTO sinh viên với `testdata/student_dto_allowlist.json`; thêm quét ở header, SSE | Không rò trong tiêu đề phản hồi / SSE | API |
| TC-10-07 | AC3 | Bài QUIZ do GV giao (bài thi PE `OPEN`); `sv.kha` bắt đầu lượt `IN_PROGRESS` | Chat riêng: «Giải thích SQL injection», hoặc câu nội dung môn học (`POST /chat/sessions/{id}/messages`) | Chat **từ chối** nội dung môn học (thông báo trong giờ thi); `GET /me/exam-lock` báo khoá | API + E2E |
| TC-10-08 | AC3 | Hết giờ / nộp bài thi | Hỏi lại cùng câu | Chat mở lại, trả lời; `exam-lock` mở | API + E2E |
| TC-10-09 | AC3 | `quiz-integrity.spec.ts` | `pnpm -C frontend exec playwright test quiz-integrity.spec.ts` | Xanh; chứng minh cả hai chiều (từ chối khi đang làm, mở lại khi hết giờ) **trên cùng màn làm bài** với bài QUIZ (bài thi PE) GV giao; QC đọc spec: không phải mock | E2E |
| TC-10-10 | AC4 | `EXAM_PAPER` `READY` (nội dung chứa chuỗi lạ «ZQ-DE-THI-17») và `ANSWER_KEY` (chuỗi «ZQ-DAP-AN-17») | **Ngoài giờ thi**: hỏi chat «tìm tài liệu ZQ-DE-THI-17» | Đề `EXAM_PAPER` tìm được như hiện tại (`audience ALL`); `ANSWER_KEY` **không** (`audience GRADING`) | API |
| TC-10-11 | AC4 | `sv.kha` đang có lượt bài thi PE `IN_PROGRESS` | Gọi `search_library` qua agent (hỏi chat nội dung chứa chuỗi lạ) hoặc test tool: `go test ./internal/agent -run SearchLibrary… -v` (tên test: Q-S10-22) | **Không** trả đoạn của `EXAM_PAPER` lẫn `ANSWER_KEY`; sau giờ thi lại tìm được `EXAM_PAPER` | API + GO |
| TC-10-12 | AC4 | – | Cùng TC-10-11 nhưng SV **khác** (không có lượt thi) trong lúc `sv.kha` đang thi | SV khác vẫn tìm được `EXAM_PAPER` (khoá theo từng sinh viên) | API |
| TC-10-13 | AC5 | `sv.kha` chỉ có lượt luyện `TOPIC` / `MOCK` | `GET /me/exam-lock`; hỏi chat nội dung môn học | `exam-lock` không khoá; chat trả lời bình thường, cả khi MOCK đang làm; thi thử đang làm **không** đổi `exam-lock` (so JSON trước/sau) | API |
| TC-10-14 | AC6 | MOCK hết giờ (ép `deadline_at`) và đóng bởi worker | `SELECT count(*) FROM exam_events WHERE …` (SV, thời điểm lượt) trước/sau | Không có dòng nào mới (kể cả rời tab / dán) | DB |
| TC-10-15 | AC7 | MOCK chưa nộp | `explain` mọi `itemId` | `409 NOT_SUBMITTED`; không đường nào khác trả `explanation` (xem TC-10-04) | API |
| TC-10-16 | AC8 | `sv.kha` (lớp 1 có lượt) | Lấy `question_id` lớp 2 (GV lớp 2 `GET …/questions`); `PUT …/answers/{itemId}` với `question_id` đó; `GET` câu / trích dẫn lớp 2 | `422 INVALID_QUESTION`; không đọc được câu / trích dẫn lớp 2 (`403`/`404`) | API |
| TC-10-17 | AC1 | Bài thi PE `OPEN` | Trong đó, SV tạo lượt luyện cùng chủ đề **ngay lúc đang thi** | Vẫn không có câu của bài thi OPEN | API |

## Nhánh lỗi
Phủ bởi TC-10-04 (lỗi cũng không rò), 15, 16.

## Phân quyền
STUDENT lớp khác → `403`/`404`; staff gọi API làm bài → `403`.

## Lịch sử sửa TC
(trống)
