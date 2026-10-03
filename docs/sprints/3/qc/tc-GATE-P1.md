# QC test case — GATE-P1 (cổng nghiệm thu phase P1 + "Bạn tự kiểm")
Nguồn: `docs/phases/P1.md` mục "Cổng nghiệm thu" và "Bạn tự kiểm" (nguyên văn, từng dòng → bước đo được) + `docs/sprints/3/plan.md` (quyết định phạm vi sprint 3: `make eval` hoãn; chỉ `TestProviderContract` với `fake` + replay; ghi khoá thật = BLOCKED). Hộp đen cho đến khi chấm. Chạy ở gốc `TA_Agent_v2-s3` (nhánh `sprint/3-pu-p1`) **sau khi 5 story P1 có `report-US-P1-0N.md` PASS**; stack test (2 gateway, Postgres, Redis, Caddy) + `go` + `golangci-lint` + `sqlc` + `k6` + `jq`. Một kết luận PASS/FAIL cho mỗi TC; thiếu công cụ → FAIL "KHÔNG KIỂM ĐƯỢC"; mục chỉ làm được với **khoá nhà cung cấp thật** → **BLOCKED** ghi rõ (không FAIL, không giả).

## Bảng A — Cổng nghiệm thu (P1.md)
| TC-id | P1.md | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-GATEP1-01 | dòng 1 | `cd /Users/kuro/Documents/TA_Agent_v2-s3 && grep -rn "openai\.\|anthropic\.\|genai\." backend-go --include=*.go \| grep -v "internal/llm/" ; test $? -eq 1; echo rc=$?` + `grep -E 'anthropic\|generative-ai\|genai' backend-go/go.mod \| wc -l` | `rc=0` (không dòng nào); `0`; chỉ `openai-go` được thêm vào `go.mod` |
| TC-GATEP1-02 | dòng 2 | `cd backend-go && go test -race -count=1 ./internal/llm/...; echo rc=$?` | `rc=0`; không `--- SKIP` ngoài `BLOCKED` của provider thật |
| TC-GATEP1-03 | dòng 3 | `go test -race -count=1 ./internal/llm/scheduler/...` + `go test -tags integration -count=1 ./internal/llm/scheduler/...` (Redis thật, 2 tiến trình) | `rc=0`; BATCH không làm INTERACTIVE chờ quá ngưỡng; hàng đầy → `OVERLOADED`; mạch mở/đóng; huỷ / deadline dừng lời gọi; in `peak_inflight ≤ 10`, `interactive_wait_p95_ms ≤ 500`, `batch_peak ≤ 5` |
| TC-GATEP1-04 | dòng 3 (lặp) | chạy lại dòng 3 `-count=3` | 3/3 xanh (không flaky); ghi thời gian |
| TC-GATEP1-05 | dòng 4 | `cd backend-go && go test ./internal/llm -run TestProviderContract -v; echo rc=$?` | `rc=0`; `PASS` cho `fake` và `fake-replay`; provider thật `SKIP BLOCKED` (xem TC-GATEP1-18) |
| TC-GATEP1-06 | dòng 5 | `go test -race -count=1 ./internal/llmconfig/... ./internal/platform/...; echo rc=$?` | `rc=0` |
| TC-GATEP1-07 | dòng 6 | `make eval` | **Phạm vi sprint 3 (plan): hoãn** — QC chỉ ghi "không thuộc sprint 3 (cần RAG + bộ vàng, P3/P10)"; nếu `make eval` có target và chạy được → ghi kết quả, nhưng **không** là điều kiện PASS của cổng sprint này |
| TC-GATEP1-08 | cả bộ | `cd backend-go && go vet ./... && golangci-lint run && sqlc diff && go test -race -count=1 ./... && go test -count=1 ./internal/contract/...; echo rc=$?` | `rc=0` (toàn bộ backend, gồm contract test PG + LLM) |
| TC-GATEP1-09 | PG không vỡ | `bash docs/sprints/2/qc/scripts/gate-pg.sh` (các TC không đụng volume) hoặc chạy lại `tc-GATE-PG` mục A; `goose status`; 2 gateway healthz/readyz | Cổng PG còn **xanh** (nguyên tắc 7, 8: hợp đồng PG nguyên vẹn); phiên bản goose = 2; `/api/v1/healthz`, `readyz` 200 |
| TC-GATEP1-10 | frontend không vỡ | `pnpm -C frontend lint && pnpm -C frontend build`; `bash scripts/ui-antipatterns.sh`; `audit.mjs` + `sweep.mjs` + `proto-curl.sh all` (như `audit-baseline.md`) | rc=0; FAIL 0, PASS ≥ nền; ≥ 497 PASS; `FORBIDDEN`=0 |
| TC-GATEP1-11 | CI | `gh run list --workflow ci.yml --branch sprint/3-pu-p1 --limit 1` → `headSha` = `git rev-parse origin/sprint/3-pu-p1`; cả hai job `Go`, `Frontend` | `conclusion=success` ở HEAD |

