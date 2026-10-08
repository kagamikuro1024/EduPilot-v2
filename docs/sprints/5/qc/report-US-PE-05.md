# Báo cáo QC — US-PE-05 (làm bài trắc nghiệm: bắt đầu / làm tiếp, xáo theo lượt, tự lưu, một nơi ghi, nộp, tự nộp hết giờ, `/exams/[id]/take`)
**Kết luận: PASS có điều kiện** — không FAIL. Điều kiện: (1) giao diện làm bài (TC-11…13, 37…51) chỉ chấm bằng e2e của dev, QC chưa thăm dò tay bằng `playwright-cli` + ảnh, chưa đo INP / Lighthouse; (2) TC-19 (QC tự cài đặt xáo `SHA-256 → PCG → Fisher–Yates` rồi so 100 %) và TC-25 (k6 `autosave` 300 VU) chưa chạy; (3) khoá chat `ep:exam_lock` lúc bắt đầu chưa có (chuyển PE-07, dev ghi trong handoff); (4) CI GitHub không chạy (billing). Bản chấm `881a069` (mã PE-05 = `5279a01` + `953644d`). Stack thật của QC: `EP_PORT_OFFSET=100 SEED_ON_EMPTY_DB=true pnpm dev` + 30 tài khoản seed; API gọi qua Caddy bằng `scripts/q5lib.py`, `q5a2.py`, `q5b.py`, `q5c_deadline.py`, `q5d.py`, `q5e.py` (QC tự viết). Không có lỗ hổng bảo mật; không lộ đáp án.

