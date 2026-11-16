# Dev · sprint 11 đợt 3: LCP 8 route (US-P10-08) + Nợ L6 (US-P10-09)

Nhánh `sprint/11-p10-eval`. Không đụng `observability` / `insights`.

## Cách đo
`benchmarks/lighthouse/run.sh` cần stack Docker `ep-*` (RAM đang dành cho QC), nên đo bằng bộ tương đương nhẹ hơn: gateway Go chạy thẳng (`go build`) + Postgres 18 / Redis 8 / MinIO trong 3 container riêng, dữ liệu `scripts/seed-t1.mjs --no-vectors` (1.000 SV · 20 lớp) + 60 ticket mở cho `t1gv1` (câu hỏi dài như đo 9.5), bản **build production** `output: standalone` của frontend, một proxy HTTPS cùng origin (như Caddy: `/api/*` → gateway, còn lại → Next; không nén JSON API, Next tự nén HTML/JS). `lhci collect` với `lighthouserc.ep.json`, **throttle devtools**, 3 lượt, trung vị; MacBook Apple Silicon, không worker, không pgbouncer, không Caddy. **Trạng thái người dùng quay lại** (P-11-5): `auth-ep.cjs` gọi `POST /me/tracking-notice/ack` sau khi đăng nhập; `LH_FIRST_VISIT=1` bỏ bước đó và đăng nhập bằng `t1sv0002` (chưa ack) cho số «lần đầu».
Chưa đo trên stack T1 Docker đầy đủ: số có thể lệch vài trăm ms (Caddy, pgbouncer); phần tử LCP và thứ tự mắt xích thì giống.

## Kết quả (LCP ms, trung vị 3 lượt)
| Route | Trước (HEAD) | Sau U1 + U2 + U4 | Sau đổi chữ (a) | Phần tử LCP trước → sau |
| --- | --- | --- | --- | --- |
| `/` | 1.949 | 1.927 | **1.943** | chữ khung (không đổi) |
| `/chat` | 1.529 | 1.524 | **1.526** | chữ PreShell |
| `/threads` | 1.519 | 1.525 | **1.514** | PreShell (đã ack). Lần đầu: lượt 1 = 5.155 (hộp thoại thời gian học), lượt 2–3 = 1.519–1.544 |
| `/inbox` | 1.520 (chưa có ticket) | 1.517 | **1.513** (60 ticket, 2 dòng ở 412 px) | PreShell |
| `/gradebook` | 5.700 | 5.524 | **1.537** | câu «Có điểm cộng ghi tay không được tính theo công thức» → PreShell |
| `/me` | 6.639 | 5.703 (có đổi headline) | **1.475** | `p.headline` → ghi chú cố định → PreShell |
| `/settings/llm` | 4.737 | 4.504 | **1.468** | «Đang dùng cấu hình mặc định của máy chủ…» → PreShell |
| `/assignments/[id]` | 5.184 (FCP 3.958) | 4.841 | **4.834** (FCP 831; lần đầu 4.843) | `p.total` → đoạn đề bài (dữ liệu) |

Ngưỡng `lighthouserc*.json` giữ 2.500. TBT (devtools) 17–31 ms mọi route.

## Trung thực về LCP: phần lớn mức giảm là do ĐỔI phần tử lớn nhất, không phải dữ liệu hiện sớm hơn
`/gradebook`, `/me`, `/settings/llm` đạt vì chữ dài sau phiên (nhỏ hơn / bằng `h1` thì không còn là ứng viên LCP) bị bỏ theo luật chữ UI: LCP chuyển sang chữ PreShell (≈ 1,5 s). **Dữ liệu chính vẫn hiện muộn.** Đo thêm thời điểm dữ liệu chính hiện (puppeteer, cùng throttle devtools: 562,5 ms / 1,474 Mbps / CPU 4×, tắt cache, 5 lượt mỗi ô, trung vị, bản HEAD trước đợt 3 `2f033ba1` đối với bản sau, cùng stack): thời điểm = lúc phần tử đánh dấu đầu tiên xuất hiện trong DOM (`input[data-cell]` của lưới sổ điểm; `[data-part=grade-sentence]` của `/me`; tiêu đề phần «Mô hình theo tác vụ» của `/settings/llm`, chỉ dựng sau khi providers + routes về).
| Route | LCP trước → sau (ms) | Dữ liệu chính hiện trước → sau (ms) | Giảm thật |
| --- | --- | --- | --- |
| `/gradebook` | 6.156 → 2.072 | 6.791 → 6.055 | −0,74 s (U1 + U2 + U4) |
| `/me` | 7.032 → 1.568 | 6.996 → 5.274 | −1,72 s (U1 + U2 + U4) |
| `/settings/llm` | 5.436 → 4.700 (dữ liệu) | 5.384 → 4.700 | −0,68 s (U1 + U2 + U4) |
(LCP của cột này đo bằng puppeteer nên cao hơn Lighthouse vài trăm ms vì tính cả logo; kết luận cùng chiều với bảng Lighthouse ở trên.)
Hệ quả: người dùng thấy trang nhanh hơn 0,7–1,7 s thật, nhưng thấy DỮ LIỆU sau ≈ 4,7–6 s dưới throttle này. Con số «LCP ≤ 2,5 s» của ba route này nói về chữ khung, không phải về dữ liệu. Muốn dữ liệu hiện < 2,5 s chỉ có SSR (b).

