# PU — Nền giao diện "Red Thread / Academic Instrument"

| Ước lượng | Phụ thuộc | Nhánh |
| --- | --- | --- |
| 1,5 tuần | PG | `feat/pu-ui-foundation` |

**Mục tiêu:** dựng MỘT lần hệ token, app shell, primitive và lớp dữ liệu frontend; dựng lại các màn đã có backend theo đúng `design/DESIGN.md`. Mọi phase sau chỉ lắp ghép, không phát minh lại.

Đọc trước (bắt buộc, theo thứ tự): `design/DESIGN.md` → `design/INTEGRATION.md` → `UX.md` → mở `design/edupilot-ui-v3.html` trong trình duyệt để xem hướng thị giác (đổi vai trò ở góc trên phải). Prototype là tham chiếu cảm quan, KHÔNG phải mã nguồn để chép.

**Luật của phase này:** không đổi hành vi sản phẩm, không đổi hợp đồng API, không thêm tính năng. Chỉ thay lớp trình bày và lớp dữ liệu phía client.

## Lát việc

**L1. Token và nền chữ** (`AGENT_PROMPT` Phase 1)
- [ ] `frontend/src/shared/styles/tokens.css` (đã có sẵn từ kit) là nguồn duy nhất; ánh xạ vào cấu hình Tailwind v4 (`@theme`) để chỉ dùng lớp ngữ nghĩa; xoá bảng màu mặc định `gray/slate/zinc/neutral` khỏi theme
- [ ] Be Vietnam Pro qua `next/font` (400/500/600/700, `display: swap`, subset `vietnamese`), fallback đúng `--ep-font`; `tabular-nums` cho bảng
- [ ] Chuẩn hoá: focus ring `--ep-focus`, selection, scrollbar, input, button; `prefers-reduced-motion`
- [ ] Logo + favicon từ `frontend/public/brand/`; không đặt logo trong ô vuông bo góc
- [ ] ESLint / stylelint: cấm hex, `rgb(`, `oklch(`, bóng và bo góc tuỳ ý ngoài `shared/styles/`; cấm `fetch(` ngoài `shared/`

**L2. App shell và điều hướng theo vai trò** (`DESIGN.md` §1, §2)
- [ ] `AppShell`, `Sidebar` (216 / 72 px, nhãn chữ luôn hiện, mục hiện tại = vạch đỏ 2 px, không viên đỏ đặc), `TopBar` 56 px (bộ chọn lớp, `⌘K`, thông báo, hồ sơ), `CommandPalette`, `NotificationPopover` (khung; dữ liệu thật ở P4)
- [ ] Điều hướng đúng từng vai trò; nhóm "Hệ thống" cho Observability / Cấu hình LLM / Tích hợp; bộ đếm chỉ hiện khi ảnh hưởng ưu tiên hành động
- [ ] Mobile < 720: bottom nav tối đa 5 đích + "Thêm"; tablet 720–1099: sidebar thu gọn
- [ ] Route chưa có backend hiện `EmptyState` giải thích, không ẩn khỏi điều hướng của vai trò đó

**L3. Primitive đủ 8 trạng thái** (`AGENT_PROMPT` Phase 2, `DESIGN.md` §10, §15)
- [ ] `PageHeader`, `Section`, `ActionList`, `Toolbar`, `DataTable` (ảo hoá + phân trang con trỏ + bàn phím), `Tabs`, `SegmentedControl`, `InlineNotice`, `StatusText` / `StatusChip`, `Field`, `EmptyState`, `Skeleton` (khớp hình nội dung), `Drawer` (420–520 px, mobile toàn màn), `Dialog` (chỉ cho việc cần bảo vệ), `Composer` (tối đa 820 px), `Button`, `Menu`, `Popover`
- [ ] Mỗi primitive có: default, hover, focus, active/selected, disabled, loading, empty, error
- [ ] Lớp hành vi từ `UX.md` mục 4: TanStack Query + `apiClient`, `useSSE`, `useAutosaveDraft`, `useUndoableAction` (xác nhận tĩnh lặng tại chỗ, không toast "Thành công"), `<ConfirmIrreversible>` dựng trên `Dialog`, `<OfflineBanner>`
- [ ] `/dev/ui`: trưng bày mọi primitive × mọi trạng thái, ở 375 / 900 / 1280 / 1440 px, có đoạn văn tiếng Việt dài để thử tràn chữ và cắt dấu

**L4. Dựng lại các màn đã có backend**
- [ ] `/login`, `/register`, `/profile`, `/settings`
- [ ] `/chat`: cột hội thoại giữa, tối đa 840 px; lịch sử bên trái thu gọn được; không sidebar phải; citation mở tại chỗ (`CitationList`); kết quả tool là khối cấu trúc gọn; ô nhập luôn sẵn sàng; nút Dừng
- [ ] `/threads`, `/threads/[id]`: hàng ngăn bằng đường kẻ; trạng thái xác nhận là chữ + icon xanh, không tô nền; câu trả lời đã xác nhận có đường kẻ xanh 1 px; nháp AI trầm hơn và ghi `Chờ xác nhận`; hành động duyệt nằm ngay cạnh nháp (`VerificationState`)
- [ ] `/documents`, `/analytics`, `/assignments` (tạm, đến khi `/calendar` thay thế ở P8)
- [ ] Trang `/` tạm thời: khung "Hôm nay" theo vai trò với dữ liệu sẵn có (hạn bài tập, thread mới); dữ liệu xếp hạng thật đến ở P2
- [ ] Lời văn theo bảng `DESIGN.md` §13; nút là động từ

**L5. Cổng tự động**
- [ ] Playwright chụp ảnh 7 route đại diện × 2 bề rộng làm ảnh mốc (visual regression)
- [ ] axe trên mọi route; Lighthouse CI cấu hình mobile theo ngân sách `UX.md` mục 3
- [ ] Script `scripts/ui-antipatterns.sh` hiện thực các phép `grep` ở `design/INTEGRATION.md` mục 5

## Cổng nghiệm thu
```bash
pnpm -C frontend lint && pnpm -C frontend build
bash scripts/ui-antipatterns.sh                          # 0 vi phạm
pnpm -C frontend exec playwright test ui-foundation.spec.ts   # bàn phím đi hết /chat và /threads; zoom 200% vẫn dùng được
pnpm -C frontend exec lhci autorun
cd backend-go && go test ./internal/contract/...         # backend không bị đụng
```

## Bạn tự kiểm (đây là phase duy nhất mà mắt người quan trọng hơn test)
- Mở `/dev/ui` cạnh `design/edupilot-ui-v3.html`: cùng một "giọng" thị giác chưa?
- Trả lời 10 câu hỏi nghiệm thu cuối `AGENT_PROMPT.md` cho các màn đã dựng. Câu nào "không" thì chưa xong.
- Đếm bằng mắt: có card lồng card không? Có hơn một nút đỏ đặc trong cùng vùng làm việc không? Màu đỏ nào không mang nghĩa (đang ở đâu / cần hành động / đã sửa-xác nhận)?
- Dùng `/chat` trên điện thoại thật: bàn phím ảo bật lên, ô nhập vẫn với tới được.
- Đọc to mọi chuỗi tiếng Việt trên màn sinh viên: có từ kỹ thuật nào lọt ra không?

## Ghi cho luận văn
Ảnh trước / sau của `/chat` và `/threads`; bảng token; lý do chọn "đỏ là tín hiệu, không phải nền"; điểm Lighthouse và axe.
