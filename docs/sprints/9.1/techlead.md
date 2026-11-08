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

## TL-3 (pha 2, đợt 6: thiết kế "Dùng thử", PM giao 2026-11-08)

**Câu hỏi (PM):** thiết kế nút "Dùng thử" đăng nhập một chạm vai sinh viên / giảng viên. Ràng buộc an toàn theo `plan.md` mục "Chủ dự án duyệt báo cáo":
- chỉ bật khi `DEMO_MODE=true`;
- tài khoản và lớp sandbox riêng;
- đặt lại định kỳ bằng seed, không `down -v`, không đụng lớp trình diễn;
- giới hạn tần suất;
- không lộ mật khẩu.

Cần nêu cả ảnh hưởng OpenAPI / migration / auth, rủi ro và ước lượng.

**TL trả lời:**

### Hiện trạng bản demo (đọc trên VPS 2026-11-08, chỉ đọc)
- Gateway chạy `APP_ENV=dev`, có `LLM_DEMO_FILE=/seed/demo/fake_llm.json`. **Biến `DEMO_MODE` chưa tồn tại** (`config.go` chỉ có `LLM_DEMO_FILE`, dòng 399–402).
- **Mọi tác vụ LLM đều đi 9router** `ag/gemini-3.8-flash`: CHAT, CLASSIFY, GRADING, INSIGHT, QUESTION_GEN, UTILITY. EMBEDDING đi Gemini thật (`gemini-embedding-001`).
- Provider 9router có `rpm_limit = 20`, **giá `price_in` / `price_out` = 0**. Bảng `llm_budgets` **rỗng**.
  → Ngân sách VND hiện có (`internal/llm/budget`, theo hệ thống / lớp × ngày / tháng) **không chặn được gì**, vì chi phí luôn tính bằng 0.
  → 20 rpm là chung cho cả lớp trình diễn và khách.
- Bản demo có 2 lớp. Mail đi Mailpit, không ra ngoài.

### Chọn: sandbox **trong cùng stack**, lớp sandbox riêng, nhóm tài khoản cho mượn theo lượt

| Tiêu chí | A. Cùng stack, lớp + tài khoản sandbox (**chọn**) | B. Stack sandbox riêng (`-p ep-sandbox`) |
| --- | --- | --- |
| Khớp ràng buộc PM | Đủ: không `down -v`, không đụng lớp trình diễn | Đặt lại = dựng lại DB của stack sandbox, trái chữ "không `down -v`" → phải mở lại ràng buộc |
| Cô lập | Theo lớp (`CourseAccessGuard`) + chặn thêm cho tài khoản sandbox | Hoàn toàn (DB / Redis riêng) |
| Hạ tầng | Không thêm | Thêm một bộ gateway / worker / Postgres / Redis / frontend. Cần tên miền phụ hoặc `basePath` cho frontend (host `happysrv.de` của netcup [CHƯA KIỂM] có cho tên miền phụ không) |
| 9router 20 rpm | Vẫn chung | Vẫn chung (cùng 9router) |
| Công sức | L (≈ 3–3,5 ngày) | L+ (≈ 4–5 ngày, thêm vận hành) |

B chỉ đáng làm khi PM muốn cô lập tuyệt đối. Trong cả hai cách, rủi ro lớn nhất (hạn mức 9router chung) là như nhau.

### Thiết kế A

**1. Tài khoản và lớp sandbox (seed, không migration)**
- Seed tạo **một lớp sandbox** gồm:
  - mã lớp riêng (ví dụ `79xxxx`), tên "Lớp dùng thử";
  - **nhóm tài khoản cho mượn**: 10 sinh viên `thu.sv01…10@thu.edupilot.local`, 3 giảng viên `thu.gv01…03@thu.edupilot.local`, đều là thành viên **chỉ** của lớp sandbox;
  - các sinh viên khác của lớp sandbox (≈ 20, như lớp 1), không cho mượn.
- Mật khẩu ngẫu nhiên lúc seed, không lưu ở đâu. Endpoint không cần mật khẩu.
- **Dấu nhận biết sandbox = miền email dành riêng `@thu.edupilot.local`**, đọc từ claim `email` của JWT. `IssueSession` đã ký email vào token: `session.go` dòng 336.
  - Không cần cột mới, nên không có migration.
  - Đăng ký / mời bằng miền này bị từ chối khi `DEMO_MODE=true` (một dòng kiểm ở `register` / `invite`).

