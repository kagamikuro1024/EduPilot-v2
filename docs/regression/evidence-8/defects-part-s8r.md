
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
