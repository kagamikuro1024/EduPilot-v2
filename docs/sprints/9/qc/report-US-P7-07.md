# QC report — US-P7-07 (hàng đợi chấm `ep:grading`) · Kết luận vòng 1: **FAIL** (AC5) → vòng 2: FAIL (AC5; BUG-P7-07-4) → vòng 3: PASS (AC5) nhưng BUG-P7-07-5 chặn cổng → **vòng 4 (`d37c8138`): PASS** — xem mục 6
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

## 4. Vòng 2 — `e7aeff77` (máy chủ `90bd69d0`, frontend `d10af653`) · Kết luận: **FAIL** (chỉ AC5)
CI `e7aeff77`: **success**.

| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh` (28 tên cũ + `TestGradeAllProgressDetail`): grading 25, integration 4 | PASS 29/29 |
| Playwright desktop `queue progress`, `queue default filter and flags`, `grading states`, `grading a11y` | xanh; axe 10 lượt, 0 lỗi |
| `k6 inspect benchmarks/load/grading-vs-chat.js` | đọc được (rc 0) |
| **`k6 run` thật trên stack seed** (QC dựng thêm 30 bài MATCHED cho một bài tập — lớp 1 chỉ có 30 sinh viên ACTIVE, **không dựng được 200 bài** như US yêu cầu): `PHASE=baseline` | 240 mẫu, `first_event_ms` p95 69 ms, `ttft_ms` p95 116 ms, `http_req_failed` 0 % — đạt |
| `PHASE=grading` (`BASELINE_TTFT_MS=116`, `grade-all` giây 5, xếp 30 bài) | ngưỡng đạt: `first_event_ms` p95 24,6 ms (< 500), `ttft_ms` p95 42,4 ms (< 139), `grading_queued` 30, 228 mẫu; mọi dòng `llm_audit` task `GRADING` làn `BATCH`. **Nhưng** kịch bản ném `TypeError: Cannot read property 'sid' of undefined` ở `chatVU` (`grading-vs-chat.js:86`) trong vòng lặp không `sleep`: **≈ 2,19 triệu dòng lỗi (≈ 420 MB)** trong 40 s; ít nhất một luồng chat không chạy được |
| `jobs.result` của lô chấm đang chạy | `{"done":0,"total":30,"failed":13}` (BUG-P7-07-2 đã sửa); xong → `{"failed":3,"graded":0,"flagged":0}` |

| AC | Vòng 2 |
| --- | --- |
| AC1 tiến độ `{done, failed, total}`, "Đang chấm n/total" | PASS — `jobs.result` đúng dạng; `queue progress` xanh; không còn đếm theo `rows.length` |
| **AC5 làn `BATCH`, chat không bị ảnh hưởng, `k6 run grading-vs-chat.js` đạt** | **FAIL** — lệnh `Kiểm:` chạy được nhưng (a) lỗi chỉ số luồng ở `PHASE=grading` (BUG-P7-07-4), (b) số đo ở 30 bài, chưa phải 200 bài (cần US-S9-SEED) |
| AC2–AC4, AC6–AC10 | giữ PASS |

- **BUG-P7-07-4 (Trung bình, AC5):** `chatVU` lấy `d.sessions[exec.vu.idInTest - 1]`; ở `PHASE=grading` có thêm kịch bản `grade` nên `idInTest` của luồng chat vượt 20 (nghi: VU của `grade` nhận id trước, lệch chỉ số) → `undefined` → ném lỗi liên tục, làm nhiễu số đo và đầy log. Đề nghị dùng `(exec.vu.idInTest - 1) % VUS` hoặc `exec.vu.idInScenario`, thêm `sleep` ngoài nhánh lỗi, rồi chạy lại ở 200 bài (US-S9-SEED).
- BUG-P7-07-3 (việc `grading.batch` mồ côi khi Redis mất ánh xạ lô) vẫn là ghi nhận của dev, không đổi.

## 5. Vòng 3 — `8e478910` (k6 sửa ở `92718308`) · Kết luận: **PASS** (AC5), kèm BUG-P7-07-5 chặn cổng G7-10
CI `8e478910`: **success**. Stack dựng từ DB trống ở HEAD.

| Lệnh | Kết quả |
| --- | --- |
| `k6 run grading-vs-chat.js` `PHASE=baseline` | rc 0; 240 mẫu; `first_event_ms` p95 76,7 ms; `ttft_ms` p95 118 ms; `http_req_failed` 0 % |
| `PHASE=grading` (`BASELINE_TTFT_MS=119`, `grade-all` giây 5, **30 bài** QC dựng thêm — lớp 1 chỉ có 30 sinh viên ACTIVE) | rc 0; **0 dòng lỗi** (BUG-P7-07-4 đã sửa: trước ≈ 2,19 triệu); `first_event_ms` p95 28,2 ms (< 500); `ttft_ms` p95 44,8 ms (< 143); `grading_queued` 30; 240 mẫu chat; mọi dòng `llm_audit` task `GRADING` làn `BATCH` |
| `go test -race ./internal/grading/...` (cổng G7-08) | PASS 128 test, không `DATA RACE` |
| `scripts/go-test-all.sh` (cổng G7-10) | **ĐỎ 2/2 lượt**: `TestRetryThreeThenDead` (`queue_more_test.go:58`: `.dead` XLEN 0, kỳ vọng 1); chạy riêng 3/3 xanh |

- **AC5 → PASS**, với giới hạn: số đo ở 30 bài, không phải 200 (seed không có 200 sinh viên; bản 200 bài cần `seed-t1.mjs` của P10 — dev đã ghi). Phần "200 việc `BATCH` không làm chat chờ > 500 ms" được `TestChatUnaffectedDuringBatchGrading` phủ ở mức Scheduler (xanh). **Đề nghị PM xác nhận chấp nhận giới hạn này** (nếu không, AC5 chuyển FAIL cho tới khi có seed 200 bài).
- **BUG-P7-07-5 (Trung bình, test đua; chặn G7-10):** `TestRetryThreeThenDead` đợi dòng kết quả `FAILED` ở DB rồi đọc ngay `XLEN ep:grading.dead`, nhưng `retryOrFail` ghi `FAILED` (`GradingFailPermanent`) **trước**, sau đó mới `XADD` `.dead` → đua; dưới tải của `go test ./...` song song thì đỏ ổn định (2/2 lần trên máy QC, 1 lần ở lượt dò đầu). Sửa: `eventually` cho `XLEN` / `DeadTotal`, hoặc `XADD` trước khi đánh dấu `FAILED`.
- `go test -list`: sprint 9 thêm 425 hàm `Test…` Go, **0 xoá**; 0 ca Playwright bị xoá.

## 6. Vòng 4 — `d37c8138` (CI success) · Kết luận: **PASS**
- **BUG-P7-07-5 đã sửa (sửa test, mã không đổi):** `TestRetryThreeThenDead` giờ `eventually` (20 s) tới khi `.dead` XLEN = 1 **và** `DeadTotal = 1` rồi mới kiểm số lời gọi LLM; vẫn đỏ nếu `.dead` không đúng một mục (QC đọc diff `queue_more_test.go`: không bỏ khẳng định nào). Lý do dev giữ thứ tự `FAILED` → `XADD` (đảo sẽ để mục `.dead` mồ côi khi thua đua) hợp lý.
- **G7-10: `bash scripts/go-test-all.sh` rc 0** — 57 gói `ok` (`-race -tags testroutes`) + `TimingEqualized` 3 lượt, chạy một mình trên máy QC; `go vet`, `golangci-lint` (0 issues), `sqlc diff` sạch. `TestRetryThreeThenDead` `-race` thêm 3 lượt riêng: xanh.
- `p7-go-tests.sh` (25 grading + 4 integration): PASS 29/29.
- AC5 giữ PASS ở quy mô seed (30 bài); PM đã chấp nhận (proposals, cổng P7 vòng 1): tải 200 bài thuộc P10 (`seed-t1`), ghi nợ.
