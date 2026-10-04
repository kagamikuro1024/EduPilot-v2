# DEV handoff — US-P2-09 (vào lớp bằng mã, quản lý mã và thành viên)
Nhánh `sprint/4-p2`. Chưa kiểm trên stack compose thật (xem "Chưa làm").

## Làm gì
**Backend** (`internal/course`, `internal/httpapi/coursehttp`)
- `Preview` / `Join` (`POST /courses/join/preview`, `POST /courses/join`; `RequireRole(STUDENT)`, email đã xác minh): chung MỘT bước `lookupByCode` — luôn đúng một truy vấn `GetCourseByJoinCode` (mã sai định dạng tra mã không thể khớp), so sánh hằng thời gian, một quyết định gộp không rẽ nhánh sớm ⇒ sáu nguyên nhân (sai / cũ / tắt / hết hạn / lưu trữ / sai tên miền) cho cùng 404 `JOIN_CODE_INVALID`. `COURSE_FULL` (409) và `PENDING` là hai trạng thái duy nhất khác biệt.
- Giới hạn đoán mã (`guesslimit.go`): hai ZSET Redis `ep:join:fail:user:{uid}` (5 / 10 phút) và `ep:join:fail:ip:{ip}` (20), cửa sổ trượt bằng Lua (`ZREMRANGEBYSCORE` + `ZCARD`), chỉ ghi THẤT BẠI `JOIN_CODE_INVALID`, kiểm TRƯỚC khi tra mã (mã đúng cũng 429, `retry_after` = giây còn lại). Redis lỗi ⇒ mở cửa + log `error` ≤ 1 lần / 30 s. Dùng đồng hồ `platform/clock` (giả được trong test). Mã thử không vào log.
- `Join`: khoá dòng `courses` `FOR UPDATE` rồi `LockEnrollment`: ACTIVE / PENDING sẵn có ⇒ trả nguyên trạng `already_member=true` (50 yêu cầu song song ⇒ một dòng, sĩ số không đếm đôi); đầy ⇒ `COURSE_FULL`; `require_approval` hoặc MSSV tự khai trùng snapshot của người KHÁC đang ACTIVE / PENDING ⇒ `PENDING` (+ `warning=EMAIL_MISMATCH`) bất kể lớp có bật duyệt; REMOVED ⇒ CÙNG dòng về PENDING (snapshot cũ giữ nguyên). `audit_log` một dòng (không email / MSSV / mã), outbox `course.join_requested`.
- `Regenerate` (`POST …/join-code/regenerate`, `Manage`): một `UPDATE` thay mã (mã cũ chết ngay), thử lại ≤ 5 lần khi trùng, audit chỉ 2 ký tự đầu của mã cũ / mới; `GET …/join-code` (`StaffOrAdmin`); `PUT …/join-settings` (`Manage`, `version`; hạn trong tương lai ≤ 366 ngày, sĩ số ≥ số đang học ≤ 1.000, tên miền chữ thường; khoá vắng = giữ nguyên, null / rỗng = bỏ).
- Thành viên: `GET …/members` (MỘT truy vấn gồm cả `counts`, kể cả trang rỗng; `(status_changed_at DESC, user_id)`; `q` tên không dấu / MSSV / tiền tố email), `approve|reject` (`StaffOrAdmin`), `DELETE` (`Manage`), `undo` (≤ 60 s theo `platform/clock`, cùng quyền với hành động gốc: mời ra chỉ giảng viên / Admin). Hàng `EMAIL_MISMATCH`: TA ⇒ 403 `mismatch_needs_teacher`; giảng viên thiếu `confirm_mismatch` ⇒ 422. Duyệt / hoàn tác mời ra kiểm lại sĩ số ⇒ `COURSE_FULL`. Giảng viên / TA đổi bằng assign ⇒ 422 ở các đường này.
- Worker: `course.join_requested` ⇒ `JOIN_REQUEST` cho giảng viên VÀ TA ("{tên} xin vào lớp {mã}", link `/class/members?course=…&tab=pending`; thân của GIẢNG VIÊN thêm "— email chưa khớp MSSV", TA thân thường); `course.join_decided` ⇒ `JOIN_APPROVED` / `JOIN_REJECTED`. Bỏ qua yêu cầu không còn chờ. Đăng ký ở `cmd/worker/registry.go`.
- `httpapi.OptionalIdempotencyKey` (có `Idempotency-Key` thì phát lại, không thì chạy): dùng cho `join` và `admin/courses/{id}/assign`.
- openapi 0.11.0 (+10 thao tác, tổng 61), `contract.courseJoinScenarios` phủ mọi status khai báo. Caddy che `/join/<mã>` thành `/join/REDACTED`; Next gửi `Referrer-Policy: no-referrer` + `Cache-Control: no-store` cho `/join` và `/join/:path*`.

