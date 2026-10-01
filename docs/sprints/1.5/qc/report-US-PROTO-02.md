# QC report — US-PROTO-02 (Giảng viên: vận hành lớp)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, commit `990c243` và nền chung.
Spec: `docs/sprints/1.5/spec/` (v3). TC: `docs/sprints/1.5/qc/tc-US-PROTO-02.md`.
**Tóm tắt:** 7/7 AC PASS. Nghiệp vụ vận hành lớp dành cho Giảng viên / Trợ giảng hoàn chỉnh: màn Hôm nay hiển thị đúng danh mục 6 việc cần xử lý; Hộp thư hỗ trợ nhận câu hỏi và xử lý mượt mà; Điểm danh bàn phím dưới 60s, hỗ trợ hoàn tác và giả lập offline; Duyệt thành viên phân quyền chuẩn giữa GV và TA.

## Cổng nghiệm thu đã chạy
| Lệnh / Kiểm tra | KQ | Ghi chú |
| --- | --- | --- |
| `proto-curl.sh tc_02_01` (6 route GV) | PASS | Toàn bộ các route `/inbox`, `/students`, `/students/sv-3`, `/attendance`, `/class/members` đều `MO` |
| `proto-curl.sh tc_02_07` (phân quyền SV/Admin/TA) | PASS | SV và Admin bị chặn (`CHAN`); TA mở được nhưng không có `Tạo lại mã` hay `Mời ra khỏi lớp` |
| "Hôm nay" GV (`/`) | PASS | 6 việc cần xử lý theo đúng thứ tự (ticket 3 ngày 4 giờ đầu tiên, buổi 10 đang diễn ra, BT03 cần xem kỹ, thiết lập lớp mới, duyệt SV...); không card KPI phản mẫu |
| Hộp thư hỗ trợ (`/inbox`) | PASS | Đầy đủ thông tin tuổi ticket, lý do chuyển (độ tin cậy < 0,80), nhận ticket và gửi trả lời kèm lưu tri thức |
| Điểm danh (`/attendance`) | PASS | Buổi 10 (29/10/2026), bàn phím `1`–`4`, `P` phát biểu, Hoàn tác 5s, trạng thái lưu tức thì, chế độ giả lập mất mạng tự đồng bộ khi có mạng |
| Duyệt thành viên (`/class/members`) | PASS | Danh sách thành viên lớp 2, duyệt yêu cầu SV D thành công, đổi vai SV D thấy ngay lớp mới |
| Responsive 375px/390px | PASS | `/inbox` và `/attendance` hiển thị dạng hàng có kẻ, không card bọc card, vùng chạm đạt chuẩn ≥ 44px |

## AC chi tiết
| AC | KQ | Ghi chú |
| --- | --- | --- |
| AC1 | PASS | 6 việc cần làm hôm nay bám sát FLOWS F14, thông báo phân công lớp mới kèm mã tham gia |
| AC2 | PASS | Nhận ticket, soạn câu trả lời, trạng thái cập nhật sang Claimed/Answered, có tùy chọn lưu tri thức |
| AC3 | PASS | Điểm danh bàn phím mượt, hoàn tất buổi 10 báo cáo 27 có mặt, 1 muộn, 2 vắng, cập nhật tức thì vào sổ điểm |
| AC4 | PASS | Duyệt thành viên lớp 761988, đồng bộ trạng thái sang phiên sinh viên D |
| AC5 | PASS | Giao diện mobile 375px dùng tốt, vùng chạm ≥ 44px, không cuộn ngang |
| AC6 | PASS | Nhánh lỗi: Giả lập mất mạng xếp hàng thay đổi rồi đồng bộ; `/students?state=error` có nút `Thử lại` |
| AC7 | PASS | Phân quyền: SV và Admin bị chặn; TA không có nút `Tạo lại mã` hay `Mời ra khỏi lớp` |

## Lỗi
Không có lỗi phát hiện.

## Đề nghị
Đóng US-PROTO-02 với kết luận **PASS**.
