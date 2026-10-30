# QC test case — US-P6-05 (nhắc nhở khi chưa có công thức: status, banner, khoá `Chốt điểm`, "Hôm nay", chuông tuần, AI nói thật)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-P6-05 (AC1–AC9) + `SRS.md` 2, 3, 4.5, 6; `docs/phases/P6.md` ("Bạn tự kiểm" mục 3); `docs/DEMO_SCRIPT.md` bước 6; `FLOWS.md` F10 nhánh 1, F14, F3. Viết trước khi có code, không đọc code của dev (v1, 2026-10-31).

**Quy ước.** Lớp 1 seed = `CONFIRMED`; lớp 2 seed = chưa có `COURSE_POLICY` (`NO_POLICY`). Các ca đổi trạng thái lớp 2 làm sau khi hết ca "lớp 2 nguyên bản"; dựng lại DB trước khi lặp. `GO` = `bash docs/sprints/8/qc/scripts/p6-go-tests.sh`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P6-05-01 | AC1 | seed nguyên bản | `j $Q2/grade-scheme/status -H "$TCH"`; `j $Q/grade-scheme/status -H "$TCH"` | lớp 2: `"state":"NO_POLICY"`, `has_policy:false`, `finalize_blocked_reason` có (chuỗi không rỗng), `gradebook:{status:"OPEN",snapshot_version:0}`; lớp 1: `"state":"CONFIRMED"`, `confirmed_at`, `scheme_no:1`, `pending_draft:false`, không `finalize_blocked_reason` |
| TC-P6-05-02 | AC1 | lớp 2 | Tải `quy-che-lop2.pdf` (chờ READY) → gọi `status` | `NOT_EXTRACTED` (có quy chế, chưa có nháp), `has_policy:true` |
| TC-P6-05-03 | AC1 | như trên | `extract` → xong → `status` | `DRAFT_UNCLEAR`, `unclear_count:1`, `pending_draft` (theo spec: true khi `CONFIRMED` + nháp; ở lớp 2 chưa CONFIRMED ghi nhận giá trị thật) |
| TC-P6-05-04 | AC1 | như trên | Điền "Làm tròn đến 0,1" qua `PUT …/grade-scheme` | `DRAFT_READY`, `unclear_count:0` |
| TC-P6-05-05 | AC1 | như trên | `confirm` | `CONFIRMED`, `finalize_blocked_reason` biến mất |
| TC-P6-05-06 | AC1 | lớp 1 `CONFIRMED`; tải quy chế mới → nháp mới | `status` lớp 1 | `state:"CONFIRMED"` **và** `pending_draft:true` |
| TC-P6-05-07 | AC1, AC9 | — | `for T in "$SVA" "$ADM"; do code -H "$T" $Q/grade-scheme/status; done`; `code -H "$TA_" …` | `403 403`; TA `200` |
| TC-P6-05-08 | AC1 | repo | `GO ./internal/grade "TestStatusStates|TestStatusPendingDraft|TestStatusRBAC" -tags integration` | PASS |
| TC-P6-05-09 | AC2 | seed nguyên bản; TEACHER | `playwright-cli`: mở `/gradebook?course=<C2>` | **một** `InlineNotice` mỏng cố định đầu trang: "Công thức điểm chưa xác nhận" + liên kết `Tải quy chế` (vì `NO_POLICY`); không câu giải thích dài; chụp; đếm: đúng 1 banner |
| TC-P6-05-10 | AC2 | như trên; TA | đăng nhập `ta@` mở trang lớp 2 | banner cùng chữ, liên kết `Xem công thức` (không `Xác nhận công thức`, không `Tải quy chế` cho TA) |
| TC-P6-05-11 | AC2 | lớp 2 `DRAFT_READY` | TEACHER mở `/gradebook` | liên kết `Xác nhận công thức` (theo spec "`Xác nhận công thức` cho TEACHER") |
| TC-P6-05-12 | AC2 | lớp 1 `CONFIRMED`; STUDENT | TEACHER mở lớp 1; `sv.gioi@` mở `/me` | lớp 1: **không** banner; STUDENT: **không** banner ở bất kỳ trang nào |
| TC-P6-05-13 | AC2 | sau xác nhận lớp 2 | Tải lại `/gradebook` lớp 2 | banner biến mất |
| TC-P6-05-14 | AC2 | repo | `playwright test gradebook.spec.ts -g 'banner unconfirmed'` | xanh (lớp 2 có banner, lớp 1 không) |
| TC-P6-05-15 | AC3 | lớp 2 `NO_POLICY`; TEACHER | Mở `/gradebook` lớp 2; tìm nút `Chốt điểm`; rê chuột, `Tab` tới nút | nút hiện nhưng `disabled` và `aria-disabled="true"`; tooltip **trên chính nút**: "Chưa có công thức điểm đã xác nhận" (hiện cả khi hover và khi focus); không icon Info rải rác |
| TC-P6-05-16 | AC3 | như trên | `curl -s -X POST $Q2/gradebook/finalize -H "$TCH" -H "$(idem)" -o /dev/null -w "%{http_code}"` (lệnh cổng P6) | `409`; thân `code:"SCHEME_NOT_CONFIRMED"` |
| TC-P6-05-17 | AC3 | như trên | Gọi bỏ `Idempotency-Key`; gọi với `$TA_` / `$SVA` | thiếu khoá: `400/422` (không 200); TA/STUDENT `403` (không bị 409 lộ trạng thái) |
| TC-P6-05-18 | AC3 | repo | `playwright test gradebook.spec.ts -g 'finalize locked reason'` | xanh |
| TC-P6-05-19 | AC4 | lớp 2 `NO_POLICY`; TEACHER/TA | Mở `/attendance` lớp 2 | `InlineNotice` mỏng phía trên bảng: "Công thức điểm chưa xác nhận" + `Xem công thức`; điểm danh vẫn thao tác được (đổi trạng thái 1 sinh viên: lưu OK) |
| TC-P6-05-20 | AC4 | lớp 2 | **Bấm giờ** điểm danh 30 sinh viên ≤ 60 s khi banner hiện (bàn phím 1–4) | ≤ 60 s; banner không làm chậm/che lưới; 0 hộp thoại |
| TC-P6-05-21 | AC4 | lớp 1 `CONFIRMED` | Mở `/attendance` lớp 1 | **không** banner |
| TC-P6-05-22 | AC4 | repo | `playwright test attendance.spec.ts gradebook.spec.ts -g 'attendance notice'` | xanh |
| TC-P6-05-23 | AC5 | lớp 2 `ACTIVE` `NO_POLICY` | `j "$GW/api/v1/me/today" -H "$TCH" \| jq '.items[] \| select(.type=="GRADE_SCHEME_UNCONFIRMED")'` (đường dẫn thật theo openapi) | có mục tiêu đề "Công thức điểm chưa xác nhận · 761988", lý do "Lớp chưa có quy chế môn học.", `href` `/gradebook/scheme?course=<C2>`; có **cùng** "Thiết lập lớp mới" của lớp 2 |
| TC-P6-05-24 | AC5 | lần lượt các trạng thái lớp 2 (TC-02..04) | đọc lý do mục "Hôm nay" | `NOT_EXTRACTED` "Đã có quy chế, chưa trích công thức."; `DRAFT_UNCLEAR` "Bản nháp còn 1 mục chưa rõ."; `DRAFT_READY` "Bản nháp sẵn sàng, chờ xác nhận." |
| TC-P6-05-25 | AC5 | TA | `j "$GW/api/v1/me/today" -H "$TA_"` | có mục (TA thấy) |
| TC-P6-05-26 | AC5 | STUDENT, ADMIN | `j … -H "$SVA"`, `-H "$ADM"` | **không** có mục `GRADE_SCHEME_UNCONFIRMED` |
| TC-P6-05-27 | AC5 | lớp 2 sau xác nhận | đo từ lúc `confirm` trả 200 đến lúc mục biến mất | ≤ **5 s** (đo 5 lần, lấy max) |
| TC-P6-05-28 | AC5 | lớp 2 lưu trữ (nếu tạo được: `PATCH` lớp thành `ARCHIVED` bằng API admin/teacher; nếu seed không có cách → "KHÔNG KIỂM ĐƯỢC" kèm lý do + `GO`) | `today` | không có mục cho lớp lưu trữ |
| TC-P6-05-29 | AC5 | repo | `GO ./internal/today "TestProviderGradeSchemeUnconfirmed|TestProviderClearedOnConfirm" -tags integration` | PASS |
| TC-P6-05-30 | AC6 | stack; đồng hồ giả (QC dùng cách của các job tuần khác trong repo: biến `NOW_OVERRIDE`/`GRADE_REMIND_AT` hoặc Go test) | Chạy job `grade.scheme_reminder` vào thứ Hai 08:05 `Asia/Ho_Chi_Minh` | thông báo `GRADE_SCHEME_REMINDER` tạo cho **TEACHER** lớp 2 (đếm `$PSQL "select user_id, type from notifications where type='GRADE_SCHEME_REMINDER'"`), không cho lớp 1 (`CONFIRMED`) |
| TC-P6-05-31 | AC6 | như trên | Chạy lại job cùng tuần | **không thêm** thông báo (`SET ep:grade.remind:{yyyyww}:{course} NX`; khử trùng `grade.scheme_reminder:{course}:{yyyyww}:{user}`) — `count` không đổi |
| TC-P6-05-32 | AC6 | như trên | Chuyển sang tuần kế (đồng hồ giả) khi lớp 2 vẫn chưa xác nhận | gửi **thêm đúng 1** chuông của tuần mới |
| TC-P6-05-33 | AC6 | lớp 2 đã `CONFIRMED` | Chạy job | không gửi |
| TC-P6-05-34 | AC6 | worker tắt qua thứ Hai 08:00, bật lại thứ Tư | Khởi động worker | chạy bù trong tuần (1 chuông), không lặp ở lần chạy kế trong tuần |
| TC-P6-05-35 | AC6, AC9 | — | Đếm người nhận: `select distinct u.role from notifications n join users u on u.id=n.user_id where n.type='GRADE_SCHEME_REMINDER'` | chỉ `TEACHER`; TA / STUDENT / ADMIN không bao giờ nhận; giảng viên lớp khác không nhận |
| TC-P6-05-36 | AC6 | repo | `GO ./internal/grade "TestReminderWeeklyOncePerWeek|TestReminderStopsWhenConfirmed|TestReminderSkipsArchived|TestReminderCatchUp|TestReminderRecipientsTeacherOnly" -tags integration` | PASS |
| TC-P6-05-37 | AC7 | lớp 2 `NO_POLICY` (STUDENT lớp 2: dùng tài khoản đã vào lớp 2; nếu `sv.gioi` thuộc cả hai lớp, chọn lớp 2 ở chat) | Chat riêng: "Cô tính điểm môn này thế nào?" | câu mẫu "Lớp chưa có công thức điểm chính thức do giảng viên xác nhận." hiện ngay (không stream chữ LLM; đếm `llm_audit` tăng **0**); **không** số / trọng số bịa; có nút `Nhờ giảng viên` (F3); không dòng số liệu điểm |
| TC-P6-05-38 | AC7 | như trên | `$PSQL "select reason, count(*) from escalation_tickets where reason='AUTO_NO_FORMULA' …"` (tên bảng/cột theo `FEAT-escalation-notify`) | tự mở **một** ticket `AUTO_NO_FORMULA`; hỏi lại câu đó lần hai cùng hội thoại → không mở ticket thứ hai (khử trùng) — QC ghi nhận thực tế; `ResponseMetadata.NoFormula = true` (xem payload SSE `done`) |
| TC-P6-05-39 | AC7 | như trên | Hỏi: "Điểm của em bao nhiêu?" (PERSONAL_GRADE) và "Nếu cuối kỳ em được 8 thì sao?" (WHAT_IF_GRADE) | cùng câu mẫu; **không** tự mở ticket; không số điểm nào |
| TC-P6-05-40 | AC7 | lớp 1 `CONFIRMED` | cùng 3 câu hỏi (sẽ trả lời thật — xem tc-US-P6-09) | **không** câu mẫu "chưa có công thức"; không ticket `AUTO_NO_FORMULA` |
| TC-P6-05-41 | AC7 | repo | `GO ./internal/agent "TestGradeFormulaNoScheme|TestPersonalGradeNoScheme|TestWhatIfNoScheme"`; `GO ./internal/escalation "TestAutoNoFormulaTicket" -tags integration` | PASS |
| TC-P6-05-42 | AC8 | lớp 1 `CONFIRMED`; tài liệu nguồn `quy-che-lop1.pdf` | Xoá tài liệu (`DELETE $Q/documents/$POLICY1 -H "$TCH"`) — làm trên bản sao DB của QC | `scheme` vẫn `CONFIRMED` và `/gradebook` tính bình thường (HP của A, B, C không đổi); `grade_schemes.source_document_id` NULL, `source_title` còn tên; trích dẫn trong UI hiện "Tài liệu nguồn đã gỡ" thay vì lỗi; **banner không hiện**; `status.has_policy=false` |
| TC-P6-05-43 | AC8 | repo | `GO ./internal/grade "TestSourceDocumentDeletedKeepsConfirmed" -tags integration` | PASS |

