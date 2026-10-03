# DEV handoff — US-P1-01 (lược đồ LLM, mã hoá khoá, dịch vụ cấu hình)
Nhánh `sprint/3-pu-p1`. Tầng dịch vụ + kho + mã hoá; chưa có handler (US-P1-04).

## Làm gì
- `db/migrations/00002_llm.sql`: 5 bảng đúng SRS 5.2–5.3 (ràng buộc đặt tên, chỉ mục, trigger `set_updated_at`, không FK tới users/courses). `00001` không đổi. `db/migrations_test.go` sửa cho 2 migration (version 2, down về 0 gỡ cả `llm_*`).
- `internal/platform/crypto`: AES-256-GCM `0x01‖nonce‖ct‖tag`, AAD, `ParseKey`; `Cipher` in ra `[REDACTED]`.
- Config: `APP_ENCRYPTION_KEY` bắt buộc ở gateway; thiếu / không base64 / ≠ 32 byte → cùng một thông điệp "APP_ENCRYPTION_KEY không hợp lệ: cần 32 byte (base64)", thoát 1. `.env.example`, `docker-compose.local.yml` và 4 test dựng env cập nhật.
- `sqlc.yaml`: `numeric` → `decimal.Decimal`; `internal/store/queries/llm.sql` (nhà cung cấp, mô hình, tuyến, ngân sách, mức dùng bằng `percentile_cont`, `InsertLLMAudit :copyfrom`, tổng chi phí).
- `internal/llmconfig`: `Service` (RBAC từ ctx, version lạc quan, giữ/thay khoá, xoá có kiểm dùng, `audit_log` mỗi thao tác, hạn mức 20/100, `SetRoute` + quy tắc AC10 + `ReindexRequired`, `Get/SetBudget`, `Usage`), `Resolver` (`DecryptKey` là hàm duy nhất trả khoá rõ, `Snapshot`), `Secret` (redact mọi dạng in/JSON/slog). Giá trị `Params` là `json.Number`, tiền là `decimal`.

## AC tự đánh giá
| AC | Lệnh | Kết quả thật |
| --- | --- | --- |
| 1 | `go test ./internal/store -run TestLLMSchema`, `./db` | PASS (5 bảng, mọi cột/kiểu/nullable/mặc định, FK chỉ llm_models→providers, routes→models); `goose status` có `00002_llm.sql`; `git diff origin/main -- db/migrations/00001*` rỗng |
| 2 | `TestLLMConstraints` | PASS 24 ca với SQLSTATE 23514/23505/23502/23503 |
| 3 | `TestLLMIndexes` | PASS: 20.000 dòng, 4 truy vấn dùng chỉ mục, không `Seq Scan`; danh sách chỉ mục `llm_audit` khớp |
| 4 | `sqlc diff` rc=0; grep SQL trong `llmconfig`/`llm` | 0 |
| 5 | `go test -race ./internal/platform/crypto` | PASS (Roundtrip, Nonce1000, Tamper 7 vị trí, AAD, WrongKey, Vector NIST TC16, ParseKey) |
| 6 | `for v in "" abc <16 byte>; APP_ENCRYPTION_KEY=$v gateway serve` | 3 lần rc=1 trong <1 s, thông điệp đúng, grep giá trị = 0 |
| 7–8 | `TestKeyAtRest`, `TestRedacted`, `TestNoKeyInOutputs`; grep `DecryptKey(` ngoài `llm/`,`llmconfig/` | PASS; grep 0 |
| 9 | `TestVersion TestKeyKeepReplace TestDeleteInUse TestAuditLogRows TestLimits TestProviderValidation` | PASS |
| 10 | `TestRouteRules` | PASS 18 ca lỗi + ca biên + ReindexRequired |
| 11 | `TestServiceRBAC` | PASS 5 vai (kể cả không có Principal) × 11 hàm |
| 12 | grep `float(32|64)` ở llmconfig/budget/cost | 0 |

Cổng: `go vet` (cả `-tags testroutes`), `golangci-lint run` (cả `--build-tags testroutes`) 0 issues, `go test -race -count=1 -tags testroutes ./...` toàn bộ ok, `sqlc diff` rc=0.

## Nợ / ghi chú
- `APP_ENCRYPTION_KEY` thành biến bắt buộc của gateway: `.env.local` đang dùng phải thêm dòng này (copy `.env.example`).
- `UsageRow` làm tròn p50/p95 về số nguyên ms ngay trong SQL (không float ở Go).
- Hook `WithOnChange` để US-P1-02 PUBLISH `ep:llm:reload`.
