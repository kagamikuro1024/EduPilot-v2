# Báo cáo QC — US-PE-06 (làm bài lập trình trong giờ thi: nháp theo ngôn ngữ, `Chạy thử`, `Nộp lời giải`, lịch sử nộp, nộp tự động từ nháp)
**Kết luận: PASS có điều kiện** — không FAIL. Điều kiện: (1) giao diện soạn mã (TC-04…08, 13, 14, 21, 22, 27, 39, 41–43) chỉ chấm bằng e2e của dev, QC chưa thăm dò tay bằng `playwright-cli` + ảnh, chưa đo axe / INP; (2) điểm của bài có code (`GRADED`, TC-34, 37) thuộc US-PE-08 — lượt dừng ở `GRADING`: KHÔNG KIỂM ĐƯỢC; (3) lệch AC9 `source` (L1) chờ PM (proposal #16); (4) TC-19 nửa "judge sống + 60 bài trong hàng nộp" và TC-25 (cap 30 bằng `EXAM_SUBMIT_COOLDOWN=1`) chưa chạy trên stack; CI GitHub không chạy (billing). Bản chấm `81bd367`. Stack thật `EP_PORT_OFFSET=100` + judge go-judge thật (arm64, `-no-seccomp`); API qua Caddy bằng `scripts/q6lib.py`, `q6a.py`, `q6b.py`, `q6c.py`, `q6d.py`, `q6e.py`, `q6f.py` (QC tự viết; bài MIXED 6 + 4 điểm, 4 test: 2 mẫu + 2 ẩn mang canary `QC-HID-`, lời giải mẫu mang `QC-REF-`).

## Lỗi / lệch
- **L1 (TC-29, AC9 vs AC10).** `GET …/submissions/{sid}` trả thêm khoá `source` (mã của chính SV). AC9 liệt kê khoá cho phép không có `source`, AC10 đòi xem lại mã. Dev đã nêu (proposals #16a); `source` chỉ của chính mình, danh sách #36 không kèm mã. Chờ PM / BA, không tính FAIL.
- **L2 (TC-11, 18).** `source` rỗng / 64 KiB + 1 trả `422 VALIDATION_FAILED` (`details` code `required` / `max`), **không** phải `SOURCE_EMPTY` / `SOURCE_TOO_LARGE` như TC. Mã HTTP đúng, không lộ nội dung. Dev tự báo 422; chấm theo mã HTTP.
- **L3 (TC-40).** `PUT draft` tại `deadline + 11 s` trả `409 ATTEMPT_ALREADY_SUBMITTED` (worker đã tự nộp lúc `deadline + ~10 s`), không phải `ATTEMPT_CLOSED`. Cùng ý: bị từ chối, bản nháp không đổi. `deadline + 9,4 s` → `200` (nhận).
- **L4 (TC-26).** Với giãn cách 15 s, lần nộp 1 luôn chấm xong trước lần 2 nên không quan sát được `SUPERSEDED` qua API (lần 1 `DONE`, `is_final=false`; lần 2 `is_final=true`); nhánh `SUPERSEDED` chỉ có test dev (`TestSupersede…` PE-02).
- **L5.** Playwright chạy 1 lượt đủ: 36 ca đỏ vì `ERR_CONNECTION_REFUSED` tại `:3330` (Next chết giữa chừng, đã gặp ở US-PE-04, L5 cũ — không phải lỗi mã). Chạy lại: `exam` + `today` + `data-layer` 104 pass / 1 flaky / 2 đỏ (SSE reconnect ở `data-layer`); chạy riêng `data-layer` `--workers=1`: **29 / 29 pass**. Tổng hợp: toàn bộ pass, không có ca đỏ thật.
- **L6.** `EXAM_GRADE`: lượt có code dừng ở `GRADING` đúng handoff (PE-08).

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `POST attempts` / `mine`: `items[].code` = `{languages, time_limit_ms, memory_limit_mb, starter_code, samples[{name,input,expected}], language, drafts}`; 2 test mẫu; canary `QC-HID-` và `QC-REF-` **0** lần trong toàn thân; không `weight`, `tests_version`, `reference`, `checker`, `hidden` |
| 02, 03 | PASS (dev) | `TestCodeItemPayloadWhitelist`, `TestHiddenTestsNeverInPayload` PASS; test mẫu chưa duyệt: QC chưa tạo (thêm tay luôn `approved=true`) |
| 04, 05 | PASS (dev e2e) | `take: code viewport gate` (390 / 1023 / 1024 / 1440) |
| 06, 07, 08 | PASS (dev e2e) | `take: code editor` PASS; QC chưa kiểm tay / axe |
| 09 | PASS | `PUT draft` `java` → `422 LANGUAGE_NOT_ALLOWED`; mỗi ngôn ngữ một bản nháp (DB `code_drafts`: cpp17 và c11 độc lập, 64 KiB c11 lưu được) |
| 10 | PASS | `TestDraftPerLanguage`, `TestLanguageNotAllowed` PASS; `take: language switch keeps drafts` PASS |
| 11 | PASS (L2) | `base_rev=0` → `{rev:1}`, rồi `rev:2` (+1 mỗi lần); 64 KiB → 200; 64 KiB + 1 → 422; phản hồi chỉ `{rev, saved_at, server_time, deadline_at}` |
| 12 | PASS | `base_rev` cũ → `409 DRAFT_CONFLICT {current_rev:2, updated_at}`; mã trong DB giữ bản của `rev 2` (không ghi đè) |
| 13, 14 | PASS (dev e2e) | `take: draft conflict resolution`; offline: `exam-save-queue` (PE-05) — QC chưa tự cắt mạng |
| 15 | PASS | `TestDraftRevIncrements`, `TestDraftConflict409`, `TestDraftTooLarge`, `TestDraftAfterDeadline` |
| 16 | PASS | `run` `202` với `ok.cpp` → `compile_ok:true`, mẫu `AC, AC` trong 1,0 s; `wa` → `WA` kèm `input/expected/got`; `ce` → `compile_ok:false` + `compile_log`; `tle` → `TLE` (1000 ms); `re` → `RE`; `ml` → `MLE`; `stderr` (canary `QCSTDERR`, đường dẫn `/tmp/secret`) → **không** xuất hiện; chỉ **2 test mẫu** được chạy (không test ẩn); `QC-HID-` 0 lần |
| 17 | PASS | 11 lần `run` của một SV: 10 × `202`, lần 11 `429 RATE_LIMITED` `retry_after = 600` (= 10 phút − 0 s tính từ lần đầu, khớp); DB chỉ 10 dòng `RUN` (lần bị chặn không ghi); `TestRunRateLimit10Per10Min` PASS |
| 18 | PASS (L2) | thiếu key → `422 IDEMPOTENCY_KEY_REQUIRED`; `source` rỗng / 64 KiB + 1 → `422`; `language` lạ → `422 LANGUAGE_NOT_ALLOWED`; SV khác vào lượt → `404` |
| 19 | PASS (một phần) | `docker stop judge` → `run` → `503 JUDGE_UNAVAILABLE` trong **0,0 s** + `retry_after:18`; bật lại → `202`; hàng nộp đầy chưa thử |
| 20 | PASS | `TestRunSamplesOnly` (`internal/judge`), `TestRunQueuesRunRow`, `TestRunJudgeDown503`, `TestRunRateLimit10Per10Min` PASS |
| 21, 22 | PASS (dev e2e) | `take: run result display` PASS; `ui-antipatterns.sh` rc=0 |
| 23 | PASS | `submit` → `202 {submission_id}`; DB `kind=SUBMIT`, `source_sha256` có; outbox cùng giao dịch: `TestSubmitCodeEnqueues` |
| 24 | PASS | hai lần trong 15 s → `429 RATE_LIMITED` `retry_after:15`; sau 16 s → `202`; lần "ngay sau" → `429` |
| 25 | PASS (dev) | `TestSubmitCap30` (cấu hình 3) PASS; QC chưa nộp 30 lần |
| 26 | PASS (L4) | lần 1 sai `is_final=false`, lần 2 đúng `is_final=true`; lần ba (CE) cuối → `is_final=true` |
| 27 | PASS (dev e2e) | `take: submission history` (đang chấm → kết quả) |
| 28, 31, 33 | PASS | `TestSubmitCooldown`, `TestSubmitCap30`, `TestLastSubmissionCounts`, `TestSubmissionStudentViewHidesHidden`, `TestSubmissionViewCE`, `TestSubmissionsListOwnOnly`, `TestSubmissionsListCursor` PASS |
| 29 | PASS (L1) | khoá bản nộp: `compile_ok, created_at, id, is_final, language, samples, source, status`; mẫu `{memory_kb, name, time_ms, verdict}`; không `results` test ẩn, không `weight`; `QC-HID-`, `QC-REF-` 0 lần |
| 30 | PASS | `CE` → `compile_ok:false` + `compile_log`, bản mới nhất `is_final=true`; `IE`: dev `TestSubmissionViewIE` |
| 32 | PASS | `limit=2` mới nhất trước (CE rồi bản đúng), không kèm mã; SVB `404` |
| 34 | KHÔNG KIỂM ĐƯỢC | điểm theo lần `SUBMIT` cuối — lượt dừng `GRADING` (PE-08); chọn lần cuối: `TestLastSubmissionCounts` |
| 35 | PASS | SV có nháp `rev1`, nộp bài thi → `200`; DB: `SUBMIT`, `auto=true`, `DONE`, `source_sha256` = nháp |
| 36 | PASS | SV đã có 3 `SUBMIT` (auto=false), nộp bài thi → không thêm dòng `SUBMIT` auto; vẫn 3 dòng |
| 37 | PASS (một phần) | SV không có nháp → không dòng `SUBMIT` (chỉ `RUN`); `earned=0`: KHÔNG KIỂM ĐƯỢC (PE-08); `TestEmptyDraftNoSubmit` PASS |
| 38 | PASS | `TestAutoSubmitDraftWhenNone`, `TestNoAutoSubmitWhenSubmitExists`, `TestEmptyDraftNoSubmit`, `TestAutoSubmitUsesLatestDraftOnly`, `TestAutoSubmitOnTimeoutUsesGraceDraft` PASS (`TestFinalPicksLastSubmit` không tồn tại; tương đương `TestLastSubmissionCounts`) |
| 39 | PASS | SV gõ (draft mỗi 1,2 s) đến `deadline + 8,5 s` (253 bản), bản `+9,4 s` nhận; `GRADING/TIMEOUT`; `SUBMIT auto=true` có **đúng** nguồn của bản nháp cuối nhận (so chuỗi); `code_drafts.rev=254` |
| 40 | PASS (L3) | `deadline + 9,4 s` → `200`; `deadline + 11,1 s` → `409`; "Hết giờ — …": dev e2e |
| 41, 43 | PASS (dev e2e) | `take: timeout while typing`, `take: sse drop during judge` PASS |
| 42 | PASS (dev e2e) | QC chưa tự chặn SSE bằng CDP |
| 44 | PASS | `QC-HID-`, `QC-REF-`, `QCSTDERR` 0 lần ở `mine`, `draft` (cả 409), `run`, `runs/{id}`, `submit`, `submissions`, `submissions/{id}` (trạng thái đang làm); trạng thái đã nộp: `TestNoAnswerLeakCodeEndpoints` |
| 45 | PASS | SVB đọc `submission` / `run` / `list` của SVA → `404`; TA `403`, GV `403`, ADMIN `403` |
| 46 | PASS | `TestNoAnswerLeakCodeEndpoints` PASS |
| 47 | PASS | `go vet`, `golangci-lint`, `sqlc diff` rc=0; `go test -race -tags integration -p 1 -parallel 2 ./...` rc=0 (**937 PASS**, 0 FAIL); `pnpm lint`, `build:gate` rc=0; Playwright: 29/29 `data-layer` riêng; đủ các file còn lại pass (L5); `audit-login.mjs` chưa chạy ở story này |

## Việc sau
- **QC:** thăm dò tay `/exams/[id]/take` (soạn mã 1440, dải 390), ảnh + axe / INP; cap 30 với `EXAM_SUBMIT_COOLDOWN=1`; "hàng nộp đầy" cho `run`; điểm câu code sau PE-08 (TC-34, 37). Scripts: `scripts/q6lib.py`, `q6a.py`, `q6b.py`, `q6c.py`, `q6d.py`, `q6e.py`, `q6f.py`.
- **PM / BA:** chốt proposal #16 (`source` ở bản nộp; nút `Làm tiếp ở đây`).
