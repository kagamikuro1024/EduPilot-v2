# Đo phi chức năng T1 trên Mac và đánh giá LLM thật (sprint 9.5)

Thiết bị: **MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB, k6 trong VM 1,5 CPU**. Stack đo `ep-t1`, DB `edupilot_t1` (1.000 SV, 20 lớp, 409 MB). Stack demo đã `stop` trong mọi lượt đo (bằng chứng: `benchmarks/reports/demo-stack.json`). Dữ liệu mô phỏng hoàn toàn (D44). Mọi số có tệp nguồn trong `benchmarks/reports/`; số trong bảng dưới đây không có nguồn nào khác.

## Tóm tắt (đọc trong ≤ 5 phút)
- **Đạt:** (a) duyệt 300 người + 300 SSE, (b) chat (sự kiện đầu, TTFT cache và RAG), (c) chấm 50 bài khi chat, (d) dashboard / hồ sơ 360 / sổ điểm, baseline p95 13 endpoint, chaos (kill -9 worker, Redis tắt 70 s): đều trong SLO ở ×1 và ×2, không đổi mã giữa hai cấu hình.
- **Trượt:** Lighthouse trên bản build production có đăng nhập thật: LCP devtools > 2.500 ms ở 6 trong 8 route (TL-2f); TBT, CLS, kích thước script đạt mọi route.
- **Không cần chỉ mục:** `EXPLAIN` 10 truy vấn nặng nhất ≤ 13 ms; không tạo `00018` (P-95-8).
- **Phát hiện:** Redis tắt làm chat treo ≈ 19 s (TL-2d); tạo thread có lượt nhúng trong request (TL-2e); truy vấn tìm lai nối `documents × content_chunks` không lọc lớp (TL-2c).
- **Chưa đo / chưa hoàn tất:** đo LLM thật (thiếu khoá Gemini), seed T0 và chaos `docling-serve` (cần stack có `ingest`), chấm 1.000 bài thời gian thật, judge amd64, E3 chấm tay, nhiều provider.

## Bảng SLO → số đo → kết quả
| SLO (SYSTEM_DESIGN §5) | ×1 | ×2 (`--scale gateway=2 worker=2`) | Kết quả | Tệp |
| --- | --- | --- | --- | --- |
| (a) đọc p95 ≤ 300 ms, 300 người duyệt + 300 SSE + heartbeat 10/s, 5 phút | 11,5 ms | 11,4 ms | PASS | `load-a.json`, `load-a-x2.json` (và `load-a-lan1.json`, `load-a-x2-lan1.json`: lần đầu, trước khi thêm bước 403) |
| (a) ghi p95 ≤ 500 ms | 4,9 ms | 4,9 ms | PASS | cùng tệp |
| (a) sự kiện SSE đầu p95 ≤ 300 ms | 97,5 ms | 144,2 ms | PASS | cùng tệp |
| (a) 5xx < 0,5 % | 0 | 0 | PASS | cùng tệp |
| (a) STUDENT gọi `gradebook` ≥ 100 lần: 100 % `403` | 300 lần 403, 0 lần 200 | 301 lần 403, 0 lần 200 | PASS | cùng tệp (`forbidden_403`, `forbidden_leak`) |
| (b) sự kiện đầu của chat p95 ≤ 300 ms (50 luồng, provider giả 5–15 s) | 66,9 ms | 118,9 ms | PASS | `load-b-first_event.json`, `-x2` |
| (b) TTFT cache p95 ≤ 1,5 s | 103,8 ms | 102,3 ms | PASS | `load-b-ttft.json`, `-x2` |
| (b) TTFT RAG p95 ≤ 4 s | 633,9 ms | 626,4 ms | PASS | cùng tệp |
| (c) TTFT chat lúc chấm xấu đi ≤ 20 % (50 bài, provider giả 300 ms) | nền 410 ms → chấm 381 ms (−7 %) | nền 415 ms → chấm 401 ms (−3,5 %) | PASS | `load-c-baseline*.json`, `load-c-grading*.json` |
| (c) chấm 1.000 bài thời gian thật ≤ 3 giờ | đang chạy (xem dòng dưới) | – | chưa có kết quả | `load-c-1000.json` |
| (d) dashboard / 360 / sổ điểm p95 ≤ 3 s | 18,3 ms | – (AC3 chỉ (a)–(c)) | PASS | `load-d.json` |
| Baseline: p95 13 endpoint tuần tự × 200 (10 đọc ≤ 300 ms, 3 ghi ≤ 500 ms) | đọc ≤ 11,0 ms; ghi lớn nhất `w_thread` 327,5 ms | – | PASS | `load-baseline-p95.json`, `baseline.md` |
| RAM nghỉ cả stack · khởi động lạnh (up → readyz 200) | 570 MiB · 18,1 s (18,1 / 18,1 / 18,9) | – | ghi nhận | `baseline.json` |
| Image (MB) | gateway 17,7 · worker 16,7 · frontend 96,2 | – | ghi nhận | `baseline.json` |
| Chaos: `kill -9` worker giữa lượt chấm 150 bài | 150 vào = 150 xong, 0 mồ côi, 0 chấm hai lần, `XPENDING` 0 | – | PASS | `chaos.json` |
| Chaos: Redis tắt 70 s giữa lượt chấm 150 bài | 150 = 150, 0 mồ côi, 0 chấm hai lần | – | PASS (xem TL-2d về chat) | `chaos.json` |
| Lighthouse (build production, đăng nhập thật) | LCP devtools (≤ 2.500) / TBT mô phỏng (≤ 200) / CLS (≤ 0,1) / script KB (≤ 256): `/` 1574 / 28 / 0 / 191; `/chat` 1554 / 32 / 0.01 / 225; `/threads` 4896 / 33 / 0 / 216; `/inbox` 6717 / 32 / 0 / 215; `/gradebook` 5887 / 29 / 0 / 248; `/me` 6256 / 35 / 0 / 223; `/settings/llm` 4786 / 32 / 0 / 228; `/assignments/[id]` 4639 / 36 / 0.028 / 222 | – | FAIL: LCP devtools trượt ở `/threads`, `/inbox`, `/gradebook`, `/me`, `/settings/llm`, `/assignments/[id]`; TBT, CLS, script đạt mọi route | `lighthouse.json` |
| EXPLAIN 10 truy vấn | ≤ 13,1 ms (1 truy vấn), chín truy vấn ≤ 1,3 ms | – | không cần chỉ mục | `explain.json`, `docs/research/2026-11-14-explain.md` |

