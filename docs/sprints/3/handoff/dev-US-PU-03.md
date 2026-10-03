# DEV handoff — US-PU-03 (lớp dữ liệu: apiClient, TanStack Query, useSSE, nháp, hoàn tác, mất mạng)
Nhánh `sprint/3-pu-p1`. Góp ý #19–#21. Chạy: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test` (cổng **3310**, gateway giả **3312**; 96 ca desktop + mobile). `data-layer.spec.ts` chỉ chạy ở dự án desktop, tuần tự (máy chủ giả dùng chung).

## Làm gì
- `shared/data/`: `apiClient` (URL `${NEXT_PUBLIC_API_URL}/api/v1`, Bearer từ `tokenStore` — bộ nhớ, `credentials: include`, `X-Request-Id`, `Idempotency-Key` cho POST/`idempotent:true`, `If-None-Match` từ cache ETag, timeout 15 s, origin lạ ⇒ `BAD_TARGET` TRƯỚC khi gửi, thử lại GET 300/900 ms ±20 % hoặc đúng `retry_after` ≤ 5 s, `auth:expired` ≤ 1/10 s + xoá token), `ApiError` (+ `conflict`, `fieldErrors`), `tokenStore`, `netStatus`, `QueryProvider` (staleTime 30 s, gcTime 5 phút, `retry:false`, `offlineFirst`), `useCursorList`, `useIdempotentMutation`, `sse.ts` (manager singleton, fetch + ReadableStream) + `useSSE`/`useSSEStatus`, `useJob`, `useAutosaveDraft`, `useUndoableAction`, `OfflineBanner`, `ApiErrorNotice` (đếm ngược `retry_after`, "Chi tiết kỹ thuật" chỉ khi `showTechnical`).
- `shared/i18n/vi.ts`: 28 mã + 4 mã phía client → `userMessage` (không từ kỹ thuật).
- `PageState` nhận `query` + `isEmpty` (+`showTechnical`); `?state=` tắt khi `NEXT_PUBLIC_MOCK_SCREENS=0`. `InlineNotice` có `technical`. `Dialog` có `aria-describedby`; `ConfirmIrreversible` trỏ `aria-describedby` của nút tới lý do khoá.
- Root layout: `QueryProvider` + `OfflineBanner` (dải rộng 0 khi online).
- `/dev/data` (`page.dev.tsx`, chỉ build cổng) + `e2e/support/api-server.mjs` (gateway giả có kịch bản, ghi log, SSE).

