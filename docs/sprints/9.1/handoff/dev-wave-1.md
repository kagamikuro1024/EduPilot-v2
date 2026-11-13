# Dev — đợt 1 (nền chung) · sprint 9.1 pha 2

Commit: `4b6389f1` (đợt 1 và 2 chồng tệp nên chung một commit).
Stack kiểm: theo `qc/tc-00-chung.md` §môi trường (`-p ep-qc`, `/srv/work/qc`, nhánh `sprint/9.1-uiux`). Ảnh trước/sau 1440 + 375 do QC chụp (K7); đợt này gần như không đổi bố cục.

## Làm được
| Story | Nội dung | Vị trí |
| --- | --- | --- |
| US-02 | Module định dạng ngày giờ duy nhất: `frontend/src/shared/lib/datetime.ts` (`formatWhen`, `formatDay`, `formatTime`, `longDate`, `dateInputValue`, `calendarDays`); luôn ICT; null / chuỗi sai → "—". Bảng quy ước gồm "Ngày mai, 09:00" (P-91-5 Q1). `timeAgo.ts` và `features/today/format.ts` dùng lại. Đã thay mọi `Intl.DateTimeFormat` / `toLocale*` ở 16 tệp. | `e2e/datetime.spec.ts` |
| US-03 | `Button` thêm `disabledReason` (khoá bằng `aria-disabled` + `aria-describedby`, chặn click, tooltip hiện khi rê và khi focus) và `hint` (tooltip cho nút đang dùng được). Áp: Chốt điểm, Xác nhận công thức, Tải bài lên / Chấm tất cả. | `/dev/ui` |
| US-04 | `--ep-slow: 200ms`; `:active` scale 0,97 ở nút; mọi `:hover` bọc `@media (hover: hover) and (pointer: fine)` (69 quy tắc); thanh tiến độ dùng `scaleX` qua `shared/lib/progress.ts` (kẹp 0–100, NaN → 0); thanh bên đổi tức thì; menu `pop` dùng `--ep-fast`. | |
| US-21 | `AuthProvider` không gọi `refreshSession()` ở `/` khi chưa có cờ `ep_returning` (`session/returning.ts` `isReturning()`). | `e2e/landing.spec.ts` |

## `/dev/ui` — ca đặc biệt (P-91-5 Q3)
Mục "Ca đặc biệt" (`section[data-part=edge-cases]`), giữa "Trạng thái" và "Bảng 1.000 dòng":
- `[data-case=reason-not-disabled]`: nút có `disabledReason` nhưng không khoá — lý do bị bỏ qua, không tooltip, bấm được.
- `[data-case=progress]` (`data-value` = 150, −10, NaN, 50): thanh tiến độ giá trị ngoài 0–100.
- Ô nút khoá có lý do: khối Button, trạng thái `disabled`.

## Test đã sửa theo spec (không nới assertion)
- `time-ago.spec.ts`: chuỗi mới ("Hôm qua, 16:40"…) theo US-02 AC2.
- `landing.spec.ts`: "không gọi API nào" ở `/` ẩn danh; thêm "có cờ ⇒ gọi refresh" (US-21 AC1/AC2).
- `support/session.ts` (`asJwt`), `account.spec.ts` (`mockAuth`, "two tabs"): đặt cờ `ep_returning` — phiên đăng nhập thật luôn có cờ (US-21 AC6).
- `dev-ui.spec.ts` "hover đổi ít nhất một thuộc tính": bỏ qua ở project `mobile` (cảm ứng không có hover, US-04 AC2).
- `account.spec.ts` (chuỗi "Lần cuối: …"), `gradebook.spec.ts` / `grade-appeal.spec.ts` ("Đã chốt vN · …" dùng `formatWhen`): regex theo định dạng mới.
- Ảnh mốc `visual.spec`: sinh lại `inbox-1440`, `threads-390` (một lần chung đợt 1 + 2).

## Lệch / nợ
- US-03 AC5 "focus": `ButtonLink` (không có trạng thái khoá) và `Switch` dùng `title` native, không hiện khi focus bàn phím.
- US-04 AC7 (nháy lưu ô): có tiện ích `.ep-saved` + keyframes ở `base.css`, **chưa áp** vào ô sổ điểm / điểm danh / nhận xét — làm ở đợt 4.
- `DESIGN.md` §11 (dialog / drawer ≤ 200 ms): phần chữ do BA sửa; code đã dùng `--ep-slow`.
- `ui-allow:` đang đúng 10 chỗ (trần).

## Sửa sau QC (`qc/wave-1.md`; PM: P-91-7, P-91-8)
- US-21 AC3: `AuthProvider` nhớ đã bỏ qua làm mới ở `/` (chưa có cờ); rời `/` sang route khác (bấm `Đăng nhập`, chuyển trang trong ứng dụng) thì gọi `/auth/refresh` bằng cookie, còn phiên thì vào thẳng ứng dụng. Test: `landing.spec.ts` "mất cờ nhưng còn cookie".
- UX03-8: `Chấm tất cả` khoá với "Chưa có bài nộp." khi đã chọn bài tập và hàng chờ rỗng (`QueueTab.tsx`, `noneToGrade`).
- UX03-13: ở cảm ứng (`hover: none`) lý do hiện DƯỚI nút (`Button.module.css`).
- UX04-16: con trỏ nhấp nháy của chat đổi bằng `opacity` (`ChatScreen.module.css`).
- UX03-18: `dev-ui-*`, `gradebook-*` không đổi hình ở khung 1440 / 390 (ca đặc biệt nằm dưới màn đầu; ảnh mốc chỉ chụp khung nhìn) nên không sinh lại; **`DESIGN.md` chưa có đoạn về lý do nút khoá — việc của BA** (nội dung đề nghị: nút khoá luôn kèm lý do ≤ 1 câu, hiện khi rê / focus / chạm, đặt dưới nút ở cảm ứng; xem `Button.tsx`).
- AC7 nháy lưu ô (UX04-12, 13, 14, 19): PM chốt kiểm ở đợt 4.
- Ảnh mốc mới `hom-nay-gv-1440`, `hom-nay-gv-390` (US-10 AC12) thuộc đợt 3.