**2. Đăng nhập một chạm: `POST /api/v1/demo/sessions {"role": "student" | "teacher"}`**
- **Chỉ gắn route** khi `DEMO_MODE=true`. `config.Load` báo lỗi nếu `DEMO_MODE=true` và `APP_ENV=production`, cùng kiểu `LLM_DEMO_FILE` (`config.go` dòng 400–402). Tắt → route không tồn tại (404).
- Middleware dùng lại của `authhttp`, giống các route auth (`handler.go` dòng 44–57):
  - `noStore`;
  - `cookieGuard`: chặn `Origin` lạ, chống login CSRF ép khách vào phiên sandbox;
  - `limitIP("demo", 10*time.Minute, 5)`.
  - Thêm một trần toàn cục `Limiter.HitWindow(ctx, "demo", "all", time.Hour, 120)`.
- **Cho mượn:** chọn một tài khoản trong nhóm của vai đó bằng `SET ep:demo:lease:{user_id} 1 NX EX 1800` (Redis, không trạng thái trong tiến trình). Hết tài khoản rảnh → `503 SERVICE_UNAVAILABLE` + `Retry-After`.
  - Mượn được thì `Sessions.RevokeAll(..., ReasonAdmin)`: phiên của khách trước bị đá ra ≤ 1 s nhờ khoá thu hồi Redis (`session.go` dòng 451–474). Dùng lý do có sẵn nên **không đổi CHECK của `auth_sessions`**.
  - Sau đó gọi `Sessions.CreateSessionFor`, hàm đã dùng cho "nhận lời mời", không kiểm mật khẩu (`session.go` dòng 312–315).
  - Trả **cùng thân + cookie `ep_rt`** như `/auth/login` (`writeSession`, `handler.go` dòng 142). Frontend không cần nhánh phiên mới.
- Ghi `audit_log` hành động `demo_session` (vai, `user_id`), không ghi IP vào nội dung.
- **Frontend, 0 KB JS ở `/`:**
  - nút trên trang giới thiệu là liên kết `/login?thu=sv` | `/login?thu=gv`;
  - màn đăng nhập (đã là client, đã có `authSession`) thấy tham số thì gọi endpoint, nạp phiên, rồi về `/`;
  - lỗi (đang tắt / 503 / 429) → một dòng trên màn đăng nhập ("Dùng thử đang tạm đóng" / "Đang đông, thử lại sau ít phút").
  - Nút chỉ hiện khi bản dựng có `NEXT_PUBLIC_DEMO_MODE=1`. Endpoint mới là chốt an toàn thật.

**3. Chặn hành vi phá hoại của tài khoản sandbox** (một middleware, điều kiện "email thuộc miền sandbox", trả 403):
- **Quản lý tài khoản:** `POST /me/password`, `DELETE /me/sessions*`, đổi email / hồ sơ đăng nhập. Không chặn thì khách đổi mật khẩu tài khoản cho mượn, rồi sau này `/login` vào lại, đi vòng qua cơ chế cho mượn.
- **Tích hợp:** cấu hình IMAP / Teams. Máy chủ sẽ kết nối tới host do người dùng nhập, tức rủi ro SSRF.
- **Tạo lời mời** giảng viên / TA, đổi người được gán của lớp.
- **Tải tài liệu:** giữ (là một phần của trải nghiệm) nhưng có trần riêng: ≤ 5 tệp / giờ / tài khoản, bằng `HitWindow`. Tải lên chạy docling (CPU) và embedding Gemini thật.
- Ngoài các mục trên, mọi quyền như người dùng thường của lớp sandbox. `CourseAccessGuard` đã ngăn đụng lớp trình diễn, vì tài khoản sandbox không là thành viên ở đó.

**4. Hạn mức LLM** (khách gọi Gemini thật qua 9router)
- **Đặt giá thật cho model 9router** (`llm_models.price_in` / `price_out`, theo bảng giá Gemini Flash) và **một dòng `llm_budgets` phạm vi lớp sandbox**, ví dụ 20.000 đ / ngày. Khi tới mức `exhausted`, budget Manager đã chặn lớp đó (`budget.go`: `State`, `Charge`); lớp trình diễn không bị ảnh hưởng.
  - Hệ quả: số "Chi phí" ở `/analytics` của lớp trình diễn sẽ khác 0 → **PM / chủ dự án quyết**.
