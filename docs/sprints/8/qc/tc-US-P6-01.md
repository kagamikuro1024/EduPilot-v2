# QC test case — US-P6-01 (migration `00016_gradebook`, ràng buộc DB, nguồn điểm PE / P5)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-P6-01 (AC1–AC7) + `SRS.md` mục 2, 3, 5, 6, 9; `proposals.md` #3 (lược đồ dạng cuối). Viết trước khi có code, không đọc code của dev (v1, 2026-10-31).

**Quy ước.** `GO` = `bash docs/sprints/8/qc/scripts/p6-go-tests.sh` (FAIL nếu test được nêu tên không tồn tại / bị SKIP / không in `--- PASS`). Biến shell `$Q $C1 $PSQL …` từ `scripts/p6-env.sh`. `DB_EMPTY` = DB trống riêng của QC (`createdb qc_p6_$$`; `DATABASE_URL` trỏ vào đó). Mọi truy vấn SQL dưới đây QC tự viết, KHÔNG sao từ test của dev.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P6-01-01 | AC1 | `DB_EMPTY`; `backend-go/db/migrations` có `00001`…`00016` | `cd backend-go && goose -dir db/migrations postgres "$DATABASE_URL" up && goose … down && goose … up && goose … status` | cả 3 lệnh thoát 0; `status` cho `00016_gradebook` = `Applied`; không `ERROR`; `00009` vắng là chủ ý (không báo thiếu) |
| TC-P6-01-02 | AC1 | như trên | `grep -ci '^[[:space:]]*ALTER TABLE' backend-go/db/migrations/00016_gradebook.sql` và `grep -ci 'ALTER TABLE' …` | `0` cả hai (không `ALTER` ở bất kỳ chỗ nào, kể cả trong khối `-- +goose Down` được tính riêng: nếu chỉ có trong Down → ghi chú, không FAIL; trong Up → FAIL) |
| TC-P6-01-03 | AC1 | `DB_EMPTY` đã `up` | `psql -tAc "select table_name,column_name,data_type,column_default from information_schema.columns where table_name in ('grade_schemes','grade_components','grade_rules','grade_items','grade_entries','final_grade_snapshots','gradebooks') order by 1,2"` rồi đối chiếu với bảng ở `SRS.md` 5.1 | đủ cột của SRS 5.1: `version int NOT NULL DEFAULT 1` có ở `grade_schemes`, `grade_entries`, `gradebooks`; `final_grade_snapshots.unlock_reason` và `gradebooks.unlock_reason/unlocked_by/unlocked_at` có trong CREATE; `grade_entries` **không** có cột `published`; `grade_items.published_at`, `source`, `exam_id`, `scheme_id`, `key` có; `grade_components.weight numeric(5,2)`; mọi PK `uuid` (trừ `gradebooks.course_id`); QC lập bảng tick từng cột, thiếu một → FAIL |
| TC-P6-01-04 | AC1 | repo | `grep -n '00016' docs/PROGRESS.md` | có dòng ánh xạ `00016 gradebook` ở mục "Ánh xạ migration" |
| TC-P6-01-05 | AC1 | `DB_EMPTY` đã `up` | `goose down` rồi `psql -tAc "select count(*) from pg_class where relname in ('grade_schemes','grade_components','grade_rules','grade_items','grade_entries','final_grade_snapshots','gradebooks')"`; sau đó `up` lại | sau `down`: 0; kiểu enum `grade_scheme_status`, `grade_group`, `grade_rule_kind`, `grade_item_source`, `grade_entry_source`, `gradebook_status` cũng biến mất (`select count(*) from pg_type where typname like 'grade%' or typname='gradebook_status'` → 0); `up` lại sạch (không "type already exists") |
| TC-P6-01-06 | AC2 | `DB_EMPTY` + seed hoặc dữ liệu tay | Chèn 2 `grade_schemes` `DRAFT` cùng `course_id` | lỗi `23505` ở dòng thứ hai; một `DRAFT` + một `CONFIRMED` + một `SUPERSEDED` + một `SUPERSEDED` nữa → thành công (nhiều `SUPERSEDED` được); hai `CONFIRMED` → `23505` |
| TC-P6-01-07 | AC2 | có một `grade_schemes` | `INSERT grade_components` weight = `-0.01`, `100.01`, `0`, `100` | `-0.01` và `100.01` → `23514`; `0` và `100` thành công (TLR-9: trọng số 0 hợp lệ) |
| TC-P6-01-08 | AC2 | có thành phần key `midterm` | chèn lại `(scheme_id, key)='midterm'` | `23505` |
| TC-P6-01-09 | AC2 | có `exams` của lớp 1 | `INSERT grade_items` `source='EXAM'` với `exam_id` NULL; `source='MANUAL'` với `exam_id` khác NULL; `source='EXAM'` với `exam_id` hợp lệ; `max_score = 0` và `-1` | hai ca đầu `23514`; ca ba thành công; `max_score ≤ 0` → `23514` |
| TC-P6-01-10 | AC2 | có item MANUAL | `INSERT grade_entries` `score=-0.01` ; `score=0` ; cùng `(item_id, student_id)` hai lần | `-0.01` → `23514`; `0` thành công; lần hai cùng cặp → `23505` |
| TC-P6-01-11 | AC2 | hai lớp C1, C2 seed; một item của C1; một sinh viên chỉ ở C2 | `INSERT grade_entries(course_id=C2, item_id=<item của C1>, student_id=…)`; `INSERT grade_items(course_id=C2, exam_id=<bài thi của C1>)`; `INSERT grade_items(course_id=C2, scheme_id=<scheme của C1>)`; `INSERT grade_entries(course_id=C1, item của C1, student_id=<sv chỉ ở C2>)` | cả 4 lỗi `23503` (FK phức hợp), không dòng nào được ghi |
| TC-P6-01-12 | AC2 | `DB_EMPTY` đã `up` | `psql -tAc "select indexdef from pg_indexes where tablename in (<7 bảng>) and indexname not like '%pkey'"` | mọi chỉ mục **không phải khoá chính** của 7 bảng có cột đầu là `course_id` (trừ chỉ mục duy nhất nối hẹp theo luật SRS 5.1 — đối chiếu từng dòng, ngoại lệ nào không có trong SRS → FAIL); mọi bảng có cột `course_id` NOT NULL |
| TC-P6-01-13 | AC2 | lệnh `Kiểm:` của US | `GO ./internal/store "TestSchemaGradebook|TestGradebookOneDraftOneConfirmed|TestGradebookCrossCourseFK|TestGradebookChecks|TestIndexesStartWithCourseID" -tags integration`; `cd backend-go && sqlc generate && sqlc diff; echo rc=$?` | `RESULT: PASS`; `sqlc diff` rc=0, rỗng |
| TC-P6-01-14 | AC3 | có một dòng `final_grade_snapshots` (chèn bằng vai trò ứng dụng — `psql` với user của gateway, không phải superuser) | `UPDATE final_grade_snapshots SET numeric_grade=1`; `DELETE FROM final_grade_snapshots`; `TRUNCATE final_grade_snapshots` | cả ba bị từ chối, `SQLSTATE P0001`, thông điệp có "snapshot is immutable"; số dòng không đổi |
| TC-P6-01-15 | AC3 | như trên | `select tgname, tgtype from pg_trigger where tgrelid='final_grade_snapshots'::regclass and not tgisinternal` | có ≥ 1 trigger mức dòng (BEFORE UPDATE OR DELETE) **và** ≥ 1 trigger mức câu lệnh BEFORE TRUNCATE (`tgtype & 32 = 32`) |
| TC-P6-01-16 | AC3 | như trên | chèn `(course_id, student_id, version)` trùng; chèn hợp lệ version mới | trùng → `23505`; hợp lệ → thành công; `UPDATE` lên bản vừa chèn vẫn bị chặn |
| TC-P6-01-17 | AC3 | 100 sinh viên × 2 version, `ANALYZE` | `EXPLAIN select * from final_grade_snapshots where course_id=:c and version=2 order by student_id limit 100` | không `Seq Scan`; dùng chỉ mục `(course_id, version, student_id)` |
| TC-P6-01-18 | AC3 | superuser | Thử `ALTER TABLE final_grade_snapshots DISABLE TRIGGER ALL` bằng **vai trò ứng dụng** (không superuser) | bị từ chối (không phải chủ bảng / quyền) — ghi nhận; nếu vai trò ứng dụng là chủ bảng và gỡ trigger được → ghi vào mục "Rủi ro" của báo cáo (không FAIL AC, vì AC chỉ nói UPDATE/DELETE/TRUNCATE) |
| TC-P6-01-19 | AC3 | lệnh `Kiểm:` | `GO ./internal/store "TestSnapshotImmutable|TestSnapshotTruncateBlocked|TestSnapshotUniqueVersion|TestSnapshotVersionIndex" -tags integration` | `RESULT: PASS` |
| TC-P6-01-20 | AC4 | seed; bài thi PE tuần 9 (`Kiểm tra tuần 9 — Mật mã`, trắc nghiệm) đã `PUBLISHED`, lớp 1; item EXAM `weekly` nối bài đó | Đọc điểm item `weekly` của A, B, C qua `GET $Q/gradebook?limit=100` (TEACHER) | A 10,00 · B 8,00 · C: **không có lượt GRADED** → quy chế 1.4 + bài `PUBLISHED` → 0 (hiển thị 0, giải trình "không dự → 0 theo quy chế") — đối chiếu `seed/expected_exam_scores.csv`; 24 dòng `MCQ` khớp từng dòng (QC so bằng script `jq`/`diff`) |
| TC-P6-01-21 | AC4 | một bài thi MCQ của lớp 1 chuyển `status` từ `PUBLISHED` sang chưa công bố (hoặc dùng bài thi khác chưa công bố, có item EXAM nối) | Đọc điểm item EXAM qua `…/gradebook` | các ô là **thiếu** (không phải 0; hiển thị "Chưa công bố"/trống), sinh viên → `INCOMPLETE` (`GET …/gradebook/preview` trạng thái "Thiếu điểm") |
| TC-P6-01-22 | AC4 | bài thi đã công bố | Giảng viên sửa điểm sinh viên X của bài thi (PE: `adjusted_score` mới) rồi đọc lại `…/gradebook` | ô của X đổi ngay sang giá trị mới (không cần chạy lại gì); `grade_entries` **không** có bản sao điểm thi (`select count(*) from grade_entries e join grade_items i on i.id=e.item_id where i.source='EXAM'` → 0) |
| TC-P6-01-23 | AC4 | bài thi `max_score=10`; item EXAM `max_score=20` (thử tạo công thức nháp tay qua `PUT …/grade-scheme` hoặc dùng Go test) | điểm thi 8,0 | điểm hiển thị 16,00 (8 ÷ 10 × 20); decimal, không sai số (QC thử 7,33 ÷ 10 × 30 = 21,99) |
| TC-P6-01-24 | AC4 | lệnh `Kiểm:` | `GO ./internal/grade "TestExamScorePublishedOnly|TestExamScoreFollowsAdjust|TestExamScoreMissingIsNotZero|TestExamScoreScale|TestExamScoreMatchesSeedCSV" -tags integration` | `RESULT: PASS`; `TestExamScoreMatchesSeedCSV` thật sự đọc 24 dòng (QC `-v` xem log, không chỉ PASS rỗng) |
| TC-P6-01-25 | AC5 | seed lớp 1 | So đầu vào P5 của A, B, C: truy vấn QC tự viết `select sum(points) filter (where kind in ('SPEAK','BONUS')), abs(sum(points) filter (where kind='PENALTY')), count(*) filter (where status='ABSENT')` trên `participation_events` / `attendance_records` rồi so với giải trình `GET $Q/gradebook/students/$SVB_ID/explain` | A: cộng 3,00 · vắng 0; B: 0,50 · vắng 2; C: 0,25 · vắng 5 (khớp `seed/expected_crm.csv`); giải trình dùng đúng các số này |
| TC-P6-01-26 | AC5 | lớp 1 | Thêm bằng API P5 cho B: một buổi `EXCUSED`, một buổi `LATE` (hoặc dùng B đã có `late 1`); đọc lại giải trình B | `absent` của B vẫn 2 (`EXCUSED`/`LATE` không tính); `P = 0` |
| TC-P6-01-27 | AC5 | lớp 1 | Chuyển một sinh viên rời lớp (`enrollments.status` ≠ `ACTIVE`) — hoặc dùng sv rút lớp có sẵn nếu seed có; đọc `…/gradebook` | sinh viên đó không nằm trong bảng điểm; `preview` không tính |
| TC-P6-01-28 | AC5 | repo | `grep -rn 'study_time_daily\|student_risk\|student_observations' backend-go/internal/grade --include=*.go \| grep -v _test.go` | rỗng |
| TC-P6-01-29 | AC5 | lệnh `Kiểm:` | `GO ./internal/grade "TestInputsFromP5|TestInputsExcusedLateNotAbsent|TestInputsActiveOnly|TestInputsMatchExpectedCRM" -tags integration` | `RESULT: PASS` |
| TC-P6-01-30 | AC5 (D17) | lớp 1 | Thay `study_time_daily` của C thành 600 phút/ngày (SQL tay), đọc lại giải trình + `preview` của C; trả về như cũ | HP của C vẫn 3,0 · F; không dòng giải trình nào nhắc thời gian học |

