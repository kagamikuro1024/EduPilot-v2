# Câu hỏi Tech Lead — sprint 5

## TL-2 (US-PU-06 — LCP ≤ 2,5 s không đạt được bằng khung vẽ máy chủ; cần chọn hướng)
**Đã làm (trên `sprint/5-pe`, chưa commit):** (1) `AuthGate` vẽ **khung trước phiên từ máy chủ** (`shared/shell/PreShell.tsx` + `intro.ts`: thanh trên + cột bên cùng lớp CSS với AppShell, h1 + câu phụ của route; HTML tĩnh có chữ ngay, CLS 0); (2) `next.config.ts` `experimental.inlineCss: true` (CSS vào HTML, không chờ 3 tệp CSS). Đo bằng lệnh TL-1 (12×, 3 lượt, trung vị): TBT `/inbox` 195→85–114, `/threads` 205→107, `/settings/llm` 195→110, `/chat` 198→168, `/gradebook` 177→171, `/` 190–212→200, `/login` 130; **FCP ≈ 1,05–1,2 s như trước; LCP KHÔNG đổi (3,4–4,2 s).**
**Vì sao LCP không đổi (đọc mã lantern, `@paulirish/trace_engine` 0.0.53, `getFirstPaintBasedGraph`):** LCP mô phỏng = thời điểm hoàn thành MỌI nút mạng + CPU **kết thúc trước mốc vẽ LCP quan sát được** (bản pessimistic coi mọi request là chặn vẽ). Trên máy quan sát (không throttle) mọi chunk JS (≈ 254 KB gz) về xong ở 33–60 ms, còn lần vẽ đầu ở 52–76 ms (bị chính tác vụ boot JS chen trước) ⇒ toàn bộ JS nằm trong đồ thị LCP ⇒ LCP ≈ thời gian tải + chạy 254 KB JS ở 1,6 Mbps ≈ 3,4 s, bất kể chữ đã có trong HTML. `/login` (JS 191 KB, không khung app, không mock) cũng **LCP 3,3 s** ⇒ sàn của khung (react-dom 72 + next client 43 + router 39 KB gz ≈ 155 KB) đã vượt ngân sách 2,5 s ở 1,6 Mbps.
**Đã thử và bỏ:** `import()` tách AppShell/SessionProvider/ThreadsBackground khỏi layout (`AuthedApp`): JS tổng tăng 252→292 KB, TBT không giảm — hoàn lại. `inlineCss` làm HTML 387 KB (CSS lặp trong payload RSC) và lần vẽ đầu **muộn hơn** (76 ms so với 52 ms) — LCP không hưởng lợi.
**Phương án:** (a) cắt JS khởi động ≥ 100 KB gz — vượt khả năng khi vẫn dùng App Router + react-query ở root (đã thấy sàn 155 KB); (b) chỉ đo LCP bằng `throttlingMethod: devtools` (throttle thật, LCP thật ≈ FCP vì chữ đã ở HTML) — đổi cách đo, cần TL / PM chấp nhận; (c) giữ LCP ở `warn`, trả TBT về `error` (đã khả thi với số trên) và ghi nợ "sàn khung". Câu hỏi: chọn (b) hay (c)? Có cách hợp lệ khiến lần vẽ đầu **đứng trước** mốc hoàn thành JS trong trace quan sát (ví dụ HTML nhỏ + CSS tối thiểu), ngoài việc chạy đua mili-giây?

**TL trả lời (2026-10-08):**

**Kết luận.** Chưa chọn (b) hay (c). LCP đỏ hiện nay **chủ yếu là lỗi thật, không phải do cách đo**. Đo bằng throttle thật (`devtools`), LCP vẫn 3,6–4,3 s ở 5 / 6 route người dùng, vì phần tử LCP là chữ do client vẽ **sau** khi có phiên, không phải chữ của `PreShell`. Như vậy là trái AC8. Việc phải làm trước, và không cần PM: sửa để chữ từ máy chủ là phần tử LCP (mục 2). Sau đó đo lại cả hai cách. Nếu simulate vẫn > 2,5 s trong khi devtools ≤ 2,5 s thì PM chọn giữa (b') và (c) ở mục 4; TL khuyên (b'). (b) với lý do "LCP thật ≈ FCP" đã bị số đo bác. Không có cách hợp lệ nào khiến lần vẽ đầu đứng trước JS trong trace quan sát (mục 3).

