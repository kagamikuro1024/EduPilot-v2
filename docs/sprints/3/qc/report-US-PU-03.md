# Báo cáo QC — US-PU-03 (lớp dữ liệu: apiClient, TanStack Query, useSSE, useJob, nháp, hoàn tác, mất mạng)
**Kết luận: PASS (sau vòng sửa 1; nhóm Q còn lại dựa spec dev)** — vòng 1: FAIL do 2 lỗi nhẹ do QC tự đo bằng máy chủ giả của QC (BUG-PU03-1: `traceId` không lấy từ header `X-Request-Id`; BUG-PU03-2: sự kiện `reconnect` mở **hai** kết nối SSE cùng lúc). Phần còn lại: bộ Playwright của dev xanh, phần backend thật đạt, nhưng **nhóm TC "Q" mới đo độc lập một phần** (liệt kê rõ ở "Chưa đo độc lập").

- Bản chấm `00425f0` (chứa `dc2ecab` + sửa P1-04/PU-01; không có WIP US-PU-04), `build:gate` (`NEXT_PUBLIC_API_URL=http://localhost:3311` đóng sẵn trong `package.json`), `next start -p 3400`; gateway thật `testroutes` cổng 8080/8081 + worker (`WORKER_HEALTH_ADDR=:8095`). **Q-QC-PU03-1:** QC chọn (a) — vì URL được đóng lúc build và trùng cổng 3311 nên QC chạy máy chủ giả **của QC** (`scripts/pu03-fake-api.mjs`, cổng 3311) thay cho `api-server.mjs` của dev và điều khiển bằng `window.__ep` trong Chrome thật. Q-QC-PU03-2/-3 vẫn **chờ BA** (QC đo: `retry_after:2` → chờ 2006 ms rồi thử; 429 SSE giữ nguyên `retry_after:5`).
- Máy chủ giả thứ hai (cổng 3512) làm "kẻ lạ" cho TC-02.

## Lỗi
### BUG-PU03-1 (thấp) — `traceId` bỏ qua header `X-Request-Id` (AC2, TC-05)
Tái hiện: máy chủ giả trả `500 {"code":"INTERNAL","message":"m"}` (không `trace_id` ở thân) kèm header `X-Request-Id: hdr-trace-1` (đã `Access-Control-Expose-Headers`); trong `/dev/data` chạy `await __ep.apiClient.get('/d')` → `error.traceId === undefined` (3 biến thể: 500, 422, `X-Trace-Id`). Mong đợi (TC-05): "thiếu thì lấy từ `X-Request-Id`". Mã: `shared/data/apiClient.ts:83` chỉ đọc `b.trace_id`.

