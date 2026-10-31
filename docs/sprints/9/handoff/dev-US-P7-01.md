# DEV handoff — US-P7-01 (lược đồ `00017_assignments`)
Nhánh `sprint/9-p7`. Spec: US-P7-01 AC1–AC5, SRS mục 5; TLR-3 A, 8, 11, 16; proposals sprint 9 #2, #7 (dev, cùng loại sprint 8 #7).

## Làm
- `db/migrations/00017_assignments.sql`: sáu bảng (`assignments`, `submissions`, `grading_results`, `grade_appeals`, `mail_inbound`, `integration_configs`) trong một giao dịch, `id uuidv7()`, mọi FK phức hợp có `course_id`, đúng hai `ALTER` (`grade_items_assignment_fkey`, `reminder_source ADD VALUE 'ASSIGNMENT'`; `grep -c '^ALTER '` = 2, không ADD COLUMN), triggers bất biến (`submissions.blob_key/sha256/extracted_text`, `grading_results.ai_scores/ai_total_before_late` → `P0001 "ai draft is immutable"`), cột sinh `needs_review`, chỉ mục bắt đầu bằng `course_id` (ngoại lệ: `grading_results_lease_idx`). `Down`: `DROP … CASCADE` ở `assignments` gỡ FK của `grade_items` (Down không có ALTER); xoá `reminder_log` nguồn `ASSIGNMENT`.
- **Lệch nhỏ so với SRS (ghi để PM biết):** SRS viết `ON DELETE RESTRICT`, AC4 đòi `23503`. `RESTRICT` trả `23001`; để khớp AC dùng mặc định `NO ACTION` (`23503`, giống `00016`) — vẫn từ chối xoá bản ghi còn được trỏ tới.
- sqlc generate (models); truy vấn đi cùng từng story dùng nó.
- Test (`internal/store/assignments_schema_test.go`): `TestSchemaAssignments`, `TestUuidv7Defaults`, `TestMigration00017Transactional`, `TestAssignmentsChecks`, `TestSubmissionsMatchedChecks`, `TestGradingResultsChecks`, `TestAiScoresImmutable`, `TestAiScoresFirstWriteAllowed`, `TestAppealsUnique`, `TestAssignmentsCrossCourseFK`, `TestGradeItemAssignmentFK`, `TestAssignmentDeleteRestrictedByGradeItem`, `TestIndexesStartWithCourseID_Assignments`, `TestMigration00017Shape`.
- Sửa test cũ (proposals #7): `TestMigration00016UpDownUp` gỡ 00017 trước (`down` ×2, `up` ×2), `db/migrations_test.go` 16 → 17.

## Chưa làm trong story này
- AC6 (`TestAssignmentsRBACMatrix`, `TestAllCourseRoutesGuarded` cho route P7): các route chưa tồn tại; ma trận được dựng cùng route và chạy đủ ở US-P7-10 (route cuối của L0–L3b) / US-P7-11.
