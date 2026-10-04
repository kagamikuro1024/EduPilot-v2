# DEV handoff — US-P2-07 (CourseAccessGuard thật, lớp của tôi, hồ sơ, tuỳ chọn, bộ chọn lớp)
Nhánh `sprint/4-p2`. Chưa kiểm trên stack compose thật (xem "Chưa làm").

## Làm gì
**Backend**
- Migration: KHÔNG thêm — `00003_course_foundation.sql` đã có đủ 8 bảng; AC1–3 là test lược đồ / ràng buộc / chỉ mục trong `internal/store/course_test.go`.
- `auth.CourseAccessGuard(resolver, mode)` thật: `CourseResolver.Resolve(ctx, Principal, uuid) (Membership{Found, Role, Status}, error)` thay `CanAccess`; 6 chế độ `Member|Staff|Teacher|StaffOrAdmin|Manage|MemberOrAdmin`; id không phải uuid ⇒ 404; lỗi tra ⇒ 503 (kể cả ADMIN); ADMIN chỉ qua ở chế độ có "hoặc ADMIN"; vai lấy từ `enrollments` chứ không từ JWT; không cache (1 lần gọi / request); lớp không tồn tại ⇒ 403 `reason=course` giống người ngoài lớp. `course.Resolver` cài bằng MỘT truy vấn `GetMembership` theo `(course_id, user_id)`.
- `internal/course` (mới) + `internal/httpapi/coursehttp`: `GET /me/courses` (SV: ACTIVE + PENDING của mình; GV/TA: lớp được phân công; ADMIN: `items: []`; con trỏ `(enrollments.created_at, id)`, ETag/304, không `join_code`; MỘT câu SQL), `GET /courses/{id}` (`MemberOrAdmin`; SV không có `counts`/thành viên/mã; GV, TA thêm `counts{students_active,students_pending}`; `my_role`; ADMIN xem bản cơ bản, lớp không có thật ⇒ 404).
- `internal/user/profile.go` + `userhttp.MountMe`: `GET/PUT /me/profile` (email, role, id, status… ⇒ 422 trường lạ; tên 1–100; `student_code` `^[A-Za-z0-9]{6,15}$` hoặc rỗng để xoá, chuẩn hoá chữ hoa, chỉ SV; sai `version` ⇒ 409 + `current_version`; KHÔNG đụng `enrollments.student_code_snapshot`; `audit_log` một dòng không chứa MSSV), `GET/PUT /me/settings` (tạo lười `INSERT … ON CONFLICT DO NOTHING`, khoá lạ ⇒ 422, `version` ⇒ 409).
- `store.ChunksForCourse` (`courses.sql`): đường ĐỌC chunk duy nhất, luôn `course_ids @> array[$1]::uuid[]` (GIN).
- `openapi.yaml` 0.9.0 (+6 thao tác, tổng 42; tag `courses`), `contract.courseScenarios` phủ mọi status khai báo; miễn trừ duy nhất `GET /courses/{id}` 503 (`exempt.go`, kiểm bằng `TestGuardResolverError503`).

**Frontend**: `shared/session/myCourses.ts` (`useMyCourses` → `GET /me/courses?limit=100`, một trang; `mockCourseFor` ánh xạ `class_code` → lớp mô phỏng: 761987↔`int1006-1`, 761988↔`int1006-2`, lạ → lớp đầu). `session.tsx`: phiên `jwt` (không phải ADMIN) lấy lớp THẬT ACTIVE làm nguồn: `courses`/`course`/`hasCourse` suy ra từ đó, `realCourses`/`realCourseId` mới; lựa chọn lưu `localStorage` `ep:ui:course` (id hoặc `all`); `?course=<uuid>` chỉ có hiệu lực nếu là lớp của mình, và chỉ được xử lý sau khi `/me/courses` tải xong. Khi chưa tải được (đang tải / lỗi) hoặc phiên mô phỏng: giữ nguyên hành vi cũ (các ca e2e dùng `asJwt` không đổi). `AppShell`: bộ chọn liệt kê `761987 · An ninh mạng` (có `title`), "Tất cả lớp của tôi" cho GV/TA có >1 lớp, "Quản lý lớp này" (GV, TA), "Tham gia lớp bằng mã" (SV); màn "Bạn chưa vào lớp nào" thành `PageHeader` + nút `Tham gia lớp bằng mã` → `/join` (câu đúng AC12; bỏ ô nhập mã cũ, `/join` đã có ô nhập).

