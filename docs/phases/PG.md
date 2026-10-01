# PG — Nền Go

| Ước lượng | Phụ thuộc | Nhánh |
| --- | --- | --- |
| 3,5 tuần | P0 | `feat/pg-go-base` |

**Mục tiêu:** dựng nền tảng Go viết mới cho một **gateway không trạng thái** — khung dịch vụ, tầng dữ liệu, chuẩn HTTP, auth nền, hạ tầng SSE, hạ tầng chạy — đủ để mọi phase tính năng sau chỉ việc lắp vào. **Không có một tính năng nghiệp vụ nào ở phase này.**

Đọc trước: `ARCHITECTURE.md` mục 2 và mục 5 ("Quy ước cho MỌI API"); `SYSTEM_DESIGN.md` mục 2, 3.2, 3.3; luật 10–15 trong `CLAUDE.md`.

**Luật của phase này:** viết mới theo `ARCHITECTURE.md` mục 5 (D45) — không hợp đồng kế thừa, không đối chiếu bản ghi của hệ cũ, không migrate dữ liệu Project III, goose bắt đầu từ `00001`. Gặp việc nghiệp vụ (lớp học, chat, threads, tài liệu, chấm bài…) thì KHÔNG làm ở đây: nó thuộc phase tính năng tương ứng.

Ngoài phạm vi: mọi endpoint nghiệp vụ; vòng đời tài khoản đầy đủ (P2); giao diện (PU); lời gọi LLM (P1).

## Lát việc

**L1. Khung dịch vụ**
- [ ] `backend-go` theo cấu trúc ở `ARCHITECTURE.md` mục 2; Go 1.27 (D48); `cmd/gateway` và `cmd/worker` (worker chỉ có vòng lặp consumer rỗng + healthcheck)
- [ ] `platform/config`: đọc env, giá trị mặc định cho dev, kiểm lúc khởi động — thiếu biến bắt buộc thì chết sớm kèm tên biến
- [ ] `platform/log`: `slog` JSON; `platform/otel`: OpenTelemetry, `trace_id` có mặt trong MỌI dòng log và trong thân lỗi trả về
- [ ] `platform/redis` (Redis 8): client dùng chung, quy ước tiền tố khoá và TTL mặc định
- [ ] Graceful shutdown: ngừng nhận kết nối mới, chờ request đang chạy tới deadline, đóng SSE có sự kiện báo trước
- [ ] Giới hạn ở mọi tầng: timeout đọc/ghi HTTP, kích thước body tối đa, deadline mỗi request đặt trên `context.Context` của request và truyền xuống mọi lời gọi ra ngoài (DB, Redis, HTTP)
- [ ] Pool pgx có `DB_MAX_CONNS`; log truy vấn chậm > 200 ms kèm `trace_id`
- [ ] Makefile: `run`, `test`, `lint`, `sqlc`, `migrate`

**L2. Dữ liệu**
- [ ] goose từ `00001` (D45): CHỈ bảng nền tảng — `users`, `audit_log`, `jobs`, `outbox`; bảng nghiệp vụ do phase sở hữu nó tạo
- [ ] Bật extension `vector` (PostgreSQL 18 + pgvector, D48); `pgvector-go` cho cột vector; quy ước index HNSW và `halfvec` ghi sẵn trong migration mẫu để P1/P8 dùng lại
- [ ] `sqlc.yaml` + `sqlc generate`; enum Postgres → kiểu Go; `sqlc diff` là cổng CI
- [ ] Chạy sạch trên DB trống; `goose down` về `00001` rồi `up` lại vẫn xanh
- [ ] `platform/blob` (minio-go, MinIO trong compose): `Put/Get/PresignGet/Delete` + URL ký sẵn cho cả tải lên và tải xuống; không bao giờ ghi file vào đĩa cục bộ của gateway
- [ ] `platform/outbox`: ghi dòng outbox trong CÙNG transaction với việc nghiệp vụ + consumer `outbox.dispatch` ở worker (retry 3 lần, dead-letter). Chưa có nhà sản xuất nào cũng được — P4 là người dùng đầu tiên

**L3. Chuẩn HTTP** (`ARCHITECTURE.md` mục 5, luật 10–15 `CLAUDE.md`)
- [ ] `httpapi/`: chi; middleware request-id, recover, CORS, rate limit trên Redis
- [ ] **Định dạng lỗi thống nhất** `{code, message, details?, retry_after?}` + bảng mã → status: 401 chưa đăng nhập, 403 sai quyền, 409 xung đột, 422 validation, 429 rate limit, 503 quá tải
- [ ] Helper phân trang con trỏ: `?cursor=&limit=` (mặc định 30, tối đa 100) → `{items, next_cursor}`, sắp theo `(created_at, id)`; cấm OFFSET
- [ ] Middleware `Idempotency-Key`: lưu Redis 24 h, gửi lại cùng khoá trả đúng response cũ, không sinh bản ghi thứ hai
- [ ] Helper khoá lạc quan cột `version` → 409 kèm giá trị hiện tại; helper `ETag` + `If-None-Match`
- [ ] Việc dài: bảng `jobs`, trả `202 {job_id}`, `GET /jobs/{id}`, tiến độ đẩy qua SSE sự kiện `job.progress`
- [ ] Mỗi helper có test riêng: gửi đôi → một bản ghi; con trỏ không bỏ sót và không lặp khi có bản ghi chen vào; ghi sai `version` → 409

