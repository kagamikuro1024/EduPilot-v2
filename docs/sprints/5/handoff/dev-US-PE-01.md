# DEV handoff — US-PE-01 (lược đồ thi hằng tuần, Quiz Engine, tính điểm, quyền theo lớp)
Nhánh `sprint/5-pe`, spec `FEAT-weekly-exam` v1.6 (lược đồ v1.5). Migration **`00006_weekly_exam.sql`** (ghi vào `docs/PROGRESS.md`; `00005` là `vn_fold` của P2).

## Làm gì
- **Migration `00006`**: 16 enum + 13 bảng đúng SRS 5.2–5.14 (cột / kiểu / CHECK / chỉ mục), FK phức hợp `(course_id, …)` ở 5.16, `UNIQUE (course_id, id)` làm đích, `code_submissions` theo v1.5 (`lease_until`, `next_attempt_at`, `fail_count`, `enqueued_at`). Hai UNIQUE hoãn (`code_testcases (problem_id, position)`, `exam_items (exam_id, position)`) là **constraint** `DEFERRABLE INITIALLY DEFERRED` (Postgres không cho `CREATE UNIQUE INDEX … DEFERRABLE`; chỉ mục sinh ra cùng tên). `sqlc generate` cập nhật `store/models.go`.
- **`internal/quiz`** (`Grade`): MCQ_SINGLE / TRUE_FALSE / MCQ_MULTI (`PARTIAL`, `ALL_OR_NOTHING`), `shopspring/decimal` + `DivRound(…,16)`, `ErrInvalidOption`, `ErrBadKey`; chữ ký có thêm `options ...string` (id hợp lệ của câu) để kiểm id lạ — bỏ trống thì không kiểm (khi chấm lại).
- **`internal/exam`**: `Score` / `RoundToStep` / `CodeEarned` (điểm bài thi = code thuần, `void` tính đủ điểm); `Routes()` = **bảng 56 thao tác của SRS 6.2** (số thứ tự, method, đường dẫn, chế độ guard, `Idempotency-Key`) — nguồn quyền duy nhất cho router, test ma trận và kiểm openapi.
- **`auth`**: 4 chế độ guard mới `MemberRole`, `StaffRole`, `TeacherRole`, `StudentRole` — ADMIN không bao giờ qua; thành viên ACTIVE sai vai → 403 `reason="role"`, người ngoài / PENDING / REMOVED → `"course"` (chế độ P2 cũ giữ nguyên `"course"`).
- **12 mã lỗi** (`apierr` + enum `Error.code` của `openapi.yaml`, lời văn tiếng Việt).

## AC (kết quả thật)
| AC | Kết quả |
| --- | --- |
| 1 | `TestExamSchema` (13 bảng, 16 enum đủ nhãn, cột từng bảng, kiểu `numeric(5,2)`…), `TestExamIndexesCourseFirst` (đúng 9 ngoại lệ), `TestExamForeignKeys` (24 FK phức hợp + 10 FK `users`, UNIQUE đích), `TestExamMigrationDownUp` (`down` về 0 rồi `up`) — PASS |
| 2 | `TestExamConstraints`: **90 ca** ghi sai → đúng SQLSTATE `23514` / `23505` / `23503` (gồm UNIQUE hoãn nổ lúc COMMIT, trộn dữ liệu giữa lớp ở 11 bảng) — PASS |
| 3 | `TestExamIndexes`: EXPLAIN 13 truy vấn thật trên 20.000 câu / 2.000 lượt / 20.000 bài nộp, 0 `Seq Scan` ở bảng lớn — PASS |
| 4 | `TestExamGuardMatrix` (55 route × 4 vai JWT × 7 tình trạng = **1.540 ca**, kèm `reason`), `TestExamAdminDenied`, `TestExamStudentRouteRoleOnly`, `TestExamRoutesTable` — PASS |
| 5 | **Một phần.** Tầng DB: FK phức hợp chặn trộn lớp (`TestExamConstraints`, `TestExamForeignKeys`). `TestExamIsolation` / `TestAllExamRoutesGuarded` ở tầng HTTP **chờ route** (PE-03 / 04 …): sẽ thêm cùng story đầu tiên đăng ký route |
| 6 | `TestQuizGrade` (29 ca), `TestQuizPartialFormula` (K = 1…5, so với `big.Rat` chính xác, > 40 ca), `TestQuizInvalidOption` — PASS |
| 7 | `TestScoringDecimal`: ví dụ bắt buộc **7.75** + 44 ca so với oracle hữu tỉ — `matched 45/45`; `grep -rn 'float64\|float32' internal/quiz internal/exam` ngoài test = 0 |
| 8 | `TestOnlySuggestImportsLLM` PASS (kiểm từng tệp; `quiz` và `judge` không import llm). `go list -deps ./internal/judge` chờ gói `judge` (US-PE-02) |
| 9, 10, 11, 12 | **Chưa** — cần dịch vụ + handler sửa / danh sách / audit / lớp lưu trữ của câu hỏi và bài thi (PE-03, PE-04). Sẽ làm trong các story đó (`TestVersionConflict`, `TestListCursor`, `TestAuditRowsPerAction`, `TestArchivedCourseWrites409` …) |
| 13 | **Một phần.** `TestExamErrorCodes` (12 mã có trong enum, lời văn riêng, tổng 49) + `TestExamRoutesNotAheadOfSpec` (openapi không có thao tác PE ngoài bảng 56) PASS. Thao tác PE được thêm vào `openapi.yaml` **theo từng story có handler** (`contract_test` yêu cầu mọi thao tác có route); đủ 56 ở cổng PE |

## Lệnh tự kiểm
`cd backend-go && make lint sqlc-check` (0 issues ×3); `make test` (race + testroutes + integration mail / auth) xanh trừ **`internal/today TestTodayQueryBudget` đỏ một lần khi chạy cả bộ** (7 > 5 truy vấn; chạy riêng PASS — lỗi chập chờn đã có từ sprint 4, không liên quan PE; `db/migrations_test.go` đổi version mong đợi 5 → 6 vì thêm `00006`); `go test ./internal/store -run 'TestExam' -v`, `go test ./internal/exam ./internal/quiz ./internal/auth ./internal/contract -run 'Exam|Quiz|Scoring' -v`.

## Nợ / ghi chú
- **CI GitHub không chạy được**: job báo "recent account payments have failed or your spending limit needs to be increased" (4 s, cả Go và Frontend, mọi commit sprint 5). Chỉ có kết quả chạy local; chủ dự án cần xử lý billing.
- AC5 (HTTP), AC9–AC12 và phần còn lại của AC13 như bảng trên.
- `Routes()` cho #11 / #12 / #13 ghi `StaffRole`; luật "sau khi bài đóng chỉ Teacher" kiểm ở service (SRS 6.2).
