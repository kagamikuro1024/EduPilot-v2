# TC đợt 6 — Trang giới thiệu v2, clip và video (US-UX91-08, 09)

Phiên bản 1 · 2026-11-08. **US-UX91-22 (Dùng thử) đã bị bỏ (P-91-4): không có TC và không có `tryout.spec.ts`.** Nút màn đầu chỉ gồm `Đăng nhập` (đỏ), `Tạo tài khoản`, `Xem video`.
Quy ước chung và kiểm chung: `tc-00-chung.md`. Trang: `/` ở trạng thái khách (context mới).
Công cụ riêng: Chromium (mặc định); **WebKit và Firefox** của Playwright (có sẵn trong `~/Library/Caches/ms-playwright` và trên VPS) cho ca "không hỗ trợ `animation-timeline`" (AC15) và "WebM" (US-09 AC11); `ffprobe` / `ffmpeg` trên VPS cho video.

Nguồn đối chiếu số liệu (đọc được trong repo): `docs/sprints/6/report.md` dòng 34–35, `docs/sprints/8/report.md` dòng 24, `docs/sprints/9/report.md` dòng 25.

---
## US-UX91-08 — Trang giới thiệu v2

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX08-1 | AC1 | khách, 1440 | Mở `/`; đọc `h1`, dòng dẫn, danh sách `h2` theo thứ tự DOM | `h1` «Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân»; dòng dẫn «AI soạn nháp. Giảng viên quyết. Điểm do mã tính.»; `h2` đúng thứ tự: «Xem EduPilot làm việc» → «AI soạn nháp, giảng viên quyết, mã tính điểm» → «Khác biệt, có số đo» → «Cho giảng viên và sinh viên» → «Bắt đầu với lớp của bạn» | E2E |
| TC-UX08-2 | AC1 | khách | `landing.spec.ts` | Danh sách `h2` trong spec khớp đúng TC-UX08-1 | GREP |
| TC-UX08-3 | AC2 | khách, 1440 | Đếm nút / liên kết dạng nút ở màn đầu (trong 900 px đầu); đọc `background-color` | Đúng **1** nút đỏ (`Đăng nhập`); `Tạo tài khoản`, `Xem video` là nút phụ (≤ 2 nút phụ). | E2E |
| TC-UX08-4 | AC2 | khách | `mouse.down` lên mỗi nút | `transform` ≈ scale 0,97 | E2E |
| TC-UX08-5 | AC2 | khách | Quét CSS của trang: luật `:hover` có nằm trong `@media (hover: hover)`; trên `hasTouch` + `isMobile` chạm nút | Hover chỉ đổi màu khi `hover: hover`; cảm ứng không hover dính | E2E + GREP |
| TC-UX08-6 | AC3 | khách, 1440 | `PerformanceObserver('largest-contentful-paint')` lấy phần tử LCP; ghi thời điểm request clip so với sự kiện `load` (`performance.timing`, `resource timing`) | Phần tử LCP là **poster** ảnh của clip (hoặc `h1` nếu AC9-5 cho phép); request `.mp4` / `.webm` của clip bắt đầu **sau** `load` | E2E |
| TC-UX08-7 | AC3 | khách | Đọc thuộc tính `<video>` của clip | `muted`, `loop`, `playsinline`, `aria-hidden="true"`, không `controls`; có hai `<source>` mp4 + webm | E2E |
| TC-UX08-8 | AC3 | khách | Sau 3 s: `video.paused`, `currentTime` tăng; `HEAD` hai tệp clip | Đang phát, lặp; mỗi tệp ≤ 1.048.576 byte | E2E + FFP |
| TC-UX08-9 | AC3 | khách, `reduce` | Mở `/` ghi request | **Không** có request tới clip; chỉ poster | E2E |
| TC-UX08-10 | AC3 | khách | `addInitScript`: `navigator.connection = { saveData: true }` | Không tải clip, chỉ poster | E2E |
| TC-UX08-11 | AC3 | khách | `addInitScript` ép `document.visibilityState = 'hidden'` trước khi tải; sau đó đổi sang `visible` + `visibilitychange` | Hidden → không tải / không phát; khi tab hiện lại → clip tải và phát (trừ `reduce` / `Save-Data`, P-91-5 Q5) | E2E |
| TC-UX08-12 | AC3 | khách | `route.abort()` mọi tệp clip | Poster giữ nguyên; không `console.error` / `pageerror` | E2E |
| TC-UX08-13 | AC3 | khách, 375 | Mở `/` | Poster tĩnh hoặc clip dưới nút; đạt TC-UX08-31 (LCP) | E2E |
| TC-UX08-14 | AC4 | khách, 1440 | Đợi 6 s; tìm nút `Tạm dừng`; đo kích thước; Tab tới + Enter | Nút hiện, ≥ 44 × 44 px, `aria-pressed`, dùng được bằng bàn phím; bấm → clip dừng, poster hiện lại, nhãn đổi `Phát` | E2E |
| TC-UX08-15 | AC4 | khách | Dừng clip; tải lại trang (cùng phiên) | Trạng thái dừng được nhớ trong phiên (clip không tự phát lại); tab mới → mặc định phát | E2E |
| TC-UX08-16 | AC5 | khách | Ghi request; bấm `Xem video` | Trước khi bấm **không** có request tới `gioi-thieu-v1.mp4 / .webm`; sau bấm: hộp thoại (`dialog[open]` hoặc `role=dialog`) mở và video có nguồn | E2E |
| TC-UX08-17 | AC5 | khách | Hộp thoại mở: Tab nhiều lần; `Esc` | Focus bị giữ trong hộp thoại (không ra ngoài); `Esc` đóng; focus **về nút** `Xem video` | E2E |
| TC-UX08-18 | AC5 | khách | Phát vài giây rồi đóng | `video.paused`; `video.currentSrc` rỗng / không còn nguồn (đã bỏ nguồn) | E2E |
| TC-UX08-19 | AC5 | khách | `ffprobe` tệp video đầy đủ; đọc `#video-loi` | Không rãnh âm (`codec_type=audio` không có); `#video-loi` có trong DOM | FFP + E2E |
| TC-UX08-20 | AC5 | khách | Đo JS route `/` (tổng gzip các chunk `.js` tải ở `/`) trước (v1: 216 KB, `audit/data/landing.json`) và sau | Chênh lệch ≤ 1,5 KB (≤ 0,5 KB nội tuyến + ≤ 1 KB mở video); tổng ≤ 256 KB | E2E |
| TC-UX08-21 | AC5 | khách, `reduce` | Bấm `Xem video` | Hộp thoại mở **tức thì** (hoạt cảnh ≤ 1 ms) | E2E |
| TC-UX08-22 | AC6 | khách, 1440 | Cuộn từng bước 120 px qua khối «Xem EduPilot làm việc»; mỗi bước đọc `position` khung ghim, `opacity` của ba ảnh, `h3` bước đang chiếm ≥ 60 % khung nhìn | Khung `position: sticky` ghim một bên; ba `h3` + mỗi bước một câu; bước *i* ≥ 60 % → ảnh *i* hiện (Chat → Duyệt bài → Sổ điểm); chuyển ≤ 200 ms hoặc theo cuộn; hết khối thì không ghim nữa (khung cuộn đi) | E2E |
| TC-UX08-23 | AC6 | khách | Nội dung ba bước | (1) chat riêng có nguồn và dòng «đã ẩn thông tin cá nhân»; (2) duyệt bài AI chấm nháp; (3) sổ điểm có giải trình. Kiểm chữ + ảnh tương ứng | E2E + TAY |
| TC-UX08-24 | AC6 | khách | `HEAD` ba ảnh WebP; thuộc tính `<img>` | Mỗi ảnh ≤ 92.160 byte, có `width` / `height`, `loading="lazy"`, `alt` riêng khác nhau; xem ảnh: chữ trong ảnh đọc được ở 100 % | FFP + TAY |
| TC-UX08-25 | AC6 | khách | Mới tải trang, đo | Chỉ `transform` / `opacity` được animate trong khối (quét `getAnimations()`) | E2E |
| TC-UX08-26 | AC7 | khách, 1024 | Mở khối | Không ghim (`position` ≠ sticky); ba bước xếp dọc, mỗi bước kèm ảnh; không mất chữ / ảnh | E2E |
| TC-UX08-27 | AC7 | khách, `reduce` ở 1440 | Như trên | Không ghim; trạng thái cuối; đủ chữ + ảnh | E2E |
| TC-UX08-28 | AC7, AC15 | khách | Firefox hoặc WebKit (không `animation-timeline`; xác nhận bằng `CSS.supports('animation-timeline: view()')` = false) ở 1440 | Không ghim, trang hiện đủ nội dung tĩnh | E2E |
| TC-UX08-29 | AC8 | khách, 1440 | Cuộn tới khối sơ đồ; ghi `startTime` của `getAnimations()` ba bước và vạch | Ba bước hiện lần lượt, lệch ≤ 120 ms, mỗi bước ≤ 400 ms; vạch đỏ `transform: scaleX` (375 px: `scaleY`) | E2E |
| TC-UX08-30 | AC8 | khách | Cuộn đi rồi cuộn lại khối | **Không** chạy lại (một lần mỗi lượt xem); chữ ba bước là HTML thật (tìm được bằng `getByText`), vạch `aria-hidden`; đỏ chỉ ở vạch nối (quét `color` / `background` các phần tử khác không đỏ) | E2E |
| TC-UX08-31 | AC9 | khách, 1440 | Cuộn tới «Khác biệt, có số đo»; lấy mẫu giá trị hiển thị mỗi 50 ms | «146», «22» đếm từ 0 tới giá trị cuối ≤ 900 ms, một lần; «30/30» và «0,05 %» hiện tĩnh | E2E |
| TC-UX08-32 | AC9 | khách | Đọc DOM | Giá trị thật nằm trong DOM; bản đếm `aria-hidden="true"`; trình đọc nghe giá trị cuối | E2E |
| TC-UX08-33 | AC9 | — | Đối chiếu chữ ba câu với nguồn: (1) «146 yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp» + «bộ phát hiện đạt recall 0,960 trên 200 mẫu tự soạn» ↔ `6/report.md` dòng 34, 35; (2) «30/30 sinh viên mẫu có điểm học phần khớp tính tay» ↔ `8/report.md` dòng 24; (3) «Chấm hàng loạt, thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 %» ↔ `9/report.md` dòng 25; số «22 bài / 20 sinh viên» có trong báo cáo | Từng chữ khớp nguồn; **không** có «mọi», «toàn bộ», «được che trước mọi lời gọi AI», «độ trễ chat» chung chung | E2E + TAY |
| TC-UX08-34 | AC9 | khách | Tìm dòng «Đo trên dữ liệu mô phỏng.» dưới khối | Có đúng một dòng chú thích | E2E |
| TC-UX08-35 | AC10 | khách, 1440 | Quét khối «Cho giảng viên và sinh viên» | Hai cột «Giảng viên» / «Sinh viên» cạnh nhau (cùng `top`, `left` khác nhau); mỗi vai ≤ 4 mục, mỗi mục một câu; **không** `role=tablist` / `tab`; không `button` đổi vai; tên mục dùng «Hỏi đáp công khai», không «Threads» | E2E |
| TC-UX08-36 | AC10 | khách, 375 và 719 | Xếp dọc ở < 720 px | Hai cột xếp dọc; không tràn ngang | E2E |
| TC-UX08-37 | AC10a | — | Kiểm ở **đợt 5** (gate): handoff đợt 5 ghi ba ảnh khung ghim, clip, poster chụp lại từ UI mới | Ba ảnh khung ghim khớp UI sau đợt 3–5 (so ảnh `chat`, `grading-review`, `gradebook` hiện hành); nếu không → FAIL AC10a ở wave-5 | TAY |
| TC-UX08-38 | AC11 | khách, 1440 và 375 | `getComputedStyle` cho `h1`, `h2`, chữ thân, khoảng cách dọc giữa hai mục | 1440: `h1` ≥ 56, `h2` ≥ 40, thân ≥ 18, khoảng ≥ 96 px; 375: ≥ 36 / 28 / 16 / 64 | E2E |
| TC-UX08-39 | AC11 | khách, 1440×900 | Cuộn từng `h2` tới đầu khung nhìn; đếm `h2` nằm trong khung | Chỉ 1 `h2` trong khung | E2E |
| TC-UX08-40 | AC12 | khách, 1440 | Đọc `box-shadow` khung ảnh; so với bóng của popover (`getComputedStyle` menu mở ở ứng dụng hoặc biến `--ep-*` tương ứng); rê chuột | Cùng họ bóng với popover (§7), không màu mới; rê → nâng ≤ 6 px bằng `translateY` trong ≤ 200 ms | E2E |
| TC-UX08-41 | AC12 | khách, `isMobile` + `hasTouch` | Chạm khung | Không hiệu ứng hover; Tab → vòng focus thấy được ở khung / nút; không glow, nghiêng, kính mờ (`backdrop-filter` = none, `rotate` / `perspective` = none) | E2E |
| TC-UX08-42 | AC12 | khách, `reduce` | Rê chuột | Không nâng | E2E |
| TC-UX08-43 | AC13 | khách | `lighthouse` mobile (4G chậm thật, 375 px), ≥ 3 lượt, lấy trung vị; lặp desktop 1440 | LCP ≤ 2,5 s ở cả hai; CLS ≤ 0,1; ghi từng lượt | LH |
| TC-UX08-44 | AC13 | khách | Tổng JS gzip route `/` (Network) | ≤ 256 KB | E2E |
| TC-UX08-45 | AC13 | — | `git diff 1b72f54... -- frontend/package.json`; `curl -s /` (không JS) | Không thêm thư viện; HTML đã chứa `h1` và nội dung chính (Server Component) | GREP |
| TC-UX08-46 | AC14 | khách, `reduce` | Mở và cuộn hết trang; đọc DOM | Không clip chạy (chỉ poster); sơ đồ đủ ba bước; số hiện giá trị cuối ngay (146, 22); bước xếp dọc không ghim; `document.getAnimations()` = 0 | E2E |
| TC-UX08-47 | AC14 | — | Quét CSS landing: mọi `animation-timeline` / hiệu ứng theo cuộn nằm trong `@media (prefers-reduced-motion: no-preference) { @supports (animation-timeline: view()) { … } }` | Không luật nào ngoài khối đó | GREP |
| TC-UX08-48 | AC15 | khách | Firefox / WebKit không hỗ trợ; ngoài ra `javaScriptEnabled: false` ở Chromium | Hiện đủ nội dung, không khối ẩn vĩnh viễn (`opacity: 0` / `visibility: hidden`) | E2E |
| TC-UX08-49 | AC15 | — | `rg -n "IntersectionObserver" frontend/src/features/landing frontend/src/app/page.tsx`; tìm cờ `data-anim` | Không `IntersectionObserver` dự phòng cho hiệu ứng theo cuộn; trạng thái ẩn ban đầu chỉ áp khi `[data-anim]` do script nội tuyến đặt | GREP |
| TC-UX08-50 | AC16 | khách, 1440 và 375 | axe (K1); thứ bậc `h1 → h2 → h3`; Tab qua `Xem video`, `Tạm dừng`, đóng hộp thoại; mọi `img` có `alt` | 0 serious / critical; thứ bậc đúng; dùng được bằng bàn phím; focus ring thấy được; không truyền nghĩa chỉ bằng màu | E2E |
| TC-UX08-51 | AC17 | khách | `curl -s /` rồi `rg "@edupilot.local\|Edupilot#\|mật khẩu mặc định"` (cả HTML + chunk JS của route) | 0 kết quả; còn dòng «Dữ liệu trên bản demo là dữ liệu mô phỏng.», «Đồ án tốt nghiệp · Lê Văn Quang Trung», liên kết GitHub | E2E |
| TC-UX08-52 | AC18 | giảng viên, SV | Đăng nhập; mở `/` (cờ `RETURNING_KEY`) | Là «Hôm nay», không thấy trang giới thiệu dù một frame (bắt bằng `MutationObserver` ở `document-start`) | E2E |
| TC-UX08-53 | AC19 | khách | `route.abort()` một ảnh khung ghim; abort clip; abort video đầy đủ rồi bấm `Xem video` | Khung giữ kích thước (CLS ≤ 0,02 từ lỗi), có `alt`; clip lỗi → poster; video lỗi → hộp thoại hiện poster + liên kết «Mở video trong tab mới»; không lỗi console | E2E |
| TC-UX08-54 | AC20 | khách | Bấm `Đăng nhập`, `Tạo tài khoản` ở màn đầu và CTA cuối | Tới `/login`, `/register` | E2E |
| TC-UX08-55 | AC21 | — | `landing.spec.ts` diff; chạy spec | Có: clip không tải khi `reduce` / `Save-Data`; nút `Tạm dừng`; `Xem video` mở / đóng, không request video trước bấm; ghim ở 1440, xếp dọc ở 1024 / `reduce`; hai cột vai (không tab); số cuối khi `reduce`; ảnh mốc `landing-1440`, `landing-390` (khách, `reduce`, đồng hồ đóng băng). Spec xanh, không nới | E2E |
| TC-UX08-56 | AC21 | — | Chụp `landing-1440`, `landing-390` trước / sau + 4 ảnh cuộn (đầu, ghim 1, ghim 3, số liệu) ở `reduce=no-preference` | Đủ ảnh ở `shots/wave-6/`; bản đối chiếu với `audit/shots/home__anon__*` | MAC |
| TC-UX08-PQ | (kiểm chéo) | khách | `/chat`, `/gradebook` | Chuyển `/login?next=…` (không lộ nội dung); trang giới thiệu không có dữ liệu cá nhân / email / token trong HTML | E2E |

