# SRS FEAT-scaffold Dọn mặt bằng + khung Go gateway và Next.js mới
Phiên bản 1 · 2026-10-01 · Trạng thái: DRAFT

## 1. Mục đích và phạm vi
Dời toàn bộ mã Project III vào `legacy/` (D45), dựng khung chạy được của hai phần mới — gateway Go và frontend Next.js 16 — cùng stack local mới (D48), để từ PG trở đi mọi dòng mã nằm trên nền mới. Không có service Python (D46). Không có tính năng nghiệp vụ.

Kết quả quan sát được: `pnpm dev` dựng 6 service healthy; `GET /healthz` trả 200; trang `/` tiếng Việt dùng token + Be Vietnam Pro; gốc repo sạch mã cũ.

## 2. Người dùng và quyền
Không áp dụng — story hạ tầng.

## 3. Luồng chính và nhánh lỗi
Không áp dụng — story hạ tầng. (Nhánh lỗi nằm ở AC4, AC5 của `US.md`.)

## 4. Yêu cầu chức năng

### 4.1 Bảng dời / giữ (mọi mục gốc trong `git ls-files` ở `5cfb4af`)

Cách dời: `git mv <mục> legacy/<mục>` (giữ nguyên đường dẫn bên dưới). Dời trong **một commit riêng**, không lẫn mã mới, để `git log --follow` đọc được.

| # | Mục gốc | Số file | Xử lý | Sau |
| --- | --- | --- | --- | --- |
| 1 | `.claude/` | 4 | Giữ nguyên | `.claude/` |
| 2 | `.dockerignore` | 1 | Giữ, viết lại: thêm `legacy`, bỏ dòng của cấu trúc cũ | `.dockerignore` |
| 3 | `.env.example` | 1 | Dời bản cũ; viết bản mới (FR-10) | `legacy/.env.example` + `.env.example` mới |
| 4 | `.gitattributes` | 1 | Giữ nguyên | `.gitattributes` |
| 5 | `.github/workflows/keep-huggingface-space-awake.yml` | 1 | Giữ nguyên chỗ cho tới khi có trả lời Q1 | `.github/` |
| 6 | `.gitignore` | 1 | Giữ, cập nhật: bỏ dòng Java/Gradle, thêm `frontend/.next/`, `node_modules/`, `backend-go/bin/` | `.gitignore` |
| 7 | `AGENTS.md`, `CLAUDE.md` | 2 | Giữ nguyên | – |
| 8 | `Dockerfile`, `Dockerfile.ai` | 2 | Dời | `legacy/` |
| 9 | `README.md` | 1 | Dời (README mới không thuộc story này) | `legacy/README.md` |
| 10 | `backend-java/` | 156 | Dời | `legacy/backend-java/` |
| 11 | `benchmarks/` | 13 | Dời (script đánh giá của Project III; khung đánh giá mới dựng ở P10 — xem Q2) | `legacy/benchmarks/` |
| 12 | `data/Mordern_Network_Security_Threats.pdf`, `data/QMB12ch6b.pdf`, `data/Quyche.pdf` | 3 | Dời sang seed (D10) | `seed/documents/` |
| 13 | `data/benchmark_failed_cases.jsonl`, `data/benchmark_ground_truth.jsonl`, `data/golden_dataset_quyche.jsonl` | 3 | Dời (bộ dữ liệu đánh giá của Project III — xem Q2) | `legacy/data/` |
| 14 | `data/tmp/492218d5-5f99-4eb4-b18a-06e55141e824.pdf` | 1 | Dời (file tải lên còn sót, chưa rõ nội dung — xem Q3) | `legacy/data/tmp/` |
| 15 | `data_pipeline/` | 15 | Dời | `legacy/data_pipeline/` |
| 16 | `db/` | 28 | Dời (schema cũ; goose mới bắt đầu `00001` ở `backend-go/db/migrations`) | `legacy/db/` |
| 17 | `docker-compose.yml`, `docker-compose.local.yml`, `docker-compose.prod.yml`, `docker-compose.hf.yml` | 4 | Dời; viết `docker-compose.local.yml` mới (FR-8) | `legacy/` + `docker-compose.local.yml` mới |
| 18 | `docs/` | toàn bộ | Giữ nguyên | `docs/` |
| 19 | `frontend/src/shared/styles/tokens.css` | 1 | Giữ nguyên chỗ | – |
| 20 | `frontend/public/brand/` (`favicon.svg`, `logo-edupilot.svg`, `logo-edupilot-mark.svg`) | 3 | Giữ nguyên chỗ | – |
| 21 | Mọi file còn lại trong `frontend/` (cấu hình gốc, `package-lock.json`, `Dockerfile`, `out.css`, `public/` ngoài `brand/`, `src/app`, `src/components`, `src/config`, `src/features`, `src/hooks`, `src/lib`, `src/proxy.ts`, `src/services`, `src/store`, `src/types`) | 104 | Dời **trước** khi tạo frontend mới | `legacy/frontend/` |
| 22 | `huggingface/` | 4 | Dời | `legacy/huggingface/` |
| 23 | `improve_UI.md`, `spec.md` | 2 | Dời | `legacy/` |
| 24 | `package.json` (gốc) | 1 | Giữ, viết lại scripts (FR-9) | `package.json` |
| 25 | `pnpm-workspace.yaml` | 1 | Giữ (`packages: [frontend]`) | `pnpm-workspace.yaml` |
| 26 | `requirements.txt`, `requirements.dev.txt` | 2 | Dời | `legacy/` |
| 27 | `scripts/dev.mjs` | 1 | Giữ chỗ, viết lại (FR-9) | `scripts/dev.mjs` |
| 28 | `scripts/team-up.sh`, `scripts/ui-antipatterns.sh` | 2 | Giữ nguyên (không đụng thay đổi chưa commit của chủ dự án ở `team-up.sh`) | – |
| 29 | `shared-proto/` | 1 | Dời (D46: không gRPC) | `legacy/shared-proto/` |
| 30 | `src/` | 37 | Dời (D46: không có `src/` Python mới) | `legacy/src/` |
| 31 | (mới) `backend-go/`, `seed/`, `pnpm-lock.yaml`, `frontend/` mới | – | Tạo mới | – |