## Từng phép đo
### (a) 300 người duyệt + 300 SSE + heartbeat 10/s (US-NFR-04 AC1, AC3, AC4)
Câu hỏi: ở T1 hệ thống có giữ đọc p95 ≤ 300 ms, ghi p95 ≤ 500 ms, SSE đầu ≤ 300 ms, 5xx < 0,5 % không, và nhân đôi gateway / worker có đổi gì không. Phương pháp: `node benchmarks/load/run-t1.mjs browse [--tag x2]`, 5 phút, k6 trong VM, 300 SV rải 20 lớp, tỉ lệ đọc : ghi ≈ 20 : 1, thêm 1 yêu cầu/giây của SV gọi `gradebook` (phải 403). Kết quả: xem bảng. Giới hạn: provider giả; k6 và stack chung 6 vCPU.
### (b) chat (AC5)
`chat-first_event` (50 luồng, provider giả trễ 5–15 s) và `chat-ttft` (20 luồng, 300 ms). Sự kiện đầu p95 66,9 ms (×1) và 118,9 ms (×2); TTFT cache p95 103,8 / 102,3 ms; TTFT RAG p95 633,9 / 626,4 ms. Ghi riêng hai nhóm theo yêu cầu.
### (c) chấm bài khi chat (AC2)
`grading-baseline` rồi `grading-grading` cùng một lần chạy cấu hình: TTFT chat p95 lúc nền 410 ms, lúc chấm 381 ms (×1); 415 và 401 ms (×2). 50 bài BT4 lớp T1-01, hai lượt mỗi bài. **Chấm 1.000 bài thời gian thật** (AC9): đang chạy: provider giả của worker trễ 20 s mỗi lượt (`K6_GRADE_LAT=20000-20000`), 20 lớp × 50 bài BT4, hai lượt mỗi bài, chat 20 luồng song song (gateway trễ 300 ms) trong `grading-baseline --duration 160m`; `benchmarks/load/grade-all-t1.mjs` ghi `load-c-1000.json` (tiến độ mỗi phút, đã xong / lỗi / mồ côi / chấm hai lần, số giờ). So TTFT chat p95 của `load-c-1000-baseline` (k6) với nền 410 ms. Kết quả được bổ sung sau, không ngoại suy từ lượt 50 bài.
### (d) dashboard (AC1)
`dashboard` 2 phút, 5 người, giảng viên đọc Hôm nay / danh sách SV / sổ điểm / hồ sơ 360: p95 18,3 ms (SLO 3.000 ms). Bảng truy vấn nặng nhất rút ra từ kịch bản này ở `docs/research/2026-11-14-explain.md`.
### Baseline (US-NFR-07)
RAM nghỉ 570 MiB cả stack (postgres 330 MiB), image gateway 17,7 MB, worker 16,7 MB, frontend 96,2 MB, khởi động lạnh trung vị 18,1 s; 13 endpoint tuần tự × 200: mọi đọc p95 ≤ 11,0 ms, ghi lớn nhất `w_thread` 327,5 ms. Lighthouse: bản build production, đăng nhập thật bằng tài khoản T1, 3 lần mỗi route, trung vị (`lighthouse.json`). TBT mô phỏng ≤ 36 ms, CLS ≤ 0,028, script ≤ 248 KB đạt ngưỡng ở mọi route. **LCP devtools trượt (≤ 2.500 ms) ở 6 trong 8 route:** `/threads` 4896 ms, `/inbox` 6717 ms, `/gradebook` 5887 ms, `/me` 6256 ms, `/settings/llm` 4786 ms, `/assignments/[id]` 4639 ms (đạt: `/` 1574 ms, `/chat` 1554 ms). LCP mô phỏng 2.857–3.459 ms. `/assignments/[id]` (nợ G7-23: 4,7 s) vẫn 4.639 ms. Không nới ngưỡng; nguyên nhân chưa điều tra (nghi ngờ: LCP đợi dữ liệu API sau bước làm mới phiên; mạng devtools giả lập 150 ms).
### Chaos (US-NFR-06)
`kill -9` một trong hai worker giữa lượt chấm 150 bài: 150 vào = 150 xong, 0 mồ côi, 0 chấm hai lần, `XPENDING` 0, hồi trong 113 s. Redis tắt 70 s giữa lượt chấm 150 bài: 150 = 150, 0 mồ côi, 0 chấm hai lần, hồi trong 129 s; `/me/today` trả 200 suốt lúc Redis tắt. Chat lúc Redis tắt treo ≈ 19 s rồi trả câu suy giảm (TL-2d). `docling-serve` dừng 2 phút: chưa chạy.
### EXPLAIN và chỉ mục (US-NFR-05)
Xem `docs/research/2026-11-14-explain.md`.
### LLM thật (US-NFR-08)
**Chưa hoàn tất.** Lý do: `.env.local` không có `GEMINI_API_KEY`. Đã chuẩn bị: `deploy/vps-llm.mjs` (`PHASE=probe|seed|live`, chuỗi model chọn đầu phiên, `gemini-chat` + `gemini-embed`), `benchmarks/real-llm/real_llm.py` (có `--resume`), `benchmarks/real-llm/export-evidence.mjs` (xuất trước `down -v`). Dữ liệu gửi Gemini Free (nội dung có thể được Google dùng để cải thiện sản phẩm) chỉ gồm dữ liệu mô phỏng (D44) đã che tên / MSSV.

