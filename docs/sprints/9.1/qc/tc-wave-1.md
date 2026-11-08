# TC đợt 1 — Nền chung (US-UX91-02, 03, 04, 21)

Phiên bản 1 · 2026-11-08. Quy ước và kiểm chung: `tc-00-chung.md`. Spec: `docs/specs/FEAT-uiux-91/US.md`.
Màn trưng bày primitive: `/dev/ui` (stack `APP_ENV=dev`; nếu dev đổi đường dẫn, ghi ở handoff).

---
## US-UX91-02 — Định dạng ngày giờ dùng chung

Dữ liệu vào cho TC định dạng: QC chặn (`page.route`) phản hồi API của màn tương ứng và **đặt lại các trường thời gian** theo bảng dưới; đồng hồ trình duyệt cố định `2026-11-08T14:00:00+07:00` (Chủ nhật) bằng `page.clock.setFixedTime`. Màn dùng: Threads (`/threads`, ngày thread), Chat (`/chat`, lịch sử phiên), Tài liệu (`/documents`, cột Cập nhật), Thư viện (`/library`, "cập nhật"), Hộp thư (`/inbox`), Thiết bị (`/settings`, "Lần cuối").

| Mã đầu vào | ISO (+07:00) | Chuỗi mong đợi (AC2) |
| --- | --- | --- |
| I1 | `2026-11-08T09:20:00` | `Hôm nay, 09:20` |
| I2 | `2026-11-07T16:40:00` | `Hôm qua, 16:40` |
| I3 | `2026-11-02T14:00:00` | `Thứ Hai, 14:00` |
| I4 | `2026-09-21T14:00:00` | `Thứ Hai, 21/09 · 14:00` |
| I5 | `2025-09-21T14:00:00` | `Thứ Hai, 21/09/2025 · 14:00` |
| I6 | `2026-11-08T00:01:00` | `Hôm nay, 00:01` |
| I7 | `2026-11-07T23:59:00` | `Hôm qua, 23:59` |
| I8 | `2026-11-01T23:59:00` (đổi tuần: Chủ nhật tuần trước) | không phải `Thứ …, 23:59` cùng tuần; mong đợi theo luật "trong 6 ngày trước": 7 ngày trước → dạng xa hơn `Chủ Nhật, 01/11 · 23:59` |
| I9 | `2026-11-09T09:00:00` (ngày mai) | **chưa có hàng trong AC** — xem `questions.md` Q1; tạm ghi nhận chuỗi thực tế |
| I10 | `null` / `"không-phải-ngày"` | `—` |

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX02-1 | AC1 | — | `rg -n "Intl\.DateTimeFormat\|toLocaleDateString\|toLocaleTimeString" frontend/src --glob '!**/*.test.*' --glob '!**/*.spec.*'` rồi loại các tệp thuộc mô-đun định dạng ở `shared/lib` (tên do handoff nêu) | Không còn dòng nào ở 13 tệp liệt kê (`admin/users/api.ts`, `chat/RealChat.tsx`, `documents/RealDocuments.tsx`, `inbox/RealInbox.tsx`, `library/RealLibrary.tsx`, `me/meApi.ts`, `members/ClassSettings.tsx`, `settings/security/DevicesSection.tsx`, `students/RealProfile.tsx`, `threads/RealThreadDetail.tsx`, `threads/RealThreads.tsx` (cả dòng 158), `today/format.ts`, `shared/lib/timeAgo.ts`) cho mục đích hiển thị; ghi từng dòng còn sót kèm lý do (nếu là chuỗi tải xuống / email thì loại) | GREP |
| TC-UX02-2 | AC2 | giảng viên, 1440 | `/threads` với I1…I8 gán cho 8 thread liên tiếp | Mỗi thread hiện đúng chuỗi ở bảng; chữ «Thứ Hai … Chủ Nhật», **không** «Thứ 2» | E2E |
| TC-UX02-3 | AC2 | sinh viên, 1440 | `/chat` lịch sử phiên đặt I1, I2, I3, I4 | Đúng bảng | E2E |
| TC-UX02-4 | AC2 | giảng viên | `/documents` cột «Cập nhật» I4, I5; `/library` (sinh viên) «cập nhật» I3, I4 | Đúng bảng; **không** còn dạng `00:10 02-11` | E2E |
| TC-UX02-5 | AC2 (chỉ ngày) | giảng viên | Nơi chỉ hiển thị ngày (hạn bài, ngày báo cáo Insights) với `2026-09-21` và `2025-09-21` | `21/09` và `21/09/2025` | E2E |
| TC-UX02-6 | AC2 (tương đối ngắn) | giảng viên | Chuông thông báo với bản ghi cách 30 s, 5 phút, 3 giờ, 26 giờ | `vừa xong`, `5 phút trước`, `3 giờ trước`, sau đó `Hôm qua, HH:mm` | E2E |
| TC-UX02-7 | AC3 | giảng viên + sinh viên | `timezoneId: 'America/Los_Angeles'`; mở Threads, Tài liệu, Chat, Thư viện với I1 / I2 | Chuỗi y hệt TC-UX02-2…4 (giờ ICT, không lệch 14–15 giờ) | E2E |
| TC-UX02-8 | AC3 | giảng viên | Lặp TC-UX02-7 với `timezoneId: 'Pacific/Kiritimati'` (UTC+14) | Như trên | E2E |
| TC-UX02-9 | AC4 | giảng viên | Đặt `created_at` = `"abc"`, `null`, `""`, `"2026-13-45"` ở 4 thread | Hiện `—`; không chữ `Invalid Date`; không `pageerror` / `console.error` | E2E |
| TC-UX02-10 | AC5 | — | `pnpm -C frontend test` (hoặc lệnh dev nêu) lọc theo mô-đun; đọc **tên ca** (không đọc cài đặt) | Có ca cho: mọi hàng AC2, 23:59 / 00:01 ICT, đổi tuần, khác năm, `null`, múi giờ trình duyệt khác; tất cả xanh. Thiếu nhóm nào → FAIL | UNIT |
| TC-UX02-11 | AC5 | — | QC tự chạy lại I1…I10 trực tiếp bằng `node` import mô-đun đã build (hoặc qua trang `/dev/…` nếu có) | Khớp bảng | UNIT |
| TC-UX02-12 | AC6 | — | Chụp `threads`, `chat`, `inbox` trước / sau (K7) | Ngày giờ đổi kiểu theo AC2; không đổi gì khác ngoài dự kiến | MAC |
| TC-UX02-13 | AC7 | sinh viên, 375 | Threads + Chat với I5 (chuỗi dài nhất) | `scrollWidth ≤ clientWidth`; chuỗi được phép xuống dòng, không cắt chữ | E2E |
| TC-UX02-PQ | (kiểm chéo) | STUDENT | Mở `/documents`, `/inbox` | 403 như hiện có; mô-đun mới không làm lộ ngày giờ ở nơi bị chặn | E2E |

