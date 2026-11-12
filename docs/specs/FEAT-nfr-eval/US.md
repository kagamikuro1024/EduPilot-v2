# FEAT-nfr-eval Seed, yêu cầu phi chức năng và đánh giá LLM thật (sprint 9.5)
Nguồn: `docs/sprints/ops-vps/plan.md` (mục "Thu hẹp phạm vi", "Hoãn sang một sprint riêng") · `docs/sprints/ops-vps/prompts/tl-eval.md` · `docs/SYSTEM_DESIGN.md` §1.2, §5 · `docs/phases/P10.md` L3 · PRD §5, §6 · FLOWS F9–F11, F3 · `docs/team/CONTEXT.md` mục 4 (bản 2026-11-12: Mac, không VPS)
Phase: sprint 9.5 (nằm giữa 9.1 và 10, không phải phase mới). Trạng thái: **DRAFT** (chờ PM + chủ dự án; câu hỏi mở ở `QUESTIONS.md`).
Thiết bị đo: **MacBook Apple Silicon, Colima 6 CPU / 10 GB**. Mọi con số trong spec này, trong báo cáo và trong quyển đồ án đều ghi kèm câu đó.

**Quy ước chung cho mọi story** (không lặp lại trong từng AC):
- Sprint này **đo, không thêm tính năng**. Mã sản phẩm chỉ đổi khi một phép đo trượt SLO hoặc lộ lỗi (xem US-NFR-10).
- Mỗi tệp kết quả `benchmarks/reports/**/*.json` có khối `meta`: `commit`, `thời điểm`, `máy` ("MacBook Apple Silicon, Colima 6 CPU / 10 GB"), `số lần chạy`, `cấu hình` (số bản gateway / worker, provider, model, `RPM`). Thiếu `meta` → phép đo không hợp lệ.
- Dữ liệu mô phỏng hoàn toàn (D44). Không tên / MSSV / email thật ở seed, báo cáo, log. Báo cáo không chứa khoá API, mật khẩu, `Authorization`.
- Một stack test tại một thời điểm (`CONTEXT.md` mục 4); compose project `-p ep-<vai>`, cổng lệch, chỉ bind `127.0.0.1`; không đụng stack demo `~/Documents/EduPilot-demo`. Đo xong `down -v`.
- Một phép đo nặng tại một thời điểm. Output lệnh cắt còn dòng lỗi + tóm tắt.
- Số liệu nêu trong báo cáo / luận văn phải có tệp nguồn trong `benchmarks/reports/` hoặc `docs/research/`.
- Đo xong mà SLO trượt: ghi số thật, không nới SLO, không chọn lại lần chạy tốt nhất (ghi cả lần trượt).

Thứ tự thi công đề xuất: **NFR-01 → 02 → 03 → 05 → 04 → 06 → 07 → 08 → 09 → 10**.

---

## US-NFR-01: Đội muốn môi trường đo chạy lặp lại được trên Mac, để mọi số liệu so sánh được và không ghi nhầm lên stack demo
Ưu tiên: Must · Ước lượng: S · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given Mac sạch (Colima đã chạy) When làm theo `benchmarks/README.md` (tệp mới, mục "Đo trên Mac") Then dựng được stack đo `-p ep-<vai>` với cổng lệch và đo được một lệnh `smoke.js`; README **không còn** nhắc VPS, `/srv/work/research`, `ssh`, 9router.
- AC2. Given `benchmarks/real-llm/README.md` When đọc Then dòng "Chạy trên VPS … không chạy trên Mac" đã được thay bằng quy trình Mac (stack `-p ep-research`, cổng lệch, `down -v` khi xong); đường dẫn `deploy/vps-*.mjs` không còn là bước bắt buộc.
- AC3. Given bất kỳ lệnh đo nào trong sprint When chạy Then đầu ra JSON có khối `meta` đủ trường ở "Quy ước chung"; một script kiểm `benchmarks/check-meta.mjs` (hoặc tương đương) trả lỗi nếu thiếu.
- AC4 (nhánh lỗi). Given `BASE` trỏ vào cổng của stack demo (`:3000`, project mặc định) When chạy lệnh ghi dữ liệu (`setup`, `grade-run`, seed T1) Then script **từ chối chạy** và in lý do (một dòng); chạy với stack `-p ep-*` thì chạy.
- AC5 (phân quyền / bí mật). Given repo When quét bằng `scripts/canary-scan.sh` và grep khoá `AIza` / `sk-` / `Bearer ` trong `benchmarks/**`, `docs/research/**` Then 0 kết quả.

