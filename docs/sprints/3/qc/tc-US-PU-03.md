# QC test case — US-PU-03 (lớp dữ liệu: `apiClient`, TanStack Query, `useSSE`, `useJob`, nháp, hoàn tác, xác nhận, mất mạng)
Nguồn: `docs/specs/FEAT-ui-foundation/US.md` US-PU-03 AC1–AC24 + `SRS.md` 3.1–3.4 (luồng), 6.1 (`apiClient`), 6.2 (28 mã máy chủ + 4 mã client → `userMessage`), 6.3 (thử lại, đếm ngược), 6.4 (`useSSE`, giao thức PG 6.8), 6.5 (hook hành vi), 5.3 (khoá lưu trữ), 8.7 (bảo mật client). Gateway thật: stack sprint 2 (`docker-compose.test.yml`, route thử `/api/v1/_test/*`, 2 bản gateway).

Tiền điều kiện chung: bản `gbuild` chạy `next start -p 3300`; trang kiểm `/dev/data` (chỉ build gate). Công cụ: **Q** = máy chủ giả **của QC** `docs/sprints/3/qc/scripts/pu03-fake-api.mjs` (QC viết lúc chạy: `Bun.serve`, kịch bản từng đường dẫn, ghi lại header / thời điểm / thứ tự request; frontend trỏ tới nó bằng `NEXT_PUBLIC_API_URL`) điều khiển bằng Eval trình duyệt thật qua `/dev/data`; **R** = gateway thật (stack sprint 2, token `gateway token --role ADMIN|STUDENT --ttl 30m` dán ở cổng dev); **P** = Playwright của dev (`$PW data-layer.spec.ts -g …`, `rc=0`); **S** = shell; **T** = tay. QC **không** chấp nhận chỉ `P` xanh: mỗi AC có ≥ 1 TC `Q`/`R` do QC tự đo. Thiếu `data-layer.spec.ts` / `/dev/data` → TC FAIL "KHÔNG KIỂM ĐƯỢC".

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PU03-01 | AC1 | **Q** | `/dev/data` gọi `apiClient.get('/ping')`; đọc header ở máy chủ giả | URL = `${NEXT_PUBLIC_API_URL}/api/v1/ping`; `Authorization: Bearer <token>` (có token); `X-Request-Id` khớp `[A-Za-z0-9._:-]{1,64}`; `Accept: application/json`; cookie gửi kèm (`credentials: include`) |
| TC-PU03-02 | AC1 (an toàn) | **Q** | `apiClient.get('https://evil.example/x')` (và `//evil.example`, `http://localhost:9/x` origin khác) với token đã đặt | Bị từ chối **trước khi gửi**: `TypeError` hoặc `ApiError code=BAD_TARGET`; máy chủ giả của "evil" (cổng khác) **0 request**, 0 header `Authorization` |
| TC-PU03-03 | AC1 | – | **S** `grep -rn 'setItem(' frontend/src/shared/data \| grep -ci token` | `0` |
| TC-PU03-04 | AC2 | **Q** + SRS 6.2 | Vòng 28 dòng: máy chủ giả trả `{status, thân {code,message,trace_id,retry_after?}}` cho từng mã SRS 6.2 (22 mã PG + `OVERLOADED`, `LLM_NOT_CONFIGURED`, `LLM_UNAVAILABLE`, `PROVIDER_IN_USE`, `MODEL_DIMS_MISMATCH`, `ROUTE_INVALID`) | Mỗi dòng: `error.code`, `error.status`, `error.retryAfter`, `error.traceId` khớp; `userMessage` **khác rỗng, tiếng Việt, đúng lời bảng SRS 6.2**; số dòng đã chạy = 28 |
| TC-PU03-05 | AC2 | **Q** | Mã lạ `{code:"WHATEVER"}`; `retry_after` ở thân **và** header khác nhau (thân 7, header 9); `trace_id` chỉ có ở header `X-Request-Id` | Mã lạ → "Có lỗi xảy ra. Dữ liệu của bạn vẫn an toàn. Thử lại sau ít phút."; `retryAfter` = 7 (thân ưu tiên); chỉ header → lấy header; `traceId` lấy từ thân, thiếu thì từ `X-Request-Id` |
| TC-PU03-06 | AC2 | – | **P** `$PW data-layer.spec.ts -g 'error mapping'; echo rc=$?` | `rc=0` |
| TC-PU03-07 | AC3 | **Q** | Máy chủ giả trả `<html>502 Bad Gateway</html>`; thân rỗng; JSON hỏng `{"code":`; `res.destroy()` giữa chừng; `AbortController.abort()` | `code` lần lượt `BAD_GATEWAY`, `PARSE_ERROR` (hoặc `BAD_GATEWAY` cho rỗng 5xx), `PARSE_ERROR`, `NETWORK`, `ABORTED`; `userMessage` không chứa `<html`, `ECONN`, `Unexpected token`; nói chuyện gì xảy ra + dữ liệu an toàn ("Chữ bạn đã nhập vẫn được giữ") |
| TC-PU03-08 | AC4 | `/dev/data?as=student`, **Q** | Gây `VERSION_CONFLICT`, `INTERNAL` với `trace_id` 32 hex rồi mở lỗi | `document.body.innerText` **không** chứa `VERSION_CONFLICT`, `INTERNAL`, chuỗi `trace_id`; chữ hiển thị là tiếng Việt |
| TC-PU03-09 | AC4 | `?as=admin` (và `teacher`, `ta`) | Như 08 | Có mục "Chi tiết kỹ thuật" **gấp sẵn** (`<details>` không `open`) chứa `trace_id`; TA / GV / Admin đều có, SV không |
| TC-PU03-10 | AC5 | **Q** | GET → 503, 503, 200; ghi thời điểm | 3 request; cách nhau ≥ 240 ms rồi ≥ 720 ms (300 ms / 900 ms ±20 %); kết quả cuối thành công |
| TC-PU03-11 | AC5 | **Q** | GET → 404; GET → 429 `retry_after:30`; GET → 429 `retry_after:2`; GET → NETWORK ×3 | 404: 1 request; 429/30: 1 request, lỗi mang `retryAfter=30`; 429/2: tôn trọng chờ ≈ 2 s rồi thử (≤ 2 lần thử lại); NETWORK ×3: đúng 3 request rồi `NETWORK` |
| TC-PU03-12 | AC5 | **Q** | POST không `Idempotency-Key` → NETWORK; POST có khoá → NETWORK rồi 200; POST có khoá → 503 | Không khoá: 1 request; có khoá + NETWORK: 2 request **cùng** khoá; có khoá + 503 (đã nhận HTTP): **1** request, không thử lại |
| TC-PU03-13 | AC6 | **Q** | `apiClient.post` không truyền khoá; `put({idempotent:true})`; `put` không cờ | POST luôn có khoá khớp `[A-Za-z0-9._:-]{8,128}`; PUT có khoá khi `idempotent:true`; PUT không cờ: không khoá |
| TC-PU03-14 | AC6 | **Q** | `useIdempotentMutation`: gửi lỗi mạng → `Gửi lại`; rồi thành công; rồi gửi mới | Lần gửi lại dùng **cùng** khoá (2 request cùng khoá); sau thành công, gửi mới dùng khoá **khác**; phản hồi `Idempotent-Replayed: true` → `result.replayed=true` và **không** có chữ thông báo nào trên màn |
| TC-PU03-15 | AC6 | **R** | Hai tab cùng gửi `POST /api/v1/_test/items` cùng `Idempotency-Key` (truyền tay) | `select count(*) from _test_items where name='x'` = `1`; một phản hồi `Idempotent-Replayed: true` |
| TC-PU03-16 | AC7 | **Q** | 5 request song song cùng nhận 401 `TOKEN_EXPIRED`; rồi 5 request nữa sau 11 s | Lần đầu đúng **1** sự kiện `auth:expired`; không request nào bị gửi lại; `tokenStore` đã xoá token; sau > 10 s mới phát lần hai; thử với `TOKEN_INVALID`, `UNAUTHENTICATED` |
| TC-PU03-17 | AC7 | `DEV_AUTH=1` | Sau `auth:expired` | Shell hiện cổng "Dán token quản trị để tiếp tục" (xem US-PU-04 AC9) |
| TC-PU03-18 | AC8 | **R** | `GET /api/v1/_test/items/{id}` hai lần; rồi `PUT` đổi, GET lại | Lần hai có `If-None-Match` = ETag lần một, phản hồi `304`, `data` giữ **tham chiếu** (`===`) và không nhấp nháy; sau `PUT` → `200` dữ liệu mới; `POST/PUT` **không** gửi `If-None-Match` |
| TC-PU03-19 | AC9 | **R** | `PUT /api/v1/_test/items/{id}` `version` cũ; `POST /api/v1/_test/items {}` | `error.conflict.currentVersion` = giá trị hiện tại, `error.conflict.current` có nội dung; `fieldErrors(error)` có khoá `name` |
| TC-PU03-20 | AC9 | **Q** | Màn kiểm dùng `Field`: gửi 422 `details:[{field:"name",message:"…"}]` | Ô `name` `aria-invalid="true"`, `aria-describedby` trỏ lời lỗi dưới ô; **nội dung các ô khác không mất** |
| TC-PU03-21 | AC10 | **Q** | 503 `retry_after:3` rồi 429 không `retry_after` | Chữ "Hệ thống đang bận. Bạn có thể thử lại sau 3 giây." → "2 giây" → "1 giây"; `role=status`; vùng live cập nhật ≤ 1 lần / giây (đếm mutation); `Thử lại` `disabled` kèm lý do (`aria-describedby`) tới hết đếm rồi bấm được (≈ 3,1 s); không `retry_after`: bấm được ngay |
| TC-PU03-22 | AC11 | **Q** | Đọc mặc định của `QueryProvider` qua `/dev/data` (hoặc hành vi): làm stale 30 s bằng `page.clock` | `staleTime` 30 s (không refetch trong 29 s, refetch sau 31 s khi focus lại), `gcTime` 5 phút, `retry:false`, `refetchOnWindowFocus` bật, `networkMode` offlineFirst (chạy được khi `setOffline` với dữ liệu cache) |
| TC-PU03-23 | AC11 | **R** | `useCursorList` trên `GET /api/v1/_test/items` (≥ 250 bản ghi) đi hết trang; `limit=500` | Đúng 250 id duy nhất, không trùng; `limit` mặc định 30; `limit=500` bị kẹp **100** (ghi lại ở request); cursor truyền nguyên chuỗi |
| TC-PU03-24 | AC11 | **Q** | Thân `INVALID_CURSOR` ở trang 2 | Quay về trang đầu **đúng một lần**, rồi hiện lỗi (không lặp vô hạn) |
| TC-PU03-25 | AC12 | **Q** | Máy chủ SSE giả phát `retry: 3000`, `event: ready`, một khung `data:` hai dòng, `: hb` | Handler nhận đúng 1 sự kiện, `data` nối bằng `\n`; request có `Authorization`, **không** `token=` trên URL; không dùng `EventSource` (không request kiểu `eventsource` ở Network); `: hb` bị bỏ qua |
| TC-PU03-26 | AC12 | – | Đọc `useSSEStatus` qua các pha | Chuyển `connecting → open → reconnecting → closed` đúng thứ tự |
| TC-PU03-27 | AC13 | **Q** | Phát e1…e5, đóng ngang, phát e11…e13 trong lúc đứt (máy chủ giả giữ bộ đệm, trả theo `Last-Event-ID`) | Handler nhận đúng e1…e13, mỗi cái 1 lần, thứ tự; `Last-Event-ID` ở lần nối = id của e5; khoảng cách hai lần nối đầu ≥ 800 ms |
| TC-PU03-28 | AC13 | **Q** | Chuỗi 5 lần đứt liên tiếp ghi khoảng cách; giữ kết nối ≥ 30 s rồi đứt | Trễ ≈ 1 s, 2 s, 4 s, 8 s, 15 s (±20 %); sau ≥ 30 s ổn định trễ về 1 s |
| TC-PU03-29 | AC13 | **Q** | Gửi lại khung có `id` ≤ id đã xử lý; id so số học (`9-0` sau `10-0`) | Bỏ qua trùng; so `(ms, seq)` số học, không so chuỗi |
| TC-PU03-30 | AC13 | **Q** | Sau một sự kiện điều khiển không có `id` (`ready`) rồi đứt | `Last-Event-ID` vẫn là id **dữ liệu** cuối (không bị `ready` ghi đè) |
| TC-PU03-31 | AC13 | **R** | `context.setOffline(true)` 10 s trong lúc `POST /api/v1/_test/events` phát n=1..30 từ ngoài | Bật mạng lại: nối được, nhận đủ 1…30, không mất / không trùng |
| TC-PU03-32 | AC14 | **Q** | `event: reconnect` (3 lý do) | Nối lại **≤ 200 ms** với `Last-Event-ID` (không backoff); `token_expired`: phát `auth:expired` trước và **không** nối tới khi có token mới |
| TC-PU03-33 | AC14 | **Q** | `event: shutdown`; `event: resync` | `shutdown`: nối lại sau 1–3,2 s; `resync`: `onResync` gọi 1 lần, mặc định `invalidateQueries` mọi query đăng ký SSE, rồi tiếp tục nhận; `ready` → `open` |
| TC-PU03-34 | AC15 | **Q** | Lần lượt 401; 429 `SSE_LIMIT_REACHED` `retry_after:5`; 503; treo | 401: dừng + `auth:expired`, **không** vòng lặp; 429: đợi ≥ 5 s, tối đa 3 lần rồi `degraded` (chữ "Bạn đang mở nhiều cửa sổ; cập nhật tự động tạm dừng"); 503: backoff như AC13; treo: sau **> 40 s** (đẩy `page.clock` 41 s) huỷ và nối lại |
| TC-PU03-35 | AC15 | – | Khi `degraded`: hook phụ thuộc (ví dụ `useJob`) | Rơi về thăm dò (xem TC-PU03-39) |
| TC-PU03-36 | AC16 | **Q** | Mount 3 component dùng `useSSE`; sau đó unmount hết | Network: đúng **1** request `/api/v1/events`; unmount: kết nối đóng (máy chủ giả thấy `close`) ≤ 200 ms; không callback sau đó, không timer rò (đếm `setTimeout` còn lại qua spy) |
| TC-PU03-37 | AC16 | **R** | Mở 3 tab cùng người dùng (token Student) | Tab 3 nhận `429 SSE_LIMIT_REACHED` → xử lý như AC15 (`degraded`); 2 tab đầu vẫn nhận sự kiện |
| TC-PU03-38 | AC17 | **R** | `POST /api/v1/_test/jobs {"steps":4}` → `useJob(id)` | Mount đầu gọi `GET /jobs/{id}` **trước** khi nghe SSE; chuỗi `progress` quan sát được không giảm, kết thúc `100`, `status="SUCCEEDED"` |
| TC-PU03-39 | AC17 | **R** + ép `degraded` | Như 38 | Vẫn kết thúc nhờ thăm dò: ≥ 2 request `GET /jobs/{id}` cách nhau ≈ 2 s; dừng khi `SUCCEEDED` (không thăm dò tiếp) |
| TC-PU03-40 | AC17 | **R** | `kind:"test.fail"`; rời trang giữa chừng rồi quay lại | `FAILED` hiện `error` bằng `userMessage`; rời trang → dừng thăm dò, quay lại tiếp tục từ trạng thái hiện tại; `progress` không giảm trên UI kể cả khi SSE đến trễ |
| TC-PU03-41 | AC18 | `/dev/data` nháp | Gõ "Xin chào thầy"; chờ 2,5 s; đóng tab đột ngột (`close({runBeforeUnload:false})`); mở lại cùng ngữ cảnh | Ô có "Xin chào thầy"; `localStorage` có khoá `ep:draft:<userId\|anon>:<key>` dạng `{v:1,text,savedAt}` |
| TC-PU03-42 | AC18 | – | Gõ rồi đóng sau 1 s; gõ rồi `visibilitychange→hidden`; gỡ gắn kết | `pagehide`/`hidden`/unmount ghi **ngay** (không đợi 2 s); sau 2,5 s chắc chắn có |
| TC-PU03-43 | AC18 | – | `status` qua `idle → saving → saved`; `clear()` sau gửi thành công | Đúng chuỗi trạng thái; sau `clear` khoá bị xoá |
| TC-PU03-44 | AC18 | – | Nhét bản nháp `savedAt` > 30 ngày rồi khởi động; nháp > 100 KB; `localStorage.setItem` ném `QuotaExceededError` (giả) | Bản > 30 ngày bị dọn; > 100 KB bị từ chối **có thông báo**; đầy bộ nhớ: `status="error"` và ô **vẫn giữ chữ** |
| TC-PU03-45 | AC18 | dev build | `useAutosaveDraft("llm.apiKey")`, `"password"`, `"secret"`, `"mykey"` | Mỗi lần ném `TypeError` ở dev (từ chối khoá bí mật) |
| TC-PU03-46 | AC19 | **Q** | `useUndoableAction`: bấm thao tác (ví dụ ẩn mục) | UI đổi ≤ 100 ms (`expect.poll`); dòng `Đã … · Hoàn tác` (`[data-part=undo-line]`) **trong cùng vùng**, không toast nổi, không chữ "Thành công" ở `role=status` toàn cục; tự biến sau 5 s (`page.clock` 5,2 s) |
| TC-PU03-47 | AC19 | **Q** | `Hoàn tác` trong 5 s; ghi lên gateway lỗi 500; thao tác mới ghi đè mục rồi bấm `Hoàn tác` dòng cũ | `Hoàn tác` gửi **thao tác bù ngay** (đóng tab sau đó không mất: máy chủ giả đã nhận), UI về trạng thái trước; lỗi 500 → UI về cũ + `role=alert` + `Thử lại`; `Hoàn tác` bị bỏ qua nếu đã bị thay đổi mới hơn ghi đè (khoá theo mục) |
| TC-PU03-48 | AC20 | `tsc` | **S** tệp tạm gọi `<ConfirmIrreversible open onClose onConfirm title="x" confirmLabel="Xoá" />` (thiếu `consequence`); `cd frontend && npx tsc --noEmit` | Lỗi biên dịch nêu `consequence`; xoá tệp tạm → `tsc` sạch, `git status` rỗng |
| TC-PU03-49 | AC20 | `/dev/ui` | `disabledReason`: nút xác nhận; trong `loading` bấm `Esc` / nền; đọc lời hậu quả | Nút `disabled` + `aria-describedby` nêu lý do; `loading`: không đóng được; hậu quả nêu **con số** (ví dụ "Công bố điểm cho 28 sinh viên, gửi 28 mail"); `confirmLabel` là động từ |
| TC-PU03-50 | AC20 | **S** | `bash scripts/ui-antipatterns.sh` + gieo `<ConfirmIrreversible>` ở một chỗ ngoài danh sách việc cần bảo vệ | Phép lint bắt chỗ ngoài danh sách; danh sách hiện có sạch |
| TC-PU03-51 | AC21 | **Q** | `context.setOffline(true)`; rồi `setOffline(false)` + 1 request thành công; ca hai: online nhưng 2 request liên tiếp `NETWORK` | Banner "Mất kết nối mạng. Chữ bạn đã nhập vẫn được giữ; hệ thống sẽ gửi lại khi có mạng." ≤ 1 s, `role=status` `aria-live=polite`; đẩy nội dung xuống (CLS ≤ 0,02), không che; biến ≤ 2 s sau khi có mạng + request thành công; ô nhập gõ được khi offline; ca hai cũng hiện |
| TC-PU03-52 | AC22 | **Q** | `<PageState>`: pending; 503; thành công + `isEmpty`; `?state=error` ở `MOCK_SCREENS=1` và `0` | pending: `[aria-busy=true]`; lỗi: `role=alert` + `Thử lại` gọi **đúng 1** request `refetch` (không tải lại trang: `window.__nav` không đổi); empty: trạng thái rỗng; `MOCK_SCREENS=0`: `?state=error` không có tác dụng |
| TC-PU03-53 | AC23 | **Q** | Dán token ở cổng dev rồi dùng; nghe `console` | `localStorage`, `sessionStorage` không chứa token; `location.href` không chứa token; `console` không in token; tải lại → token mất (chỉ trong bộ nhớ) |
| TC-PU03-54 | AC23 | **S** | `pbuild && grep -rl 'DEV_AUTH\|Dán token' frontend/.next/static \| wc -l` | `0` (bản không bật cờ không chứa mã cổng token) |
| TC-PU03-55 | AC24 | bản gate | **A** `audit.mjs` (4 vai + spec, như baseline); `sweep.mjs only:'student'`; `proto-curl.sh all`; `grep -rn 'fetch(' frontend/src \| grep -v shared/data/ \| wc -l`; `pnpm lint` | FAIL 0, hàng PASS ≥ nền (680); `FORBIDDEN`=0; ≥ 497 PASS; `fetch(` ngoài `shared/data` = 0; lint rc=0 |
| TC-PU03-56 | AC24 | – | **S** đo kích thước JS mỗi route từ log `next build` so với `audit-baseline` (ghi tay số nền khi build lúc chạy) và ngân sách US-PU-05 AC5 | Thêm TanStack Query **không** làm vượt ngân sách; nạp ở layout, không nạp lại mỗi route |
| TC-PU03-57 | tổng | – | **P** `$PW data-layer.spec.ts; echo rc=$?` (toàn bộ) | `rc=0`, 0 skip (trừ test `@real` ghi rõ lý do) |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Token lộ sang origin khác | 02 |
| Mã lỗi lạ; thân HTML; JSON hỏng; mất kết nối; huỷ | 05, 07 |
| Sinh viên thấy `trace_id` / mã trần | 08, 09 |
| POST tự thử lại không khoá | 12 |
| Bão `auth:expired` | 16 |
| Mất / trùng sự kiện SSE khi đứt | 27–31 |
| SSE im lặng > 40 s; bị giới hạn 2 kết nối / người | 34, 37 |
| Đóng tab đột ngột mất chữ; bộ nhớ đầy; khoá bí mật | 41–45 |
| Hoàn tác khi gateway lỗi | 47 |
| Mất mạng giữa lúc gõ | 51 |
| Token trong storage / URL / console | 03, 53, 54 |

