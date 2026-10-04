# DEV handoff — US-P2-08 (Admin mở lớp, gán giảng viên / TA, lưu trữ; thông báo; chuông thật)
Nhánh `sprint/4-p2`. Chưa kiểm trên stack compose thật (xem "Chưa làm").

## Làm gì
**Backend**
- `internal/course`: `Service.Create` (một transaction = lớp + mã tham gia + gán + outbox + audit), `Update`, `Archive`, `AdminList` (MỘT truy vấn tổng hợp, không `join_code`, không sinh viên), `Assign` / `assignTx` (dùng chung cho `POST /admin/courses/{id}/assign`, `PUT /courses/{id}/assistants` và `teacher_id`/`ta_ids` lúc mở lớp), `AssistantCandidates`, `Notifications` / `MarkRead`; `NewJoinCode` (`crypto/rand` + `big.Int`, 31 ký tự); `Notifier.HandleAssigned` (topic `course.assigned`, đăng ký ở `cmd/worker/registry.go`).
- Gán: khoá dòng `courses` `FOR UPDATE` → kiểm vai/trạng thái (sai ⇒ 422 `ROLE_MISMATCH` "Người này không phải giảng viên.") → gỡ giảng viên cũ TRƯỚC khi gán mới (chỉ mục một giảng viên ACTIVE) → người từng `REMOVED` dùng lại dòng cũ → một dòng outbox `course.assigned` cho mỗi người MỚI; `ta_ids` vắng / `null` = giữ, `[]` = gỡ hết, còn lại thay toàn bộ; audit_log trước / sau chỉ khi có đổi. Lớp `ARCHIVED` ⇒ 409 `COURSE_ARCHIVED` ở sửa / gán / trợ giảng.
- Thông báo: dedupe `course.assigned:<outbox_id>` (`ON CONFLICT … DO NOTHING`); **người đã bị gỡ trước khi worker chạy không nhận gì** (thân giảng viên chứa mã tham gia). `GET /notifications` (cursor, `unread_only`, `unread_count`, `Cache-Control: private, no-store`), `POST /notifications/{id}/read` 204, idempotent, của người khác ⇒ 404.
- Mã lỗi mới `JOIN_CODE_INVALID`, `COURSE_FULL`, `COURSE_ARCHIVED` (apierr + hai enum openapi). openapi 0.10.0 (+9 thao tác, tổng 51); `contract.courseAdminScenarios` phủ mọi status khai báo.
- **Lỗi thật tìm được bằng contract test:** `[]uuid.UUID` không mã hoá được khi pool chạy simple-protocol (PgBouncer) ⇒ `GetUsersForAssign` nhận `text[]` rồi cast `::uuid[]`. Các story sau tránh truyền `[]uuid.UUID` vào sqlc.
- `course.changed` / `today.invalidate` CHƯA ghi outbox (chưa có handler ⇒ sẽ thành dead-letter): US-P2-11 thêm cùng handler vào `Update`, `Archive`, `assignTx`.

