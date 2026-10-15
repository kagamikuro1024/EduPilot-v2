# QC report — US-S7-SEED — **vòng 2** (BUG-S1, TC-15)
Ngày 2026-10-16. Commit kiểm: `sprint/7-p4-p5` @ `a1ff631e` (code dev `182a6d74`, CI `38797158070` xanh). Stack `edupilot100` **sạch** dựng lại từ nhánh (`down -v`, `up --build --profile ingest`, `--scale gateway=2`); `seed.mjs` từ DB trống **416 s** (docling thật), kết thúc `check-crm-seed.mjs` ok.

## Kết luận vòng 2: **FAIL (thấp)** — 24/25 TC PASS; BUG-S1 và TC-15 đã đóng; **1 TC FAIL mới: TC-S7-SEED-17** (BUG-S2)
| TC | Vòng 2 | Bằng chứng |
| --- | --- | --- |
| 12 (BUG-S1) | **PASS** | `escalation_tickets` lớp 761987 = **5**: `OPEN` 2 (một quá hạn `waiting_since` > 80 h), `CLAIMED` 1 (bởi `teacher@`), `ANSWERED` 1, `CLOSED` 1 (`STUDENT_CONFIRMED`); sinh viên B (`sv.kha`) **0** ticket; lớp 2 **0**; Mailpit 50 thư (có "Giảng viên đã trả lời…"). `check-crm-seed.mjs`: `tickets=5`; chat lịch sử 150 câu / 33 phiên ghi thẳng (`gateway admin seed-chat`), không `llm_audit` cho chat seed (dev đã nêu) |
| 14 | PASS | `run-job ticket.sla`: ticket quá hạn `sla_stage 2`; `/me/today` mục `TICKET` `overdue` |
| 15 (TC-15) | **PASS** | hàng chờ 2, `post_reports` 5, 1 ghim, 1 thread ẩn, 1 bình luận ẩn có lý do, bài AI VERIFIED 3 / CORRECTED 2 / REJECTED 1, `knowledge_items` 6, `POST_REPORTED` ở `Hôm nay`; RAG: `node scripts/check-knowledge-seed.mjs` → `tri thức=6 (POST 5, TICKET 1) có trích dẫn=6` (mỗi câu hỏi bằng đúng chữ đoạn tri thức trả citation "Hỏi đáp đã được giảng viên xác nhận"). Hỏi bằng tiêu đề thread vẫn `citations: []` — do embedding giả băm cả chuỗi (dev giải thích: không xảy ra với embedding thật). QC chấp nhận kiểm bằng script cho provider giả |
| 01–11, 13, 16, 18–25 | PASS (chạy lại) | 270 dòng điểm danh, A/B/C khớp `expected_crm.csv`, ABSENT 21, phát biểu 65 · 16,25 (B 2 = 0,50), 68 ngày · 4.110 dòng · 1.883.002 s, risk `AT_RISK 1 / WATCH 2 / NONE 27` (lớp 2 NONE 24), `STUDENT_ATTENTION` "1 sinh viên mới vào diện cần chú ý · 761987", cờ lớp `t` / `f`, phân quyền A/B/C/D/TA/ADMIN đúng, `gate-p4.sh` và `gate-p5.sh` rc=0; ba lần dựng độc lập cho cùng số liệu (TC-04) |
| 17 | **FAIL (BUG-S2)** | AC6: "`NODE_ENV=production node scripts/seed.mjs; echo $?` → khác 0". Thực tế `scripts/seed.mjs:37` chỉ chặn khi `APP_ENV === "production"`; với `NODE_ENV=production` (và `API_URL` trỏ stack thật) seed **chạy** và thoát `0`. **Đính chính**: vòng 1 tôi ghi PASS vì lúc đó thoát `1` — do không kết nối được `https://localhost/api/v1`, không phải do chặn production. `APP_ENV=production` thoát `1` đúng ("Seed bị chặn ở production."). Sửa: chặn cả `NODE_ENV=production` hoặc sửa AC / TC thành `APP_ENV` |
| 22, 23 | PASS (giữ từ vòng 1) | bước 14 / lỗi API không đổi ở vòng 2 |
Quan sát: `check-chat-seed.mjs` (sprint 6) vẫn báo "thread lớp 1 = 11 (cần 12)", "PENDING = 3 (cần 4)", "sv.gioi thấy 10 thread (cần 12)" — #41 (ACCEPTED) sửa script này nhưng commit của dev chưa có trên nhánh `a1ff631e` (`git log` chỉ có "PM quyết #41"). Không nằm trong `gate-p4/p5`.

---

# QC report — US-S7-SEED (dữ liệu seed sprint 7, lệnh quản trị, hai cổng)
Ngày 2026-10-16. Commit kiểm: `sprint/7-p4-p5` @ `0f9ff865` (CI `38782040813` **xanh**, 10 ph 59 s). Bộ TC: `tc-US-S7-SEED.md` (một chỗ sửa theo `#35`: TC-S7-SEED-08, ghi ở "Lịch sử sửa TC"). Handoff: `dev-US-S7-SEED.md`. Stack: `edupilot100`, **dựng sạch** từ nhánh (`down -v`, `up --build --profile ingest`, `docker-compose.test.yml`, `--scale gateway=2`, docling thật, judge `-no-seccomp`), `node scripts/seed.mjs` từ DB trống **175 s** (lần có docling), chạy lại 19 s. Lưu ý dựng: cổng MinIO của stack QC là 9300 nên phải đặt `BLOB_PUBLIC_ENDPOINT=localhost:9300` (nếu không bước 11 "fetch failed" — do lệch cổng của QC, không phải lỗi seed).

