# STATE S4X (sprint 4 N còn lại + DEF-rerun) — 2026-11-21 ~21:45, ep-r @aba738ef
Bảng sprint 4: P 533, F 3, LT 6, K 3, N 2 (chạy `cd evidence-3 && python3 scripts/smk34-merge.py 4`). Kết quả: evidence-4/results-smk34/results-s4n.tsv. DEF: evidence-defrerun/results-S4X.tsv.

## Sprint 4 (N 9 → 2)
- P: TC-P202-48, P202-31 (admin /observability ép 401 lên 5 request song song, 1 refresh, phát lại đúng 1 lần), P212-21 (audit 130/130 + ma trận 4×32), P212-24, GATEP2-22, GATEP2-24, P208-34 (rerun DEF-R4-10).
- LT (nguồn: gate-PE A9, «Kết quả của tôi» sprint 6, US-P10-14 c3220b5c): P202-44, GATEP2-28 (nav 9/13/16/7, TC gốc 7/12/15/6; phần hành vi đạt).
- N: TC-GATEP2-99 — qc2 chạy `bash evidence-4/scripts/s4x-gate99.sh` lúc mọi agent đã dừng rồi tự down -v/docker rm các stack phụ; ảnh chụp hiện trạng logs/s4x/gate99-snapshot.txt. TC-P212-14 — qc2 (cửa sổ SEED).
- Spec đã sửa: specs-qc2/qc2-r4/smkt3-p20231.spec.ts (chờ > staleTime 30 s, visibilitychange bubbles), r4ui-audit.spec.ts (ma trận settle 2,6 s), s4x-probe.spec.ts (soi cut/chặn quyền).

## DEF-rerun (ghi results-S4X.tsv; ĐÓNG ở defects-closed.md, defects-4.md)
- P: R4-10 (P208-34), R4-05 (P201-11, GATEP2-44, GATEP2-53), R5-06, R5-07, R5-08, R5-09, R5-10 (API + UI), R5-11, R3-01, R7-02.
- N còn lại (chỉ R9.5-17):
  - R9.5-17 (TC-N02-02/03): seed.mjs không chạy lại được trên ep-r: DEF-R9.5-26 (limit=100, > 479 lớp), rồi bản sao evidence-defrerun/R9.5-17/seed-qc.mjs (đổi 3 truy vấn sang q=<mã>) dừng bước 6 vì sv34 chưa xác minh (đã vá SQL), sv38 DISABLED, worker down. Cần: worker lên, sv38 ACTIVE, rồi `bash evidence-defrerun/R9.5-17/run.sh` (qua khoá, xoá rate limit mỗi giây; đếm bảng trước/sau, 7 check-*-seed). Check hiện tại: assignments 0; chat/crm/docs/grade/knowledge 1; exam 2.
## Cách chạy chung
Playwright qua evidence-4/scripts/s4n-pw.sh <tên> desktop <spec…> (khoá /tmp/qc2-pw.lock); script DEF qua /tmp/qc2-pw.sh; xoá rate limit TRONG khoá (không trước khi chờ khoá). Hàng đợi tổng: evidence-defrerun/run-s4x-post.sh.
