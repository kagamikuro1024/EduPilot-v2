# DEV handoff — US-PROTO-04 Hiểu lớp + hệ thống
Nhánh `sprint/1.5-mock-ui` · commit: `US-PROTO-04: ...`

## Đã làm (theo thứ tự lát dọc)
- Mock data: `frontend/src/mock/{insights,analytics,system}.ts` (bộ báo cáo lỗ hổng kiến thức 2 lớp, telemetry 50 yêu cầu observability, cấu hình 3 provider LLM và tích hợp Mail/IMAP/Teams).
- Màn hình tính năng (`frontend/src/features/`):
  - `today/AdminHome.tsx`: ActionList việc vận hành (Gemini lỗi 3 lần trong 15p; sắp chạm 80% ngân sách tháng; 2 việc dead-letter; lớp 761988 chưa có giảng viên hoạt động); bấm việc → đúng màn; rỗng 'Hệ thống đang vận hành bình thường'.
  - `insights/`: lớp 1: chủ đề A (AES/CBC) và B (Hàm băm/chữ ký) đứng đầu, câu mẫu đã ẩn danh (không tên, không MSSV 2022xxxx); chủ đề < 3 SV hỏi không có câu mẫu; mục Tài liệu chưa đề cập. Lớp 2: chủ đề C (Tường lửa) đứng đầu. Tạo báo cáo mới có tiến độ → báo cáo 29/10; Tạo thread ghim ghi vào `KEYS.insightThreads`; Tạo buổi ôn tập ghi vào `KEYS.calendarExtras`.
  - `analytics/`: hoạt động học, hỗ trợ (ticket, thời gian trả lời), bảo vệ thông tin cá nhân, chất lượng chấm (tỷ lệ GV sửa điểm AI); **chi phí chỉ GV thấy, TA không thấy (chuỗi 'chi phí' không xuất hiện trong HTML của TA)**. Đổi khoảng 7/30 ngày; biểu đồ SVG sạch sẽ, không thẻ KPI.
  - `observability/`: dải trạng thái (1.284 yêu cầu hôm nay, p95 3,1s, lỗi 0,6%, model dự phòng 1,2%, chi phí 42.000đ), bảng 50 yêu cầu. Admin: mở Drawer bắt buộc nhập lý do kiểm toán trước → hiện nội dung đã che (`[[SV_1]]`), link chi tiết, ghi 'Đã ghi nhật ký kiểm toán'. **GV: chỉ thấy dải + số tổng hợp lớp mình, hàng không mở được và chuỗi `[[SV_` không xuất hiện trong HTML của GV.**
  - `settings/llm`: 3 provider, bảng tác vụ → model, chuỗi dự phòng, embedding tách riêng, ngân sách 62%. Admin: Test kết nối theo hàng (412ms / lỗi mẫu có cách khắc phục), đổi model CHAT tại chỗ + Hoàn tác, khoá API chỉ ghi ('••••3f9a · đã kết nối'). **GV: chỉ xem, không có nút Test kết nối và không có nút sửa.**
  - `settings/integrations`: Mail (đã kết nối, kiểm 08:55), IMAP (chưa cấu hình), Teams (cần quản trị viên trường đồng ý). Gửi thư thử / Kiểm tra kết quả tại chỗ (Admin; GV chỉ xem).
  - `admin/courses`: 2 lớp 761987 và 761988; Mở lớp (form tại chỗ) gán GV; Lưu trữ lớp qua ConfirmIrreversible.
  - `admin/users`: lọc theo vai (GV, TA, Admin, 57 SV); Mời giảng viên (link mời 72h); khoá/mở khoá tài khoản + Hoàn tác; không có tính năng tạo SV.
- Routes app: `frontend/src/app/(app)/{insights,analytics,observability,settings/llm,settings/integrations,admin/courses,admin/users}/page.tsx`.

## File đổi
`frontend/src/features/{insights,analytics,observability,settings,admin}/**`
`frontend/src/features/today/AdminHome.tsx`
`frontend/src/mock/{insights,analytics,system}.ts`
`frontend/src/app/(app)/{insights,analytics,observability,settings/llm,settings/integrations,admin/courses,admin/users}/page.tsx`
`docs/sprints/1.5/shots/04-*.png`

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile
pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh
F=http://localhost:3000 bash docs/sprints/1.5/qc/scripts/proto-curl.sh tc_04_01 tc_04_02 tc_04_03 tc_04_04 tc_04_06
```

## Test đã chạy và kết quả
- `proto-curl.sh tc_04_01 tc_04_02 tc_04_03 tc_04_04 tc_04_06`: PASS 100% (/insights không lộ tên/MSSV; GV không thấy `[[SV_` ở /observability; GV không thấy 'Test kết nối' ở /settings/llm; TA không thấy 'chi phí' ở /analytics; phân quyền chặn đúng).
- Trình duyệt: ảnh chụp desktop 1440 và mobile 390 tại `docs/sprints/1.5/shots/04-*.png`.

## AC tự đánh giá
04-AC1 ✓ · 04-AC2 ✓ · 04-AC3 ✓ · 04-AC4 ✓ · 04-AC5 ✓ · 04-AC6 ✓.

## Nợ / chưa làm / cần hỏi
- Chi tiết yêu cầu ở Observability của Admin mở drawer hiển thị log kiểm toán mô phỏng.

## Cập nhật theo spec v4 (Proposals #16)
- Phân định rõ các khối cấu hình thành các panel sạch sẽ (`configPanel`, `integration`): viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc `var(--ep-radius-md)`, padding `var(--ep-space-6)` ở `/settings/llm` và `/settings/integrations`.
- Danh sách chủ đề ở `/insights` được đóng gói thành các panel độc lập rõ ràng (`.topic`, `.gap`).
