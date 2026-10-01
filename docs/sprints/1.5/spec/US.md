# FEAT-prototype-ui Prototype giao diện toàn bộ tính năng
Nguồn: PRD §3–§4 (M0–M14), FLOWS F2–F17, `docs/DEMO_SCRIPT.md`, `docs/sprints/1.5/plan.md`; hợp đồng giao diện `docs/design/DESIGN.md` §14, `docs/design/INTEGRATION.md` mục 2. Chi tiết dữ liệu và tương tác từng route: `SRS.md` mục 4.
Phiên bản 5 · 2026-10-01 · v5 thêm UI polish (#17 → 00-AC7…AC10, 01-AC11, 02-AC9, 02-AC10, 04-AC7) và Threads như thật (#18 → 01-AC4, 01-AC10, 01-AC12…AC14, 02-AC11). Lịch sử phiên bản: `SRS.md` dòng đầu.

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

Móc đo dev thêm (`data-part`): `brand`, `chat-history`, `chat-thread`, `chat-composer`, `inbox-list`, `inbox-detail`, `inbox-reply`, `provider-status`, `provider-action`.

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
- AC6. Given điện thoại 375 px When dùng mọi route của US Then không cuộn ngang, vùng chạm ≥ 44 px, bottom nav ≤ 5 đích, lịch sử chat ẩn.
  Kiểm: DevTools 375 × 812, đi lần lượt các route; ảnh chụp đính vào báo cáo QC.
- AC7. Given B ở `/me` When nhập CK = 11 vào What-if Then báo lỗi tại ô. Sau bước điểm danh của GV, QT của B = 8,5; sau công bố BT03 (8,0), QT = 8,7 và What-if CK = 8,0 hiện 8,3 (`SRS.md` 4.1).
  Kiểm: tay.
- AC8 (nhánh lỗi). Given Sinh viên D When nhập mã `ABCDEFG` ở `/join` Then thấy câu chung "Mã không hợp lệ hoặc đã hết hạn…", không lộ lý do; nhập `BX4P9TW` → xem trước lớp 761988 → `Tham gia lớp` → "chờ giảng viên duyệt". Và: `?state=error` trên `/chat` giữ nguyên chữ trong composer.
  Kiểm: tay; `visible student /join/BX4P9TW sv-4 | grep -c 761988` ≥ 1.
- AC9 (phân quyền). Given vai TA / GV / Admin When mở `/chat`, `/me`, `/practice`, `/library`, `/assignments/bt03`, `/join` Then màn chặn quyền; Admin cũng bị chặn ở `/threads`, `/calendar`.
  Kiểm: `for v in ta teacher admin; do for r in /chat /me /practice /library /assignments/bt03 /join; do open_as $v $r; done; done` → đều `CHAN`; `open_as admin /threads; open_as admin /calendar` → `CHAN`.
- AC10 (#18). Given B ở `/threads` When đọc danh sách Then số phản hồi mỗi thread đúng cột "n" ở `SRS.md` 4.3.1 B (`t-cbc` 4, `t-salt` 3, `t-sqli` 3, `t-pin-rubric` 4, …). When mở `/threads/t-cbc` Then thấy từ trên xuống: khối câu hỏi gốc (Đặng Gia An, thời gian, chủ đề, tuần, "4 người tham gia"); khối câu trả lời AI `Chờ xác nhận` mẫu S1 + `Nguồn tham khảo (2)`; **một** vùng "Thảo luận (4)" gồm 4 phản hồi của sv-10, TA (có khối trích bài sv-10), sv-7, sv-19 theo thời gian; ô "Phản hồi của bạn" ở cuối chính vùng đó với `Gửi phản hồi` (chính) và `Hỏi trợ lý AI` (phụ). Không còn tiêu đề "Thảo luận của lớp" hay "Phản hồi trong thread này".
  Kiểm: `visible student /threads/t-cbc sv-2 | grep -cE 'Thảo luận của lớp|Phản hồi trong thread này'` → `0`; `visible student /threads/t-cbc sv-2 | grep -c 'Thảo luận (4)'` ≥ 1; tay: đếm bài từng thread trong danh sách so bảng 4.3.1 B.
- AC11 (#17e). Given B ở `/chat` 1440 px When phiên trống và khi phiên đã có ≥ 6 tin Then lịch sử phiên là một panel (viền, nền, bo góc FR-X11) rộng 240 px; hội thoại và composer cùng mép trái và cùng bề rộng; composer ghim đáy (cách đáy màn ≤ 24 px), hội thoại cuộn bên trong. Given 375 px Then lịch sử ẩn (AC6).
  Kiểm (Console): `const r = p => document.querySelector('[data-part=' + p + ']').getBoundingClientRect(); [r('chat-thread').left === r('chat-composer').left, r('chat-thread').width === r('chat-composer').width, innerHeight - r('chat-composer').bottom <= 24]` → `[true, true, true]` ở cả hai trạng thái.
- AC12 (#18). Given B ở `/threads/t-cbc` When ô soạn trống và bấm `Hỏi trợ lý AI` Then AI đăng `Gợi ý thêm` của S1 ("…bao nhiêu khối bản rõ bị ảnh hưởng?") theo trình tự 4.3.1 E. When gõ "Còn CTR thì sao, có cần IV ngẫu nhiên không?" rồi bấm `Hỏi trợ lý AI` Then phản hồi của B hiện trước, AI trả lời ngay dưới theo S2 với dòng "↳ trả lời Trần Thu Uyên". When gửi phản hồi "@AI dùng lại nonce có sao không" bằng `Gửi phản hồi` Then xử lý như bấm `Hỏi trợ lý AI`. Trong lúc AI soạn, hai nút khoá; bấm `Dừng` giữ phần đã hiện + "Đã dừng" + `Hỏi lại`.
  Kiểm: tay, bấm giờ.
- AC13 (#18). Given B ở `/threads/t-salt` When gửi phản hồi "Em hiểu rồi ạ" Then phản hồi hiện ngay cuối vùng thảo luận và trang cuộn tới; ~2 s sau "Phạm Quốc Bảo đang trả lời…"; ~6 s sau phản hồi của TA có khối trích bài của B, nội dung cột "Phản hồi trễ của TA" mẫu H1; chuông có chấm chưa đọc "Phạm Quốc Bảo đã trả lời trong «Vì sao cần muối (salt) khi băm mật khẩu?»", bấm → về đúng phản hồi; `/threads` hiện `t-salt` 5 phản hồi. Given gửi xong chuyển ngay sang `/calendar` (trước 6 s) Then tới hạn chuông vẫn có thông báo, quay lại thread thấy phản hồi TA. Gửi phản hồi thứ hai trong cùng thread → không có phản hồi trễ nữa.
  Kiểm: tay, bấm giờ; tải lại trang ở giây thứ 3 rồi chờ.
- AC14 (nhánh lỗi, #18). Given B tạo thread "WPA3 chặn được KRACK không?" chủ đề "An toàn mạng không dây" (nhờ AI bật) Then AI hiện ngay toàn văn "Mình chưa đủ chắc chắn để gợi ý câu này từ tài liệu của lớp…", không nguồn, nhãn `Đang chờ giảng viên`, không có `Chờ xác nhận`; `/inbox` của GV không thêm ticket (Q4). Given ô soạn trống Then `Gửi phản hồi` khoá. Given gõ dở phản hồi rồi rời trang và quay lại Then chữ còn nguyên. Given phản hồi chứa `20229002` Then Dialog hai lối (như AC4).
  Kiểm: tay; đổi vai GV → `/inbox` số ticket không đổi.

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

### Ngoài phạm vi
`/admin/audit`, `/admin/health`, trace Jaeger thật, ngân sách theo lớp.

### Phụ thuộc
US-PROTO-00; thread ghim hiện ở `/threads` của US-PROTO-01.
