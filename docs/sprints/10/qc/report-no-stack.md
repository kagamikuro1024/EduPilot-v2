# QC sprint 10 — pha 2, phần KHÔNG cần stack (2026-11-14)
Nhánh `sprint/10-practice` ở `345c8866` (dev: «Q13 dung sai 60 s»). Bản sao của QC: `~/Documents/TA_Agent_qc5`. **Chưa dựng `ep-qc`**; không chạy e2e / k6 / Lighthouse / axe (lệnh PM). Trạng thái từng TC: `tc-status.md` (250 TC; sau P-10-8 xem bảng đầu `tc-status.md`).

**Kết luận tạm (chưa phải kết luận story): không có FAIL mức Trung bình / Cao ở phần kiểm được. Một FAIL Thấp (TC-12-12), bốn mục cần PM quyết định (F1, F3 + hai câu hỏi), phần còn lại chờ stack.** Các story chưa thể kết luận PASS/FAIL vì AC chính (API / E2E / hiệu năng) cần stack.

## 1. Môi trường và thời điểm chạy (để dev loại khỏi phép đo tải nếu cần)
`go test` dùng testcontainers (Postgres + Redis) → thêm tải ngắn lên Colima trong lúc stack `ep-t1` của dev chạy. Mọi lệnh chạy `nice -n 19`, `-p 1 -parallel 1`.
| Lệnh | Giờ (2026-11-14) | Thời gian |
| --- | --- | --- |
| `go vet` (5 gói) + `go test -race ./internal/quiz/...` | 00:53:32 – 00:53:45 | 13 s |
| `go test ./internal/practice -v` | 00:53:57 – 00:54:15 | 18 s |
| `./internal/library -run SearchLibrary`, `./internal/agent/...`, `./internal/exam/...`, `./internal/contract` (`-tags testroutes`) | 00:54:27 – 00:57:12 | 165 s (exam 108 s) |
| `sqlc diff`, `go vet ./...`, mutation test allowlist | ≈ 01:00 | < 30 s |
Máy: MacBook Apple Silicon, Colima 6 CPU / 10 GB; Go 1.27.1.

## 2. Kết quả `go test`
| Gói / test | Kết quả |
| --- | --- |
| `internal/quiz` (`TestGradeShort` 32 ca, `TestNormalizeShort`, `TestBuildMatrix`, `TestNoLLMImport`, `TestQuizGrade`, `TestQuizInvalidOption`, `TestQuizPartialFormula`, `TestShuffleStable`, `TestGradeShortBadKey`), `-race` | ok |
| `internal/practice` — 28 test, gồm `TestExtractQuestions`, `TestGenerateCitations`, `TestPracticeNoAnswerLeak`, `TestPracticeNoExamLeak`, `TestPracticeStudentDTOAllowlist`, `TestWrongAnswerSignalsNoIdentity`, `TestMockOfflineTolerance`, `TestMockFlow`, `TestExplainAnswer`, `TestExplainMockAfterSubmit`, `TestEssayFeedback`, `TestTodayWeakTopic`, 6 test `Forms*`, 4 test `DemoFixtures*`, `TestLibraryToolHidesExamPaperWhileLocked` | 28/28 PASS, **0 `SKIP`** |
| `internal/library -run SearchLibrary` (Q22: `TestSearchLibraryTool`, `…ExcludesAnswerKey`, `…NoIdentityParam`) | PASS |
| `internal/agent/...`, `internal/exam/...`, `internal/exam/similarity` | ok |
| `internal/contract` (`-tags testroutes`) | ok (31 s) |
| `go vet ./...`, `sqlc diff` | mã 0 |
Cả năm test của TC-12-08 (`TestExtractQuestions`, `TestBuildMatrix`, `TestPracticeNoAnswerLeak`, `TestPracticeNoExamLeak`, `TestWrongAnswerSignalsNoIdentity`) PASS và chạy thật.
**Kiểm độ nhạy (TC-01-15):** thêm trường `AnswerKey string \`json:"answer_key"\`` vào `ItemView` ở bản sao → `TestPracticeStudentDTOAllowlist` FAIL («ItemView … feedback»); đã khôi phục tệp, `git status` sạch. Test có răng.
Q20: QC có thể thêm test vào gói `practice` (`package practice_test`, dùng `newRig`) cho AC4 độc lập; chưa viết (chờ pha 2 đầy đủ).

