# QC sprint 11 (FEAT-p10-eval) — tổng kết (cập nhật 11-25, gồm «Lượt kiểm 25/11»; số ca theo bảng bên dưới)

Môi trường: stack `ep-qc` (HTTPS `https://localhost:943`, Postgres 18 + pgvector, compose ingest), dựng lại ở HEAD nhánh `sprint/11-p10-eval` nhiều lần (lần cuối `31c19e0d`, gồm `7545f033`, `51c14cd8`); DB giữ từ lượt seed sạch đầu (20/20 bước, 60 tài khoản, 107 câu luyện đề). Máy: MacBook Apple Silicon, Colima; load máy 4–46 do agent khác dùng chung (số hiệu năng ghi kèm load).

## Kết quả (đạt / tổng; bằng chứng từng ca: `scripts/results-p10.jsonl`, tiến độ `scripts/progress.py`)
| Nhóm | Tổng | Đạt | Trượt | Chưa kiểm | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| US-P10-01…03 Quan sát AI + cảnh báo | 52 | 52 | 0 | 0 | TC-01-10 PASS (EmptyState lớp QCT3) |
| US-P10-04…05 Insights | 54 | 54 | 0 | 0 | TC-04-11 PASS với Gemini thật (90 câu → 5 cụm, A/B đứng đầu); D-11-17/18 đóng |
| US-P10-06…07 Điểm thi + seed/E2E | 40 | 40 | 0 | 0 | TC-07-14 PASS @8b40dbbc (PROGRESS F16/F17 ☑, D-11-22 đóng); TC-06-21, 07-03, 07-09, 07-13 PASS |
| US-P10-14 `/admin/audit` | 100 | 100 | 0 | 0 | |
| `/assignments` SV (P-11-14) | 6 | 6 | 0 | 0 | |
| P-11-15 (DTO điểm cho SV) | 4 | 4 | 0 | 0 | ca 4 (UI «Chi tiết») PASS |
| @real F3 / F5 / F9+F11 / F16 / F17 | 8 | 8 | 0 | 0 | LLM thật (9router) ở F9/F11, TC-04-05…; Gemini thật ở TC-04-11 |
| U1 (làm mới phiên sớm) | 3 | 3 | 0 | 0 | |
| LCP 8 route (T1, `ep-t1`) | 8 | 8 | 0 | 0 | 7/8 ở T1; `/assignments/[id]` 4.794 ms (Nợ AC6, D-11-24) → **đạt** ở ep-qc sau LCP-2a/2c: 1.909 ms (trung vị 3, devtools, `?course=`), TC-08-02 P (L2.10) |
| LCP `/practice/[id]` (TC-12-17) | 1 | 1 | 0 | 0 | 1.909 ms (devtools, `?course=`), script 220 KB ≤ 256; lượt kiểm cuối 1.932 ms (L2.10, L2.11) |
| **Tổng** | **276** | **276** | **0** | **0** | hết Nợ AC6 (LCP); URL trần `/practice/[id]` ≈ 5,4 s là giới hạn đã biết, không chấm |

## Thật / giả
- LLM chữ: mặc định tuyến **9router thật** (`ag/gemini-3.8-flash`) cho CHAT/CLASSIFY/UTILITY/GRADING/QUESTION_GEN/INSIGHT; công tắc `/settings/llm` kiểm bằng INSIGHT (bật → 9router, tắt → fake env, bật lại). Ca cần fake (tên lỗi, 429, chậm 60 s, 1 triệu hàng, seed) dùng provider `QC fake` / `QC lỗi` / `QC chậm`.
- Embedding: **giả** (hash tất định) trừ TC-04-11: `gemini-embedding-001` thật theo P-11-47, ≈ 10/200 lời gọi (lượt cuối 3 lời gọi, 0 lỗi), provider đã gỡ, tuyến EMBEDDING trả về `fake-eval`.
- Decision / chấm: giả ở mọi ca trừ F9/F11 (chấm thật 9router).

## Ca không kiểm được (loại a/b/c)
Không có ca nào ở loại (a) thiết bị / (b) người thật / (c) ngoài phạm vi trong nhóm đã kiểm, trừ `/admin/health` (phase PR, P-11-30 — không tính FAIL). Không còn ca «chưa kiểm».

