### DEF-R8-01 — Xem trước sổ điểm lớp 1 lệch bảng chuẩn ở 7/30 SV (chưa xác minh)
- Mức: Cao (nếu là lỗi tính điểm) · Sprint gốc: 8 · TC gốc: TC-P6-SEED/G6-12 (preview vs expected_final_grades.csv) · Hồi quy do sprint sau?: KHÔNG RÕ (nghi sprint 9 `v2` — công thức lớp 1 bản 2 có 4 thành phần — hoặc nhóm khác sửa điểm/mục bài tập ở lớp 1)
- Môi trường: ep-r, be0c0d96, TEACHER, `GET /courses/{lớp 1}/gradebook/preview?limit=100`
- Bước tái hiện: `python3 docs/sprints/8/qc/scripts/p6-preview-vs-csv.py --by-account` (đã đổi container sang ep-r-postgres-1) sau khi lưu preview vào /tmp/pv.json
- Kỳ vọng: 30/30 khớp · Thực tế: 23/30 (sv10 qt 8,5 vs 6,0; sv23 hp_raw 4,1 vs 4,0 …). `p6-hand-calc.py` 30/30 và `go test ./internal/grade` xanh nên engine đúng; nghi dữ liệu lớp 1 đã đổi.
- Bằng chứng: `docs/regression/evidence-8/logs/preview-vs-csv.log` · Chạy lại lần 2: ổn định (2 lần)
