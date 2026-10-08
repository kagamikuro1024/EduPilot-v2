# Báo cáo QC — US-PE-04 (bài thi: tạo, chọn câu, xem trước, lên lịch / bỏ lịch / gia hạn, bộ lập lịch mở–đóng, danh sách, "Hôm nay", giao diện `/exams`)
**Kết luận: PASS có điều kiện** — không FAIL. Điều kiện: giao diện (TC-42…49) chỉ chấm bằng e2e của dev (QC chưa thăm dò tay bằng `playwright-cli` + ảnh); `EXAM_UPCOMING` (48 giờ trước) chưa quan sát được do chỉ một việc "recommended" hiện ra; CI GitHub không chạy (billing). Bản chấm `757a237` (API) và `203334e` (Go + Playwright toàn bộ). Stack thật của QC: `EP_PORT_OFFSET=100 SEED_ON_EMPTY_DB=true pnpm dev` (gateway, worker, Postgres, Redis, MinIO, Caddy, frontend, judge), tài khoản seed (`teacher`, `ta`, `sv.gioi`, `sv.kha`, `admin`); API gọi qua Caddy bằng `scripts/q3lib.py`, `q4_exams.py`, `q4b_exams.py`, `q4c_timing.py` (QC tự viết). Không có lỗ hổng bảo mật.

