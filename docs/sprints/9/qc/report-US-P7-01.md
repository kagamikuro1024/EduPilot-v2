# QC report — US-P7-01 (lược đồ `00017_assignments`) · Kết luận vòng 1: **PASS** (AC1–AC5; AC6 chuyển nơi kiểm theo proposal #8 — chờ PM; BUG-P7-01-1 lỗ hổng test)
Ngày 2026-10-31. Commit kiểm: `sprint/9-p7` @ `a16e9c17` (US-P7-01 ở `4ca8d484`). Bộ TC: `tc-US-P7-01.md`. Handoff: `dev-US-P7-01.md`. CI `4ca8d484` và `a16e9c17`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `goose up → down (một bước) → up → status` trên DB trống (`qc-p7-pg`, `pgvector:pg18`) | rc=0 cả ba; `00017` `Applied`; sau `down` bảng gỡ, sau `up` đủ |
| `grep -c '^ALTER '` / `ADD COLUMN` / `NO TRANSACTION` ở `00017_assignments.sql`; `git diff 82105320 HEAD -- db/migrations` | `2` / `0` / `0`; chỉ có `00017` mới, `00001…00016` không đổi; `Down` có `DELETE FROM reminder_log WHERE source_type::text = 'ASSIGNMENT'` (không dùng giá trị enum mới trong cùng tệp) |
| `p7-go-tests.sh ./internal/store "<14 test của dev>" -race` | PASS 14/14 (`TestSchemaAssignments`, `TestUuidv7Defaults`, `TestMigration00017Transactional`, `TestAssignmentsChecks`, `TestSubmissionsMatchedChecks`, `TestGradingResultsChecks`, `TestAiScoresImmutable`, `TestAiScoresFirstWriteAllowed`, `TestAppealsUnique`, `TestAssignmentsCrossCourseFK`, `TestGradeItemAssignmentFK`, `TestAssignmentDeleteRestrictedByGradeItem`, `TestIndexesStartWithCourseID_Assignments`, `TestMigration00017Shape`) |
| `go test -race ./internal/store/... ./db/...`; `sqlc generate && sqlc diff`; `go vet -tags testroutes ./...` | xanh; `sqlc diff` sạch; `go vet` sạch |
| **Probe QC** `scripts/p7-schema-probe.sql` (55+ ca, mỗi ca đọc SQLSTATE + tên ràng buộc) | PASS toàn bộ: `max_score` 0 / 100,25 / 10,1 → `23514`, 0,25 và 100 → ok; `late_penalty` âm, `late_max_days` 15, `allow_late` + 0 ngày, `appeal_days` 31, `channels` `{}` / lạ, TEAMS thiếu id, mã nộp thư 1 / 13 ký tự / chữ thường → `23514`; mã trùng cùng lớp `23505`, khác lớp ok; `MATCHED` thiếu `student_id` `23514`; trùng `(assignment, student, version)` `23505`; sinh viên ngoài lớp / bài tập lớp khác → `23503` (FK phức hợp); `UNMATCHED` trùng nhau **không bị chặn**; `external_ref` trùng `23505`; `file_size` 0 và 20 MiB + 1 → `23514`, 20 MiB ok; không cột `late_days`/`is_late`; đổi `sha256`/`blob_key` bài nộp và đổi `extracted_text` lần hai `P0001`, ghi `extracted_text` từ NULL ok; `grading_results`: `REVIEWED` khi `QUEUED`, `PUBLISHED` thiếu `published_at`, cờ lạ → `23514`, `round` trùng `23505`, `needs_review` là cột sinh (đúng theo cờ; ghi trực tiếp `428C9`); `ai_scores` / `ai_total_before_late`: ghi lần đầu từ NULL ok, đổi `P0001`, **ghi lại cùng giá trị ok** (Q-01-a: đạt), `criteria`/`total`/`status` sửa được; `grade_items` trỏ bài tập lớp khác `23503`; xoá bài tập đang được `grade_items` trỏ tới `23503` |
| Đột biến (`/tmp/qc-mut9`, đã xoá) | M1 bỏ UNIQUE `MATCHED` theo phiên bản: **đỏ** `TestSubmissionsMatchedChecks`; M2 bỏ trigger `ai_scores`: **đỏ** `TestAiScoresImmutable`; M4 bỏ `max_score % 0,25`: **đỏ** `TestAssignmentsChecks`; M5 bỏ UNIQUE `round`: **đỏ** `TestGradingResultsChecks`; **M3 `grade_items_assignment_fkey … ON DELETE CASCADE`: SỐNG** (BUG-P7-01-1) |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 migration một giao dịch, đúng hai `ALTER`, up / down / up sạch, `uuidv7()`, Down gỡ `reminder_log` | PASS | cổng ở trên |
| AC2 ràng buộc cấp DB (assignments, submissions) + chỉ mục bắt đầu `course_id` | PASS | probe + `TestAssignmentsChecks`, `TestSubmissionsMatchedChecks`, `TestIndexesStartWithCourseID_Assignments` |
| AC3 `ai_scores` / `ai_total_before_late` bất biến | PASS | probe, `TestAiScoresImmutable`, `…FirstWriteAllowed`; M2 |
| AC4 FK phức hợp `grade_items` → `assignments`, xoá bị từ chối | PASS (hành vi đúng; **test chưa giữ**: BUG-P7-01-1) | probe `23503` ở bài không có bài nộp |
| AC5 vi phạm từng ràng buộc cho mã lỗi đúng | PASS | probe + `TestGradingResultsChecks`, `TestAppealsUnique` |
| AC6 ma trận route × vai (`TestAssignmentsRBACMatrix`, `TestAllCourseRoutesGuarded`) | **chuyển nơi kiểm** theo #8 (route chưa tồn tại; dev ghi ở handoff) | tên test chưa có trong repo |

