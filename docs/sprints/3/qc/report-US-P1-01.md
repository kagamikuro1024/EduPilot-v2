# Báo cáo QC — US-P1-01 (migration `00002_llm`, `platform/crypto`, `llmconfig`)
**Kết luận: FAIL** — 34/35 TC PASS, 1 TC FAIL (TC-P101-19, định dạng `APP_ENCRYPTION_KEY`; lệch spec v1.2). Phần còn lại đạt.

- Bản chấm: commit bàn giao `1b71421` (nhánh `sprint/3-pu-p1`), trong **worktree QC riêng** `../TA_Agent_qcp1` (detached) — không chạm worktree của dev. Tệp dev đang sửa dở (US-P1-03) không nằm trong bản chấm.
- Môi trường QC: Postgres `pgvector:pg18` + Redis + MinIO riêng, gateway `go build -tags testroutes` chạy native :8080/:8081 (không dùng Caddy / compose của dev). Máy: macOS arm64, Go 1.27.1.
- Công cụ: `psql` (docker exec), chương trình Go của QC `scripts/p101-probe` (API công khai `llmconfig`), `node:crypto` giải mã chéo (không có `cryptography` ở Python, QC không cài thêm), `go test` của dev (`-race -tags testroutes -v`, 244 test, 0 FAIL, SKIP chỉ ở provider thật).
- Q-QC: Q-QC-P101-1 (không CLI; QC dùng chương trình Go thay) — SRS v1.2 đã trả lời; Q-QC-P101-2 — SRS v1.2 trả lời và **là căn cứ FAIL TC-19**.

## Lỗi
### BUG-P101-1 — `APP_ENCRYPTION_KEY` chấp nhận chuỗi mà spec v1.2 (US AC6, SRS 5.6) bắt từ chối
Bước tái hiện (`/tmp/qcp1/gw-test serve`, `DATABASE_URL`/`REDIS_URL` trỏ cổng chết để ra đúng thông điệp khoá trước khi chạm DB), khoá 32 byte hợp lệ ở các dạng:

| Dạng khoá | Spec v1.2 | Thực tế |
| --- | --- | --- |
| base64 chuẩn có đệm | chấp nhận | chấp nhận ✓ |
| khoảng trắng / xuống dòng **hai đầu** | chấp nhận (cắt) | chấp nhận ✓ |
| khoảng trắng **ở giữa** | từ chối | từ chối ✓ |
| base64url (có `-` hoặc `_`) | **từ chối** | **chấp nhận** (đi tiếp tới kết nối Redis, không thoát) ✗ |
| không đệm (43 ký tự) | **từ chối** | **chấp nhận** ✗ |
| xuống dòng **ở giữa** | **từ chối** | **chấp nhận** ✗ |
| 16 byte, 33 byte, "abc", rỗng | từ chối | từ chối ✓ (rc=1 trong 0,1 s, đúng câu "APP_ENCRYPTION_KEY không hợp lệ: cần 32 byte (base64)", không in giá trị khoá) |

Ghi chú: spec v1.2 được commit lúc 13:22 ngày 03/10, **sau** bàn giao của dev; dev chưa có cơ hội làm theo. Xử lý cần PM/dev: siết `crypto.ParseKey` (từ chối `-`/`_`, thiếu đệm, `\r`/`\n`/khoảng trắng bên trong) và thêm ca vào `TestParseKey`.