## Nhánh lỗi (SRS mục 3 + AC6)
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-01-31 | Không lẫn lớp — đọc | `j $Q2/gradebook -H "$TCH"` (lớp 2 chưa có công thức) | không có sinh viên / điểm nào của lớp 1; trạng thái "chưa có công thức" (xem TC-P6-06); không dòng của lớp 1 |
| TC-P6-01-32 | Không lẫn lớp — IDOR đường dẫn | lấy `item_id` của lớp 1 (`$PSQL "select id from grade_items where course_id='$C1' limit 1"`); `curl -X PUT $Q2/grade-entries -H "$TCH" -d '{"item_id":"<id>","student_id":"<sv>","score":"5","version":1}'` và `POST $Q2/grade-items/<id>/publish` | `404` cả hai; không dòng nào đổi (`select count(*) from audit_log where entity='grade_entries'` không tăng) |
| TC-P6-01-33 | Bài thi lớp khác không nối được | tạo item EXAM ở C2 với `exam_id` của lớp 1 (qua API `PUT …/grade-scheme` của lớp 2 nếu cho, hoặc Go test) | `422` (API) / `23503` (DB) |
| TC-P6-01-34 | Sinh viên rời lớp giữ snapshot cũ | sau khi lớp 1 có snapshot (TC-P6-08), cho một sinh viên rời lớp rồi chốt lại | snapshot cũ của sinh viên đó còn nguyên; không có snapshot mới cho sinh viên đó |
| TC-P6-01-35 | lệnh `Kiểm:` | `GO ./internal/grade "TestCrossCourseIsolation|TestExamFromOtherCourseRejected|TestLeftStudentExcluded" -tags integration` | `RESULT: PASS` |

