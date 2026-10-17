# QC report — US-P5-05 — **vòng 2** (BUG-A)
Ngày 2026-10-15 (tối). Commit kiểm: `sprint/7-p4-p5` @ `28ab4fb7` (code dev `5be69dbc`, `a908fd9`). Stack `edupilot100` dựng lại từ mã mới, một gateway để đọc `_test/llm/payloads`. Dữ liệu: lớp `QC risk` (`108503`), `sv04` (AT_RISK, có ghi chú canary `CANARY-OBS-4F`).

## Kết luận vòng 2: **PASS** — BUG-A đã sửa; TC-16…22 và E1 kiểm được
Dev đổi task `INSIGHT` → `UTILITY` (làn mặc định `NEAR_REALTIME`, không xin làn). Spec crm-360 AC5 / SRS 4.8.5 còn ghi "task `INSIGHT`" — BA sửa theo handoff (QC không sửa TC; TC-16 viết `INSIGHT`, QC đọc là `UTILITY` theo thay đổi spec của PM/dev).

| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 16 | PASS | `POST …/students/{sv04}/ai-summary` → `200 {draft:{summary,strengths[1],concerns[1],suggestions[1]}}`; `fake_calls` 0 → **1**; `llm_audit`: `UTILITY | NEAR_REALTIME | env:fake | ok | pii_masked_count 0`; thân yêu cầu (bộ đệm payload, 1 mục): prompt hệ thống + JSON số liệu (`attendance{absent 6,present 2,recorded 8}`, `participation`, phút học theo tuần…); grep tên "Vũ Minh Long", MSSV `20224786`, `sv04@`, `CANARY-OBS` = 0 |
| 17 | PASS | `student_observations` 3 → 3 sau khi soạn (không lưu tới khi `Lưu`) |
| 18 | PASS | lưu `kind=AI_SUMMARY, ai_edited=false` → ghi nhận; lưu bản sửa `ai_edited=true` ghi nhận |
| 19 | PASS gián tiếp | provider fake không bịa tên theo yêu cầu; `TestAISummaryRedactsInventedName` xanh |
| 20 | PASS (e2e) | `students.spec` `ai summary edit then save` xanh (nháp `aisum:<uid>` ở máy) |
| 21 | PASS gián tiếp | không dừng được provider fake trên stack; `TestAISummaryLLMDown503`, `TestAISummaryTimeout504` xanh; `503 LLM_UNAVAILABLE` đường thật đã thôi bị `BAD_LANE` |
| 22 | PASS | lần thứ 10 `200`, lần 11 `429`; STUDENT, ADMIN `403`; `NOT_ENOUGH_DATA` + không gọi LLM đã đạt ở vòng 1 |
| E1 | PASS gián tiếp | `TestAISummary*` (schema sai → `503`, không lưu) — không có cách ép provider fake trả JSON sai trên stack |
Lệnh handoff: `go test -race -tags testroutes ./internal/crm -run 'TestAISummary|TestMe' -v` → xanh (có `TestAISummaryThroughRealGateway`, `TestAISummaryNoDataNoLLM`, `TestAISummaryRateLimited`); `./internal/contract/... ./internal/httpapi/...` xanh khi chạy cả gói. Các TC còn lại của vòng 1 giữ PASS (không chạy lại: diff vòng 2 chỉ chạm `aisummary` và hợp đồng của nó).

---

