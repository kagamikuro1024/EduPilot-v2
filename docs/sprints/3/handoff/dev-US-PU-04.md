# DEV handoff — US-PU-04 (khung ứng dụng: phiên jwt/demo, điều hướng theo vai, chuông, màn chặn)
Nhánh `sprint/3-pu-p1`. Góp ý #22–#24. Chạy: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test shell.spec.ts` (Next :3310, gateway giả :3312). Ca `no-backend` cần bản `NEXT_PUBLIC_MOCK_SCREENS=0 pnpm build:gate` rồi `NEXT_PUBLIC_MOCK_SCREENS=0 playwright test shell.spec.ts -g no-backend`.

## Làm gì
- `shared/session/jwt.ts` (`checkToken`: chỉ giải mã HS256 + role + `exp`); `session.tsx`: `source: "jwt"|"demo"`, `identity`, `logout`, `expired`; claim thắng cookie `ep_demo_*`, màn mock dùng người/lớp mock cùng vai (Sinh viên → `sv-2`).
- Cổng dán token `TokenGate.tsx` (ô password `autocomplete=off`, "Token không hợp lệ." / "Phiên đã hết hạn"); chỉ vào bundle khi `NEXT_PUBLIC_DEV_AUTH=1` nhờ alias `@ep/token-gate` (#23); build thường: "Cần đăng nhập" không có ô nhập.
- `AppShell`: thứ tự DOM header → sidebar → main; `data-part=sidebar|topbar|bottom-nav`; sidebar 216 (≥1100, `ep:ui:sidebar`) / 72 (720–1099) / không có (<720); nút thu gọn `aria-expanded`; `aria-label` khi thu gọn; đệm dưới `main` ở <720; hồ sơ: ẩn "Đổi vai" ở phiên jwt, `Đăng xuất` xoá token; màn chặn "Bạn không có quyền xem màn này"; `NoBackend` (`MOCK_SCREENS=0`, bảng `mockBackend` ở `nav.ts`).
- `NotificationPopover` (props `items`/`unread`, `aria-live`, chuỗi rỗng theo spec; khối thử ở `/dev/data`); `CommandPalette` combobox + `aria-activedescendant`, "Không thấy mục nào khớp.", sửa bỏ dấu cho chữ Đ hoa; `PageHeader` dùng `<div>` thay `<header>` (để `header h1`=0); vùng chạm 44 px cho `.seeAll a`, `.steps a` của `StaffHome` (AC14).

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `metrics`: sidebar 216 ở 1440/1100, 72 ở 1099/900/720, không có ở 719/375; header cao 56 ở ≥720; `ep:ui:sidebar`=`collapsed` nhớ qua tải lại; ở 375 nút cuối `/threads` nằm trên thanh dưới, `padding-bottom ≥ 56` |
| 2 | `active indicator`: `::before` rộng 2px màu `--ep-red`, nền mục không đỏ, số phần tử nền đỏ ≤ số chấm/huy hiệu, `backdrop-filter:none`, nền không trong suốt |
| 3 | `nav per role`: 5 ngữ cảnh khớp nhãn + `href` + thứ tự (SV 7, SV chưa vào lớp 1, TA 12 không "Hệ thống", GV 15 với 5 nhóm, Admin 6); SV không có chữ "Cấu hình LLM" |
| 4 | `badges`: hậu tố chỉ `inbox`/`grading`, giá trị >0; `grep 'badge: [0-9]' nav.ts`=0 |
| 5 | `mobile` 4 vai × 375/390: ≤5 đích, mỗi đích ≥44 px, `Thêm` đủ (tổng−4) mục, `Esc` đóng và trả focus về `Thêm`, không tràn ngang. **Không làm** "tên trang + 1 hành động ngữ cảnh" trên thanh trên (#22) |
| 6 | `topbar`: `header h1`=0; 4 nhóm (chọn lớp, tìm nhanh, chuông, hồ sơ); nút hồ sơ `aria-label` `Tài khoản: …`; "Đổi vai" có ở demo |
| 7 | `palette`: `Ctrl K`, focus ô nhập, "diem danh" ⇒ "Điểm danh" đứng đầu, `↑↓` đổi `aria-activedescendant`, Tab×10 vẫn trong hộp, `Esc` trả focus, Enter ⇒ `/attendance`; SV: không thấy, "zzzz" ⇒ "Không thấy mục nào khớp.", ô trống = 7 mục |
| 8 | `notifications` (`/dev/data`): chấm theo `unread`, mở khung không xoá chấm, 2 hàng đủ tiêu đề + "Hộp thư hỗ trợ · 2 phút trước", rỗng đúng chuỗi, `aria-live` "Có 1 thông báo mới" một lần (hiển thị lại: 0 thay đổi DOM) |
| 9 | `session`: token ADMIN (cookie GV) ⇒ nav Admin 6 mục, không "Đổi vai", có email claim, `Đăng xuất` ⇒ quay phiên demo; `abc`, `a.b.c`, rỗng, role `SUPERUSER` ⇒ "Token không hợp lệ."; hết hạn ⇒ "Phiên đã hết hạn"; `localStorage`/`sessionStorage` không đổi, token không vào DOM/cookie/URL; JWT STUDENT ⇒ nav Sinh viên + chặn `/settings/llm` + hồ sơ hiện email claim. `pbuild`: `grep -rl 'DEV_AUTH\|Dán token' .next/static`=**0**, `/settings/llm` có "Cần đăng nhập", 0 `type="password"`, `/dev/ui`=404 |
| 10 | `forbidden`: SV và TA ở `/settings/llm`, `/observability`, `/admin/users` và GV ở `/admin/courses` ⇒ "Bạn không có quyền xem màn này" + "Trang này dành cho …" + 1 liên kết `Về Hôm nay`, **0** request `/api/v1/`; GV mở được `/settings/llm`; SV chưa vào lớp ở `/chat` ⇒ "Bạn chưa vào lớp nào" |
| 11 | `no-backend` (bản `MOCK_SCREENS=0`): 4 vai × mọi mục nav trừ `/settings/llm`: `[data-part=empty-no-backend]` đúng phase P2…P10, 1 nút, mục nav còn; `/settings/llm`, `/dev/ui` không bị ảnh hưởng; SV không có từ kỹ thuật |
| 12 | `keyboard`: Tab đầu = "Bỏ qua điều hướng", Enter ⇒ focus trong `main`, 1 `main`, `nav[aria-label]`; thứ tự skip → header → sidebar → main; `Esc` ở menu hồ sơ trả focus |
| 13 | `regression-1.5`: `LEFT page-title` 240 (1440) / 16 (390), brand cao 56, logo 32, mark 28. **`audit.mjs` đầy đủ: QC chạy** |
| 14 | `touch` (thiết bị cảm ứng): `TOUCH_SRC`=`[]` và `ox`=0 ở `/` + 2 đích thanh dưới, 4 vai × 375/390 (sửa 2 liên kết 18–21 px của `StaffHome`) |

## Cổng frontend đã chạy
`eslint .` 0 vấn đề · `tsc` sạch · `ui-antipatterns.sh` 0 ✗ · `lint-selftest.sh` 7/7, 19/19 · `ui-allow:` = 10 · `build:gate` + `playwright test` (2 dự án): **111 passed, 43 skipped** (ca chỉ-desktop ở dự án mobile) · bản thường: `/dev/ui` 404, `state-cell`=0, gate strings=0.

## Nợ / ghi chú
- Ca `no-backend` chỉ chạy ở bản `MOCK_SCREENS=0` (không nằm trong lượt `playwright test` mặc định).
- `/settings/llm` sau khi dán token vẫn là bản mock (`LlmSettings`) cho tới US-P1-05.

## Sửa lỗi QC (report-US-PU-04)
| BUG | Đã sửa | Tự kiểm |
| --- | --- | --- |
| BUG-PU04-1 bảng "Thêm" ở điện thoại chỉ đóng bằng `Esc` / `Đóng` (AC5, TC-21) | Bảng "Thêm" nay là bảng trượt từ đáy (`<Drawer sheet>`): ở < 720 px `dialog` cao theo nội dung (≤ 75 % màn), dính đáy, chừa nền phía trên để **bấm ngoài** đóng; **vuốt xuống** > 64 px từ đầu nội dung đóng; `Esc` / `Đóng` như cũ; cả ba cách trả focus về nút `Thêm`. Kèm: `Dialog` đóng qua `onCancel` + state (không dựa sự kiện `close` đến muộn — làm hộp thoại vừa mở lại bị đóng nhầm) | `shell.spec.ts › BUG-PU04-1` (cảm ứng thật, vuốt bằng CDP `Input.dispatchTouchEvent`): bảng cao < 784 px, dính đáy, số mục = tổng − 4; bấm tại y=20, vuốt 180 px, `Esc` đều đóng và `Thêm` nhận focus |
| Ghi chú PM: ca `shell › touch` cứng cổng 3310 | Cổng Playwright có MỘT nguồn: `e2e/support/env.ts` (`E2E_PORT`, mặc định 3310) dùng cho `playwright.config.ts`, `asDemo` (cookie) và mọi `browser.newContext({ baseURL })`; không còn `localhost:3310` trong spec | `grep -rn 'localhost:3310' frontend/e2e` = 0 |
