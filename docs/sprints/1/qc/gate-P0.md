# Cổng nghiệm thu P0 — kết luận: PASS
Chạy `/gate P0` theo `.claude/skills/gate/SKILL.md` và mục "Cổng nghiệm thu" của `docs/phases/P0.md`, từng lệnh đúng như viết (không sửa lệnh). Nhánh `sprint/1-p0-prep`, HEAD `origin` = `c69b1fb`. 2026-10-01. Stack đã dừng sau khi chạy (`pnpm dev:down` → 0 container).

## Lệnh của cổng (docs/phases/P0.md)
| # | Lệnh | KQ | Đầu ra |
| --- | --- | --- | --- |
| 1 | `pnpm dev && pnpm dev:status` | PASS | `pnpm dev` exit 0 ("[dev] Sẵn sàng …"); `dev:status` liệt kê 6 container, đều `(healthy)`: `frontend` (3000), `gateway` (8080), `mailpit` `axllent/mailpit:v1.27` (1025, **8025**), `minio` (9000-9001), `postgres` `pgvector/pgvector:pg18` (5433), `redis` `redis:8` (6380) |
| 2 | `curl -fsS localhost:8080/healthz` | PASS | `{"status":"ok"}`, exit 0. Thêm: Mailpit `localhost:8025/` → 200 |
| 3 | `gh run list --branch sprint/1-p0-prep --limit 1` | PASS | `completed success … CI sprint/1-p0-prep push 36880299860` trên HEAD `c69b1fb` (đã chờ run xong; `Frontend success`, `Go success`) |
| 4 | `test -s backend-go/api/openapi.yaml` | PASS | rc=0 |
| 5 | `test -s docs/DEMO_SCRIPT.md` | PASS | rc=0 |
| 6 | `test -d legacy/backend-java && test ! -d backend-java` | PASS | rc=0 |

Không có lệnh nào lệch thực tế (tên nhánh, cổng đều khớp) → không có đề xuất mới vào `proposals.md`.

## Definition of Done chung (SKILL bước 4)
| Mục | KQ | Ghi chú |
| --- | --- | --- |
| Migration chạy sạch trên DB trống | Không áp dụng | P0 không có migration (`backend-go/db/` chưa tạo; US-P0-02 SRS §5) |
| Test cũ không đỏ | PASS | `go vet ./...`, `go test -race ./...` (httpapi, platform ok), `pnpm -C frontend lint` rc=0; CI xanh |
| API mới có trong `openapi.yaml` | PASS | Chỉ có `GET /healthz`, đã có (OpenAPI 3.1, `redocly lint` 0 lỗi — report US-P0-02) |
| STUDENT gọi API giảng viên → 403 | Không áp dụng | Chưa có API nghiệp vụ |
| Seed cho màn mới | Không áp dụng | Chưa có màn nghiệp vụ; `scripts/seed.mjs` ngoài phạm vi P0 |
| Không PII/secret trong log và diff | PASS | TC-36 (diff không khớp mẫu secret), log gateway không giá trị env (TC-25/26), `.env.example` chỉ giá trị dev giả, không dùng lại khoá JWT cũ |

## Luật mở rộng trên diff của phase (4b)
Không `fetch(` trong `frontend/src` (0); không ghi đĩa trong `backend-go` (0); không có `internal/store/queries/` (không OFFSET/danh sách); không lời gọi LLM; không POST nào mới. Không vi phạm.

## Phản mẫu UI và UX (4c, 4d)
- `bash scripts/ui-antipatterns.sh` → 10 dòng `✓`, 0 `✗`. `tokens.css` thêm 1 chú thích `ui-allow` (proposals #8, PM đã chấp nhận).
- Màn mới: chỉ trang `/` trống (lang vi, logo, token) — chưa có hợp đồng ở DESIGN §14 và §22 chưa áp dụng cho khung này. Chưa có Provider "Hôm nay" cần đăng ký (không module sinh việc).
- Cổng UX (axe, Playwright 375 px, Lighthouse CI): chưa cấu hình ở P0 (CI SRS "ngoài phạm vi"); không màn nào cần trạng thái tải/rỗng/lỗi.

## Mức luồng (4e)
P0 là hạ tầng: không luồng F1–F18 nào có "Phase hoàn tất" = P0 nên không có spec E2E cần chạy. Dòng P0 ở `docs/PROGRESS.md` hiện vẫn "Chưa" — **việc của PM/dev cập nhật**, QC không sửa file đó.

## Story đã chấm trong sprint
| Story | KQ | Report |
| --- | --- | --- |
| US-P0-02 | PASS (159 PASS; TC-34/FR-10 đã được PM chốt ở proposals #10, spec cập nhật `c3ff3e5`) | `report-US-P0-02.md` |
| US-P0-03 | PASS (AC4 đã đóng: `ci/red-check` đã xoá, TC-12 PASS) | `report-US-P0-03.md` |
| US-P0-01 | PASS (109 PASS; BUG-1 đã sửa) | `report-US-P0-01.md` |

## Bạn tự kiểm (chủ dự án làm tay — từ `docs/phases/P0.md`)
- Mở `http://localhost:3000`: trang trống đúng font và màu thương hiệu.
- Mở `http://localhost:8025`: thấy Mailpit.
- Đọc `docs/DEMO_SCRIPT.md`: đọc to thử, có vừa 15 phút không.
(`pnpm dev` để dựng lại stack, `pnpm dev:down` để dừng.)

## Ghi cho luận văn
Ghi `docs/thesis-notes/P0.md` (hiện chỉ thấy `docs/thesis-notes/sprint-1.md`, chưa commit): lý do viết mới (D45); quy mô mã Project III đã dời vào `legacy/` (`cloc legacy`) để làm mốc "kế thừa = 0 dòng".

## Kết luận
6/6 lệnh cổng PASS; DoD không FAIL (3 mục không áp dụng vì P0 chưa có mã nghiệp vụ); không lỗi mở. Cổng P0 **PASS**.
