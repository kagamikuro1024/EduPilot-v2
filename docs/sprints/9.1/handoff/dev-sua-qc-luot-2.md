# Dev — sửa FAIL của QC lượt 2 (đợt 1, 2, 3, 6) · sprint 9.1

Commit: `9.1 sửa QC lượt 2` trên `sprint/9.1-uiux`. Căn cứ: `qc/wave-1.md`, `wave-2.md`, `wave-3.md`, `wave-6.md` (kiểm ở `6a0e8584`) và P-91-9. Gate chạy trên VPS (xem cuối).

| Story · TC | Nguyên nhân | Sửa | Vị trí |
| --- | --- | --- | --- |
| US-06 AC3 (UX06-7, P-91-9) | `session.tsx` giữ lớp `ARCHIVED` làm lớp nhớ và lớp mặc định | Thêm `live` = lớp `ACTIVE`. Lớp nhớ chỉ hợp lệ khi nằm trong `live` (hoặc `all`); giá trị cũ trỏ lớp lưu trữ bị xoá khỏi `ep:ui:course`; `setCourse` chỉ ghi nhớ lớp `ACTIVE`. Lớp lưu trữ vẫn mở xem tạm trong phiên từ bộ chọn (không ghi nhớ) | `shared/session/session.tsx`; test `class-join.spec.ts` (ca lớp lưu trữ đổi theo AC3: chọn tay không còn «tải lại vẫn giữ») |
| US-01 AC1 #3 (UX01-4) | Nút `Mở lớp` chưa có chú thích | `hint` của `Button` (tooltip, `aria-describedby`): «Người được gán nhận thông báo kèm mã tham gia.» (10 từ), chỉ ở chế độ mở lớp | `features/admin/courses/CoursePanel.tsx` |
| US-01 AC5 #34 (UX01-11) | `Switch` chỉ dùng `title` native (không hiện khi focus bàn phím, không `aria-describedby`) | `Switch` dùng chung nay vẽ tooltip thật (`role="tooltip"`, `aria-describedby`, hiện khi rê / focus / chạm, chỉ `opacity`); hưởng lợi cả công tắc «Cho nộp muộn» | `shared/ui/Field.tsx`, `Field.module.css` |
| US-10 AC3 (UX10-10) | Chữ máy chủ mang mã lớp ở mọi hàng («· 761988», «lớp 761988», «vào lớp 761988 đang chờ…»); harness không có mã nên ca cũ không bắt được | `withoutClass(title, course)` bỏ ba dạng đó khi **một lớp** đang chọn; «Tất cả lớp của tôi» giữ nguyên chữ và hiện `classLabel` ở `meta`. Không đổi API / Go | `features/today/format.ts`, `TodayActions.tsx`; `today.spec.ts` dùng tiêu đề kiểu máy chủ (có mã lớp) rồi kiểm không còn «761988» |
| US-10 suite có sẵn | 4 ca `panels.spec.ts:419` đòi 2 Panel («Việc cần xử lý» + «Sắp tới») | Theo AC3 (bỏ khối trùng tiêu đề trang): kỳ vọng 1 Panel, `h2 = ["Sắp tới"]`, đổi tên ca | `e2e/panels.spec.ts` |
| US-08 AC5 (UX08-16…18) | Chỉ `pause()` khi đóng; `currentSrc` còn mp4 (Chromium không xoá `currentSrc` khi gỡ `<source>` + `load()`) | Đóng hộp thoại: dừng rồi thay phần tử `<video>` bằng bản sao rỗng nguồn (huỷ luôn việc tải); mở lại nạp lại `<source>` từ bản mẫu | `features/landing/LandingMedia.tsx`; `landing.spec.ts` kiểm 0 `<source>`, `currentSrc` rỗng, mở lại có 2 nguồn |
| US-09 AC10 (UX09-16) | `run.sh` truyền `-e EP_BASE_URL=""`; `record.mjs` dùng `??` nên `BASE = ""` | `||` thay `??`; README ghi rõ stack thử phải trỏ cổng Caddy (có `/api`), không phải cổng frontend (`17000`) | `scripts/landing-video/record.mjs`, `README.md` |
| US-03 AC8 (UX03-18, P-91-9) | Ảnh mốc tĩnh không chụp được lý do khi rê / focus | Ca E2E trên `/dev/ui`: khối `Button`, ô `disabled`, nút «Lưu thay đổi»: `aria-disabled`, `aria-describedby`, lý do ẩn → hiện khi focus (Enter / Space không kích hoạt) → hiện khi rê (desktop) hoặc chạm (mobile), `data-calls = 0`; ca «Ca đặc biệt» (`reason-not-disabled`) không có tooltip. Vị trí ca: `/dev/ui`, khối `Button` ô `disabled` và mục «Ca đặc biệt» | `e2e/dev-ui.spec.ts` |

## Không đổi
- UX01-4 #3 và US-10: không đổi API, không đổi `openapi.yaml`.
- Không sửa `visual.spec` ảnh mốc `dev-ui-*`, `gradebook-*` (P-91-9).

## Cần hỏi
Không.
