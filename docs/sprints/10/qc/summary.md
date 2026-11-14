# QC sprint 10 (FEAT-practice, P9) — kết luận pha 2

2026-11-14. Bản đầu ở `57f4cb25`; **kiểm lại sau các commit «sửa QC»** (L1 `ac5d41b3`, L3 `110ea0de`, L4 `4f256b30`, L7–L16 `eb1ac4f6`…`9f062c23`, #3 `ffbee0a3`) trên stack dựng lại từ trống ở HEAD. Nhánh `sprint/10-practice`, stack `ep-qc` thật (MacBook Apple Silicon, Colima 6 CPU / 10 GB, provider LLM `fake`, docling thật). k6 / Lighthouse / axe: **hoãn theo chủ dự án**. Chi tiết, bước tái hiện, bằng chứng: `report-stack.md`; trạng thái từng TC: `tc-status-pha2.md`; ảnh: `shots/`; script: `scripts/`; đề xuất: `../proposals.md` P-10-9.

## Kết luận theo story (cập nhật sau kiểm lại)
«Một AC FAIL → story FAIL» (`QC.md`). Chỉ tính phạm vi ưu tiên đã chạy (AC chính, phân quyền, nhánh lỗi, API + màn 1440 / 375); TC chưa chạy ở `tc-status-pha2.md`. L6 là Nợ chấp nhận (P-10-10), không tính FAIL.

| Story | Kết luận | Ghi chú |
| --- | --- | --- |
| US-P9-01 Lược đồ, Quiz Engine, DTO | **PASS** | Không đổi. Chưa chạy `goose down/up` |
| US-P9-02 Trích đề cũ | **PASS** (phạm vi ưu tiên) | **L1 đã sửa**: seed sạch trích 23 câu `EXTRACTED` từ đề docling thật, đáp án khớp bảng 23/23, mọi câu có trích dẫn `{document_id, page}`; câu 24 bỏ vì trùng stem với câu `PENDING` đã có (`DUPLICATE`, đúng AC5). Đề lỗi câu: INVALID ở 2, 3, 4, NO_ANSWER ở 5. Đề lỗi schema: job `FAILED` `LLM_UNAVAILABLE`, không lưu câu (TC-02-07b PASS). Còn mở: TC-02-14 (`OVERLOADED`) không kiểm được, chờ PM |
| US-P9-03 Sinh câu | **PASS** | L9 đã sửa (`types:[]` → 422) |
| US-P9-04 `/questions` + duyệt hàng loạt | **PASS** | L10, L11, L16 đã sửa: bulk trả `CITATION_REQUIRED`; menu «Soạn tay» → «Câu trắc nghiệm» / «Bài lập trình»; nhãn nguồn «Đề mẫu luyện tập — Mạng máy tính, trang 1» |
| US-P9-05 Dựng đề theo ma trận | **PASS** (có Nợ L6) | L7 đã sửa (409 `{available:0}`), L8 đã sửa (ma trận không `difficulty` bù đủ 10 câu). L6 `count=0` → 201: Nợ sprint 11 |
| US-P9-06 Luyện theo chủ đề | **PASS** | L15 đã sửa (một dòng «Giảng viên chưa mở câu hỏi»; vẫn còn liên kết «Lịch sử luyện tập»). Trích dẫn trong phản hồi kiểm được khi câu có citations (câu `EXTRACTED` đã duyệt: 1 trích dẫn); mức phủ citations của seed xem P9-12 |
| US-P9-07 Thi thử | **PASS** | Không đổi |
| US-P9-08 Tự luận + nhận xét | **PASS** | Không đổi |
| US-P9-09 Lịch sử, điểm yếu, Hôm nay | **PASS** | L3 đã sửa (0,5 s) |
| US-P9-10 Liêm chính | **PASS** | Không đổi |
| US-P9-11 Nhập Forms | **PASS** (phạm vi ưu tiên) | L12 đã sửa: ánh xạ theo tên tiêu đề (3/3 khớp), chữ lỗi «Chỉ nhận tệp Excel (.xlsx) hoặc CSV.», «tối đa 5 MB». Chưa đổi: ô chọn tệp mặc định (thẩm mỹ) |
| US-P9-12 Seed + cổng | **FAIL** | L4 sửa gần hết: có câu `EXTRACTED` (23) và `GENERATED` (4), Quiz `PUBLISHED`, lớp 761988 có 20 câu, `check-practice-seed.mjs` OK; seed sạch exit 0, 504 s. **Còn lại:** (a) chỉ 3 / 107 câu `APPROVED` có citations (spec: «có `citations` thật»; 104 câu `MANUAL` không có) — cần PM / BA nói AC chỉ áp cho câu AI hay cho cả 80; (b) 1 stem trùng giữa lớp 761987 và 761988 («Mô tả nào đúng về MFA?»), spec: lớp 2 không chứa câu lớp 1. k6 / Lighthouse / axe hoãn |

PASS: 11 / 12. FAIL: 1 / 12 (P9-12).

## Lỗi theo mức
| Mức | Số | Mã |
| --- | --- | --- |
| Cao | 0 | L1 đã sửa (kiểm lại PASS) |
| Trung bình | 0 mở | L3, L5 đã sửa; L4 còn hai điểm Thấp ở P9-12 |
| Thấp | 2 mở + 1 Nợ | L4(a) citations 3/107, L4(b) trùng 1 stem lớp 2; L6 Nợ sprint 11 (P-10-10); L13, L14 ghi nhận; còn lại đã sửa |
| Nghiêm trọng | 0 | |

## Phát hiện về quy trình (cần PM)
1. Dev sửa test có sẵn / đổi nhãn để xanh CI (`ac2a7aef`: `panels.spec.ts` +4 dòng, nhãn menu trả về «Câu trắc nghiệm» làm TC-04-02 đỏ). Không thấy sửa contract test, `expected_final_grades.csv`, `t.Skip` hay nới assertion; `today/staff_test.go` 6 → 7 có P-10-8 F1. Xin PM quyết P-10-9.
2. Công thức `OVERLOADED` ở handoff không làm đầy hàng cho việc nền (TC-02-14, 03-08 không kiểm được) và `llm_audit` không có payload (6 TC kiểm PII ở lời gọi LLM không kiểm được): cần dev chỉ cách.
3. Môi trường (đã sửa ở `ffbee0a3`, QC kiểm lại: presign trỏ cổng 9500 khi offset): `.env.local` mặc định đặt `BLOB_PUBLIC_ENDPOINT=localhost:9000` nên stack có offset PUT tệp lên MinIO của stack khác (`ep-qc` ghi nhầm vào stack demo). Đã sửa cục bộ (`9500`, không commit). Stack demo `edupilot` có thể có vài đối tượng thừa ở bucket do lượt đầu của QC (UPLOAD_INCOMPLETE).

## Seed sạch (TC-12-01…06)
Dựng lại từ trống (`down -v` → `pnpm dev` với `SEED_ON_EMPTY_DB=true`, `57f4cb25`): `scripts/seed.mjs` **mã 0, 334 s** (kể cả docling); chạy lại **mã 0, 21 s**, `question_bank` 129 / `practice_attempts` 42 / `exams` 3 không đổi (idempotent, TC-12-06 PASS). Nội dung: 104 `APPROVED` (14 chủ đề; 60 một đáp án, 12 nhiều đáp án, 20 đúng–sai, 8 trả lời ngắn, 2 tự luận, 2 code; EASY 39, MEDIUM 51, HARD 14), 20 `PENDING` + 5 `AI_DRAFT`, 42 lượt của 13 SV (1 lượt dở). Sai spec: 0 câu có citations, không có `EXTRACTED`, Quiz `SCHEDULED`, lớp 2 không có câu (L4); `node scripts/check-practice-seed.mjs` ở `57f4cb25` lỗi `Không thấy lớp 761987` (L5), sau bản sửa `5d9c54e3` báo `OK: 107 APPROVED, 26 PENDING, Quiz SCHEDULED, 23 lượt của 6 sinh viên mẫu`.

## Đang dở (tạm dừng theo chủ dự án, 2026-11-14)
Chưa chạy lại: TC-02-14 / 03-08 (`OVERLOADED`, chờ PM), 4 TC PII ở mức lời nhắc (02-06, 02-16, 03-06 chưa có test Go). Stack `ep-qc` còn chạy ở HEAD (dữ liệu seed + dữ liệu thử của QC); gỡ bằng `docker compose -p ep-qc down -v` khi không cần. Script kiểm lại: `qc/scripts/recheck_*.py`.
