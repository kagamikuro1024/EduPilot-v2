# QC report — US-P5-06 — **chấm lại TC-17 theo #38** (2026-10-16, `2e18f7d6`)
Theo `proposals.md` #38 (ACCEPTED, quyết định sprint 5): LCP chỉ chấm ở `lighthouserc.devtools.json` (throttle thật, ngưỡng `error` 2.500 ms, trung vị); lượt simulate không chấm LCP. Số 2.707 ms của lượt simulate ở lần chạy trước **không áp dụng**.
- **CI** (`38782040813`, commit `0f9ff865`, xanh): job `Frontend Lighthouse (devtools B)` chạy `/me` 3 lượt, "Checking assertions against 4 URL(s), 12 total run(s)": không có lỗi `largest-contentful-paint` (chỉ một cảnh báo `warn`, ở `/dev/ui` theo cấu hình); CI không in con số vì bước tóm tắt chỉ chạy khi lỗi, và không có artifact (chỉ tải lên khi `failure()`).
- **QC chạy lại cùng cấu hình** (`pnpm build:gate`, `lhci autorun --config=lighthouserc.devtools.json --collect.url=http://localhost:3310/me`, Chrome headless shell của Playwright, macOS arm64, 3 lượt): LCP **1.492 / 1.478 / 1.446 ms, trung vị 1.478 ms** ≤ 2.500 ms; assertion xanh. JS 242,8 KB (≤ 250 KB), a11y 0 vi phạm (từ lượt trước).
## Kết luận TC-17: **PASS** — story US-P5-06 **PASS** (26/26 TC + E1, E2)

---


