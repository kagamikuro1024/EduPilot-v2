# QC đợt 6 — Trang giới thiệu v2, clip và video (US-UX91-08, 09)

**Kết luận (kiểm lại lần 3, 2026-11-09, ở `9aea5b95`): PASS có giới hạn.** US-08 PASS (AC5 đã sửa). US-09 PASS có giới hạn (AC10: lỗi `run.sh` đã hết; chưa quay trọn được trên `ep-qc`, xem bên dưới). **AC10a của US-08: FAIL, ghi ở `wave-5.md` (TC-UX08-37).**
Lượt 1 (ở `62f4833b`) là FAIL; bảng lượt 1 giữ nguyên bên dưới làm lịch sử. Có **một đính chính** lượt 1: TC-UX08-18 ghi PASS «bỏ nguồn» nhưng phép kiểm khi đó yếu (`getAttribute("src") === null` luôn đúng vì nguồn nằm ở thẻ `<source>`); đã kiểm lại đúng, kết quả là FAIL bên dưới.

## Kiểm lại lần 3 ở `9aea5b95` (2026-11-09)
Stack `-p ep-qc` dựng lại từ `9aea5b95` (mã sản phẩm = `ed0aa499`, gồm `43f6212c` sửa QC lượt 2), Playwright `--workers=1` (stack thật) và harness `build:gate`. Spec `qc-uiux91-w6-real.spec.ts` (TC-UX08-16…18), chạy `scripts/landing-video/run.sh` trên VPS.
| TC | Trước | Nay | Ghi chú |
| --- | --- | --- | --- |
| UX08-16, 17, 18 (AC5) | **FAIL (Thấp)** | PASS | Mở: `currentSrc` = `…/gioi-thieu-v1.mp4`, 2 `<source>`. Đóng: `currentSrc` rỗng, 0 `<source>`, 0 request mới; Esc / bấm ngoài đều dừng. Mở lại: nguồn nạp lại. Focus trap, reduce mở tức thì giữ nguyên |
| UX09-16 (AC10) | **FAIL (Thấp)** | PASS có giới hạn | README có đủ ba cách chạy. `EP_HOST=127.0.0.1:1 STILLS_ONLY=1 run.sh` (không `EP_BASE_URL`): `record.mjs` nay dựng `https://127.0.0.1:1/login` rồi báo `ECONNREFUSED`, không còn «Cannot navigate to invalid URL». Chưa quay trọn: `EP_BASE_URL=https://localhost:14443` (Caddy `ep-qc`) báo `net::ERR_CERT_AUTHORITY_INVALID` (chứng chỉ tự ký; `record.mjs` không có `ignoreHTTPSErrors`), cổng HTTP của Caddy chuyển hướng về `https://127.0.0.1/` (cổng 443). Bản demo công khai (chứng chỉ hợp lệ) không chạm vào theo CONTEXT mục 4. Đề nghị (không chặn): README ghi stack tự ký cần chứng chỉ hợp lệ hoặc `record.mjs` nhận `ignoreHTTPSErrors` |

