# QC test case — US-P1-03 (Scheduler: 3 làn, token bucket, mạch ngắt, hạn chót, suy giảm, single-flight, ngân sách, Redis mất)
Nguồn: `docs/specs/FEAT-llm-gateway/US.md` US-P1-03 AC1–AC15 + `SRS.md` 4.3 (bảng tác vụ → làn), 4.4 (ngân sách), 8.1 (biến `LLM_*`), `docs/SYSTEM_DESIGN.md` §3.1, §5 (SLO: chờ hàng p95 ≤ 500 ms, TTFT +20 %). Hộp đen; đo bằng **tải thật hai gateway + Redis thật** và công cụ `cmd/llmload` của dev **cộng** kịch bản k6 / Bun riêng của QC (`scripts/p103-*.js`) để không chỉ tin số do công cụ dev in.

Tiền điều kiện chung: stack test 2 gateway (`testroutes`) + Redis + Postgres; `fake` độ trễ `5000-15000` ms cho tải; `LLM_MAX_CONCURRENCY=10`, `LLM_BATCH_SHARE=0.5`, `LLM_QUEUE_MAX=200` mặc định (đặt qua `.env.local` + `$C up -d`). `RDS`, `PSQL`, `tok`, `api`, `$A/$T/$S`, `GW` như `FEAT-llm-gateway/US.md`. Docker (colima) cấp ≥ 4 CPU / 6 GB cho phép tải; ghi cấu hình máy vào report. Công cụ: **G** = `go test` (dev; `-tags integration` cần Redis), **L** = tải (`cmd/llmload` + k6), **S** = shell/curl, **D** = đo qua Redis / DB. Thiếu `cmd/llmload` → TC loại **L** của dev FAIL "KHÔNG KIỂM ĐƯỢC"; QC vẫn đo bằng k6 riêng.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P103-01 | AC1 | – | **G** `go test -race ./internal/llm/scheduler/... -run 'TestLaneOrder\|TestFIFOWithinLane\|TestDefaultLaneTable\|TestBadLane' -v` | `ok`, không SKIP |
| TC-P103-02 | AC1 (đo ngoài) | `LLM_MAX_CONCURRENCY=1`, `fake` 2 s | **S** bơm theo thứ tự thời gian 3 BATCH, 3 NEAR_REALTIME, 3 INTERACTIVE (`CHAT`) khi chỗ đang bị 1 việc chiếm; ghi thứ tự bắt đầu qua `llm_audit.created_at`/`queue_wait_ms` | Sau khi chỗ trống: INTERACTIVE chạy trước, rồi NEAR_REALTIME, rồi BATCH; trong mỗi làn đúng thứ tự đến (FIFO) |
| TC-P103-03 | AC1 (bảng làn) | route thử | **S** gọi 7 tác vụ không chỉ định làn, đọc `llm_audit.lane`; chỉ định hạ làn; yêu cầu `GRADING`/`QUESTION_GEN`/`INSIGHT` với `lane=INTERACTIVE`; `EMBEDDING` câu hỏi lúc chat nâng INTERACTIVE | `CHAT` INTERACTIVE; `CLASSIFY`, `UTILITY` NEAR_REALTIME; `GRADING`, `QUESTION_GEN`, `INSIGHT`, `EMBEDDING` BATCH; hạ làn được; ba tác vụ BATCH-only yêu cầu INTERACTIVE → `ErrBadLane` (lỗi 4xx rõ ràng); `EMBEDDING` nâng làn cho câu hỏi chat được |
| TC-P103-04 | AC2 | 2 gateway + Redis | **L** `go test -tags integration ./internal/llm/scheduler/... -run TestTwoProcessesGlobalConcurrency -v` | `ok`; in `peak_inflight ≤ 10`, `completed=400` |
| TC-P103-05 | AC2 (QC đo độc lập) | 2 gateway thật | **L** k6 bơm 200 yêu cầu `CHAT` `fake 200–400 ms` vào **mỗi** gateway đồng thời (qua cổng riêng của từng bản hoặc Caddy hai upstream); song song **D** lấy mẫu `ZCARD ep:llm:inflight:<provider>` mỗi 5 ms (`redis-cli` vòng lặp hoặc script Bun) | Đỉnh `ZCARD` **≤ 10** (cộng cả hai tiến trình); 400 yêu cầu hoàn tất (0 lỗi trừ `OVERLOADED` hợp lệ — nếu có, nêu số); sau khi xong `ZCARD`=0 trong ≤ lease; không rò chỗ |
| TC-P103-06 | AC2 (rò chỗ) | – | **D** `kill -9` một gateway khi đang giữ chỗ; theo dõi `ZCARD` | Chỗ của tiến trình chết được thu hồi sau ≤ lease (không giữ mãi); gateway còn lại không bị khoá chặt |
| TC-P103-07 | AC3 | `rpm_limit=60`, `tpm_limit=6000` | **G** `-tags integration -run 'TestTwoProcessesRPM\|TestTPMReconcile' -v` | `ok`; in số thực tế và trần |
| TC-P103-08 | AC3 (QC đo độc lập) | nhà `fake` `rpm_limit=60` | **L** bơm liên tục từ hai gateway trong 10 s; đếm `llm_audit` trong cửa sổ | Số lời gọi ≤ `60 × 10/60 + burst` = 10 + 6 = **16** (burst = 10 % = 6, tối thiểu 1); vượt hạn mức thì **chờ** (không lỗi `RATE_LIMIT` cho người gọi); tổng token ước tính ≤ `6000 × 10/60 + burst` |
| TC-P103-09 | AC3 | cột `rpm_limit`/`tpm_limit` null | **D** gửi tải vượt 60 rpm khi cột null | Dùng mặc định `LLM_DEFAULT_RPM=60`, `LLM_DEFAULT_TPM=100000` |
| TC-P103-10 | AC4 | `LLM_BATCH_SHARE=0.5` | **G** `-tags integration -run TestBatchDoesNotStarveInteractive -v` | `ok`; in `interactive_wait_p95_ms`, `ttft_ratio`, `batch_peak` |
| TC-P103-11 | AC4 (QC đo độc lập — số cụ thể) | 200 BATCH chờ, `fake 5–15 s` | **L** k6 (kịch bản `p103-batch-vs-chat.js`): 200 việc `GRADING` + 25 `CHAT` tốc độ ≤ 50 % công suất (≤ 5 đồng thời); đo `queue_wait_ms` của INTERACTIVE từ `llm_audit`; đo đỉnh BATCH đang chạy từ `ZCARD`; chạy lại 1 lần **không có BATCH** làm mốc | `interactive_wait_p95 ≤ 500 ms`; `batch_peak ≤ 5` **khi có INTERACTIVE chờ/vừa chạy trong 2 s**; chênh p95 so với mốc không BATCH ≤ 100 ms; TTFT (đo token đầu của `Stream`) không chậm hơn **+20 %** so với mốc |
| TC-P103-12 | AC4 (không INTERACTIVE) | – | **L** chỉ 200 BATCH | BATCH dùng tới **10** chỗ (không bị kìm khi rảnh) |
| TC-P103-13 | AC4 (tay) | – | **S** `cd backend-go && go run ./cmd/llmload --batch 200 --chat 25` | In `interactive_wait_p95_ms ≤ 500`, `batch_peak ≤ 5`; QC ghi số **in ra** cạnh số tự đo (TC-11); lệch lớn → báo "nghi lỗi công cụ" và lấy số tự đo |
| TC-P103-14 | AC5 | `LLM_QUEUE_MAX=200`, `LLM_MAX_CONCURRENCY=1`, `fake 10 s` | **S** 201 `POST /api/v1/_test/llm/chat` song song (k6/`xargs -P201`); đo thời gian phản hồi của yêu cầu bị từ chối | **Đúng 1** phản hồi `503` thân `{"code":"OVERLOADED",…,"retry_after":N}` + header `Retry-After: N`, `1 ≤ N ≤ 30`, trả **≤ 50 ms**, kết nối không bị giữ; `message` tiếng Việt |
| TC-P103-15 | AC5 | – | **S** yêu cầu INTERACTIVE chờ > 10 s (`LLM_QUEUE_WAIT_MAX=10s`) | Nhận `OVERLOADED` sau ≈ 10 s |
| TC-P103-16 | AC5 | – | **D** sau các lần từ chối: `inflight` và token bucket không đổi; `llm_audit.status='overloaded'` đúng số yêu cầu bị từ chối | Từ chối **không** chiếm chỗ / token; có dòng audit `overloaded` |
| TC-P103-17 | AC5 | – | **S** đổ đầy hàng BATCH và NEAR_REALTIME | `OVERLOADED` tương tự; công thức `retry_after = clamp(ceil(độ_dài_hàng ÷ số_chỗ × trễ_trung_bình_s), 1, 30)` (đối chiếu 3 điểm số) |
| TC-P103-18 | AC5 | – | **G** `-run 'TestQueueFullOverloaded\|TestQueueWaitMax\|TestRetryAfterFormula'` | `ok` |
| TC-P103-19 | AC6 | nhà `fake` lỗi 100 % `SERVER` + nhà B tốt | **S** 5 lời gọi liên tiếp rồi lời gọi thứ 6; đo độ trễ thêm; `GET /admin/llm/providers` (khi P1-04 có) | Sau 5 lỗi liên tiếp mạch **mở**: lời gọi 6 **không** tới nhà A (bỏ qua ngay sang B, thêm < 5 ms); `circuit: open` |
| TC-P103-20 | AC6 | – | **S** đợi 30 s → bán mở: gửi 3 lời gọi cùng lúc; cho A thành công / thất bại | Bán mở cho **đúng 1** lời gọi thử (2 lời gọi kia sang B); thành công → `closed`; thất bại → `open` lại 30 s |
| TC-P103-21 | AC6 | – | **S** 4 lỗi + 1 thành công + 4 lỗi; `BAD_REQUEST` ×10; `AUTH`, `RATE_LIMIT`, `TIMEOUT`, `NETWORK` ×5 | Thành công xen giữa đặt lại bộ đếm (không mở); `BAD_REQUEST` **không** tính; bốn loại còn lại đều tính tới 5 → mở |
| TC-P103-22 | AC6 (dùng chung) | 2 gateway | **D** gây 5 lỗi qua gateway 1; gọi qua gateway 2; `redis-cli GET ep:llm:cb:<provider_id>` | Gateway 2 thấy mạch **mở** ngay (không phải tự gom 5 lỗi); khoá Redis có trạng thái |
| TC-P103-23 | AC6 | – | **G** `-run 'TestBreakerOpensAfter5\|TestBreakerHalfOpen\|TestBreakerResetOnSuccess\|TestBreakerIgnoresBadRequest'`; `-tags integration -run TestBreakerSharedAcrossProcesses` | `ok` |
| TC-P103-24 | AC7 | `ctx` hạn 3 s | **S** route thử với `timeout`/header hạn 3 s (hoặc `curl -m 3`); `fake` 10 s; hàng chờ 2 s trước | Toàn bộ (chờ hàng + thử lại + fallback) **≤ 3 s**; trả `DEADLINE_EXCEEDED` (504) kèm `message` tiếng Việt; lời gọi nhà cung cấp dừng ngay (máy chủ giả thấy `close`) |
| TC-P103-25 | AC7 | – | **S** không hạn: INTERACTIVE/NEAR_REALTIME `fake` 40 s; BATCH `fake` 130 s; tuyến `params.timeout_s=10` rồi `=100` | 30 s → `DEADLINE_EXCEEDED`; BATCH 120 s; `timeout_s=10` thu hẹp thành 10 s; `timeout_s=100` **không nới** quá 30 s (INTERACTIVE) |
| TC-P103-26 | AC7 | – | **G** `-run 'TestDeadlineIncludesQueueWait\|TestDeadlineDefaults\|TestTimeoutParamOnlyNarrows'` | `ok` |
| TC-P103-27 | AC8 | `fake` 5–15 s | **S** `timeout 2 curl -sk -N -H "$A" -X POST $GW/api/v1/_test/llm/chat -d '{"task":"CHAT","prompt":"x","stream":true}'`; sau đó `curl -sk -H "$A" $GW/api/v1/_test/llm/stats \| jq .provider_inflight` trong ≤ 1 s; `$PSQL -c "select status from llm_audit order by created_at desc limit 1"` | `provider_inflight` → `0` trong **≤ 1 s**; `ZCARD inflight` = 0; dòng audit `cancelled`; **không** thử lại và **không** fallback sau huỷ (máy chủ nhà B 0 request); token đã sinh được tính ngân sách |
| TC-P103-28 | AC8 | – | **G** `-run 'TestClientCancelStopsProvider\|TestCancelNoRetryNoFallback'` | `ok` |
| TC-P103-29 | AC9 | tắt mọi nhà cung cấp (hoặc mạch hở hết) | **S** `POST /api/v1/_test/llm/chat {"task":"CHAT","prompt":"hạn nộp bài?","passages":[4 đoạn có điểm]}` | **HTTP 200**, `degraded:true`; `text` mở đầu **đúng** "AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:"; ≤ **3** đoạn điểm cao nhất, nguyên văn, mỗi đoạn kèm tên tài liệu + trang; `llm_audit.degraded=true` |
| TC-P103-30 | AC9 | – | **S** không có `passages`; yêu cầu BATCH (`GRADING`) khi mọi nhà chết | "AI tạm thời không khả dụng. Câu hỏi của bạn đã được ghi lại, giảng viên sẽ xem." `degraded:true`; BATCH **không** suy giảm (lỗi `LLM_UNAVAILABLE` 503 hoặc xếp hàng theo US) |
| TC-P103-31 | AC9 (từ cấm) | – | **S** `grep -ciE 'provider\|fallback\|trace\|RAG\|PII\|LLM\|prompt' <<< "$text"` trên cả hai chuỗi suy giảm | `0` |
| TC-P103-32 | AC9 | – | **G** `-run 'TestDegradedExtractive\|TestDegradedNoPassages\|TestBatchNoDegrade'` | `ok` |
| TC-P103-33 | AC10 | `Shareable=true`, NEAR_REALTIME/BATCH | **S** 50 yêu cầu giống hệt cùng lúc (cùng tiến trình); đếm lời gọi nhà cung cấp (máy chủ Q / `fake.Calls` / `llm_audit` số dòng `provider≠cache`) | **1** lời gọi nhà cung cấp; cả 50 nhận **cùng** kết quả |
| TC-P103-34 | AC10 (không gộp) | – | **S** 50 yêu cầu giống hệt ở `INTERACTIVE`; ở `GRADING`; ở `Shareable=false` (mặc định) | Mỗi cái 1 lời gọi riêng (50 lời gọi) — **không bao giờ** gộp |
| TC-P103-35 | AC10 | – | **S** lời gọi chung lỗi (503) cho 50 yêu cầu; sau đó 1 nhóm mới; huỷ 1 người chờ trong 50 | Lỗi chia cho cả nhóm; nhóm mới sau đó được thử lại **độc lập**; huỷ 1 người **không** huỷ lời gọi khi còn người chờ |
| TC-P103-36 | AC10 | – | **G** `-run 'TestSingleFlightShareable\|TestNoSingleFlightInteractive\|TestSingleFlightCancelOneWaiter'` | `ok` (`fake.Calls == 1`) |
| TC-P103-37 | AC11 | `daily_limit=100.000`, `monthly_limit=2.000.000` (VND, qua dịch vụ / API); giá `fake` biết trước | **D** đẩy chi phí tới 79 %, 80 %, 85 %, 100 %: đọc `outbox` `topic='llm.budget.warn'` và log | Tại ≥ 80 %: trạng thái `warn`, **đúng 1** sự kiện outbox `llm.budget.warn` cho mỗi phạm vi mỗi kỳ (tăng thêm 5 % không thêm sự kiện), log `warn`; phép tính tiền bằng `decimal` (không lệch 1 đồng; đối chiếu bảng tay) |
| TC-P103-38 | AC11 | – | **S** ở 100 %: gửi `GRADING` (BATCH); gửi `CHAT` và `CLASSIFY` | BATCH → `LLM_UNAVAILABLE` `details.reason="budget_exhausted"`; INTERACTIVE/NEAR_REALTIME → **mô hình rẻ nhất đang bật** (`price_in+price_out` nhỏ nhất, đối chiếu `llm_audit.model`) và **vẫn trả lời**; chat sinh viên **không bao giờ** bị tắt |
| TC-P103-39 | AC11 | – | **S** hai phạm vi (hệ thống + lớp): lớp cạn, hệ thống chưa; sang ngày mới (đổi ngày hệ thống bằng đồng hồ test / tạo bản ghi ngày kế) | Phạm vi nào chạm trước chi phối đúng phạm vi đó; sang ngày: trạng thái ngày về `ok`, sự kiện `warn` lại được phát một lần cho kỳ mới |
| TC-P103-40 | AC11 | – | **G** `-run 'TestBudgetWarn80Once\|TestBudgetExhaustedBatchStops\|TestBudgetExhaustedInteractiveCheapest\|TestBudgetTwoScopes\|TestBudgetDayRollover'`; `grep -rnE 'float(32\|64)' backend-go/internal/llm/budget backend-go/internal/llm/cost --include=*.go \| grep -v _test.go \| wc -l` | `ok`; `0` |
| TC-P103-41 | AC12 | – | **S** gây từng từ chối: `OVERLOADED`, `LLM_UNAVAILABLE` (ngân sách), `LLM_NOT_CONFIGURED`, `DEADLINE_EXCEEDED`; đọc thân + `llm_audit` + bộ đếm | Mỗi lần có mã lỗi + `message` **tiếng Việt** + (nếu có) `retry_after`; **không** timeout trơn / đóng kết nối câm; có dòng `llm_audit` mỗi lần |
| TC-P103-42 | AC12 (giữa luồng) | stream đang chạy | **S** giết nhà cung cấp giữa chừng khi đã gửi vài token; và khi **chưa** gửi byte nào | Đã gửi byte: `event: error` có `code` rồi đóng; chưa gửi byte: chuyển fallback; không treo |
| TC-P103-43 | AC12 | – | **G** `-run 'TestEveryRejectionHasCode\|TestStreamMidFailure'` | `ok` |
| TC-P103-44 | AC13 | Redis | **G** `-tags integration -run 'TestRedisDownFailsOpenLocal\|TestRedisBackReconciles' -v` | `ok` |
| TC-P103-45 | AC13 (QC đo độc lập) | stack | **S** `$C stop redis` giữa lúc bơm tải 10 req/s; đếm thành công; log `error` mỗi 30 s; `$C start redis` | **Không treo**: yêu cầu vẫn đi (giới hạn cục bộ); log `error` ≤ 1 lần / 30 s; `llm_audit` vẫn ghi đủ; Redis về → quay lại toàn cục **không restart** (`ZCARD` có lại); ngân sách đối soát lại từ `llm_audit` (chi phí bằng tổng `cost_est`) |
| TC-P103-46 | AC14 | stack | **S** `curl -sk -o /dev/null -w '%{http_code}\n' -H "$T" $GW/api/v1/_test/llm/stats`; `$TA_`, `$S`, không token; binary không `testroutes`; `curl -sk -H "$A" …/stats \| jq 'keys'` | `403` cho TEACHER/TA/STUDENT; `401` không token; `404` binary thường; `keys` **không** có `prompt` / `text` |
| TC-P103-47 | AC15 | – | **S** từng biến sai: `LLM_BATCH_SHARE=1.5`, `0`, `-1`; `LLM_MAX_CONCURRENCY=0`, `-3`; `LLM_QUEUE_MAX=-1`; `LLM_BREAKER_FAILS=0`; `timeout 10 ./bin/gateway serve 2>&1 \| grep -c <TÊN_BIẾN>` | Mỗi lần gateway **từ chối khởi động** (`rc≠0`) và thông báo nêu **tên biến**; `LLM_BATCH_SHARE=1` được chấp nhận |
| TC-P103-48 | AC15 | – | **S** không đặt biến nào; đọc giá trị hiệu lực (log khởi động / `stats`) | Mặc định đúng: 10 / 0,5 / 200 / 10 s / 30 s / 5 / 30 s / 60 / 100000; `go test ./internal/config/... -run TestLLMEnv -v` → `ok` |
| TC-P103-49 | tổng | – | **S** `cd backend-go && go vet ./... && golangci-lint run && go test -race ./... && go test -tags integration ./internal/llm/scheduler/... ; echo rc=$?` | `rc=0` (QC chạy tự; lặp `-count=3` các test tích hợp để bắt flaky) |
| TC-P103-50 | SLO | – | **L** ghi số đo vào report: p95 chờ hàng INTERACTIVE, TTFT ratio, đỉnh inflight, cấu hình máy | Số đo thực; không nới ngưỡng; hand-measure thắng công cụ |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| BATCH làm chat sinh viên treo (luật 11) | 11, 12 |
| Vượt hạn mức đồng thời khi 2 gateway | 05, 06 |
| Hàng đợi vô hạn / giữ kết nối khi quá tải | 14–16 |
| Mạch mở không bao giờ đóng; `BAD_REQUEST` làm mở mạch | 20, 21 |
| Chờ hàng không trừ vào hạn | 24 |
| Huỷ mà vẫn gọi nhà cung cấp / thử lại sau huỷ | 27 |
| Mọi nhà chết → im lặng / lộ từ kỹ thuật | 29–31 |
| Gộp nhầm yêu cầu cá nhân / INTERACTIVE | 34 |
| Ngân sách cạn tắt chat sinh viên | 38 |
| Redis mất → treo | 45 |
| Tiền bằng `float64` | 37, 40 |

## Câu hỏi cho BA / PM
- **Q-QC-P103-1** — AC4 đo TTFT "+20 %" cần so với một mốc: QC dùng mốc "không có BATCH" cùng cấu hình `fake`; `fake` có độ trễ ngẫu nhiên 5–15 s nên QC cần `FAKE` độ trễ cố định cho mốc. Chấp nhận đặt `fake` độ trễ cố định cho TC-P103-11? — *chờ trả lời*.
- **Q-QC-P103-2** — AC13 Redis mất: QC dừng container Redis thật (`$C stop redis`) — có ảnh hưởng 2 gateway khác (idempotency, rate limit PG)? QC chấp nhận ghi nhận hành vi, chỉ chấm phần Scheduler. — *chờ trả lời*.
- **Q-QC-P103-3** — AC11 "chi phí ước tính của ngày": múi giờ chuyển ngày là UTC hay `Asia/Ho_Chi_Minh`? TC-P103-39 cần biết. — *chờ trả lời*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-llm-gateway, APPROVED 2026-10-03).

Tổng: 50 TC.