# QC report — US-P5-05 (hồ sơ 360, ghi chú riêng tư, nhận xét AI, `Nhắn sinh viên`)
Ngày 2026-10-15. Commit kiểm: `sprint/7-p4-p5` @ `aff08c2c` (code dev `b23b6477`; CI xanh). Bộ TC: `tc-US-P5-05.md` (không sửa chữ TC). Handoff: `dev-US-P5-05.md` (#29 #32 #33 áp dụng). Stack `edupilot100` dựng lại từ mã mới; một gateway để đọc `_test/llm/payloads`. Dữ liệu thử: lớp `QC risk` (`108503`) như report US-P5-04; ghi chú / nhắn sinh viên thử trên lớp 761987 (sinh viên C = `sv.nguyco`).

## Kết luận: **FAIL** — 1 lỗi cao (BUG-A: `ai-summary` luôn `503`), kéo theo TC-16…21 và E1 không kiểm được
Còn lại (hồ sơ, audit, phân quyền, ghi chú, nhắn sinh viên, không rò nhãn, mock) đạt.

## 1. Lỗi
**BUG-A (cao) — AC5 / AC6: `POST …/students/{uid}/ai-summary` luôn trả `503 LLM_UNAVAILABLE` trên gateway thật.**
Tái hiện: giảng viên `teacher@`, lớp `108503`, `POST /courses/{id}/students/{uid sv04}/ai-summary` → `503 {"code":"LLM_UNAVAILABLE","retry_after":30}` sau ≈ 20 ms; `GET /_test/llm/stats` `fake_calls` không tăng (không có lời gọi nào tới provider). Nguyên nhân (đọc `llm_audit`): `select task,lane,status,error_kind,attempts from llm_audit where task='INSIGHT'` → `INSIGHT | BATCH | error | BAD_LANE | 0`. `internal/crm/aisummary.go:152-153` xin `LaneNearRealtime`; `internal/llm/types.go:60-71` (`ResolveLane`) chỉ cho **hạ** làn so với mặc định, mặc định của `INSIGHT` là `BATCH` ⇒ `ErrBadLane` (`internal/llm/gateway.go:117`). Bài kiểm của dev (`aisummary` với `Structurer` giả, `crm/profile_test.go:291` chỉ khẳng định `*req.Lane == NEAR_REALTIME`) không đi qua `llm.Gateway` thật nên không bắt được. Hậu quả: tính năng "Soạn nhận xét tổng hợp" không bao giờ ra bản nháp; TC-16 yêu cầu chính xác "task `INSIGHT`, làn `NEAR_REALTIME`" nên spec và quy tắc làn (SRS 4.3 "chỉ hạ làn") mâu thuẫn — cần PM / BA quyết (cho phép nâng làn cho `INSIGHT` như `EMBEDDING`, hoặc sửa spec dùng `BATCH`) rồi dev sửa và thêm một bài kiểm đi qua `llm.Gateway` thật + provider fake.

## 2. Kết quả TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `200`; `{student{id,full_name,student_code,joined_at}, risk{level,sentence,reasons[],computed_at}, attendance, participation, study{minutes_7d,tracked,weeks[]}, activity}`; `Cache-Control: no-store`; không `ETag` |
| 02 | PASS | gọi 3 lần → `audit_log` `entity='student_profile'` 5 → 8; mỗi dòng `action='view'`, `actor_id`, `course_id`; `before`/`after` rỗng |
| 03 | PASS | UI thật: vào hồ sơ trực tiếp 3 lần = 3 request `…/profile` = +3 dòng audit; mở bằng bấm dòng ở danh sách ×3 = 3 request (audit tính theo đúng sinh viên bấm) |
| 04 | PASS | trigger `before insert on audit_log` raise → `profile` `503 SERVICE_UNAVAILABLE`, không dữ liệu; gỡ trigger → `200` |
| 05 | PASS | `GET …/observations` thêm 1 dòng `view_observations`; `CANARY-OBS-4F` trong `audit_log` = 0 |
| 06 | PASS | sinh viên PENDING / REMOVED / uid lạ → `404` ×3 |
| 07 | PASS | `log_statement='all'`: `profile` 0 lần `chat_messages` |
| 08 | PASS | ma trận `profile` / `observations` / `attendance` / `study-time` / `ai-summary`: STUDENT (kể cả uid mình), ADMIN, TA lớp khác → `403`; TEACHER lớp đó `200`; sinh viên lớp khác → `404`. (`outreach` xem TC-27) |
| 09 | PASS | TA `PUT`/`DELETE` ghi chú của teacher → `403`; teacher `PUT` ghi chú TA → `403`; mọi staff đọc được cả hai |
| 10 | PASS | `POST` `201`; danh sách mới nhất trước (`next_cursor`); `PUT` `200` version 1→2; `DELETE` `204`, dòng mất thật; `audit_log` không chứa nội dung |
| 11 | PASS | 6 tag, tag rỗng, tag 31 ký tự, body rỗng, body 2.001 ký tự → `422`×5; sai version `409` |
| 12 / 30 | PASS (e2e) | `students.spec` `observation draft survives offline`, `observation conflict` xanh (API giả); chưa thử tay offline / hai ngữ cảnh trên stack thật; `PrivateMark` "Chỉ giảng viên/TA thấy" có ở tab Ghi chú (UI thật) |
| 13 | PASS | `CANARY-OBS-4F` ở bộ đệm payload LLM = 0; trò chuyện của sinh viên không có lời gọi nào chứa ghi chú |
| 14 | PASS | UI thật 1280: đầu trang tên, MSSV · mã lớp, `Thêm ghi chú`, `Nhắn sinh viên`, câu nhận định + "Chỉ giảng viên/TA thấy"; năm tab; Tổng quan một đoạn + term/value, không thẻ KPI (0 phần tử KPI); Điểm: "Điểm cộng đã ghi 0,00 · Lớp chưa có công thức điểm đã xác nhận."; Hoạt động học 4 `svg` (biểu đồ tuần) — chỉ tab này có biểu đồ |
| 15 | PASS | chữ "Tuần trước"; 720 px không tràn; `PageState` đủ (xem TC-31) |
| 16 | **FAIL** (BUG-A) | không có lời gọi LLM nào ra tới provider; không kiểm được "đúng 1 lời gọi `INSIGHT`/`NEAR_REALTIME`", schema, thân yêu cầu |
| 17 | KHÔNG KIỂM ĐƯỢC → FAIL | không có bản nháp (BUG-A). `student_observations` không đổi sau lời gọi lỗi (2 → 2) |
| 18 / 19 / 20 | KHÔNG KIỂM ĐƯỢC → FAIL | phụ thuộc bản nháp AI (BUG-A); e2e `ai summary edit then save` (API giả) xanh nhưng không chứng minh đường thật |
| 21 | **FAIL** một phần | `503 LLM_UNAVAILABLE` + `retry_after 30` có (nhưng do BUG-A, không phải do provider chết); không thử provider chậm / quá tải / `504` |
| 22 | PASS | lớp mới + sinh viên không dữ liệu → `409 NOT_ENOUGH_DATA`, `llm_audit` `INSIGHT` không thêm dòng (không gọi LLM); lần 11 trong 1 giờ → `429` (10 lần đầu — kể cả lần lỗi — tính vào hạn mức); STUDENT, ADMIN → `403` |
| 23 | PASS | `POST …/outreach` + `Idempotency-Key` → `201`; `escalation_tickets` `kind=TEACHER_OUTREACH`, `status=ANSWERED`, `claimed_by` và `answered_by` có; tin trong phiên "Tin nhắn từ giảng viên" của C; thông báo `TICKET_ANSWERED` + thư Mailpit "Giảng viên vừa nhắn cho bạn" |
| 24 | PASS | UI thật `sv.nguyco`: tin nhãn "Giảng viên · TS. Lê Thu Hà" trong phiên "Tin nhắn từ giảng viên"; mỗi tin có `Hỏi tiếp` / `Đã rõ`; không tràn ở 375 px |
| 25 | PASS | thân nêu tên sinh viên khác → `422 PII_IN_ANSWER` |
| 26 | PASS | sinh viên AT_RISK (`sv04`) nhận nhắn: 0 chữ "rủi ro" / "cần chú ý" / `AT_RISK` ở tin chat của anh ấy, `notifications`, thư Mailpit, JSON `me/today`, `notifications`, `me/courses`, `me/settings` (hai trùng khớp còn lại là chính câu hỏi do QC gõ và tên phiên chat lấy từ câu đó) |
| 27 | PASS | trần ngày: tin thứ 4 → `429`; PENDING / REMOVED → `409 STUDENT_NOT_ACTIVE`; thiếu `Idempotency-Key` → `422`; cùng key hai lần → `201`, 1 tin; STUDENT / ADMIN `403` |
| 28 | PASS | chat "Giảng viên ghi gì về em?" / "Em có bị coi là cần chú ý không?" → "Mình chưa tìm thấy nội dung này trong tài liệu của lớp."; không `[[…]]`, không nhãn / ghi chú |
| 29 | PASS | `go test ./internal/agent …` (nằm trong 1.010 test xanh): `TestStudentAsksAboutOwnNotes`, `TestNoToolReadsObservations` |
| 31 | PASS | `page.route` 500 → "Không tải được hồ sơ sinh viên. Dữ liệu và ghi chú đang gõ không bị mất." + `Thử lại`; `audit_log` không thêm dòng |
| 32 | PASS | `grep -rn "@/mock" frontend/src/features/students` = 0; `pnpm build:gate` xanh; `pnpm lint` sạch; Hôm nay staff không còn khối giả |
| 33 | PASS | `ui-antipatterns.sh` rc=0; Playwright desktop `students/panels/tokens/a11y/class-join/today/activity`: 163 passed, 7 skipped, 0 fail, 0 flaky |
| E1 | KHÔNG KIỂM ĐƯỢC → FAIL | "JSON sai schema" cần `ai-summary` tới được provider (BUG-A) |
| P1 | PASS | `sv04` (và `sv.nguyco`) đọc `…/observations` của chính mình → `403` |

## 3. Lệnh của handoff
`go test -race -tags testroutes ./internal/crm/... ./internal/escalation/... ./internal/agent/... ./internal/integration/... ./internal/contract/... ./internal/httpapi/...` (trong gói rộng 1.010 test / 28 gói) xanh; `sqlc diff` rc=0; `build:gate` + e2e + `grep @/mock` + `ui-antipatterns.sh` như trên. `visual.spec.ts` không chạy (handoff ghi đỏ trên macOS dev; chờ CI Linux).

## 4. Ghi nhận
- Colima hết đĩa lúc 15:35 (`No space left on device` khi đăng nhập `500`); đã `docker builder prune` + `image prune` (không đụng volume). Dữ liệu QC trước thời điểm đó không bị ảnh hưởng (kiểm lại các ghi `audit_log`).
- Bản mô phỏng còn ở khung sinh viên ("Bản mô phỏng · dữ liệu giả", US-P5-06) — ngoài phạm vi story này.
