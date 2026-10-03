# Báo cáo QC — GATE-P1 (cổng nghiệm thu phase P1: LLM Gateway)
**Kết luận: FAIL** — hai lý do: (a) CI GitHub đỏ ở HEAD (TC-11, cùng BUG-PU05-1); (b) `cmd/llmload` do dev viết báo `interactive_wait_p95_ms=1430` > 500 ở lần chạy chính (TC-16) — số tự đo của QC đạt, nhưng chưa giải thích được lệch. Phần còn lại đạt; "2 nhà thật" và ghi âm thật BLOCKED (thiếu khoá).

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