## Câu hỏi cho BA / PM
- **Q-QC-PU03-1** — Cổng dán token dev dùng `NEXT_PUBLIC_DEV_AUTH=1`; QC cần cách đặt `NEXT_PUBLIC_API_URL` trỏ tới máy chủ giả của QC mà không sửa mã: (a) biến khi `next build` (đúng — buộc build lại); (b) thêm `?api=` ở `/dev/data`. QC giả định (a) — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Đúng (a): `NEXT_PUBLIC_API_URL` là biến lúc **build** (build lại khi đổi); không có tham số `?api=` (v1.1, "Quy ước kiểm chung"). Cho phép sửa đổi máy chủ giả bằng cổng của chính nó (build riêng cho QC).
- **Q-QC-PU03-2** — AC5 "tôn trọng `retryAfter` nếu ≤ 5 s": TC-PU03-11 giả định thử lại sau đúng `retryAfter` thay cho 300/900 ms. Đúng? — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Đúng, và đã làm rõ (v1.1, AC5 + SRS 6.3): khi `retry_after` ≤ 5 s, lần thử lại chờ **đúng** `retry_after` (thay cho 300 / 900 ms, không cộng, không jitter, sai số 0…+250 ms). Thêm ca kiểm 503 `retry_after:2`.
- **Q-QC-PU03-3** — AC15 `429 SSE_LIMIT_REACHED` "đợi `retry_after` (≥ 5 s)": gateway đang trả `retry_after:5`; TC dùng đúng 5. — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Xác nhận: dùng đúng 5 s (SRS PG 6.1: `SSE_LIMIT_REACHED` kèm `retry_after: 5`; PU AC15 "≥ 5 s").
- **TC-PU03-55 (báo cáo QC `report-US-PU-03.md`: lệnh `grep 'fetch('` ra 1 dòng — `query.refetch()`)** — QC đề nghị sửa lệnh Kiểm thành `\bfetch\(`.
  - **Trả lời (BA, 2026-10-03):** Đồng ý. AC24 (v1.2) ghi lệnh `grep -rnE '\bfetch\(' frontend/src | grep -v 'shared/data/' | wc -l` → `0`; `\b` không khớp `refetch(`. QC sửa lệnh ở TC-PU03-55. **Q-QC-PU03-2 / -3** đã được trả lời ngay dưới từng câu từ v1.1 (commit `4707078`): `retry_after` ≤ 5 s → chờ **đúng** `retry_after` (QC đo 2006 ms cho `retry_after:2` là đạt, trong 0…+250 ms); `SSE_LIMIT_REACHED` chờ `retry_after` = 5 s.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md (FEAT-ui-foundation, APPROVED 2026-10-03).

Tổng: 57 TC.