**Frontend**: `/join` và `/join/[code]` thật (`features/join/JoinScreen.tsx`: ô nhập tự viết hoa / bỏ `0 O 1 I L`, một nút chính, xem trước "An ninh mạng · 761988 · TS. … · HK1 2026–2027", các trạng thái cần duyệt / đầy / đã vào / chờ, đếm ngược "Thử lại sau 07:41.", link xác minh email; `/join/<mã>` xem trước ngay rồi `history.replaceState` về `/join`), `/class/settings` (mới, `ClassSettings.tsx`: mã lớn, sao chép mã / liên kết, `Tạo lại mã` không primary + `ConfirmIrreversible` nêu số, form `Lưu cài đặt` một nút chính, lỗi tại ô, xung đột version; TA chỉ đọc), `/class/members` (`MembersView.tsx`: tab Thành viên / Chờ duyệt (N) / Trợ giảng (GV), tìm, duyệt / từ chối từng hàng và hàng loạt `Duyệt N`, mời ra, mọi thao tác lạc quan + `UndoLine` gọi `undo`, hàng mismatch có nhãn và xác nhận ngắn, `?tab=` mở đúng). `useCursorList` nhận `idOf`; `useClassCourse()` chọn lớp giảng viên / TA đang làm việc từ lớp THẬT. Mock `JoinPreview` bị xoá; hằng `JOIN_SENT_KEY` chuyển sang `StudentHome` (mô phỏng, US-P2-11 thay).

## AC tự đánh giá (chạy thật)
| AC | Kết quả |
| --- | --- |
| 1 | `TestJoinPreviewShape` (đúng 5 khoá, 5 trạng thái, không mã / id / sinh viên, không ghi danh), `TestJoinOpen`, `…CodeCaseInsensitive` (chữ thường + khoảng trắng), `…SnapshotStudentCode` (đổi MSSV sau không đổi snapshot), `…Audit` PASS |
| 2 | `TestJoinTwiceOneEnrollment` (+ `Idempotency-Key` phát lại), `TestJoinConcurrentOneEnrollment` (50 goroutine ⇒ 1 dòng, 1 yêu cầu tạo mới), `integration.TestJoinConcurrentTenRequests` PASS |
| 3 | `TestJoinUniformFailure` (6 nguyên nhân × preview / join, cùng thân sau khi bỏ `trace_id`), `TestJoinFailureTimingEqualized` (trung vị 20 mẫu xen kẽ, lệch ≤ 35 %), `TestJoinFullIsDistinct` PASS |
| 4 | `TestJoinRateLimit5Per10Min` (5 sai ⇒ mã đúng 429, `retry_after` ≈ 600; sau 9 phút còn ≈ 60; sau 10 phút vào được), `…PerIP` (20 người, mỗi người sai 1 lần), `…OnlyFailures` (8 `COURSE_FULL` + 25 thành công không đếm), `…SharedAcrossInstances`, `TestJoinAttemptNotLogged`, `TestJoinGuessSimulation` (100 mã: 0 trúng, 95 yêu cầu 429 sau lần thứ 5) PASS |
| 5 | `TestRegenerateOldCodeDeadImmediately`, `…NewCodeWorks`, `…ExistingMembersUnaffected`, `…TAForbidden` (TA / GV lớp khác / SV 403, lưu trữ 409), `…AuditNoFullCode` PASS |
| 6 | `TestJoinSettingsGetShape`, `…Validation` (7 ca + xoá bằng null, version 409), `…EffectDisabled` / `Expired` / `Domain` / `Capacity` / `Approval`, `…TAForbidden` PASS |
| 7 | `TestJoinRequiresApprovalPending`, `TestPendingHasNoAccess` (4 route 403 `reason=course`), `TestJoinRequestNotifiesStaff`, `TestPendingResubmitNoDuplicate` PASS |
| 8 | `TestApproveReject`, `TestRemoveTeacherOnly`, `TestApproveRechecksCapacity`, `TestUndoWithin60s`, `TestUndoExpired` (đồng hồ giả), `TestDecisionNotifiesStudent`, `TestCannotRemoveStaffViaMembers`, `TestMemberActionsAudit` PASS |
| 9 | `TestRemovedLosesAccessImmediately`, `TestRejoinReusesRowPending` (cùng id dòng, snapshot cũ), `TestRemovedKeepsLearningData` PASS |
| 10 | `integration`: `TestJoinStudentCodeCollisionPending` (worker thật: GV có "— email chưa khớp MSSV", TA không), `TestMismatchApprovalNeedsTeacher`, `TestMismatchTAForbidden` (kể cả có cờ), `TestMismatchReverseOrder` PASS |
| 11 | `TestJoinUnverified403`, `TestJoinNonStudent403` (Admin / GV / TA), `TestJoinNoJWT401` PASS |
| 12 | `TestMembersAndJoinRBACMatrix`: 8 thao tác quản lý × 8 vai / quan hệ + preview / join × 8 = 80 ca PASS |
| 13 | `TestMembersListCursor` (thứ tự ổn định qua trang), `…Filters`, `…MinimalFields` (đúng 9 khoá, không hash / ics / điểm), `…NoNPlusOne` (2 câu SQL gồm guard, không phụ thuộc số dòng) PASS |
| 14 | `class-join.spec.ts` 'join page' ×4, 'join link login redirect', header `no-referrer` / `no-store`: ô nhập, 1 nút chính, các chuỗi, đếm ngược 07:4x, link xác minh, `/login?next=%2Fjoin%2FBX4P9TW` rồi về xem trước và URL thành `/join`, 375 px `ox ≤ 0` + `cut` + `TOUCH_SRC` sạch PASS. `grep` log Caddy / frontend: chạy tay (chưa có compose) |
| 15 | 'class settings page' ×4 (mã, sao chép có clipboard thật, hộp xác nhận đúng câu, lưu một nút chính, lỗi tại ô, xung đột version, TA chỉ đọc, SV bị chặn và 0 yêu cầu `join-code`) PASS |
| 16 | 'class members page' ×6 (chọn nhiều ⇒ `Duyệt 3`, `Hoàn tác` ⇒ 3 `undo`; mời ra + hoàn tác; mismatch xác nhận ngắn + `confirm_mismatch`; TA không có nút Duyệt; tab Trợ giảng thêm / bớt; rỗng; 375 px sạch; SV bị chặn) PASS |
| 17 | `@real join flow end to end`: `test.skip` (cần compose + seed US-P2-12); các bước tương đương chạy được ở AC2–AC4, AC7–AC9 và các ca e2e trên |

