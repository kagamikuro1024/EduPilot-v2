# DEV handoff — US-P6-01 (migration `00016_gradebook`, store, nguồn điểm)
Nhánh `sprint/8-p6`. Spec: `docs/specs/FEAT-gradebook/` (SRS 5, 4.3.6).

## Đã làm
- `backend-go/db/migrations/00016_gradebook.sql`: 6 enum, 7 bảng (`grade_schemes`, `grade_components`, `grade_rules`, `grade_items`, `grade_entries`, `gradebooks`, `final_grade_snapshots`), chỉ CREATE (không `ALTER`). Gồm 4 bổ sung của Tech Lead (#3): `UNIQUE (course_id, id)` trên `grade_items`; chỉ mục `(course_id, version, student_id)` của snapshot; trigger `BEFORE TRUNCATE` cùng trigger theo dòng; `gradebooks` tạo lúc xác nhận (US-P6-04).
- `store/queries/grade.sql` (+ `sqlc generate`, `sqlc diff` sạch): `GradeActiveStudents`, `GradeInputEntries`, `GradeInputExams`, `GradeInputP5`, `GradeSchemeItemsMeta`. Mảng id dùng `::text[]::uuid[]` (PgBouncer, như `courses.sql`).
- `internal/grade/types.go` (kiểu công thức / luật / đầu vào, hằng `MaxScale`, `TotalWeight`, `DivPrecision = 20`, `NormalizeDigits = 10`), `internal/grade/inputs.go` (`ActiveStudents`, `LoadItems`, `LoadInputs`: đúng 3 truy vấn cho cả lô).
- `PROGRESS.md` mục "Ánh xạ migration": thêm `00016 gradebook`.

## Quyết định nhỏ (không đổi hành vi)
- `final_grade_snapshots` không có `updated_at` (bảng bất biến, không cần trigger `set_updated_at`); `numeric_grade` NOT NULL, CHECK [0;10].
- `grade_schemes.confirmed_by/at`: CHECK `(status = 'DRAFT') = (confirmed_at IS NULL AND confirmed_by IS NULL)` (SUPERSEDED giữ `confirmed_*`).
- `grade_items.assignment_id` chưa FK (P7, `00017`); CHECK `(source = 'ASSIGNMENT') = (assignment_id IS NOT NULL)` thêm để bảo toàn ý "nguồn ⇔ id".
- Test DB của gói `grade` chạy không cần thẻ `-tags integration` (cùng tiền lệ `internal/crm`); lệnh `-tags integration` ở spec vẫn chạy được.

## Kiểm
- `go test ./internal/store -run 'TestSchemaGradebook|TestGradebookOneDraftOneConfirmed|TestGradebookChecks|TestGradebookCrossCourseFK|TestIndexesStartWithCourseID|TestSnapshot|TestMigration00016' -v`: xanh (up → down → up sạch).
- `go test ./internal/grade`: `TestExamScore{PublishedOnly,FollowsAdjust,MissingIsNotZero,Scale,MatchesSeedCSV}`, `TestInputs{FromP5,ExcusedLateNotAbsent,ActiveOnly,MatchExpectedCRM}`, `TestManualEntriesAndClosed`, `TestCrossCourseIsolation`, `TestExamFromOtherCourseRejected`, `TestLeftStudentExcluded`: xanh.
- `golangci-lint run ./internal/grade/... ./internal/store/...`: 0 issue.
- AC7 (RBAC): story này chưa có route; `TestRBACMatrix` + `TestAllCourseRoutesGuarded` được thêm dần cùng route ở US-P6-03…09.

## Cần hỏi
Không.
