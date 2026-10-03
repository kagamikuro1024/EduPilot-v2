# DEV handoff — US-P1-01 (DỞ DANG — PM dừng sprint 3 theo chủ dự án)
Nhánh: `sprint/3-pu-p1`. **Trạng thái: chưa xong, chưa nghiệm thu.** Dừng ở điểm build/vet/lint xanh; phần còn lại ghi dưới. Chưa có US-P1-02…05, chưa có PU.

## Đã làm
- `backend-go/db/migrations/00002_llm.sql`: 5 bảng đủ cột/ràng buộc/chỉ mục/trigger `set_updated_at` đúng SRS 5.2–5.3 (có `ponytail:` về phân vùng `llm_audit`). Down gỡ 5 bảng. `00001` không đổi.
- `db/migrations_test.go`: round-trip cập nhật (version 2, down về 0 gỡ cả `llm_*`, status nêu 00002). Test này sửa vì số migration đổi, không nới assertion.
- `internal/platform/crypto`: AES-256-GCM, định dạng `0x01‖nonce‖ct‖tag`, AAD, `ParseKey` (base64 → đúng 32 byte), `Cipher` in ra `[REDACTED]`. Test: `TestRoundtrip`, `TestNonceUnique1000`, `TestTamper` (7 vị trí + cụt), `TestAADMismatch`, `TestWrongKey`, `TestVector` (NIST GCM Test Case 16), `TestParseKey`, `TestCipherRedacted` — PASS `-race`.
- Config: `APP_ENCRYPTION_KEY` bắt buộc ở **gateway** (worker: kiểm nếu có); sai → `APP_ENCRYPTION_KEY không hợp lệ: cần 32 byte (base64)`, không in giá trị. `.env.example` có giá trị dev, compose truyền cho gateway + worker. Các test dựng env gateway (`config_test`, `serve_test`, `contract/rig_test`, `sse/publisher_test`) thêm biến này.
- `sqlc.yaml`: `pg_catalog.numeric` → `decimal.Decimal` / `decimal.NullDecimal`; `internal/store/queries/llm.sql` (nhà cung cấp, mô hình, tuyến, ngân sách; khoá tư vấn theo tác vụ). `sqlc diff` rc=0.
- `internal/llmconfig` (chưa có test): `Secret` (redact mọi dạng in/JSON/slog), lỗi kiểu (`ErrVersionConflict`, `ErrProviderInUse`, `ErrRouteInvalid`, `ErrDimsMismatch`, `ErrInvalid`, `ErrLimit`…), `Service` phần **nhà cung cấp**: `ListProviders/GetProvider/CreateProvider/UpdateProvider/DeleteProvider/RecordTest` — RBAC từ ctx (ghi: ADMIN; đọc: ADMIN+TEACHER), version lạc quan, giữ/thay khoá (nonce mới), `key_status` tính tại chỗ (`unreadable` khi AAD sai), thay danh sách mô hình có chặn mô hình đang dùng, hạn mức 20/100, một dòng `audit_log` mỗi thao tác trong cùng giao dịch (ảnh chụp không chứa khoá), hook `WithOnChange` cho `PUBLISH ep:llm:reload` (US-P1-02).

## Kết quả đã chạy (thật)
`go build ./... && go vet ./...` sạch · `golangci-lint run` và `--build-tags testroutes`: 0 issues · `sqlc diff` rc=0 · `go test -race -tags testroutes` các gói `db`, `platform/*`, `cmd/*`, `contract`, `httpapi/sse`, `store`: PASS (chưa chạy lại toàn bộ `./...`).

## Chưa làm của US-P1-01 (AC chưa đạt)
- AC1 phần test `TestLLMSchema`; AC2 `TestLLMConstraints` (≥14 ca); AC3 `TestLLMIndexes` (20.000 dòng + EXPLAIN) — chưa viết.
- AC6 chạy tay gateway với khoá hỏng (logic config đã có, chưa chạy lệnh `for v in …`).
- AC7–AC9, AC11, AC12: chưa có test `TestKeyAtRest`, `TestNoKeyInOutputs|TestRedacted`, `TestVersion|TestKeyKeepReplace|TestDeleteInUse|TestAuditLogRows|TestLimits`, `TestServiceRBAC` (code có, test chưa).
- `Service` phần **tuyến** (`ListRoutes`, `SetRoute` + quy tắc AC10 + `ReindexRequired`, params bằng `json.Number`, không float) và **ngân sách** (`GetBudget/SetBudget`): chưa viết (SQL đã có trong `llm.sql`).
- `Resolver.DecryptKey(ctx, id)` (hàm duy nhất trả khoá rõ) chưa viết.
- `go.mod`: `go mod tidy` đã chạy; `openai-go` chưa vào vì chưa có mã dùng (US-P1-02).
- Handoff đầy đủ, AC tự đánh giá: viết khi hoàn tất story.

## Nợ / cần hỏi PM
- `APP_ENCRYPTION_KEY` thành biến **bắt buộc** của gateway làm đổi tập "7 biến bắt buộc" của PG (QC `pg01.sh` TC thiếu biến, `.env.local` cũ phải thêm biến) — SRS P1 8.1 yêu cầu; ghi để QC sprint 2/3 biết. `.env.local` hiện có sẽ cần dòng `APP_ENCRYPTION_KEY` (copy từ `.env.example`).
- Chưa động `frontend/` hay `docs/sprints/3/proposals.md`.
