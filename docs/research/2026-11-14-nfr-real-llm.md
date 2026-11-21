# Đo phi chức năng T1 trên Mac và đánh giá LLM thật (sprint 9.5)

Thiết bị: **MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB, k6 trong VM 1,5 CPU**. Stack đo `ep-t1`, DB `edupilot_t1` (1.000 SV, 20 lớp, 409 MB). Stack demo đã `stop` trong các lượt đo, trừ ba lượt ghi `running` (mục «Lần đo bị nhiễu, lần đo lại»; bằng chứng: `benchmarks/reports/demo-stack.json`, `meta.demo_stack`). Dữ liệu mô phỏng hoàn toàn (D44). Mọi số có tệp nguồn trong `benchmarks/reports/`; số trong bảng dưới đây không có nguồn nào khác.

## Tóm tắt (đọc trong ≤ 5 phút)
- **Đạt:** (a) duyệt 300 người + 300 SSE, (b) chat (sự kiện đầu, TTFT cache và RAG), (c) chấm 50 bài khi chat, (d) dashboard / hồ sơ 360 / sổ điểm, baseline p95 13 endpoint, chaos (kill -9 worker, Redis tắt 70 s): đều trong SLO ở ×1 và ×2, không đổi mã giữa hai cấu hình.
- **Trượt:** Lighthouse trên bản build production có đăng nhập thật: LCP devtools > 2.500 ms ở 6 trong 8 route (TL-2f); TBT, CLS, kích thước script đạt mọi route.
- **Không cần chỉ mục:** `EXPLAIN` 10 truy vấn nặng nhất ≤ 13 ms; không tạo `00018` (P-95-8).
- **Phát hiện:** Redis tắt làm chat treo ≈ 19 s (TL-2d); tạo thread có lượt nhúng trong request (TL-2e); truy vấn tìm lai nối `documents × content_chunks` không lọc lớp (TL-2c).
- **Đã làm sau cùng:** seed T0 sạch (314 s, `seed-t0-checks.json`), chaos `docling-serve` dừng 2 phút (3/3 tài liệu `READY`, 0 mồ côi, hồi trong 41 s), đo LLM thật R1–R5 qua 9router (`ag/gemini-3.8-flash`, `ep-qc`, 2026-11-15); R6 (k6) chỉ có lượt Gemini trực tiếp `gemini-3.1-flash-lite` (2026-11-14); sửa được lỗi `const` (TL-2h2).
- **Chấm 1.000 bài qua 9router:** lần 1 **FAIL** (322/1.000 xong, 678 `FAILED` sau 0,55 h: mất nhà cung cấp làm hết lượt thử, sửa ở P-11-23, `docs/sprints/9.5/report.md` mục 2); lần 2 **PASS** (1.000/1.000, 0 `FAILED`, 1,68 h).
- **Chưa đo / hoãn:** R6 qua 9router, judge amd64, E3 chấm tay, nhiều provider. R3 mask và Threads / khủng hoảng đã đo lại đủ mẫu (DEF-R9.5-10/11, bảng dưới).

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
| (c) chấm 1.000 bài thời gian thật ≤ 3 giờ (9router `ag/gemini-3.8-flash`, 5 luồng) | lần 1: 322/1.000 xong, 678 `FAILED`, 0,55 h; lần 2: 1.000/1.000, 0 `FAILED`, 0 mồ côi, 0 chấm hai lần, 1,68 h | – | lần 1 **FAIL**, lần 2 **PASS** | `load-c-1000-9router-run1-hong.json`, `load-c-1000-9router-run2.json` (lần Gemini trực tiếp dừng ở 235/1.000: `load-c-1000-dung-theo-chu-du-an.json`) |
| (d) dashboard / 360 / sổ điểm p95 ≤ 3 s | 18,3 ms | – (AC3 chỉ (a)–(c)) | PASS | `load-d.json` |
| Baseline: p95 13 endpoint tuần tự × 200 (10 đọc ≤ 300 ms, 3 ghi ≤ 500 ms) | đọc ≤ 11,0 ms; ghi lớn nhất `w_thread` 327,5 ms | – | PASS | `load-baseline-p95.json`, `baseline.md` |
| RAM nghỉ cả stack · khởi động lạnh (up → readyz 200) | 570 MiB · 18,1 s (18,1 / 18,1 / 18,9) | – | ghi nhận | `baseline.json` |
| Image (MB) | gateway 17,7 · worker 16,7 · frontend 96,2 | – | ghi nhận | `baseline.json` |
| Chaos: `kill -9` worker giữa lượt chấm 150 bài | 150 vào = 150 xong, 0 mồ côi, 0 chấm hai lần, `XPENDING` 0 | – | PASS | `chaos.json` |
| Chaos: Redis tắt 70 s giữa lượt chấm 150 bài | 150 = 150, 0 mồ côi, 0 chấm hai lần | – | PASS (xem TL-2d về chat) | `chaos.json` |
| Lighthouse (build production, đăng nhập thật) | LCP devtools (≤ 2.500) / TBT mô phỏng (≤ 200) / CLS (≤ 0,1) / script KB (≤ 256): `/` 1574 / 28 / 0 / 191; `/chat` 1554 / 32 / 0.01 / 225; `/threads` 4896 / 33 / 0 / 216; `/inbox` 6717 / 32 / 0 / 215; `/gradebook` 5887 / 29 / 0 / 248; `/me` 6256 / 35 / 0 / 223; `/settings/llm` 4786 / 32 / 0 / 228; `/assignments/[id]` 4639 / 36 / 0.028 / 222 | – | FAIL: LCP devtools trượt ở `/threads`, `/inbox`, `/gradebook`, `/me`, `/settings/llm`, `/assignments/[id]`; TBT, CLS, script đạt mọi route | `lighthouse.json` |
| EXPLAIN 10 truy vấn | ≤ 13,1 ms (1 truy vấn), chín truy vấn ≤ 1,3 ms | – | không cần chỉ mục | `explain.json`, `docs/research/2026-11-14-explain.md` |

