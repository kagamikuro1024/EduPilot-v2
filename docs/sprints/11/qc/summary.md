# QC sprint 11 (FEAT-p10-eval) — tổng kết đợt 1–3

Môi trường: stack `ep-qc` (HTTPS `https://localhost:943`, Postgres 18 + pgvector, compose ingest), dựng lại ở HEAD nhánh `sprint/11-p10-eval` nhiều lần (lần cuối `31c19e0d`, gồm `7545f033`, `51c14cd8`); DB giữ từ lượt seed sạch đầu (20/20 bước, 60 tài khoản, 107 câu luyện đề). Máy: MacBook Apple Silicon, Colima; load máy 4–46 do agent khác dùng chung (số hiệu năng ghi kèm load).

## Kết quả (đạt / tổng; bằng chứng từng ca: `scripts/results-p10.jsonl`, tiến độ `scripts/progress.py`)
| Nhóm | Tổng | Đạt | Trượt | Chưa kiểm | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| US-P10-01…03 Quan sát AI + cảnh báo | 52 | 51 | 0 | 1 | TC-01-10 (EmptyState GV lớp chưa có yêu cầu AI) chờ stack |
| US-P10-04…05 Insights | 54 | 53 | 0 | 1 chặn | TC-04-11 (D-11-17): sửa ở `5852c9c2` đã vào build, chờ hạn mức Gemini ngày |
| US-P10-06…07 Điểm thi + seed/E2E | 40 | 35 | 4 | 1 | TC-06-21 (1.000 bài) chờ stack; 07-03c, 07-09, 07-13, 07-14 |
| US-P10-14 `/admin/audit` | 100 | 100 | 0 | 0 | |
| `/assignments` SV (P-11-14) | 6 | 6 | 0 | 0 | |
| P-11-15 (DTO điểm cho SV) | 4 | 3 | 0 | 1 | ca 4 (UI «Chi tiết») cần dữ liệu có `detail` |
| @real F3 / F5 / F9+F11 / F16 / F17 | 8 | 8 | 0 | 0 | LLM thật (9router) ở F9/F11, TC-04-11, TC-05-07 |
| U1 (làm mới phiên sớm) | 3 | 3 | 0 | 0 | |
| LCP 8 route | 8 | 4 | 0 | 4 sơ bộ | devtools, load 3,5–5,2, dữ liệu seed ep-qc — SƠ BỘ, không T1, chưa chấm (D-11-24); chờ đo T1 |

## Thật / giả
- LLM chữ: mặc định tuyến **9router thật** (`ag/gemini-3.8-flash`) cho CHAT/CLASSIFY/UTILITY/GRADING/QUESTION_GEN/INSIGHT; công tắc `/settings/llm` kiểm bằng INSIGHT (bật → 9router, tắt → fake env, bật lại). Ca cần fake (tên lỗi, 429, chậm 60 s, 1 triệu hàng, seed) dùng provider `QC fake` / `QC lỗi` / `QC chậm`.
- Embedding: **giả** (hash tất định) trừ TC-04-11: `gemini-embedding-001` thật theo P-11-47, ≈ 5/200 lời gọi, provider đã gỡ, tuyến EMBEDDING trả về `QC lỗi embed` (tắt, ENV lấp).
- Decision / chấm: giả ở mọi ca trừ F9/F11 (chấm thật 9router).

## Ca không kiểm được (loại a/b/c)
Không có ca nào ở loại (a) thiết bị / (b) người thật / (c) ngoài phạm vi trong nhóm đã kiểm, trừ `/admin/health` (phase PR, P-11-30 — không tính FAIL). Mọi ca «chưa kiểm» ở bảng trên là chờ stack (TL dùng ep-qc) chứ không phải kết quả cuối.

## Môi trường đã dựng để tránh «không kiểm được»
GV thứ hai `qct2` + lớp `QCT2` / `QCT3` (kiểm GV lớp khác, lớp trống, 04-21, 04-23…) · provider `QC *` + `llm_route.py` (real/fake) · trigger tạm, role tạm (đã DROP) · fixture `au_fixture.sql` 100.000 dòng `audit_log` · fixture 1 triệu hàng `llm_audit` (đã xoá) · `Caddyfile.kc` đồng bộ với `deploy/caddy/Caddyfile` · override compose `/tmp/ep-qc.seed.override.yml` (tick worker 20–60 s).

## Lỗi: xem `defects-p10.md` (16 đã sửa và kiểm lại PASS; 6 còn mở khi viết).

## Dữ liệu QC để lại trên ep-qc (không đụng stack demo `edupilot*`)
Lớp `QCT2`, `QCT3`; GV `qct2@edupilot.local`; thread «QC ghim …» / «QC nháp …»; sự kiện lịch «Ôn tập: …»; ≈ 70 câu chat QC (có PII giả `nguyenvanan.qc@gmail.com`, `qcpii*@example.com`); bài tập «QC F9 …», phúc khảo QC; 4 hàng `llm_audit` seed dời −3 giờ; provider `QC 9router` / `QC fake` / `QC lỗi` / `QC lỗi embed` / `QC chậm`. Dọn bằng `docker compose down -v` khi đóng sprint.

## Lệnh chạy lại
`cd ~/Documents/TA_Agent_qc5/frontend && QC_REAL=1 QC_BASE=https://localhost:943 … npx playwright test -c playwright.qc-real-flow.config.ts e2e/qc-real-*.spec.ts` (xem `real-flows.md`) · `python3 docs/sprints/11/qc/scripts/p10_*.py` · `python3 docs/sprints/11/qc/scripts/progress.py`.