### 4.2 Yêu cầu

| FR | Hệ thống phải… | AC |
| --- | --- | --- |
| FR-1 | Dời mã theo bảng 4.1 bằng `git mv`, trong một commit riêng trước mọi mã mới | AC1 |
| FR-2 | Có module Go duy nhất ở `backend-go/` (`go.mod`, Go 1.27) với `cmd/gateway/main.go`, `internal/platform/` (đọc env, `slog` JSON ra stdout), `internal/httpapi/` (router chi) theo ARCHITECTURE §2; chỉ dùng thư viện trong ARCHITECTURE §3 | AC3 |
| FR-3 | Gateway nghe cổng 8080; `GET /healthz` trả 200, `Content-Type: application/json`, body `{"status":"ok"}`; chỉ là kiểm sống, không gọi DB/Redis | AC2, AC7 |
| FR-4 | Khi khởi động, gateway kiểm biến bắt buộc `DATABASE_URL`, `REDIS_URL` (ARCHITECTURE §8); thiếu bất kỳ biến nào → ghi **một** dòng log lỗi liệt kê mọi biến thiếu, thoát mã 1, không panic. P0 chỉ kiểm có mặt, chưa kết nối | AC4 |
| FR-5 | Gateway dừng êm khi nhận SIGTERM/SIGINT (đóng server có hạn chờ) để `dev:down` không phải giết cứng | AC5 |
| FR-6 | `backend-go/api/openapi.yaml`: OpenAPI 3.1, mô tả `GET /healthz` và schema response | AC3 |
| FR-7 | Test Go table-driven: (a) cấu hình — đủ biến / thiếu một / thiếu cả hai → đúng danh sách biến thiếu; (b) `/healthz` → 200 + body đúng | AC3, AC4 |
| FR-8 | `docker-compose.local.yml` mới, đủ 6 service, mọi service có healthcheck, image ghim tag (D48): `postgres` = `pgvector/pgvector:pg18` (5433→5432), `redis` = `redis:8` (6380→6379), `minio` (9000 API, 9001 console), `mailhog` (1025 SMTP, 8025 UI), `gateway` (build `backend-go/Dockerfile`, 8080), `frontend` (Node 24, 3000). `gateway` phụ thuộc `postgres`, `redis` healthy. Volume có tên cho postgres, redis, minio. Không mount hay build gì trong `legacy/` | AC2, AC6 |
| FR-9 | `package.json` gốc: `dev` = chạy `scripts/dev.mjs` → tạo `.env.local` từ `.env.example` nếu chưa có, kiểm Docker, rồi `docker compose … -p edupilot up -d --build --wait` (thoát 0 khi mọi service healthy, khác 0 nếu có service không healthy); `dev:status` / `dev:logs` / `dev:down` dùng cùng `--env-file .env.local -f docker-compose.local.yml -p edupilot` | AC2, AC5 |
| FR-10 | `.env.example` mới chỉ chứa biến P0 dùng (`DATABASE_URL`, `REDIS_URL`, `BLOB_*`, `SMTP_*`, `MAIL_FROM`) với giá trị dev giả (vd mật khẩu `edupilot-dev`); không secret thật | AC7 |
| FR-11 | `backend-go/Dockerfile` nhiều tầng, chạy bằng user không phải root; healthcheck không cần `curl` trong image (vd gateway có cờ `-healthcheck` tự gọi `http://127.0.0.1:8080/healthz`) | AC2 |
| FR-12 | `frontend/` mới: Next.js 16 App Router + React 19 + TypeScript, quản lý bằng pnpm (lockfile ở gốc workspace), Node 24; `src/app/layout.tsx` đặt `<html lang="vi">`, nạp Be Vietnam Pro qua `next/font/google` (subset `vietnamese`, `latin`), import `src/shared/styles/tokens.css`; `src/app/page.tsx` là trang trống tiếng Việt có logo `public/brand/logo-edupilot.svg`; mọi màu, cỡ chữ, khoảng cách lấy từ token `--ep-*`; ESLint cấu hình cho Next 16 (`pnpm -C frontend lint` chạy `eslint .`) | AC2, AC3 |
| FR-13 | Không file cấu hình nào (workspace, compose, `go.mod`, ESLint, tsconfig) trỏ tới `legacy/`; `.dockerignore` có dòng `legacy` | AC6 |