## 3. Lỗi / ghi nhận
- **BUG-P7-01-1 (Thấp, lỗ hổng test — mã đúng):** `TestAssignmentDeleteRestrictedByGradeItem` xoá bài tập `a1` mà fixture đã gắn một bài nộp (`submissions` FK) vào `a1`, nên lỗi `23503` đến từ `submissions` chứ không từ `grade_items`. Đột biến M3 (`ON DELETE CASCADE` ở `grade_items_assignment_fkey`) vẫn xanh. Probe QC (bài tập **không** có bài nộp) đỏ với M3 và xanh với mã thật. Đề nghị đổi test dùng bài tập không có bài nộp.
- **Q-01-b (BA / PM):** `assignments_teams_chk` là `('TEAMS' = ANY(channels)) = (teams_class_id IS NOT NULL) AND (teams_class_id IS NULL) = (teams_assignment_id IS NULL)`: chặt hơn spec (AC2 chỉ nói `TEAMS ∈ channels ⇒` hai id không NULL) — còn cấm **có** id khi không có kênh `TEAMS` (probe: `23514`). Hệ quả cho US-P7-02: `PUT` bỏ kênh Teams phải xoá luôn hai id, nếu không `23514`. QC coi là chấp nhận được; ghi để dev US-P7-02 / US-P7-11 biết.
- Lệch có chủ ý đã được PM chấp nhận: `ON DELETE RESTRICT` (SRS) → `NO ACTION` (`23001` vs `23503`) — #7. `type = 'QUIZ'` cho phép ở DB (P7 từ chối ở API: US-P7-02).
- `go.mod` đổi `golang.org/x/text`, `moby/moby/api` từ `indirect` sang trực tiếp (xem report US-P7-06).

## 4. Vòng 2 — `e424a20b` · Kết luận: **PASS**
- **BUG-P7-01-1 đã sửa:** `TestAssignmentDeleteRestrictedByGradeItem` dùng bài tập không có bài nộp; đột biến `ON DELETE CASCADE` ở `grade_items` làm test **đỏ**.
- **Q-01-b:** dev chuẩn hoá `""` → NULL cho hai id Teams (`TestTeamsIdsFollowChannel` xanh). Câu hỏi về `assignments_teams_chk` chặt hơn spec vẫn là của BA / PM.
