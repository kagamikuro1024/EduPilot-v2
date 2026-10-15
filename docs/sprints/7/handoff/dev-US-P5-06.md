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
- AC8: PM chốt #34 — màn chặn chung của khung ("Bạn không có quyền xem màn này" + nút `Về Hôm nay`), không chuyển hướng riêng; `me staff blocked` kiểm cả ba vai (teacher, ta, admin): URL giữ `/me`, bấm nút → `/`, không gọi API sinh viên.
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

## Vòng 2 — TC-17 (Lighthouse `/me`)
Theo PM: `/me` vào `lighthouserc.json` (simulate) và `lighthouserc.devtools.json` (cùng ngưỡng), `lighthouse-auth.cjs` (vai student), `lighthouse-api.mjs` (ba API `me/*` theo hợp đồng thật để màn dựng đủ), `ci.yml` (devtools B: `/gradebook /me /settings/llm /dev/ui`; 8 URL).
- **Lần đo đầu đỏ thật**: devtools `/me` LCP **5,5 s** (ngưỡng 2,5 s), FCP 4,1 s; `/chat` 1,5 s, `/threads` 1,5 s. Gốc: `shared/shell/intro.ts` không có `/me` nên `AuthSkeleton` rơi về khung xương (không có chữ từ máy chủ) → chữ đầu tiên chỉ vẽ sau JS + `/auth/refresh` + `/me/courses`. Sửa: thêm `"/me": { title: "Kết quả của tôi", description: "761987" }` (h1 thật là ứng viên LCP duy nhất của màn). Sau sửa: LCP **1,47 s**, FCP 0,83 s (3 lượt).
- Simulate `/me` (3 lượt): TBT 27 / 29 / 31 ms, CLS 0, JS 237 KB (ngưỡng 200 ms / 0,1 / 250 KB). Cả 8 URL của lượt simulate đều qua cục bộ (TBT 15–38 ms).
- `e2e/lcp.spec.ts`: thêm ca `/me` (dùng `mockMe`), 7/7 xanh — cần ACCEPTED #37 (xem `proposals.md`).
- **TBT simulate đỏ ở CI (run `38767723328`, `/`: 40 / 274 / 312 ms, trung vị 274)**: không tái hiện cục bộ (16–24 ms). Log "benchmark máy chạy" của ba lượt CI gần nhất cho thấy đuôi dài ngẫu nhiên trên cùng `/` ở runner nhanh (benchmarkIndex ≈ 3000): `489 / 50 / 44` (run xanh `38754083522`), `96 / 72 / 69`, `40 / 274 / 312`. Mười hai lượt CI gần nhất: simulate đỏ 5 lần, trong đó 4 lần ngày 13/10 trước P5. Không do mã P5. Giảm nhiễu: `numberOfRuns` simulate 3 → 5 (trung vị chịu được 2 lượt xấu thay vì 1); ngưỡng 200 ms giữ nguyên. Đây là giảm nhiễu, không có bằng chứng đo trước/sau; nếu vẫn đỏ thì phải đo nguyên nhân tác vụ dài trên runner.