- Đã có sẵn, không đổi: `chat.RatePerMin` theo người (`chat/send.go` dòng 65). Mỗi khách giữ một tài khoản nên trần này là theo từng khách. `crm` AI theo giờ (`aisummary.go` dòng 137).
- **Rủi ro còn lại:** 20 rpm của 9router là chung. Nhiều khách chat cùng lúc có thể làm chậm / 429 chat của lớp trình diễn; cùng làn `INTERACTIVE` nên Scheduler không ưu tiên được. → **Trước buổi bảo vệ: `DEMO_MODE=false`** (sửa env, khởi động lại gateway), hoặc giảm hạn mức sandbox về 0.

**5. Đặt lại sandbox định kỳ** (không `down -v`, không đụng lớp trình diễn)
- **Không xoá cứng theo lớp:** có 50 bảng mang `course_id`, phần lớn FK không `ON DELETE CASCADE`, và `audit_log` chỉ cho ghi thêm (trigger `audit_log_no_update`, `00001_pg_platform.sql` dòng 135). Xoá theo lớp vừa dễ gãy vừa phải sửa mỗi khi thêm bảng.
- **Xoay lớp thay vì xoá.** Mỗi đêm cron trên VPS chạy `scripts/sandbox-reset.sh`:
  1. Admin API **lưu trữ** lớp sandbox hiện tại (`adminArchive`). Lớp đã lưu trữ từ chối ghi bằng `409 COURSE_ARCHIVED` (`coursehttp/admin.go` dòng 360), nên tab cũ của khách không ghi được nữa.
  2. Xoá các khoá `ep:demo:lease:*`.
  3. `node scripts/seed.mjs --sandbox`: một chế độ mới của seed hiện có, chỉ dựng lớp sandbox + nhóm tài khoản, bằng **API thật**. Seed idempotent (US-P2-12 AC6), chạy được ở `APP_ENV=dev` của demo.
     - Tài khoản cho mượn đã có thì chỉ ghi danh vào lớp mới.
     - Đặt `llm_budgets` của lớp mới.
  4. `Sessions.RevokeAll` cho mọi tài khoản sandbox (qua lệnh `gateway admin`, kiểu `run-job` có kiểm `APP_ENV`).
- Lớp trình diễn không bị chạm: mọi bước lọc theo mã lớp sandbox hoặc miền email sandbox.
- **Chi phí LLM mỗi lần đặt lại:** seed có chấm bài / trích công thức qua 9router [SUY LUẬN: cỡ vài chục lời gọi]. Nên để `--sandbox` dựng **ít bài hơn** lớp 1 (khoảng 8 bài nộp).
- **Dữ liệu khách nhập** (chat, Hỏi đáp, tài liệu) nằm lại trong lớp đã lưu trữ. Ghi Nợ: lệnh dọn chữ tự do của lớp sandbox đã lưu trữ > 7 ngày, đúng danh sách bảng chat / forum / ticket / document. Chủ dự án quyết có cần trước bảo vệ không.

### Ảnh hưởng
- **OpenAPI:** thêm `POST /demo/sessions`:
  - request `{role}`;
  - `200` dùng lại schema phản hồi của `/auth/login`;
  - lỗi `403` (Origin), `404` (tắt), `422` (vai sai), `429` (`RATE_LIMITED` + `retry_after`), `503` (hết tài khoản rảnh).

  Contract test chạy với `DEMO_MODE=true`, và một ca `false` → 404.
- **Migration:** **không có.** Dùng lại `ReasonAdmin`; sandbox nhận biết bằng miền email; lớp nhận biết bằng mã lớp trong cấu hình seed.
- **Auth (cần PM / chủ dự án duyệt, theo `AGENTS.md` "Khi nào DỪNG"):**
  - một đường cấp phiên **không mật khẩu** cho tài khoản sandbox;
  - một middleware 403 cho tài khoản sandbox ở các route tài khoản / tích hợp / mời;
  - một điều kiện từ chối miền email sandbox khi đăng ký / mời;
  - cấu hình mới `DEMO_MODE` (bị cấm ở production).