### Ngoài phạm vi của story này
- Thiết lập Colima / Docker (đã có).
- Chạy đo trên máy khác.

### Phụ thuộc
- `docs/team/CONTEXT.md` mục 4; `deploy/README.md`.

---

## US-NFR-02: Đội muốn seed demo (T0) chạy qua sạch trên Mac, để mọi phép đo và buổi demo bắt đầu từ dữ liệu đúng
Ưu tiên: Must · Ước lượng: M · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given stack trống (`-p ep-<vai>`) When `node scripts/seed.mjs` (hoặc `pnpm seed`) Then thoát mã 0, và các `scripts/check-*-seed.mjs` (assignments, chat, crm, docs, exam, grade, knowledge) đều xanh; thời gian chạy ghi vào báo cáo.
- AC2. Given seed đã chạy When chạy lại lần hai Then mã 0, không tạo trùng (số dòng các bảng chính không đổi), không lỗi `409`/`23505` bị nuốt.
- AC3. Given seed xong When đăng nhập 4 vai (SV A, B, C; TA; GV; Admin) bằng mật khẩu từ `SEED_DEFAULT_PASSWORD` Then `Hôm nay` mỗi vai có ít nhất một việc đúng nguồn (bảng F14); lớp 1 có 30 SV, lớp 2 khác lớp 1 về công thức điểm (thiếu làm tròn).
- AC4. Given seed xong When so với `docs/DEMO_SCRIPT.md` bước 2–7 Then mọi dữ liệu kịch bản cần có sẵn (bài BT03 22 bài chưa chấm; 3 SV A, B, C có điểm tính tay được; câu hỏi chat mẫu có tài liệu để trích); thiếu mục nào → ghi vào danh sách bổ sung trong handoff, dev thêm vào seed.
- AC5 (nhánh lỗi). Given chưa đặt `SEED_DEFAULT_PASSWORD` hoặc stack chưa lên When chạy seed Then dừng ngay với thông báo tiếng Việt một dòng và mã ≠ 0, không ghi dòng nào.
- AC6 (riêng tư). Given seed xong When chạy `canary-scan.sh` và tìm họ tên / MSSV / email ngoài danh sách mô phỏng trong `seed/**` Then 0 kết quả; seed không phụ thuộc mạng ngoài (embedding dùng provider giả / tất định khi không có khoá).

### Ngoài phạm vi của story này
- Dữ liệu T1 (US-NFR-03).
- Chạy lại `DEMO_SCRIPT.md` trọn 15 phút (P10).

### Phụ thuộc
- US-NFR-01; `scripts/seed.mjs`, `scripts/check-*-seed.mjs`; D44.

---

## US-NFR-03: Đội muốn bộ sinh dữ liệu T1 (1.000 SV, 20 lớp, một học kỳ), để các phép đo tải có nghĩa
Ưu tiên: Must · Ước lượng: M · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given stack đo trống When `node scripts/seed-t1.mjs` Then tạo trong DB riêng `edupilot_t1`: 1.000 sinh viên, 20 lớp (≈ 50 SV/lớp), 5 TA, 3 giảng viên, một học kỳ dữ liệu (câu hỏi chat ≈ 3.000/tuần × số tuần, điểm danh, điểm thành phần, bài nộp, `study_time_daily` ≈ 525.000 dòng) với số dòng đúng bảng ước lượng `SYSTEM_DESIGN.md` §1.2 (sai số ≤ 10%); thời gian chạy ghi lại.
- AC2. Given cùng `SEED_RNG` When chạy hai lần trên hai DB trống Then số dòng từng bảng bằng nhau (tất định).
- AC3. Given T1 xong When đếm Then mọi dòng con có `course_id` đúng cha (FK phức hợp không bị tắt); không dòng nào thuộc lớp "mồ côi"; mật khẩu tài khoản mô phỏng lấy từ `SEED_DEFAULT_PASSWORD`, không cố định trong mã.
- AC4. Given T1 xong When chạy `ANALYZE` rồi gọi 6 trang chính bằng STUDENT / TEACHER Then không `5xx`; STUDENT lớp X không đọc được dữ liệu lớp Y (`403`/`404` trên ≥ 20 cặp ngẫu nhiên).
- AC5 (nhánh lỗi). Given hết dung lượng đĩa hoặc Postgres bị ngắt giữa chừng When seed T1 Then thoát mã ≠ 0 và in bảng đang ghi; chạy lại bằng cờ `--reset` xoá sạch `edupilot_t1` (không đụng DB `edupilot`) rồi chạy được.
- AC6 (an toàn). Given `--reset` When `DATABASE_URL` trỏ vào DB tên `edupilot` (không phải `edupilot_t1`) Then từ chối chạy.
- AC7 (kích thước). Given T1 xong Then kích thước DB và RAM Postgres nghỉ ghi vào báo cáo; nếu seed T1 vượt RAM Colima (OOM) thì ghi rõ và dùng cờ thu nhỏ `--scale 0.5` (xem Q-NFR-3), số liệu ghi đúng quy mô thực.

