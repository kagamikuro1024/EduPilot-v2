# Báo cáo QC — US-PE-01 (migration `00006`, 13 bảng, Quiz Engine, điểm `decimal`, guard theo lớp)
**Kết luận: FAIL 1 TC (TC-24/25 — bug B1 làm tròn điểm khi `max_score` không nguyên); các TC HTTP (TC-12…20, 28…37) CHƯA KIỂM ĐƯỢC vì route PE chưa đăng ký (dev ghi đúng trong handoff: AC5 HTTP, AC9–AC12 chờ PE-03 / PE-04); còn lại PASS.** Bản chấm `c5f93d7` (`origin/sprint/5-pe`, mã PE-01 = `18e860e`). Stack QC: Postgres `pgvector/pgvector:pg18` riêng (`qc5-pg`, cổng 45533) + `gateway migrate`; Go chạy trong worktree riêng `TA_Agent_qc5`. Số liệu tự đo, không lấy từ dev. Migration có tên `00006` (không phải `00005` như `tc`): `00005` là `vn_fold` của P2 (Q27); TC chấm theo thực tế. Không có lỗ hổng bảo mật.

## Lỗi / lệch
- **B1 (TC-24 / TC-25 — FAIL, AC7).** `exam.Score` chia `raw ÷ total` làm tròn 16 chữ số **trước** khi nhân `max_score` (`score.go: raw.DivRound(total, 16).Mul(maxScore)`). Khi `max_score` **không nguyên** và kết quả đúng nằm **chính xác giữa hai bước**, số cắt cụt rơi xuống dưới điểm giữa nên làm tròn **xuống** (đúng phải lên): kiểm vi sai với `big.Rat` 20.000 ca ngẫu nhiên (`seed 20261009`): **166 lệch**, tất cả ở `max_score` ∈ {1.5, 4.5, 7.5, 99.99}; `max_score` ∈ {10, 20, 100}: **0 lệch**. Ví dụ: `max=4.5`, `step=0.25`, `raw=7/4`, `total=3` → đúng `2.75`, hệ thống `2.5` (lệch một bước); `max=1.5`, `step=0.25`, `raw=49/4`, `total=21` → đúng `1.00`, hệ thống `0.75`. *Repro:* chép `docs/sprints/5/qc/scripts/qc_score_diff_test.go` (bỏ 2 dòng đầu) vào `backend-go/internal/exam/zz_qc_score_test.go` → `go test ./internal/exam -run TestQCScoreDiff -v`. `max_score` cho phép `numeric(5,2)` ∈ (0, 100] nên giá trị này hợp lệ. *Sửa gợi ý:* tính `raw × max_score ÷ total` rồi `DivRound` một lần (nhân trước, chia sau). Rủi ro thực: thấp với thang 10 / 100, nhưng AC7 đòi "làm tròn một lần" chính xác. `TestScoringDecimal` của dev (45/45) không bắt được vì chỉ dùng `max` nguyên.
- **L1 (TC-26).** AC7 nêu `internal/exam/testdata/exam_scoring.csv`; dev dùng oracle `big.Rat` ngay trong `score_test.go` (45 ca, `matched 45/45`), không có tệp CSV. QC thay bằng vi sai 20.000 ca (B1) — không FAIL riêng.
- **L2 (TC-05).** `gateway migrate down` lùi **về 0** (6 migration), không có `down-to 00004`; vẫn đạt (sau `up` schema giống hệt, chỉ khác token `\restrict` ngẫu nhiên của `pg_dump`).
- **L3 (TC-02).** `p501-schema.py` báo 2 "lệch" **giả** do bộ phân tích bảng SRS (dòng gộp `input / input_blob_key` của `code_testcases`; `char(64)` = `bpchar` của `source_sha256`); đối chiếu tay: đúng `text` nullable, `character(64)`.
- **L4 (TC-39).** `go test -race ./...` **không tag** đỏ ở `internal/auth` (`TestRefresh*`, `TestLoginTimingEqualized`: `connect postgres … connection refused :32773`) vì các test này dùng container dùng chung `edupilot-test-*` mà máy QC không còn chạy — hạ tầng, không phải mã. Cùng các test đó **PASS** khi chạy `-tags integration` (QC tự dựng container).
- **L5.** CI GitHub không chạy được (billing), như báo cáo US-PU-06; mọi kết quả ở đây là chạy local.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `gateway migrate` → 13 bảng PE, `goose` version **6** (`00006_weekly_exam.sql`) |
| 02 | PASS (L3) | `p501-schema.py` (SRS 5.1–5.14): **16 enum** đúng giá trị và thứ tự; 13 bảng, **192 cột** khớp tên / null / kiểu / `numeric(p,s)`; 0 bảng thừa / thiếu |
| 03 | PASS | cả 13 bảng đều có `course_id` và ≥ 1 chỉ mục có `course_id`; 0 bảng thiếu |
| 04 | PASS | `git diff origin/sprint/4-p2 -- db/migrations/0000[1-4]*` rỗng; sha256 `00001…00005` trùng sprint 4; `ALTER` bảng cũ trong `00006` = **0** |
| 05 | PASS (L2) | `down` → `up` ×3: sau `down` 0 bảng PE, 0 enum PE, version 0; sau `up` `pg_dump --schema-only` giống hệt lần đầu (ba lần) |
| 06 | PASS | `TestExamSchema` PASS |
| 07 | PASS (một phần) | QC tự viết **52 ca** `INSERT` (rollback) ở `question_bank`, `question_options`, `code_problems`, `code_testcases`, `exams`: 38 ca sai đều đúng SQLSTATE (`23514` ×30, `23503` ×3, `23505` ×1, `22P02` ×2…), 11 ca biên lành thành công (`title` 120, `position` 1 / 8, `time_limit_ms` 100, `memory_limit_mb` 16, `weight` 0, `max_score` 100…); 0 lệch. `exam_items`, `exam_attempts`, `exam_answers`, `code_submissions`, `exam_events`, `exam_appeals`: chỉ có `TestExamConstraints` của dev (90 ca, PASS) |
| 08 | PASS | các ca biên lành ở TC-07 |
| 09 | PASS | `TestExamConstraints` PASS |
| 10 | PASS (một phần) | `TestExamIndexes` PASS (dev: 13 truy vấn, 0 `Seq Scan`); QC chưa tự `EXPLAIN` |
| 11 | PASS | `TestExamIndexes` PASS |
| 12 | KHÔNG KIỂM ĐƯỢC | route `…/questions`, `…/exams` chưa đăng ký (PE-03 / PE-04); thay bằng TC-15 |
| 13 | KHÔNG KIỂM ĐƯỢC | như TC-12 |
| 14 | KHÔNG KIỂM ĐƯỢC | như TC-12 (`/me/exam-lock` chưa có) |
| 15 | PASS | `TestExamGuardMatrix` (55 route × 4 vai × 7 tình trạng = **1.540 ca**), `TestExamAdminDenied`, `TestExamStudentRouteRoleOnly`, `TestExamRoutesTable` PASS |
| 16–19 | KHÔNG KIỂM ĐƯỢC | cách ly lớp ở tầng HTTP chưa kiểm được; tầng DB: FK phức hợp `(course_id, …)` chặn trộn lớp (`option lớp khác` → `23503`, `TestExamForeignKeys` PASS) |
| 20 | KHÔNG KIỂM ĐƯỢC | `TestExamIsolation` / `TestAllExamRoutesGuarded` chưa có (dev ghi "chờ route") |
| 21 | PASS | QC vi sai `quiz.Grade`: **220 ca** `MCQ_MULTI` (K = 1…5 × mọi (TP, FP) × `PARTIAL` / `ALL_OR_NOTHING`) khớp `points × max(0, (TP−FP)/K)`, 0 lệch, không điểm âm; bỏ trống `0`; id lạ → `ErrInvalidOption`; id trùng `(a,a,b)` PARTIAL = đủ điểm (khử trùng); SINGLE / TRUE_FALSE đúng / sai = `3 / 0` |
| 22 | PASS | `TestQuizGrade`, `TestQuizPartialFormula`, `TestQuizInvalidOption` PASS |
| 23 | PASS | QC `p501-score.py` (`fractions`): `raw = 55/12`, thang 10 → `7,6389`; bước 0,25 ⇒ **7.75**; 0,01 ⇒ **7.64**; 0,1 ⇒ 7.6; 0,5 ⇒ 7.5; 1 ⇒ 8; khớp ví dụ trong `TestScoringDecimal` |
| 24 | **FAIL** (B1) | 166 / 20.000 ca lệch ở `max_score` không nguyên |
| 25 | **FAIL** (B1) | kẹp `[0, max]` và `void` đúng ở các ca chạy; lệch do làm tròn B1 |
| 26 | PASS (L1) | `TestScoringDecimal` `matched 45/45`; `grep float64\|float32` ở `internal/quiz`, `internal/exam` ngoài test = **0** |
| 27 | PASS | `go list -deps ./internal/quiz ./internal/judge \| grep -c internal/llm` = **0**; trong `internal/exam` không tệp sản xuất nào import `internal/llm` (chưa có `suggest.go`); `TestOnlySuggestImportsLLM` PASS |
| 28–36 | KHÔNG KIỂM ĐƯỢC | khoá lạc quan, cursor, `audit_log`, lớp lưu trữ — cần handler (PE-03 / PE-04); dev ghi "chưa" |
| 37 | KHÔNG KIỂM ĐƯỢC | `openapi.yaml` 69 thao tác (không đổi so với sprint 4); 56 thao tác PE thêm theo từng story (dev: `TestExamRoutesNotAheadOfSpec` PASS) |
| 38 | PASS (một phần) | `TestExamErrorCodes` PASS; golden: `git diff origin/sprint/4-p2 -- …/golden \| grep -c '^-[^-]'` = **0** |
| 39 | PASS có điều kiện (L4, L5) | `go vet` rc=0; `golangci-lint` 0 issues; `sqlc diff` rc=0; `go test -race -count=1 -tags integration -p 1 -parallel 2 -v ./...` rc=0: **783 PASS, 0 FAIL, 3 SKIP** (`TestRecordReplay` cần khoá thật, `TestReloadChild`, `TestSchedWorker` tiến trình con) |
| 40 | PASS | SQL ngoài `store/` ở `internal/exam`, `quiz`, `judge`: 0 (1 dòng khớp là chú thích) |

## Việc sau
- **Dev:** sửa B1 (`Score`: nhân `max_score` trước khi chia, `DivRound` một lần) và thêm ca `max` không nguyên vào `TestScoringDecimal`.
- **QC:** chấm lại TC-24 / 25 khi có fix B1; TC-12…20, 28…37 ở US-PE-03 / PE-04 khi route có mặt (đến cổng PE: 56 thao tác, `TestExamIsolation`). Scripts: `docs/sprints/5/qc/scripts/p501-schema.py`, `p501-score.py`, `qc_score_diff_test.go`, `qc_quiz_diff_test.go`.
