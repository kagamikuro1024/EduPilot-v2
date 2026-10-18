# Sprint 8: P6 Sổ điểm và điểm cuối kỳ từ quy chế môn học

**Trạng thái:** bản nháp; chủ dự án hoãn sprint 8 (2026-10-18), chưa giao việc.
**Nhánh:** `sprint/8-p6` tách từ `main` (sau merge sprint 7). Dev làm ở `../TA_Agent_v2-s7`, PM ở `../TA_Agent_v2-s6` (detached), BA ở `/tmp/ba-wt`.

## Mục tiêu
- Công thức điểm đến từ file quy chế môn học, **AI chỉ trích bản nháp, giảng viên xác nhận, code tính** (luật 5: `shopspring/decimal`, không `float64`, không prompt nào tính hay làm tròn điểm).
- Điểm cuối kỳ đúng tuyệt đối và giải trình được từng thành phần; nhắc khi lớp chưa có công thức.
- Nguồn điểm đổ về sổ: phát biểu (P5), **điểm bài thi PE** (nợ PE (5)), nhập XLSX giữa kỳ / cuối kỳ, sửa ô trực tiếp. Bài tập P7 nối ở sprint 9.
- **Luồng đi trọn:** F10. **Cổng:** `gate P6` + **chủ dự án tự tính tay 3 sinh viên A, B, C** (ROADMAP).

## Story (lát dọc)

