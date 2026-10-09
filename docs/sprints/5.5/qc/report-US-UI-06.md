# Báo cáo QC — US-UI-06 (Admin → `Panel`)
**Kết luận: PASS có điều kiện** — AC1…AC6 đạt theo số đo của QC; điều kiện: chữ nút chính (`Mời giảng viên` / `Mở lớp`) khác AC (`Mời người dùng` / `Tạo lớp`) — dev giữ chữ đang chạy vì AC6 ghi "chữ không đổi", chưa có góp ý PM → chờ BA (Q-QC-UI06-1); ảnh mốc `settings-llm` kiểm ở UI-07. Bản chấm `d20e1fe` (handoff UI-06), so với trước `1d6caea` (UI-05). QC đo bằng `scripts/qc-measure.spec.ts`: (1) phiên giả `admin` × 7 route (trước + sau); (2) **stack thật** (`pnpm dev` + seed, đăng nhập `admin@`, `teacher@`) 7 route Admin + `/settings/llm` Giảng viên × 1440 / 1024 / 375 — 24 lượt, 24 pass, ảnh `shots/ui06-real/`, số `measure/ui06-real.json`. Bộ e2e của dev ở `d20e1fe`: **153 pass, 0 fail** (`panels`, `account`, `shell`, `settings-llm`, `a11y`, `contrast`).

## Lỗi / lệch
- **L1 (AC1 / AC2 — chữ nút).** Nút chính hiện là **`Mời giảng viên`** (`/admin/users`) và **`Mở lớp`** (`/admin/courses`); AC ghi `Mời người dùng`, `Tạo lớp`. Dev nêu trong handoff (giữ chữ đang chạy vì AC6 "chữ không đổi"). Số nút primary đúng (1). Chờ BA chọn chữ.
- **L2 (TITLE).** `/settings/llm` (Admin và Giảng viên) đo **2**, `/settings` **1**: tiêu đề của `Dialog` đang đóng nằm trong DOM của Panel (cùng gốc L1 của UI-04 / UI-05; Q-QC-UI04-2); dev đã sửa probe của test để bỏ h2 ẩn. Không tính FAIL.
- **L3.** `/admin/courses` đếm 2 nút primary trong DOM: `Mở lớp` + nút "Lưu trữ lớp" của `ConfirmIrreversible` đang đóng (không hiển thị); hiển thị đúng 1.
- **L4 (TOUCH, 375 px).** Chỉ còn liên kết ẩn "Bỏ qua điều hướng" (159 × 40), như các story trước.
- **L5.** `visual.spec` chưa đổi ảnh nào ở story này (đúng handoff: không route nào của 14 ảnh đổi); `lhci` không chạy được ở máy QC (UI-07 / CI).

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| TC-UI6-01 (AC1) | PASS (L1) | `/admin/users` thật: **1** panel (Toolbar tìm kiếm + lọc vai + bảng), 1 nút primary, NEST 0, TITLE 0, STRONG 0, WALL 0, `ox` 0, lề 12 px ở 375; trạng thái bằng `StatusText` (chấm + chữ), thao tác hàng trong menu `…`; `account.spec.ts` 50 / 14 / 0 |
| TC-UI6-02 (AC2) | PASS (L1, L3) | `/admin/courses`: 1 panel, 1 nút primary hiển thị; Admin vào `/exams`, `/questions`, `/class/members` bị chặn (`shell.spec.ts -g 'route access'` pass); không có nội dung học liệu / bài thi / chat của lớp |
| TC-UI6-03 (AC3) | PASS (L2) | `/settings/llm` Admin và Giảng viên: **5** `[data-ep-panel]` ở 1440 / 1024 / 375 (PM #1: Kết nối nhà cung cấp · Mô hình theo tác vụ · Chuỗi dự phòng · Mô hình tìm kiếm tài liệu · Mức dùng và ngân sách), NEST 0, STRONG 0, WALL 0, `ox` 0; khoá API không lộ (dev: DOM không có `sk-` / `api_key`); `settings-llm.spec.ts` 14 / 16 / 0 |
| TC-UI6-04 (AC4) | PASS | Admin `/` 1 panel; `/observability` 2 panel (dải trạng thái gọn, không thẻ số liệu); `/settings/integrations` 3; `/settings` 2 — NEST 0, WALL 0, STRONG 0; `shell.spec.ts -g 'nav per role'` pass (6 mục) |
| TC-UI6-05 (AC5) | PASS có điều kiện | axe có dữ liệu (dev, 10 ca desktop trong `a11y.spec.ts` ở `d20e1fe`): 0 `critical` / `serious`; `contrast.spec.ts -g 'status text'` pass; `axe-allow.json` không đổi; ảnh mốc `settings-llm` sinh lại ở UI-05 (visual 14 / 14 hai lần ở HEAD UI-05, xem báo cáo UI-05); QC tự quét axe bằng bộ riêng: chưa |
| TC-UI6-06 (AC6) | PASS | `nav.ts` không đổi (diff 0); số pass / skip / fail trước → sau (QC chạy cả hai commit): `account` 50 / 14 / 0 → 50 / 14 / 0; `shell` 21 / 23 / 0 → 21 / 23 / 0; `settings-llm` 14 / 16 / 0 → 14 / 16 / 0; `package.json` / lock không đổi; `ui-antipatterns.sh` **22 `✓`**, `--selftest` **22 / 22** |

## Việc sau
- BA: Q-QC-UI06-1 (chữ nút chính Admin).
- QC: quét axe bằng công cụ riêng + ảnh mốc 14 / 14 hai lần + `lhci` ở UI-07.
