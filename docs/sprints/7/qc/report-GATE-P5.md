# QC — báo cáo cổng P5 (CRM, hồ sơ 360, thời gian học) + seed sprint 7
Ngày 2026-10-16. Commit: `sprint/7-p4-p5` @ `0f9ff865`; CI `38782040813` **xanh**. Bộ kiểm: `gate-P5.md` (G5-01…34), `tc-US-P5-01…06.md`, `tc-US-S7-SEED.md`, `.claude/skills/gate/SKILL.md`. Cùng stack sạch với `report-GATE-P4.md`. Rà soát thay đổi spec / TC từ APPROVED: xem mục 1 của `report-GATE-P4.md` (dùng chung; kết quả: 0 TC bị vai khác sửa; 2 commit sửa spec không ghi số proposal — `803393b7`, `30f81cae`; test cũ bị sửa đều dẫn tới proposal ACCEPTED trừ ba dòng helper của BUG-S7-02).

**Cập nhật 2026-10-16 (#38):** US-P5-06 TC-17 chấm lại theo LCP devtools → **PASS** (1.478 ms ≤ 2.500 ms); `G5-23` thành PASS. Điều kiện (3) bên dưới đã đóng; còn BUG-S1 / TC-S7-SEED-15 (chờ dev, #40).

## Kết luận cổng P5: **KHÔNG ĐẠT** — 1 TC FAIL + 1 một phần ở `G5-32` (US-S7-SEED) và 1 TC FAIL ở `G5-23`/US-P5-06; không lỗi nghiêm trọng
Phần lõi P5 (điểm danh, lịch, thời gian học, at-risk, hồ sơ 360, ghi chú, nhận xét AI, nhắn sinh viên, không rò dữ liệu nhạy cảm, hiệu năng) đạt. Không đạt vì: (1) **BUG-S1** — seed lớp 1 có 129 ticket (126 `OPEN` tự sinh), B có ticket (TC-S7-SEED-12; AC4); (2) vế RAG của TC-S7-SEED-15 không kiểm được; (3) US-P5-06 TC-17: `/me` LCP mô phỏng 2,7 s > 2,5 s. Sửa BUG-S1 + PM quyết (2), (3) thì chạy lại `check-crm-seed.mjs` và `G5-32` là đủ.

## 1. Số liệu (mục D)
| Số liệu | Kết quả |
| --- | --- |
| TC theo story (PASS / FAIL / KHÔNG KIỂM ĐƯỢC) | P5-01: 0 FAIL · P5-02: 0 FAIL · P5-03: 0 FAIL (vòng 3) · P5-04: 0 FAIL (3 gián tiếp) · P5-05: 0 FAIL (vòng 2; 3 gián tiếp) · P5-06: **1 FAIL** (TC-17) · SEED: **1 FAIL** (TC-12), 1 một phần (TC-15), 23 PASS |
| Điểm danh 30 SV (giả lập) | điện thoại 375 px: **4,4 s** (≈ 8 chạm, kịch bản QC) — e2e `attendance 375 scripted under 15s` xanh; bàn phím 1–4: 30 phím **58 ms**, gõ + `Lưu điểm danh` ≈ **3 s**; 1 `PUT complete:true` cho cả 30 hàng, 0 hộp thoại. Điện thoại thật: phần chủ dự án tự kiểm (Q-QC-P5-14) |
| p95 heartbeat (k6 10 req/s × 60 s) | **8,82 ms**, 601 lượt, 0 lỗi (ngưỡng < 20 ms) |
| p95 `GET …/students` trang đầu lớp 1.000 SV; hồ sơ lớp 1.000 SV | `TestStudentsListP95Under300ms` và `TestProfileUnder3s` xanh trong lượt full `-race` có timing (số p95 ≈ 49 ms ở report US-P5-04; không đo lại bằng `curl` — QC không gieo `seed-perf.sql`) |
| Test Go / Playwright trước–sau | Go `Test…` 1.477 → 1.927 (0 bị xoá); Playwright 279 → 394 khai báo; lượt đủ 692 passed / 13 failed (`visual.spec` macOS) / 369 skipped / 0 flaky |
| Vòng chấm AC13 (P5-02) | AC13 chấm ở vòng 2 (report US-P5-02), không đổi |
| Volume test (BUG-S7-02) | 2,80 → 2,71 GB sau một lượt full Go, 18 volume, 0 container mồ côi |

## 2. Dòng A — lệnh cổng
| ID | KQ | Bằng chứng |
| --- | --- | --- |
| G5-01 | PASS | `go test -race ./internal/crm/... ./internal/activity/...` ok (75 s + 31 s), không `DATA RACE`; có test idle / tab ẩn / hai tab / STUDENT đọc observation `403` |
| G5-02 | PASS | `TestToolMyAttendance`, `TestToolMyParticipation`, `TestPersonalToolsHaveNoIdentityParam` ok |
| G5-03 | PASS | `attendance.spec.ts` 17 passed (desktop; mobile chạy cùng tệp), 0 flaky, có bước ngắt mạng |
| G5-04 | PASS | `bash scripts/gate-p5.sh` rc=0 "GATE P5: PASS" (15 bước, gồm `check-crm-seed.mjs --no-rerun` trên stack sạch); `rc=1` khi chèn lệnh đỏ |
| G5-05 | PASS có điều kiện | như G4-07: vet / lint sạch; full `-race`: lần 1 một test thời gian (`course.TestJoinFailureTimingEqualized`) đỏ khi tải, lần 2 xanh; không test nào bị xoá |
| G5-06 | PASS | `contract` ok; `sqlc diff` rc=0; `TestMigrations_RoundTrip` ok; `00013_crm` có 1 `ALTER TABLE`, `00015_observation` 2 (số `00014` thuộc P4: `forum_ai_regen`, #25); hợp đồng chỉ cộng (ngoài các dòng #11 …) |
| G5-07 | PASS | `pnpm lint`, `pnpm build` ok; `ui-antipatterns.sh` rc=0 |
| G5-08 | PASS có điều kiện | xem mục 1 của `report-GATE-P4.md` |
| G5-09 | PASS | k6 p95 8,82 ms, 0 lỗi |
| G5-10 | PASS | `TestGradeNeverImportsActivity`, `TestGradeQueriesNeverReadStudyTime`, `TestGradeRuleScannersCatchViolations` ok; tự thử đỏ hai đường đã làm ở US-P5-03 vòng 1 (gói giả import `crmhttp` → FAIL; gói giả gọi `CrmActiveStudentIDs` → FAIL, đã xoá); hôm nay chỉ chạy lại bài quét tự kiểm |

## 3. Dòng B — kiểm tay
| ID | KQ | Bằng chứng |
| --- | --- | --- |
| G5-11 | PASS | số đo ở mục 1 (giả lập; không điện thoại thật) |
| G5-12 | PASS | ngắt mạng, đánh vắng 3 người, nối lại: 3 `PUT`, "Đã lưu"; tải lại giữa chừng giữ 3 thay đổi, `Lưu điểm danh` offline rồi online gửi `complete:true`; hai người cùng lưu: thông báo xung đột "… vừa đổi 1 dòng · Giữ của tôi · Lấy của …", thay đổi riêng giữ nguyên. (Một lần chạy script lặp báo "2 thay đổi" vì hàng đã ở trạng thái Vắng từ lần trước — nhiễu trạng thái của script) |
| G5-13 | PASS | C ở "Cần chú ý": "Cần chú ý: vắng 2/2 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 31 phút mỗi tuần; không hỏi AI hay đăng bài ở Threads trong 1 tuần." |
| G5-14 | PASS | sinh viên: 0 ghi chú / nhãn / canary ở `/`, `/me`, `/chat`, `/threads`, `/settings`, `/calendar`; thấy thời gian học của mình (`/me`, `…/me/study-time`) |
| G5-15 | PASS | lớp khoá → công tắc bật + khoá + tooltip; lớp cho tắt: tắt công tắc → nhịp `204` không đo, `study_time_daily` không đổi (US-P5-03 TC-34, 36, 43; US-P5-06 TC-22) |

## 4. Dòng C — kiểm chéo
| ID | KQ | Bằng chứng |
| --- | --- | --- |
| G5-16 | PASS | crawl 12 route của sinh viên AT_RISK có canary: 0 khoá `risk*`/`observation*`/`at_risk`/`note`, 0 canary; hỏi chat "Giảng viên ghi gì về em?" trả "Mình chưa tìm thấy…"; log gateway / worker, payload LLM, `audit_log`, `llm_audit`: 0 `CANARY-OBS-4F` (US-P5-04 TC-30/31, US-P5-05 TC-26/28, US-P5-06 TC-06/07 chạy lại trên stack mới) |
| G5-17 | PASS | ADMIN `403` ở `tickets`, `profile`, `attendance`; TA lớp 1 gọi lớp 2 `403`; sinh viên lớp khác `404` |
| G5-18 | PASS | mỗi lần gọi `profile` = 1 dòng `audit_log` `view` (+3 sau 3 lần); `view_observations`; `503` khi `audit_log` lỗi (trigger tạm); UI: 3 lần mở = 3 request |
| G5-19 | PASS | xem G5-11, 12; `get_my_attendance` / `get_my_participation` và hỏi hộ: `TestToolMy*`, `TestPersonalToolsHaveNoIdentityParam` ok |
| G5-20 | PASS | `p5-01-ui.mjs`: nút tạo lịch theo vai (STUDENT 0), xem trước 11 dòng "Sẽ tạo 11 buổi · bỏ 1 ngày nghỉ", offline giữ chữ, 375 px không tràn; `generate` thật cần `Idempotency-Key` (422 khi thiếu), `dry_run` không cần (200) |
| G5-21 | PASS | US-P5-03 vòng 3 PASS (idle 2 phút, tab ẩn, hai tab, heartbeat không ghi Postgres, cờ lớp, `COURSE_HAS_STUDENTS`, đua 50 lượt, `/exams/**` không đo, luật cứng `grade`) |
| G5-22 | PASS | lý do bằng lời cho C ✓; `TestStudents360ExplainNoSeqScan`, `TestStudentsListP95Under300ms` xanh; phân trang `limit 500 → 422`, `cursor` sai → `422`; "Lớp cần chú ý" ≤ 5 dòng + `attention_total`, ẩn khi rỗng |
| G5-23 | **FAIL** | không gamify: 0 từ / `<progress>` / thẻ KPI; 375 px `/attendance`, `/me` không tràn — PASS. **FAIL ở US-P5-06 TC-17** (LCP mô phỏng `/me` 2,7 s > 2,5 s; xem `report-US-P5-06.md`) |
| G5-24 | PASS | ma trận API trên stack sạch: `me/*` STUDENT; `students`, `profile`, `observations`, `attendance`, `study-time` STUDENT / ADMIN `403`, TA / TEACHER đúng; lớp khác `403`/`404`; `PUT …/settings` TA / STUDENT / ADMIN `403` |
| G5-25 | PASS có điều kiện | `generate` không khoá → `422`; `outreach` không khoá → `422`, cùng khoá hai lần → 1 tin; `participation` theo `client_event_id` (e2e). **Phát hiện ngoài danh sách**: `POST …/observations` bỏ qua `Idempotency-Key` — cùng khoá hai lần tạo **hai** ghi chú; thiếu khoá vẫn `201` (SKILL mục 4b: POST có tác dụng phụ mới đều nhận khoá). Spec crm-360 4.8.4 không đòi khoá nên không FAIL; PM quyết |
| G5-26 | PASS | con trỏ + `limit ≤ 100` ở `students`, `me/attendance` (3 trang × 3 không trùng/sót), `observations`, `sessions`; 0 `OFFSET` trong `store/queries`; EXPLAIN lớp 100 SV không `Seq Scan`: Go xanh |
| G5-27 | PASS | tải / rỗng / lỗi và 375 px: `/attendance`, `/students` (lỗi + `Thử lại`, "0 sinh viên" + `Bỏ lọc`), `/students/[id]` (lỗi nói dữ liệu không mất), `/me`, ngăn kéo lịch (e2e + UI thật) |
| G5-28 | PASS | `ai-summary`: payload `task UTILITY`, `NEAR_REALTIME`, chỉ số liệu, 0 tên / MSSV / email / canary; log gateway / worker 0 `CANARY-` |
| G5-29 | PASS | migration sạch trên DB trống (dựng sạch hai lần); `00013` 1 `ALTER`, `00015` 2 `ALTER` |
| G5-30 | PASS | `grep -rn "@/mock" frontend/src/features/{attendance,students,me}` = 0 / 0 / 0 |
| G5-31 | PASS có điều kiện | `eval innerText` 12 màn: 0 từ kỹ thuật ở màn sinh viên (ở `/inbox` của giảng viên còn "Độ tin cậy" — màn nhân viên, cho phép). Trái luật câu mô tả dưới tiêu đề (`~/.claude/CLAUDE.md`): `/students` "Chuyên cần và nhãn rủi ro chỉ giảng viên/TA thấy" (đã báo ở report US-P5-04), `/class/settings` "Lớp 761987. Chia sẻ mã hoặc liên kết cho sinh viên; bạn quyết định ai được vào." (từ P2), `/settings` mật khẩu (từ P1). Không có trong TC/spec — PM quyết giữ hay xoá |
| G5-32 | **FAIL** | `tc-US-S7-SEED.md`: 23 PASS, **TC-12 FAIL**, TC-15 một phần (RAG) — xem `report-US-S7-SEED.md`; `seed/expected_crm.csv` khớp; 270 dòng; idempotent; chặn production |
| G5-33 | PASS | F7 (lịch → điểm danh → mất mạng → hai người cùng lưu), F8 (hồ sơ + audit lỗi): e2e + UI thật; F14: `SESSION_UNMARKED` (buổi 10 hôm nay), `STUDENT_ATTENTION` "1 sinh viên mới vào diện cần chú ý · 761987", khối `attention` ≤ 5, `POST_REPORTED`, `TICKET` quá hạn đều có ở `Hôm nay` |
| G5-34 | PASS | không `fetch(` ngoài `shared/`, không thư viện mới, không secret, `float64` ở `internal/crm` / `activity` chỉ trong chú thích (điểm cộng dùng `decimal`), `internal/grade` chưa có (luật cứng được test giữ) |

## 5. Việc cần làm để đổi sang ĐẠT
1. Dev sửa seed bước 12 để lớp 761987 có đúng ~5 ticket "đủ trạng thái" và **không** ticket của B (BUG-S1); chạy lại `check-crm-seed.mjs` và TC-S7-SEED-12, 16.
2. PM quyết vế RAG của TC-S7-SEED-15 (kiểm được hay bỏ khỏi AC5) và tiêu đề thread seed lấy từ chữ trích PDF.
3. PM quyết US-P5-06 TC-17 (LCP 2,7 s mô phỏng; cổng LCP đang `off`).
4. (Khuyến nghị) `POST …/observations` nhận `Idempotency-Key`; xoá các dòng mô tả dưới tiêu đề ở `/students`.

## 6. "Bạn tự kiểm" (chủ dự án) — theo `P5.md`
Bấm giờ điểm danh 30 sinh viên trên điện thoại thật; ngắt chế độ máy bay giữa lúc điểm danh; xem sinh viên C ở "Cần chú ý" với đúng lý do; đăng nhập sinh viên (không thấy ghi chú / nhãn, thấy thời gian học); tắt công tắc ở lớp cho tắt rồi mở `/chat` 1 phút và xem `study_time_daily` không tăng. Ghi `docs/thesis-notes/P5.md`.
