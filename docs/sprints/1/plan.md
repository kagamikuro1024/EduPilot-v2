# Sprint 1 — P0 Chuẩn bị (viết mới toàn bộ, D32)

Trạng thái: **ĐÃ DUYỆT 2026-10-01**, lập lại cùng ngày sau D32 · Nhánh: `sprint/1-p0-prep` (đã push lên `origin`)

## Mục tiêu
Xong trọn P0 trong sprint này: kịch bản demo; mã Project III dời vào `legacy/`; khung chạy được của Go gateway + Python AI + Next.js mới; stack local mới có MailHog; CI xanh. Chạy `/gate P0` cuối sprint.

## Lịch sử
- Bản 1 (sáng 2026-10-01): P0 kiểu port — ghi openapi + golden từ Java. Chủ dự án duyệt.
- Bản 2 (cùng ngày): chủ dự án chốt **viết mới toàn bộ, mã cũ chỉ tham khảo** (D32). Golden/contract Java, mốc k6/RAG của Java bị bỏ. `docs/phases/P0.md` viết lại; `PG.md` đánh dấu chờ viết lại trước sprint 2.

## Story (theo thứ tự thi công)

| # | Story | Feature | Truy vết | Ước lượng | Ai | Trạng thái |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | US-P0-01 | FEAT-demo-script | FLOWS F2, F3, F5, F7, F9, F10, F17 → P0 L0 | S | ba | Spec + `DEMO_SCRIPT.md` xong (commit `5cfb4af`); 9 câu hỏi mở, không chặn sprint 1 |
| 2 | US-P0-02 | FEAT-scaffold | ARCHITECTURE §1–3, SYSTEM_DESIGN §2 → P0 L1 | M (1 ngày) | dev | Chờ spec |
| 3 | US-P0-03 | FEAT-ci | ARCHITECTURE §10 (test) → P0 L2 | S (0,5 ngày) | dev | Chờ spec |

### US-P0-02 — Lập trình viên muốn có mặt bằng sạch và khung chạy được của ba phần mới
- AC tóm tắt:
  - AC1. Mã Project III nằm trong `legacy/` (dời bằng `git mv`, giữ lịch sử); gốc repo không còn `backend-java/`, `src/` cũ, `shared-proto/` cũ, compose cũ. Tài liệu, `.claude/`, `scripts/team-up.sh`, `scripts/ui-antipatterns.sh`, `frontend/src/shared/styles/tokens.css`, `frontend/public/brand/` giữ nguyên chỗ.
  - AC2. `pnpm dev` dựng stack mới: postgres + pgvector, redis, minio, mailhog, gateway, ai, frontend; `pnpm dev:status` thấy mọi container healthy; `curl localhost:8080/healthz` → 200; `localhost:8025` là MailHog; `localhost:3000` trả trang tiếng Việt dùng token + Be Vietnam Pro.
  - AC3. `backend-go/api/openapi.yaml` hợp lệ, mô tả `/healthz`. `go test ./...`, `pytest -q`, `npm run lint && npm run build` (frontend) xanh.
  - AC4 (nhánh lỗi). Thiếu biến env bắt buộc → gateway thoát với thông báo rõ biến nào thiếu, không panic.
- Không chạm: bất kỳ tính năng nghiệp vụ nào; nội dung trong `legacy/`.
- Rủi ro: frontend mới đè lên `frontend/` đang chứa `tokens.css` + brand → dời mã cũ trước, giữ hai thứ đó.

### US-P0-03 — Lập trình viên muốn mỗi lần push đều được kiểm tự động
- AC tóm tắt:
  - AC1. Push lên `origin` → `.github/workflows/ci.yml` chạy Go (`go vet`, `golangci-lint`, `go test -race`), Python (`pytest -q`), frontend (`lint`, `build`, `bash scripts/ui-antipatterns.sh`); `gh run list --branch sprint/1-p0-prep --limit 1` xanh.
  - AC2. CI không đụng `legacy/`, không gọi LLM thật.
  - AC3 (nhánh lỗi). Một test cố ý đỏ trên nhánh tạm → CI đỏ đúng job đó (không `continue-on-error`); nhánh tạm xoá sau khi kiểm.

## Quyết định PM tự chốt
- Nhánh tạo từ `chore/edupilot-v2-docs` (HEAD `1b72f54`), không phải `main`: `main` chưa có bộ tài liệu v2. Thay đổi chưa commit của chủ dự án ở `scripts/team-up.sh` đi theo nhánh, PM không đụng.
- Tên nhánh theo `docs/team/README.md` (`sprint/N-…`).
- Cập nhật tài liệu nền theo D32 (PM làm, chủ dự án cho toàn quyền): `DECISIONS.md` (D32), `CLAUDE.md` (nguyên tắc 6–8, lệnh), `ARCHITECTURE.md` (cấu trúc, contract test, quy ước API), `PRD.md` M0 (bỏ migrate dữ liệu cũ), `phases/P0.md` (viết lại), `phases/PG.md` (đánh dấu), `team/PM.md` (ngoại lệ PG).
- Gateway cổng 8080, frontend 3000, MailHog 1025/8025.
- Package manager frontend mới: pnpm (khớp `CLAUDE.md`); không mang `package-lock.json` cũ sang.
- Phiên bản: Go 1.27 (máy dev), Node 20, Python 3.11.

## Toolchain (PM đã cài)
Homebrew + `node@20 pnpm colima docker docker-compose docker-buildx openjdk@21 gh k6 cloc go python@3.11`. Shell mới: `source ~/.zprofile`. Docker qua `colima start`. `gh` đã đăng nhập `kagamikuro1024`.

## Trả lời của chủ dự án (2026-10-01)
1. Duyệt kế hoạch: có. Toàn quyền máy; `dev` được tự cài công cụ.
2. `docs/DEMO_SCRIPT.md`: `ba` ghi (đề xuất PM, chủ dự án không phản đối).
3. Push lên `origin`: được phép.
4. Viết mới toàn bộ, mã cũ chỉ tham khảo → D32.