## 3. Hợp đồng `openapi.yaml` so với TC / SRS §6
16/16 thao tác của SRS §6 có mặt (`extractQuestions`, `generateQuestions`, `bulkReviewQuestions`, `questionGenStats`, `getPracticeOverview`, `startPractice`, `getPracticeAttempt`, `savePracticeAnswer`, `patchPracticeAttempt`, `submitPracticeAttempt`, `explainPracticeAnswer`, `practiceEssayFeedback`, `listPracticeHistory`, `getPracticeWeakness`, `previewFormsImport`, `importFormsResults`), cộng `GET /jobs/{id}` (Q5). Mã trạng thái khớp TC (`409 NOT_ENOUGH_QUESTIONS`, `ATTEMPT_CLOSED`, `TAB_TAKEN_OVER`… ở `409`; `422` cho `TOO_LONG`, `INVALID_*`; `X-Practice-Tab` ở `start`/`GET`/`PUT`/`PATCH`; `Idempotency-Key` ở `start`, `submit`, `feedback`, `extract`, `generate`, `bulk`, `forms-import`, không ở `explain`).
| # | Lệch / ghi chú | Mức | Liên quan |
| --- | --- | --- | --- |
| H1 | `explainPracticeAnswer` **không khai `429`** dù SRS §6 #11 có `429` | Thấp (hợp đồng lệch SRS) | TC-06-11, 12-14 |
| H2 | `extractQuestions`, `generateQuestions` không khai `429`; mã không có trần 5 việc/lớp/giờ, ≤ 60 trang (Q-P9-12). QUESTIONS ghi «đã áp dụng trong spec» nhưng US/SRS không có AC/FR/dòng API | Cần PM/BA quyết định (Q-S10-27) | TC-02-19, 02-20, 03-13 |
| H3 | `PracticeResult.score` **bắt buộc** cho mọi lượt, kể cả `TOPIC`; US-P9-06 AC7: «không điểm» | Kiểm UI chờ stack (TC-06-16: TOPIC không hiện điểm) | TC-06-16 |
| H4 | `PracticeEssayFeedback.injected` (boolean) trả về cho SV; P-10-7 Q15: không hiển thị cờ cho SV | Kiểm UI chờ stack (TC-08-06) | TC-08-06 |
| H5 | `PracticeStartRequest.matrix` là `[object]` không mô tả; kết quả lịch sử / quan trọng là `object` ẩn danh | Thấp (hợp đồng lỏng); TC-05-02, 05-04 cần ví dụ ma trận | TC-05 |
| H6 | `PracticeGenStats` có thêm `pending` (spec: `{unchanged, edited, rejected}`) | Thông tin | TC-03-11 |
| H7 | `FormsImportRequest` + `mapping` + `item_id` ở `preview` trả kết quả đã chấm (AC5), không chỉ 20 dòng đầu như SRS #15 | Thông tin (dev nêu ở handoff) | TC-11-08 |

## 4. Migration `00019_practice.sql` (đọc, đối chiếu TC-01-02)
Khớp SRS §5: enum `practice_mode`, `practice_status`; `practice_attempts` (FK `(course_id, student_id)→enrollments`, CHECK `(mode='MOCK')=(deadline_at IS NOT NULL)`, UNIQUE từng phần `(course_id, student_id, mode, COALESCE(topic,'')) WHERE status='IN_PROGRESS'`, chỉ mục `(course_id, student_id, started_at DESC, id DESC)` và `(status, deadline_at) WHERE IN_PROGRESS AND MOCK`); `practice_items` (FK phức hợp tới `practice_attempts` và `question_bank`, không CASCADE về ngân hàng; UNIQUE `(attempt_id, position)`; thêm cột `question` snapshot — dev nêu mục 1 handoff); `practice_answers` (UNIQUE `item_id`); `forms_import_batches` UNIQUE `(course_id, item_id, file_hash)`, `forms_import_rows`. **Không có `practice_exams`** (đúng US v1.1). Chạy `goose up/down/up` thật: chờ stack.

