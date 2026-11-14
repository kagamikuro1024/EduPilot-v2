# Dev — Sprint 10.7 (UI-R)

## Lô 1 — `/insights` hiện báo cáo lớp khác (điều tra → sửa)
- Nguyên nhân: `/insights` là màn mô phỏng (không có API). `mockCourseFor(class_code)` rơi về `COURSES[0]` (761987) cho mã lớp thật không có trong mock (KC106…), `reportFor` lại rơi về `TOPICS[COURSE_1]`; `courses` dedupe theo id nên lớp lạ gộp vào 761987. Không rò dữ liệu (hằng số trong bundle, không truy vấn theo lớp).
- Sửa: `mockCourseFor(real)` giữ id / mã / tên THẬT cho lớp ngoài mock; `reportFor` trả `null` → «Chưa có báo cáo» (không thân chữ), header dùng nhãn lớp thật; trạng thái rỗng của màn dùng cùng mẫu. Test: `frontend/e2e/insights-course.spec.ts`.
- **Nợ (trái D51 «cuối sprint 10 không còn mock»):** `/insights` và các màn mô phỏng còn lại (analytics, observability…) thay bằng API thật ở sprint 11 (M13-A/B). Các mock khác vẫn còn `?? X[COURSE_1]` fallback theo lớp — sẽ hết khi nối API.
- `e2e/visual.spec.ts` «insights» đỏ sẵn trên `main` (lệch chữ dưới 1 px so với ảnh gốc, cùng máy) — không do lô này.

## Nợ chuyển từ 10.6 (PM 2026-11-14)
- (d) Bộ lọc «Bài tập» của hàng chờ chấm mất khi rời route (10.6 chỉ giữ Trạng thái / Cần xem kỹ qua `sessionStorage`; chọn bài tập bị `relaxed`/danh sách bài tập tải sau ghi đè). Làm cùng hợp đồng khung §23.
- (k) Dòng báo xong sau «Chấm lại» (`Đã chấm xong N bài`) — nếu QC thấy còn thiếu ở một nhánh (lô không tới SUCCEEDED, tải lại trang giữa chừng), bổ sung.

## Nợ cho hợp đồng khung §23 (techlead-ui)
- **Toolbar 390 px: ba nút xuống 2 dòng.** `/grading` tab Hàng chờ (giảng viên), `Toolbar` `end` = «Tải bài lên» + «Chấm tất cả» + «Công bố» (nút Công bố luôn hiện từ 10.6 (l)). `shared/ui/Layout.module.css`: `.toolbarEnd` không `flex-wrap`, nút co lại nên mỗi nhãn ngắt dòng. Ảnh chuẩn `visual.spec.ts-snapshots/grading-queue-390.png` (P-106-16) ghi nhận hiện trạng này. Gợi ý: `.toolbarEnd { flex-wrap: wrap }` + nút `flex: none` ở ≤ 719 px; hợp đồng §23 nên định nghĩa hàng hành động trên mobile (≤ 1 nút chính + menu `⋯`, theo luật «tối đa 2 hành động phụ»). Khi sửa phải cập nhật lại ảnh chuẩn trên Linux/CI.
