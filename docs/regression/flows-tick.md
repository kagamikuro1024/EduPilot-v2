# Đối chiếu luồng F1 / F2 / F14 / F15 / F18 để tick PROGRESS.md (qc2, 25/11)

Căn cứ: HEAD `sprint/11-p10-eval` ≥ 2366ee52 (worktree qc2 @5b9aabf7, merge cùng nhánh). Luật tick (PM): spec E2E xanh cho đường chính + ít nhất một nhánh lỗi trong `docs/FLOWS.md`.
Không sửa PROGRESS.md (PM ghi). Mọi lượt chạy không gọi LLM.

## Kết luận
| Luồng | Kết luận | Lý do ngắn |
| --- | --- | --- |
| F1 Tài khoản | **TICK ĐƯỢC** | `account.spec.ts` xanh (đường chính SV, mời GV / TA, quên + đặt lại, phiên, chống dò, link hết hạn / đã dùng); `qc-real-f1-account` 5/6 xanh trên ep-r gồm mạo danh MSSV và thu hồi phiên; Go auth xanh. Ca «dò mật khẩu» của spec gốc đỏ vì spec (không phải sản phẩm): xem §F1 |
| F2 Mở lớp → vào lớp | **TICK ĐƯỢC** | `class-join.spec.ts` 51 ca xanh; `qc-real-f2-class` 4/4 xanh (qc2seedt2); Go `TestCourseIsolation` + `TestAllCourseRoutesGuarded` + course xanh. Ghi chú: «ticket đang mở chuyển cho GV mới» không có mã riêng và không có spec |
| F14 Hôm nay | **THIẾU** | đường chính + nhánh lỗi xanh nhưng 3 nguồn việc trong bảng FLOWS không có mã: SV «tài liệu mới», Admin «dead-letter», Admin «sao lưu thất bại» |
| F15 Cấu hình LLM, ngân sách, tích hợp | **TICK ĐƯỢC** | `settings-llm` (19) + `settings-integrations` (8) + `decision-settings` xanh; `qc-real-f15-llm` 2 xanh (1 bỏ: cần `FAKE_LLM_VALID_KEY`); Go ngân sách / fallback / breaker xanh |
| F18 Kết thúc học kỳ | **THIẾU** | chỉ `Lưu trữ lớp` có mã; xuất ZIP, nhân bản học kỳ, job giữ dữ liệu chưa có (phase PR) |

## Các lượt chạy (log trong `evidence-flows/`)
| Lượt | Lệnh | Kết quả |
| --- | --- | --- |
| Mock trong máy | `E2E_API_PORT=3412 pnpm build:gate` (rc=0), rồi `E2E_PORT=3410 E2E_API_PORT=3412 playwright test account class-join today settings-llm settings-integrations decision-settings settings-error --workers=2 --retries=0` (cổng 3410/3412 trống trước khi chạy) | **212 passed, 0 failed**, 88 skipped (`pw-mock.log`). 88 skipped = 15 ca `@real` bỏ ở cả hai project + ca khai báo chỉ-desktop / chỉ-mobile; không ca không-@real nào bị bỏ ở cả hai project |
| Go hẹp (container dùng chung, mỗi test DB riêng; `go-flows.sh`) | auth (Register / Verify / Forgot / Reset / Lockout / Accept / Refresh / Resend / Throttle / Roster), course (Assign / Archive / Rejoin / Remove / Join / ChangeTeacher / TA / Pending / Undo / Member), integration (`TestCourseIsolation`, `TestAllCourseRoutesGuarded`, `TestAssignTeacher*`), `internal/today` toàn gói, llm + llmconfig (Budget / Fallback / Breaker / Provider / Route / Key / Verify) | **tất cả `ok`**, không FAIL (`go-flows.log`, `go-integration-v.log`: 6 test PASS, 0 SKIP) |
| Stack-only trên ep-r (`qc-real-flow.config`, `--workers=1`, không LLM; `pw-real-flows.log`, `pw-real-rerun.log`, `pw-f15-rerun2.log`, `pw-f1-do-mat-khau-rerun.log`) | `qc-real-f1-account`, `qc-real-f15-llm` | F1: 5/6 xanh, «dò mật khẩu» đỏ (spec: xem §F1); F15: 2 passed + 1 skipped (chạy riêng); lượt đầu F15/F2 đỏ vì `RATE_LIMITED` sau ca dò (môi trường) |
| Stack-only trên qc2seedt2 (`pw-f2-t2.log`, `lockout_real-t2.log`) | `qc-real-f2-class`; `lockout_real.py` | F2: **4/4 xanh**; lockout thay thế: ĐẠT |