## 5. Dữ liệu
Không áp dụng — story hạ tầng (không migration; `backend-go/db/migrations/` chưa tạo).

## 6. API
| Endpoint | Quyền | Request | Response | Lỗi |
| --- | --- | --- | --- | --- |
| `GET /healthz` | Công khai (healthcheck) | – | 200 `application/json` `{"status":"ok"}` | Không có lỗi nghiệp vụ; gateway không lên thì kết nối bị từ chối |

Không gRPC (D46). Không tool agent.

Cổng dev cố định: gateway 8080, frontend 3000, MailHog 1025 / 8025, MinIO 9000 / 9001, Postgres 5433, Redis 6380 (lệch cổng mặc định để không đụng dịch vụ sẵn có trên máy dev).

## 7. Giao diện
Không áp dụng — story hạ tầng. (Trang `/` chỉ là trang trống kiểm font + token; khung giao diện thật dựng ở PU.)

## 8. Phi chức năng áp dụng
| Nhóm | Yêu cầu | Nguồn |
| --- | --- | --- |
| Không trạng thái | Gateway không ghi đĩa cục bộ, không giữ trạng thái người dùng trong bộ nhớ | CLAUDE.md luật 10 |
| Phiên bản | Go 1.27, Node 24 LTS, PostgreSQL 18 + pgvector, Redis 8, Next.js 16 + React 19, pnpm; ghim bản vá trong `go.mod`, `pnpm-lock.yaml`, tag image | D48 |
| Bí mật | Không secret thật trong repo; `.env.local` nằm trong `.gitignore` | CLAUDE.md "Cấm tuyệt đối" |
| Log | `slog` JSON; không log giá trị biến env (chỉ tên biến thiếu) | CLAUDE.md "Quy ước code" |
| Hạ tầng | Chỉ thành phần có trong SYSTEM_DESIGN §2 / ARCHITECTURE §3 | CLAUDE.md "Cấm tuyệt đối" |

