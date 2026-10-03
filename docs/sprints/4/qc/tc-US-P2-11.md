# QC test case — US-P2-11 (trang "Hôm nay": `internal/today`, xếp hạng bằng luật cứng, lọc đúng người, cache 60 s, giao diện `/`)
Nguồn: `docs/specs/FEAT-course-foundation/US.md` US-P2-11 AC1–AC14 + `SRS.md` 4.7 (bảng bậc, nguồn việc P2), 7.6 (khác biệt mock ↔ thật), PRD M14 (≈ 3 giây; việc xong biến mất ≤ 60 s), FLOWS F14 (bảng nguồn việc), `design/DESIGN.md` §14.1. **Trọng tâm tấn công:** sinh viên thấy việc / dữ liệu của người khác, việc dành cho staff lọt sang sinh viên, ADMIN xem việc trong lớp, `/courses/{id}/today` của lớp không thuộc mình, cache trả dữ liệu cũ sau khi quyền đổi, xếp hạng bằng LLM.

Tiền điều kiện chung: stack test + seed (US-P2-12); `$A,$T,$TA_,$SVA,$SVB,$SVC,$SVD`, `C1`, `C2`, `RDS`, `PSQL`; đồng hồ thật (đo ≤ 60 s thật) — các TC cần "tuổi 48 giờ" QC sửa `enrollments.created_at` / `status_changed_at` trong DB (ghi rõ). Công cụ: **S** `scripts/p211.sh`, **D**, **A**, **G**, **K** = k6 (`benchmarks/load/today.js` của dev **và** kịch bản riêng của QC `scripts/p211-today.js`). Thiếu route → FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P211-01 | AC1 | repo | **S** `grep -rn 'internal/llm"' backend-go/internal/today \| grep -v _test.go \| wc -l`; `go doc ./internal/today Provider` | `0` (không LLM); `Provider{Name(); Items(ctx, Viewer, Scope)}`, `Register(p)`; `Viewer` gồm `UserID`, `Role`, `EmailVerified`, `Courses []CourseRef` (**chỉ** lớp `ACTIVE`); Provider **không** nhận id người / lớp ngoài `Viewer` |
| TC-P211-02 | AC1 (Provider lỗi) | – | **S** làm một Provider trả lỗi (QC chặn bảng liên quan qua DB tạm `revoke select` hoặc test Go) | Phần còn lại vẫn trả `200`; Provider hỏng bị bỏ + log `warn`; **không** sập trang |
| TC-P211-03 | AC1/AC2 | – | **G** `-run 'TestProviderRegistry\|TestAggregatorSkipsFailingProvider\|TestRankingDeterministic\|TestRankTierOrder\|TestRankOverdueFirst\|TestRankStableTiebreak\|TestRankAllTierPairs'` | `ok` |
| TC-P211-04 | AC2 (xếp hạng — QC tự kiểm) | GV có: `EMAIL_MISMATCH`, `JOIN_REQUEST` (một quá 48 h), `COURSE_SETUP` ở nhiều lớp | **S** `curl -sk -H "$T" $GW/api/v1/me/today \| jq -r '.actions[] \| [.kind,.urgency,.course.class_code,.age_minutes] \| @tsv'`; so với bảng bậc SRS 4.7 | Thứ tự `(quá hạn giảm dần, bậc tăng dần, tuổi giảm dần, mã lớp tăng dần, id tăng dần)`: mục `overdue` (JOIN_REQUEST > 48 h) **trước mọi bậc**; `EMAIL_MISMATCH`(40) < `JOIN_REQUEST`(45) < `COURSE_SETUP`(80); **xác định** (gọi 20 lần → cùng thứ tự); Admin: lỗi nhà cung cấp(10) < ngân sách cạn(20) < cảnh báo(30) < lớp không GV(40) < lời mời hết hạn(50); SV: `VERIFY_EMAIL`(10) < `JOIN_CODE`(20) < `JOIN_PENDING`(30) |
| TC-P211-05 | AC3 (Sinh viên — chưa lớp) | SV D | **S** `curl -sk -H "$SVD" $GW/api/v1/me/today \| jq '.no_course, .recommended'` | `no_course=true`; `recommended.kind=="JOIN_CODE"` ("Nhập mã tham gia lớp"), lý do "Bạn chưa vào lớp nào. Nhập mã do giảng viên cung cấp để bắt đầu.", `href=/join`, `estimate_minutes=1`; **tối đa MỘT** `recommended` |
| TC-P211-06 | AC3 (chưa xác minh) | SV chưa xác minh email | **S** `me/today` | `recommended.kind=="VERIFY_EMAIL"` đứng trước (lý do "Chưa xác minh email thì chưa vào được lớp. Kiểm tra hộp thư {a***@x.com}." — email **che**, không lộ đầy đủ), `href=/verify-email` |
| TC-P211-07 | AC3 (chỉ PENDING) | SV chỉ `PENDING` | **S** `me/today` | `JOIN_PENDING` ("Chờ giảng v…"); không dữ liệu lớp |
| TC-P211-08 | AC3 (không việc gấp, timeline) | SV A ở 2 lớp, hôm nay có buổi 10 (lớp 1) và buổi 4 (lớp 2) | **S** `courses/$C1/today`, `me/today` | `{no_course,email_verified,recommended,timeline[],continue[]}`; `recommended=null` khi không việc gấp; `timeline` có cả hai buổi hôm nay (theo `starts_at`, giờ VN); `continue` rỗng ở P2 |
| TC-P211-09 | AC3 | – | **G** `-run 'TestStudentTodayNoCourse\|TestStudentTodayUnverified\|TestStudentTodayPendingOnly\|TestStudentTodayNoUrgent\|TestStudentTodayTimeline\|TestStudentTodayAtMostOneRecommended'` | `ok` |
| TC-P211-10 | AC4 (GV / TA) | GV 2 lớp | **S** `me/today` (tất cả lớp) và `courses/$C2/today` (một lớp) bằng `$T`, `$TA_` | `{count, actions[], attention[], upcoming[]}`; mỗi mục `{id,kind,title,reason,urgency,href,course:{id,class_code},age_minutes?,steps?}`; ở chế độ tất cả lớp mục ghi **rõ lớp**; có mục của cả 761987 và 761988; `count` đúng (đối chiếu SQL); trả tối đa 50; `attention[]` rỗng; `upcoming[]` ≤ 10 buổi trong 7 ngày theo `starts_at`; `reason` tiếng Việt **sinh từ dữ liệu** (đổi dữ liệu → lý do đổi: tạo thêm 1 yêu cầu → "N+1 yêu cầu…") |
| TC-P211-11 | AC4 | – | **G** `-run 'TestStaffTodaySingleCourse\|TestStaffTodayAllCourses\|TestStaffTodayReasonsFromData\|TestStaffTodayCountAndCap\|TestStaffTodayUpcoming'` | `ok` |
| TC-P211-12 | AC5 (nguồn việc staff) | lớp 2: 3 PENDING; 1 `EMAIL_MISMATCH` ở lớp 1 | **S** GV và TA `me/today`; đổi `created_at` của yêu cầu cũ nhất thành 49 giờ trước | `JOIN_REQUEST`: "N yêu cầu vào lớp {mã} đang chờ duyệt, cũ nhất {tuổi}." → `/class/members?course={id}&tab=pending` (TA **và** GV); `overdue` khi > 48 h; **`EMAIL_MISMATCH` chỉ GV** (TA không thấy); `COURSE_SETUP` chỉ GV, 4 bước tự tick theo dữ liệu (≥ 1 SV ACTIVE/PENDING; `documents` `COURSE_POLICY`; `class_sessions`; tài liệu…) |
| TC-P211-13 | AC5 (tự biến) | – | **S** thêm dữ liệu cho từng bước (tạo buổi, tài liệu…) rồi gọi lại; `setup/dismiss` | Bước tự tick; đủ 4 → `COURSE_SETUP` **biến**; `dismiss` ẩn mục (idempotent) |
| TC-P211-14 | AC5 | – | **G** `-run 'TestProviderJoinRequest\|TestProviderEmailMismatchTeacherOnly\|TestProviderCourseSetupSteps\|TestProviderCourseSetupDisappears\|TestProviderCourseSetupDismiss\|TestProviderOverdueAfter48h'` | `ok` |
| TC-P211-15 | AC6 (Admin) | Admin; nhà cung cấp `fake` lỗi; ngân sách 85 %; lớp không GV; lời mời quá hạn | **S** `curl -sk -H "$A" $GW/api/v1/me/today \| jq '.actions[].title'` | Có đủ: "Nhà cung cấp {tên} đang lỗi. Chat của sinh viên có thể dùng dự phòng." → `/settings/llm`; "Chi phí AI hôm nay đã dùng 85 % ngân sách." → `/settings/llm`; "Lớp {mã} chưa có giảng viên." → `/admin/courses`; "{N} lời mời giảng viên đã hết hạn chưa được dùng." → `/admin/users?status=INVITED`; xếp đúng bậc 10 < 20 < 30 < 40 < 50 |
| TC-P211-16 | AC6 (**Admin không xem việc trong lớp**) | – | **S** Admin `GET /courses/$C1/today` | **`403`** (Admin không xem việc trong lớp); không dữ liệu lớp |
| TC-P211-17 | AC6 | – | **G** `-run 'TestAdminTodayLLMProviderError\|TestAdminTodayBudget\|TestAdminTodayCourseNoTeacher\|TestAdminTodayExpiredInvites\|TestAdminCannotCourseToday'` | `ok` |
| TC-P211-18 | AC7 (**cách ly — tấn công chính**) | SV A (2 lớp), B (lớp 1), X (`sv.lech`, PENDING), D | **S** mỗi người gọi `me/today` và `courses/<lớp mình>/today`; trích mọi `course.id`, tên, email, MSSV, `join_code`, `user_id` trong thân bằng `jq`/`grep` | Mỗi phản hồi chỉ có lớp / buổi / yêu cầu của **chính người đó**; **không** `user_id`, tên, email, MSSV, mã tham gia của người khác; mọi `course.id` ∈ lớp `ACTIVE` của người đó |
| TC-P211-19 | AC7 (kind staff không sang SV) | lớp 2 có `JOIN_REQUEST`, `EMAIL_MISMATCH`, `COURSE_SETUP` | **S** SV A, B, C, D, X gọi `today`; `grep -cE 'JOIN_REQUEST\|EMAIL_MISMATCH\|COURSE_SETUP'` | **0** — việc chỉ-dành-cho-staff **không bao giờ** xuất hiện ở phản hồi Sinh viên |
| TC-P211-20 | AC7 (IDOR lớp) | B không học lớp 2 | **S** B `GET /courses/$C2/today`; X (PENDING) `GET /courses/$C2/today`; người `REMOVED` | **`403`** (cả `PENDING`, `REMOVED`, ngoài lớp) |
| TC-P211-21 | AC7 (fuzz) | – | **S** 200 cặp (người, lớp) ngẫu nhiên (QC sinh từ seed) gọi `courses/{lớp}/today` | `2xx` **chỉ** khi người đó là thành viên `ACTIVE`; mọi `course.id` trong kết quả ∈ lớp của người đó; mọi cặp khác `403` |
| TC-P211-22 | AC7 | – | **G** `-run 'TestStudentTodayOnlyOwnData\|TestStaffKindsNeverInStudentResponse\|TestTodayOutsider403\|TestTodayFuzzCourseScope'` | `ok` |
| TC-P211-23 | AC8 (ma trận vai) | 4 vai + không JWT | **S** `me/today` và `courses/{id}/today` × {SV, TA, GV, Admin, PENDING, REMOVED, ngoài lớp, không JWT, hết hạn}; thử tham số `?role=ADMIN`, `?viewer=…`, `?user_id=…` | `me/today`: không JWT `401`; mỗi vai đúng dạng (SV / staff / admin); `courses/{id}/today`: thành viên đúng vai 200; ngoài lớp / PENDING / REMOVED / **Admin** `403`; **dạng phản hồi không đổi theo tham số client**; ≥ 24 ca |
| TC-P211-24 | AC8 | – | **G** `-run TestTodayRBACMatrix -v` | `ok` |
| TC-P211-25 | AC9 (cache) | Redis | **S** gọi `me/today` hai lần; đếm truy vấn SQL (`pg_stat_statements` / log) lần hai; `$RDS ttl "ep:today:<uid>:all"` | Lần hai từ Redis (0 truy vấn bảng nghiệp vụ); `ttl ≤ 60`; khoá dạng `ep:today:{user_id}:{scope}` (`all` hoặc id lớp) |
| TC-P211-26 | AC9 (**vô hiệu theo sự kiện ≤ 2 s**) | GV có `JOIN_REQUEST`; cache đã nóng | **S** duyệt hết yêu cầu (UI/API); đo thời gian tới khi `me/today` của GV không còn `JOIN_REQUEST`; thử cho sự kiện `course.join_requested`, `course.join_decided`, `course.member_changed`, `course.assigned`, `course.changed`, `user.verified`, `roster.imported` | Khoá của **người bị ảnh hưởng** (staff của lớp, người liên quan) bị xoá **≤ 2 s**; `JOIN_REQUEST` biến **không đợi 60 s**; mỗi sự kiện trong 7 loại làm đúng người bị ảnh hưởng đổi |
| TC-P211-27 | AC9 (**quyền đổi, cache cũ**) | cache của SV B nóng (có lớp 1) | **S** GV mời B ra khỏi lớp 1; B gọi `me/today` ngay | Không còn dữ liệu lớp 1 (hoặc ≤ 2 s) — **cache không trả dữ liệu của lớp đã mất quyền**; `courses/$C1/today` của B → `403` ngay |
| TC-P211-28 | AC9 (Redis chết) | `stop redis` | **S** gọi `today` | Tính trực tiếp (**mở cửa**), vẫn đúng; log `error` ≤ 1 lần / 30 s |
| TC-P211-29 | AC9 | – | **G** `-tags integration -run 'TestTodayCacheHit\|TestTodayInvalidatedByOutbox\|TestTodayRedisDownFailsOpen\|TestTodayTTLSafetyNet'` | `ok` |
| TC-P211-30 | AC10 (truy vấn, p95) | 2 lớp × 30 SV | **D** đếm truy vấn SQL / yêu cầu (tắt cache); **K** `k6 run benchmarks/load/today.js` và `scripts/p211-today.js` (20 req/s, 60 s, token GV + SV luân phiên); ETag | ≤ **5** truy vấn / yêu cầu; `http_req_duration p(95) < 300 ms`; 0 lỗi; `ETag` + `304` cho cùng nội dung; ghi số đo và cấu hình máy (số tự đo thắng số công cụ nếu lệch) |
| TC-P211-31 | AC10 | – | **G** `-run 'TestTodayQueryBudget\|TestTodayETag'` | `ok` |
| TC-P211-32 | AC11 (`/` Sinh viên) | SV D, B, A; 375 px | **A** `/` của D, B, A; đếm nút primary; ô nhập mã; timeline; "Tiếp tục học"; thẻ số liệu | D: ô nhập mã 7 ký tự **ngay trên trang** thay cho khuyến nghị (nhập → `/join/<mã>`); B/A: **≤ 1 `button.primary`**, lời chào + ngày, **một** khuyến nghị có lý do + thời lượng ("Xác minh email của bạn · 1 phút"), timeline có "Buổi 10"; "Tiếp tục học" ẩn khi rỗng; **không** thẻ số liệu / biểu đồ; không việc gấp: "Hôm nay bạn không có việc gấp." + buổi kế tiếp |
| TC-P211-33 | AC11 | – | **A** khung xương khi tải (chặn mạng chậm); lỗi (`page.route` 500); từ kỹ thuật; `TOUCH_SRC`, `AUDIT_SRC` 375; đo thời gian "ba giây": từ tải trang tới khi thấy hành động chính | Khung xương `aria-busy`; lỗi chuẩn + `Thử lại`; `getByText(/RAG\|PII\|trace\|provider/i).count()=0`; `[]`; `ox:0`; hành động chính hiện trong **≈ 3 s** (ghi LCP / thời gian) |
| TC-P211-34 | AC11 | – | **G** `$PW today.spec.ts -g 'student today'` | `rc=0` |
| TC-P211-35 | AC12 (`/` GV / TA / Admin) | GV, TA, Admin | **A** `/` mỗi vai: tiêu đề, `ActionList`, `COURSE_SETUP` 4 bước, `upcoming`, "Lớp cần chú ý", chế độ "Tất cả lớp của tôi" | GV: tiêu đề `^\d+ việc cần xử lý hôm nay$` (0 → "Không có việc cần xử lý hôm nay."); hàng gồm tiêu đề + một dòng lý do + "lớp 761988" (tất cả lớp) + một hành động; `COURSE_SETUP` mở dần ngay trong hàng (không modal); `upcoming` dải mảnh; "Lớp cần chú ý" **ẩn** khi rỗng; **không** KPI / hero số; TA **không** thấy `COURSE_SETUP`/`EMAIL_MISMATCH`; Admin thấy việc Admin |
| TC-P211-36 | AC12 (việc biến mất ≤ 60 s) | GV | **A** duyệt hết yêu cầu ở `/class/members`, quay lại `/`; đo | Hàng `JOIN_REQUEST` **biến ≤ 60 s** (thực tế ≤ 2 s nhờ vô hiệu theo sự kiện) |
| TC-P211-37 | AC12 | – | **G** `$PW today.spec.ts -g 'staff today\|admin today'` | `rc=0` |
| TC-P211-38 | AC13 (không dữ liệu mock) | build gate | **A** `/` mọi vai: tìm chuỗi dữ liệu mock cũ; **S** `grep -rn 'mock' frontend/src/app/\(app\)/page.tsx frontend/src/features/today \| wc -l`; `audit.mjs` | `innerText` không chứa "Phiếu hỗ trợ", "QUIZ01", "Bài tập 03"; `grep` `0`; mục của phase chưa làm (ticket P4, chấm bài P7, thread, QUIZ P9) **không hiện**; `audit.mjs` FAIL 0 (route `/` thật nên số hàng có thể đổi — ghi chênh và lý do); `docs/PROGRESS.md` "Nợ" có Provider P4–P10 (đọc tệp) |
| TC-P211-39 | AC13 | – | **G** `$PW today.spec.ts -g 'no mock items'` | `rc=0` |
| TC-P211-40 | AC14 (lỗi) | – | **S** Provider lỗi (TC-02); làm DB chậm (`pg_sleep` qua test) quá `REQUEST_TIMEOUT` | Phần còn lại trả; quá hạn → `504 DEADLINE_EXCEEDED` |
| TC-P211-41 | AC14 (UI) | – | **A** `page.route` 500 rồi thành công; làm mới thất bại khi đã có dữ liệu; offline | "Chưa tải được việc hôm nay. Dữ liệu của bạn không bị ảnh hưởng." + `Thử lại` (**đúng 1** request); dữ liệu cũ **vẫn hiện** (không nháy trống); `OfflineBanner` khi mất mạng |
| TC-P211-42 | AC14 | – | **G** `-run 'TestTodayProviderErrorPartial\|TestTodayDeadline504'`; `$PW today.spec.ts -g 'today errors'` | `ok`; `rc=0` |
| TC-P211-43 | tổng (không LLM) | – | **S** `grep -rn 'llm' backend-go/internal/today --include=*.go -il \| grep -v _test`; `select count(*) from llm_audit` trước-sau khi gọi `today` 50 lần | Không phụ thuộc LLM; `llm_audit` **không tăng** (xếp hạng không dùng LLM) |
| TC-P211-44 | tổng | – | **S** `go vet && golangci-lint run && go test -race -count=1 ./internal/today/... && go test -tags integration ./internal/integration/...` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| SV thấy dữ liệu / việc của người khác | 18, 21 |
| Việc dành cho staff lọt sang SV | 19 |
| Admin xem việc trong lớp; ngoài lớp / PENDING / REMOVED xem lớp | 16, 20, 23 |
| Cache trả dữ liệu cũ sau khi mất quyền | 27 |
| Việc đã xử lý không biến mất | 26, 36 |
| Provider lỗi làm sập cả trang | 02, 40 |
| Xếp hạng không ổn định / dùng LLM | 04, 43 |
| Email lộ đầy đủ ở lý do | 06 |
| Dữ liệu mock lẫn dữ liệu thật | 38 |
| Chậm hơn SLO | 30 |

## Câu hỏi cho BA / PM
- **Q-QC-P211-1** — TC-P211-33 "ba giây": QC đo thời gian từ tải trang tới khi hành động chính hiện (không phải LCP); đồng ý với phép đo này? — *chờ xác nhận*.
- **Q-QC-P211-2** — Bậc của `VERIFY_EMAIL` so với `JOIN_CODE` khi **cả hai** đúng (SV chưa xác minh **và** chưa vào lớp): US AC3 đặt `VERIFY_EMAIL` trước. QC chấm theo đó. — *chờ xác nhận*.
- **Q-QC-P211-3** — TC-P211-26 có 7 loại sự kiện: QC chỉ có thể kích hoạt các sự kiện có API tương ứng (join request/decision, member change, assign, roster import, verify); `course.changed` qua sửa lớp. Nếu có sự kiện không kích hoạt được bằng HTTP, ghi N/A và chạy test Go của dev. — *thông báo*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1 (FEAT-course-foundation, APPROVED 2026-10-03).

Tổng: 44 TC.
