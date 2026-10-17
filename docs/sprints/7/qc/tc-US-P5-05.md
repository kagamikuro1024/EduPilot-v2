# QC test case — US-P5-05 (hồ sơ 360, ghi chú riêng tư, nhận xét AI, `Nhắn sinh viên`)
Nguồn: `US.md` US-P5-05 AC1…AC10 + `SRS.md` 2, 4.8, 6.4, 7.4; FLOWS F8; PRD M6; `DESIGN.md` §14.7; `FEAT-escalation-notify` SRS 4.2.7 (`CreateOutreach`). Hộp đen, không đọc code dev.
Biến như `tc-US-P5-01.md`. `uidC` = id của C (`sv.nguyco`), `uidA`, `uidB`. Hồ sơ: `$Q/students/$uidC/profile`. Ghi chú: `$Q/students/$uidC/observations`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P5-05-01 | AC1 (nội dung) | seed + `crm-risk-run` | `j "$Q/students/$uidC/profile" -H "$TCH" -i` | `200`; `{student:{id,full_name,student_code,joined_at}, risk:{level,sentence,reasons[]}, attendance:{recorded,present,late,excused,absent}, participation:{speak_count,points_total}, study:{minutes_7d,weeks[],tracked}, activity:{asks_last_week,posts_last_week}}`; `Cache-Control: no-store`; không `ETag` |
| TC-P5-05-02 | AC1 (audit mỗi lần gọi) | `select count(*) from audit_log where entity='student_profile' and entity_id='$uidC'` = n | gọi `profile` 3 lần bằng `curl` | n+3: mỗi lần gọi 1 dòng (`action='view'`, `actor_id`, `course_id`); nội dung ghi chú không có ở audit |
| TC-P5-05-03 | AC1 (UI: mở 3 lần = 3 request, 3 dòng audit) | `playwright-cli`, teacher | mở `/students/$uidC` → quay lại → mở lại → quay lại → mở lại; đếm request `GET …/profile` trong `network`; so dòng audit | đúng 3 request (không cache: `staleTime 0`, `refetchOnMount always`, `gcTime 0`) và 3 dòng `audit_log` |
| TC-P5-05-04 | AC1 (fail-closed) | cài trigger tạm lỗi ở `audit_log` insert (DB test) | `profile` | `503`, **không** trả dữ liệu; gỡ trigger |
| TC-P5-05-05 | AC1 (đọc ghi chú) | có ghi chú | `GET …/observations` | `action='view_observations'` thêm 1 dòng audit; `audit_log` không chứa nội dung ghi chú (grep canary `CANARY-OBS-4F` = 0) |
| TC-P5-05-06 | AC1 (không phải lớp) | sinh viên `REMOVED`/`PENDING`/lớp khác | `profile` | `404` |
| TC-P5-05-07 | AC1 (không đọc chat) | – | `log_statement='all'`, gọi `profile`; grep `chat_messages` | 0 |
| TC-P5-05-08 | AC2 (phân quyền) | – | ma trận: `profile`, `observations`, `ai-summary`, `outreach`, `attendance`, `study-time` × `$SVC` (uid của mình), `$SVA`, `$ADM`, TA lớp 1, TEACHER lớp 1, TA/TEACHER lớp khác (lớp 3), sinh viên ở lớp khác | STUDENT (kể cả `uid` mình)/ADMIN → `403`; TA, TEACHER lớp 1 ✓; staff lớp khác → `403`; sinh viên lớp khác → `404` |
| TC-P5-05-09 | AC2 (ghi chú: tác giả) | teacher viết ghi chú `$OB`; TA | TA `PUT`/`DELETE` `$OB`; TA `GET` danh sách; teacher `PUT` ghi chú của TA | TA sửa/xoá của teacher → `403` (và ngược lại); mọi staff của lớp **đọc** được mọi ghi chú |
| TC-P5-05-10 | AC3 (CRUD ghi chú) | teacher | `POST …/observations {"tags":["lo-lang"],"body":"CANARY-OBS-4F …","kind":"NOTE"}`; `GET` (mới nhất trước, cursor); `PUT {tags,body,version}`; `DELETE` | `201`; danh sách mới nhất trước, phân trang con trỏ; `PUT` `200` `version`+1; `DELETE` xoá thật + `audit_log` không nội dung |
| TC-P5-05-11 | AC3 (validation) | – | `tags` 6 phần tử; tag 0 hoặc 31 ký tự; `body` rỗng; `body` 2.001 ký tự; sai `version` | `422` ×4; `409 VERSION_CONFLICT` |
| TC-P5-05-12 | AC3 (UI) | `playwright-cli`, tab `Ghi chú` | quan sát; gõ dở; offline; online | `PrivateMark` "Chỉ giảng viên/TA thấy"; ô nhập tự lưu nháp (`obs:<uid>`); không mất chữ khi mạng lỗi; nút `Lưu` hoàn lại + `Thử lại` |
| TC-P5-05-13 | AC3/AC8 (không vào lời nhắc) | ghi chú canary | chat của mọi sinh viên, thread AI, `ai-summary`; xem payload LLM (provider giả) | `CANARY-OBS-4F` **không** xuất hiện ở bất kỳ thân yêu cầu LLM nào của chat/threads; ngoài `ai-summary` không lời gọi LLM nào chứa ghi chú |
| TC-P5-05-14 | AC4 (UI hồ sơ) | `playwright-cli` 1280, `/students/$uidC` | quan sát đầu trang và tab | đầu trang: tên, MSSV, lớp, `StudentRiskSummary` **chỉ khi** `WATCH`/`AT_RISK`, hành động phụ `Thêm ghi chú`, `Nhắn sinh viên` (không nút chính thứ ba); `Tabs`: Tổng quan (đoạn tóm tắt bằng lời + danh sách `term/value`; **không** 6 thẻ KPI), Chuyên cần (bảng buổi × trạng thái), Điểm ("Điểm cộng đã ghi" = tổng điểm sự kiện tham gia; rỗng "Lớp chưa có công thức điểm đã xác nhận." + `Xác nhận công thức` cho TEACHER), Hoạt động học (biểu đồ nhỏ xu hướng phút học theo tuần + số câu hỏi/bài đăng mỗi tuần — chỉ ở đây có biểu đồ), Ghi chú |
| TC-P5-05-15 | AC4 (trạng thái, nhãn) | – | tải chậm; lỗi; `resize 720 900`; `eval innerText` | `<PageState>` đủ; dùng được ≥ 720 px; chữ "Tuần trước" ở khối hoạt động; không câu mô tả dưới tiêu đề khối |
| TC-P5-05-16 | AC5 (AI tóm tắt, một lần gọi, không danh tính) | teacher; provider giả ghi thân yêu cầu | `POST …/students/$uidC/ai-summary`; `GET $GW/api/v1/_test/llm/stats` trước/sau; xem thân yêu cầu | đúng **1** lời gọi `llm.Structured` (task `INSIGHT`, làn `NEAR_REALTIME`, schema `{summary,strengths[0–3],concerns[0–3],suggestions[0–3]}`); thân yêu cầu chỉ số liệu (đếm chuyên cần, phát biểu, phút học 4 tuần, câu hỏi/bài đăng mỗi tuần, mã lý do); **không** tên, MSSV, email, nội dung ghi chú, nội dung chat (grep tên/MSSV/email của C = 0); `masked_count = 0` |
| TC-P5-05-17 | AC5 (không lưu tới khi `Lưu`) | – | gọi `ai-summary`; `select count(*) from student_observations` trước/sau | `200 {draft}` không lưu gì; UI hiện bản nháp nhãn `AI` trong ô sửa được |
| TC-P5-05-18 | AC5 (lưu, đã sửa) | – | lưu bản gốc → `kind='AI_SUMMARY'`, `ai_edited=false`; sửa chữ rồi lưu → `ai_edited=true` | nhãn "Nhận xét tổng hợp · AI" / "· Đã sửa bởi giảng viên" |
| TC-P5-05-19 | AC5 (mô hình bịa tên) | provider giả trả bản nháp chứa tên một sinh viên trong roster | `ai-summary` | tên bị `Redact` (`[đã ẩn]`) trong phản hồi |
| TC-P5-05-20 | AC5 (nháp AI ở máy) | – | nhận nháp, tải lại trang | nháp AI còn (`aisum:<uid>`) |
| TC-P5-05-21 | AC6 (lỗi AI) | provider giả chết / quá tải / chậm | `ai-summary` | `503 LLM_UNAVAILABLE` hoặc `OVERLOADED` + `retry_after`; UI "Chưa soạn được nhận xét. Thử lại" + `Thử lại`; chữ ghi chú đang gõ không mất; quá 20 s → `504` |
| TC-P5-05-22 | AC6 (giới hạn, không dữ liệu, vai) | – | gọi 11 lần trong 1 giờ; sinh viên lớp mới không dữ liệu (`$CN`); `$SVA`, `$ADM` | lần 11 `429 RATE_LIMITED`; `409 NOT_ENOUGH_DATA` và **không** gọi LLM (`/_test/llm/stats` không tăng); `403` ×2 |
| TC-P5-05-23 | AC7 (`Nhắn sinh viên`) | teacher; `uidC` | `POST …/students/$uidC/outreach {"body":"Chào em, thầy muốn hỏi thăm …"} -H "$idem"` | `201 {ticket}`; một giao dịch: ticket `kind=TEACHER_OUTREACH` `ANSWERED` (người gửi là `claimed_by`/`answered_by`), tin `STAFF_ANSWER` trong phiên "Tin nhắn từ giảng viên" của C, outbox `escalation.answered` → chuông + thư `outreach_message` (theo cờ của C) |
| TC-P5-05-24 | AC7 (sinh viên đọc, hỏi tiếp) | – | `$SVC` mở `/chat`; `Hỏi tiếp` | thấy tin nhãn "Giảng viên · …"; `Hỏi tiếp` → ticket `OPEN` ở `/inbox` của giảng viên |
| TC-P5-05-25 | AC7 (PII khác) | – | body chứa tên/MSSV/email của sinh viên **khác** | `422 PII_IN_ANSWER` |
| TC-P5-05-26 | AC7 (không lộ nhãn rủi ro) | C `AT_RISK` | grep tiếng "rủi ro", "cần chú ý", nhãn, lý do at-risk trong: tin chat của C, `notifications.title/body`, thư Mailpit, JSON mọi route sinh viên | 0 chỗ |
| TC-P5-05-27 | AC7 (trần/ngày, trạng thái, idempotent) | – | 4 `outreach` trong 1 ngày tới C (nhiều nhân viên cộng lại); sinh viên `REMOVED`/`PENDING`; cùng `Idempotency-Key` hai lần | tin thứ 4 `429 OUTREACH_LIMIT`; `409 STUDENT_NOT_ACTIVE`; lần hai cùng key → không thêm tin |
| TC-P5-05-28 | AC8 (hỏi AI về ghi chú) | C có ghi chú canary và `AT_RISK` | `$SVC` hỏi "Giảng viên ghi gì về em?", "Em có bị coi là cần chú ý không?" | câu mẫu / trả lời học thuật thông thường (không tiết lộ), không `[[…]]`; ghi chú và nhãn không nằm trong lời nhắc, ngữ cảnh RAG hay kết quả tool |
| TC-P5-05-29 | AC8 (kiến trúc) | repo | `cd backend-go && go test ./internal/agent -run 'TestStudentAsksAboutOwnNotes\|TestNoToolReadsObservations' -v` | `ok` |
| TC-P5-05-30 | AC9 (mất mạng / đồng thời) | `playwright-cli` | gõ ghi chú rồi offline; hai giảng viên cùng sửa một ghi chú (hai ngữ cảnh) | nháp giữ; `Lưu` hoàn lại + `Thử lại`; sửa đồng thời → `409` và hiện bản của người kia để chọn |
| TC-P5-05-31 | AC9 (mở hồ sơ khi mạng lỗi) | `page.route` 500 / offline | mở `/students/$uidC`; kiểm `audit_log` | `PageState` lỗi (nói dữ liệu không mất, có `Thử lại`); **không** dòng audit mới khi dữ liệu chưa trả về |
| TC-P5-05-32 | AC10 (mock) | repo | `grep -rn "@/mock" frontend/src/features/students \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint`; mở `/` (Hôm nay staff) | `0`; xanh; Hôm nay staff không còn "Lớp cần chú ý" giả |
| TC-P5-05-33 | UI | – | `bash scripts/ui-antipatterns.sh; echo $?`; `$PW students.spec.ts` (toàn bộ) | `0`; xanh; ghi số test, flaky |

## Nhánh lỗi
TC-P5-05-04, 06, 11, 19, 21, 22, 25, 27, 30, 31. Thêm TC-P5-05-E1: `ai-summary` trả JSON sai schema → lỗi rõ (kỳ vọng `5xx` có mã, không lưu).

## Phân quyền
TC-P5-05-08, 09, 22, 26, 28. Thêm TC-P5-05-P1: sinh viên đọc `observations` của **chính mình** → `403` (cổng P5 "STUDENT đọc observation → 403").

## Câu hỏi (Q-QC)
- **Q-QC-P5-10** AC1 "mở 3 lần = 3 dòng audit" trên UI cần đếm `network`; Playwright của QC chạy trên stack seed. OK.
- **Q-QC-P5-11** AC5: kiểm "đúng một lần gọi LLM" và nội dung thân yêu cầu cần log provider giả. Nếu `/_test/llm/stats` không trả thân yêu cầu, nhờ dev nêu cách đọc (đã dùng ở sprint 6: `fake_calls`). Chưa rõ nơi xem thân; đề xuất `LLM_AUDIT` hoặc log DEBUG của provider giả.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.
