# Sprint 9.1: câu hỏi kỹ thuật cho Tech Lead

## TL-1 (pha 1: khả thi kỹ thuật, PM giao 2026-11-08)

**Câu hỏi (PM):** ý kiến kỹ thuật về (A) cách dựng video giới thiệu bằng Playwright + ffmpeg trên VPS, (B) chuyển động không thư viện cho trang giới thiệu, theo hướng chủ dự án chốt 2026-11-08 (`plan.md` mục cuối). Phần (D), khả thi từng đề xuất trong `audit/findings.md`, sẽ viết khi BA push file đó.

**TL trả lời:**

### A. Video giới thiệu: chọn `page.screencast` của Playwright 1.63, mã hoá bằng ffmpeg 7.1 trong container

**Chọn:**
- Ghi hình bằng `page.screencast.start({ onFrame, size: 1440×900, quality: 90 })`. API này lấy khung JPEG kèm `timestamp` (ms).
- Con trỏ dùng `screencast.showActions({ cursor: "pointer", fontSize: 0 })`.
- Phụ đề tiếng Việt là `screencast.showOverlay(html)`: khung chữ vẽ ngay trong trang, nên thành phụ đề in vào hình.
- ffmpeg chỉ làm ba việc:
  1. ghép khung theo `timestamp` thành 30 fps;
  2. mã hoá mp4 H.264 và webm VP9;
  3. cắt một khung làm poster.

**Lý do:** không cần cài thêm gói npm, không cần `libass` và tệp phông cho ffmpeg. Phụ đề được trình duyệt vẽ nên dấu tiếng Việt luôn đúng. Không thêm thư viện vào dự án.

**Bằng chứng** (PoC trên VPS 12 vCPU / 32 GB, 2026-11-08; mã ở `docs/research/poc/landing-video/`):
- `playwright-core@1.63.0` có `Screencast.start({path|onFrame,size,quality})`, `showActions({cursor,duration,fontSize,position})`, `showOverlay`, `showChapter` (`types.d.ts`, `interface Screencast`). Tài liệu: <https://playwright.dev/docs/api/class-screencast>.
- Ảnh `mcr.microsoft.com/playwright:v1.63.0-noble` (đã có trên VPS) có `ffmpeg-1011` riêng của Playwright, nhưng bản này chỉ có bộ mã hoá `libvpx` VP8. Máy VPS (Debian 13.7) **chưa có ffmpeg**; bản cài được qua apt là `7:7.1.5-0+deb13u1`. Trong `debian:trixie-slim` + `apt install ffmpeg` có đủ `libx264` và `libvpx-vp9`.
- **Cỡ tệp, đo thật.** Cảnh xấu nhất: cuộn liên tục trang giới thiệu suốt 44,5 s ở 1440×900, 768 khung. Cột cuối nhân bitrate cho video 90 s:

| Đầu ra | bitrate | ước tính 90 s |
| --- | --- | --- |
| `screencast.start({path})` (VP8 của Playwright) | 895 kbps | 10,1 MB |
| JPEG → H.264 `-preset slow -tune stillimage -crf 26` | 784 kbps | 8,8 MB ✗ |
| JPEG → H.264 `-crf 28` | 687 kbps | 7,7 MB |
| JPEG → H.264 `-crf 30` | 599 kbps | 6,7 MB |
| H.264 → VP9 `-crf 40 -b:v 0` | 345 kbps | 3,9 MB |
| webm VP8 → H.264 `-crf 26` (nén hai lần) | 851 kbps | 9,6 MB |

  Thời gian mã hoá cho 44 s video: H.264 khoảng 10 s, VP9 khoảng 30 s.
- **Poster:** cắt từ khung H.264 `-crf 28`, WebP q75. Bản 1440 px nặng 71 KB, bản 1280 px nặng 59 KB.
- **Con trỏ:** có hiện (`tile.png` trong PoC). Con trỏ chạy tới điểm bấm, có chấm đỏ khi bấm, phần tử bị bấm có viền xanh, sau `duration` thì con trỏ ẩn. Đặt `fontSize: 0` thì tắt được nhãn tiếng Anh ("Mouse wheel", "Click") ở góc màn hình.
- **Tua video (Range):** bản demo trả `206` + `accept-ranges: bytes` cho tệp trong `public/` (đi qua Caddy 2.11). Safari tua được video. Tệp trả về với `cache-control: public, max-age=0`, nghĩa là mỗi lần xem trình duyệt hỏi lại máy chủ (thường nhận `304`).

**Cách dựng đề xuất** (`scripts/landing-video/`, chạy lại được):
1. `record.mjs`, chạy trong ảnh Playwright 1.63. Mỗi cảnh là một hàm: `goto` → thao tác → `showOverlay(phụ đề)` → chờ. Script ghi khung JPEG và `frames.ffconcat`; mỗi `duration` = hiệu `timestamp` của hai khung liền nhau ÷ 1000.
   - **Lỗi PoC đã gặp:** đổi đơn vị ms → s bằng ngưỡng làm video dài thêm 9 s. Luôn chia 1000.
