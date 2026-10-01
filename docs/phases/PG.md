> **D32 (2026-10-01): phase này viết lại trước sprint 2.** Không còn "port bất biến": gateway Go viết mới theo ARCHITECTURE §5, không golden Java, không xoá Java (Java đã ở `legacy/`). Phần dưới chỉ còn giá trị như danh mục nhóm nghiệp vụ cần có (auth, chat+SSE, threads, documents, analytics, rate limit, stateless). PM viết lại phase này khi lập sprint 2.

# PG — Port gateway từ Spring Boot sang Go

| Ước lượng | Phụ thuộc | Nhánh |
| --- | --- | --- |
| 3 tuần | P0 | `feat/pg-go-port` |

**Mục tiêu:** `backend-go` thay thế hoàn toàn `backend-java` mà frontend và Python không phải sửa một dòng.

Đọc trước: `SYSTEM_DESIGN.md` mục 2, 3.2, 3.3.

**Luật của phase này:** hợp đồng bất biến. Không thêm tính năng, không "cải thiện" JSON, không đổi tên trường. Thấy lỗi của bản Java → giữ nguyên hành vi, ghi vào Nợ.

## Lát việc

**L1. Khung dự án**
- [ ] `backend-go` theo cấu trúc ở `ARCHITECTURE.md` mục 2; `cmd/gateway`, `cmd/worker` (worker rỗng, chỉ healthcheck)
- [ ] `platform/`: config env (giữ tên biến cũ), slog JSON, OpenTelemetry → Jaeger, Redis, graceful shutdown
- [ ] `httpapi/`: chi, middleware request-id, recover, CORS, mã lỗi thống nhất khớp `GlobalExceptionHandler`
- [ ] Makefile: `run`, `test`, `lint`, `sqlc`, `migrate`
- [ ] Giới hạn ở mọi tầng: timeout đọc/ghi HTTP, kích thước body, deadline mỗi request truyền xuống gRPC, thời gian stream tối đa 120 s, tối đa 2 SSE mỗi người
- [ ] Pool pgx có `DB_MAX_CONNS`; log truy vấn chậm > 200 ms kèm `trace_id`

**L2. Database**
- [ ] goose: chép V1–V26 nguyên văn thành `00001`–`00026`; lệnh đánh dấu baseline cho DB đang có; chạy sạch trên DB trống
- [ ] `sqlc.yaml` + queries cho mọi bảng hiện có; enum Postgres → kiểu Go; `pgvector-go` cho cột vector

**L3. Auth + user**
- [ ] Đăng ký, đăng nhập, đổi mật khẩu, profile, preferences
- [ ] JWT cùng secret / claims / thời hạn → token do Java cấp vẫn dùng được; bcrypt đọc hash của Spring
- [ ] Middleware RBAC theo role

**L4. Chat + SSE + gRPC**
- [ ] `aiclient/` sinh từ `shared-proto` (không sửa proto)
- [ ] Chat session, message, feedback; SSE giữ nguyên tên sự kiện và payload
- [ ] Xử lý ngắt kết nối giữa chừng, lỗi 429 từ LLM, fallback (tương đương `AiFallbackIntegrationTest`)
- [ ] Rate limit Redis (tương đương `RateLimitIntegrationTest`)

**L5. Threads + tường lửa PII**
- [ ] Thread, post, tag, Verify/Correct/Reject, similar threads
- [ ] `internal/privacy`: port `PrivacyFirewallFilter` + test tương đương

**L6. Documents + assignments + analytics + internal**
- [ ] Upload, trạng thái, sửa chunk, thống kê; callback nội bộ từ Python (token nội bộ)
- [ ] Assignments; analytics summary, topic difficulty, at-risk, privacy events; intent; health

**L6b. Nền không trạng thái (nội bộ, không đổi hợp đồng API)**
- [ ] `platform/blob` (minio-go): `Put/Get/PresignGet/Delete`; MinIO trong compose; upload tài liệu ghi vào object storage thay cho volume `shared_uploads`; Python đọc file qua blob key (sửa `data_pipeline` tương ứng); script chuyển file cũ
- [ ] Rà soát: không còn biến toàn cục giữ trạng thái người dùng, không ghi đĩa cục bộ (`grep -rn "os.Create\|os.WriteFile" internal/` chỉ còn trong test)
- [ ] Compose thêm Caddy (TLS nội bộ, nén, proxy `/api` → gateway, `/` → Next.js, tắt buffering cho SSE) và PgBouncer (transaction mode)
- [ ] Chứng minh nhân bản được: `docker compose up --scale gateway=2`, chạy contract test + smoke k6 qua Caddy, vẫn xanh

**L7. Contract test + chuyển đổi**
- [ ] `internal/contract`: phát lại golden vào gateway Go, so status + JSON (bỏ qua `ignore.json`)
- [ ] Đổi `Dockerfile`, mọi `docker-compose.*.yml`, `scripts/dev.mjs`, deploy HF sang `backend-go`; image multi-stage < 40 MB
- [ ] CI: bỏ `mvn`, thêm `go vet`, `golangci-lint`, `go test -race`, `sqlc diff`
- [ ] Điền cột Go trong `benchmarks/reports/port.md`
- [ ] Xoá `backend-java` — CHỈ sau khi contract test 100% và bạn đã tự bấm thử

## Cổng nghiệm thu
```bash
cd backend-go && go vet ./... && golangci-lint run && go test -race ./...
cd backend-go && go test ./internal/contract/... -v     # 100% endpoint khớp
pytest -q                                               # Python không đổi, vẫn xanh
pnpm -C frontend build
test ! -d backend-java
docker compose -f docker-compose.local.yml up -d --scale gateway=2 && k6 run benchmarks/load/smoke.js   # p95 không tệ hơn mốc Java
```

## Bạn tự kiểm (không bỏ qua)
- Đăng nhập bằng tài khoản CŨ (hash do Java tạo).
- Chat một câu: chữ chạy dần (stream), có citation; tắt mạng giữa chừng không treo.
- Tạo thread có MSSV → bị chặn như trước.
- Upload một PDF → PROCESSING → READY.
- Đọc toàn bộ diff của `internal/auth` và `internal/privacy`.
- Tắt một trong hai bản gateway khi đang chat: phiên kế tiếp vẫn chạy, không mất đăng nhập.
- Xoá volume uploads cũ rồi mở lại một tài liệu đã upload: vẫn tải được (đã nằm ở object storage).

## Ghi cho luận văn
Bảng Java ↔ Go: RAM nghỉ, khởi động, image, p95; số dòng code; khó khăn khi port SSE và JWT tương thích.