| # | Story | Lát | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 1 | **US-P6-01** Migration `00016_gradebook` (`grade_schemes`, `grade_components`, `grade_rules`, `grade_items`, `grade_entries`, `final_grade_snapshots`; `version`, `unlock_reason` trong CREATE), store sqlc. `grade_items` nguồn `EXAM` nối điểm bài thi PE đã công bố, `PARTICIPATION` đọc từ P5 | P6 L1 | M | – |
| 2 | **US-P6-02** `internal/grade` Grade Engine: trọng số, trần điểm cộng, bậc trừ vắng, làm tròn, thang chữ, điều kiện liệt, đọc từ `grade_rules.params`; giải trình từng sinh viên có `source_quote`. Test table-driven theo `seed/expected_final_grades.csv` (30/30) + biên làm tròn x,25 / x,75 / x,5. Test kiến trúc: không `float64`, không import `activity` / LLM | P6 L3 | L | 1 |
| 3 | **US-P6-03** Trích công thức: `internal/grading.ExtractGradeScheme` qua `llm.Structured` (schema `{components[], rules[], citations[], unclear_items[]}`, mỗi mục có trang trích); việc nền 202 + tiến độ SSE; bộ kiểm bằng code thêm `unclear_items` (trọng số ≠ 100 %, thiếu trần, thiếu làm tròn…); provider giả có bản ghi cho quy chế seed | P6 L1 | M | 1 |
| 4 | **US-P6-04** `/gradebook/scheme` (`DESIGN.md` §14.11, `GradeSchemeReview`): văn bản để duyệt, mỗi mục cạnh đoạn trích (mở đúng trang PDF); mục "chưa rõ" chặn `Xác nhận công thức`; xem trước tác động lên 3 sinh viên mẫu; `<ConfirmIrreversible>` nêu hậu quả bằng con số + `Idempotency-Key`; tải quy chế mới → bản nháp mới + diff. Chỉ TEACHER xác nhận | P6 L1, L4 | L | 2, 3 |
| 5 | **US-P6-05** Nhắc nhở: `GET …/grade-scheme/status`, banner (TEACHER / TA), cảnh báo trên `/attendance`, khoá `Chốt điểm` kèm lý do; job tuần gửi chuông khi chưa có `COURSE_POLICY` / còn DRAFT; Provider `GRADE_SCHEME_UNCONFIRMED` (70) cho "Hôm nay"; tool điểm khi chưa CONFIRMED trả "chưa có công thức chính thức" → agent nói rõ + nút "Nhờ giảng viên" (F3) | P6 L2 | S | 1 |
| 6 | **US-P6-06** `/gradebook` (thay mock): bảng kiểu bảng tính (Enter xuống, Tab sang), cột tên dính, ảo hoá 1.000 dòng, xem trước tức thì; 409 hiện giá trị của người kia và hỏi giữ bên nào; giải trình từng sinh viên; dòng ghi chú cố định "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường" | P6 L4 | L | 2 |
| 7 | **US-P6-07** Nhập điểm XLSX (ánh xạ cột → xem trước → báo dòng lỗi: MSSV không có trong lớp, điểm ngoài thang → ghi; idempotent theo (item, sinh viên)); xuất XLSX theo mẫu cột có dòng ghi chú (excelize) | P6 L3, L4 | M | 6 |
| 8 | **US-P6-08** `Chốt điểm` (TEACHER): snapshot + khoá + audit, `409` khi chưa CONFIRMED; `Mở khoá có lý do` → snapshot `version` mới + `unlock_reason`, bản cũ bất biến, sinh viên bị đổi điểm được báo | P6 L3, L4 | M | 6 |
| 9 | **US-P6-09** `/me` phần điểm: câu nhận định điểm hiện tại → `GradeCalculation` dạng phép tính tuyến tính → what-if tại chỗ `Nếu cuối kỳ được [ 8,0 ], điểm học phần sẽ là…`; nối dây `get_my_grade_summary`, `what_if_final_grade` (danh tính từ JWT); at-risk bật tín hiệu neo `GRADE` (nợ sprint 7) | P6 L4 | M | 2, 8 |
| 10 | **US-S8-SEED** Seed: lớp 1 có quy chế + công thức đã xác nhận + điểm (gồm điểm PE, phát biểu, XLSX giữa kỳ); **lớp 2 cố ý chưa có quy chế**; `seed/demo/quy-che-lop2.pdf` cố ý thiếu quy tắc làm tròn (DEMO_SCRIPT bước 6); `gradebook.spec.ts`; `gate-p6.sh` | – | M | 1–9 |
| 11 | **US-S8-UI** Nợ sprint 7 (#42): quét luật chữ trên UI ở `/settings/llm`, `/settings/integrations`, khối bảo mật `/settings` (không câu mô tả dưới tiêu đề) | nợ | S | – |

**Thứ tự cho dev:** 11 (khởi động, khi chờ spec) → 1 → 2 → 3 → 5 → 4 → 6 → 7 → 8 → 9 → 10.

**Spec do BA viết:** `docs/specs/FEAT-gradebook/` (US-P6-01…09, US-S8-SEED). Tech Lead thẩm định trước `APPROVED`. TC ở `docs/sprints/8/qc/`.

## Quyết định PM tự chốt (đổi được qua `proposals.md`)
- **Migration:** `00016_gradebook` (ARCHITECTURE đã dời số ở sprint 7, #25).
- **`seed/expected_final_grades.csv` là đáp án độc lập:** BA soạn bằng tính tay / bảng tính từ quy chế seed, **không** sinh bằng mã của dev; QC kiểm lại tay 3 dòng; dev không được sửa tệp này (luật "Cấm tuyệt đối" trong `AGENTS.md`). Lệch → sửa mã, hoặc DỪNG hỏi.
- **Quy chế seed:** dùng `seed/documents/Quyche.pdf` (đã nạp là `COURSE_POLICY` từ sprint 6) cho lớp 1; BA viết thêm `seed/demo/quy-che-lop2.pdf`. Chưa có quy chế môn học thật của chủ dự án (PROGRESS, việc chỉ chủ dự án làm được).
- **Xác nhận công thức và Chốt điểm** luôn là thao tác của TEACHER trên giao diện; không job, không agent nào làm thay.
- Mặc định mọi test dùng provider giả; trích công thức với LLM thật chỉ khi chủ dự án cho phép.

## Nợ đem vào
- **PE (5):** nối điểm bài thi vào sổ điểm (story 1).
- **Sprint 7:** neo `GRADE` của at-risk (story 9); luật chữ màn cũ (story 11).

## Cắt được nếu trễ
Không (ROADMAP: P6 không có mục cắt). Trễ thì báo chủ dự án.

## Rủi ro
- **Tính điểm là vùng nhạy cảm nhất của đồ án** (mục tiêu G5: 30/30 khớp bảng tính tay). Mọi thay đổi hành vi tính điểm không có trong phase file → DỪNG hỏi chủ dự án.
- **Trích công thức từ PDF tiếng Việt:** bảng / chú thích dễ đọc sai; vì vậy luôn có trích dẫn trang + mục "chưa rõ" chặn xác nhận.
- **Chủ dự án tự tính tay 3 sinh viên** là điều kiện cổng: cần anh dành ~20 phút cuối sprint.
