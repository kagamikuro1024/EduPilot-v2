# DEV handoff — US-UI-05 (Giảng viên / TA → Panel)
Nhánh `sprint/5.5-ui-panels`; D59 (a). `nav.ts` không đổi (`git diff -- nav.ts` = 0 dòng).

## Làm gì
Cùng cơ chế với UI-04 (`Section panel` — tiêu đề vùng ngoài, nội dung một `Panel`; `PanelSection` để chia nhóm trong panel). Không `Panel` lồng; không CSS khung riêng (đã gỡ nền / viền / bo góc của `.list`/`.detail` ở inbox, `.doc`/`.panel` ở duyệt bài, `.integration`, `.strip`, `.rule`…).
- **Hôm nay** (`StaffToday`): "Việc cần xử lý" và "Sắp tới" = mỗi vùng một Panel; rỗng / tải = Panel.
- **`/class/members`**: `Tabs` ngoài; tab Thành viên / Chờ duyệt = một Panel chứa `Toolbar` + bảng (Toolbar chuyển từ trên Tabs vào trong panel; thông báo hoàn tác / lỗi vẫn ở trên panel); tab Trợ giảng = một Panel; tab Nhập danh sách (`RosterImport`) = một Panel, ba `PanelSection` (Chọn tệp → Xem trước → Kết quả; nút `Xem trước` / `Nhập N sinh viên` giữ nguyên chỗ logic). `/class/settings`: mỗi mục một Panel.
- **`/exams` (Staff)**: nhóm đã là Panel từ UI-04. **`/exams/[id]`**: `Tabs` ngoài; mỗi tab một Panel; `Thông tin` = form chia `PanelSection` (nhập chung · Thời gian · Xáo trộn · Chấm điểm · Sau khi công bố điểm · Lưu); `Câu hỏi` = danh sách + tổng điểm là **một** ô `tone="strong"`; `Xem trước` = Panel (tiêu đề đề thi h2 → h3 vì nằm trong panel); danh sách lỗi lên lịch (`InlineNotice`) chuyển **vào** Panel; form Gia hạn trong Panel riêng; Drawer "Chọn câu từ ngân hàng" không đổi (không chứa Panel).
- **`/exams/[id]/results`**: `Tabs` ngoài, mỗi tab (Bảng điểm / Xem lại điểm / Thống kê) một Panel. **`/exams/[id]/similarity`**: một Panel (dòng "Độ giống chỉ là gợi ý…", bộ lọc, bảng cặp, cặp đang mở); hai khối mã cạnh nhau là hai `PanelSection tone="strong"` (mỗi khối bọc `div` riêng để quy tắc kề-nhau của ô nhấn không lệch cột).
- **`/questions`**: `Toolbar` + `DataTable` trong một Panel.
- **Màn mock**: `/inbox` — **một** Panel chứa danh sách 380 px | chi tiết ngăn bằng đường kẻ dọc (trước: hai khung riêng); tên sinh viên trong chi tiết h2 → h3. `/students` một Panel (Toolbar + bảng); `/students/[id]` mỗi tab một Panel (không thẻ). `/attendance` một Panel (phiên + bảng). `/gradebook` một Panel (Toolbar + bảng); `/gradebook/scheme` — các bước là `PanelSection` trong Panel chính, nguồn trích là Panel thứ hai cạnh bên. `/grading` mỗi tab một Panel; `/grading/[id]` bài + rubric = hai Panel cạnh nhau. `/documents`, `/insights`, `/analytics` mỗi vùng một Panel; `/observability`: dải trạng thái (bỏ kẻ trên / dưới và lề dưới riêng) trong một Panel + hai vùng; `/settings/llm`: năm vùng mỗi vùng một Panel; `/settings/integrations`: mỗi đường một Panel (tiêu đề h2 ra ngoài qua `Section title`).
- Test: `panels.spec.ts` + `staff home`, `members`, `exams staff`, `exam editor`, `questions`, `exam results|similarity`, `staff routes` (bảng 7.2 × `teacher` và `ta` × 1440 / 1024; `/grading/[id]` + `/students/[id]` 5 tab; `/attendance`, `/inbox` ở 375); `support/staffMock.ts` (gateway giả đọc-chỉ: bài thi, câu hỏi, thành viên, join-code, `/admin/llm/*`); `support/panels.ts`: TITLE chỉ đếm `h1/h2` đang hiển thị (hộp thoại đóng không tính).
- Sửa kèm ngoài phạm vi (cần biết): ảnh mốc `inbox-*` và `gradebook-*` **từ Sprint 5 là trang "Bạn không có quyền xem màn này"** (nav của Sinh viên): `visual.spec.ts` đóng băng đồng hồ ở 29/10/2026, token của `asDemo` chỉ sống 15 phút từ giờ thật nên phiên hết hạn và rơi về Sinh viên. `asDemo` nhận thêm `expIn`, `visual.spec.ts` truyền 366 ngày (cùng cách `settings-llm` đã làm). Hệ quả: ảnh mốc Staff nay thật; `home-*` cũng đổi (xem dưới).