- **Cấu hình demo (không phải code):** giá model 9router; ngân sách lớp sandbox; cron trên VPS; `NEXT_PUBLIC_DEMO_MODE=1` cho bản dựng frontend của demo.
- **Tài liệu:** `ARCHITECTURE.md` §5 (endpoint), `deploy/README.md` (cron, cách tắt trước bảo vệ), `DEMO_SCRIPT.md` (không dùng tài khoản sandbox để trình diễn).

### Rủi ro
| Rủi ro | Mức | Chặn bằng |
| --- | --- | --- |
| Bật nhầm ở production | Cao nếu xảy ra | `config.Load` từ chối `DEMO_MODE` + `APP_ENV=production`; route không gắn khi tắt; contract test hai trạng thái |
| Khách giữ tài khoản cho mượn lâu dài (đổi mật khẩu rồi `/login`) | TB | Chặn đổi mật khẩu / quản lý phiên cho miền sandbox; cho mượn mới thì `RevokeAll` |
| Login CSRF (trang lạ ép trình duyệt vào phiên sandbox) | Thấp | `cookieGuard` (Origin / `Sec-Fetch-Site`) như `/auth/refresh` |
| SSRF qua cấu hình IMAP | Cao nếu không chặn | Middleware 403 cho tích hợp |
| Tốn hạn mức 9router / làm chậm lớp trình diễn | **Cao** | Ngân sách lớp sandbox (cần giá > 0); giới hạn theo IP / toàn cục; **tắt `DEMO_MODE` trước bảo vệ** |
| Khách nhập thông tin thật vào chat | TB | Lớp xoay hằng đêm; lệnh dọn chữ tự do (Nợ); dữ liệu mô phỏng (D44) không trộn vào lớp trình diễn |
| Hai khách dùng chung một tài khoản | Thấp | Cho mượn theo lượt bằng `SET NX` + đá phiên cũ |

### Ước lượng: **L, khoảng 3–3,5 ngày dev** (chưa tính QC)
- Gateway M–L (1,5–2 ngày):
  - config `DEMO_MODE`;
  - endpoint + cho mượn + trần tần suất;
  - middleware chặn sandbox, chặn miền khi đăng ký / mời;
  - OpenAPI + contract test;
  - test bảng (tắt / bật / production / hết tài khoản / Origin lạ / đổi mật khẩu bị 403).
- Seed `--sandbox` + `scripts/sandbox-reset.sh` + lệnh `gateway admin` thu hồi phiên sandbox + cron: M (1 ngày).
- Frontend S (0,5 ngày): liên kết trên trang giới thiệu, xử lý `?thu=` ở `/login`, dòng lỗi.

### Đề xuất cho PM (chép vào `proposals.md` nếu đồng ý)
6. **"Dùng thử" theo thiết kế A của TL-3.** Gồm:
   - `POST /demo/sessions` chỉ có khi `DEMO_MODE=true`, cấm ở production;
   - nhóm tài khoản sandbox `@thu.edupilot.local` cho mượn 30 phút bằng Redis;
   - lớp sandbox xoay hằng đêm bằng seed `--sandbox` + lưu trữ lớp cũ;
   - chặn đổi mật khẩu / quản lý phiên / tích hợp / mời cho tài khoản sandbox;
   - ngân sách LLM riêng cho lớp sandbox;
   - không migration.

   **Cần chủ dự án duyệt vì đụng auth.** Kèm hai quyết định:
   - (a) đặt giá thật cho model 9router, khiến "Chi phí" của lớp trình diễn khác 0;
   - (b) tắt `DEMO_MODE` trong buổi bảo vệ.

### Bổ sung TL-3 (2026-11-08): `DEMO_MODE` bật cả trong buổi bảo vệ. Trần mặc định để lớp trình diễn ít bị ảnh hưởng
**Căn cứ:** P-91-3; chủ dự án đồng ý (a), chọn **bật** ở (b).

**Cơ chế đã có:**
- Lớp nào có ngân sách cạn thì gateway (`llm/gateway.go` dòng 126–132) xử lý theo làn:
  - **INTERACTIVE / NEAR_REALTIME không bị từ chối**, mà chuyển sang `Registry.Cheapest()` (model chat bật có giá thấp nhất, `registry.go` dòng 96–97);
  - BATCH thì dừng.
