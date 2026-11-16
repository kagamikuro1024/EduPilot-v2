### DEF-R8-s8core-01 — Giải trình không có dòng «không dự → 0 theo quy chế»
- Mức: Trung bình
- Sprint gốc: 8 · TC gốc: TC-P6-01-20, TC-P6-02-25 · Hồi quy do sprint sau? KHÔNG (engine.go chưa từng sinh chữ này từ 859ca52d; báo cáo gốc PASS chỉ dựa test Go không kiểm chữ)
- Bước tái hiện: GET /courses/{lớp}/gradebook/students/{sv.nguyco}/explain (lớp 761987 hoặc QD8A1), xem dòng «Kiểm tra hằng tuần (20%)».
- Kỳ vọng: dòng 0 với giải trình «không dự → 0 theo quy chế» + source_quote điều 1.4 (SRS 4.3.3, US-P6-02 AC7). Thực tế: expr «0», value «0», filled_by_user=true, không quote, không chữ «không dự».
- Bằng chứng: docs/regression/evidence-8/logs/s8core-a1-p1.log, /tmp tạo từ a1cache (script scripts/s8core-a1-p1.py).
- Nghi ngờ mã: backend-go/internal/grade/engine.go:198-204 (nhánh closed && zero[c.Key] chỉ thêm 0 vào scores, không thêm Line/Text; comment engine.go:251).

### DEF-R8-s8core-02 — float64 chuyển điểm thi trong seed tool
- Mức: Thấp
- Sprint gốc: 8 (luật 5) · TC gốc: TC-P6-02-15 · Hồi quy do sprint sau? CÓ (commit 2e041190, sprint 11 đợt 2)
- Bước tái hiện: grep -rln 'grade\|score' backend-go/internal --include=*.go | xargs grep -n float64 (loại các từ khoá không liên quan).
- Kỳ vọng: không float64 chạm điểm. Thực tế: backend-go/internal/seedtool/examhistory.go:130 `float64(c)*2.5` ghi điểm thi (đúng chính xác, chỉ ở công cụ seed; mã sản phẩm sạch).
- Bằng chứng: docs/regression/evidence-8/logs/s8core-static.log (mục 02-15).

### Ghi chú điều tra (không phải DEF): lớp 761987 lệch 8/30 so expected_final_grades.csv
Công thức không đổi (1 bản CONFIRMED v3, scheme_no 1). Nguyên nhân là dữ liệu bị nhóm QC/sprint khác ghi đè trên lớp seed: midterm của sv04,05,06,07,09,10 = 0,2,4,6,8,10 nguồn QUIZ v2 (nhập Forms, 12:41:45 hôm nay); điểm cộng SPEAK 0,25 thêm cho sv22 (14:15) và sv23 (13:13). Tính tay Fraction từ đầu vào thật khớp preview 30/30 (logs/s8core-drift.log); bản sao sạch QD8A1 khớp CSV 30/30.



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
