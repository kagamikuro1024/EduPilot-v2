# DEV handoff — US-P5-01 (migration `00013_crm`, lịch buổi học: xem trước, sửa, huỷ, buổi bù, "buổi chưa điểm danh")
Nhánh: `sprint/7-p4-p5`. Commit: `US-P5-01: …` (một commit). CI: xem cuối tệp.

## Đã làm (theo thứ tự lát dọc)
- **Migration `00013_crm.sql`** (SRS 5.1–5.6, proposals #5): enum `attendance_status`, `participation_type`, `observation_kind`, `risk_level`; bảng `attendance_records` (`UNIQUE (course_id, session_id, student_id)`, không có `attendance_course_session_idx` — TLR-3), `participation_events` (`client_event_id UNIQUE`), `student_observations`, `student_risk`; hai view `v_attendance_summary`, `v_student_360`; trigger `set_updated_at`. **Đúng một `ALTER TABLE`** (`class_sessions`: `attendance_done_at/by`, `UNIQUE (course_id, id)`, CHECK đi đôi). `Down` dùng `alter table` chữ thường nên `grep -c '^ALTER TABLE'` = 1. `v_student_360.study_seconds_7d` = `0` ở bản này (xem **cần PM** 1).
- **Store / sqlc** (`queries/today.sql`): `LockSession`, `UpdateSession`, `SessionHasAttendance`, `DeleteSession`; `ListSessions` thêm `version`, `attendance_done_at`, `has_attendance`; `TodayStaffPending` thêm cột `unmarked` (gộp vào truy vấn tổng hợp — **không** thêm truy vấn thứ 6 vì `TestTodayQueryBudget` ≤ 5).
- **`internal/course/sessions.go`**: `GenerateSessions` có `DryRun` (giao dịch `ReadOnly`, `GetCourse` thay `LockCourse`, không audit / outbox); trả `would_create` (đúng `session_no` + giờ của lần chạy thật) và `skipped:[{date, reason EXCLUDED|DUPLICATE}]`; audit `class_sessions` / `generate` (trước là `course` / `sessions_generated`, không test nào ghim). `UpdateSession` (version → `*SessionVersionConflictError`, trùng `starts_at` → `ErrSessionDuplicate`, audit `update`), `DeleteSession` (có điểm danh → `ErrSessionHasAttendance`, khoá ngoại 23503 là lưới cuối; audit `delete`); cả hai khoá lớp rồi khoá dòng buổi, lớp lưu trữ → `409 COURSE_ARCHIVED`. `auditEntity` tổng quát hoá `audit`.
- **HTTP** (`coursehttp`): `POST …/sessions/generate?dry_run=true` bỏ qua middleware `Idempotency-Key` (`idemUnlessDryRun`, TLR-12), `PUT` / `DELETE …/sessions/{sid}` (`Staff`), `GET …/sessions` thêm `version` và `attendance:{done_at, has_records}` **chỉ cho nhân viên**. Mã lỗi mới `SESSION_DUPLICATE`, `SESSION_HAS_ATTENDANCE` (409) ở `apierr`, enum `Error.code` của `openapi.yaml`, `ApiError.ts`, `vi.ts`. `openapi.yaml`: dry_run, `SessionGeneratePreview`, `SessionUpdateRequest`, `CourseSession`, hai thao tác mới.
- **`Hôm nay`** (`internal/today`): `KindSessionUnmarked`, `TierSessionUnmarked = 12` (TLR-10; 15 đã là `TierExamGradeError`), `SessionUnmarkedGrace = 24h`; mục "Điểm danh buổi {no} đang diễn ra · {mã lớp}" / "Buổi {no} đã kết thúc, chưa điểm danh · {mã lớp}" (`Overdue` sau `ends_at`), ≤ 3 mục mỗi lớp, chỉ lớp ACTIVE, chỉ Giảng viên + TA (`allowed`). Bước "Tạo lịch buổi học" của `COURSE_SETUP` có `Href = /calendar?course=…&new=sessions`.
- **Frontend** (`features/calendar`): `SessionsDrawer` (`Drawer` + `Field`, `FilterChips` cho Thứ, `DataTable` xem trước, `ActionList` + `OverflowMenu` cho từng buổi, sửa tại chỗ, xoá có `UndoLine` 5 s, `Xoá` khoá + lý do khi đã có điểm danh); `sessionsApi.ts` (`useIdempotentMutation`: bấm `Tạo N buổi` = khoá mới, thử lại sau lỗi mạng = cùng khoá); nút `Tạo lịch buổi học` (nút chính của vùng, chỉ Staff; `Thêm sự kiện` thành nút thường); `?new=sessions` mở sẵn ngăn kéo. 375 px: ngăn kéo 100 vw.

## File đổi
Backend: `db/migrations/00013_crm.sql`, `db/migrations_test.go`, `api/openapi.yaml`, `internal/store/queries/today.sql` (+ `today.sql.go`, `models.go` sinh lại), `internal/course/{admin,sessions}.go`, `internal/httpapi/apierr/apierr.go`, `internal/httpapi/coursehttp/{admin,handler,sessions}.go`, `internal/today/{today,providers}.go`.
Test: `internal/store/crm_schema_test.go` (mới), `internal/store/course_test.go` (dòng 28), `internal/course/sessions_crm_test.go` (mới), `internal/today/session_unmarked_test.go` (mới), `internal/contract/{contract_test,scenarios_test}.go`.
Frontend: `features/calendar/{SessionsDrawer,sessionsApi}.tsx|ts` (mới), `RealCalendar.tsx`, `Calendar.module.css`, `calendarApi.ts` (`export WD`), `shared/data/ApiError.ts`, `shared/i18n/vi.ts`, `e2e/calendar.spec.ts`, `e2e/support/cal-fixtures.ts` (thêm route `…/sessions**`, ghi `Idempotency-Key`).
Tài liệu: `docs/PROGRESS.md` (ánh xạ `00013 crm`), `docs/sprints/7/proposals.md` (#22; lúc commit gọi là #20, #23 = "#21").

## Test cũ phải đổi (đúng theo proposal ACCEPTED)
| Test | Đổi | Căn cứ |
| --- | --- | --- |
| `db/migrations_test.go:70` | version 12 → 13 | #11 |
| `contract_test.go` | 187 → 189 thao tác; hai thao tác mới vào bảng `want` của `TestSpec_SecurityDeclared` | #11 (đúng hai route `PUT` / `DELETE …/sessions/{sid}` của SRS 6.1) |
| `contract/scenarios_test.go` | thêm lời gọi cho từng status mới (`TestSpec_EveryDocumentedStatusExercised`) | #11 |
| `store/course_test.go:28` | thêm `attendance_done_at`, `attendance_done_by` vào cột `class_sessions` | #5 (ALTER `class_sessions` đã ACCEPTED; AC3 buộc ghim cột thêm) |

Test sprint 2 về `generate` (`sessions_test.go`) **không đổi** và vẫn xanh.

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/store -run 'TestSchemaAttendance|TestAttendanceCrossCourseFK|TestSchemaParticipation|TestParticipationChecks|TestSchemaObservations|TestSchemaStudentRisk|TestSchemaClassSessionsCrm|TestViewsColumns|TestIndexesStartWithCourseID|TestMigration00013SingleAlter' -v
go test -count=1 ./db/...                         # round trip up / down / up, version 13
go test -count=1 -race ./internal/course -run 'TestGenerate|TestSession|TestMakeup' -v
go test -count=1 -race ./internal/today -run 'TestSessionUnmarked|TestCourseSetupSessionsStepHref|TestTodayQueryBudget' -v
go test -count=1 -race -tags testroutes ./internal/contract/... ./internal/httpapi/...
sqlc generate && sqlc diff                          # sạch
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test calendar.spec.ts today.spec.ts
bash ../scripts/ui-antipatterns.sh                  # rc=0
```
Ví dụ AC4 bằng tay (gateway chạy, `$TCH` = header giảng viên): `curl -sk -X POST "$Q/sessions/generate?dry_run=true" -H "$TCH" -H 'Content-Type: application/json' -d '{"weekdays":[1],"start_time":"09:00","end_time":"11:30","from":"2026-10-12","to":"2026-12-28","exclude_dates":["2026-11-02"]}'` → `would_create` 11 phần tử, `skipped:[{"date":"2026-11-02","reason":"EXCLUDED"}]` (đã kiểm bằng `TestGenerateDryRunNoWrites`).

## Test đã chạy và kết quả (local, colima)
`./internal/store` ok · `./db` ok · `./internal/course` ok (toàn gói, 120 s) · `./internal/today` ok · `./internal/exam -run Today|…` ok · `./internal/contract` + `./internal/httpapi/...` ok (gồm `TestAllCourseRoutesGuarded`, `TestRouteSpecParity`) · `golangci-lint` 0 issue · `go vet` thường / `testroutes` / `integration` sạch · `sqlc diff` sạch · `playwright calendar.spec.ts today.spec.ts` desktop + mobile: xanh (14 + 39) · `ui-antipatterns` rc=0 · `eslint .` sạch · `tsc --noEmit` sạch.

## AC tự đánh giá
AC1 ✓ · AC2 ✓ · AC3 ✓ (`TestIndexesStartWithCourseID` kiểm mọi chỉ mục phức hợp của 4 bảng mới) · AC4 ✓ · AC5 ✓ (với **cần PM** 2: `skipped` ở 201 giữ là số) · AC6 ✓ · AC7 ✓ · AC8 ✓ (`TestSessionsRBACMatrix`: TA ✓ / GV ✓ / STUDENT, ADMIN, ngoài lớp, GV lớp khác 403, không JWT 401; `TestSessionCrossCourse404`; `TestSessionsArchived`; `TestAllCourseRoutesGuarded` xanh) · AC9 ✓ (e2e `generate sessions preview and create`, `edit delete session`, `…375`; bước tick là `staff_test.go:216` đã có + `TestCourseSetupSessionsStepHref`) · AC10 ✓ · AC11 ✓ (`TestGenerateConcurrentTwoKeys`; e2e `generate sessions offline retry`: 4 lần gửi cùng một khoá).

## Cần PM / nợ
1. **`v_student_360.study_seconds_7d`**: SRS 5.6 đặt view ở `00013` nhưng bảng `study_time_daily` thuộc `00014` (SRS 5.7). `00013` dựng view với `0::bigint`; **US-P5-03 phải `CREATE OR REPLACE VIEW`** cùng tên / kiểu cột (và `Down` của `00014` trả lại bản `0`). Ghi ở proposals **#20 (a)**.
2. **`skipped` ở `201`** vẫn là số (hợp đồng P2, ba test sprint 2 ghim), AC5 viết như mảng — proposals **#20 (b)**. **`TOO_MANY_SESSIONS`** vẫn là `422 VALIDATION_FAILED` + `details[].code`, số "74 / 60" nằm trong `message` — proposals **#20 (c)**. Làm theo hiện trạng, chờ PM.
3. Dòng `Kiểm:` của spec dùng `-tags integration`; gói `internal/course`, `internal/today`, `internal/store` không có thẻ đó (testcontainers luôn chạy), nên test mới nằm cùng chỗ với test cũ, chạy bằng lệnh trên.
4. `SESSION_UNMARKED_GRACE` là hằng `today.SessionUnmarkedGrace` (24 h), chưa là biến môi trường — `service.go` đang đặt `SLA` cứng cùng kiểu; thêm env khi cần chỉnh.
5. Bấm `Esc` khi menu `…` đang mở trong ngăn kéo đóng luôn cả ngăn kéo (hành vi chung của `Drawer` + `Popover`, không riêng story này); e2e đóng menu bằng bấm lại nút.
6. `PUT …/sessions/{sid}` là **thay toàn bộ**: bỏ trống `room` / `topic` = xoá (đã ghi ở `openapi.yaml`).
7. Chưa chạy toàn bộ `go test ./...` trên máy (chỉ các gói chạm tới); CI chạy đủ.