## Lỗi / lệch
- **L1 (TC-14).** `schedule` bài thiếu mục và có `opens_at` quá khứ trả cả `NO_ITEMS` và `OPENS_IN_PAST` trong một lượt (`details[]` đủ). Với bài **đủ mục** (TC-14 đủ các lỗi `ITEM_NOT_APPROVED`, `QUESTION_REJECTED`…) QC không dựng hết tổ hợp — dev `TestScheduleValidationsAll` PASS.
- **L2 (TC-10).** Xem trước: hai lần gọi cho **thứ tự đáp án khác nhau** (bài một câu: 2 thứ tự khác nhau / 6 lần), `preview:true`, khung `{preview, exam, items[{item_id, position, type, points, stem, options, code}]}`; QC chưa so thứ tự **câu** trên bài nhiều câu (mục 5 câu, hai lần so khoá sai trong script đầu).
- **L3 (TC-29).** `extend` trả `422 LIMIT_OUT_OF_RANGE` (không phải `CLOSES_NOT_LATER`) khi mốc mới cách mốc cũ **> 24 giờ**, đúng ý "cũ + 24 h + 1 s → 422"; mã lỗi `LIMIT_OUT_OF_RANGE` thay vì mã riêng — ghi nhận.
- **L4 (TC-39).** Việc của sinh viên ở "Hôm nay" nằm ở trường `recommended` (một việc đứng đầu), không phải `actions[]`: khi bài đang mở, SV A và SV B đều thấy `EXAM_OPEN` "Bài thi … đang mở · Mở đến 03:01 09/10. Bạn có 5 phút để làm." (`href=/exams/{id}/take`, 5 phút); `EXAM_UPCOMING` cho bài mở sau 20 giờ **không** hiện cùng lúc vì chỉ một việc được đề xuất — chưa kiểm riêng.
- **L5.** CI GitHub không chạy (billing); mọi kết quả là chạy local.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `POST /exams` (`Idempotency-Key`) → `201`, `DRAFT`, `shuffle_questions=true`, `shuffle_options=true`, `max_score=10.00`, `rounding_step=0.01`, `multi_scoring=PARTIAL`, `appeal_days=7`, `version=1` |
| 02 | PASS | `title` rỗng / 121 → `TITLE_LENGTH`; `instructions` 4.001 → `INSTRUCTIONS_TOO_LONG`; `duration` 4 → `DURATION_TOO_SHORT`; 301 → `LIMIT_OUT_OF_RANGE`; 90 > khung 60 → `DURATION_EXCEEDS_WINDOW`; biên 4.000 ký tự, 5, 300 phút → `201` |
| 03 | PASS | TA tạo `201`; SV `403`; ADMIN `403`; `+07:00` và `Z` cùng thời điểm → cùng `opens_at` (`2026-12-01T01:00:00Z`) |
| 04 | PASS | cùng key → cùng `id`; cùng key khác thân → `422 IDEMPOTENCY_KEY_REUSED` |
| 05, 09, 12, 18, 22, 27, 30, 32, 35, 38, 41 | PASS (dev) | `TestCreateExamDefaults`, `TestExamValidation`, `TestPutItems`, `TestPutItemsRejects`, `TestPreviewNoAttemptCreated`, `TestPreviewShapeEqualsStudent`, `TestScheduleValidationsAll`, `TestScheduleEffects`, `TestUnscheduleRules`, `TestEditLockMatrix`, `TestExamStateMachine`, `TestEffectiveStatus`, `TestExtendOnlyLater`, `TestExtendRecomputesRunningDeadlines`, `TestDeleteOnlyDraft`, `TestCloneExam`, `TestExamListStaffVsStudent`, `TestExamDetailStudentProjection`, `TestScheduleRaceWithQuestionReject`, `TestExamEditVersionConflict`, `TestExamProvidersStudent`, `TestExamTodayInvalidation` đều PASS trong lượt tích hợp `go test -race -tags integration ./...` (**907 PASS, 0 FAIL**) |
| 06 | PASS (một phần) | `PUT …/items` 5 MCQ → `200`, `kind=MCQ`, `position` 1…5; thay toàn bộ; QC chưa dựng `MIXED` (câu code không qua được duyệt do script thiếu test ẩn) — dev test bao |
| 07 | PASS | câu `DRAFT` / `PENDING` / `REJECTED` / lưu trữ → `ITEM_NOT_APPROVED`; lớp khác → `QUESTION_NOT_IN_COURSE`; trùng → `DUPLICATE_ITEM`; rỗng → `NO_ITEMS`; `points` 0,01 và 100 qua, 0 và 100,01 → `LIMIT_OUT_OF_RANGE` |
| 08 | PASS | `PUT items` khi `SCHEDULED` → `409 EXAM_LOCKED` `details.reason="status"` |
| 10 | PASS (L2) | xem L2; không tạo lượt (`exam_attempts` không đổi: dev `TestPreviewNoAttemptCreated`) |
| 11 | PASS | SV preview `403`; ADMIN preview `403`; GV `POST …/attempts` → `404` (route không áp cho staff) |
| 13 | PASS | GV `schedule` → `200 SCHEDULED`; thông báo `EXAM_SCHEDULED` tới SV A và SV B: "Bài thi … mở lúc Chủ nhật, 02:51 08/11, làm trong 45 phút" (≤ 4 s); `outbox` có `exam.scheduled` |
| 14 | PASS (L1) | bài không mục + `opens_at` quá khứ → `422` cả `NO_ITEMS`, `OPENS_IN_PAST` |
| 15 | PASS | `opens_at` = +59 s → `OPENS_IN_PAST`; +64 s → `200 SCHEDULED` (`EXAM_MIN_LEAD_SECONDS=60`) |
| 16 | PASS | TA `schedule` → `403`; gọi lần 2 khi đã `SCHEDULED` → `200`, **0** thông báo thêm |
| 17 | PASS (dev) | `409 COURSE_ARCHIVED`: `TestScheduleEffects` |
| 19 | PASS | `unschedule` → `200 DRAFT`; thông báo lịch cũ bị thu hồi (không còn bản `EXAM_SCHEDULED` của bài đó), thông báo huỷ lịch do `HandleUnscheduled` |
| 20 | PASS (một phần) | `unschedule` khi `DRAFT` → `409`; `opened` / `has_attempts`: dev `TestUnscheduleRules` |
| 21 | PASS (một phần) | bài `SCHEDULED`: `opens_at`, `closes_at`, `duration_minutes`, `shuffle_questions`, `shuffle_options`, `max_score`, `rounding_step`, `multi_scoring` → `409 EXAM_LOCKED` `reason=status`; `reveal_answers`, `appeal_days`, `title`, `instructions` sửa được (`200`); các trạng thái `OPEN` / `CLOSED`: dev `TestEditLockMatrix` |
| 23 | PASS | bài mở sau 65 s, đóng sau 6 phút: `SCHEDULED` 19:51:56 → **`OPEN` 19:52:58 (trễ 0,4 s sau `opens_at` 19:52:57)** → **`CLOSED` 19:58:58 (trễ 0,5 s sau `closes_at`)**, ≤ 10 s; `outbox`: `exam.scheduled` 1, `exam.opened` 1, `exam.closed` 1 (mỗi mốc một lần) |
| 24, 25, 26 | PASS (dev, một phần) | `TestEffectiveStatus`, `TestExamStateMachine`; QC chưa tắt worker / chạy hai worker / `kill` leader |
| 28 | PASS (dev) | `TestExtendRecomputesRunningDeadlines`; QC thử gia hạn ở `SCHEDULED` (xem TC-29) |
| 29 | PASS (L3) | `closes_at` mới bằng cũ / ngắn hơn → `422 CLOSES_NOT_LATER`; +24 h + 1 s → `422 LIMIT_OUT_OF_RANGE`; +24 h → `200`; TA `extend` → `403` |
| 31 | PASS | `DELETE` nháp: TA `403`, GV `204`, đọc lại `404`; `DELETE` `SCHEDULED` → `409`; `clone` bằng TA → `201`, "… (bản sao)", `DRAFT`, 3 mục, `opens_at` null |
| 33 | PASS | `GET /exams` GV và TA: `status`, `effective_status`, `items_count`, `attempts{started,graded}`, `version`…; SV chỉ thấy bài **không phải nháp** (3 bài `SCHEDULED`, không có `DRAFT`), thân bài của SV: `id,title,instructions,kind,opens_at,closes_at,duration_minutes,max_score,status,my_attempt,my_score` (không `items`, không trạng thái nội bộ); SV `GET` bài nháp → `404` |
| 34 | PASS | `?status=SCHEDULED` → chỉ `SCHEDULED`; `limit=2` → 2; `cursor=x` → `422` |
| 36 | PASS | 6 `PUT` cùng `version`: `[200, 409, 409, 409, 409, 409]` |
| 37 | PASS | 12 lần `schedule` song song `REJECT` một câu của bài: **0** bài `SCHEDULED` chứa câu `REJECTED` |
| 39 | PASS (L4) | `recommended` của SV = `EXAM_OPEN` đúng chữ / giờ / 5 phút / `href`; sau khi bài `CLOSED` không còn |
| 40 | PASS (dev) | `TestExamTodayInvalidation` (≤ 2 s qua outbox), `TestExamProvidersStudent` |
| 42–46 | PASS (dev e2e) | `exam.spec.ts › exam editor` (4 ca desktop, mobile bỏ qua theo thiết kế) PASS; `ui-antipatterns.sh` rc=0, 19 `✓`; `pnpm lint` rc=0 |
| 47 | PASS (dev e2e) | `exam.spec.ts › student exam list` (có bài / rỗng) PASS |
| 48 | PASS | `account.spec.ts › nav per role after login` (**SV 8, TA 13, GV 16**) và `shell.spec.ts › nav per role` (nhãn + href + thứ tự) PASS; `tc-US-PU-04` đã sửa theo góp ý #1 |
| 49 | PASS (dev e2e) | như 47 |
| 50 | PASS | `go vet`, `golangci-lint` 0 issues, `sqlc diff` rc=0; `go test -race -count=1 -tags integration -p 1 -parallel 2 ./...` rc=0 (**907 PASS**, 0 FAIL); `pnpm lint`, `build:gate` rc=0; Playwright (không `@real`, không `visual`; **không** có stack compose chạy cùng): **357 pass, 103 skip, 0 fail**; `audit-login.mjs` bốn vai chưa chạy ở story này |

## Việc sau
- **QC:** thăm dò UI `/exams` bằng `playwright-cli` + ảnh (cổng PE); TC-24 / 25 (tắt worker, hai worker); `EXAM_UPCOMING` riêng; `MIXED` (câu code duyệt được); chấm lại TC-14 đủ tổ hợp lỗi. Scripts: `scripts/q4_exams.py`, `q4b_exams.py`, `q4c_timing.py`.
