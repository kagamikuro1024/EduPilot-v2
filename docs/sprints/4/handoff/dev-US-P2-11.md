# DEV handoff — US-P2-11 ("Hôm nay" cho mọi vai)
Nhánh `sprint/4-p2`. Chưa kiểm trên stack compose thật (xem "Chưa làm").

## Làm gì
**Backend `internal/today`** (không import `internal/llm` — `TestNoLLMImport` đọc import thật)
- `Provider{Name; Items(ctx, Viewer, Scope)}`, `Aggregator.Register` / `Collect`: gọi Provider **song song**, mỗi Provider hạn 150 ms, lỗi / panic / quá hạn ⇒ bỏ + log `warn` (`provider=…`), lọc **theo vai** (bảng `allowed`: việc dành cho staff không bao giờ ra cho sinh viên dù Provider trả nhầm), xếp, cắt 50 (`count` vẫn là tổng); ctx yêu cầu hết hạn ⇒ lỗi ⇒ 504 `DEADLINE_EXCEEDED`.
- `Rank`: khoá cứng `(Overdue giảm, Tier tăng, AgeMinutes giảm, mã lớp tăng, id tăng)`, `sort.SliceStable`. Bảng bậc đúng SRS 4.7.
- `Viewer` do `Service.viewer` dựng bằng MỘT truy vấn (`TodayViewer`: người dùng + ghi danh ACTIVE / PENDING của CHÍNH họ trong lớp còn mở); `Viewer.Courses` chỉ ACTIVE, yêu cầu chờ duyệt nằm ở trường riêng. Provider không nhận id người / lớp nào ngoài `Viewer`.
- Provider P2: `StudentProvider` (VERIFY_EMAIL, JOIN_CODE, JOIN_PENDING — không truy vấn DB; "một lớp" không hiện JOIN_PENDING của lớp khác), `StaffProvider` (JOIN_REQUEST = PENDING không cảnh báo; EMAIL_MISMATCH chỉ GV; COURSE_SETUP chỉ GV, bốn bước tự tick, hết khi xong 4 hoặc `dismissed_at`; mỗi nguồn MỘT truy vấn tổng hợp cho mọi lớp), `AdminProvider` (nhà cung cấp AI lỗi — `last_test_ok=false` hoặc mạch mở; ngân sách ≥ 80 % / 100 %; lớp không giảng viên; lời mời hết hạn). Mỗi Provider tự thoát sớm nếu sai vai ⇒ không tốn truy vấn. Dữ liệu cổng AI (mạch, % ngân sách) qua interface `LLMSignals` do `httpapi/todayllm.go` cấp (gói today không biết `internal/llm`).
- `Service.Get`: cache Redis `ep:today:{uid}:{all|id lớp}` TTL 60 s (JSON đã dựng), Redis lỗi ⇒ tính trực tiếp + log ≤ 1 dòng / 30 s, mỗi lệnh Redis hạn 250 ms. Dạng phản hồi chỉ theo vai trong JWT (tham số client bị bỏ qua). Sinh viên: `{no_course, email_verified, recommended (≤ 1), timeline (hôm nay ICT + 7 ngày, ≤ 8, NOW / NEXT / DONE), continue: []}`; staff: `{count, actions, attention: [], upcoming (≤ 10, 7 ngày)}`; admin: `{count, actions}`. `reason` sinh từ dữ liệu + đồng hồ (`Age`: phút < 60, giờ < 48, ngày).
- `Invalidator.Handle` cho 7 topic: `course.join_requested`, `course.join_decided`, `course.assigned` (đã có) nối chuỗi với thông báo (`outbox.Chain`); thêm sự kiện mới ở nguồn: `course.member_changed` (mời ra, hoàn tác), `course.changed` (sửa / lưu trữ lớp, cài đặt tham gia, chia sẻ tài liệu, bỏ qua thiết lập), `roster.imported` (kèm `user_ids`), `user.verified` (xác minh email). Xoá `all` + khoá theo lớp của giảng viên / TA, người trong payload; `course.changed` xoá cả người học và Admin; `user.verified` xoá mọi lớp của người đó. Đăng ký ở `cmd/worker/registry.go`.
- HTTP: `GET /me/today` (mọi vai đăng nhập), `GET /courses/{id}/today` (`Member`: sinh viên / TA / GV ACTIVE của lớp; Admin, PENDING, REMOVED, người ngoài ⇒ 403), `POST /courses/{id}/setup/dismiss` (`Teacher`, 204, idempotent, audit một lần). ETag + `Cache-Control: private, no-cache`, 304. openapi 0.13.0 (+3 thao tác, tổng 67).
- `benchmarks/load/today.js` (k6: 20 req/s, p95 < 300 ms).

