# Smoke hồi quy sprint 8 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm D, SmkD) · Ngày: 2026-11-15
Tổng TC cũ: 512 · Đạt: 0 · Trượt: 0 · Không chạy (K): 512   (P+F+K = T). **K loại a/b/c = 0; chưa chạy (mẫu) = 512** — sprint 8 CHƯA được đi (hết ngân sách lượt của tác vụ sau khi dành toàn bộ cho sprint 9 ưu tiên + sự cố môi trường ep-r: Postgres crash-recovery 19:59, khoá đăng nhập IP, tranh khoá route LLM).

## Bằng chứng một phần (không gán cho TC cụ thể, chưa ánh xạ sang mã TC)
- `go test -race ./internal/grade/...` (G6-01): đã chạy xong — 134 PASS, 0 FAIL; log `docs/regression/evidence-8/logs/go-grade.log`.
- `go test ./internal/grading -run TestExtractGradeScheme` (G6-03): --- PASS: TestExtractGradeScheme (1.04s); ok  	github.com/edupilot/backend-go/internal/grading	1.793s; rc=0
- `p6-hand-calc.py` (G6-12): 30/30 khớp expected_final_grades.csv, 18/18 khớp cases-tong-hop.csv, rc=0 (độc lập với mã dev).
- `go test ./internal/grade` (TestEngineExpected30, TestEngineCases, TestNoFloatInGrade, TestForbiddenImports, TestConfirmOnlyFromHandler): PASS cả 5 (`go-grade-core.log`); bản `-race` đủ gói chạm timeout 10 phút dưới tải (134 PASS, 0 FAIL, rc=1 do timeout, không do test đỏ).
- `p6-preview-vs-csv.py --by-account` trên lớp 1: 23/30 khớp, 7 lệch (vd sv10 qt 8,5 vs 6,0; sv23 hp_raw 4,1 vs 4,0) — CHƯA xác minh: có thể sổ lớp 1 đã đổi công thức/bản 2 do nhóm khác hoặc seed v2; xem DEF-R8-01.
- `p6-rbac.sh` trên lớp riêng QD8C/QD8B: 29 PASS, FAIL do 401/429/502/503 thoáng qua của ep-r (không kết luận), cần chạy lại khi stack ổn định.
- Script sẵn dùng: `docs/sprints/8/qc/scripts/p6-*` (hand-calc, lop2, rbac, preview-vs-csv…) — chưa chạy trên ep-r.

## Tổng hợp theo US
| US | TC tổng | P | F | K | Ghi chú |
|---|---|---|---|---|---|
| P6-01 | 41 | 0 | 0 | 41 | MẪU: chưa chạy |
| P6-02 | 49 | 0 | 0 | 49 | MẪU: chưa chạy |
| P6-03 | 48 | 0 | 0 | 48 | MẪU: chưa chạy |
| P6-04 | 49 | 0 | 0 | 49 | MẪU: chưa chạy |
| P6-05 | 46 | 0 | 0 | 46 | MẪU: chưa chạy |
| P6-06 | 63 | 0 | 0 | 63 | MẪU: chưa chạy |
| P6-07 | 59 | 0 | 0 | 59 | MẪU: chưa chạy |
| P6-08 | 47 | 0 | 0 | 47 | MẪU: chưa chạy |
| P6-09 | 50 | 0 | 0 | 50 | MẪU: chưa chạy |
| P6-SEED | 37 | 0 | 0 | 37 | MẪU: chưa chạy |
| S8-UI | 23 | 0 | 0 | 23 | MẪU: chưa chạy |

## Kết quả từng ca
| TC | KQ | Ghi chú |
|---|---|---|
| TC-P6-01-01…TC-S8-UI-23 | K | MẪU: chưa chạy — sprint 8 chưa được bắt đầu (cần chạy lại nguyên sprint; không phải K a/b/c) |

## Trượt: bước tái hiện
Không có ca nào đã chạy.

## Ca K và lý do
- Toàn bộ 512 ca: chưa chạy (mẫu). Cần lớp riêng cho ca phá huỷ sổ điểm (QD8*), `p6-env.sh` trỏ ep-r.
