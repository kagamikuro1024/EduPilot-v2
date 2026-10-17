# Sprint 7: câu hỏi kỹ thuật cho Tech Lead

Dev / QC ghi câu hỏi kỹ thuật ở đây (`docs/team/RESEARCH.md`). Mỗi mục: `## TL-<số> (<ai>, <ngày>): <tiêu đề>`, câu hỏi, rồi Tech Lead trả lời ngay dưới.

## TL-1 (PM, 2026-10-14): Lighthouse simulate đỏ `resource-summary:script:size` sau khi thêm `StudyTimeTracker` (US-P5-03)

**Câu hỏi:** 4 commit `e98866c → ca9314b` xoay quanh ngân sách script 256.000 B. Byte tăng từ đâu? Cách gọn nhất để giữ ngưỡng, **không nới**?

**TL trả lời:**

**Kết luận:** tracker **không** phải thủ phạm: nạp thẳng (không lười, không trễ) chỉ thêm **≈ 0,8 KB** gzip. Mọi trang chỉ còn < 1 KB dư vì **khung ứng dụng chở ≈ 16 KB mã mock xuống mọi trang**, kể cả trang admin và trang có lớp thật. Bỏ phần đó ra khỏi gói chung thì mọi trang dư ≈ 15 KB. Khi đó tracker nạp thẳng được và gỡ được hai mẹo `React.lazy` + trễ 10 s.

**1. Số CI (artifact `lhci-reports-simulate`; số đo giống nhau ở cả 3 lượt mỗi URL):**

| Run · commit | `/chat` | `/settings/llm` | Ghi chú |
| --- | --- | --- | --- |
| 38475796261 · `f45454c` (tracker nạp thẳng) | 255.770 | **256.910** ✘ | `/` 241.485, `/threads` 247.921, `/gradebook` 251.258: mọi trang sát trần |
| 38477156424 · `e98866c` (tách gói) | **256.351** ✘ | **258.007** ✘ | tách thêm gói làm **tăng** tổng (thêm request, mỗi request có phần đầu HTTP + mã nạp) |
| 38479208927 · `843b49c` | ✔ | **257.542** ✘ | |
| 38480563555 · `d2601d0` | ✔ | **257.765** ✘ | |
| 38482696309 · `04e13df` (`React.lazy`) | **256.524** ✘ | 255.665 | |
| 38484054558 · `ca9314b` (gắn trễ 10 s) | ✔ | ✔ | xanh nhờ gói tracker nạp **sau** cửa sổ đo, dư vài trăm byte; commit kế thêm ≈ 1 KB vào phần chung là đỏ lại |

`/settings/llm` là trang **admin**: tracker không bao giờ gắn ở đó (`role === "student"`), vậy mà vẫn đỏ. Nghĩa là byte tăng nằm ở **phần chung**, không ở tracker.

**2. Byte nằm ở đâu.** PoC ở `/tmp/tl-fe`: `git archive ca9314b`, `pnpm build:gate`, `next start`, cộng gzip các script ban đầu của từng route. Thêm một lượt build với `productionBrowserSourceMaps` để chia byte theo tệp nguồn. Số tuyệt đối lệch với Lighthouse (cách đếm khác), nên chỉ dùng **chênh lệch**:

| Thay đổi (cộng dồn từ `ca9314b`) | `/chat` | `/settings/llm` |
| --- | --- | --- |
| gốc `ca9314b` | 255.995 | 271.098 |
| bỏ `<ThreadsBackground />` khỏi `app/(app)/layout.tsx` | **−10.439** | **−10.435** |
| bỏ badge mock của `AppShell` (`BT03_SEED`, `ticketStats`, `reviewPending`, `mergeTickets`) | **−5.427** | **−5.430** |
| tracker **nạp thẳng**, bỏ `lazy` + trễ 10 s | **+785** | **+785** |
| `RoutesSection.tsx` nhập `@/shared/domain/LLMRouteTable` thay vì thùng `@/shared/domain` | – | **−4.180** |
| **tổng** | **−15.081** | **−19.260** |