## F1 Tài khoản — **TICK ĐƯỢC**
| Mục FLOWS | Spec / test | Lượt chạy |
| --- | --- | --- |
| Đường chính SV: đăng ký → mail xác minh → xác minh → đăng nhập → `/` có ô nhập mã | mock `account.spec.ts`: register page (nhãn, lỗi từng ô, màn «Kiểm tra email» + đếm ngược gửi lại), verify page (tự xác minh đúng một lần, token rời URL), login; **@real** `qc-real-f1-account`: register → Mailpit → xác minh → đăng nhập → ô nhập mã | mock xanh; @real ✓ (ep-r, 5,1 s) |
| Đường chính GV / TA: Admin mời → link 72 h → tự đặt mật khẩu | mock: invite page, admin users (mời tại chỗ, gửi lại cùng khoá); @real mời GV: Admin mời → Mailpit → đặt mật khẩu → khoá → mất phiên → mở khoá; Go `TestAcceptInvite*` (Expired72h, Reuse, Race, OnlyWhileInvited) | xanh; @real ✓ |
| Quên mật khẩu: câu chung, link 30 phút, thu hồi phiên cũ | mock: forgot page (một câu chung bất kể email), reset page; @real: reset thu hồi phiên máy khác (token máy B chết ngay); Go `TestForgotUniform`, `TestForgotTimingEqualized`, `TestResetRevokesAllSessions` | xanh; @real ✓ |
| Phiên: access 15 phút + refresh xoay vòng, `/settings` liệt kê thiết bị | mock: no token in storage, two tabs refresh, refresh hỏng, settings security (đổi mật khẩu, thiết bị, đăng xuất); @real login + refresh qua Caddy; Go `TestRefresh*` (Rotates, Reuse*, Absolute*, ConcurrentSameToken) | xanh; @real ✓ |
| Chống dò: 5 lần sai chờ tăng dần, 10 lần khoá 15 phút + mail | mock «login throttled (…): đếm ngược mỗi giây, nút khoá kèm lý do»; Go `TestLockoutBackoffSchedule`, `TestLockoutAfter10`, `TestLockoutBlocksCorrectPassword`, `TestLockoutMailOnce`; @real `dò mật khẩu` | mock + Go xanh; **@real spec gốc ĐỎ (xem dưới)**; kiểm thay thế `lockout_real.py` **ĐẠT** |
| **Quy tắc an toàn số 1:** MSSV tự khai không mở dữ liệu | @real `mạo danh MSSV`: đăng ký bằng MSSV của SV trong lớp, email khác → PENDING + cảnh báo lệch, không thấy dữ liệu SV thật; mock members «email chưa khớp MSSV — giảng viên xác nhận rồi duyệt; TA không thấy nút Duyệt»; Go `TestVerifyPromotesRosterPending`, `TestRosterInviteAcceptLinksOwnRow` | @real ✓ (4,0 s); mock + Go xanh |
| Nhánh lỗi: link hết hạn / đã dùng → trang giải thích + gửi lại; mail không tới → gửi lại sau 60 s; email chưa xác minh không vào lớp | mock verify page (thành công / đã dùng / hết hạn + gửi lại), register đếm ngược gửi lại, today «email chưa xác minh», join page «chưa xác minh email»; Go `TestVerifyEmailExpired24h`, `TestResendThrottle60s`, `TestJoinUnverified403` | xanh |
| OIDC Microsoft (tuỳ chọn, phase PR) | không thuộc phạm vi tick | — |
**Ca đỏ và vì sao không chặn tick.** `qc-real-f1-account` «dò mật khẩu» đỏ cả hai lượt (ep-r): lượt 1 chạy lúc máy tải 12 → quá 120 s; lượt 2 máy nhẹ (tải 2,6, cửa sổ giới hạn IP đã hết) vẫn đỏ nhưng **không phải timeout**: với `--timeout=400000` spec dừng ở `expect(wrong >= 10)` nhận 9. Chẩn đoán (`evidence-flows/dbg_lock.py` lặp đúng vòng lặp của spec: hỏi lại mỗi 2,5 s khi gặp 429): ngoài bậc chờ tăng dần `LOGIN_THROTTLED` (2, 4, 8, 16, 32 s), các lượt hỏi dồn ≈ 24 lượt/phút tự chạm `429 RATE_LIMITED` theo IP (Retry-After tới 38 s, xen kẽ giữa các bậc), nên tới hết 60 vòng (127,8 s) chỉ có 9 lần sai thật (lần sai thứ 8 ở 74,8 s, thứ 9 ở 92,6 s). Đây là lỗi của **spec** (hỏi dồn thay vì chờ `Retry-After`), không phải sản phẩm. Kiểm thay thế `evidence-flows/lockout_real.py` (cùng kịch bản, chờ đúng `Retry-After`) trên qc2seedt2 (`lockout_real-t2.log`): 10 lần sai thật sau 67 s (bậc chờ 2, 4, 8, 16, 32 s), `locked_until` ≈ 15 phút, đúng mật khẩu khi khoá → `429 LOGIN_THROTTLED` Retry-After 900, **đúng 1** thư «Tài khoản EduPilot bị khoá tạm thời» ⇒ **ĐẠT**. Đề nghị QC / dev sửa vòng lặp spec gốc theo `Retry-After` (không phải việc của qc2: không sửa test cũ).