## U1 đổi hành vi phiên phía client (QC kiểm)
Trước: `AuthProvider` gọi `POST /auth/refresh` trong `useEffect` sau khi React hydrate (≈ 3,5 s dưới throttle). Sau: `<script>` nội tuyến trong `<head>` (`shared/data/earlyRefresh.ts`, gắn ở `app/layout.tsx`) gọi ngay khi HTML được phân tích, đặt promise lên `window.__epRefresh`; `authSession.doRefresh` lấy đúng MỘT lần rồi `delete`.
- Điều kiện gọi giữ nguyên: `/` mà máy chưa từng đăng nhập (`ep_returning` ≠ `1`) thì KHÔNG gọi; mọi route khác gọi (kể cả `/login`).
- Cùng khoá `navigator.locks` `ep-refresh` như trước: hai tab tải cùng lúc không bao giờ có hai làm mới chạy chồng (cookie xoay vòng: token cũ dùng lại ⇒ gateway thu hồi phiên `REFRESH_REUSE`). Trình duyệt không có `navigator.locks`: chạy thẳng (như nhánh dự phòng cũ).
- Hết thời gian 15 s như cũ (`AbortSignal.timeout`). Lỗi mạng ở lời gọi sớm → `NETWORK` như trước (không đăng xuất). Header `X-Request-Id` không còn gửi ở lời gọi sớm (gateway không đòi).
- Token vẫn chỉ ở bộ nhớ; không đổi cookie, không đổi gateway, không đổi mô hình auth. CSP `script-src 'unsafe-inline'` (10.8) cho phép script nội tuyến.
- Rủi ro đã biết: nếu `__epRefresh` bị tạo mà AuthProvider không bao giờ gọi `refreshSession` (không có đường như vậy hôm nay: cùng điều kiện), cookie vẫn xoay vòng một lần — không hại.
**Test e2e cho QC** (`frontend/e2e/early-refresh.spec.ts`, desktop): (1) một lần tải = đúng một `POST /auth/refresh`, `window.__epRefresh` bị xoá; (2) `/` khách = 0 lần; (3) **hai tab tải cùng lúc** `/me`: `maxActive = 1` (không chạy chồng) và mỗi tab đúng một lần. QC nên thêm: đăng nhập thật ở hai tab (stack Docker, không mock) tải lại cùng lúc ×20, 0 lần `SESSION_REVOKED`.

## Nguyên nhân (waterfall `/me`, throttle devtools)
HTML 0–0,9 s → làm mới phiên 0,7–1,9 s (sau U1; trước đó chỉ bắt đầu khi JS chạy, ≈ 3,5 s) → các chunk JS tải thành nhiều đợt, tới 3,5 s → `/me/courses` 3,5–4,8 s → 5 truy vấn dữ liệu 5,1–5,7 s → vẽ. FCP 0,84 s nhờ PreShell. **Chuỗi này không rút được dưới 2,5 s chỉ bằng sửa phía client** (JS xong ≈ 3,5 s dưới throttle), đúng như TL-2: cách duy nhất là để chữ lớn nhất sau phiên không vượt `h1` của PreShell (đường (a)). U1–U4 chỉ cắt 0,1–0,35 s mỗi route (tăng tốc thật, không đủ để đạt).
Phát hiện: LCP là **diện tích hộp chữ** (một đoạn xuống dòng ở 412 px chiếm cả bề rộng); mọi đoạn chữ sau phiên phải nhỏ hơn `h1` (≈ 9.000 px² với «Kết quả của tôi», ≈ 4.800 px² với «Sổ điểm»).