2. `encode.sh`, chạy trong `debian:trixie-slim` + ffmpeg:
   - mp4: `-fps_mode cfr -r 30 -c:v libx264 -preset slow -tune stillimage -crf 28 -maxrate 700k -bufsize 1400k -pix_fmt yuv420p -movflags +faststart -an`;
   - webm: VP9 `-crf 40 -b:v 0`;
   - poster WebP 1280 px q75.
   - Cảnh thật ít cuộn hơn PoC nên tệp sẽ nhỏ hơn. Có `-maxrate` thì cảnh cuộn nhiều nhất của bản 90 s vẫn dưới 8 MB.
3. Đầu ra ở `frontend/public/landing/`. Tên tệp có số phiên bản (`gioi-thieu-v1.mp4`) để sau này có thể đặt cache dài cho tệp. Không bắt buộc làm ở sprint này.

**Trên trang:**
```html
<video controls preload="none" playsinline poster="/landing/gioi-thieu-v1.webp" width="1280" height="800"
       aria-label="Video giới thiệu EduPilot" aria-describedby="video-loi">
  <source src="/landing/gioi-thieu-v1.webm" type="video/webm"> <!-- VP9 trước: nhẹ bằng một nửa -->
  <source src="/landing/gioi-thieu-v1.mp4" type="video/mp4">
</video>
```
- Có `controls` gốc của trình duyệt nên không cần JS và dùng được bằng bàn phím.
- `-an` = không có rãnh âm thanh: hết rủi ro bản quyền nhạc, không lo tự phát kèm tiếng.
- Video không có tiếng thì WCAG 1.2.1 đòi bản chữ thay thế. Đặt các câu phụ đề vào một đoạn ẩn với mắt (`#video-loi`, lớp ẩn trực quan) để không thêm chữ lên UI (luật chữ UI). Nguồn: <https://www.w3.org/WAI/WCAG22/Understanding/audio-only-and-video-only-prerecorded>.

