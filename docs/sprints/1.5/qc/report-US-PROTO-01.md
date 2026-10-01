# QC report — US-PROTO-01 (Sinh viên)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, commit `0c45f15` và nền chung.
Spec: `docs/sprints/1.5/spec/` (v3). TC: `docs/sprints/1.5/qc/tc-US-PROTO-01.md`.
**Tóm tắt:** 9/9 AC PASS. Trải nghiệm sinh viên đầy đủ, mượt mà từ hỏi riêng D1–D3 (ẩn thông tin cá nhân, chặn hỏi hộ người khác, tự động chuyển giảng viên), Threads chặn rò rỉ MSSV qua Dialog 2 lối, luyện đề, What-if sổ điểm chuẩn công thức và kịch bản vào lớp của SV D.

## Cổng nghiệm thu đã chạy
| Lệnh / Kiểm tra | KQ | Ghi chú |
| --- | --- | --- |
| `proto-curl.sh tc_01_01` (12 route SV B) | PASS | Toàn bộ 12 route đều `MO` với vai student |
| `proto-curl.sh tc_01_05` (quét từ cấm SV) | PASS | Hoàn toàn sạch từ cấm kỹ thuật (RAG, PII, fallback, trace, confidence, độ tin cậy...) |
| `proto-curl.sh tc_01_08` (SV D xem trước lớp 761988) | PASS | Lớp 761988 hiển thị đúng ở link join |
| `proto-curl.sh tc_01_09` (phân quyền vai khác) | PASS | TA, GV, Admin đều bị `CHAN` khi mở route riêng của SV |
| Tương tác D1 ở `/chat` | PASS | "Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI", câu trả lời chảy, vắng 2 buổi, +0,75, có nguồn tham khảo (2), không lộ `[[SV_1]]` |
| Tương tác D2 ở `/chat` | PASS | Từ chối lịch sự: "chỉ trả lời được thông tin của chính bạn", không lộ dữ liệu SV C |
| Tương tác D3 ở `/chat` | PASS | "AI chưa đủ chắc chắn về câu này", tự chuyển giảng viên, hiện "Đang chờ giảng viên · vừa gửi" |
| Dialog 2 lối ở `/threads` | PASS | Gõ MSSV 20229002 hiện Dialog đúng 2 lối: `Chuyển sang chat riêng` và `Ẩn thông tin rồi đăng` |
| What-if ở `/me` | PASS | Nhập 11 báo lỗi tại ô; ban đầu QT 8,3; sau điểm danh 8,5; sau công bố BT03 đạt 8,7 và What-if CK 8,0 ra 8,3 |
| Responsive 375px/390px | PASS | Bottom nav 5 mục, không cuộn ngang, vùng chạm ≥ 44px, drawer "Thêm" chuẩn |

## AC chi tiết
| AC | KQ | Ghi chú |
| --- | --- | --- |
| AC1 | PASS | 12 route SV mở được, khung nhìn đầu bám sát DESIGN §14.x, một hành động chính rõ ràng |
| AC2 | PASS | Trả lời D1 nêu đúng vắng 2 buổi, +0,75 phát biểu, 2 nguồn tham khảo, D2 từ chối hỏi hộ |
| AC3 | PASS | D3 tự chuyển giảng viên, khi giảng viên trả lời có nhãn GV và nút `Đã rõ` |
| AC4 | PASS | Threads bắt PII (MSSV) mở dialog 2 lối, giữ nguyên chữ khi chuyển sang chat riêng |
| AC5 | PASS | Sinh viên không thấy từ kỹ thuật AI, không thấy nhãn rủi ro, `/assignments/bt03` trước công bố hiển thị "Đang chấm", không lộ điểm nháp |
| AC6 | PASS | Dùng tốt ở 375px, bottom nav ≤ 5 đích, lịch sử chat ẩn vào drawer khi màn hình hẹp |
| AC7 | PASS | Khớp chính xác mốc điểm SRS 4.1 v3 (8,3 → 8,5 → 8,7; What-if CK 8,0 = 8,3); nhập 11 báo lỗi tại ô |
| AC8 | PASS | Nhập mã sai báo câu chung; nhập `BX4P9TW` xem trước lớp 761988 rồi chờ duyệt; composer giữ chữ khi lỗi |
| AC9 | PASS | Chặn toàn bộ vai TA/GV/Admin truy cập route học tập riêng của SV |

## Lỗi
Không có lỗi phát hiện.

## Đề nghị
Đóng US-PROTO-01 với kết luận **PASS**.