## Kiểm lại ở `6a0e8584` (2026-11-09)
Stack `-p ep-qc` dựng lại từ `6a0e8584` (frontend build lại), Playwright (`qc-uiux91-w6-real.spec.ts`, 27 ca, `--workers=1`), Lighthouse 12 devtools throttle 3 lượt × mobile + desktop, `ffprobe`, `curl`.
| TC | Trước | Nay | Ghi chú |
| --- | --- | --- | --- |
| UX08-11 | FAIL | PASS | Tab ẩn lúc tải: 0 request; hiện lại: 1 request clip, đang phát. Tab đang phát → ẩn → hiện: dừng rồi phát lại |
| UX08-16, 17, 18 | PASS | **FAIL (Thấp)** | Đóng bằng `Esc` / bấm ngoài: video dừng (đạt), nhưng **không bỏ nguồn**: sau khi đóng `currentSrc = …/landing/gioi-thieu-v1.mp4`, vẫn còn hai `<source src>` (AC5: «dừng và bỏ nguồn khi đóng»). Tái hiện: mở `Xem video`, `play()`, `Esc`, đọc `video.currentSrc` |
| UX08-29, 30, 31 | FAIL | PASS (spec mới) | Gắn cuộn đúng spec P-91-7 (3b): vạch `scaleX` 0,40 → 1 theo cuộn, cuộn ngược thì co lại; `--n` 0 → 77 → 146 theo cuộn, giữ 146; `reduce` hiện trạng thái cuối |
| UX08-33 | PASS | PASS | Không còn số «22» trong spec |
| UX08-38 | PASS | PASS | 1440: `h1` 64, `h2` 44, thân 18, giữa mục ≥ 106 px; 375: `h1` 41, `h2` 28, thân 16, ≥ 64 |
| UX08-39 | FAIL | Bỏ | P-91-8 (b) |
| UX08-43 | PASS | PASS | Mobile LCP **2,12 s** (2,07 / 2,12 / 2,19; phần tử LCP là `h1`), CLS 0,001, perf 93–97, a11y 100. Desktop LCP 0,15–0,20 s (ảnh poster), CLS 0, perf 100 |
| UX08-44 | PASS | PASS | JS gzip các chunk của `/` = **217.349 B** (11 chunk; lượt 1: 253.675 B) ≤ 256 KB; HTML không còn bảng email mẫu |
| UX08-25, AC13 | FAIL | PASS (spec mới) | Ngoại lệ số đếm `--n` và đổi màu nhỏ của sơ đồ (P-91-7 (3)); khối ghim chỉ `transform` / `opacity` |
| UX08-51 | FAIL | PASS | `/` cho khách: HTML và chunk không còn `@edupilot.local` / `Edupilot#` (`AuthedHome` nạp động) |
| UX08-24 | PASS | PASS | Ảnh khung ghim 1600 px: 48.646 / 90.202 / 48.052 B (≤ 92.160). Ghi chú: 90.202 B vượt 90.000 nếu «90 KB» tính theo 1000 |
| UX08-56 | PASS | PASS | Ảnh sau chụp lại: `shots/wave-6/` |
| UX09-4 | FAIL | PASS | Poster clip **58.526 B** (≤ 61.440); mp4 213.882 B, webm 399.980 B, 7,03 s |
| UX09-16 | FAIL | **FAIL (Thấp)** | Có `scripts/landing-video/README.md`. Nhưng lệnh «bản demo» của README (`ENV_FILE=/opt/edupilot/.env.local bash scripts/landing-video/run.sh`, tức `EP_HOST`) lỗi: `run.sh` truyền `-e EP_BASE_URL=""` vào container, `record.mjs` dùng `process.env.EP_BASE_URL ?? https://${EP_HOST}` (`??` không bắt chuỗi rỗng) nên `BASE = ""` → «Cannot navigate to invalid URL». Với `EP_BASE_URL=http://127.0.0.1:17000` (frontend ep-qc): đăng nhập quá hạn 20 s (cổng 17000 không có `/api`). Chưa kiểm được phần quay / mã hoá còn lại khi vượt lỗi này |
Các TC còn lại của đợt 6 chạy lại đều PASS (UX08-1…10, 12…15, 19…23, 26…28, 32, 34…37, 40…42, 45…50, 52…56, PQ; UX09-1…3, 5…15, 17…19, PQ). UX08-37 vẫn chờ gate đợt 5; UX09-20 chưa xem tay.

---
# Lượt 1 ở `62f4833b` (lịch sử)

