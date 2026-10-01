# FEAT-prototype-ui Prototype giao diện toàn bộ tính năng
Nguồn: PRD §3–§4 (M0–M14), FLOWS F2–F17, `docs/DEMO_SCRIPT.md`, `docs/sprints/1/prototype/plan.md`; hợp đồng giao diện `docs/design/DESIGN.md` §14, `docs/design/INTEGRATION.md` mục 2. Chi tiết dữ liệu và tương tác từng route: `SRS.md` mục 4.

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
- AC4. Given B soạn bài ở `/threads` có MSSV `20229002` When bấm đăng Then mở Dialog đúng hai lối `Chuyển sang chat riêng` / `Ẩn thông tin rồi đăng`; chọn lối 1 → `/chat` có sẵn bản nháp, không mất chữ.
  Kiểm: tay.
- AC5. Given vai Sinh viên When đọc văn bản hiển thị mọi route của US Then không có từ kỹ thuật AI, không số độ tin cậy, không điểm nháp, không nhãn rủi ro / ghi chú về chính mình.
  Kiểm: `for r in / /chat /threads /threads/t-cbc /practice /library /calendar /me /assignments/bt03; do visible student $r sv-2; done | grep -inE 'RAG|PII|fallback|trace|provider|confidence|redaction|độ tin cậy|cần chú ý|rủi ro'` → không in gì. Tay: `/assignments/bt03` trước khi GV công bố không có con số điểm.
- AC6. Given điện thoại 375 px When dùng mọi route của US Then không cuộn ngang, vùng chạm ≥ 44 px, bottom nav ≤ 5 đích, lịch sử chat ẩn.
  Kiểm: DevTools 375 × 812, đi lần lượt các route; ảnh chụp đính vào báo cáo QC.
- AC7. Given B ở `/me` When nhập CK = 8,0 vào What-if Then hiện 8,3; nhập 11 → báo lỗi tại ô. Sau bước điểm danh của GV, QT của B = 8,5; sau công bố BT03 = 8,8 (`SRS.md` 4.1).
  Kiểm: tay.
- AC8 (nhánh lỗi). Given Sinh viên D When nhập mã `ABCDEFG` ở `/join` Then thấy câu chung "Mã không hợp lệ hoặc đã hết hạn…", không lộ lý do; nhập `BX4P9TW` → xem trước lớp 761988 → `Tham gia lớp` → "chờ giảng viên duyệt". Và: `?state=error` trên `/chat` giữ nguyên chữ trong composer.
  Kiểm: tay; `visible student /join/BX4P9TW sv-4 | grep -c 761988` ≥ 1.
- AC9 (phân quyền). Given vai TA / GV / Admin When mở `/chat`, `/me`, `/practice`, `/library`, `/assignments/bt03`, `/join` Then màn chặn quyền; Admin cũng bị chặn ở `/threads`, `/calendar`.
  Kiểm: `for v in ta teacher admin; do for r in /chat /me /practice /library /assignments/bt03 /join; do open_as $v $r; done; done` → đều `CHAN`; `open_as admin /threads; open_as admin /calendar` → `CHAN`.

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
- AC3. Given `/attendance` buổi 5 lớp 1 When dùng bàn phím (↑/↓, `1`–`4`, `P`) đánh 2 vắng, 1 muộn, +phát biểu cho B rồi `Lưu điểm danh` Then mỗi thao tác hiện dòng Hoàn tác 5 s và trạng thái lưu; kết thúc "Đã hoàn tất buổi 5 · 27 có mặt, 1 muộn, 2 vắng"; `/me` của B lên 8,5. Toàn bộ ≤ 60 s.
  Kiểm: tay, bấm giờ; lặp lại ở 375 px bằng chạm.
- AC4. Given Sinh viên D đã gửi yêu cầu vào lớp 761988 When GV mở `/class/members` lớp 2 và `Duyệt` D Then D thành thành viên; đổi vai về D → bộ chọn lớp có 761988.
  Kiểm: tay theo `DEMO_SCRIPT.md` 01:10–01:55.
- AC5. Given điện thoại 375 px When dùng `/inbox` và `/attendance` Then dùng được: `/inbox` là danh sách → chi tiết; `/attendance` hàng có kẻ, không card, vùng chạm ≥ 44 px.
  Kiểm: DevTools 375 × 812; ảnh chụp.
- AC6 (nhánh lỗi). Given `/attendance` When bật "Giả lập mất mạng" và đổi 2 ô Then thấy "Đang chờ mạng · 2 thay đổi"; tắt công tắc → "Đã lưu …", không mất thay đổi. Given `/students?state=error` Then lỗi có `Thử lại`.
  Kiểm: tay.
- AC7 (phân quyền). Given Sinh viên / Admin When mở các route của US Then màn chặn quyền. Given TA When mở `/class/members` Then xem và `Duyệt` được, không có `Tạo lại mã`, không có `Mời ra khỏi lớp`.
  Kiểm: `for v in student admin; do for r in /inbox /students /students/sv-3 /attendance /class/members; do open_as $v $r sv-2; done; done` → đều `CHAN`; `visible ta /class/members | grep -c 'Tạo lại mã'` → `0`.

### Ngoài phạm vi
Tạo lịch buổi học hàng loạt (chỉ có bộ chọn buổi), `/class/settings`, nhận xét tổng hợp AI ở hồ sơ 360.

### Phụ thuộc
US-PROTO-00; dữ liệu D3 từ US-PROTO-01.

## US-PROTO-03: Giảng viên muốn xem sổ điểm, xác nhận công thức, duyệt và công bố bài chấm, duyệt câu hỏi, quản lý tài liệu để thấy "AI chỉ nháp, người quyết"
Ưu tiên: Must · Ước lượng: L · Sprint: 1 (bổ sung)

Route: `/gradebook`, `/gradebook/scheme`, `/grading`, `/grading/[submissionId]`, `/questions`, `/documents` — `SRS.md` 4.5.

### Tiêu chí nghiệm thu
- AC1. Given Giảng viên When mở từng route Then mở được, khung nhìn đầu đúng §14.10–14.14, 14.17.
  Kiểm: `for r in /gradebook /gradebook/scheme /grading /grading/sub-bt03-sv-2 /questions /documents; do open_as teacher $r; done` → đều `MO`.
- AC2. Given `/grading` lọc mặc định When mở bài B → thấy thông báo vàng "Hai lượt chấm lệch 1,5 điểm" ở tiêu chí 2 → sửa tiêu chí 2 thành 2,5 → `Duyệt bài` → về hàng chờ → `Công bố` Then tổng 8,5 (đã trừ nộp muộn 0,5); `/assignments/bt03` của B có điểm và nhận xét; sổ điểm có BT03; QT của B = 8,8.
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

### Ngoài phạm vi
`/admin/audit`, `/admin/health`, trace Jaeger thật, ngân sách theo lớp.

### Phụ thuộc
US-PROTO-00; thread ghim hiện ở `/threads` của US-PROTO-01.
