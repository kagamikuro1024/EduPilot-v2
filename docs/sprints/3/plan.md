# Sprint 3 — PU Nền giao diện + P1 LLM Gateway

Trạng thái: **PM tự duyệt 2026-10-02** (chủ dự án: "chạy cuốn chiếu, lập kế hoạch sprint kế và giao BA trong lúc QC test", giao toàn quyền khi vắng mặt) · Nhánh: `sprint/3-pu-p1`, worktree `../TA_Agent_v2-s3`, **xếp chồng**: tạo từ `sprint/2-pg` rồi gộp `sprint/1.5-mock-ui` (commit `2890554`). Khi 1.5 và 2 vào `main`, PM gộp `main` vào nhánh này trước khi mở PR.

## Mục tiêu
- **PU**: nâng `frontend/src/shared/` của prototype (token, shell, primitive, lớp trạng thái) thành nền thật đủ 8 trạng thái, có lớp dữ liệu nói chuyện với gateway Go (TanStack Query + `apiClient` + `useSSE` trên `/api/v1/events` và `/api/v1/jobs/{id}` của PG), lint chặn giá trị viết cứng, `/dev/ui`, cổng Playwright + axe + Lighthouse.
- **P1**: `internal/llm` (openai-go, registry provider tương thích OpenAI, fallback, `llm_audit`), `internal/llm/scheduler` (3 làn, token bucket trên Redis, cầu dao, hạn mức, trần ngân sách, provider `fake` + replay), migration `00002 llm`, `internal/llmconfig` + AES-GCM, API cấu hình và `/settings/llm` **thật** thay màn mock.
- Cuối sprint: `gate PU`, `gate P1`.

## Story (thứ tự thi công)

| # | Story | Phase/lát | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 1 | US-PU-01 Token + font + chuẩn hoá + lint chặn hex/rgb/oklch/bóng/bo góc/`fetch(` ngoài `shared/` | PU L1, L5 (lint) | S | – |
| 2 | US-PU-02 Primitive đủ 8 trạng thái + `/dev/ui` (375/900/1280/1440, chữ Việt dài) | PU L3 | L | 01 |
| 3 | US-PU-03 Lớp dữ liệu: `apiClient` (định dạng lỗi PG, Idempotency-Key, cursor), TanStack Query, `useSSE` (Last-Event-ID, nối lại), `useAutosaveDraft`, `useUndoableAction`, `<ConfirmIrreversible>`, `<OfflineBanner>` | PU L3 | M | 01, PG |
| 4 | US-PU-04 Shell thật: sidebar 216/72, topbar 56, bottom nav < 720, tablet thu gọn, `EmptyState` cho route chưa có backend | PU L2 | M | 02 |
| 5 | US-PU-05 Cổng tự động: Playwright ảnh mốc 7 route × 2 bề rộng, axe mọi route, LHCI mobile, `ui-antipatterns.sh` đủ phép INTEGRATION §5 | PU L5 | M | 02, 04 |
| 6 | US-P1-01 Migration `00002 llm` (5 bảng dạng cuối) + `platform/crypto` AES-GCM + `internal/llmconfig` CRUD, key chỉ ghi | P1 L3 | M | PG |
| 7 | US-P1-02 `internal/llm`: chat/stream/structured/embed, registry, map lỗi chung, retry + jitter, fallback, `llm_audit` + `trace_id`, embed khoá 1536, provider `fake` + replay, cổng grep SDK | P1 L1, L2 | L | 06 |
| 8 | US-P1-03 Scheduler: 3 làn, token bucket RPM/TPM trên Redis, semaphore, BATCH share, `OVERLOADED` + 503, cầu dao, deadline, suy giảm `degraded`, single-flight, trần ngân sách 80/100% | P1 L1b | L | 07 |
| 9 | US-P1-04 API cấu hình (`/api/v1/admin/llm/...`, `POST …/test`, nạp lại registry) + `openapi.yaml` + contract test | P1 L3 | M | 06–08 |
| 10 | US-P1-05 `/settings/llm` thật trên primitive PU (4 phần, key chỉ ghi, Test từng dòng, bảng usage) thay màn mock | P1 L4 | M | 03, 09 |

Spec: `docs/specs/FEAT-ui-foundation/` (US-PU-01…05) và `docs/specs/FEAT-llm-gateway/` (US-P1-01…05) — BA viết, PM duyệt. TC: `docs/sprints/3/qc/`.

## Quyết định PM tự chốt
- **D53 — giữ CSS Modules + token `--ep-*`**, không đưa Tailwind vào ở PU (PU L1 nhắc Tailwind v4 `@theme`). Lý do: prototype 1.5 đã dựng toàn bộ `shared/` bằng CSS Modules, đạt `ui-antipatterns`; thêm Tailwind là viết lại + hai quy ước song song. Lint (stylelint/ESLint) chặn giá trị viết cứng thay vai trò của "chỉ lớp ngữ nghĩa". Đổi được nếu chủ dự án muốn.
- **PU L4 (dựng lại màn đã có backend)**: tới sprint 3 backend chỉ có nền PG, chưa có auth/chat/threads/documents (P2, P3, P8). Màn thật duy nhất sprint này là `/settings/llm` (US-P1-05). Các màn mock khác giữ dữ liệu mock nhưng **phải chạy trên primitive đã nâng chuẩn** (không vỡ khi đổi primitive); việc thay bằng màn thật đi theo phase sở hữu backend (D51). Ghi vào PROGRESS mục Nợ: "PU L4 /login /register /profile → P2; /chat /threads → P3; /documents /analytics → P8/P10".
- **Phiên đăng nhập thật chưa có (P2)**: API cấu hình LLM dùng JWT + RBAC của PG; dev phát token Admin bằng công cụ của PG (`gateway token`) cho test và cho màn `/settings/llm` ở chế độ dev. Cookie `ep_demo_*` của prototype chỉ còn cho màn mock.
- **`make eval`** (cổng P1) cần RAG + golden set (P3, P10) → sprint 3 chỉ chạy `TestProviderContract` với provider `fake` + **replay**. Ghi âm response thật cần key provider thật → **việc cần chủ dự án**: cung cấp key (đặt ở `.env.local`, không commit). Không có key thì AC ghi âm đánh dấu BLOCKED, phần còn lại vẫn xong.
- Thư viện mới đều có trong bảng `ARCHITECTURE.md`: `@tanstack/react-query`, `@tanstack/react-virtual`, Playwright, `@axe-core/playwright`, Lighthouse CI, `openai-go`. Không thêm gì khác.

## Rủi ro
- Sửa primitive dùng chung làm vỡ 31 màn mock → US-PU-05 chụp ảnh mốc trước khi sửa; `audit.mjs` của QC 1.5 chạy lại sau mỗi story PU.
- Nhánh xếp chồng: lỗi QC 1.5 vòng 2 sửa trên `sprint/1.5-mock-ui` → PM gộp lại vào `sprint/3-pu-p1` sau mỗi vòng.
- Scheduler trên Redis dùng chung nhiều bản gateway: test chạy 2 tiến trình thật, không chỉ goroutine.
