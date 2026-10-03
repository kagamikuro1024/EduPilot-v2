# Báo cáo QC — US-P1-03 (Scheduler: 3 làn, đồng thời toàn cục, token bucket, mạch ngắt, hạn chót, suy giảm, single-flight, ngân sách)
**Kết luận: FAIL** — 2 lỗi: BUG-P103-1 (hạn 30 s: client nhận kết nối đóng thay vì 504 JSON) và BUG-P103-2 (BATCH dùng hết 10 chỗ khi không có chat — spec v1.4 đến sau bàn giao bắt luôn ≤ 5). 46/50 TC PASS (kể cả vài TC chỉ dựa test dev, ghi rõ); FAIL: TC-25, TC-12 (theo v1.4); TC-14 và TC-48 có ghi chú.

- Bản chấm: commit `ef56cf0` (gồm `7d5d769` + 3 bản sửa P1-01/02) trong worktree QC riêng `../TA_Agent_qcp1`, đã gỡ sau khi chạy. Không chạm worktree của dev.
- Môi trường: 2 gateway `go build -tags testroutes` chạy native (:8080, :8081) dùng chung Redis + Postgres + MinIO riêng; máy chủ OpenAI giả của QC (`scripts/p1-run/qserver.mjs`, :9701–9703; độ trễ / mã lỗi chọn được, ghi request, đếm đỉnh đồng thời); probe Go `scripts/p103-probe` cho single-flight. Đặt `RATE_LIMIT_*_PER_MIN=100000`, `LLM_DEFAULT_RPM/TPM` cao cho ca không đo hạn mức. Máy: macOS arm64 (Docker qua colima).
- Q-QC: Q-QC-P103-1 (mốc `fake` độ trễ cố định) — SRS v1.2 đã trả lời; QC dùng độ trễ cố định cho mốc và ngẫu nhiên 5–15 s cho `llmload`. Q-QC-P103-2 / -3 (Redis dừng ảnh hưởng gì; múi giờ ngày ngân sách): **chờ BA** — QC ghi hành vi thực, chấm phần Scheduler.

## Lỗi
### BUG-P103-1 — INTERACTIVE hết hạn 30 s: client nhận *kết nối đóng*, không phải `504 DEADLINE_EXCEEDED` (AC7, AC12, SRS "không bao giờ im lặng")
Tái hiện: một nhà `openai_compatible` trả chậm 45 s; `curl -m 60 -H "$A" -H 'Content-Type: application/json' -X POST localhost:8080/api/v1/_test/llm/chat -d '{"task":"CHAT","prompt":"x"}'`.
Thực tế (3/3 lần): `Empty reply from server`, `http=000 t=30.0 s`; log gateway ghi `status:504 duration_ms:30008` nhưng thân không tới client. Mong đợi: `504 {"code":"DEADLINE_EXCEEDED","message":…}`.
Nguyên nhân (suy từ hành vi, QC không đọc mã): hạn của `ctx` = `REQUEST_TIMEOUT` của PG (30 s) trùng đúng hạn mặc định LLM 30 s → hai bộ đếm đua nhau; máy chủ đóng kết nối trước khi ghi 504. Đặt `LLM_REQUEST_TIMEOUT=20s` không đổi kết quả (vì ctx đã có hạn 30 s nên mặc định LLM không được áp).
Đối chứng: `timeout_s=3` ở tuyến → `504 DEADLINE_EXCEEDED` đúng ở 3,1 s. Chỉ hạn mặc định 30 s lỗi.

### BUG-P103-2 — BATCH chiếm cả 10 chỗ khi không có chat; chat đầu tiên chờ ~720 ms (lệch spec v1.4, góp ý #6)
Spec v1.4 (ACCEPTED, commit sau bàn giao): "BATCH **luôn** ≤ `ceil(MAX × share)`, bỏ ngoại lệ dùng hết công suất". Thực tế ở `ef56cf0`: 100 việc `GRADING` một mình → `ZRANGE ep:llm:inflight:<Q1>` có **10** phần tử `B:` ở t≈0,65 s; chat đầu tiên và wave chat ngay sau đó chờ hàng 718–737 ms (đo `llm_audit.queue_wait_ms`).
Việc cần dev: cài v1.4 (trần BATCH vĩnh viễn, bỏ `--prime`). Không phải lỗi so với bàn giao, nhưng cổng P1 chạy theo spec hiện hành → FAIL cho tới khi sửa. Sau khi sửa QC chạy lại TC-11/12/13.