Hạn chế Mac cho mọi số trong bảng: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB. k6 và stack chung 6 vCPU nên độ trễ là cận trên bảo thủ [INFERENCE: chưa kiểm bằng máy đo riêng]; mạng là loopback trong VM nên p95 là cận dưới về phần mạng.

## Từng phép đo
Mỗi phép đo US-NFR-02…08 một mục, đủ 6 phần: câu hỏi · phương pháp · cấu hình · số liệu · PASS/FAIL · giới hạn.

### US-NFR-02: seed T0 trên Mac
- **Câu hỏi:** seed demo T0 có chạy qua sạch trên Mac, chạy lại lần hai không tạo trùng không.
- **Phương pháp:** `node scripts/seed.mjs` trên stack trống, rồi các `scripts/check-*-seed.mjs`; chạy lại lần hai, so số dòng các bảng chính trước / sau.
- **Cấu hình:** stack `ep-research`, provider giả (chat + nhúng) như `pnpm dev`, `docling-serve` thật (`--profile ingest`), `SEED_RNG` 20261029, bản `9d2b02b3`, 2026-11-13.
- **Số liệu:** lần 1 mã thoát 0, 314 s (bước chậm nhất: bước 11 135 s, bước 19 78 s, bước 10 38 s); lần 2 mã thoát 0, 20 s; 10/10 kiểm tra xanh (assignments, chat, crm, docs, knowledge, exam bank, exam scores, exam demo, grade, docs timing); số dòng trước = sau (`rows_before_equals_after: true`). Nguồn: `seed-t0.json`, `seed-t0-checks.json`.
- **PASS/FAIL:** AC1, AC2 PASS. AC3 (Hôm nay từng vai), AC4 (đối chiếu DEMO_SCRIPT), AC6 (quét canary): không có số trong `benchmarks/reports/`, báo cáo này không kết luận.
- **Giới hạn:** một lượt; thời gian seed phụ thuộc `docling-serve` (bước 11, 15 cần profile `ingest`). Hạn chế Mac: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB; `docling-serve` và stack chung 6 vCPU nên thời gian seed là của máy này, không phải máy chủ; không có k6.