## F2 Mở lớp → vào lớp — **TICK ĐƯỢC** (kèm 1 ghi chú)
| Mục FLOWS | Spec / test | Lượt chạy |
| --- | --- | --- |
| Đường chính: Admin mở lớp + gán GV → GV nhập roster / chia sẻ mã → SV xem trước, nhập mã, vào lớp (có duyệt) | mock `class-join.spec.ts`: admin courses (mở lớp, gán lại PUT assign), join page (xem trước → tham gia; cần duyệt / chờ duyệt), join link `/join/<mã>` qua đăng nhập, roster import UI (dòng 7 và 19 → Nhập 28), class settings (mã lớn, tạo lại mã có xác nhận), members (duyệt, từ chối, Hoàn tác); @real `qc-real-f2-class`: bộ chọn lớp theo seed, nhập roster-2-loi.csv, vào lớp end-to-end 375×812 (xem trước → vào → GV duyệt → tạo lại mã, mã cũ chết ngay), Hôm nay | mock xanh; **@real 4/4 ✓ trên qc2seedt2** |
| Nhánh lỗi vào lớp: sai mã, đầy lớp, bị giới hạn, chưa xác minh email | mock «join page: cần duyệt, chờ duyệt, sai mã, đầy lớp, bị giới hạn đếm ngược, chưa xác minh email»; @real 6 mã sai ⇒ 429; Go `TestJoinUniformFailure`, `TestJoinFullIsDistinct`, `TestJoinUnverified403`, `TestJoinConcurrentOneEnrollment` | xanh |
| Đổi GV giữa kỳ: GV mới nhận thông báo, GV cũ mất quyền ngay, ghi `audit_log` | mock admin courses «sửa và gán lại dùng PUT / assign», bell «Phân công lớp làm mới danh sách lớp»; Go `TestAssignOldTeacherLosesAccessNow`, `TestChangeTeacherMidTerm`, `TestAssignAudit`, `TestAssignNotifyOnce`, `TestAssignTeacherNotifies` (integration) | xanh |
| GV thêm / bớt TA (Admin cũng làm được) | mock members «tab Trợ giảng (thêm / bớt)», TA chỉ đọc; Go `TestTeacherManagesOwnTAs`, `TestOtherTeacherCannotManageTAs`, `TestAssignReplacesTAs` | xanh |
| SV bị mời ra: mất quyền ngay, dữ liệu học tập giữ, vào lại bằng mã nối đúng bản ghi cũ | mock members «mời ra có Hoàn tác»; Go `TestRemovedLosesAccessImmediately`, `TestRemovedKeepsLearningData`, `TestRejoinReusesRowPending` (cùng dòng enrollment, MSSV chụp cũ giữ nguyên, vào lại luôn chờ duyệt) | xanh |
| Lớp đã lưu trữ: mã tham gia tự tắt | Go `TestArchiveCourse` (`join_enabled = false`); mock «admin courses … lưu trữ có xác nhận nêu số» | xanh |
| Cô lập lớp | Go `TestCourseIsolation`, `TestAllCourseRoutesGuarded` (+ `…DetectsMissing`): PASS, 0 SKIP (`go-integration-v.log`) | xanh |
**Ghi chú (không chặn tick):** FLOWS ghi giảng viên mới nhận «toàn bộ ticket đang mở». Ticket không có cột người nhận (không có `assignee` trong `internal/escalation`), chỉ có phạm vi lớp nên GV mới thấy chúng ngầm; không có test hay spec khẳng định điều này. Nếu PM muốn tick chặt, cần một test «GV mới thấy ticket cũ sau `assign`».
**Môi trường F2 @real:** ep-r không dùng được cho `qc-real-f2-class` ca đầu: `sv.moi` («D cố ý chưa vào lớp nào») đang ACTIVE ở 3 lớp do lượt QC khác (lượt 1 và lượt chạy lại đều đỏ ở `expect(nD).toBe(0)`: Received 3, các ca sau không chạy vì spec `serial`). Chuyển sang qc2seedt2 (seed sạch hơn): `sv.moi` đang ACTIVE ở 761988 do lượt cũ; GV mời `sv.moi` ra bằng API sản phẩm `DELETE /courses/{id}/members/{uid}` (REMOVED, không xoá dòng) rồi chạy: 4/4 xanh. Spec này chỉ chạy lại được một lần trên một seed.

