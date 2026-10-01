---
name: gate
description: Chạy cổng nghiệm thu của một phase EduPilot v2 và báo cáo trung thực. Dùng khi chủ dự án gõ /gate <MÃ PHASE>.
argument-hint: "<P0|PG|PU|P1..P10|PR>"
disable-model-invocation: true
---

Chạy cổng nghiệm thu cho phase: $ARGUMENTS

1. Đọc mục "Cổng nghiệm thu" trong `docs/phases/$0.md`.
2. Chạy TỪNG lệnh, đúng như viết. Không sửa lệnh, không bỏ qua lệnh, không sửa test hay golden file để qua.
3. Với mỗi lệnh báo: PASS / FAIL + trích 10 dòng đầu ra liên quan.
4. Kiểm Definition of Done chung: migration chạy sạch trên DB trống; test cũ không đỏ; API mới có trong `backend-go/api/openapi.yaml`; STUDENT gọi API giảng viên nhận 403; seed có dữ liệu cho màn mới; không có PII/secret trong log và diff.
4b. Kiểm luật mở rộng trên diff của phase: không `fetch(` ngoài `frontend/src/shared/`; không `OFFSET` hay danh sách thiếu `limit` trong `internal/store/queries/`; không ghi đĩa cục bộ trong `internal/`; mọi lời gọi LLM mới có `task` và đi qua Scheduler; POST có tác dụng phụ mới đều nhận `Idempotency-Key`; việc dài trả 202. Báo từng vi phạm kèm file:dòng.
4c. Chạy `bash scripts/ui-antipatterns.sh` và báo từng vi phạm. Với mỗi màn mới, đối chiếu hợp đồng route ở `docs/design/DESIGN.md` §14 và 10 điều kiện ở §22; nêu rõ điều kiện nào chưa đạt. Module mới có sinh việc cần người xử lý mà chưa đăng ký Provider cho "Hôm nay" → báo thiếu.
4d. Kiểm cổng UX (`docs/UX.md` mục 6) cho mọi màn mới: chạy axe + Playwright 375 px + Lighthouse CI nếu đã cấu hình; liệt kê màn nào thiếu trạng thái tải/rỗng/lỗi.
4e. Kiểm mức luồng: với mỗi luồng trong `docs/FLOWS.md` mà phase này hoàn tất (cột "Phase hoàn tất" ở mục 1), xác nhận có spec E2E đi trọn đường chính và ít nhất một nhánh lỗi; chạy chúng; cập nhật dòng tick luồng trong `docs/PROGRESS.md`.
5. Nếu có FAIL: nêu nguyên nhân gốc và đề xuất sửa, KHÔNG tự sửa trong lệnh này.
6. Nếu tất cả PASS: in lại danh sách "Bạn tự kiểm" của phase để chủ dự án làm tay, và nhắc ghi `docs/thesis-notes/$0.md`.