## Môi trường
| Mục | Giá trị |
| --- | --- |
| Commit kiểm | `62f4833b` (nhánh `sprint/9.1-uiux`; mã đợt 6 là `09960dd1` + `6159c3e0`) |
| Hạ tầng | VPS `/srv/work/qc`, stack `-p ep-qc` dựng từ đúng commit trên (frontend build lại), Caddy `https://localhost:14443` (cùng đường đi như demo). Khách = context mới, không cookie. Playwright 1.63 Chromium; WebKit và Firefox của Playwright cho TC-28 |
| Công cụ | `frontend/e2e/qc-uiux91-w6-real.spec.ts` (`playwright.qc-real.config.ts`), Lighthouse 12 (devtools throttle, 3 lượt × mobile + desktop), `ffprobe` / `ffmpeg` trên VPS |
| Suite có sẵn | `landing.spec.ts` 13 ca pass; toàn suite 839 pass, 0 fail |
| Spec QC | 27 ca; lần chạy cuối pass trừ các ca là lỗi sản phẩm bên dưới (đã loại fail do QC giả định sai, xem «Sửa TC») |

## US-UX91-08 — Trang giới thiệu v2
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX08-1, 2 | 1 | PASS | `h1` «Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân», dòng dẫn, năm `h2` đúng thứ tự; `landing.spec.ts` khớp |
| UX08-3, 4 | 2 | PASS | Đúng 1 nút đỏ `Đăng nhập` (≈ rgb 208,0,22); `Tạo tài khoản`, `Xem video` trắng; `:active` → scale 0,97 |
| UX08-5 | 2 | PASS | 71 luật `:hover` toàn app, 0 ngoài `@media (hover: hover)`; `.frame:hover` nằm trong `(hover: hover) and (pointer: fine)` + `no-preference` |
| UX08-6, 7, 8 | 3 | PASS | Phần tử LCP là poster của clip; request clip sau `load`; `muted loop playsinline aria-hidden`, không `controls`; mp4 + webm; đang phát; mp4 213.882 B, webm 399.980 B (≤ 1.048.576) |
| UX08-9, 10 | 3 | PASS | `reduce` và `Save-Data`: không request clip |
| UX08-11 | 3 | **FAIL (Thấp)** | Tab đang phát → ẩn → hiện lại: dừng rồi phát lại (đạt). Nhưng tab **ẩn ngay lúc tải** rồi hiện: sau 2,5 s vẫn **0 request** clip, `paused` (AC3: «khi tab hiện lại thì clip tải (nếu chưa) và phát»). Tái hiện: `addInitScript` ép `document.hidden = true`, mở `/`, đổi sang `visible` + `visibilitychange` (bubbles) |
| UX08-12, 13 | 3 | PASS | Chặn clip: giữ poster, không lỗi console. 375 px: LCP là `h1` (xem AC13) |
| UX08-14, 15 | 4 | PASS | Nút `Tạm dừng` ≥ 44 × 44 px, `aria-pressed`, Enter dừng, poster hiện, tải lại vẫn dừng |
| UX08-16, 17, 18 | 5 | PASS | Trước khi bấm không request video đầy đủ; mở `dialog` có `<source>` mp4 + webm, phát thì có request; focus bị giữ, `Esc` đóng, focus về `Xem video`, đóng thì dừng và bỏ nguồn |
| UX08-19 | 5 | PASS | Video đầy đủ: `controls`, `preload="none"`, `playsinline`, không `autoplay`, không rãnh âm; `#video-loi` có trong DOM, ẩn trực quan |
| UX08-20 | 5 | PASS có ghi chú | Tổng JS gzip các chunk của `/` = 253.675 B (≤ 256 KB). Không so được độ chênh với v1 (216 KB) cùng phương pháp: dev đo 217 KB trên demo bằng cách khác |
| UX08-21 | 5 | PASS | `reduce`: hộp thoại mở tức thì |
| UX08-22, 23, 24 | 6 | PASS | ≥ 1100 px: khung `position: sticky`, ảnh đổi Chat → Duyệt bài → Sổ điểm theo cuộn, ghim hết khối thì cuộn đi; ba `h3` + một câu mỗi bước; ảnh 28.276 / 53.134 / 27.638 B (≤ 92.160), `width`/`height`, `loading=lazy`, `alt` riêng |
| UX08-25 | 6 | PASS | Trong khối ghim chỉ `transform` / `opacity` |
| UX08-26, 27, 28 | 7 | PASS | 1024 px và `reduce`: không ghim, đủ chữ + ảnh. Firefox (không có `animation-timeline`): không ghim, không khối ẩn. WebKit của Playwright có hỗ trợ nên không dùng làm ca «không hỗ trợ» |
| UX08-29, 30 | 8 | **FAIL (Thấp)** | Sơ đồ và vạch đỏ gắn với **vị trí cuộn** (view timeline): vạch `scaleX` 0,22 → 1 trong ~400 px cuộn, cuộn ngược thì co lại (0,22 ở −500 px). AC8 ghi «lệch ≤ 120 ms, mỗi bước ≤ 400 ms… một lần mỗi lượt xem» (thời gian, không lặp). Chữ ba bước là HTML thật, vạch `aria-hidden` (đạt) |
| UX08-31 | 9 | **FAIL (Thấp)** | Số đếm gắn cuộn: `--n` 0 → 77 → 146 trong ~100 px cuộn rồi giữ 146; không phải «≤ 900 ms, một lần». Giá trị thật (146) nằm sẵn trong DOM |
| UX08-32, 34 | 9 | PASS | Giá trị thật trong DOM; đúng một dòng «Đo trên dữ liệu mô phỏng.» |
| UX08-33 | 9 | PASS có ghi chú | Ba câu khớp nguồn: «146 yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp» + «Bộ phát hiện đạt recall 0,960 trên 200 mẫu tự soạn», «30/30 sinh viên mẫu có điểm học phần khớp tính tay», «Chấm hàng loạt: thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 %» (dấu `:` thay `,`). Không có «mọi / toàn bộ / độ trễ chat». Bỏ số «22 bài / 20 sinh viên»: `9/report.md` không có (đúng yêu cầu «đối chiếu trước khi giữ»; P-91-7) |
| UX08-35, 36 | 10 | PASS | Hai cột «Giảng viên» / «Sinh viên» cùng `top` (5300), `left` 160 / 740, mỗi vai 4 mục, không tab, không «Threads»; 700 px xếp dọc, không tràn ngang |
| UX08-37 | 10a | CHƯA ĐẾN HẠN | Kiểm ở gate đợt 5 |
| UX08-38 | 11 | PASS | 1440: `h1` 64, `h2` 44, đoạn thân 18 px, cách mục ≥ 96; 375: `h1` ≥ 36, `h2` ≥ 28, đoạn thân 16 px, cách mục ≥ 64. Chữ chú thích 12–14 px (nhãn đo, chân trang, bảng minh hoạ) không tính là chữ thân |
| UX08-39 | 11 | **FAIL (Thấp)** | Cuộn từng `h2` tới đầu khung 1440 × 900: ba trong năm lần còn một `h2` khác nằm trong khung: «AI soạn nháp…» + «Khác biệt, có số đo» (y 750); «Khác biệt…» + «Cho giảng viên…» (y 632); «Cho giảng viên…» + «Bắt đầu…» (y 810) |
| UX08-40, 41, 42 | 12 | PASS | Bóng khung `0 10px 30px rgba(71,32,37,.12)` = `--ep-shadow-popover`; nâng `translateY(-6px)` 180 ms chỉ trong `(hover: hover) and (pointer: fine)` + `no-preference`; cảm ứng không hover; `reduce` không nâng |
| UX08-43 | 13 | PASS | Lighthouse mobile (devtools throttle) LCP 2,43 / 2,39 / 2,39 s (trung vị 2,39 s; phần tử LCP là `h1`), CLS 0,001, perf 97, a11y 100. Desktop LCP 0,16–0,18 s, CLS 0, perf 100. Giới hạn: mobile còn 0,11 s so với ngưỡng 2,5 s |
| UX08-44, 45 | 13 | PASS | JS 253,7 KB ≤ 256 KB; `package.json` không đổi; HTML máy chủ đã có `h1` |
| (AC13) | 13 | **FAIL (Thấp)** | «Chỉ `transform` / `opacity` được animate»: `Landing.module.css` có `@keyframes mark-on` (`border-color`, `color`), `strike` (`text-decoration-color`), `count` (`--n`). Đo thật: `getAnimations()` ra `borderBottomColor`, `color`, `textDecorationColor`. `count` là cơ chế số đếm (P-91-7) |
| UX08-46, 47 | 14 | PASS | `reduce`: không clip, 0 animation, số cuối (146) ngay; mọi `animation-timeline` / `view-timeline` / `timeline-scope` nằm trong `@media (prefers-reduced-motion: no-preference) { @supports (animation-timeline: view()) … }` |
| UX08-48, 49 | 15 | PASS | `javaScriptEnabled: false` (cuộn hết trang) và Firefox: không khối ẩn vĩnh viễn; không `IntersectionObserver`, không `data-anim` |
| UX08-50 | 16 | PASS | axe 1440 và 375: 0 serious / critical; `h1 → h2 → h3` không nhảy cấp; mọi `img` có `alt` |
| UX08-51 | 17 | **FAIL (Thấp)** | HTML sạch. Chunk JS của `/` (`/_next/static/chunks/0q6ro7buziooj.js`) chứa bảng `frontend/src/mock/identity.ts`: 7 email mẫu (`sv.gioi@edupilot.local`, … `admin@edupilot.local`), không mật khẩu. Có từ `84158de9` (US-P2-02), không phải do đợt 6. AC17 yêu cầu 0 kết quả. Có dòng «Dữ liệu trên bản demo là dữ liệu mô phỏng.» |
| UX08-52 | 18 | PASS | Đã đăng nhập mở `/` → Hôm nay, `MutationObserver` không thấy trang giới thiệu dù một frame |
| UX08-53 | 19 | PASS | Chặn ảnh khung ghim: khung giữ 623 × 390; chặn clip: poster; chặn video đầy đủ rồi `Xem video`: có «Mở video trong tab mới», không lỗi console |
| UX08-54 | 20 | PASS | `Đăng nhập` → `/login`, `Tạo tài khoản` → `/register` (màn đầu và CTA cuối) |
| UX08-55 | 21 | PASS | `landing.spec.ts`: 13 ca (clip, `Tạm dừng`, `Xem video`, ghim 1440, xếp dọc 1024, vai, axe) |
| UX08-56 | 21 | PASS | 14 ảnh `shots/wave-6/landing-{1440,390}-*.png` (đầu, ghim 1–3, sơ đồ, số đo, vai, CTA). 390 px: khoảng trống ~300 px giữa nút và ảnh (dev đã nêu) |
| UX08-PQ | — | PASS | `/chat`, `/gradebook` khi chưa đăng nhập → `/login?next=`; HTML trang giới thiệu không có dữ liệu cá nhân / token |

