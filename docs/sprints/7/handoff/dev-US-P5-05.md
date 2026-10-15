# DEV handoff — US-P5-05 (hồ sơ 360, ghi chú riêng, nhận xét tổng hợp AI, nhắn sinh viên)
Nhánh: `sprint/7-p4-p5`. Commit: `US-P5-05: …`. Số ghim: `#11` (contract 216 → 224: 8 thao tác mới); test cũ sửa theo `#29` (ACCEPTED, xem "Test cũ" dưới). Không migration mới (`student_observations`, `v_student_360`, `student_risk` đã có ở `00013` / `00015`).

## Đã làm
- **`internal/crm`**: `Profile` (một giao dịch: `audit_log` `entity='student_profile'`, `action='view'` rồi đọc; ghi audit lỗi → `503`, không trả dữ liệu), `StudentAttendance`, `observations.go` (CRUD, tác giả mới được sửa / xoá, `version` → `409`, `audit_log` không nội dung), `aisummary.go` (đúng MỘT `llm.Structured`, đầu vào chỉ là số liệu: đếm chuyên cần, phát biểu, phút học 4 tuần, hỏi / đăng mỗi tuần, mã lý do; kết quả quét `privacy.Detect` và che tên roster; hạn `AI_SUMMARY_TIMEOUT` 20 s → `504`, `AI_SUMMARY_PER_HOUR` 10 → `429`, chưa có dữ liệu → `409 NOT_ENOUGH_DATA` không gọi LLM), `outreach.go` + `escalation/outreach.go` (`CreateOutreach` dùng chung với P4: ticket `TEACHER_OUTREACH` `ANSWERED`, tin `STAFF_ANSWER`, outbox `escalation.answered`; `PII_IN_ANSWER` khi nêu sinh viên khác; `OUTREACH_PER_DAY` 3; `Idempotency-Key` bắt buộc).
- **HTTP** (`crmhttp`, guard `Staff`): `GET …/students/{uid}/profile` (`Cache-Control: no-store`), `GET …/attendance`, `GET/POST …/observations`, `PUT/DELETE …/observations/{oid}`, `POST …/ai-summary`, `POST …/outreach`. STUDENT / ADMIN → `403`; sinh viên lớp khác → `404`.
- **Cấu hình**: `AI_SUMMARY_TIMEOUT`, `AI_SUMMARY_PER_HOUR`, `OUTREACH_PER_DAY`. **OpenAPI**: 8 thao tác, contract 216 → 224.
- **Frontend** `features/students`: `RealProfile` (đầu trang, `StudentRiskSummary` chỉ khi WATCH / AT_RISK, hai hành động phụ, năm tab, mỗi lần mở đúng MỘT request `…/profile`: `staleTime 0`, `gcTime 0`, không tự gọi lại), `ProfileNotes` (nháp tự lưu `obs:<uid>`, `409` hiện bản của người kia), `ProfileOutreach`. **AC10**: xoá `StudentProfile.tsx`, `StudentsView.tsx` (mô phỏng); `features/students/**` không còn import `@/mock/*` (`grep -rn "@/mock" frontend/src/features/students | wc -l` → 0); `mock/roster.ts` còn đúng `rosterOf` (Gradebook dùng). Phiên không có lớp thật: câu "Chưa có lớp nào.".
- **Bổ sung vế còn nợ của P5-04 AC3**: `profile_perf_test.go` `TestProfileUnder3s` (lớp 1.000 và lớp 100 theo SRS 9.4: hồ sơ < 3 s kèm đúng một dòng audit; lớp 100: `CrmProfileGet`, `CrmProfileStudyWeeks` không `Seq Scan` trên năm bảng lớn).

