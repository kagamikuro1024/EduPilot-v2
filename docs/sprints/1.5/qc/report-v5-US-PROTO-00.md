# QC report v5 — US-PROTO-00 (nền: shell, vai, đổi vai, AUDIT, TOUCH, LEFT, HEADER, 404, `ep_demo_state` hỏng)  · Kết luận: **FAIL**

Nhánh `sprint/1.5-mock-ui` · HEAD `8bb6861` (trên `89dfe38`) · ngày 01/10/2026 · server `http://localhost:3400` (`next start`, không build lại).
TC: `docs/sprints/1.5/qc/tc-US-PROTO-00.md` — **69 TC** (TC-00-01…TC-00-69).
Công cụ đã dùng: **C** `evidence-v5/proto-curl.log` · **A** `evidence-v5/audit.json`, `audit-fails.json`, `v24.json` · **B** Chrome for Testing 150 headless, profile riêng `/tmp/qcv5-v5q00`, Puppeteer qua `browser` (chạy lại AUDIT/TOUCH/LEFT nguyên văn của US.md trong trang). Ảnh: `docs/sprints/1.5/shots/qc-v5/v5q00/`.

**Tóm tắt:** 62/69 TC PASS; 7 FAIL — TC-00-47, TC-00-52, TC-00-58, TC-00-59, TC-00-62, TC-00-63, TC-00-69.

## Cổng đã chạy

| Lệnh / script | Kết quả |
| --- | --- |
| `pnpm -C frontend lint` | exit 0 (`eslint .`, không cảnh báo) |
| `pnpm -C frontend exec next build` (chạy trong `git worktree` tạm `/tmp/qcv5-build` ở HEAD để **không** đụng `.next` của server đang chạy) | **exit 0**, 32 route dựng xong |
| `bash scripts/ui-antipatterns.sh` | exit 0 — 11 mục ✓, 0 ✗ |
| Thử đỏ: thêm `backdrop-filter: blur(4px)` vào `shared/shell/AppShell.module.css` trong worktree | script **đỏ** (✗ "Hiệu ứng bị cấm", ✗ "Nền trong suốt / kính mờ"), exit 1; hoàn lại → exit 0 |
| `grep -rnE '#[0-9a-fA-F]{3,8}' frontend/src --include=*.css \| grep -v tokens.css` | 0 dòng |
| `proto-curl.log` (máy chạy sẵn) | 497 PASS / 0 FAIL; phần `tc_00_*` = 219 dòng PASS |
| AUDIT ma trận đầy đủ (tôi chạy tay): 4 vai × 31 route × {1440, 390} + `/login` × {1440, 390, 375} | 259 phép đo, **mọi dòng `{ox:0, cut:[], ell:[]}`**, 0 lỗi console |
| AUDIT `?state=empty|error` × mọi route mở được của 4 vai × {390, 375} | 228 phép đo, mọi dòng `{ox:0, cut:[], ell:[]}` |
| TOUCH trong 7 trạng thái tương tác ở 375 | mọi dòng `[]` |

## TC

