# Handoff dev: sprint 9.1 pha 1

Nhánh `sprint/9.1-uiux` (đã gộp `ops/vps-deploy`). Mọi build / test chạy trên VPS 12 vCPU / 32 GB (`/srv/work/dev`).

## A. CI `ops/vps-deploy`

**Nguyên nhân (tái hiện được, trùng số với CI):** `/chat` 258 360 B và `/me` 257 410 B so với ngưỡng 256 000 B (`resource-summary:script:size`). Next prefetch chunk của mọi route trong thanh bên; chunk của `/practice` mang dữ liệu mô phỏng `@/mock/*` (~17 KB gzip) và được nạp tĩnh. Chunk đó chỉ nằm trong manifest của `/practice`, nhưng bị prefetch ở `/chat`, `/me`.

**Sửa (không nới ngân sách):** `features/practice/PracticeLazy.tsx` nạp lười ba màn Luyện đề (`dynamic`, cùng mẫu với `ChatScreen`, `LibraryScreen`…); ba `page.tsx` của `/practice` trỏ về đó. Commit `8c68df9a` trên `ops/vps-deploy`.

| Kiểm | Kết quả |
| --- | --- |
| `lhci autorun` (simulate, 8 URL × 1 lượt) trên VPS, trước sửa | `/chat` 258 360, `/me` 257 410: đỏ như CI |
| cùng lệnh, sau sửa | không còn `✘` |
| Playwright cả bộ, `CI=1` (2 worker, retry 1), desktop + mobile, `--grep-invert @real` | 824 passed, 464 skipped, 0 failed (12,5 phút) |
| GitHub CI `ops/vps-deploy` @ `8c68df9a` | xanh (run 42475773053) |

Đã gộp `ops/vps-deploy` vào `sprint/9.1-uiux`. Cả bộ chạy lại trên nhánh sprint (có trang giới thiệu mới) ở mục dưới.

## B. Trang giới thiệu mới + video

### Video (`scripts/landing-video/`, dựng lại: `bash scripts/landing-video/run.sh` trên VPS)
- Theo TL-1 A: `page.screencast.start({onFrame, size 1440×900, quality 90})` → khung JPEG + `frames.ffconcat` (độ dài mỗi khung = hiệu `timestamp` ÷ 1000) → ffmpeg 7.1 trong `debian:trixie-slim` (`encode.sh`): mp4 H.264 `-preset slow -tune stillimage -crf 28 -maxrate 700k`, webm VP9 `-crf 46`, poster WebP 960 px q68.
- Phụ đề tiếng Việt bằng `screencast.showOverlay` (phông Be Vietnam Pro đúng, đã xem khung). 7 câu, nằm ở mảng `CAPTIONS` của `record.mjs` và `VIDEO_CAPTIONS` của `Landing.tsx` (lời thay cho video không tiếng, WCAG 1.2.1, `#video-loi` ẩn với mắt).
- **Lệch TL-1:** không dùng `showActions`. Nó luôn in nhãn "Mouse move" / "Click" lên hình (`fontSize: 0` không tắt được, đã thấy ở khung 16 s). Con trỏ vẽ bằng `addInitScript` (vòng đỏ, thu nhỏ khi nhấn), 20 dòng trong `record.mjs`: phương án dự phòng TL-1 đã nêu ở rủi ro 4.
- **Lệch TL-1 / PM:** quay trên bản demo thật nhưng **chỉ đọc** (không gửi chat, không duyệt / công bố / sửa điểm; chat mở lại một hội thoại seed có sẵn). Không ghi gì vào DB, không tốn hạn mức 9router, nên không cần stack `-p ep-video`. Mỗi cảnh đăng nhập mới: refresh token xoay vòng, dùng lại phiên đã bị thu hồi (đã gặp).
- Kịch bản: thẻ mở đầu 4 s → chat riêng của sinh viên (dòng "Đã ẩn … thông tin cá nhân", nguồn tham khảo) → hộp thư hỗ trợ → duyệt bài AI chấm nháp (5,25, đoạn trích, tiêu chí lệch) → sổ điểm + "Giải trình" → Hôm nay → thẻ kết. **64,6 s.**
- Tệp ở `frontend/public/landing/`: `gioi-thieu-v1.mp4` 1,0 MB, `.webm` 1,5 MB, `.webp` 52 KB (đều < 8 MB). VP9 ở đây to hơn H.264 vì cảnh gần như tĩnh (mp4 ~128 kbps); `<source>` đặt mp4 trước.
- Việc cần biết khi dựng lại: các màn ứng dụng không bao giờ `networkidle` (SSE), nên dùng `waitUntil: "load"`; giảng viên mặc định ở lớp 761988 (chưa có bài, chưa có công thức), script chọn lớp 761987 qua bộ chọn lớp.

