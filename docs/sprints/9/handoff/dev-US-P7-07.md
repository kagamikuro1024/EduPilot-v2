# Handoff — US-P7-07 (hàng đợi chấm `grading.jobs`)

## Đã làm
- **Nạp (`internal/grading/queue_api.go`, `Batches`):** `GradeAll` / `GradeOne` / `Retry` — một giao dịch Postgres ghi `grading_results` `QUEUED` (`INSERT … ON CONFLICT DO NOTHING` theo `UNIQUE (course_id, submission_id, round)`) + `jobs` (`grading.batch`) + outbox `job.enqueue`. `XADD ep:grading` nằm ở handler `grading.batch` (`Queue.RegisterBatch`) chạy **sau COMMIT** và trả `jobs.ErrDeferred` — không gọi LLM trong consumer outbox. Redis chết → `202` vẫn trả, DB có `QUEUED`, máy quét nạp lại.
- **Consumer (`queue.go`, `Queue`)**, theo mẫu `judge/queue.go`: nhóm `grading` ở stream `ep:grading`, `XAUTOCLAIM` tin treo quá `GRADING_LEASE`, stream `ep:grading.dead` chỉ để đếm. `GRADING_WORKERS` goroutine mỗi worker; mỗi bài một thuê (`lease_owner` UUID mỗi lần nhận, `attempts + 1`), gia hạn mỗi `GRADING_HEARTBEAT`; mọi lần ghi kết quả có điều kiện `run_state = 'RUNNING' AND lease_owner = $me` (`grading.WithLease` → `EssaySaveDraft` / `EssaySaveFailed`; mất thuê → `ErrLeaseLost`, kết quả bị bỏ).
- **Thử lại:** lỗi tạm (`TransientError`, lỗi hạ tầng) → `QUEUED` + `not_before = now() + GRADING_BACKOFF` (5 s, 20 s), tối đa `GRADING_ATTEMPTS` = 3; consumer không ngủ — máy quét (`Tick`, 5 s) nạp lại hàng đã tới giờ và hàng mất tin Redis (`GRADING_REQUEUE_AFTER` = 60 s) và `RUNNING` hết thuê. Lỗi vĩnh viễn (`LLM_BUDGET_EXHAUSTED`, `GRADING_OUTPUT_INVALID`, `TEXT_TOO_LONG`) → `FAILED` ngay. Hết lượt → `FAILED` ở DB + `.dead`.
- **Bản cũ:** việc của phiên bản không còn mới nhất → `FAILED` mã `SUPERSEDED`, không gọi LLM (cột `run_state` chỉ có 4 giá trị nên `SUPERSEDED` nằm ở `error_code`; màn hàng chờ US-P7-08 chỉ liệt kê phiên bản mới nhất).
- **Tiến độ lô:** Redis giữ ánh xạ `ep:gradebatch:{job}` (tập id kết quả) và `ep:gradejob:{result}` (TTL 24 giờ); mỗi bài xong consumer đếm trong DB, `jobs.Runner.Report` (SSE `job.progress`), xong → `Complete({graded, failed, flagged})`. Hết ngân sách → đúng **một** thông báo `GRADING_BUDGET_EXHAUSTED` cho nhân viên.
- **API:** `POST …/assignments/{aid}/grade-all` (`202 {job_id, queued}`; không có bài → `200 {queued:0}`; `retry_failed` + `only_unscored:false` xếp lại FAILED), `POST …/submissions/{sid}/grade` (round + 1 chỉ khi `DONE`/`FAILED` + `DRAFT`; `REVIEWED`/`PUBLISHED` → `409 ALREADY_REVIEWED`; đang xếp hàng hoặc hai người cùng bấm → `409 VERSION_CONFLICT`), `POST …/grading-results/{rid}/retry` — cả ba `[Idem, StaffRole]`. OpenAPI 263 → **266** thao tác; kịch bản contract ở `scenarios_assignment_test.go`.
- **Đọc việc chấm:** `jobs.Scope` (mới, `internal/jobs`) + `grading.JobScope`: nhân viên ACTIVE cùng lớp đọc `GET /jobs/{id}` của `grading.batch`; STUDENT, ADMIN, giảng viên lớp khác không (xem proposals #11: trả `404`, không `403`).
- **Worker:** `cmd/worker/gradingq.go` (`newGradingQueue`, `gradingTask`), đăng ký ở `registry.go` + `tasks.go`; env `GRADING_WORKERS/LEASE/HEARTBEAT/REQUEUE_AFTER/ATTEMPTS/ATTEMPT_TIMEOUT` (giây; để trống = mặc định) ở `.env.example` + `docker-compose.local.yml`.
- **Bộ đếm (AC10):** `grading.Metrics` (`jobs_total{done,failed,retry,superseded,lease_lost}`, `dead_total`, `queue_depth`, p50 / p95 thời lượng), không nhãn nào mang tên / MSSV / văn bản bài. Repo chưa có đường thu `/metrics` (không Prometheus / OTel metric); bộ đếm nằm trong tiến trình, `Snapshot()` cho `/observability` (P10) — ghi `ponytail:` ở mã.

## Lệch spec cần biết
- SSE `job.progress` của `jobs` chỉ có phần trăm (`done+failed` / `total`), không có `{done, failed, total}`; giao diện "Đang chấm 12/85" lấy `n/total` từ số bài của lô ở US-P7-08 (danh sách / `stats`). `jobs.progress` là `smallint`.
- `GET /jobs/{id}` ngoài quyền: `404` (proposals #11).
- Lệnh `Kiểm:` ghi `-tags integration`; repo không dùng tag đó.

## Kiểm
`go test -race -tags testroutes` ở `./internal/grading ./internal/integration ./internal/jobs ./cmd/worker ./internal/contract` xanh. Test theo tên AC: `internal/grading/queue_test.go` + `queue_more_test.go` (`TestGradeAllEnqueuesOnlyUngraded`, `TestGradeAllProgressSSE`, `TestGradeAllNothingToGrade`, `TestGradeAllIdempotentKey`, `TestGradeAllConcurrentNoDuplicate`, `TestRegradeOnlyDraftOrFailed`, `TestRegradeConflict`, `TestRetryThreeThenDead`, `TestBackoffUsesNotBefore`, `TestPermanentErrorNoRetry`, `TestDeadLetterVisible`, `TestRetryEndpointRequeues`, `TestOneFailureDoesNotBlockBatch`, `TestSupersededSkippedNoLLM`, `TestGradeAllOnlyLatestVersion`, `TestWorkerConcurrencyBound`, `TestLeaseRenewedWhileRunning`, `TestGradingNotInOutboxConsumer`, `TestBudgetExhaustedBatchSingleNotice`, `TestRedisOutageRequeuesQueued`, `TestGradeAllRBAC`, `TestJobReadRBAC`, `TestGradingMetricsNoPII`, `TestGradingLogsHaveTraceNoPII`); `internal/integration/grading_queue_test.go` (`TestGradingJobSurvivesWorkerKill`, `TestSlowGradeKeepsLeaseByHeartbeat`, `TestGradingUsesBatchLane`, `TestChatUnaffectedDuringBatchGrading`).

## Chưa làm
- `k6 run benchmarks/load/grading-vs-chat.js` (AC5): chưa viết; `TestChatUnaffectedDuringBatchGrading` đo ở mức Scheduler (200 việc BATCH xếp hàng, chat vào ≤ 500 ms), chưa đo TTFT thật qua gateway.
- Mục `GRADING_ERROR` / `Chấm lỗi` ở hàng chờ và "Hôm nay": US-P7-08.
- Việc `grading.batch` mồ côi nếu Redis mất khoá ánh xạ lô (sau 24 giờ hoặc `FLUSHALL`): bài vẫn được chấm (DB + máy quét) nhưng `jobs` không tự `SUCCEEDED`.

## Test cũ đã sửa
- `internal/course/share.go`: câu `NOT_AVAILABLE` của công thức điểm đổi "Công thức điểm chưa dùng lại được (phase P6)." → "Công thức điểm chưa dùng lại được giữa các lớp." (PM yêu cầu; không test nào khẳng định câu cũ). Câu của ngân hàng câu hỏi ("… (phase P9).") cùng lỗi từ kỹ thuật, chưa đổi vì chưa được yêu cầu.
- `internal/contract/contract_test.go`: số thao tác 263 → 266.

## QC vòng 1 — sửa (BUG-P7-07-1, BUG-P7-07-2)
- **BUG-P7-07-1:** `benchmarks/load/grading-vs-chat.js` đã có (k6; `k6 inspect` đọc được). Hai lượt: `PHASE=baseline` (20 luồng chat riêng, 40 s) rồi `PHASE=grading` (giây 5 TEACHER gọi `grade-all`, chat chạy tiếp): `first_event_ms` p95 < 500, `ttft_ms` p95 ≤ 1,2 × `BASELINE_TTFT_MS` (p95 của lượt baseline), `grading_queued` count > 0. Chạy trên stack seed có bài tập ≥ 200 bài nộp chưa chấm — dựng ở US-S9-SEED; kết quả đo sẽ dán ở handoff đó.
- **BUG-P7-07-2:** `jobs.Runner.ReportDetail` (SQL `UpdateJobProgressDetail`) ghi `{done, failed, total}` của CẢ lô vào `jobs.result` khi lô còn chạy và phát trong `job.progress` (SSE) / `GET /jobs/{id}`; `Complete` ghi đè bằng `{graded, failed, flagged}`. `TestGradeAllProgressDetail`. FE dùng `result.total` thay cho `rows.length` ở "Đang chấm n/total".
- BUG-P7-07-3 (việc `grading.batch` mồ côi khi Redis mất ánh xạ lô sau 24 giờ): giữ nguyên, các bài vẫn được chấm; ghi nhận.
- FE: `QueueTab.tsx` bỏ đếm theo `rows.length`; sau `Chấm tất cả` giữ `job_id` + `queued`, dùng `useJob` (GET /jobs/{id} + SSE `job.progress`) và hiện "Đang chấm {done+failed}/{total}" (total = `queued` cho tới khi có `result.total`); e2e `queue progress`.

## Vòng 2 (BUG-P7-07-4 + đo thật trên stack có seed)
- **Sửa `benchmarks/load/grading-vs-chat.js`:** k6 cấp `idInTest` cho cả VU của kịch bản `grade` (id nằm giữa các VU chat) → chỉ số `idInTest - 1` ra `undefined` (lỗi QC thấy), còn `% VUS` thì VU 1 và VU 21 dùng CHUNG một phiên → `409 CHAT_BUSY` giả (tôi đo được 10 / 129 lượt ở lần thử đầu). Cách đúng: `POOL = VUS + 1` phiên ở lượt `grading` (21 sinh viên seed), `i = (idInTest - 1) % POOL` → mỗi VU một phiên riêng; thiếu phiên thì `sleep` rồi bỏ lượt (không quay vòng nóng); lỗi thật vẫn in kèm VU / phiên. (`exec.vu.idInScenario` không tồn tại ở k6 này.)
- **Số đo** (stack `pnpm dev` + `node scripts/seed.mjs`; provider giả `DEMO_MODE`; BT03 = **22 bài MATCHED** — seed không có 200 bài, bản 200 bài cần `seed-t1.mjs` của P10; xoá `grading_results` giữa hai lượt để `grade-all` có việc):

  | | baseline (không chấm) | grading (`grade-all` 22 bài, giây 5) |
  | --- | --- | --- |
  | `first_event_ms` p95 | 23,85 ms | 28 ms (≤ 500 ✓) |
  | `ttft_ms` p95 | 20,48 s | 20,49 s (+0,05 %, ≤ +20 % ✓) |
  | `http_req_failed` | 0 % (0/125) | 0 % (0/123) |
  | `grading_queued` | — | 22 |

  k6 thoát `0` ở cả hai lượt. TTFT tuyệt đối 20 s là do cách provider giả nhả chữ (StreamDelay), không phải độ trễ cố định 300 ms; chỉ tỉ số hai lượt là phép đo.
- `scripts/check-assignments-seed.mjs grade` giờ chờ lô đang chạy (do k6 hoặc lần trước) xong rồi mới so `expected_grading.json`.