## AC
| AC | Kết quả |
| --- | --- |
| 1 | `-g 'staff home'` pass (teacher, ta × 1440 / 1024): đúng 2 Panel ("Việc cần xử lý", "Sắp tới"), NEST = TITLE = WALL = 0, STRONG ≤ 1, không `[data-part=kpi]`. **Lệch spec: không có vùng "Lớp cần chú ý"** — `attention` trong API là `unknown[]`, chưa có hợp đồng dữ liệu; không bịa. `today.spec.ts`, `shell.spec.ts` pass |
| 2 | `-g members` pass: `Tabs` ngoài panel; bốn tab mỗi tab đúng 1 Panel, STRONG ≤ 3, NEST = 0; Nhập danh sách có `PanelSection`. `class-join.spec.ts` (không `@real`) pass. **Chưa** thêm ô nhấn thêm / bỏ qua / lỗi cho tóm tắt (giữ đoạn văn như cũ; "≤ 3" là trần). TA: nhóm tab `Trợ giảng` / `Nhập danh sách` vẫn chỉ hiện với Giảng viên (code không đổi) |
| 3 | `-g 'exams staff'` pass (số Panel = số nhóm có dữ liệu ≥ 3, không Panel trong `li`, đúng 1 nút primary). `exam.spec.ts` pass |
| 4 | `-g 'exam editor'` pass: Tabs ngoài; mỗi tab 1 Panel; `Thông tin` ≥ 4 `PanelSection`; `Câu hỏi` strongMax = 1; mở Drawer: không `[role=dialog] [data-ep-panel]`, NEST = 0 |
| 5 | `-g questions` pass; `exam.spec.ts -g 'questions bank'` pass; `route access` pass |
| 6 | `-g 'exam results\|similarity'` pass: results 1 Panel + Tabs ngoài + tab "Nghi giống nhau" chỉ hiện với Giảng viên (TA: không có); similarity: 1 Panel, mở cặp = 2 ô strong, dòng "Độ giống chỉ là gợi ý…" còn. Cuộn 1.000 dòng (khung `requestAnimationFrame`, bảng ảo hoá, 3 lần × 235 khung, máy dev): **trước story p50 16,6–16,7 ms / p95 18,2–18,3 ms; sau story p50 16,7 / p95 16,8 ms** (≤ 25 ms). Đo bằng spec tạm đã gỡ; "trước" = `cc748ac`. Chỉ là đo khung trên máy dev, không phải đo trên máy đích |
| 7 | `-g 'staff routes'` pass: bảng 7.2 × `teacher`, `ta` × 1440 / 1024 (≥ 1 Panel, TITLE / NEST / WALL = 0, STRONG ≤ 3, không tràn ngang); `/grading/[id]` 2 Panel cạnh nhau; `/students/[id]` 5 tab × 1 Panel, 0 ô nhấn; `/attendance`, `/inbox` ở 375: AUDIT sạch, `TOUCH_SRC` rỗng (bỏ qua liên kết ẩn "Bỏ qua điều hướng"), lề Panel 12 px. `ui-antipatterns` 22 ✓ / 0 ✗ |
| 8 | Không đổi `ACCESS` / `EXAM_DYNAMIC` / `nav.ts`. Chuỗi hiển thị thêm: tiêu đề `PanelSection` "Chọn tệp", "Xem trước", "Kết quả" (Nhập danh sách) và "Thời gian", "Chấm điểm" (Thông tin bài thi) — theo AC2 / AC4. Ca `exam today account class-join shell`: spec không đổi |

