# STATE smk4t2 (SmkT2, 2026-11-19)
Kết quả: evidence-4/results-smk34/results-smk4t2.tsv (bản ghi sau thắng). Script: evidence-4/scripts/smkt2*.py; log Go: logs/smkT2/.
ĐÃ XONG (P): P202-26,30,32,36,38; P203-09(duyệt lại); P205-16,21; P206-25,30,33; P207-01(duyệt lại),03,06,11; P208-39; GATEP2-02,03,04,06,07,08,10,11,12,14,15,16,20,23,25,29,46,48.
F: P201-11 (DEF-R4-05 token mời rõ trong log Caddy), P201-19 (DEF-R4-07), P202-47 (DEF-R4-06), GATEP2-44, GATEP2-53. LT: P207-14, GATEP2-09 (DEF-R4-08 phụ).
CÒN N: P202-31 (cần ≥5 request cùng token cũ: sửa spec r4ui-auth chưa đạt, apiClient chỉ refresh khi token chưa đổi), P202-41 (spec smkt2-next.spec.ts đã viết, chưa chạy), P202-42 (safe-next.spec.ts: chạy bằng playwright.qc2g.config.ts QC2G_DIR=e2e), P202-44, P202-48, P202-30 xong;
 P204-27/28/30/31 (đã sửa r4ui-acct: ghi danh lớp cho SV vì /settings của SV cần lớp (nav.ts needsCourse), spawnSync log Caddy, 150 lần Tab; lượt chạy đầu fail sau 0,5 s — kiểm lại freshUser/429 trước), P206-34,39, P207-35, P208-32,34,35,37 (đã sửa r4ui-admin; chạy lại), GATEP2-05,21,22,24,26,27,28,30,31,40–43,45,47,49–52,99.
Lượt Playwright pw-6 (/tmp/smkt2/pw-6.out) còn có thể đang chạy: smkt2-explore/dom/next + r4ui-*; kết quả ở /tmp/qc2-i/res/*.jsonl.
Dọn: xoá frontend/e2e/qc2-r4/smkt2-dom.spec.ts, smkt2-explore.spec.ts sau khi dùng (lint có lỗi any); đã xoá smkt2-dom2.
Ghi nhận cho qc2: CI nhánh qc2/regression đỏ vì spec QC e2e/qc2-* bị gom vào playwright test; /dev/data 404 trên ep-r nên P202-31 không dùng được __ep.
Đã dọn: smkt2-gw/smkt2-wk, DB qc2i_empty; qc2i-redis đã chạy lại.
