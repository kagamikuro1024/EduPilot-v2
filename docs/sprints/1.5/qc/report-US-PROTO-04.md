# QC report — US-PROTO-04 (Hiểu lớp + hệ thống)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, commit `7987496` và nền chung.
Spec: `docs/sprints/1.5/spec/` (v4 cập nhật phân tách các panel cấu hình `/settings` và phân tích `/insights`).
TC: `docs/sprints/1.5/qc/tc-US-PROTO-04.md` (25 TC).
**Tóm tắt:** 6/6 AC PASS. Toàn bộ các route phân tích (Insights, Analytics) và quản trị hệ thống (Observability, Settings LLM/Integrations, Admin Courses/Users) hoạt động chuẩn xác; các panel cấu hình và phân tích được đóng gói rõ ràng theo FR-X11; tuân thủ nghiêm ngặt quy tắc che giấu thông tin cá nhân và bảo mật nhật ký kiểm toán.

## Cổng nghiệm thu đã chạy
| Lệnh / Kiểm tra | KQ | Ghi chú |
| --- | --- | --- |
| `proto-curl.sh tc_04_01` (GV 5 route, Admin 6 route) | PASS | Toàn bộ các route hệ thống đều `MO` với vai trò tương ứng |
| `proto-curl.sh tc_04_02` (quét PII ở `/insights`) | PASS | Không rò rỉ MSSV 2022xxxx hay tên sinh viên trong câu hỏi mẫu |
| `proto-curl.sh tc_04_03` (GV ở `/observability`) | PASS | Giảng viên không thấy `[[SV_` hay chi tiết prompt của sinh viên |
| `proto-curl.sh tc_04_04` (GV ở `/settings/llm`) | PASS | Giảng viên không có nút `Test kết nối` |
| `proto-curl.sh tc_04_06` (phân quyền vai khác) | PASS | SV và TA bị `CHAN` toàn bộ route hệ thống; GV bị `CHAN` khi mở `/admin/courses` |
| Báo cáo lỗ hổng (`/insights`) | PASS | Lớp 1 có chủ đề A, B đứng đầu; câu mẫu ẩn danh; các khối chủ đề và kiến nghị phân định panel rõ ràng (FR-X11); có nút `Ghim vào Threads` |
| Quan sát AI (`/observability`) | PASS | Admin mở drawer bắt buộc nhập lý do truy cập và ghi nhận nhật ký kiểm toán; GV chỉ xem số liệu tổng hợp lớp mình |
| Cấu hình LLM & Tích hợp (`/settings/*`) | PASS | Các khối nhà cung cấp và bảng tác vụ phân tách panel độc lập có viền, nền surface; Admin test kết nối hiển thị độ trễ tức thì; khoá API hiển thị dạng che; GV chỉ xem |
| Quản trị lớp & người dùng (`/admin/*`) | PASS | Mở lớp mới gửi thông báo phân công; mời giảng viên thời hạn 72h; không có tính năng "tạo sinh viên" thủ công |

## AC chi tiết (v4)
| AC | KQ | Ghi chú |
| --- | --- | --- |
| AC1 | PASS | GV mở 5 route, Admin mở 6 route, khung nhìn đầu bám sát DESIGN §14.21–14.25 / INTEGRATION 2 |
| AC2 | PASS | Insights lớp 1 xếp hạng chủ đề A, B; câu mẫu được ẩn danh hoàn toàn; ghim sang Threads hoạt động |
| AC3 | PASS | Observability: Admin xem chi tiết yêu cầu cần lý do (có audit log); GV chỉ xem số liệu tổng quan |
| AC4 | PASS | Settings LLM: Admin test kết nối có kết quả tại chỗ, đổi model có hoàn tác, khoá API ẩn; GV chỉ xem |
| AC5 | PASS | Nhánh lỗi: test provider lỗi giải thích rõ ràng; `/insights?state=empty` có nút tạo báo cáo |
| AC6 | PASS | Phân quyền: SV và TA bị chặn; GV không được vào `/admin/courses`; TA vào `/analytics` không thấy chi phí |

## Lỗi
Không có lỗi phát hiện.

## Đề nghị
Đóng US-PROTO-04 với kết luận **PASS**.
