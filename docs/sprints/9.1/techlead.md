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
Chờ `docs/sprints/9.1/audit/findings.md` (2026-11-08 chưa có trên `sprint/9.1-uiux`).

### Đề xuất cho PM (chép vào `proposals.md` nếu đồng ý)
1. **Quay video trên stack riêng.** Vấn đề: `plan.md` ghi "quay màn hình bản demo thật"; quay trên `/opt/edupilot` sẽ ghi chat / duyệt / công bố vào DB demo và tốn hạn mức 9router. Đề xuất: quay trên stack `-p ep-video` cùng bản dựng, seed như demo, provider `fake`. Lý do: tất định, quay lại được, không đụng demo (TL-1 A, rủi ro 1). Ảnh hưởng: chỉ đổi chữ trong `plan.md`; hình như nhau.
2. **Ngoại lệ chuyển động cho trang giới thiệu trong `DESIGN.md` §11.** Vấn đề: hướng chủ dự án chốt (hiệu ứng mở đầu, xuất hiện theo nhịp, chiều sâu) trái §11 "No staggered page-load choreography" và có thể trái §7 (bóng). Đề xuất: thêm mục "Ngoại lệ: trang giới thiệu (Persuade)":
   - mở đầu ≤ 600 ms;
   - phần tử LCP không bắt đầu từ `opacity: 0`;
   - hiệu ứng theo cuộn chỉ dùng `transform` / `opacity`, bọc `prefers-reduced-motion: no-preference`.

   Lý do: TL-1 B. Ảnh hưởng: BA sửa `DESIGN.md` §11; không đổi code app.
3. **Rút thời lượng dialog / drawer về ≤ 200 ms.** Vấn đề: `Dialog.module.css` dòng 10, 25 dùng `--ep-slow` 250 ms; `DESIGN.md` §11 ghi 200–250 ms, trái luật ≤ 200 ms vừa chốt. Đề xuất: `--ep-slow` → 200 ms (hoặc dùng `--ep-normal`), sửa §11 thành "≤ 200 ms". Ảnh hưởng: một dòng token + một dòng doc; ảnh mốc không đổi.
