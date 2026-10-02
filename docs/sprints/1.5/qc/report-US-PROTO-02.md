# QC report — US-PROTO-02 (Giảng viên: vận hành lớp)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, commit `0ca6069` và nền chung.
Spec: `docs/sprints/1.5/spec/` (v4 cập nhật phân tách panel 02-AC5 và kiểm duyệt Threads 02-AC8).
TC: `docs/sprints/1.5/qc/tc-US-PROTO-02.md` (31 TC).
**Tóm tắt:** 8/8 AC PASS (bao gồm 02-AC5 phân định 2 panel độc lập ở `/inbox` và 02-AC8 quyền kiểm duyệt Threads của GV/TA). Vận hành lớp trơn tru: Hôm nay đủ 6 việc cần xử lý; Hộp thư hỗ trợ 2 panel có viền, nền surface, bo góc và cuộn độc lập; Điểm danh bàn phím dưới 60s có hoàn tác và offline; Duyệt thành viên phân quyền chuẩn; Kiểm duyệt Threads inline lưu trạng thái `CORRECTED` kèm bản đối chiếu.

## Cổng nghiệm thu đã chạy
| Lệnh / Kiểm tra | KQ | Ghi chú |
| --- | --- | --- |
| `proto-curl.sh tc_02_01` (6 route GV) | PASS | Toàn bộ các route `/inbox`, `/students`, `/students/sv-3`, `/attendance`, `/class/members` đều `MO` |
| `proto-curl.sh tc_02_07` (phân quyền SV/Admin/TA) | PASS | SV và Admin bị chặn (`CHAN`); TA mở được nhưng không có `Tạo lại mã` hay `Mời ra khỏi lớp` |
| Bố cục 2 panel `/inbox` (02-AC5) | PASS | Split view rõ ràng: cột danh sách ticket (378px) và panel chi tiết (770px), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc, padding đầy đủ, cuộn độc lập |
| Kiểm duyệt Threads của GV/TA (02-AC8) | PASS | Chi tiết thread có nút `Xác nhận`, `Chỉnh sửa`; bấm `Chỉnh sửa` mở ô sửa inline, lưu thành `Đã được giảng viên sửa & xác nhận` (CORRECTED) kèm nút "Xem câu trả lời AI gốc"; nút `Loại khỏi tri thức` có Hoàn tác |
| "Hôm nay" GV (`/`) | PASS | 6 việc cần xử lý theo đúng thứ tự (ticket 3 ngày 4 giờ đầu tiên, buổi 10 đang diễn ra, BT03 cần xem kỹ, thiết lập lớp mới, duyệt SV...); không card KPI phản mẫu |
| Điểm danh (`/attendance`) | PASS | Buổi 10 (29/10/2026), bàn phím `1`–`4`, `P` phát biểu, Hoàn tác 5s, trạng thái lưu tức thì, chế độ giả lập mất mạng tự đồng bộ khi có mạng |
| Duyệt thành viên (`/class/members`) | PASS | Danh sách thành viên lớp 2, duyệt yêu cầu SV D thành công, đổi vai SV D thấy ngay lớp mới |
| Responsive 375px/390px | PASS | `/inbox` mobile chuyển thành danh sách → chi tiết; `/attendance` hàng có kẻ, không card, vùng chạm đạt chuẩn ≥ 44px |

## AC chi tiết (v4)
| AC | KQ | Ghi chú |
| --- | --- | --- |
| AC1 | PASS | 6 việc cần làm hôm nay bám sát FLOWS F14, thông báo phân công lớp mới kèm mã tham gia |
| AC2 | PASS | Nhận ticket, soạn câu trả lời, trạng thái cập nhật sang Claimed/Answered, có tùy chọn lưu tri thức |
| AC3 | PASS | Điểm danh bàn phím mượt, hoàn tất buổi 10 báo cáo 27 có mặt, 1 muộn, 2 vắng, cập nhật tức thì vào sổ điểm |
| AC4 | PASS | Duyệt thành viên lớp 761988, đồng bộ trạng thái sang phiên sinh viên D |
| AC5 | PASS | Desktop phân định rõ 2 panel độc lập (viền, nền surface, bo góc, cuộn riêng); mobile dùng tốt, vùng chạm ≥ 44px (v4, Proposal #16) |
| AC6 | PASS | Nhánh lỗi: Giả lập mất mạng xếp hàng thay đổi rồi đồng bộ; `/students?state=error` có nút `Thử lại` |
| AC7 | PASS | Phân quyền: SV và Admin bị chặn; TA không có nút `Tạo lại mã` hay `Mời ra khỏi lớp` |
| AC8 | PASS | Quyền kiểm duyệt câu trả lời AI của GV/TA: Xác nhận, Chỉnh sửa inline lưu CORRECTED kèm nút xem bản gốc, Loại bỏ có Hoàn tác (v4, Proposal #15) |

## Lỗi
Không có lỗi phát hiện.

## Đề nghị
Đóng US-PROTO-02 với kết luận **PASS**.