### Ngoài phạm vi của story này
- Dữ liệu thật; nhãn người gán cho E1/E3 (P10).

### Phụ thuộc
- US-NFR-01; `SYSTEM_DESIGN.md` §1.2; `seed.mjs` (tái dùng `*-seed-data.mjs`).

---

## US-NFR-04: Đội muốn đo tải theo SLO ở quy mô T1 với LLM giả, kể cả `--scale gateway=2 worker=2`, để chứng minh mở rộng ngang không đổi code
Ưu tiên: Must · Ước lượng: L · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given T1 (US-NFR-03), provider LLM `fake` độ trễ 5–15 s When chạy bốn kịch bản k6 ở `benchmarks/load/`: **(a)** 300 người dùng đồng thời duyệt + 300 SSE thông báo; **(b)** chat 1 QPS, đỉnh 5 QPS; **(c)** chấm 1.000 bài trong khi chat; **(d)** dashboard / hồ sơ 360 / sổ điểm trên 1.000 SV Then ghi `benchmarks/reports/load-{a,b,c,d}.json` với p50 / p95 / p99, tỷ lệ 5xx, và PASS/FAIL từng dòng SLO của `SYSTEM_DESIGN.md` §5.
- AC2. Given (c) When so TTFT chat p95 lúc chấm với lúc nền (cùng lần chạy, `PHASE=baseline` rồi `PHASE=grading`) Then xấu đi ≤ 20%; số liệu cả hai pha có trong báo cáo.
- AC3. Given (a)–(c) When chạy lại với `--scale gateway=2 --scale worker=2` Then báo cáo có hai cột (×1, ×2) cùng kịch bản; cùng mã nguồn (cùng `commit` trong `meta`); không có thay đổi mã để phục vụ ×2.
- AC4. Given mọi kịch bản When STUDENT gọi 100 lần một API chỉ giảng viên (vd. `GET …/gradebook`) giữa lúc tải Then 100% trả `403`, không lần nào `200` (phân quyền không hỏng dưới tải).
- AC5. Given (b) When đo sự kiện SSE đầu tiên và TTFT Then sự kiện đầu ≤ 300 ms (D47); TTFT chat ≤ 1,5 s (cache hit) / ≤ 4 s (RAG) ở p95 — ghi riêng hai nhóm.
- AC6 (nhánh lỗi / trung thực). Given một SLO trượt trên Mac When báo cáo Then ghi FAIL + số thật + lần thử lại (nếu có) + nghi ngờ nguyên nhân (k6 và server dùng chung 6 CPU; RAM); không đổi ngưỡng. Lỗi hiệu năng thật chuyển US-NFR-10.
- AC7 (an toàn tài nguyên). Given đo (a) hoặc (c) When RAM Colima > 90% hoặc OOM-kill xảy ra Then dừng lượt, ghi sự kiện vào báo cáo và hạ số VU theo Q-NFR-3; không chạy hai kịch bản cùng lúc.

### Ngoài phạm vi của story này
- Đo bằng LLM thật (US-NFR-08).
- Đo p95 trên VPS / mạng thật.