### Trang (`frontend/src/features/landing/`)
- Theo mục 3.A của `audit/findings.md` và các quyết định Q-UX91-5 + [ĐỔI DESIGN]: hero (h1 giữ nguyên, vạch đỏ, "AI soạn nháp. Giảng viên quyết. Điểm do mã tính.") + video; sơ đồ ba bước có vạch đỏ nối; ba số đo viết thành câu ("146 yêu cầu gửi AI, 0 lần lộ họ tên hay MSSV", "30/30 …", "22 bài chấm cùng lúc, chat không chậm thêm") kèm dòng "Đo trên dữ liệu mô phỏng"; hai nút radio Giảng viên / Sinh viên (mỗi vai 4 mục, đổi bằng CSS `:has`); CTA cuối trang.
- **Chuyển động: 0 KB JS, chỉ CSS** (TL-1 B). `Landing` vẫn là Server Component. Toàn bộ hiệu ứng nằm trong `@media (prefers-reduced-motion: no-preference)`; hiệu ứng theo cuộn còn nằm trong `@supports (animation-timeline: view())`. Ngoài các khối đó là trạng thái cuối, đủ chữ.
  - mở đầu: vạch đỏ vẽ ra, đoạn dẫn + nút trượt lên; h1 và khung video **không** mờ dần (LCP);
  - theo cuộn (`view()`): khối hiện dần, sơ đồ ba bước (vạch đỏ `scaleX` / `scaleY` ở điện thoại, dấu bước đổi sang đỏ lần lượt, "7,5" bị gạch, "8,5" hiện), số đếm lên (`@property --n` + `counter()`, chỉ số nguyên: 146, 30, 22), ảnh sản phẩm nhô lên, ảnh điện thoại trôi ngược chiều cuộn;
  - nút: `:active` thu 0,97, 140 ms; đổi vai: mờ chéo 160 ms.
- Video: `<video controls preload="none" poster>` thuần HTML, không tự phát, không tiếng. Nút phát đỏ là hai pseudo-element CSS, bấm xuyên qua tới video; `PlayState.tsx` (client, ~0,5 KB, không render gì) gắn `data-playing` khi phát để ẩn nó (`:paused` chưa có trong Chromium dùng ở đây, đã thử và bỏ).
- Đã xoá `giang-vien-duyet-bai(.-640).webp` (không còn dùng).
- **Không làm (xin PM xác nhận):** nút phụ "Xem video 75 giây ▶" ở hero (video nằm ngay dưới nên nút chỉ cuộn vài chục px); đường đỏ SVG chạy từ "5,25" tới "Duyệt bài" trong khung sản phẩm (hero là video, không phải ảnh tĩnh để vẽ lên); dự phòng `IntersectionObserver` cho trình duyệt thiếu `animation-timeline` (TL-1: nội dung hiện tĩnh, không mất chữ; thêm JS vào bundle của `/` không đáng).

### Số đo (bản demo công khai, VPS 12 vCPU / 32 GB, Lighthouse 12, ẩn danh)
| Chỉ số | Di động (throttle thật, 3 lượt) | Di động (simulate, 3 lượt) | Máy tính (simulate, 3 lượt) |
| --- | --- | --- | --- |
| LCP | 2,30 · 2,31 · 2,32 s (phần tử LCP: h1) | 2,63 s (h1; simulate tính cả 216 KB JS vào đồ thị LCP, CI cũng không chấm LCP ở lượt này) | 0,55 · 0,55 · 0,61 s (video poster) |
| TBT | 114–131 ms | 57–70 ms | 0 ms |
| CLS | 0,002 | 0 | 0 |
| JS truyền | 216 KB (< 256 KB) | – | – |
| Hiệu năng / A11y | 97 / – | 96–97 / 100 | 100 / 100 |