**Frontend**: `/` thật (`features/today/*`, không còn `@/mock`): `StudentToday` (một việc nên làm với lý do + "Khoảng N phút", ô nhập mã 7 ký tự ngay trên trang khi chưa vào lớp — nhập xong chuyển `/join/<mã>`, dòng thời gian, "Hôm nay bạn không có việc gấp.", "Tiếp tục học" không hiện), `StaffToday` ("N việc cần xử lý hôm nay" / "Không có việc cần xử lý hôm nay.", `ActionList` đúng thứ tự máy chủ, "lớp 761988" ở chế độ Tất cả lớp, `COURSE_SETUP` mở 4 bước tại chỗ + `Bỏ qua`, dải "Sắp tới", không "Lớp cần chú ý"), `AdminToday`; lỗi chuẩn "Chưa tải được việc hôm nay. Dữ liệu của bạn không bị ảnh hưởng." + `Thử lại`; làm mới lỗi vẫn giữ dữ liệu cũ (`keepPreviousData`) + dải cảnh báo; tự làm mới 60 s và ngay sau khi duyệt / nhập roster (`TODAY_KEY`). `mockBackend("/")` = null. Shell test "keyboard" chuyển sang `/gradebook` (route mô phỏng) vì `/` giờ cần đăng nhập.

## AC tự đánh giá (chạy thật)
| AC | Kết quả |
| --- | --- |
| 1 | `TestProviderRegistry`, `TestAggregatorSkipsFailingProvider` (lỗi / panic / chậm ⇒ bỏ, log `provider=`), `TestRankingDeterministic` (1.000 lần xáo), `TestNoLLMImport` PASS |
| 2 | `TestRankTierOrder` (3 vai), `TestRankOverdueFirst`, `TestRankStableTiebreak`, `TestRankAllTierPairs` (mọi cặp bậc) PASS |
| 3 | `TestStudentTodayNoCourse`, `…Unverified` (VERIFY_EMAIL trước, `no_course` vẫn true, email che `a***@…`), `…PendingOnly` ("đã gửi 2 giờ trước."), `…NoUrgent` (`recommended=null`), `…Timeline` (DONE / NOW / NEXT, ICT, loại hôm qua và > 7 ngày), `…AtMostOneRecommended` PASS |
| 4 | `TestStaffTodaySingleCourse`, `…AllCourses`, `…ReasonsFromData` ("3 yêu cầu vào lớp 761988 đang chờ duyệt" / "Cũ nhất đã chờ 2 ngày." / `age_minutes` / `urgency=overdue`), `…CountAndCap` (110 việc ⇒ `count` 110, 50 mục), `…Upcoming` (≤ 10, 7 ngày, sắp xếp) PASS |
| 5 | `TestProviderJoinRequest` (TA + GV; hàng chờ xác minh email của roster không tính), `…EmailMismatchTeacherOnly`, `…CourseSetupSteps` (tick theo dữ liệu, FAILED không tính), `…CourseSetupDisappears` (kể cả tài liệu CHIA SẺ), `…CourseSetupDismiss` (TA / SV 403, không JWT 401, idempotent, audit 1 lần), `…OverdueAfter48h` (49 h nổi lên trước; 47 h không) PASS |
| 6 | `TestAdminTodayLLMProviderError` (kể cả mạch mở, bỏ nhà cung cấp tắt), `…Budget` (79 / 85 / 100 %), `…CourseNoTeacher`, `…ExpiredInvites` (lời mời còn sống thì không tính), `TestAdminCannotCourseToday` PASS — `LLMSignals` giả cho % ngân sách / mạch (adapter thật chưa chạy với Redis ngân sách thật) |
| 7 | `TestStudentTodayOnlyOwnData`, `TestStaffKindsNeverInStudentResponse`, `TestTodayOutsider403`, `TestTodayFuzzCourseScope` (200 cặp), `integration.TestRosterLinkRequiresVerifiedEmail` (nay có `GET /me/today` của kẻ tấn công: chỉ JOIN_PENDING của chính họ, không có gì của B) PASS |
| 8 | `TestTodayRBACMatrix`: `/me/today` 7 người gọi × 3 biến thể tham số (21) + `/courses/{id}/today` 9 người gọi + uuid hỏng = 31 ca; dạng phản hồi không đổi theo tham số PASS |
| 9 | `TestTodayCacheHit` (lần hai 0 truy vấn, TTL ≤ 60), `TestTodayInvalidatedByOutbox` (7 topic, người liên quan mất khoá, người ngoài còn), `TestTodayInvalidatedByApprove` (worker thật: duyệt hết ⇒ JOIN_REQUEST biến ≤ 2 s), `TestTodayRedisDownFailsOpen` (5 yêu cầu, 1 dòng log), `TestTodayTTLSafetyNet` PASS |
| 10 | `TestTodayQueryBudget` (≤ 5 truy vấn kể cả guard: GV tất cả lớp, GV một lớp, SV), `TestTodayETag` (304) PASS; k6 `today.js` viết, **chưa chạy** |
| 11 | `today.spec.ts` ×4 'student today' (ô nhập mã, email chưa xác minh + một nút chính, chờ duyệt không nút, không việc gấp + timeline) + 375 px (`ox`, `cut`, `TOUCH_SRC` sạch) PASS; ba giây / `@real` chưa đo |
| 12 | `today.spec.ts` 'staff today' (tiêu đề `^\d+ việc cần xử lý hôm nay$`, thứ tự, tất cả lớp, 4 bước tại chỗ, Bỏ qua, 0 việc, TA), 'admin today', 375 px PASS; `@real` skip |
| 13 | `grep -rn mock frontend/src/features/today "frontend/src/app/(app)/page.tsx" \| wc -l` = 0; 'no mock items' PASS; nợ Provider các phase sau ghi vào `PROGRESS.md` "Nợ" |
| 14 | `TestTodayProviderErrorPartial`, `TestTodayDeadline504`; e2e 'today errors' (alert đúng chuỗi, `Thử lại` đúng 1 yêu cầu) PASS |

