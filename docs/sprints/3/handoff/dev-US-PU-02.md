# DEV handoff — US-PU-02 (primitive đủ trạng thái, `/dev/ui`, CitationList, VerificationState)
Nhánh `sprint/3-pu-p1`. Góp ý #13–#18 ở `docs/sprints/3/proposals.md`. Chạy: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test` (68 ca desktop + mobile, ổn định 3/3 lần).

## Làm gì
- `shared/ui/registry.ts`: ma trận 24 khối × 8 trạng thái (**117** ô áp dụng, **75** ô N/A kèm mã lý do A–H) đúng bảng SRS 7.2; `/dev/ui` (`src/app/dev/ui/{page.dev.tsx,DevUi.tsx,cells.tsx,fixtures.ts}`) vẽ từ registry: `[data-part=primitive][data-name]`, `[data-part=state-cell][data-state]`, ô N/A là `data-na` + `data-reason`; mỗi ô có `data-part=work-region` và đếm lần gọi handler (`data-calls`). Thêm: bề rộng 375/900/1280/1440, `?as=student|ta|teacher|admin`, bảng 1.000 dòng, phân trang con trỏ, khung xương → nội dung, bốn lời `VerificationState`, chuỗi Việt dài.
- `shared/domain/{CitationList,VerificationState}` (+ `index.ts`): nguồn mở tại chỗ (`aria-expanded`, không điều hướng); bốn lời đúng chữ, đường kẻ xanh 1 px không tô nền, nút duyệt chỉ khi `canReview`.
- `DataTable`: `virtual` (ảo hoá bằng `@tanstack/react-virtual`, nạp lười; tiêu đề dính; dòng 48 px; ↑ ↓ Home End Enter Esc; `aria-current`), `pagination` (`onLoadMore(nextCursor)` đúng một lần mỗi con trỏ; khung xương đáy; lỗi trang kế giữ dòng + `Thử lại`), `loading` (n dòng khung xương cao đúng bằng dòng thật). `ActionList` `loading`.
- Nâng chuẩn: `Dialog/Drawer` (bẫy Tab thật, `dismissible`, `loading`, `error`), `ConfirmIrreversible` (`loading` khoá Esc/nền/Đóng; `error` đổi nút thành `Thử lại`; không truyền ⇒ hành vi cũ), `Popover` (`defaultOpen`), `MenuList` (↑ ↓ Home End, `disabled`, `selected`), `Tabs/SegmentedControl/FilterChips` (`disabled`), `Composer` (`error` + `Gửi lại`, `aria-busy`, id theo `useId`, vòng `--ep-focus`), `CommandPalette` (`loading`, `initialQuery`), `Button` nhãn dài xuống dòng, hover cho Checkbox/Switch/Composer, `cursor: not-allowed` cho mọi điều khiển bị khoá, `aria-current` cho dòng đang chọn.
- Cổng dev: `pageExtensions` + `page.dev.tsx`, `app/dev/layout.tsx` + `app/dev/[...slug]` (404), `package.json › build:gate`.
- `e2e/`: `dev-ui.spec.ts` (21 ca), `ui-foundation.spec.ts` (quét 4 vai × mọi route, `primary-allow.json` = `[]`), `support/{audit,fixtures}.ts` (đoạn AUDIT/TOUCH nguyên văn của QC 1.5).

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `matrix`: 24 khối đúng thứ tự registry, **117** ô không N/A, **75** ô `data-na` đều có `data-reason` |
| 2 | `states`: hover đổi thuộc tính tính toán (14 khối, kể cả nền hàng / khung cha); focus bàn phím = `--ep-focus` (hoặc vạch inset của hàng bảng); selected có `aria-*`/`:checked` + dấu không chỉ màu; disabled: `disabled`/`aria-disabled`, `cursor:not-allowed`, `data-calls="0"` sau `click({force})` |
| 3 | `loading/empty/error`: `aria-busy`, không `animate-spin`, ô `empty` đúng 1 hành động, ô `error` `role=alert` + `Thử lại`, không lời `/^[A-Z_]{3,}$/` |
| 4 | `overflow`: AUDIT `{ox:0,cut:[],ell:[]}` ở **375, 640, 900, 1280, 1440**; TOUCH ở 375 = `[]` (đã sửa nút duyệt 36 → 44 px và nhãn nút dài, góp ý #17) |
| 5 | `datatable`: 1.000 dòng, `tbody tr` < 80 sau khi cuộn hết, tới dòng 999 < 1 s, dòng cao ∈ [44,52], `th` sticky, ↑ ↓ Home End, Enter mở Drawer, Esc đóng ⇒ `scrollTop` và dòng chọn giữ nguyên. Đo khung hình: p95 ghi vào annotation `frame-p95-ms` (không chặn) |
| 6 | `cursor`: 2 lần gọi `["c2","c3"]` dù cuộn nhanh 40 lần, dừng khi `null`; lỗi trang 2: dòng trang 1 còn, `role=alert` + `Thử lại` ⇒ gọi lại `c2`, tải được, rồi `c3`. **Nguồn dữ liệu giả trong trang** — góp ý #16 |
| 7 | `overlay`: Tab ×20 không thoát Dialog, Esc đóng, focus về nút mở; Drawer 480 px ở 900 (∈[420,520]) và = innerWidth ở 375; ConfirmIrreversible `loading` không đóng bằng Esc; Dialog lỗi `role=alert`, Drawer `aria-busy`; Popover đóng bằng Esc / bấm ngoài; Menu ↑ ↓ |
| 8 | `composer`: rộng ≤ 820, đúng 1 `[data-variant=primary]`, nút phụ `ghost`, ô nhập trong khung nhìn ở 375×330, ô trống ⇒ gửi khoá, gõ tiếp được khi `loading` |
| 9 | `cls`: đếm từ lúc khung xương xuất hiện, CLS ≤ 0,05; chiều cao vùng lệch khung xương ≤ 8 px (ActionList, DataTable, Section). Bắt được lỗi thật: `next/dynamic` làm bảng ảo nhảy chiều cao ⇒ dùng `React.lazy` + `Suspense` có khung cao đúng |
| 10 | `domain`: "Nguồn tham khảo (2)", mở tại chỗ (URL không đổi, 1 tab), rỗng đúng câu; bốn lời + "Xem câu trả lời AI gốc"; `?as=student` không có nút duyệt, `?as=teacher` có đủ ba; khối đã xác nhận nền `rgba(0,0,0,0)`, viền trái `1px` |
| 11 | grep AC11 = **0**; `ui-antipatterns.sh` sạch (0 ✗), `lint-selftest.sh` 7/7 và 19/19; `ui-allow:` vẫn **10** |
| 12 | `data-part` còn đủ: `page-title brand chat-thread chat-composer chat-history provider-status provider-action bell-dot settings-section nav-badge thread-row thread-form col-qt col-status student-row inbox-reply chart-axis-label chart-point`. **Chưa chạy `audit.mjs` đầy đủ / `visual.spec.ts`** (PU-05 nộp ảnh mốc; audit đầy đủ là lượt của QC như PU-01). Quét thay thế: 4 vai × mọi route nav × 1440 + 390 ⇒ 0 trang tràn ngang |
| 13 | `offline-input`: `context.setOffline(true)` ⇒ Enter ⇒ ô giữ nguyên chữ, `role=alert` "vẫn còn trong ô", `Gửi lại`; bật mạng ⇒ gửi lại được |
| 14 | `one-primary` (/dev/ui) + `ui-foundation › one-primary sweep` 4 vai × mọi route: 0 vùng > 1 primary sau khi đổi "Nhận" ở `/inbox` (góp ý #18); `primary-allow.json` = `[]` (≤ 5). **Chưa quét** hộp thoại / ngăn mở bằng một thao tác ở route thật |
| 15 | Tay (QC): `/dev/ui` ở 1440 và 375 |
| 16 | `rm -rf .next && pnpm build` ⇒ `/dev/ui`, `/dev/data`, `/dev/xyz` = **404**; `grep -rl 'state-cell' .next/static .next/server` = **0**; `build:gate` ⇒ `/dev/ui` 200 (góp ý #14) |

## Nợ / ghi chú
- `/dev/data` (PU-03) sẽ thêm `page.dev.tsx` cùng cổng.
- Một số ô hover/focus/overlay (Dialog, Drawer, CommandPalette, Popover) kiểm bằng ca `overlay` riêng vì cần mở bằng nút.
- Lỗi vặt gặp khi chạy test: `next start` bỏ rơi tiến trình `next-server` sau khi Playwright kết thúc (giữ cổng 3300) ⇒ kiểm `lsof -i :3300` trước khi dựng lại.

## Sửa lỗi QC (report-US-PU-02)
| BUG | Đã sửa | Tự kiểm |
| --- | --- | --- |
| BUG-PU02-1 hàng `DataTable` focus bàn phím không có `--ep-focus` | Bỏ hai quy tắc ghi đè (`.clickable:focus-visible { box-shadow: none }` và vạch inset của ô đầu) trong `DataTable.module.css`; hàng nhận luôn quy tắc chung `:focus-visible { box-shadow: var(--ep-focus) }` của `tokens.css`. Đồng thời giảm 2 dòng `ui-allow:` (10 → 8) | `dev-ui.spec.ts › BUG-PU02-1`: `getComputedStyle(tr).boxShadow` === giá trị tính của `var(--ep-focus)` và ≠ `none`; `ui-antipatterns` 0 ✗ |
| BUG-PU02-2 `Dialog` / `Drawer` / `ConfirmIrreversible` không khoá cuộn nền | Hook `shared/ui/useScrollLock.ts` (đếm số hộp thoại đang mở; `overflow:hidden` ở `<html>` + bù độ rộng thanh cuộn, mở khoá khi hộp cuối đóng) gắn vào `Overlay` (Dialog, Drawer, ConfirmIrreversible) và `CommandPalette` | `dev-ui.spec.ts › BUG-PU02-2`: mở từng hộp, `mouse.wheel(800)` ⇒ `scrollY` không đổi, `<html>` `overflow:hidden`; đóng ⇒ hết khoá và cuộn lại được |
