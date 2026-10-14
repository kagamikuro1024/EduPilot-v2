# DEV handoff — US-P5-04 (danh sách sinh viên + at-risk theo tuần, lý do bằng lời, ngưỡng theo lớp, "Hôm nay")
Nhánh: `sprint/7-p4-p5`. Commit: `US-P5-04: …` (một commit). Số ghim: `#11` (contract 215 → 216: thêm `listStudents`); test cũ sửa theo `#14` (ACCEPTED): hai ca `mock course mapping` ở `class-join.spec.ts` đổi màn mẫu `/students` → `/gradebook`, giữ nguyên mọi khẳng định; theo `#30` (ACCEPTED): `TestNoQueryLinksByStudentCode` thêm đúng hai tên `CrmStudentsList`, `CrmStudentsCount` vào danh sách trắng (tìm MSSV tiền tố), giữ regex và mọi khẳng định khác; theo `#28` (ACCEPTED): `TestTodayQueryBudget` ca staff (hai ca giảng viên) 5 → 6, ca sinh viên giữ ≤ 5, mọi khẳng định khác giữ nguyên. Không migration mới (`student_risk`, `v_student_360` đã có ở `00013` / `00015`).

## Đã làm
- **`internal/crm/risk.go`** — hàm thuần `Evaluate(weeks, thresholds)` (tuần yếu = ≥ `signals_required` tín hiệu CÓ dữ liệu và thấp **và** neo chuyên cần thấp; `AT_RISK` hai tuần liền, `WATCH` chỉ tuần gần nhất; so sánh `<`; tỷ lệ chuyên cần bằng `decimal`, không chia: `ok < min × recorded`) và `Sentence(level, reasons)` (mẫu SRS 4.7.5; dựng từ `reasons` đã lưu nên hồ sơ P5-05 dùng lại, cùng nguồn với danh sách).
- **`internal/crm/riskjob.go`** — `RunCourse` (một lớp: đọc chuyên cần / hoạt động / thời gian học 4 tuần trọn bằng 3 truy vấn nhóm, một câu UPSERT `jsonb_to_recordset` cho cả lớp, `entered_at` giữ nguyên khi đã `AT_RISK`, xoá dòng của người không còn `ACTIVE`, outbox `crm.risk_updated` — một giao dịch), `RunAll`, `HandleSettingsChanged`, `RiskJob` (tick 1 phút; `now ≥ RISK_JOB_AT` ICT và `SET ep:crm.risk:done:{yyyymmdd} NX EX 48h`; lỗi → `DEL` dấu; leader `ep:lock:crm.risk`). `STUDY` có dữ liệu khi `tracking_notice_ack_at ≤ đầu tuần W` **và** (lớp không cho tắt **hoặc** `track_study_time`) — Q1.
- **`internal/crm/risk_settings.go`** + `course.PutSettings`: thân `risk{…}` (từng trường tuỳ chọn, trộn lên `courses.settings.risk` và mặc định `0,70 / 1 / 60 / 2 / 3`), khoảng sai → `422 risk.<trường>`, sai `version` → `409`, TA / STUDENT / ADMIN → `403` (guard sẵn), `audit_log` `course_risk_updated` (cũ / mới), outbox `course.settings_changed` → tính lại riêng lớp đó. `GET /courses/{id}` (staff) trả `settings.risk`.
- **`GET /courses/{id}/students?q=&filter=&cursor=&limit=`** (`crmhttp`, Staff): `v_student_360` + cờ `tracked`; khoá con trỏ `(risk_rank, vn_fold(tên), id)` (base64url, `422 INVALID_CURSOR`); `limit` mặc định 50, tối đa 100; `filter` lặp hoặc `a,b` = giao, lạ → `422`; `q` = tên không dấu (chứa, `strpos(vn_fold…)` nên `%` / `_` không phải ký tự đại diện) hoặc MSSV tiền tố; `total` riêng, `Cache-Control: no-store`; không cột điểm, không câu lý do; `risk:{level, computed_at}`.
- **"Hôm nay"**: `today.AttentionSource` (cài ở `crm.Service`: chỉ `internal/crm` đọc `student_risk`), mục `STUDENT_ATTENTION` (bậc 95, `Href /students?filter=attention&course=…`), khối `attention` (≤ 5, mới vào trước, cụm lý do ngắn "vắng 3/4 buổi; học ít") + `attention_total`; rỗng ⇒ `[]`. `today.Topics()` giữ 24 (hai topic mới đăng ký thẳng ở worker như `crm.attendance_saved`).
- **Worker**: `riskTask`; `crm.risk_updated` → `today.invalidate`; `course.settings_changed` → `HandleSettingsChanged`. Cấu hình `RISK_JOB_AT` (`HH:MM`, mặc định `02:00`).
- **OpenAPI**: `listStudents`, `StudentList` / `StudentRow`, `RiskThresholds(Input)`, `TodayAttention`, `TodayStaff.attention_total`; `CourseSettingsInput.risk`.
- **Frontend**: `features/students/RealStudents.tsx` (bảng ảo hoá, `useCursorList`, ô tìm + chip nằm trong URL `?q=&filter=` nên quay lại từ hồ sơ vẫn còn, `?filter=` từ Hôm nay bật chip; lỗi + `Thử lại`; rỗng + `Bỏ lọc`), `StudentsScreen` (lớp thật → bản thật; phiên mô phỏng giữ màn mô phỏng; "Tất cả lớp" → chọn một lớp), `members/RiskSection.tsx` (năm dòng nhãn + thanh + giá trị, lưu lạc quan, 409 hoàn lại, TA chỉ xem), khối "Lớp cần chú ý" ở `StaffToday`.