1. **Số đo.** Bản chép `frontend/` đúng @ `eb19e09` (đã `cmp` 5 tệp), đặt trong `/tmp`, chạy `build:gate` + `next start` + `lighthouse-api.mjs` trên cổng riêng; Lighthouse 12.6.1, Chromium của Playwright, 3 lượt mỗi URL, lấy trung vị, `benchmarkIndex` 3661–3822, CPU 4× mặc định. Chạy cùng một bản dựng với hai `throttlingMethod`:

   | URL | simulate LCP | simulate FCP | **devtools LCP** | devtools FCP | phần tử LCP (cả hai cách) |
   |---|---|---|---|---|---|
   | `/` | 3.928 | 1.059 | **4.197** | 839 | `h1` "Chào Uyên" 148×35, client, sau phiên |
   | `/chat` | 4.016 | 1.054 | **4.236** | 837 | `p.ChatScreen…introText` 380×96, client |
   | `/threads` | 3.988 | 1.054 | **4.224** | 832 | `p.Layout…desc` "Câu hỏi công khai của An ninh mạng – 761987…" 380×72, client |
   | `/inbox` | 4.025 | 1.205 | **3.812** | 840 | `p.Layout…desc` (simulate); devtools: nút đã bị gỡ khỏi DOM |
   | `/gradebook` | 3.949 | 1.204 | **831** | 831 | devtools: nút đã gỡ = chữ `PreShell` ⇒ đạt; simulate: `p.desc` client |
   | `/settings/llm` | 3.898 | 1.054 | **3.588** | 827 | `p.Layout…desc`, client |
   | `/dev/ui` | 3.487 | 1.656 | 2.918 | 2.918 | `li` của `/dev/ui` |

   Ba lượt devtools lệch nhau ≤ 60 ms. TBT simulate 15–29 ms (máy này; TL-1: CI ≈ 12× máy này).

2. **Nguyên nhân thật (P1) và hướng sửa.** `AuthGate` trả `<PreShell>` khi chưa có phiên. Khi `authenticated`, nó trả một cây khác (`SessionProvider` + `children`), nên `h1` / `p.desc` của trang là **nút DOM mới**. Chrome phát ứng viên LCP mới mỗi khi một phần tử mới được vẽ trong viewport **lớn hơn** ứng viên hiện tại. Ứng viên cũ đã bị gỡ khỏi DOM vẫn được tính, và mọi phần tử đều tính theo diện tích hiển thị lúc vẽ đầu. Chữ sau phiên ở 5 route (tên lớp, "Chào …", lời giới thiệu chat) lớn hơn chữ `PreShell`, nên LCP = lúc có phiên ≈ JS + `/auth/refresh` + `/me/courses`. `/gradebook` là trường hợp đã đúng: chữ máy chủ vẫn lớn nhất, nên LCP = FCP = 831 ms. Điều kiện cần giữ, kiểm được: **trong màn hình đầu (viewport mobile của Lighthouse), khối chữ hoặc ảnh lớn nhất phải có sẵn trong HTML máy chủ; nội dung sau phiên không được thêm phần tử nào lớn hơn nó.** Hai cách làm, dev chọn (đều trong phạm vi AC1 / AC8):
   - (i) Tiêu đề và câu phụ thật của route render ở máy chủ, **ngoài** `AuthGate`, nên trước và sau phiên vẫn là cùng một nút DOM, không remount. Chữ theo phiên ("Chào Uyên", tên lớp) đặt vào phần tử nhỏ hơn khối máy chủ. Cách này bền: thêm màn mới không tái phát.
   - (ii) Tối thiểu: trong `intro.ts`, câu phụ máy chủ của mỗi route có diện tích ≥ khối chữ lớn nhất xuất hiện sau phiên. Ví dụ `/chat` đưa câu giới thiệu vào `intro`; `/` thêm câu phụ để không bị `h1` 148×35 vượt. Rẻ hơn nhưng dễ vỡ khi sửa chữ.
   Kiểm: AC8 (`largest-contentful-paint-element` bắt đầu bằng `<h1` / `<p` **của `PreShell`** + spec `lcp before refresh`), và một lượt `--collect.settings.throttlingMethod=devtools` → LCP ≈ FCP như `/gradebook`.

