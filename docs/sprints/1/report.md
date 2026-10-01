# Sprint 1 — báo cáo

Mục tiêu: xong trọn P0 (viết mới toàn bộ, D45) · Kết quả: **3/3 story PASS · cổng P0 PASS** · Nhánh `sprint/1-p0-prep` (đã push, chưa merge)

| Story | Feature | Trạng thái | QC | Ghi chú |
| --- | --- | --- | --- | --- |
| US-P0-01 | FEAT-demo-script | PASS | `qc/report-US-P0-01.md` — 14 TC, 9/9 AC | `docs/DEMO_SCRIPT.md` 15 phút, F2→F3→F5→F7→F9→F10→F17; 9 câu hỏi sản phẩm mở (không chặn P0) |
| US-P0-02 | FEAT-scaffold | PASS | `qc/report-US-P0-02.md` — 38 TC, 7/7 AC | Mã Project III → `legacy/` (376 file, giữ lịch sử); gateway Go `/healthz`; Next.js 16 + token + Be Vietnam Pro; stack 6 service healthy |
| US-P0-03 | FEAT-ci | PASS | `qc/report-US-P0-03.md` — 16 TC, 5/5 AC | CI 2 job (Go, Frontend), runner ghim `ubuntu-24.04`; chứng minh đỏ đúng job bằng nhánh tạm (đã xoá) |

Cổng: `qc/gate-P0.md` — 6/6 lệnh PASS, CI xanh trên `c69b1fb`.

## Thay đổi hướng trong sprint (chủ dự án chốt)
1. **D45 — viết mới toàn bộ**, mã cũ chỉ tham khảo. Bỏ golden/contract Java, mốc k6/RAG của Java, migrate dữ liệu cũ.
2. **D46 — chỉ Go, bỏ service Python AI.** Căn cứ: `thesis-notes/legacy-perf.md` (hệ cũ chậm vì 2–6 lời gọi LLM nối tiếp trước chữ đầu, N+1, DB xa, ingest chung tiến trình — không vì ngôn ngữ).
3. **D47 — luật tốc độ** cho đường hỏi–đáp; **D48 — phiên bản nền** (Go 1.27, Node 24, Postgres 18 + pgvector, Redis 8, Next.js 16, Mailpit…).
4. Quy trình: QC viết TC song song với dev; đội góp ý qua `proposals.md`, PM quyết (11 góp ý, chấp nhận 11).
Toàn bộ tài liệu nền (CLAUDE, ARCHITECTURE, SYSTEM_DESIGN, WORKFLOW, mọi phase, team, skills) đã cập nhật; PG viết lại thành "Nền Go"; migration đánh số từ `00001`, nguyên tắc "phase đầu tiên dùng bảng tạo nó ở dạng cuối".

## Luồng end-to-end đã đi trọn
Chưa có (P0 không có tính năng người dùng).

## Số liệu
- QC: 337 kiểm PASS trên 68 TC; 2 lỗi mức thấp đều là lệch chữ spec/tài liệu, đã sửa.
- CI: run cuối `36880299860` xanh (Go + Frontend).
- Mã: `legacy/` 39.476 dòng (354 file) dời ra; mã mới 304 dòng (10 file). 43 commit trên nhánh.
- Thời gian: P0 lịch 0,5 tuần, xong trong 1 ngày. Chi phí LLM: 0 (không gọi LLM thật).

## Việc nợ chuyển sang sprint kế
- Image object storage lâu dài (đang dùng fork `pgsty/minio`) — chốt ở lát blob của PG (proposals #7).
- `typescript` ghim `^5` vì `typescript-eslint` chưa hỗ trợ TS 7.
- GitHub Actions ghim theo major, không theo SHA (proposals #11).
- `thesis-notes/legacy-perf.md` phần thân còn tiếng Anh.

## Rủi ro và điều cần chủ dự án quyết
1. **Lịch:** D45 thêm ≈ 3–5 tuần; D46 bớt việc dựng service Python. Lịch 23,5 tuần ở WORKFLOW §6 chưa đổi — cần chủ dự án chọn: lùi vạch bảo vệ hay cắt theo danh sách.
2. **9 câu hỏi sản phẩm của kịch bản demo** (`specs/FEAT-demo-script/QUESTIONS.md`), đáng chú ý: Q3 dưới ngưỡng tin cậy thì tự mở ticket hay chờ sinh viên bấm; Q6 ngày bảo vệ mặc định LLM thật hay `DEMO_MODE`. Chặn P1/P2/P4/P7, không chặn PG.
3. Workflow `keep-huggingface-space-awake.yml` vẫn chạy, giữ Space của Project III thức — tắt khi không cần.
4. Nhánh sprint tách từ `chore/edupilot-v2-docs` (bộ tài liệu chưa vào `main`): merge cả hai.

## Đề xuất sprint 2
PG — Nền Go (`phases/PG.md`, 3,5 tuần): bắt đầu lát khung dịch vụ + tầng dữ liệu (pgx, goose `00001 pg_platform`, sqlc) và httpapi (định dạng lỗi, phân trang con trỏ, `Idempotency-Key`), mỗi story kết thúc bằng contract test với `openapi.yaml`.