## F14 Hôm nay — **THIẾU** 3 nguồn việc (đường chính + nhánh lỗi đã xanh)
Spec: `today.spec.ts` (22 ca × 2 project, xanh trong lượt mock): SV chưa có lớp → ô nhập mã → `/join/<mã>`; email chưa xác minh đứng trước; chỉ có yêu cầu chờ duyệt; không việc gấp + dòng thời gian; «Tiếp tục học» (≤ 3 phiên, thứ tự); GV: tiêu đề đếm việc, nhóm Gấp / Hôm nay / Khi rảnh, `Thiết lập lớp mới` 4 bước + Bỏ qua, TA không có Bỏ qua, khối «Lớp cần chú ý»; Admin: việc của hệ thống, không có lớp; **nhánh lỗi:** «today errors: lỗi chuẩn đúng chuỗi, Thử lại gọi lại đúng 1 yêu cầu»; không dữ liệu mô phỏng cũ; 375 px sạch. Go `internal/today` toàn gói `ok` (50,9 s): từng nguồn có provider + test, kèm `TestTodayRBACMatrix`, `TestStaffKindsNeverInStudentResponse`, `TestTodayOutsider403`, `TestTodayFuzzCourseScope`, `TestTodayRedisDownFailsOpen`, `TestTodayProviderErrorPartial`, `TestTodayDeadline504`, `TestTodayInvalidatedBy*`.
Đối chiếu bảng «nguồn việc» của FLOWS với các `Kind` trong `backend-go/internal/today` (41 kind):
| Vai | Nguồn trong FLOWS | Trạng thái |
| --- | --- | --- |
| SV | chưa có lớp · email chưa xác minh · hạn nộp < 24 h · bài thi đang làm / mở / sắp mở · QUIZ / thi 48 h · điểm vừa công bố · phúc khảo có phản hồi · chủ đề sai nhiều · câu trả lời GV chưa đọc | có (`JOIN_CODE`, `VERIFY_EMAIL`, `ASSIGNMENT_DUE`, `EXAM_*`, `ASSIGNMENT_GRADED`, `*_APPEAL_REPLY`, `PRACTICE_WEAK_TOPIC`, `STAFF_ANSWER_UNREAD`) |
| SV | **tài liệu mới** | **không có kind và không có test** |
| GV / TA | mọi nguồn trong bảng (buổi chưa điểm danh, cờ điểm danh bằng mã, ticket, phúc khảo, cần xem kỹ, chưa công bố, AI chờ xác nhận, bài bị báo cáo, chưa khớp, yêu cầu vào lớp, email lệch MSSV, công thức chưa xác nhận, thiết lập lớp, câu hỏi chờ duyệt, phúc khảo thi, hoãn công bố, code chấm lỗi, cặp code giống nhau, SV cần chú ý) | có (`SESSION_UNMARKED`, `CHECKIN_REVIEW`, `TICKET`, `APPEAL`, `GRADING_REVIEW`, `PUBLISH_PENDING`, `AI_CONFIRM`, `POST_REPORTED`, `UNMATCHED_SUBMISSION`, `JOIN_REQUEST`, `EMAIL_MISMATCH`, `GRADE_SCHEME_UNCONFIRMED`, `COURSE_SETUP`, `QUESTION_REVIEW`, `EXAM_APPEAL`, `EXAM_PUBLISH_HOLD`, `EXAM_GRADE_ERROR`, `EXAM_SIMILARITY`, `STUDENT_ATTENTION`) |
| Admin | Provider LLM lỗi · sắp chạm trần ngân sách · lớp không có giảng viên | có (`LLM_PROVIDER_ERROR`, `LLM_BUDGET_WARN/EXHAUSTED`, `COURSE_NO_TEACHER`; thêm `INVITE_EXPIRED`) |
| Admin | **hàng dead-letter có việc** · **sao lưu gần nhất thất bại** | **không có kind và không có test** (`AdminProvider` chỉ có bốn nguồn trên). Cả hai phụ thuộc `/admin/health` + nguồn sao lưu, FLOWS F16 ghi «thuộc phase PR» |
**Kết luận F14: THIẾU** ba nguồn: SV «tài liệu mới»; Admin «dead-letter», «sao lưu thất bại». Nếu PM chuyển hai nguồn Admin sang PR và bỏ «tài liệu mới» khỏi bảng F14 trong FLOWS.md thì phần còn lại đủ điều kiện tick (đường chính + nhánh lỗi xanh).