## Phân quyền (AC7)
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-01-36 | Ma trận 20 route × 5 vai | `bash docs/sprints/8/qc/scripts/p6-rbac.sh` | `RBAC: PASS`, 0 dòng FAIL: STUDENT chỉ `…/me/grades` (+ `what-if`); TA `GET` được, ghi `403`; TEACHER qua cổng; ADMIN `403` mọi route lớp; không đăng nhập `401` |
| TC-P6-01-37 | Giảng viên lớp khác (nếu có tài khoản thứ hai; nếu seed không có, tạo qua API admin + gán lớp khác) | `GET $Q/gradebook`, `PUT $Q/grade-entries`, `POST $Q/gradebook/finalize` bằng token giảng viên đó | `404` (không lộ lớp tồn tại) |
| TC-P6-01-38 | Mọi route lớp qua `CourseAccessGuard` | `GO ./internal/server "TestAllCourseRoutesGuarded"`; QC thêm: `grep -n 'grade-scheme\|gradebook\|grade-entries\|grade-items\|me/grades' backend-go/api/openapi.yaml \| wc -l` | PASS; mọi route ở SRS 6 xuất hiện trong `openapi.yaml` (đếm ≥ 19 operation; thiếu một → FAIL luật 7) |
| TC-P6-01-39 | `GO` RBAC | `GO ./internal/grade "TestRBACMatrix" -tags integration` | `RESULT: PASS` |

## Script chạy được
`docs/sprints/8/qc/scripts/p6-env.sh`, `p6-rbac.sh`, `p6-go-tests.sh`.

## Câu hỏi cho BA / PM (không tự đoán)
- Không có.

## Lịch sử sửa TC
(chưa có)
