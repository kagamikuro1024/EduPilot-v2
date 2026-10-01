# DEV handoff — US-PROTO-03 Đánh giá
Nhánh `sprint/1.5-mock-ui` · commit: `US-PROTO-03: ...`

## Đã làm (theo thứ tự lát dọc)
- Mock data: `frontend/src/mock/{assess,gradebook,questions,documents}.ts` (bộ 100 câu hỏi An ninh mạng, 12 tài liệu với quyền hạn, BT03 dữ liệu chấm theo proposals #13).
- Màn hình tính năng (`frontend/src/features/`):
  - `gradebook/`: lớp 1 — 30 hàng, cột dính tên/MSSV, cột BT01, BT02, BT03 (chỉ hiện sau khi công bố), cộng/trừ, QT tạm tính tính bằng `calcQt`/`qtOf`, CK để trống, trạng thái; dòng 'điểm chính thức nằm ở hệ thống quản lý đào tạo của trường'. Sửa ô tại chỗ (Enter/Tab) cho GV, ô mẫu xung đột 409 (Lê Thu Hà vừa sửa thành 8,0 · Giữ của tôi / Dùng bản mới); mở giải trình hàng B = phép tính tuyến tính khớp số; Xuất XLSX trong menu; Chốt điểm (GV) qua `FinalizeGrades` nêu hậu quả bằng số ('Chốt 30 sinh viên; 30 chưa có điểm cuối kỳ') và khoá khi thiếu CK. Lớp 2: banner cố định 'Công thức điểm chưa xác nhận', Chốt điểm khoá kèm lý do, tự hết banner khi công thức confirmed. TA: xem cả lớp, không sửa ô, không Chốt điểm.
  - `gradebook/scheme`: layout tài liệu duyệt: lớp 2 — nếu chưa tải quy chế: rỗng + nút 'Tải quy chế' có tiến độ → bản nháp QT 30/CK 70, +0,2 phát biểu trần +0,6, −0,5 vắng từ buổi 3, mục 'chưa rõ: quy tắc làm tròn' chặn nút xác nhận; điền D5 'Làm tròn đến 0,1' → hết chặn → 'Xác nhận công thức' (sticky, chỉ GV) mở `ConfirmGradeScheme` nêu hậu quả → confirmed, gỡ banner bên sổ điểm. Panel nguồn bên phải ≥ 1100px / Drawer khi hẹp. Lớp 1: xem công thức đã xác nhận. TA: chỉ xem, không có nút 'Xác nhận công thức'.
  - `grading/`: tab Hàng chờ chấm (28 bài BT03; lọc mặc định Cần xem kỹ + Chưa duyệt = 4 bài, B ở đầu do 2 lượt lệch 1,5đ) và tab Bài tập (BT01–BT03, QUIZ01, Tạo bài tập form tại chỗ lưu nháp). Chọn bài đã duyệt → `Công bố` (chỉ GV, qua `PublishGrades`) ⇒ ghi `bt03.status='published'`; TA thấy 'Chỉ giảng viên công bố điểm' và vẫn Duyệt bài được.
  - `grading/[submissionId]`: bài văn bản của B bên trái; bên phải 4 tiêu chí (mock/assess.ts) có điểm AI, đoạn trích, nhận xét sửa được; thông báo vàng ở tiêu chí 2 'Hai lượt chấm lệch 1,5 điểm'; sửa tiêu chí 2 thành 2,5 → tổng tính lại ngay = 8,5 (sau trừ nộp muộn 0,5 = 8,0 theo proposals #13); `Duyệt bài` → approved và quay lại hàng chờ.
  - `questions/`: 100 câu hỏi (80 duyệt, 20 chờ), lọc trạng thái, chủ đề, độ khó, loại, nguồn; Drawer xem câu → Duyệt / Chỉnh sửa / Loại; Tạo câu hỏi luồng riêng (thêm 5 câu chờ duyệt).
  - `documents/`: 12 tài liệu với trạng thái, quyền xem của SV, dùng cho AI; dropzone tải file tại chỗ có tiến độ; file mẫu 'scan-khong-co-chu.pdf' → báo FAILED; ANSWER_KEY khoá không cho SV xem và không dùng cho AI SV; bật/tắt cờ tại chỗ + Hoàn tác; xoá qua `DeleteDocument`.
- Routes app: `frontend/src/app/(app)/{gradebook,gradebook/scheme,grading,grading/[submissionId],questions,documents}/page.tsx`.

## File đổi
`frontend/src/features/{gradebook,grading,questions,documents}/**`
`frontend/src/mock/{assess,gradebook,questions,documents}.ts`
`frontend/src/app/(app)/{gradebook,gradebook/scheme,grading,grading/[submissionId],questions,documents}/page.tsx`
`docs/sprints/1.5/shots/03-*.png`

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile
pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh
F=http://localhost:3000 bash docs/sprints/1.5/qc/scripts/proto-curl.sh tc_03_01 tc_03_06
```

## Test đã chạy và kết quả
- `proto-curl.sh tc_03_01 tc_03_06`: PASS 100% (mọi route mở được với GV; TA thấy 'Chỉ giảng viên công bố điểm' ở /grading, không thấy 'Xác nhận công thức' ở /gradebook/scheme; SV/Admin bị chặn).
- Trình duyệt: ảnh chụp desktop 1440 và mobile 390 tại `docs/sprints/1.5/shots/03-*.png`.

## AC tự đánh giá
03-AC1 ✓ · 03-AC2 ✓ · 03-AC3 ✓ · 03-AC4 ✓ · 03-AC5 ✓ · 03-AC6 ✓.

## Nợ / chưa làm / cần hỏi
- Xuất XLSX trong Sổ điểm tạo file blob tải về mô phỏng bảng điểm.