| TC-id | PASS/FAIL | Cách kiểm | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-00-01 | PASS | C | `proto-curl tc_00_01`: `/`, `/inbox`, `/chat`, `/gradebook`, `/me` → `/login`; cookie vai rác → 307, không 5xx |
| TC-00-02 | PASS | B | `/login` sạch cookie + localStorage: 4 vai (Sinh viên / Trợ giảng / Giảng viên / Admin); chọn Sinh viên → A·Nguyễn Minh Trung, B·Trần Thu Uyên, C·Lê Quang Huy, D·Phạm Ngọc Linh, mỗi người 1 dòng mô tả (chữ màn thứ hai đã đối chiếu đủ 4 người). Ảnh `001-login-1440.webp` |
| TC-00-03 | PASS | B | Chọn B → `/`; nút hồ sơ `aria-label="Tài khoản: Trần Thu Uyên"`; dải "Bản mô phỏng · dữ liệu giả" 12 px (body 15 px), màu `lab(51.151 4.287 2.31)` = xám, **không đỏ**; ở 390 nằm ngay dưới thanh trên (top 56) ở cả 4 vai (đo cả bốn). Ảnh `003-sv-b-home-1440.webp`, `090-banner-student-390.webp` |
| TC-00-04 | PASS | C | `tc_00_01`: dải mô phỏng ở `/` của student/ta/teacher/admin đều ≥ 1 |
| TC-00-05 | PASS | B | `document.cookie` sau khi chọn vai: `ep_demo_role=student`, `ep_demo_person=sv-2`, `ep_demo_course=int1006-1` |
| TC-00-06 | PASS | B | SV B gửi D3 ở `/chat` → "Đang chờ giảng viên"; menu hồ sơ → `Giảng viên · Lê Thu Hà` đổi tại chỗ (URL `/`, không tải lại tay); `/inbox` GV có phiếu "Thi cuối kỳ… tờ A4…", tab `Đang chờ` 5 → 6, `Tất cả` 6 → 7. Ảnh `050-chat-D3-sv.webp`, `051-inbox-GV-co-D3.webp` |
| TC-00-07 | PASS | B | Menu hồ sơ → `Đặt lại dữ liệu demo`: phiếu D3 biến mất (`Đang chờ` về 5, `Tất cả` 6), `/gradebook` về "công thức quá trình 40% · cuối kỳ 60% · đã xác nhận", 30 SV |
| TC-00-08 | PASS | B | localStorage chỉ một khoá `ep_demo_state`; giá trị không chứa `eyJ` / `Bearer` / `token`. Sau `Đặt lại`, khoá bị xoá rồi app ghi lại **chỉ** mốc đồng hồ `{"clock":{"t0":…}}` (không còn trạng thái demo) — ghi nhận để PM biết khoá không ở trạng thái "không tồn tại" |
| TC-00-09 | PASS | C/B | lint exit 0 · build exit 0 (worktree `/tmp/qcv5-build` ở HEAD) · `ui-antipatterns.sh` exit 0, 0 `✗` (bảng Cổng) |
| TC-00-10 | PASS | C | `tc_00_static`: không `fetch(` ngoài `src/shared`; không thêm phụ thuộc ngoài `lucide-react`; localStorage chỉ `ep_demo_state`; không token; không màu cứng |
| TC-00-11 | PASS | B | Bộ chọn lớp GV 1440 có: "An ninh mạng – 761987", "An ninh mạng – 761988", "Tất cả lớp của tôi", "Quản lý lớp này" (+ "Mời trợ giảng"). Đổi lớp → `/students` 30 SV·761987 ↔ 24 SV·761988, `/gradebook` "Thứ Năm 09:00–11:30" ↔ "Thứ Ba 07:00–08:30". Ảnh `021-chooser-gv-1440-open.webp` |
| TC-00-12 | PASS | B | A: 761987 + 761988; B và C: chỉ 761987; D: nút ghi "Chưa có lớp", popover chỉ "Tham gia lớp bằng mã" |
| TC-00-13 | PASS | C | `tc_00_04`: SV D "Hôm nay" có ô nhập "mã tham gia" ≥ 1 |
| TC-00-14 | PASS | C | `tc_00_05`: 13 route `?state=error` đều có `Thử lại`; `/inbox?state=empty` có "Không còn câu hỏi" |
| TC-00-15 | PASS | B | 9 route `?state=loading` (SV `/threads`,`/me`,`/library`; GV `/inbox`,`/students`,`/gradebook`; Admin `/admin/users`,`/observability`,`/`): chỉ khối skeleton theo hình vùng nội dung, **0 phần tử tròn quay giữa trang** (lọc phần tử `animation-iteration-count: infinite` + bo tròn ≥ 50 % → rỗng). Ảnh `030-loading-*.webp` |
| TC-00-16 | PASS | B | `?state=error` → `Thử lại` bỏ tham số, về trạng thái thường (SV `/library`, GV `/inbox`, Admin `/admin/users`). Câu lỗi nêu vấn đề + khắc phục, không từ kỹ thuật với SV: "Không tải được nội dung này. Kết nối vẫn được giữ và chữ bạn đã nhập không bị mất. Thử lại, hoặc quay lại sau ít phút." Ảnh `031-error-student-_library.webp`, `031-error-teacher-_inbox.webp` |
| TC-00-17 | PASS | C | `tc_00_06`: 16 lệnh `open_as` đúng CHAN/MO như US |
| TC-00-18 | PASS | C | `tc_00_06`: màn chặn có "Bạn không có quyền mở trang này", một câu lý do theo vai, nút `Về Hôm nay`, không lộ nút của `/inbox` |
| TC-00-19 | PASS | C | `tc_00_matrix`: 124 ô (4 vai × 31 route) đúng; TA `/gradebook` MO, GV `/observability` MO |
| TC-00-20 | PASS | B | `nav a` hiện ra: SV 7 (1440) / 4 (390), TA 12 / 4, GV 15 / 4, Admin 6 / 4 — không route nào ngoài quyền; ⌘K: SV 7 mục, TA 12, GV 15, Admin 6, không mục bị chặn |
| TC-00-21 | PASS | B | GV ở `/attendance` → đổi vai SV B: về `/`, h1 "Chào Uyên", không còn chữ "Điểm danh buổi / Vắng phép / + Phát biểu". Ảnh `060-doi-vai-tu-attendance.webp` |
| TC-00-22 | PASS | A | `audit 00-AC7 brand` ×4 vai @1440: `{img:32, brandH:56, bBottom:56, hBottom:56, border:"1px", ratio:4.7}` (188:40 = 4,7) |
| TC-00-23 | PASS | A | `00-AC7 thu gọn` GV 1440: `{w:32,h:32,cx:36,colW:72,colCx:36}`. Ảnh `101-sidebar-thu-gon-1440.webp` |
| TC-00-24 | PASS | A/B | `00-AC7 topbar` SV + GV @390: `{img:[28,28], tap:[44,44], before:true}`; bấm mark ở `/inbox` → về `/` (đo tay) |
| TC-00-25 | PASS | A | `00-AC7 logo /login`: 40 @1440, 32 @390, 32 @375, `ox:0` cả ba |
| TC-00-26 | PASS | B | 414 và 375, 4 vai: mark 28×28, ảnh gốc 40×40 (không méo), link 44×44, `aria-label="EduPilot — Hôm nay"`; đổi route (`/`,`/chat`,`/threads`,`/me`), đổi vai và `Đặt lại dữ liệu demo` → vẫn 28×28 tại x=16, y=14 |
| TC-00-27 | PASS | A | `00-AC8 chooser` GV 1440 ở `/`, `/inbox`, `/gradebook`: text "761987 · An ninh mạng" đủ, w=198 ≤ 360, ô tìm nhanh `full:true, clipped:false`, `ell:[]` |
| TC-00-28 | PASS | A | `00-AC8 chooser ≤240` @1024/1099/720: w=198, `trunc:false`, `title` + `aria-label` = "761987 · An ninh mạng"; `ell:[]` |
| TC-00-29 | PASS | A | 1100: đủ tên, w=198 ≤ 360 · 1099 và 720: w=198 ≤ 240 · 719: chỉ "761987" (w=90) + mũi tên |
| TC-00-30 | PASS | A/B | GV 390: nút "761987"; popover "An ninh mạng – 761987", `cut:[]`, `ox:0` |
| TC-00-31 | PASS | A/B | SV A (390/375) popover đọc đủ cả hai lớp (`bộ chọn lớp mở (SV A, 2 lớp)` PASS); D "Chưa có lớp" không cắt; Admin topbar "Toàn hệ thống · INT1006" không cắt; `ell:[]`, `cut:[]` |
| TC-00-32 | PASS | C | `tc_00_08`: `title="761987 · An ninh mạng"` ≥ 1 và `aria-label` chứa tên đủ |
| TC-00-33 | PASS | C | `tc_00_09`: 4 chuỗi `Tài khoản: …` ≥ 1; SV A/C/D đúng tên; aria-label của GV/TA/Admin không ghi tên vai |
| TC-00-34 | PASS | A/B | `00-AC9` 11 dòng @1440 và 390/375: dòng 1 = tên người, dòng 2 = vai, `cut:false` cho cả 4 vai + SV A…D. Ảnh `091-menu-{student,ta,teacher,admin}-390.webp`, `041-menu-ho-so-sv.webp` |
| TC-00-35 | PASS | B | Menu mở → `Admin · Đỗ Hoàng Nam`: nút đổi ngay `Tài khoản: TS. Lê Thu Hà` → `Tài khoản: Đỗ Hoàng Nam`, chữ cái avatar H → N, không tải lại trang |
| TC-00-36 | PASS | A | Tôi chạy đủ ma trận: 4 vai × 31 route × {1440, 390} = 248 phép đo + `/login` 1440/390/375 → **mọi dòng `{ox:0, cut:[], ell:[]}`**, kể cả 67 màn chặn quyền; 0 lỗi console / pageerror |
| TC-00-37 | PASS | A | `audit.json` 375: 15 route SV + GV `/attendance`, `/inbox` → `{ox:0,cut:[],ell:[]}`; `/login` 375 `ox:0` |
| TC-00-38 | PASS | A | 390: `/library` (SV), `/attendance`, `/observability` (GV, Admin), `/admin/users` → `ox = 0`; 375 `/library`, `/attendance` → `ox = 0`. Ảnh `110-library-390.webp`, `112-attendance-390.webp`, `113-observability-390.webp`, `114-admin-users-390.webp` |
| TC-00-39 | PASS | A | `00-AC10 h3 dạng danh sách ở 390` và `ở 719`: 5/5 route PASS mỗi mức (không `<table>` nhìn thấy, 2–3 dòng/hàng, chip đủ chữ, `cut:[]`) |
| TC-00-40 | PASS | A | `00-AC10 h3 gradebook`: có `data-scroll-x`, cột tên dính trái sau khi cuộn 220 px, "QT tạm tính" không cắt, `ox:0` |
| TC-00-41 | PASS | A | `00-AC10 h2` GV 375: chú thích phím tắt ẩn, "Vắng phép" đủ chữ, `cut:[]`, vùng chạm ≥ 44 |
| TC-00-42 | PASS | A | `00-AC10 h4` `/students/sv-3` 390: dải tab cuộn ngang, tab "Ghi chú" tự cuộn vào khung, "Hoạt động học" không cắt, `ox:0` |
| TC-00-43 | PASS | A | `00-AC10 h3 bảng ở 720` 5/5 PASS và `dạng danh sách ở 719` 5/5 PASS (đúng ngưỡng) |
| TC-00-44 | PASS | A | Tôi chạy `?state=empty` và `?state=error` trên **mọi route mở được** của 4 vai ở 390 và 375 = 228 phép đo → tất cả `{ox:0, cut:[], ell:[]}` |
| TC-00-45 | PASS | A/B | 9 kịch bản trong `audit.json` (chat sau D1, thread sau Hỏi trợ lý AI, form thread + hộp thoại hai lối, menu hồ sơ, bộ chọn lớp, sidebar thu gọn, ticket mở, drawer Thêm) đạt ở 1440/390/375. Dòng máy FAIL `bộ chọn lớp mở` GV 1440 `cut:["An ninh mạng – 761987"]` là **lỗi công cụ**: phần tử bị đếm có `display:none` (rect 0×0, trong `coursePanel`), AUDIT không lọc phần tử ẩn. Đo tay: chữ hiện ở x 285–523 trong popover 240–540, không cắt — ảnh `021-chooser-gv-1440-open.webp` → PASS |
| TC-00-46 | PASS | C/A | `tc_00_07` (brand ×4 vai), `tc_00_hooks` (chat-history/thread/composer, inbox-list, provider-status ×3, provider-action ×3); mở ticket ở `/inbox` → `inbox-detail=1`, `inbox-reply=1` (đo tay) |
| TC-00-47 | **FAIL** | B | `[data-scroll-x]` **không** nằm trên `body`/`main`/`header`/khối bao trang (toàn bộ nằm trên hàng tab/chip và vùng cuộn bảng `/gradebook`) — phần chống lách đạt. Nhưng **5 vùng không cuộn được** ở cả 390 và 375 (`scrollWidth == clientWidth`): SV `/practice` tabs (343/343), SV `/calendar` segmented (202/202), GV `/questions` chips (331/331), GV `/analytics` segmented (138/138), GV `/grading` tabs (343/343). Mỗi vùng này không che chữ nào (`hiddenCut = 0`) → xem BUG-v5-00-6 + câu hỏi PM |
| TC-00-48 | PASS | C | `ui-antipatterns.sh` 0 `✗`; grep màu cứng ngoài `tokens.css` = 0 dòng |
| TC-00-49 | PASS | A | `LEFT (00-AC11) = 240`: 60 dòng (mọi route của 4 vai) @1440 đều 240 |
| TC-00-50 | PASS | A | `LEFT (00-AC11) = 16`: 60 dòng @390 đều 16 |
| TC-00-51 | PASS | B | 1920 và 1100 (SV `/`,`/chat`,`/calendar`,`/me`; GV `/`,`/inbox`,`/gradebook`,`/students`): `LEFT` = 240 ở mọi ô; khối đọc 1008 px (nội dung 960 + 2×24) **căn trái** tại `main.left` = 216, không dịch ra giữa khi 1920; lưới/bảng rộng hơn (1440 px) bắt đầu cùng lề; `/chat` và `/calendar` cùng 240 |
| TC-00-52 | **FAIL** | B | `?state=loading/empty/error`, màn chặn quyền, màn "Bạn chưa vào lớp nào" (SV D): `LEFT` = 240 @1440 và 16 @390, đúng 1 `[data-part=page-title]` ✔. Nhưng **404** (`/khong-co-trang`, cả 4 vai) và màn "Trang này gặp sự cố": `LEFT` = **24** @1440 (390 vẫn 16) → tiêu đề nhảy lề khi đổi trạng thái. Ảnh `040-404-admin-1440.webp`, `043-loi-chay-trang.webp` |
| TC-00-53 | PASS | A | `HEADER đặc (00-AC12)`: 139 dòng (mọi route × 1440/390/375, sau cuộn 200 px) `backdrop-filter: none`, màu nền đặc. Ghi chú: Chrome 150 trả `lab(98.785 1.832 0.984)` thay vì `rgb(…)` — không có kênh alpha, vẫn là màu đặc (xem mục "Sửa công cụ") |
| TC-00-54 | PASS | C/B | `tc_00_15`, `tc_00_16` PASS; `ui-antipatterns.sh` xanh trên cây sạch; thử đỏ trong `git worktree` `/tmp/qcv5-build`: thêm `backdrop-filter: blur(4px)` → script in 2 dòng ✗ và exit 1; hoàn lại → exit 0; worktree đã xoá |
| TC-00-55 | PASS | B | SV B `/chat` 390: gửi một tin, cuộn 200 px — thanh trên nền đặc (`lab(98.785 …)`, `backdrop-filter: none`), không chữ nào lộ mờ qua thanh. Ảnh `070-chat-390-cuon200.webp` |
| TC-00-56 | PASS | C | `tc_00_13`: không cookie → không "This page could"; 4 vai đều có "Không tìm thấy trang" + `Về Hôm nay`; có `app/error.tsx` và `app/not-found.tsx` |
| TC-00-57 | PASS | A/B | 4 vai × `/khong-co-trang` + `/threads/khong-co`, `/assignments/khong-co`, `/practice/khong-co`, `/students/sv-999`: không trang tiếng Anh, không trắng trang; id lạ ra màn rỗng trong khung app (h1 "Thread" / "Bài tập" / "Làm bài" / "Hồ sơ sinh viên", sidebar + điều hướng còn) |
| TC-00-58 | **FAIL** | B | `{`, `null`, `[]`, `{"x":1}`: cả 6 route (`/`,`/practice`,`/chat`,`/threads`,`/me`,`/calendar`) vào được bằng dữ liệu gốc, không màn lỗi ✔. `{"practice":"x","tickets":5,"bell":7}`: cả 6 route ra "Trang này gặp sự cố" tiếng Việt có `Thử lại` + `Về Hôm nay` ✔ (đúng nhánh cho phép). **Nhưng** mệnh đề cuối không đạt: từ màn đó **không tới được** `Đặt lại dữ liệu demo` (không còn sidebar lẫn nút hồ sơ), `Thử lại`/`Về Hôm nay` quay lại chính màn lỗi; phải xoá `localStorage` bằng DevTools mới thoát → BUG-v5-00-2 |
| TC-00-59 | **FAIL** | B | 404 → `Về Hôm nay` về `/` đúng vai (h1 "Hôm nay") ✔; `Thử lại` không treo vòng lặp ✔. **Nhưng** cả màn 404 lẫn màn "Trang này gặp sự cố" **không nằm trong khung app**: `document.querySelector('nav')` = null, thanh trên rỗng (không logo, không bộ chọn lớp, không nút hồ sơ), tiêu đề ở lề 24 px. Ảnh `040-404-admin-1440.webp`, `043-loi-chay-trang.webp` |
| TC-00-60 | PASS | C | `tc_00_14`: không chuỗi "N ngày/giờ/phút trước" ngoài `mock/derive.ts`; SV B không có "12 ngày trước"; Admin `/` và `/admin/courses` có "hôm qua 16:40"; Admin `/` không có "08:30" |
| TC-00-61 | PASS | B | Sau `Đặt lại`: chuông B = "Tài liệu mới: Chương 5 — Quản lý khoá và PKI · Thư viện · hôm qua 14:00" + "Bài tập 03 đã nộp, đang chờ chấm · Bài tập · 6 ngày trước"; chuông GV = "Bạn được phân công lớp An ninh mạng – 761988. Mã tham gia: BX4P9TW · Quản trị viên · hôm qua 16:40"; Admin `/` = "Phân công · hôm qua 16:40", `/admin/courses` hàng 761988 = "Mới mở · hôm qua 16:40". Ảnh `080-chuong-sv-b.webp`, `081-chuong-gv.webp`, `082-admin-home.webp`, `115-admin-courses-1440.webp` |
| TC-00-62 | **FAIL** | B | Lệch đồng hồ bằng `evaluateOnNewDocument` (mỗi mốc một trang mới để không cộng dồn): chuông SV B **không có mục D3 nào** nên không thể đọc chuỗi "vừa gửi → 59 phút trước → 1 giờ trước → 23 giờ trước"; nhãn D3 ở `/chat` đứng yên "Đang chờ giảng viên · **vừa gửi**" ở +59 phút, +60 phút, +23 giờ. Phiếu D3 ở `/inbox` GV chạy đúng tới +60 phút ("vừa xong" → "59 phút trước" → "1 giờ trước") nhưng ở **+23 giờ ghi "hôm qua 09:20"** thay vì "23 giờ trước" (SRS 4.8 N6: < 24 giờ → "N giờ trước"). Mốc khác đúng: BT03 "6 ngày trước" → "7 ngày trước" ở +23/+24 giờ; Chương 5 "hôm qua 14:00" → "1 ngày trước" ở +24 giờ |
| TC-00-63 | **FAIL** | B | Gửi D3, chờ **125 giây thật**, tải lại: `/inbox` GV ghi "Trần Thu Uyên · **2 phút trước**" (đồng hồ chạy theo giờ thật ✔) nhưng **chuông SV B vẫn chỉ có 2 mục cũ, không có mục D3** và nhãn ở `/chat` vẫn "vừa gửi" → không đọc được "2 phút trước" như TC yêu cầu. Sau `Đặt lại dữ liệu demo`: `/me` = "Cập nhật Thứ Năm, 29 tháng 10 09:20", Hôm nay = "Thứ Năm, 29 tháng 10 · tuần 10" ✔ |
| TC-00-64 | PASS | A | `TOUCH (00-AC15)`: 28 dòng (12 route SV + GV `/attendance`, `/inbox`) × 375/390 đều `[]` |
| TC-00-65 | PASS | B | 375, SV B: chip chủ đề 233×44 / 179×44 (11 chip, h = 44), nhãn "Nhờ AI trả lời gợi ý (Socratic)" 301×45 — bấm vào **chữ nhãn** đổi `checked` true → false; `Báo cáo` 76×44; `Hỏi trợ lý AI` 145×44; `Tuần` 49×44, `Tháng` 60×44, `Danh sách` 88×44; `Thoát về Luyện đề` 147×44; liên kết quay lại "Threads" 77×44; "Luyện đề" 83×44. Bộ lọc Thư viện và các liên kết còn lại nằm trong `TOUCH = []` của TC-00-64 |
| TC-00-66 | PASS | B | 375, 14 route SV: `document.querySelectorAll('[data-inline]')` = **0 phần tử** ở mọi route → không có nút/chip/tab/liên kết quay lại nào mượn `data-inline` để lách |
| TC-00-67 | PASS | A/B | Tôi chạy `TOUCH` nguyên văn ở 375 trong 7 trạng thái: drawer `Thêm`, popover bộ chọn lớp, hộp thoại hai lối của form Threads, dải chip đã cuộn 300 px, `/library?state=empty`, `?state=error`, `/inbox` GV sau khi mở ticket → **tất cả `[]`**. Ảnh `120-drawer-them-375.webp`, `121-dialog-pii-375.webp`, `122-inbox-ticket-375.webp` |
| TC-00-68 | PASS | C | `tc_00_16`: có `mock/derive.ts` với `ticketStats`, `attentionSet`, `navBadges`, `ago`; không còn `overdue: [` / `answeredByAi` / `escalated: [`; shell/ui không `color-mix(… transparent` |
| TC-00-69 | **FAIL** | A/B | `v24.json` TC-00-69: 6/12 ca FAIL. Đo tay khớp: @1440 và @1100 nút hồ sơ ghi **tên vai** — GV `"H Giảng viên"` (w 144), TA `"B Trợ giảng"` (137), Admin `"N Admin"` (114); chỉ SV ghi tên người (`"U Trần Thu Uyên"`, 169). @390 cả 4 vai chỉ còn chữ cái (đúng spec). `aria-label` đúng (`Tài khoản: <tên>`) và menu đúng (dòng 1 tên, dòng 2 vai) nhưng chữ **nhìn thấy** trên nút vẫn là tên vai → #24b chưa sửa. Ảnh `010-topbar-teacher-1440.webp`, `010-topbar-teacher-1100.webp`, `010-topbar-ta-1440.webp`, `010-topbar-ta-1100.webp`, `010-topbar-admin-1440.webp`, `010-topbar-admin-1100.webp`, `010-topbar-student-1440.webp` |

