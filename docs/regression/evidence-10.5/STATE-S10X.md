# STATE S10X (qc2, 2026-11-21)
Đã xong (ghi smoke-10.5.md / smoke-10.6.md / results-S10X.tsv): TC-G07-12 F (DEF-R106S-16), TC-G10-13 P, TC-D08-14 F (DEF-R105S-02), TC-D04-11 N.
Dữ liệu tôi tạo trên ep-r: lớp QD105A (mk105.py), enrollments TEACHER/TA lớp QCEMPTYr107 bị đặt created_at=now() để lên đầu danh sách /me/courses (picker chỉ ~100 lớp đầu); QR91A: không còn ca hồi quy / bài nộp được khôi phục.

## CÒN LẠI (ep-r rebuild lúc 20:40; chờ «ep-r lên lại»)
1. Audit 10.7 (DEF-R10.7-01..04, DEF-R10.8-02 đã P): `rm -rf evidence-10.7/audit/rr-defrerun evidence-defrerun/R10.7/*.log; (cd evidence-defrerun && nohup bash run-r107.sh &)`. Lượt trước empty-teacher 2 failed vì picker không thấy lớp QCEMPTYr107 (không phải lỗi sản phẩm; đã sửa dữ liệu). Sau ALLDONE: đọc evidence-10.7/audit-rr-defrerun.md.
2. TC-D01-07: `cd evidence-10.5/s10x && SEED_DEFAULT_PASSWORD=… /tmp/qc2-pw.sh python3 d0107.py` (a: 200 lời gọi nhịp 1,6/s; b: lỗi 0,5 chạy tới đủ 200 lời gọi thực; c: mặc định 426/682). ≤ 8 phút/lượt: tách phần nếu cần.
3. TC-D04-09 / D04-01: `python3 d0409.py p1` (12 bài + chấm, cờ bật), rồi `d0409.py always_yes`, `always_no`, `error`, rồi `d0409.py fin` — mỗi lượt qua khoá. Ghi vào results-s10x.jsonl.

## Cập nhật 21:30
- D04-09 P (ghi smoke-10.5.md). D01-07 N: (c) P, (a) p50/p95 lệch do tải máy (4–10) / nhóm khác dùng chung `_test/llm/fake`, (b) 0,24 (nghi bản giả bị nhóm khác đổi); cần ep-r riêng hoặc khoá cả bản giả + máy rảnh; chạy: `cd evidence-10.5/s10x; SEED_DEFAULT_PASSWORD=… /tmp/qc2-pw.sh timeout 470 python3 d0107.py a|b|c`.
- Lỗi script cũ (b3_lat2.py): `time.sleep(i/rate)` tính từ lúc luồng bắt đầu nên 200 bài mất ~35 phút; bản s10x/d0107.py sửa bằng mốc T0.
- D04-01: lượt 1 (điều khiển bằng route thử) KHÔNG hợp lệ: tầng decision của cờ chèn lệnh chạy ở WORKER, route `_test/llm/fake` chỉ đổi bản giả của gateway. Làm lại bằng `LLM_DECISION_FAKE_MODE` của worker: `run-s10x3.sh` (always_yes, always_no, error, fin; mỗi lượt restart worker rồi khôi phục env rỗng) — xem run-s10x3.log và dòng TC-D04-01 cuối results-s10x.jsonl. Nếu bị kill giữa lượt: kiểm worker `docker inspect ep-r-worker-1 | grep FAKE_MODE` phải rỗng, nếu không chạy `source docs/regression/env/ep-r-env.sh; $EPR_C up -d --no-deps --force-recreate worker`.
- Audit 10.7: `run-s10x2.sh` đang chạy run-r107.sh nền (log evidence-defrerun/R10.7/all.log). Xong → evidence-10.7/audit-rr-defrerun.md, rồi ghi P/F DEF-R10.7-01..04 vào results-S10X.tsv + defects-closed.md.
- Việc 10.6 (G07-12 F DEF-R106S-16, G10-13 P) đã chuyển Main: evidence-10.6/s10x/KETLUAN.md.