## Lệch spec / nợ
- **#10** trong `proposals.md`: AC13 tìm MSSV vs 4.2.5 — `ListMembers` thêm vào danh sách trắng của `TestNoQueryLinksByStudentCode` (truy vấn MSSV-trùng đặt đúng tên spec `EnrollmentConflictByStudentCode`).
- **#9** trong `proposals.md`: `CHECK notifications_link_chk` (`^/[^/\\]`) của 00003 không nhận link `/` nên `JOIN_APPROVED` lưu `link` NULL; chuông mở `/` khi không có link.
- `already_member` cũng là `true` khi gửi lại yêu cầu đang PENDING (SRS chỉ nêu "các lần sau"); `status` phân biệt.
- `JOIN_REQUEST` chỉ sinh khi ghi danh thành PENDING (lớp không bật duyệt mà vào ACTIVE thì không có thông báo).
- Hoàn tác một lần duyệt không rút lại thông báo `JOIN_APPROVED` đã gửi (hiếm: hoàn tác ≤ 60 s).
- `course.member_changed` / `today.invalidate`: US-P2-11 thêm cùng handler (hiện chưa ghi để không sinh dead-letter).
- Sổ sách kỹ thuật: kiểm thời gian ≤ 35 % chạy ở mức service (không Redis) để không dính giới hạn; vẫn có thể dao động khi máy rất tải.
- Chưa làm: `curl` tay trên compose (AC3 `sort -u`, AC4, AC12, grep log Caddy), `@real`.

## Kết quả cổng (đã chạy)
- `make lint sqlc-check`: 0 issues ×3, `sqlc diff` rc=0. `make test` (`-race`, testroutes + integration) xanh.
- Lỗi hạ tầng test gặp và xử lý: `TestCourseIndexes` (20.000 dòng) xin 32 MB `/dev/shm` của Postgres dùng chung (64 MB) làm gói khác lỗi "could not resize shared memory segment" ⇒ tắt worker song song cho phiên đó. Bộ đếm đoán mã theo IP tồn tại 10 phút ở Redis dùng chung nên kịch bản contract phải gửi `X-Forwarded-For` riêng cho `/courses/join*`.
- Frontend: eslint + tsc sạch, `ui-antipatterns` 0 ✗, `lint-selftest` 7/7 + 19/19; Playwright (không kể `visual`, `--workers=2`): 291 passed, 0 failed.
