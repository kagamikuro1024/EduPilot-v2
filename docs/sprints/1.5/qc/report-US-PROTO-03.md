# QC report — US-PROTO-03 (Đánh giá)  · Kết luận: PASS
Nhánh `sprint/1.5-mock-ui`, commit `801f9e0` và nền chung.
Spec: `docs/sprints/1.5/spec/` (v4 cập nhật phân định 2 panel độc lập ở màn chấm bài 03-AC1; điểm số theo v3: BT03 = 8,0; QT = 8,7).
TC: `docs/sprints/1.5/qc/tc-US-PROTO-03.md` (29 TC).
**Tóm tắt:** 6/6 AC PASS (bao gồm 03-AC1 cấu trúc 2 panel độc lập ở `/grading/[submissionId]`). Toàn bộ khâu đánh giá thể hiện triệt để nguyên tắc "AI chỉ nháp, người quyết": duyệt bài 2 cột độc lập có cảnh báo lệch điểm, cho phép sửa tiêu chí; công thức điểm bắt buộc giảng viên xác nhận mới mở khoá sổ điểm; câu hỏi và tài liệu phân quyền chặt chẽ.

## Cổng nghiệm thu đã chạy
| Lệnh / Kiểm tra | KQ | Ghi chú |
| --- | --- | --- |
| `proto-curl.sh tc_03_01` (6 route GV) | PASS | Toàn bộ 6 route `/gradebook`, `/gradebook/scheme`, `/grading`, `/grading/sub-bt03-sv-2`, `/questions`, `/documents` đều `MO` |
| `proto-curl.sh tc_03_06` (phân quyền TA / SV / Admin) | PASS | TA không có `Công bố` ("Chỉ giảng viên công bố điểm"), không có `Xác nhận công thức`; SV/Admin bị `CHAN` toàn bộ |
| Bố cục 2 panel `/grading/[submissionId]` (03-AC1) | PASS | Split view rõ ràng: Panel xem bài nộp sinh viên (645px ~56%) và Panel rubric/điểm (503px ~44%), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc, padding đầy đủ, cuộn độc lập |
| Hàng chờ chấm bài (`/grading`) | PASS | Lọc mặc định hiển thị bài SV B ở đầu kèm lý do cờ lệch điểm giữa 2 lượt AI |
| Chi tiết chấm bài (`/grading/sub-bt03-sv-2`) | PASS | Cảnh báo vàng "Hai lượt chấm lệch 1,5 điểm" ở tiêu chí 2; sửa tiêu chí 2 thành 2,5; nộp muộn 1 ngày trừ 0,5; công bố đạt 8,0 |
| Sổ điểm & Công thức điểm (`/gradebook/scheme`) | PASS | Lớp 761988 có banner chưa xác nhận công thức và khoá `Chốt điểm`; tải quy chế rút bản nháp, điền D5 xong mới cho `Xác nhận công thức`; xác nhận xong gỡ banner |
| Giải trình sổ điểm (`/gradebook`) | PASS | Hàng SV B hiển thị đầy đủ chi tiết phép tính tuyến tính khớp SRS 4.1 v3; xuất XLSX giả lập có dòng ghi chú điểm chính thức |
| Quản lý tài liệu (`/documents`) | PASS | File `scan-khong-co-chu.pdf` báo FAILED với lý do dễ hiểu; `ANSWER_KEY` đánh dấu không hiện cho sinh viên và được bảo vệ |

## AC chi tiết (v4)
| AC | KQ | Ghi chú |
| --- | --- | --- |
| AC1 | PASS | 6 route đánh giá mở được, bố cục khớp DESIGN §14.10–14.14, 14.17; riêng `/grading/[submissionId]` phân định 2 panel độc lập rõ ràng (v4, Proposal #16) |
| AC2 | PASS | Duyệt bài SV B, sửa tiêu chí 2 lên 2,5, công bố đạt 8,0 (đã trừ nộp muộn 0,5), QT SV B cập nhật 8,7 theo spec v3 |
| AC3 | PASS | Lớp 2 hiển thị banner công thức chưa xác nhận, khoá chốt điểm; sau khi điền D5 và xác nhận thì gỡ banner, mở khoá chốt điểm |
| AC4 | PASS | Giải trình tuyến tính sổ điểm lớp 1 khớp các mốc điểm; menu Xuất XLSX hoạt động chuẩn |
| AC5 | PASS | Nhánh lỗi: file PDF scan hỏng báo FAILED rõ ràng; chốt điểm khi thiếu điểm cuối kỳ hiện cảnh báo |
| AC6 | PASS | Phân quyền: TA chỉ duyệt bài, không được công bố; không được xác nhận công thức; SV và Admin bị chặn |

## Lỗi
Không có lỗi phát hiện.

## Đề nghị
Đóng US-PROTO-03 với kết luận **PASS**.
