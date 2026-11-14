# Sprint 10.6: góp ý

| # | Ai | Vấn đề | Đề xuất | Lý do + bằng chứng | Ảnh hưởng | Quyết định PM |
| --- | --- | --- | --- | --- | --- | --- |
| P-106-1 | ba (Q-G-1, 2, 3, 5, 6, 10 FEAT-grading-guidance; nhãn S7) | Câu đụng điểm / quyền / PII | Phương án an toàn trong spec | — | — | **XÁC NHẬN cả 6** (PM 2026-11-14): Q-G-1 cấm cứng công bố lô trộn phiên bản (`MIXED_GUIDANCE`), chấm lại mặc định chỉ bài còn nháp; Q-G-2 TEACHER bật 3 cờ + duyệt, ADMIN bật cờ ca hồi quy, TA chỉ đọc; Q-G-3 ca hồi quy lưu theo tham chiếu, xuất văn bản tắt; Q-G-5 M4 chỉ thread / post công khai; Q-G-6 xoá dữ liệu SV → lưu ý chuyển «Cần xem lại»; Q-G-10 reflect không đọc nhận xét sửa tay. **Nhãn S7:** giữ `expected` = 7,00 (nháp AI lượt 1, test mã dùng); thêm khoá `teacher_label` (S7 = 8,00) cho MỌI phép đo MAE (E3, PoC, 9.5 đo lại); QC áp dụng trong TC. |