### BUG-PU03-2 (thấp–vừa) — `event: reconnect` tạo 2 kết nối (AC14, TC-32)
Tái hiện: `/dev/data` → bấm "Gắn 3 người nghe"; máy chủ SSE giả gửi `event: ready` rồi `event: reconnect` `{"reason":"server_restart"}` (giữ kết nối), hai kết nối sau trả `ready` và giữ. Máy chủ thấy **3** yêu cầu `/api/v1/events` ở 0, 2, 3 ms và **2** lần client đóng (kỳ vọng 2 yêu cầu, 1 đóng). Thời gian nối lại 2 ms (≤ 200 ms: đạt). Hậu quả: một thoáng 2 kết nối cùng một tab, làm tới hạn `SSE_LIMIT_REACHED` (3 kết nối / người) nhanh hơn.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | máy chủ giả QC: `Authorization: Bearer <token>`, `X-Request-Id` uuid khớp `[A-Za-z0-9._:-]{1,64}`, `Accept: application/json` |
| 02 | PASS | `https://evil.example/x`, `//evil.example`, `http://localhost:3512/x` → `ApiError BAD_TARGET`; "kẻ lạ" 0 request, máy chủ giả 0 request, không header `Authorization` ra ngoài |
| 03 | PASS | `grep 'setItem(' shared/data \| grep -ci token` = 0 |
| 04 | PASS | 28 dòng (22 mã PG + 6 mã LLM/cấu hình, kể cả `GONE`, `METHOD_NOT_ALLOWED`): `code` và `status` khớp 28/28; mọi `userMessage` là tiếng Việt, không từ kỹ thuật; 26 lời khác nhau (`IDEMPOTENCY_KEY_REQUIRED`, `GONE`, `INTERNAL`, mã lạ dùng lời chung "Có lỗi xảy ra…"); `retryAfter` 30 ở 429/503 |
| 05 | **FAIL** | mã lạ → lời chung đúng; `retry_after` thân 7 > header 9 → **7**; chỉ header 4 → tự thử lại sau ≈ 4 s (hành vi AC5); **trace từ header: BUG-PU03-1** |
| 06, 57 | PASS | `playwright data-layer` dự án desktop: **25/25**, lặp 25 lần liên tiếp trên máy QC đều xanh (0 skip; mobile bỏ qua theo thiết kế); lần chạy cả hai dự án: 73 pass. Hai lần đầu đỏ lẻ tẻ (SSE/`useJob`) khi **trùng cổng 3311** với tiến trình của dev (máy chủ giả dùng chung) — không tái hiện khi chạy riêng |
| 07 | PASS | HTML 502 → `BAD_GATEWAY` (3 request, ≈ 1,2 s); 502 thân rỗng → `BAD_GATEWAY`; 200 thân rỗng → `PARSE_ERROR`; JSON hỏng → `PARSE_ERROR`; `abort()` → `ABORTED`; lời không chứa `<html`, `ECONN`, `Unexpected token`. "Đứt kết nối" không đo được bằng `destroy()` của máy chủ (Chrome tự gửi lại ổ cắm đứt); chỉ có spec dev (`page.route abort`) |
| 10 | PASS | 503, 503, 200: 3 request, cách **345 ms**, **924 ms** (≥ 240 / 720) |
| 11 | PASS | 404: 1 request; 429 `retry_after:30`: 1 request, `retryAfter=30`; 429 `retry_after:2`: 2 request cách **2006 ms** |
| 12 | PASS | POST có khoá + 503 (đã nhận HTTP): **1** request. (POST không khoá/đứt mạng: spec dev) |
| 13 | PASS | POST tự có khoá `ep-<uuid>` khớp `[A-Za-z0-9._:-]{8,128}`; `put({idempotent:true})` có khoá; `put` không cờ không khoá |
| 15 | PASS | gateway thật: 2 POST đồng thời cùng `Idempotency-Key` → `201` + `201 Idempotent-Replayed: true`, bảng `_test_items` đúng **1** dòng |
| 16 | PASS | 5 × 401 `TOKEN_EXPIRED` song song: đúng **1** `auth:expired`, 5 request (không gửi lại), `tokenStore` về `null` |
| 18 | PASS | gateway thật: `GET` có `ETag: W/"v1"`; `If-None-Match` → `304`; `PUT` (kể cả có `If-None-Match`) → `200`; `GET` lại → `200` dữ liệu mới. Tham chiếu `===` ở trình duyệt: spec dev `etag` |
| 19 | PASS | gateway thật: `PUT` version cũ → `409 VERSION_CONFLICT` `details.current` + `current_version`; `POST {}` → `422` `details[{field:name,code:required}]` |
| 23, 24 | PASS | gateway thật: 250 bản ghi, 3 trang `limit=100`, **250 id duy nhất**, 0 trùng; `limit` mặc định 30; `limit=500` → máy chủ trả `422` (nên client phải kẹp 100 — spec dev `cursor list`: request `limit=100`); `cursor=garbage` → `422 INVALID_CURSOR` |
| 27 | PASS | máy chủ giả: listener 0 nhận **13** sự kiện e1…e13, mỗi cái 1 lần, đúng thứ tự (server gửi bù lặp e4, e5); 2 kết nối; `Last-Event-ID` = `1000-5`; cách nhau 1063 ms (≥ 800); `Authorization: Bearer …`, URL không có `token=` |
| 29 | PASS | khung `10-0`, `9-0`, `10-1`, `10-1` (trùng): listener thấy `A, C` (so số học; bỏ `9-0` và trùng) |
| 30 | PASS | sau `ready` không có `id` đứng sau dữ liệu `1000-1`, nối lại mang `Last-Event-ID: 1000-1` |
| 32 | **FAIL** | nối lại 2 ms (≤ 200 ms) nhưng **BUG-PU03-2** |
| 33 | PASS (một phần) | `shutdown`: nối lại sau **2108 ms** (1000–3200). `resync`/`invalidateQueries`: spec dev |
| 36 | PASS | 3 người nghe: **1** kết nối `/events`; gỡ hết → máy chủ thấy đóng trong ≤ 500 ms |
| 37 | PASS | gateway thật, token STUDENT: hai luồng đầu `200`, luồng thứ ba `429 SSE_LIMIT_REACHED` `retry_after:5` + `Retry-After: 5` |
| 38 | PASS | gateway thật + worker: `POST _test/jobs {steps:4}` → `202 {job_id}`; thăm dò: `QUEUED:0 → RUNNING:0 → RUNNING:50 → SUCCEEDED:100` (không giảm). (Hook `useJob` trong trình duyệt: spec dev) |
| 40 | PASS (một phần) | `kind:"test.fail"` → `FAILED` `error {code:JOB_FAILED,message:"Việc chạy thất bại."}`. Hiển thị `userMessage` và rời/quay lại trang: spec dev |
| 48 | PASS | tệp tạm thiếu `consequence` → `tsc`: "Property 'consequence' is missing…"; xoá → `tsc` sạch, `git status` sạch |
| 54 | PASS | `pnpm build` thường: 0 tệp trong `.next/static` chứa `DEV_AUTH`/`Dán token`, `state-cell`, `data-part="primitive"`, `3311`; `/dev/data`, `/dev/ui` 404 |
| 55 | PASS (ghi chú) | `audit.mjs` **683 hàng FAIL 0**; `sweep` SV 42 hàng `FORBIDDEN` 0; `proto-curl.sh` **497/0**; `pnpm lint` rc=0; `ui-antipatterns.sh` rc=0, `lint-selftest` 7/7 + 19/19, `ui-allow` = 10. Lệnh `grep -rn 'fetch(' … \| grep -v shared/data/` trong AC24 ra **1** dòng vì khớp `query.refetch()` ở `shared/ui/PageState.tsx:53` (không phải `fetch(` gọi mạng; luật ESLint `ep/no-raw-fetch` sạch) → **chờ BA**: sửa lệnh Kiểm thành `\bfetch\(` |
| 56 | ghi số | bản dựng thường: 53 tệp JS, **1.421.373 byte** (gzip 440.952); `QueryClient` nằm ở 1 tệp (nạp ở layout). Next 16 không in bảng kích thước và chưa có số nền / ngân sách (US-PU-05 AC5) → chưa so được |
| 49 | PASS (spec dev) | `dev-ui › confirm` (`disabledReason`, `aria-describedby`) nằm trong 68 ca xanh của PU-02 |