## Nhánh lỗi (SRS mục 3)
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-05-44 | `status` ở lớp không thuộc tài khoản | giảng viên lớp khác gọi `$Q/grade-scheme/status` | `404` |
| TC-P6-05-45 | Redis mất khoá nhắc | xoá `ep:grade.remind:*` giữa hai lần chạy cùng tuần | khử trùng ở mức thông báo (`grade.scheme_reminder:{course}:{yyyyww}:{user}`) vẫn chặn gửi trùng — chuông không đôi |
| TC-P6-05-46 | Hai worker cùng chạy | khởi hai worker; chạy job đồng thời | đúng một chuông mỗi (lớp, tuần, giảng viên) — khoá leader `ep:lock:grade.remind` |

## Phân quyền
Đã phủ: TC-07 (`status` 403 STUDENT/ADMIN), TC-26 (Hôm nay), TC-35 (người nhận chuông), TC-10 (nút theo vai), TC-17 (finalize 403).

## Script chạy được
`scripts/p6-env.sh`, `p6-rbac.sh`, `p6-go-tests.sh`; UI/bấm giờ: `playwright-cli`.

## Câu hỏi cho BA / PM (không tự đoán)
- Q-05-a (TC-P6-05-38): hỏi lại câu "cách tính điểm" lần hai khi đã có ticket `AUTO_NO_FORMULA` mở — spec nói "tự mở ticket" nhưng không nói khử trùng. QC kiểm theo `FEAT-escalation-notify` 4.2.3 (nếu có khử trùng ở đó thì áp dụng); nếu cả hai im lặng, QC chỉ ghi nhận.
- Q-05-b (TC-P6-05-28): chưa biết cách tạo lớp `ARCHIVED` trong stack seed; nếu không có API → ca dựa vào Go test.

## Lịch sử sửa TC
(chưa có)
