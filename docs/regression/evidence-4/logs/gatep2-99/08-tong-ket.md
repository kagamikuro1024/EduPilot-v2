# TC-GATEP2-99 — dọn môi trường qc2 (P-11-100), 22/11

- `compose stop` qc2seedt2 (kể cả qc2seedt2-docling-1, thuộc profile ingest) — container giữ lại ở trạng thái Exited, volume nguyên.
- Dừng + `docker rm` (KHÔNG `-v`, KHÔNG `down -v`): qc2s-gw/worker/mail/redis, 14 container qc2seedt-*, 14 container qc2fresh-*, 6 container qc2i-*.
- Không đụng: qc2seedt3 (QC), ep-r (12 chạy), ep-qc, các container edupilot-test-*.
- Tiến trình sót của qc2: không có (02-procs.txt chỉ còn Playwright của qc2seedt3 — QC).
- git status worktree sạch sau khi khôi phục benchmarks/reports/seed-t0.json (file sinh ra do seed) và xoá test-results/ (đầu ra Playwright).
- Volume + cỡ: 03-volumes.txt, 07-other-volumes.txt; DB phụ trong ep-r-postgres-1: 04-epr-dbs.txt. Không xoá gì — chờ chủ dự án quyết.