## Lệch spec / nợ
- AC4 của US-P2-10 ("`GET /me/today` không có gì về lớp 1 hay B"): kẻ tấn công đang PENDING vẫn thấy `JOIN_PENDING` "Yêu cầu vào lớp {mã} đã gửi …" của CHÍNH họ (AC3 bắt buộc) — không có tên, email, MSSV, id của B.
- `JOIN_REQUEST` chỉ đếm hàng chờ duyệt thường; hàng `EMAIL_MISMATCH` có mục riêng, hàng `EMAIL_UNVERIFIED` (roster chờ xác minh email) không tính vào cả hai.
- `LLM_PROVIDER_ERROR` dựa `last_test_ok=false` hoặc mạch mở; nhà cung cấp chưa từng kiểm (null) không báo lỗi.
- Bước `policy` / `documents` của `COURSE_SETUP` dẫn tới `/documents?course=` (màn mô phỏng, P3 / P8 làm thật).
- Ảnh chụp visual của `/` thay đổi (màn mô phỏng cũ không còn): sinh lại ở Cổng P2 trong docker `playwright:v1.63.0-noble`.
- Chưa làm: `curl` tay trên compose (`jq '.no_course, .recommended.kind'`, `ttl "ep:today:<uid>:all"`), k6, `@real`, đo "ba giây" (Q-QC-P211-1).

## Kết quả cổng (đã chạy)
- Go: `make lint sqlc-check` 0 issues ×3; `make test` (race, testroutes + integration) xanh trước lần dọn rig cuối (`internal/today/rig_test.go`) và thêm assertion `today` vào `TestRosterLinkRequiresVerifiedEmail`; hai gói đó, `contract`, `integration` chạy lại xanh.
- Frontend: eslint + tsc sạch, `ui-antipatterns` 0 ✗, `lint-selftest` 7/7 + 19/19; Playwright không `visual` (`--workers=2`): 320 passed + `shell` keyboard sau sửa, 0 failed.