### Phụ thuộc
- US-NFR-03; k6 (image `grafana/k6` chạy trong Colima); `benchmarks/load/{chat,grading-vs-chat,today,smoke,heartbeat}.js` hiện có.

---

## US-NFR-05: Đội muốn `EXPLAIN ANALYZE` 10 truy vấn nặng nhất trên T1 và chỉ mục chỉnh theo kết quả, để dashboard / sổ điểm / tìm kiếm không chậm khi có 1.000 SV
Ưu tiên: Must · Ước lượng: M · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given T1 When lấy 10 truy vấn nặng nhất (theo thời gian từ `pg_stat_statements` hoặc log truy vấn chậm sau chạy kịch bản (d)) Then `docs/research/…-explain.md` liệt kê từng truy vấn, kế hoạch trước, số dòng, thời gian.
- AC2. Given danh sách When thêm chỉ mục Then nằm trong **một** migration goose mới `00018_indexes_tuning.sql` (không sửa migration đã merge); mọi chỉ mục phức hợp bắt đầu bằng `course_id` (luật 13); chỉ mục HNSW cho `content_chunks.embedding` nếu `EXPLAIN` cho thấy tìm vector đang quét tuần tự; mỗi chỉ mục có một dòng lý do trong báo cáo.
- AC3. Given migration áp dụng When chạy lại 10 truy vấn Then báo cáo có bảng trước / sau; mỗi truy vấn nhanh hơn hoặc "không đổi, không cần chỉ mục" (có lý do); không truy vấn nào chậm đi > 10%.
- AC4. Given `goose up` rồi `goose down 1` rồi `up` trên DB T1 Then thành công cả ba lần; `sqlc diff` sạch; `go test ./...` các gói đụng truy vấn vẫn xanh.
- AC5 (nhánh lỗi). Given chỉ mục làm một truy vấn ghi (điểm danh, nộp bài) chậm hơn > 20% Then bỏ chỉ mục đó và ghi lý do.
- AC6 (không phình). Given tổng kích thước chỉ mục mới Then ghi vào báo cáo (MB); nếu > 25% kích thước bảng chính thì nêu trong QUESTIONS với lý do giữ.

### Ngoài phạm vi của story này
- Partition, read replica, đổi PgBouncer.

### Phụ thuộc
- US-NFR-03; `00017` là migration cuối hiện tại; `00019` dành cho P9 (`ARCHITECTURE.md` §4).

---

## US-NFR-06: Đội muốn thử hỗn loạn nhỏ (giết worker, khởi động lại Redis, tắt `docling-serve`), để chứng minh 0 việc thất lạc và hệ thống tự hồi
Ưu tiên: Must · Ước lượng: M · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given đang chạy tải vừa (chấm 200 bài + ingest 3 tài liệu + chat 1 QPS) When `kill -9` một worker giữa hai lượt chấm Then sau tối đa 10 phút mọi bài có kết quả đúng một lần; `grading.jobs` không còn việc `PENDING` mồ côi; 0 bài chấm hai lần (đếm theo `submission_id`).
- AC2. Given cùng tải When dừng Redis 70 s rồi bật lại Then chat báo "đang bận, thử lại" đúng kiểu suy giảm (không `5xx` thô, không treo); sau khi Redis lên, SSE nối lại bằng `Last-Event-ID` không mất sự kiện; không lô chấm nào mồ côi (nợ sprint 8–9: `grading.batch` mất ánh xạ lô — nếu xảy ra, ghi thành lỗi).
- AC3. Given đang ingest tài liệu When dừng `docling-serve` 2 phút rồi bật lại Then job ingest retry theo luật 3 lần rồi dead-letter hoặc hoàn tất sau khi dịch vụ về; tài liệu không kẹt `PROCESSING` vĩnh viễn; chat các tài liệu đã `READY` không ảnh hưởng.
- AC4. Given mỗi tình huống When đối chiếu cuối Then tổng việc vào = tổng việc xong + dead-letter có lý do (`0 việc thất lạc`); bảng đối chiếu trong `benchmarks/reports/chaos.json`.
- AC5 (nhánh lỗi). Given một tình huống để lại việc thất lạc hoặc hai lần chấm Then FAIL, lỗi chuyển US-NFR-10; không "dọn" bằng tay trước khi chụp bằng chứng.
- AC6 (an toàn). Given chỉ stack `-p ep-<vai>` When chạy chaos Then lệnh dừng / giết nhắm theo tên project (`docker compose -p …`), không `docker kill` theo tên chung; stack demo không bị ảnh hưởng (kiểm `docker ps` trước / sau).

