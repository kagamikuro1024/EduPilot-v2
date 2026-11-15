# QC sprint 10.6 — báo cáo L4 backend (mức Go; chưa lên stack)

HEAD kiểm `5bcd7c13` (gồm L4 backend `b96fb6d9`, 4 test P-106-7, P-106-6). Chỉ chạy test Go và đọc mã; API thật chờ L4 frontend và lượt dựng `ep-qc`.

## 1. Cổng `-run` không rỗng (P-106-7)
`bash docs/sprints/10.6/qc/scripts/check_run_names.sh` (mọi `Test*` nêu trong TC):
- **GD-BUG-1 đã đóng**: `TestReflectOutputPII`, `TestReflectInputNoIdentity`, `TestGuidanceSetAtVersion`, `TestGuidanceIsolation` có thật, `--- PASS`. Cũng PASS: `TestEssayPromptUnchangedAtV0`, `TestGuidanceBlockDeterministic`, `TestGuidanceNoTotalsInPrompt`, `TestGuidanceVersionOnlyViaApply`, `TestPublishMixedGuidance`, `TestReflectInputNoSubmissionText`, `TestReflectValidate`.
- **GD-BUG-2 (Thấp)**: còn **thiếu** hai tên thuộc lát L4 backend đã bàn giao: `TestRegradeKeepsLatePolicy` (TC-G05-25), `TestRegradeNeverOverwritesTeacherScores` (TC-G05-08). Dev có `TestRegradeStale` kiểm cùng ý (điểm / trạng thái bài đã duyệt, đã sửa, đủ vòng không đổi) nhưng **chưa có** test cho quy tắc trừ muộn khi chấm lại. Đề nghị: đặt đúng tên hoặc BA vá SRS §9, và thêm phép kiểm trừ muộn / làm tròn y hệt.
- Còn thiếu thuộc lát sau (L5+): `TestRegressCLIRefusesExport`, `TestRegressionCaseIdempotent`, `TestRegressionCaseNoText`, `TestRejectReasonHints`, `TestSimilarNoBodyLog`.

## 2. Lỗi mới
**GD-BUG-3 (Trung bình)** — `TestDecideCallersAllowlist` (ranh giới an toàn 10.5, US-DM-09 AC1) **ĐỎ** ở HEAD: `gọi Decide / Cascade ngoài danh sách trắng: [assignmenthttp.guidanceDecide]`. Nguyên nhân: handler mới gọi `h.Guidance.Decide(...)` (`GuidanceService.Decide` — duyệt lưu ý, không liên quan `llm.Decide`), bộ quét AST bắt theo tên phương thức. Hậu quả: test ranh giới «chỉ 3 nơi được gọi decision» không xanh ở nhánh 10.6, `go test ./internal/llm` đỏ, CI sẽ chặn. **Đề nghị sửa đúng chỗ**: đổi tên phương thức của `GuidanceService` (ví dụ `Resolve`), **không** thêm vào danh sách trắng / `unrelated` (nới một test an toàn cần proposal PM).

## 3. Test Go đã chạy
| Gói / test | Kết quả |
| --- | --- |
| `TestApplyVersion`, `TestGuidanceDecideText`, `TestGuidanceVersionOnlyViaApply`, `TestRegradeStale`, `TestGuidanceStats`, `TestFlagsSnapshotSurvivesLostPublish` (P-106-6) | PASS |
| `TestPublishMixedGuidance`, `TestResultViewGuidance` | PASS |
| `./internal/contract -tags testroutes` (318 thao tác, 8 route lưu ý; so khoá `METHOD /path`: không khoá nào bị xoá) | PASS |
| `./internal/httpapi/...`, `./internal/assignment/...`, `./internal/grading/...`, `./internal/llm/provider|cost|fake|scheduler` | PASS |
| `./internal/llm` | **FAIL** — `TestDecideCallersAllowlist` (GD-BUG-3) |

## 4. Đọc mã so với TC (khớp SRS 4.2.9–4.2.13, P-106-2)
- **Duyệt / áp dụng** (`guidance_decide.go`): `Decide` kiểm `version` (409 `VERSION_CONFLICT` kèm bản hiện tại), `409 ITEM_NOT_DRAFT`, sửa chữ chỉ khi chưa duyệt (`RETIRE` + `text` → 422), chữ sửa qua `ValidateText` (từ khoá / PII / độ dài → `422 GUIDANCE_TEXT_REJECTED`), `audit_log` trong cùng giao dịch. `Apply`: khoá `assignments` `FOR UPDATE`, `expected_version` lệch → 409, không có gì đổi → `409 NOTHING_TO_APPLY` (giao dịch quay lui), gán `approved_in_version` / `retired_in_version` = N+1, ghi `grading_guidance_versions`, đổi `assignments.guidance_version` chỉ ở đây; trả `stale{…}`. Route `apply`, `reflect`, `regrade-stale` có `Idem`; `decide` dùng `version` (TC-G04-05, 08…10).
- **Chấm lại** (`RegradeStale`): chỉ `status='DRAFT'`, bản ghim < bản hiện hành, `first_reviewed_at IS NULL`, không tiêu chí sửa (so decimal), `round < 20`, chỉ bài nộp mới nhất, tối đa `REGRADE_MAX_BATCH`, `GradingInsertQueued` round + 1 (UNIQUE chặn trùng); `skipped{reviewed, edited, max_round}`; 200 khi không có bài, 202 khi có job.
- **Công bố** (`publish.go`): cờ bật → bài đã duyệt (`first_reviewed_at`) miễn, đếm `exempt{EDITED, ACCEPTED_AS_IS}` (+ `audit_log`); bài chưa từng duyệt mà lệch bản → `409 MIXED_GUIDANCE` trước mọi ghi. Cờ tắt: không so; **phản hồi vẫn có `exempt:{EDITED:0, ACCEPTED_AS_IS:0}`** (TC-G05-18 viết «vắng / rỗng» — chấp nhận được; xác nhận ở stack).
- **Số đo** (`Stats`): gom theo `guidance_version` của kết quả hiện hành `REVIEWED/PUBLISHED`, `decimal`, `n < 5` → `null`, `Cache-Control: no-store`. Lưu ý: `mae_total` dùng tổng điểm tiêu chí của giảng viên làm «tổng trước trừ muộn chốt»; cần đối chiếu với `total` + `late_penalty` ở stack (bài nộp muộn, điều chỉnh tay).
- **RBAC** (`assignmenthttp/guidance.go`): cờ — `Manage`; xem tổng quan, `regrade-stale`, `stats` — `Staff` (GV + TA); `reflect`, `decide`, `apply` — `TeacherRole`. Xác nhận ADMIN / TA ở stack (TC-G03-24, G04-11, G05-24).

## 5. Chưa kiểm (chờ stack, sau L4 frontend)
TC-G04-03…14 và TC-G05-05…25 ở mức API/DB (quyết định, áp dụng đồng thời, 250 nháp stale, kill worker, trừ muộn, đếm `stale`), TC-G06-01…04, 08…10 (đối chiếu số đo bằng SQL, 1.000 kết quả p95 ≤ 500 ms), RBAC bốn vai, toàn bộ E2E giao diện.