3. **Phần đúng trong phân tích lantern, và câu hỏi cuối.** Lantern (`FirstContentfulPaint.getRenderBlockingNodeData` / `getFirstPaintBasedGraph`, trace_engine 0.0.53) loại một request khỏi đồ thị LCP chỉ khi nó **bắt đầu hoặc kết thúc sau** mốc LCP quan sát. Một script cũng bị loại nếu tác vụ `EvaluateScript` đầu tiên của nó **bắt đầu sau** mốc đó. Hai PoC trang tĩnh (h1 + p có sẵn trong HTML, một script 253 KB không nén được, `python3 -m http.server`, 3 lượt):
   - Script `async` trong `<head>`: FCP 642–671 ms, **LCP 2.101 ms**, LCP quan sát 40–55 ms, script tải xong ở 8–10 ms.
   - Script chèn sau `requestAnimationFrame` + `setTimeout(0)`: **LCP 2.145–2.149 ms**, vì script vẫn tải xong ở 31–32 ms, trước lần vẽ ở 34–38 ms.
   ⇒ Trên localhost, JS gần như luôn về trước lần vẽ. Riêng việc tải 253 KB ở 1,6 Mbps đã tạo sàn ≈ 2,1 s cho LCP mô phỏng, chưa tính CPU. **Không có cách hợp lệ** để đảo thứ tự. Cách duy nhất là cố ý trì hoãn tải JS tới sau lần vẽ (`setTimeout` dài, chờ `load`). Làm vậy chỉ đổi số trong lab, người dùng 4G thật không được lợi gì (với họ HTML vốn đã vẽ trước JS), còn hydrate thì chậm đi. Đó là nới cách đo trá hình (AC6): **không làm**. Sau khi sửa P1, mốc cắt sẽ lùi về lúc vẽ khung (≈ 50–80 ms), nên `/auth/refresh`, `/me/courses` và CPU sau hydrate bị loại. Nhưng ≈ 254 KB JS vẫn nằm trong đồ thị ⇒ simulate ước 2,2–3,3 s [SUY LUẬN: sàn PoC 2,1 s + CPU chạy JS trước lần vẽ × 4; `/login` của dev 3,3 s]. Có thể vẫn đỏ.

