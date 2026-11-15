# Smoke hồi quy sprint 9.5 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r + stack riêng ep-r95-* · Người chạy: qc2 · Ngày: 2026-11-15
Tổng TC cũ: {T} · Đạt: {P} · Trượt: {F} · Không chạy (K, loại a/b/c): {K} · Lỗi thời (LT): {LT} · Chưa chạy (mẫu, CM): {CM}   (P+F+K+LT+CM = T)
Cách chạy:
- FILE/SCAN/DIFF: đọc tệp, `check-meta`, `canary-scan.sh` (ADMIN, ep-r), `git log/diff ff00b5aa..2d730a4e` (khoảng sprint 9.5), tính lại số liệu bằng Python từ `benchmarks/reports/**`.
- API/seed trên ep-r: `check-*-seed.mjs`, đăng nhập 4 vai, `/me/today`, sổ điểm, chạy lại `seed.mjs`. ep-r bị 7 agent dùng chung nên dữ liệu seed đã bị đổi (xem DEF-R9.5-17).
- Stack riêng (bản sao script trong `evidence-9.5/scripts/`, mã sản phẩm không sửa): postgres `ep-r95-pg`, gateway/worker/redis `ep-r95-*`, DB `edupilot_t1_r95` do `seed-t1` nạp (1.000 SV); chạy `check-t1-seed`, `check-t1-api`, vòng lặp 403 + p95 200 lần, `explain.mjs`, `chaos.mjs` (kill -9 worker, Redis tắt 70 s), k6 `dashboard` (container 1,5 CPU / 1,5 GB), Redis tắt + chat.
- Máy chủ giả `fake-gateway.mjs` cho TC harness 429/401 (real_llm.py).
- Sự cố: lúc 12:59:54 UTC `ep-r-postgres-1` crash/recovery khi qc2 nạp seed-t1 vào DB phụ của chính postgres đó (I/O quá tải, checkpoint 268 s); hết recovery 13:02:05; qc2 xoá 2 DB phụ và chuyển sang postgres riêng. Không kill/stop container ep-r-*.
- Quy ước kết quả: ca môi trường bị chặn đã chạy lại; K (a/b/c) = 0 vì không ca nào cần thiết bị thật / người thật / ngoài đồ án; LT = spec/quyết định sau (P-95-8: 9router thay Gemini trực tiếp, không tạo 00018); CM = chưa chạy (mẫu) do hết quỹ lượt chạy, chưa có script hoặc cần dựng thêm (docling riêng, pg_stat_statements, Lighthouse, README Mac sạch): PM quyết có chạy nốt không.
