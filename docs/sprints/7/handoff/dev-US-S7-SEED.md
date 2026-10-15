# DEV handoff — US-S7-SEED (dữ liệu seed sprint 7, hai cổng)
Nhánh: `sprint/7-p4-p5`. Commit: `US-S7-SEED: …`. Số ghim: không (không thao tác OpenAPI mới). `#35` (ACCEPTED): lệnh seed chỉ `APP_ENV ∈ {dev, test}`.

**Trạng thái:** đã chạy thật trên stack sạch (2026-10-16, project `edupilot`, profile `ingest`): xem mục "Chạy thật". `gate-p4.sh` PASS, `gate-p5.sh` PASS (có `check-crm-seed.mjs`).

## Đã làm
- **Lệnh quản trị** (`cmd/gateway/seedcmd.go`, `internal/seedtool`): `gateway admin seed-study-time --course C --weeks N --profile FILE|-`, `seed-age --course C (--tickets --ticket ID [--hours H] | --activity [--weeks N])`, `crm-risk-run --course C`. Danh sách cho phép `APP_ENV ∈ {dev, test}` kiểm TRƯỚC mọi việc (production, `staging`, `local`, rỗng, giá trị lạ → mã thoát 1, không chạm DB). In số dòng đã đổi; chạy lại in `0 dòng`. `--profile -` đọc stdin vì gateway chạy trong container (distroless) không thấy thư mục `seed/`.
  - `seed-study-time`: `ON CONFLICT … GREATEST` (chỉ ghi khi tăng); tuần n (1..N, N = tuần hiện tại) chia đều cho các ngày đã trôi qua, 1–3 khu vực luân phiên; tổng giây mỗi tuần đúng `phút × 60`.
  - `seed-age --tickets`: lùi `created_at`, **`waiting_since`**, `claimed_at`, `answered_at`, `closed_at` của ĐÚNG một ticket (mặc định 192 giờ); không đổi trạng thái / nội dung. `--activity`: đưa tin chat USER, thread, bình luận của tuần hiện tại về các tuần trước theo băm `id` (chạy lại không dịch thêm).
- **Kế hoạch số liệu** `scripts/crm-plan.mjs` (hàm thuần, Fisher–Yates theo `SEED_RNG`) → `seed/crm_plan.json`, `seed/expected_crm.csv`; `seed/study_profile.json`. Cố định: A vắng 0; B vắng buổi 3, 6 + muộn 1 (2 SPEAK = +0,50); C vắng 2, 5, 7, 8, 9 + muộn 1; sv12 và sv19 vắng chỉ buổi 9; 12 người khác vắng đúng một buổi trong 1–7 ⇒ ABSENT cả lớp 21 (18–26), phát biểu 65.
- **`seed.mjs`**: bước 3 đặt `allow_study_time_opt_out=false` cho lớp 761988 (TEACHER, sau khi gán giảng viên, trước khi có sinh viên; `409 COURSE_HAS_STUDENTS` = đã áp dụng). Bước 14 điểm danh buổi 1–9 (`PUT …/attendance complete:true`, 270 dòng) + `POST …/participation` với `client_event_id` cố định (sha1 → UUID). Bước 15 gọi ba lệnh admin (`SEED_ADMIN_CMD` bỏ `create` làm tiền tố; mặc định `docker compose exec gateway /gateway admin`), bắt buộc kết quả lớp 1 `AT_RISK 1, WATCH 2` và lớp 2 `AT_RISK 0`. Bước 16: 5 ticket (sv14 OPEN mới, sv15 OPEN lùi 192 giờ, sv16 CLAIMED, sv17 ANSWERED + tri thức, sv18 CLOSED) bằng chat có sẵn của bước 12 → `escalate` → `claim` → `answer` → `confirm`; mỗi bước chỉ làm khi trạng thái còn thấp hơn (idempotent; ca quá hạn chỉ lùi khi ticket mới tạo < 2 giờ). Bước 17: 2 báo cáo OPEN (sv20, sv21), thread tự ẩn (sv22–24), 1 thread ghim, 1 bình luận (sv25) bị `Ẩn bài` có lý do; hàng đợi ≥ 2. `node --check` sạch.
- **`scripts/check-crm-seed.mjs`** (API thật, từng tài khoản): 270 dòng, bảng A/B/C với `expected_crm.csv`, ABSENT 18–26, đúng một `filter=attention` (C), mức rủi ro `AT_RISK 1 / WATCH 2`, `STUDENT_ATTENTION` ở Hôm nay, lớp 2 không at-risk / ticket, đủ bốn trạng thái ticket (≥ 5), hàng đợi ≥ 2, A/B/C chỉ đọc của mình + 403 ở danh sách / hồ sơ, D không có lớp, ADMIN 403; chạy lại `seed.mjs` so số liệu (idempotent).
- **`scripts/gate-p4.sh`** (P4.md + `-race`, `strict_test` cho `TestEscalationToMail`, `TestNotifyAcrossTwoGateways`, `TestOutboxSurvivesWorkerKill`; Playwright escalation + threads-moderation + threads-crisis) — **chạy: `GATE P4: PASS`**. **`scripts/gate-p5.sh`** (P5.md + `-race`; `TestToolMyAttendance`, `TestToolMyParticipation`, `TestProfileUnder3s`, `TestSeedCRMExpected`, `TestSeedIsolation`, `TestSeedAdminForbidden`, tool agent, lệnh admin, contract, kế hoạch tươi, `check-crm-seed.mjs`, Playwright attendance + me + students + activity). Chạy mọi bước trừ `check-crm-seed.mjs` (cần stack): PASS.

