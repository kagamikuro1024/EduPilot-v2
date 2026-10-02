# QC report — US-PROTO-01 (Sinh viên)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, commit `29eb482` và nền chung.
Spec: `docs/sprints/1.5/spec/` (v4 cập nhật luồng Threads 01-AC4 và Reply Composer 01-AC10).
TC: `docs/sprints/1.5/qc/tc-US-PROTO-01.md` (38 TC).
**Tóm tắt:** 10/10 AC PASS (bao gồm 01-AC4 form tạo thread chi tiết và 01-AC10 reply composer). Trải nghiệm sinh viên hoàn thiện tuyệt đối: hỏi riêng AI D1–D3 (bảo vệ thông tin cá nhân, chặn hỏi hộ, tự chuyển GV), Threads có form tạo thread đầy đủ, quét PII dialog 2 lối, chuyển hướng sau khi đăng, Reply Composer có nút Hỏi trợ lý AI; What-if tính điểm chuẩn xác theo spec v3/v4.

## Cổng nghiệm thu đã chạy
| Lệnh / Kiểm tra | KQ | Ghi chú |
| --- | --- | --- |
| `proto-curl.sh tc_01_01` (12 route SV B) | PASS | Toàn bộ 12 route đều `MO` với vai student |
| `proto-curl.sh tc_01_05` (quét từ cấm SV) | PASS | Hoàn toàn sạch từ cấm kỹ thuật (RAG, PII, fallback, trace, confidence, độ tin cậy...) |
| `proto-curl.sh tc_01_08` (SV D xem trước lớp 761988) | PASS | Lớp 761988 hiển thị đúng ở link join |
| `proto-curl.sh tc_01_09` (phân quyền vai khác) | PASS | TA, GV, Admin đều bị `CHAN` khi mở route riêng của SV |
| Form tạo thread (`/threads`, 01-AC4) | PASS | Đủ các trường: Tiêu đề (Input), Chủ đề (Select), Nội dung chi tiết (Textarea), checkbox hỏi AI gợi ý Socratic (mặc định bật); quét PII (Dialog 2 lối); đăng thành công chuyển hướng ngay sang `/threads/${id}` |
| Chi tiết thread & Reply Composer (`/threads/[id]`, 01-AC10) | PASS | Bố cục panel rõ ràng: Khối câu hỏi gốc, Khối câu trả lời AI (kèm trích dẫn nguồn mở rộng), Danh sách phản hồi, Khối Reply Composer ở cuối trang; nút `Hỏi trợ lý AI` sinh gợi ý Socratic; quét PII khi gửi phản hồi |
| Hỏi riêng D1, D2, D3 (`/chat`) | PASS | D1 che 2 PII, trả lời chảy, vắng 2 buổi, +0,75 phát biểu, 2 nguồn tham khảo; D2 từ chối lịch sự; D3 tự chuyển giảng viên |
| Sổ điểm của tôi (`/me`) | PASS | Ban đầu QT 8,3; sau điểm danh 8,5; sau công bố BT03 8,7 và What-if CK 8,0 ra 8,3; nhập 11 báo lỗi tại ô |
| Responsive 375px/390px | PASS | Bottom nav 5 mục, không cuộn ngang, vùng chạm ≥ 44px, drawer "Thêm" chuẩn |

## AC chi tiết (v4)
| AC | KQ | Ghi chú |
| --- | --- | --- |
| AC1 | PASS | 12 route SV mở được, khung nhìn đầu bám sát DESIGN §14.x, một hành động chính rõ ràng |
| AC2 | PASS | Trả lời D1 nêu đúng vắng 2 buổi, +0,75 phát biểu, 2 nguồn tham khảo, D2 từ chối hỏi hộ |
| AC3 | PASS | D3 tự chuyển giảng viên, khi giảng viên trả lời có nhãn GV và nút `Đã rõ` |
| AC4 | PASS | Form tạo thread đầy đủ, quét PII dialog 2 lối, chuyển hướng tức thì sang `/threads/${id}` (v4, Proposal #15) |
| AC5 | PASS | Sinh viên không thấy từ kỹ thuật AI, không thấy nhãn rủi ro, `/assignments/bt03` trước công bố hiển thị "Đang chấm", không lộ điểm nháp |
| AC6 | PASS | Dùng tốt ở 375px, bottom nav ≤ 5 đích, lịch sử chat ẩn vào drawer khi màn hình hẹp |
| AC7 | PASS | Khớp chính xác mốc điểm SRS 4.1 v3/v4 (8,3 → 8,5 → 8,7; What-if CK 8,0 = 8,3); nhập 11 báo lỗi tại ô |
| AC8 | PASS | Nhập mã sai báo câu chung; nhập `BX4P9TW` xem trước lớp 761988 rồi chờ duyệt; composer giữ chữ khi lỗi |
| AC9 | PASS | Chặn toàn bộ vai TA/GV/Admin truy cập route học tập riêng của SV |
| AC10 | PASS | Reply Composer ở cuối chi tiết thread, nút Hỏi trợ lý AI gợi ý Socratic có nguồn, quét PII phản hồi (v4, Proposal #15) |

## Lỗi
Không có lỗi phát hiện.

## Đề nghị
Đóng US-PROTO-01 với kết luận **PASS**.