- Khi model 9router đã có giá thật (a), model rẻ nhất trên demo là `chat-gia` (fake, giá 0, đọc bản ghi `LLM_DEMO_FILE`). Vậy **lớp sandbox cạn hạn mức thì khách vẫn được trả lời (từ bản ghi), và 9router không bị gọi thêm.** Chỉ lớp sandbox có dòng `llm_budgets`, nên lớp trình diễn không bao giờ bị chuyển.
- **Còn thiếu:** trần **theo phút**. Ngân sách ngày không chặn được một đợt dồn trong một phút, mà 20 rpm là theo phút.

**Đề xuất cho dev (thay đổi nhỏ, không đụng token bucket của Scheduler):**
1. **Trần LLM theo phút cho lớp sandbox: 6 lời gọi / phút** (≤ 30 % của 20 rpm), nên lớp trình diễn luôn còn ≥ 14 rpm.
   - Ở `Scheduler.BudgetState(ctx, courseID)` (`scheduler.go` dòng 444): nếu `courseID` là lớp sandbox thì đếm `INCR` cửa sổ cố định 60 s trên Redis, vượt thì trả `BudgetExhausted`. Lời gọi đó đi theo nhánh có sẵn ở trên (chat → `chat-gia`, BATCH → dừng). **Không xếp hàng**, nên không chiếm hàng đợi của lớp trình diễn.
   - Mỗi tin chat theo D47 là 1 CLASSIFY + 1 CHAT, nên 6 rpm ≈ **3 câu hỏi / phút cho tất cả khách**. Câu vượt trần vẫn được trả lời, chỉ là từ bản ghi.
   - Id lớp sandbox: lệnh đặt lại ghi vào Redis `ep:demo:sandbox_course`, vì lớp xoay mỗi đêm. Gateway đọc kèm cache ≤ 60 s.
   - **Núm chỉnh nóng:** số 6 đọc từ Redis `ep:demo:llm_rpm` (mặc định 6). PM đặt `0` trong buổi bảo vệ nếu thấy chat lớp trình diễn chậm: mọi câu của sandbox sang bản ghi, 9router chỉ còn lớp trình diễn. Không cần khởi động lại.
2. **Ngân sách ngày của lớp sandbox ≈ 300 lời gọi** quy ra VND = 300 × chi phí trung bình một lời gọi đo từ `llm_audit` sau khi đặt giá. Tháng = 30 × ngày. Đặt mỗi lần đặt lại.
3. **Theo tài khoản cho mượn:** 10 tin chat / phút. Trần chung `CHAT_RATE_PER_MIN` mặc định là 20 (`config.go` dòng 357). Thêm trần 40 tin / lượt mượn 30 phút, đếm bằng `HitWindow` khoá theo `user_id`, cửa sổ = lượt mượn.
4. **Cấp phiên:** 5 phiên / 10 phút / IP; **60 phiên / giờ toàn cục** (thay cho 120 ở trên). Nhóm 10 SV + 3 GV vốn đã giới hạn số khách đồng thời ở 13.
5. **BATCH của sandbox** ("Chấm tất cả", tạo câu hỏi, báo cáo Insights): chịu chung trần 6 / phút ở mục 1. Seed `--sandbox` để ≤ 8 bài nộp, nên "Chấm tất cả" tốn khoảng 16 lời gọi (hai lượt chấm) trải qua vài phút, không dồn.
6. **Tải tài liệu:** 3 tệp / giờ / tài khoản, ≤ 5 MB. Embedding đi Gemini trực tiếp, không qua 9router, nên không ăn 20 rpm.

**Kiểm (dev / QC):**
- k6 hoặc script: 10 khách sandbox chat liên tục trong lúc 20 sinh viên lớp trình diễn chat. Kỳ vọng:
  - p95 TTFT của lớp trình diễn tăng ≤ 20 % so với khi không có khách;
  - lượt `llm_audit` của sandbox đi 9router ≤ 6 / phút;
  - phần vượt có `model = fake-chat`.
- Đặt `ep:demo:llm_rpm=0` → không còn lời gọi 9router nào từ lớp sandbox.

**Ước lượng thêm:** S (≈ 0,5 ngày), nằm trong L của TL-3.

**Huỷ theo P-91-4** (2026-11-08): bỏ "Dùng thử". Không làm endpoint `/demo/sessions`, `DEMO_MODE`, sandbox hay đặt giá 9router; TL-3 và phần bổ sung chỉ còn để tham khảo.