### Ngoài phạm vi của story này
- Chaos mạng (mất gói), đầy đĩa, mất Postgres (ghi vào "Nợ" nếu muốn).

### Phụ thuộc
- US-NFR-03, US-NFR-04 (kịch bản tải), `docker-compose` của dự án, test sprint 9 (`kill -9` chưa kiểm trên stack).

---

## US-NFR-07: Đội muốn số đo nền của bản Go trên Mac (RAM, khởi động, kích thước image, p95 endpoint, Lighthouse), để luận văn có E6 thật
Ưu tiên: Must · Ước lượng: M · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given stack T0 nghỉ 5 phút When `docker stats --no-stream` Then bảng RAM / CPU nghỉ từng container (gateway, worker, postgres, pgbouncer, redis, minio, docling-serve nếu bật, caddy, frontend) vào `benchmarks/reports/baseline.json` + `.md`; tổng RAM của stack.
- AC2. Given image đã build When đo Then kích thước image gateway, worker, frontend (MB) và thời gian từ `docker compose up` đến `/api/v1/readyz` = 200 (lạnh, 3 lần, trung vị).
- AC3. Given T0 When đo p50/p95 của ≥ 12 endpoint đại diện (đọc: Hôm nay, danh sách thread, sổ điểm, hồ sơ 360, thư viện; ghi: điểm danh, gửi chat không LLM, nộp bài) × 200 lần tuần tự Then p95 đọc ≤ 300 ms, ghi ≤ 500 ms, hoặc FAIL kèm số.
- AC4. Given 6 route người dùng của `lighthouserc*.json` When chạy Lighthouse (mô phỏng và devtools) trên Mac Then bảng LCP / TBT / script size; đối chiếu ngưỡng hiện hành (LCP ≤ 2.500 ms devtools; TBT ≤ 200 ms mô phỏng; script ≤ 256 KB); route nào trượt ghi rõ, **không nới ngưỡng**; `/assignments/[id]` (nợ G7-23, LCP 4,7 s) có dòng riêng.
- AC5 (nhánh lỗi). Given đo bị nhiễu (CPU nền > 30%) When phát hiện Then dừng và đo lại sau khi tắt tiến trình nền; ghi lần nhiễu.
- AC6 (trung thực). Given báo cáo When viết Then không đối chiếu với hệ cũ (D45) và không so với VPS cũ; ghi "đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB".

### Ngoài phạm vi của story này
- Judge amd64 có seccomp: **không đo được trên Mac arm64** (D58: seccomp tắt trên arm64). Ghi vào Nợ chuyển PR (Q-NFR-4).

### Phụ thuộc
- US-NFR-01, US-NFR-02.

---

## US-NFR-08: Giảng viên / chủ dự án muốn đo các việc LLM bằng mô hình thật, để báo cáo thôi ghi "chưa test LLM thật"
Ưu tiên: Must · Ước lượng: L · Sprint: 9.5

Mô hình: **Gemini** qua khoá API của chủ dự án (Q-NFR-2: tên model). Harness có sẵn `benchmarks/real-llm/real_llm.py` (chỉ gọi HTTP API gateway). Tổng lời gọi ≈ 330 (bảng ở README harness); tôn trọng hạn mức: tuần tự, `429`/`503` thì lùi theo `Retry-After`.