## Môi trường đã dựng để tránh «không kiểm được»
GV thứ hai `qct2` + lớp `QCT2` / `QCT3` (kiểm GV lớp khác, lớp trống, 04-21, 04-23…) · provider `QC *` + `llm_route.py` (real/fake) · trigger tạm, role tạm (đã DROP) · fixture `au_fixture.sql` 100.000 dòng `audit_log` · fixture 1 triệu hàng `llm_audit` (đã xoá) · `Caddyfile.kc` đồng bộ với `deploy/caddy/Caddyfile` · override compose `/tmp/ep-qc.seed.override.yml` (tick worker 20–60 s).

## Lỗi: xem `defects-p10.md` — D-11-01…25; đóng/PASS hết (D-11-22 đóng @8b40dbbc; D-11-24 đóng: TC-08-02 P 1.909 ms @f2ffcff7). Lỗi Gate P10 (GP10-01…08, R6-01, EXAMTICK…): `docs/regression/gate-p10-demo.md`.

## Dữ liệu QC để lại trên ep-qc (không đụng stack demo `edupilot*`)
Lớp `QCT2`, `QCT3`, `QCT4`; GV `qct2@edupilot.local`; thread «QC ghim …» / «QC nháp …»; sự kiện lịch «Ôn tập: …»; ≈ 70 câu chat QC (có PII giả `nguyenvanan.qc@gmail.com`, `qcpii*@example.com`); bài tập «QC F9 …», phúc khảo QC; 4 hàng `llm_audit` seed dời −3 giờ; provider `QC 9router` / `QC fake` / `QC lỗi` / `QC lỗi embed` / `QC chậm`. Dọn bằng `docker compose down -v` khi đóng sprint.

## Lệnh chạy lại
`cd ~/Documents/TA_Agent_qc5/frontend && QC_REAL=1 QC_BASE=https://localhost:943 … npx playwright test -c playwright.qc-real-flow.config.ts e2e/qc-real-*.spec.ts` (xem `real-flows.md`) · `python3 docs/sprints/11/qc/scripts/p10_*.py` · `python3 docs/sprints/11/qc/scripts/progress.py`.

## Việc còn lại
- TC-07-14: PASS @8b40dbbc (D-11-22 đóng).
- Stack: `ep-t1` và `ep-qc` đã dừng (25/11, volume giữ, không `-v`); dữ liệu ep-qc: PENDING 761987 = 150, 107 APPROVED (E5 chờ chủ dự án duyệt).
- Dọn dữ liệu QC trên ep-qc khi đóng sprint (QCT2/3/4, `qct2@`, provider `QC *`, fixture audit 100k).

## Lượt kiểm 25/11 (ep-qc, HEAD `3f91ed90`; chi tiết `docs/regression/gate-p10-demo.md` L2.7–L2.11)
| Mục | Kết quả | Ghi chú |
| --- | --- | --- |
| Vai DB (R6-01, TC-P3-01-49) | P | gateway nối bằng `edupilot_app`, không superuser / không bypass RLS; `update pii_events` → 42501; migrate 29 → 31 trên volume cũ không lỗi permission |
| `exam.tick` (EXAMTICK) | P | tự nộp quá hạn → `GRADED` (`CLOSED` / `TIMEOUT`); lượt lỗi (xoá `exam_items` bằng SQL) → đúng 1 dòng log «quá ngưỡng» sau 5 lần, lượt vẫn `IN_PROGRESS`; `go test ./internal/exam` xanh |
| F14 | P | SV «N tài liệu mới · lớp …» hiện rồi biến mất sau khi mở `/library`; Admin «Hàng đợi có N tin lỗi hẳn» (dòng ≤ 7 ngày), không lộ payload; 1440 + 375 |
| UI | P | LCP `?course=` ≤ 1.932 ms; đúng 1 `GET …/mine` ở `/assignments/[id]`; không nháy tên lớp; tiêu đề Markdown (`#`) hiện đúng ở bài tập, chat, bài AI trong thread; GP10-06 (`/me` «Điểm danh lúc») P |
| Judge (77091b87) | P | `go test -race -tags integration ./internal/judge` ×2 xanh; `TestJudgeSignalNotLostBeforeMark` PASS |
| Quan sát (không chấm) | – | `GET practice/attempts/<id>` gọi 2 lần ở `/practice/[id]?course=`; `exam` + `db` chạy song song có 1 lần đỏ không tái hiện (theo dõi ở CI) |
