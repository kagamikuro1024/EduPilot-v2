# QC — Cổng nghiệm thu sprint 5.5 (UI panel nổi, FEAT-ui-panels v1.1)
Nguồn: `docs/specs/FEAT-ui-panels/US.md` US-UI-07 AC1–AC9, `SRS.md` 7.4 / 8, `TL-REVIEW.md` "Quyết định PM" (TLR-1…11), `docs/sprints/5.5/plan.md`. QC chạy lại **mọi** lệnh ở máy QC trên **bản build** (không `next dev`), một stack, Playwright ≤ 2 worker; không tin số trong handoff.

**Điều kiện vào cổng:** cả 7 story có `report-US-UI-0N.md` PASS (hoặc PASS có điều kiện PM ghi); CI GitHub xanh ở HEAD `sprint/5.5-ui-panels` (Go + Frontend + `judge-attacks`); D59 đã ghi (`git log -S'D59' -- docs/DECISIONS.md`, kiểm bằng quan hệ tổ tiên — TLR-10); US-UI-01 AC6 có dòng quyết định của chủ dự án / PM.

## A. Kiểm theo US-UI-07 (mỗi dòng một bước đo được)
| # | AC | Lệnh QC chạy (Kiểm của spec) |
| --- | --- | --- |
| G1 | AC1 (ảnh trước / sau các màn chính) | `ls docs/sprints/5.5/handoff/ui-07/before/*.png \| wc -l` → `39` và `…/after/*.png \| wc -l` → `39`; mở bảng handoff → 13 dòng, mỗi dòng đủ ba bề rộng. |
| G2 | AC2 (ảnh mốc `visual.spec.ts` — quy tắc Q-QC-PU06-4) | `docker run --rm --ipc=host -v $PWD:/work -w /work/frontend mcr.microsoft.com/playwright:v1.63.0-noble pnpm exec playwright test visual.spec.ts --workers=2` → `14 passed` (hai lần liên tiếp); `git diff --name-only <BASE55> -- frontend/e2e/visual.spec.ts-snapshots \| sort` ≡ hợp các danh sách ảnh trong `dev-US-UI-02…06.md` (khớp 1 : 1, không ảnh thừa / thiếu). |
| G3 | AC3 (axe — 0 critical / serious trên mọi route) | `$PW a11y.spec.ts` → pass; `jq 'length' frontend/e2e/axe-allow.json` → không lớn hơn trước sprint; `$PW contrast.spec.ts` → mọi cặp chữ × nền ≥ 4,5 (bảng `SRS.md` 5.3 in ra). |
| G4 | AC4 (không còn "nền một màu từ đầu đến cuối" và không panel lồng) | `$PW panels.spec.ts` → pass cho mọi hàng của 7.4 (in số route × vai đã kiểm) và tổng thời gian ≤ 180 s (`--reporter=list`, đọc dòng cuối `passed (…)`); `bash scripts/ui-antipatterns.sh` → 22 `✓`; `bash scripts/ui-antipatterns.sh --selftest \| tail -1` → `22 / 22`. |
| G5 | AC5 (Lighthouse — không kém cuối sprint 5, vẫn `error`) | `cd frontend && pnpm build:gate && pnpm exec lhci autorun; echo rc=$?` → `rc=0`; `pnpm exec lhci autorun --config=lighthouserc.devtools.json; echo rc=$?` → `rc=0`; `git diff <BASE55> -- frontend/lighthouserc.json frontend/lighthouserc.devtools.json .github/workflows/ci.yml \| grep -E 'cpuSlowdownMultiplier\|skipAudits\|onlyAudits\|numberOfRuns\|maxNumericValue'` → không dòng làm ngưỡng lỏng đi; `gh run list --workflow ci.yml --branch sprint/5.5-ui-panels --limit 3 --json headSha,conclusion` → `success` ở `headSha` của HEAD (nếu CI không chạy vì hạ tầng — không phải do mã — thì PM quyết, QC chạy tay hai lệnh trên). |
| G6 | AC6 (không hồi quy chức năng, quyền, lint, build) | `$PW --grep-invert @real` → không ca FAIL, số pass ≥ số đầu sprint; `pnpm -C frontend lint && pnpm -C frontend build; echo rc=$?` → `rc=0`; `git diff <BASE55> -- frontend/package.json pnpm-lock.yaml \| grep -E '^\+ '` → không dòng. |
| G7 | AC7 (phân quyền — không đổi) | `$PW shell.spec.ts -g 'nav per role\|route access'` → pass; QC chạy `audit-login.mjs` (4 vai) + `matrix()` → không đổi so với `docs/sprints/4/qc/report-GATE-P2.md`. |
| G8 | AC8 (hạng mục cấm giữ nguyên) | `$PW panels.spec.ts -g 'wall\|red area'` → pass; `bash scripts/ui-antipatterns.sh` → 22 `✓`; đối chiếu số "nút chính" với `docs/sprints/5/handoff` → không ngoại lệ mới. |
| G9 | AC9 (chủ dự án duyệt và bàn giao) | `test -f docs/sprints/5.5/handoff/dev-US-UI-07.md`; `grep -n 'sprint 5.5' docs/PROGRESS.md \| grep -i 'chủ dự án\\|gật'` → ≥ 1 dòng; `grep -n 'UI-0[1-7]' docs/PROGRESS.md` → có. |