---
## US-UX91-03 — Nút khoá nói lý do

| TC | AC | Vai / cỡ | Tiền điều kiện | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- | --- |
| TC-UX03-1 | AC1 | — / 1440 | `/dev/ui` có nút khoá kèm lý do (dev ghi vị trí ở handoff) | Đọc DOM nút: thuộc tính `disabled`, `aria-disabled`, `aria-describedby`; nội dung phần tử được trỏ | Nhãn nút nguyên; `aria-disabled="true"`; **không** có thuộc tính `disabled`; `aria-describedby` trỏ tới `span` ẩn trực quan chứa đúng lý do | E2E |
| TC-UX03-2 | AC1 | — | như trên | Rê chuột vào nút | Tooltip hiện chứa lý do | E2E |
| TC-UX03-3 | AC1 | — | như trên | `click()` nút; đếm request và sự kiện `onClick` (quan sát qua mạng / trạng thái màn) | Không phát sinh hành động nào | E2E |
| TC-UX03-4 | AC1 | — | như trên | `getComputedStyle(nút).opacity`; so với nút `disabled` thường | `0.45` cho cả `[aria-disabled="true"]` | E2E |
| TC-UX03-5 | AC1 | — | — | Chạy toàn bộ `pnpm -C frontend exec playwright test` | Không spec nào đỏ vì `toBeDisabled()`; liệt kê spec dev đã sửa nếu có, kiểm `git diff` không nới assertion | E2E |
| TC-UX03-6 | AC2 | giảng viên, 1440 | Lớp có công thức **chưa xác nhận** (nếu seed không có: admin mở lớp mới trên `ep-qc`, chưa tải quy chế) | `/gradebook` | `Chốt điểm` khoá; lý do «Cần xác nhận công thức trước» | E2E |
| TC-UX03-7 | AC2 | giảng viên | Công thức có N mục «chưa rõ» (tải quy chế mơ hồ bằng fixture hoặc sửa bản nháp) | `/gradebook/scheme` | `Xác nhận công thức` khoá; lý do «Còn N mục chưa rõ» với N đúng (đổi N → đổi chữ) | E2E |
| TC-UX03-8 | AC2 | giảng viên | Lớp chưa có bài nộp (lớp mới) | `/grading` | `Chấm tất cả` khoá; lý do «Chưa có bài nộp» | E2E |
| TC-UX03-9 | AC2 | giảng viên | Lớp có bài nộp chờ chấm (761987) | `/grading` tab Hàng chờ | `Chấm tất cả` **bấm được** (không `aria-disabled`) | E2E |
| TC-UX03-10 | AC2 | giảng viên | trạng thái khoá của `Tải bài lên` | `/grading` | Khoá thì có lý do bằng chữ nêu điều kiện thật; không khoá thì không `aria-disabled` — **chữ lý do chưa được AC quy định**, xem `questions.md` Q2 | E2E |
| TC-UX03-11 | AC2 | sinh viên | trang chat trống | `/chat` | Không có nút `Phiên mới` khoá (ẩn thay vì khoá; chi tiết ở US-13) | E2E |
| TC-UX03-12 | AC3 | — | nút khoá có lý do | Tab tới nút; chụp; Enter; Space | Focus ring thấy được (outline / box-shadow khác `none`); tooltip hiện; Enter / Space không kích hoạt hành động | E2E |
| TC-UX03-13 | AC4 | giảng viên, 375, `hasTouch` | nút khoá có lý do | Chạm nút; đo thời gian lý do còn hiện; chạm chỗ khác | Lý do hiện **dưới** nút, còn ≥ 3 s hoặc tới khi chạm chỗ khác; không phụ thuộc hover | E2E |
| TC-UX03-14 | AC5 | — | một nút có `disabledReason` nhưng **không** `disabled` | Rê chuột, click | Không tooltip; nút hoạt động bình thường. (Cần `/dev/ui` trưng bày ca này — xem `questions.md` Q3) | E2E |
| TC-UX03-15 | AC6 | STUDENT, TA | — | STUDENT mở `/gradebook`, `/grading`; đọc DOM | 403; `Chốt điểm`, `Chấm tất cả`, `Xác nhận công thức` **không có trong DOM** | E2E |
| TC-UX03-16 | AC6 | giảng viên lớp 1 | — | Đọc chữ lý do ở TC-6…8 | Lý do là chuỗi tĩnh, không có tên / MSSV / điểm của người khác | E2E |
| TC-UX03-17 | AC7 | — | tooltip | `getAnimations()` ngay khi hiện; `reduce` bật | `opacity` ≤ 120 ms; `reduce` → tức thì (≤ 1 ms hoặc không có hoạt cảnh) | E2E |
| TC-UX03-18 | AC8 | — | — | Chụp `dev-ui`, `gradebook` 1440 / 390 trước / sau | `dev-ui` có thêm trạng thái nút khoá lý do; `gradebook` đúng dự kiến | MAC |

