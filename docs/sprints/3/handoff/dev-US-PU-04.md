# DEV handoff — US-PU-04 (khung ứng dụng) — **DỞ DANG, chưa bàn giao cho QC**
Nhánh `sprint/3-pu-p1`. Dừng giữa chừng vì chủ dự án chuyển chỗ. Mã đã commit nhưng story CHƯA xong: còn 1 lỗi đã biết + chưa chạy cổng đầy đủ.

## Đã làm (có trong commit)
- `shared/session/jwt.ts` (`checkToken`: giải mã HS256, role ∈ ADMIN/TEACHER/TA/STUDENT, `exp`; không xác minh chữ ký); `session.tsx` có hai nguồn `source: "jwt" | "demo"`, `identity`, `logout`, `expired` (claim thắng cookie `ep_demo_*`; màn mock dùng người mock cùng vai/lớp mock đầu; `auth:expired` + hết hạn giữa chừng ⇒ về demo).
- `shared/session/TokenGate.tsx` (lazy, chỉ khi `NEXT_PUBLIC_DEV_AUTH=1`; ô password `autocomplete=off`; "Token không hợp lệ." / "Phiên đã hết hạn"); build thường: "Cần đăng nhập".
- `AppShell`: DOM header → sidebar → main; `data-part=sidebar|topbar|bottom-nav`; 216 (≥1100, nhớ `ep:ui:sidebar`) / 72 (720–1099) / không sidebar (<720); `aria-expanded` ở nút thu gọn; `aria-label` khi thu gọn; padding dưới `main` ở <720; menu hồ sơ ẩn "Đổi vai" ở phiên jwt; "Đăng xuất" xoá token; màn chặn "Bạn không có quyền xem màn này"; `NoBackend` (`MOCK_SCREENS=0`, bảng `mockBackend` ở `nav.ts`).
- `NotificationPopover` (props `items`, `unread`, vùng `aria-live`, chuỗi rỗng theo spec) + khối thử ở `/dev/data`.
- `CommandPalette`: combobox + `aria-activedescendant`, "Không thấy mục nào khớp.", sửa bỏ dấu cho chữ Đ hoa (lỗi cũ: `Điểm danh` không khớp "diem danh").
- `PageHeader` dùng `<div>` thay `<header>` (để `header h1` = 0 theo AC6); vùng chạm 44 px cho `.seeAll a` và `.steps a` của StaffHome (AC14).
- `build:gate` thêm `NEXT_PUBLIC_DEV_AUTH=1`; `e2e/shell.spec.ts` (16 ca).

## Đã chạy (thật)
- `playwright test shell.spec.ts --project=desktop --workers=1` trên `build:gate`: **15 passed, 1 skipped** (`no-backend` chỉ chạy ở bản `NEXT_PUBLIC_MOCK_SCREENS=0`).
- Bản dựng `NEXT_PUBLIC_MOCK_SCREENS=0 build:gate`: `shell.spec.ts -g no-backend` **1 passed**.
- `tsc --noEmit` sạch; `eslint src` sạch (trước khi thêm `shell.spec.ts` — chưa chạy lại `eslint .`).

## LỖI ĐÃ BIẾT — chặn bàn giao
- **AC9/AC23: bản dựng thường (`pnpm build`) VẪN chứa mã cổng token**: `grep -rl 'DEV_AUTH\|Dán token' .next/static` = **2**, `grep -rl token-gate .next/static .next/server` = **3** (kỳ vọng 0). `const TokenGate = process.env.NEXT_PUBLIC_DEV_AUTH === "1" ? lazy(() => import(...)) : null` không loại được chunk. Hướng sửa (như PU-02): tách bằng `pageExtensions`/tệp chỉ có ở bản dev, hoặc đặt cổng sau một route `*.dev.tsx`; rồi chạy lại `pbuild; grep`, và `curl -b ep_demo_role=admin :3310/settings/llm` không có `type="password"` (lần kiểm này chưa xác nhận được vì máy chủ chưa lên kịp).

## Chưa làm / chưa chạy
- Toàn bộ Playwright (`playwright test`, 2 dự án) sau thay đổi này; `ui-antipatterns.sh`, `lint-selftest.sh`, `eslint .`, đếm `ui-allow:` (≤ 10) — chưa chạy.
- AC5 "tên trang (h1) + đúng một hành động ngữ cảnh" trên thanh trên điện thoại: KHÔNG làm (giữ thanh trên 1.5: logo mark, bộ chọn lớp, tìm, chuông, hồ sơ; h1 nằm trong trang). Cần PM/BA xác nhận — chưa ghi `proposals.md`.
- `audit.mjs` đầy đủ (AC13) để QC chạy; chỉ đo `LEFT`/brand/logo/mark ở `shell.spec.ts › regression-1.5` (pass).
- Còn US-PU-05, US-P1-05, cổng PU + P1: chưa bắt đầu.