## Kết luận: **FAIL** — 23/25 TC PASS; 1 FAIL (TC-12), 1 PASS một phần (TC-15); không lỗi mức nghiêm trọng
Một AC không đạt: AC4 / TC-S7-SEED-12 (xem BUG-S1). Các gate do dev viết (`gate-p4.sh`, `gate-p5.sh`) chạy `rc=0` và thoát khác 0 khi một lệnh đỏ.

## 1. Lỗi
**BUG-S1 (trung bình) — AC4 / TC-12: 129 ticket ở lớp 761987, không phải 5; sinh viên B có ticket.**
`select kind,status,count(*) from escalation_tickets where course_id='<761987>' group by 1,2` → `QUESTION|OPEN|126`, `CLAIMED|1`, `ANSWERED|1`, `CLOSED|1` (tổng 129; lặp lại ở ba lần dựng độc lập). 125–126 ticket `OPEN` sinh tự động vì trợ lý (LLM giả) trả lời độ tin cậy thấp ở hầu hết ≈ 150 câu chat bước 12 — dev đã ghi ở "Quan sát (không sửa)". TC-12 yêu cầu "5 phần tử … **không** ticket của B": `B` (`sv.kha`) có 5 ticket `OPEN` (A 4, C 5; 30 sinh viên đều có ≥ 1). Hậu quả: kịch bản demo "D3" (B tự tạo ticket mới), `/inbox` của giảng viên 100+ dòng, chuông giảng viên: thông báo `TICKET_NEW` gộp ("122 câu hỏi đang chờ giảng viên") nên số chưa đọc **không đổi** khi có ticket mới (đo ở G4-11). Gợi ý: đổi câu hỏi bước 12 hoặc chỉ cho bước 12 ở chế độ không tạo ticket tự động.

