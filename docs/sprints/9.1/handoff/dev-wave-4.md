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

## Sửa QC đợt 4 (P-91-10) — commit `9.1 sửa QC đợt 4`
| # | Lỗi QC | Sửa | Vị trí |
| --- | --- | --- | --- |
| 1 | US-15 AC7 lỗi duyệt không ở thanh dưới, không `Thử lại` | `approve` bắt lỗi (trừ `VERSION_CONFLICT`, vẫn hỏi giữ bên nào) và hiện `InlineNotice` + `Thử lại` ngay trong thanh dưới | `SubmissionReview.tsx`, `.module.css` (`footErr`) |
| 2 | US-15 AC3 thanh dưới bị cắt ở 1440×900 | chiều cao cột phải = khung nhìn − vị trí đỉnh cột − 56 px (đo lúc dựng + khi resize, chỉ ≥ 1100 px) thay cho hằng 130 px | `SubmissionReview.tsx` (`panelRef`) |
| 3 | «Đã duyệt N/M» N = 0, M co lại | N, M mang theo URL (`?n=&m=`): N tăng mỗi lần duyệt, M = số bài của hàng chờ lúc mở (P-91-10 (3)) | `SubmissionReview.tsx` |
| 4 | US-16 AC1 ô nhập «7», «8,5» | `cell.display` mới của `DataTable`: ô không focus hiện giá trị đệm 0 (`padScore`), focus hiện giá trị gốc + chọn hết, đang có nháp thì hiện nháp (P-91-10 (2)) | `DataTable.tsx`, `DataTableVirtual.tsx`, `Gradebook.tsx` |
| 5 | US-16 AC2 `⋯` tiêu đề cột thường trực | độ mờ đặt trên chính nút (`opacity:0`, hiện khi rê / focus-within; chỉ lộ sẵn khi `hover:none`) | `Gradebook.module.css` |
| 6 | US-16 AC10 ô nhập 32 px ở 375 | `.cellInput { min-height: 44px }` dưới 720 px | `DataTable.module.css` |
| 7 | US-12 AC7 khối cảnh báo 62 px ở 375 | di động: nhãn nút «Xem» (tên trợ năng vẫn «Xem công thức»), vùng chạm 44 px bù bằng lề âm để khối một dòng ≤ 56 px | `RealAttendance.tsx`, `.module.css` |

Test đổi theo AC mới (không nới): `gradebook.spec.ts` ô không focus «6,00», «8,50», «8,00», focus → «6»; ô đang có nháp giữ chữ gõ. Thêm: `assignment-lifecycle.spec.ts` (thanh dưới trong 900 px, N/M, lỗi duyệt ở thanh dưới + Thử lại), `attendance.spec.ts` (di động ≤ 56 px, nút ≥ 44 px), `gradebook.spec.ts` (`⋯` cột mờ 0 → 1 khi focus).
Gate VPS (`/srv/work/dev`): lint sạch, build xanh, `ui-antipatterns` sạch; `gradebook` + `attendance` + `assignment-lifecycle` (desktop + mobile) xanh.
Không kiểm ở 375 bằng stack thật: đo di động ở 390 (project `mobile`); QC kiểm lại 375 + khung 1440×900 trên `ep-qc`.

## Sửa QC đợt 4 lỗi #8 (US-15 AC7, 409)
- `SubmissionReview.tsx`: khi `PUT grading-results` trả 409, ô điểm GIỮ giá trị đã gõ (không lùi); một hộp duy nhất "Người khác vừa nhập {điểm của họ}" + `Giữ của tôi` (ghi đúng điểm đã gõ, chỉ tiêu chí đang sửa, trên version của họ) / `Dùng của họ` (ô lấy giá trị của họ). Không còn dải đỏ thứ hai, không còn dòng "Hoàn tác" cho lần lưu bị 409.
- `useUndoableAction`: thêm `handled?(e)` — lỗi màn hình đã tự xử lý thì không rollback, không báo lỗi thứ hai.
- E2E: `assignment-lifecycle.spec.ts` — "review version conflict" (giữ giá trị, một thông báo, PUT version 9) + "Dùng của họ". Xanh trên bản build cổng (desktop).