Rủi ro đã biết:
- Máy dev đang có `node@20` (plan sprint 1, Toolchain); dev cài Node 24 trước khi làm.
- Image `mailhog/mailhog` chỉ có bản amd64; trên máy arm64 (colima) cần `platform: linux/amd64`. Nếu không chạy được thì DỪNG và báo PM, không tự đổi sang công cụ khác.
- Frontend mới tạo trong `frontend/` đang chứa `tokens.css` + `brand/`: dời mã cũ trước (FR-1), rồi mới chạy trình tạo dự án; không để trình tạo dự án ghi đè hai thứ này.
- `.env.example` cũ chứa khoá JWT mẫu; sau khi dời vẫn nằm trong `legacy/` và lịch sử git. Không dùng lại khoá này cho stack mới.

## 9. Kiểm thử
| Tầng | Nội dung | Lệnh |
| --- | --- | --- |
| Unit Go | FR-7 (a), (b) | `cd backend-go && go test -race ./...` |
| Tĩnh | `go vet`, OpenAPI lint, ESLint, phản mẫu UI | AC3 trong `US.md` |
| Khói stack | `pnpm dev` → healthy → curl 3 cổng → `dev:down` | AC2, AC5 |
| Nhánh lỗi | Chạy gateway thiếu env | AC4 |

Không E2E Playwright (chưa có luồng). Không seed.

## 10. Câu hỏi mở và quyết định đã chốt
Không áp dụng — story hạ tầng. Câu hỏi: `QUESTIONS.md`. Chi tiết kỹ thuật BA tự chốt theo tài liệu sẵn có: biến bắt buộc ở P0 là `DATABASE_URL`, `REDIS_URL` (ARCHITECTURE §8); `/healthz` chỉ kiểm sống; tên project compose `edupilot`; `pnpm dev` chạy nền và chờ healthy để cổng nghiệm thu P0 (`pnpm dev && pnpm dev:status`) chạy được; `benchmarks/` cũ vào `legacy/` (D46 chỉ giữ Python cho script đánh giá **mới** ở P10).

## 11. Truy vết
| PRD | FLOWS | Phase | US | FR | Kiểm |
| --- | --- | --- | --- | --- | --- |
| §5 Triển khai (`pnpm dev`, MailHog) | – | P0 L1 | US-P0-02 AC1 | FR-1 | lệnh AC1 |
| §5 | – | P0 L1 | AC2 | FR-3, FR-8, FR-9, FR-11, FR-12 | `pnpm dev`, curl |
| – | – | P0 L1 | AC3 | FR-2, FR-6, FR-7, FR-12 | `go test`, redocly lint, `pnpm lint/build` |
| – | – | P0 L1 | AC4 | FR-4, FR-7 | chạy gateway thiếu env |
| – | – | P0 L1 | AC5 | FR-5, FR-9 | `pnpm dev:down` |
| D45 | – | P0 L1 | AC6 | FR-8, FR-13 | grep `legacy` |
| – | – | P0 L1 | AC7 | FR-3, FR-10 | curl body, grep secret |
