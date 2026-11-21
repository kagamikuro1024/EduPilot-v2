# STATE S9N (2026-11-21) — 9.1 / 9.5 / DEF-R10.7
## Kết quả
- smoke-9.1.md: tổng tính 361 (LT 53): P 352 · F 1 (TC-UX12-2, lượt cũ; DEF-R9.1-06 đã sửa chưa chạy lại) · N 8. Đã P lượt này: UX01-8, UX03-11, UX04-3, UX08-5, UX09-16, UX13-5/6/9/10/11/12/13/15/16/17, UX16-10, UX16-14, UX17-8/9; LT có nguồn: UX13-1 (3b259963 + P-107-15), UX08-28 (US-08 AC7 sửa 10.7).
- smoke-9.5.md: 142 TC: P 136 · F 5 · N 1 (TC-N07-09). Mới P: N03-10/12, N06-06/07/08, N07-04, N08-16/17, N09-01/07/11, N07-10, N04-12, N05-02; F mới: N07-02 (DEF-R9.5-25). DEF-R9.5-24 = P (results-21nov.tsv). DEF-R9.5-23: tài liệu + rerun (không chạy thêm T1).
## CÒN LẠI 9.1 (N) và cách chạy
Cần khoá Playwright (rất tranh chấp; chờ hàng chục phút). Chạy: `cp evidence-9.1/scripts/pwlock.sh /tmp/smks/pwlock.sh; cp env/ep-r-lib.sh /tmp/ep-r-lib.sh; QC_TAG=<chữ> PWL_MAX=470 PWL_WAIT=10800 /tmp/smks/pwlock.sh bash evidence-9.1/run8.sh <spec> <lớp> <grep>` rồi `cd evidence-9.1 && python3 make_overrides7.py && python3 build_smoke91.py && python3 build_smoke91_md.py`.
- UX18-4/6/8/PQ: `qc-uiux91-ru1-real - TC-UX18-4` (đã sửa: màu đường ray đọc ở ::before). Nền: master3.sh (cùng r91sv UX13-7/8 `qc-uiux91-r91sv-real 761987 TC-UX13-6`, tag y).
- UX13-7/8: spec đã sửa theo AC3 (DELETE ngay + POST restore) — xem defects-9.1.md mục SmkS4; kết quả thật chưa có.
- UX12-9/10: `qc-uiux91-rw4-real QR91A TC-UX12-9/10`. UX12-2 (DEF-R9.1-06): `qc-uiux91-ru1-real - TC-UX12-2`. UX16-14 làm mới (tuỳ chọn): `qc-uiux91-rn4-real QR91H UX16-14`. (master1.sh giữ các dòng này nếu còn sống.)
## CÒN LẠI 9.5
- N07-09 Lighthouse: `bash evidence-9.5/scripts/master-lh.sh` (chờ khoá; /tmp/smks/pwlock.sh; smk91 phải chạy: `source env/smk91-env.sh; cp env/smk91.override.yml /tmp/smk91.override.yml; $SMK_C up -d --no-build`). Ghi P/F vào results-21nov.tsv rồi `python3 evidence-9.5/scripts/apply-21nov.py`.
## DEF-R10.7-01..04 (CHƯA có kết quả)
`bash evidence-defrerun/run-r107-s9n.sh` đang chạy từng giai đoạn (log evidence-defrerun/R10.7/all.log; xong → evidence-10.7/audit-rr-defrerun.md). Đọc: 01 H3/E7/P5 teacher+admin /observability; 02 H7/X3 /gradebook; 03 X3 student /practice 375; 04 T2 /settings/llm. Ghi results-S9N.tsv + defects-closed.md khi có.
## Môi trường
smk91 đang chạy (13 container, qc2 giữ); smk91-postgres có DB edupilot_t1 dở dang (N03-10) — xoá được. Container qc2-n03-pg đã xoá. Docker stop/start chỉ đụng smk91-*.
