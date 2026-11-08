# Dev — đợt 3 (Hôm nay) · sprint 9.1 pha 2

Commit: xem `9.1 đợt 3` trong `git log` nhánh `sprint/9.1-uiux`. Stack kiểm: `-p ep-qc`, `/srv/work/qc` (theo `qc/tc-00-chung.md`).
Ảnh sau 1440 + 390: `dev-wave-3-shots/` (cũng là ảnh mốc mới `hom-nay-gv-*` ở `frontend/e2e/visual.spec.ts-snapshots/`). Ảnh trước do QC chụp (K7).

## Làm được
| Story | Nội dung | Vị trí |
| --- | --- | --- |
| US-10 | Việc của giảng viên / TA / Admin chia nhóm có `h3` kèm số ("Gấp · 2", "Hôm nay · 1", "Khi rảnh · 2"; tên truy cập "Gấp, 2 việc"): Gấp = `urgency ≠ normal`; Khi rảnh = `EXAM_SIMILARITY` / `UNMATCHED_SUBMISSION` mức thường; Hôm nay = còn lại; nhóm rỗng không hiện; thứ tự máy chủ giữ trong nhóm (`groupOf` ở `TodayActions.tsx`). Một nút đỏ duy nhất = việc đầu của nhóm đầu; các việc còn lại `ghost` có mũi tên. Một lớp đang chọn thì không còn mã lớp ở hàng / dòng phụ (chỉ hiện ở "Tất cả lớp"). Bỏ khối "Việc cần xử lý" trùng tiêu đề trang. ≥ 1440 px: "Lớp cần chú ý" và "Sắp tới" ở cột phải 320 px; nhỏ hơn nằm dưới. | `features/today/{TodayActions,StaffToday,AdminToday,Today.module.css}` |
| US-11 | Sinh viên: thứ tự Việc nên làm tiếp → **Hôm nay** (dòng thời gian) → Tiếp tục học; "Tiếp tục học" chỉ giữ phiên gần nhất cho mỗi cặp (tiêu đề không phân biệt hoa thường, lớp), tối đa 3 hàng. Ngày giờ đã dùng `formatWhen` từ đợt 1. | `features/today/StudentToday.tsx` |

## Test
- `today.spec.ts`: thêm nhóm (4 nhóm kiểu việc, một nút đỏ, ghost + mũi tên, không lặp mã lớp, nhóm rỗng, `urgency high` của nguồn Khi rảnh vẫn là Gấp, cột phải ở 1440 / dưới ở 1024, thứ tự + khử trùng của sinh viên). Các test cũ của Hôm nay giữ nguyên, không phải sửa.
- Ảnh mốc `hom-nay-gv-1440`, `hom-nay-gv-390` mới (gateway giả trong `visual.spec.ts`). `home-1440` / `home-390` (sinh viên, phiên mô phỏng) không đổi hình trong ngưỡng nên không sinh lại.
- Chạy trên VPS: lint 0 lỗi, `ui-antipatterns` sạch, build, `today.spec` 40 passed, `visual.spec` 16 passed.

## Lệch / ghi chú
- US-10 AC5 chỉ rõ ≥ 1440 px là cột phải; 1100–1439 px xếp dưới như hiện nay.
- Dữ liệu kiểm: nhóm "Khi rảnh" cần việc `EXAM_SIMILARITY` / `UNMATCHED_SUBMISSION` mức thường — có ở fixture của test và ảnh mốc; ở seed lớp 761987 tuỳ dữ liệu.