## Lỗi

| ID | Route | Vai | Bước tái hiện | Thấy | Mong đợi | Mức | AC | Ảnh |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BUG-v5-00-1 | mọi route | SV B | DevTools: `localStorage.setItem('ep_demo_state','{"practice":"x","tickets":5,"bell":7}')` → tải lại `/` | Mọi route ra "Trang này gặp sự cố"; `Thử lại` và `Về Hôm nay` quay lại chính màn đó; màn lỗi không có sidebar/nút hồ sơ nên **không thể** bấm `Đặt lại dữ liệu demo` — chỉ thoát được bằng DevTools | Có lối thoát trong app (ví dụ nút `Đặt lại dữ liệu demo` ngay trên màn lỗi, hoặc tự bỏ state hỏng như 4 dạng JSON kia) | **cao** | 00-AC13 (TC-00-58) | `043-loi-chay-trang.webp` |
| BUG-v5-00-2 | `/khong-co-trang`, màn "Trang này gặp sự cố" | 4 vai | Mở URL không tồn tại ở 1440 | Không có sidebar (`nav` = null), thanh trên rỗng (không logo/bộ chọn lớp/nút hồ sơ); `[data-part=page-title]` lề **24 px** thay vì 240 | Màn lỗi nằm trong khung app, tiêu đề cùng lề 240 px như mọi trạng thái khác | vừa | 00-AC11, 00-AC13 (TC-00-52, 59) | `040-404-admin-1440.webp` |
| BUG-v5-00-3 | `/chat`, chuông SV | SV B | Gửi D3; chờ 2 phút thật hoặc lệch đồng hồ +59 phút / +1 giờ / +23 giờ; mở lại `/chat` và chuông | Nhãn "Đang chờ giảng viên · **vừa gửi**" đứng yên ở mọi mốc; chuông SV không sinh mục nào cho D3 (chỉ 2 mục seed) | Nhãn theo `ago()` (N6): "59 phút trước", "1 giờ trước"…; chuông B có mục D3 ghi "2 phút trước" | vừa | 00-AC14 / FR-X13 (TC-00-62, 63) | — |
| BUG-v5-00-4 | `/inbox` | GV | Gửi D3 rồi lệch đồng hồ +23 giờ, xem phiếu D3 | "hôm qua 09:20" | "23 giờ trước" (SRS 4.8 N6: < 24 giờ → "N giờ trước", mới tới luật "khác ngày") | vừa | 00-AC14 (TC-00-62) | — |
| BUG-v5-00-5 | mọi route (thanh trên) | GV, TA, Admin | Mở `/` ở 1440 hoặc 1100 | Nút hồ sơ ghi "Giảng viên" / "Trợ giảng" / "Admin" (SV ghi "Trần Thu Uyên") | Ghi tên người: "TS. Lê Thu Hà", "Phạm Quốc Bảo", "Đỗ Hoàng Nam" | vừa | 00-AC9 (#24b, TC-00-69) | `010-topbar-teacher-1440.webp`, `010-topbar-ta-1100.webp`, `010-topbar-admin-1440.webp` |
| BUG-v5-00-6 | `/practice`, `/calendar` (SV); `/questions`, `/analytics`, `/grading` (GV) | SV, GV | 390 và 375: `[...document.querySelectorAll('[data-scroll-x]')].map(e => [e.scrollWidth, e.clientWidth])` | 5 vùng có `scrollWidth == clientWidth` (không cuộn được) nhưng vẫn mang `data-scroll-x` — tức AUDIT bỏ qua nội dung bên trong mà không có lý do | `data-scroll-x` chỉ trên vùng thực sự cuộn ngang | thấp | 00-AC10 (TC-00-47) | — |

**Câu hỏi cho PM** (TC-00-47): `data-scroll-x` nằm trên component `Tabs` dùng chung, nên ở bề rộng mà hàng tab vừa đủ chỗ thì vùng không cuộn. PM muốn (a) giữ nguyên và sửa câu TC thành "mọi vùng `data-scroll-x` phải là hàng tab/chip/bảng, cuộn khi tràn", hay (b) yêu cầu dev chỉ gắn thuộc tính khi thật sự tràn? Hiện không có chữ nào bị giấu (`hiddenCut = 0` trong cả 5 vùng).

## Phản mẫu UI / phân quyền / ngưỡng 375–390 (soi ảnh theo DESIGN §21, §22)

- **Không có phản mẫu bị cấm**: không glass (thanh trên đặc ở 139 ca đo), không spinner giữa trang (9 màn `?state=loading` chỉ có skeleton khối), không chữ gradient / nảy / gamification (`ui-antipatterns.sh` 11 ✓), không màu cứng ngoài `tokens.css`.
- **Phân quyền**: 124 ô ma trận đúng; điều hướng và ⌘K của mỗi vai không lộ route ngoài quyền; đổi vai giữa chừng không để lại nội dung của vai cũ; màn chặn có một câu lý do + `Về Hôm nay`.
- **375–390 px**: shell, `/login`, màn chặn, 404 đều `ox = 0`; thanh dưới 4 đích + `Thêm` (≤ 5); vùng chạm ≥ 44 px kể cả trong drawer/dialog/popover.
- Soi ảnh 390: `114-admin-users-390.webp` — dải tab lọc cắt chữ "Sin…" ở mép phải (vùng `data-scroll-x` nên AUDIT bỏ qua); không có dấu hiệu cuộn nhìn thấy, cùng kiểu lỗi PM nêu ở #24c cho chip Threads. `112-attendance-390.webp` — ô "Buổi 10 · 29/10 · đang diễn" bị cắt mất chữ "ra" (đúng như #24d, thuộc TC-02-91 của story 02).

## Cảm nhận khi dùng như chủ dự án

- Luồng nền chạy mượt: chọn vai → `/` → đổi vai tại chỗ → `/inbox` thấy ngay phiếu vừa hỏi; `Đặt lại dữ liệu demo` trả mọi con số về mốc gốc. Phần này đã đủ để demo.
- **Chỗ gượng nhất là nhãn thời gian phía sinh viên** (`/chat`): câu vừa gửi mãi mãi "vừa gửi", trong khi bên giảng viên đã "2 phút trước". Người xem demo rất dễ bắt được điểm này nếu để máy chạy vài phút.
- **Màn lỗi/404 trông như rơi ra khỏi sản phẩm**: trắng trang, không logo, không điều hướng, chữ dính sát mép trái 24 px — khác hẳn phần còn lại (240 px). Nhìn giống trang lỗi mặc định đã dịch sang tiếng Việt hơn là một màn của EduPilot.
- **Nút hồ sơ ghi "Giảng viên"/"Admin"** làm thanh trên mất cảm giác "đang là ai"; SV thì ghi tên người nên trông thiếu nhất quán giữa các vai (ảnh `010-topbar-teacher-1440.webp` so với `010-topbar-student-1440.webp`).
- Drawer `Thêm` của SV ở 375 chiếm trọn màn hình cho đúng 3 mục (Thư viện, Lịch, Kết quả của tôi), phần dưới trống hơn nửa màn — nhìn gượng, nên là bảng trượt cao theo nội dung (ảnh `120-drawer-them-375.webp`).
- `/admin/users` ở 390: nút `Khoá` đứng trơ một mình dưới mỗi người, không nhóm với trạng thái — nhìn hơi rỗng; hàng người dùng trông giống bảng vỡ hơn là thẻ.
- Dải "Bản mô phỏng · dữ liệu giả" ở 390 nằm riêng một dòng xám ngay dưới thanh trên: đọc được, không chói, đúng ý "chữ phụ".

## Đề nghị

1. Sửa **BUG-v5-00-1** trước (mức cao): bỏ khoá `ep_demo_state` không đọc được thay vì ném lỗi (như đã làm với `{`, `null`, `[]`, `{"x":1}`), hoặc thêm nút `Đặt lại dữ liệu demo` ngay trên màn "Trang này gặp sự cố". Đây cũng là rủi ro thật khi người xem demo còn state của bản build cũ trong trình duyệt.
2. Đưa 404 và màn lỗi vào khung app (sidebar + thanh trên) để lề tiêu đề về 240/16 (BUG-v5-00-2) — hết luôn TC-00-52 và TC-00-59.
3. Nối nhãn trạng thái D3 ở `/chat` (và mục chuông của SV) vào `ago()` của `derive.ts`; chỉnh thứ tự luật N6 để < 24 giờ luôn ra "N giờ trước" (BUG-v5-00-3, -4).
4. #24b: đổi chữ trên nút hồ sơ của GV/TA/Admin sang tên người (BUG-v5-00-5) — sửa một chỗ, hết TC-00-69.
5. Thống nhất `data-scroll-x` (BUG-v5-00-6) sau khi PM trả lời câu hỏi ở trên; nhân tiện xem lại dải tab `/admin/users` 390 đang cắt chữ ở mép như #24c.

## Sửa công cụ QC v5 (ghi nhận, không phải TC)

Các script trong `docs/sprints/1.5/qc/scripts/` đã được QC chỉnh cho hợp giao diện v5.1 trước vòng này:

1. `audit.mjs` chấp `lab()` / `oklab()` là màu đặc — Chrome 150 trả `getComputedStyle().backgroundColor` dạng `lab(98.785 1.832 0.984)` chứ không còn `rgb(…)` (ảnh hưởng phép đo `HEADER đặc (00-AC12)`).
2. Hàm `click` trong `audit.mjs` ưu tiên `aria-label` và bấm phần tử con bấm được (button/a nằm trong `li`) thay vì bấm hàng.
3. Kịch bản `form thread + hộp thoại hai lối` mở form bằng nút `Đặt câu hỏi` trước khi gõ (form v5.1 không mở sẵn).
4. `pii-matrix.mjs` chỉ tính hộp thoại **đang nhìn thấy** (`getBoundingClientRect().width > 0`).
5. `TOUCH_SRC` theo đề xuất #21: chỉ tính `label` của checkbox/radio, bỏ `label` bọc input thường.
6. `proto-curl.sh` `tc_04_08` và biến `${s}` theo đề xuất #23.
7. `threads-timeline.mjs`: mở form trước khi tạo thread; chỉ tính nhãn ở trang thread; mở "Nguồn tham khảo" trước khi đọc nguồn; nền TA = 4 việc; T15e +2/+3; chờ 0,9 s trước khi rời trang (T13).
8. `regress-v24.mjs` là script mới cho hồi quy #24 (TC-00-69, TC-01-153, TC-02-90, TC-02-91).

**Lỗi công cụ phát hiện trong vòng này:** đoạn `AUDIT` nguyên văn của US.md không lọc phần tử `display:none` — phần tử ẩn có rect `0×0` luôn bị tính là "cắt". Đây là nguyên nhân dòng FAIL `teacher / 1440 bộ chọn lớp mở → cut:["An ninh mạng – 761987"]` trong `audit-fails.json`; đo tay cho thấy chữ hiện đủ (TC-00-45). Đề nghị thêm `if (!e.getClientRects().length) continue;` vào đoạn AUDIT của US.md nếu PM đồng ý (sửa spec, không sửa lén).