## AC tự đánh giá
| AC | Kết quả thật |
| --- | --- |
| 1 | `apiClient basics`: header `Authorization`, `X-Request-Id` uuid, `Accept`; `https://evil.example/x` ⇒ `BAD_TARGET`, 0 request ra ngoài; `grep 'setItem(' … token` ở `shared/data` = 0 |
| 2 | `error mapping`: bảng **28** dòng (status + thân) ⇒ `code`, `status`, `retryAfter`, `userMessage` khác rỗng, `conflict` của `VERSION_CONFLICT`; mã lạ ⇒ câu chung |
| 3 | `non-json`: HTML 502 ⇒ `BAD_GATEWAY` (không `<html`), JSON hỏng ⇒ `PARSE_ERROR`, đứt ⇒ `NETWORK` ("Chữ bạn đã nhập vẫn được giữ"), huỷ ⇒ `ABORTED` |
| 4 | `error display`: `?as=student` không có `VERSION_CONFLICT` / `INTERNAL` / 32 hex; `?as=admin` có `<details>` gấp sẵn chứa `trace_id` |
| 5 | `retry`: 503,503,200 = 3 request cách ≥ 240 / 720 ms; 404 = 1; 429 `retry_after:30` = 1; 503 `retry_after:2` = 2 request cách 2000–2250 ms; PATCH không khoá + đứt = 1; POST có khoá + đứt = 2 request cùng khoá |
| 6 | `idempotency key`: khoá khớp regex; sau lỗi `Gửi lại` dùng cùng khoá (4 request cùng khoá), gửi mới khoá khác; `Idempotent-Replayed: true` ⇒ `replayed` và không hiện gì. **`@real` (2 tab cùng khoá) chưa chạy** |
| 7 | `auth expired`: 5 request song song 401 ⇒ đúng 1 `auth:expired`, 5 request (không gửi lại), token bị xoá |
| 8 | `etag` (gateway giả): lần hai gửi `If-None-Match`, `304` trả đúng tham chiếu cache (`===`). **`@real` trên `_test/items` chưa chạy** |
| 9 | `field errors`: 422 ⇒ ô `name` `aria-invalid=true`, lỗi dưới ô, nội dung giữ nguyên; `error.conflict` ở bảng mã |
| 10 | `retry-after notice`: "3 giây" ⇒ "2 giây", nút `Thử lại` `disabled` rồi bấm được sau hết giờ; `role=status` |
| 11 | `cursor list`: gộp + loại trùng (a,b + b,c ⇒ 3), `limit=500` ⇒ request `limit=100`, `INVALID_CURSOR` ⇒ đúng 1 lần quay về trang đầu. **250 bản ghi thật `@real` chưa chạy**; mặc định `QueryProvider` do `makeQueryClient` |
| 12 | `sse parse`: `retry:`, `ready`, khung `data` hai dòng nối `\n`, `: hb` bỏ qua; header `Authorization`, URL không `token=` |
| 13 | `sse reconnect`: e1…e13 mỗi sự kiện đúng một lần, theo thứ tự (máy chủ gửi bù lặp e4, e5), `Last-Event-ID` = id của e5, cách lần nối đầu ≥ 800 ms |
| 14 | `sse control`: `reconnect` nối lại ≤ 600 ms (không backoff) mang `Last-Event-ID`; `shutdown` ⇒ nối sau 1–3,7 s; `resync` ⇒ vô hiệu hoá query đúng 1 lần |
| 15 | `sse errors`: 401 ⇒ 1 request, `closed`, `auth:expired`; 429 `retry_after:5` ×3 ⇒ `degraded` + câu "Bạn đang mở nhiều cửa sổ; cập nhật tự động tạm dừng", thử lại một lần sau 60 s; treo > 40 s (`page.clock` +41 s) ⇒ nối lại. 503 dùng chung backoff (không có ca riêng) |
| 16 | `sse singleton`: gắn 3 người nghe ⇒ **1** request `/events`; gỡ hết ⇒ máy chủ thấy đóng; `@real` 3 tab chưa chạy |
| 17 | `useJob`: GET trước (1 request khi SSE mở), `progress` 60 → 40 không giảm, kết thúc `SUCCEEDED:100`; SSE hỏng ⇒ thăm dò cách ≈ 2 s tới `SUCCEEDED`; `FAILED` hiện lỗi. **`@real` `_test/jobs` chưa chạy** |
| 18 | `autosave`: gõ, 2,5 s, `page.close({runBeforeUnload:false})`, mở lại ⇒ có "Xin chào thầy"; `QuotaExceededError` ⇒ `status=error`, chữ vẫn trong ô; `useAutosaveDraft("llm.apiKey")` ném lỗi; `clear()` xoá khoá |
| 19 | `undoable`: đổi UI ≤ 100 ms (trong lúc máy chủ trễ 400 ms), `[data-part=undo-line]` cùng vùng, `Hoàn tác` gửi `PUT {on:false}` ngay, biến sau 5,2 s (`page.clock`), 500 ⇒ UI về cũ + `role=alert`, không có `status` "Thành công" |
| 20 | `ui-foundation › ConfirmIrreversible`: tệp `tsc` thiếu `consequence` ⇒ lỗi nêu `consequence`; `dev-ui › confirm`: `disabledReason` ⇒ nút `disabled` + `aria-describedby` trỏ đúng lý do |
| 21 | `offline banner`: `setOffline(true)` ⇒ hiện ≤ 1 s, ô nhập vẫn gõ được; có mạng + một request thành công ⇒ biến ≤ 2 s |
| 22 | `pagestate`: pending ⇒ `aria-busy`; 500 ⇒ `role=alert` + `Thử lại` ⇒ đúng +1 request; `MOCK_SCREENS=0` tắt `?state=` ở `useRouteState` (chưa dựng bản `MOCK_SCREENS=0` để chạy) |
| 23 | `token hygiene`: sau khi dán token — `localStorage`/`sessionStorage`/cookie/URL/console không chứa token. **`pbuild; grep DEV_AUTH` thuộc cổng dán token ở US-PU-04** |
| 24 | `eslint` rc=0, `fetch(` ngoài `shared/data` = 0, `ui-antipatterns` 0 ✗, `lint-selftest` 7/7 + 19/19, `ui-allow` = 10, bản dựng thường không chứa `state-cell`/mã `/dev/*`. **`audit.mjs` và ngân sách JS (LHCI) là lượt của PU-05 / QC** |

## Nợ / ghi chú
- Các ca `@real` (6, 8, 11, 13, 16, 17) cần stack Go + bản dựng `NEXT_PUBLIC_API_URL=https://localhost`; chưa chạy (cùng nội dung đã kiểm bằng gateway giả).
- `useSSE` chưa có ca riêng cho `503` (dùng chung đường backoff với lỗi mạng).