### US-NFR-03: dữ liệu T1
- **Câu hỏi:** bộ sinh T1 có tạo đúng quy mô thiết kế (1.000 SV, 20 lớp, một học kỳ) để các phép đo tải có nghĩa không.
- **Phương pháp:** `node scripts/seed-t1.mjs` (SQL trực tiếp, tất định theo `SEED_RNG`) vào DB riêng `edupilot_t1`; đếm dòng từng bảng, đo kích thước DB và RAM Postgres nghỉ.
- **Cấu hình:** stack `ep-t1`, quy mô 1 (không hạ), có vectơ, `SEED_RNG` 20261029. Lượt ghi trong `seed-t1.json`: bản `a96bf120`, 2026-11-15.
- **Số liệu:** 50 s; 1.009 tài khoản, 20 lớp, 1.040 ghi danh, 300 buổi, 15.000 điểm danh, 90.000 tin chat, 525.000 dòng `study_time_daily`, 4.000 bài nộp, 3.000 kết quả chấm, 100 tài liệu, 20.000 đoạn; DB 404 MB; RAM Postgres 227,5 MiB. Lúc đo EXPLAIN (13–14/11) DB ghi 409 MB (`docs/research/2026-11-14-explain.md`). Kiểm cô lập lớp (`benchmarks/check-t1-api.mjs`, bản `e006e845`, 2026-11-13): 24 cặp SV lớp X đọc lớp Y, 0 rò rỉ, 0 lỗi 5xx (`t1-api-check.json`).
- **PASS/FAIL:** AC1, AC7 PASS (đủ quy mô, không thu nhỏ); AC4 PASS (24 cặp ≥ 20, 0 rò rỉ, 0 5xx). AC2 (tất định qua hai DB), AC3 (đếm `course_id` mồ côi): không có số trong tệp nguồn, không kết luận.
- **Giới hạn:** chỉ một lượt có tệp; dữ liệu mô phỏng (D44), phân bố không lấy từ lớp thật. Hạn chế Mac: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB; thời gian seed và kích thước DB là của máy này; kiểm cô lập lớp gọi qua loopback trong VM.

### US-NFR-04: tải T1 với LLM giả, ×1 và ×2
- **Câu hỏi:** ở T1 hệ thống có giữ SLO đọc / ghi / SSE / TTFT / dashboard / chấm-khi-chat không, và nhân đôi gateway / worker có đổi gì không.
- **Phương pháp:** k6 qua `node benchmarks/load/run-t1.mjs <kịch bản> [--tag x2]`: (a) `browse` 5 phút, 300 SV rải 20 lớp, đọc : ghi ≈ 20 : 1, thêm 1 yêu cầu/giây SV gọi `gradebook` (phải 403); (b) `chat-first_event` (50 luồng) và `chat-ttft` (20 luồng); (c) `grading-baseline` rồi `grading-grading` (50 bài BT4 lớp T1-01, hai lượt mỗi bài) và chấm 1.000 bài (`benchmarks/load/grade-all-t1.mjs`); (d) `dashboard` 2 phút, 5 người.
- **Cấu hình:** stack `ep-t1`, DB `edupilot_t1`, k6 trong VM 1,5 CPU; ×1 = 1 gateway 1 worker, ×2 = `--scale gateway=2 worker=2`, không đổi mã; `benchmarks/load/k6-stack.yml` (nới hạn mức IP / người, `LLM_MAX_CONCURRENCY` 100) từ lượt thứ hai, `load-d.json` và các tệp `*-lan1` chạy trước khi bật tệp này; provider giả trễ 5–15 s ở (b) sự kiện đầu, 300 ms ở (b) TTFT và (c) 50 bài; chấm 1.000 bài dùng 9router `ag/gemini-3.8-flash`, 5 luồng.
- **Số liệu:** bảng «SLO → số đo» ở trên, dòng (a)–(d); tệp `load-a*.json`, `load-b-*.json`, `load-c-*.json`, `load-d.json`.
- **PASS/FAIL:** (a), (b), (c) 50 bài, (d) PASS ở ×1 và ×2; (c) 1.000 bài: lần 1 FAIL (322/1.000, 678 `FAILED`, 0,55 h; 9router bị khởi động lại giữa lượt, sửa ở P-11-23), lần 2 PASS (1.000/1.000, 1,68 h).
- **Giới hạn:** provider giả ở (a)–(c) 50 bài nên độ trễ không gồm LLM thật; (d) chạy trước `k6-stack.yml`. Hạn chế Mac: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB; k6 (VM 1,5 CPU) và stack chung 6 vCPU nên độ trễ là cận trên bảo thủ [INFERENCE: chưa kiểm bằng máy đo riêng]; mạng loopback trong VM nên p95 là cận dưới về phần mạng; chấm 1.000 bài gọi 9router chạy cùng máy.

