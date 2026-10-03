# QC test case — US-P2-08 (Admin mở lớp, gán giảng viên / TA, lưu trữ, thông báo phân công, chuông thật, `/admin/courses`)
Nguồn: `docs/specs/FEAT-course-foundation/US.md` US-P2-08 AC1–AC13 + `SRS.md` 4.2 (sinh mã), 6.x (admin courses, assign, notifications), 7.5 (mock `/admin/courses`), PRD M0 ("thông báo phân công ≤ 60 s"; "STUDENT gọi API mở lớp nhận 403"). **Trọng tâm tấn công:** sinh viên / TA / GV gọi API Admin, GV cũ còn quyền sau khi bị thay, đọc thông báo người khác (IDOR), mã tham gia đoán được / trùng, Admin thêm sinh viên vào lớp, ghi vào lớp đã lưu trữ.

Tiền điều kiện chung: stack test + seed (US-P2-12); `$A`, `$T`, `$TA_`, `$SVA/B/D`, `C1`, `C2` như `US.md`; hai giảng viên mẫu (`teacher@`, `teacher2@` — nếu seed không có, QC mời qua US-P2-06). Công cụ: **S** `scripts/p208.sh`, **D**, **A**, **G**. Đồng hồ thật (đo ≤ 60 s thật). Thiếu route → FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P208-01 | AC1 | Admin | **S** `K=$(idem); j -X POST $GW/api/v1/admin/courses -H "$A" -H "Idempotency-Key: $K" -d '{"subject_code":"INT1006","class_code":"999001","name":"An ninh mạng","semester":"2026-2027-HK1"}'` hai lần cùng `$K`; `select count(*) from courses where class_code='999001'` | `201` rồi cùng thân + `Idempotent-Replayed: true`; **1** lớp; lớp `ACTIVE`, `join_code` 7 ký tự thuộc `ABCDEFGHJKMNPQRSTUVWXYZ23456789`, `join_enabled=true`, `join_require_approval=false` |
| TC-P208-02 | AC1 | – | **S** `class_code` trùng (khác hoa-thường); `class_code` `ab`, `a b`, 21 ký tự, `<x>`; `semester` `2026`, `2026-2027-HK4`; `name` rỗng / 121; `capacity` 0 / 1001; thiếu `Idempotency-Key`; thân có `join_code` ở `APP_ENV=production` | Trùng → `409 CONFLICT` `details.field="class_code"`; còn lại `422` (liệt kê ô); thiếu key → `422 IDEMPOTENCY_KEY_REQUIRED`; production + `join_code` trong thân → `422` |
| TC-P208-03 | AC1 (đua) | – | **S** 10 yêu cầu song song **cùng `class_code`**, **khác** `Idempotency-Key` | **1** thắng (`201`), 9 `409`; không lớp trùng (`count`=1) |
| TC-P208-04 | AC1 | – | **S** `teacher_id` hợp lệ trong thân | Gán trong **cùng transaction** (1 lớp + enrollment GV `ACTIVE` `joined_via=ADMIN`); sai vai → `422` và **không** tạo lớp (rollback: `count`=0) |
| TC-P208-05 | AC1 | – | **G** `-run 'TestCreateCourse\|TestCreateCourseIdempotent\|TestCreateCourseDuplicateClassCode\|TestCreateCourseValidation'` | `ok` |
| TC-P208-06 | AC2 (mã tham gia) | – | **S** tạo 2.000 lớp thử qua API / hoặc gọi hàm sinh qua test; thống kê ký tự của các `join_code`; `grep -rn '"math/rand"' backend-go/internal/course \| wc -l` | Chỉ 31 ký tự (không `0 O 1 I L`); mỗi ký tự ≈ 3,2 % ± 0,25 điểm phần trăm (trên 100.000 mã: QC chạy hàm sinh qua test Go tạm / dùng test của dev + tự tính chi-bình-phương); `grep` `0` (không `math/rand`) |
| TC-P208-07 | AC2 | – | **S** ép trùng mã (INSERT một lớp có mã X rồi tạo lớp khi RNG trả X — qua test); 5 lần trùng liên tiếp | Thử lại ≤ 5 lần rồi `500` rõ ràng (không vòng vô hạn, không mã trùng); không mã trùng trong DB (`UNIQUE`) |
| TC-P208-08 | AC2 | – | **G** `-run 'TestJoinCodeAlphabet\|TestJoinCodeDistribution\|TestJoinCodeCollisionRetry\|TestJoinCodeFixedOnlyOutsideProduction'` | `ok` |
| TC-P208-09 | AC3 (sửa, lưu trữ) | lớp thử `L9` | **S** `PUT /admin/courses/{id}` đúng / sai `version`; `POST …/archive`; `join` bằng mã `L9` bởi `$SVD`; lưu trữ lần hai | Sửa đúng `version` → `200`; sai → `409 VERSION_CONFLICT`; lưu trữ → `status=ARCHIVED`, `archived_at`, `join_enabled=false`; `join` → **`404 JOIN_CODE_INVALID`** (đồng nhất); thành viên **giữ nguyên**, chỉ còn đọc; lần hai `200` không đổi; `audit_log` mỗi thao tác |
| TC-P208-10 | AC3 (ghi vào lớp đã lưu trữ) | `L9` ARCHIVED | **S** thử mọi thao tác ghi: sửa, `assign`, `join-code/regenerate`, `join-settings`, `approve`, `DELETE member`, import roster, share document | Mỗi thao tác → `409 COURSE_ARCHIVED`; DB **không đổi** |
| TC-P208-11 | AC3 | – | **G** `-run 'TestUpdateCourse\|TestUpdateVersionConflict\|TestArchiveCourse\|TestArchivedRejectsWrites\|TestArchiveIdempotent'` | `ok` |
| TC-P208-12 | AC4 (gán — sai vai) | – | **S** `POST /admin/courses/{id}/assign` với `teacher_id` là SV / TA / ADMIN, DISABLED, không tồn tại, uuid rác; `ta_ids` chứa TEACHER | `422` mỗi ca; DB **không đổi** (kể cả enrollment, `outbox`) |
| TC-P208-13 | AC4 (**GV cũ mất quyền ngay**) | `C1` có GV1; GV1 đang đăng nhập (access còn hạn) | **S** gán GV2 cho `C1`; ngay sau, GV1 `GET /courses/$C1/members` bằng **access cũ** (không đăng nhập lại); GV2 gọi | GV1 → **`403` ngay** (guard tra `enrollments`, không dùng vai JWT); GV1 `status=REMOVED`; GV2 `ACTIVE` `joined_via=ADMIN`; đúng **1** GV `ACTIVE` mỗi lớp |
| TC-P208-14 | AC4 (TA thay toàn bộ) | `C1` có TA1, TA2 | **S** `assign {ta_ids:[TA3]}`; `assign {}` (bỏ trống); `assign {ta_ids:[]}` | `[TA3]` thay toàn bộ (TA1, TA2 `REMOVED`); bỏ trống → **giữ nguyên**; `ta_ids:[]` theo SRS (QC ghi hành vi: rỗng = gỡ hết hay giữ) |
| TC-P208-15 | AC4 (nguyên tử) | – | **D** gây lỗi giữa chừng (ép `outbox` lỗi / test) rồi đếm `enrollments`/`outbox`; `outbox` `course.assigned` mỗi người **mới** | Mọi thay đổi + `outbox` trong **một** transaction (rollback không để lại gì); chỉ người **mới** được gán có `outbox`; gán lại người đã có → không thông báo lại |
| TC-P208-16 | AC4 | – | **G** `-run 'TestAssignTeacher\|TestAssignRejectsWrongRole\|TestAssignReplacesTAs\|TestAssignOldTeacherLosesAccessNow\|TestAssignOneActiveTeacher\|TestAssignAtomicWithOutbox\|TestAssignNoRenotify\|TestAssignAudit'` | `ok` |
| TC-P208-17 | AC5 (**≤ 60 s**) | worker chạy | **S** gán GV2 cho lớp; đo thời gian tới khi `notifications` có dòng (`select count(*) … where type='COURSE_ASSIGNED'`); `curl -H "$T2" /notifications` | **≤ 60 s** (thực tế ≤ 2 s; ghi số); **đúng 1** dòng; tiêu đề "Bạn được phân công lớp {tên} – {mã lớp}"; thân "Mã tham gia: {mã}. Chia sẻ mã hoặc đường dẫn {APP_PUBLIC_URL}/join/{mã} cho sinh viên."; `link=/class/settings?course={id}` |
| TC-P208-18 | AC5 (TA) | – | **S** gán TA | "Bạn được phân công làm trợ giảng lớp {tên} – {mã lớp}"; thân "Giảng viên phụ trách: {tên GV}."; `link=/class/members?course={id}`; **TA không nhận mã tham gia** |
| TC-P208-19 | AC5 (giao lại) | – | **D** giao lại cùng tin `course.assigned` (`XADD` lại; `kill -9` worker rồi `XAUTOCLAIM`) | **Không** tạo dòng thứ hai (`dedupe_key`); `count(*)`=1 |
| TC-P208-20 | AC5 | – | **G** `go test ./internal/integration -run 'TestAssignTeacherNotifies\|TestAssignNotifyOnce'` | `ok` |
| TC-P208-21 | AC6 | GV1 của `C1`, GV2 của `C2` | **S** GV1 `PUT /courses/$C1/assistants {ta_ids:[…]}`; GV2 `PUT /courses/$C1/assistants`; `GET …/assistant-candidates?q=` | GV1 thay tập TA được; **GV2 (lớp khác) → `403`**; TA / SV → `403`; `assistant-candidates` ≤ 20 người vai TA `ACTIVE`/`INVITED`, tìm theo tên không dấu hoặc tiền tố email, chỉ `{id, full_name, email}` (**không** SV, GV, ADMIN) |
| TC-P208-22 | AC6 | – | **G** `-run 'TestTeacherManagesOwnTAs\|TestOtherTeacherCannotManageTAs\|TestAssistantCandidates\|TestChangeTeacherMidTerm'` | `ok` |
| TC-P208-23 | AC7 (**IDOR thông báo**) | thông báo của GV `ID_GV` | **S** `$SVB` `POST /notifications/$ID_GV/read`; `GET /notifications`; thử `ID` lạ / rác | **`404`** cho thông báo người khác (cùng mã và thân với `ID` lạ); `GET` chỉ thấy **của mình**; `ID_GV` vẫn chưa đọc (`read_at IS NULL`) |
| TC-P208-24 | AC7 | – | **S** `GET /notifications?limit=1&cursor=…` đi hết; `limit=101`; `unread_only=true`; `POST …/read` hai lần; `link` | Cursor `(created_at DESC, id DESC)`; mặc định 30 / tối đa 100; `unread_count` = số `read_at IS NULL`; `read` idempotent; `link` luôn **đường dẫn nội bộ** (`/…`, không `https://`, không `//`); không `user_id` trong đường dẫn |
| TC-P208-25 | AC7 | – | **G** `-run 'TestNotificationsList\|TestNotificationsOnlyMine\|TestNotificationReadIdempotent\|TestNotificationOtherUser404\|TestNotificationsUnreadCount'` | `ok` |
| TC-P208-26 | AC8 (**ma trận quyền Admin**) | – | **S** 5 thao tác (`GET/POST /admin/courses`, `PUT /{id}`, `POST …/assign`, `POST …/archive`) × {ADMIN, TEACHER, TA, STUDENT, không JWT, hết hạn}; kể cả `GET` | **Chỉ ADMIN**; TEACHER / TA / STUDENT → `403 FORBIDDEN` `details.reason="role"` **kể cả `GET`**; không JWT `401`; hết hạn `401 TOKEN_EXPIRED`; không ca lệch (≥ 40) |
| TC-P208-27 | AC8 (tấn công) | – | **S** token giả (`alg=none`, sửa vai, secret sai); `X-Role: ADMIN`; vai DB ≠ JWT; GV của lớp tự `assign` lớp mình | `401`/`403`; không tạo / sửa lớp; GV (cả GV của lớp) **không** `assign` được |
| TC-P208-28 | AC8 | – | **G** `-run TestAdminCoursesRBACMatrix -v` | `ok`; ≥ 40 ca |
| TC-P208-29 | AC9 | ≥ 3 lớp | **S** `GET /admin/courses?status=&q=&semester=&cursor=&limit=`; `jq '.items[0]\|keys'`; `grep -ciE 'join_code\|students"\|email'` | Mục `{id,class_code,subject_code,name,semester,status,teacher:{id,full_name}\|null,assistants_count,students_active,students_pending,capacity,version}`; **không** `join_code`, không danh sách SV (Admin không đọc nội dung lớp); đếm đúng với SQL; cursor đi hết không trùng; 1 truy vấn |
| TC-P208-30 | AC9 | – | **G** `-run 'TestAdminCoursesList\|TestAdminCoursesNoJoinCodeNoStudents\|TestAdminCoursesNoNPlusOne\|TestAdminCoursesCursor'` | `ok` |
| TC-P208-31 | AC10 | Admin `/admin/courses` | **A** bảng, nút primary, Drawer `Mở lớp`; mở lớp; nhấp đúp; `Lưu trữ` → xác nhận | Cột Mã lớp, Học phần, Giảng viên, Sĩ số ("30 / 30"), Trạng thái; **một** nút `Mở lớp`; Drawer tại chỗ (không `role=dialog` khi mở form); xong: "Đã mở lớp. Đã gửi thông báo phân công cho {tên}." tại chỗ (không toast); nhấp đúp → **1** lớp; `Lưu trữ` mở `ConfirmIrreversible` nêu số ("Lưu trữ lớp 761987: 30 sinh viên chỉ còn quyền đọc; mã tham gia ngừng…") — `role=dialog` chỉ ở đây |
| TC-P208-32 | AC10 | 375 | **A** `AUDIT_SRC`, `TOUCH_SRC`, axe, bàn phím; `grep useDemoSlice` ở `/admin/courses` | Sạch; 0 `serious`; `< 720 px` thành danh sách; mã mock bị thay |
| TC-P208-33 | AC10 | – | **G** `$PW class-join.spec.ts -g 'admin courses page'` | `rc=0` |
| TC-P208-34 | AC11 (chuông thật) | GV2 vừa được gán | **A** đăng nhập GV2, đo thời gian chấm chưa đọc; mở khung; bấm mục; đếm `unread`; mở khung không xoá chấm | Chấm `bell-dot` ≤ **35 s** (refetch 30 s + focus); khung hiện "Bạn được phân công lớp …"; bấm → URL = `link`, đánh dấu đọc (lạc quan), `unread` giảm 1; mở khung **không** xoá chấm; hết chưa đọc → chấm biến; rỗng "Chưa có thông báo. Khi có việc…" |
| TC-P208-35 | AC11 (thời gian) | – | **A** thông báo tạo 59 giây, 5 phút, 3 giờ, hôm qua 16:40, 5 ngày trước (sửa `created_at` trong DB) | "vừa xong", "N phút trước", "N giờ trước", "hôm qua 16:40", "dd/MM" theo **ngày lịch** (đồng hồ thật, `Asia/Ho_Chi_Minh`); `grep -rnE '[0-9]+ (ngày\|giờ\|phút) trước' frontend/src \| grep -v shared/lib/timeAgo` → rỗng |
| TC-P208-36 | AC11 | – | **G** `$PW class-join.spec.ts -g 'bell real'` | `rc=0` |
| TC-P208-37 | AC12 | Admin | **A** ép: email trùng `class_code` (409), 422, người sai vai ("Người này không phải giảng viên."), sai `version` ("Lớp này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?"), offline, lưu trữ lớp đã lưu trữ ("Lớp này đã được lưu trữ.") | Lỗi tại ô, giữ chữ; offline → `OfflineBanner` + `Gửi lại` dùng **cùng** `Idempotency-Key` (so header) |
| TC-P208-38 | AC12 | – | **G** `$PW class-join.spec.ts -g 'admin courses errors'` | `rc=0` |
| TC-P208-39 | AC13 (F2 phía Admin) | DB trống (không seed) | **S** Admin mở lớp + gán GV (qua API / UI); đăng nhập GV; chuông; "Quản lý lớp này" | GV thấy chuông + mã ≤ 60 s; `/class/settings` hiện mã đúng với `courses.join_code` |
| TC-P208-40 | AC13 (**Admin không thêm SV**) | – | **S** `grep -rnE "POST.*(admin/courses/.*(students\|members\|enroll))" backend-go/openapi.yaml \| wc -l`; thử `POST /admin/courses/{id}/members`, `…/enroll`, `…/students`, `assign {student_ids:[…]}`; UI tìm nút "Thêm sinh viên" | `0`; mọi đường thử `404`/`422`; không `enrollments` STUDENT mới; UI không có nút |
| TC-P208-41 | AC13 | – | **G** `go test ./internal/integration -run TestOpenCourseAssignThenTeacherSeesCode -v` | `ok` |
| TC-P208-42 | tổng | – | **S** `go vet && golangci-lint run && go test -race -count=1 ./internal/course/... && go test -tags integration ./internal/integration/...` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Sinh viên / TA / GV gọi API Admin; token giả | 26, 27 |
| GV cũ còn quyền sau khi bị thay | 13 |
| GV lớp khác sửa TA lớp này | 21 |
| IDOR đọc / đánh dấu thông báo người khác | 23 |
| Mã tham gia trùng / đoán được / `math/rand` | 06–08 |
| Ghi vào lớp đã lưu trữ | 09, 10 |
| Gán dở dang (mất nguyên tử), thông báo gửi đôi | 15, 19 |
| Admin thêm sinh viên vào lớp | 40 |
| Hai yêu cầu tạo lớp cùng mã | 03 |
| `link` thông báo mở chuyển hướng ra ngoài | 24 |

## Câu hỏi cho BA / PM
- **Q-QC-P208-1** — TC-P208-14: `assign {ta_ids:[]}` (mảng rỗng) có nghĩa gỡ hết TA hay giữ nguyên? US: "thay thế toàn bộ khi có `ta_ids` (bỏ trống = giữ nguyên)". QC hiểu `[]` = gỡ hết, vắng = giữ. — *chờ xác nhận*.
- **Q-QC-P208-2** — TC-P208-06: phân bố mã đo trên 100.000 mã cần hàm sinh gọi trực tiếp; QC dùng test Go tạm (`qc_probe_test.go`) hoặc test của dev; chấp nhận chỉ chạy test dev nếu không có đường HTTP sinh hàng loạt. — *chờ xác nhận*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1 (FEAT-course-foundation, APPROVED 2026-10-03).

Tổng: 42 TC.