**Frontend**: `/admin/courses` thật (`features/admin/AdminCourses.tsx`, `courses/{api.ts,CoursePanel.tsx}`): bảng `DataTable`, một nút chính `Mở lớp` (ẩn khi khung mở nên luôn đúng một), khung mở / sửa / gán lại **tại chỗ** (`Section`, theo proposal #5 đã duyệt), dòng tĩnh sau khi gửi, `ConfirmIrreversible` lưu trữ nêu số, lỗi tại ô (`Mã lớp này đã có.`, `Người này không phải giảng viên.`), sai version (`Giữ thay đổi của tôi` / `Dùng bản mới`), `Gửi lại` dùng cùng khoá, `Lớp này đã được lưu trữ.`. Chuông thật: `shared/session/notifications.ts` (`refetchInterval` 30 s, `refetchOnWindowFocus`, `staleTime 0`, đánh dấu đã đọc lạc quan), `NotificationPopover` thêm `failed` ("Chưa tải được thông báo."); `shared/lib/timeAgo.ts` (ngày lịch giờ Việt Nam) là nguồn duy nhất của "N phút trước". Phiên mô phỏng giữ chuông mô phỏng.

## AC tự đánh giá (chạy thật)
| AC | Kết quả |
| --- | --- |
| 1 | `TestCreateCourse` (201 đúng 8 khoá, không `join_code`, mã 7 ký tự hợp lệ, `join_enabled`, kèm GV/TA ⇒ 2 outbox), `…Idempotent` (cùng khoá ⇒ cùng thân, `Idempotent-Replayed: true`, 1 lớp; thiếu khoá 422), `…DuplicateClassCode` (409 `details.field=class_code`), `…Validation` (12 ca + khoá lạ; không lớp nào được tạo) PASS |
| 2 | `internal/course` (gói trong): `TestJoinCodeAlphabet`, `…Distribution` (100.000 mã, mỗi ký tự 3,23 % ± 0,25), `…CollisionRetry` (trùng 4 lần rồi thành công ở lần 5; trùng mãi ⇒ `ErrCodeExhausted` sau đúng 5 lần, không tạo lớp), `…FixedOnlyOutsideProduction` (production ⇒ 422 `NOT_ALLOWED`) PASS; `grep -rn '"math/rand"' internal/course` → 0 |
| 3 | `TestUpdateCourse`, `…CapacityBelowActive`, `…VersionConflict`, `TestArchiveCourse` (`join_enabled=false`, thành viên giữ nguyên, vẫn đọc được), `…Idempotent` (lần hai không đổi, 1 audit), `TestArchivedRejectsWrites` (sửa / gán / trợ giảng bởi Admin và GV ⇒ 409 `COURSE_ARCHIVED`) PASS. Mã đã tắt ⇒ `JOIN_CODE_INVALID` thuộc US-P2-09 (đã có mã lỗi) |
| 4 | `TestAssignTeacher`, `…RejectsWrongRole` (8 ca: TA, SV, ADMIN, DISABLED, không có, ta_ids sai; không gán ai cả), `…ReplacesTAs` (absent / null giữ, thay, `[]` gỡ hết, dùng lại dòng), `…OldTeacherLosesAccessNow` (GV cũ 403 ngay), `…OneActiveTeacher` (5 lần xoay A↔B: luôn 1 ACTIVE, 2 dòng), `…AtomicWithOutbox`, `…NoRenotify`, `…Audit` PASS |
| 5 | `internal/integration` (relay + consumer THẬT): `TestAssignTeacherNotifies` (< 2 s), `TestAssignNotifyOnce` (xử lý lại 3 lần ⇒ 1 dòng); `internal/course`: `TestAssignNotificationTexts` (đúng tiêu đề / thân / link của GV và TA; TA không thấy mã), `TestAssignRemovedBeforeWorkerGetsNothing` PASS |
| 6 | `TestTeacherManagesOwnTAs`, `…OtherTeacherCannotManageTAs` (GV lớp khác, TA, SV ⇒ 403), `TestAssistantCandidates` (không dấu, INVITED có, DISABLED không, ≤ 20, chỉ 3 khoá), `TestChangeTeacherMidTerm` PASS |
| 7 | `TestNotificationsList` (cursor, thứ tự, 422), `…OnlyMine`, `…ReadIdempotent`, `…OtherUser404`, `…UnreadCount` PASS |
| 8 | `TestAdminCoursesRBACMatrix`: 5 thao tác × 9 tình huống = 45 ca (SV / TA / GV, TA và GV của chính lớp, JWT SV nhưng DB ADMIN ⇒ 403, không JWT ⇒ 401, ADMIN ⇒ 2xx, JWT ADMIN nhưng DB SV ⇒ không bị chặn) PASS |
| 9 | `TestAdminCoursesList` (đủ 12 khoá, lọc status / semester / q không dấu / tiền tố mã), `…NoJoinCodeNoStudents`, `…NoNPlusOne` (1 câu SQL / 25 lớp), `…Cursor` PASS |
| 10 | `class-join.spec.ts` 'admin courses page' ×3: 5 cột, đúng 1 `primary`, khung tại chỗ (không dialog), dòng tĩnh "Đã mở lớp. Đã gửi thông báo phân công cho Trần Văn Giảng.", bấm đúp ⇒ 1 POST, hộp xác nhận chỉ xuất hiện khi lưu trữ với đúng câu, rỗng "Chưa có lớp nào. Mở lớp đầu tiên.", 375 px `ox ≤ 0` + `TOUCH_SRC` sạch PASS. Phần `@real` (đếm `count(*)`): chạy tay |
| 11 | 'bell real' ×2 (chấm theo `unread_count`, mở khung không xoá chấm, "Phân công lớp · 5 phút trước", `page.clock.runFor(31 s)` ⇒ thông báo mới, bấm ⇒ POST read + chuyển trang + chấm giảm; lỗi ⇒ "Chưa tải được thông báo." không chặn app), `time-ago.spec.ts` (ngày lịch UTC+7). `grep -rnE '[0-9]+ (ngày|giờ|phút) trước' frontend/src | grep -v shared/lib/timeAgo` → rỗng (đổi 2 chỗ: fixture `DevData` dùng `timeAgo`, câu chú thích `mock/derive.ts`) |
| 12 | 'admin courses errors': mã trùng tại ô + giữ chữ, 422 theo ô, 503 ⇒ `Gửi lại` cùng `Idempotency-Key`, sai version đúng câu + hai nút (giữ chữ, gửi lại với version mới), lưu trữ lớp đã lưu trữ ⇒ `role=alert` "Lớp này đã được lưu trữ." PASS |
| 13 | `TestOpenCourseAssignThenTeacherSeesCode` (worker thật: GV thấy chuông + `Mã tham gia: <mã>.` + `…/join/<mã>` ≤ 60 s; đọc ⇒ `unread_count` 0; `/admin/courses/{id}/(students|members|enroll)` ⇒ 404 / 405, 0 sinh viên được thêm); `grep -rnE "POST.*(admin/courses/.*(students|members|enroll))" api/openapi.yaml` → 0 PASS |

## Lệch spec / nợ
- **#8** trong `proposals.md`: AC11 "URL = `link`" vs SRS 4.9 (`?course=` được tiêu thụ khỏi URL): test khẳng định `pathname` + bộ chọn lớp.
- AC10 "Drawer (không modal)": theo #5 đã duyệt (khung `Section` tại chỗ).
- `changed` của `assign` SRS chỉ ghi tên khoá: `teacher` (bool), `added_ta` / `removed_ta` (mảng id).
- #4 (PM ACCEPTED, điều kiện Origin / Sec-Fetch-Site): chỉ `POST /auth/refresh` và `POST /auth/logout` ĐỌC cookie `ep_rt`, cả hai đã qua `cookieGuard`; story sau thêm endpoint đọc cookie phải dùng cùng guard.
- `course.changed` / `today.invalidate`: US-P2-11.
- Chưa làm: `curl` tay trên compose (AC4 `GET …/members` ⇒ 403 chờ US-P2-09), phần `@real`.

## Kết quả cổng (đã chạy)
- `make lint sqlc-check`: 0 issues ×3, `sqlc diff` rc=0. `make test` (`-race`, mọi tag) xanh, gồm contract (51 thao tác) và `internal/integration`.
- Frontend: eslint + tsc sạch, `ui-antipatterns` 0 ✗, `lint-selftest` 7/7 + 19/19; Playwright (không kể `visual`) kết quả ở cuối báo cáo PM.
