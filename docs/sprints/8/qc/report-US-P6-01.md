# QC report — US-P6-01 (migration `00016_gradebook`, ràng buộc DB, nguồn điểm PE / P5) · Kết luận: **PASS** (chấm lại 2026-10-31 theo `proposals.md` #8 ACCEPTED; vòng 1: FAIL vì AC7 chưa có route)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `12fa90bf` (đã gồm `ddbc8948` US-P6-01, `7df6eff3`, `859ca52d`). Bộ TC: `tc-US-P6-01.md` (không sửa chữ TC). Handoff: `dev-US-P6-01.md`. DB trống riêng của QC: container `qc-p6-pg` (`pgvector/pgvector:pg18`, cổng 55432, đã/sẽ xoá sau lượt), `goose v3.28.0` cài ngoài repo. **Không có stack gateway** (US này chưa có route) nên TC qua API là "KHÔNG KIỂM ĐƯỢC" (ghi rõ bên dưới), thay bằng đẩy dữ liệu xấu trực tiếp vào DB.

Luật QC: "KHÔNG KIỂM ĐƯỢC" tính là FAIL. AC7 là AC về route; dev tự ghi ở handoff "story này chưa có route … `TestRBACMatrix` được thêm dần ở US-P6-03…09". Đề nghị PM chuyển việc kiểm AC7 sang story route cuối (xem `proposals.md` #8): QC sẽ chạy lại TC-36…39 khi đủ route (US-P6-09).

## 0. Chấm lại theo #8 và #7 (PM, 2026-10-31)
- **#8 ACCEPTED:** AC7 (phân quyền, `TestRBACMatrix`, `TestAllCourseRoutesGuarded`, TC-P6-01-36…39) **chuyển nơi kiểm** sang US-P6-09 (ma trận đủ 20 route); chữ AC giữ nguyên. AC1–AC6 đã PASS ở vòng 1 và không có commit mới đụng `00016`, `internal/grade/inputs.go` hay `grade.sql` sau `12fa90bf` (`git log 12fa90bf..HEAD -- backend-go` rỗng) → kết luận **PASS**. Nợ chuyển: QC chạy TC-36…39 + `p6-rbac.sh` ở US-P6-09; dev ghi vào mỗi handoff story route hàng `TestRBACMatrix` đã thêm.
- **#7 ACCEPTED:** BUG-P6-01-2 (sửa `db/migrations_test.go` trước ACCEPTED) giữ nguyên là **lỗi quy trình** (PM ghi: lần thứ 3 từ sprint 7); thay đổi tự nó được chấp nhận, không hoàn tác.

## 1. Cổng nghiệm thu đã chạy (vòng 1)
| Lệnh | Kết quả |
| --- | --- |
| `goose up` (DB trống, 16 bước) → `down` → `up` → `status` | PASS: `OK 00016_gradebook.sql (34 ms)` → `down` OK (20 ms) → `up` → `version: 16`; sau `down` còn **0** bảng `grade_*`/`gradebooks`/`final_grade_snapshots` và **0** kiểu enum `grade%`/`gradebook_status` |
| `grep -ci '^[[:space:]]*ALTER TABLE' db/migrations/00016_gradebook.sql` | PASS: `0` |
| `sqlc diff` (sqlc ở `~/go/bin`) | PASS: rc=0, rỗng |
| `go vet ./internal/grade/... ./internal/store/...` | PASS |
| `p6-go-tests.sh ./internal/store "TestSchemaGradebook\|…9 test" -tags integration` | PASS (9/9, không SKIP) |
| `p6-go-tests.sh ./internal/grade "TestExamScore*\|TestInputs*\|TestCrossCourseIsolation\|TestExamFromOtherCourseRejected\|TestLeftStudentExcluded" -tags integration` | PASS (12/12, không SKIP); `TestExamScoreMatchesSeedCSV` đọc `expected_exam_scores.csv` (24 dòng `MCQ`) |
| `go test ./internal/activity -run 'TestGradeNeverImportsActivity\|TestGradeQueriesNeverReadStudyTime'` | PASS ở `12fa90bf`; **FAIL ở `ddbc8948`** (xem BUG-P6-01-1) |
| `go test ./db/...` (`TestMigrations_RoundTrip`) | PASS ở `12fa90bf`, sau khi dev sửa số `15 → 16` (xem BUG-P6-01-2) |
| `bash docs/sprints/8/qc/scripts/p6-schema-probe.sql` (psql, 29 ca, giao dịch ROLLBACK) | PASS 29/29 |
| CI | run `41184610941` (`12fa90bf`) đang chạy lúc QC kiểm; run trước của `ca4d8b08` đỏ (WIP) |

## 2. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 up/down/up | PASS | như bảng trên; `status` thấy `00016_gradebook.sql` đã áp dụng; `00009` vắng không báo lỗi |
| 02 không `ALTER` | PASS | `0` (cả `grep -ci 'ALTER TABLE'` toàn tệp) |
| 03 cột đúng SRS 5.1 | PASS | `version int` có ở `grade_schemes`, `grade_entries`, `gradebooks`; `final_grade_snapshots.unlock_reason`, `gradebooks.{unlock_reason, unlocked_by, unlocked_at}` có trong CREATE; `grade_entries` không có `published`; `grade_items.{published_at, source, exam_id, scheme_id, key}` có; `weight`, `max_score`, `score` đều `numeric(5,2)`; PK `uuid`, trừ `gradebooks.course_id` |
| 04 `PROGRESS.md` | PASS | dòng 88 `00016 | gradebook | P6 …` |
| 05 `down` sạch | PASS | 0 bảng, 0 enum sau `down`; `up` lại không "already exists" |
| 06 một DRAFT / một CONFIRMED | PASS | hai `DRAFT` cùng lớp → `23505`; hai `CONFIRMED` → `23505`; nhiều `SUPERSEDED` được; scheme lớp 2 độc lập (probe 06a–e) |
| 07 weight ∈ [0;100] | PASS | `-0,01` và `100,01` → `23514`; `0` và `100` được (probe 07a–d) |
| 08 `(scheme, key)` duy nhất | PASS | trùng → `23505`; component gắn scheme lớp khác → `23503` |
| 09 `grade_items` | PASS | `EXAM` thiếu `exam_id` / `MANUAL` có `exam_id` → `23514`; `EXAM` hợp lệ được; `max_score 0` và `-1` → `23514` |
| 10 `grade_entries` | PASS | `score -0,01` → `23514`; `0` được; trùng `(item, student)` → `23505` |
| 11 FK phức hợp chéo lớp | PASS | entry lớp 2 → item lớp 1, item lớp 2 → bài thi lớp 1, item lớp 2 → scheme lớp 1, entry lớp 1 → sinh viên chỉ ở lớp 2: cả 4 `23503` |
| 12 chỉ mục bắt đầu bằng `course_id` | PASS (có ghi chú) | mọi chỉ mục thường bắt đầu bằng `course_id`. Sáu chỉ mục **duy nhất** không bắt đầu bằng `course_id`: `grade_components (scheme_id, key)`, `grade_entries (item_id, student_id)`, `grade_items (scheme_id, exam_id)`, `grade_items (scheme_id, key)`, `grade_rules (scheme_id, kind) WHERE kind<>'FAIL_CONDITION'`, `grade_schemes (extract_job_id)` — đều do chính US-P6-01 AC2 / `SRS.md` 5.1 chỉ định; không có ngoại lệ ngoài spec. Mọi bảng có `course_id NOT NULL` (7/7) |
| 13 test Go store + `sqlc` | PASS | 9/9, `sqlc diff` rỗng |
| 14 `UPDATE`/`DELETE`/`TRUNCATE` snapshot | PASS | cả ba `P0001`, thông điệp `snapshot is immutable` (đọc `prosrc` của hàm trigger) |
| 15 trigger | PASS | `final_grade_snapshots_immutable_row` (mức dòng, BEFORE) và `final_grade_snapshots_immutable_truncate` (mức câu lệnh, BEFORE TRUNCATE, `tgtype & 32`) |
| 16 `(course, student, version)` duy nhất | PASS | trùng → `23505`; version mới được; `UPDATE` bản mới vẫn `P0001` |
| 17 `EXPLAIN` không `Seq Scan` | PASS | 200 dòng: planner chọn Seq Scan (bảng quá nhỏ, cost 7); **10.000 dòng** (1.000 sinh viên × 10 phiên bản): `Index Scan using final_grade_snapshots_version_idx`, `Index Cond: (course_id = … AND version = 2)` |
| 18 gỡ trigger bằng vai trò ứng dụng | KHÔNG KIỂM ĐƯỢC (ghi nhận) | QC chỉ có vai trò chủ sở hữu/superuser của container thử, không có vai trò ứng dụng tách riêng của gateway (chưa dựng stack). Ghi vào "rủi ro": nếu vai trò ứng dụng là chủ bảng thì `ALTER TABLE … DISABLE TRIGGER` gỡ được bất biến. TC không tính vào AC |
| 19 test snapshot | PASS | gộp trong TC-13 |
| 20 điểm PE qua API | KHÔNG KIỂM ĐƯỢC qua API (chưa có route `…/gradebook`) | thay bằng `TestExamScorePublishedOnly`, `TestExamScoreMissingIsNotZero` PASS — là bằng chứng của dev, QC chưa kiểm độc lập. Kiểm lại ở US-P6-06 / US-S8-SEED |
| 21–23 bài thi chưa công bố → thiếu; sửa điểm → đổi ngay; thang | một phần | `TestExamScoreFollowsAdjust`, `TestExamScoreScale` PASS; TC-22 DB-level: `select count(*) from grade_entries … source='EXAM'` — không có đường ghi (item `EXAM` không có entry) chưa kiểm vì chưa có route ghi |
| 24 `Kiểm:` Go AC4 | PASS | 5/5 test AC4 PASS |
| 25–27 P5 đầu vào, `EXCUSED`/`LATE`, `ACTIVE` | PASS bằng test (`TestInputsFromP5`, `TestInputsMatchExpectedCRM`, `TestInputsExcusedLateNotAbsent`, `TestInputsActiveOnly`); qua giải trình trên API: chưa kiểm được | A/B/C khớp `expected_crm.csv` là trong test của dev; QC kiểm lại độc lập ở US-S8-SEED |
| 28 `grep` bảng thời gian học | PASS ở `12fa90bf` (rỗng) | ở `ddbc8948` `grade.sql` và `inputs.go` có chữ `study_time_daily` / `student_risk` trong **chú thích** → xem BUG-P6-01-1 |
| 29 test P5 | PASS | 4/4 |
| 30 D17 qua API | KHÔNG KIỂM ĐƯỢC qua API; PASS ở mức mã | `go list`/`grep` không có `internal/activity` trong `internal/grade`; `TestGradeNeverImportsActivity`, `TestGradeQueriesNeverReadStudyTime` PASS |
| 31–32 không lẫn lớp qua API | KHÔNG KIỂM ĐƯỢC qua API | DB-level: probe 11a–d PASS; Go `TestCrossCourseIsolation` PASS |
| 33 bài thi lớp khác không nối | PASS (DB-level `23503`); `422` qua API: KHÔNG KIỂM ĐƯỢC | probe 11b |
| 34 snapshot cũ khi rời lớp | KHÔNG KIỂM ĐƯỢC | cần `finalize` (US-P6-08) |
| 35 test cô lập | PASS | `TestCrossCourseIsolation`, `TestExamFromOtherCourseRejected`, `TestLeftStudentExcluded` |
| 36–39 RBAC / `CourseAccessGuard` / `openapi.yaml` | **KHÔNG KIỂM ĐƯỢC (chưa có route)** | `TestRBACMatrix` chưa có; `grep -n 'grade-scheme\|gradebook\|grade-entries\|grade-items\|me/grades' backend-go/api/openapi.yaml` → chưa có route sổ điểm. Dev đã ghi nợ ở handoff |

## 3. AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 migration dạng cuối | PASS | TC-01…05 |
| AC2 ràng buộc cấp DB | PASS | TC-06…13 (29/29 ca probe + 9/9 Go) |
| AC3 snapshot bất biến | PASS | TC-14…17, 19 (TC-18 ghi nhận) |
| AC4 điểm bài thi PE | PASS (mức test của dev; API: chờ US-P6-06) | 5/5 test Go PASS; QC chưa kiểm được độc lập qua `…/gradebook` |
| AC5 điểm cộng / trừ từ P5 | PASS (như trên) | 4/4 test Go PASS; guard P5 PASS sau `7df6eff3` |
| AC6 không lẫn lớp | PASS (DB-level + test Go) | probe 11a–d, 3/3 test Go; qua API chờ route |
| AC7 phân quyền | **KHÔNG KIỂM ĐƯỢC → FAIL** | chưa có route nào; xem proposals #8 |

## 4. Lỗi
- **BUG-P6-01-1 (Trung bình — đã sửa, ghi nhận):** commit `ddbc8948` làm đỏ test cũ `TestGradeQueriesNeverReadStudyTime` (`internal/activity/grade_rule_test.go:93`, "Should be empty, but was [../store/queries/grade.sql: nhắc student_risk …study_time_daily]"). Nguyên nhân: câu chú thích `-- … Không đọc study_time_daily, student_risk, student_observations.` ở `internal/store/queries/grade.sql:25` (kéo theo `grade.sql.go:177`) và `internal/grade/inputs.go:61`. Tái hiện: `git switch --detach ddbc8948 && cd backend-go && go test ./internal/activity -run TestGradeQueriesNeverReadStudyTime`. Đã sửa ở `7df6eff3` (commit mang nhãn US-P6-02, không phải US-P6-01); tại `12fa90bf` PASS. Handoff của US-P6-01 chỉ chạy `internal/grade` + `internal/store`, không chạy bộ test toàn cục trước khi sang story kế (luật "CI xanh trước khi sang story kế" của `dev.md`): CI của `ddbc8948` không xanh. **Việc đã làm đúng:** sửa chú thích, không sửa test.
- **BUG-P6-01-2 (Nghiêm trọng về quy trình, đã tự khai):** dev sửa test cũ `backend-go/db/migrations_test.go` (`require.Equal(t, 15, version)` → `16`, chú thích) **trước khi** PM ACCEPTED proposal #7 (commit `12fa90bf` ghi "#7, chờ PM"). Sprint 7 từng vi phạm hai lần; luật trong `dev.md`: "Test cũ chỉ sửa khi proposal ACCEPTED". Thay đổi tự nó đúng và cơ học (một số + chú thích, đúng như chú thích trong test dặn, lược đồ down+up vẫn so giống hệt, `git diff 4c86d357..HEAD -- db/migrations_test.go` đúng 2 dòng), nhưng QC không tự coi là đã hợp lệ: **chờ PM ACCEPTED**; nếu PM không chấp nhận thì hoàn tác và CI đỏ.
- **BUG-P6-01-3 (Thấp):** `AC7` không kiểm được vì spec đặt AC phân quyền theo route vào story không có route. Không phải lỗi dev; là thứ tự story. Đề nghị: proposals #8.

## 5. Kiểm phản mẫu / luật mở rộng / phân quyền
- Không có route mới, không UI mới: `ui-antipatterns` không áp dụng; không `fetch` trần.
- Luật 13: mọi chỉ mục thường bắt đầu bằng `course_id`; ngoại lệ là sáu UNIQUE do spec chỉ định (TC-12).
- Luật 14: cột `version` có ở bảng sửa đồng thời; `final_grade_snapshots` chủ ý không có `updated_at`/`version` (bảng bất biến, ghi ở handoff).
- Không thư viện mới (`git diff 4c86d357.. --name-only -- go.mod frontend/package.json` rỗng); không secret; không PII; không `float64` (`internal/grade` chỉ có chữ "float" trong chú thích `types.go:2`).
- `git diff 4c86d357.. -- '*_test.go'`: chỉ **thêm** `helpers_test.go`, `inputs_test.go`, `gradebook_schema_test.go`; test cũ bị sửa: duy nhất `db/migrations_test.go` (BUG-P6-01-2).

## 6. Rủi ro / ghi nhận cho PM
- Vai trò ứng dụng của gateway có thể là chủ bảng → có thể `DISABLE TRIGGER` trên `final_grade_snapshots`; kiểm quyền DB ở US-P6-08 (TC-P6-08-46 và `\dt+ owner`).
- `probe` SQL của QC (`docs/sprints/8/qc/scripts/p6-schema-probe.sql`) dùng được lại cho `00017` của P7 (đổi tên bảng).

## Đề nghị
1. PM quyết proposals #7 (test cũ) và #8 (chuyển kiểm AC7 sang US-P6-09).
2. Với #8 chấp nhận: coi US-P6-01 **PASS có điều kiện**, QC chạy lại TC-36…39 + `TestRBACMatrix` ở US-P6-09; dev ghi vào handoff mỗi story route đã thêm `TestRBACMatrix` hàng nào.
3. Dev: trước mỗi push, `scripts/go-test-all.sh` (không chỉ gói của story) để CI xanh trước story kế.
