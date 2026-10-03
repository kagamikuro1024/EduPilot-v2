# Báo cáo QC — GATE-P1 (cổng nghiệm thu phase P1: LLM Gateway)
**Kết luận cuối (vòng sửa 1): PASS** (2 nội dung BLOCKED vì thiếu khoá thật; `gate-pg.sh` / TC-22 chưa chạy, xem bảng). Vòng 1 là FAIL vì hai lý do: (a) CI GitHub đỏ ở HEAD (TC-11, cùng BUG-PU05-1); (b) `cmd/llmload` do dev viết báo `interactive_wait_p95_ms=1430` > 500 ở lần chạy chính (TC-16) — số tự đo của QC đạt, nhưng chưa giải thích được lệch. Phần còn lại đạt; "2 nhà thật" và ghi âm thật BLOCKED (thiếu khoá).

| TC | KQ | Số đo |
| --- | --- | --- |
| 01 | PASS | `grep openai\.|anthropic\.|genai\.` ngoài `internal/llm/` = 0 dòng; `go.mod` chỉ có `openai-go/v3` (`anthropic|generative-ai|genai` = 0) |
| 02 | PASS | `go test -race -count=1 ./internal/llm/...` rc=0 (5 gói `ok`) |
| 03, 04 | PASS | `scheduler` `-race -count=3` ok (7,8 s); `-tags integration` Redis thật: `peak_inflight=10 completed=400`, `interactive_wait_p95_ms=2` (cơ sở 4), `ttft_ratio=0.98`, `batch_peak=5` |
| 05 | PASS | `TestProviderContract`: fake, fake-replay, **openai, gemini** PASS; anthropic SKIP (BLOCKED) |
| 06 | PASS | `-race ./internal/llmconfig/... ./internal/platform/...` rc=0 (9 gói `ok`) |
| 07 | N/A (BA: hoãn) | `make eval` không thuộc sprint 3 |
| 08 | PASS | `go vet ./...` và `-tags testroutes` ok; `golangci-lint` 0 issues; `sqlc diff` sạch; `go test -race -count=1 -tags testroutes ./...` rc=0 (23 gói ok); contract rc=0 |
| 09 | PASS một phần | `goose` version 2; 2 gateway `readyz` 200; `go test` toàn bộ (gồm contract PG + testcontainers) rc=0. QC **không** chạy `gate-pg.sh` (nó `down -v` stack `edupilot-test-*` không phải của QC) |
| 10 | PASS | lint, pbuild, antipatterns rc=0; `audit.mjs` 674/674, sweep 0 xấu, `proto-curl` 493/0 (xem GATE-PU) |
| 11 | **FAIL** | CI đỏ ở HEAD (Go `success`, Frontend `failure`): xem GATE-PU TC-09 / BUG-PU05-1 |
| 12 | PASS | 2 nhà `fake` (Fake-A, Fake-B) chuyển tuyến CHAT: sau PUT, lời gọi kế tiếp dùng nhà mới sau **93 / 19 / 18 ms**; mọi PUT 200; `llm_audit` ghi đúng nhà/mô hình |
| 13 | BLOCKED | không có khoá nhà thật |
| 14 | PASS | `bad-key` → UI "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá."; canary `sk-LEAK-CANARY-5e9d21b7` + `good-key`/`bad-key`: 0 lần ở DOM, storage, console, log 2 gateway, `audit_log`, `pg_dump` (xem report P1-05 TC-09) |
| 15 | PASS có ghi chú | tắt nhà chính (còn nhà phụ trong chuỗi): chat qua nhà phụ, `fallback_index=0` (đếm trong chuỗi **đang bật**, như `scenario-P1.md` bước 10). Nhà chính **lỗi nhưng còn bật** (Q-local, cổng đóng): `Fake-A/fake-chat`, **`fallback_index=1`**, `llm_audit fb=1`, log `WARN "chuyển nhà cung cấp dự phòng" from=Q-local to=Fake-A error_kind=NETWORK`. Câu gate "tắt nhà chính ⇒ fallback_index:1" lệch hành vi; chờ BA (đã nêu ở P1-03) |
| 16 | **FAIL (số công cụ) / PASS (số tự đo)** | `llmload -batch 200 -chat 25` (`LLM_MAX_CONCURRENCY=10`, `fake` 5–15 s): `interactive_wait_p95_ms=1430`, `batch_peak=5`, chat 25/25 200, `batch_ok=11` → công cụ báo "KHÔNG ĐẠT". Tự đo (trạng thái sạch, 200 BATCH, CHAT ≤ 4 đồng thời, 20 lượt): `queue_wait_ms` p95 **2 ms**, max 132 ms; tổng thời gian CHAT p50 7221 ms so với 7900 ms mốc không BATCH (**không chậm hơn**). Chưa giải thích lệch 1430 vs 2 (nghi tải CHAT đúng ngưỡng 5 đồng thời + 5 BATCH = 10 = công suất). Quan sát: với `fake` 5–15 s, 186/200 BATCH hết hạn `DEADLINE_EXCEEDED` (đúng thiết kế, hạn BATCH 120 s) và chuỗi hết hạn làm **mạch** nhà `Fake-A` mở: ngay sau đó BATCH nhận `503`, CHAT sang đường rút gọn. Đề nghị dev xác nhận hết hạn BATCH có nên tính vào cầu dao nhà cung cấp |
| 17 | PASS | tắt hết nhà: HTTP 200, `degraded:true`, "AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:" + **3 đoạn** nguyên văn (4 đoạn truyền vào), phản hồi 6 ms |
| 18 | BLOCKED | không có khoá nhà thật (ghi âm openai/gemini đã có; anthropic thiếu) |
| 19 | một phần | bảng tác vụ → mô hình lấy từ `GET /admin/llm/routes`; ảnh `shots/after/settings-llm-*.png`; `shots/before` của `/settings/llm` có (12 ảnh); golden set theo nhà: hoãn (cần khoá thật) |
| 20 | PASS | ma trận 13 thao tác × 8 danh tính (ADMIN, TEACHER, TA, STUDENT, không token, `alg=none`, secret sai, sửa vai): ADMIN 2xx/422/404, TEACHER đọc 200 và mọi ghi **403**, TA/STUDENT **403**, 4 loại không-hợp-lệ **401**; DB trước/sau `6/4/12/25 → 6/4/12/25` |
| 21 | PASS | canary toàn hệ thống sau tải: 0 ở log, `audit_log`, `pg_dump` |
| 22 | chưa chạy | hai lượt trên DB mới (`down -v`) — QC không phá DB `edupilot-test-*`; DB QC (`qcp1-pg`) áp migration 00002 từ trống (version 2) |
| 99 | PASS | dọn dẹp ở cuối phiên |