## Test
- Go: `profile_test.go` (`TestProfileOpenWritesOneAuditPerCall`, `…FailClosedWhenAuditFails`, `…RBACMatrix`, `…NotInCourse404`, `TestObservationCRUD` / `Validation` / `VersionConflict` / `AuthorOnlyEdit`, `TestObservationsReadAudited`, `TestStudentAttendanceTab`, `TestAISummary*` ×7), `outreach_test.go` (`TestOutreach*` ×6), `internal/agent/notes_access_test.go` (`TestStudentAsksAboutOwnNotes`, `TestNoToolReadsObservations`), `integration/observation_prompt_test.go` (`TestPromptNeverContainsObservations`, canary `CANARY-OBS-4F`), contract kịch bản 8 thao tác.
- e2e `students.spec.ts`: `profile tabs and header`, `profile no kpi cards`, `profile panels per tab` (chuyển từ `panels.spec`, #29b), `profile open three times three requests`, `profile states and offline error`, `observation draft survives offline`, `observation conflict`, `ai summary edit then save`, `outreach reaches student chat`, `students no real class`.

## Test cũ (theo `#29`, ACCEPTED)
- `panels.spec.ts`: bỏ `"/students/sv-1"` khỏi `STAFF_ROUTES` (a); cắt vế `/students/[id]` khỏi ca "/grading/[submissionId]…", khẳng định năm tab chuyển nguyên văn sang `students.spec.ts` (b).
- `tokens.spec.ts` "diacritics" (c): **lệch một chi tiết so với #29**: đổi sang `/threads` chứ không `/gradebook`, vì `/gradebook` không có `<input>` (đo: chỉ `/threads`, `/documents` có) nên vẫn `input: false`. Thêm một dòng chờ ô nhập có mặt (`waitFor({ state: "attached" })`) vì `/threads` nạp lười. Mọi khẳng định giữ nguyên. Đề xuất #32(A) — PM đã ACCEPTED.

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1
go test -count=1 -race -tags testroutes ./internal/crm/... ./internal/escalation/... ./internal/agent/... ./internal/integration/... ./internal/contract/... ./internal/httpapi/...
sqlc generate && sqlc diff
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate
E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test students.spec.ts panels.spec.ts tokens.spec.ts --project=desktop
grep -rn "@/mock" src/features/students | wc -l      # 0
bash ../scripts/ui-antipatterns.sh                   # rc=0
```

## Cần PM / nợ
- `visual.spec.ts` 13 ca đỏ trên máy dev macOS 27 (cả `dev-ui`, tĩnh): lệch dựng ảnh của máy, không liên quan; chờ CI Linux.
- `docs/ARCHITECTURE.md` (BA): env `AI_SUMMARY_TIMEOUT`, `AI_SUMMARY_PER_HOUR`, `OUTREACH_PER_DAY`; 8 route hồ sơ.

## Vòng 2 (sửa theo `qc/report-US-P5-05.md` vòng 1)
Commit `US-P5-05: vòng 2 (BUG-A)`.
- **BUG-A (cao): `ai-summary` luôn `503`.** Gốc: `aisummary.go` xin `INSIGHT` + `NEAR_REALTIME`; mặc định của `INSIGHT` là `BATCH` và `llm.ResolveLane` chỉ cho HẠ làn (số làn lớn hơn mặc định) ⇒ `ErrBadLane` ⇒ `llm_audit` `INSIGHT | BATCH | error | BAD_LANE`, không lời gọi nào tới provider; test của vòng 1 dùng `Structurer` giả nên không bắt.
- **Sửa:** dùng task `UTILITY` (đã có, làn mặc định `NEAR_REALTIME`, SRS llm-gateway 4.3 ghi "tóm tắt, đặt tên, việc nhỏ"), không xin làn. Không đổi schema, không đổi `ResolveLane` hay bảng làn (các test ghim "không nâng INSIGHT" giữ nguyên). Hệ quả cho tài liệu: crm-360 AC5 / SRS 4.8.5 ghi "task `INSIGHT`" → BA sửa thành `UTILITY` (làn `NEAR_REALTIME` giữ nguyên). Tuyến mô hình của `UTILITY` chọn ở `/settings/llm` (dòng `UTILITY`), không còn dùng dòng `INSIGHT` cho việc này.
- **Test:** `profile_test.go` `TestAISummaryPayloadHasNoIdentity` đổi khẳng định task / làn theo trên (test sprint 7 của chính story, giữ mọi khẳng định về danh tính). Thêm `TestAISummaryThroughRealGateway`: `llm.Gateway` thật (registry + provider giả + `Auditor` ghi `llm_audit`) — có bản nháp, đúng MỘT lời gọi tới provider, `llm_audit` = `UTILITY | NEAR_REALTIME | ok`. Đã chứng minh đỏ trên mã cũ (INSIGHT + `NEAR_REALTIME` → lỗi) và xanh trên mã mới.
- TC-16…21 và E1 của QC giờ kiểm được trên gateway thật (`GET /_test/llm/stats` `fake_calls` tăng 1 mỗi lần soạn).
- Lệnh QC chạy: `go test -count=1 -race -tags testroutes ./internal/crm -run 'TestAISummary' -v`.
