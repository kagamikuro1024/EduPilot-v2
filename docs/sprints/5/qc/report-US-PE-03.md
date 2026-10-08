# Báo cáo QC — US-PE-03 (ngân hàng câu hỏi: soạn, test, zip, lời giải mẫu, duyệt, AI gợi ý nháp)
**Kết luận: FAIL 1 TC (TC-13 — bug B1: ký tự NUL gây 500) ; PASS có điều kiện với phần còn lại.** Bản chấm `d88c98e` (`origin/sprint/5-pe`). Stack thật của QC: `EP_PORT_OFFSET=100 SEED_ON_EMPTY_DB=true pnpm dev` (gateway, worker, Postgres, Redis, MinIO, Caddy, frontend, **judge thật** `-no-seccomp`; dựng 150 s, seed 38 s) + máy chủ LLM giả tương thích OpenAI do QC viết (`scripts/qserver.py`, ghi mọi thân yêu cầu). Gọi API qua Caddy bằng `scripts/q3lib.py`, `q3_code.py`, `q3_review.py`, `q3_misc.py` (QC tự viết, không dùng test của dev). Tài khoản seed `teacher`, `ta`, `sv.gioi`, `admin`; lớp 761987 (C1) / 761988 (C2). Không có lỗ hổng bảo mật.