## Việc sau
Dev: BUG-PU05-1; giải thích `llmload` 1430 ms (hoặc chỉnh công cụ); xác nhận hết hạn BATCH vs cầu dao. BA: lệch "tắt nhà chính ⇒ fallback_index".

## Vòng sửa 1 (dev `eb563fe`; QC chấm lại trên stack riêng, worktree `eb563fe`)
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 11 | **PASS** | CI ở HEAD `eb563fea`: run `37141391915` `success` (Go + Frontend); hai run trước cũng `success` |
| 16 | **PASS** (có điều kiện đo nêu rõ) | (1) **Điều kiện đo của dev:** `fake` 5–15 s + `LLM_MAX_CONCURRENCY=10` + `LLM_DEFAULT_RPM=6000 LLM_DEFAULT_TPM=100000000`: `llmload -batch 200 -chat 25` → `interactive_wait_p95_ms=1`, `batch_peak=5`, chat 25/25, `batch_rejected=0`, `rc=0`. (2) **QC tái hiện giải thích của dev:** cùng lệnh với RPM mặc định 60 → `interactive_wait_p95_ms=1338`, rc=1 và công cụ **in gợi ý** "chờ token RPM, không phải hàng đợi" — số 1430 cũ là chờ **token RPM** (5 BATCH + 5 CHAT ≈ 1 yêu cầu/s = 60 RPM), không phải hàng Scheduler. (3) **Số tự đo (4 CHAT đồng thời, 200 BATCH, RPM 6000):** `queue_wait_ms` p95 **2 ms** (mốc không BATCH: 2 ms), max 2 ms; thời gian CHAT p50 10.193 ms so với 10.041 ms mốc (**+1,5 % ≤ +20 %**); 20/20 CHAT 200, 0 `degraded`. (4) **Cầu dao:** sau 188/200 BATCH hết hạn `DEADLINE_EXCEEDED`, mạch `Fake-A` vẫn `closed`; `INSIGHT`, `CHAT`, `INSIGHT` (gateway 2) kế tiếp đều `200` (trước: BATCH nhận 503). `TestOwnDeadlineNotCountedToBreaker` PASS; `go test -race ./internal/llm/...` 131 test ok. **Ghi chú thiết kế (dev đã nêu, không FAIL):** khi RPM của nhà thật cạn, INTERACTIVE vẫn chờ ≤ 1 token vì chưa giữ riêng RPM cho INTERACTIVE — đề nghị BA/PM cân nhắc (mặc định 60 RPM là thấp cho 1.000 SV) |
| 15 (+ TC-P102-19, TC-P104-46, TC-P105-22 theo #33) | **PASS** | nghĩa mới: (a) tắt nhà chính bằng công tắc → nhà kế trả lời, `fallback_index=0`, `llm_audit fb=0`; (b) nhà chính **lỗi còn bật** (`openai_compatible` cổng đóng) → `fallback_index=1`, `llm_audit fb=1 ok`, log `WARN "chuyển nhà cung cấp dự phòng" from=Q-down to=Fake-A error_kind=NETWORK`; khớp nghĩa #33 ở cả bốn TC đã sửa |
Chưa đổi: TC-09/22 (`gate-pg.sh`, hai lượt DB mới) vẫn không chạy vì `down -v` stack `edupilot-test-*` không phải của QC — ghi ở lượt đầu; TC-13/18 BLOCKED (thiếu khoá thật).

## Bổ sung: TC-13 và TC-18 chạy với khoá thật (PM giao; khoá lấy từ `.env.local` gitignored, **không in / không chép / không commit**)
Giới hạn đã giữ: `gpt-4o-mini` và `gemini-3.6-flash`, `max_tokens` ≤ 300, ≤ 10 lần gọi mỗi nhà. Stack riêng của QC (Postgres + Redis + 2 gateway + worker, bản `sprint/4-p2` đã gồm P1); cấu hình qua **API** `/admin/llm/*` (cùng đường UI gọi), khoá được xoá khỏi DB QC sau khi chạy.
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 13 | **PASS** | thêm 2 nhà thật: `OpenAI` (`gpt-4o-mini`) và `Gemini` (`gemini-3.6-flash`) → `201`, `has_key=true`, `last_test.ok=true` cả hai (mỗi nhà **1** lời gọi kiểm tra kết nối). Tuyến `CHAT=[OpenAI, Gemini]`, `max_tokens=300`: (1) hỏi thử → **OpenAI** trả lời (`fallback_index=0`, vào 25 / ra 8 token); (2) **tắt OpenAI** bằng công tắc → **Gemini** trả lời (`fallback_index=0` theo nghĩa #33: tắt ⇒ 0, vào 16 / ra 8); (3) đảo `CHAT=[Gemini, OpenAI]` → Gemini chính trả lời (vào 16 / ra 8). `llm_audit` đúng 3 dòng `ok`, khoá không xuất hiện ở thân phản hồi / log |
| 18 | **PASS** (openai, gemini); anthropic **BLOCKED** | `LLM_RECORD=1 go test ./internal/llm -run TestRecordReplay`: openai **4** lời gọi (3 Structured + 1 nhúng 1536 chiều; 3 lần thử HTTP cho Structured), vào 335 / ra 349 token; gemini **4** lời gọi (3 lần thử HTTP cho Structured), vào 48 / ra 313 token. Phát lại `env -i`: `TestProviderContract` fake, fake-replay, **openai, gemini PASS**, anthropic SKIP (không có khoá); `TestEmbedReplay` openai + gemini PASS; `TestReplayHasNoSecrets` PASS; QC quét độc lập: khoá thật trong `testdata/replay` = **0** tệp, mẫu `Authorization|Bearer|sk-|AIza|org-|x-api-key` = **0**. Bản ghi mới chỉ dùng để thử (đã `git checkout` lại, không commit) |
**Tổng số lần gọi nhà thật:** OpenAI = 1 (kiểm tra) + 1 (hỏi) + 4 (ghi) = **6**; Gemini = 1 (kiểm tra) + 2 (hỏi) + 4 (ghi; ≤ 3 lần thử HTTP cho Structured) = **7** — đều ≤ 10; token tổng ≈ OpenAI 335 vào / 357 ra, Gemini 80 vào / 329 ra (cộng cả hai phần). **Kết luận GATE-P1: không còn BLOCKED ngoài anthropic** (chưa có khoá).