## AC tự đánh giá (chạy thật)
| AC | Kết quả |
| --- | --- |
| 1 | `TestCourseSchema` (8 bảng, đủ cột từng bảng, `uuid[]`/`vector(1536)`/`character(7)`, đúng 1 GIN `course_ids`, chỉ mục phức hợp bảng thuộc lớp bắt đầu `course_id`) PASS |
| 2 | `TestCourseConstraints`: 40 ca, SQLSTATE 23514/23505 (join_code 0/O/1/I/L/thường/6 ký tự, trùng class/join, semester, ARCHIVED↔archived_at, ARCHIVED+join_enabled, capacity 0/1001, domain, trùng enrollment, 2 GV ACTIVE, snapshot cho TA, REMOVED thiếu removed_at, buổi học…) + 2 lệnh hợp lệ chèn được PASS |
| 3 | `TestCourseIndexes`: 3.000 lớp, 20.000 SV + ghi danh, 3.000 thông báo, 20.000 chunk ⇒ EXPLAIN dùng chỉ mục, không `Seq Scan`, chunk `Bitmap Index Scan` trên GIN PASS |
| 4 | `auth`: `TestGuardMatrix` (4 vai JWT × 7 tình trạng × 6 chế độ = 168 ca), `…NonUUID404`, `…ResolverError503`, `…NoCache`, `…AdminDeniedByDefault`, `…NonexistentSameAsOutsider` (Q-QC-P207-1: cùng thân 403) PASS; ca `course` ở `TestGetCourseOutsider403` chạy qua DB thật |
| 5 | `integration`: `TestAllCourseRoutesGuarded` (chi.Walk + 3 vai ngoài lớp, mọi route ⇒ 403 `reason=course`), `…DetectsMissing` (route quên guard bị lộ), `TestCourseIsolation` PASS. **Ngoại lệ: `GET/PUT /courses/{id}/llm-budget` (hợp đồng ADMIN-only của s3) — proposal #7.** `today`/`members`/`join-code`/`sessions`/`notifications` chưa tồn tại: các story sau phải được bộ quét này tự bắt |
| 6 | `TestChunksForCourseIsolation` (c1/[c1,c2]/[c3]), `TestNoUnscopedChunkQuery` (mọi truy vấn đọc `content_chunks` phải chứa `course_ids`, đúng 1 đường đọc) PASS |
| 7 | `TestMeCoursesStudent/Staff/AdminEmpty/NoJoinCode/Cursor/ETag/NoNPlusOne` (1 câu SQL / 22 lớp) PASS |
| 8 | `TestGetCourseStudentView/StaffView/AdminBasic/Outsider403` PASS |
| 9 | `TestProfileGetPut/ReadOnlyFields/VersionConflict/StudentCodeDoesNotChangeSnapshot/Audit` PASS |
| 10 | `TestSettingsDefaultsLazyCreate/Put/UnknownKey422/OwnerOnly`, `TestProfileSettingsAllRoles` PASS |
| 11 | `class-join.spec.ts` 'course picker' ×6 (GV 2 lớp + "Tất cả" + "Quản lý lớp này"; SV 2 lớp; PENDING không hiện; `?course=` lạ bị bỏ; `?course=` của mình đè + `localStorage`; 'Chưa có lớp'; `AUDIT_SRC` 1440/390 ⇒ `ell: []`, `ox: 0`) PASS. `@real` seed: `test.skip` (US-P2-12 chưa có) |
| 12 | 'no course screen': 7 route cùng tiêu đề + câu + nút, không "Bạn không có quyền"; `/` và `/join` không hiện; `courseCalls` = 0 PASS |
| 13 | 'mock course mapping' ×2: GV chọn 761988 ⇒ `/inbox` hiện "An ninh mạng – 761988", 0 gọi API lớp; class_code lạ ⇒ lớp mô phỏng đầu PASS. `audit.mjs` (QC) chưa chạy lại trong repo này |
| 14 | `TestMeEndpointsRequireJWT`, `TestMeEndpointsNoUserIDParam`, `TestSettingsOwnerOnly` PASS |

## Lệch spec / nợ
- **#7** trong `proposals.md`: AC5 "mọi `{id}` dưới `/courses/` có guard" mâu thuẫn hợp đồng bất biến `GET/PUT /courses/{id}/llm-budget` (ADMIN-only, 403 `role`, 422 khi id không phải uuid; `TestRBACMatrix`, `TestBudgetPutRules`). Giữ nguyên, ngoại lệ là danh sách đóng `adminOnly` trong `TestAllCourseRoutesGuarded`.
- `GET /me/courses` ở frontend lấy một trang 100 lớp, không "tải tiếp".
- Phiên `jwt` mà `/me/courses` lỗi ⇒ khung rơi về bộ lớp mô phỏng (không chặn). Staff chưa được phân công lớp nào ⇒ `hasCourse=false` nhưng màn mô phỏng vẫn dùng lớp mô phỏng đầu (chỉ SV có màn "chưa vào lớp").
- `student_code` chỉ sinh viên đặt được (GV/TA ⇒ 422 `NOT_AVAILABLE`) — suy từ "MSSV" của SV; SRS không nói rõ.
- Ảnh mốc Linux (`visual.spec`) không chạy lại: đường mô phỏng (cookie) không đổi DOM; CI sẽ xác nhận.
- Chưa làm: `curl` tay trên stack compose (`/me/courses`, `403` lớp 2, `422` role), `@real`.

## Kết quả cổng (đã chạy)
- `make lint sqlc-check`: 0 issues ×3, `sqlc diff` rc=0. `make test` (`-race -tags integration`) xanh, gồm `internal/contract` (42 thao tác, mọi status khai báo được sinh, trừ 503 miễn trừ).
- Frontend: eslint + tsc sạch, `ui-antipatterns` 0 ✗, `lint-selftest` 7/7 + 19/19. Playwright (`--workers=2`, không kể `visual`): 252 passed / 88 skipped / 0 failed.
