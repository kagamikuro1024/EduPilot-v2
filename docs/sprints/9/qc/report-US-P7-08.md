# QC report — US-P7-08 (hàng chờ + màn duyệt) · Kết luận vòng 1: **PASS** (`TestTAReviewsCannotPublish` chuyển sang US-P7-09 theo luật #8 — chờ PM xác nhận)
Ngày 2026-11-01. Commit kiểm: `sprint/9-p7` @ `c487e4fa` (phần máy chủ `f5bbfa57`, frontend `c487e4fa`). Bộ TC: `tc-US-P7-08.md`. Handoff: `dev-US-P7-08.md` (chưa có mục frontend). CI `c487e4fa`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh`, `-race -tags integration` (httpapi `testroutes`), mọi tên của dòng `Kiểm:` đang tồn tại: assignment 19, httpapi 2, today 5 | PASS 26/26. Thiếu `TestTAReviewsCannotPublish` — dev chuyển US-P7-09 (route `publish` chưa có ở story này) |
| Hộp đen trên stack thật (`scripts/p7-api-08.py`, dữ liệu 3 kết quả `FAILED` của US-P7-07) | PASS 0 lỗi. Hàng chờ mặc định (DRAFT) không có bài FAILED, `status=FAILED` → 3 bài; con trỏ `limit 2` + trang kế 1 dòng; `q` thoát `%`, `q` theo tên → 1; `no-store`. STUDENT / ADMIN 403, TA / TEACHER 200 ở hàng chờ, chi tiết và `review`. Chi tiết theo id kết quả và theo id bài nộp trả cùng kết quả. Sửa điểm: `2,3` → `SCORE_STEP`, `100` → `SCORE_OUT_OF_RANGE`, `1,234` → `SCORE_PRECISION`, tiêu chí lạ → `CRITERION_UNKNOWN`, `-1` → `SCORE_OUT_OF_RANGE`, sai `version` → 409 kèm `current`. `FAILED` nhập đủ bốn điểm (`2.25`, `1`, `1`, `1`) → 200, **tổng 5,25** (decimal), `run_state DONE`, `error_code` xoá, `ai_scores` NULL, `audit_log`, `version` tăng. `review` → REVIEWED; lần hai idempotent; sửa khi REVIEWED → về DRAFT; chưa đủ điểm → `SCORES_REQUIRED` / `NOT_READY`; khoản trừ muộn tay `0,5` → tổng 4,50, `manual: true`. "Hôm nay": `GRADING_ERROR` có khi còn FAILED, sinh viên không thấy `GRADING_*` |
| Playwright desktop (`assignment-lifecycle.spec.ts` + `a11y.spec.ts`): `queue default filter and flags`, `grading ta cannot see publish`, `review two columns`, `evidence scroll and highlight`, `review amber notice for disagree`, `review edit score undo`, `review version conflict`, `review comment autosave`, `review draft restore`, `review key A next`, `review running locked`, `grading states`, `grading a11y` | 13/13 xanh; axe 4 lượt, 0 lỗi |
| `grep -rn "mock/assess" frontend/src`; `ui-antipatterns.sh` | rỗng; rc 0 |
| Đột biến (`/tmp/qc-mut9`, đã xoá) R1 bỏ kiểm bước 0,25 khi sửa điểm · R2 cho sửa khi `QUEUED` / `RUNNING` | cả 2 **đỏ**: `TestEditScoreStep`; `TestReviewWhileRunningRejected` |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 hàng chờ: cột, lọc mặc định, cờ lên đầu, con trỏ, p95, không seq scan | PASS | Go (`TestQueueDefaultFilter`, `…NeedsReviewFirst`, `…CursorPagination`, `…P95Under300ms`, `…ExplainNoSeqScan`, `TestNeedsReviewGeneratedColumn`); hộp đen; Playwright |
| AC2 màn duyệt hai cột 52–58 / 42–48, đồng bộ trích dẫn | PASS | `review two columns`, `evidence scroll and highlight` |
| AC3 cờ lệch hai lượt, `InlineNotice` hổ phách | PASS | `review amber notice for disagree` |
| AC4 sửa điểm tại chỗ, bước 0,25, khoá lạc quan, hoàn tác | PASS | R1; hộp đen; `review edit score undo`, `review version conflict` |
| AC5 nhận xét tự lưu nháp | PASS | `review comment autosave`, `review draft restore` |
| AC6 `Duyệt bài` + phím `A` | PASS | hộp đen; `TestReviewSetsReviewed`, `…NeedsScores`, `…KeepsFlagsHistory`, `…Idempotent`; `review key A next` |
| AC7 chấm dở / đang chạy / bị thay phiên bản | PASS | R2; `TestReviewWhileRunningRejected`, `TestReviewSupersededRejected`; `review running locked` |
| AC8 phân quyền (TA duyệt, không công bố) | PASS cho phần đã có route; `TestTAReviewsCannotPublish` → US-P7-09 (#8) | hộp đen ma trận; `TestReviewRBACMatrix`, `TestSignedGetChecksRole`; `grading ta cannot see publish` xanh |
| AC9 thay mock, trạng thái, bàn phím, trợ năng | PASS | grep rỗng; `grading states`, `grading a11y` |
| AC10 nguồn việc "Hôm nay" `GRADING_ERROR` / `GRADING_REVIEW` | PASS | hộp đen `GRADING_ERROR`; `TestProviderGradingError`, `…GradingReview`, `…StaffAggregatedOneQuery`, `…StudentAggregatedOneQuery`, `TestTodayQueryBudget` |

## 3. Lỗi / ghi nhận
- Không có bug.
- **Giới hạn kiểm:** UI của màn duyệt chỉ kiểm bằng Playwright (gateway giả); stack QC dựng từ `6f9f219a` chưa có frontend mới nên chưa chụp / đo trên dữ liệu thật — làm khi US-S9-SEED có kết quả chấm `DRAFT` mẫu S1–S7.
- Lệch có chủ ý của dev (SRS 4.8.3): bài `FAILED` sửa được và khi đủ điểm thì `run_state = DONE` — cần cho nhập tay (AC6); QC xác nhận đúng và không ghi điểm 0.
- Test cũ dev sửa (proposals #13 ACCEPTED): `TestNoQueryLinksByStudentCode` thêm `AssignmentQueue` vào danh sách trắng — QC đã xem `review.sql`: MSSV chỉ để hiển thị và lọc `ilike` trong phạm vi lớp, mọi nối theo khoá `user_id` / `enrollments`; `TestShareNotAvailableKinds` đổi chữ.
- BUG-S9-01 (#12): nhãn "Bản mô phỏng · dữ liệu giả" không còn ở màn thật (thấy trong ảnh `p704-unmatched-*-1440.png`).
