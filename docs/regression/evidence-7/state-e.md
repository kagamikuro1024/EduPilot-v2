# State nhóm E — slice E_s7_p4b (SmkE.SliceE) — dừng theo lệnh PM (lần 2)

Kết quả: /tmp/qc2e/res/E_s7_p4b.jsonl (đã khử trùng, 1 dòng/TC). Lỗi: /tmp/qc2e/defects/E_s7_p4b.md (DEF-R7-E-01..03). Tổng đã ghi 77/80: {'P': 75, 'F': 2}.
F: TC-P4-06-47, TC-P4-07-15.

## Chưa có kết quả (3)
TC-P4-05-13, TC-P4-05-14, TC-P4-05-15
- TC-P4-05-13/14/15 (job ticket.autoclose chạy mỗi giờ, mốc bị lệch do worker/judge bị restart nhiều lần): dữ liệu đã dựng sẵn ở /tmp/qc2e/e7/s2.json (a8 ANSWERED 8 ngày, a6 trả lời lại, o8 OPEN, c8 CLAIMED quá 7 ngày). Cần: chờ job (judge khởi động 17:0x UTC +n giờ) rồi kiểm; bằng chứng thay thế đã có: go test TestAutoCloseAfter7Days/Idempotent/SkipsOpen PASS. TC 15 (UI "Đã đóng" + Hỏi tiếp cho CLOSED AUTO_7D): UI trạng thái CLOSED + Hỏi tiếp đã kiểm ở 05-27 (CLOSED do dismiss).
- Môi trường: ep-r đã khôi phục (1 worker edupilot-worker-test, không FAKE_LLM_ERROR_RATE, judge đang chạy, readyz 200), lock global đã gỡ.
- Dữ liệu thử còn lại: lớp E7C1, E7C2, E7S18/19/21/22/23, E7SCL (+ tài khoản e7gv/e7ta); ticket E7K* đã xoá.

## Cập nhật 24/11 (qc2, theo PM)
Kỳ thi «QC E7 khoá chat» (01a1e01f-4e89-7a7f-b90b-2a8a92e5cb87, lớp E7C1) có 0 mục nên exam.tick lỗi mỗi 5 s trên worker ep-r. Đã thêm 1 câu TRUE_FALSE + 1 exam_item (evidence-7/fix-e7-exam-item.sql; bước đầu dùng ESSAY lỗi «đáp án đúng không đọc được», đã đổi). Lượt 01a1e01f-4fe9-… chuyển IN_PROGRESS → GRADED, worker hết lỗi. Không xoá gì.