## Chưa đo được và vì sao
| Việc | Lý do |
| --- | --- |
| Đo LLM thật (R1–R6), chọn model đầu phiên | thiếu khoá Gemini |
| Seed T0 sạch + `check-*-seed.mjs` (US-NFR-02) | cần stack `ep-research` có docling |
| Chaos `docling-serve` | cần stack có `ingest` |
| Judge amd64 có seccomp | không đo được trên arm64 (D58) |
| E3 chấm tay, E4 nhiều provider | thuộc P10 |
| Mạng thật / VPS | ngoài phạm vi (CONTEXT mục 4) |

## Danh sách câu cần vá ở báo cáo cũ (US-NFR-09 AC3)
| Vị trí | Câu hiện có | Câu mới đề xuất |
| --- | --- | --- |
| `docs/sprints/8/report.md`, `docs/sprints/9/report.md`, `PROGRESS.md` (các dòng "giả lập" / "chưa đo" về T1) | "chưa đo T1" | "Đo trên MacBook Apple Silicon (Colima 6 CPU / 10 GB): T1 1.000 SV đạt SLO đọc / ghi / SSE / TTFT / dashboard ở ×1 và ×2; số trong `docs/research/2026-11-14-nfr-real-llm.md`" |
| `PROGRESS.md` mục Nợ | – | thêm: TL-2c (truy vấn tìm lai), TL-2d (chat treo khi Redis tắt), TL-2e (nhúng trong request tạo thread) |