## B. Nhóm kiểm bắt buộc (QC tự đo, độc lập với dev)
| # | Nhóm | Cách đo | Mong đợi |
| --- | --- | --- | --- |
| B1 | **Bốn khẳng định NEST / TITLE / STRONG / WALL** | `playwright-cli eval` trên **mọi route × vai** của SRS 7.4 (trạng thái có dữ liệu) ở 1440 / 1024 / 375; không dùng `panels.spec.ts` của dev | 0 vi phạm; số route × vai đã kiểm in ra khớp bảng 7.4 |
| B2 | **Tương phản** | QC tự tính WCAG 2.x từ màu đã giải (`getComputedStyle`) cho mọi cặp {ink, ink-2, ink-3, red, green, blue} × {canvas, surface, surface-strong, surface-subtle, red-soft}; so với bảng 5.3 | mọi chữ ≥ 4,5 : 1 (amber ≥ 3 cho biểu tượng); `--ep-ink-3` trên canvas ≥ 4,5 (TLR-8) |
| B3 | **Lề mobile (TLR-4)** | `getBoundingClientRect` của panel con của `Section` ở 375 / 390: khoảng cách tới mép màn | đúng **12 px**, không cộng dồn lề `Page`; không tràn ngang (`scrollWidth ≤ innerWidth`) |
| B4 | **Bóng / bán kính đúng một chỗ (TLR-1)** | `grep -rln 'ep-radius-panel\|ep-elevation-1' frontend/src`; `grep -rnE 'filter[[:space:]]*:\|backdrop-filter' frontend/src \| wc -l`; gieo một `box-shadow` lạ ở `features/` rồi chạy `ui-antipatterns.sh` | chỉ `shared/ui/Panel.module.css`; `0`; phép bắt bóng lạ (rồi hoàn tác) |
| B5 | **Đỏ là tín hiệu (TLR-9)** | tính diện tích điểm ảnh có ΔE2000 ≤ 10 so với `--ep-red` trên ảnh 13 màn chính (QC tự viết script, ảnh `playwright-cli`) | < 8 % mỗi màn; không panel nền đỏ |
| B6 | **Ảnh trước / sau (13 màn × 3 bề rộng)** | `playwright-cli` (thời gian đóng băng 2026-10-29T09:20:00+07:00) trên `origin/sprint/5-pe` cuối và HEAD 5.5 | 78 ảnh (39 + 39) vào `docs/sprints/5.5/qc/shots/`; nhận xét thị giác: nền ≠ panel, tiêu đề ngoài panel, không card lồng, không tường KPI |
| B7 | **Hiệu năng (TLR-5)** | `lhci` (mặc định + devtools) thu `resource-summary:script`, LCP, TBT, CLS; so số cuối sprint 5 (`dev-US-PU-06.md`) | LCP ≤ 2.500 ms, TBT ≤ 200 ms, CLS ≤ 0,1; JS mỗi route tăng ≤ 2 KB; cổng vẫn `error` |
| B8 | **Không hồi quy** | Go (`go vet`, `golangci-lint`, `go test -race`), Playwright `--grep-invert "@real"` (≤ 2 worker), `visual.spec.ts` trong image `mcr.microsoft.com/playwright:v1.63.0-noble` hai lần | 0 FAIL; `14 passed` hai lần; số pass ≥ đầu sprint 5.5 |
| B9 | **Phân quyền không đổi** | `audit-login.mjs` bốn vai + `matrix()` | không đổi so với `docs/sprints/4/qc/report-GATE-P2.md` |
| B10 | **Khả năng dùng** | `TOUCH_SRC` 375 px: không phần tử tương tác < 44 px; `AUDIT_SRC`: `{ox:0, cut:0}` ở 1440 / 1024 / 375; bàn phím + focus nhìn thấy trong panel | sạch |

## C. Duyệt thị giác (người)
QC đặt 13 màn × 3 bề rộng cạnh ảnh trước; PM / chủ dự án xác nhận "đã xem, đồng ý" (US-UI-07 AC9 — dòng ghi ở `docs/PROGRESS.md`). QC không tự duyệt hướng thị giác, chỉ báo lệch so với D59 (panel có kỷ luật: canvas đậm hơn panel một bậc, tiêu đề ngoài panel, không card lồng, không tường KPI, đỏ chỉ là tín hiệu).

## D. Điều kiện PASS cổng
Mọi dòng A và B PASS; không còn bug mở (chỉ lệch nhỏ PM chấp nhận); không đỏ nào "giải quyết" bằng nới ngưỡng / sửa ảnh mốc không lý do (TLR-3: ảnh sinh lại ở từng story, handoff liệt kê ảnh + lý do); số `ui-allow:` không tăng; không thêm thư viện.

## E. Dọn dẹp
Gỡ stack QC (`down -v`), `gstart` cổng 3300, Chrome / phiên `playwright-cli`; `docker volume prune -f`; không `pkill node` chung; không đụng cổng 3100; chỉ file QC trong `docs/sprints/5.5/qc/**` được commit.

## Câu hỏi cho BA / PM
- **Q-QC-UI-1** — AC8 / B5: ảnh 13 màn dùng dữ liệu mock cố định (đóng băng thời gian) — chấp nhận làm đại diện cho "dữ liệu thật"? — *chờ PM*.
- **Q-QC-UI-2** — B7: `lhci` tăng hạn mức JS ≤ 2 KB / route so với số cuối sprint 5 (`dev-US-PU-06.md`); nếu dev hạ số cuối sprint 5 vì CI khác máy QC, lấy máy QC làm chuẩn? — *chờ PM*.
