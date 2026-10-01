# QC report — US-P0-02  · Kết luận: PASS
Nhánh `sprint/1-p0-prep`, spec `docs/specs/FEAT-scaffold/` v2, handoff `docs/sprints/1/handoff/dev-US-P0-02.md`. Chạy 2026-10-01 trên macOS (Docker Desktop, Go 1.27.1, Node 24, pnpm 12.8.1).
TC: `docs/sprints/1/qc/tc-US-P0-02.md` (38 TC). Kết quả đầy đủ một lượt: `docs/sprints/1/qc/run-US-P0-02.log` — **159 PASS, 1 FAIL** (TC-34 kiểm FR-10, không thuộc AC; xem BUG-1). 7/7 AC PASS (AC7 "không áp dụng" — ràng buộc thay thế PASS).

## Cổng nghiệm thu đã chạy (lệnh → PASS/FAIL, 10 dòng đầu ra)
| Lệnh | KQ | Ghi chú |
| --- | --- | --- |
| `cd backend-go && go vet ./... && go test -race -count=1 -v ./...` (TC-20) | PASS | 10 test/subtest PASS, 0 FAIL/SKIP; có ca "rỗng coi như thiếu" |
| `pnpm exec redocly lint backend-go/api/openapi.yaml` (TC-21) | PASS | "valid", 2 cảnh báo (thiếu `license`, `/healthz` không có 4XX) — không phải lỗi; AC yêu cầu 0 lỗi |
| `pnpm -C frontend lint && pnpm -C frontend build` (TC-22) | PASS | |
| `bash scripts/ui-antipatterns.sh` (TC-23) | PASS | 10 dòng `✓`, 0 `✗` |
| `pnpm dev` (TC-10) | PASS | exit 0, 6 service healthy, in "Sẵn sàng" |
| `pnpm --silent dev:status --format '{{.Service}} {{.Health}}'` (TC-11) | PASS | frontend gateway mailpit minio postgres redis — đều `healthy` |
| `pnpm dev:down` (TC-28) | PASS | 0 container label `edupilot` sau down |

## AC
| AC | Cách kiểm | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| AC1 | TC-01…08 | PASS | Gốc đúng 18 mục; mọi file Project III ở `legacy/` (R100/C100, nội dung không đổi); `492218d5` = 0; 3 PDF ở `seed/documents` cùng blob với base; `tokens.css` + 3 brand còn; `git log --follow` = 4 commit; commit dời riêng (chỉ rename + 1 `D`), mã mới đến sau |
| AC2 | TC-09…19, 32, 33 | PASS | 6 service, healthcheck đủ, image ghim tag, cổng đúng, gateway phụ thuộc postgres/redis healthy; `/healthz` 200 `application/json` `{"status":"ok"}`; Mailpit/MinIO/Postgres 18 + pgvector/Redis 8 đều trả lời; `lang="vi"`, logo, CSS có `--ep-*` và `Be Vietnam Pro`; không service Python; gateway non-root, healthcheck không curl. **Phần "bằng mắt" (trang `/` đúng logo/font/màu): chủ dự án tự kiểm** — không tính FAIL |
| AC3 | TC-20…24 | PASS | `go vet`/`go test -race` xanh; OpenAPI 3.1 hợp lệ, mô tả `/healthz`, schema `status: string`; `@redocly/cli` trong devDependencies + lockfile (FR-14); lint/build/antipatterns xanh; `layout.tsx` có `Be_Vietnam_Pro`, `tokens.css`, subset vietnamese+latin |
| AC4 | TC-25, 26, 27 | PASS | Thiếu cả hai / một biến / rỗng / khoảng trắng → exit 1, đúng 1 dòng ERROR slog JSON nêu đúng tên biến thiếu, không nêu biến đã có, không log giá trị, không panic. SIGTERM → thoát 0,1 s, mã 0; `/healthz` vẫn ok khi DB/Redis không tồn tại; không ghi file |
| AC5 | TC-28, 29, 30 | PASS | `dev:down` → 0 container; `dev:status`/`dev:logs` thấy đủ 6; gateway dừng ExitCode 0 (không bị kill); `dev:down` lần 2 exit 0; volume còn sau down |
| AC6 | TC-07, 31, 32 | PASS | `$C config \| grep -c legacy` = 0; workspace/package.json/go.mod/Dockerfile/compose sạch; `.dockerignore` có `legacy`; không mount từ `legacy/`; image gateway không COPY legacy |
| AC7 | TC-33, 34 | PASS (n/a có ràng buộc thay thế) | Body chỉ khoá `status`; POST → 405; đường dẫn lạ → 404 không lộ stack; không `sk-…`/`AKIA…`; không dùng lại secret JWT cũ; `.env.local` bị gitignore |

## Lỗi
- **BUG-1 (thấp, lệch spec — không thuộc AC):** `.env.example` có thêm `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` ngoài danh sách FR-10. Cần cho container postgres, giá trị giả. Đề xuất sửa spec, không sửa code: proposals #10. TC-34 sẽ PASS khi PM chốt.

## Kiểm phản mẫu UI / luật mở rộng / phân quyền
- Phản mẫu UI: `ui-antipatterns.sh` sạch. `tokens.css` thêm đúng 1 chú thích `ui-allow` (proposals #8, PM đã chấp nhận); không màu cứng, `fetch` trần ngoài `tokens.css`/`shared` (TC-37). `page.tsx` dùng `style` nội tuyến chỉ với token `--ep-space-*`.
- Luật mở rộng: gateway không ghi đĩa (TC-27, 37), không trạng thái; không `float64`. Lan phạm vi: commit `US-P0-02:` chỉ đụng vùng story, không đụng `CLAUDE.md`, `.claude/`, `team-up.sh`, `ui-antipatterns.sh`, workflow `keep-huggingface…` (TC-35). Không secret trong dòng thêm (TC-36). Không test có sẵn bị sửa.
- Thư viện: `go.mod` chỉ `go-chi/chi/v5` (ARCHITECTURE §3); frontend `next`/`react`/`react-dom` + công cụ lint/TS; `@redocly/cli` (FR-14). Không tailwind/zustand/tanstack (ngoài phạm vi).
- Phân quyền, PII, idempotency, phân trang, migration, 375 px: không áp dụng (không API nghiệp vụ, UI thật, migration).
- Image MinIO là fork `pgsty/minio` (proposals #7, PM chấp nhận cho P0; nợ chốt ở PG).
- TC đã sửa khi chạy (lỗi của TC, không phải của dev; ghi ở "Lịch sử sửa TC"): TC-05, 14, 15, 17, 18, 21, 35. Không nới assertion nào để che lỗi dev.

## Đề nghị
PASS, cho phép merge US-P0-02 về luồng sprint. PM quyết proposals #10 (sửa FR-10 thêm `POSTGRES_*`). Chủ dự án xem trang `http://localhost:3000` cho phần "bằng mắt" của AC2.