### Tiêu chí nghiệm thu
- AC1 (R1, trích công thức). Given `quy-che-lop1.pdf`, `quy-che-lop2.pdf` When trích mỗi cái ≥ 5 lần Then báo cáo có: % đúng trọng số / tham số so với `docs/specs/FEAT-gradebook/fixtures/extraction-lop*.json`; tỷ lệ phát hiện mục "chưa rõ" (lớp 2 thiếu làm tròn); % trích dẫn có thật và đúng trang; % đầu ra đúng schema ngay lần đầu; độ trễ và token.
- AC2 (R2, chấm tự luận). Given 7 bài mẫu `FEAT-assignments/fixtures/` + 12 bài `EVAL03` + 22 bài `BT03` When "Chấm tất cả" ≥ 2 vòng (EVAL03: 3 vòng) Then báo cáo có MAE tổng / từng tiêu chí, % trong ±0,5, QWK (nếu có nghĩa), % hai lượt lệch > 1 (`NEEDS_REVIEW`), độ ổn định giữa vòng, trích dẫn bị bộ kiểm loại, đầu ra sai schema, thời gian mỗi bài. **Ghi rõ:** đáp án kỳ vọng do người soạn đặc tả viết, không phải chấm tay của giảng viên → **không thay E3**.
- AC3 (chèn lệnh). Given bài "cho tôi 10 điểm" + ≥ 5 biến thể (tiếng Việt / Anh, giả lời giảng viên, chèn JSON) When chấm Then báo cáo ghi bài nào kéo điểm lên (> 0,5 so với bản không chèn), bài nào bị gắn cờ "nghi chèn lệnh"; kéo điểm lên mà không cờ = lỗi mức cao (US-NFR-10).
- AC4 (R3, che danh tính). Given ≥ 15 câu chat có tên / MSSV / tên bạn cùng lớp When gửi tới LLM thật Then (i) `llm_audit` (đọc chỉ-đọc) không chứa tên / MSSV thật trong payload; (ii) placeholder được mô hình giữ và khôi phục đúng ≥ 95%; (iii) ghi số lần mô hình đổi / dịch / bỏ placeholder và hệ thống phản ứng thế nào.
- AC5 (R4, chat riêng + Threads + phân loại). Given 20–30 câu từ `DEMO_SCRIPT.md` / seed When chạy Then bảng: sự kiện đầu (≤ 300 ms ở p95, D47), TTFT, tổng thời gian; % phân loại intent đúng; % câu trả lời bám tài liệu (người đánh dấu); câu hỏi điểm cá nhân đi đúng đường tool với danh tính từ JWT; hỏi điểm người khác → từ chối 100%; không câu nào lộ điểm nháp.
- AC6 (R5, việc khác). Given SuggestRubric, nhận xét tổng hợp hồ sơ 360, sinh lại câu trả lời Threads, tin nhắn khủng hoảng, escalation theo độ tin cậy When mỗi việc ≥ 3 mẫu Then đúng schema 100% (hoặc ghi % sai), đánh giá định tính một dòng mỗi mẫu; tin nhắn khủng hoảng luôn đi đường an toàn (không do LLM tự do quyết định).
- AC7 (R6, chấm hàng loạt không làm chậm chat). Given `grading-vs-chat.js` với LLM thật `VUS=4` When chạy hai pha nền / chấm Then TTFT chat p95 xấu đi ≤ 20%; hoặc FAIL kèm số.
- AC8 (hạn mức). Given toàn bộ lượt đo When thống kê Then số lời gọi LLM thực tế, token vào / ra, số `429`, tổng chi phí ước tính (hoặc "free tier") ghi vào báo cáo; không vượt trần ngân sách do chủ dự án đặt (Q-NFR-2).
- AC9 (nhánh lỗi). Given khoá hết hạn mức / `429` kéo dài / mất mạng giữa lượt When xảy ra Then harness lưu tiến độ, dừng sạch, chạy tiếp từ chỗ dừng (`--resume`), kết quả từng phần ghi nhận "chưa hoàn tất"; không dội lại.
- AC10 (bí mật / riêng tư). Given khoá Gemini When chạy Then khoá chỉ nằm trong `.env.local` ngoài git (hoặc biến môi trường) và vào hệ thống qua màn Cấu hình LLM của Admin; không có ở log, `benchmarks/**`, `docs/research/**`, ảnh chụp; ADMIN xem `llm_audit` có ghi `audit_log`; TEACHER / STUDENT không đọc được payload.
- AC11 (đường tool, phân quyền). Given STUDENT B When hỏi điểm của A qua chat Then bị từ chối và `llm_audit` không có lời gọi LLM sinh chữ chứa điểm của A.