### US-NFR-05: EXPLAIN 10 truy vấn nặng nhất và chỉ mục
- **Câu hỏi:** ở T1 truy vấn nào nặng nhất, có cần thêm chỉ mục không.
- **Phương pháp:** `EXPLAIN (ANALYZE, BUFFERS)` từng truy vấn qua `node benchmarks/explain.mjs before`; thử chỉ mục ứng viên trong giao dịch rồi `rollback`.
- **Cấu hình:** stack `ep-t1`, DB `edupilot_t1`, tham số đại diện lớp `T1-01`. Cách chọn 10 truy vấn: `docs/research/2026-11-14-explain.md`.
- **Số liệu:** truy vấn nặng nhất `rag_hybrid` 13,1 ms; chín truy vấn còn lại ≤ 1,3 ms; HNSW và hai chỉ mục ghép không được planner chọn hoặc không đổi kế hoạch. Nguồn: `explain.json`.
- **PASS/FAIL:** không truy vấn nào chạm SLO đọc 300 ms; không cần chỉ mục, không tạo `00018` (P-95-8).
- **Giới hạn:** tham số một lớp; văn bản truy vấn trong `explain.mjs` là bản viết lại tham số cố định của SQL ứng dụng. Hạn chế Mac: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB; thời gian là thời gian thực thi trong Postgres, không qua mạng, không có k6.

### US-NFR-06: hỗn loạn
- **Câu hỏi:** giết worker, tắt Redis, tắt `docling-serve` giữa lượt việc nền có làm mất hay lặp việc không.
- **Phương pháp:** `node benchmarks/chaos.mjs`: `kill -9` một trong hai worker giữa lượt chấm 150 bài; dừng Redis 70 s giữa lượt chấm 150 bài; dừng `docling-serve` 2 phút giữa lúc nạp 3 tài liệu; đếm việc vào / xong / mồ côi / chấm hai lần, `XPENDING`.
- **Cấu hình:** stack `ep-t1`, bản `6402ddff`, 2026-11-13; tải nền: chấm 150 bài (3 lớp × BT4) + chat, provider giả trễ 2 s mỗi lượt chấm (`K6_GRADE_LAT=2000-2000`); 2 worker ở ca `kill -9`.
- **Số liệu:** `kill -9`: 150 vào = 150 xong, 0 mồ côi, 0 chấm hai lần, `XPENDING` 0, hồi trong 113 s. Redis tắt 70 s: 150 = 150, 0 mồ côi, 0 chấm hai lần, hồi trong 129 s; `/me/today` trả 200 suốt lúc Redis tắt; chat treo ≈ 19 s rồi trả câu suy giảm (TL-2d). `docling-serve` dừng 2 phút: 3/3 `READY`, 0 mồ côi, 0 hàng mở, hồi trong 41 s. Nguồn: `chaos.json`.
- **PASS/FAIL:** PASS cả ba ca về việc nền; chat lúc Redis tắt là phát hiện TL-2d (chuyển PM).
- **Giới hạn:** mỗi ca một lượt. Hạn chế Mac: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB; thời gian hồi phụ thuộc máy này; k6 và stack chung 6 vCPU [INFERENCE: chưa kiểm bằng máy đo riêng]; mạng loopback trong VM.

### US-NFR-07: số đo nền của bản Go
- **Câu hỏi:** bản Go tốn bao nhiêu RAM, khởi động bao lâu, image lớn bao nhiêu, p95 endpoint đơn lẻ và Lighthouse ra sao.
- **Phương pháp:** `node benchmarks/baseline.mjs` (RAM nghỉ qua `docker stats`, khởi động lạnh `up → /readyz 200` × 3, kích thước image); 13 endpoint tuần tự × 200; Lighthouse 3 lần mỗi route, lấy trung vị.
- **Cấu hình:** stack `ep-t1` (DB `edupilot_t1`) lúc nghỉ, bản `6402ddff`, 2026-11-13; Lighthouse chạy bản build production, đăng nhập thật bằng tài khoản T1.
- **Số liệu:** RAM nghỉ 570 MiB cả stack (postgres 330 MiB); image gateway 17,7 MB, worker 16,7 MB, frontend 96,2 MB; khởi động lạnh trung vị 18,1 s; mọi đọc p95 ≤ 11,0 ms, ghi lớn nhất `w_thread` 327,5 ms; Lighthouse: TBT mô phỏng ≤ 36 ms, CLS ≤ 0,028, script ≤ 248 KB ở mọi route; LCP devtools `/` 1574, `/chat` 1554, `/threads` 4896, `/inbox` 6717, `/gradebook` 5887, `/me` 6256, `/settings/llm` 4786, `/assignments/[id]` 4639 ms; LCP mô phỏng 2.857–3.459 ms. Nguồn: `baseline.json`, `load-baseline-p95.json`, `lighthouse.json`.
- **PASS/FAIL:** RAM, khởi động, image: ghi nhận (không có ngưỡng). p95 endpoint PASS. Lighthouse FAIL: LCP devtools > 2.500 ms ở 6/8 route; TBT, CLS, script PASS. Không nới ngưỡng; nguyên nhân chưa điều tra trong 9.5 (nghi ngờ: LCP đợi dữ liệu API sau bước làm mới phiên; mạng devtools giả lập 150 ms).
- **Giới hạn:** Lighthouse đo trên cùng máy với stack. Hạn chế Mac: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB; RAM, khởi động lạnh là của VM này; p95 endpoint qua loopback trong VM nên là cận dưới về phần mạng; k6 và stack chung 6 vCPU nên độ trễ là cận trên bảo thủ [INFERENCE: chưa kiểm bằng máy đo riêng].