---
## US-UX91-09 — Clip và video đầy đủ

| TC | AC | Cách | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX09-1 | AC1 | — | `ffprobe -v error -show_entries format=duration -of csv=p=0 gioi-thieu-v1.mp4` | 60 ≤ giây ≤ 90 | FFP |
| TC-UX09-2 | AC1 | — | `ffprobe` luồng | Độ phân giải 1440 × 900 (hoặc 1440×900 trong khung); **không** luồng audio | FFP |
| TC-UX09-3 | AC1, AC13 | — | Xem toàn bộ video; trích khung mỗi 3 s (`ffmpeg -ss … -frames:v 1`) | Trình tự: chat (tên được che) → hộp thư → duyệt bài AI chấm → sổ điểm giải trình → Hôm nay; con trỏ hiện; phụ đề tiếng Việt in vào hình, dấu đúng, không tràn, không trùng cảnh; ghi giây lỗi | TAY |
| TC-UX09-4 | AC2 | — | `ffprobe` clip mp4 + webm | 6 ≤ giây ≤ 8; cùng nguồn quay; kích thước ≤ 1.048.576 byte mỗi tệp; poster `.webp` ≤ 61.440 byte; tên `gioi-thieu-clip-v1.mp4 / .webm / .webp` | FFP |
| TC-UX09-5 | AC2 | — | Trích khung đầu và khung cuối clip; tính chênh lệch (`ffmpeg … ssim`) | SSIM ≥ 0,90 (lặp liền mạch); xem tay: không con trỏ giật giữa chừng, không phụ đề | FFP + TAY |
| TC-UX09-6 | AC2 | — | Kiểm cấu trúc mp4: `moov` đứng trước `mdat`; codec | H.264, `+faststart` | FFP |
| TC-UX09-7 | AC3 | — | `ls -la frontend/public/landing/`; `curl -I` ba URL video đầy đủ | Có `gioi-thieu-v1.mp4`, `.webm`, `.webp`; mỗi định dạng < 8 MB; 200 | FFP |
| TC-UX09-8 | AC4 | khách | Đọc thẻ `<video>` của video đầy đủ | Có `controls`, `preload="none"`, `playsinline`, `poster`; **không** `autoplay`; không request video trước khi bấm | E2E |
| TC-UX09-9 | AC5 | khách, 1440 + 375 | `lighthouse` / `PerformanceObserver`; `<link rel=preload as=image>` của poster ở ≥ 720 px | LCP ≤ 2,5 s cả hai; poster ≤ 60 KB; ≥ 720 px có preload ảnh; < 720 px poster không phải LCP hoặc LCP đạt | LH + E2E |
| TC-UX09-10 | AC6 | khách | Đọc `#video-loi` và `aria-describedby`; so từng câu với phụ đề trong video (TC-UX09-3) | Có đủ các câu phụ đề **theo thứ tự**, khớp phụ đề in; ẩn trực quan (không hiện trên UI); hộp thoại / `<video>` `aria-describedby="video-loi"` | E2E + TAY |
| TC-UX09-11 | AC7 | khách, 375, `hasTouch` | Mở hộp thoại; Tab / chạm điều khiển gốc (phát / dừng / tua) | Điều khiển gốc dùng được; vùng chạm ≥ 44 px; video co theo bề rộng, không tràn ngang | E2E |
| TC-UX09-12 | AC8 | — | `curl -s -D- -o /dev/null -H 'Range: bytes=0-99' <mp4>` và `<webm>` | `206`, `accept-ranges: bytes`, `content-range` | FFP |
| TC-UX09-13 | AC8 | khách | Chromium + WebKit: phát rồi `video.currentTime = 30` | Phát tiếp từ 30 s (không quay về 0, không treo) | E2E |
| TC-UX09-14 | AC9 | — | Đọc `scripts/landing-video/` (chỉ để kiểm an toàn): liệt kê hành động ghi (`click` nút gửi / duyệt / công bố, `POST`, `fill` rồi Enter) | Chỉ đọc trên demo hoặc chạy trên stack riêng `-p ep-video`; không gửi chat, không duyệt / công bố / sửa điểm; chat mở lại hội thoại seed | GREP |
| TC-UX09-15 | AC9 | — | Xem khung hình: tìm email thật, MSSV thật, khoá, URL nội bộ (`localhost`, `.local`, IP, `9router`) | Không có; tên là dữ liệu mô phỏng (D44) | TAY |
| TC-UX09-16 | AC10 | VPS | `bash scripts/landing-video/run.sh` (trỏ vào stack `ep-qc`); đọc README | Sinh lại clip + video đầy đủ + poster, cùng quy cách (thời lượng, dung lượng); README nêu lệnh | FFP |
| TC-UX09-17 | AC11 | khách | `addInitScript` ép `canPlayType('video/webm')` = `''` | Rơi về MP4, phát được | E2E |
| TC-UX09-18 | AC11 | khách | Ép `canPlayType` rỗng cho cả webm và mp4 | Poster + liên kết tải; không `console.error` | E2E |
| TC-UX09-19 | AC12 | khách, `reduce` | Mở `/`; bấm `Xem video` | Clip không phát (poster); video đầy đủ không tự phát; không chuyển động quanh video | E2E |
| TC-UX09-20 | AC13 | — | Xem hết clip hai vòng liền | Không giật khi lặp; ghi giây lỗi | TAY |
| TC-UX09-PQ | (kiểm chéo) | khách | `curl -I` tệp video bằng đường dẫn đoán trước (`/landing/../…`, `/edupilot/…`) | Chỉ phục vụ tệp công khai ở `public/landing/`; không đi tới MinIO / tệp riêng | FFP |

