# DEV handoff — US-PU-06 (khung vẽ từ máy chủ, TBT / LCP — nợ #14 + #15 sprint 4)
Nhánh `sprint/5-pe`. AC chính thức của story viết ở `FEAT-ui-foundation` v1.4 (BA); số dưới đây theo prompt dev + TL-1 / TL-2.

## Làm gì
- **Khung trước phiên vẽ từ máy chủ** (`shared/shell/PreShell.tsx` + `intro.ts`): khi chưa có phiên, `AuthGate` trả thanh trên + cột bên (cùng lớp CSS với `AppShell`, cùng hình học ⇒ CLS 0) + tiêu đề / câu phụ của route (HTML tĩnh, có chữ ngay từ HTML). Route có mục: `/`, `/chat`, `/threads`, `/inbox`, `/gradebook`, `/settings/llm`; route khác giữ khung xương. **Quy tắc LCP (AC8):** khối chữ vẽ đầu phải là khối chữ LỚN NHẤT màn sẽ có sau phiên (chữ client lớn hơn ⇒ ứng viên LCP mới); `/chat` đưa cả đoạn giới thiệu vào khung; `/` dùng tiêu đề dài hơn "Chào {tên}".
- **`experimental.inlineCss`** (`next.config.ts`): CSS vào HTML. TBT 12× giảm mạnh (`/inbox` 195→85–130); HTML to hơn (≈ 387 KB, CSS lặp trong payload RSC) nhưng FCP không đổi.
- **`font-display: block`** (đề xuất #14, đụng `FEAT-ui-foundation` AC5 `swap`): với `swap`, phông thật về sau làm chữ thật rộng hơn chữ khung ⇒ ứng viên LCP mới lúc có phiên (đo bằng `PerformanceObserver` + mạng giả 1,6 Mbps). Phông được `next/font` preload nên khoảng chờ ≈ 0,5–0,8 s ở mạng 4G chậm. `tokens.spec.ts` sửa theo; hoàn lại 1 dòng + 1 test nếu PM từ chối.
- **`/dev/ui`** vẽ từ máy chủ: bỏ `useSearchParams` + `Suspense` (hook `useQueryParam` bằng `useSyncExternalStore`).
- **e2e `lcp.spec.ts` (`lcp before refresh`)** 6 route: giữ `/auth/refresh`, mạng + CPU giả như Lighthouse; khẳng định mọi ứng viên LCP trước phiên thuộc `PreShell` và KHÔNG có ứng viên mới sau phiên.
- Đã thử và **hoàn lại**: tách `AppShell` bằng `import()` (JS tổng 252→292 KB); giữ khung xương tới khi `/me/courses` về; `useDeferredValue` cho lớp thật; `justify` + giãn chữ để bù phông dự phòng (đạt e2e nhưng devtools Lighthouse vẫn 3,5 s ở 4/6 route).

## Số đo (máy dev arm64, benchmarkIndex ≈ 3700, Lighthouse 12.6.1, 3 lượt, trung vị)
| Route | TBT simulate 12× (≤ 170) | LCP **devtools** 4× (≤ 2.500) | LCP simulate 4× | FCP | CLS |
| --- | --- | --- | --- | --- | --- |
| `/` | 91 | 1.443 | 3.4–3.9 s | ≈ 1,05 s | 0 |
| `/chat` | 162 | 1.494 | 3.5–4,0 s | ≈ 1,05 s | 0 |
| `/threads` | 102 | 1.491 | 3.5–4,2 s | ≈ 1,05 s | 0 |
| `/inbox` | 131 | 1.913 | 3.5–4,1 s | ≈ 1,2 s | 0 |
| `/gradebook` | 82 | 1.929 | 3.6–3,9 s | ≈ 1,2 s | 0 |
| `/settings/llm` | 113 | 1.491 | 3.5–4,0 s | ≈ 1,05 s | 0 |
| `/dev/ui` | 184 | 2.919 | 3.2–3,5 s | 1,66 s | 0 |
Trước story (bản `1229114` sprint 4): TBT 12× 195–239, LCP 3,4–4,2 s ở cả hai cách.
- **TBT:** đạt ≤ 170 ở 6 route người dùng (`/chat` sát: 162; `/dev/ui` giữ `warn`).
- **LCP:** ssửa lỗi thật (devtools 3,5–4,3 s → 1,4–1,9 s). **Simulate vẫn > 2,5 s** — sàn tải ≈ 254 KB JS ở 1,6 Mbps (cả `/login` 3,3 s), TL-2 mục 3. Góp ý **#14** (b'): thêm lượt `lhci` devtools chỉ cho LCP. Góp ý **#14 ACCEPTED**: `lighthouserc.json` (simulate) trả TBT 6 route về `error` (ngưỡng 200, 4× không đổi), LCP không chấm ở lượt này; thêm `lighthouserc.devtools.json` + bước CI `lighthouse ci (devtools, chỉ LCP)` — LCP ≤ 2.500 ms `error` cho 6 route, `/dev/ui` `warn`.
- `/dev/ui` (devtools 2,9 s với `block`; 0,76 s khi còn `swap`): chỉ có ở bản dev, giữ `warn`.

## Gate
`pnpm -C frontend lint` sạch; `tsc` sạch; `ui-antipatterns` 0 ✗; Playwright không visual + `lcp.spec.ts`: xem commit. Ảnh visual chưa sinh lại (không đổi giao diện sau phiên); CI sẽ báo nếu lệch.

## Nợ
- BA sửa `FEAT-ui-foundation` AC4 / AC5 / AC6 (v1.5), QC sửa `tc-US-PU-06` + `tokens.spec` theo #14 (dev đã sửa `tokens.spec.ts` sang `block`; QC có thể viết lại).
- `/` cho vai GV / TA / Admin: sau phiên có đoạn trống "Khi có yêu cầu vào lớp…" lớn hơn chữ khung (chỉ khi không có dữ liệu); gate dùng sinh viên nên không thấy.
- `intro.ts` giữ chữ khung riêng — đổi chữ màn thật thì phải đổi ở đây (e2e `lcp before refresh` bắt hồi quy với dữ liệu giả).
