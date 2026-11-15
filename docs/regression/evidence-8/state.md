# Trạng thái QC s8core (P6-01..04) — dừng theo lệnh PM
Đã ghi 131/187 ca vào `part-s8core.tsv` (P/F; gộp từ logs/s8core-results.jsonl bằng scripts/s8core-build-tsv.py; bản ghi cuối mỗi TC thắng).
Lớp riêng đang có (SQL/API): QD8A1 (bản sao sạch lớp 1 + công thức v3; id /tmp/s8core/QD8A1.id; đã chốt/mở khoá, đã confirm 3 bản), QD8A2 (quy chế lop1 READY + 1 tài liệu OTHER đang dở), QD8B1 (quy chế lop2 READY), QD8A3/A5/A6/A4 (tạo dở: prep bị dừng). DB phụ qcd8probe còn (drop database qcd8probe khi xong). Chưa đụng route LLM / khoá global.

## Ca CHƯA chạy (còn 56)
TC-P6-02-04 TC-P6-03-01 TC-P6-03-02 TC-P6-03-03 TC-P6-03-06 TC-P6-03-07 TC-P6-03-08 TC-P6-03-10 TC-P6-03-11 TC-P6-03-12 TC-P6-03-13 TC-P6-03-17 TC-P6-03-18 TC-P6-03-19 TC-P6-03-20 TC-P6-03-21 TC-P6-03-22 TC-P6-03-23 TC-P6-03-25 TC-P6-03-30 TC-P6-03-31 TC-P6-03-32 TC-P6-03-33 TC-P6-03-34 TC-P6-03-35 TC-P6-03-36 TC-P6-03-37 TC-P6-03-38 TC-P6-03-39 TC-P6-03-40 TC-P6-03-41 TC-P6-03-42 TC-P6-04-01 TC-P6-04-02 TC-P6-04-03 TC-P6-04-04 TC-P6-04-05 TC-P6-04-06 TC-P6-04-07 TC-P6-04-13 TC-P6-04-17 TC-P6-04-19 TC-P6-04-20 TC-P6-04-22 TC-P6-04-23 TC-P6-04-29 TC-P6-04-30 TC-P6-04-31 TC-P6-04-34 TC-P6-04-35 TC-P6-04-38 TC-P6-04-39 TC-P6-04-41 TC-P6-04-44 TC-P6-04-45 TC-P6-04-46

## Việc tiếp theo
1. Pha trích (cần khoá global: UTILITY→`QC fake`, hoàn lại 9router): TC-P6-03-01..03, 06..08, 10..13, 25, 32..33, 37..42 trên QD8A2 (lop1) / QD8B1 (lop2): scripts/s8core-prep.py đã dựng lớp+tài liệu; chưa viết s8core-px.py.
2. Pha stub OpenAI-compatible (bản ghi chỉnh): TC-P6-03-17..23, 30, 31, 34 (QD8A4 tài liệu dài >60000 ký tự), 35, 36, 39 (kill worker): cần provider stub + khoá global.
3. UI Playwright thật (1440/375, spec chưa viết): TC-P6-02-04, 04-01..07, 04-13(API, dễ), 04-17, 04-19, 04-20, 04-22, 04-23, 04-29..31, 04-34, 04-35, 04-38, 04-39, 04-41, 04-44..46. Dev e2e (13 test 'scheme …') đã xanh → đã ghi P cho 04-09/14/18/24/32/36/40/48; chạy `scripts/ui-antipatterns.sh` rc=0 (TC-P6-04-05 chưa ghi P, còn thiếu ca Playwright 'scheme review layout' — đã xanh).
4. TC-P6-01-33 (item EXAM lớp khác → 422 qua API; Go test TestExamFromOtherCourseRejected PASS + probe 11b 23503), TC-P6-02-43/44 (tham chiếu 26/27/23/24 — đã P: ghi P).
5. Ghi defects-part-s8core.md (đã tạo) và dọn: git worktree /tmp/qc-mut-s8core đã xoá.

## Phát hiện chính
- DEF-R8-s8core-01: giải trình thiếu dòng 'không dự → 0 theo quy chế' (TC-P6-01-20, 02-25).
- TC-P6-02-15 F thấp: internal/seedtool/examhistory.go:130 float64 cho điểm thi.
- Lớp 1 gốc lệch 8/30 so CSV KHÔNG phải do công thức (bản 1 v3 nguyên) mà do nhóm khác ghi đè: midterm 6 SV (sv04,05,06,07,09,10) nguồn QUIZ v2 lúc 12:41:45 (Forms import, sprint 10), + SPEAK 0,25 cho sv22, sv23 lúc 14:15/13:13. Lớp sao sạch QD8A1: 30/30 (TC-P6-02-03 P).