## US-UX91-09 — Clip và video đầy đủ
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX09-1, 2 | 1 | PASS | `gioi-thieu-v1.mp4` 64,73 s (60–90), 1440 × 900, H.264, **không** luồng audio; webm VP9 cùng thông số |
| UX09-3 | 1 | PASS | Khung mỗi 3 s: chat (tên và MSSV được che, nguồn) → hộp thư → chấm bài (AI chấm nháp, đoạn trích) → sổ điểm → giải trình từng dòng → Hôm nay → khung kết. Phụ đề khớp `#video-loi` |
| UX09-4 | 2 | **FAIL (Thấp)** | Poster clip `gioi-thieu-clip-v1.webp` **64.030 B** > 61.440 B (60 KB). Chạy lại `run.sh` ra 64.080 B. Clip 7,03 s, mp4 213.882 B, webm 399.980 B đạt. Dev đã tự khai vượt 4 KB |
| UX09-5 | 2 | PASS | SSIM khung đầu / cuối 0,994 (≥ 0,90). Khung đầu và cuối cùng một cảnh, không phụ đề; có vòng tròn đỏ (con trỏ tự vẽ) đứng yên ở đáy khung |
| UX09-6 | 2 | PASS | `ftyp, moov, free, mdat` (`+faststart`), H.264 |
| UX09-7 | 3 | PASS | mp4 1,11 MB, webm 1,64 MB, poster 52,9 KB; đủ ba tệp |
| UX09-8 | 4 | PASS | `controls`, `preload="none"`, `playsinline`, có `poster`, không `autoplay`; không request trước khi bấm |
| UX09-9 | 5 | PASS | Poster có `<link rel=preload as=image media="(min-width: 720px)">`; LCP theo UX08-43 |
| UX09-10 | 6 | PASS | `#video-loi` có đủ 7 câu phụ đề theo thứ tự, `aria-describedby="video-loi"` |
| UX09-11 | 7 | PASS | 375 px cảm ứng: video 337 px (không tràn), điều khiển gốc |
| UX09-12 | 8 | PASS | `Range: bytes=0-99` → `206`, `accept-ranges: bytes`, `content-range` cho cả bốn tệp |
| UX09-13 | 8 | PASS | Chromium và WebKit: `currentTime = 30` → phát tiếp (31,9 s, không dừng) |
| UX09-14 | 9 | PASS | `record.mjs`: chỉ đăng nhập, chọn lớp, mở dòng / phiên / giải trình; không gửi chat, duyệt, công bố, POST |
| UX09-15 | 9 | PASS | Khung hình: tên và MSSV mô phỏng, không email thật, khoá, URL nội bộ |
| UX09-16 | 10 | **FAIL (Thấp)** | Không có README trong `scripts/landing-video/` (chỉ có chú thích đầu `run.sh`). `run.sh` chạy được (video 64,87 s / 1,11 MB, clip 6,97 s / 212 KB / 407 KB) nhưng cứng host bản demo và đọc `/opt/edupilot/.env.local`, không trỏ được `ep-qc` |
| UX09-17, 18 | 11 | PASS | `canPlayType('video/webm')` rỗng → phát MP4; cả hai rỗng → hộp thoại có liên kết tải, không `console.error` |
| UX09-19 | 12 | PASS | `reduce`: clip không phát; video đầy đủ không tự phát |
| UX09-20 | 13 | CHƯA XEM TAY | Hai vòng liền: SSIM 0,994 gợi ý liền mạch; chưa xem tay |
| UX09-PQ | — | PASS | `/landing/../etc/passwd`, `/edupilot/x`, `/landing/%2e%2e/package.json` → trang HTML của ứng dụng, không tệp nào lộ |

