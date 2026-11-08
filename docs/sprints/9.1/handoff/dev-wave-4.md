# Dev — đợt 4 (Chấm bài, Sổ điểm, Điểm danh) · sprint 9.1 pha 2

Commit: xem `9.1 đợt 4` trong `git log` nhánh `sprint/9.1-uiux`. Stack kiểm: `-p ep-qc`, `/srv/work/qc` (theo `qc/tc-00-chung.md`).
Ảnh sau 1440 + 390: `dev-wave-4-shots/sau-*` (đồng thời là ảnh mốc cập nhật ở `frontend/e2e/visual.spec.ts-snapshots/`: `attendance-*`, `grading-queue-*`, `grading-review-*`, `gradebook-*`). Ảnh trước do QC chụp (K7).

## Làm được
| Story | Nội dung | Vị trí |
| --- | --- | --- |
| US-12 | Lưới điểm danh: radio "Có mặt" chấm mực (`--ep-ink`), chấm đỏ chỉ cho Muộn / Vắng phép / Vắng (Q-UX91-4); hàng vắng có nhãn chữ trạng thái; nút ghi phát biểu dạng `+` (hover/focus ở desktop, hiện thường trực ở 375 px), giá trị điểm cộng đọc từ công thức lớp (`speakBonusOf`); thanh lưu dính đáy ở di động `Lưu điểm danh · N có mặt · M vắng` trên thanh điều hướng đáy; desktop nút lưu cùng hàng đầu bảng; cảnh báo công thức thu một hàng ở 375 px; nháy lưu `.ep-saved` ở hàng vừa lưu xong. | `features/attendance/`, `shared/domain/AttendanceGrid.*` |
| US-14 | Hàng chờ chấm: một bài tập thì ẩn cột "Bài tập", tên bài hiện trên đầu bảng; lý do cờ gộp vào ô Sinh viên (bỏ cột "Lý do cờ"); hàng bấm được và có liên kết `Duyệt` mũi tên; chip "Cần xem kỹ" căn đáy với ô chọn (`Toolbar alignEnd`); ô Bài tập co theo nội dung; 375 px hàng xếp dọc có nút Duyệt riêng. | `features/grading/QueueTab.tsx`, `Grading.module.css` |
| US-15 | Duyệt bài: mỗi tiêu chí một hàng (tên · ô điểm · `/tối đa`), bỏ nhãn lặp; một ghi chú "Bấm một dòng trích để xem trong bài" ở đầu cột thay cho lặp ở mỗi đoạn trích; bấm đoạn trích → tô sáng giữ ≈ 1 s rồi mờ trong 200 ms (`data-fading`); thanh dưới cột phải dính gồm `Duyệt bài` (nút đỏ duy nhất), tiến độ "Đã duyệt N/M", phím tắt `A`, nút `‹ ›` chuyển bài; lệch hai lượt chấm hiện inline dưới tiêu đề tiêu chí; nháp khôi phục, đổi điểm tự lưu kèm nháy `.ep-saved`; di động 375 px có tab "Bài làm" / "Chấm", thanh duyệt dính đáy. Dữ liệu kiểm hai lượt lệch > 1 điểm: đã có sẵn ở seed lớp 761987 (sv.kha nộp Bài tập 03, mẫu S7: lượt 1 = 1,0, lượt 2 = 2,5 ở tiêu chí 2; P-91-5 Q4). | `features/grading/SubmissionReview.*` |
| US-16 | Sổ điểm: điểm không làm tròn, mỗi cột đệm `0` tới số chữ số lẻ lớn nhất (tối thiểu 2), dấu phẩy thập phân; tiêu đề cột một dòng, trạng thái đã công bố là chấm nhỏ `pubDot` kèm chú giải ở chân bảng, menu `⋯` hiện khi hover/focus; một vùng cuộn dọc cao bằng khung nhìn còn lại (`useTableHeight`), tiêu đề và cột tên dính, giữ ảo hoá của `DataTable`; cột Trạng thái chỉ hiện khi có sinh viên chưa đủ điểm hoặc drift (hàng đủ điểm để trống); đầu trang có liên kết "Thiếu điểm: N sinh viên" lọc nhanh; ô lưu xong nháy `.ep-saved`. | `features/gradebook/Gradebook.*` |

## Test
- `attendance.spec.ts`: cập nhật nút phát biểu theo nhãn mới `Ghi phát biểu`, kiểm đếm `+N`.
- `assignment-lifecycle.spec.ts`: hàng chờ có 7 cột (thay Lý do cờ bằng Duyệt); thông báo lệch điểm đổi sang "Lệch 1,5 điểm so với lượt 2".
- `gradebook.spec.ts`: kiểm pubDot thay dòng chữ phụ; ẩn cột Trạng thái khi cả lớp đủ điểm; đệm số thập phân "7,00".
- Ảnh mốc cập nhật / thêm: `attendance-1440/390`, `grading-queue-1440/390`, `grading-review-1440/390`, `gradebook-1440/390` (chạy qua fake gateway của spec).
- Gate VPS: lint sạch, `ui-antipatterns` sạch, 120 test pass (gồm cả 4 spec chính của đợt).

## Dữ liệu kiểm (P-91-5 Q4)
- Lớp 761987: sinh viên `sv.kha` (Bùi Thanh Khải, MSSV `20229002`), bài Bài tập 03 — mẫu S7: tiêu chí 2 lệch 1,5 điểm (lượt 1 = 1,0, lượt 2 = 2,5).