# QC report — US-P5-06 (`/me`: chuyên cần, điểm cộng, thời gian học; không gamify; `/settings`)
Ngày 2026-10-15 (tối). Commit kiểm: `sprint/7-p4-p5` @ `28ab4fb7`. Bộ TC: `tc-US-P5-06.md` (không sửa chữ TC). Handoff: `dev-US-P5-06.md` (#32 #33 #34). CI `a908fd9` đỏ ở `Frontend Lighthouse (simulate)` (`total-blocking-time`), route không rõ trong log. Dữ liệu thử: lớp `QC risk` (`108503`); `sv05` = "B" (vắng 6/8, 3 lần phát biểu = +0,75 do QC thêm bằng SQL, `note` chứa canary).

## (Đã thay bằng kết luận ở đầu tệp) Kết luận lần trước: **FAIL chỉ vì TC-17 (Lighthouse `/me` không đo được)** — 25 TC còn lại PASS, 0 lỗi chức năng
`lighthouserc.json` không có route `/me` nên `pnpm exec lhci autorun` không phủ AC6 (LCP, JS). QC đo thay: `/me` ở 375 px nạp **≈ 230 KB** JS nén (`encodedBodySize`, 3 lần) — dưới 250 KB; LCP không lấy được qua `PerformanceObserver`. Theo luật "không kiểm được = FAIL"; PM thêm `/me` vào lhci hoặc chấp nhận số đo thay thế thì đổi thành PASS.

## Kết quả TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | `…/me/attendance`: `{attendance:{recorded 8,present 2,late 0,excused 0,absent 6,sessions:[{session_no,starts_at,status}]},next_cursor}` mới nhất trước (buổi 8 → 1); `…/me/participation`: `{speak_count 2,points_total "0.50",events:[{date,type,points}]}`; khớp DB (6 vắng; Σ điểm 0,50); không `note`/`marked_by`/`version`/`created_by`/canary (grep 0) |
| 02 | PASS | `?student_id=` / `?student=` (uid của sv13 vắng 0) bị bỏ qua: vẫn trả số của chính người gọi (6) |
| 03 | PASS | lớp mới: `recorded 0, sessions []`, `speak_count 0, events []`; PENDING, REMOVED, ngoài lớp → `403` |
| 04 | PASS | TA, TEACHER, ADMIN × hai route → `403` ×6 |
| 05 | PASS | `limit=3` đi hết 3 trang: buổi 8…1 không trùng, không sót; `limit=0` / `cursor=zzz` → `422` |
| 06 | PASS | crawl `sv04` (AT_RISK, canary, đã có ghi chú): `me/today`, `me/settings`, `notifications`, `me/courses`, `…/me/study-time`, `…/threads`, chat — 0 `CANARY-OBS`/`"risk`/`observation`/"Cần chú ý:"/"Đang theo dõi:"; (SSE `/events` 30 s không chạy riêng; `TestStudentCrawlNoCanary` 12 route xanh) |
| 07 | PASS | UI `sv04`: `/`, `/me`, `/chat`, `/threads`, `/settings`, `/calendar` — 0 câu nhận định, 0 canary, 0 "rủi ro" |
| 08 | PASS | `/me` 1280: "Kết quả của tôi" + mã lớp; "Chuyên cần — Bạn có mặt 2/8 buổi, vắng 6." + danh sách buổi vắng (ngày, buổi); "Điểm cộng đã ghi — Phát biểu 3 lần · +0,75." + sự kiện gần đây; dấu phẩy thập phân. Khối Thời gian học ẩn vì sv05 không có dữ liệu (xem TC-14) |
| 09 | PASS | không có điểm thành phần / giải trình / "nếu cuối kỳ" / việc sắp tới có điểm / nhãn mô phỏng |
| 10 | PASS | 0 `<progress>`/`role=progressbar`; 0 từ gamify trong `innerText` và trong `main` (6 khớp regex nằm trong CSS class `AppShell-…badge` của chuông thông báo, không hiển thị); không thẻ KPI |
| 11 | PASS | `/me` +0,75; chat "Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?" trả cùng số (`points_total: 0.75`) |
| 12 | PASS | thêm 1 phát biểu cho B: `…/me/participation` 2 → 3, `"0.75"` ngay lập tức; header `Cache-Control: no-store` (Q-QC-P5-12 chấp nhận "tức thì") |
| 13 | PASS | `{weeks,this_week_minutes,tracking:{enabled,locked}}` của chính mình; `[]` khi chưa có dữ liệu |
| 14 | PASS | không dữ liệu → khối ẩn (không "0 phút"); "Đang tắt": e2e `me study time off shows state` + `tracking.enabled=false` đã kiểm ở API (US-P5-03 TC-34/43) |
| 15 | PASS (e2e) | e2e `me study trend small no judgement` xanh; không có sinh viên "tuần thấp" trong dữ liệu thử |
| 16 | PASS | 375×812: không tràn ngang; không vùng chạm < 44 px trong `main`; một cột |
| 17 | **FAIL** | a11y: `a11y.spec` 140 lượt quét, critical/serious/moderate/minor 0 — PASS; Lighthouse: xem kết luận (lhci không phủ `/me`; JS ≈ 230 KB) |
| 18 | PASS | `page.route` 500: "Không tải được kết quả của bạn. Dữ liệu của bạn không bị mất." + `Thử lại`; "Chưa có buổi nào được điểm danh.": e2e `me states`; offline giữ nội dung: e2e `me offline keeps content` |
| 19 | PASS (khác chữ TC, theo #34) | teacher, ta, admin mở `/me`: URL giữ `/me`, màn "Bạn không có quyền xem màn này · Trang này dành cho sinh viên." + nút `Về Hôm nay` (không chuyển hướng tự động — PM chốt #34) |
| 20 | PASS | e2e `me two courses isolated` + API: số theo từng lớp, lớp mới 0 |
| 21 | PASS | `grep -rn "@/mock" frontend/src/features/me` = 0; `pnpm build:gate` xanh; `pnpm lint` sạch |
| 22 | PASS | `/settings` `sv.kha` (hai+ lớp): một dòng mỗi lớp, nhãn mã lớp + công tắc; lớp khoá bật + `disabled`, tooltip "Lớp này luôn ghi thời gian học" ở chính công tắc, không dòng mô tả; lớp cho tắt đổi được, lạc quan không toast (đã chạy ở US-P5-03; e2e `study time opt out on allowed course`, `…locked on strict course`) |
| 23 | PASS | `tracking.enabled=false` ở lớp cho tắt, lớp khoá vẫn `true`; sai `version` → `409` + `InlineNotice` (US-P5-03 TC-34/35) |
| 24 | PASS | chưa ack: nhóm "Thời gian học" không hiện (e2e `study time group hidden before acknowledgement`); nhân viên không thấy (e2e) |
| 25 | PASS | `/class/settings` 375 px: công tắc đo thời gian học lớp có sinh viên khoá, tooltip "Lớp đã có sinh viên" (kiểm ở US-P5-03 TC-41; không tràn) |
| 26 | PASS (tên khác) | `ui-antipatterns.sh` rc=0; `me.spec.ts` 9 ca xanh; hai tên `-g` trong TC (`study time switch per course`, `study time locked course`) không có trong `settings.spec.ts` — chạy tương đương `activity.spec.ts` (16 ca) xanh; tổng Playwright desktop `activity/me/panels/students/tokens/a11y/settings`: **118 passed, 0 fail, 0 flaky** |
| E1 | PASS | `/me` khi `…/me/attendance` 500 → `PageState` lỗi + `Thử lại` |
| E2 | PASS gián tiếp | REMOVED → `403` (TC-03); không duyệt tay "đang xem rồi bị xoá" |

## Lệnh của handoff
`go test -race -tags testroutes ./internal/crm -run 'TestAISummary|TestMe'` xanh (`TestMeAttendanceOwnOnly`, `…NoInternalFields`, `…EmptyCourse`, `TestMeParticipationMatchesDB`, `TestMeRoutesStaff403`, `TestMeNotCached`, `TestMeTwoCoursesIsolated`); `./internal/contract/... ./internal/httpapi/...` xanh khi chạy cả gói (xem US-P5-03 vòng 3 về `TestStudentNeverSeesConfidence` chạy riêng); `build:gate`, `me.spec`, `panels.spec`, `grep @/mock` = 0, `ui-antipatterns.sh` rc=0, `pnpm lint` sạch.

## Chưa kiểm
AC4 e2e hai ngữ cảnh "ghi phát biểu rồi `/me` + chat khớp" (handoff nói cần stack seed): QC kiểm tương đương bằng SQL + API + chat (TC-11/12). `@real` e2e bị skip ở project desktop.