## Ghi chú không thành lỗi
- **TC-14 (spec mâu thuẫn — chờ BA):** AC5 định nghĩa "hàng có 200 *đang chờ* → yêu cầu thứ 201 bị từ chối", nhưng lệnh kiểm "201 yêu cầu song song ⇒ đúng 1 `503`". Với `LLM_MAX_CONCURRENCY=1`, 1 yêu cầu chạy + 200 chờ ⇒ cả **201 được nhận, 0 bị từ chối**; yêu cầu **202** (và 203, 204) bị từ chối. Hành vi khớp định nghĩa, lệch lệnh kiểm. Đề nghị BA sửa lệnh kiểm thành 202.
- **TC-14 lời nhắn:** `message` = "Hệ thống đang rất đông. Vui lòng thử lại sau ít giây." — SRS bảng lỗi ghi "…Thử lại sau N giây." (có `retry_after` ở thân/header nên không mất thông tin); nhỏ.
- **TC-06 (kill -9):** khoá `ep:llm:inflight` còn 10 phần tử sau khi tiến trình chết; phần tử hết hạn bị loại *lười* khi có yêu cầu kế (yêu cầu mới vào ngay, 17 ms, ZCARD→0) — không rò chỗ, nhưng `stats.inflight` cũ tới lúc đó. QC chưa đo mốc đúng bằng lease.
- `cmd/llmload`: chạy được (`-secret`), kết quả `interactive_wait_p95_ms=3`, 25/25 chat, 200 BATCH ok; nhưng tuyến `INSIGHT` của QC trỏ `fake` nên `batch_peak=2` không có nghĩa — QC dựa số tự đo (TC-11).
- Các test cần thời gian thật không chạy: hạn BATCH 120 s, sang ngày ICT, hai phạm vi ngân sách → chỉ test dev (đánh dấu "dev").

