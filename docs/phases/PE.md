# PE — Thi hằng tuần: trắc nghiệm + lập trình C/C++ (M15)

| Ước lượng | Phụ thuộc | Nhánh |
| --- | --- | --- |
| 2,5 tuần | P2 (lớp, ghi danh, CourseAccessGuard), PU, P1 (gợi ý nháp đề) | `sprint/5-pe` |

**Mục tiêu:** giảng viên giao bài kiểm tra lấy điểm hằng tuần ngay trên hệ thống — **trắc nghiệm** và **lập trình C/C++ chấm tự động bằng bộ test ẩn** — sinh viên làm trong khung giờ, máy chấm, điểm tự công bố khi bài thi đóng với cả lớp (D56). Yêu cầu của thầy hướng dẫn, chủ dự án chốt 2026-10-03 (D54–D57).

**Luật của phase này:**
- Tính điểm là code thuần (`shopspring/decimal`, luật 5); LLM không chấm, không tính điểm bài thi. AI chỉ **gợi ý nháp** câu hỏi / test, giảng viên duyệt (D57).
- Code sinh viên chỉ chạy trong **sandbox** riêng (D55): không mạng, giới hạn CPU / RAM / thời gian / output / số tiến trình. Gateway và worker không bao giờ chạy `gcc` hay binary của sinh viên.
- Chấm là việc nền (luật 12): nộp → 202 + job → SSE tiến độ; consumer idempotent, retry 3, dead-letter.
- Phase đầu tiên dùng bảng ngân hàng câu hỏi → **PE tạo** `question_bank`, `question_options` ở dạng cuối (D45); P9 chỉ dùng lại cho luyện đề.

Ngoài phạm vi: luyện đề / thi thử không tính điểm (P9); trích câu hỏi từ đề cũ bằng LLM (P9); nối điểm thi vào sổ điểm (P6 tạo `grade_items` trỏ tới bài thi — PE lưu điểm ở `exam_attempts` và công bố cho sinh viên); ngôn ngữ ngoài C/C++; giám thị bằng webcam.

## Lát việc

**L1. Dữ liệu + quyền**
- [ ] Migration kế tiếp sau P2: `question_bank` (type `MCQ_SINGLE` / `MCQ_MULTI` / `TRUE_FALSE` / `CODE`, topic, difficulty, stem, answer_key jsonb, explanation, origin `MANUAL` / `AI_DRAFT`, review_status), `question_options`, `code_problems` (đề, ngôn ngữ cho phép `c11` / `cpp17`, giới hạn thời gian / bộ nhớ, mẫu khởi đầu, checker `EXACT` / `TOKENS` / `FLOAT_EPS`), `code_testcases` (input, expected, `is_sample`, trọng số; blob nếu lớn), `exams` (course_id, kind `MCQ` / `CODE` / `MIXED`, mở / đóng, thời lượng, xáo trộn, trạng thái `DRAFT` / `SCHEDULED` / `OPEN` / `CLOSED` / `PUBLISHED`, `version`), `exam_items`, `exam_attempts` (một mỗi SV mỗi bài, `started_at`, `deadline_at`, `submitted_at`, điểm `numeric`, `version`), `exam_answers`, `code_submissions` (mã nguồn, ngôn ngữ, trạng thái chấm, kết quả từng test, thời gian / bộ nhớ), `exam_events` (rời tab, dán), `similarity_reports`. Index bắt đầu bằng `course_id`.
- [ ] Quyền: TEACHER soạn / lên lịch / xem kết quả lớp mình; TA xem + soạn nháp (không công bố); STUDENT chỉ thấy bài của lớp mình, chỉ bài của chính mình; **đáp án, test ẩn, nhật ký liêm chính không bao giờ trả cho sinh viên** (kể cả sau khi công bố: chỉ kết quả từng test mẫu + đúng/sai tổng).

**L2. Sandbox chấm code** (theo `docs/research/2026-10-03-code-judge.md`)
- [ ] Thêm **một** container sandbox vào compose (D55); worker Go gọi qua HTTP nội bộ; không công bố cổng ra ngoài
- [ ] `internal/judge`: biên dịch (gcc / g++ ghim phiên bản, `-O2 -std=c11|c++17`), chạy từng test với giới hạn của bài, so đầu ra theo checker; kết quả `AC` / `WA` / `TLE` / `MLE` / `RE` / `CE` / `OLE`; điểm = tổng trọng số test đạt
- [ ] Hàng đợi Redis Streams `judge.submit` (luật 12), consumer song song `JUDGE_WORKERS`, idempotent theo `submission_id`; chấm lại toàn bài khi giảng viên sửa test (job 202)
- [ ] Chạy thử với test mẫu trong lúc làm bài (`Chạy thử`, giới hạn 10 lần / 10 phút / SV), chỉ test `is_sample`
- [ ] Test tấn công bắt buộc: vòng lặp vô hạn → TLE; fork bomb; đọc `/etc/passwd`; mở socket; ghi 1 GB ra stdout; cấp phát 2 GB → đều bị chặn, không ảnh hưởng bài khác

