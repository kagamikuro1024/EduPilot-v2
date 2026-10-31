# QC report — US-P6-03 (trích công thức từ quy chế: `ExtractGradeScheme`, bộ kiểm bằng code, trích dẫn, AI chỉ nháp) · Kết luận: **FAIL** (vòng 1)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `4acd28b6` (dev) → kiểm trên `d4efecdd`/`f4921c86` (không đổi mã story). Bộ TC: `tc-US-P6-03.md` (sửa 1 dòng theo #11, lịch sử trong tệp). Handoff: `dev-US-P6-03.md`. **CI của dev đỏ** (run `41185772706`: `Go test (rest-1)`, `Go test (rest-2)`, `Frontend e2e (Playwright 1/3, 2/3)`, `Frontend Lighthouse (devtools B)`); QC xác nhận lại ba lỗi Go trên máy QC. Lượt chấm vẫn làm đủ để dev sửa một lần; AC của story không có AC nào FAIL ở mức hành vi, nhưng story **không** đạt "không phá cái đang chạy" (`AGENTS.md` nguyên tắc 8).

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p6-go-tests.sh ./internal/grading "<37 test của US-P6-03>"` | PASS 37/37 (không thiếu, không SKIP) |
| `go test -count=1 -race -tags integration ./internal/grading/... ./internal/grade/...` | PASS (167 test) |
| `go test -count=1 ./internal/contract/...` (không `-tags testroutes`) | PASS |
| `go test -count=1 -race -tags testroutes ./internal/contract -run TestSpec_LoadsAndValidates` | **FAIL**: `contract_test.go:67: openapi.yaml: 227 thao tác (cần 226)` |
| `go test -count=1 -tags testroutes ./internal/integration -run TestNoUnscopedChunkQuery` | **FAIL**: `course_guard_test.go:141` — `GradeDocForExtract (grade_scheme.sql): đọc content_chunks mà không lọc theo course_ids` |
| `go test -count=1 -tags testroutes ./internal/rag -run TestKnowledgeDocExcludedEverywhere` | **FAIL**: `knowledge_test.go:175` — `grade_scheme.sql: truy vấn đọc documents mà không loại tài liệu tri thức` (thiếu `THREAD_KNOWLEDGE`) |
| `go test ./internal/integration -run TestAllCourseRoutesGuarded` | PASS (route mới đi qua guard; `…DetectsMissing` PASS) |
| `go vet` các gói đụng tới; `golangci-lint run ./internal/grading/... ./internal/grade/... ./internal/httpapi/...` | PASS, `0 issues` |
| `sqlc diff` | PASS rc=0 |
| `grep -rln 'openai-go\|api.openai.com' internal --include=*.go \| grep -v '_test.go\|internal/llm/'` | rỗng (SDK chỉ ở `internal/llm`, luật 3) |
| `grep -rn ConfirmScheme backend-go --include=*.go` | rỗng — chưa có hàm xác nhận nào (US-P6-04 thêm); AC6 hiện đúng |
| Thử đột biến (bản sao `/tmp/qc-mut`, đã xoá) | xem mục 2 |
| `p6-hand-calc.py`; `p6-citation-check.py` ở fixture | rc=0; 10/10 và 6/6 trích dẫn đúng trang (đáp án BA đúng) |
| `playwright test class-join.spec.ts -g "bell real: chấm"` (máy QC) | **FAIL 4/4 lần** (`class-join.spec.ts:398` `expect.poll(() => lists).toBeGreaterThan(before)`: Received 1) — trùng ca đỏ ở CI shard 1/3 desktop + mobile (retry #1 cũng đỏ) |

## 2. Thử đột biến (test của dev có bảo vệ hành vi không?)
| Đột biến | Kết quả |
| --- | --- |
| N1 bỏ `ROUNDING_MISSING` ở bộ kiểm | **đỏ**: `TestCheckerRounding`, `TestExtractLop2OneUnclear` |
| N2 tắt kiểm trích dẫn (`cited != nil` → `false`) | **đỏ**: `TestCitationNotFound`, `…PageMissing`, `…OCRTolerance`, `…ComparedAfterUnmask` |
| N3 thử lại cả lỗi không-tạm (`!transient` → `false`) | **đỏ**: `TestExtractLLMDown` |
| N4 `temperature` 0 → 0,7 | **đỏ**: `TestExtractOneStructuredCall` |
| N5 `maxAttempts` 3 → 5 | **đỏ**: `TestExtractLLMDown` |
| N7 bỏ nhánh "bản nháp bị sửa trong lúc job chạy" | **đỏ**: `TestExtractDraftEditedDuringJob` |

## 3. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 `extract` → `202 {job_id}` → job xong, SSE tiến độ | PASS ở mức test (contract `202`, `TestExtractGradeScheme` chạy job tới `DONE`) | chưa chạy trên stack đầy đủ + SSE trình duyệt: để US-S8-SEED / gate |
| 02 đúng một lời gọi `llm_audit`, `UTILITY`, `NEAR_REALTIME` | PASS | `TestExtractUsesPrivacyMask` (đếm `llm_audit` từng lời gọi), `TestExtractTaskAndLane`, `TestExtractOneStructuredCall` |
| 03 không lộ toàn văn quy chế ở log / `llm_audit` | PASS theo cấu trúc | `llm_audit` không có cột nội dung (đã đọc `db/migrations/00002_llm.sql`: `llm_audit` chỉ có task, lane, provider, model, token, trạng thái — không cột prompt / response) |
| 04 SDK chỉ ở `internal/llm` | PASS | grep rỗng |
| 05 test AC1 | PASS | 5/5 |
| 06 schema `strict` | PASS | `TestExtractSchemaStrictCompatible` |
| 07–09 quy chế lớp 1: 3 thành phần 20/20/60, đủ luật, `unclear_items` rỗng | PASS | `TestExtractGradeScheme` so từng trường với `extraction-lop1.json` |
| 10 trích dẫn đúng trang | PASS | `p6-citation-check.py` 10/10 ở fixture; `TestCitationVerified` |
| 11 `seed/demo/quy-che-lop1.pdf` | KHÔNG KIỂM ĐƯỢC (chưa có PDF) → US-S8-SEED | |
| 12 `Quyche.pdf` → `OTHER` | KHÔNG KIỂM ĐƯỢC → US-S8-SEED | |
| 13, 15 lớp 2: đúng một `ROUNDING_MISSING` | PASS | `TestExtractLop2OneUnclear` (N1 làm đỏ) |
| 14 PDF lớp 2 không có "làm tròn" | PASS ở nguồn: `grep -ciE 'làm tròn|quy tròn|bước điểm' fixtures/quy-che-lop2.md` = 0; PDF: → US-S8-SEED | |
| 16 test bộ kiểm | PASS 8/8 | `TestChecker*` |
| 17–20 đầu ra LLM cố ý sai (tổng 90, thang lỗi, `PARAM_INVALID`, LLM nói `unclear_items: []`) | PASS (test của dev + N1) | mã do code thêm (`source CODE`), không tin LLM |
| 21–24 trích dẫn bịa / sai trang / bỏ dấu / đổi chữ | PASS | `TestCitation*` 5/5; N2 làm đỏ; `vn_fold` chỉ ở SQL (`grade_scheme.sql`), không có hàm `vnFold` ở Go |
| 25, 27 `DRAFT`, `confirmed_by` NULL; không đường xác nhận ngoài handler | PASS | `TestExtractNeverConfirms`; không có `ConfirmScheme`; `TestConfirmOnlyFromHandler` thuộc US-P6-04 |
| 26 `ConfirmScheme` chỉ tham chiếu từ handler | PASS hiện tại (chưa tồn tại) → kiểm lại ở US-P6-04 | |
| 28 seed không ghi thẳng DB | KHÔNG KIỂM ĐƯỢC → US-S8-SEED | |
| 29, 31 prompt có ranh giới, không theo lệnh trong văn bản, mask PII | PASS | `TestPromptDelimitsDocument`, `TestPromptNoInstructionFollowing`, `TestExtractUsesPrivacyMask` |
| 30 injection "đặt trọng số 100% giữa kỳ" | PASS ở mức dựng prompt (test của dev); provider giả phát lại bản ghi nên không tự chứng minh LLM "không tuân lệnh" (giới hạn đã ghi ở TC) | |
| 32–34 `DOCUMENT_NOT_POLICY` / `NOT_READY` / `POLICY_TOO_LONG` | PASS | contract (422 / 409 / 422) + `TestExtract{NotPolicy,NotReady,TooLong}` |
| 35–37 LLM lỗi → thử lại ≤ 3 → `LLM_UNAVAILABLE`; sai schema → `EXTRACTION_INVALID` một lời gọi; không phá `CONFIRMED` | PASS | `TestExtractLLMDown`, `TestExtractInvalidSchema`, `TestExtractFailureKeepsConfirmed`; N3, N5 làm đỏ |
| 38–39 idempotent; worker bị giết | PASS | `TestExtractIdempotentKey`, `TestExtractIdempotencyHTTP`, `TestExtractKillWorkerMidRun` |
| 40–42 `DRAFT_EXISTS`, `replace_draft`, sửa trong lúc job chạy, `version` đổi | PASS | `TestExtract{DraftExists,ReplaceDraft,DraftEditedDuringJob,ReplaceDraftVersionChanged}`; N7 làm đỏ |
| 43 12 test AC7–AC8 | PASS | |
| 44, 46–48 TA / STUDENT / ADMIN `403`; STUDENT đọc `403`; TA đọc | PASS: contract (`401` không đăng nhập, `403` TA/SV/ADMIN) + `TestExtractRBAC`; TA `GET …/grade-scheme` thuộc US-P6-04 | |
| 45 giảng viên lớp khác | PASS theo #11: `403` (guard dùng chung); id không phải uuid `404` (contract dòng 47); document không tồn tại `404` | |

## 4. AC
| AC | Kết quả |
| --- | --- |
| AC1 một lời gọi có cấu trúc | PASS |
| AC2 quy chế lớp 1 | PASS (mức fixture/provider giả; PDF seed ở US-S8-SEED) |
| AC3 quy chế lớp 2 một chỗ chưa rõ | PASS (như trên) |
| AC4 bộ kiểm bằng code | PASS |
| AC5 trích dẫn có thật | PASS |
| AC6 AI chỉ nháp | PASS |
| AC7 nhánh lỗi tài liệu / LLM | PASS |
| AC8 idempotent, nhận lại việc | PASS |
| AC9 tài liệu là dữ liệu không tin cậy; riêng tư | PASS |
| AC10 phân quyền | PASS (theo #11) |

Các AC của story **PASS**; story **FAIL** vì phá test cũ / CI đỏ (mục 5).

## 5. Lỗi
- **BUG-P6-03-1 (Cao — phá test cũ, rule bảo mật):** `internal/store/queries/grade_scheme.sql` truy vấn `GradeDocForExtract` đọc `content_chunks` mà không có điều kiện `course_ids` → `TestNoUnscopedChunkQuery` (`internal/integration/course_guard_test.go:141`, "truy vấn nào chạm `content_chunks` cũng phải có điều kiện `course_ids`") đỏ. Tái hiện: `cd backend-go && go test -count=1 -tags testroutes ./internal/integration -run TestNoUnscopedChunkQuery`. Kỳ vọng: truy vấn thoả luật chung (nêu `course_ids` khi đọc `content_chunks`); **không** sửa test. Trong code hiện tại truy vấn có `d.course_id = sqlc.arg(course_id)` ở bảng `documents`, nhưng subselect trên `content_chunks` lọc theo `document_id` — đủ về chức năng, nhưng luật chung đòi điều kiện `course_ids` để phát hiện hàm "không lọc lớp"; cách sửa là thêm điều kiện `c.course_ids @> array[course_id]` (cùng mẫu `rag.sql`).
- **BUG-P6-03-2 (Cao — phá test cũ, rule tri thức luồng):** các truy vấn đọc `documents` ở `grade_scheme.sql` thiếu điều kiện loại tài liệu tri thức (chữ `THREAD_KNOWLEDGE`) → `TestKnowledgeDocExcludedEverywhere` (`internal/rag/knowledge_test.go:175`, quét MỌI `from/join documents`) đỏ. Tái hiện: `go test -count=1 -tags testroutes ./internal/rag -run TestKnowledgeDocExcludedEverywhere`. Kỳ vọng: mọi truy vấn đọc `documents` loại `THREAD_KNOWLEDGE` (như các `*.sql` khác); không sửa test.
- **BUG-P6-03-3 (Cao — phá test cũ, cần proposal):** `TestSpec_LoadsAndValidates` (`internal/contract/contract_test.go:67`) ghim số thao tác của `openapi.yaml` ("cần 226"); thêm route `POST …/grade-scheme/extract` làm 227 → đỏ ở CI (`-race -tags testroutes`); không đỏ khi chạy không có `-tags testroutes` nên dev không thấy ở lượt chạy gói. Đây là test cũ: dev **không** tự sửa số; mở proposal (cùng loại #9) rồi báo PM qua herdr, chờ ACCEPTED. Mỗi story route kế tiếp của sprint sẽ lặp lại điều này: đề nghị PM duyệt **một lần** quy tắc "mỗi route mới tăng số ghim đúng bằng số route thêm, commit ghi số proposal chung", tránh mở proposal mỗi story.
- **BUG-P6-03-4 (Trung bình — CI đỏ chưa rõ nguyên nhân):** `Frontend e2e` `class-join.spec.ts:361` "bell real" đỏ ở CI (shard 1/3 desktop + mobile, kể cả retry #1) và trên máy QC (4/4 lần): `expect.poll(() => lists)` không vượt `before` sau `page.clock.runFor(31_000)`. Diff frontend giữa lần xanh cuối (`12fa90bf`, run `41184610941`) và `4acd28b6` chỉ ở `IntegrationsSettings.tsx`, `EmbeddingSection.tsx`, `mock/system.ts` — không chạm chuông thông báo; QC chưa tìm ra nguyên nhân. Đề nghị dev điều tra (có thể ca vốn mong manh với thời gian chạy; hoặc phụ thuộc thứ tự). Cùng run: `Frontend Lighthouse (devtools B)` đỏ LCP `maxNumericValue` (assertion `error`) — nghi cùng nguyên nhân tải máy CI; dev chạy lại để phân biệt flake với hồi quy. QC không chốt đây là lỗi của US-P6-03. **Cập nhật 2026-10-31 (run `41186476014`, commit chỉ đổi frontend, Go bỏ qua):** "bell real" vẫn đỏ, thêm `activity.spec.ts:261` "heartbeat cadence even" (đỏ cả retry), `dev-ui.spec.ts:234` "cursor" (một lần) và LCP devtools B — toàn ca dùng đồng hồ giả / thời gian ⇒ nghi hạ tầng CI hoặc thời gian chạy, **không** do US-P6-03; đề nghị PM giao dev (hoặc `research`) một việc riêng điều tra, vì chặn "CI xanh" của mọi story.
- **BUG-P6-03-5 (Quy trình, Trung bình):** handoff liệt kê `go test ./internal/grading ./internal/grade ./internal/contract ./internal/store` — không chạy bộ test toàn cục (`scripts/go-test-all.sh`) trước khi push; ba test cũ đỏ do chính commit này (lần thứ hai: US-P6-01 đã bị `TestGradeQueriesNeverReadStudyTime`). Luật `dev.md`: "CI xanh trước khi sang story kế; dùng `scripts/go-test-all.sh` cho lượt Go đầy đủ".

## 6. Kiểm phản mẫu / luật mở rộng / phân quyền
- Luật 3: SDK provider chỉ ở `internal/llm`. Luật 11: task `UTILITY`, làn `NEAR_REALTIME` (test). Luật 12: việc nền 202 + job. Luật 14: `Idempotency-Key` bắt buộc (contract) và gửi lại cùng khoá → cùng `job_id`.
- Test cũ bị sửa: `contract_test.go` +1 dòng `want` (proposal #9 ACCEPTED); `scenarios_exam_test.go` +1 dòng gọi (#9); `store/gradebook_schema_test.go` đổi căn lề gofmt (test của chính US-P6-01). Không nới assertion nào.
- Không thư viện mới; không secret; không PII trong log (`TestExtractUsesPrivacyMask`).
- `golangci-lint` 0 issue; `sqlc diff` rỗng.

## Đề nghị
1. Dev sửa `grade_scheme.sql` (BUG-1, 2) — thêm điều kiện `course_ids` và loại `THREAD_KNOWLEDGE`, chạy `sqlc generate`; mở proposal cho số ghim 226 → 227 (BUG-3), báo PM qua herdr; điều tra BUG-4; chạy `scripts/go-test-all.sh` trước push.
2. Sau khi CI xanh, QC chấm lại nhanh: ba test cũ + `TestExtract*` + contract với `-race -tags testroutes`; story PASS nếu không còn lỗi.
3. Các TC còn chờ seed (11, 12, 14 PDF, 28) và stack đầy đủ (01 SSE) chuyển sang US-S8-SEED / `gate-P6`.