---
## US-UX91-04 — Chuyển động chức năng của ứng dụng

Công cụ đo trong trang (E2E): `getComputedStyle`, `el.getAnimations()`, lấy mẫu `background-color` mỗi `requestAnimationFrame`, `PerformanceObserver({type:'layout-shift'})`.

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX04-1 | AC1 | — / 1440 | `/dev/ui`: với mỗi biến thể `primary`, `secondary`, `ghost`, `text`, icon: `mouse.move` lên nút, `mouse.down()`, đợi 120 ms, đọc `transform` | `matrix(0.97, 0, 0, 0.97, …)`; `transition-duration` của `transform` ≤ 100 ms; hàm easing = `--ep-ease-out` | E2E |
| TC-UX04-2 | AC1 | — | `mouse.up()`; đợi 150 ms | `transform` về `none` / `matrix(1,…)` | E2E |
| TC-UX04-3 | AC2 | — | Quét mọi `*.css` / `*.module.css` ở `frontend/src/shared` và `frontend/src/features`: script `docs/sprints/9.1/qc/scripts/hover-check.mjs` (QC viết: duyệt khối `{}` , báo mọi luật có `:hover` không nằm trong `@media (hover: hover) and (pointer: fine)`) | 0 vi phạm; liệt kê tệp:dòng nếu có | GREP |
| TC-UX04-4 | AC2 | giảng viên, 375, `hasTouch` | Chạm một dòng danh sách (Hộp thư) rồi chụp | Dòng không giữ nền hover sau khi chạm xong | E2E |
| TC-UX04-5 | AC3 | — | `rg -n "(--ep-[a-z-]+):\s*([0-9.]+)(ms\|s)" frontend/src/**/tokens.css`; đổi sang ms | Không token chuyển động nào > 200 ms (`--ep-slow` ≤ 200 ms) | GREP |
| TC-UX04-6 | AC3 | giảng viên | `/gradebook` bấm `Chốt điểm` (hộp thoại) và `/calendar` `Tạo lịch buổi học` (drawer); `getAnimations()` ngay sau khi mở | Mỗi hoạt cảnh mở ≤ 200 ms (so với 250 ms ở bản trước) | E2E |
| TC-UX04-7 | AC3 | — | `rg -n "Drawer / dialog: ≤ 200" docs/design/DESIGN.md` (§11) | Có đúng một dòng ghi quy định | GREP |
| TC-UX04-8 | AC4 | — | `rg -n "transition:\s*width\|transition-property:\s*width" frontend/src` | 0 kết quả ở `GradeScheme`, `Documents`, `Questions`, `Practice` (và ở nơi khác nếu là thanh tiến độ) | GREP |
| TC-UX04-9 | AC4 | giảng viên / sinh viên | Màn có thanh tiến độ (Luyện đề đang làm: sang câu kế tiếp; Tài liệu đang tải lên) | Phần tử thanh dùng `transform: scaleX(...)` với `transform-origin: left`; thời lượng 140 ms; `offsetWidth` của thanh **không đổi** khi tiến độ đổi; CLS = 0 | E2E |
| TC-UX04-10 | AC5 | giảng viên, 1440 | Bấm `Thu gọn`; lấy mẫu `main.getBoundingClientRect()` mỗi frame trong 300 ms; đọc `getComputedStyle(main).transitionProperty` | `transitionProperty` không chứa `margin-left`; bề rộng đổi tức thì hoặc hoàn tất ≤ 160 ms; tổng CLS ≤ 0,05 | E2E |
| TC-UX04-11 | AC6 | giảng viên | Mở menu tài khoản, chọn lớp, `⋯` ở sổ điểm; đọc `transform-origin` và `getAnimations()` của lớp phủ | Gốc xuất phát ở **nút gọi** (không `50% 50%` của màn hình); `translateY(-4px)` + `opacity`; ≤ 160 ms. Dialog (`Chốt điểm`) ở giữa màn, gốc không đổi | E2E |
| TC-UX04-12 | AC7 | giảng viên, 1440 | `/gradebook`: sửa một ô điểm, Tab ra; lấy mẫu `background-color` ô mỗi frame 0–600 ms; quan sát có toast chữ «Thành công» | Màu nền chuyển sang tông xanh `--ep-green` nhạt rồi về màu thường trong **≤ 200 ms** kể từ lúc đạt đỉnh; không toast; không âm thanh | E2E |
| TC-UX04-13 | AC7 | giảng viên | `/attendance`: đổi một dòng sang Vắng, bấm `Lưu điểm danh` | Nháy xanh nhạt ở phần tử xác nhận lưu theo cùng đặc tả | E2E |
| TC-UX04-14 | AC7 | giảng viên | `/grading/‹id›` sửa nhận xét rồi rời ô | Nháy xanh nhạt ở ô nhận xét | E2E |
| TC-UX04-15 | AC8 | — | `git diff 1b72f54... -- frontend/package.json` | Không thêm dependency chuyển động (framer-motion, gsap, lottie, motion…) | GREP |
| TC-UX04-16 | AC8 | — | Quét `@keyframes` trong `frontend/src/**/*.css` (script `keyframes-check.mjs`): liệt kê thuộc tính được animate; quét `animation-delay` / `transition-delay` dạng so le (`calc(var(--i)…)`, `nth-child`) | Mọi keyframe chỉ `transform` / `opacity` (ngoài danh sách loại trừ của story: spinner `Button`, ba chấm Threads, Red Thread); không stagger; không bounce / parallax. Keyframe khác (vd. `shimmer` đổi `background-position`) → ghi lỗi | GREP |
| TC-UX04-17 | AC8 | giảng viên | Mở `/` và `/inbox` tải mới; so thời điểm bắt đầu của `getAnimations()` | Không phần tử nào bắt đầu lệch nhau theo thứ tự (không stagger) | E2E |
| TC-UX04-18 | AC9 | giảng viên, `reduce` | `rg -n "ep-motion-safe" frontend/src`; `base.css` giữ luật lưới `1ms` | Lớp `.ep-motion-safe` tồn tại; luật lưới `1ms` còn nguyên | GREP |
| TC-UX04-19 | AC9 | giảng viên, `reduce` | Lặp TC-UX04-12 với `reducedMotion: 'reduce'` | Ô vẫn đổi màu nền và tắt sau ≤ 200 ms; **không** `transform` / trượt | E2E |
| TC-UX04-20 | AC10 | giảng viên | `/inbox` bấm `J`, `K`; `/grading/‹id›` bấm `A`; `/attendance` bấm `1`–`4`; `⌘K` | `document.getAnimations()` ngay sau phím = 0 (trạng thái đổi tức thì) | E2E |
| TC-UX04-21 | AC11 | — | Thanh tiến độ nhận 150, −10, `NaN` (dev nêu chỗ truyền được: `/dev/ui` hoặc mock phản hồi tải lên) | Kẹp 0–100, không tràn khỏi khung. Không có cách truyền → `questions.md` Q3 | E2E |
| TC-UX04-22 | AC12 | — | Chạy `visual.spec.ts`; so danh sách snapshot trong `git diff` | Chỉ đổi những ảnh mà handoff giải thích (`dev-ui` nếu trưng bày `:active`); các ảnh khác không đổi | E2E |
| TC-UX04-PQ | (kiểm chéo) | STUDENT | `/gradebook`, `/attendance` | 403 hiện có; không nháy / chuyển động mới ở màn 403 | E2E |