**L3. Soạn đề và ngân hàng câu hỏi**
- [ ] `/questions` (giảng viên / TA): tạo câu trắc nghiệm và bài code (đề Markdown, giới hạn, test mẫu + ẩn, nhập test bằng tệp zip), duyệt, gắn chủ đề / độ khó
- [ ] Gợi ý nháp bằng AI (D57): sinh câu trắc nghiệm hoặc test bổ sung qua `internal/llm` (làn BATCH, `Structured`), luôn vào trạng thái nháp chờ duyệt; giảng viên chạy lời giải mẫu qua sandbox để xác nhận test đúng trước khi dùng
- [ ] `/exams` (giảng viên): tạo bài thi từ ngân hàng, đặt khung giờ + thời lượng, xáo trộn, xem trước như sinh viên; sửa khi `DRAFT` / `SCHEDULED`; khoá sửa khi đã có người làm

**L4. Làm bài (sinh viên)**
- [ ] Bài thi xuất hiện ở "Hôm nay" (Provider "bài thi sắp mở / đang mở") và `/calendar` khi P8 có
- [ ] Bắt đầu → đồng hồ riêng `deadline_at = min(started_at + thời lượng, đóng)` tính phía máy chủ; tự lưu câu trả lời và mã nguồn (`useAutosaveDraft`); mất mạng không mất bài; hết giờ tự nộp
- [ ] Trắc nghiệm: xáo câu + đáp án theo hạt giống `attempt_id`; một lần nộp
- [ ] Code: trình soạn thảo mã (textarea có đánh số dòng + tab; **không thêm thư viện editor** trừ khi PM duyệt), chọn C / C++, `Chạy thử` test mẫu, `Nộp bài` một lần mỗi bài code (hoặc nộp lại tới hạn — PM chốt ở spec, mặc định một lần cuối cùng tính điểm)
- [ ] Màn hình dùng được ở 375 px cho trắc nghiệm; code yêu cầu ≥ 1024 px và nói rõ

**L5. Liêm chính** (D56)
- [ ] Khoá chat AI: khi có `exam_attempt` đang mở, Redis `ep:exam_lock:<user_id>` (TTL = thời gian còn lại); P3 (chat) phải tôn trọng khoá này — PE viết sẵn hàm kiểm và test
- [ ] Ghi log rời tab / dán (`exam_events`), chỉ giảng viên xem; ghi rõ trên giao diện đây không phải giám thị, không tự trừ điểm
- [ ] So độ giống mã nguồn giữa các bài nộp sau khi bài đóng (chuẩn hoá token, bỏ tên biến / chú thích; phương pháp theo research); báo cặp nghi vấn + mức giống cho giảng viên, không tự kết luận

**L6. Công bố và kết quả**
- [ ] Bài đóng (hết khung giờ của cả lớp) → job chấm xong → **tự công bố** (D56): sinh viên thấy điểm, đúng/sai từng câu trắc nghiệm, kết quả từng test mẫu; giảng viên có thể hoãn công bố hoặc chấm lại trước khi đóng
- [ ] `/exams/[id]/results` (giảng viên): bảng điểm lớp (`DataTable`, cursor), phân bố điểm, câu sai nhiều nhất, cặp nghi chép, xuất CSV; sửa điểm thủ công có lý do + `audit_log`
- [ ] Phúc khảo tối thiểu: sinh viên gửi yêu cầu kèm lý do → việc cho giảng viên ở "Hôm nay"

## Cổng nghiệm thu
```bash
cd backend-go && go test -race ./internal/exam/... ./internal/judge/...
cd backend-go && go test ./internal/judge -run TestSandboxAttacks -v        # vòng lặp vô hạn, fork bomb, đọc file hệ thống, socket, output 1 GB, cấp phát 2 GB → bị chặn
cd backend-go && go test ./internal/exam -run TestScoringDecimal -v         # tính điểm trọng số bằng decimal, khớp bảng tính tay
cd backend-go && go test ./internal/exam -run TestNoAnswerLeak -v           # SV không lấy được đáp án / test ẩn / nhật ký liêm chính qua mọi API
cd backend-go && go test ./internal/contract/...
pnpm -C frontend exec playwright test exam.spec.ts                          # làm trắc nghiệm + code, mất mạng giữa chừng, hết giờ tự nộp, xem điểm sau khi đóng
k6 run benchmarks/load/exam-submit.js                                       # 60 SV nộp code trong 5 phút: chấm xong p95 ≤ 60 s, chat INTERACTIVE không chậm
```

## Bạn tự kiểm
- Soạn một bài code có 5 test ẩn, chạy lời giải mẫu → 100 %; nộp lời giải sai một nhánh → điểm đúng tỉ lệ test đạt.
- Làm thử bằng tài khoản sinh viên trên điện thoại (trắc nghiệm) và máy tính (code); tắt mạng 30 s giữa chừng → không mất bài.
- Trong giờ thi hỏi chat AI về nội dung → bị từ chối; hết giờ → hỏi lại được.
- Sau khi bài đóng: sinh viên thấy điểm, không thấy test ẩn; giảng viên thấy cặp nghi chép.

## Ghi cho luận văn
Kiến trúc chấm code cách ly (sandbox, hàng đợi, giới hạn tài nguyên), bảng kết quả kiểm thử tấn công, số đo thông lượng chấm, vì sao điểm thi không qua LLM.
