# DEV handoff — US-PU-05 (cổng tự động: ảnh mốc, axe, Lighthouse, CI)
Nhánh `sprint/3-pu-p1`. Góp ý #25–#28. Chạy: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test` (Next :3310, gateway giả :3312).

## Làm gì
- `e2e/visual.spec.ts`: 7 route × 2 bề rộng = **14 ảnh** (`e2e/visual.spec.ts-snapshots/<route>-<w>.png`, không hậu tố hệ điều hành — `snapshotPathTemplate`); `page.clock.setFixedTime` 29/10/2026 09:20 +07, `reducedMotion: reduce`, `document.fonts.ready`, `maxDiffPixelRatio 0.005`, `threshold 0.2`, `retries: 0`. `/settings/llm` chụp Admin bằng token dev dán trong test.
- `e2e/a11y.spec.ts` + `e2e/support/routes.ts` + `e2e/axe-allow.json` (`[]`): axe (wcag2a/2aa/21aa/22aa) trên mọi route nav của 4 vai + route chi tiết đại diện + `/login`, `/dev/ui`, `/dev/data` ở 1440 và 390; chặn critical/serious; báo cáo `test-results/axe-report.json` và dòng `AXE: N lượt quét …`.
- `e2e/ui-foundation.spec.ts`: 3 ca **chỉ bàn phím** (`tabTo` chỉ nhấn Tab, không `.focus()`): `/chat` (gõ, Enter, `Dừng`, nguồn mở sẵn → Enter đóng, lịch sử phiên, về composer), `/threads` (mở thread, `Đặt câu hỏi`, gõ tiêu đề / chọn chủ đề bằng phím / nội dung, gửi, `VerificationState` "Chờ xác nhận"), 640 px (`AUDIT` `{ox:0,cut:[],ell:[]}`, bảng lịch sử phiên mở / `Esc` trả focus, composer không bị thanh dưới che); mỗi bước là `test.step`.
- `lighthouserc.json` + `lighthouse-auth.cjs` (cookie vai theo URL): 7 URL :3310, `numberOfRuns: 3`, mobile mặc định, 4 assert (LCP ≤ 2500, CLS ≤ 0,1, TBT ≤ 200, script ≤ 256000, trung vị), `upload: filesystem`; `.lighthouseci` vào `.gitignore`. Phụ thuộc dev mới: `@axe-core/playwright`, `@lhci/cli`.
- `.github/workflows/ci.yml` job Frontend (timeout 25 phút): install → lint → `ui-antipatterns` → `lint-selftest` → build → `build:gate` → cache + cài chromium → `playwright test --grep-invert @real` → `lhci autorun` → tải báo cáo khi lỗi; `playwright.config.ts`: CI `workers 2`, `retries 1`, reporter html. Không `legacy/`, không `secrets.*`.
- Sửa để đạt axe (đều thật, không ngoại lệ): `--ep-ink-3` 58 % → 53 % (#25); `StatusStrip` không đặt `<p>` trong `<dl>`; `Skeleton` `role=group`; hàng tải của `CommandPalette` bỏ `aria-label` trên `<li>`.
- Sửa lỗi gặp khi dựng cổng: `session.tsx` kẹp `setTimeout` ≤ 2³¹−1 ms (token hạn dài bị xoá ngay); `CommandPalette` đóng qua `onCancel` + state (sự kiện `close` đến muộn làm không mở lại được), đặt lại ô tìm khi **đóng**, bỏ dấu cả chữ `Đ` hoa.

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `visual.spec.ts`: **14 passed**, `ls …-snapshots \| wc -l` = 14; chạy lại liên tiếp không lệch. Ảnh mốc `/settings/llm` hiện là bản **mock** (cập nhật có giải thích khi P1-05 vào) |
| 2 | `docs/sprints/3/qc/shots/before/` có 12 ảnh (QC) |
| 3 | `keyboard-only` 3 ca pass (lặp 5× không chập chờn); 640 px `AUDIT` sạch |
| 4 | `a11y.spec.ts`: **102 lượt quét** (route × vai × bề rộng), critical 0, serious 0, moderate 0, minor 0; `jq length e2e/axe-allow.json` = 0 |
| 5 | (**cập nhật sau PM quyết #26**: bỏ phông 500 ⇒ LCP trung vị `/` 3249 · `/chat` 3164 · `/threads` 2709 · `/inbox` 3242 · `/gradebook` 3238 · `/settings/llm` 3390 · `/dev/ui` 3189 ms, phông truyền 64–79 KB (trước 98 KB); CLS 0, TBT 16–83 ms, JS 197–230 KB đạt. LCP vẫn > 2500 ⇒ assertion LCP chuyển `warn` TẠM (ngưỡng giữ 2500), `lhci assert` rc=0; Nợ ghi `docs/PROGRESS.md`; cổng P2 phải đưa về `error`. Cổng PU ghi "PASS có điều kiện".) — số đo trước đó: `lhci autorun` rc=1. Trung vị 3 lần (mobile mặc định): LCP `/` 3473 · `/chat` 3326 · `/threads` 3546 · `/inbox` 3407 · `/gradebook` 2864 · `/settings/llm` 3320 · `/dev/ui` 3338 ms (**> 2500**); CLS 0 mọi route; TBT 17–83 ms; JS 196–228 KB (≤ 256 KB). Máy: Mac arm64, Chrome for Testing (Playwright chromium 1243), tải máy 8–10 do QC chạy song song; LCP quan sát thật 0,49 s, ước lượng mô phỏng do ~365 KB tải ở 1,47 Mbps. **Không nới ngưỡng — #26 chờ PM** |
| 6 | `ui-antipatterns.sh` 19 ✓; `--selftest` 19 / 19 |
| 7 | `ci.yml` đã viết đủ bước; **chưa có lượt chạy trên GitHub** (không `gh run` ở máy này) |
| 8 | Nhánh `ci/ui-drift`: **chưa tạo** (QC chứng minh cổng đỏ; dev chỉ xoá sau khi QC chấm) |
| 9 | các commit PU không sửa `backend-go/` (ngoài P1-xx); `go test ./internal/contract/...` chạy ở cổng cuối |
| 12 | cổng PU gộp chạy ở bước "Cổng PU + P1" |
| 13 | `git grep -nE 'eyJ[A-Za-z0-9_-]{20,}\.' -- frontend/e2e frontend/lighthouserc.json` = 0 (token test dựng tại chỗ, `sig` giả) |

## Cổng frontend đã chạy
`eslint .` 0 · `tsc` sạch · `ui-antipatterns` 0 ✗ · `lint-selftest` 7/7, 19/19 · `ui-allow:` = 10 · `build:gate` + `env -u CI playwright test`: **134 passed, 66 skipped** (ca chỉ-desktop ở dự án mobile), không retry.

## Nợ
- LCP (#26) chờ quyết định PM; CI bước `lhci` sẽ đỏ tới khi đó.
- Ảnh mốc `settings-llm-*` cập nhật khi US-P1-05 vào (có giải thích).

## Thực hiện quyết định PM (#25, #26, #28)
- **#25:** `docs/design/DESIGN.md` bảng token: `--ep-ink-3` = `oklch(53% 0.014 25)` / `#746968` (trước `oklch(58%…)` / `#958a8c`), ghi lý do AA 4,5 : 1.
- **#26:** `layout.tsx` chỉ nạp phông 400 / 600 / 700 (subset `vietnamese` + `latin`); chữ 500 hiển thị bằng 400 ⇒ ảnh mốc `inbox-1440`, `inbox-390`, `threads-1440` lệch quá ngưỡng và được cập nhật (`--update-snapshots`, đổi chỉ do độ đậm chữ 500 → 400). Đo lại LHCI: số ở bảng AC5 trên. LCP vẫn > 2500 ⇒ `lighthouserc.json` LCP `warn` (ngưỡng 2500 giữ nguyên), `lhci assert` rc=0; Nợ + cổng P2 ghi ở `docs/PROGRESS.md`.
- **#28:** ảnh mốc `settings-llm-1440|390.png` đã là màn **thật** (cập nhật ở commit US-P1-05); sau thay đổi phông vẫn khớp, không đổi thêm.