## Kết quả từng TC
| TC | KQ | Bằng chứng (đo thật) |
| --- | --- | --- |
| 01 | PASS | goose `up` → `00002_llm`; `max(version_id)=2` |
| 02 | PASS | đúng 5 bảng `llm_*` |
| 03 | PASS | cột / kiểu / nullable khớp SRS 5.2 từng cột (đối chiếu DDL SRS); khoá chính `uuid default uuidv7()`; không FK tới `users`/`courses`; FK nội bộ chỉ `llm_models→llm_providers`, `llm_task_routes→llm_models` |
| 04 | PASS | `git diff origin/main -- 00001*` rỗng |
| 05 | PASS | `migrate down` gỡ cả `llm_*`; `up` lại version 2 |
| 06 | PASS | `TestLLMSchema` |
| 07 | PASS | 26 `INSERT` sai tự viết (type, base_url, kind, dims, task, trùng `(task,order)`, order âm, budget scope/course, hai budget system, trùng course, `daily>monthly`, giá âm, `status`, `lane`) — mỗi ca bị chặn đúng ràng buộc, 0 dòng lọt |
| 08 | PASS | ca lành (mỗi bảng, 7 task, scope system/course) được nhận; `delete` mô hình đang có tuyến bị FK chặn |
| 09 | PASS | `TestLLMConstraints` |
| 10 | PASS | 11 chỉ mục đúng SRS 5.3 (`llm_audit`: course+created partial, created, task+created, trace; routes `(task,order)` unique; models `(provider,model)` unique; budgets 2 unique từng phần) |
| 11 | PASS | 20.000 dòng + `ANALYZE`; 3 truy vấn `usage` đều `Index Scan` / `Index Only Scan` trên `llm_audit_created_idx`, 0 `Seq Scan` (truy vấn theo `course_id` planner chọn `created_idx` — vẫn không Seq Scan) |
| 12 | PASS | chỉ mục theo lớp của `llm_audit` bắt đầu `course_id` |
| 13 | PASS | `sqlc diff` rc=0 |
| 14 | PASS | grep SQL trong `llmconfig` / `llm` = 0 |
| 15 | PASS | `go test -race ./internal/platform/crypto` (Roundtrip, Nonce1000, Tamper, AAD, WrongKey, Vector) |
| 16 | PASS | bản mã do Go tạo (`01‖nonce12‖ct‖tag16`, dài = 29 + độ dài khoá) giải được bằng **`node:crypto` AES-256-GCM với AAD `llm_providers:<id>`**; bản mã do `node:crypto` tạo giải được bằng `Resolver.DecryptKey` Go → trả `node-made-key-9999` |
| 17 | PASS | sửa nonce / ct / tag → từ chối; sửa **byte phiên bản** thành `0x02` → Go từ chối (`không đọc được khoá`); AAD sai bị từ chối; 1.000 nonce khác nhau: `TestNonceUnique1000` |
| 18 | PASS | `""`, `abc`, 16 byte, 33 byte: rc=1 trong 0,1 s, đúng thông điệp, không in lại giá trị; không mở cổng (xem BUG-P101-1 cho các dạng khác) |
| 19 | **FAIL** | BUG-P101-1 |
| 20 | PASS | canary `sk-LEAK-CANARY-7f3a9c1e`: `api_key_enc` là `bytea`; `position(canary in api_key_enc)`=0; `pg_dump | grep 7f3a9c1e` = 0 |
| 21 | PASS | chép `api_key_enc` của B sang A (AAD sai): A `key_status=unreadable`, B/C/D `ok` |
| 22 | PASS | `TestKeyAtRest` |
| 23 | PASS | `TestNoKeyInOutputs`, `TestRedacted` |
| 24 | PASS | `%v %+v %#v`, `json.Marshal`, `slog` JSON của `Provider` và `ProviderInput` (khoá = canary): không có canary, không có 4 ký tự cuối `9c1e`, có `[REDACTED]`; `DecryptKey(` ngoài `llm/`,`llmconfig/` = 0 |
| 25 | PASS | `TestVersion TestKeyKeepReplace TestDeleteInUse TestAuditLogRows TestLimits` |
| 26 | PASS | mỗi thao tác sửa tăng `audit_log` đúng +1; `before/after/entity_id` không chứa canary (0 dòng) |
| 27 | PASS | sai `version` → `ErrVersionConflict{Current:3}`; không gửi `api_key` → `md5(api_key_enc)` không đổi; `api_key=""` → lỗi "khoá API không được rỗng; bỏ trường này để giữ khoá cũ" và bản mã không đổi; gửi khoá khác/cùng → bản mã đổi (nonce mới); xoá nhà đang dùng → `ErrProviderInUse{[CHAT CLASSIFY UTILITY]}` |
| 28 | PASS | xoá nhà không dùng: mô hình đi cùng, 0 mô hình mồ côi; nhà thứ 21 → "tối đa 20 nhà cung cấp" (QC tự tạo tới 20); trần 100 mô hình: `TestLimits` (QC không tự tạo 101) |
| 29 | PASS | 15 ca sai (chuỗi rỗng, 5 mô hình, sai loại ×2, EMBEDDING 2 mô hình, dims 768, nhà tắt, trùng, `temperature=2.1`, `max_tokens=40000`, `timeout_s=0`, `retries=6`, khoá lạ, task lạ): lỗi đúng quy tắc, 0 dòng `llm_task_routes` đổi; ca biên (0 / 2 / 1 / 32768 / 5 / 300, chuỗi 4 mô hình) được nhận |
| 30 | PASS | đặt `EMBEDDING` lần đầu `ReindexRequired=false`, đổi sang mô hình khác `true` |
| 31 | PASS | `TestRouteRules` |
| 32 | PASS | ma trận 5 danh tính × 8 hàm: ADMIN tất cả; TEACHER chỉ 4 hàm đọc; TA / STUDENT / không Principal: `ErrForbidden` mọi hàm; `actor` lấy từ ctx (không có tham số) |
| 33 | PASS | `price_*`, `daily/monthly_limit` là `numeric`; grep `float32|float64` ở `llmconfig`, `llm/cost` = 0 (`llm/budget` chưa có ở commit bàn giao) |
| 34 | PASS | `go vet` (cả `testroutes`), `golangci-lint` (cả `--build-tags testroutes`) 0 issue, `go test -race -tags testroutes ./...` 22 gói ok |
| 35 | PASS | không `sk-…` thật trong `backend-go` ngoài test |

## Ghi chú (không phải lỗi)
- `migrate down` xoá cả `00001` lẫn `00002` (CLI lùi hết) — TC chỉ cần sạch `llm_*`.
- `llm_audit` truy vấn theo `course_id` không dùng chỉ mục theo lớp ở 20.000 dòng (planner chọn `created_idx`); không Seq Scan nên đạt AC3, đáng đo lại ở tải lớn.
- Tệp đã chạy: `docs/sprints/3/qc/scripts/p101-probe/`, `scripts/p1-run/`.

## Việc sau
Sửa BUG-P101-1 → QC chạy lại TC-19 (và TC-18).