## 2. Kết quả TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `attendance_records` lớp 1 = **270**; buổi 1–9 `attendance_done_at` có; buổi 10 (16/10, hôm nay) chưa |
| 02 | PASS | `v_attendance_summary`: A 9/9/0/0/0; B 9/6/1/0/2; C 9/3/1/0/5 — khớp `seed/expected_crm.csv` (kèm phát biểu A 12 / 3,00, B 2 / 0,50, C 1 / 0,25) |
| 03 | PASS | ABSENT tổng **21** (18–26); `participation_events` 65 (≈ 60), Σ 16,25, A nhiều nhất; **B đúng 2 `SPEAK` = 0,50** |
| 04 | PASS | ba lần dựng độc lập (stack 1 + hai lần `down -v`): 270 / 21 / 65 · 16,25 / thời gian học 68 ngày · 4.110 dòng · 1.883.002 s / risk 27·2·1 / ticket 129 / 5 báo cáo / 6 tri thức / 12 thread — **giống hệt** |
| 05 | PASS | `TestSeedCRMExpected` ok (trong `gate-p5.sh`; vị trí `./internal/seedtool`, không cần tag `integration`) |
| 06 | PASS | `gateway admin seed-study-time --course 761987 --weeks 10 --profile -` (đọc hồ sơ từ stdin): `count(distinct date)` **68** (≥ 60); 4.110 dòng |
| 07 | PASS | phút/tuần: C `120 ×6, 30, 28, 35, 85` (tuần 7–9 < 60); A 300 ×9; B 150 ×9; lớp 2 có 3 tuần |
| 08 | PASS (theo #35) | production / staging / `local` / rỗng → thoát `1`, "chỉ chạy khi APP_ENV thuộc dev, test"; `dev`, `test` → `0`, "0 dòng" (chạy lại). TC viết `local → 0`: spec đổi theo `#35`, TC sửa (có ghi lịch sử) |
| 09 | PASS | chạy lại: `count` 4.110, `Σ active_seconds` 1.883.002 — không đổi |
| 10 | PASS | `AT_RISK` 1 (C = Ngô Ngọc Cẩm, câu "Cần chú ý: vắng 2/2 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 31 phút mỗi tuần; không hỏi AI hay đăng bài ở Threads trong 1 tuần."), `WATCH` 2, `NONE` 27; lớp 2 `NONE` 24; `students?filter=attention` đúng 1 |
| 11 | PASS | `STUDENT_ATTENTION` "1 sinh viên mới vào diện cần chú ý · 761987"; `attention` = [Ngô Ngọc Cẩm], tổng 1 |
| 12 | **FAIL** | BUG-S1 |
| 13 | PASS | `seed-age --tickets --ticket T --hours 24`: "1 dòng", `created_at` và `waiting_since` lùi 24 h; hash `id|status|kind|student|answer` của ticket và `id|content` của `chat_messages` trước/sau **không đổi**; `seed-age --activity` chạy lại "0 dòng"; `APP_ENV=production seed-age` → thoát `1` |
| 14 | PASS | `gateway admin run-job ticket.sla`: ticket `OPEN` quá 80 h → `sla_stage 2`; `/me/today` mục `TICKET` "Câu hỏi chờ giảng viên · lớp 761987" `urgency=overdue` |
| 15 | **PASS một phần** | hàng chờ kiểm duyệt 2 (thread 3 báo cáo `OFF_TOPIC`, thread 2 báo cáo), `post_reports` 5 (đều `OPEN`), 1 thread ghim (`pinned_at`), 1 thread ẩn, 1 bình luận ẩn có `hidden_reason`; bài AI `VERIFIED` 3 / `CORRECTED` 2 / `REJECTED` 1; `knowledge_items` 6; `Hôm nay` có `POST_REPORTED` "2 bài bị báo cáo chờ xem". **Vế RAG không kiểm được**: hỏi đúng tiêu đề thread đã `VERIFIED` ở chat riêng → `citations: []`, `low_confidence: true` (embedding giả không có ngữ nghĩa; dev ghi "chưa gọi `rag.SearchStudent` riêng"). Ghi chú: tiêu đề 5 thread seed lấy từ chữ trích PDF ("ĐẠI HỌC QUỐC GIA HÀ NỘI TRƯỜNG QUÓC TÉ …", "▪ Example: Auger's Plumbing Service") — không đọc được như câu hỏi thật |
| 16 | PASS | chạy lại `seed.mjs` (19 s, "không đổi") rồi `check-crm-seed.mjs`: 270 / 65 / ticket 129 / 5 báo cáo / 6 tri thức / 12 thread không đổi, `idempotent=ok` |
| 17 | PASS | `NODE_ENV=production node scripts/seed.mjs` → thoát `1`; `docker-compose.local.yml:127 SUPPORT_RESOURCES_VI` có; dựng bằng `compose up` + `seed.mjs` (không qua `pnpm dev` vì stack QC dùng cổng riêng) — đủ dữ liệu của story |
| 18 | PASS | `bash scripts/gate-p4.sh` `rc=0` ("GATE P4: PASS", 8 bước, `-race`, ba test chạy thật); `gate-p5.sh` `rc=0` ("GATE P5: PASS", 15 bước, gồm `check-crm-seed.mjs --no-rerun`, Playwright, `TestSeedCRMExpected`). QC chèn `run "QC injected red" false` ở bản sao tạm của từng script → `rc=1`, dòng "FAIL | QC injected red", "GATE P5/P4: FAIL"; bản sao đã xoá, `git status` sạch |
| 19 | PASS (ghi chú) | `escalation.spec.ts` có `inbox: danh sách → … → Gửi`, `inbox claim conflict` (hai người `Nhận`), `observability failed mail`, `knowledge save`; `threads-moderation.spec.ts` phủ F4; `attendance.spec.ts` có `offline queue sync` (đánh vắng 3 người), `two teachers conflict`, `375 scripted under 15s`, chạy desktop + mobile; tất cả xanh. Ghi chú: các spec chạy trên gateway **giả** (`page.route`), không dùng tài khoản seed như TC viết |
| 20 | PASS | A, B, C: `me/attendance` `200`, `…/students/{uid}/profile` và `…/students` `403`; `sv.moi`: `me/today` `200`, `me/attendance` `403`; TA: lớp 1 `students` `200`, lớp 2 `403`; ADMIN: `tickets`, `profile`, `attendance` `403`; crawl `sv04` (AT_RISK + canary) không có khoá `risk*`/`observation*`/canary (xem report US-P5-06) |
| 21 | PASS | `TestSeedIsolation`, `TestSeedAdminForbidden` ok (`./internal/crm`) |
| 22 | PASS | `kill -9` tiến trình `seed.mjs` khi `attendance_records` = **210** (7/9 buổi, `participation_events` 0); chạy lại → **270**, 0 cặp trùng `(course,session,student)`, ABSENT 21, phát biểu 65 · 16,25, `check-crm-seed.mjs` ok — cùng số liệu như chạy một lần |
| 23 | PASS | `compose stop gateway` rồi `seed.mjs`: `[1/17] Admin đầu tiên…` → "Seed lỗi: POST /auth/login → 503", thoát `1` |
| 24 | PASS | `761987 | t`, `761988 | f`; `track_study_time=false` 0 người; `PUT $Q/settings {allow_study_time_opt_out:false}` teacher `409`, TA / `$SVA` / ADMIN `403` |
| 25 | PASS | `/settings` của A (API `study_time.courses`): 761987 `locked:false`, 761988 `locked:true`; chạy lại seed không đổi hai giá trị, không lỗi |

## 3. Phụ lục — vài số liệu
Kế hoạch seed tất định (`crm-plan.mjs` so `seed/crm_plan.json`, `seed/expected_crm.csv`: `cmp` bằng nhau). Mailpit sau seed: 50 thư (có "Giảng viên đã trả lời câu hỏi của bạn" tới sinh viên ANSWERED).