**Rủi ro:**
1. **Quay trên bản demo `/opt/edupilot` sẽ ghi vào DB demo.** Kịch bản có gửi chat, duyệt bài, có thể có công bố (không đảo ngược được). Vai ADMIN mở prompt thì sinh dòng `audit_log`. Nếu chat gọi LLM thật thì tốn hạn mức 9router và câu trả lời mỗi lần một khác. → Nên quay trên stack riêng `-p ep-video`, seed giống demo, provider `fake`/replay. Kết quả tất định, quay lại được, cùng bản dựng nên hình giống hệt (xem đề xuất 1).
2. **Font của phụ đề `showOverlay` chưa chắc.** Lần 1 đặt `'Be Vietnam Pro'` trong `style` thì ra phông dự phòng. Lần 2 thì ra đúng phông, nhưng cỡ chữ bị bỏ vì dấu `"` trong tên phông làm vỡ thuộc tính `style`. [CHƯA CHẮC] Overlay có dùng `@font-face` của trang ổn định hay không. Dev kiểm bằng ảnh một khung. Nếu sai phông thì đổi sang phụ đề ASS của ffmpeg (`subtitles=`, cần tệp TTF Be Vietnam Pro, giấy phép OFL).
3. **Ở bề rộng 375 px**, video 1440 thu còn khoảng 343 px, chữ trong app khó đọc. Phụ đề trong nguồn nên ≥ 36 px. Bản dọc 9:16 cho điện thoại để sau, khi chủ dự án cần.
4. **Viền xanh của `showActions`** quanh phần tử được bấm không theo màu thương hiệu, và chưa có tuỳ chọn tắt riêng viền này. Nếu chủ dự án thấy không ổn: `cursor: "none"` + con trỏ tự vẽ bằng `addInitScript` (khoảng 20 dòng, chỉ nằm trong script quay).
5. Poster tải ngay cả khi `preload="none"` (`<video poster>` không có `loading="lazy"`). Nếu video nằm trong vùng thấy đầu tiên thì poster có thể thành phần tử LCP (<https://web.dev/articles/lcp>, poster của `<video>` được tính LCP). Khi đó cần `preload` poster như ảnh hero hiện nay (`Landing.tsx` gọi `preload` của `react-dom`).

**Ước lượng:** vừa (script khoảng 150 dòng + ba lệnh ffmpeg + một lượt quay thử).

### B. Chuyển động trang giới thiệu: làm được hết bằng CSS, không cần thư viện, gần như không thêm JS

**Ràng buộc gốc:**
- `/` render `HomeGate`. Đây là thành phần client và import sẵn `AppShell` + `TodayScreen` (`HomeGate.tsx` dòng 4–5). Vì vậy mọi JS thêm cho trang giới thiệu cộng thẳng vào bundle của `/`, route đang sát ngân sách 256 KB (dev đang sửa CI đỏ ở `resource-summary:script:size`).
- → `Landing` giữ là Server Component. Chuyển động viết trong `Landing.module.css`. Chỉ thêm JS khi CSS không làm được.

| Hiệu ứng (chủ dự án chốt) | Cách làm | JS | Rủi ro / điều kiện |
| --- | --- | --- | --- |
| Mở đầu có chủ ý | CSS `animation` khi tải trang: chữ phụ, nút, khung ảnh nối nhau 40–60 ms, `transform` + `opacity`, `--ep-ease-out`, 400–600 ms | 0 | **Phần tử LCP không được bắt đầu từ `opacity: 0`.** Chromium chỉ tính LCP khi phần tử có độ mờ > 0, nên mờ dần sẽ đẩy LCP lùi theo thời lượng hiệu ứng. Phần tử LCP là `h1` ở 375 px và ảnh hero ở 1440 px: chỉ cho chúng trượt nhẹ bằng `transform`. Mâu thuẫn `DESIGN.md` §11 "No staggered page-load choreography" → [ĐỔI DESIGN] (đề xuất 2) |
| Xuất hiện theo nhịp khi cuộn | `animation-timeline: view()` + `animation-range: entry 0% entry 100%` | 0 | Baseline từ 10/2026: Chrome, Safari 26, Firefox 159 (MDN `animation-timeline`; Safari 26.0 release notes). Trình duyệt cũ hơn: bọc trong `@supports`, nội dung hiện tĩnh, không mất chữ |
| Sơ đồ "AI soạn nháp → giảng viên duyệt → mã tính" | Cùng `view()`: ba bước có `animation-range` lệch nhau; vạch đỏ dùng `scaleX` với `transform-origin: left` | 0 | Chữ ba bước là HTML thật (trình đọc màn hình đọc đủ); vạch đỏ để `aria-hidden`. Đỏ ở đây đúng chữ ký "Red Thread" (§11) |
| Ảnh / khung có chiều sâu, chuyển nhẹ theo cuộn | `view()` + `translateY` ±16–24 px, `scale` 0,98→1 | 0 | "Chiều sâu" mà cần bóng thì trái §7 (chỉ `Panel` / menu / dialog có bóng) → [ĐỔI DESIGN], gộp vào đề xuất 2. Thay bằng đường kẻ + lệch lớp thì không cần đổi |
| Số liệu đếm lên | `@property --n { syntax: "<integer>" }` + `counter(n)` trong `::after`, chạy theo `view()` | 0 | Chỉ đếm được số nguyên: `146`, `30` đếm được; `+0,05 %` để tĩnh vì đếm từ 0 lên 0,05 không có nghĩa. Số thật để trong DOM; bản đếm để `aria-hidden` cho trình đọc màn hình không đọc từng số. Đã có `tabular-nums` nên không xô bố cục |
| Nút chính phản hồi tinh tế | `transition: transform var(--ep-fast)`; `:active { scale(.97) }`; hover đặt trong `@media (hover: hover) and (pointer: fine)` | 0 | Không |
| View transitions | Bỏ ở trang này | – | `Link` của Next là điều hướng trong cùng tài liệu, nên `@view-transition` (giữa hai tài liệu) không chạy. `<ViewTransition>` của React còn là cờ thử nghiệm của Next. Không đáng rủi ro |
| Video | Mục A | 0 | – |

**Giảm chuyển động (bằng chứng PoC `rm.mjs`, Chromium 153):**
- Luật chung ở `base.css` dòng 51–53 (`animation-duration: 1ms !important` khi `reduce`) **không tắt được** animation theo cuộn. Với `view()`, thời lượng bị quy thành tỉ lệ của dòng thời gian cuộn, nên hiệu ứng vẫn chạy.
- Số đo độ mờ khi khối vừa vào khung nhìn 15 %:

| Giảm chuyển động | Có bọc `@media` | Độ mờ đo được | Kết luận |
| --- | --- | --- | --- |
| `reduce` | không | 0,136 | Hiệu ứng vẫn chạy |
| `reduce` | có | 1 | Đã tắt đúng |

- → Mọi hiệu ứng theo cuộn phải nằm trong `@media (prefers-reduced-motion: no-preference) { @supports (animation-timeline: view()) { … } }`. Trạng thái mặc định ngoài khối này là hiện đủ.
- Khi `reduce`: giữ mờ dần ngắn của phần mở đầu, bỏ dịch chuyển (theo `skill://animate`).

**Ảnh mốc và đo:**
- `visual.spec.ts` chụp `/` ở vai sinh viên đã đăng nhập (dòng 22), với `reducedMotion: "reduce"` (dòng 11). Đổi trang giới thiệu **không** buộc sinh lại 14 ảnh mốc.
- `landing.spec.ts` dòng 31 khớp nguyên văn ba `h2`. Đổi nội dung thì test phải đổi theo spec của BA, không tự sửa.
- **Không cổng tự động nào đo LCP của trang giới thiệu.** `lighthouserc.json` đặt `largest-contentful-paint: off`, và `lighthouse-auth.cjs` gán `/` → vai `student` (đo "Hôm nay", không đo trang giới thiệu). Vì vậy LCP ≤ 2,5 s của `plan.md` cần một lượt đo ẩn danh riêng (Lighthouse hoặc Playwright `PerformanceObserver`) trong handoff của dev.
- TBT không đổi: hiệu ứng chỉ dùng `transform` / `opacity`, Chromium chạy trên compositor.

**Thư viện:** không có hiệu ứng nào trong danh sách cần thư viện. Không mở proposal thư viện.

**Ước lượng:** vừa (CSS khoảng 120–180 dòng, 0 KB JS).

### C. Chuyển động trong ứng dụng (≤ 200 ms, không thư viện)
- **Lệch với luật mới, cần sửa:**
  - `Dialog.module.css` dòng 10 (`.overlay[open]`) và dòng 25 (`.drawer[open]`) dùng `--ep-slow` = **250 ms** (`tokens.css` dòng 67).
  - `DESIGN.md` §11 ghi "Drawer/dialog: 200–250ms".
  - → Đổi token về ≤ 200 ms và sửa §11 (đề xuất 3).
  - Ảnh mốc không đổi: `visual.spec` chạy với `reduce` và `animations: "disabled"`.
- **Ngoài phạm vi luật:** spinner của `Button` (700 ms, lặp) và ba chấm `Threads` là chỉ báo chạy liên tục, không phải chuyển trạng thái. Ba chấm đã có nhánh `reduce`.

### D. Khả thi từng đề xuất của BA
Xem TL-2.

### Đề xuất cho PM (chép vào `proposals.md` nếu đồng ý)
1. **Quay video trên stack riêng.** Vấn đề: `plan.md` ghi "quay màn hình bản demo thật"; quay trên `/opt/edupilot` sẽ ghi chat / duyệt / công bố vào DB demo và tốn hạn mức 9router. Đề xuất: quay trên stack `-p ep-video` cùng bản dựng, seed như demo, provider `fake`. Lý do: tất định, quay lại được, không đụng demo (TL-1 A, rủi ro 1). Ảnh hưởng: chỉ đổi chữ trong `plan.md`; hình như nhau.
2. **Ngoại lệ chuyển động cho trang giới thiệu trong `DESIGN.md` §11.** Vấn đề: hướng chủ dự án chốt (hiệu ứng mở đầu, xuất hiện theo nhịp, chiều sâu) trái §11 "No staggered page-load choreography" và có thể trái §7 (bóng). Đề xuất: thêm mục "Ngoại lệ: trang giới thiệu (Persuade)":
   - mở đầu ≤ 600 ms;
   - phần tử LCP không bắt đầu từ `opacity: 0`;
   - hiệu ứng theo cuộn chỉ dùng `transform` / `opacity`, bọc `prefers-reduced-motion: no-preference`.

   Lý do: TL-1 B. Ảnh hưởng: BA sửa `DESIGN.md` §11; không đổi code app. **Chủ dự án đã chốt 2026-11-08** (`plan.md`, "Quyết định của chủ dự án"); khi BA viết mục mới, nên kèm ba điều kiện kỹ thuật ở trên.
3. **Rút thời lượng dialog / drawer về ≤ 200 ms.** Vấn đề: `Dialog.module.css` dòng 10, 25 dùng `--ep-slow` 250 ms; `DESIGN.md` §11 ghi 200–250 ms, trái luật ≤ 200 ms vừa chốt. Đề xuất: `--ep-slow` → 200 ms (hoặc dùng `--ep-normal`), sửa §11 thành "≤ 200 ms". Ảnh hưởng: một dòng token + một dòng doc; ảnh mốc không đổi.

## TL-2 (pha 1: khả thi từng đề xuất trong `audit/findings.md`, PM giao 2026-11-08)

**Câu hỏi (PM):** với mọi mục mức Cao / TB và mọi [ĐỔI DESIGN] của `findings.md`, đánh giá:
- khả thi với stack hiện có;
- rủi ro hiệu năng, a11y, ảnh mốc;
- cỡ việc S / M / L;
- nên gộp hay bỏ.

Căn cứ: các quyết định Q-UX91-1…5 và [ĐỔI DESIGN] trang giới thiệu (`plan.md`, "Quyết định của chủ dự án").

**TL trả lời:**

### Quy ước
- **Cỡ:**
  - S: ≤ nửa ngày, ≤ 3 tệp, không đổi API;
  - M: 0,5–1,5 ngày hoặc đụng primitive dùng chung;
  - L: > 1,5 ngày, hoặc cần API / dữ liệu mới.
- **Ảnh mốc** là 14 ảnh của `visual.spec.ts`, gồm 7 cảnh × 1440 / 390 (dòng 21–29): `home` (SV), `chat` (SV), `threads` (SV), `inbox` (GV), `gradebook` (GV), `settings-llm`, `dev-ui`. Cảnh nào không nằm trong danh sách này thì không phải sinh lại ảnh mốc.
- Các thông số chung đều kiểm được trong repo:
  - mọi mục dưới đây không thêm thư viện;
  - không mục nào chạm ngân sách JS, trừ UX-1 (đã tính trong TL-1 B: 0 KB).

### Bảng đánh giá

| ID | Mức | Khả thi + cách làm (bằng chứng) | Cỡ | Rủi ro: hiệu năng · a11y · ảnh mốc | Gộp / bỏ / điều kiện |
| --- | --- | --- | :---: | --- | --- |
| UX-1 | Cao | Có. Toàn bộ là CSS trong `Landing.module.css`, `Landing` vẫn là Server Component (TL-1 B). Hai cơ chế: **(1) theo cuộn** `animation-timeline: view()` cho xuất hiện / chiều sâu / vạch đỏ SVG `stroke-dashoffset`; **(2) chạy một lần khi vào khung** cho số đếm và sơ đồ luồng, dùng một đoạn `<script>` nội tuyến khoảng 0,4 KB (cùng kiểu đoạn `data-returning` ở `app/page.tsx` dòng 16): `IntersectionObserver` gắn `data-in` lên khối, CSS animation theo thời gian chạy khi có `data-in`. Không cần React island | L (cả video) | LCP: `h1` / ảnh hero không bắt đầu từ `opacity: 0`. a11y: hiệu ứng cuộn **phải** bọc `prefers-reduced-motion: no-preference`, vì luật `1ms` ở `base.css` không tắt được `view()` (PoC `rm.mjs`). Ảnh mốc: 0 (`home` là "Hôm nay" của SV) | **Bỏ phần "dự phòng `IntersectionObserver` cho `view()`"** trong findings §4.2. `view()` là Baseline từ 10/2026; trình duyệt cũ thấy trang tĩnh, đủ nội dung. Đoạn script ở (2) đã thay việc đó cho những gì cần chạy một lần. `landing.spec.ts` dòng 31 khớp nguyên văn `h2`: spec mới của BA phải nêu `h2` mới |
| UX-2 | Cao | Có. Đổi chữ + số đếm `@property --n { syntax: "<integer>" }` + `counter()`. Q-UX91-5: thêm dòng chú thích "đo trên dữ liệu mô phỏng" | S | Chỉ đếm được số nguyên (`146`, `30`, `22`). Số thật nằm trong DOM; bản đếm để `aria-hidden`. Có `tabular-nums` nên không xô bố cục (CLS 0) | Gộp vào UX-1. Số phải giữ nguồn từ `report.md` của sprint 6 / 8 / 9 (chú thích ở `Landing.tsx` dòng 7) vì quyển đồ án dùng lại |
| UX-3 | TB | (a) Có: CSS cắt / khung điện thoại; ảnh 640 px đã có sẵn. (b) Tab GV / SV cần trạng thái, tức là thêm JS vào bundle của `/`. (c) Có. (d) Có: `AuthProvider.tsx` dòng 28–34 luôn gọi `refreshSession()`. Khi đang ở `/` và không có cờ `ep_returning` thì bỏ qua lời gọi, `dropSession("anonymous")` luôn. Hết `401` ở console | (a) S, (b) S, (c) S, (d) S | (d) không đổi bảo mật: cookie `ep_rt` vẫn được thử ở `/login`. Người còn phiên nhưng mất cờ chỉ phải bấm "Đăng nhập". Test `landing.spec.ts` dòng 17 ("không gọi API nào ngoài refresh") vẫn qua | **(b) đổi thành "mỗi vai 4 mục, hai cột, không tab"**: cùng mục tiêu bớt chữ, 0 JS, 0 rủi ro bàn phím / ARIA của tab. Nếu giữ tab: dùng `Tabs` có sẵn, đo lại `resource-summary:script` của `/`. (d) nên làm ngay, độc lập trang mới |
| UX-4 | Cao | Có. Xoá `description=` là chuỗi giải thích. Giữ dạng `meta` khi là dữ liệu: ví dụ `RealAttendance.tsx` dòng 117 (mã lớp · buổi · giờ), `RealMe.tsx` dòng 40, `ClassSettings.tsx` dòng 66. **Không sửa `Demo*.tsx`** (`DemoChat`, `DemoCalendar`, `DemoDocuments`, `DemoLibrary`): chỉ phiên mô phỏng dùng (`ChatScreen.tsx`, `CalendarScreen.tsx` dòng 13–14), nên số chỗ thật ít hơn 55 | M | a11y: `Drawer` `description` có thể đang là `aria-describedby` của dialog. Xoá thì dialog vẫn có tên từ `title`, không lỗi axe. Ảnh mốc: `chat`, `threads` (nếu có mô tả) | Phép bắt mới trong `ui-antipatterns.sh`: chỉ bắt **chuỗi tĩnh** > 40 ký tự ở `PageHeader` / `Section`. Bắt cả biểu thức thì báo nhầm các dòng dữ liệu ở trên. Có e2e đã khoá luật này cho riêng một màn (`settings-integrations.spec.ts` dòng 154): dùng làm mẫu AC |
| UX-5 | Cao | Có, **không cần API mới.** `TodayItem` đã có `urgency` (`overdue` / `high` / `normal`), `kind`, `age_minutes` (`openapi.yaml` schema `TodayItem`). Khối "Lớp cần chú ý" đã có (`StaffToday.tsx` dòng 43–53), nguồn là `attention` (sinh viên AT_RISK), chỉ cần chuyển sang cột phải ≥ 1440. Q-UX91-2: nhóm Gấp / Hôm nay / Khi rảnh tính ở client từ `urgency` + `kind` | M | Ảnh mốc: 0 (`home` chụp vai SV). a11y: nhãn nhóm là tiêu đề chữ (`h3`), đạt §16 (không mã hoá chỉ bằng màu) | **Bỏ [ĐỔI DESIGN] "biến thể nút `link` mới"**: `Button` đã có `ghost` (chữ `--ep-ink-2`, không đỏ; `Button.module.css` dòng 27) + `iconEnd` mũi tên. **→ PM / BA:** "Gấp = quá 72 giờ làm việc" nên định nghĩa là `urgency == "overdue"` do backend tính. Backend đã có SLA giờ làm việc (`today/service.go` dòng 58, `EscalateAfter: 72h`) và ngưỡng riêng từng loại việc (phúc khảo 48 giờ: `TestProviderAppealOverdue48h`; buổi chưa điểm danh). Tính lại 72 giờ ở client là hai nguồn sự thật. Mẫu 3.B ghi nội dung cột phải theo lớp ("5 câu chờ quá 72 giờ") nhưng API trả **sinh viên** cần chú ý; muốn theo lớp thì cần API mới (L): giữ theo sinh viên |
| UX-6 | TB | Có. Ở Hôm nay, chữ tuổi lấy từ `age_minutes`, định dạng bằng `shared/lib/timeAgo.ts`. Ở Hộp thư, chấm đỏ chỉ cho hàng quá hạn | S | Ảnh mốc: `inbox` 1440 + 390 | Gộp với UX-5 (cùng quy tắc "đỏ = overdue") |
| UX-7 | Cao | Có. Q-UX91-4: chấm mặc định dùng mực. Thanh lưu dính đáy ở 375 px: `position: sticky; bottom: 0` + `padding-bottom: env(safe-area-inset-bottom)`; danh sách thêm khoảng đệm bằng chiều cao thanh để không che hàng cuối | M | a11y: trạng thái vẫn có chữ ("Có mặt"); thanh dính đáy không được che ô đang focus (`scroll-padding-bottom`). Ảnh mốc: 0. Bàn phím ảo trên di động đẩy thanh lên: QC kiểm trên máy thật hoặc emulation | "+ phát biểu" chỉ hiện khi hover / focus thì **ở 375 không có hover**. Ở di động giữ nút hiện thường trực (biểu tượng `+`, vùng chạm 44 px), chỉ bỏ chữ lặp |
| UX-8 | TB | Có. Gom các bộ định dạng về một `shared/lib/datetime.ts`, dựa trên `timeAgo.ts` và `today/format.ts`. **Lỗi thật:** 8 bộ định dạng **không đặt `timeZone`** nên chạy theo múi giờ của máy: `RealChat.tsx` 36–37, `RealDocuments.tsx` 14, `RealThreads.tsx` 16, `RealThreadDetail.tsx` 16, `RealLibrary.tsx` 11, `Gradebook.tsx` 383, `GradeScheme.tsx` 32, `FinalizeGrades.tsx` 12. Định dạng `00:10 02-11` là do ICU của `vi-VN`; `timeAgo.ts` dòng 6 đã dùng `en-GB` để có `dd/MM` ổn định | M | Ảnh mốc: `chat`, `threads`, `inbox`, `gradebook` (chữ ngày đổi). `visual.spec` đóng băng đồng hồ (`FROZEN`), nên "Hôm nay, 14:00" ra tất định | Làm ngay như BA đề xuất; đây là sửa lỗi, không chỉ thẩm mỹ |
| UX-9 | TB | Có. Q-UX91-1: sửa 3 nhãn ở `shell/nav.ts` dòng 38, 71, 79, 80 | S | Test: `shell.spec.ts` dòng 120, 122 khớp nguyên văn nhãn nav, nên spec mới phải nêu nhãn mới. Ảnh mốc: mọi ảnh 1440 có thanh bên (`home`, `chat`, `threads`, `inbox`, `gradebook`, `settings-llm`); ảnh 390 nếu thanh dưới của SV có chữ "Threads". "Hỏi đáp công khai" (17 ký tự) dài hơn "Threads": kiểm cắt chữ ở thanh bên và thanh dưới 375 (§22.7) | Gộp UX-9 + UX-8 + UX-4 vào **một** đợt sinh lại ảnh mốc, để khỏi sinh lại ba lần |
| UX-10 | TB | Có, chỉ đổi bố cục / CSS | S | Ảnh mốc: 0. Phép bắt "Panel chứa phần tử có `border` + `border-radius`" trong `ui-antipatterns.sh` dễ báo nhầm (ô nhập, chip có viền là hợp lệ) | Phép bắt tự động: **bỏ**. QC kiểm bằng `panels.spec` NEST (đã có từ sprint 5.5) |
| UX-11 | Cao | Phần bố cục có: một `Panel`, lịch sử phẳng, `Phiên mới` ẩn khi phiên trống. Sheet di động đã có (`chat/SessionSheet.tsx`). Xoá có hoàn tác 5 s: hoãn lệnh xoá ở client bằng `useUndoLine`; tab đóng trong 5 s thì phiên **không** bị xoá, lệch về phía an toàn. **Gợi ý "lấy từ dữ liệu sinh viên" chưa có nguồn trong chat** (`RealChat.tsx` không có gì về gợi ý). Hạn bài lấy được từ dữ liệu Hôm nay của SV (`timeline`, `recommended`); "chủ đề sai nhiều" chưa thấy API | Bố cục M; gợi ý theo hạn bài S; "chủ đề sai nhiều" L (API mới) | Ảnh mốc: `chat` 1440 + 390. `/chat` nằm trong Lighthouse (ngân sách JS): không thêm query mới chặn lúc tải; gợi ý nạp sau. D47: gợi ý **không** gọi LLM | **Bỏ "chủ đề sai nhiều"** ở 9.1 (ghi Nợ). Giữ gợi ý từ hạn bài / lịch. Hiệu ứng tin nhắn 160 ms đạt luật ≤ 200 ms |
| UX-12 | TB | Có, chỉ đổi chữ + ẩn vùng chi tiết khi danh sách rỗng | S | Ảnh mốc: `inbox` | Gộp với UX-6 (cùng màn) |
| UX-13 | TB | Có, nhưng **không có primitive Tooltip** (`shared/ui/index.ts`). Nút `disabled` gốc không nhận focus, nên lý do không bao giờ hiện bằng bàn phím. Cách làm: prop `disabledReason` trên `Button`. Khi có prop thì dùng `aria-disabled="true"` (không dùng `disabled`), chặn `onClick`, `aria-describedby` trỏ tới một `span` ẩn trực quan chứa lý do; hiện chữ bằng CSS `::after` khi `:hover` / `:focus-visible` | M | a11y: đây chính là cách đúng (vẫn focus được, đọc được lý do). Test: `toBeDisabled()` của Playwright tính cả `aria-disabled`, nên test cũ không gãy [SUY LUẬN theo tài liệu Playwright; dev chạy lại bộ e2e]. Chạm trên di động: lý do hiện khi chạm (focus). Ảnh mốc: những cảnh có nút khoá (`gradebook`) | Một cơ chế trong `Button` (đúng đề xuất BA); không viết Tooltip riêng từng màn. `.btn:disabled { opacity: .45 }` (`Button.module.css` dòng 34) phải áp cả `[aria-disabled=true]` |
| UX-14 | TB | Có, đổi cột / hàng ở trang danh sách chấm | S | Ảnh mốc: 0 | Gộp với UX-15 (cùng luồng chấm) |
| UX-15 | TB | Bố cục một hàng / tiêu chí, gợi ý phím `A` (`Kbd` có sẵn), thanh dính đáy: có. **"‹ 3/12 ›"** cần thứ tự hàng chờ; chỉ có khi đi từ trang danh sách (bộ đệm TanStack đang giữ danh sách). Mở bằng đường dẫn trực tiếp thì không có, nên ẩn | M | **Highlight 600 ms ở mẫu 3.D trái luật ≤ 200 ms của app.** Hoặc coi là "tín hiệu mờ dần", không phải chuyển cảnh (→ PM), hoặc giữ nền tĩnh 1,5 s rồi tắt bằng 200 ms. Ảnh mốc: 0 | → PM: quyết highlight 600 ms; Red Thread 280 ms (đã có, `shared/motion/redThread.ts` dòng 6) cũng > 200 ms nhưng DESIGN §11 ghi < 300 ms: ghi rõ là ngoại lệ đã có |
| UX-16 | TB | Căn phải + `tabular-nums`: có. **"Hai chữ số thập phân" là làm tròn khi hiển thị:** sổ điểm hiện nguyên chuỗi decimal của máy chủ (`Gradebook.tsx` dòng 18 chỉ đổi `.` → `,`), `7,915` sẽ thành `7,92`. **"Cuộn theo trang, tiêu đề dính" không làm được với bảng này:** bảng là `DataTable` ảo hoá `virtual={{ height: 560 }}` (dòng 340; US-PU-02 cho T1 1.000 SV). Vùng cuộn ngang `overflow-x: auto` (`DataTable.module.css` dòng 2) giữ tiêu đề dính trong chính nó, không dính theo trang | M (với cách thay thế) | Q-UX91-3 "Đủ điểm / Thiếu điểm": S. Ảnh mốc: `gradebook` 1440 + 390 | **Thay "hai chữ số thập phân"** bằng đệm số 0 tới số chữ số lẻ lớn nhất của cột (`0,500` / `7,915`). Căn được dấu phẩy mà không làm tròn. **→ PM:** làm tròn điểm hiển thị đụng luật tính điểm, cần chủ dự án quyết. **Thay "cuộn theo trang"** bằng chiều cao vùng cuộn = khung nhìn (`--vh: calc(100dvh - <đầu trang>)`): một thanh cuộn, giữ ảo hoá. Chuyển sang `useWindowVirtualizer` là L và đụng AC hiệu năng T1 |
| UX-17 | TB | Có. `Chart.tsx` là primitive dùng chung, chỉnh token cỡ chữ / độ dày đường | S | Ảnh mốc: `dev-ui` nếu trang danh mục có biểu đồ | BA xếp "Để sau": đồng ý |
| UX-19 | TB | Có. Công tắc đã có `aria-checked` (`Field.module.css` dòng 50); màu lệch là do `:hover` (dòng 90–91) | S | Ảnh mốc: 0 | – |
| UX-20 | TB | Có ở client. `preview` = `left(t.body, 160)` trong SQL (`forum.sql` dòng 19) là markdown thô. Làm sạch ở client bằng `shared/lib/markdown.ts` (`parseMarkdown` → lấy chữ) | S | Ảnh mốc: `threads` 1440 + 390 | Không đổi SQL / API ở 9.1. Cắt 160 ký tự **trước** khi bỏ markdown thì sau khi bỏ ký hiệu còn ngắn hơn 140: chấp nhận |
| UX-21 | TB | Có, gộp hai khối của `/me` thành một (cùng dữ liệu) | S | `/me` có trong Lighthouse; ảnh mốc: 0. Không thêm query | – |
| UX-25 | TB | (a) Có ở client. `session.tsx` dòng 66: chưa chọn thì đọc `ep:ui:course`, rồi lấy "lớp đầu" theo thứ tự API. "Lớp có việc chờ" cần số việc theo lớp → L. **Cách S:** GV / TA có > 1 lớp và chưa từng chọn thì mặc định "Tất cả lớp" (Hôm nay đã hỗ trợ `isAll`). (c) Seed: dev | (a) S | Ảnh mốc: 0 | → PM chọn "mặc định Tất cả lớp" (S) hay "lớp có việc" (L) |
| UX-26 | TB | Có, một đợt ở `shared/ui` + `base.css`. Thanh tiến độ: 4 chỗ `transition: width` (`Documents`, `GradeScheme`, `Questions`, `Practice` `.module.css`) đổi sang `scaleX` + `transform-origin: left`. Track đã `overflow: hidden` + bo tròn, phần lấp không bo nên không méo. Khung vỏ: `AppShell.module.css` dòng 17, 82, 150 | M | **Reduced-motion "giảm thay vì tắt":** đổi `base.css` dòng 51–53 sẽ làm mọi animation hiện có chạy lại khi `reduce`. Nên giữ luật `1ms` làm lưới và **cho phép riêng** `opacity` / màu bằng lớp `.ep-motion-safe`, khỏi phải rà 71 chỗ. Ảnh mốc: 0 (`reduce` + `animations: "disabled"`). Gate `hover: hover` cho 71 `:hover` là đổi lớn nhưng cơ học | Gộp với TL-1 đề xuất 3 (dialog / drawer 250 → 200 ms) |
| [ĐỔI DESIGN] trang giới thiệu | – | Đã chốt. Mục mới của `DESIGN.md` nên ghi: mở đầu ≤ 600 ms; phần tử LCP không bắt đầu từ `opacity: 0`; theo cuộn chỉ dùng `transform` / `opacity`; bọc `prefers-reduced-motion: no-preference` (TL-1 đề xuất 2) | S (doc) | – | – |
| [ĐỔI DESIGN] UX-5 nút `link` | – | Không cần: dùng `ghost` có sẵn | – | – | **Bỏ** |
| [ĐỔI DESIGN] UX-9 §2 nav | – | Đã chốt (Q-UX91-1) | S (doc) | Xem UX-9 | – |

### Nên gộp thành 6 đợt (giảm số lần sinh lại ảnh mốc và số lần chạm primitive)
1. **Nền chung (làm ngay)**, mỗi mục chạm primitive một lần:
   - UX-26 + TL-1 đề xuất 3;
   - UX-13 (`Button`);
   - UX-8 (`datetime.ts`);
   - UX-3(d) (`AuthProvider`).
2. **Chữ:** UX-4 + UX-12 + UX-22 + UX-9. Sinh lại ảnh mốc **một lần** cho cả đợt 1 + 2.
3. **Hôm nay + Hộp thư:** UX-5 + UX-6.
4. **Chấm bài:** UX-14 + UX-15. **Sổ điểm:** UX-16 + Q-UX91-3. **Điểm danh:** UX-7.
5. **Chat + Threads + `/me`:** UX-11 (bỏ "chủ đề sai nhiều") + UX-20 + UX-21. **Tài liệu + Insights / Quan sát AI:** UX-19 + UX-10.
6. **Trang giới thiệu + video:** UX-1 + UX-2 + UX-3(a–c), song song, độc lập các đợt trên.

### Cần PM / chủ dự án quyết (không phải câu kỹ thuật)
1. **UX-16, làm tròn khi hiển thị.** "Hai chữ số thập phân" đổi `7,915` → `7,92` trên sổ điểm. TL đề xuất đệm số 0, không làm tròn.
2. **UX-5, định nghĩa "Gấp".** Định nghĩa là `urgency == "overdue"` của backend (ngưỡng riêng từng loại việc), hay đúng nghĩa đen "72 giờ làm việc" cho mọi việc.
3. **UX-15, highlight 600 ms trong app.** Ngoại lệ của luật ≤ 200 ms, hay rút ngắn.
4. **UX-25(a), lớp mặc định.** "Tất cả lớp" (S) hay "lớp có việc chờ" (L, cần API).

### Đề xuất cho PM (chép vào `proposals.md` nếu đồng ý)
4. **UX-16:** thay "số cố định hai chữ số thập phân" bằng "đệm số 0 tới số chữ số lẻ lớn nhất của cột, không làm tròn", và thay "cuộn theo trang" bằng "vùng cuộn cao bằng khung nhìn, giữ ảo hoá". Lý do: làm tròn hiển thị đụng luật điểm; bảng ảo hoá + cuộn ngang không cho tiêu đề dính theo trang (TL-2 UX-16). Ảnh hưởng: mẫu 3.E.
5. **UX-1 / UX-3(b) / UX-5 / UX-10 / UX-11:**
   - bỏ "dự phòng `IntersectionObserver` cho `view()`";
   - bỏ tab GV / SV (hai cột, mỗi vai 4 mục);
   - bỏ [ĐỔI DESIGN] nút `link` (dùng `ghost`);
   - bỏ phép bắt tự động "Panel chứa khung có viền";
   - bỏ gợi ý "chủ đề sai nhiều" ở 9.1 (ghi Nợ, cần API).

   Lý do: mỗi mục hoặc thêm JS vào `/`, hoặc trùng thứ đã có, hoặc cần API ngoài phạm vi sprint giao diện (TL-2). Ảnh hưởng: phạm vi spec `FEAT-uiux-91` nhỏ lại, không mất mục tiêu nào của BA.
