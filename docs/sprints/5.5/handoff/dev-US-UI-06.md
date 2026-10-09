# DEV handoff — US-UI-06 (Admin → Panel)
Nhánh `sprint/5.5-ui-panels`; D59 (a). `nav.ts` không đổi.

## Làm gì
- `/admin/users`: `Toolbar` (tìm kiếm + lọc vai) chuyển vào trong **một** `Panel` cùng bảng; form "Mời giảng viên" (`InvitePanel`) = `Section panel` (hiện tại chỗ, như cũ). `/admin/courses`: bảng trong một `Panel` (bỏ `.tableBlock` margin riêng); form mở / sửa lớp (`CoursePanel`) = `Section panel`. `/` Admin (`AdminToday`): "Việc cần bạn xử lý" một Panel; rỗng / tải = Panel.
- `/settings/llm`, `/observability`, `/settings/integrations`, `/settings`: đã chuyển ở UI-04 / UI-05 (cùng màn dùng chung với Staff / Sinh viên).
- Nút chính: `Mời giảng viên`, `Mở lớp` — **chữ giữ nguyên** (AC1 / AC2 ghi `Mời người dùng`, `Tạo lớp`; AC6 "chữ không đổi" → giữ chữ đang chạy).
- Test: `panels.spec.ts` + `admin users`, `admin courses`, `settings llm`, `admin routes`; `a11y.spec.ts` + hai ca axe **có dữ liệu** (admin: users / courses / llm; giảng viên: bài thi, soạn, kết quả, giống nhau, câu hỏi, thành viên, cài đặt lớp) — ca `axe: <vai>` cũ chạy khi gateway không có nên chỉ thấy trạng thái lỗi của các màn API thật; `contrast.spec.ts` + `status text trên panel`; `support/staffMock.ts` + `mockAdminApi`.
- `ci.yml`: `timeout-minutes` của job Frontend 25 → 40. Lần chạy `26adacc` (UI-05) bị huỷ ở bước `lighthouse ci (devtools)` vì chạm 25 phút (Playwright 10,8 phút + lhci 4 + 8,7); `c4276bc` xong trong 22 phút. Không đổi nội dung bước nào.

## AC
| AC | Kết quả |
| --- | --- |
| 1 | `-g 'admin users'` pass: 1 Panel (Toolbar + bảng), 1 nút primary hiển thị, NEST = TITLE = WALL = 0, STRONG = 0, AUDIT sạch ở 1440 / 1024 / 375. `account.spec.ts` pass. Hộp thoại mời / khoá / đặt lại giữ nguyên |
| 2 | `-g 'admin courses'` pass: 1 Panel, 1 nút primary hiển thị (`Mở lớp`; nút "Lưu trữ lớp" của `ConfirmIrreversible` đóng không tính), không có chữ mã tham gia / bài thi / chat riêng, AUDIT sạch ba bề rộng. `shell.spec.ts -g 'route access'` pass (Admin → `/exams`, `/questions`, `/class/members` bị chặn) |
| 3 | `-g 'settings llm'` pass: **năm** vùng, mỗi vùng h2 ngoài + 1 Panel (số Panel = số h2 hiển thị), STRONG ≤ 3, AUDIT sạch ba bề rộng, DOM không có `sk-…` / `api_key`. **Lệch AC ("hai vùng")** — xem proposal #1: giữ năm vùng vì AC cũng ghi "hành vi và chữ không đổi". `settings-llm.spec.ts` pass |
| 4 | `-g 'admin routes'` pass (7 route Admin × 1440 / 1024: ≥ 1 Panel, TITLE / NEST / WALL = 0, STRONG ≤ 3, không tràn ngang); `/observability`: dải trạng thái trong 1 Panel; `shell.spec.ts -g 'nav per role'` pass (6 mục) |
| 5 | Ảnh `settings-llm-1440 / 390` đã sinh lại ở UI-05 (xem `dev-US-UI-05.md`); UI-06 **không đổi ảnh nào** vì không route nào của 14 ảnh đổi bề mặt (Admin users / courses / `/` không có trong `visual.spec`). axe có dữ liệu: **20 lượt quét, critical 0, serious 0, moderate 0, minor 0**; `axe-allow.json` không đổi (≤ 3). `contrast.spec.ts -g 'status text'`: "Đang dùng" 5,89 : 1; "Chờ nhận lời mời" 18,58 : 1; "Đã khoá" 18,58 : 1 (chữ màu ink, chấm màu) — ≥ 4,5 |
| 6 | `ACCESS` / `EXAM_DYNAMIC` / `nav.ts` không đổi; `account`, `shell`, `settings-llm`: spec không sửa, số ca không đổi. Toàn bộ Playwright: **507 pass / 167 skip / 0 fail** (UI-05: 502 / 162 + 1 ca `/grading` + 4 ca admin); thêm sau lần chạy này: 2 ca axe + 1 ca contrast |

## Gate
`pnpm lint`, `tsc --noEmit` sạch; `ui-antipatterns.sh` 22 ✓ / 0 ✗, `--selftest` 22/22; `lint-selftest.sh` 7/7 + 22/22; không thêm thư viện.

## Chưa chạy / nợ
- `visual.spec.ts` chưa chạy lại cho riêng story này (không route nào đổi); CI chạy trên HEAD.
- Lighthouse chỉ có ở CI (`lighthouse ci` các bước của job Frontend); chưa trích số CLS / LCP / TBT — gom ở UI-07.
- Nhóm proposal #1 (AC3 hai / năm vùng), #2 (hai vùng thiếu dữ liệu), #3 (`FEAT-ui-foundation` 7.2), #4 (ảnh nền cũ) ở `docs/sprints/5.5/proposals.md` chờ PM.
