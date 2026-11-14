# QC — kiểm lát L1 (US-DM-01, US-DM-02) ở `425d2a74` — mức Go + DB (chưa dựng stack)
2026-11-14. Không dựng `ep-qc`; DB trống tạm (`pgvector/pgvector:pg18`, đã dừng) + `go test`. Không gọi mạng ngoài, không khoá. Kết luận **một phần**: phần kiểm được PASS; phần cần stack / lát sau xếp bên dưới. Không phải kết luận story.

## Đã chạy
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| TC-D01-24 (6 test AC9) | PASS | `TestDecideAudit`, `TestDecideNoSDK`, `TestDecideMasked`, `TestDecideErrorMap`, `TestDecideRealGuard`, `TestDecideFakeDeterministic` đều PASS (`go test ./internal/llm/... -v`); thêm `TestDecideFakeOverrides`, `TestDecideNotConfigured`, `TestDecideBucketAndNoWait`, `TestDecideTimeoutRedirectAndNoGeneration`, `TestDecideInputCeiling`, … PASS |
| TC-D02-01 | PASS | `TestCascadeTable`, `TestCascadeFailClosed`, `TestCascadeTimeout`, `TestCascadeNotUsed` PASS |
| TC-D02 (cấu hình hiệu lực / làm mới) | PASS mức Go | `TestConfigEffective`, `TestConfigSnapshotRefresh` PASS (chưa kiểm hai gateway, Q-S105-8) |
| TC-D01-02 | PASS | `go list -deps ./...` = 0 gói `genai` / `generative-ai-go`; `go.mod`, `go.sum`, `package.json` không đổi |
| TC-D01-22 | PASS | `.env.example`: `LLM_DECISION_ALLOW_REAL=false`; compose chỉ chuyển biến (`${…:-}`), không đặt `true` |
| TC-D01-28, D08-19 (phần migration) | PASS | `gateway migrate up` → `down` (về 0) → `up` trên DB trống: sạch, `version=20`; `00020_decision.sql` không sửa file cũ |
| TC-D01-23 | PASS | `llm_models`: `gemini-decision-1` `price_in=2600`, `price_out=0`; `decision-fake` 0 |
| TC-D01-28 | PASS | chỉ mục duy nhất `llm_providers_one_decision` chặn provider `decision` thứ hai (`23505`); `type` / `kind` `decision` hợp lệ |
| TC-D02-13 | PASS mức DB | `decision_gates` seed 3 hàng `enabled=false`: THREADS 0,050 / 800, INJECTION 0,500 / 3000, CHAT 0,950 / 800; `course_decision_gates` rỗng |
| SRS §5.2 | PASS | `llm_audit.cascade_gate`, `cascade_path` có; CHECK `cascade_path` ∈ FAST / FALLBACK / FAIL_CLOSED; định nghĩa CHECK đọc từ `pg_get_constraintdef` (không có `FAIL_RULES`; thử chèn bị chặn trước bởi NOT NULL `task` nên chưa chạm CHECK); chỉ mục `llm_audit_decide_idx (created_at, cascade_gate) WHERE task='DECIDE'`; CHECK `gate`, `timeout_ms` 200–5000 chặn giá trị sai |
| Quét khoá | PASS | diff L1 không chứa mẫu khoá `AIza…` / `sk-…` |

## Ghi nhận (chưa là lỗi)
- Dev sửa test cũ **theo spec**: `TestClientSurface` 4 → 5 hàm (+`Decide`), `TestNoNPlusOne` `Len 20 → 21` (provider `decision` seed), `ps[0]` → `ps[1]`; `embedStub` thêm `Decide`. Khớp SRS 4.1 / 5.1 (handoff nêu). Không thấy nới assertion / `t.Skip`.
- `TestNoNPlusOne` giờ phụ thuộc thứ tự `ps[1]` — ghi nhận dễ vỡ, không lỗi.

## Chưa kiểm được ở lát này (cần stack và / hoặc lát sau)
- **HTTP API:** `openapi.yaml` chưa có endpoint SRS §6 #1–5, #7 (lát L5); #6 (provider `type=decision`, `409 DECISION_REAL_DISABLED`, `POST …/test`) kiểm khi có stack: TC-D01-19, 20, 26, 27.
- **TC cần cổng nối vào Threads / chấm / chat** (D01-09…13, D02-03…10, 14, 20, 23): đợi L2–L4.
- TC-D01-03, 15, 21 (máy chủ giả): cần stack + base URL (`LLM_DECISION_BASE_URL` / cột `base_url`) — dựng khi có stack.
- TC-D02-16 (hai gateway): chờ PM xếp RAM (Q-S105-8).

## Đề nghị
Dựng `ep-qc` **một lần** khi L2 (Threads) xong để kiểm API provider + cổng đầu tiên cùng lúc, thay vì dựng riêng cho L1 (build ≈ 5–10 phút, tốn RAM).
