# QC report — US-P7-07 (hàng đợi chấm `ep:grading`) · Kết luận vòng 1: **FAIL** (AC5: thiếu `benchmarks/load/grading-vs-chat.js`, chưa đo TTFT qua gateway; 9/10 AC còn lại PASS)
Ngày 2026-11-01. Commit kiểm: `sprint/9-p7` @ `c487e4fa` (mã US-P7-07 không đổi từ `f5bbfa57`). Bộ TC: `tc-US-P7-07.md`. Handoff: `dev-US-P7-07.md`. CI `c487e4fa`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh`, `-race -tags integration`, mọi tên của dòng `Kiểm:`: grading 24, integration 4 | PASS 28/28 |
| `go vet -tags testroutes ./...`; `go test -race -tags testroutes` `contract jobs cmd/... store db today` | xanh (`contract` 269 thao tác) |
| Hộp đen trên stack thật (`scripts/p7-api-07.py`) | PASS 0 lỗi. 3 sinh viên nộp → `grade-all` `202 queued 3`, đúng 3 dòng `round 1`; cùng `Idempotency-Key` → cùng `job_id`; 10 request song song khoá khác → vẫn 3 dòng, không cặp `(submission, round)` trùng, `queued 0`; STUDENT / ADMIN 403, TA 200 / 202; `GET /jobs/{id}`: TEACHER + TA 200, STUDENT + ADMIN **404** (#11). Provider giả chưa có bản ghi chấm → cả 3 `FAILED LLM_UNAVAILABLE` sau **3** lần (`attempts = 3`, `ep:grading.dead` đếm); lỗi vĩnh viễn `attempts = 1`; mọi dòng `llm_audit` task `GRADING` làn `BATCH`; `retry` → 202; `grade` khi đang xếp hàng → 409. **SUPERSEDED:** dừng worker (`docker pause`), `grade-all` v1, nộp v2, chạy lại → v1 `FAILED SUPERSEDED`, `grade-all` sau đó chỉ xếp v2. Log gateway + worker **không** có canary văn bản bài, MSSV, email |
| Đột biến (`/tmp/qc-mut9`, đã xoá) P1 bỏ bỏ-qua bản cũ · P2 bỏ giãn cách `not_before` · P3 bỏ `UNIQUE (course, submission, round)` | cả 3 **đỏ**: `TestSupersededSkippedNoLLM`; `TestBackoffUsesNotBefore`; `TestGradeAllConcurrentNoDuplicate` (+ các test hàng đợi, vì `ON CONFLICT` cần chính ràng buộc đó) |
| `ui-antipatterns.sh`; grep `mock/assess` | rc 0; rỗng |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 `Chấm tất cả` 202, một giao dịch rồi `XADD`, không xếp lại bài đã có kết quả | PASS (có lệch, BUG-P7-07-2) | hộp đen; `TestGradeAllEnqueuesOnlyUngraded`, `…ProgressSSE`, `…NothingToGrade` |
| AC2 idempotent theo `(submission, round)` | PASS | hộp đen; P3 |
| AC3 worker bị giết: nhận lại, không chấm đôi, không mất | PASS ở mức Go | `TestGradingJobSurvivesWorkerKill`, `TestSlowGradeKeepsLeaseByHeartbeat`; QC không `kill -9` được giữa hai lượt (provider giả quá nhanh) |
| AC4 thử lại 3 lần, `not_before`, dead-letter, lỗi vĩnh viễn không thử lại | PASS | hộp đen (`attempts = 3` / `1`, `.dead`); P2 |
| AC5 làn `BATCH`; chat không bị ảnh hưởng (p95 chờ ≤ 500 ms, TTFT ≤ +20 %); `k6 run benchmarks/load/grading-vs-chat.js` | **FAIL** | Go: `TestGradingUsesBatchLane`, `TestChatUnaffectedDuringBatchGrading` xanh, hộp đen làn `BATCH` đúng. **`benchmarks/load/grading-vs-chat.js` không tồn tại** (thư mục có `chat.js`, `exam-submit.js`, `heartbeat.js`, `smoke.js`, `today.js`); dòng `Kiểm:` của US chạy lệnh này nên không chạy được; TTFT thật qua gateway chưa đo (test Go chỉ đo hàng chờ ở Scheduler) |
| AC6 số worker, mỗi bài một thuê | PASS | `TestWorkerConcurrencyBound`, `TestLeaseRenewedWhileRunning`, `TestGradingNotInOutboxConsumer` |
| AC7 phiên bản cũ `SUPERSEDED`, không gọi LLM | PASS | hộp đen; P1 |
| AC8 hết ngân sách (một thông báo gộp), Redis chết nạp lại, khoá lạc quan | PASS ở mức Go | `TestBudgetExhaustedBatchSingleNotice`, `TestRedisOutageRequeuesQueued`, `TestRegradeConflict`; QC không dừng Redis 70 s trên stack |
| AC9 phân quyền | PASS | hộp đen; `TestGradeAllRBAC`, `TestJobReadRBAC`, `TestJobReadOthers404` |
| AC10 bộ đếm không PII; log có `trace_id`, không văn bản bài | PASS theo US | `TestGradingMetricsNoPII`, `TestGradingLogsHaveTraceNoPII`; log stack sạch. Bộ đếm nằm trong tiến trình (`Snapshot()` cho `/observability` P10), **không** có `/metrics` thu được — US không đòi, TC-13 `curl $GW/metrics` không áp dụng |

## 3. Lỗi / ghi nhận
- **BUG-P7-07-1 (Cao, AC5):** thiếu `benchmarks/load/grading-vs-chat.js`; TTFT / chờ hàng thật qua gateway chưa đo. Dev viết kịch bản (k6 đã cài) hoặc BA / PM đổi dòng `Kiểm:` bằng proposal; QC chạy lại.
- **BUG-P7-07-2 (Thấp, AC1):** `job.progress` chỉ có phần trăm (`jobs.progress smallint`), không `{done, failed, total}` như US. Chữ "Đang chấm n/total" nằm ở `QueueTab.tsx:254` và dùng `rows.length` (số dòng đã tải) làm tổng — sai khi hàng chờ phân trang con trỏ. Dev tự ghi là lệch spec, chưa có proposal.
- **BUG-P7-07-3 (Thấp):** việc `grading.batch` mồ côi (không tự `SUCCEEDED`) nếu Redis mất ánh xạ lô sau 24 giờ hoặc `FLUSHALL` (dev tự ghi); các bài vẫn được chấm.
- Chưa kiểm được trên stack: kill -9 giữa hai lượt, gia hạn thuê 100 s, Redis dừng 70 s — chỉ có test Go; cần provider giả có độ trễ cấu hình (US-S9-SEED).
- Test cũ dev sửa: `course/share.go` câu `NOT_AVAILABLE` (PM yêu cầu; `share_test` sửa sau — #13 ACCEPTED), `contract_test.go` 263 → 266.