## Sửa TC (ghi lý do)
- TC-UX08-3: đọc màu qua canvas (`lab(...)`, không phải `rgb`).
- TC-UX08-11: tách hai kịch bản (đang phát → ẩn → hiện; ẩn lúc tải → hiện). Cả hai theo AC3.
- TC-UX08-16: `preload="none"` không tự request khi mở hộp thoại; kiểm `<source>` có nguồn sau khi bấm, rồi request khi phát.
- TC-UX08-22: chọn ảnh trên cùng trong chồng ảnh (ba ảnh xếp chồng, `opacity` ảnh 1 luôn 1).
- TC-UX08-33: dấu `,` / `:` đều chấp nhận. TC-UX08-38: chữ thân là đoạn > 60 ký tự ngoài sơ đồ, chú thích, chân trang.
- TC-UX08-53: ảnh trùng cho < 1100 px là `display: none` (0 × 0), chỉ tính ảnh đang hiện.

## Việc dev nêu mà QC xác nhận
Số «22» bỏ đúng; poster clip vượt 4 KB (FAIL UX09-4); khoảng trống 300 px ở < 720 px (không phải lỗi AC); LCP mô phỏng dao động (đã dùng throttle thật).

## File
Chạy lại: `QC_REAL=1 pnpm exec playwright test -c playwright.qc-real.config.ts <spec> --project=desktop` (không đặt `QC_REAL` thì spec `*-real` tự bỏ qua, không lẫn vào suite CI).
Spec: `frontend/e2e/qc-uiux91-w6-real.spec.ts`, `qc-uiux91-w6-shots-real.spec.ts`. Ảnh: `docs/sprints/9.1/qc/shots/wave-6/`. Log VPS: `/srv/work/qc/out/{w6b,w6c,w6d,w6e,w6f,w6-lh.txt,w6-media.txt,runsh.log}`.