## 5. Fixture đề mẫu và `fake_llm.json` (Q4, Q6)
PDF đọc bằng PDFKit (swift), đối chiếu `seed/practice/de-mau.json`:
| Tệp | Kết quả |
| --- | --- |
| `de-mau-trac-nghiem.pdf` | 4 trang × 6 câu = **24 câu** A–D, tiêu đề «ĐỀ MẪU LUYỆN TẬP — MẠNG MÁY TÍNH», **không** có dòng đáp án trong đề |
| `dap-an-mau.pdf` | 24 dòng «N. X»; **khớp 24/24** với `de-mau.json` (phân bố A/B/C/D = 6/6/6/6) |
| `dap-an-thieu.pdf` | 22 dòng; thiếu câu **22 và 24**; 22 dòng còn lại khớp |
| `de-mau-loi-cau.pdf`, `de-mau-loi-schema.pdf` | Có, tiêu đề khớp chuỗi `match` của bản ghi |
`seed/demo/fake_llm.json`: có `practice-extract-de-mau` (24 câu, **không** `correct_label` → đáp án chỉ lấy từ tài liệu đáp án; thiết kế đúng AC3), `…-loi-cau`, `…-loi-schema`, `practice-generate-mau`, `…-loi`, `…-loi-schema`, `practice-feedback-sql`, `practice-feedback-ecb-cbc`. **Không có bản ghi cho `explain`** (giải thích đáp án) và không có cách bật `OVERLOADED` ghi ở handoff (Q-S10-29).
Hệ quả cho TC: TC-02-01/02 phải trích **kèm** `answer_key_document_id` (không kèm thì cả 24 câu vào «bỏ qua `NO_ANSWER`», 0 câu lưu: đó là TC-02-02b). Đã cập nhật TC (lịch sử ở `tc-US-P9-02.md`). Cảnh báo của dev: chạy `de-mau-loi-schema` nhiều lần mở cầu dao LLM → chạy cuối cùng hoặc `down -v` sau đó.
Đường dẫn: US-P9-02 AC2 ghi `backend-go/internal/practice/testdata/`; thực tế `seed/practice/` (Thấp, BA vá đường dẫn).

## 6. Kiểm diff không cần stack (TC-12-12, TC-12-22)
| Kiểm | Kết quả |
| --- | --- |
| `grep @/mock/practice frontend/src` | 0; `mock/practice.ts` đã xoá (**đạt**) |
| `frontend/src/mock/questions.ts` | **còn** (235 dòng), **không có import nào** → US-P9-12 AC4 «xoá cùng đợt nếu không còn người dùng» → **FAIL (Thấp)** (F2) |
| `fetch(` trần ở `features/practice`, `features/questions`, `FormsImportDialog` | 0 |
| `go.mod`, `frontend/package.json` | không đổi (không thư viện lạ) |
| `float64` mới trong `internal/practice`, `internal/quiz` (mã không test) | 0 |
| `t.Skip` / `SKIP` | 0 |
| `bash scripts/ui-antipatterns.sh` | sạch |
| Test có sẵn bị sửa | Xem F1; còn lại hợp lệ theo spec: `exam/question_test.go` (`TestTypeNotSupportedYet` → `TestShortEssayTypes`, theo P-10-5), `db/migrations_test.go` (17 → 19), `contract_test.go` (288 → 304 thao tác, thêm 2 dòng bảng route, không xoá dòng nào), `integration/course_guard_test.go` (thêm truy vấn mới vào danh sách), `panels.spec.ts` (`/practice/at-symmetric` → `/practice/history` vì mock bị xoá) |