- **Bài học LCP di động:** poster nằm trong màn hình đầu thì LCP 3,1 s (poster 52 KB tranh băng thông 1,6 Mbps với phông + 215 KB JS; đổi poster 84 → 52 KB và nạp lười ảnh sản phẩm không giúp). Sửa: ở < 720 px khối chữ đầu trang cao gần hết màn hình, video nằm dưới nếp gấp, `preload` poster chỉ cho ≥ 720 px; chữ là LCP.
- Không cổng tự động nào đo LCP của trang giới thiệu (`lighthouse-auth.cjs` gán `/` vào vai sinh viên); số trên là lượt đo tay: `/srv/work/dev/lh.sh` (simulate), `lhdt.sh` (throttle thật, ngoài repo).
- axe: `landing.spec.ts` (1440 + 375) 0 lỗi nghiêm trọng / nặng. Console có một lỗi `401 /auth/refresh` ở mọi lần vào trang công khai (đã ghi UX-3d).

### Test đã đổi / thêm (`e2e/landing.spec.ts`)
- Sửa danh sách `h2` (spec cũ khớp nguyên văn ba `h2`; nay bốn: "AI soạn nháp, giảng viên quyết, mã tính điểm", "Khác biệt, có số đo", "Cho giảng viên và sinh viên", "Bắt đầu với lớp của bạn") và `getByRole("link", { name: "Tạo tài khoản" }).first()` vì có thêm CTA cuối. **Đây là sửa test theo yêu cầu thiết kế mới, không phải nới assertion; BA / QC cần đưa vào spec.**
- Thêm: video không tải khi chưa bấm + có poster + lời thay; nút phát ẩn / hiện theo sự kiện; hiện dần theo cuộn và reduced-motion hiện đủ ngay; đổi vai Giảng viên / Sinh viên bằng chuột và mũi tên.

### Triển khai
Bản demo ở `https://v2202609419695521098.happysrv.de/` đã dựng lại **chỉ frontend** từ `sprint/9.1-uiux` (`/opt/edupilot` đang ở nhánh này). Ảnh mốc `visual.spec` chụp `/` ở vai sinh viên đã đăng nhập nên không phải sinh lại.

Ảnh: `docs/sprints/9.1/handoff/dev-phase1-shots/` (1440 và 375, đầu trang, sơ đồ, số đo, đổi vai, CTA). Video: `/landing/gioi-thieu-v1.mp4` trên bản demo.

## Cổng chạy trên nhánh `sprint/9.1-uiux` (VPS, mã ở `765dabfb` + CSS di động về sau)
| Kiểm | Kết quả |
| --- | --- |
| `pnpm lint`, `bash scripts/ui-antipatterns.sh` | sạch (22 phép ✓) |
| `lhci autorun` simulate, 8 URL × 5 lượt (cấu hình CI, ngưỡng JS 256 KB giữ nguyên) | không `✘` |
| Playwright cả bộ, `CI=1`, desktop + mobile | 827 passed, 467 skipped, 0 failed (12,3 phút) |
| GitHub CI | xanh ở `765dabfb` (run 42480063791) và `7666513f` (run 42481287269); lần đẩy cuối chỉ đổi CSS di động, chờ CI |

## Cần PM / BA
1. Hai mục "Không làm" ở trên.
2. `DESIGN.md` §11 / mục "Trang giới thiệu (Persuade)" do BA sửa (đã có quyết định chủ dự án): code đã theo ngoại lệ (mở đầu ≤ 600 ms, phần tử LCP không mờ dần, hiệu ứng cuộn chỉ `transform` / `opacity` trong `no-preference`; ngoại lệ nhỏ: màu viền dấu bước và đếm số).
3. Dialog / drawer 250 ms → ≤ 200 ms (đề xuất 3 của TL-1) thuộc pha 2, chưa làm.