Chia theo nguồn (gzip, trang `/chat`): `src/mock/threads.ts` 6,9 KB, `mock/assess.ts` 3,7 KB, `mock/core.ts` 3,3 KB, `features/threads/threadsActions.ts` 1,3 KB, `mock/support.ts` + `derive.ts` + `notes.ts` + `state.ts` ≈ 1,7 KB. Ở `/settings/llm` thùng `@/shared/domain` kéo theo `AttendanceGrid`, `CitationList`, `Markdown`, `ExamTimer`, `VerificationState` (≈ 4,5 KB gồm CSS module).

**3. Làm gì (theo thứ tự, mỗi bước một commit):**
1. **Tracker về dạng đơn giản:** `import { StudyTimeTracker } from "./TrackingNotice"`, bỏ `TRACKER_DELAY_MS` và `React.lazy` của tracker. Trễ 10 s là né cửa sổ đo của Lighthouse, không giảm byte; nó còn làm hộp thoại minh bạch và nhịp đầu chậm 10 s. `CommandPalette` lười thì giữ được (không hại), nhưng không cần cho ngân sách.
2. **Mã mô phỏng ra khỏi gói chung.** Thêm `shared/shell/DemoShellExtras.tsx` gồm `<ThreadsBackground />` + phép tính badge mock. `AppShell` nạp nó bằng `React.lazy`, **chỉ khi phiên đang ở chế độ mô phỏng** (`!realCourseId && role !== "admin"`). Lý do: với lớp thật, badge mock luôn bằng 0 (`ticketStats(…, scope)` lọc theo id lớp thật, vé mock không khớp), và `DemoThreads` chỉ hiện khi không có lớp thật (`ThreadsScreen.tsx:14-15`). Phiên thật (mọi URL Lighthouse trừ admin, và người dùng thật) không bao giờ tải gói này. Hành vi demo giữ nguyên.
3. **Không nhập qua thùng `@/shared/domain`** ở trang không cần cả bộ: `RoutesSection.tsx` nhập thẳng `@/shared/domain/LLMRouteTable`. Cùng lý do mà `TrackingNotice.tsx` đã ghi chú khi nhập `useActivityHeartbeat`.
4. **Không nới ngưỡng, không đổi `numberOfRuns`, không thêm URL bỏ qua.** Sau bước 1–3, mỗi URL phải dư ≥ 10 KB. Ghi số mới vào handoff để lần sau biết còn bao nhiêu chỗ.

Minh hoạ bước 2 (không phải bản để chép):

```tsx
// AppShell.tsx
const DemoShellExtras = lazy(() => import("./DemoShellExtras").then((m) => ({ default: m.DemoShellExtras })));
const demo = !realCourseId && role !== "admin";
const [badges, setBadges] = useState({ inbox: 0, grading: 0 });
// …
{demo && <Suspense fallback={null}><DemoShellExtras scope={scope} now={now} onBadges={setBadges} /></Suspense>}

// DemoShellExtras.tsx: chứa ThreadsBackground + useDemoSlice(tickets/bt03/approved) → onBadges({ inbox, grading })
```

**Rủi ro:**
- Phiên mô phỏng thấy badge chậm một nhịp (gói lười tải xong mới có số). Chấp nhận vì là dữ liệu giả.
- Admin không còn `ThreadsBackground`. Admin không đăng thread mô phỏng nên không mất gì.
- E2E demo có kiểm badge / phản hồi trễ của trợ giảng thì phải chờ (`expect.poll`), không được sửa assertion.

**Ngoài phạm vi, ghi nợ:** `session.tsx` / `myCourses.ts` / `clock.ts` vẫn nhập `@/mock/core` (≈ 3,3 KB) cho chế độ mô phỏng. Gỡ khi màn mock cuối cùng được thay (D51), không làm trong US-P5-03.
