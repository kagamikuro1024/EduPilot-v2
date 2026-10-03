# DEV handoff — US-P1-03 (Scheduler, mạch ngắt, ngân sách, suy giảm)
Nhánh `sprint/3-pu-p1`. Góp ý #5–#8 ở `docs/sprints/3/proposals.md` (image > 40 MB, cửa sổ 2 s, bucket burst, đường dẫn gói config).

## Làm gì
- `internal/llm/scheduler`: cài `llm.Gate`. Hàng đợi cục bộ 3 làn FIFO (theo nhà cung cấp, ưu tiên INTERACTIVE > NEAR_REALTIME > BATCH), `LLM_QUEUE_MAX` mỗi làn, INTERACTIVE chờ tối đa `LLM_QUEUE_WAIT_MAX` → `ErrOverloaded{RetryAfter}`; ZSET `ep:llm:inflight:<provider>` (lease = hạn chót + 10 s, phần tử `I|N|B:<req>`), BATCH ≤ `ceil(MAX×share)` khi có INTERACTIVE đang chờ/chạy/vừa chạy < 2 s; token bucket RPM/TPM bằng Lua (`redis.call('TIME')` trong script, dung lượng = burst 10 %, yêu cầu lớn hơn dung lượng trừ nợ), đối soát TPM sau lời gọi; mạch ngắt Lua `ep:llm:cb:<provider>` (5 lỗi → mở 30 s → bán mở 1 lượt thử → đóng/mở lại); Redis mất → backend cục bộ (log error 1 lần/30 s, thử lại Redis mỗi giây, quay lại tự động + đối soát ngân sách).
- `internal/llm/budget`: bộ đếm Redis số nguyên 1/10.000 đ (hệ thống + lớp × ngày + tháng, `Asia/Ho_Chi_Minh`), `warn` ≥ 80 % → đúng một dòng outbox `llm.budget.warn` mỗi phạm vi mỗi kỳ (`SET NX`), `exhausted` ≥ 100 %, đối soát từ `llm_audit` (`Reconcile`). Cache hạn mức vô hiệu khi `Registry.Reload`.
- `internal/llm` (gateway): ngân sách cạn → BATCH `ErrUnavailable{budget_exhausted}` (`llm_audit.status=budget_blocked`), INTERACTIVE/NEAR_REALTIME → mô hình chat rẻ nhất (`Registry.Cheapest`); chuỗi chết → INTERACTIVE suy giảm trích nguyên văn (`degrade.go`), làn khác `ErrUnavailable{all_providers_failed}`; single-flight trong tiến trình (`flight.go`); hạn chót gồm chờ hàng, `timeout_s` chỉ thu hẹp.
- `internal/llm/llmrt`: dựng Registry + Scheduler + Budget + Auditor + Gateway cho gateway/contract rig; `cmd/gateway` dùng nó. Route thử `_test/llm/stats` có `queue_depth`, `inflight`, `inflight_batch`, `provider_inflight`, `circuit`, `redis_down`.
- `cmd/llmload` (không vào image): bơm BATCH + CHAT qua `_test/llm/chat`, in `interactive_wait_p95_ms`, `batch_peak`.
- Config `LLM_*` có kiểm tra (`TestLLMEnv`).

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `TestLaneOrder`, `TestFIFOWithinLane`, `TestDefaultLaneTable`, `TestBadLane` PASS |
| 2 | `-tags integration TestTwoProcessesGlobalConcurrency`: 2 tiến trình × 200 yêu cầu, `peak_inflight=10`, `completed=400`, ZCARD cuối = 0 |
| 3 | `TestTwoProcessesRPM`: rpm 60 → 16 lượt/10 s (trần 16); tpm 6.000 → 1.600 token (trần 1.600). `TestTPMReconcile` PASS |
| 4 | `TestBatchDoesNotStarveInteractive` (Redis thật, MAX=10, share 0,5): `interactive_wait_p95_ms=1`, `ttft_ratio=1.04`, `batch_peak=5`. Tay qua compose test, 2 gateway: `go run ./cmd/llmload --batch 100 --chat 20` → `interactive_wait_p95_ms=0 batch_peak=5 chat_fail=0 batch_ok=100` (fake 0,8–1,2 s; cần nâng `RATE_LIMIT_USER_PER_MIN` và `LLM_DEFAULT_RPM/TPM` khi đo). **Lưu ý góp ý #6**: không có CHAT "mồi" thì CHAT đầu chờ cả lượt BATCH |
| 5 | `TestQueueFullOverloaded` (≤ 50 ms, `retry_after` 1–30, không chiếm chỗ), `TestQueueWaitMax`, `TestRetryAfterFormula` PASS. Chưa chạy tay 201 yêu cầu song song qua curl |
| 6 | `TestBreakerOpensAfter5`, `TestBreakerHalfOpen`, `TestBreakerResetOnSuccess`, `TestBreakerIgnoresBadRequest`, `TestBreakerOpenSkipsProvider` (đồng hồ giả) PASS; `TestBreakerSharedAcrossProcesses` (integration, 2 tiến trình) PASS. Hiển thị `circuit` ở `GET providers`: US-P1-04 |
| 7 | `TestDeadlineIncludesQueueWait`, `TestDeadlineDefaults`, `TestTimeoutParamOnlyNarrows` PASS |
| 8 | `TestClientCancelStopsProvider` (≤ 1 s, Active=0, inflight=0, audit `cancelled`), `TestCancelNoRetryNoFallback` PASS |
| 9 | `TestDegradedExtractive` (≤ 3 đoạn theo điểm, «trích» — tên, tr. N, ≤ 600 ký tự, không từ kỹ thuật), `TestDegradedNoPassages`, `TestBatchNoDegrade` PASS |
| 10 | `TestSingleFlightShareable` (50 → `fake.Calls==1`), `TestNoSingleFlightInteractive` (INTERACTIVE / GRADING / không Shareable), `TestSingleFlightCancelOneWaiter` PASS |
| 11 | `TestBudgetWarn80Once`, `TestBudgetExhaustedBatchStops`, `TestBudgetExhaustedInteractiveCheapest` (Postgres thật: rẻ nhất = min(price_in+price_out), hoà theo created_at), `TestBudgetTwoScopes`, `TestBudgetDayRollover` (ICT), `TestBudgetReconcileFromAudit` PASS; outbox `llm.budget.warn` +1 mỗi phạm vi mỗi kỳ |
| 12 | `TestEveryRejectionHasCode`, `TestStreamMidFailure` (+ `llm`: `TestStreamMidFailure`) PASS; mọi từ chối có kiểu lỗi ánh xạ `OVERLOADED / LLM_NOT_CONFIGURED / LLM_UNAVAILABLE / DEADLINE_EXCEEDED` |
| 13 | `TestRedisDownFailsOpenLocal` (proxy TCP cắt Redis: vẫn cấp chỗ, giới hạn cục bộ MAX=2, log error 1 lần, quay lại toàn cục), `TestRedisBackReconciles` PASS |
| 14 | Route thử: ADMIN-only (contract: 401/403/200) |
| 15 | `TestLLMEnv` ở `internal/platform/config` (không phải `internal/config`); `LLM_BATCH_SHARE=1.5 gateway serve` → rc=1 nêu tên biến |

`BenchmarkSchedulerAcquire` (Redis qua colima, 11 goroutine): ~1.600 lượt cấp/giây (3 lệnh Lua mỗi lượt).

Cổng: `go vet` (cả `testroutes`, `integration`) sạch; `golangci-lint run` ×3 cấu hình 0 issues; `go test -race -count=1 -tags testroutes ./...` ok; `-tags integration` các test scheduler/llm PASS; `sqlc diff` rc=0.

## Nợ / ghi chú
- Trạng thái mạch ghi vào Redis có TTL 600 s: sau một đợt lỗi liên tiếp, mạch còn mở khi gateway khởi động lại — đó là hành vi mong muốn (chia sẻ), nhưng khi dev thử lại ngay có thể thấy `degraded` / `LLM_UNAVAILABLE` cho tới khi hết 30 s.
- Ưu tiên giữa các làn chỉ chính xác trong từng tiến trình; giữa tiến trình dựa vào trần BATCH và gợi ý `ep:llm:wait:INTERACTIVE` (`ponytail:` — có thể cần hàng đợi toàn cục nếu đo thấy lệch).
- Single-flight chỉ trong tiến trình (`ponytail:` trong `flight.go`).