### US-NFR-08: LLM thật
- **Câu hỏi:** các việc LLM (trích công thức, chấm, che danh tính, chat, Threads, khủng hoảng, gợi ý rubric, tóm tắt, chat khi chấm) chạy với mô hình thật ra sao.
- **Phương pháp:** `benchmarks/real-llm/real_llm.py` (R1–R5) và k6 (R6); mỗi phép đo một bộ dữ liệu cố định, ghi kết quả thô vào `benchmarks/reports/real-llm/*.json`.
- **Cấu hình, số liệu:** hai đoạn và bảng R1–R6 ngay dưới đây.
- **PASS/FAIL:** theo ngưỡng của US-NFR-08: R3 lượt 2 PASS (khôi phục 35/35 ≥ 95 %; lượt 1 không đủ ≥ 20 probe); R4 sự kiện đầu p95 14 ms ≤ 300 ms PASS; R5 khủng hoảng 4/4 đi đường an toàn PASS; R6 TTFT lúc chấm không xấu đi (mẫu nhỏ, chỉ lượt Gemini trực tiếp). R1, R2 và độ đúng nội dung R4/R5: AC không đặt ngưỡng, chỉ ghi số.
- **Giới hạn:** xem dưới bảng. Hạn chế Mac: đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB; 9router chạy cùng máy với stack nên độ trễ LLM gồm đường mạng từ Mac tới nhà cung cấp, không phải từ máy chủ; R6: k6 và stack chung 6 vCPU [INFERENCE: chưa kiểm bằng máy đo riêng].

**Cấu hình LLM áp dụng từ 2026-11-14 (luật của chủ dự án):** sinh chữ (chat, chấm, trích công thức, insight, eval) chỉ qua 9router trên Mac (`http://127.0.0.1:20128/v1`; từ container `http://host.docker.internal:20128/v1`; model `ag/gemini-3.8-flash`); dựng bằng `CHAT_PROVIDER=9router PHASE=live MODEL=ag/gemini-3.8-flash BASE=https://localhost:<cổng> node deploy/vps-llm.mjs` (nhà cung cấp `router-chat`, loại `openai_compatible`, khoá lấy từ `~/.9router/db/data.sqlite` bảng `apiKeys`, không ghi vào repo). Embedding mặc định `embed-gia` (fake); `GEMINI_API_KEY` chỉ còn cho decision model và embedding khi PM cho phép (`EMBED_PROVIDER=gemini`).

**R1–R5:** đo lại qua 9router trên **MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB**, stack `ep-qc`, 2026-11-15, `9router/ag/gemini-3.8-flash` (`audit.models` trong `extract.json`, `grade-*.json`), `rpm_limit` 20, embedding giả; riêng Threads (R5) dùng nhúng thật `gemini-embedding-001` cho lớp 761987 (P-11-37, 24 request HTTP). Kết quả thô: `benchmarks/reports/real-llm/*.json` (`meta.config` ghi model + RPM); báo cáo lượt đo: `docs/sprints/9.5/report.md` mục 3, 3a. **R6:** chỉ có lượt Gemini trực tiếp (`gemini-3.1-flash-lite` Free Tier, `ep-research`, 2026-11-14; `meta.config.provider` của `load-c-*-real.json`). Số R1–R5 của lượt Gemini trực tiếp không còn tệp nguồn (lượt 9router đã ghi đè) nên đã bỏ; `real-llm/lan-1-khong-hop-le/` là lượt hỏng, không dùng (DEF-R9.5-14).

