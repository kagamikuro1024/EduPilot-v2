# DEV handoff — US-P5-06 (`/me` của sinh viên: chuyên cần, điểm cộng, nhịp học)
Nhánh: `sprint/7-p4-p5`. Commit: `US-P5-06: …`. Số ghim: `#11` (contract 224 → 226: `getMyAttendance`, `getMyParticipation`); test cũ sửa theo `#32`, `#33` (ACCEPTED). Không migration mới.

## Đã làm
- **API** (`crmhttp/me.go`, `crm/me.go`): `GET /courses/{id}/me/attendance` và `…/me/participation` (con trỏ `?limit=&cursor=`). Guard `Member` + vai lớp STUDENT; danh tính chỉ từ JWT (tham số `student_id` bị bỏ qua); TA / giảng viên / admin / người ngoài lớp / PENDING → `403`. Thân: `{attendance:{recorded,present,late,excused,absent,sessions:[{session_no,starts_at,status}]}, next_cursor}` và `{participation:{speak_count,points_total,events:[{date,type,points}]}, next_cursor}`; không `note`, `marked_by`, `version`, id. `points_total` là chuỗi decimal 2 chữ số (Σ `points` ở DB, không áp trần). Lớp chưa điểm danh → số 0, `sessions: []`. `Cache-Control: no-store` (luật 15). Tái dùng `CrmStudentAttendance` / `CrmParticipationList` / hai truy vấn tổng của tool AI, nên `/me` và câu trả lời chat cùng đọc một nguồn.
- **Frontend** `features/me`: `MeScreen` → `RealMe` (nạp lười). Khối **Chuyên cần** (một câu "Bạn có mặt 7/9 buổi, vắng 2." + các buổi vắng / muộn / vắng có phép; rỗng: "Chưa có buổi nào được điểm danh." không nút), **Điểm cộng đã ghi** ("Phát biểu 2 lần · +0,50." + sự kiện gần đây), **Thời gian học** ở cuối (`TrendChart` cao 96 px + "Tuần này: N phút"; `tracking.enabled=false` → "Đang tắt"; chưa có tuần nào → khối ẩn). Điểm thành phần / giải trình / "nếu cuối kỳ được…" / việc sắp tới có điểm không hiển thị (P6, P7). Không streak, xếp hạng, vòng tiến độ. Đọc mới mỗi lần mở / quay lại tab; lỗi tải → `ApiErrorNotice` có `Thử lại`; lỗi khi làm mới giữ nội dung cũ. Phiên không có lớp: "Chưa có lớp nào.".
- **AC9 (thay mock)**: `features/me/**` không còn `@/mock/*` (`grep -rn "@/mock" frontend/src/features/me | wc -l` → 0); `mock/derive.ts` bỏ `fmtStamp` (hết nơi dùng). Các hằng còn lại của `mock/student.ts`, `mock/grades.ts` vẫn có nơi dùng khác (Assignment, Gradebook, DemoChat) nên giữ.

## Test
- Go `crm/me_test.go`: `TestMeAttendanceOwnOnly`, `TestMeAttendanceNoInternalFields` (canary note, phân trang), `TestMeAttendanceEmptyCourse`, `TestMeParticipationMatchesDB`, `TestMeRoutesStaff403`, `TestMeNotCached`, `TestMeTwoCoursesIsolated`, `TestStudentCrawlNoCanary` (canary `CANARY-OBS-4F` + AT_RISK, 12 route của sinh viên kể cả `me/*`), `TestStudentNeverSeesObservations`; `TestStudentNeverSeesRisk` (P5-04) nay cũng phủ hai route mới. Contract: `me/attendance` và `me/participation` mỗi cái 200 / 401 / 403 (giảng viên, TA, admin) / 404 / 422 (limit, cursor).
- e2e `me.spec.ts` (9 ca): `me content and layout`, `me no gamification`, `me study trend small no judgement`, `me study time off shows state`, `me states`, `me offline keeps content`, `me staff blocked`, `me two courses isolated`, `me 375`.
- Test cũ (`#32` B, `#33`): `panels.spec.ts` ba ca có `/me` (16 route Sinh viên; "/chat, /threads, … /me, /calendar"; "mobile gutter" @375 và @390) thêm `mockMe(page)` (`e2e/support/meMock.ts`); mọi khẳng định giữ nguyên.

## Chưa làm / khác spec
- AC8 "TA / TEACHER / ADMIN vào `/me` được chuyển về `/`": thực tế là màn "Bạn không có quyền xem màn này" (khung ứng dụng từ sprint PU, không phải code của story này); `me staff blocked` khẳng định hành vi hiện có và không gọi API của sinh viên. Cần PM chọn: sửa spec cho khớp hoặc thêm chuyển hướng ở `AppShell`.
- AC4 vế e2e `me matches chat after teacher records speak` (hai ngữ cảnh, chat thật + giảng viên ghi phát biểu) và `lhci` route `/me`: cần stack đã seed (US-S7-SEED); phần máy chủ đã có `TestMeNotCached`, `TestToolMyParticipation`. Lighthouse chạy ở CI.
- AC10 (`/settings` công tắc theo lớp, hai lớp khác cấu hình) đã được `activity.spec.ts` của US-P5-03 phủ; `me two courses isolated` phủ phần `/me` "Đang tắt" ở lớp khoá-đo-tắt.

## Lệnh QC chạy
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1
go test -count=1 -race -tags testroutes ./internal/crm ./internal/contract/... ./internal/httpapi/...
cd ../frontend && E2E_API_PORT=3412 pnpm build:gate
E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test me.spec.ts panels.spec.ts --project=desktop
grep -rn "@/mock" src/features/me | wc -l      # 0
bash ../scripts/ui-antipatterns.sh             # rc=0
```
