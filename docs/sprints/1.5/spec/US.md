# FEAT-prototype-ui Prototype giao diện toàn bộ tính năng
Nguồn: PRD §3–§4 (M0–M14), FLOWS F2–F17, `docs/DEMO_SCRIPT.md`, `docs/sprints/1.5/plan.md`; hợp đồng giao diện `docs/design/DESIGN.md` §14, `docs/design/INTEGRATION.md` mục 2. Chi tiết dữ liệu và tương tác từng route: `SRS.md` mục 4.
Phiên bản 5.1 · 2026-10-01 · v5 (APPROVED) thêm UI polish (#17 → 00-AC7…AC10, 01-AC11, 02-AC9, 02-AC10, 04-AC7) và Threads như thật (#18 → 01-AC4, 01-AC10, 01-AC12…AC14, 02-AC11). **v5.1 (#19, 37 lỗi QC thăm dò `qc/explore-v4.md`) thêm 36 AC: 00-AC11…AC15, 01-AC15…AC28, 02-AC12…AC21, 03-AC7…AC8, 04-AC8…AC12** (34 AC cho E1–E37; 01-AC28 và 02-AC21 chốt ba điểm QC hỏi thêm ở `qc/tc-US-PROTO-01.md`); bảng E → AC: `SRS.md` 4.10; nguồn số liệu: `SRS.md` 4.8; ba điểm chốt: `SRS.md` 4.3.1 J. Lịch sử phiên bản: `SRS.md` dòng đầu.

**Cập nhật theo góp ý #20 (PM chốt, 01/10; không đổi số AC):** 03-AC8 gọi tài liệu đang xử lý là "Chờ xử lý" (không "Chờ xác nhận", nhãn này dành riêng cho câu AI); 01-AC22 và 01-AC11 nêu rõ `/chat` ở 720–1099 px như 390 px; 01-AC18 thêm dòng "Đã ẩn n thông tin cá nhân" sau lối `Ẩn thông tin rồi đăng` (`SRS.md` 4.3.3). Nguồn: `docs/sprints/1.5/proposals.md` #20.

## Quy ước kiểm chung

Stack chạy: `pnpm dev` (hoặc `pnpm -C frontend dev`) → `http://localhost:3000`. `$F` = `http://localhost:3000`.

```bash
# mở route với một vai; in "CHAN" nếu ra màn chặn quyền, "MO" nếu không
open_as() {  # $1 vai (student|ta|teacher|admin)  $2 đường dẫn  $3 người (tuỳ chọn: sv-1..sv-4)
  curl -s -b "ep_demo_role=$1; ep_demo_person=${3:-}; ep_demo_course=int1006-1" "$F$2" \
    | grep -q 'Bạn không có quyền mở trang này' && echo "CHAN $1 $2" || echo "MO   $1 $2"; }
# văn bản hiển thị (bỏ thẻ script) của một route
visible() { curl -s -b "ep_demo_role=$1; ep_demo_person=${3:-}" "$F$2" | perl -0pe 's#<script.*?</script>##gs; s#<[^>]+># #g'; }
```

Đo bố cục (FR-X12, `SRS.md` 4.7): DevTools đúng bề rộng, dán đoạn sau vào Console. `ox` = số px tràn ngang trang; `cut` = chữ bị cắt ngang bởi khung chứa; `ell` = chữ bị `…` mà không có `title` đủ chữ. Đạt khi `{ ox: 0, cut: [], ell: [] }`. Vùng cố ý cuộn ngang đánh dấu `data-scroll-x`.

```js
// AUDIT
(() => { const ox = document.documentElement.scrollWidth - innerWidth, cut = [], ell = [];
  for (const e of document.querySelectorAll('main *, header *')) {
    if (e.children.length || !e.textContent.trim() || e.closest('[data-scroll-x]')) continue;
    if (getComputedStyle(e).textOverflow === 'ellipsis' && e.scrollWidth > e.clientWidth && !e.title) { ell.push(e.textContent.trim()); continue; }
    let p = e.parentElement; while (p && getComputedStyle(p).overflowX === 'visible') p = p.parentElement;
    if (!p) continue; const a = e.getBoundingClientRect(), b = p.getBoundingClientRect();
    if (a.right > b.right + 1 || a.left < b.left - 1) cut.push(e.textContent.trim().slice(0, 30));
  } return { ox, cut, ell }; })()
```

Móc đo dev thêm (`data-part`): `brand`, `chat-history`, `chat-thread`, `chat-composer`, `inbox-list`, `inbox-detail`, `inbox-reply`, `provider-status`, `provider-action`; v5.1 thêm: `page-title` (h1 của mọi route), `nav-badge-inbox`, `nav-badge-grading`, `bell-dot`, `thread-form`, `thread-row`, `thread-question`, `chat-sessions-button`, `cal-event`, `att-session-select`, `att-row`, `student-row` (kèm `data-student-id`), `today-task` (kèm `data-task-id`), `col-qt`, `col-status`, `settings-section`, `chart-point`, `chart-axis-label`.

Hai đoạn đo thêm (v5.1). `TOUCH` liệt kê vùng bấm nhỏ hơn 44 px (chạy ở 375 / 390 px), đạt khi trả `[]`; liên kết nằm giữa câu chữ đánh `data-inline` được miễn, ô tích tính theo nhãn bọc nó. `LEFT` đo lề trái tiêu đề trang.

```js
// TOUCH
[...document.querySelectorAll('main a, main button, main [role=tab], main [role=radio], main label, main input, header a, header button, nav a')]
  .filter(e => e.offsetParent && !e.closest('[data-inline]') && !(e.matches('input') && e.closest('label')))
  .map(e => { const b = e.getBoundingClientRect(); return { t: (e.getAttribute('aria-label') || e.textContent).trim().slice(0, 24), w: Math.round(b.width), h: Math.round(b.height) }; })
  .filter(x => x.w < 44 || x.h < 44)
// LEFT
Math.round(document.querySelector('[data-part=page-title]').getBoundingClientRect().left)
```

ID mẫu cố định (dev dùng đúng các ID này để lệnh kiểm chạy được): người `sv-1`…`sv-4` = Sinh viên A…D; `/threads/t-cbc`; `/practice/at-symmetric` (luyện chủ đề), `/practice/at-quiz01` (QUIZ01); `/assignments/bt03`; `/students/sv-3`; `/grading/sub-bt03-sv-2`; `/join/BX4P9TW`.

## US-PROTO-00: Chủ dự án muốn vào prototype bằng một vai bất kỳ và đổi vai giữa chừng để đi trọn kịch bản demo trong một trình duyệt
Ưu tiên: Must · Ước lượng: M · Sprint: 1 (bổ sung)

### Tiêu chí nghiệm thu
- AC1. Given chưa có cookie vai When mở `$F/` Then chuyển về `/login`; chọn vai (Sinh viên thì chọn A/B/C/D) → vào `/` của vai đó; dải "Bản mô phỏng · dữ liệu giả" luôn hiện.
  Kiểm: `curl -s -o /dev/null -w '%{redirect_url}\n' $F/` chứa `/login` (hoặc HTML trả về là màn chọn vai); tay: chọn Sinh viên B → thấy tên "Trần Thu Uyên" ở menu hồ sơ; `visible student / sv-2 | grep -c 'Bản mô phỏng'` ≥ 1.
- AC2. Given đang ở vai Sinh viên B và đã gửi D3 When `Đổi vai` sang Giảng viên Then `/inbox` có ticket D3 (trạng thái lưu qua `localStorage` khoá `ep_demo_state`); `Đặt lại dữ liệu demo` → ticket D3 biến mất, mọi màn về dữ liệu gốc mục 4.1.
  Kiểm: tay trên trình duyệt; DevTools → Application → Local Storage chỉ có khoá `ep_demo_state` của prototype, không có gì giống token.
- AC3. Given mã nguồn When chạy kiểm tĩnh Then sạch.
  Kiểm: `pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh`; `grep -rn 'fetch(' frontend/src --include=*.ts --include=*.tsx | grep -v 'src/shared/' ` → không in gì; `git diff main -- frontend/package.json` không thêm phụ thuộc nào ngoài `lucide-react`.
- AC4. Given Giảng viên When mở bộ chọn lớp Then có lớp 761987, lớp 761988, "Tất cả lớp của tôi", "Quản lý lớp này"; Sinh viên A thấy 2 lớp; B thấy 1; D không có lớp và "Hôm nay" là ô nhập mã.
  Kiểm: tay; `visible student / sv-4 | grep -c 'mã tham gia'` ≥ 1.
- AC5 (nhánh lỗi). Given route bất kỳ When thêm `?state=loading`, `?state=empty`, `?state=error` Then thấy skeleton đúng hình / trạng thái rỗng có một hành động / lỗi có câu khắc phục + `Thử lại`; không spinner giữa trang.
  Kiểm: `for s in empty error; do visible teacher "/inbox?state=$s" | grep -cE 'Thử lại|Không còn câu hỏi'; done` → mỗi lần ≥ 1; `loading` kiểm bằng mắt (skeleton, không spinner); tay trên 3 route mỗi nhóm.
- AC6 (phân quyền). Given vai bất kỳ When mở route ngoài quyền của vai theo ma trận `SRS.md` mục 2 Then thấy màn chặn quyền ("Bạn không có quyền mở trang này" + `Về Hôm nay`), không thấy nội dung trang; điều hướng của vai không có route đó.
  Kiểm:
  ```bash
  for r in /chat /me /practice /library /join; do open_as teacher $r; done   # đều CHAN
  for r in /inbox /attendance /gradebook /grading /admin/courses /observability; do open_as student $r sv-2; done  # đều CHAN
  for r in /chat /inbox /gradebook /calendar; do open_as admin $r; done      # đều CHAN
  open_as ta /class/members; open_as ta /admin/users                         # MO, CHAN
  ```
- AC7 (#17a). Given mọi vai ở 1440 px When nhìn sidebar Then vùng brand cao 56 px, viền dưới liền một đường với viền dưới topbar, logo đầy đủ cao 32 px; thu gọn sidebar → chỉ mark 32 × 32. Given 390 px Then topbar có mark 28 px link về `/`. Given `/login` Then logo cao 40 px (1440) / 32 px (390).
  Kiểm (Console, 1440): `document.querySelector('[data-part=brand] img').getBoundingClientRect().height` → `32`; `document.querySelector('[data-part=brand]').getBoundingClientRect().bottom === document.querySelector('header').getBoundingClientRect().bottom` → `true`; ảnh `/`, `/login` ở 1440 và 390 đính báo cáo QC.
- AC8 (#17b). Given Giảng viên ở 1440 px When nhìn topbar Then bộ chọn lớp hiện đủ "761987 · An ninh mạng", ô tìm nhanh hiện đủ chữ. Given 1024 px Then tên dài có `…` và rê chuột thấy tooltip đủ tên. Given 390 px Then chỉ "761987"; mở bộ chọn thấy tên đầy đủ.
  Kiểm: `AUDIT` ở 1440 trên `/`, `/inbox`, `/gradebook` → `ell: []`; `curl -s -b "ep_demo_role=teacher; ep_demo_course=int1006-1" $F/ | grep -c 'title="761987 · An ninh mạng"'` ≥ 1.
- AC9 (#17f). Given vai Giảng viên / Trợ giảng / Admin / Sinh viên B When mở menu hồ sơ Then dòng 1 là tên người, dòng 2 (chữ phụ) là vai; không vai nào hiện tên vai ở chỗ tên người.
  Kiểm:
  ```bash
  for p in "teacher:TS. Lê Thu Hà" "ta:Phạm Quốc Bảo" "admin:Đỗ Hoàng Nam" "student:Trần Thu Uyên"; do
    curl -s -b "ep_demo_role=${p%%:*}; ep_demo_person=sv-2; ep_demo_course=int1006-1" $F/ | grep -c "Tài khoản: ${p#*:}"; done   # mỗi dòng ≥ 1
  ```
  Tay: mở menu ở 3 vai, chụp ảnh.
- AC10 (#17h). Given mọi route của mọi vai When mở ở 1440 px và 390 px (route SV, `/attendance`, `/inbox` thêm 375 px) Then `AUDIT` trả `{ ox: 0, cut: [], ell: [] }`. Riêng các chỗ v4 lỗi (`SRS.md` 4.7 h1–h4): `/library` (SV), `/attendance`, `/observability`, `/admin/users` hết tràn ngang; dưới 720 px bảng của `/students`, `/grading`, `/documents`, `/admin/courses`, `/admin/users` hiện dạng danh sách, `/gradebook` cuộn ngang có cột tên dính trái; `/attendance` ẩn chú thích phím tắt; tab `/students/sv-3` cuộn ngang, không cắt "Hoạt động học".
  Kiểm: chạy `AUDIT` lần lượt từng route (danh sách route theo vai: `SRS.md` mục 2), ghi bảng route × bề rộng → kết quả vào báo cáo QC; ảnh 390 px của 4 route h1.
- AC11 (#19 E23). Given mọi vai, mọi route trừ `/login` When đo tiêu đề trang Then lề trái của `[data-part=page-title]` giống nhau ở mọi route: **240 px ở 1440 px** (sidebar 216 + `var(--ep-space-6)`), **16 px ở 390 px** (`var(--ep-space-4)`); khối đọc / biểu mẫu rộng tối đa 960 px, căn trái (không căn giữa); bảng và lưới rộng hơn vẫn bắt đầu từ cùng lề.
  Kiểm (Console): `LEFT` trên `/`, `/chat`, `/threads`, `/practice`, `/library`, `/me`, `/calendar` (SV B), `/inbox`, `/students`, `/gradebook` (GV) ở 1440 → đều `240`; ở 390 → đều `16`.
- AC12 (#19 E31). Given mọi route ở mọi bề rộng, kể cả khi đã cuộn 200 px When nhìn thanh trên Then nền đặc: không `backdrop-filter`, không màu có độ trong suốt; chữ phía sau không lộ qua thanh (DESIGN §21 cấm glass).
  Kiểm (Console): `(h => [getComputedStyle(h).backgroundColor, getComputedStyle(h).backdropFilter])(document.querySelector('header'))` → `['rgb(…)', 'none']` (không `rgba(`, không `/ 0.`); `grep -rnE 'backdrop-filter|transparent\)' frontend/src/shared/shell frontend/src/shared/ui` → không in gì; `bash scripts/ui-antipatterns.sh` sạch **và** script có thêm hai mẫu này (QC thử thêm `backdrop-filter: blur(4px)` vào một CSS → script phải đỏ, rồi hoàn lại). Tay: `/chat` 390, gửi một tin, cuộn 200 px: không chữ lộ mờ.
- AC13 (#19 E1, FR-X16). Given đường dẫn không tồn tại hoặc một route ném lỗi When mở Then thấy màn tiếng Việt trong app: 404 "Không tìm thấy trang" + `Về Hôm nay`; lỗi chạy "Trang này gặp sự cố" + `Thử lại` + `Về Hôm nay`. Trang lỗi mặc định tiếng Anh của Next.js ("This page couldn't load", "404 This page could not be found") **không bao giờ** hiện. Given `ep_demo_state` trong `localStorage` bị hỏng (JSON sai) When tải lại Then app tự dùng dữ liệu gốc, không sập.
  Kiểm: `curl -s $F/khong-co-trang | grep -c 'This page could'` → `0`; `curl -s -b "ep_demo_role=teacher" $F/khong-co-trang | grep -c 'Không tìm thấy trang'` ≥ 1; `ls frontend/src/app/error.tsx frontend/src/app/not-found.tsx` có cả hai; tay: DevTools → Application → `localStorage.setItem('ep_demo_state','{')` rồi tải lại → vào được `/`.
- AC14 (#19 E26, E30). Given mọi chuỗi thời gian tương đối trong app When đọc Then đều tính từ đồng hồ giả lập (`SRS.md` 4.8 N6), không chuỗi cứng: chuông của SV B ghi "Bài tập 03 đã nộp, đang chờ chấm · 6 ngày trước" (nộp 23/10 08:10) và "Tài liệu mới: Chương 5 — Quản lý khoá và PKI · hôm qua 14:00"; mốc phân công lớp 761988 là "hôm qua 16:40" ở Admin `/`, `/admin/courses` và chuông GV.
  Kiểm: `grep -rnE '[0-9]+ (ngày|giờ|phút) trước' frontend/src --include=*.ts --include=*.tsx | grep -v 'mock/derive.ts'` → không in gì; `visible student / sv-2 | grep -c '12 ngày trước'` → `0`; `for r in / /admin/courses; do visible admin $r | grep -ci 'hôm qua 16:40'; done` → mỗi lần ≥ 1; `visible admin / | grep -c '08:30'` → `0`; tay: mở chuông B và chuông GV đối chiếu.
- AC15 (#19 E22, FR-X14). Given 375 px (và 390 px) When đo vùng bấm ở mọi route SV (`/`, `/chat`, `/threads`, `/threads/t-cbc`, `/practice`, `/practice/at-symmetric`, `/practice/history`, `/library`, `/calendar`, `/me`, `/assignments/bt03`, `/join`) và ở `/attendance`, `/inbox` của GV Then `TOUCH` trả `[]`. Riêng các phần tử v4 lỗi: 11 chip chủ đề Threads (cao 40), ô tích "Nhờ AI trả lời…" (14 × 16 → vùng nhãn ≥ 44), `Báo cáo` (76 × 40), `Hỏi trợ lý AI` (117 × 40), bộ lọc Thư viện, Tuần / Tháng / Danh sách (40), liên kết quay lại "Threads", "Thoát về Luyện đề", "Kết quả của tôi", "Hôm nay", "Luyện đề" (cao 22).
  Kiểm (Console): `TOUCH` ở mỗi route → `[]`; ảnh 375 px đính báo cáo QC.

### Ngoài phạm vi
Đăng nhập thật, màn tài khoản F1, cookie `httpOnly`, đồng bộ trạng thái giữa hai trình duyệt khác nhau.

### Phụ thuộc
Nháp PM `eb488d6` (`src/shared`, `src/mock`). Quyết định PM trong `plan.md`.

## US-PROTO-01: Sinh viên muốn thấy việc nên làm, hỏi riêng, hỏi công khai, luyện đề, xem tài liệu, lịch, kết quả và bài tập của mình để thử trọn trải nghiệm học
Ưu tiên: Must · Ước lượng: L · Sprint: 1 (bổ sung)

Route: `/`, `/chat`, `/threads`, `/threads/[id]`, `/practice`, `/practice/[attemptId]`, `/practice/history`, `/library`, `/calendar`, `/me`, `/assignments/[id]`, `/join` — dữ liệu và tương tác ở `SRS.md` 4.3.

### Tiêu chí nghiệm thu
- AC1. Given Sinh viên B When mở từng route của US Then route mở được, khung nhìn đầu đúng `DESIGN.md` §14.x tương ứng, đúng một hành động chính mỗi vùng.
  Kiểm: `for r in / /chat /threads /threads/t-cbc /practice /practice/at-symmetric /practice/history /library /calendar /me /assignments/bt03 /join; do open_as student $r sv-2; done` → đều `MO`; tay: so khung nhìn đầu với §14.x.
- AC2. Given B ở `/chat` lớp 1 When gửi D1 Then thấy dòng "Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI", câu trả lời hiện dần nêu vắng 2 buổi và +0,75, khối số liệu gọn, `Nguồn tham khảo (2)` mở ngay dưới; không có chuỗi `[[`. When gửi D2 Then bị từ chối lịch sự, không có con số nào về Lê Quang Huy.
  Kiểm: tay theo `DEMO_SCRIPT.md` 02:15–03:45.
- AC3. Given B gửi D3 When câu trả lời xong Then thấy "AI chưa đủ chắc chắn về câu này" và "Đang chờ giảng viên · vừa gửi"; sau khi GV trả lời (US-PROTO-02 AC2), B thấy câu trả lời có nhãn giảng viên và `Đã rõ` đóng câu hỏi.
  Kiểm: tay theo `DEMO_SCRIPT.md` 04:15–06:10 (bỏ bước Mailpit).
- AC4. Given B ở `/threads` When mở form tạo thread Then thấy các trường: Tiêu đề câu hỏi (Input rõ ràng, bắt buộc), Chủ đề (Select theo `THREAD_TOPICS`), Nội dung chi tiết (Textarea, bắt buộc), checkbox "Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng" (mặc định bật) (Proposal #15).
  When soạn bài có MSSV `20229002` trong Tiêu đề hoặc Nội dung rồi bấm `Đăng câu hỏi` Then mở Dialog đúng hai lối: chọn lối 1 (`Chuyển sang chat riêng`) → sang `/chat` mang toàn bộ bản nháp, không mất chữ; chọn lối 2 (`Ẩn thông tin rồi đăng`) → ẩn MSSV thành `[đã ẩn]` rồi tạo thread.
  When tạo thread hợp lệ Then **chuyển ngay sang `/threads/${id}`**; khối AI chạy đúng trình tự `SRS.md` 4.3.1 E: "Trợ lý AI đang soạn…" (~1,2 s) → chữ chảy có nút `Dừng` → `Nguồn tham khảo (n)` → nhãn `Chờ xác nhận`. Nội dung theo mẫu chọn ở 4.3.1 D (#18): tiêu đề "Dùng lại IV trong CTR có sao không?" (Mật mã đối xứng) → mẫu S2, kết thúc bằng câu hỏi ngược về XOR hai bản mã, nguồn Chương 3 tr. 18–20; tiêu đề "Vì sao ECB làm lộ ảnh?" → mẫu S1, nguồn Chương 3 tr. 14–17. Hai câu hỏi khác nhau không ra cùng một câu trả lời.
  Kiểm: tay, bấm giờ; tạo 2 thread với 2 tiêu đề trên rồi so 2 câu trả lời; quan sát URL đổi sang `/threads/...` ngay sau khi đăng.
- AC5. Given vai Sinh viên When đọc văn bản hiển thị mọi route của US Then không có từ kỹ thuật AI, không số độ tin cậy, không điểm nháp, không nhãn rủi ro / ghi chú về chính mình.
  Kiểm: `for r in / /chat /threads /threads/t-cbc /practice /library /calendar /me /assignments/bt03; do visible student $r sv-2; done | grep -inE 'RAG|PII|fallback|trace|provider|confidence|redaction|độ tin cậy|cần chú ý|rủi ro'` → không in gì. Tay: `/assignments/bt03` trước khi GV công bố không có con số điểm.
- AC6. Given điện thoại 375 px When dùng mọi route của US Then không cuộn ngang, vùng chạm ≥ 44 px (đo bằng `TOUCH`, 00-AC15), bottom nav ≤ 5 đích, panel lịch sử chat thu lại và thay bằng nút `Phiên trước (4)` (01-AC22).
  Kiểm: DevTools 375 × 812, đi lần lượt các route; ảnh chụp đính vào báo cáo QC.
- AC7. Given B ở `/me` When nhập CK = 11 vào What-if Then báo lỗi tại ô. Sau bước điểm danh của GV, QT của B = 8,5; sau công bố BT03 (8,0), QT = 8,7 và What-if CK = 8,0 hiện 8,3 (`SRS.md` 4.1).
  Kiểm: tay.
- AC8 (nhánh lỗi). Given Sinh viên D When nhập mã `ABCDEFG` ở `/join` Then thấy câu chung "Mã không hợp lệ hoặc đã hết hạn…", không lộ lý do; nhập `BX4P9TW` → xem trước lớp 761988 → `Tham gia lớp` → "chờ giảng viên duyệt". Và: `?state=error` trên `/chat` giữ nguyên chữ trong composer.
  Kiểm: tay; `visible student /join/BX4P9TW sv-4 | grep -c 761988` ≥ 1.
- AC9 (phân quyền). Given vai TA / GV / Admin When mở `/chat`, `/me`, `/practice`, `/library`, `/assignments/bt03`, `/join` Then màn chặn quyền; Admin cũng bị chặn ở `/threads`, `/calendar`.
  Kiểm: `for v in ta teacher admin; do for r in /chat /me /practice /library /assignments/bt03 /join; do open_as $v $r; done; done` → đều `CHAN`; `open_as admin /threads; open_as admin /calendar` → `CHAN`.
- AC10 (#18). Given B ở `/threads` When đọc danh sách Then số phản hồi mỗi thread đúng cột "n" ở `SRS.md` 4.3.1 B (`t-cbc` 4, `t-salt` 3, `t-sqli` 3, `t-pin-rubric` 4, …). When mở `/threads/t-cbc` Then thấy từ trên xuống: khối câu hỏi gốc (Đặng Gia An, thời gian, chủ đề, tuần, "4 người tham gia"); khối câu trả lời AI `Chờ xác nhận` mẫu S1 + `Nguồn tham khảo (2)`; **một** vùng "Thảo luận (4)" gồm 4 phản hồi của sv-10, TA (có khối trích bài sv-10), sv-7, sv-19 theo thời gian; ô "Phản hồi của bạn" ở cuối chính vùng đó với `Gửi phản hồi` (chính) và `Hỏi trợ lý AI` (phụ). Không còn tiêu đề "Thảo luận của lớp" hay "Phản hồi trong thread này".
  Kiểm: `visible student /threads/t-cbc sv-2 | grep -cE 'Thảo luận của lớp|Phản hồi trong thread này'` → `0`; `visible student /threads/t-cbc sv-2 | grep -c 'Thảo luận (4)'` ≥ 1; tay: đếm bài từng thread trong danh sách so bảng 4.3.1 B.
- AC11 (#17e). Given B ở `/chat` 1440 px When phiên trống và khi phiên đã có ≥ 6 tin Then lịch sử phiên là một panel (viền, nền, bo góc FR-X11) rộng 240 px; hội thoại và composer cùng mép trái và cùng bề rộng; composer ghim đáy (cách đáy màn ≤ 24 px), hội thoại cuộn bên trong. Given dưới 1100 px (720–1099, 390, 375) Then panel lịch sử thu lại thành nút `Phiên trước (4)` (AC6, 01-AC22; #20).
  Kiểm (Console): `const r = p => document.querySelector('[data-part=' + p + ']').getBoundingClientRect(); [r('chat-thread').left === r('chat-composer').left, r('chat-thread').width === r('chat-composer').width, innerHeight - r('chat-composer').bottom <= 24]` → `[true, true, true]` ở cả hai trạng thái.
- AC12 (#18). Given B ở `/threads/t-cbc` When ô soạn trống và bấm `Hỏi trợ lý AI` Then AI đăng `Gợi ý thêm` của S1 ("…bao nhiêu khối bản rõ bị ảnh hưởng?") theo trình tự 4.3.1 E. When gõ "Còn CTR thì sao, có cần IV ngẫu nhiên không?" rồi bấm `Hỏi trợ lý AI` Then phản hồi của B hiện trước, AI trả lời ngay dưới theo S2 với dòng "↳ trả lời Trần Thu Uyên". When gửi phản hồi "@AI dùng lại nonce có sao không" bằng `Gửi phản hồi` Then xử lý như bấm `Hỏi trợ lý AI`. Trong lúc AI soạn, hai nút khoá; bấm `Dừng` giữ phần đã hiện + "Đã dừng" + `Hỏi lại`.
  Kiểm: tay, bấm giờ.
- AC13 (#18). Given B ở `/threads/t-salt` When gửi phản hồi "Em hiểu rồi ạ" Then phản hồi hiện ngay cuối vùng thảo luận và trang cuộn tới; ~2 s sau "Phạm Quốc Bảo đang trả lời…"; ~6 s sau phản hồi của TA có khối trích bài của B, nội dung cột "Phản hồi trễ của TA" mẫu H1; chuông có chấm chưa đọc "Phạm Quốc Bảo đã trả lời trong «Vì sao cần muối (salt) khi băm mật khẩu?»", bấm → về đúng phản hồi; `/threads` hiện `t-salt` 5 phản hồi. Given gửi xong chuyển ngay sang `/calendar` (trước 6 s) Then tới hạn chuông vẫn có thông báo, quay lại thread thấy phản hồi TA. Gửi phản hồi thứ hai trong cùng thread → không có phản hồi trễ nữa.
  Kiểm: tay, bấm giờ; tải lại trang ở giây thứ 3 rồi chờ.
- AC14 (nhánh lỗi, #18). Given B tạo thread "WPA3 chặn được KRACK không?" chủ đề "An toàn mạng không dây" (nhờ AI bật) Then AI hiện ngay toàn văn "Mình chưa đủ chắc chắn để gợi ý câu này từ tài liệu của lớp…", không nguồn, nhãn `Đang chờ giảng viên`, không có `Chờ xác nhận`; `/inbox` của GV không thêm ticket (Q4). Given ô soạn trống Then `Gửi phản hồi` khoá. Given gõ dở phản hồi rồi rời trang và quay lại Then chữ còn nguyên. Given phản hồi chứa `20229002` Then Dialog hai lối (như AC4).
  Kiểm: tay; đổi vai GV → `/inbox` số ticket không đổi.
- AC15 (#19 E1). Given B ở `/practice/at-symmetric` When trả lời đủ 10 câu — đúng câu 1–7 (đáp án đúng theo `correct`), chọn sai câu 8–9, gõ câu 10 "Em chưa rõ" — rồi `Kiểm tra` → `Xem kết quả` Then thấy "Bạn đúng 7/10 câu" và "Câu cần ôn (3)" (câu 8, 9, 10), mỗi câu có "Bạn chọn", "Đáp án đúng", giải thích, `Nguồn tham khảo`; `Ôn lại 3 câu sai` mở lượt mới chỉ gồm 3 câu đó; `/practice/history` có hàng đầu "Mật mã đối xứng · 10 câu · 7/10 · vừa xong"; chủ đề yếu ở `/practice` cộng dồn (Mật mã đối xứng: sai +3, tổng +10); không trang lỗi mặc định. When tải lại ở câu 4 (đã trả lời câu 1–3) Then mở lại "Câu 4/10", đáp án câu 1–3 còn nguyên. When `Luyện 10 câu` ở `/practice` Then luôn bắt đầu "Câu 1/10" của lượt mới. Chấm trả lời ngắn theo `SRS.md` 4.3.4.
  Kiểm: tay theo thứ tự trên, Console không lỗi; nhánh lỗi: 10/10 → "Bạn đúng cả 10 câu" + `Luyện chủ đề khác`.
- AC16 (#19 E11). Given `/practice/at-quiz01` (8 câu) và `/practice/at-symmetric` (10 câu) When so chữ đề Then không cặp câu nào trùng (sau khi bỏ dấu, chữ thường); trong lúc QUIZ01 chưa nộp, `Theo chủ đề` / `Ôn lại câu sai` / đề cũ không hiện câu QUIZ01 và không giải thích đáp án của chúng.
  Kiểm: tay: đọc câu 1–8 của QUIZ01 đối chiếu câu 1–10 của luyện đề (câu 1 luyện đề phải khác câu 1 QUIZ01); làm một lượt luyện, không giải thích nào trùng đề QUIZ01.
- AC17 (#19 E2, FR-X18). Given SV D (chưa vào lớp) When mở từng route: `/chat`, `/threads`, `/threads/t-cbc`, `/practice`, `/practice/at-symmetric`, `/practice/history`, `/library`, `/calendar`, `/me`, `/assignments/bt03` Then cùng màn "Bạn chưa vào lớp nào" (không phải màn chặn quyền) với ô nhập mã, không dòng nào của lớp, không phiên chat của người khác; sidebar chỉ có `Hôm nay`; bộ chọn lớp "Chưa có lớp". When D vào lớp 761988 (sau GV `Duyệt`) Then `/chat` "Chưa có phiên nào", `/me` "Chưa có điểm quá trình", `/practice/history` rỗng — không dữ liệu cá nhân nào của B. Chi tiết từng route: `SRS.md` 4.3.2.
  Kiểm:
  ```bash
  for r in /chat /threads /threads/t-cbc /practice /practice/at-symmetric /practice/history /library /calendar /me /assignments/bt03; do
    a=$(visible student $r sv-4 | grep -c 'Bạn chưa vào lớp nào')
    b=$(visible student $r sv-4 | grep -ciE 'phiên trước|Cách chọn độ dài khoá RSA|Nộp muộn Bài tập 03|CBC khác ECB|Chương 3 — Mật mã')
    echo "$r $a $b"; done          # mỗi dòng: ≥ 1 rồi 0
  visible student / sv-4 | grep -cE 'Chat riêng|Luyện đề|Thư viện|Lịch'   # 0 (sidebar chỉ Hôm nay)
  visible student / sv-4 | grep -ci 'mã tham gia'                          # ≥ 1
  ```
  Tay: GV `Duyệt` D → đổi vai D → kiểm `/chat`, `/me`, `/practice/history` như trên.
- AC18 (#19 E5, FR-X18). Given B ở `/threads` When đăng câu hỏi P1 (`SRS.md` 4.3.3: tiêu đề "Hỏi về bài tập 03", nội dung "Mail của em là uyen.tt229002@sv.edupilot.test, SĐT 0912345678.") Then **không đăng**, mở Dialog "Bài này có thông tin cá nhân" nêu "1 địa chỉ email, 1 số điện thoại" (không in lại giá trị), đúng hai nút `Chuyển sang chat riêng` · `Ẩn thông tin rồi đăng` + đóng; `Ẩn thông tin rồi đăng` → thread chứa đúng `Mail của em là [đã ẩn], SĐT [đã ẩn].`; `Chuyển sang chat riêng` → `/chat` mang nguyên bản nháp, không thread nào được tạo. Cùng hành vi cho `Gửi phản hồi`, `Hỏi trợ lý AI` (ô có chữ) và GV / TA `Lưu và xác nhận`. Các câu P2–P4 cho kết quả đúng bảng `SRS.md` 4.3.3 (P3 không bị chặn nhầm). GV / TA thấy hai nút `Ẩn thông tin rồi đăng` · `Quay lại sửa`.
  Ghi nhãn (#20c): sau lối `Ẩn thông tin rồi đăng`, ngay dưới bài vừa đăng hiện dòng chữ phụ "Đã ẩn n thông tin cá nhân" — cùng câu với chat riêng (01-AC2, không có vế "trước khi gửi cho AI"); n đếm **mọi loại** khớp (email, số điện thoại, MSSV, họ tên, điểm gắn danh tính), không in lại giá trị. P1 → "Đã ẩn 2 thông tin cá nhân".
  Kiểm (Console, sau lối `Ẩn thông tin rồi đăng`): `localStorage.getItem('ep_demo_state').includes('0912345678')` → `false`; `…includes('uyen.tt229002@')` → `false`; `document.body.innerText.includes('0912345678')` → `false`; số nút hành động trong dialog: `document.querySelectorAll('[role=dialog] button:not([aria-label=Đóng])').length` → `2`. Tay: P2, P3, P4; phản hồi chứa `20229002` (đã có ở AC14).
  Kiểm (#20c, sau lối ẩn với P1): `document.body.innerText.includes('Đã ẩn 2 thông tin cá nhân')` → `true`; với phản hồi chỉ có MSSV `20229002` → `includes('Đã ẩn 1 thông tin cá nhân')` → `true`.
- AC19 (#19 E13, E14). Given B mở form tạo thread When nhìn ô Chủ đề Then mặc định "Chọn chủ đề" (không giá trị), danh sách **không có "Thông báo"** (GV / TA có); `Đăng câu hỏi` khoá tới khi đủ tiêu đề + chủ đề + nội dung. When đăng với chủ đề "Mật mã đối xứng" Then chi tiết ghi "Trần Thu Uyên (bạn) · Mật mã đối xứng · Tuần 10 · vừa xong" (không thay chủ đề / tuần bằng "Câu hỏi của bạn"); hàng danh sách ghi "Mật mã đối xứng · Tuần 10 · vừa xong · Chờ xác nhận", có chip "Của bạn" và trích đoạn tối đa 120 ký tự như mọi hàng khác; URL `/threads/t-new-1` (số thứ tự trong phiên, không dùng thời gian).
  Kiểm (Console, trước khi chọn): `[...document.querySelectorAll('[data-part=thread-form] option')].map(o => o.textContent)` → phần tử đầu `Chọn chủ đề`, không có `Thông báo`; tay: đăng thread rồi đối chiếu chi tiết + danh sách.
- AC20 (#19 E15). Given B ở `/threads/t-rsa-key` When gõ "Em nghĩ 2048 bit an toàn đến 2030 theo NIST SP 800-57." rồi `Hỏi trợ lý AI` Then phản hồi của B hiện trước (tên B, không nhãn AI), ngay dưới là khối "Trợ lý AI của lớp" (`Chờ xác nhận`) theo mẫu K1 (`SRS.md` 4.3.1 C: câu hỏi ngược về dữ liệu giữ bí mật 15 năm; nguồn Chương 5 trang 3–5), ô soạn **trống** sau khi bấm, không chữ của AI nào lọt vào ô soạn, không có chữ "lan truyền lỗi" (nội dung CBC) trong thread RSA. When ô trống và bấm lần nữa Then AI đăng `Gợi ý thêm` của K1 ("vì sao khoá ECC 256 bit…"), không lặp lại mẫu cũ.
  Kiểm: `visible student /threads/t-rsa-key sv-2 | grep -c 'lan truyền lỗi'` → `0`; tay: chuỗi thao tác trên, rồi `/threads/t-cbc` ô trống → `Gợi ý thêm` của S1 (khớp AC12).
- AC21 (#19 E32, E34, FR-X19). Given B ở `/threads` 1440 × 900, form chưa mở When nhìn Then danh sách thread là chính: hàng đầu tiên cách đỉnh trang ≤ 420 px; form nằm sau một nút chính `Đặt câu hỏi` và mở tại chỗ. Given nút gửi đang khoá Then ngay cạnh nút có dòng nêu đúng phần còn thiếu — `Đăng câu hỏi`: "Còn thiếu: tiêu đề, chủ đề, nội dung" (chỉ liệt kê phần thiếu thật); `Gửi phản hồi`: "Nhập nội dung phản hồi"; `Kiểm tra` ở luyện đề: "Chọn một đáp án để kiểm tra" — và nút có `aria-describedby` trỏ tới dòng đó.
  Kiểm (Console, 1440 × 900): `document.querySelector('[data-part=thread-row]').getBoundingClientRect().top` ≤ `420`; sau khi mở form: `[...document.querySelectorAll('button:disabled')].every(b => b.getAttribute('aria-describedby'))` → `true`; tay: mỗi bước gõ dòng nhắc thu hẹp theo phần còn thiếu.
- AC22 (#19 E20; #20b). Given B ở `/chat` 390 px **hoặc 720–1099 px** When nhìn đầu vùng chat Then có nút `Phiên trước (4)` cao ≥ 44 px; bấm → bảng trượt từ đáy liệt kê 4 phiên (tiêu đề + dòng phụ) và `Phiên mới`; chọn một phiên → hiện câu hỏi và câu trả lời của phiên đó (chỉ đọc) cùng nút `Hỏi tiếp` mở phiên mới mang ngữ cảnh; hội thoại và composer cùng mép trái, cùng bề rộng (như 01-AC11). Ở ≥ 1100 px vẫn là panel `chat-history` (01-AC11).
  Kiểm (Console, 390): `document.querySelector('[data-part=chat-sessions-button]').getBoundingClientRect().height` ≥ `44`; tay: mở từng phiên, đóng bảng bằng vuốt xuống / `Esc`.
- AC23 (#19 E4, E26, FR-X17). Given trước khi GV công bố BT03 When B mở chuông Then có "Bài tập 03 đã nộp, đang chờ chấm · 6 ngày trước". When GV `Duyệt bài` B rồi `Công bố` (US-PROTO-03 AC2, xác nhận) và đổi sang B Then chuông có chấm chưa đọc, mục đầu "Điểm Bài tập 03 đã được công bố · vừa xong"; mục "đang chờ chấm" **biến mất**; bấm mục → `/assignments/bt03` có điểm 8,0 và chấm biến khi hết thông báo chưa đọc. Tương tự D nhận "Bạn đã được duyệt vào lớp An ninh mạng – 761988" sau GV `Duyệt`. Bảng sự kiện đầy đủ: `SRS.md` 4.9.
  Kiểm: tay theo `DEMO_SCRIPT.md` 08:00–10:00 + 01:10–01:55; Console: `document.querySelector('[data-part=bell-dot]') !== null` → `true` ngay sau công bố, `false` sau khi đã đọc hết.
- AC24 (#19 E10, E26). Given B ở `/library` Then 10 mục có tên, tuần, ngày tải, số trang, dung lượng đúng bảng `SRS.md` 4.8 N5; mọi `Nguồn tham khảo` của AI (chat D1, Threads, luyện đề) và chuông dùng đúng cột "Tên hiển thị" của bảng đó (ví dụ "Quy chế môn học An ninh mạng – 761987", "Chương 5 — Quản lý khoá và PKI").
  Kiểm: `visible student /library sv-2 | grep -c 'Chương 5 — Quản lý khoá và PKI'` ≥ `1`; `… | grep -c 'Modern Network Security Threats'` ≥ `1`; `… | grep -cE 'Đáp án đề|Mordern'` → `0`; tay: mở mọi `Nguồn tham khảo` của D1, `t-cbc`, luyện đề — tiêu đề nằm trong bảng N5.
- AC25 (#19 E24). Given B ở `/calendar` chế độ Tuần 1440 px Then lưới tuần cao theo nội dung (không cao cố định): khoảng trống dưới lưới ≤ 48 px; mỗi cột ≥ 140 px; nhãn sự kiện ("QUIZ01 đóng · Mật mã đối xứng") tối đa 2 dòng, không ngắt giữa từ.
  Kiểm (Console): `[...document.querySelectorAll('[data-part=cal-event]')].every(e => e.getBoundingClientRect().height <= 2 * parseFloat(getComputedStyle(e).lineHeight) + 16)` → `true`; tay: ảnh 1440 px.
- AC26 (#19 E27). Given B ở `/me` lúc mới vào Then "Cập nhật Thứ Năm, 29 tháng 10 09:20". When GV `Lưu điểm danh` (đồng hồ giả lập lúc đó, ví dụ 09:24) rồi B mở lại `/me` Then ghi đúng giờ giả lập lúc lưu ("… 09:24") và QT 8,5; sau GV `Công bố` BT03 → mốc lúc công bố và QT 8,7. (Số QT giữ nguyên #12 / #13.)
  Kiểm: tay theo `DEMO_SCRIPT.md` 07:00–10:00, đối chiếu đồng hồ.
- AC27 (#19 E36). Given mọi vai ở `/threads/t-cbc` 1440 px When đo thẻ "Câu hỏi gốc" Then từ đáy phần tử nội dung cuối tới đáy thẻ ≤ 21 px (`var(--ep-space-5)` + 1 px), không khoảng trống.
  Kiểm (Console): `(c => c.getBoundingClientRect().bottom - c.lastElementChild.getBoundingClientRect().bottom)(document.querySelector('[data-part=thread-question]'))` ≤ `21`.
- AC28 (#19, PM chốt `SRS.md` 4.3.1 J1, J2). Given B ở `/threads/t-cbc` (đã có câu AI chính S1, chưa dùng `Gợi ý thêm`) When ô soạn trống và bấm `Hỏi trợ lý AI` lần 1 Then AI đăng `Gợi ý thêm` của S1 ("Thử thêm: nếu một khối bản mã CBC bị hỏng…") theo trình tự E; When bấm lần 2 và lần 3 (ô vẫn trống) Then **không** có bài mới (số "Thảo luận (n)" không đổi) và hiện dòng thông báo tại ô soạn "Trợ lý đã đưa hết gợi ý có trong tài liệu của lớp…", nút không bị khoá; When gõ một câu rồi bấm `Hỏi trợ lý AI` (bao nhiêu lần cũng được) Then mỗi lần là một cặp mới: phản hồi của B + một bài AI. Given thread có câu hỏi gốc không khớp mẫu (tạo từ AC14) When ô trống bấm `Hỏi trợ lý AI` Then không bài mới, dòng "Mình đã báo giảng viên; câu trả lời sẽ hiện ngay trong thread này." Given B ở một thread **chưa** có phản hồi nào của sinh viên trong phiên (ví dụ `t-salt` sau Đặt lại dữ liệu) When gõ "Em hiểu rồi ạ" rồi bấm `Hỏi trợ lý AI` (không bấm `Gửi phản hồi`) Then phản hồi của B hiện, và ~2 s sau "Phạm Quốc Bảo đang trả lời…", ~6 s sau phản hồi của TA (mẫu H1) + chuông "Phạm Quốc Bảo đã trả lời trong «Vì sao cần muối (salt) khi băm mật khẩu?»", song song với khối AI; bấm tiếp `Gửi phản hồi` / `Hỏi trợ lý AI` có chữ ở cùng thread → **không** có phản hồi trễ thứ hai. Given ô trống bấm `Hỏi trợ lý AI` trong thread chưa có phản hồi của sinh viên Then không phản hồi trễ nào. Given TA hoặc GV bấm `Hỏi trợ lý AI` có chữ Then không phản hồi trễ.
  Kiểm: tay, bấm giờ (dung sai như `SRS.md` 4.3.1 E, G); đọc số n trong tiêu đề "Thảo luận (n)" trước và sau lần bấm thứ hai (không đổi) và sau mỗi lần bấm có chữ (+2).

### Ngoài phạm vi
Phúc khảo đầy đủ (chỉ có form gửi), thi thử đủ ma trận, ICS thật, xem trước PDF thật (hiện trang mẫu).

### Phụ thuộc
US-PROTO-00.

## US-PROTO-02: Giảng viên / trợ giảng muốn xử lý việc trong ngày, trả lời câu hỏi, điểm danh, xem sinh viên và duyệt thành viên để thử vận hành một lớp
Ưu tiên: Must · Ước lượng: L · Sprint: 1 (bổ sung)

Route: `/` (GV/TA), `/inbox`, `/students`, `/students/[id]`, `/attendance`, `/class/members` — `SRS.md` 4.4.

### Tiêu chí nghiệm thu
- AC1. Given Giảng viên When mở `/` Then thấy "6 việc cần xử lý hôm nay" theo thứ tự `SRS.md` 4.4, mỗi việc ghi tên lớp, ticket chờ 3 ngày 4 giờ đứng đầu, có việc "Thiết lập lớp mới — 761988"; chuông có thông báo phân công lớp 761988 kèm `BX4P9TW`; không thẻ số liệu, không biểu đồ ở khung nhìn đầu.
  Kiểm: `for r in / /inbox /students /students/sv-3 /attendance /class/members; do open_as teacher $r; done` → đều `MO`; tay so §14.1.
- AC2. Given ticket D3 của B ở `/inbox` When `Nhận` → gõ D4 → `Gửi trả lời` Then ticket chuyển Claimed rồi Answered, dòng "Đã gửi thư thông báo… (mô phỏng)" hiện tại chỗ; ticket mẫu "đã có người nhận" hiện "Phạm Quốc Bảo đã nhận lúc 09:12" thay nút `Nhận`.
  Kiểm: tay theo `DEMO_SCRIPT.md` 04:45–05:35.
- AC3. Given `/attendance` buổi 10 lớp 1 (29/10) When dùng bàn phím (↑/↓, `1`–`4`, `P`) đánh 2 vắng, 1 muộn, +phát biểu cho B rồi `Lưu điểm danh` Then mỗi thao tác hiện dòng Hoàn tác 5 s và trạng thái lưu; kết thúc "Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng"; `/me` của B lên 8,5. Toàn bộ ≤ 60 s.
  Kiểm: tay, bấm giờ; lặp lại ở 375 px bằng chạm.
- AC4. Given Sinh viên D đã gửi yêu cầu vào lớp 761988 When GV mở `/class/members` lớp 2 và `Duyệt` D Then D thành thành viên; đổi vai về D → bộ chọn lớp có 761988.
  Kiểm: tay theo `DEMO_SCRIPT.md` 01:10–01:55.
- AC5. Given màn hình `/inbox` trên desktop When hiển thị Then phân định rõ thành 2 panel độc lập (Proposal #16): cột danh sách ticket (bên trái, max 380px) và panel chi tiết ticket (bên phải), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc (`var(--radius-sm)` hoặc `var(--radius-md)`), padding rõ ràng, và cuộn độc lập. Trên điện thoại 375 px: `/inbox` chuyển thành danh sách → chi tiết; `/attendance` hàng có kẻ, không card, vùng chạm ≥ 44 px.
  Kiểm: DevTools 375 × 812 và 1440 px; ảnh chụp.
- AC6 (nhánh lỗi). Given `/attendance` When bật "Giả lập mất mạng" và đổi 2 ô Then thấy "Đang chờ mạng · 2 thay đổi"; tắt công tắc → "Đã lưu …", không mất thay đổi. Given `/students?state=error` Then lỗi có `Thử lại`.
  Kiểm: tay.
- AC7 (phân quyền). Given Sinh viên / Admin When mở các route của US Then màn chặn quyền. Given TA When mở `/class/members` Then xem và `Duyệt` được, không có `Tạo lại mã`, không có `Mời ra khỏi lớp`.
  Kiểm: `for v in student admin; do for r in /inbox /students /students/sv-3 /attendance /class/members; do open_as $v $r sv-2; done; done` → đều `CHAN`; `visible ta /class/members | grep -c 'Tạo lại mã'` → `0`.
- AC8. Given Giảng viên hoặc Trợ giảng ở chi tiết thread `/threads/[id]` có câu trả lời AI `Chờ xác nhận` When bấm `Chỉnh sửa` Then mở ô sửa inline chứa nội dung câu trả lời; When sửa nội dung và bấm `Lưu và xác nhận` Then trạng thái câu trả lời chuyển thành `Đã được giảng viên sửa & xác nhận` (CORRECTED), hiển thị nội dung đã sửa kèm nút/chi tiết "Xem câu trả lời AI gốc" để đối chiếu (Proposal #15); When bấm `Xác nhận` Then trạng thái chuyển thành `Đã được giảng viên xác nhận`; When bấm `Loại khỏi tri thức` Then câu trả lời bị ẩn và có dòng Hoàn tác.
  Kiểm: tay.
- AC9 (#17c). Given Giảng viên ở `/inbox` 1440 px When nhìn panel danh sách Then hàng tab / chip lọc cuộn ngang, đủ chữ ("Tất cả 6"); mỗi ticket: tên + thời gian, câu hỏi tối đa 2 dòng, dòng meta xuống dòng được; không chữ nào bị cắt ngang. Given 375 px Then chỉ thấy danh sách; bấm ticket D3 → chi tiết toàn bề rộng, URL có `?ticket=`, có `← Hộp thư` (≥ 44 px); `←` hoặc Back của trình duyệt về danh sách đúng vị trí cuộn.
  Kiểm: `AUDIT` ở 1440 và 375 trên `/inbox` → `cut: []`; tay ở 375.
- AC10 (#17d). Given Giảng viên mở một ticket ở `/inbox` 1440 px When nhìn panel chi tiết Then ô "Trả lời của bạn" cách khối thông tin ngay trên ≤ 24 px, `Gửi trả lời` ngay dưới ô soạn; không khoảng trống kéo giãn.
  Kiểm (Console): `const q = document.querySelector('[data-part=inbox-reply]'); q.getBoundingClientRect().top - q.previousElementSibling.getBoundingClientRect().bottom` ≤ `24`.
- AC11 (#18). Given sau `Đặt lại dữ liệu demo`, B tạo một thread mới có câu AI khớp mẫu (01-AC4) When đổi vai sang Giảng viên Then "Hôm nay" có việc "Câu hỏi mới: «<tiêu đề>» · <chủ đề> · vừa xong" nhãn `Chờ xác nhận`, tiêu đề thành "7 việc cần xử lý hôm nay", chuông có "Câu hỏi mới trong Threads: «<tiêu đề>»"; bấm việc → thread; `Xác nhận` → việc rời "Hôm nay", về "6 việc". Thread nhánh không khớp (01-AC14) hiện nhãn `Cần giảng viên trả lời`, GV gửi phản hồi → việc rời. Trợ giảng thấy giống Giảng viên.
  Kiểm: tay theo thứ tự trên, một trình duyệt.
- AC12 (#19 E3, FR-X17). Given GV ở lớp 761987 (hoặc "Tất cả lớp của tôi") When bấm `Duyệt` ở thẻ "3 yêu cầu vào lớp chờ duyệt · 761988" Then mở `/class/members?tab=pending` (tham số `course` đã bỏ khỏi URL), thanh trên và bộ chọn lớp chuyển sang 761988, tab "Chờ duyệt" mở sẵn với 3 yêu cầu (+ D nếu đã gửi); When bấm `Mở lớp` và từng bước "Thiết lập lớp mới — 761988" Then mở `/class/members` · `/gradebook/scheme` · `/calendar` · `/documents` **của lớp 761988**. Bảng liên kết: `SRS.md` 4.9.
  Kiểm: tay; Console sau khi bấm: `document.cookie.includes('ep_demo_course=int1006-2')` → `true`; `location.search` không còn `course=`.
- AC13 (#19 E9). Given GV ở `/` When bấm `Trả lời` ở thẻ "Câu hỏi của Lý Gia Thảo đã chờ 3 ngày 4 giờ" Then URL `/inbox?ticket=tk-5`; phiếu của Thảo được chọn sẵn, danh sách cuộn tới hàng đó (hàng nằm trong khung nhìn), panel chi tiết hiện câu "Nhóm em muốn phân tích vụ lộ dữ liệu…"; ở 375 px mở thẳng chi tiết có `← Hộp thư`.
  Kiểm (Console): `document.querySelector('[data-part=inbox-detail]').innerText.includes('Nhóm em muốn phân tích')` → `true`; tay ở 375 px.
- AC14 (#19 E19, FR-X13). Given GV ở mọi route Then badge Hộp thư = 5 và Chấm bài = 4 (`SRS.md` 4.8 N9), cả ở sidebar và thanh dưới. When `Nhận` một phiếu Then Hộp thư thành 4 ngay, không tải lại; When `Duyệt bài` của B Then Chấm bài thành 3 và thẻ Hôm nay ghi "3 bài Bài tập 03 cần xem kỹ", chip lọc cùng số; badge ẩn khi về 0.
  Kiểm (Console): `[...document.querySelectorAll('[data-part^=nav-badge]')].map(e => e.dataset.part + ':' + e.textContent)` → `["nav-badge-inbox:5","nav-badge-grading:4"]`, sau hai thao tác → `…inbox:4`, `…grading:3`; `grep -n 'badge:' frontend/src/shared/shell/nav.ts` → không còn số cứng.
- AC15 (#19 E7, E8). Given GV ở `/attendance` buổi 10 ở 390 px và 375 px When nhìn Then: ô "Buổi" rộng ≥ 160 px hiện "Buổi 10 · 29/10"; công tắc "Giả lập mất mạng" ở hàng riêng, không đè lên ô nào; dòng tóm tắt ("30 có mặt · 0 muộn · …") xuống dòng giữa các cụm, không ngắt trong cụm; mỗi SV là hàng 2 dòng — dòng 1 tên (rộng ≥ 140 px, tối đa 2 dòng) và nút `Phát biểu` ≥ 44 × 44; dòng 2 bốn nút "Có mặt · Muộn · Vắng phép · Vắng" mỗi nút ≥ 44 × 44, đủ trong một chiều rộng màn, không cuộn ngang; nút đang chọn có dấu tick + chữ đậm (không chỉ màu). Đủ 4 trạng thái + Phát biểu cho một SV trong **một** màn hình.
  Kiểm: `AUDIT` ở 375 và 390 → `{ ox: 0, cut: [], ell: [] }`; `TOUCH` → `[]`; Console: `document.querySelector('[data-part=att-session-select]').getBoundingClientRect().width` ≥ `160`; tay: bấm giờ ≤ 60 s như 02-AC3 bằng chạm.
- AC16 (#19 E18). Given `/attendance` When bật "Giả lập mất mạng", đánh vắng 1 SV, tắt công tắc Then "Đã lưu hh:mm" và hết dòng "Đang chờ mạng". When `Lưu điểm danh` Then "Đã hoàn tất buổi 10 · …" với số đúng thực tế; nút chuyển "Đã lưu" (vô hiệu, màu trung tính, không đỏ) tới khi có thay đổi mới; dòng trạng thái không bao giờ ghi "· 0 thay đổi" (n = 0 thì bỏ cụm; công tắc bật + 0 thay đổi → "Đang giả lập mất mạng").
  Kiểm: tay theo thứ tự trên; Console ở mọi bước: `document.body.innerText.includes('· 0 thay đổi')` → `false`.
- AC17 (#19 E16). Given GV ở `/threads/t-cbc` When `Chỉnh sửa` Then `Lưu và xác nhận` chỉ bật khi nội dung (sau trim) khác rỗng **và** khác bản đang hiển thị; ô rỗng → nút khoá + "Nội dung không được để trống"; nội dung không đổi → nút khoá + "Chưa có thay đổi — dùng Xác nhận nếu nội dung đã đúng". Nhãn "Đã được giảng viên sửa & xác nhận" và "Xem câu trả lời AI gốc" chỉ xuất hiện khi nội dung đã khác bản AI gốc.
  Kiểm (Console): `[...document.querySelectorAll('button')].find(b => b.textContent.includes('Lưu và xác nhận')).disabled` → `true` ở hai trạng thái (rỗng, không đổi); tay: sửa một chữ → nút bật → lưu → nhãn "sửa & xác nhận".
- AC18 (#19 E12, FR-X13). Given GV ở lớp 761987 When mở `/students` và `/` Then cùng một tập N "Cần chú ý" (`SRS.md` 4.8 N3): số trên chip `Cần chú ý` = số hàng sau lọc = số hàng cột "Rủi ro" ghi "Cần chú ý" (N = 8); cột Rủi ro không còn "Theo dõi" / "Không" (dùng "–"); Hôm nay ghi "Lớp cần chú ý · 8 sinh viên", 3 người đầu theo thứ tự N3, mỗi người cũng "Cần chú ý" ở `/students`, kèm `Xem cả 8`.
  Kiểm: `visible teacher /students | grep -c 'Theo dõi'` → `0`; Console ở `/students` sau khi bấm chip: `document.querySelectorAll('[data-part=student-row]').length` = số trên chip = `[...document.querySelectorAll('[data-part=student-row]')].filter(r => r.innerText.includes('Cần chú ý')).length`; tay: đối chiếu 3 người ở Hôm nay.
- AC19 (#19 E28, FR-X13). Given GV ở lớp 761987 When mở `/students`, `/attendance`, `/gradebook`, `/class/members` Then danh sách sinh viên cùng một thứ tự: `sv-n` tăng dần, ba người đầu Nguyễn Minh Trung, Trần Thu Uyên, Lê Quang Huy.
  Kiểm (Console, mỗi trang): `[...document.querySelectorAll('[data-part=student-row]')].map(r => r.dataset.studentId).join()` → chuỗi giống hệt nhau ở bốn trang và tăng dần theo `n`.
- AC20 (#19 E29, FR-X13). Given GV ở `/` lúc mới vào Then thẻ "4 bài Bài tập 03 cần xem kỹ" có dòng phụ đúng 4 lý do thật: "1 lệch hai lượt chấm · 1 bài ngắn bất thường · 1 trùng đoạn với bài khác · 1 AI không chắc ở một tiêu chí"; hàng chờ `/grading` ghi đúng lý do từng bài (B: "Hai lượt chấm lệch 1,5 điểm"). Sau `Duyệt bài` B thẻ thành "3 bài…" và dòng phụ không còn "lệch hai lượt chấm".
  Kiểm: `for k in '1 lệch hai lượt chấm' '1 bài ngắn bất thường' '1 trùng đoạn với bài khác' '1 AI không chắc ở một tiêu chí'; do visible teacher / | grep -c "$k"; done` → mỗi lần ≥ `1`; tay: duyệt B rồi đọc lại.
- AC21 (#19, PM chốt `SRS.md` 4.3.1 J3). Given GV ở `/` lúc mới vào (6 việc) When B tạo thread thứ nhất có câu AI khớp mẫu (01-AC4) rồi GV mở `/` Then "7 việc cần xử lý hôm nay"; thẻ "Câu hỏi mới: «<tiêu đề>»" nằm **ngay trên** thẻ "1 câu trả lời của AI chờ bạn xác nhận" (thẻ này vẫn ghi 1, của `t-cbc`, không đếm thread mới), dưới thẻ "4 bài Bài tập 03 cần xem kỹ". When B tạo thread thứ hai Then "8 việc", thread thứ hai đứng trên thread thứ nhất. When GV `Xác nhận` câu AI của thread thứ hai Then "7 việc", thẻ của thread thứ hai biến mất, thread thứ nhất còn nguyên. Given thread nhánh không khớp (01-AC14) Then thẻ ghi `Cần giảng viên trả lời` và chỉ rời khi GV / TA gửi phản hồi.
  Kiểm: tay theo thứ tự trên (Đặt lại dữ liệu trước); Console: `[...document.querySelectorAll('[data-part=today-task]')].map(e => e.dataset.taskId)` → thứ tự đúng (ticket, điểm danh, review, `thread-new-2`, `thread-new-1`, `ai-pending`, setup, members) — dev thêm `data-part="today-task"` + `data-task-id` cho thẻ Hôm nay.

### Ngoài phạm vi
Tạo lịch buổi học hàng loạt (chỉ có bộ chọn buổi), `/class/settings`, nhận xét tổng hợp AI ở hồ sơ 360.

### Phụ thuộc
US-PROTO-00; dữ liệu D3 từ US-PROTO-01.

## US-PROTO-03: Giảng viên muốn xem sổ điểm, xác nhận công thức, duyệt và công bố bài chấm, duyệt câu hỏi, quản lý tài liệu để thấy "AI chỉ nháp, người quyết"
Ưu tiên: Must · Ước lượng: L · Sprint: 1 (bổ sung)

Route: `/gradebook`, `/gradebook/scheme`, `/grading`, `/grading/[submissionId]`, `/questions`, `/documents` — `SRS.md` 4.5.

### Tiêu chí nghiệm thu
- AC1. Given Giảng viên When mở từng route Then mở được, khung nhìn đầu đúng §14.10–14.14, 14.17. Riêng `/grading/[submissionId]` phân định rõ thành 2 panel độc lập (Proposal #16): Panel xem bài nộp sinh viên (bên trái, 52–58%) và Panel rubric / điểm số (bên phải, 42–48%), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc, padding rõ ràng và cuộn độc lập.
  Kiểm: `for r in /gradebook /gradebook/scheme /grading /grading/sub-bt03-sv-2 /questions /documents; do open_as teacher $r; done` → đều `MO`.
- AC2. Given `/grading` lọc mặc định When mở bài B → thấy thông báo vàng "Hai lượt chấm lệch 1,5 điểm" ở tiêu chí 2 → sửa tiêu chí 2 thành 2,5 → `Duyệt bài` → về hàng chờ → `Công bố` Then tổng các tiêu chí 8,5, trừ nộp muộn 0,5 → công bố 8,0; `/assignments/bt03` của B có điểm 8,0 và nhận xét; sổ điểm có BT03; QT của B = 8,7.
  Kiểm: tay theo `DEMO_SCRIPT.md` 08:00–09:35.
- AC3. Given lớp 2 When mở `/gradebook` Then banner công thức chưa xác nhận, `Chốt điểm` khoá có lý do; `/gradebook/scheme` → điền D5 → `Xác nhận công thức` (qua hộp xác nhận) → về `/gradebook` lớp 2 hết banner.
  Kiểm: tay theo `DEMO_SCRIPT.md` 10:00–11:15.
- AC4. Given lớp 1 `/gradebook` When mở giải trình hàng B Then phép tính tuyến tính khớp `SRS.md` 4.1; `Xuất XLSX` trong menu cho dòng xác nhận mô phỏng; có dòng "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường".
  Kiểm: tay.
- AC5 (nhánh lỗi). Given `/documents` When tải "scan-khong-co-chu.pdf" Then trạng thái FAILED với lý do đọc hiểu được; Given ô mẫu xung đột ở `/gradebook` When sửa Then dòng 409 hỏi giữ bên nào; Given `Chốt điểm` lớp 1 khi chưa có CK Then hộp xác nhận nêu số và không cho chốt.
  Kiểm: tay.
- AC6 (phân quyền). Given TA When ở `/grading` Then `Duyệt bài` được, không có `Công bố` (thấy "Chỉ giảng viên công bố điểm"); ở `/gradebook/scheme` không có `Xác nhận công thức`; ở `/gradebook` không sửa được ô, không `Chốt điểm`. Given Sinh viên / Admin When mở route của US Then màn chặn quyền.
  Kiểm: `visible ta /grading | grep -c 'Chỉ giảng viên công bố điểm'` ≥ 1; `visible ta /gradebook/scheme | grep -c 'Xác nhận công thức'` → `0`; `for v in student admin; do for r in /gradebook /gradebook/scheme /grading /questions /documents; do open_as $v $r sv-2; done; done` → đều `CHAN`.
- AC7 (#19 E21). Given GV ở `/gradebook` 390 px Then cột tên (dính trái), "QT tạm tính" và "Trạng thái" đều trong khung nhìn đầu không phải cuộn ngang; các cột còn lại (BT01–BT03, cộng / trừ, Cuối kỳ) cuộn ngang, có dòng gợi ý "Kéo ngang để xem BT01–BT03 và Cuối kỳ" phía trên bảng và mép phải bảng lộ một phần cột kế. Given `/students`, `/grading`, `/documents`, `/admin/courses`, `/admin/users` ở 390 px Then dạng danh sách mà không cột nào biến mất: số cặp "Nhãn: giá trị" ở dòng phụ = số cột − 1 (`/documents`: loại · tuần · Dùng cho AI · Hiện cho SV · trạng thái · ngày). Given `/students/sv-3` 390 px Then thanh tab cuộn ngang, mép tab kế lộ ra, "Ghi chú" tới được và tự cuộn vào khung khi chọn.
  Kiểm: `AUDIT` ở 390 trên các route trên → `cut: []`; Console ở `/gradebook`: `['col-qt','col-status'].map(p => document.querySelector('[data-part=' + p + ']').getBoundingClientRect().right <= innerWidth)` → `[true, true]`; tay: vuốt tab, ảnh 390 px.
- AC8 (#19 E10, E33; #20a). Given GV ở `/documents` Then 12 dòng đúng bảng `SRS.md` 4.8 N5 (tên hiển thị, loại, tuần, ngày); không tên tệp dạng gạch dưới và không chữ "Mordern" ngoài Drawer chi tiết; tên dài xuống dòng, không ngắt giữa từ; mọi tên của 10 tài liệu SV thấy trùng chữ với `/library`; 2 `ANSWER_KEY` ghi "Không hiển thị cho sinh viên"; tài liệu `PROCESSING` hiện **"Chờ xử lý"** ở "Dùng cho AI" ("Chờ xác nhận" dành riêng cho câu trả lời AI) và cờ "Hiện cho SV" khoá.
  Kiểm: `visible teacher /documents | grep -c 'Chương 3 — Mật mã đối xứng và chế độ vận hành'` ≥ `1`; `… | grep -cE 'Chuong[0-9]_|Mordern'` → `0`; `… | grep -c 'Hạ tầng khoá công khai'` → `0`; tay: tải lên một file mẫu, trong lúc "Đang xử lý" cờ Dùng cho AI ghi "Chờ xử lý" (không "Có"); so tên với `/library`.

### Ngoài phạm vi
Nhập XLSX điểm, mở khoá sau chốt, phúc khảo phía giảng viên, khớp tay bài nộp chưa khớp.

### Phụ thuộc
US-PROTO-00; `/assignments/bt03` từ US-PROTO-01.

## US-PROTO-04: Giảng viên và Admin muốn xem lỗ hổng kiến thức, số liệu lớp, quan sát AI, cấu hình và quản trị lớp để thấy phần "hiểu lớp + hệ thống"
Ưu tiên: Must · Ước lượng: M · Sprint: 1 (bổ sung)

Route: `/insights`, `/analytics`, `/observability`, `/settings/llm`, `/settings/integrations`, `/` (Admin), `/admin/courses`, `/admin/users` — `SRS.md` 4.6.

### Tiêu chí nghiệm thu
- AC1. Given Giảng viên When mở `/insights`, `/analytics`, `/observability`, `/settings/llm`, `/settings/integrations` Then mở được; Given Admin When mở `/`, `/observability`, `/settings/llm`, `/settings/integrations`, `/admin/courses`, `/admin/users` Then mở được; khung nhìn đầu đúng §14.21–14.25 / INTEGRATION mục 2.
  Kiểm: `for r in /insights /analytics /observability /settings/llm /settings/integrations; do open_as teacher $r; done; for r in / /observability /settings/llm /settings/integrations /admin/courses /admin/users; do open_as admin $r; done` → đều `MO`.
- AC2. Given `/insights` lớp 1 When `Tạo báo cáo mới` Then có tiến độ rồi danh sách chủ đề A, B đầu; câu mẫu không có tên / MSSV; chủ đề dưới 3 sinh viên không hiện câu mẫu; `Tạo thread ghim` → thread ghim mới ở `/threads`; đổi lớp 2 → chủ đề C đầu.
  Kiểm: tay theo `DEMO_SCRIPT.md` 12:00–13:05; `visible teacher /insights | grep -cE '2022[0-9]{4}|Trần Thu Uyên|Lê Quang Huy'` → `0`.
- AC3. Given Admin ở `/observability` When mở một yêu cầu Then phải nhập lý do trước; nội dung đã che chỉ có `[[SV_1]]`, không tên thật; có dòng "Đã ghi nhật ký kiểm toán". Given Giảng viên Then chỉ thấy dải trạng thái + số tổng hợp, hàng không mở được.
  Kiểm: tay; `visible teacher /observability | grep -c '\[\[SV_'` → `0`.
- AC4. Given Admin ở `/settings/llm` When `Test kết nối` và đổi model tác vụ CHAT Then kết quả tại chỗ + dòng Hoàn tác; khoá API chỉ hiện dạng che. Given Giảng viên Then cùng màn nhưng không có nút sửa / test.
  Kiểm: tay; `visible teacher /settings/llm | grep -c 'Test kết nối'` → `0`.
- AC5 (nhánh lỗi). Given `/settings/llm` When test provider lỗi mẫu Then lỗi nói rõ vấn đề + cách khắc phục; Given `/insights?state=empty` Then rỗng + `Tạo báo cáo mới`.
  Kiểm: tay.
- AC6 (phân quyền). Given Sinh viên / TA When mở `/observability`, `/settings/llm`, `/settings/integrations`, `/admin/courses`, `/admin/users` Then màn chặn quyền; Given Giảng viên When mở `/admin/courses` Then màn chặn quyền; Given TA ở `/analytics` Then không có mục chi phí (Q2).
  Kiểm: `for v in student ta; do for r in /observability /settings/llm /settings/integrations /admin/courses /admin/users; do open_as $v $r sv-2; done; done; open_as teacher /admin/courses` → đều `CHAN`; `visible ta /analytics | grep -ci 'chi phí'` → `0`.
- AC7 (#17g). Given Admin ở `/settings/llm` 1440 px When nhìn danh sách nhà cung cấp Then cột trạng thái và cột nút thẳng hàng ở mọi hàng (lưới cột cố định). Given 390 px Then mỗi hàng xếp dọc: thông tin → trạng thái → nút rộng 100 % (≥ 44 px); dòng hạn mức xuống dòng, không `…`.
  Kiểm (Console, 1440): `['provider-status','provider-action'].map(p => new Set([...document.querySelectorAll('[data-part=' + p + ']')].map(e => Math.round(e.getBoundingClientRect().left))).size)` → `[1, 1]`; `AUDIT` ở 390 → `ell: []`.
- AC8 (#19 E6, E25, FR-X13). Given GV ở `/analytics` lớp 761987 (khoảng 7 ngày) Then "Câu chờ quá 24 giờ" = 3 = số hàng có nhãn "Quá 24 giờ" ở `/inbox` (số này là ảnh chụp hiện tại, không đổi khi chuyển 7 / 30 ngày); "AI tự trả lời 98% trong 392 câu hỏi; 6 câu chuyển sang giảng viên" (6/392 → 98%); khoảng 30 ngày: 99% trong 1.424 câu, 19 chuyển (`SRS.md` 4.8 N2). When SV B hỏi D3 Then "Câu đã chuyển giảng viên" 7 ngày thành 7 và tỉ lệ vẫn tính bằng công thức; When GV `Nhận` phiếu quá 24 giờ Then "Câu chờ quá 24 giờ" ở **cả hai** màn thành 2.
  Kiểm: `visible teacher /inbox | grep -c 'Quá 24 giờ'` → `3`; `visible teacher /analytics | grep -c 'AI tự trả lời 98%'` ≥ `1`; tay: chuyển 30 ngày (99%), `Nhận` phiếu rồi đối chiếu hai màn.
- AC9 (#19 E17). Given GV ở `/insights` lớp 761987 When `Tạo thread ghim` ở chủ đề 1 Then nút đổi thành "Đã ghim · Xem thread" (liên kết `/threads/<id>`), không tạo thêm khi bấm lại hay khi quay lại màn từ nơi khác; `/threads` tăng đúng +1 (12 → 13), đúng một thread ghim cho chủ đề đó; `Đặt lại dữ liệu demo` → về 12.
  Kiểm (Console ở `/threads` sau hai lần thử bấm): `document.querySelectorAll('[data-part=thread-row]').length` → `13`; tay: đặt lại dữ liệu → `12`.
- AC10 (#19 E30, FR-X13). Given Admin / GV When đọc mốc phân công lớp 761988 Then cùng một mốc "hôm qua 16:40" ở Admin `/` ("Phân công · hôm qua 16:40"), `/admin/courses` ("Mới mở · hôm qua 16:40") và chuông GV; không còn "08:30" hay "Hôm qua 16:40" ở chỗ khác mốc.
  Kiểm: `for r in / /admin/courses; do visible admin $r | grep -ci 'hôm qua 16:40'; done` → mỗi lần ≥ `1`; `visible admin / | grep -c '08:30'` → `0`; tay: chuông GV.
- AC11 (#19 E35). Given Admin ở `/settings/llm` 1440 px When đo khoảng cách từ đáy mỗi phần tới đỉnh phần kế (Nhà cung cấp, Tác vụ → model, Mô hình embedding, Ngân sách) Then bằng nhau ở mọi cặp (lệch 0 px); bảng embedding không có hàng trống ở đầu.
  Kiểm (Console): `[...document.querySelectorAll('[data-part=settings-section]')].map((s, i, a) => i ? Math.round(s.getBoundingClientRect().top - a[i - 1].getBoundingClientRect().bottom) : null).slice(1)` → mọi giá trị bằng nhau; tay: ảnh cuộn xuống cuối.
- AC12 (#19 E37). Given GV ở `/analytics` "Hoạt động học" Then biểu đồ có trục giá trị ≥ 3 mốc có số (0, nửa max, max), điểm cuối có nhãn số; mỗi điểm có nhãn đọc được ("T5 29/10: 47 câu") hiện khi rê chuột và khi focus bằng bàn phím; kèm dòng "Cao nhất: … câu · Thấp nhất: … câu". 7 ngày = 7 điểm; 30 ngày = 10 cụm 3 ngày.
  Kiểm (Console): `[document.querySelectorAll('[data-part=chart-point]').length, document.querySelectorAll('[data-part=chart-axis-label]').length]` → `[7, ≥3]` (khoảng 7 ngày), `[10, ≥3]` (30 ngày); `[...document.querySelectorAll('[data-part=chart-point]')].every(p => p.getAttribute('aria-label'))` → `true`.

### Ngoài phạm vi
`/admin/audit`, `/admin/health`, trace Jaeger thật, ngân sách theo lớp.

### Phụ thuộc
US-PROTO-00; thread ghim hiện ở `/threads` của US-PROTO-01.