## Lỗi / lệch
- **B1 (TC-13 / TC-01 — FAIL).** Ký tự **NUL (U+0000)** trong `stem`, `title`, nội dung đáp án, `input` hoặc `expected` của test → **`500 INTERNAL`** (log gateway: `invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)`), thay vì `422` có mã lỗi. Riêng `name` có NUL được chặn đúng (`422 TEST_NAME`). *Repro:* `POST /courses/{C1}/questions` `{"type":"TRUE_FALSE","title":"t","topic":"x","stem":"a\u0000b","value":true}`; `POST …/testcases` `{"name":"n","input":"1","expected":"a\u0000"}`. AC4 nói test phải UTF-8 hợp lệ; người dùng gõ / dán NUL không nên gây 500. *Sửa gợi ý:* loại / từ chối `\x00` ở lớp kiểm hợp lệ (`TEST_NOT_UTF8` / `INVALID_TEXT`).
- **L1 (TC-35 — kiểm bằng nhà cung cấp giả).** Với provider `fake` mặc định của `pnpm dev`, gợi ý MCQ cho `LLM_UNAVAILABLE` (đầu ra của `fake` không khớp schema); gợi ý test code chạy được. QC dựng provider `openai_compatible` giả đúng schema để kiểm luồng thật (xem TC-35). Không phải lỗi mã; `docs` nên ghi "gợi ý AI cần provider thật hoặc server giả có schema".
- **L2 (TC-07).** Thân `TRUE_FALSE` dùng trường **`value`** (không phải `answer` như TC ghi); `answer` → `422 unknown`. TC chấm theo mã / SRS (`value`).
- **L3 (TC-02).** `MCQ_MULTI` chọn **tất cả** đáp án trả `NO_CORRECT_OPTION` (không có mã riêng "tất cả đúng"); vẫn bị từ chối, 422.
- **L4 (TC-43).** `archived=true` trả cả câu chưa lưu trữ (22 so với 21 mặc định ⇒ "gồm cả lưu trữ", không phải "chỉ lưu trữ"). TC ghi "mặc định ẩn câu lưu trữ": đúng; ngữ nghĩa `archived=true` = bao gồm — ghi nhận.
- **L5 (UI / e2e).** Lượt Playwright đầy đủ trên máy QC (stack compose đang chạy → RAM) **bị sập server Next giữa chừng** (`ERR_CONNECTION_REFUSED` :3330): **293 pass, 52 fail, 1 flaky**; mọi ca của **PE-03** (`exam.spec.ts › questions bank` 5 ca × 2 dự án, `markdown.spec.ts` 3 × 2) **PASS**; 52 fail nằm ở `today`, `ui-foundation`, `account`, `dev-ui`… (đã xanh 330 / 0 fail ở lượt PU-06 khi không có stack) — do hạ tầng, không phải mã PE-03. Chạy lại 1 worker cũng treo; QC dừng.
- **L6 (TC-31 / 32 / 33 / 38).** Khoá "đang dùng", sửa test sau đóng và `ITEM_NOT_APPROVED` cần route bài thi (`/exams`, US-PE-04): chấm ở báo cáo US-PE-04.
- **L7.** CI GitHub không chạy (billing); mọi kết quả là chạy local.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `201`, `DRAFT`, `origin=MANUAL`, id đáp án là uuid, `answer_key = {"option_ids":[<uuid>]}` |
| 02 | PASS (L3) | 16 ca sai đều `422 VALIDATION_FAILED` + mã chi tiết: `OPTION_COUNT` (1 / 9), `OPTION_BODY_INVALID` (rỗng / 1.001), `DUPLICATE_OPTION`, `SINGLE_MULTIPLE_CORRECT`, `NO_CORRECT_OPTION`, `STEM_TOO_LONG`, `TITLE_LENGTH`, `INVALID_DIFFICULTY`, `INVALID_OPTION_ID`, `required` |
| 03 | PASS | 2 và 8 đáp án; MULTI 1 và n−1 đúng; `stem` 8.000; `title` 120 → `201` cả sáu |
| 04 | PASS (một phần) | API trả `stem` / `body` **nguyên văn** (so từng ký tự); giao diện hiện chữ: `markdown.spec.ts` (8 chuỗi thù địch, không `dangerouslySetInnerHTML`) PASS; QC chưa đi tay bằng `playwright-cli` |
| 05 | PASS | 4 câu ("Tất cả các đáp án trên", "Cả hai đáp án", "Không có đáp án nào", "A và B") → `201` kèm cảnh báo `UNPINNED_COMPOSITE`; `pinned_last:true` → không cảnh báo |
| 06, 08, 12, 16, 22, 26, 30, 34, 39, 42, 45, 47, 54 | PASS (dev) | các test Go tương ứng PASS trong lượt tích hợp (xem `report-US-PE-01.md`, 783 PASS, 0 FAIL); QC không chạy lại riêng |
| 07 | PASS (L2) | `TRUE_FALSE {"value":true}` → `201`, `answer_key={"value":true}`; thiếu → `422 VALUE_REQUIRED`; thừa `options` → `422 OPTION_COUNT`; `SHORT`, `ESSAY` → `422 TYPE_NOT_SUPPORTED`; `XYZ` → `422 INVALID_TYPE` |
| 09 | PASS | mặc định đọc lại: `time_limit_ms 1000`, `memory_limit_mb 256`, `output_limit_kb 1024`, `checker EXACT`, `starter_code {}` |
| 10 | PASS | `time_limit_ms` 99 ✗ / 100 ✓ / 10000 ✓ / 10001 ✗; `memory` 15 ✗ / 16 ✓ / 1024 ✗(1.025 không; 1.024 ✓ theo dev) / 1025 ✗; `output` 0 ✗ / 1 ✓ / 16384 ✓ / 16385 ✗ (`LIMIT_OUT_OF_RANGE`); `FLOAT_EPS` thiếu eps → `FLOAT_EPS_REQUIRED`; `EXACT` + eps → chấp nhận (`200`); `java` / `[]` → `LANGUAGE_NOT_ALLOWED` |
| 11 | PASS | QC đọc `tests_version`: thêm / sửa test tăng đúng (2, 3, 4, 5, rồi 6 sau `PUT` test); đổi giới hạn / checker / ngôn ngữ: dev `TestCodeProblemBumpsTestsVersion` PASS (QC không đo lại từng bước) |
| 13 | **FAIL** (B1) | `weight` mặc định 1; −1 / 1001 → `LIMIT_OUT_OF_RANGE`; 1,5 → `type`; 0 và 1000 ✓; `name` rỗng / 61 → `TEST_NAME`, 60 ✓; input **64 KiB** vào DB, **64 KiB + 1** ra kho (DB: `input IS NULL AND input_blob_key IS NOT NULL` = 1 dòng); 1 MiB + 1 → `413`; **NUL → 500** |
| 14 | PASS | `tests_version` +1 cho mỗi `POST` / `PUT` test; QC đọc lại sau `PUT` (5 → 6) |
| 15 | PASS | SV `403`, ADMIN `403`, TA `200`, GV `200`; GV gọi `…/questions/{qid}/testcases` qua đường dẫn lớp khác → `404`; GV thấy cả test ẩn |
| 17 | PASS | zip 5 test (`sample1`, `t2`, `t10.ans`, `t3`, `t4`): `dry_run` → `200 {would_create:5, created:0}` (số test không đổi 6); thật → `201 {created:5}`, số test 6 → 11; `sample1.is_sample=true`; thứ tự tự nhiên `sample1, t2, t3, t4, t10`; `.ans` nhận |
| 18 | PASS | 7 zip lỗi + 101 test: `ZIP_MISSING_PAIR`, `DUPLICATE_NAME`, `ZIP_NESTED`, `ZIP_PATH` (`../`, `/abs`), `ZIP_UNEXPECTED_FILE`, `TEST_NOT_UTF8`, `TOO_MANY_TESTS`; mỗi lỗi nêu đúng tệp; **số test không đổi (11)** sau cả bộ |
| 19 | PASS | zip-slip (`../evil.in`, `/abs.in`) bị từ chối `ZIP_PATH`; (container gateway không có shell nên không `find`; ghi nhận qua phản hồi + DB không có test) |
| 20 | PASS | (a) 60 tệp × 1 MiB giải nén 60 MiB (72 KB nén): `422 TEST_ZIP_INVALID` trong **< 0,1 s**; (b)/(c) tệp 11 MiB: kết nối bị đóng sớm (giới hạn thân yêu cầu), gateway còn sống (`readyz` `200`); RAM gateway 15 MiB, worker 10 MiB sau bộ thử |
| 21 | PASS (một phần) | mặc định "thêm vào cuối": số test 6 → 11; `mode=replace` chưa thử |
| 23 | PASS | `POST …/reference/verify` có `Idempotency-Key` → `202 {job_id}`; job `SUCCEEDED`, `ok:true`, `per_test` 3 × `AC`; `reference_verified_version` = `tests_version` (6); `llm_audit` không có dòng nào từ verify (4 dòng đều `QUESTION_GEN` của gợi ý) |
| 24 | PASS | test `h2` có `expected` sai: verify → `ok:false`, `per_test` `h2 = WA`; cờ **không** ghi (`reference_verified_version` NULL) |
| 25 | PASS (một phần) | cùng `Idempotency-Key` → cùng `job_id`; thiếu key → `422`; sau khi đổi test, cờ về NULL và verify lại → `ok:true`; chưa thử `docker stop judge` |
| 27 | PASS | `REQUEST` → `PENDING`, `APPROVE` → `APPROVED` (GV `200`, TA `200`); SV `403`; ADMIN `403`; sai `version` → `409` |
| 28 | PASS | `APPROVE` khi: không test → `CODE_TESTS_MISSING` + `TOTAL_WEIGHT_ZERO` + `REFERENCE_NOT_VERIFIED`; chỉ test mẫu → `CODE_TESTS_MISSING`; chỉ test ẩn `weight 0` → `TOTAL_WEIGHT_ZERO`; chưa verify → `REFERENCE_NOT_VERIFIED`; đổi test sau duyệt → `DRAFT`, cờ về NULL |
| 29 | PASS | DB: `INSERT … origin='AI_DRAFT', review_status='APPROVED'` thiếu `reviewed_by` → `23514 question_bank_review_chk`; câu AI do GV / TA duyệt được (`200`); SV không thấy (`403`) |
| 31 / 32 / 33 / 34 | KHÔNG KIỂM ĐƯỢC (L6) | cần route bài thi |
| 35 | PASS | `POST …/questions/suggest` (`Idempotency-Key`) → `202`; job `SUCCEEDED`, 5 yêu cầu: tạo **3** (`MCQ_SINGLE`, `MCQ_MULTI`, `TRUE_FALSE`), bỏ **2** (`OPTION_COUNT`, `SINGLE_MULTIPLE_CORRECT`); câu `origin=AI_DRAFT`, `review_status=PENDING`; DB: `created_by` = GV bấm, `ai_job_id` có; cùng key → cùng job; thiếu key `422`; `count` 0 / 21 → `422`; SV `403` |
| 36 | PASS (một phần) | câu sai bị bỏ (xem 35); không cấu hình: job `FAILED` `LLM_NOT_CONFIGURED` "Chưa cấu hình AI. Nhờ quản trị viên thêm nhà cung cấp."; lỗi dữ liệu: `LLM_UNAVAILABLE` (L1); `OVERLOADED` chưa thử |
| 37 | PASS | máy chủ giả ghi 2 yêu cầu: prompt người dùng chỉ có `Chủ đề / Độ khó / Số câu` (MCQ) và `Đề bài / Giới hạn / Ví dụ` (test); **0** email / MSSV / tên sinh viên / `sv.` (regex quét toàn thân); `api_key` không nằm trong thân |
| 38 | PASS (một phần) | SV đọc / liệt kê câu AI → `403`; `422 ITEM_NOT_APPROVED` chờ PE-04 |
| 40 | PASS | `kind=CODE_TESTS` → job `SUCCEEDED`, tạo 3; mọi test `approved=false`, `source=AI_DRAFT`; `expected` = đầu ra chạy lời giải mẫu thật (`3`, `-2`, `3000000` cho `1 2`, `-5 3`, `1000000 2000000`) |
| 41 | PASS | `tests.total` chỉ đếm test đã duyệt (3 trước và sau khi có 3 test AI chưa duyệt); chưa nộp bài (chờ PE-06) |
| 43 | PASS (L4) | lọc `review_status`, `topic`, `difficulty`, `type`, `origin`, `archived`, `q` (không phân biệt hoa thường; 100 ký tự ✓, 101 → `422`); `limit` 0 / 101 → `422`; `cursor` rác → `422`; `limit=5` → 5 |
| 44 | PASS | 13 thao tác × {SV, ADMIN, SV lớp khác} = **39 × `403`**; không token = **13 × `401`**; GV / TA `200`; câu của C1 gọi qua C2 → `404` (GET, PUT, `duplicate`, `/code`, `testcases`) |
| 46 | PASS | `duplicate` → `201`, tiêu đề "… (bản sao)", `DRAFT`, `MANUAL`, `tests_version=1`, `reference_verified_version` null, test được sao (`total 3`) |
| 48–52 | PASS (dev e2e, L5) | `exam.spec.ts › questions bank` 5 ca × 2 dự án PASS; `ui-antipatterns.sh` rc=0 (19 `✓`); `pnpm lint` rc=0; `build:gate` rc=0. QC chưa thăm dò tay bằng `playwright-cli` |
| 53 | PASS | `GET /me/today` GV: `QUESTION_REVIEW` "1 câu hỏi chờ duyệt · lớp 761987" ≤ 2 s sau `REQUEST`; `GET /courses/{C1}/today` có; TA nhìn thấy cùng việc; SV không có |
| 55 | PASS có điều kiện | `go vet`, `golangci-lint`, `sqlc diff` rc=0 và `go test -race -tags integration ./...` xanh ở lượt `c5f93d7` (PE-01/02); chưa chạy lại toàn bộ ở `d88c98e` |

## Việc sau
- **Dev:** B1 (NUL → `422`).
- **QC:** chấm lại TC-13 khi sửa; TC-31–33, 38 ở US-PE-04; UI thăm dò bằng `playwright-cli` + ảnh ở cổng PE; chạy lại Playwright đầy đủ khi không có stack compose (L5). Scripts: `scripts/q3lib.py`, `q3_code.py`, `q3_review.py`, `q3_misc.py`, `qserver.py`.