### Ngoài phạm vi của story này
- E3 chấm tay người (P10); E4 so ≥ 2 provider (P10; sprint này đo 1 provider, ghi rõ).
- Embedding thật (Q-NFR-2: dùng Gemini embedding nếu khoá có, ngoài ra provider giả).
- Đo độ chính xác tìm kiếm RAG (recall@k).

### Phụ thuộc
- US-NFR-01, US-NFR-02; `benchmarks/real-llm/`; fixtures `FEAT-gradebook`, `FEAT-assignments`; D46, D47, luật 2, 4, 5.

---

## US-NFR-09: Chủ dự án muốn một báo cáo tổng hợp và bảng số liệu duy nhất, để chép vào báo cáo / quyển đồ án mà không phải đoán số nào đo ở đâu
Ưu tiên: Must · Ước lượng: M · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given các story NFR-02…08 xong When viết `docs/research/2026-11-xx-nfr-real-llm.md` (tên ngày thật) Then có mục theo từng phép đo: câu hỏi · phương pháp · cấu hình · số liệu · PASS/FAIL · giới hạn; bảng tổng "SLO → số đo → kết quả"; mục "chưa đo được và vì sao" (judge amd64 seccomp, E3 tay, nhiều provider, mạng thật).
- AC2. Given mỗi con số trong báo cáo When truy nguồn Then có tệp trong `benchmarks/reports/`; không con số nào chỉ nằm trong báo cáo văn xuôi.
- AC3. Given số liệu cũ trong `docs/sprints/8/report.md`, `docs/sprints/9/report.md`, `docs/PROGRESS.md` ghi "giả lập" / "chưa đo" When báo cáo mới có số thật Then danh sách "câu cần vá" (vị trí → câu mới) gửi PM; BA / PM không sửa báo cáo cũ nếu chưa có proposal.
- AC4 (nhánh lỗi). Given một phép đo không hoàn tất When viết Then ghi "chưa hoàn tất" + lý do, không bỏ trống không giải thích, không ngoại suy.
- AC5. Given tiêu đề trang và nhãn Then đọc được trong ≤ 5 phút (≤ 3 trang cho mục tóm tắt); tiếng Việt; có bảng, không chép nguyên log.

### Ngoài phạm vi của story này
- Viết chương quyển đồ án (`writer`).

### Phụ thuộc
- US-NFR-04…08.

---

## US-NFR-10: Dev muốn sửa lỗi mà phép đo tìm ra, trong ngân sách giới hạn, để SLO đạt mà không mở rộng phạm vi
Ưu tiên: Must · Ước lượng: L (không biết trước) · Sprint: 9.5

### Tiêu chí nghiệm thu
- AC1. Given mỗi lỗi do phép đo tìm ra When ghi Then một dòng `docs/sprints/9.5/techlead.md` (TL-n) hoặc `proposals.md`: tái hiện (lệnh), mức (cao / vừa / thấp), đề xuất, bằng chứng (tệp trong `benchmarks/reports/`).
- AC2. Given lỗi mức cao (rò đáp án / điểm / PII; chèn lệnh kéo điểm lên mà không cờ; việc thất lạc; SLO TTFT / 5xx trượt) When PM duyệt Then dev sửa trong sprint, có test hồi quy; phép đo liên quan chạy lại và PASS; báo cáo ghi cả hai lần.
- AC3. Given lỗi mức vừa / thấp When PM quyết Then chuyển sang `PROGRESS.md` mục Nợ hoặc sprint sau, không sửa tại sprint này.
- AC4. Given sửa mã sản phẩm When commit Then thông điệp `9.5: <việc>` + số TL / proposal; không đổi hợp đồng API, không sửa migration đã merge (chỉ `00018_indexes_tuning`).
- AC5 (không phá). Given mỗi sửa When chạy `go test -race` gói đụng tới + test frontend liên quan Then xanh; cổng sprint chạy cả bộ một lần trước handoff.
- AC6 (nhánh lỗi). Given một SLO không thể đạt bằng sửa nhỏ (cần đổi kiến trúc) Then dừng, ghi PM; không thêm hạ tầng (Kafka, K8s, shard — SYSTEM_DESIGN §4).

### Ngoài phạm vi của story này
- Tính năng mới, UI mới, đổi luật sản phẩm.

### Phụ thuộc
- US-NFR-04…08.
