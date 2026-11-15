# Trạng thái slice C_s6_p3b (SmkE.SliceC) — dừng theo lệnh PM

Kết quả đã ghi: `/tmp/qc2e/res/C_s6_p3b.jsonl` (34 dòng: 30 P, 4 T; có thể trùng id — giữ dòng cuối). Thư mục làm việc: `/tmp/qc2e/C/` (courses.json = lớp riêng C6C1/C6C2, SV sv31–sv38; docs.json, rem2.json, exams.json, q.json).

## Đã xong (đã rec)
- P3-07: 12,13,14,15,20,21,22,23,25,27
- P3-08: 18,23,24,25,26(T, P-6-10),28
- P8-03: 01,02,03 (T, #13/#sprint9), 05,08,14,15,16,34,35,36(cần kiểm lại dòng cuối trong log p8r2),37,38,39,40,41,43,60

## Đã chạy nhưng CHƯA rec (kết quả ở /tmp/qc2e/C/gores.json, gocal.log)
- Go test pass: P3-05-18,21(2 pass),26(2 pass; 5 tên, kiểm đủ tên),42,51,54,56; P3-06-43 ; P3-06-46; P3-07-05,06; P3-07-10(?) — đọc gores.json để chốt; calendar TestReminder* rc=0 (dùng cho P8-03-35/42).
- P3-07-16/17/18/19/24/26: E1 chưa chạy được vì stack 503/429 (e1loop.sh bị kill); lệnh: E1_TOKEN=<sv.kha> python3 benchmarks/eval_pii.py --out-dir /tmp/qc2e/C/r1 (API_URL=https://localhost:1143/api/v1). 07-18 dùng --min-recall 0.99 để ép không đạt.

## Chưa làm (tiếp theo)
1. Upload docs xong? (c0.py bị kill: ds trong docs.json nếu đã in) → lock thi: c1_lock.py (P3-05-22..25, P3-06-40..42, 05-27): tạo bài thi mở sau 30 s, sv33 vào thi.
2. Chat thật (không cần fake): 05-33, 35, 37, 38, 40, 41, 54, 55, 57; threads 06-06,09,11,13,14,22,34,46; 07-02..11.
3. Cần khoá global + `_test/llm/fake`: 05-02,05,07,09,11,13,14,17,18,19,20,21,36,56; 07-03,04.
4. UI Playwright (https://localhost:1143): 05-09,14,17..19,27,35,44,50,52,53; 06-07,08,09,11,13,34,36,37,38,42; 07-07,08,09; 08-11,29; P8-03-11,22,44..49.
5. P8-03: 17,24,28(K: cần URL công khai/Google),30–33 (chat lịch), 42 (1000 SV), 55 (dừng Mailpit), 59, 50/51; P3-08: 01–06,10–12,14–17,19–21,27,29.
6. Cuối: dedupe jsonl (giữ dòng cuối), ghi /tmp/qc2e/defects/C_s6_p3b.md (chưa có lỗi nào; ghi nhận: reminder_source 4 giá trị, template mail `reminder`, API settings `reminders` — đều drift hợp lệ, không phải lỗi).

## Việc dọn
- C6C2 đang ARCHIVED (admin archive); sv38 status DISABLED; sv36 enrollment PENDING; sv34 email_verified_at=null; mailpit chạy bình thường. Bỏ qua/khôi phục nếu cần.