## Chưa đo độc lập (chỉ dựa bộ Playwright của dev, 25/25 desktop ×25 lần)
TC-08/09 (ẩn mã kỹ thuật theo vai), 14, 17 (cổng token — thuộc US-PU-04), 20, 21 (đếm ngược), 22 (`QueryProvider` mặc định + `page.clock`), 25, 26, 28 (backoff 1/2/4/8/15 s), 31 (offline 10 s ở gateway thật), 34, 35, 39, 41–47, 50 (lint chỗ dùng `ConfirmIrreversible`), 51–53. Các ca `@real` phía trình duyệt (cần bản dựng `NEXT_PUBLIC_API_URL` trỏ gateway thật) chưa chạy.

## Việc sau
Dev: BUG-PU03-1, BUG-PU03-2. BA: lệnh Kiểm AC24 (`\bfetch\(`), Q-QC-PU03-2/-3. Sau khi sửa: QC chạy lại TC-05, TC-32 và hoàn tất các TC ở mục "Chưa đo độc lập" bằng máy chủ giả của QC.

## Vòng sửa 1 (dev `74d7fb0` BUG-PU03-1, `690337a` BUG-PU03-2; đo trên `ea9289a`, bản dựng `NEXT_PUBLIC_API_URL=http://localhost:3511` + máy chủ giả của QC)
- **BUG-PU03-1 đã sửa (TC-05 PASS):** 500 chỉ có header `X-Request-Id: hdr-trace-1` → `error.traceId = "hdr-trace-1"`; 422 → `hdr-trace-2`; có cả `trace_id` thân (`body-trace`) và header khác → **ưu tiên thân**; không có cả hai → `undefined`.
- **BUG-PU03-2 đã sửa (TC-32 PASS):** `event: reconnect` → máy chủ thấy **2** kết nối (0 ms và **58 ms**), client đóng **1** lần (trước: 3 kết nối, đóng 2).
- Hồi quy TC-27: 13 sự kiện e1…e13, duy nhất, 2 kết nối, `Last-Event-ID: 1000-5`, cách 981 ms.
- **Verdict US-PU-03: PASS** — với điều kiện đã nêu: nhóm TC "Q" ở mục "Chưa đo độc lập" chỉ có bộ Playwright của dev (xanh 25/25 khi không dùng chung cổng). TC-56 (ngân sách JS) chờ số nền PU-05; AC24 lệnh `grep 'fetch('` chờ BA (`\bfetch\(`).