---
## Bảng phủ AC đợt 6
| Story | AC | TC |
| --- | --- | --- |
| US-08 | AC1 | 08-1, 2 |
| | AC2 | 08-3, 4, 5 |
| | AC3 | 08-6…13 |
| | AC4 | 08-14, 15 |
| | AC5 | 08-16…21 |
| | AC6 | 08-22…25 |
| | AC7 | 08-26, 27, 28 |
| | AC8 | 08-29, 30 |
| | AC9 | 08-31…34 |
| | AC10 | 08-35, 36 |
| | AC10a | 08-37 (kiểm ở wave-5) |
| | AC11 | 08-38, 39 |
| | AC12 | 08-40, 41, 42 |
| | AC13 | 08-43, 44, 45 |
| | AC14 | 08-46, 47 |
| | AC15 | 08-28, 48, 49 |
| | AC16 | 08-50 |
| | AC17 | 08-51 |
| | AC18 | 08-52 |
| | AC19 | 08-53 |
| | AC20 | 08-54 |
| | AC21 | 08-55, 56 |
| US-09 | AC1 | 09-1, 2, 3 |
| | AC2 | 09-4, 5, 6 |
| | AC3 | 09-7 |
| | AC4 | 09-8 |
| | AC5 | 09-9 |
| | AC6 | 09-10 |
| | AC7 | 09-11 |
| | AC8 | 09-12, 13 |
| | AC9 | 09-14, 15 |
| | AC10 | 09-16 |
| | AC11 | 09-17, 18 |
| | AC12 | 09-19 |
| | AC13 | 09-3, 20 |