---
## US-UX91-21 — Trang công khai không gọi làm mới phiên

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX21-1 | AC1 | khách, 1440 | Context mới (không `localStorage`, không cookie); ghi mọi request; mở `/`; đợi `networkidle` | **Không** có `POST /auth/refresh`; theo AC6 không có request `/api/*` nào; trạng thái phiên anonymous ngay; console không có 401 | E2E |
| TC-UX21-2 | AC1 | khách | Lặp ở 375 px | Như trên | E2E |
| TC-UX21-3 | AC2 | giảng viên | Đăng nhập; mở tab mới cùng context → `/`; ghi request; `addInitScript` gắn `MutationObserver` để bắt mọi lúc `h1` của trang giới thiệu hiện | Có `POST /auth/refresh`; hiện «Hôm nay»; **không** thấy trang giới thiệu dù một frame (không nháy) | E2E |
| TC-UX21-4 | AC3 | khách có cookie | Đăng nhập; `localStorage.clear()`; mở `/` | Thấy trang giới thiệu có nút `Đăng nhập` | E2E |
| TC-UX21-5 | AC3 | — | Từ TC-4 bấm `Đăng nhập` | Tới `/login`, có `POST /auth/refresh` bằng cookie; phiên còn → vào thẳng ứng dụng (không dừng ở form) | E2E |
| TC-UX21-6 | AC3 | — | Xoá cả cookie rồi `/login` | Hiện form đăng nhập bình thường | E2E |
| TC-UX21-7 | AC4 | khách | `addInitScript`: `Storage.prototype.getItem = () => { throw new DOMException('SecurityError') }`; mở `/` | Không `pageerror`, hiện trang giới thiệu như khách | E2E |
| TC-UX21-8 | AC4 | khách | Chặn `localStorage` hoàn toàn: `Object.defineProperty(window, 'localStorage', { get() { throw new DOMException('denied', 'SecurityError') } })` trong `addInitScript` | Như TC-UX21-7: không lỗi, như khách | E2E |
| TC-UX21-9 | AC5 | khách | Mở `/chat`, `/gradebook` | Chuyển `/login?next=%2Fchat` (và `%2Fgradebook`); không lộ nội dung | E2E |
| TC-UX21-10 | AC6 (ngoài phạm vi) | khách | Mở `/login`, `/register` không có cookie | Vẫn gọi `POST /auth/refresh` như hiện tại (route công khai khác không đổi) | E2E |
| TC-UX21-11 | AC6 | — | `landing.spec.ts` sau khi dev sửa: đọc dòng kiểm «không gọi API» | Đổi thành «không gọi API nào ở ẩn danh»; có test «có cờ ⇒ có gọi refresh»; `git diff` không nới (không xoá assertion khác) | GREP |
| TC-UX21-PQ | (kiểm chéo) | STUDENT | Sau khi đăng nhập SV, `/` → Hôm nay sinh viên (không phải giảng viên) | Đúng vai | E2E |

