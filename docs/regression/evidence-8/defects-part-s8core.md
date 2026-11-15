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