## Lỗi / lệch
- **L1 (TC-22 / 32 / 35).** `PUT …/answers`, `POST …/submit` và `…/takeover` **bắt buộc** header `X-Exam-Tab` (UUID): thiếu → `422 VALUE_REQUIRED` ("Thiếu mã tab (X-Exam-Tab…"). TC viết "có `X-Exam-Tab`" cho lưu nhưng không nêu cho `submit`; chấm theo mã. Thân lưu là `{items:[{item_id, answer:{option_ids|value}}]}` (TC ghi `answers`).
- **L2 (TC-24).** Hạn mức lưu tính từ **đầu phút theo SV**: sau 12 lần lưu trước đó, lần thứ **233** mới `429` (= 240 / phút gồm các lần trước); khớp AC (241 → 429).
- **L3 (TC-27).** QC đo "tự nộp sau hết giờ" bằng thăm DB mỗi 1 s sau mốc `deadline + 11 s`: đã `GRADED` ở thời điểm đo đầu tiên (`deadline + 12,2 s`), `submitted_at` ghi **đúng `deadline_at`** (`20:25:53.593`); không thấy độ trễ dài hơn `grace + 10 s`. Độ phân giải đo ~1 s.
- **L4 (TC-33).** `auto_score` của lượt nộp tay (3 câu chọn đáp án đầu theo thứ tự xáo) = `0.00` — hợp lý với đáp án chọn; không có trường điểm trong phản hồi `submit`.
- **L5.** CI GitHub không chạy (billing); mọi kết quả là chạy local.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `POST …/attempts` → `201`; `started_at` = giờ máy chủ, `deadline_at = started_at + 5 phút` (20:20:53 → 20:25:53); khoá: `attempt{id,exam_id,status,started_at,deadline_at,server_time,writer}`, `exam{…}`, `items[]` |
| 02 | PASS (dev) | `deadline = closes_at` khi còn ít giờ, `EXAM_NOT_OPEN reason=closing`: `TestStartAttempt*` PASS |
| 03 | PASS | bài `DRAFT` → `409 EXAM_NOT_OPEN {reason:not_scheduled}`; `SCHEDULED` chưa tới → `{reason:not_yet, opens_at}` |
| 04 | PASS | thiếu `Idempotency-Key` → `422`; TA `403`, GV `403`, ADMIN `403`; SV không thuộc lớp (`sv.moi`) `403` |
| 05 | PASS | hai `POST` cùng key → cùng `attempt.id` |
| 06, 10, 17, 21, 26, 30, 34, 39, 44, 55, 58 | PASS (dev) | `TestStartAttemptOK`, `TestResumeSameAttempt`, `TestClockNotResetOnReload`, `TestOneAttemptRaceStart`, `TestStartAfterSubmit409`, `TestAttemptPayloadWhitelist`, `TestAttemptPayloadNoAnswerCanary`, `TestShuffleDeterministic`, `TestShuffleDiffersAcrossStudents`, `TestSaveAnswersDiff`, `TestSaveRateLimit`, `TestSaveAfterDeadlineGrace`, `TestGraceAcceptsLastSave`, `TestAutoSubmitOnTimeout`, `TestAutoSubmitOnClose`, `TestSubmitManual`, `TestSubmitIdempotentReplay`, `TestSingleWriterTab`, `TestTakeoverFlipsWriter`, `TestStaleTabCannotOverwrite`, `TestStudentGuardStates`, `TestRemovedMidExam403`, `TestAfterSubmitSummaryOnly`, `TestResultBeforePublish409` PASS trong `go test -race -tags integration ./...` (**935 PASS**, 0 FAIL) |
| 07 | PASS | `POST` lần hai (khoá khác) → `200`, **cùng lượt**, `started_at` y nguyên; `GET attempts/mine` cùng lượt |
| 08 | PASS | **20 `POST` song song** của một SV: mã `{200, 201}`, **1** `attempt.id`, 1 dòng DB |
| 09 | PASS | sau nộp: `POST attempts` → `409 ATTEMPT_ALREADY_SUBMITTED` |
| 11, 12, 13 | PASS (dev e2e) | `exam-clock.spec.ts` 8 ca (lệch −300 / 0 / +37 / +300 s, mỗi lần đổi ≤ 1 s, định dạng, mốc 5 / 1 phút) PASS; `take: clock skew` PASS; **QC kiểm thêm**: lưu tại `deadline + 9,2 s` → `200`, tại `deadline + 11,0 s` → `409 ATTEMPT_CLOSED` (máy chủ quyết, không theo đồng hồ trình duyệt) |
| 14 | PASS | `POST attempts` / `GET mine`: canary đặt trong **giải thích** của 7 câu: **0** lần; khoá cấm (`answer_key`, `is_correct`, `explanation`, `correct`, `override`, `reference`, `expected`) **0**; tập khoá: `answer, attempt, body, closes_at, code, deadline_at, duration_minutes, exam, exam_id, id, instructions, is_you, item_id, items, kind, multi_scoring, options, points, position, server_time, started_at, status, stem, title, type, writer` |
| 15 | PASS | cùng một câu: SV A và SV B có **cùng tập** id đáp án (uuid) nhưng **thứ tự khác**, thứ tự câu khác; không cờ đúng / sai |
| 16 | PASS | phản hồi lưu chỉ `{saved_at, server_time, deadline_at}` (không đúng / sai, không điểm) |
| 18 | PASS | **30 SV** thật ở lớp 761987: **99,8 %** cặp có thứ tự câu khác nhau (≥ 95 %); cùng SV hai lần ⇒ cùng thứ tự câu và đáp án; hoán vị đầy đủ (không trùng / sót) |
| 19 | CHƯA CHẠY | QC chưa tự cài đặt `SHA-256 → PCG → Fisher–Yates` để so 100 % (dev: `TestShuffleDeterministic`) |
| 20 | PASS (một phần) | chấm theo `option_id` (không theo vị trí): điểm lượt hết giờ 5,71 đúng với đáp án chọn trong thứ tự xáo; tắt xáo chưa thử |
| 22 | PASS | lưu 1 câu → `200`; lô 3 câu → `200`; DB `exam_answers` 1 → 4 dòng (chỉ câu gửi); `option_ids: []` → `200`, dòng bị xoá (4 → 3); id lạ → `422 INVALID_OPTION_ID`; `MCQ_SINGLE` 2 id → `422`; `item_id` không thuộc bài → `422`; thân 64 KiB + → `413 PAYLOAD_TOO_LARGE` |
| 23 | PASS | SV khác lưu vào lượt người khác → `404`; lượt đã nộp → `409 ATTEMPT_ALREADY_SUBMITTED`; quá hạn → `409 ATTEMPT_CLOSED` (TC-28) |
| 24 | PASS (L2) | 245 `PUT` trong 1 s: `200` ×232, **`429` từ lần 233** (đã có 12 lần trước trong phút); phút sau lưu được |
| 25 | CHƯA CHẠY | k6 `autosave` 300 VU chưa chạy (`benchmarks/load/exam-submit.js` chưa có; PE-09) |
| 27 | PASS (L3) | lượt 5 phút hết giờ khi SV đã lưu: `status=GRADED`, `submit_reason=TIMEOUT`, `submitted_at = deadline_at`, `auto_score = 5.71`; khớp mong đợi tự tính (4 / 7 × 10 = 5,714… → bước 0,01 ⇒ **5.71**: 2 câu đơn đúng + MULTI `{0,2}` đúng + TRUE_FALSE đúng; MULTI `{0,1}` PARTIAL = 0; câu đơn sai và bỏ trống = 0) |
| 28 | PASS | `deadline + 9,2 s` → `200`; `deadline + 11,0 s` → `409 ATTEMPT_CLOSED` |
| 29, 31 | PASS (dev) | `TestAutoSubmitOnClose`; `effective_status`/`ATTEMPT_CLOSED` khi worker chết: QC chưa tắt worker |
| 32 | PASS | `POST submit` → `200 {status:"GRADED", submitted_at, answered:3, total:7}`; **cùng `Idempotency-Key`** → cùng thân (`submitted_at` giống); khoá khác → `409 ATTEMPT_ALREADY_SUBMITTED`; `PUT answers` sau nộp → `409`; **không** có điểm / đúng sai trong phản hồi |
| 33 | PASS | nộp khi làm 3 / 7 → `answered=3`, `total=7`; câu bỏ trống 0 điểm |
| 35 | PASS | `PUT` với tab T1 khi tab T2 đang giữ → `409 ATTEMPT_OTHER_TAB` + `details.writer_seen_at`; `POST …/takeover` (T2) → `200`; `PUT` từ T2 → `200` |
| 36 | PASS | sau takeover của T2, `PUT` của T1 (bản cũ) → `409`; `exam_answers` giữ đáp án của T2 (DB: id đáp án = T2) |
| 37, 38 | PASS (dev e2e) | `take: two tabs`, `take: duplicate tab`, `take: reload writer` PASS |
| 40, 41, 42, 43 | PASS (dev e2e) | `exam-save-queue.spec.ts` 7 ca (gộp 2 s, lùi 1-2-4-8-15 s, lỗi cố định dừng, khôi phục từ máy, kho hỏng) và `take: offline mid exam` PASS; QC chưa tự cắt mạng bằng CDP |
| 45, 46, 47, 48, 49 | PASS (dev e2e) | `take: mcq 375` PASS; `ui-antipatterns.sh` rc=0, 19 `✓`; axe / INP / Lighthouse chưa đo (điều kiện 1) |
| 50, 51, 52 | PASS (dev e2e) | `take: submit confirm and after` PASS (hộp xác nhận nêu số câu, nộp lại cùng `Idempotency-Key`, màn sau nộp không điểm) |
| 53 | PASS | `GET attempts/mine` sau nộp: `{attempt:{id,status:"GRADED",submitted_at,submit_reason:"MANUAL"}, exam:{id,title,closes_at,status}}` — không đề, không điểm; `GET …/result` trước công bố → `409` |
| 54 | PASS | SV B đọc `…/attempts/{aid của A}/result` → `404`; TA, GV, ADMIN gọi route SV → `403` |
| 56 | PASS | SV ngoài lớp → `403`; (lớp lưu trữ `409 COURSE_ARCHIVED`: `TestStudentGuardStates`) |
| 57 | PASS | GV mời SV ra giữa giờ: ngay `PUT answers` → `403`, `GET mine` → `403`; lượt vẫn còn trong DB (không bị xoá); QC `undo` khôi phục |
| 59 | PASS | `GET exam`, `attempts/mine`, `attempts/{aid}/result` (SV A) trước công bố: 0 canary, 0 khoá cấm |
| 60 | PASS | `go vet`, `golangci-lint` 0 issues, `sqlc diff` rc=0; `go test -race -count=1 -tags integration -p 1 -parallel 2 ./...` rc=0 (935 PASS, 0 FAIL); `pnpm lint`, `build:gate` rc=0; Playwright (không `@real`, không `visual`): **403 pass, 103 skip, 0 fail**; `audit-login.mjs` chưa chạy ở story này |

## Việc sau
- **QC:** TC-19 (xáo theo đặc tả), TC-25 (k6), tắt worker / mạng bằng CDP, UI `/exams/[id]/take` tay bằng `playwright-cli` + ảnh + axe / INP; khoá chat (PE-07). Scripts: `scripts/q5lib.py`, `q5a2.py`, `q5b.py`, `q5c_deadline.py`, `q5d.py`, `q5e.py`.
