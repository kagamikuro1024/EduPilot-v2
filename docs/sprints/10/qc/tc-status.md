# Trạng thái pha 2 theo từng TC — sprint 10 (phần không cần stack)
Cập nhật 2026-11-14, nhánh `sprint/10-practice` ở `345c8866`. Chỉ các TC chạy được **không cần stack** có kết quả; mọi TC khác «chờ stack». Chưa chạy e2e / k6 / Lighthouse / axe. Chi tiết: `report-no-stack.md`.

| Trạng thái | Số TC |
| --- | --- |
| chờ stack | 214 |
| PASS một phần (mức Go); phần API/E2E chờ stack | 15 |
| PASS (không stack) | 12 |
| chờ quyết định PM | 4 |
| NGOÀI PHẠM VI | 1 |
| FAIL (Thấp) | 1 |
| **Tổng** | 247 |

| TC | Trạng thái | Ghi chú |
| --- | --- | --- |
| TC-01-01 | chờ stack | Đã đọc `00019_practice.sql`: tên bảng, enum, FK phức hợp, CHECK, UNIQUE từng phần, chỉ mục khớp TC-01-02; không có `practice_exams`. Chạy `goose` thật chờ stack |
| TC-01-02 | chờ stack | Đọc SQL khớp (xem report mục 4); `\d` trên DB thật chờ stack |
| TC-01-03 | chờ stack |  |
| TC-01-04 | chờ stack |  |
| TC-01-05 | PASS (không stack) | `sqlc diff` mã 0 |
| TC-01-06 | chờ stack |  |
| TC-01-07 | PASS (không stack) | `TestGradeShort` 32 ca (≥ 30), kể cả «Hà Nội»/«ha noi», NFD, nhiều dấu câu cuối |
| TC-01-08 | chờ stack |  |
| TC-01-09 | chờ stack |  |
| TC-01-10 | PASS (không stack) | `go test -race ./internal/quiz/...` ok |
| TC-01-11 | PASS (không stack) | `go list -deps ./internal/quiz` 0 gói `internal/llm`; không `float64` trong mã không test; `TestNoLLMImport` PASS |
| TC-01-12 | chờ stack |  |
| TC-01-13 | chờ stack |  |
| TC-01-14 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestPracticeStudentDTOAllowlist` PASS; quét đen API chờ stack |
| TC-01-15 | PASS (không stack) | Thêm trường `answer_key` vào `ItemView` ở bản sao → test **FAIL** đúng chỗ; đã khôi phục tệp (sạch `git status`) |
| TC-01-16 | chờ stack |  |
| TC-01-P1 | chờ stack |  |
| TC-01-P2 | chờ stack |  |
| TC-01-P3 | chờ stack |  |
| TC-01-P4 | chờ stack |  |
| TC-02-01 | chờ stack |  |
| TC-02-02 | chờ stack |  |
| TC-02-03 | chờ stack |  |
| TC-02-04 | PASS (không stack) | `TestExtractQuestions` PASS (đề 24 câu) |
| TC-02-05 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestExtractQuestions`; fixture `dap-an-mau.pdf` đối chiếu `de-mau.json` 24/24 khớp (mục 5) |
| TC-02-06 | chờ stack |  |
| TC-02-07 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestDemoFixturesExtract`/`ValidSchemas` PASS; bản ghi `loi-cau`, `loi-schema` có trong `fake_llm.json` |
| TC-02-08 | chờ stack |  |
| TC-02-09 | chờ stack |  |
| TC-02-10 | chờ stack |  |
| TC-02-11 | chờ stack |  |
| TC-02-12 | chờ stack |  |
| TC-02-13 | chờ stack |  |
| TC-02-14 | chờ stack | Cần bật lỗi `OVERLOADED` của provider giả trên stack (Q-S10-29) |
| TC-02-15 | chờ stack |  |
| TC-02-16 | chờ stack |  |
| TC-02-17 | chờ stack |  |
| TC-02-18 | chờ stack |  |
| TC-02-19 | chờ quyết định PM | Trần 5 việc/lớp/giờ chưa có trong mã; không có AC/FR/openapi 429 cho `extract` (Q-S10-27) |
| TC-02-20 | chờ quyết định PM | Trần ≤ 60 trang chưa có trong mã (Q-S10-27) |
| TC-02-P1 | chờ stack |  |
| TC-02-P2 | chờ stack |  |
| TC-02-P3 | chờ stack |  |
| TC-02-P4 | chờ stack |  |
| TC-02-P5 | chờ stack |  |
| TC-03-01 | chờ stack |  |
| TC-03-02 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestGenerateCitations` PASS |
| TC-03-03 | chờ stack |  |
| TC-03-04 | chờ stack |  |
| TC-03-05 | chờ stack |  |
| TC-03-06 | chờ stack |  |
| TC-03-07 | chờ stack |  |
| TC-03-08 | chờ stack |  |
| TC-03-09 | chờ stack |  |
| TC-03-10 | chờ stack |  |
| TC-03-11 | chờ stack |  |
| TC-03-12 | chờ stack |  |
| TC-03-13 | chờ quyết định PM | Như TC-02-19 (Q-S10-27) |
| TC-04-01 | chờ stack |  |
| TC-04-02 | chờ stack |  |
| TC-04-03 | chờ stack |  |
| TC-04-04 | chờ stack |  |
| TC-04-05 | chờ stack |  |
| TC-04-06 | chờ stack |  |
| TC-04-07 | chờ stack |  |
| TC-04-08 | chờ stack |  |
| TC-04-09 | chờ stack |  |
| TC-04-10 | chờ stack |  |
| TC-04-11 | chờ stack |  |
| TC-04-12 | chờ stack |  |
| TC-04-13 | chờ stack |  |
| TC-04-14 | chờ stack |  |
| TC-04-15 | chờ stack |  |
| TC-04-16 | chờ stack |  |
| TC-04-17 | chờ stack |  |
| TC-04-P1 | chờ stack |  |
| TC-04-P2 | chờ stack |  |
| TC-04-P3 | chờ stack |  |
| TC-04-P4 | chờ stack |  |
| TC-04-P5 | chờ stack |  |
| TC-05-01 | PASS (không stack) | `TestBuildMatrix` PASS |
| TC-05-02 | chờ stack |  |
| TC-05-03 | chờ stack |  |
| TC-05-04 | chờ stack |  |
| TC-05-05 | chờ stack |  |
| TC-05-06 | chờ stack |  |
| TC-05-07 | chờ stack |  |
| TC-05-08 | chờ stack |  |
| TC-05-09 | chờ stack |  |
| TC-05-10 | chờ stack |  |
| TC-05-11 | chờ stack |  |
| TC-05-12 | chờ stack |  |
| TC-05-13 | chờ stack |  |
| TC-05-14 | chờ stack |  |
| TC-05-15 | chờ stack |  |
| TC-05-16 | chờ stack |  |
| TC-05-17 | chờ stack |  |
| TC-05-18 | chờ stack |  |
| TC-06-01 | chờ stack |  |
| TC-06-02 | chờ stack |  |
| TC-06-03 | chờ stack |  |
| TC-06-04 | chờ stack |  |
| TC-06-05 | chờ stack |  |
| TC-06-06 | chờ stack |  |
| TC-06-07 | chờ stack |  |
| TC-06-08 | chờ stack |  |
| TC-06-09 | chờ stack |  |
| TC-06-10 | chờ stack |  |
| TC-06-11 | chờ stack |  |
| TC-06-12 | chờ stack |  |
| TC-06-13 | chờ stack |  |
| TC-06-14 | chờ stack |  |
| TC-06-15 | chờ stack |  |
| TC-06-16 | chờ stack |  |
| TC-06-17 | chờ stack |  |
| TC-06-18 | chờ stack |  |
| TC-06-19 | chờ stack |  |
| TC-06-20 | chờ stack |  |
| TC-06-21 | chờ stack |  |
| TC-06-22 | chờ stack |  |
| TC-06-23 | chờ stack |  |
| TC-06-24 | chờ stack |  |
| TC-06-25 | chờ stack |  |
| TC-06-25b | chờ stack |  |
| TC-06-26 | chờ stack |  |
| TC-07-01 | chờ stack |  |
| TC-07-02 | chờ stack |  |
| TC-07-03 | chờ stack |  |
| TC-07-04 | chờ stack |  |
| TC-07-05 | chờ stack |  |
| TC-07-06 | chờ stack |  |
| TC-07-07 | chờ stack |  |
| TC-07-08 | chờ stack |  |
| TC-07-09 | chờ stack |  |
| TC-07-10 | chờ stack |  |
| TC-07-11 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestMockOfflineTolerance` PASS (Q13: `client_ts ≤ hạn` nhận) |
| TC-07-11b | PASS một phần (mức Go); phần API/E2E chờ stack | `TestMockOfflineTolerance` PASS (`client_ts > hạn` bỏ + `notes`) |
| TC-07-11c | PASS một phần (mức Go); phần API/E2E chờ stack | `TestMockOfflineTolerance` PASS (> 60 s → `409 ATTEMPT_CLOSED`) |
| TC-07-12 | chờ stack |  |
| TC-07-13 | chờ stack |  |
| TC-07-14 | chờ stack |  |
| TC-07-15 | chờ stack |  |
| TC-07-16 | chờ stack |  |
| TC-07-16b | chờ stack |  |
| TC-07-17 | chờ stack |  |
| TC-07-18 | chờ stack |  |
| TC-07-19 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestMockOfflineTolerance`, `TestMockFlow` PASS |
| TC-07-19b | PASS một phần (mức Go); phần API/E2E chờ stack | `TestMockOfflineTolerance` PASS |
| TC-07-20 | chờ stack |  |
| TC-07-21 | chờ stack |  |
| TC-07-22 | chờ stack |  |
| TC-07-23 | chờ stack |  |
| TC-07-24 | chờ stack |  |
| TC-07-25 | chờ stack |  |
| TC-07-26 | chờ stack |  |
| TC-08-01 | chờ stack |  |
| TC-08-02 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestEssayFeedback` PASS |
| TC-08-03 | chờ stack |  |
| TC-08-04 | chờ stack |  |
| TC-08-05 | chờ stack |  |
| TC-08-06 | chờ stack |  |
| TC-08-07 | chờ stack |  |
| TC-08-08 | NGOÀI PHẠM VI | P-10-7 Q16; dev xác nhận không có cách bật (handoff) |
| TC-08-09 | chờ stack |  |
| TC-08-10 | chờ stack |  |
| TC-08-11 | chờ stack |  |
| TC-08-12 | chờ stack |  |
| TC-08-13 | chờ stack |  |
| TC-08-14 | chờ stack |  |
| TC-09-01 | chờ stack |  |
| TC-09-02 | chờ stack |  |
| TC-09-03 | chờ stack |  |
| TC-09-04 | chờ stack |  |
| TC-09-05 | chờ stack |  |
| TC-09-06 | chờ stack |  |
| TC-09-07 | chờ stack |  |
| TC-09-08 | chờ stack |  |
| TC-09-09 | chờ stack |  |
| TC-09-10 | PASS (không stack) | `TestWrongAnswerSignalsNoIdentity` PASS |
| TC-09-11 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestWrongAnswerSignalsNoIdentity` PASS (k ≥ 3, không định danh) |
| TC-09-12 | chờ stack |  |
| TC-09-13 | chờ stack |  |
| TC-09-14 | chờ stack |  |
| TC-09-15 | chờ stack |  |
| TC-09-16 | chờ stack |  |
| TC-09-17 | chờ stack |  |
| TC-09-18 | chờ stack |  |
| TC-09-19 | chờ stack |  |
| TC-09-20 | chờ stack |  |
| TC-09-21 | chờ stack |  |
| TC-10-01 | PASS (không stack) | `TestPracticeNoExamLeak` PASS |
| TC-10-02 | chờ stack |  |
| TC-10-03 | PASS (không stack) | `TestPracticeNoAnswerLeak` PASS |
| TC-10-04 | chờ stack |  |
| TC-10-05 | chờ stack |  |
| TC-10-06 | chờ stack |  |
| TC-10-07 | chờ stack |  |
| TC-10-08 | chờ stack |  |
| TC-10-09 | chờ stack |  |
| TC-10-10 | chờ stack |  |
| TC-10-11 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestLibraryToolHidesExamPaperWhileLocked`, `TestSearchLibraryTool`, `TestSearchLibraryExcludesAnswerKey`, `TestSearchLibraryNoIdentityParam` PASS |
| TC-10-12 | chờ stack |  |
| TC-10-13 | chờ stack |  |
| TC-10-14 | chờ stack |  |
| TC-10-15 | chờ stack |  |
| TC-10-16 | chờ stack |  |
| TC-10-17 | chờ stack |  |
| TC-11-01 | chờ stack |  |
| TC-11-02 | chờ stack |  |
| TC-11-03 | chờ stack |  |
| TC-11-04 | chờ stack |  |
| TC-11-05 | chờ stack |  |
| TC-11-06 | chờ stack |  |
| TC-11-07 | chờ stack |  |
| TC-11-08 | chờ stack |  |
| TC-11-09 | chờ stack |  |
| TC-11-10 | chờ stack |  |
| TC-11-11 | chờ stack |  |
| TC-11-12 | chờ stack |  |
| TC-11-13 | chờ stack |  |
| TC-11-14 | PASS một phần (mức Go); phần API/E2E chờ stack | `TestFormsImportNoLLM`, `TestFormsImportScoreDecimal` PASS; `llm_audit` trên stack chờ stack |
| TC-11-15 | chờ stack |  |
| TC-11-16 | chờ stack |  |
| TC-11-17 | chờ stack |  |
| TC-11-18 | chờ stack |  |
| TC-12-01 | chờ stack |  |
| TC-12-02 | chờ stack | Handoff ghi 82 `APPROVED` (US: 80) — Q-S10-28 |
| TC-12-03 | chờ stack |  |
| TC-12-04 | chờ stack |  |
| TC-12-05 | chờ stack | `check-practice-seed.mjs` cần seed trên stack |
| TC-12-06 | chờ stack |  |
| TC-12-07 | PASS (không stack) | `go test -race ./internal/quiz/...` ok |
| TC-12-08 | PASS (không stack) | 5 test PASS, 0 `SKIP` trong đầu ra `-v` |
| TC-12-09 | chờ stack |  |
| TC-12-10 | chờ stack |  |
| TC-12-11 | chờ stack |  |
| TC-12-12 | FAIL (Thấp) | `grep @/mock/practice frontend/src` = 0 và `mock/practice.ts` đã xoá (đạt); **`frontend/src/mock/questions.ts` (235 dòng) còn, không ai import** — AC4 nói xoá cùng đợt nếu không còn người dùng |
| TC-12-13 | chờ stack |  |
| TC-12-14 | PASS một phần (mức Go); phần API/E2E chờ stack | 16/16 thao tác SRS §6 có trong `openapi.yaml` (304 thao tác); `go test -tags testroutes ./internal/contract` ok. Lệch: `explain` thiếu `429` (SRS #11); `extract`/`generate` không khai `429` (Q-S10-27) |
| TC-12-15 | chờ stack |  |
| TC-12-16 | chờ stack |  |
| TC-12-17 | chờ stack |  |
| TC-12-18 | chờ stack |  |
| TC-12-19 | PASS một phần (mức Go); phần API/E2E chờ stack | `bash scripts/ui-antipatterns.sh` sạch; axe chờ stack |
| TC-12-20 | chờ stack |  |
| TC-12-21 | chờ stack |  |
| TC-12-22 | chờ quyết định PM | Không `t.Skip`, không secret, không `float64` điểm, không `fetch(` trần, không thư viện mới. **Có sửa test có sẵn:** `today/staff_test.go` nới ngưỡng truy vấn «sinh viên» 6 → 7 (commit `4f44403f`, không số proposal) — xem F1 trong report |