## Số ca (non-`@real`, non-`visual`)
| | Sau story | Trước story (UI-04) |
| --- | --- | --- |
| Toàn bộ Playwright | 502 pass / 162 skip / 0 fail | 490 pass / 150 skip |
Chênh đúng +12 / +12 = 12 ca mới (desktop chạy, `mobile` skip). Thêm 1 ca (`/grading` + `/students`) sau lần chạy này (+1 / +1). Bộ `exam`, `today`, `account`, `class-join`, `shell`: không sửa tệp, số ca không đổi.

## Ảnh mốc (14 ảnh, image `mcr.microsoft.com/playwright:v1.63.0-noble`, `--update-snapshots=all`; sau đó 2 lần không `--update` = 14 passed)
Đổi 8 ảnh:
| Ảnh | Lý do |
| --- | --- |
| `inbox-1440.png`, `inbox-390.png` | Trước: trang chặn quyền (xem trên). Sau: Hộp thư hỗ trợ thật, một Panel chứa danh sách và chi tiết |
| `gradebook-1440.png`, `gradebook-390.png` | Trước: trang chặn quyền. Sau: Sổ điểm thật, Toolbar + bảng trong một Panel |
| `home-1440.png`, `home-390.png` | Trước: phiên hết hạn nên không hiện đúng trang Sinh viên đã vào lớp; sau: "Chào Uyên", vùng "Việc nên làm tiếp" là một Panel |
| `settings-llm-1440.png`, `settings-llm-390.png` | Năm vùng cấu hình mỗi vùng một Panel (tiêu đề ngoài) |
Giữ nguyên 6 ảnh: `chat-*`, `threads-*` (đã đổi ở UI-04), `dev-ui-*`.

## Gate
`pnpm lint`, `tsc --noEmit` sạch; `ui-antipatterns.sh` 22 ✓ / 0 ✗, `--selftest` 22/22; `lint-selftest.sh` 7/7 + 22/22; `ui-allow:` vẫn 9; không thêm thư viện.

## Chưa chạy / nợ
- Vùng "Lớp cần chú ý" của `/` (xem AC1) — chờ hợp đồng dữ liệu `attention`.
- Thống kê (`StatsPanel`) và Phúc khảo (`AppealsPanel`) của kết quả bài thi: bọc Panel nhưng không có ca `panels.spec` riêng (mock chưa dựng dữ liệu thống kê).
- Lighthouse (`lhci`) chưa chạy; `@real` chưa chạy; đo cuộn 1.000 dòng chỉ trên máy dev.
- Ảnh mốc Sprint 5 `inbox-*` / `gradebook-*` sai vai (xem trên) — báo PM / QC vì bằng chứng UI-02 / UI-03 cho hai ảnh này không phản ánh màn Staff.

## Sửa lỗi QC — B1 (AC3, `/exams` rỗng; QC FAIL TC-UI5-03)
- Nguyên nhân: ở `ExamsHome` (Staff) trạng thái **rỗng** (`PageState empty`), **tải** (`loading`) và "Chưa chọn lớp" đặt `EmptyState` / `Skeleton` trơn trên canvas; test của tôi chỉ chạy mock có dữ liệu. Phía Sinh viên rỗng / tải đã trong Panel từ UI-04 (xem dưới).
- Sửa (`features/exam/ExamsHome.tsx`): rỗng, tải, "Chưa chọn lớp" của Staff và tải của Sinh viên (hai chỗ `Skeleton` đầu hàm) đều bọc trong **một** `Panel`. Chữ không đổi.
- Test mới `panels.spec.ts -g 'exams empty'` (`mockStaffApi(page, { emptyExams: true })`): Giảng viên `/exams` rỗng = đúng 1 Panel, `EmptyState` và nút `Tạo bài thi` của nó nằm trong Panel, NEST / TITLE / WALL = 0; Sinh viên `/exams` rỗng = 1 Panel. Pass.
- L1 (TITLE với `Dialog` đóng nằm trong DOM của Panel): phép đo TITLE trong `e2e/support/panels.ts` chỉ đếm `h1` / `h2` **đang hiển thị** (hộp thoại đóng không tính) — chờ BA chốt định nghĩa (Q-QC-UI04-2); chưa đổi cấu trúc `Dialog`.
- Các màn Staff / Admin khác đã kiểm trạng thái rỗng bằng đọc mã: bảng rỗng của `DataTable` nằm trong Panel (`members`, `questions`, `results`, `similarity`, `admin/*`); chưa thêm ca mock rỗng riêng cho từng màn.