## Test
- Đơn vị `internal/crm/risk_test.go`: `TestRiskTable` (đủ 16 ca SRS 9.3), `TestRiskAnchorRequired`, `TestRiskInsufficientDataNotWeak`, `TestRiskIdempotent`, `TestRiskSentenceFromData` / `…WatchPrefix` / `…ConsistentWithCounts`.
- Tích hợp `risk_db_test.go`: `TestRiskEnteredAtKept`, `TestRiskIdempotentDB`, `TestRiskJobDoesNotTouchRemoved`, `TestRiskJobRunsOncePerDay`, `TestRiskJobCatchUpAfterWorkerDown`, `TestRiskJobFailureRetries`, `TestRiskJobLeaderOnce`, `TestSettingsRiskRange` / `RBAC` / `Version`, `TestRiskDefaultsWhenUnset`, `TestRiskRecomputedAfterSettings`, và (nợ P5-03) `TestRiskStudyNullWhenOptedOut`, `TestRiskStudyPerCourseConfig`, `TestRiskNotAckedStudyNull` (tự thử đỏ: bỏ vế `allow_study_time_opt_out` thì hai ca đầu đỏ).
- `students_test.go`: `TestStudentsListActiveOnly`, `…SortOrder`, `…CursorFull1000` (1.000 sinh viên, 10 trang, không trùng / sót, thứ tự giữ), `TestStudentsFilterAttention` / `Absent` / `Inactive` (giao hai bộ lọc), `TestStudentsSearchFoldedName`, `TestStudentsFilterInvalid`, `TestStudentsRoutesStudent403`, `TestStudentsNewCourseNoDivideByZero`, `TestStudentsBeforeFirstRiskRun`, `TestStudentsArchivedReadable`, `TestStudentsTrackedFlag`, `TestStudentNeverSeesRisk` (11 route của sinh viên không có `risk*` / `observation*` / `watch`).
- `students_perf_test.go` (dữ liệu `generate_series` đúng SRS 9.4: 1 lớp 1.000 + 19 lớp 100, 15 buổi, ≥ 280.000 dòng `study_time_daily`, ≥ 5.000 `participation_events`, `ANALYZE`; `EXPLAIN` chạy đúng câu SQL + tham số do tracer pool ghi lại): `TestStudents360ExplainNoSeqScan` (lớp 100: trang đầu / giữa / `attention` / `q` không `Seq Scan` trên 5 bảng lớn, không chạm `chat_messages`), `TestStudentsListP95Under300ms` (lớp 1.000, trang đầu: p50 ≈ 42 ms, p95 ≈ 49 ms), `TestStudents360LargeCoursePerf` (truy vấn 360 một sinh viên dùng chỉ mục).
- Kiến trúc: `TestSensitiveQueriesOnlyFromCRM` (quét AST: hàm sqlc chạm `student_risk` / `student_observations` / `v_student_360` chỉ gọi từ `internal/crm`; tự thử đỏ bằng một lời gọi giả ở `today`), `internal/agent` `TestAgentHasNoRiskOrObservationAccess` (tự thử đỏ).
- `internal/today/attention_test.go`: `TestStudentAttentionProvider`, `TestAttentionBlockMax5`, `TestAttentionHiddenWhenNone`, `TestAttentionStaffOnly`. Contract: kịch bản `listStudents` 200 / 401 / 403 / 404 / 422.
- e2e: `students.spec.ts` (`students list filters search keep state`, `students deep link filter`, `students states`, `students virtual cursor`, `students demo session`), `class-join.spec.ts` `risk thresholds` (2 ca), `today.spec.ts` `staff attention block` (2 ca), `a11y.spec.ts` `/students`.

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1
go test -count=1 -race -tags testroutes ./internal/crm -run 'TestRisk|TestSettingsRisk|TestStudents|TestStudent|TestSensitive' -v
go test -count=1 -race -tags testroutes ./internal/today -run 'Attention' -v
go test -count=1 ./internal/agent -run TestAgentHasNoRiskOrObservationAccess -v
go test -count=1 -race -tags testroutes ./internal/contract/... ./internal/course ./internal/store ./db/...
sqlc generate && sqlc diff
cd ../frontend && pnpm build:gate    # E2E_API_PORT=3412 E2E_PORT=3410 khi chạy playwright
pnpm exec playwright test students.spec.ts a11y.spec.ts -g 'students' --project=desktop
pnpm exec playwright test class-join.spec.ts -g 'risk thresholds|mock course mapping' today.spec.ts -g 'attention' --project=desktop
bash ../scripts/ui-antipatterns.sh   # rc=0
```

## Tự đánh giá AC
AC1, AC2, AC4, AC5 (phần câu sinh từ dữ liệu; route hồ sơ ở P5-05), AC6, AC7, AC8, AC9, AC10: làm + test. **AC3**: (a) và (b) trang đầu / một sinh viên đo được ở trên; vế **hồ sơ** (`GET …/profile` không `Seq Scan`, `TestProfileUnder3s`) thuộc US-P5-05 vì route chưa có — truy vấn 360 một sinh viên đã kiểm bằng `TestStudents360LargeCoursePerf`.

## Cần PM / nợ
- `docs/ARCHITECTURE.md` (BA): env `RISK_JOB_AT`; route `GET /courses/{id}/students`; topic `course.settings_changed`, `crm.risk_updated`.
- `DataTable` ảo hoá: ở `AdminUsers` `pagination.nextCursor` là hằng `"next"` nên bảng chỉ xin trang thứ hai (cơ chế "mỗi con trỏ xin đúng một lần" khoá theo giá trị). `RealStudents` truyền con trỏ thật nên đi hết nhiều trang; `AdminUsers` chưa sửa (ngoài phạm vi).
- Hồ sơ `/students/[id]` vẫn là màn mô phỏng tới US-P5-05; dòng danh sách và khối "Lớp cần chú ý" đã trỏ tới route đó.
