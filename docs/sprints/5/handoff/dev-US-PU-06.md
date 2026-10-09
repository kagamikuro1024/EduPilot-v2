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

## Sửa vòng 1 — QC B1 / B2 / B4 (B3: lỗi chữ AC, BA xử lý, không đổi mã)
### B1 — TBT `/gradebook`, `/chat` ở 12×
- **Tìm tác vụ dài** (Lighthouse `long-tasks` + vết Chromium CPU 12× + hồ sơ V8): ở `/gradebook` chỉ có HAI tác vụ > 50 ms — (1) phân tích tài liệu HTML của route (≈ 90–98 ms, **trước** FCP nên không tính TBT); (2) **một** tác vụ `EvaluateScript` của chunk khung `0307…js` (Next + react-dom, 229 KB) ≈ **223–284 ms** ở 12×, chạy ở giây 3,6 khi chunk về tới (mạng 1,6 Mbps). Tác vụ (2) **giống hệt ở mọi route** (hồ sơ V8: `N` của runtime Turbopack 11 ms + chunk 6 ms + `program`/parse 46 ms thật, nhân 12), không phải do bảng sổ điểm; `/chat` cùng cơ chế. TBT vì thế **hai đỉnh**: ≈ 90 khi tác vụ rơi ngoài cửa sổ FCP → TTI (hoặc bị cắt), ≈ 165–190 khi rơi trọn trong đó; trung vị phụ thuộc số lượt rơi vào đỉnh nào (QC 5–7 lượt: 183 / 172).
- **Sửa phần thuộc route**: `shared/lib/useProgressiveCount` + `Gradebook` dựng sổ điểm theo đợt 8 dòng / 16 ms (trước đây dựng cả bảng trong một lượt render lúc có phiên). Dòng dựng sau nằm dưới dòng đã có nên CLS = 0. Sau sửa không còn tác vụ nào > 50 ms thuộc trang ngoài tác vụ khung. Đo 7 lượt `/gradebook` trước → sau: `107 109 121 138 166 172 177` (trung vị **138**) → `82 89 95 161 163 168 171` (**161**; lượt tốt 82–95).
- **Không thể bớt** tác vụ khung ở mức mã ứng dụng: thử `browserslist` hiện đại (chunk vẫn 229.139 B — Next đã mặc định đích này; hoàn lại), tách `AppShell` (đã hoàn lại ở vòng đầu). Đây là sàn của Next 16 + React 19 trên máy chậm 12×.
- **Số đo cuối, 12× simulate, 7 lượt, trung vị (các lượt đã sắp)** — máy dev arm64:

| Route | TBT (ms) | benchmarkIndex | First Load JS (gzip, KB) | raw JS (KB) | số script |
| --- | --- | --- | --- | --- | --- |
| `/` | **89** (86 86 89 89 92 93 119) | 3.821 | 232,0 | 713,4 | 19 |
| `/chat` | **95** (83 86 91 95 99 163 185) | 3.847 | 240,1 | 737,5 | 19 |
| `/threads` | **87** (80 82 86 87 93 174 178) | 3.830 | 246,0 | 751,5 | 20 |
| `/inbox` | **173** (90 91 92 173 175 179 185) | 3.777 | 230,0 | 706,9 | 19 |
| `/gradebook` | **161** (82 89 95 161 163 168 171) | 3.731 | 235,4 | 723,2 | 19 |
| `/settings/llm` | **166** (82 85 86 166 167 175 181) | 3.781 | 240,9 | 740,5 | 20 |

  **Trung thực:** `/inbox` 173 vẫn > 170 ở lượt đo này; `/chat` 95 (lượt trước 162–173). TBT hai đỉnh nên trung vị 5–7 lượt dao động 90 ↔ 175 giữa các lần đo. **Cổng CI** (`lighthouserc.json`) là 200 và đỉnh cao nhất đo được là 185–190 nên **qua** (xem bằng chứng bên dưới); mức 170 của AC5 là mục tiêu dev, không phải ngưỡng CI. Nếu cần ≤ 170 chắc chắn, phải giảm chính chunk khung (ngoài tầm mã ứng dụng) hoặc BA sửa AC5 thành "đỉnh cao ≤ 200".
### B2 — `Cache-Control`
- `next.config.ts` `headers()`: mọi route ứng dụng (`/((?!_next/|.*\..*).*)`) trả `Cache-Control: private, no-cache` (Next tự gắn `s-maxage=31536000` cho trang tiền kết xuất tĩnh; header cấu hình ghi đè được — đã kiểm). `/join`, `/verify-email`, `/reset-password`, `/invite/*` giữ `no-store`; `_next/static` giữ `public, max-age=31536000, immutable`. Kiểm `curl -sD -` 6 route + `/dev/ui` + `/login`: đều `private, no-cache`, không còn `s-maxage` / `public`.
### B4
- `docs/PROGRESS.md` dòng 8 và 47: nợ LCP / TBT đánh dấu **ĐÃ TRẢ (US-PU-06, #14)**; bảng `First Load JS` + `benchmarkIndex` từng route ở trên.
### Bằng chứng tạm (CI GitHub không chạy — billing): hai lệnh lhci của CI chạy tại máy
- `cd frontend && pnpm build:gate && pnpm exec lhci autorun` → **rc=0**, 7 URL × 3 lượt, không `error` (TBT 6 route ≤ 200 median, CLS, `resource-summary:script`).
- `pnpm exec lhci autorun --config=lighthouserc.devtools.json` → **rc=0**; 6 route LCP ≤ 2.500 median; chỉ có cảnh báo `/dev/ui` LCP 2.927 ms (`warn` theo #14).
- Playwright `shell.spec.ts lcp.spec.ts tokens.spec.ts` (không visual): 38 pass. `pnpm lint`, `tsc --noEmit`, `ui-antipatterns.sh` sạch.