## Đã làm
- **U1** `<script>` trong `<head>` (`app/layout.tsx`) bắt đầu `POST /auth/refresh` khi phân tích HTML, **cùng khoá `ep-refresh`** giữa các tab; `authSession.doRefresh` dùng lại đúng MỘT lần rồi xoá khỏi `window` (cookie xoay vòng: dùng lại token là bị thu hồi phiên). Cùng điều kiện với AuthProvider (`/` chưa từng đăng nhập thì không gọi). Test `e2e/early-refresh.spec.ts`: một lần tải = đúng một POST, `window.__epRefresh` bị xoá, `/` khách = 0 lần, hai tab tải cùng lúc không bao giờ có hai làm mới chạy chồng (`maxActive = 1`).
- **U2** nạp `Real*` ngay khi module màn được nạp (`MeScreen`, `InboxScreen`, `ThreadsScreen`).
- **U4** `prefetch={false}` cho link thanh bên (nạp khi rê chuột / focus), dòng thread và dòng `ActionList`.
- **U3 không làm:** các route đạt là nhờ PreShell, không nhờ rút chuỗi dữ liệu; route còn trượt (`/assignments/[id]`) trượt vì chữ dữ liệu, không vì chuỗi. U3 thêm đường 403/404 và rủi ro lớp nhớ sai mà không đổi kết quả đo.
- **(a)** `/me`: headline → dòng `nhãn + giá trị` («Điểm quá trình 8,75 / 10», «Điểm học phần 7,0 (B)»), cỡ chữ `item`; ghi chú chính thức một dòng cỡ `meta`. `/gradebook`: «Điểm cộng ghi tay» / «Điểm trừ ghi tay» thay câu dài. `/settings/llm`: «Chưa có nhà cung cấp» + nút `Thêm nhà cung cấp`. `/assignments/[id]`: PreShell theo tiền tố đường dẫn («Bài tập»), `p.total` cỡ `section`. Đã vá `FEAT-gradebook` US AC3 / AC13 + SRS, `FEAT-uiux-91` US, `FEAT-llm-gateway` SRS.
- Bộ đo: `auth-ep.cjs` (ack + `LH_FIRST_VISIT`), `summarize.mjs` thêm `lcp_element`; `e2e/lcp.spec.ts` dùng dữ liệu xấu nhất của T1 (không nhà cung cấp, cả hai cảnh báo ghi tay) và thêm ca `/assignments/<id>` (ứng viên đầu là chữ PreShell).

## Route còn trượt (AC6, không tính FAIL story)
`/assignments/[id]`: LCP **4.834 ms** quay lại / **4.843 ms** lần đầu. Phần tử `<p>` đoạn đề bài (354 × 24 px, ≈ 8.500 px²) lớn hơn `h1` «Bài tập» (≈ 4.200 px²). Là dữ liệu của bài tập nên (a) không với tới. Cách duy nhất: (b) SSR dữ liệu đầu trang + làm mới phiên phía máy chủ (đụng auth) — chờ chủ dự án. `docs/PROGRESS.md` mục Nợ đã cập nhật.

## P-11-20 (`/settings/llm` TBT)
Cấu hình CI (`lighthouserc.json`, API giả, mô phỏng, 5 lượt) chạy cục bộ: TBT 18 · 21 · 27 · 34 · 49 ms, **p80 = 34 ms** (≤ 160). Trên T1 thật (mô phỏng, 5 lượt): 13 · 13 · 14 · 16 · 21, p80 = 16 ms. Không tái hiện được đuôi dài 155–230 ms của runner GitHub trên máy này (chỉ có một tác vụ dài 77 ms ở chunk khung `0307…js`, giống mọi route); không có đoạn mã riêng của `/settings/llm` để cắt. Câu hỏi còn lại là nhiễu runner: CI chạy 5 lượt mô phỏng, ngưỡng 200 giữ nguyên.

## L6 / P-10-10 (US-P10-09)
`practice.StartIn.Count` thành `*int`: thiếu ⇒ mặc định theo chế độ; có mặt (kể cả `0`) phải trong 5–50, ngoài thì `422 VALIDATION_FAILED`. Test: `TestStart…` thêm `0`, `-1`; hợp đồng `scenarios_practice_test` thêm `count:0` → 422.

## Môi trường (ghi để người sau khỏi mất công)
Có một tiến trình ngoài (nghi `pkill -f next` của phiên khác, `[SUY LUẬN]`) làm chết `next start` / server standalone giữa lượt đo (chunk 502 → màn «Trang này gặp sự cố» ở một lượt `/gradebook`, đã xác minh bằng log 502). Bản đo cuối chạy server trong vòng lặp tự khởi động lại; mọi lượt trong bảng là lượt không có 502.

## 2026-11-16 dev2 — dừng theo lệnh chủ dự án
Xong: DEF-R5-03, R5-04, R8-A2 (mỗi cái 1 commit, test đỏ-trước). Còn (chưa bắt đầu): R5-06 nhãn QUEUED (CodeResults.tsx:91), R5-08 dòng tóm tắt + gập câu (StudentResult.tsx), R5-09 Staff results lệch AC10 (3 điểm; điểm 1 cần BA/PM quyết), R3-01 chữ EMBEDDING (settings/llm/labels.ts:13, dùng nhãn «Đánh chỉ mục» của observability/api.ts:44).

## 2026-11-16 dev — dừng theo lệnh chủ dự án (5 lỗi qc2)
- Xong: DEF-R8-A1 (6c75bd54), R5-11 (3f748ec2), R5-10 (dd160463: `tests_stale` ở `GET …/results/{aid}`).
- Còn: R5-07 (tín hiệu liêm chính mất khi đóng tab: `useIntegrityEvents.ts` flush không `keepalive`/`sendBeacon` lúc `pagehide`; xem lại số 9,4 s). P103-08 bị qc2 rút (lỗi kịch bản QC).
