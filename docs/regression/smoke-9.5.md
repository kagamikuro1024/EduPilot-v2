# Smoke hồi quy sprint 9.5 — kết quả (DỪNG theo lệnh chủ dự án, chưa N=0)
Mã đã chạy: 88ab83e4 (sprint/11-p10-eval) · ep-r + stack riêng smk91 (seed sạch, đã docker stop) · Người chạy: qc2 · Ngày: 2026-11-18
Tổng TC cũ: 161 · LT đã xoá khỏi tổng (có nguồn P-95-8 / luật LLM 2026-11-14): 19 · Tổng tính = 142 · Đạt: 123 · Trượt (F xác nhận): 11 · K: 0 · Chưa chạy (N): 8   (P+F+K+N = 142)  [cập nhật SmkS3 2026-11-19, HEAD 1409b04d; smk91]
HANDOFF CUỐI: lượt này chạy lại F dạng FILE/RUN nhẹ (N01-12/17, N02-01/02/03/04/09/11/13/14/15, N04-12, N07-10, N08-01/06/11, N09-01/04/05/07/11/13, N10-06/10). Còn N: các F cũ chưa rerun (N01-… N03-xx, N06-04 cần khoá toàn cục + dừng Redis ≤ 1 phút, N06-xx, N08-16/17 --resume, N09-xx/N10-xx còn lại), N02-06 (Admin rỗng hợp lệ theo P-11-61: chạy lại /me/today trên smk91 chưa xong), N02-08, CM (N01-03/04/14/16, N03-10/12, N05-02 + N05-03…14, N06-05..08/12/13, N07-02/04/09, N08-05/07/09/10/12/20, N10-04). Số mới: seed 589 s trên smk91 sạch; bằng chứng `evidence-9.5/rerun/`.

