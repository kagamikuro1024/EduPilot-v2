# DEV handoff — US-P0-02
Nhánh `sprint/1-p0-prep` · commit cuối: xem `git log --oneline -8` (các commit `US-P0-02: …`). Đã push.

## Đã làm (theo thứ tự lát dọc)
- Dời mã (một commit riêng `c4b87a9`, 376 file `R100` + 1 `D`): theo SRS 4.1; lịch sử giữ (`git log --follow` = 4 commit).
- Gateway Go (`backend-go/`, Go 1.27.1, chỉ `chi` v5): `cmd/gateway` (cờ `-healthcheck`, SIGTERM/SIGINT → dừng êm mã 0), `internal/platform` (LoadConfig: rỗng sau trim = thiếu, slog JSON), `internal/httpapi` (`GET /healthz`), `api/openapi.yaml` (3.1), `Dockerfile` (distroless nonroot).
- Frontend Next.js 16.3 + React 19 + TS 5.9 (`frontend/`): `lang="vi"`, Be Vietnam Pro qua `next/font/google`, import `tokens.css`, trang `/` có logo; `Dockerfile` (ngữ cảnh = gốc repo, standalone, user `node`).
- Stack: `docker-compose.local.yml` (6 service, mọi service có healthcheck, image ghim tag, volume có tên), `.env.example`, `scripts/dev.mjs`, `package.json` (`dev`, `dev:status/logs/down` cùng `-p edupilot`), `@redocly/cli` vào devDependencies gốc (proposals #3).
- Không có migration/store/seed (story hạ tầng).

## File đổi
`legacy/**` (dời), `seed/documents/*.pdf`, `backend-go/**`, `frontend/{package.json,tsconfig.json,next.config.ts,eslint.config.mjs,.gitignore,Dockerfile,src/app/*}`, `frontend/src/shared/styles/tokens.css` (+1 chú thích `ui-allow`, proposals #8), `.env.example`, `.dockerignore`, `.gitignore`, `docker-compose.local.yml`, `package.json`, `pnpm-workspace.yaml`, `pnpm-lock.yaml`, `scripts/dev.mjs`, `docs/sprints/1/proposals.md` (#7–#9).

## Lệnh QC chạy để kiểm
Đúng các lệnh trong AC1–AC7 của `docs/specs/FEAT-scaffold/US.md` (mỗi shell: `source ~/.zprofile`). Hai chỗ lệch chữ AC → dùng lệnh thay thế:
- AC2: `pnpm --silent dev:status --format '{{.Service}} {{.Health}}'` (không dùng `-s`; proposals #9).
- AC4 SIGTERM: gateway cần cổng 8080 trống — chạy trước `pnpm dev:down`.

## Test đã chạy và kết quả
- `cd backend-go && go vet ./... && go test -race ./... && golangci-lint run` → xanh, 0 issue.
- `pnpm exec redocly lint backend-go/api/openapi.yaml` → valid, 0 lỗi, 2 cảnh báo (thiếu `license`, `/healthz` không có 4XX — cố ý).
- `pnpm -C frontend lint && pnpm -C frontend build` → xanh. `bash scripts/ui-antipatterns.sh` → 0 `✗`.
- `pnpm dev` → exit 0 (≈50 s khi cache), 6 service healthy; `curl :8080/healthz` → `{"status":"ok"}`; `:8025/api/v1/info` → 200; `:3000` có `lang="vi"`; `@font-face "Be Vietnam Pro"` có trong CSS.
- AC4: thiếu env → exit 1, 1 dòng log nêu `DATABASE_URL`, `REDIS_URL`; rỗng/khoảng trắng → exit 1; SIGTERM → exit 0. AC5: `pnpm dev:down` → 0 container. AC6: `$C config | grep -c legacy` = 0. AC1: danh sách gốc khớp, không thiếu `legacy/…`, `492218d5` = 0.

## AC tự đánh giá
AC1 ✓ · AC2 ✓ (lệnh `-s` thay bằng `--silent`; phần "bằng mắt" chủ dự án xem `http://localhost:3000`, tôi chưa mở trình duyệt) · AC3 ✓ · AC4 ✓ · AC5 ✓ · AC6 ✓ · AC7 ✓ (n/a theo spec; body đúng, `.env.example` không secret).

## Nợ / chưa làm / cần hỏi
- Image MinIO: `minio/minio` không còn pull được → dùng `pgsty/minio:RELEASE.2026-08-04T00-00-00Z` (proposals #7). Cần PM xác nhận trước khi đụng object storage thật (P8+).
- `typescript` ghim `^5` vì `typescript-eslint` chưa hỗ trợ TS 7 (lint văng lỗi). `pnpm-workspace.yaml` có `allowBuilds: unrs-resolver: false` để `pnpm install` không lỗi `ERR_PNPM_IGNORED_BUILDS`.
- Phiên bản pnpm (12.8.1) ghim tay trong `frontend/Dockerfile`; chưa có trường `packageManager`.
- Chưa mở trình duyệt xem trang `/` (chỉ kiểm HTML/CSS bằng curl).
- `.github/workflows/keep-huggingface-space-awake.yml` giữ nguyên (Q1). CI = US-P0-03, chưa làm.
