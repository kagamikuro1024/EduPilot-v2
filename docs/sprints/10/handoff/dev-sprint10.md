# DEV handoff — Sprint 10 (P9 Luyện đề), US-P9-01 … P9-12
Nhánh: `sprint/10-practice`. Commit cuối: xem `git log -1`. Spec: `docs/specs/FEAT-practice/{SRS,US,QUESTIONS}.md` (APPROVED v1.0); đề xuất đổi tài liệu nền P-10-1…6 đã áp (PM chấp nhận).

**Chưa chạy trên stack compose thật** (stack test nhường QC): chưa có số đo trên gateway thật, chưa chạy seed, chưa chạy `docling` / judge, chưa chạy k6, Lighthouse, axe. Mọi test dưới đây là Go (Postgres + Redis testcontainers) và Playwright với gateway giả; ngoài ra có một lượt thử tay bằng gateway native + frontend dev (luyện chủ đề, thi thử mất mạng / có mạng, nộp, 375 px).

## Đã làm (theo thứ tự lát dọc)
- **Quiz Engine** (`internal/quiz`): `SHORT` (`NormalizeShort`), `BuildMatrix` (ma trận chủ đề × độ khó, tránh câu đúng 30 ngày), `Shuffle` — code thuần, không import `internal/llm`. `exam.Validate` mở `SHORT` / `ESSAY` cho ngân hàng, vẫn chặn ở bài thi (`TYPE_NOT_SUPPORTED_IN_EXAM`).
- **Migration** `00019_practice` (`practice_attempts/items/answers`, `forms_import_batches/rows`); sqlc `practice.sql`, `forms.sql`.
- **`internal/practice`**: dựng đề (`start.go`), làm bài TOPIC (chấm ngay, khoá câu) / MOCK (`version` + `X-Practice-Tab`, đáp án chỉ lộ sau nộp, dung sai ngoại tuyến 60 s) (`attempt.go`), đóng lượt hết giờ (`job.go`, task `practice.close`), lịch sử / điểm yếu (`history.go`), giải thích SSE cache 24 h + nhận xét tự luận qua `grading.Feedback` (`llm.go`), trích đề / sinh câu có trích dẫn (`extract.go`, `generate.go`), duyệt hàng loạt (`bulk.go`), **nhập Forms** (`forms.go`).
- **REST**: 16 thao tác (#1–#16, `practice.Routes()`), `openapi.yaml`, contract test (304 thao tác), kịch bản mọi status.
- **Hôm nay**: `PRACTICE_WEAK_TOPIC` (bậc 85, STUDENT); outbox `practice.updated`.
- **Liêm chính**: `library.Service.Search` ẩn `EXAM_PAPER` khi sinh viên đang có lượt bài thi; `exam.Review` chặn duyệt câu EXTRACTED / GENERATED không trích dẫn (`CITATION_REQUIRED`).
- **Frontend**: `features/practice/*` (trang chủ, làm theo chủ đề, thi thử, kết quả, lịch sử), `/questions` (Trích từ đề, Sinh từ tài liệu, thanh duyệt hàng loạt, SHORT / ESSAY), **Sổ điểm → Nhập từ Forms** (`features/gradebook/FormsImportDialog.tsx`).
- **Seed**: `step19Practice` (`scripts/seed.mjs`, 82 APPROVED + 20 PENDING + Quiz 01 + ≈ 40 lượt) + `scripts/check-practice-seed.mjs`. **Chưa chạy.**
- **k6**: `benchmarks/load/practice-mock.js` (viết, chưa chạy).

## Nhập Forms (US-P9-11) — hành vi cần QC biết
- `uploads/presign` `purpose=forms_import` (`.xlsx` / `.csv` ≤ 5 MB, chỉ TEACHER). `POST …/forms-import/preview {upload_id}` → cột + 20 dòng đầu + gợi ý cột; **cùng đường đó với `mapping` + `item_id` + `answer_source`** → thêm bảng điểm đã chấm (AC5 "Xem trước kết quả"; không ghi). Cả hai đánh dấu tệp "đã xem trước" (Redis, 1 giờ); `POST …/forms-import` thiếu dấu → `409 PREVIEW_REQUIRED`.
- Khớp sinh viên: email **đã xác minh** hoặc MSSV của dòng `ROSTER` / `ADMIN` (MSSV `CODE` tự khai không mở dữ liệu). Cả email và MSSV cùng có mà trỏ hai người khác nhau → "Chưa khớp" `MISMATCH`. Trùng sinh viên → dòng có dấu thời gian mới nhất (không phân tích được dấu thời gian → dòng sau thắng).
- Chấm bằng `grade()` của `internal/practice` (= `quiz.Grade`), điểm = đúng × thang cột ÷ tổng, `decimal`, làm tròn 2. Đáp án không khớp lựa chọn nào → 0 và đếm "ô không khớp". Nhiều lựa chọn tách bằng `;` (cách Forms ghi).
- Ghi `grade_entries(source='QUIZ')` vào cột MANUAL đã chọn qua `GradeImportApply` (thêm tham số `source`; nhập XLSX truyền `IMPORT`), **không** đặt `published_at`; audit `forms_imports` (hash, số khớp / chưa khớp, không có email / MSSV). Cùng tệp (hash) vào cùng cột → trả lại lô cũ (`replayed: true`, vẫn `201`), không ghi gì.
- Test: `internal/practice/forms_test.go` (`TestFormsImportFlow`, `TestFormsMatchRules`, `TestFormsImportScoreDecimal`, `TestFormsImportNoLLM`, `TestFormsManualAnswerAndSemicolonCSV`, `TestFormsImportGuards`), kịch bản hợp đồng ở `scenarios_grade_test.go`, e2e `gradebook.spec.ts` ("forms import …").

## Lệnh QC chạy
```bash
cd backend-go && go vet ./... && go test -race ./...                 # gồm internal/practice, internal/contract (-tags testroutes cho contract)
cd backend-go && go test -tags testroutes ./internal/contract
pnpm -C frontend lint && pnpm -C frontend exec tsc --noEmit && bash scripts/ui-antipatterns.sh
pnpm -C frontend build:gate && pnpm -C frontend exec playwright test e2e/practice.spec.ts e2e/quiz-integrity.spec.ts e2e/questions-extract.spec.ts e2e/gradebook.spec.ts --project=desktop
node scripts/check-practice-seed.mjs          # sau khi `pnpm dev` + seed (STEPS=20)
k6 run -e BASE=https://localhost -e COURSE=<lớp> -e TOKENS="…" benchmarks/load/practice-mock.js
```

## Khác biệt so với spec / quyết định của dev (đề nghị PM / BA xác nhận)
1. **Lượt luyện chụp câu** (`practice_items.question` + `answer_key`) — SRS 5 chỉ ghi `answer_key`; thêm `question` để sửa câu giữa chừng không làm lệch lượt.
2. **Ngân hàng cho luyện đề** = câu `APPROVED`, chưa lưu trữ, và không thuộc bài thi chưa `PUBLISHED` (Q-P9-3).
3. **`practice.updated`** ghi `{user_id, student_id}` (không có `course_id` như FR-19) — chỉ để huỷ cache "Hôm nay" của sinh viên.
4. **FR-23** chỉ áp ở `library.Service.Search` (công cụ `search_library`); Chat và Threads đã khoá theo khoá bài thi sẵn có.
5. **Bulk "Loại"** hoãn 5 s để Hoàn tác; đóng tab trong 5 s thì lệnh Loại mất.
6. **Hôm nay "Ôn chủ đề"** ẩn 3 ngày sau khi luyện chủ đề đó nên seed không hiện ngay (API không lùi ngày được).
7. **Forms**: không có thao tác "gán tay" riêng cho dòng Chưa khớp (không có trong danh sách thao tác #15–#16): dòng chưa khớp hiện trong xem trước / kết quả (định danh đã che), giảng viên nhập tay ở ô điểm của Sổ điểm.
8. Sửa lỗi nhân đường: `MockRun` lặp vô hạn "Đang nộp…" khi máy chủ đã đóng lượt (`ATTEMPT_CLOSED`) mà hàng đợi còn câu chưa gửi — nay nộp lấy kết quả theo phần đã lưu (ca e2e "ATTEMPT_CLOSED").
9. `shared/shell/nav.ts`: `/practice` bỏ khỏi bảng "màn chưa có backend" (build `NEXT_PUBLIC_MOCK_SCREENS=0` hiện màn thật).

## AC tự đánh giá
| Story | Kết quả | Ghi chú |
| --- | --- | --- |
| P9-01 loại SHORT / ESSAY | ✓ | `TestGradeShort`, `exam.Validate` |
| P9-02 trích đề cũ | ✓ trên fake LLM | `TestExtractQuestions` (đề 24 câu); chưa chạy docling + LLM thật |
| P9-03 sinh có trích dẫn | ✓ trên fake LLM | `TestGenerateCitations`; E5 (tỷ lệ giữ nguyên / sửa / loại) cần chủ dự án duyệt 20 câu thật |
| P9-04 duyệt hàng loạt | ✓ | e2e `questions-extract.spec.ts` |
| P9-05 dựng đề | ✓ | `TestBuildMatrix`; "≤ 3 s" chưa đo trên stack thật |
| P9-06 luyện chủ đề | ✓ | `practice.spec.ts` |
| P9-07 thi thử | ✓ | `practice.spec.ts` (mất mạng, tải lại, `ATTEMPT_CLOSED`, `TAB_TAKEN_OVER`); k6 chưa chạy |
| P9-08 tự luận | ✓ | `grading.Feedback` không điểm, trích dẫn phải nguyên văn |
| P9-09 điểm yếu + Hôm nay | ✓ | `TestWrongAnswerSignalsNoIdentity` (k ≥ 3) |
| P9-10 liêm chính | ✓ | `TestPracticeNoAnswerLeak`, `TestPracticeNoExamLeak`, `TestLibraryToolHidesExamPaperWhileLocked`, `quiz-integrity.spec.ts` |
| P9-11 nhập Forms | ✓ AC1–AC12 (xem chú ý 7) | `forms_test.go` |
| P9-12 seed + cổng | ✗ chưa chạy | seed, `check-practice-seed.mjs`, k6, Lighthouse, axe, E5, "import một file Forms thật" (chủ dự án) |

## Nợ / cần hỏi
- Chạy seed + `check-practice-seed.mjs`, k6, Lighthouse 3 route, axe khi có stack; ghi số đo "MacBook Apple Silicon, Colima 6 CPU / 10 GB".
- Chủ dự án tự kiểm: duyệt 20 câu AI sinh (E5), nhập một tệp Forms thật (đặc biệt định dạng dấu thời gian của MS / Google Forms: dev hỗ trợ `YYYY-MM-DD HH:mm:ss`, `M/D/YY`, `M/D/YYYY`, `YYYY/MM/DD h:mm:ss A` — định dạng khác coi như không có dấu thời gian).
- `practice.updated` thiếu `course_id` (mục 3) — nếu PM muốn đúng FR-19 thì sửa SRS hoặc thêm trường.

## Trả lời câu hỏi QC (P-10-7)
| Q | Trả lời (hợp đồng ở `openapi.yaml`) |
| --- | --- |
| Q3 | `POST …/questions` (`QuestionInput`): `SHORT` → `accepted: [string]` (1–10 đáp án; không `options` / `correct`; thiếu → 422 `ANSWER_EMPTY`). `ESSAY` → `model_answer: string` và `rubric: string` (≤ 2.000 ký tự; thiếu → 422 `MODEL_ANSWER_REQUIRED` / `RUBRIC_REQUIRED`). Chi tiết câu trả về ở `answer_key`: SHORT `{accepted}`, ESSAY `{model_answer, rubric}`. **Spec trước đây thiếu ba trường này; đã bổ sung.** |
| Q4 | `seed/practice/` (dựng bằng `node scripts/make-practice-pdf.mjs` từ `de-mau.json`, đúng 24 câu, đáp án **tách** ở `dap-an-mau.pdf` dạng "N. X"): `de-mau-trac-nghiem.pdf` (4 trang, 6 câu / trang), `dap-an-mau.pdf` (đủ 24), `dap-an-thieu.pdf` (thiếu câu 22, 24), `de-mau-loi-cau.pdf`, `de-mau-loi-schema.pdf`. **Bảng đáp án đúng** = `de-mau.json` (`questions[].correct`, 0 = A); phân bố A/B/C/D = 6/6/6/6. Tải lên với loại `EXAM_PAPER` (đề) và `ANSWER_KEY` (đáp án). |
| Q5 | `GET /api/v1/jobs/{id}` → `result {created: [uuid], skipped: [{number, page, reason}]}`; `reason` ∈ `NO_ANSWER` (đề không có đáp án và không đưa bảng đáp án), `NO_ANSWER_MATCH` (bảng đáp án thiếu số câu), `DUPLICATE`, `INVALID`. Cùng nội dung ở SSE `job.progress` cuối. Sinh câu: `reason` ∈ `CITATION_REQUIRED`, `TYPE_NOT_ALLOWED`, `INVALID`, `DUPLICATE`, `EXTRA`. |
| Q6 | Bật bằng **nội dung tài liệu / chủ đề**, không có cờ: provider giả (DEMO_MODE, `LLM_DEMO_FILE=/seed/demo/fake_llm.json`) trả bản ghi khi lời nhắc chứa chuỗi `match`. Trích câu: đề có tiêu đề `ĐỀ MẪU LUYỆN TẬP — MẠNG MÁY TÍNH` → 24 câu hợp lệ; `ĐỀ MẪU LỖI CÂU — …` (không bảng đáp án) → 1 câu lưu, `INVALID` ở câu 2 (stem rỗng), 3 (1 phương án), 4 (nhãn đáp án `F` không tồn tại), `NO_ANSWER` ở câu 5; `ĐỀ MẪU LỖI SCHEMA — …` → job `FAILED`, `error.code = LLM_UNAVAILABLE`, không lưu câu nào (gateway loại đầu ra sai schema như lỗi nhà cung cấp; lặp nhiều lần sẽ mở cầu dao LLM của stack — `down -v` hoặc chờ nguội). Sinh câu: `count: 5`, loại mặc định, chủ đề `Mẫu sinh câu` → 5 câu có nguồn; `Mẫu sinh lỗi` → 1 câu lưu, `CITATION_REQUIRED` (2), `TYPE_NOT_ALLOWED` (3), `INVALID` (4), `DUPLICATE` (5), `EXTRA` (6); `Mẫu sinh sai schema` → `FAILED` `LLM_UNAVAILABLE`. Nhận xét tự luận: đề bài chứa "SQL injection" hoặc "ECB và CBC" (các câu ESSAY của seed) → 3 tiêu chí, không điểm. **Giải thích (stream)**: có bản ghi `practice-explain` (P-10-8 Q29; provider giả nay áp bản ghi demo dạng chuỗi cho cả luồng) — `Giải thích thêm` trả 3 câu cố định thay vì "Đây là câu trả lời giả lập cho: …". Kiểm các bản ghi: `go test ./internal/practice -run DemoFixtures -v`. |
| Q7 | `answer_key_document_id` sai loại → `422 WRONG_DOCUMENT_TYPE`; thuộc lớp khác hoặc không tồn tại → `404`; chưa READY → `409 DOCUMENT_NOT_READY` (giống `document_id`, kiểm trước khi xếp việc). |
| Q12 | `server_now` nằm trong thân `AttemptView`, ở cả `POST …/practice/attempts` (start) lẫn `GET …/practice/attempts/{id}`. Không dùng header `Date`. |
| Q16 | Không có cách bật: `ESSAY` chỉ vào lượt `TOPIC`; ma trận `MOCK` không nhận `ESSAY` (không tham số `start`, không cấu hình lớp). AC4 nhánh "bật" → **NGOÀI PHẠM VI** (BA ghi vào spec). |
| Q20 | `cd backend-go && go test -count=1 -v ./internal/practice -run TestWrongAnswerSignalsNoIdentity` (thêm test của QC vào cùng gói, `package practice_test`, dùng `newRig`). |
| Q22 | Tool `search_library` nằm ở `internal/library` (đăng ký ở `internal/agent`): `go test -count=1 -v ./internal/library -run SearchLibrary` (`TestSearchLibraryTool`, `TestSearchLibraryNoIdentityParam`, `TestSearchLibraryExcludesAnswerKey`). |
| Q23 | Ví dụ JSON ở `openapi.yaml` → `FormsImportRequest.example`. `answer_source=BANK`: mỗi phần tử `mapping.questions[]` là `{col, question_id}` (đáp án lấy từ ngân hàng). `answer_source=MANUAL`: `{col, accepted: ["B"]}` (đáp án đúng gõ tay, so khớp như câu SHORT). `col` là tên cột Excel (`A`, `B`, …) hoặc theo `header_row`; `email_col` / `code_col` / `time_col` tuỳ chọn. |
| Q24 | Điểm = `đúng / tổng × max_score` của `grade_items` MANUAL đã chọn, `decimal`, làm tròn **nửa lên** (`DivRound`) về 2 chữ số thập phân — chính là độ chính xác của `grade_items.max_score` / `grade_entries.score` (`numeric(5,2)`; nhập XLSX cũng từ chối quá 2 chữ số, `SCORE_PRECISION`). `grade_items` không có cột "bước" riêng; bước làm tròn 0,01–1 của công thức chỉ áp lên điểm tổng kết, không áp lên từng dòng nhập. Ví dụ 7/9 × 10 = 7,78; 2/3 × 8 = 5,33. |

**Đổi hành vi theo P-10-7 Q13 (FR-17):** `PUT …/answers` ở lượt `MOCK` quá `deadline_at`: trong 60 s chỉ nhận lựa chọn có `client_ts ≤ deadline_at` (lưu); `client_ts > deadline_at` → bỏ, trả 200 như đã nhận, ghi một dòng vào `notes` của lượt ("Một số lựa chọn gửi sau giờ làm bài đã không được tính." — không lặp); thiếu `client_ts` hoặc quá 60 s → `409 ATTEMPT_CLOSED`. Trước đó mã nhận `client_ts ≤ hạn + 60 s`. Test: `TestMockOfflineTolerance`. Thêm sqlc `PracticeAttemptAddNote` (không đổi schema).

## Sửa QC pha không stack (P-10-8)
| Mục | Việc đã làm |
| --- | --- |
| F1 | Ngưỡng Hôm nay sinh viên 7 truy vấn (6 + 1 gộp luyện đề) giữ nguyên; chú thích ở `today/staff_test.go` ghi P-10-8. |
| F2 | Xoá `frontend/src/mock/questions.ts` (không còn import). |
| F3 / Q-S10-27 | Trần chi phí LLM (Redis cửa sổ giờ cố định, cùng cơ chế nhận xét tự luận): `extract` ≤ 5 việc / lớp / giờ; `generate` ≤ 5 việc / lớp / giờ; việc thứ 6 → `429 RATE_LIMITED` + `Retry-After` (giây tới giờ kế), **không xếp việc**. Chỉ 60 trang đầu của đề được trích: trang > 60 bỏ, `result.skipped` có `{reason: "PAGE_LIMIT", page: 61}`. Redis lỗi → cho qua (fail-open như các hạn mức khác); hạn mức ngày của Scheduler vẫn áp. Test: `TestJobLimitsPerCourse`, `TestExtractPageCap`. openapi: `429` ở `extract`, `generate`. |
| F4 | `explain` có `429`: ≤ 30 lượt **cần LLM** / giờ / sinh viên (câu đã cache 24 giờ không tính), kiểm TRƯỚC khi mở luồng. Trước đây SRS #11 ghi 429 nhưng mã không có; con số 30 là mặc định của dev (nhận xét tự luận là 10) — BA/PM đổi nếu muốn. Test: `TestExplainRateLimit`. |
| Q-S10-29 | (1) `practice-explain` trong `seed/demo/fake_llm.json` (xem Q6). (2) Bật `OVERLOADED` trên stack, **không cần** `docker-compose.test.yml`: đặt cho gateway và worker `LLM_MAX_CONCURRENCY=1 LLM_QUEUE_MAX=1 LLM_QUEUE_WAIT_MAX=1s FAKE_LLM_LATENCY=15000-15000` rồi `up -d gateway worker` (kết hợp được với `LLM_DEMO_FILE`); gửi 3 yêu cầu cùng lúc. Việc `extract` / `generate` (BATCH): hàng đầy → `ErrOverloaded` → worker thử lại 2 s, 5 s rồi báo `OVERLOADED` ("Chưa trích được. Thử lại."). `Giải thích thêm` (INTERACTIVE): yêu cầu thứ hai chờ > 1 s → lỗi giải thích tại chỗ. [INFERENCE: đọc `scheduler.Admit`; chưa chạy trên stack.] Cách khác: `docker-compose.test.yml` + `POST /_test/llm/fake {ErrorRate:1,…}` cho lỗi nhà cung cấp (`LLM_UNAVAILABLE`), không phải `OVERLOADED`. |
| Q-S10-30 | Kết quả nhập Forms: mỗi dòng Chưa khớp có nút «Nhập ở Sổ điểm» (đóng hộp thoại, về Sổ điểm để nhập tay). e2e `gradebook.spec.ts` (forms import flow). |
| Q-S10-31 | `Loại` hàng loạt: lệnh hoãn 5 s được gửi NGAY khi `pagehide` bằng `fetch keepalive` (`apiClient` thêm `keepalive`; hàng đợi ở mức module vì BulkBar bị gỡ khi vùng chọn rỗng); Hoàn tác thì không gửi gì. e2e `questions-extract.spec.ts` («Loại hàng loạt: pagehide…»). |

## Bổ sung 2026-11-14 (hoàn thiện sprint 10 trước báo cáo)
| Mục | Nội dung |
| --- | --- |
| Q14 / Q25 | Điểm lượt luyện = `đúng × 10 ÷ tổng`, `decimal.Round(1)` (nửa lên với số dương). Câu nhiều đáp án chấm `quiz.AllOrNothing`: đúng khi trùng toàn bộ tập lựa chọn, không điểm từng phần. |
| Q-S10-28 | Ngưỡng ≥ 80 là phía QC; không có thay đổi mã. |
| Bật `OVERLOADED` / giải thích giả | (1) Giải thích giả: chạy gateway với `DEMO_MODE` và `LLM_DEMO_FILE=/seed/demo/fake_llm.json` (bản ghi `practice-explain`). (2) `OVERLOADED`: `LLM_MAX_CONCURRENCY=1 LLM_QUEUE_MAX=1 LLM_QUEUE_WAIT_MAX=1s FAKE_LLM_LATENCY=15000-15000 docker compose -p ep-qc up -d gateway worker`, rồi gửi 3 yêu cầu cùng lúc. **Sửa trong đợt này:** `docker-compose.local.yml` trước đây chỉ chuyển bốn biến đó cho `gateway`; nay `worker` cũng nhận (việc `extract` / `generate` chạy ở worker, nếu không thì không bao giờ gặp `OVERLOADED`). Chưa chạy trên stack. |
| CI đỏ trên `sprint/10-practice` | (a) `golangci-lint`: 14 lỗi (contextcheck, gochecknoglobals, errcheck, ineffassign, S1016, unused) ở `internal/practice`, `internal/quiz`, `internal/store`; sửa mã / `//nolint` có lý do, không nới cấu hình. (b) Playwright: `exam.spec.ts` (hai ca "questions bank") — mục chọn trong menu Tạo bị đổi tên "Soạn tay" làm hỏng `createMenu(... "Câu trắc nghiệm")`: trả lại tên cũ; chi tiết câu hỏi đọc `citations` khi mock không có trường này: nay `?? []`. `panels.spec.ts` (ba ca) — `/practice`, `/practice/history` đã là màn thật nên không còn Panel khi gateway không trả lời; ba ca thêm `practiceApi(page)` (gateway giả theo hợp đồng, có sẵn trong `e2e/support/practice-fixtures.ts`). Không sửa kỳ vọng nào. |
| Chạy lại tại Mac | `go vet ./...`; `go test ./internal/practice/... ./internal/quiz/... ./internal/today/...` xanh; `golangci-lint run ./...` 0 lỗi; `pnpm -C frontend lint` + `build` xanh; Playwright desktop: `exam.spec.ts` "questions bank" (5), `panels.spec.ts` (3 ca nêu trên) xanh trên bản `build:gate`. |

## Sửa theo QC pha 2 (2026-11-14)
| Lỗi | Việc đã làm |
| --- | --- |
| L3 | `practice.updated` nay mang `course_id`; `today.Invalidator` xoá khoá `all` + khoá theo lớp của CHÍNH người luyện, không xoá cache staff. Test `TestPracticeUpdatedInvalidatesCourseScope`. (Mục "Khác biệt" 3 ở trên không còn đúng: payload đúng SRS FR-19.) |
| L1 (TL-1) | `ingest/chunk.go`: `mergeShort` đưa tiêu đề của mẩu bị gộp vào văn bản và chỉ gộp cùng trang; tiêu đề không thân thành khối riêng. `practice/extract.go` + `PracticeDocChunks` lấy `heading` vào lời nhắc. Test hồi quy `TestChunkExamPaperKeepsEveryQuestion` dùng Markdown docling THẬT (`ingest/testdata/de-mau-trac-nghiem.md`), đỏ ở mã cũ. **Tài liệu đã nạp trước bản vá vẫn giữ đoạn cũ**: tải lại đề mẫu (hoặc seed lại) mới trích đủ 24 câu; RAG cũng được hưởng bản vá. |
| L4 | Seed bước 20: câu `GENERATED` (bản ghi `practice-generate-seed`, 5 câu, 3 đã duyệt, đều có citations), câu `EXTRACTED` (tải `seed/practice/de-mau-trac-nghiem.pdf` + `dap-an-mau.pdf` làm tài liệu lớp rồi `extract`), Quiz mở sau ~14 s, đóng sau ~2,5 phút, 3 sinh viên làm, chờ `PUBLISHED`; nhận Quiz cũ ở DRAFT / SCHEDULED xa để dời lịch; lớp 761988 có 20 câu `APPROVED` riêng. `check-practice-seed.mjs` kiểm thêm các điều đó. |
| L9, L10 | `generate` `types: []` → `422`; bulk `APPROVE` câu `EXTRACTED` không trích dẫn → `CITATION_REQUIRED`. |
| L12 | Forms nhận cột theo chữ cái HOẶC tên ở dòng tiêu đề (không phân biệt hoa thường; chữ cái thắng khi trùng); chữ lỗi tệp Forms: «Chỉ nhận tệp Excel (.xlsx) hoặc CSV.», «tối đa 5 MB». Ô chọn tệp mặc định của hộp thoại: chưa đổi. |
| L11 | Menu Tạo câu hỏi: «Soạn tay» là nhóm cha (Câu trắc nghiệm / Bài lập trình), «Trích từ đề», «Sinh từ tài liệu», «Gợi ý từ AI»; không dòng phụ. `exam.spec.ts` chỉ đổi hàm `createMenu` theo P-10-9. |
| L7, L8 | Chủ đề có trong ngân hàng nhưng 0 câu đủ điều kiện → `409 NOT_ENOUGH_QUESTIONS {available:0}`. `BuildMatrix` bù từ chủ đề khác (rồi bất kỳ) trước khi cắt ngắn đề (P-10-9). |
| L15, L16 | Lớp chưa mở câu hỏi: «Giảng viên chưa mở câu hỏi» (một dòng). `citations[].document_title` ở chi tiết câu; nhãn nguồn «<tên tài liệu>, trang N». |
| L6, L13, L14 | L6 → nợ sprint 11 (P-10-10, PM). L13, L14 là ghi nhận của QC, không sửa. |
| QC #3 | `scripts/dev.mjs`: khi có `EP_PORT_OFFSET`, `BLOB_PUBLIC_ENDPOINT` mặc định `localhost:<EP_PORT_MINIO>` (không còn ghi nhầm vào MinIO cổng 9000 của stack khác). |

### Trả lời QC phát hiện #2
- **`OVERLOADED` cho việc nền (TC-02-14, TC-03-08):** công thức ở trên làm đầy hàng của MỘT tiến trình; worker chạy việc tuần tự nên không bao giờ đầy. Bộ lập lịch giữ đồng thời toàn cục trong Redis nhưng hàng chờ nằm trong từng tiến trình, nên nhân bản worker cũng không làm đầy hàng. Cách kiểm được: (a) mức Go: `Worker.call` thử lại 2 s, 5 s rồi trả `OVERLOADED` ("Chưa trích được. Thử lại.") khi `llm.ErrOverloaded` — chưa có test riêng, dev chưa viết; (b) trên stack chỉ kiểm được `explain` (INTERACTIVE) như TC-06-20. [INFERENCE: đọc `scheduler.Admit` và `extract.go:call`, chưa chạy.] Xin PM quyết: ghi TC-02-14 / 03-08 «kiểm ở mức Go» hoặc dev thêm test `ErrOverloaded` ở sprint 11.
- **Payload LLM đã che (TC-06-14, 06-15, 02-06, 02-16, 03-06, 08-12):** `llm_audit` KHÔNG lưu nội dung lời nhắc, và không có nơi nào cho ADMIN xem payload đã che: đó là chủ ý (không lưu PII, kể cả đã che). Cách kiểm: ở mức Go, `stubLLM` trong `internal/practice/llm_test.go` ghi lại lời nhắc cuối (`lastUser`); `TestExplainAnswer` và `TestEssayFeedback` dùng nó để kiểm lời nhắc không có định danh; `TestExtractQuestions` / `TestGenerateCitations` (`staff_test.go`) chưa kiểm lời nhắc. QC thêm ca vào cùng gói (`package practice_test`, dùng `newRig` + `withLLM`) thay vì đọc DB.