| # | Phép đo | Kết quả (thật) | Tệp |
| --- | --- | --- | --- |
| R1 | Trích công thức từ 2 quy chế, 5 lần mỗi quy chế | 10/10 `SUCCEEDED`, 9,1–15,3 s (trung vị 13,6 s); `weights_sum` 100 cả 10 lần; mọi luật `rules_ok`; 0 luật bịa; trích dẫn đúng trang 10/10 (`quy-che-lop1.md`) và 7/7 (`quy-che-lop2.md`) ở cả 10 lần | `extract.json` |
| R2 | Chấm tự luận `EVAL03` (12 bài, 3 vòng) và `BT03` lớp 1 | `EVAL03` vòng 1 / 2 / 3: MAE tổng 0,792 / 0,729 / 0,688; MAE tiêu chí 0,25 / 0,255 / 0,255; QWK 0,905 / 0,912 / 0,913; trong ±0,5: 6/12, 7/12, 9/12; cần xem 0/12; 0 bài lỗi; giữa các vòng 7/12 giống hệt, lệch TB 0,229, tối đa 1,0. `BT03`: 8 bài AI chấm (8 hàng REVIEWED từ seed đã loại, P-11-25(2)): MAE tổng 1,188, MAE tiêu chí 0,297, QWK 0,916, trong ±0,5: 2/8, 0 bài lỗi | `grade-report.json` (`grade-eval.json`, `grade-bt03.json`) |
| R3 | Che danh tính: 31 probe, placeholder tới mô hình (P01–P31) | **24 probe tới mô hình; khôi phục 35/35 giá trị (100 %, ngưỡng ≥ 95 %)**; giữ 23/23, biến đổi (hoa / thường / dịch / đánh vần / ngoặc / viết tắt / tên riêng) 12/12; còn sót placeholder 0; đổi / bỏ 0; lộ qua đánh vần 0 ở cả hai lượt (bộ dò v3 chặt nhất, chấm lại offline: mọi chuỗi ≥ 3 ký tự đơn ngăn bởi khoảng trắng / gạch / chấm / phẩy, mọi probe). 7 probe nhắc tên SV khác bị chặn `OTHER_PERSON` trước mô hình (đúng, tính riêng). Lượt 1 (24 probe): 17 tới mô hình, 24/25 (96 %), không đủ ≥ 20 probe (`report.md` 3a) | `mask.json`, `mask-luot1.json` |
| R4 | Chat: 26 tin, phân loại và sinh chữ | ý định đúng 24/26; sự kiện đầu p95 14 ms (max 15 ms); token đầu p50 21 ms, p95 4.217 ms; xong p95 4.600 ms; 0 lỗi; tin khủng hoảng 1 (Q24, nhận đúng `CRISIS`) | `chat.json` |
| R5 | Threads (5 bài), khủng hoảng (4 tin), gợi ý rubric, tóm tắt AI | Threads: **5/5 có AI trả lời** sau 6,1–12,1 s (p50 9,1 s), cả 5 có trích dẫn, 1/5 trả lời một phần (tài liệu thiếu nội dung); nhúng thật lớp 761987. Khủng hoảng: **4/4** `CRISIS`, đúng câu an toàn, không sinh chữ; sự kiện đầu ≤ 11 ms. `suggest_rubric` 3/3 tổng 10,0 (10,4–12,9 s); `ai_summary` 3/3, 0/3 chứa tên / MSSV (6,8–8,7 s). Độ đúng nội dung Threads chưa đo (không có mốc) | `threads.json`, `crisis.json`, `misc.json` |
| R6 | k6 (4 luồng chat, 40 s) khi KHÔNG chấm và khi chấm `BT03` — Gemini trực tiếp, chưa đo lại qua 9router | TTFT chat p95 nền 16.916 ms (trung vị 726 ms, 21 mẫu); lúc chấm 6.699 ms (trung vị 427 ms, 27 mẫu); sự kiện đầu p95 20 ms cả hai. Mẫu nhỏ | `load-c-baseline-real.json`, `load-c-grading-real.json` |