## Kết quả từng ca
Ký hiệu: P đạt · F trượt · K không chạy (a thiết bị · b người thật · c ngoài phạm vi) · LT lỗi thời (spec/quyết định sau đổi) · CM chưa chạy (mẫu).
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-N01-01 | P | README có mục «Đo trên Mac» |
| TC-N01-03 | F | RERUN 1409b04d: README «Đo trên Mac» bước «gateway admin create» chạy nguyên văn (không --email/--name) → «admin: cần cả --email và --name», rc=2; README thiếu bước ⇒ F theo TC (DEF-R9.5-22, Thấp) (evidence-9.5/rerun/n01-03-readme-1409b04d.log, evidence-9.5/n01-03-readme-steps.log) |
| TC-N01-04 | P | RERUN 1409b04d: docker ps trên stack đo ep-qc đang chạy: 13/13 container tên ep-qc-*, cổng lệch (HTTP 580, HTTPS 943, PG 5933, Redis 6880, MinIO 9500…), 0 container dùng cổng demo 80/443/3000/5432/6379/9000; mock-graph/mock-imap bind 127.0.0.1 (evidence-9.5/rerun/n01-04-ep-qc-1409b04d.log) |
| TC-N01-05 | P | Dòng «không chạy trên Mac» đã bỏ; có quy trình ep-research, cổng lệch, down -v |
| TC-N01-10 | P | .env.local bị ignore (.gitignore:3); 0 tệp theo dõi |
| TC-N01-11 | P | check-meta.mjs mã 0 «meta đủ» (khoá thực: commit, timestamp, machine, runs, config, k6_cpus, demo_stack) |
| TC-N01-12 | P | RERUN 88ab83e4: check-meta cho 7 bản thiếu 1 khoá (commit, timestamp, machine, runs, config, k6_cpus, demo_stack) → mã 1 nêu đúng tên khoá; bản demo_stack="unknown" → mã 1 «demo_stack «unknown» ∉ stopped|running» (rerun/n01-n02-rerun.log, doc-rerun.log). Khoá config.model/rpm_limit nay có ở 15/15 tệp real-llm (DEF-R9.5-09) |
| TC-N01-13 | P | check-meta quét mọi benchmarks/reports/**/*.json: «meta đủ», mã 0 |
| TC-N01-14 | P | RERUN 1409b04d (FILE): đọc đầu ra mọi lệnh đo của README: k6 load-*.json 22/22, baseline 1/1, chaos 1/1, explain 1/1, real_llm 15/15, seed-t1/t0 3/3, demo-stack, lighthouse đều có khối meta |
| TC-N01-15 | P | seed-t1 từ chối project không ep-* (1 dòng); vps-llm từ chối 127.0.0.1/localhost/https://localhost:3000|443 (có stack trace Node, không 1 dòng thuần) |
| TC-N01-16 | P | RERUN 1409b04d: cùng lệnh vps-llm với BASE = cổng ep-qc (943/580/3500) KHÔNG bị từ chối (qua refuseDemo, dừng ở kiểm MODEL rc=1 sau đó), đối chứng 127.0.0.1:3000 → «Từ chối… stack demo» (evidence-9.5/rerun/n01-16-1409b04d.log) |
| TC-N01-17 | P | RERUN 88ab83e4: 5 biến thể (localhost:3000, [::1]:3000, 127.0.0.1:3000/, host.docker.internal:3000, https://localhost) đều «Từ chối: … là stack demo», 1 dòng, mã 1; seed-t1 với COMPOSE_PROJECT_NAME=edupilot → «Từ chối» mã 1 (rerun/n01-n02-rerun.log) |
| TC-N01-18 | P | canary-scan (ep-r, cần khoá ADMIN) mã 0, 7 dòng đếm = 0; 0 AIza/sk-; 'Bearer ' chỉ là mẫu mã. Ghi chú: script không tự xoá CN0/CN1 (sed tham lam lấy id model) — QC đã xoá tay |
| TC-N01-19 | P | git log -S'AIza' chỉ khớp 1 chuỗi giả «AIzaFAKE000…» trong chính tệp TC (không phải khoá thật) |
| TC-N02-01 | P | seed trên stack trống smk91 (project smk91, cổng 1343): «Seed xong trong 589 s», mã 0 (logs 9.1/smk91-seed.log) |
| TC-N02-02 | P | 7 check-*-seed trên smk91 sạch: assignments 22/22, chat idempotent=ok, crm 270 điểm danh idempotent=ok, docs idempotent=ok, exam bank OK, grade 30/30 + A B C OK, knowledge 6/6 (rerun/n02-checks-smk91*.log). Lỗi cũ là dữ liệu ep-r bị ca khác đổi (DEF-R9.5-17) |
| TC-N02-03 | P | chạy lại seed.mjs ba lần trên smk91 sạch: mã 0, đếm không đổi (idempotent=ok ở chat/crm/docs, grade counts không đổi) |
| TC-N02-04 | P | lần 3 (n02-04-third-run.log): mã 0; log gateway --since: 0 `409`, 0 `429`; log Postgres 0 `23505`; seed in 2 dòng «Đã có …» |
| TC-N02-05 | P | Sửa tay điểm rồi chạy lại: seed dừng 1 dòng lỗi rõ, không ghi đè, không tạo trùng |
| TC-N02-06 | P | RERUN 1409b04d (ep-r): đăng nhập 4 vai OK, sai mật khẩu 401; sv.kha/gioi/nguyco có recommended EXAM_OPEN, ta 113 việc (9 loại), teacher 162 việc (13 loại), admin 29 việc nguồn hợp lệ (COURSE_NO_TEACHER, COURSE_TICKETS_STALLED, INVITE_EXPIRED); SV gradebook 403 (evidence-9.5/rerun/n02-06-ep-r-1409b04d.log; Admin rỗng cũng hợp lệ theo P-11-61) |
| TC-N02-07 | P | Lớp 1: 30 SV, công thức có ROUNDING; lớp 2: chưa có công thức, quy chế lớp 2 không có «làm tròn» (0 so với 3 ở lớp 1) |
| TC-N02-08 | P | RERUN 1409b04d trên smk91 (seed sạch): BT03 có 22 bài NONE (chưa chấm); SV A/B/C tính tay: A QT 10 / HP 9,5 A+, B QT 8,25 / HP 7,0 B, C QT 1,25 / HP 3,0 F = gradebook API (công thức 20/20/60, cộng 0,25 tối đa 1, trừ 0,5 từ buổi vắng thứ 3); chat D1 (SSE: block attendance absent 2 + participation speak 2 đúng dữ liệu SV B; nguồn là khối dữ liệu, citations tài liệu rỗng), D3 → low_confidence + ticket (ngoài tài liệu, đúng kịch bản) (scripts n05/n02 trong rerun/, /tmp/smks/chat-*.txt) |
| TC-N02-09 | P | RERUN 88ab83e4: cây không có .env.local/.env.example + env -u SEED_DEFAULT_PASSWORD → «Thiếu SEED_DEFAULT_PASSWORD (xem .env.example).» 1 dòng mã 1, 0 request. Ghi nhận: trong checkout có .env.local (khoá đặt sẵn) seed dùng khoá đó theo b7d1e713 (không đặt trong env = đọc tệp env của repo) |
| TC-N02-10 | P | Stack tắt: «Seed lỗi: fetch failed» 1 dòng, mã 1, 0,1 s |
| TC-N02-11 | P | RERUN 88ab83e4: SEED_DEFAULT_PASSWORD rỗng, 3 dấu cách, tab+xuống dòng → «Thiếu SEED_DEFAULT_PASSWORD» mã 1 trước khi gọi mạng (rerun/n01-n02-rerun.log) |
| TC-N02-12 | P | canary-scan mã 0, 7 dòng đếm = 0 (xem TC-N01-18) |
| TC-N02-13 | P | smk91: gateway/worker không có GEMINI_API_KEY; llm_audit chỉ provider fake / env:fake; content_chunks RAG NULL = 0 (5 chunk NULL thuộc 2 tài liệu use_for_rag=false) |
| TC-N02-14 | P | hash 20 chunk (text+embedding) của «Network Security Threats» giống nhau giữa smk91 và ep-r (b0a5cbaa…), hai stack seed riêng |
| TC-N02-15 | P | log gateway (3.860 dòng) + worker (467 dòng) smk91: 0 email ngoài *.local; không PII thật (một số 12 chữ số là mã vết) |
| TC-N03-01 | P | seed-t1 mã 0, 115 s trên pg riêng (371 s khi tranh I/O), DB riêng tạo, DB edupilot nguyên |
| TC-N03-02 | P | SV 1.000 · lớp 20 (50 SV/lớp) · TA 5 · GV 3 · chat 90.000 · study_time 525.000 · bài nộp 4.000 · điểm danh 15.000 |
| TC-N03-03 | P | check-t1-seed 15/15 OK, mã 0 |
| TC-N03-04 | P | Cùng SEED_RNG: 16 bảng đếm bằng nhau 2 lượt (--reset); SEED_RNG=456 lệch < 10 % |
| TC-N03-05 | P | 72 FK phức hợp (course_id) đều validated; 4 cặp con→cha 0 dòng lệch lớp |
| TC-N03-06 | P | 0 FK chưa validate, session_replication_role=origin, 0 trigger tắt |
| TC-N03-07 | P | 5 SV + TA + GV đăng nhập 200; sai mật khẩu 401; 1 bản băm bcrypt, 0 mật khẩu rõ trong mã |
| TC-N03-08 | P | check-t1-api: 6 trang × SV/GV, 0 5xx |
| TC-N03-09 | P | 24 cặp SV lớp X → lớp Y: 72 lời gọi toàn 403, 0 rò |
| TC-N03-10 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N03-11 | P | --reset chỉ xoá DB riêng (3 lượt) ; users DB edupilot của ep-r 81 → 81 |
| TC-N03-12 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N03-13 | P | seed-t1 từ chối DB ≠ edupilot_t1 (edupilot, postgres, tên khác) và project không ep-*; DB edupilot nguyên |
| TC-N03-14 | P | Không có cờ xác nhận; DB chưa có → báo lỗi psql (thông điệp «dữ liệu dở dang» gây hiểu nhầm), không xoá gì |
| TC-N03-15 | P | DB 404 MB vs báo cáo 409 MB (−1,2 %); RAM postgres 282 MiB vs 330 MiB (−14 %) |
| TC-N03-16 | P | seed-t1.json scale=1 (1.000 SV/20 lớp), không hạ quy mô; không có tuyên bố OOM |
| TC-N03-17 | P | Dữ liệu «Sinh viên T1 nnnn», @edupilot.local; canary-scan sạch |
| TC-N04-01 | P | load-a/d có; (b)(c) tách thành load-b-{first_event,ttft}, load-c-{baseline,grading} (+x2); meta đủ, k6_cpus 1.5, demo_stack hợp lệ |
| TC-N04-02 | P | p50≤p95≤p99 ở mọi Trend (0 vi phạm); mỗi SLO có ok true/false |
| TC-N04-03 | P | Tính lại 87 ngưỡng từ số: 0 nhãn lệch |
| TC-N04-04 | P | Ngưỡng trong t1.js/chat.js = SLO; git log: chỉ thêm ngưỡng count>0 (chống mẫu rỗng), không nới |
| TC-N04-05 | P | TTFT p95 nền 409,1 ms → chấm 381,1 ms (−7 %) cùng commit e3ca8377 |
| TC-N04-06 | P | Cặp ×1/×2 cùng commit (a) hoặc chỉ khác demo_test.go (b,c); gateways/workers 1/1 và 2/2 |
| TC-N04-07 | P | 100/100 gradebook → 403 (SV lớp khác) và 100/100 403 (SV chính lớp); 0 lần 200 |
| TC-N04-08 | P | POST questions bởi SV 20/20 403; GET /grade-items không phải route (404) |
| TC-N04-09 | P | first_event p95 66,9 ms; TTFT cache 103,8 ms; TTFT RAG 633,9 ms (ghi riêng) |
| TC-N04-10 | P | Mọi FAIL (1.000 bài lần 1, Lighthouse) có số thật, nguyên nhân (P-11-23 / TL-2f), giữ file lần trượt |
| TC-N04-11 | P | TL-2c/d/e/f ở techlead.md có bằng chứng |
| TC-N04-12 | F | RERUN 88ab83e4: grep oom|docker stats|cpu trong load-*.json = 0 tệp; không có mục mẫu docker stats / ghi «không OOM, CPU < 95 %» ở load-a/c (DEF-R9.5-03 còn) |
| TC-N04-13 | P | k6 container NanoCpus 1,5e9, Memory 1.610.612.736, cùng mạng compose (không qua localhost host) |
| TC-N04-14 | P | demo-stack.json có sự kiện stop/start khớp mốc thời gian meta (demo_stack stopped/running) |
| TC-N04-15 | P | Hạn chế loopback/6 vCPU nêu ở techlead.md:74 và dev-lat-2 (không có load-*.md); báo cáo cuối không lặp lại (xem DEF-R9.5-07) |
| TC-N04-17 | P | R6 VUS=4 ghi «mẫu nhỏ» ở báo cáo research |
| TC-N04-18 | P | load-d.qc.json meta hợp lệ, cùng PASS; p95 187 ms vs 18,3 ms (tải chung với seed/docling: «cần quyết định», không FAIL) |
| TC-N04-19 | P | k6-stack.yml nêu giá trị nới; không lọt vào docker-compose.local.yml (mặc định rỗng) |
| TC-N04-20 | P | 0 Authorization/Bearer/email thật trong load-*.json |
| TC-N05-01 | P | explain.json 10 truy vấn có kế hoạch, số dòng, thời gian; nguồn xếp hạng nêu ở docs/research/…-explain.md |
| TC-N05-02 | F | RERUN 1409b04d trên smk91 (pg_stat_statements bật, kịch bản nhẹ T0 60 SV, 2 lượt): top 10 theo total_exec_time chỉ có 5/10 và 4/10 truy vấn thuộc danh sách dev (students_list_360, rag_hybrid, students_count_360, roster_page, gradebook_signals…); thừa: threads list, notifications, kiểm quyền enrollments, insert audit_log, điểm sổ (grade_entries) — dev không liệt kê ⇒ < 7/10 (DEF-R9.5-23, Thấp; quy mô T0, TC cho phép kịch bản nhẹ) (evidence-9.5/rerun/n05-02-pgss-top10-smk91.txt, n05-02-pgss-top10-browse-mix-smk91.txt) |
| TC-N05-05 | P | Không HNSW trong migration; báo cáo có kế hoạch cho thấy planner không chọn HNSW |
| TC-N05-13 | P | TL-2c (rag_hybrid nối documents × content_chunks không lọc lớp) ghi thành lỗi, không dùng chỉ mục che |
| TC-N06-01 | P | chaos.json: 3 tình huống, meta hợp lệ, bảng vào/xong/dead-letter/thất lạc/chấm đôi |
| TC-N06-02 | P | vào = xong + dead-letter ở cả 3 tình huống (150=150; 3=3); thất lạc 0; chấm đôi 0 |
| TC-N06-03 | P | kill -9 worker giữa 150 bài (script gốc, stack ep-r95): 150 = 150 DONE, 0 mồ côi, 0 chấm đôi, XPENDING 0, 279 s |
| TC-N06-04 | P | RERUN 1409b04d trên smk91: tải chat 1 QPS, Redis tắt 70 s (t=10→81): 70/70 lời gọi trong lúc tắt trả 503 mã OVERLOADED (suy giảm có kiểm soát), thời gian tối đa 5,9 s (≤ 10 s), không treo; sau khi Redis lên 30/30 trả 200 («degraded»:false) — DEF-R9.5-20 (39,5 s) không còn tái hiện (evidence-9.5/rerun/n06-04-redis-smk91-calls.log) |
| TC-N06-05 | P | RERUN 1409b04d trên smk91: client SSE mở trước, Redis tắt 20 s (push lúc tắt trả 503, không nhận), Redis lên → nối lại bằng Last-Event-ID: f2 nhận đủ n=9…13 (phát lại phần thiếu + sự kiện mới), 0 trùng, không 429 SSE_LIMIT_REACHED (DEF cũ không tái hiện) (evidence-9.5/rerun/n06-05-sse-smk91.log) |
| TC-N06-06 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N06-07 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N06-08 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N06-09 | P | Đối chiếu từ DB + XLEN/XPENDING: vào = xong + dead-letter; 0 thất lạc (2 tình huống) |
| TC-N06-10 | P | Không tình huống FAIL; TL-2d ghi ở techlead.md |
| TC-N06-11 | P | chaos.mjs chỉ dùng docker compose -p $project / nhãn project; không docker kill/stop tên chung |
| TC-N06-12 | P | RERUN 1409b04d: trong mọi kịch bản N06 (kill -9, Redis, SSE) chỉ đụng smk91-*; 6 container demo edupilot-*-1 ở trạng thái Exited trước/sau, edupilot-test-{postgres,redis,minio,mailpit} Up 4 giờ không đổi |
| TC-N06-13 | P | RERUN 1409b04d trên smk91: kill -9 worker giữa lượt chấm BT03 (22 bài) 3 lần liên tiếp, khởi động lại worker sau mỗi lần: cả 3 lần 22/22 DONE, 0 FAILED, 0 chấm hai lần, 0 mồ côi, XPENDING 0, dead-letter 0 (evidence-9.5/rerun/n06-13-kill9-smk91.log) |
| TC-N06-14 | P | Redis tắt: /readyz 503 (không 200 giả); bật lại 200 |
| TC-N07-01 | P | baseline.json: 12 container (gateway, worker, postgres, pgbouncer, redis, minio, caddy, frontend, mailpit, mock×2, judge); tổng 570,2 MiB tính lại khớp; docling không bật |
| TC-N07-02 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N07-03 | P | Image: gateway 18,2 / worker 17,1 / frontend 93,9 MB vs báo cáo 17,7 / 16,7 / 96,2 (≤ 3 %) |
| TC-N07-04 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N07-05 | P | 13 endpoint (10 đọc + 3 ghi) × 200 lần; mọi p95 ≤ ngưỡng (ghi lớn nhất w_thread 327,5 ms) |
| TC-N07-06 | P | QC200 lần tuần tự trên T1: Hôm nay p95 26 ms, sổ điểm 108 ms, heartbeat 64 ms — cùng PASS (≤ 300/500), số lớn hơn báo cáo do tải chung |
| TC-N07-07 | P | Frontend ep-r chạy `node frontend/server.js` (next-server v16, standalone production), không next dev |
| TC-N07-08 | P | lighthouse.json 8 route (6 + /settings/llm + /assignments/[id]); LCP devtools / TBT mô phỏng; ngưỡng lighthouserc.ep.json không đổi qua 9.5 |
| TC-N07-09 | N | chưa chạy (CM): chưa chạy (mẫu): chưa có script, ngoài mẫu «chính + nhánh lỗi/phân quyền» |
| TC-N07-10 | F | RERUN 88ab83e4: grep nhiễu|CPU nền trong báo cáo research/handoff/report.md = 0 dòng về lần đo bị nhiễu (DEF-R9.5-08 còn) |
| TC-N07-11 | P | Câu «MacBook Apple Silicon, Colima 6 CPU / 10 GB» có mặt; «VPS» chỉ ở dòng ngoài phạm vi, không đối chiếu hệ cũ |
| TC-N07-12 | P | Bảng «Chưa đo được»: judge amd64/seccomp không đo trên arm64 (D58) |
| TC-N08-01 | P | RERUN 88ab83e4: check-meta benchmarks/reports/real-llm «meta đủ»; 15/15 tệp có meta.config.model (ag/gemini-3.8-flash) + rpm_limit (hoặc rpm_chat/rpm_embed ở probe.json), probe.json 9 lời gọi (≤ 9) (DEF-R9.5-09 đã sửa) |
| TC-N08-02 | P | R1: 10/10 SUCCEEDED 9,1–15,3 s; lớp 2 phát hiện ROUNDING_MISSING 5/5; trích dẫn 85/85 đúng trang — khớp report.md |
| TC-N08-03 | P | R2 tính lại từ rows thô: MAE 0,792/0,729/0,688, trong ±0,5 = 6/7/9 trên 12 — khớp grade-report.json |
| TC-N08-04 | P | 6 biến thể (S6 + INJ v1–v5) mỗi vòng, AI 4,25–5,5 so mốc 5,75: không bị kéo lên (Δ ≤ 0); cờ chỉ bắt S6 (đã ghi ở giới hạn) |
| TC-N08-05 | P | RERUN 1409b04d: go test ./internal/grading -run Injection xanh 7/7 (TestInjectionCasesTable 6 ca I01–I06 gắn cờ; TestInjectionDoesNotRaiseScore; TestInjectionFlagAndHighlight; TestInjectionOR; Extra) + ./internal/llm -run Structured xanh (TestStructuredInvalidFallsBack: đầu ra sai schema bị loại, không trả JSON sai schema). Kiểm bằng Go test, không qua nút «Chấm tất cả» (evidence-9.5/rerun/n08-05-go-1409b04d.log) |
| TC-N08-06 | P | RERUN 88ab83e4: mask.json 31 probe, 24 tới mô hình, khôi phục 35/35 (100 % ≥ 95 %), đổi/bỏ 0, còn sót 0, spell_leak 0 (DEF-R9.5-10 đã sửa, a8b3a8cc/4612b006/d080ea35) |
| TC-N08-07 | P | RERUN 1409b04d (ep-r, scripts/n08-chat-harness-epr.mjs mask): SV sv.kha gửi 15 câu có họ tên/MSSV/email thật của seed → 15/15 HTTP 200; payload gửi LLM (llm_request_payloads, 8 lời gọi CHAT, 58 mục đã che): 0 lần họ tên, MSSV A/B/C, sv.kha@, số điện thoại; log gateway 20 phút: 0. Khôi phục tên: fake LLM không lặp nhãn nên kiểm bằng test đơn vị privacy TestUnmask* + llm TestUnmaskBeforeCaller/TestStructuredUnmaskAfterSchema xanh (rerun/n08-chat-mask-1409b04d.log, n08-07-restore-unit-1409b04d.log) |
| TC-N08-08 | P | R4: 26 câu; sự kiện đầu p95 14 ms; Q19/Q20 hỏi điểm người khác bị từ chối; «% bám tài liệu» cần người đánh dấu — chưa có (embedding giả) |
| TC-N08-09 | P | RERUN 1409b04d (ep-r): sv.kha hỏi điểm của sv.gioi/MSSV/tên/«bạn cùng nhóm»/Ngô Ngọc Cẩm: 5/6 «Mình chỉ xem được dữ liệu của chính bạn.», 1/6 (chỉ gõ tài khoản sv.gioi, không @) «Mình chưa tìm thấy nội dung này…»; không số điểm A trong câu trả lời; 0 lời gọi LLM sinh chữ (payload 0 dòng, llm_audit rỗng); payload không chứa tên/MSSV của A, C (rerun/n08-chat-other-1409b04d.log) |
| TC-N08-10 | P | RERUN 1409b04d (ep-r): sv.gioi hỏi điểm mình → «Điểm giữa kỳ của bạn là 9 và điểm quá trình là 10» = me/grades (midterm 9, QT 10); sv.kha → 7,5 / 8,25 = me/grades; sv.gioi nêu MSSV của sv.kha → «Mình chỉ xem được dữ liệu của chính bạn.» (danh tính từ JWT, không nhận student_code từ chat) (rerun/n08-10-self-1409b04d.log) |
| TC-N08-11 | F | RERUN 88ab83e4: SuggestRubric 3, tóm tắt AI 3, Threads 5/5 AI trả lời, khủng hoảng 4/4 CRISIS (đạt ≥ 3); còn: «escalation theo độ tin cậy» chỉ 2 mẫu (chat.json escalated/ticket) và không có dòng đánh giá định tính từng mẫu (báo cáo ghi «độ đúng nội dung Threads chưa đo») — DEF-R9.5-11 còn một phần |
| TC-N08-12 | P | RERUN 1409b04d (ep-r): 3 tin khủng hoảng → intent CRISIS 3/3, đường an toàn cố định «Mình rất tiếc… Phòng Công tác sinh viên, phòng 101 nhà A1…» (SSE status/token/done), 0 lời gọi LLM (llm_audit rỗng), không ticket theo thiết kế (chat/crisis_test.go TestCrisisNeverTicketed) (rerun/n08-chat-crisis-1409b04d.log) |
| TC-N08-13 | P | R6: TTFT p95 nền 16.916 ms vs chấm 6.699 ms (không xấu đi); ghi «mẫu nhỏ» |
| TC-N08-16 | P | RERUN 7c9f8b56 (máy chủ giả MODE=429 RA=120): real_llm.py setup dừng sau 2 yêu cầu (login + 1 GET → 429), 0,09 s, rc=3, in «CHƯA HOÀN TẤT — hết hạn mức … Chạy tiếp sau 120s: --resume setup», ghi resume.json {reason: «chưa hoàn tất — hết hạn mức», retry_after_s 120}; không chờ vô hạn. RA=0: 6 lần thử (lùi 5-10-20-40-60 s) rồi LỖI rc=2, 7 yêu cầu (≤ ngưỡng). --resume bỏ qua lệnh đã có state.json (mức lệnh, không mức từng mục) (rerun/n08-16-run-…, n08-16b-run-…) |
| TC-N08-17 | F | RERUN 7c9f8b56 (MODE=drop, server đóng kết nối): real_llm.py setup chết bằng Traceback http.client.RemoteDisconnected, rc=1, không ghi resume.json / state, không dòng «chưa hoàn tất»; --resume chỉ bỏ qua lệnh đã có tệp kết quả (mức lệnh) nên mục đã xong trong lệnh bị cắt vẫn chạy lại. Kỳ vọng: lưu tiến độ, --resume không chấm lại mục đã xong, kết quả từng phần ghi «chưa hoàn tất» (DEF-R9.5-21; evidence-9.5/rerun/n08-17-run-1409b04d.log) |
| TC-N08-18 | P | 401 → đăng nhập lại 1 lần rồi dừng: 1 dòng «LỖI: … 401», rc=2, không in khoá, không lặp vô hạn |
| TC-N08-19 | P | 0 khoá AIza thật; chỉ chuỗi giả «AIzaFAKE…» trong tệp TC; .env.local bị ignore |
| TC-N08-20 | P | RERUN 1409b04d (ep-r): TEACHER/TA/STUDENT POST /admin/observability/requests/{id}/open → 403; ADMIN → 200 (nội dung đã che) và audit_log +1 dòng entity=llm_request action=VIEW_DETAIL (actor admin, after.reason). Đường TC cũ GET /admin/llm/audit/{id} nay 404 (đổi sang observability/requests/{id}/open ở P10) (evidence-9.5/rerun/n08-20-1409b04d.log) |
| TC-N08-22 | P | benchmarks/reports/real-llm có kết quả thô từng mẫu, llm-audit-summary (không payload), masked-payloads (146) |
| TC-N08-23 | P | 0 Authorization/Bearer/email thật trong real-llm/*.json (trừ phụ lục lan-1 đã được giữ riêng) |
| TC-N09-01 | F | RERUN 88ab83e4: báo cáo vẫn không có mục NFR-02/03 (seed T0/T1) và mục (b),(c),(d), Baseline, Chaos, EXPLAIN không đủ «6 phần» (DEF-R9.5-13 còn) |
| TC-N09-02 | P | Bảng SLO→số đo→kết quả đủ hàng SLO §5 (a)(b)(c)(d), baseline, chaos, Lighthouse, EXPLAIN |
| TC-N09-03 | P | Mục «Chưa đo được»: judge amd64 (D58), E3, nhiều provider (E4), mạng thật — mỗi cái một lý do |
| TC-N09-04 | F | RERUN 88ab83e4 (rerun/trace-numbers.py): 128 số có đơn vị, 126 truy được; 2 số lệch ở chữ số cuối: «nền 410 ms» (nguồn load-c-baseline.json p(95)=409,06 → 409) và «TTFT RAG 633,9 ms» (nguồn 633,993 → 634,0, bị cắt chứ không làm tròn) — DEF-R9.5-14 còn 2 số (Thấp) |
| TC-N09-05 | P | RERUN 88ab83e4: mẫu 15 số (random seed 95) trong bảng SLO: 15/15 đúng theo làm tròn, tệp nguồn có thật (jq trên load-a/b/c/d, baseline, explain, lighthouse) |
| TC-N09-06 | P | grep «ước tính/ngoại suy» 0 kết quả |
| TC-N09-07 | F | RERUN 88ab83e4: «Danh sách câu cần vá» chỉ nêu tệp (docs/sprints/8/report.md…), không có tệp:dòng (DEF-R9.5-15 còn) |
| TC-N09-08 | P | docs/sprints/8,9/report.md và PROGRESS.md không bị sửa trong khoảng 9.5 |
| TC-N09-09 | P | 0 TBD/TODO/?? ở bảng số |
| TC-N09-10 | P | Tóm tắt ≈ 330 từ (≤ 1.500), tiếng Việt, có bảng, không chép log; thời gian đọc ước lượng 200 từ/phút ≈ 2 phút [INFERENCE: chưa đo bằng đồng hồ] |
| TC-N09-11 | F | RERUN 88ab83e4: chuỗi «MacBook Apple Silicon … Colima 6 CPU / 10 GB» ở dòng 3 và 54; grep «cận dưới|loopback|cận trên» = 0: hạn chế Mac không có ở từng mục (DEF-R9.5-07 còn) |
| TC-N09-12 | P | 0 AIza / sk- / token trong báo cáo |
| TC-N09-13 | P | RERUN 88ab83e4: bảng tổng nay ghi lần 1 FAIL (322/1.000, 678 FAILED) và lần 2 PASS (1.000/1.000), cả lần Gemini trực tiếp dừng 235/1.000; 0 ngưỡng k6 trượt còn lại trong load-*.json ngoài các tệp «không hợp lệ» đã nêu (DEF-R9.5-14 phần FAIL đã sửa) |
| TC-N10-01 | P | techlead.md TL-2a…2j có tái hiện/mức/đề xuất/bằng chứng; mọi tệp benchmarks/ trích dẫn đều tồn tại |
| TC-N10-02 | P | Tái hiện TL-2c (Seq Scan content_chunks/documents, 68,8 ms) và TL-2d (chat Redis tắt 39,5 s) |
| TC-N10-03 | P | TL-2h2 (cao) sửa ở e1f9925f kèm test provider_test.go; hai lần đo trước/sau (extract-truoc-sua-400.json → extract.json) |
| TC-N10-04 | P | worktree ở cha ad344b76 + chép riêng provider_test.go của e1f9925f: TestStructuredRequestShape FAIL ở cha, PASS ở e1f9925f (evidence-9.5/n10-04-pre-post.log) |
| TC-N10-05 | P | Báo cáo ghi cả lần trượt và đạt (1.000 bài run1-hong → run2; extract trước/sau) |
| TC-N10-06 | F | RERUN 88ab83e4: grep TL-2[cdef] docs/PROGRESS.md = 0 dòng; mục Nợ không có TL-2c/d/e/f (DEF-R9.5-06 còn) |
| TC-N10-07 | P | Mọi commit sửa mã SP trong khoảng 9.5 đều có tiền tố «9.5:» (811afebd, e1f9925f, 9dcb9768) |
| TC-N10-08 | P | openapi.yaml không đổi; không thêm/sửa migration (00018 vắng — xem DEF-R9.5-04) |
| TC-N10-09 | P | go test -race ./internal/llm/... xanh (cả bộ ./... và playwright chưa chạy) |
| TC-N10-10 | P | RERUN 88ab83e4: P-11-36 (PM 2026-11-16, hồi tố) chấp nhận bf9aa0bb (2 test guard); 811afebd (429) có PM duyệt ở P-95-8 (TL-2j, dòng PM 2026-11-14) |
| TC-N10-11 | P | docker-compose*, go.mod/sum, package.json, pnpm-lock không đổi |
| TC-N10-12 | P | 1 lỗi mức cao (TL-2h/2h2) ≤ 5, đã sửa |
| TC-N10-13 | P | Diff mã SP không có float64 mới/fetch trần/secret |
| TC-N10-14 | P | go vet (llm, grading, course) sạch; 403 chéo lớp vẫn đạt (TC-N03-09, TC-N04-07) |