## Bảng B — "Bạn tự kiểm" (P1.md)
| TC-id | P1.md | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-GATEP1-12 | Tự kiểm 1a | Thêm **2 nhà cung cấp `fake`** ở `/settings/llm` (hoặc API); đổi `CHAT` giữa hai nhà; chat ngay (`_test/llm/chat`) sau mỗi lần; `select provider, model, fallback_index, created_at from llm_audit order by created_at desc limit 6` | Mỗi lần đổi, lời gọi kế tiếp dùng nhà mới ≤ 1 s; `llm_audit` ghi **đúng** nhà / mô hình của từng lời gọi |
| TC-GATEP1-13 | Tự kiểm 1b | "2 provider **thật**" (OpenAI, Gemini, …) | **BLOCKED — thiếu khoá nhà cung cấp thật** (chỉ chứng minh được bằng `fake`); ghi rõ, chủ dự án tự thử khi có khoá |
| TC-GATEP1-14 | Tự kiểm 2 | Nhập khoá sai (`bad-key`) → `Test kết nối` báo lỗi rõ ở UI; thao tác đủ loại với `$CANARY`; quét: DOM / localStorage / console / HAR (UI), `docker compose logs`, `audit_log`, `outbox`, `llm_audit`, `jobs`, `pg_dump` | Báo lỗi rõ tiếng Việt; khoá **không** hiện lại ở UI; `$CANARY` và đuôi **0 lần** ở mọi nơi (log gồm gateway, caddy, postgres, redis) |
| TC-GATEP1-15 | Tự kiểm 3 | Tắt nhà chính (công tắc) → `_test/llm/chat` | `fallback_index:1`; `llm_audit.fallback_index=1`; log `warn` có `from`/`to`; chat không lỗi |
| TC-GATEP1-16 | Tự kiểm 4 | `go run ./cmd/llmload --batch 200 --chat 25` **và** k6 riêng của QC (TC-P103-11), `fake 5–15 s`, `LLM_MAX_CONCURRENCY=10` | `interactive_wait_p95_ms ≤ 500`; `batch_peak ≤ 5`; TTFT INTERACTIVE không chậm hơn **+20 %** so với mốc không BATCH; "chữ đầu tiên vẫn ra trong SLO" (`SYSTEM_DESIGN.md` §5); số tự đo thắng số công cụ khi lệch |
| TC-GATEP1-17 | Tự kiểm 5 | Tắt hết nhà cung cấp → `_test/llm/chat` với `passages` | HTTP 200, `degraded:true`; mở đầu "AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:"; trích ≤ 3 đoạn nguyên văn; **không treo** (phản hồi ≤ 2 s); không từ kỹ thuật |
| TC-GATEP1-18 | Contract nhà cung cấp thật | `LLM_RECORD=1 … go test ./internal/llm -run TestRecordReplay` | **BLOCKED — thiếu khoá nhà cung cấp thật** (ghi, không FAIL) |
| TC-GATEP1-19 | Luận văn | Ghi cho luận văn: sơ đồ gateway; bảng tác vụ → mô hình (từ `GET routes`); số đo tải TC-GATEP1-16; ghi chú "golden set theo nhà cung cấp: hoãn (cần khoá thật)" | `report-GATE-P1.md` có bảng tác vụ → mô hình, số đo tải, ảnh `/settings/llm` trước/sau (`shots/before` vs `shots/after`) |

## Bảng C — Tấn công tổng hợp (QC, bổ sung ngoài P1.md)
| TC-id | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- |
| TC-GATEP1-20 | Ma trận quyền 13 thao tác × 6 danh tính (TC-P104-01) chạy lại **trên stack cuối** + token giả (`alg=none`, secret sai, sửa vai) | Không ca nào lệch; 0 thay đổi DB từ token giả |
| TC-GATEP1-21 | Canary toàn hệ thống (TC-P104-16) lặp lại sau khi chạy tải TC-GATEP1-16 | `$CANARY` 0 lần ở mọi log / bảng / dump |
| TC-GATEP1-22 | Chạy cổng PG + P1 **hai lần liên tiếp** trên DB mới (`down -v` rồi `up`) | Cả hai lần xanh; migration `00002` áp được từ DB trống và từ DB đã có dữ liệu PG |

## Dọn dẹp
| TC-id | Bước | Kết quả mong đợi |
| --- | --- | --- |
| TC-GATEP1-99 | `$C down -v`; xoá container thử, máy chủ giả Q, Chrome tạm; `docker ps` | Không container / tiến trình QC sót; cây git chỉ có artefact QC |

## Câu hỏi cho BA / PM
- **Q-QC-GATEP1-1** — Cổng P1 gồm `make eval` (P1.md dòng 6) nhưng plan sprint 3 hoãn; QC ghi TC-GATEP1-07 là "ngoài phạm vi sprint" (không FAIL). Đồng ý? — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Đồng ý. `make eval` hoãn (plan sprint 3, D53); TC-GATEP1-07 ghi "ngoài phạm vi", không FAIL. Nợ ghi vào `PROGRESS.md` (P3 / P10).
- **Q-QC-GATEP1-2** — "2 provider thật" cần khoá: PM / chủ dự án có cung cấp khoá thử (hạn mức nhỏ) cho QC chạy TC-GATEP1-13/18 không? Nếu không, giữ BLOCKED và đưa vào "Nợ". — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Khoá thật là việc của chủ dự án (Q11 của `FEAT-llm-gateway`), chưa có. Giữ BLOCKED có chủ đích cho TC-GATEP1-13 / 18 và đưa vào "Nợ"; các TC còn lại dùng `fake`. Không FAIL.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo P1.md + plan sprint 3.

Tổng: 23 TC (2 BLOCKED có chủ đích: 13, 18; 1 ngoài phạm vi: 07).
