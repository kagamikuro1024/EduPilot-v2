# Lỗi hồi quy sprint 10.7

### DEF-R10.7-01 — Trang Quan sát quyết định lệch khung (teacher: meta lặp tên lớp, lớp rỗng tự dựng; admin: nhịp mục 24 ≠ 48)
- Mức: Trung bình
- Sprint gốc: 10.7 · TC gốc: audit H3, E7, P5 · Hồi quy do sprint sau?: CÓ (nghi sprint 11: khối «bảng theo cổng» của /observability, P-11-xx)
- Môi trường: ep-r, be0c0d96, teacher / admin, 1440 và 375
- Bước tái hiện: 1. `cd frontend && QC_REAL=1 QC_BASE=https://localhost:1143 QC_PW=… QC_PHASE=sau3 QC_ROLE=teacher QC_STATE=data pnpm exec playwright test -c playwright.qc-audit.config.ts` 2. `python3 docs/regression/evidence-10.7/scripts/audit_report.py sau3`
- Kỳ vọng: meta một dòng không chứa lớp đang chọn; lớp rỗng dùng EmptyState (`data-ep-empty`); mục→mục 48 · Thực tế: teacher/observability meta lặp lớp (4 lượt), 0 `data-ep-empty` ở lớp rỗng (2), admin/observability mục→mục [24,48,48,48]
- Bằng chứng: evidence-10.7/audit-sau3.md, audit/sau3/*.jpg · Nghi ngờ mã: `frontend/src/app/(app)/observability` (khối decision)
- Chạy lại lần 2: ổn định (sau3 và sau)

### DEF-R10.7-02 — Sổ điểm: đầu trang cao hơn 3,4 px và nút «Chốt điểm» mất khi đang tải / lỗi
- Mức: Thấp
- Sprint gốc: 10.7 · TC gốc: audit H7, X3 · Hồi quy do sprint sau?: CÓ (nghi nút «Chốt điểm» thêm ở sprint 8/9 sau khung 10.7)
- Môi trường: ep-r, be0c0d96, teacher (và ta cho H7), 1440 và 375
- Bước tái hiện: như DEF-01 với `/gradebook` (audit teacher data, loading, error)
- Kỳ vọng: cùng nút đầu trang ở cả ba trạng thái; chiều cao đầu trang một giá trị · Thực tế: có dữ liệu có «Chốt điểm», khi tải/lỗi không; 66,4 ≠ 63 (1440), 134,4 ≠ 131 (375)
- Bằng chứng: evidence-10.7/audit-sau3.md (H7, X3)
- Chạy lại lần 2: ổn định

### DEF-R10.7-03 — Luyện tập sinh viên: tab «Theo chủ đề / Thi thử» biến mất khi đang tải và khi lỗi (375)
- Mức: Thấp
- Sprint gốc: 10.7 · TC gốc: audit X3 · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, sv.kha, 375
- Bước tái hiện: audit student, state loading / error, route `/practice`
- Kỳ vọng: giữ nhãn tab · Thực tế: tab vs [] (2 lượt)
- Bằng chứng: evidence-10.7/audit-sau3.md (X3)
- Chạy lại lần 2: ổn định

### DEF-R10.7-04 — /settings/llm: «Máy chủ riêng (tương thích OpenAI)» lặp 7 lần
- Mức: Thấp
- Sprint gốc: 10.7 · TC gốc: audit T2 · Hồi quy do sprint sau?: KHÔNG RÕ (phụ thuộc số nhà cung cấp trong ep-r: nhiều bản QC đã được tạo)
- Môi trường: ep-r, be0c0d96, admin và teacher, 1440 và 375
- Bước tái hiện: audit admin/teacher data, route `/settings/llm`
- Kỳ vọng: chuỗi dài lặp < 3 lần · Thực tế: ×7
- Bằng chứng: evidence-10.7/audit-sau3.md (T2)
- Chạy lại lần 2: ổn định

Ghi nhận: lượt audit đầu có 45 lệch (nhiều route ta/* và student/chat) do chạy cùng 215 ca e2e nặng; chạy lại riêng cho ta còn 1 (H7), cho student/teacher/admin còn các mục trên. student/chat E2 (lệch 190 px) đã xếp «không áp dụng» ở bản gốc.