## Kết quả từng TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `TestLaneOrder TestFIFOWithinLane TestDefaultLaneTable TestBadLane` |
| 02 | PASS | `MAX=1`, 1 việc đang chạy, rồi xếp 3 BATCH, 3 NEAR_REALTIME, 3 INTERACTIVE: thứ tự bắt đầu ở nhà cung cấp `HOLD I1 I2 I3 N1 N2 N3 B1 B2 B3` (INTERACTIVE > NEAR > BATCH, FIFO trong làn) |
| 03 | PASS | làn mặc định đúng 6 tác vụ (audit); `GRADING/INTERACTIVE`, `QUESTION_GEN/NEAR`, `INSIGHT/INTERACTIVE`, `CLASSIFY/INTERACTIVE` → `422 VALIDATION_FAILED`; hạ làn `CHAT/BATCH`, `CHAT/NEAR`, `CLASSIFY/BATCH` → 200, audit đúng làn; làn lạ → 422 |
| 04 | PASS | `-tags integration TestTwoProcessesGlobalConcurrency` |
| 05 | PASS | 2 gateway × 200 yêu cầu (nhà giả giữ 300 ms): lấy mẫu `ZCARD` mỗi 5 ms (1.792 mẫu) → **đỉnh 10**; nhà giả thấy đỉnh đồng thời 10; 400/400 xong; ZCARD cuối rỗng |
| 06 | PASS* | xem ghi chú (thu hồi lười) |
| 07 | PASS | `TestTwoProcessesRPM`, `TestTPMReconcile` |
| 08 | PASS | `rpm_limit=60`, 16 luồng vào cả hai gateway 10 s: **15** lời gọi tới nhà (trần 10 + burst 6 = 16); người gọi không thấy `RATE_LIMIT` (xếp hàng chờ, cuối 31/31 = 200) |
| 09 | PASS | cột null dùng mặc định 60/100.000 (chạy probe mặc định: bị giới hạn / xếp hàng); `TestLLMEnv` |
| 10 | PASS | `TestBatchDoesNotStarveInteractive` |
| 11 | PASS (theo v1.3) | 200 → 100 GRADING nền + 25 chat ≤ 5 đồng thời (400 ms/chat, nhà giả 1,5 s): `queue_wait` chat **p95 = 318 ms** (không mồi) / 328 ms (mồi), tối đa 718–737 ms (1/25); TTFT: mốc median 1.511 ms, có BATCH median 1.508 (×1,00), p95 ×1,20 (biên). Thất bại ở TC-12 (v1.4), không ở đây |
| 12 | **FAIL** (v1.4) | BUG-P103-2 |
| 13 | PASS | `llmload` in `p95=3 ms` (xem ghi chú); số tự đo 318 ms ≤ 500 |
| 14 | PASS* | yêu cầu thứ **202/203/204**: `503 OVERLOADED` sau **6–7 ms**, `retry_after=30`, header `Retry-After: 30`, `llm_audit` 3 dòng `overloaded`, nhà giả chỉ 1 request (xem ghi chú lệnh kiểm 201) |
| 15 | PASS | `MAX=1`, 1 chat chiếm chỗ; chat thứ hai chờ **10.018 ms** → `503 OVERLOADED` `retry_after=5`; nhà giả vẫn 1 request |
| 16 | PASS | yêu cầu bị từ chối không chiếm chỗ / không gọi nhà cung cấp; audit `overloaded` |
| 17 | PASS (NEAR_REALTIME) | hàng NEAR_REALTIME 200 đầy → `OVERLOADED`; làn BATCH chỉ test dev (`TestQueueFullOverloaded`); `retry_after`: 30 (hàng 200), 5 (hàng 1) |
| 18 | PASS | `TestQueueFullOverloaded TestQueueWaitMax TestRetryAfterFormula` |
| 19 | PASS | 5 lần chat với nhà chính lỗi 503 → mạch `open` ở lần thứ 5 (tính theo lời gọi, mỗi lời gọi 2 lần HTTP); lời gọi thứ 6–8: **0** request tới nhà, 6–8 ms, sang dự phòng; `GET providers` hiển thị `circuit` thuộc US-P1-04 → N/A |
| 20 | PASS | sau 30,5 s: 3 yêu cầu đồng thời → nhà chính nhận **1 lời gọi thăm dò** (+1 thử lại của chính nó), 2 yêu cầu kia sang dự phòng; thăm dò hỏng → mở lại 30 s (lời gọi sau: 0 request); thăm dò thành công → `closed` |
| 21 | PASS | 4 lỗi + 1 thành công + 4 lỗi → vẫn `closed`; 10×`400` → `closed`; 5×`401`, 5×`429`, 5×`504`, 5×đứt kết nối → mỗi loại `open` |
| 22 | PASS | mạch mở qua gateway :8080 → :8081 tới nhà chính **0 request** (Redis `ep:llm:cb:<provider_id>`) |
| 23 | PASS | `TestBreaker*`, `-tags integration TestBreakerSharedAcrossProcesses` |
| 24 | PASS | `timeout_s=3` + nhà giả chậm 10 s → `504 DEADLINE_EXCEEDED` ở **3,1 s** (hạn gồm chờ hàng; QC xếp hàng cho thấy `queue_wait` được trừ — test dev `TestDeadlineIncludesQueueWait`) |
| 25 | **FAIL** | BUG-P103-1 (hạn mặc định 30 s); `timeout_s=100` không nới (vẫn bị cắt ở 30 s) |
| 26 | PASS | `TestDeadlineIncludesQueueWait TestDeadlineDefaults TestTimeoutParamOnlyNarrows` |
| 27 | PASS | stream tới nhà giả chậm 8 s, huỷ client: `ZCARD` về 0 sau **3 ms**; nhà giả 1 request (không thử lại), nhà dự phòng 0 request; audit `cancelled/CANCELLED/attempts=1` |
| 28 | PASS | `TestClientCancelStopsProvider TestCancelNoRetryNoFallback` |
| 29 | PASS | mọi nhà 503: `200`, `degraded:true`, mở đầu **đúng** "AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:"; 4 đoạn truyền vào → trích **3** đoạn điểm cao nhất, nguyên văn, kèm «…» — tên tài liệu, tr. N; đoạn điểm thấp bị loại; audit `degraded` |
| 30 | PASS | không `passages`: "AI tạm thời không khả dụng. Câu hỏi của bạn đã được ghi lại, giảng viên sẽ xem." `degraded:true`; `GRADING` và `CLASSIFY`: `503 LLM_UNAVAILABLE` (không suy giảm) |
| 31 | PASS | 0 từ cấm (`provider|fallback|trace|RAG|PII|LLM|prompt`) |
| 32 | PASS | `TestDegraded* TestBatchNoDegrade` |
| 33 | PASS | probe `llm.Gateway`: 50 yêu cầu NEAR_REALTIME `Shareable` giống hệt → nhà giả nhận **1** request, 50 kết quả giống nhau |
| 34 | PASS | INTERACTIVE shareable → **50** request; GRADING shareable → **50**; NEAR_REALTIME không Shareable → **50** (không gộp) |
| 35 | PASS | một người chờ huỷ (hạn 300 ms): 1 lỗi, 49 còn lại nhận kết quả, vẫn **1** lời gọi. (Lỗi chung chia cho nhóm / nhóm mới thử độc lập: test dev) |
| 36 | PASS | `TestSingleFlight*` |
| 37 | PASS | ngân sách ngày 10.000 đ; mỗi lời gọi 1.000 đ (1 triệu token × 1.000 đ/M) cộng đúng `decimal` (1.000 → 10.000, không lệch 1 đồng); đạt **80 %** → đúng **1** dòng outbox `llm.budget.warn` `{"pct":80,"scope":"system","period":"day"}`; tăng tới 100 % không thêm dòng |
| 38 | PASS | 100 %: `GRADING` → `503 LLM_UNAVAILABLE` `details.reason="budget_exhausted"`, audit `budget_blocked`; `CHAT`, `CLASSIFY` vẫn trả lời bằng mô hình **rẻ nhất** (`q3-chat` 1+1 đ, không phải `q2-chat` 500+500 đ) |
| 39 | PASS (dev) | hai phạm vi + sang ngày ICT: `TestBudgetTwoScopes TestBudgetDayRollover` (QC không đổi đồng hồ) — Q-QC-P103-3 chờ BA |
| 40 | PASS | `TestBudget*`; grep `float32|float64` ở `llmconfig`, `llm/budget`, `llm/cost` = 0 |
| 41 | PASS | `OVERLOADED`, `LLM_UNAVAILABLE`, `LLM_NOT_CONFIGURED`, `DEADLINE_EXCEEDED` (khi hạn do `timeout_s`) đều có mã + `message` tiếng Việt (+ `retry_after`); hạn mặc định 30 s xem BUG-P103-1 |
| 42 | PASS (dev) | `TestStreamMidFailure` (QC không giả lập lỗi giữa luồng) |
| 43 | PASS | `TestEveryRejectionHasCode TestStreamMidFailure` |
| 44 | PASS | `TestRedisDownFailsOpenLocal TestRedisBackReconciles` |
| 45 | PASS | dừng Redis 12 s giữa tải 10 req/s: 66/66 yêu cầu `200`, trễ tối đa 335 ms; log `ERROR "Redis không dùng được, Scheduler chuyển sang giới hạn cục bộ"` (1 lần); bật lại → 53/53 `200`, không cần khởi động lại; `llm_audit` 157 dòng ok |
| 46 | PASS | `_test/llm/stats`: TEACHER/TA/STUDENT `403`, không token `401`, ADMIN `200`; khoá `audit, circuit, fake_calls, inflight, provider_inflight, queue_depth`, không `prompt`/`text` |
| 47 | PASS | 14 giá trị sai (`LLM_BATCH_SHARE` 1,5 / 0 / −1 / abc; `LLM_MAX_CONCURRENCY` 0 / −3; `LLM_QUEUE_MAX` −1 / 0; `LLM_QUEUE_WAIT_MAX=0s`; `LLM_REQUEST_TIMEOUT=-5s`; `LLM_BREAKER_FAILS=0`; `LLM_BREAKER_OPEN=0s`; `LLM_DEFAULT_RPM=0`; `LLM_DEFAULT_TPM=-1`): `rc=1` nêu đúng tên biến; `LLM_BATCH_SHARE=1`, `0.01` chạy được |
| 48 | PASS | mặc định đúng (10 / 0,5 / 200 / 10 s / 30 s / 5 / 30 s / 60 / 100.000 / 1536); `go test ./internal/platform/config -run TestLLMEnv` (góp ý #8) |
| 49 | PASS | `go vet` (+`testroutes`), `golangci-lint` (+tag), `sqlc diff` sạch; `go test -race -tags testroutes ./...` 0 FAIL; `-tags "integration testroutes" ./internal/llm/...` ok |
| 50 | ghi số | p95 chờ hàng chat (có BATCH nền, ≤ 5 chat đồng thời) 318–328 ms; TTFT median ×1,00, p95 ×1,20; đỉnh inflight 10 (2 gateway); lease/crash lười |

## Việc sau
Dev sửa BUG-P103-1 và BUG-P103-2 → QC chạy lại TC-11/12/13/25/41. BA sửa lệnh kiểm TC-14 (201→202) và trả lời Q-QC-P103-2/-3.
