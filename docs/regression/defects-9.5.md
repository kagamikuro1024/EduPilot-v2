# Lỗi hồi quy sprint 9.5 — qc2, mã be0c0d96 (sprint/11-p10-eval), 2026-11-15

Chạy trên ep-r (dữ liệu dùng chung, nhiễm bởi ca khác) và stack riêng ep-r95-* (postgres `ep-r95-pg`, gateway/worker/redis/mail/docling riêng, DB `edupilot_t1_r95`). Chạy lại lần 2 của mọi ca F chạy được: ổn định, trừ khi ghi khác. Không phát hiện lỗi nào do sprint sau gây ra; mọi lỗi dưới đây có sẵn từ sprint gốc 9.5 (hoặc từ spec/TC) trừ khi ghi khác.

### DEF-R9.5-01 — `refuseDemo` bỏ lọt `[::1]` và `host.docker.internal`
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N01-17 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: ep-r, be0c0d96, người chạy script
- Bước tái hiện: `BASE=http://[::1]:3000 PHASE=seed SEED_DEFAULT_PASSWORD=x node deploy/vps-llm.mjs`; tương tự `http://host.docker.internal:3000`. (Port 3000 không có dịch vụ nên chỉ lỗi kết nối, không ghi gì.) Ngoài ra lỗi từ chối in stack trace Node nhiều dòng thay vì 1 dòng.
- Kỳ vọng (theo TC): cả 5 biến thể bị từ chối bằng cùng cơ chế · Thực tế: `127.0.0.1`, `localhost`, `/` cuối, `https://localhost` bị từ chối; `[::1]` và `host.docker.internal` đi tiếp tới `fetch`
- Bằng chứng: docs/regression/evidence-9.5/n01-refuse.log · Nghi ngờ mã: benchmarks/lib/meta.mjs:10-16 (`DEMO_PORTS`, so `hostname` chỉ với `localhost`/`127.0.0.1`)
- Chạy lại lần 2: ổn định

### DEF-R9.5-02 — `canary-scan.sh` không xoá nhà cung cấp canary
- Mức: Thấp
- Sprint gốc: 9.5 (script US-P1-04) · TC gốc: TC-N01-18 · Hồi quy do sprint sau?: KHÔNG RÕ (phản hồi tạo nhà cung cấp nay có `models[].id`)
- Môi trường: ep-r, be0c0d96, ADMIN
- Bước tái hiện: chạy `scripts/canary-scan.sh` với `A="Authorization: Bearer <ADMIN>"`; `GET /admin/llm/providers` thấy `CN0`, `CN1` còn lại
- Kỳ vọng: 7 dòng đếm = 0 và dọn dẹp · Thực tế: 7 dòng = 0 (đạt) nhưng `DELETE` nhận id của mô hình (regex `.*"id"` tham lam lấy id cuối) nên 2 nhà cung cấp rác còn trong cấu hình toàn cục (QC đã xoá tay)
- Bằng chứng: docs/regression/evidence-9.5/n01-canary.log · Nghi ngờ mã: scripts/canary-scan.sh:19 (`sed -n 's/.*"id":"…"/…'`)
- Chạy lại lần 2: ổn định

### DEF-R9.5-03 — load-a/c không có mẫu `docker stats` / ghi «không OOM, CPU < 95 %»
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N04-12 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: ep-r, be0c0d96
- Bước tái hiện: `grep -il 'oom\|docker stats\|cpu_pct' benchmarks/reports/load-*.json` → 0 tệp; không dòng nào trong báo cáo nói không có OOM / CPU ≥ 95 % suốt 60 s
- Kỳ vọng: có mục mẫu hoặc sự kiện · Thực tế: không có · Bằng chứng: n04-misc.log · Nghi ngờ mã: benchmarks/load/run-t1.mjs (không lấy docker stats)
- Chạy lại lần 2: ổn định