4. **→ PM, chỉ khi sau P1 simulate > 2,5 s mà devtools ≤ 2,5 s:**
   - **(b') — TL khuyên.** Thêm một lượt `lhci collect` riêng với `throttlingMethod: devtools`, chỉ để kiểm LCP: 7 URL, ≥ 3 lượt, `median`, `2500`, mức `error`. Lượt simulate hiện tại giữ nguyên cho TBT / CLS / JS (ngưỡng và 4× không đổi), LCP ở lượt này chỉ ghi số. Lý do: [Lighthouse throttling.md](https://github.com/GoogleChrome/lighthouse/blob/main/docs/throttling.md) ghi simulate "suffers from edge cases", còn devtools là cách Lighthouse hỗ trợ chính thức và có trace khớp với số. Số đo ở mục 1 cho thấy devtools **vẫn bắt được** lỗi loại P1 (đỏ 5 / 6 route), nên đây không phải nới ngưỡng. Chi phí: ≈ 6 phút CI cho 21 lượt (đo trên máy này: 2 cấu hình mất 740 s).
   - **(c) LCP ở `warn`.** Không bắt được hồi quy loại P1 (chữ client lớn hơn chữ máy chủ), nên không khuyên.
   - (b) như đang viết (bỏ simulate cho LCP vì "LCP thật ≈ FCP"): giả định sai ở bản `eb19e09`; chỉ đúng sau khi sửa P1.
   - Chọn (b') hay (c) đều là đổi chữ AC4 / AC6 ("không nới cách đo"), nên cần một dòng `proposals.md` được PM chấp nhận rồi giao BA. Dev không tự đổi `lighthouserc.json`.

5. **Gợi ý về `inlineCss`.** Chính dev đã đo: HTML 387 KB và lần vẽ đầu muộn hơn. Ở devtools, FCP 827–840 ms là số **có** `inlineCss`; tôi chưa đo bản không có. Đo một lượt devtools không `inlineCss`; nếu FCP / LCP không tệ hơn thì gỡ (HTML nhỏ hơn có lợi cho người dùng 4G thật).

Lệnh tái hiện (trên bản chép, không đụng cổng 3310 / 3312 của dev): `E2E_API_PORT=3412 pnpm build:gate`; `lhci collect` với `url` → `localhost:3410`, `startServerCommand` = `LH_API_PORT=3412 LH_ORIGIN=http://localhost:3410 node lighthouse-api.mjs & exec pnpm exec next start -p 3410`, `puppeteerScript` với `ORIGIN` 3410; chạy lần hai thêm `settings.throttlingMethod: "devtools"`. Phần tử LCP: `jq '.audits["largest-contentful-paint-element"].details.items[0].items[0].node | {nodeLabel, selector, boundingRect}'`.

## TL-3 (CI: `TestJoinFailureTimingEqualized` đỏ ổn định trên GitHub Actions — nên đổi cách đo hay đổi mã?)

**Bối cảnh.** CI mới trên `EduPilot-v2`, nhánh `sprint/5-pe` (`go test -race ./...` trong job `Go`). Hai lượt liên tiếp (một lượt chạy lại) cùng đỏ ở `internal/course/join_limit_test.go:58` (P2, không liên quan PE):
`require.LessOrEqual(hi-lo / lo, 0.35)` — lượt 1: **0,403**; lượt 2: **1,375** với trung vị mỗi nguyên nhân (20 mẫu, xen kẽ): `lớp lưu trữ 1,147 ms · sai tên miền 0,924 ms · mã tắt 0,748 ms · mã hết hạn 0,694 ms · sai mã 0,626 ms · mã cũ 0,483 ms`. Cục bộ (macOS arm64) test xanh, ngưỡng 0,35 vượt dễ.
Hai lỗi thật khác của cùng lượt CI đã sửa riêng (so thời điểm µs/ns trong `attempts_test.go`; `settleGoto` chưa đợi hydrate) — không thuộc câu hỏi này.

**Quan sát.** `lookupByCode` (`internal/course/join.go`) đã làm đúng một truy vấn + một phép so sánh hằng thời gian + một quyết định gộp cho mọi nguyên nhân; chênh lệch thời gian đo được là chênh lệch **một truy vấn có / không trả dòng** (giải mã `Course`) cộng nhiễu: `go test ./...` chạy nhiều gói song song trên runner 4 vCPU (gói `course` mất 156–278 s), và các trung vị chỉ 0,5–1,1 ms nên 0,2–0,6 ms nhiễu đã vượt 35 %.

**Câu hỏi.** Test này là phép đo thống kê vi mô (sub-ms) trên máy dùng chung; không thể ổn định bằng cách nới ngưỡng mà vẫn giữ ý nghĩa, và AGENTS cấm nới assertion để xanh. Chọn:
- (a) giữ ý nghĩa, đổi **cách đo**: so sánh theo **số truy vấn / số lần cấp phát** (đếm câu SQL qua `pgx` tracer: mọi nguyên nhân đúng 1 truy vấn, cùng số vòng quyết định) thay vì mili-giây — tất định, chạy được trên CI;
- (b) giữ phép đo thời gian nhưng chuyển sang job riêng chạy tuần tự (`-p 1`, `-run TestJoinFailureTimingEqualized -count=1`) với cờ build / env `TIMING_TESTS=1` (không chạy trong `go test ./...` thường), ngưỡng 0,35 giữ nguyên;
- (c) khuyên cái khác (ví dụ chuẩn hoá thật bằng cách luôn `SELECT` cùng số cột / dòng cho cả nhánh không tìm thấy).
Dev nghiêng về (a) (bằng chứng tất định), có thể kèm (b) cho phần "thời gian thật". Đây là test hợp đồng P2 nên cần ý kiến TL trước khi sửa; dev không nới ngưỡng.

**TL trả lời (2026-10-09):**

**Kết luận.** Làm **(b)** theo quy ước đã có trong repo, **không** đổi ngưỡng, không đổi chữ AC. Không cần (c). (a) chỉ là phần thêm tuỳ chọn, không được thay cho phép đo thời gian.

1. **Đỏ vì nhiễu, không vì mã.** Ngay trong số của lượt 2: bốn nguyên nhân **cùng đường mã và cùng hình dạng dòng** (tìm thấy dòng rồi quyết định gộp: `lớp lưu trữ` 1,147 · `sai tên miền` 0,924 · `mã tắt` 0,748 · `mã hết hạn` 0,694 ms) đã lệch nhau 65 %. Chênh lệch "có dòng / không dòng" (`mã cũ` 0,483, `sai mã` 0,626) còn **nhỏ hơn** mức nhiễu đó. Hai yếu tố làm phép đo vô nghĩa: `go test -race ./...` chạy nhiều gói song song trên 4 vCPU, và `-race` làm việc giải mã dòng (`Scan` khoảng 20 trường) chậm đi nhiều lần, nên phóng to đúng phần chênh "có dòng" [SUY LUẬN: hệ số chậm của race detector theo tài liệu Go là 2–20×, chưa đo riêng cho trường hợp này].
2. **(b), dùng lại quy ước sẵn có:** ba test thời gian của `auth` (`TestLoginTimingEqualized`, `TestRegisterTimingEqualized`, `TestForgotTimingEqualized`) đã `if testing.Short() { t.Skip("đo thời gian") }`, và đó là ba chỗ duy nhất trong repo dùng `testing.Short()`. Vậy:
   - Thêm đúng khối đó vào đầu `TestJoinFailureTimingEqualized`.
   - `ci.yml` job `Go`: bước test đổi thành `go test -race -short -tags testroutes ./...`.
   - Thêm một bước ngay sau, cùng job: `go test -tags testroutes -p 1 -count=3 -run 'TimingEqualized$' ./internal/auth/ ./internal/course/`. Bước này **không** `-race`, chạy tuần tự, mỗi test 3 lần và cả 3 phải xanh; ngưỡng 0,35 và 20 mẫu giữ nguyên.
   - Không cần biến môi trường hay build tag mới. Lệnh "Kiểm" của AC3 (`go test ./internal/course/... -run '…TestJoinFailureTimingEqualized…'`, không `-short`) vẫn chạy test như cũ, nên **không phải sửa spec** và không cần dòng proposals.
   - Bằng chứng nghiệm thu: 3 lượt CI liên tiếp xanh ở bước mới; dán 4 bộ trung vị vào handoff.
3. **(c) không cần.** Phần chênh còn lại chỉ là giải mã một dòng so với không có dòng, ngoài `-race` ở mức µs [SUY LUẬN, chưa đo]. Rate limit AC4 (5 lần / 10 phút / người, 20 / IP) không cho đủ mẫu để tách µs qua mạng. Đổi truy vấn sang `LEFT JOIN` từ `VALUES` sẽ làm sqlc sinh kiểu toàn cột nullable, thêm mã ánh xạ mà không có lợi đo được.
4. **(a): làm thêm thì được, nhưng không thay.** Thay phép đo thời gian bằng đếm truy vấn là đổi cách kiểm của AC3 (APPROVED), nên cần dòng proposals và PM quyết. Nếu muốn có bằng chứng tất định *bổ sung* thì giữ thật nhỏ: một `pgx.QueryTracer` trên pool riêng của test, mỗi nguyên nhân đúng 1 lần `GetCourseByJoinCode` trong `lookupByCode`. Không bắt buộc, vì `lookupByCode` hiện đã một truy vấn + một quyết định gộp.
5. **Nếu bước tuần tự vẫn đỏ ≥ 1 / 3 lượt** (runner quá nhiễu ngay cả khi chạy một mình): dừng, ghi số đo vào đây, và để PM chọn (a) qua proposals. Không nới 0,35, không tăng số mẫu khi chưa có quyết định.