Phát hiện từ lượt Gemini trực tiếp (đã sửa mã, giữ lại): Gemini bỏ qua `const` của JSON schema nên `rule.params` ra sai nhánh `anyOf` và 10/10 lần trích bị Go loại (TL-2h2; sửa `internal/llm/provider/openai.go`, loại `gemini`: `const` → `enum` một phần tử; test `TestStructuredRequestShape`). `rpm_limit` đặt cao hơn hạn mức thật của khoá → 429 → cầu dao mở → mọi lời gọi sau bị loại ngay (`circuit_open`), nên một lượt chấm lớn có thể làm hỏng cả chat cùng lúc (TL-2j).

## Chưa đo được và vì sao
| Việc | Lý do |
| --- | --- |
| Đo LLM thật trên khoá trả phí, nhiều provider (E4) | chỉ có khoá Free Tier; thuộc P10 |
| Judge amd64 có seccomp | không đo được trên arm64 (D58) |
| E3 chấm tay, E4 nhiều provider | thuộc P10 |
| Mạng thật / VPS | ngoài phạm vi (CONTEXT mục 4) |

## Lần đo bị nhiễu, lần đo lại
**CPU nền: không đo.** Không lượt nào lấy mẫu CPU của máy Mac hay của VM trước / trong khi đo, nên báo cáo không biết lượt nào có CPU nền > 30 %, và không có lượt nào bị loại hay đo lại vì lý do đó. Dấu vết duy nhất về tải nền là trạng thái stack demo `edupilot` trong `meta.demo_stack` của `load-*.json`: `stopped` ở mọi tệp, trừ ba tệp ghi `running`: `load-c-baseline-real.json`, `load-c-grading-real.json` (R6) và `load-c-1000-9router-run1-hong.json`. Ba lượt này có thể bị nhiễu bởi stack demo; mức nhiễu không đo.

Các lượt bị lặp hoặc không dùng đều có lý do khác và được giữ tệp, không xoá:

| Tệp | Lý do lặp / không dùng | Lượt thay |
| --- | --- | --- |
| `load-a-lan1.json`, `load-a-x2-lan1.json` | lần đầu, trên gateway dev thường (hạn mức IP 5.000/phút), chưa có bước gọi `gradebook` phải 403 | `load-a.json`, `load-a-x2.json` |
| `load-c-1000-lan1-khong-hop-le.json` (và `load-c-baseline-1000-lan1.json`, chat nền đi kèm, cùng bản `aafd87f0`, tối 13/11) | lỗi script đo: `Idempotency-Key` trùng giữa 20 lớp nên chỉ 150 việc được xếp (`note` trong tệp) | `load-c-1000-9router-run2.json`, `load-c-baseline-1000.json` |
| `load-c-1000-dung-theo-chu-du-an.json` | provider giả trễ 20 s (`meta.config`); dừng ở 235/1.000 vì AC9 đòi chạy LLM thật (`note` trong tệp) | lượt 9router |
| `load-c-1000-9router-run1-hong.json` | 9router bị watchdog khởi động lại giữa lượt, 678 `FAILED`; lỗi mã, sửa ở P-11-23 | `load-c-1000-9router-run2.json` |
| `real-llm/lan-1-khong-hop-le/` | lượt Gemini trực tiếp hỏng (`circuit_open`, `RATE_LIMIT`), DEF-R9.5-14 | `real-llm/*.json` (9router, 2026-11-15) |
| `real-llm/mask-luot1.json` | chỉ 17 probe tới mô hình (cần ≥ 20) | `real-llm/mask.json` |

## Danh sách câu cần vá ở báo cáo cũ (US-NFR-09 AC3)
| Vị trí | Câu hiện có | Câu mới đề xuất |
| --- | --- | --- |
| `docs/sprints/8/report.md`, `docs/sprints/9/report.md`, `PROGRESS.md` (các dòng "giả lập" / "chưa đo" về T1) | "chưa đo T1" | "Đo trên MacBook Apple Silicon (Colima 6 CPU / 10 GB): T1 1.000 SV đạt SLO đọc / ghi / SSE / TTFT / dashboard ở ×1 và ×2; số trong `docs/research/2026-11-14-nfr-real-llm.md`" |
| `PROGRESS.md` mục Nợ | – | thêm: TL-2c (truy vấn tìm lai), TL-2d (chat treo khi Redis tắt), TL-2e (nhúng trong request tạo thread) |