### DEF-R9.5-04 — Không có `00018_indexes_tuning.sql`, đề xuất P-95-8 (dev) chưa có quyết định PM
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N05-03 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: ep-r, be0c0d96
- Bước tái hiện: `ls backend-go/db/migrations` (00016, 00017, 00019…); `docs/sprints/9.5/proposals.md` dòng P-95-8 đầu: cột «Quyết định PM» trống
- Kỳ vọng (theo TC/AC2): thêm 00018 · Thực tế: không có; EXPLAIN T1 ≤ 13 ms nên dev đề xuất không tạo. QC rerun EXPLAIN: 9/10 truy vấn ≤ 11 ms, `rag_hybrid` 68,8 ms (dev 13,1 ms) do tải chung
- Bằng chứng: docs/regression/evidence-9.5/explain-r95.json, n05-explain-run.log · Nghi ngờ mã: –
- Chạy lại lần 2: ổn định (8 TC kèm theo ghi LT)

### DEF-R9.5-05 — 9.5 sửa test có sẵn
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N10-10 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: be0c0d96, git
- Bước tái hiện: `git log --no-merges --format='%h %s' ff00b5aa..2d730a4e -- '*_test.go'` → `bf9aa0bb` («sửa 2 test guard của main»), `811afebd` (đổi kỳ vọng 429 trong provider_test.go)
- Kỳ vọng: chỉ thêm test hồi quy; sửa test có sẵn phải báo PM · Thực tế: 429 có P-95-8 (PM duyệt TL-2j); 2 test guard (decision-gates ADMIN-only, KnowledgeSimilarPosts) không có quyết định PM nêu rõ
- Bằng chứng: n08-n10-scans.log · Nghi ngờ mã: backend-go/internal/llm/provider/provider_test.go, backend-go/internal/decision/*_test.go
- Chạy lại lần 2: ổn định

### DEF-R9.5-06 — PROGRESS.md thiếu dòng Nợ cho TL-2c/d/e/f
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N10-06 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -n 'TL-2[cdef]' docs/PROGRESS.md` → 0 dòng, dù techlead.md ghi «chuyển Nợ (PROGRESS.md)»
- Kỳ vọng: mỗi lỗi vừa/thấp có dòng Nợ · Thực tế: chỉ có Lighthouse /assignments/[id] (sprint 11)
- Bằng chứng: n08-n10-scans.log · Chạy lại lần 2: ổn định

### DEF-R9.5-07 — Báo cáo cuối không nêu hạn chế Mac ở từng mục
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N09-11 (TC-N04-15) · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -c 'MacBook Apple Silicon, Colima 6 CPU / 10 GB' docs/research/2026-11-14-nfr-real-llm.md` → 1; `grep -in 'cận dưới\|loopback' docs/research/…nfr-real-llm.md docs/sprints/9.5/report.md` → 0 (chỉ ở techlead.md:74, dev-lat-2.md)
- Kỳ vọng: ở mọi mục có số · Thực tế: một lần · Bằng chứng: n08-n10-scans.log · Chạy lại lần 2: ổn định

### DEF-R9.5-08 — Không ghi lần đo bị nhiễu
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N07-10 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -niE 'nhiễu|CPU nền' docs/research/2026-11-14-nfr-real-llm.md docs/sprints/9.5/handoff/*.md` → 0
- Kỳ vọng: ghi lần nhiễu (CPU nền > 30 %) và đo lại · Thực tế: không có · Chạy lại lần 2: ổn định

### DEF-R9.5-09 — `meta.config` của tệp real-llm thiếu chuỗi model và RPM
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N08-01 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `jq -c '.meta.config' benchmarks/reports/real-llm/*.json` → 12/13 tệp chỉ có `note` «khối meta bổ sung khi gộp 9.5»; model nằm trong `audit.models`
- Kỳ vọng: chuỗi model thực đo + RPM trong meta · Thực tế: không · Bằng chứng: n08-recompute.log · Nghi ngờ mã: benchmarks/real-llm/real_llm.py (không ghi meta), commit 2d730a4e
- Chạy lại lần 2: ổn định

### DEF-R9.5-10 — R3 (mask) thiếu tỷ lệ placeholder và số lần mô hình đổi placeholder
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N08-06 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: đọc benchmarks/reports/real-llm/mask.json (15 lượt) và report.md §3: «chưa so khớp tự động tên người khác»
- Kỳ vọng: ≥ 95 % giữ + khôi phục (tính lại), có số lần đổi/dịch/bỏ placeholder · Thực tế: không có hai số này · Chạy lại lần 2: ổn định

### DEF-R9.5-11 — R5 không đủ mẫu / AI Threads không có đầu ra
- Mức: Trung bình
- Sprint gốc: 9.5 · TC gốc: TC-N08-11 · Hồi quy do sprint sau?: KHÔNG (có sẵn; P-11-19 đã xếp hàng)
- Bước tái hiện: `jq` threads.json: 3/3 `ai_state=SKIPPED` (`LOW_SCORE`, embedding giả); chat.json chỉ có Q24 là tin khủng hoảng
- Kỳ vọng: ≥ 3 mẫu mỗi việc, khủng hoảng luôn đường an toàn · Thực tế: 1 mẫu khủng hoảng, 0 bài AI trả lời · Chạy lại lần 2: ổn định

### DEF-R9.5-12 — `real_llm.py` không có `--resume`, không dừng sớm khi hết hạn mức
- Mức: Cao
- Sprint gốc: 9.5 · TC gốc: TC-N08-16, TC-N08-17 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: máy chủ giả (docs/regression/evidence-9.5/scripts/fake-gateway.mjs)
- Bước tái hiện: `MODE=429 RA=120 node …/fake-gateway.mjs`; `BASE=http://127.0.0.1:18099 python3 benchmarks/real-llm/real_llm.py setup`
- Kỳ vọng: dừng lượt, ghi «chưa hoàn tất — hết hạn mức», lưu tiến độ, chạy tiếp bằng `--resume` · Thực tế: chờ 60 s × 5 (≈ 300 s, bỏ qua `Retry-After` 120) rồi «LỖI … 429» rc=2; không lưu tiến độ; `grep resume benchmarks/real-llm/*.py` → 0 (README mục `--resume` nói có)
- Bằng chứng: n08-harness-429.log, n08-fake429.log · Nghi ngờ mã: benchmarks/real-llm/real_llm.py:100-125, 172-177 (`backoff` tối đa 60 s, `tries=6`)
- Chạy lại lần 2: ổn định

### DEF-R9.5-13 — Báo cáo research thiếu mục NFR-02/03 và «6 phần»
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N09-01 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `grep -n '^##' docs/research/2026-11-14-nfr-real-llm.md` → (a)(b)(c)(d), Baseline, Chaos, EXPLAIN, LLM; không có seed T0/T1
- Chạy lại lần 2: ổn định

### DEF-R9.5-14 — Báo cáo research lệch tệp nguồn sau khi đo lại qua 9router
- Mức: Cao
- Sprint gốc: 9.5 · TC gốc: TC-N09-04, TC-N09-05, TC-N09-13 · Hồi quy do sprint sau?: KHÔNG (có sẵn; P-95-8 đo lại R1–R6 nhưng chỉ report.md được cập nhật)
- Bước tái hiện: so docs/research/2026-11-14-nfr-real-llm.md (R2 vòng 3 «MAE 1,19, 3/12», R4 «token đầu p50 7,1 s, p95 9,5 s», sự kiện đầu 15 ms, «1.000 bài chưa có kết quả») với benchmarks/reports/real-llm/grade-report.json (MAE 0,688; 9/12), chat.json (first_token p50 21, p95 4.217 ms; first_event p95 14 ms) và load-c-1000-9router-run{1-hong,2}.json (FAIL 322/1000 rồi PASS 1000/1000 trong 1,68 h)
- Kỳ vọng: 100 % số truy được về tệp nguồn; mọi FAIL có ở bảng tổng · Thực tế: các số cũ (Gemini trực tiếp) chỉ còn trong lan-1-khong-hop-le/; run1 FAIL vắng ở bảng tổng
- Bằng chứng: n08-recompute.log · Chạy lại lần 2: ổn định

### DEF-R9.5-15 — «Câu cần vá» không có tệp:dòng
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N09-07 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: mục «Danh sách câu cần vá» của báo cáo research chỉ nêu tên 3 tệp
- Chạy lại lần 2: ổn định

### DEF-R9.5-16 — `check-meta.mjs` không kiểm giá trị `demo_stack`
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N01-12 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: sao chép load-d.json, đặt `meta.demo_stack="unknown"`, `node benchmarks/check-meta.mjs <thư mục>` → «meta đủ»
- Kỳ vọng: mã ≠ 0 · Thực tế: mã 0 · Nghi ngờ mã: benchmarks/check-meta.mjs:15 (chỉ kiểm `=== undefined || ""`)
- Bằng chứng: n01-n02-quick.log · Chạy lại lần 2: ổn định

### DEF-R9.5-17 — Seed không chạy lại được sau khi dữ liệu bị sửa; check-*-seed đếm tuyệt đối
- Mức: Trung bình
- Sprint gốc: 8/9 (seed.mjs bước 15, US-S8-SEED) · TC gốc: TC-N02-02, TC-N02-03 · Hồi quy do sprint sau?: KHÔNG RÕ (môi trường ep-r nhiễu: 7 agent cùng ghi)
- Môi trường: ep-r, be0c0d96
- Bước tái hiện: `node scripts/seed.mjs` trên ep-r → «Seed lỗi: POST …/grade-entries/import → 422 IDEMPOTENCY_KEY_REUSED» (preview: 24/30 dòng giữa kỳ không đổi, 6 dòng bị ca khác sửa lúc 12:41 UTC; seed commit lại với cùng khoá Idempotency + upload_id mới). `check-assignments-seed.mjs bt03|counts`, `check-exam-seed.mjs bank`, `check-knowledge-seed.mjs` đỏ vì BT03 đã chấm/clone, 10 câu MCQ_MULTI QC thêm, 429 hạn mức
- Kỳ vọng: mã 0, N1 = N2 · Thực tế: mã 1 ở bước 15; lượt chạy tiếp (bước 16–20 trước đó chưa chạy) tạo thêm 43 attendance_checkins, 32 practice_attempts
- Bằng chứng: n02-check-seed.log, n02-seed-rerun1.log, n02-seed-rerun3-diff-course12.txt · Nghi ngờ mã: scripts/seed.mjs:1093-1096 (`importColumn`, khoá cố định theo sha tệp)
- Chạy lại lần 2: ổn định (cùng lỗi 422 ở 2 lượt chạy)

### DEF-R9.5-18 — Hôm nay của Admin trống ngay sau seed
- Mức: Thấp
- Sprint gốc: 9.5 (spec AC3) · TC gốc: TC-N02-06 · Hồi quy do sprint sau?: KHÔNG RÕ
- Bước tái hiện: đăng nhập `admin@`, `GET /me/today` → `{"count":0,"actions":[]}`; teacher 36 việc, ta 19, sv.kha/sv.gioi có `recommended`, sv.moi `JOIN_CODE`
- Kỳ vọng: mỗi vai ≥ 1 việc · Thực tế: Admin 0 (nguồn Admin chỉ phát sinh khi LLM lỗi / ngân sách / dead-letter / lớp không GV) · Nghi ngờ mã: backend-go/internal/today/providers.go:410-432
- Chạy lại lần 2: ổn định

### DEF-R9.5-19 — Seed không từ chối khi thiếu hoặc rỗng `SEED_DEFAULT_PASSWORD`
- Mức: Thấp
- Sprint gốc: 9.5 · TC gốc: TC-N02-09, TC-N02-11 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: `env -u SEED_DEFAULT_PASSWORD API_URL=http://127.0.0.1:1/api/v1 node scripts/seed.mjs`; hoặc `SEED_DEFAULT_PASSWORD="   "` → vẫn chạy bước 1 (đọc `.env.local`/`.env.example`)
- Kỳ vọng: mã ≠ 0 «Thiếu SEED_DEFAULT_PASSWORD» trước khi ghi · Thực tế: tiếp tục · Nghi ngờ mã: scripts/seed.mjs:44,56 (`setting()` rơi về tệp env; không `trim()`)
- Bằng chứng: n01-n02-quick.log · Chạy lại lần 2: ổn định

### DEF-R9.5-20 — Redis tắt: chat treo 39,5 s (TL-2d còn mở)
- Mức: Trung bình
- Sprint gốc: 9.5 (TL-2d) · TC gốc: TC-N06-04 · Hồi quy do sprint sau?: KHÔNG (có sẵn, đã ghi ≈ 19 s)
- Môi trường: stack riêng ep-r95 (gateway + redis + 2 worker), DB T1
- Bước tái hiện: `docker stop ep-r95-redis`; SV `POST /chat/sessions/{sid}/messages` → 200 suy giảm sau 39,5 s; `/readyz` 503 (đúng); tạo phiên 503 sau 1,9 s
- Kỳ vọng: ≤ 10 s, thông điệp «đang bận, thử lại» · Thực tế: 39,5 s · Bằng chứng: n06-redis-chat.log · Chạy lại lần 2: chưa lặp (1 lượt)


## Lượt chạy lại 2026-11-18 (HEAD 88ab83e4) — qc2
Đã sửa, xác nhận P: DEF-R9.5-01 (TC-N01-17: 5 biến thể + seed-t1 project edupilot bị từ chối), DEF-R9.5-09 (TC-N08-01), DEF-R9.5-10 (TC-N08-06: 35/35), DEF-R9.5-16 (TC-N01-12), DEF-R9.5-17 (TC-N02-02/03: do dữ liệu ep-r; trên smk91 sạch 7 check + seed lặp 3 lần đều đạt), DEF-R9.5-19 (TC-N02-09/11: rỗng/trắng bị từ chối; chưa đặt hẳn = «Thiếu SEED_DEFAULT_PASSWORD»; ghi nhận: có `.env.local` trong checkout thì seed dùng khoá đó theo b7d1e713), DEF-R9.5-05 (TC-N10-10, P-11-36).
Còn F (xác nhận lần này, bằng chứng `evidence-9.5/rerun/`):
- DEF-R9.5-03 (TC-N04-12) Thấp — vẫn không có mục `docker stats`/«không OOM» ở load-a/c.
- DEF-R9.5-06 (TC-N10-06) Thấp — `grep TL-2[cdef] docs/PROGRESS.md` = 0.
- DEF-R9.5-07 (TC-N09-11) Thấp — hạn chế Mac/loopback không có ở từng mục báo cáo.
- DEF-R9.5-08 (TC-N07-10) Thấp — không ghi lần đo bị nhiễu.
- DEF-R9.5-11 còn một phần (TC-N08-11) Thấp — «escalation theo độ tin cậy» chỉ 2 mẫu, không có đánh giá định tính từng mẫu.
- DEF-R9.5-13 (TC-N09-01) Thấp — thiếu mục NFR-02/03, mục (b)(c)(d)/Baseline/Chaos/EXPLAIN không đủ «6 phần».
- DEF-R9.5-14 còn 2 số (TC-N09-04) Thấp — «nền 410 ms» (nguồn 409,06) và «TTFT RAG 633,9 ms» (nguồn 633,993 → 634,0); 126/128 số truy được (`rerun/trace-numbers.py`).
- DEF-R9.5-15 (TC-N09-07) Thấp — «câu cần vá» không có tệp:dòng.
Chưa rerun (N, cần HEAD mới): DEF-R9.5-02, -04, -12 (N08-16/17 --resume), -18 (N02-06, theo P-11-61 Admin rỗng hợp lệ), -20 (N06-04, cần khoá toàn cục dừng Redis ≤ 1 phút).
Hạ tầng: `docker exec <postgres> psql` thoát mã 2 (lỗi/ngắt kết nối) làm postmaster tưởng tiến trình con chết và reinit DB (đã thấy 4 lần ở smk91; có thể là nguyên nhân sự cố ep-r 9.5). Dùng `sh -c 'psql …; exit 0'`.
