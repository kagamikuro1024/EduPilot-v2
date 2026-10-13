# Handoff — BUG-S7-01 (#19): quét mọi test Go còn chờ đồng hồ thật

Nguyên nhân gốc `TestAuditBatches`: `flushed` tăng SAU khi `write` trả về, test đọc nó sau một lần poll 500 ms (hoặc đọc ngay) → `flushed=0 pending=0 dropped=0`. Sửa: `a.Close(ctx)` (xả hết, đồng bộ) rồi mới khẳng định.

Quy tắc áp dụng (không nới assertion, không retry, không bỏ test):

| Mã | Quy tắc |
|---|---|
| R1 | Cận TRÊN thời gian thay bằng bằng chứng hành vi (độ trễ giả đủ dài + hạn chống treo 30 s, hoặc kiểu lỗi, hoặc đếm số lần gọi). Cận DƯỚI giữ nguyên. |
| R2 | Poll/`Eventually`/hạn chờ sự kiện < 30 s nâng lên 30–60 s (hạn chống treo, không phải mốc hiệu năng). |
| R3 | `Sleep` rồi mới khẳng định thay bằng chờ sự kiện/kênh/poll trạng thái. |
| R4 | Cận trên là mốc SLO của spec: giữ, nhưng chỉ kiểm khi `testutil.TimingOn()` (đảo của `EP_SKIP_TIMING=1`; CI chạy song song đặt biến này). Chạy cục bộ/tuần tự vẫn kiểm. Helper mới ở `internal/testutil/containers.go`. |

## Đã sửa (tên test thật, theo `git diff`)

| Test (gói) | Mẫu cũ | Sửa |
| --- | --- | --- |
| `TestAuditBatches` (`llm`) | poll 500 ms rồi đọc `flushed` | R3: `Close(ctx)` rồi khẳng định |
| `TestAuditAsync` (`llm`) | đo elapsed của `Record` | R1: goroutine + hạn 30 s |
| `TestStreamCancel` (`llm`) | trễ 1 s/từ, `≤ 200 ms`, poll 300 ms | R1+R2: trễ 1 phút, hạn 30 s, poll 30 s |
| `TestReloadAtomic` (`llm`) | `Less(since, 1s)` | R1 |
| `TestLatencyErrorRateAndKey`, `TestCtxCancelStopsCall` (`llm/fake`) | cận trên latency; `Sleep(30ms)` rồi đọc `Active` | R1 (cận dưới 40 ms giữ); R3 poll `Active()==1` |
| `TestNetworkAndTimeout` (`llm/provider`) | máy chủ ngủ 300 ms vs hạn 50 ms; huỷ sau `Sleep(30ms)` | R3: máy chủ giữ yêu cầu bằng kênh; huỷ khi yêu cầu đã tới |
| `TestSDKNoInternalRetry`, `TestRateLimitKeepsStatusOnDeadline` (`llm/provider`) | `elapsed < 1s` | R1: đếm lần gọi / kiểu lỗi |
| `grantOrder` ⇒ `TestLaneOrder` và mọi test dùng nó (`llm/scheduler`) | `Sleep(30ms)` giữa các goroutine để ép thứ tự đến | R3: `waitQueued` chờ độ sâu hàng đợi |
| `TestBatchShareAlwaysCapsBatch` | INTERACTIVE `≤ 50 ms`, hạn 100 ms | R1: hạn 30 s (phải chờ ⇒ kẹt ⇒ đỏ) |
| `TestQueueFullOverloaded`, `TestQueueWaitMax`, `TestTimeoutParamOnlyNarrows` | cận trên elapsed | R1 (cận dưới giữ), R2 |
| `TestDeadlineIncludesQueueWait` | `Sleep(100ms)` chờ giữ chỗ; cận trên | R3 poll `Active()==1`; R1 |
| `TestDeadlineDefaults` | cửa sổ `[min,max]` | cận dưới giữ; cận trên R4 |
| `TestClientCancelStopsProvider`, `TestCancelNoRetryNoFallback` | `≤ 1 s`; `Sleep(80ms)` | R2; R3 poll `calls` / `Active` |
| `TestSingleFlightCancelOneWaiter` (+ nhánh huỷ tất cả) | `Sleep(100ms)` | R3 `waitActive` |
| `TestBreakerOpenSkipsProvider` | `≤ 50 ms` | R1: đếm lần gọi A giữ nguyên |
| `TestReloadChild`, `TestReloadTwoProcesses` (`llm`, tag integration) | child `Sleep(300ms)` chờ pub/sub; `CHAT=A ≤ 5 s` | R3: child poll `PUBSUB NUMSUB` trước `READY`; R2. Mốc `CHAT=B ≤ 1 s` (AC14) giữ |
| `TestReloadAcrossProcesses` (`llmconfig`, tag integration) | `Sleep(300ms)` | R3 `waitSubscribed`; mốc AC9 ≤ 1 s giữ |
| `TestBlockingReadsDoNotStarvePool` (`platform/redis`) | `Sleep(300ms)`; `Set ≤ 300 ms` | R3 poll `PoolStats`; R1: `Set` hạn 10 s khi XREAD chặn 30 s |
| `TestPool_AcquireHonorsDeadline` (`platform/db`) | `elapsed ≤ 1s` | R4 |
| `TestTodayTTLSafetyNet` (`today`) | `PExpire 5ms` + `Sleep(50ms)` | R3: poll tới khi khoá mất |
| `TestAdminTodaySurvivesHangingLLMSignals`, `TestAggregatorSkipsFailingProvider` | `Less(since, 140ms / 1s)` | R1: tín hiệu / Provider "treo" chỉ nhả khi ctx huỷ, nên trả về được = đã bị cắt |
| `TestTodayInvalidatedByApprove`, `TestQuestionReviewInvalidates`, `TestExamTodayInvalidation` | `waitFor(2 s)` + `Less(took, 2 s)` | R2 (30 s) + R4 |
| `TestLockerHitLatencyP95` (`exam`) | p95 ≤ 5 / 40 ms | R4 |
| `TestDocumentChangedNotBlockedByIngest` (`ingest`) | docling 3 s vs `< 1 s` | R1: docling 30 s vs `< 15 s` |
| `TestAssignNotifyOnce`, `TestAssignTeacherNotifies`, `TestJoinStudentCodeCollisionPending`, `TestRosterLinkRequiresVerifiedEmail` (`integration`) | `waitFor` 5–10 s; `took < 2 s` | R2 (60 s) + R4 |
| `TestDeadline_DB`, `TestDeadline_Redis`, `TestDeadline_ClientCancel`, `TestServerLimits_SlowHeader`, `coreWaitNoSleepQuery` (`httpapi`) | `elapsed ≤ 1,5 s`; `≤ ReadHeaderTimeout+5 s`; poll 3 s | R4; R2 |
| `TestGracefulShutdown`, `TestStartup_WaitsForDeps` | `Sleep(100ms)` rồi đọc `Draining()`; `elapsed ≤ 6 s` | R3 poll; R4 |
| `TestRateLimit_RedisDown_FailOpen` | so trung vị `≤ +50 ms` | R4 |
| `TestSSE_CrossInstance`, `TestSSE_TwoConnectionsTwoInstances`, `TestSSE_GatewayShutdownFailover`, `TestSSE_RedisDownMidStream`, `TestSSE_MaxDuration`, `TestSSE_TokenExpiryMidStream`, `TestSSE_FirstEventLatency` + `grace` (`httpapi/sse`) | `grace=5 s`, `want(2 s)`, `took ≤ 300 ms…5 s` | R2 (30 s) + R4 |
| `TestFromDraftNoMessageStored` (`chat`) | canary `9f3` (hex) khớp UUID ngẫu nhiên | **chưa sửa**: không phải đồng hồ; test cũ đỏ vì canary nên ghi đề xuất #23 và chờ ACCEPTED (dev đã hoàn tác bản sửa lỡ tay) |

