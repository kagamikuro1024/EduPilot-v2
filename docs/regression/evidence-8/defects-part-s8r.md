
### DEF-R8-A1 — Trích công thức: đầu ra không phải JSON bị thử lại ×3 và báo «AI đang không dùng được» thay vì EXTRACTION_INVALID (TC-P6-03-36, TC-P6-03-19)
- Mức: Thấp (không mất dữ liệu; sai mã lỗi + tốn 3 lời gọi)
- Sprint gốc: 8 (4acd28b6, 2026-10-31, US-P6-03) · Hồi quy do sprint sau? KHÔNG
- Bước tái hiện: 1. Đặt route UTILITY sang nhà cung cấp OpenAI-compatible giả (s8r-stub.py, mode `invalid`: nội dung «đây không phải JSON {»). 2. POST /courses/{lớp}/grade-scheme/extract (tài liệu COURSE_POLICY READY). 3. Đợi job. (TC-19: mode `custom` với weight «abc».)
- Kỳ vọng: job FAILED `EXTRACTION_INVALID`, đúng 1 lời gọi (SRS 4.2.2; TC-P6-03-36). Thực tế: job FAILED `LLM_UNAVAILABLE` («AI đang không dùng được»); stub nhận 3 lời gọi, llm_audit 3 dòng error/BAD_RESPONSE. Weight «abc» (TC-19) cũng fail cả việc thay vì mục PARAM_INVALID.
- Nguyên nhân [INFERENCE từ mã]: gateway coi đầu ra sai schema là BAD_RESPONSE → ErrAllProvidersFailed; backend-go/internal/grading/extract.go callLLM (dòng ~273) xếp là lỗi TẠM nên thử lại; EXTRACTION_INVALID chỉ ra khi ErrBadRequest hoặc buildDraft lỗi.
- Bằng chứng: evidence-8/logs/s8r-03c-run.log, evidence-8/logs/s8r-03b-run.log.

### DEF-R8-A2 — Đóng Drawer / Dialog làm focus rơi về <body> thay vì nút mở (TC-P6-04-03, TC-P6-07-47)
- Mức: Thấp (trợ năng bàn phím)
- Sprint gốc: 8 (US-P6-04) · Hồi quy do sprint sau? KHÔNG RÕ
- Bước tái hiện: /gradebook/scheme (bản nháp có trích dẫn), cửa sổ 1023 px, Tab tới nút «Trang N» → Enter (Drawer mở) → Esc.
- Kỳ vọng: focus về nút «Trang N» (SRS: trả focus đúng nút). Thực tế: document.activeElement = BODY. Drawer đóng đúng, cột cố định ở 1024 px đúng.
- Cùng lỗi ở TC-P6-07-47: đóng hộp thoại Nhập XLSX sau khi ghi → focus về <body> (SRS: «trả focus về nút menu»).
- Bằng chứng: evidence-8/logs/s8r-real04-run.log (real-04.spec.ts, test 04-03r), real-07.spec.ts (test 07-46,47,48).

### Ghi nhận (không tính F): TC-P6-03-31 — tên không thuộc danh sách lớp không bị che
- PDF quy chế chứa «Nguyễn Văn Test 20229999 (nvtest@example.com)» (tên giả KHÔNG thuộc lớp): prompt tới LLM che MSSV và email nhưng giữ nguyên tên. Với tên/MSSV/email của sinh viên TRONG lớp: che hết ([[SV_1]] [[MSSV_1]] [[EMAIL_1]]). Cơ chế che theo danh sách lớp (internal/privacy/roster.go); TC dùng tên ngoài danh sách. TC chấm P theo kịch bản lớp thật. PM cân nhắc có cần che tên lạ (cần NER) hay không.

### DEF-R8-A3 — Trích công thức: trọng số ngoài [0;100] bị đổi thầm thành 0, không có mục PARAM_INVALID (TC-P6-03-19)
- Mức: Trung bình (công thức sai im lặng; chỉ còn người xác nhận nhìn thấy 0 %)
- Sprint gốc: 8 (4acd28b6, US-P6-03) · Hồi quy do sprint sau? KHÔNG
- Bước tái hiện: stub OpenAI-compatible (s8r-stub.py mode custom, evidence-8/scripts/s8r-03h.py) trả extraction-lop1 với midterm 40, weekly «150», final 60 (hợp pattern số của schema); trích; GET /courses/{lớp}/grade-scheme.
- Kỳ vọng (SRS dòng 158): mục PARAM_INVALID ở components.weekly (weight ∉ [0;100]) và nháp không sẵn sàng xác nhận. Thực tế: weekly = 0, tổng 100, unclear_items = [] → DRAFT_READY.
- Nguyên nhân [INFERENCE từ mã]: backend-go/internal/grading/extract.go buildDraft (dòng ~348-352) đặt w = 0 và đẩy PARAM_INVALID vào d.static; GET tính lại mục CODE từ dữ liệu đã lưu (grade/scheme.go ~116) nên mục static biến mất.
- Bằng chứng: evidence-8/scripts/s8r-03h.py (SNEAKY), logs/s8r-03e-run.log.

### DEF-R8-A4 — Chat riêng: «Cô tính điểm môn này thế nào?» không được nhận là GRADE_FORMULA (TC-P6-05-37)
- Mức: Thấp (câu trả lời vẫn an toàn, không số bịa, nhưng sai nhánh: không câu mẫu «Lớp chưa có công thức…», mở ticket AUTO_LOW_CONFIDENCE thay vì AUTO_NO_FORMULA)
- Sprint gốc: 3 (4e691f2d, US-P3-04 — luật reFormula) áp dụng ở US-P6-05 AC7 · Hồi quy do sprint sau? KHÔNG
- Bước tái hiện: SV lớp chưa có công thức (vd QD8H7, hoặc lớp mới chưa quy chế) → chat riêng → hỏi đúng câu «Cô tính điểm môn này thế nào?».
- Kỳ vọng (TC-P6-05-37, US-P6-05 AC7): câu mẫu «Lớp chưa có công thức điểm chính thức do giảng viên xác nhận.», 0 lời gọi LLM, ticket AUTO_NO_FORMULA. Thực tế: «Mình chưa tìm thấy nội dung này trong tài liệu của lớp.» + ticket AUTO_LOW_CONFIDENCE. Câu «Công thức tính điểm học phần là gì?» thì đúng (câu mẫu + AUTO_NO_FORMULA).
- Nguyên nhân: backend-go/internal/agent/intent.go:24 reFormula (đã gấp dấu) chỉ khớp «tinh diem nhu the nao», «cach tinh diem»…; «tinh diem mon nay the nao» không khớp.
- Bằng chứng: evidence-8/scripts/s8r-chat05.py, part-s8r.tsv.
- Quan sát thêm (không F): hỏi lại cùng câu GRADE_FORMULA trong cùng hội thoại mở thêm ticket AUTO_NO_FORMULA thứ hai (mỗi tin một ticket; TC 05-38 «QC ghi nhận thực tế»).

### DEF-R8-A5 — Xoá tài liệu nguồn của công thức đã xác nhận: UI không hiện «Tài liệu nguồn đã gỡ» (TC-P6-05-42)
- Mức: Thấp (cosmetic; số liệu đúng)
- Sprint gốc: 8 (US-P6-05 AC8) · Hồi quy do sprint sau? KHÔNG RÕ
- Bước tái hiện: lớp có công thức CONFIRMED với tài liệu nguồn (QD8H3), GV xoá tài liệu (DELETE /courses/{id}/documents/{doc} → 204), mở /gradebook/scheme, bấm «Trang 1».
- Kỳ vọng (SRS FEAT-gradebook dòng 96, US AC8): trích dẫn hiện «Tài liệu nguồn đã gỡ». Thực tế: cột trích dẫn hiện «Quy chế lớp 2 · Trang 1 · <đoạn trích>» (source_title sao chép) không có chữ «đã gỡ»; không lỗi.
- Bằng chứng: frontend/e2e/qc2-r8/real-05.spec.ts, part-s8r.tsv; mã frontend/src/features/gradebook/CitationPane.tsx (không nhánh docId rỗng).