---
## Bảng phủ AC đợt 1
| Story | AC | TC |
| --- | --- | --- |
| US-02 | AC1 | 02-1 |
| | AC2 | 02-2, 3, 4, 5, 6 |
| | AC3 | 02-7, 8 |
| | AC4 | 02-9 |
| | AC5 | 02-10, 11 |
| | AC6 | 02-12 |
| | AC7 | 02-13 |
| US-03 | AC1 | 03-1…5 |
| | AC2 | 03-6…11 |
| | AC3 | 03-12 |
| | AC4 | 03-13 |
| | AC5 | 03-14 |
| | AC6 | 03-15, 16 |
| | AC7 | 03-17 |
| | AC8 | 03-18 |
| US-04 | AC1 | 04-1, 2 |
| | AC2 | 04-3, 4 |
| | AC3 | 04-5, 6, 7 |
| | AC4 | 04-8, 9 |
| | AC5 | 04-10 |
| | AC6 | 04-11 |
| | AC7 | 04-12, 13, 14 |
| | AC8 | 04-15, 16, 17 |
| | AC9 | 04-18, 19 |
| | AC10 | 04-20 |
| | AC11 | 04-21 |
| | AC12 | 04-22 |
| US-21 | AC1 | 21-1, 2 |
| | AC2 | 21-3 |
| | AC3 | 21-4, 5, 6 |
| | AC4 | 21-7, 8 |
| | AC5 | 21-9 |
| | AC6 | 21-10, 11 |