## F15 Cấu hình LLM, ngân sách, tích hợp — **TICK ĐƯỢC**
| Mục FLOWS | Spec / test | Lượt chạy |
| --- | --- | --- |
| Thêm provider → Test kết nối → gán model theo tác vụ → fallback | mock `settings-llm.spec.ts`: provider row (Test kết nối mỗi hàng, lỗi dưới hàng, thêm nhà bấm đúp Lưu chỉ 1 POST), routing (đổi mô hình lạc quan, Hoàn tác, Lên bằng Enter), embedding (chỉ 1536 chiều, đổi cần xác nhận), advanced; `decision-settings.spec.ts` (nhà «Quyết định», ba cổng, 409); Go `TestFallbackOrder`, `TestBreaker*`, `TestNoFallbackOnBadRequest`, `TestMaskOnceBeforeFallback` | xanh |
| Nhánh lỗi cấu hình | mock: wrong key 422 `PROVIDER_AUTH_FAILED` dưới ô khoá, key write-only (không vào DOM / storage / console), Test kết nối chậm 17 s (hạn 30 s), tải lỗi 503 + Thử lại, `LLM_NOT_CONFIGURED`, mất mạng khi Lưu → Gửi lại cùng Idempotency-Key, 409; roles (SV / TA chặn, GV chỉ xem, Admin đủ); @real: 403 TEACHER/TA/SV (API + UI) ✓, hai tab 409 + nạp lại ✓ | mock xanh; @real 2/2 ✓ (ep-r) |
| Ngân sách ngày / tháng, hệ thống và theo lớp; 80 % cảnh báo; 100 % BATCH dừng, INTERACTIVE sang model rẻ nhất; không tắt chat mà không báo | mock «usage \| budget: 7 cột, cảnh báo 80 %», «Theo lớp / Đổi hạn mức»; Go `TestBudgetTwoScopes`, `TestBudgetWarn80Once`, `TestBudgetExhaustedBatchStops`, `TestBudgetExhaustedInteractiveCheapest`, `TestDecideBudgetNoCheapest`, `TestBudgetDayRollover`, `TestBudgetTakesEffect`, `TestCourseBudgetAdminOnly`, `TestBatchDoesNotStarveInteractive` | xanh; mốc 100 % chỉ có Go, không có E2E giao diện |
| OpenAI-compatible / máy chủ riêng | mock «nhiều nhà cung cấp máy chủ riêng: mỗi hàng có địa chỉ riêng» | xanh |
| Tích hợp (IMAP, Teams) | mock `settings-integrations.spec.ts` (8 ca): IMAP key write-only, bốn trạng thái, đồng bộ ngay 202 rồi 429, 422 / 409, Teams chưa bật | xanh |
**Không chạy được:** `qc-real-f15-llm` «khoá sai … khoá đúng lưu được» bị `test.skip` vì gateway ep-r không đặt `FAKE_LLM_VALID_KEY` (cần đổi env gateway + rebuild: không làm); phần giao diện đã có mock `wrong key`, phần API có Go. Lượt 1 và 2 của spec này đỏ ở `uiLogin` do `RATE_LIMITED` đăng nhập dồn sau ca dò mật khẩu (môi trường); chạy riêng lúc máy nhẹ: 2 passed, 1 skipped (`pw-f15-rerun2.log`).

