### DEF-R4-01 — Mã lớp phân biệt hoa/thường: tạo được hai lớp `SK20D91` và `sk20d91`
- Mức: Trung bình
- Sprint gốc: 4 · TC gốc: TC-P208-02 (AC1: `class_code` trùng khác hoa-thường → 409) · Hồi quy do sprint sau?: KHÔNG (có sẵn: `normClass` không đổi chữ từ US-P2-08 commit 3651adab; index `courses_class_code_key` không `lower()`; báo cáo gốc ghi PASS nhưng không tái hiện)
- Môi trường: ep-r, be0c0d96, ADMIN
- Bước tái hiện: 1. `POST /admin/courses` `{"subject_code":"INT1006","class_code":"SK20D91","name":"x","semester":"2026-2027-HK1"}` → 201; 2. lặp với `class_code":"sk20d91"` (key khác) → 201 (kỳ vọng 409 `details.field=class_code`). `select class_code from courses where class_code ilike 'sk20d91'` → 2 dòng.
- Kỳ vọng: 409 CONFLICT · Thực tế: 201, hai lớp có mã chỉ khác hoa/thường (SV nhập mã lớp dễ nhầm; tìm lớp theo mã không rõ ràng)
- Bằng chứng: `evidence-4/p2c.out` (TC-P208-02) · Nghi ngờ mã: `backend-go/internal/course/admin.go:114-120` (`normClass`), `backend-go/db/migrations/00003_course_foundation.sql:37,49`
- Chạy lại lần 2: ổn định (tạo lại với mã khác: cùng kết quả). Đã lưu trữ cả hai lớp QC sau khi đo.

### DEF-R4-02 — Import roster nhận tệp `.xlsx` thực chất là văn bản (TC-P210-03)
- Mức: Thấp
- Sprint gốc: 4 · TC gốc: TC-P210-03 (AC1/AC13 magic bytes) · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, GV của lớp
- Bước tái hiện: tạo `text.xlsx` bằng `gen.py` (nội dung CSV, đuôi .xlsx); `POST /courses/{id}/roster/import?dry_run=true` multipart `file=@text.xlsx` + `Idempotency-Key` → 200 (đọc như CSV). `fake-ext.csv` (ELF), `png-as.csv`, `cut.xlsx` → 422 UNSUPPORTED_FILE đúng.
- Kỳ vọng: 422 UNSUPPORTED_FILE (đuôi/ nội dung không khớp magic bytes) · Thực tế: 200
- Bằng chứng: `evidence-4/p2h.out` · Nghi ngờ mã: bộ đọc roster ở `backend-go/internal/course/` (nhận diện theo nội dung, bỏ qua đuôi)
- Chạy lại lần 2: ổn định (không có hại: dữ liệu vẫn qua kiểm tra dòng)