## Giữ nguyên (có lý do)

- Cận dưới: `Sleep` dài hơn TTL/giây thật (`auth/account_test` 1100 ms, `exam/attempt_code_test` 2200 ms, `user/invite_test` 1100 ms, `platform/blob` 3 s, `sse/limits_test` ttl+200 ms): ngủ lâu hơn không bao giờ làm test sai.
- `Sleep(5–20 ms)` để hai mốc `now()` khác nhau (`exam/attempts_test`, `exam/similarity_svc_test`, `library_test`, `thread_test`, `store/calendar_schema_test`): chỉ cần chắc chắn "sau"; Postgres ghi µs.
- Chờ phủ định ("không có gì xảy ra"): `exam/schedule_test` (400 ms), `platform/outbox` (10 beat), `escalation/mail_test` 300 ms, `idempotency_test` 200 ms (kết quả chấp nhận cả 409 lẫn phát lại).
- Đã có `EP_SKIP_TIMING`: `chat/*`, `calendar/*`, `privacy/detect`, `auth/*TimingEqualized`, `course/join_limit`.
- Mốc SLO có biên rộng ≥ 2 s: `cmd/worker` (`≤ 2 s/5 s/10 s`), `exam TestSimilarityScale1000` (≤ 30 s), `escalation/mail_test` (60 s), `judge_test` (hang 3 s vs 2 s).
- Ngoài phạm vi CI `Go test`: các tệp `integration` còn lại (`llm/scheduler/integration_test.go`, `mail/integration_test.go`, `judge/*_integration_test.go`, `auth/account_integration_test.go`, `exam/*_integration_test.go`) — chỉ chạy ở job Judge; chưa đổi vì chưa thấy đỏ. Nếu cần quét tiếp: nói.

## Kiểm
- `go vet -tags testroutes ./...` và `-tags integration` (llm, llmconfig): sạch.
- `go test -race -tags testroutes ./...` với `EP_SKIP_TIMING=1` (như CI, một lần đầy đủ): chỉ đỏ `TestFromDraftNoMessageStored` (canary hex, đề xuất #23, chưa sửa vì chờ ACCEPTED); với canary đổi tạm, `./internal/chat ./internal/escalation` xanh.
- Số lượt nguội liên tiếp xanh trên CI: ghi vào mục dưới sau khi chạy.