## Test
- `internal/seedtool/seedtool_test.go` `TestSeedCRMExpected`: dựng 30 sinh viên + 15 buổi trong Postgres thật bằng ĐÚNG kế hoạch / hồ sơ của `seed/`, kiểm 270 dòng, buổi 10 chưa có, ABSENT 18–26, khớp `expected_crm.csv` qua `v_attendance_summary`, ≥ 60 ngày thời gian học, chạy lại 0 dòng, phút học tuần của A/B/C/sv12/sv19, rồi `crm.RunCourse`: **đúng một AT_RISK (C), hai WATCH (sv12, sv19), A và B NONE**.
- `cmd/gateway/seedcmd_test.go`: `TestSeedCmdEnvAllowlist` (production, staging, `local`, rỗng, lạ → ≠ 0, không ghi gì; dev / test → 0), `TestSeedStudyTimeCmdIdempotent`, `TestSeedStudyTimeProfileFromStdin`, `TestSeedAgeTicketsMovesWaitingSince` (ticket quá hạn `waiting_since < now − 80 h`, ticket khác không đổi), `TestSeedAgeActivityIdempotent`.
- `internal/crm/me_test.go`: `TestSeedIsolation`, `TestSeedAdminForbidden` (ADMIN 403 hồ sơ / danh sách / điểm danh; ticket do test escalation và `check-crm-seed.mjs` phủ).

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1
go test -count=1 -race -tags testroutes ./internal/seedtool ./cmd/gateway ./internal/crm -run 'TestSeed'
node ../scripts/crm-plan.mjs --json | cmp - ../seed/crm_plan.json && node ../scripts/crm-plan.mjs --csv | cmp - ../seed/expected_crm.csv
bash ../scripts/gate-p4.sh
# trên stack đã seed:
node ../scripts/check-crm-seed.mjs && bash ../scripts/gate-p5.sh
```

## Chạy thật (stack sạch, `down -v` rồi `up`)
- `node scripts/seed.mjs` từ DB trống: 17 bước, **388 s** (docling thật cho bước 11 / 13). Chạy lại: 19 s, "không đổi".
- `check-crm-seed.mjs`: attendance=270, absent=21, speak=65, risk AT_RISK 1 / WATCH 2 / NONE 27, idempotent=ok, 0 ticket ở lớp 761988.
- `gate-p4.sh` PASS (8 bước), `gate-p5.sh` PASS (15 bước), cùng lượt.
- Mail ANSWERED: Mailpit có "Giảng viên đã trả lời câu hỏi của bạn" tới sv17. Bài AI: 3 VERIFIED, 2 CORRECTED, 1 REJECTED, 7 PENDING; `knowledge_items` = 6 (chưa gọi `rag.SearchStudent` riêng).

### Lỗi tìm thấy KHI chạy thật (đều đã sửa, commit riêng)
1. **Judge arm64**: `.env.local` có `JUDGE_EXTRA_ARGS=` trống → `g++` trong sandbox báo `cannot execute 'cc1plus'` → bước 10 lỗi. Đúng như ghi chú D58 (colima arm64 cần `-no-seccomp`); chỉ đổi `.env.local` (gitignored), không đổi mã.
2. **STUDY không bao giờ có dữ liệu** (`seedtool.SeedStudyTime`): `crm.risk` chỉ tính STUDY khi sinh viên đã xác nhận thông báo TỪ TRƯỚC tuần đó (SRS 4.7.1); sinh viên seed đăng ký hôm nay nên `tracking_notice_ack_at` là NULL ở cả 4 tuần → kết quả at-risk phụ thuộc hoạt động chat rải ngẫu nhiên (live: AT_RISK 0, WATCH 1/0). `TestSeedCRMExpected` xanh vì thế giới test không có hoạt động chat nên che lỗi. Sửa: `SeedStudyTime` đặt `tracking_notice_ack_at` về trước tuần 1 cho sinh viên có dòng; test thêm khẳng định mọi tuần của mọi sinh viên có `study`.
3. **Chạy lại `seed.mjs` lỗi** ở bước 16 (`escalate → 422 MESSAGE_NOT_ESCALATABLE`: tin mới nhất giờ là `STAFF_ANSWER`) và bước 17 (`report → 404` vì thread đã tự ẩn). Sửa: chọn tin có `can_escalate` hoặc đã có `ticket` và dùng ticket đó; bước 17 thoát sớm khi hàng đợi kiểm duyệt đã ≥ 2.
4. **`TestAdminNotifiedDailyNotOnce` (US-P4-05) đỏ từ 16/10** (làm `gate-p4` FAIL): job SLA hỏi "đã báo hôm nay chưa" bằng `notifications.created_at >= đầu ngày` (đồng hồ DB thật) trong khi ngày lấy từ đồng hồ của job (đồng hồ giả của test: 12–17/10) → khi ngày thật rơi vào dải đó, thông báo ngày 1 bị coi là đã báo ngày 2. Mã chạy thật không hỏng (một đồng hồ) nhưng test đỏ vĩnh viễn từ 16/10. Sửa: `NotifyStalledToday` tra theo khoá dedupe `stalled:<lớp>:<yyyymmdd>` (cùng hàm `stalledKey` với handler).

### Quan sát (không sửa)
- Lớp 761987 có **129 ticket**, 125 là `AUTO_LOW_CONFIDENCE` OPEN do trợ lý (LLM giả) trả lời độ tin cậy thấp ở hầu hết ≈ 150 câu chat. Năm ticket "đủ trạng thái" vẫn đúng; `check-crm-seed` chỉ đòi ≥ 5. Nếu PM muốn chỉ ~5–10 ticket thì phải đổi câu hỏi bước 12 hoặc ngưỡng độ tin cậy của provider giả — việc của spec, chưa làm.

## Cần PM / nợ
- `rag.SearchStudent` trên bài AI VERIFIED chưa gọi riêng (chỉ đếm `knowledge_items`).
- Spec nói buổi `attendance_done_at` "buổi 1–9": seed gửi `complete:true` ở cả chín buổi (kiểm bằng `done_at` của phản hồi).
- `k6 run benchmarks/load/heartbeat.js` đã được QC chạy ở P5-03 (đạt).

## CI
Run `38782040813` (commit `0f9ff865`) xanh toàn bộ, gồm sửa `TestAdminNotifiedDailyNotOnce` (khoá dedupe ngày) và BUG-S7-03 (#36: `TestStudentNeverSeesConfidence`, `TestMaskConcurrentSameSession`).

## Vòng 2 (BUG-S1, #40, TimingEqualized)
Theo `proposals.md`: BUG-S1 (PM), #40 (ACCEPTED). Stack sạch `down -v` → `up --build --profile ingest` → `seed.mjs` từ DB trống 250 s.

### BUG-S1 — đúng 5 ticket, B không có ticket
- **Gốc**: `LLM_PROVIDER=fake` nhúng bằng băm của CẢ chuỗi (`fake.Vector`), không có ngữ nghĩa ⇒ mọi câu hỏi tự nhiên có cosine < `RAG_SIM_FLOOR` (0,25) ⇒ `NoContext` ⇒ `low_confidence` ⇒ `ShouldEscalate` tự mở một ticket mỗi câu (không có công tắc tắt). Bước 12 chạy ≈ 140 câu về chủ đề tài liệu ⇒ 126 ticket. Không thể "câu mẫu trả lời được" bằng câu tự nhiên với provider giả (cosine chỉ = 1 khi chuỗi y hệt đoạn tài liệu).
- **Sửa**: lựa chọn thứ hai của PM — lịch sử chat ghi thẳng: `gateway admin seed-chat --course 761987 --weeks 10 --plan -` (JSON qua stdin, cùng allowlist `APP_ENV ∈ {dev, test}`; `internal/seedtool/chat.go`). Id suy từ khoá (SHA-1) nên chạy lại không thêm gì; mốc thời gian rải theo băm trong 10 tuần (một phiên một ngày). Tin đáp `COURSE_QA`, `confidence 0,900` (câu ngoài tài liệu: `no_context`, `low_confidence`, 0,150), KHÔNG ticket, không `llm_audit`. `seed.mjs` bước 12 chỉ dựng kế hoạch (cùng 150 câu, cùng phân phối phiên: sv.gioi 4 phiên) rồi gọi lệnh này.
- **Kết quả** (`check-crm-seed.mjs`, thêm hai khẳng định: `tickets === 5`, B không có ticket): `tickets=5 {"OPEN":2,"CLAIMED":1,"ANSWERED":1,"CLOSED":1}` — sv14 OPEN, sv15 OPEN (quá hạn), sv16 CLAIMED, sv17 ANSWERED, sv18 CLOSED, đều `STUDENT_REQUEST`; risk vẫn AT_RISK 1 / WATCH 2 / NONE 27; chạy lại `idempotent=ok`; `check-chat-seed`: chat=150, outside=10, phiên ≥ 33 đạt.
- Test mới `TestSeedChatNoTickets` (ghi hai lần, 0 ticket, câu ngoài tài liệu `no_context`, mốc ≤ hôm nay, sinh viên không ACTIVE → lỗi).
- **Mất so với trước**: lịch sử chat seed không đi qua `llm_audit` / mask nữa (không còn dữ liệu chi phí LLM từ chat seed). Nếu QC / spec cần dòng `llm_audit` cho chat mẫu thì phải dùng provider thật.

### TC-15 — vì sao `citations: []`
- Không phải lỗi nạp tri thức: `knowledge_items` có 6 (5 bài AI VERIFIED / CORRECTED + 1 ticket), đã nhúng. Đoạn tri thức được nhúng là `"Câu hỏi: <tiêu đề>\n<nội dung>\nTrả lời: <đáp>"` (`knowledge.chunkText`); với provider giả chỉ chuỗi y hệt mới có cosine 1. Hỏi bằng tiêu đề thread ⇒ cosine ≈ 0 ⇒ `citations: []`, `low_confidence: true` (đúng như QC thấy). Với embedding thật điều này không xảy ra.
- Kiểm bằng đúng chữ đoạn tri thức (đọc từ Postgres): `node scripts/check-knowledge-seed.mjs` → `tri thức=6 (POST 5, TICKET 1) có trích dẫn=6`; mỗi câu trả về citation `"Hỏi đáp đã được giảng viên xác nhận"`, không `low_confidence`. Cần `docker compose … exec postgres psql` (đặt `SEED_PSQL_CMD` nếu stack không phải project `edupilot`). Script xoá phiên thử; chạy lần hai cùng câu hỏi thì trả từ cache câu trả lời (`agent.Cache`, theo lớp, chỉ giữ chữ, `citations` rỗng) — script chấp nhận khi chữ vẫn mang tên tài liệu.
- Tiêu đề 5 thread seed lấy từ chữ trích PDF (dòng đầu đoạn) nên xấu ("ĐẠI HỌC QUỐC GIA HÀ NỘI TRƯỜNG QUÓC TÉ …"): do `threadFromChunk` cần chuỗi y hệt đoạn cho AI trả lời với provider giả. Không đổi.
- Hệ quả đã biết: `check-chat-seed.mjs` (sprint 6) đòi 12 thread hiện ở lớp 1, nhưng bước 17 (SEED) cố ý tự ẩn một thread (3 báo cáo) nên script báo 11 / PENDING 3. Không nằm trong `gate-p4` / `gate-p5`; chưa sửa script cũ.

### #40 — `POST …/observations` tôn trọng `Idempotency-Key`
- `crmhttp.Handler.OptIdem` = `OptionalIdempotencyKey(d)` gắn vào route; **tuỳ chọn** chứ không bắt buộc: seed, e2e, kịch bản hợp đồng và client cũ không gửi khoá (frontend `apiClient.post` luôn gửi khoá nên đã được phủ). OpenAPI: tham số `Idempotency-Key` (`required: false`) + câu mô tả.
- `TestObservationIdempotencyKey`: cùng khoá + cùng thân → một dòng, phản hồi phát lại đúng id; cùng khoá + thân khác → 422, vẫn một dòng; khoá khác → dòng mới; không khoá → như cũ. Đã chạy: đỏ khi bỏ `OptIdem` (id khác nhau), xanh khi có.

### `TestJoinFailureTimingEqualized` — không đổi mã, không đổi test
- Gốc đã được TL-3 (sprint 5, `techlead.md`) phân tích: đo trung vị sub-ms (0,5–1,1 ms; `Scan` ~20 trường chậm hơn nhiều lần dưới `-race`) trên máy dùng chung — nhiễu 0,2–0,6 ms vượt 35 %. `lookupByCode` đã một truy vấn + một quyết định gộp cho cả sáu nguyên nhân. Quy trình được duyệt: bước song song `EP_SKIP_TIMING=1` (bỏ các test `*TimingEqualized`), rồi chạy riêng tuần tự `-p 1 -count=3` không `-race` (`ci.yml`). Lượt đỏ của QC (0,633) là `go test -race ./...` trần không `EP_SKIP_TIMING` — ngoài quy trình đó.
- Không tái hiện được dù cố ý: 28 lượt `-race` dưới 22–44 vòng lặp bận / song song với gói `contract`, `escalation`, `crm`, `thread`, `exam`, `GOMAXPROCS=2` — đều xanh. Bước tuần tự đúng quy trình: `go test -p 1 -count=3 -tags testroutes -run 'TimingEqualized$' ./internal/auth/ ./internal/course/` → ok (auth 25 s, course 2,7 s).
- Thêm `scripts/go-test-all.sh` (hai bước trên, y hệt CI) để QC / dev khỏi chạy `go test -race ./...` trần. Nếu muốn chuyển phép đo sang đếm truy vấn tất định thì là (a) của TL-3 và cần proposal; chưa làm.