## F18 Kết thúc học kỳ — **THIẾU** (chưa tick)
Chỉ bước 2 (`Lưu trữ lớp`) có mã và kiểm; bước 3, 4, 5 chưa có mã nên không thể có spec.
| Bước FLOWS | Hiện có | Bằng chứng |
| --- | --- | --- |
| 1 Chốt điểm | có (F10) | ngoài F18 |
| 2 `Lưu trữ lớp`: chỉ-đọc, mã tham gia tắt, thành viên giữ nguyên, ghi audit | **có** | Go `TestArchiveCourse` (join_enabled = false, archived_at, thành viên ACTIVE giữ nguyên, lớp vẫn đọc được, audit `course_archived`), `TestArchiveIdempotent`, `TestArchivedRejectsWrites`, ghi bị chặn ở assignment / chat / calendar / upload / document / crm / exam (các `*ArchivedCourse*`); mock `class-join.spec` «admin courses page: … lưu trữ có xác nhận nêu số», «admin courses errors: … lưu trữ lại», «course picker: lớp đã lưu trữ không là mặc định» (xanh) |
| 3 `Xuất toàn bộ dữ liệu lớp` (ZIP: sổ điểm, điểm danh, bài nộp, nhận xét) | **không có** | `openapi.yaml` không có route export cả lớp (chỉ `gradebook/export.xlsx`, `exams/{eid}/clone`, `knowledge-gap/{rid}/export`); `grep "Xuất toàn bộ dữ liệu lớp"` trong `frontend/src` = 0 |
| 4 `Nhân bản sang học kỳ mới` (tài liệu, ngân hàng câu hỏi, rubric, cấu trúc bài tập, bản nháp công thức; không mang SV / điểm / hội thoại) | **không có** | không có route clone lớp (chỉ clone bài thi); `grep "Nhân bản sang học kỳ"` = 0 |
| 5 Job giữ dữ liệu: hội thoại + `llm_audit` xoá sau thời hạn (12 tháng sau lưu trữ), heartbeat thô 7 ngày | **không có** (trừ điểm danh) | chỉ có `CHECKIN_RAW_RETENTION_DAYS` (điểm danh, 30 ngày) và rollup quan sát; không có job xoá hội thoại / `llm_audit` theo `retention_policies` |
| E2E toàn luồng | **không có** | không có spec nào đi từ chốt điểm → lưu trữ → xuất → nhân bản |
`docs/phases/PR.md` dòng 22 vẫn `[ ]`: «F18: lưu trữ lớp (chỉ-đọc), xuất dữ liệu lớp, nhân bản sang học kỳ mới, job xoá theo thời hạn». FLOWS xếp F18 vào phase PR. **Không tick F18 trước khi PR làm bước 3–5.**

## Việc đã làm lên môi trường (khai)
- ep-r: nạp lớp KC106 từ lượt G10 trước; xoá khoá Redis `ep:auth:ipfail:172.22.0.1` (bộ đếm 14 do chính spec dò mật khẩu làm đầy) trước một lượt chạy lại; không xoá dữ liệu nghiệp vụ.
- qc2seedt2: GV mời `sv.moi` ra khỏi 761988 bằng API (REMOVED).
- Đã stop ep-r và qc2seedt2 (không down, không -v). Không gọi LLM; không build lại gateway.
- Script: `go-flows.sh`, `pw-real-flows.sh`, `pw-real-rerun.sh`, `pw-real-one.sh`, `lockout_real.py`.