**L4. Auth nền** (chỉ phần mọi phase sau cần)
- [ ] `internal/auth`: ký và kiểm JWT (`golang-jwt/jwt` v5), bộ claims chuẩn (`sub`, `role`, `email`, `jti`, hạn) — danh tính của mọi request lấy từ claim, không truy DB mỗi request (D47 mục 6); `x/crypto/bcrypt` cho mật khẩu
- [ ] Middleware RBAC theo role; khung `CourseAccessGuard` (điểm cắm sẵn, P2 nối vào `enrollments`)
- [ ] **Vòng đời tài khoản an toàn đầy đủ — xác minh email, link mời, quên mật khẩu, khoá khi dò, refresh xoay vòng trong cookie `httpOnly` — thuộc P2 theo D36.** PG chỉ để sẵn chỗ cắm, không làm trùng

**L5. Hạ tầng SSE**
- [ ] `httpapi/sse`: mỗi sự kiện có `id`, heartbeat comment mỗi 25 s, đóng sạch khi client ngắt
- [ ] Fan-out qua Redis pub/sub để chạy đúng khi có nhiều bản gateway; `Last-Event-ID` đọc bù phần đã bỏ lỡ
- [ ] Giới hạn: tối đa 2 kết nối SSE mỗi người (đếm trên Redis), thời gian một stream tối đa 120 s rồi client tự nối lại
- [ ] Test: ngắt giữa chừng rồi nối lại bằng `Last-Event-ID` → không mất, không trùng sự kiện; nối vào bản gateway A, sự kiện phát từ bản B → vẫn nhận

**L6. Hợp đồng API**
- [ ] `backend-go/api/openapi.yaml` là nguồn sự thật cho mọi endpoint (D45)
- [ ] `internal/contract`: dựng gateway thật bằng `httptest`, kiểm response khớp `api/openapi.yaml` — status, trường bắt buộc, hình dạng lỗi, hình dạng phân trang. Không phát lại bản ghi của hệ cũ: hợp đồng cũ đã bỏ theo D45
- [ ] Endpoint có trong code mà thiếu mô tả trong `openapi.yaml` (hoặc ngược lại) → test đỏ

**L7. Hạ tầng chạy và cổng mở rộng ngang** (D22)
- [ ] Compose thêm Caddy 2: TLS nội bộ, nén, proxy `/api` → gateway, `/` → Next.js, **tắt buffering trên đường SSE**
- [ ] Compose thêm PgBouncer (transaction mode); pgx cấu hình hợp với transaction mode (tắt prepared statement ngầm)
- [ ] Rà soát không trạng thái (luật 10 `CLAUDE.md`): không biến toàn cục giữ trạng thái người dùng, không ghi đĩa cục bộ — `grep -rn "os.Create\|os.WriteFile" internal/` chỉ còn trong test
- [ ] Dockerfile multi-stage cho gateway và worker, image < 40 MB; CI chạy `go vet`, `golangci-lint`, `go test -race`, `sqlc diff`

## Cổng nghiệm thu
```bash
cd backend-go && go vet ./... && golangci-lint run && go test -race ./...
cd backend-go && sqlc diff                              # sinh lại không khác
cd backend-go && go test ./internal/contract/...        # mọi response khớp api/openapi.yaml
cd backend-go && go test -race ./internal/httpapi/...   # cursor, Idempotency-Key gửi đôi → một bản ghi, 409 version, ETag
cd backend-go && go test -race ./internal/httpapi/sse/... ./internal/auth/...
pnpm -C frontend build
docker compose -f docker-compose.local.yml up -d --scale gateway=2
curl -fsSk https://localhost/api/v1/healthz             # qua Caddy, 200 ở cả hai bản gateway
k6 run benchmarks/load/smoke.js                         # p95 trong SLO ở SYSTEM_DESIGN.md mục 5
```

## Bạn tự kiểm
- Xoá sạch volume DB rồi `pnpm dev`: migration chạy từ `00001` lên hết, không cần thao tác tay.
- Bỏ một biến môi trường bắt buộc: gateway chết ngay lúc khởi động và nói đúng tên biến thiếu, không chết lúc đang phục vụ.
- Mở một stream SSE thử, rút mạng 10 giây rồi cắm lại: trình duyệt tự nối lại bằng `Last-Event-ID`, không mất sự kiện nào.
- Gửi đúp một request có `Idempotency-Key` bằng hai tab: chỉ một bản ghi, response hai lần giống hệt nhau.
- Tắt một trong hai bản gateway giữa lúc đang có stream: phiên kế tiếp vẫn chạy, không mất đăng nhập.
- Đọc toàn bộ diff của `internal/auth` và `httpapi/` — đây là phần mọi phase sau kế thừa.

## Ghi cho luận văn
Lý do chọn kiến trúc không trạng thái ngay từ đầu (D22) và chi phí của nó; bảng các quy ước API dùng chung (phân trang con trỏ, idempotency, khoá lạc quan, outbox, 202 + job) kèm lý do từng cái; số liệu nền của bản Go viết mới (RAM nghỉ, thời gian khởi động, kích thước image, p95 `/healthz`).