## 7. Phát hiện cần PM / BA / dev
| # | Mức | Nội dung | Đề nghị |
| --- | --- | --- | --- |
| F1 | **Cần PM quyết định (theo luật QC: sửa test để xanh = báo PM)** | `backend-go/internal/today/staff_test.go` (commit `4f44403f`): ngưỡng «≤ N truy vấn kể cả guard (#28)» của Hôm nay **sinh viên nới 6 → 7**; chú thích: «6 + 1 truy vấn gộp luyện đề». Không có số proposal. Lý do hợp lý (nguồn mới `PRACTICE_WEAK_TOPIC`) nhưng đây là test hiệu năng (luật 13: không N+1) | PM xác nhận bằng proposal, hoặc dev gộp truy vấn luyện đề vào truy vấn hiện có để giữ 6. QC chưa kết luận FAIL |
| F2 | Thấp | `frontend/src/mock/questions.ts` còn, không ai dùng (US-P9-12 AC4, TC-12-12) | Dev xoá |
| F3 | Cần PM/BA quyết định | Q-P9-12 (trần ≤ 5 việc/lớp/giờ, ≤ 60 trang, `429 RATE_LIMITED`) không cài, không có trong US/SRS/openapi | PM/BA: bắt buộc hay bỏ? Nếu bắt buộc → AC + openapi + code |
| F4 | Thấp | Hợp đồng `explain` thiếu `429` so với SRS #11 (H1) | BA/dev đồng bộ |
| F5 | Thông tin | Forms: không có thao tác «gán tay» dòng Chưa khớp (dev mục 7); US-P9-11 AC2 nói «Chưa khớp gán tay». Dev nói giảng viên nhập tay ở ô điểm Sổ điểm | BA xác nhận AC2 (Q-S10-30) |
| F6 | Thông tin | Bulk «Loại» hoãn 5 s để `Hoàn tác`; đóng tab trong 5 s thì lệnh **mất** (dev mục 5) | Q-S10-31; TC-04-06 chờ stack kiểm hành vi |
| F7 | Thông tin | Seed ghi 82 `APPROVED` (US-P9-12 AC1: 80) | Q-S10-28 |

## 8. TC đã cập nhật theo fixture
TC-02-01, TC-02-02 (trích kèm đáp án), thêm TC-02-02b (không kèm đáp án → 0 câu lưu, 24 `NO_ANSWER`); TC-02-14, TC-06-20, TC-08-11 ghi cách bật lỗi LLM: chờ trả lời Q-S10-29; ghi chú «chờ stack» ở `tc-status.md`.

## 9. Việc còn lại (cần stack, đúng thứ tự `tc-00-chung.md`)
Dựng `ep-qc` với `docker-compose.test.yml` (route thử `_test/llm/fake` để bật lỗi LLM; Q-S10-29), `--profile ingest`, seed (`STEPS=20`), `check-practice-seed.mjs`; API (curl) cho TC trích đề, dựng đề, thi thử, Forms; E2E `qc-practice-*.spec.ts` 1440 + 375; k6 và Lighthouse ở cuối khi PM báo hết đo tải.

## 10. Quyết định PM (P-10-8, 2026-11-14) và ảnh hưởng
| Mục | Quyết định | Việc / TC |
| --- | --- | --- |
| F1 | Chấp nhận ngưỡng Hôm nay SV 6 → 7 (1 truy vấn gộp cố định, không N+1); commit test phải ghi số P-10-8 | TC-12-22: QC kiểm commit ghi số P-10-8 khi dev sửa |
| F2 | Dev xoá `frontend/src/mock/questions.ts` | TC-12-12 chờ dev |
| F3 / Q27 | **Bắt buộc**: `extract` ≤ 60 trang, ≤ 5 việc/lớp/giờ; `generate` ≤ 5 việc/lớp/giờ; `429 RATE_LIMITED` + `Retry-After`; BA thêm AC/FR; dev cài + openapi | TC-02-19, 02-20, 03-13 chờ dev (hiện FAIL nếu chạy) |
| F4 | Dev thêm `429` cho `explain` vào openapi | TC-12-14 |
| Q28 | ≥ 80 đạt | TC-12-02 |
| Q29 | Dev thêm bản ghi `explain` vào `fake_llm.json`, nêu cách bật `OVERLOADED` ở handoff | TC-02-14, 06-20, 08-11 chờ handoff + stack |
| Q30 | Đủ: hàng Chưa khớp có liên kết «Nhập ở Sổ điểm», không dòng giải thích | Thêm TC-11-19 |
| Q31 | Không chấp nhận mất lệnh: rời trang / đóng tab trong 5 s thì gửi ngay (`fetch keepalive` lúc `pagehide`); dev sửa | Thêm TC-04-06b |
F5 (Forms gán tay), F6 (Loại hoãn 5 s) và F7 (seed 82) đã khép bằng Q30, Q31, Q28. Không còn mục chờ PM ở phần không stack.
