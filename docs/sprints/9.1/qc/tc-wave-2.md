# TC đợt 2 — Chữ (US-UX91-01, 05, 06, 07)

Phiên bản 1 · 2026-11-08. Quy ước và kiểm chung: `tc-00-chung.md`.

---
## US-UX91-01 — Xoá câu giải thích dưới tiêu đề

**Bảng chuỗi cần xoá** (cột "Mảnh kiểm" là đoạn đủ ngắn và đủ riêng để `includes` / `rg` khớp; lấy từ cột "Chuỗi hiện tại" của spec, các hàng có hành động **Xoá**). Số # theo bảng của spec.

| # | Màn (vai) | Mảnh kiểm |
| --- | --- | --- |
| 1 | `/admin/courses` (admin) | `lưu trữ lớp đã kết thúc` |
| 2 | `/admin/users` (admin) | `Sinh viên tự đăng ký` |
| 3 | `/admin/courses` → mở biểu mẫu Mở lớp | `nhận thông báo kèm mã tham gia ngay khi lớp mở` |
| 4 | `/admin/users` → mở biểu mẫu Mời | `Quản trị viên không biết` |
| 5 | `/analytics` (GV) | `không phải bảng theo dõi từng sinh viên` |
| 6 | `/analytics` | `Câu hỏi sinh viên gửi trong` |
| 7 | `/analytics` | `thời gian lớp chờ bạn` |
| 8 | `/analytics` | `bị che trước khi câu hỏi đi tới model` |
| 9 | `/analytics` | `AI chỉ ra bản nháp` |
| 10 | `/analytics` | `Tiền trả cho model trong khoảng đang xem` |
| 18 | `/exams/‹id›` → Drawer chọn câu | `Chỉ hiện câu đã duyệt và chưa lưu trữ` |
| 19 | `/exams` (GV) | `soạn từ ngân hàng` |
| 20 | `/exams` (SV) | `Đồng hồ chạy khi bạn bấm Bắt đầu làm bài` |
| 23 | `/insights` | `đã bỏ tên và mã số sinh viên, nhóm theo chủ đề` |
| 24 | `/insights` (đang tạo báo cáo) | `Việc chạy nền, bạn đi làm việc khác được` |
| 25 | `/insights` | `Báo cáo chỉ dùng câu hỏi của` |
| 26 | `/insights` | `xếp theo mức độ cả lớp đang vướng` |
| 27 | `/insights` | `không tài liệu nào của lớp trả lời được` |
| 28 | `/insights` | `Bản trước tạo ngày` |
| 29 | `/join` (SV) | `không phân biệt hoa thường` |
| 32 | `/me` (SV) | dòng phụ chỉ có mã lớp trần (không còn `<p>` ngay sau `h1` chứa `761987`) |
| 33 | `/class/settings` (GV) | `Lớp 761987` ngay dưới h1 |
| 34 | `/class/settings` | `Sinh viên vào bằng mã sẽ chờ bạn duyệt` (chỉ còn trong tooltip) |
| 35 | `/class/members` (GV) | `Duyệt yêu cầu vào lớp và quản lý người học` |
| 36 | `/observability` (GV, admin) | `Nội dung câu hỏi chỉ quản trị viên hệ thống mở được`, `Hệ thống đang chạy thế nào` |
| 37 | `/observability` | `Số tổng hợp của lớp bạn phụ trách` |
| 38 | `/observability` | `Chọn một hàng để xem nội dung đã che`, `nội dung… không mở được từ đây` |
| 42 | `/practice/history` (SV) | `Bấm một lượt để xem lại từng câu` |
| 43 | `/practice/history` | `Tính trên các câu đã làm trong các lượt gần nhất` |
| 44 | `/practice` (SV) | `hoặc thi thử với đề cũ` |
| 45 | `/questions` (GV) | `Chỉ giảng viên và trợ giảng thấy` |
| 59 | khung máy chủ `intro.ts` của `/`, `/inbox`, `/threads`, `/me`, `/gradebook`, `/chat` | `description` của từng route (xem TC-UX01-3) |

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX01-1 | AC1 | theo bảng, 1440 | Mở từng màn ở cột "Màn (vai)" với dữ liệu lớp 761987 (mở biểu mẫu / drawer khi cần); `page.content()` rồi `includes(mảnh kiểm)` | Không mảnh nào còn trên màn | E2E |
| TC-UX01-2 | AC1 | — | `rg -n -F "<mảnh kiểm>" frontend/src --glob '!**/Demo*.tsx'` lần lượt cho cả bảng | 0 kết quả (ngoại lệ: chuỗi chuyển thành tooltip / `meta` / placeholder ở cột "Thay bằng" — ghi riêng) | GREP |
| TC-UX01-3 | AC1, AC3 | khách / SV / GV | Khung máy chủ: `curl -s https://<stack>/<route>` cho `/`, `/inbox`, `/threads`, `/me`, `/gradebook`, `/chat`; lấy văn bản trong khối `PreShell` | Có `h1` (title) đúng; **không** có câu phụ (`description`) cũ; chữ của route Threads là «Hỏi đáp công khai» | E2E |
| TC-UX01-4 | AC1 | — | Đối chiếu bảng "Thay bằng": #3 tooltip nút `Mở lớp`; #4 tooltip nút gửi lời mời; #18 tiêu đề drawer «Chọn câu đã duyệt từ ngân hàng»; #20 tooltip `Bắt đầu làm bài`; #25 `meta` «Đã bỏ tên và MSSV · chủ đề dưới 3 sinh viên không hiện câu mẫu»; #26 `meta` chỉ còn khoảng thời gian; #28 tiêu đề «So với báo cáo ngày {dd/MM}»; #29 placeholder `AN7K2MQ`; #42 `meta` «N lượt gần đây» | Mỗi chuỗi thay thế đúng như spec | E2E |
| TC-UX01-5 | AC2 | theo dòng | Với mọi dòng **Giữ**: #11, 14, 17, 22, 40, 41, 52, 53, 55, 56, 57, 58: mở màn tương ứng | Chữ vẫn hiện (dòng 11, 47, 48: không còn mã lớp ở dòng phụ nhưng còn dữ liệu «buổi 4 · Thứ Hai, 02/11 · 13:30–16:00 · P.405», «24 sinh viên», MSSV) | E2E |
| TC-UX01-6 | AC2 | — | `git diff 1b72f54... --stat -- 'frontend/src/**/Demo*.tsx'` | Không có tệp `Demo*.tsx` bị sửa (spec cấm) | GREP |
| TC-UX01-7 | AC3 | giảng viên | `pnpm -C frontend exec playwright test lcp.spec.ts` (7 route `/`, `/chat`, `/threads`, `/me`, `/inbox`, `/gradebook`, `/settings/llm`) | PASS cả 7; ứng viên LCP đầu là khối chữ lớn nhất sau phiên (nếu bỏ câu phụ thì `h1` là ứng viên trước và sau phiên) | E2E |
| TC-UX01-8 | AC3 | — | Đo LCP bằng `PerformanceObserver` trên 7 route trước / sau (cùng máy, 3 lượt) | LCP sau ≤ LCP trước + 100 ms | E2E |
| TC-UX01-9 | AC4 | sinh viên | Ở `/threads` gõ nội dung có email / MSSV rồi gửi → `PIIChannelDialog` | Mô tả hệ quả còn đủ («… cả lớp cùng đọc. Chúng tôi tìm thấy: …»); nói «Hỏi đáp công khai» thay «Threads» | E2E |
| TC-UX01-10 | AC4 | giảng viên | `/gradebook` → `Chốt điểm` (xác nhận); lớp phủ khác ở DESIGN §10.11 (`Tạo lại mã`) | Mô tả hệ quả (phân bố điểm, «không hoàn tác» / hậu quả) còn nguyên | E2E |
| TC-UX01-11 | AC5 | giảng viên | Rê / Tab vào nhãn hoặc nút ở #3, #4, #20, #34 | Tooltip hiện **một** câu ≤ 12 từ; phần tử có `aria-describedby` trỏ tới nó | E2E |
| TC-UX01-12 | AC5 | — | `git diff 1b72f54... -U0 -- frontend/src \| rg '^\+.*(\bInfo\b\|CircleHelp)'` | Không có biểu tượng `Info` / `CircleHelp` thêm mới | GREP |
| TC-UX01-13 | AC6 | — | Trên bản sao tạm của cây mã: thêm một tệp `.tsx` có `<PageHeader description="Câu giải thích tĩnh dài hơn bốn mươi ký tự ở đây" />`; chạy `bash scripts/ui-antipatterns.sh` | In `✗`, thoát mã 1 | GREP |
| TC-UX01-14 | AC6 | — | Tạo `<Section description={"abc"}/>` (≤ 40 ký tự) và `description={dynamicVar}` | Không `✗` (chỉ chuỗi tĩnh > 40 ký tự bị bắt) | GREP |
| TC-UX01-15 | AC6 | — | `bash scripts/ui-antipatterns.sh --selftest` | In `N / N`, có ca gieo cho phép «Câu mô tả tĩnh dưới tiêu đề», N lớn hơn trước 1 | GREP |
| TC-UX01-16 | AC7 | giảng viên | Chặn API để `meta` rỗng (`""`, `undefined`) ở `/students`, `/class/members` | Không còn dòng trống dưới `h1`; vị trí `h1` và khối kế tiếp như khi có `meta`; CLS ≤ 0,05 khi chuyển PreShell → màn thật | E2E |
| TC-UX01-17 | AC8 | SV / GV, 375 | Mở từng route đổi | Dưới `h1` không còn dòng thứ hai trống; K4 (không tràn ngang) | E2E |
| TC-UX01-18 | AC9 | STUDENT | Mở `/observability`, `/analytics` | 403: «Bạn không có quyền xem màn này» + dòng «Trang này dành cho …» y nguyên | E2E |
| TC-UX01-19 | AC10 | — | Chụp `home`, `chat`, `threads`, `inbox`, `gradebook` trước / sau 1440 + 390 (K7) | Mất dòng phụ; không đổi gì khác. Khi chạy `visual.spec.ts` chỉ các ảnh này đổi | MAC |
| TC-UX01-PQ | (kiểm chéo) | TA, SV | TA `/observability` / `/settings/llm`; SV `/insights` | TA: vai không được → 403 như trước; SV: 403; chữ bị xoá không lộ dữ liệu | E2E |

---
## US-UX91-05 — Trạng thái rỗng đúng mẫu

| TC | AC | Vai / cỡ | Tiền điều kiện | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- | --- |
| TC-UX05-1 | AC1 | giảng viên, 1440 | Lớp không có ticket «Đang chờ» (lớp mới do admin mở trên `ep-qc`) | `/inbox` | `EmptyState` tiêu đề «Chưa có câu hỏi nào cần thầy/cô xử lý.» + câu «Khi AI không đủ chắc chắn hoặc sinh viên nhờ hỗ trợ, câu sẽ hiện ở đây.» | E2E |
| TC-UX05-2 | AC1 | giảng viên | như trên | Tìm «Chọn một câu hỏi để xem» | Không có | E2E |
| TC-UX05-3 | AC1 | giảng viên | như trên | Đo nút / liên kết «Xem tất cả» | Là nút `ghost`, mép trái bằng mép trái của tiêu đề (chênh ≤ 1 px) | E2E |
| TC-UX05-4 | AC2 | giảng viên | Lớp 761987, có ticket | Mở một ticket | Không còn hàng «Đã chờ …» và «Trạng thái …»; tuổi ticket xuất hiện đúng một lần cạnh tên SV | E2E |
| TC-UX05-5 | AC3 | giảng viên | Lớp chưa có bài nộp | `/grading` | `EmptyState` «Chưa có bài nộp.» + «Sinh viên nộp trong ứng dụng, hoặc thầy/cô tải ZIP lên.» + nút `Tải bài lên` | E2E |
| TC-UX05-6 | AC3 | TA | như trên | `/grading` | TA có quyền thì có nút; nếu TA không có quyền tải thì **không** có nút (kiểm so với phân quyền hiện hành) | E2E |
| TC-UX05-7 | AC4 | giảng viên | Lớp chưa có công thức | `/gradebook` | `EmptyState` có đúng một nút chính `Thiết lập công thức`; câu «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.» **không** hiện | E2E |
| TC-UX05-8 | AC4 | giảng viên | Lớp 761987 có điểm | `/gradebook` | Câu trên hiện ở chân bảng, một dòng `meta` | E2E |
| TC-UX05-9 | AC5 | giảng viên | — | Huỷ mọi `/api/v1/**` rồi tải `/inbox`, `/grading`, `/gradebook`; sau đó bỏ chặn, bấm `Thử lại` | `ApiErrorNotice`: nêu chuyện gì xảy ra + dữ liệu có an toàn không + `Thử lại`; bấm → tải được; **không** thấy `EmptyState` | E2E |
| TC-UX05-10 | AC6 | STUDENT | — | `/inbox`, `/grading`, `/gradebook` | Thông báo 403 hiện có; không `EmptyState` giảng viên | E2E |
| TC-UX05-11 | AC7 | giảng viên, 375 | `/inbox` rỗng | Xem + đo | Không tràn ngang; nút ≥ 44 px | E2E |
| TC-UX05-12 | AC8 | — | Chụp `inbox` 1440 / 390 | Nếu dữ liệu cố định của `visual.spec` không rỗng thì ảnh không đổi; ngược lại đổi và dev giải thích | MAC |

---
## US-UX91-06 — Nhớ lớp, lớp đầu có dữ liệu

Mock: `GET /me/courses` trả `[A=761987, B=761988]` đúng thứ tự (QC cố định thứ tự bằng `page.route`); `GET /me/today` điều khiển tuỳ ca.

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX06-1 | AC1 | giảng viên, 1440 | Đăng nhập; chọn lớp B; đăng xuất; đăng nhập lại **cùng context** | `/` hiện lớp B (thanh trên «761988 …»); `localStorage['ep:ui:course']` = id lớp B | E2E |
| TC-UX06-2 | AC1 | giảng viên | Như trên nhưng chọn A rồi đổi sang B, tải lại trang | Vẫn B | E2E |
| TC-UX06-3 | AC2 | giảng viên | Context mới (không `ep:ui:course`); `today` có việc ở **cả A và B** | Lớp mặc định = A (đầu danh sách có dữ liệu) | E2E |
| TC-UX06-4 | AC2 | giảng viên | `today` chỉ có việc thuộc **B** (A không có) | Lớp mặc định = B | E2E |
| TC-UX06-5 | AC2 | giảng viên | `today` rỗng cả hai | Lớp đầu danh sách (A) | E2E |
| TC-UX06-6 | AC3 | giảng viên | Đặt `localStorage['ep:ui:course']` = UUID lạ rồi mở `/` | Giá trị bị bỏ; chọn theo AC2; không toast / `pageerror`; không trang trống | E2E |
| TC-UX06-7 | AC3 | giảng viên | `ep:ui:course` = lớp đã lưu trữ (admin lưu trữ lớp B trên `ep-qc`) | Như trên | E2E |
| TC-UX06-8 | AC4 | STUDENT / TA | Mở bộ chọn lớp | Chỉ lớp của họ; không có «Quản lý lớp này» / «Mời trợ giảng» (TA không có quyền); giảng viên có mục tương ứng nếu hiện nay có | E2E |
| TC-UX06-9 | AC5 | giảng viên, 375 | Mở bộ chọn lớp | Mỗi mục cao ≥ 44 px | E2E |
| TC-UX06-10 | AC6 | — | Chạy `visual.spec.ts` | Không ảnh mốc nào đổi do story này | E2E |
| TC-UX06-PQ | (kiểm chéo) | STUDENT | Đặt `ep:ui:course` = id lớp mà SV không thuộc | Bỏ giá trị; không đọc được dữ liệu lớp đó (`GET /courses/‹id›/…` 403); chọn lớp của mình | E2E |

---
## US-UX91-07 — Nhãn điều hướng tiếng Việt

Nhãn mong đợi: `Hỏi đáp công khai`, `Lỗ hổng kiến thức`, `Số liệu lớp học`. Cấm trong DOM điều hướng: `Threads`, `Insights`, `Analytics`.

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX07-1 | AC1 | sinh viên, 1440 | Đọc sidebar + `⌘K` | Có «Hỏi đáp công khai»; không có «Lỗ hổng kiến thức» / «Số liệu lớp học» (SV không có quyền); không «Threads» | E2E |
| TC-UX07-2 | AC1 | giảng viên, TA, 1440 | Đọc sidebar + `⌘K` | Có cả ba nhãn mới (TA nếu nav hiện có mục đó); không chữ Anh cũ | E2E |
| TC-UX07-3 | AC1 | admin | Đọc sidebar + `⌘K` | Chỉ nhãn nào admin có quyền; không chữ Anh cũ | E2E |
| TC-UX07-4 | AC1 | SV, GV, 375 | Thanh dưới + bấm «Thêm» | Nhãn mới; không chữ Anh cũ | E2E |
| TC-UX07-5 | AC2 | giảng viên | `/threads`, `/insights`, `/analytics` | `h1` = nhãn nav; `document.title` = «{nhãn} · EduPilot»; `curl -s` HTML cũng có `<title>` đúng (metadata) | E2E |
| TC-UX07-6 | AC2 | khách | `curl -s` khung `PreShell` của `/threads` | Tiêu đề «Hỏi đáp công khai» | E2E |
| TC-UX07-7 | AC3 | giảng viên / SV | `/threads/‹id›` | Liên kết quay lại ghi «Hỏi đáp công khai»; tiêu đề trang chi tiết giữ nghĩa cũ | E2E |
| TC-UX07-8 | AC4 | — | `rg -n -w "Threads\|Insights\|Analytics" frontend/src --glob '!**/*.test.*' --glob '!**/*.spec.*'` rồi loại tên biến, import, đường dẫn, comment | Còn 0 chuỗi hiển thị. Bắt buộc rà: `notifications.ts` (`AI_PENDING`, `POST_REPORTED`), `threadsActions.ts`, `PIIChannelDialog`, câu `activity`, trang giới thiệu, `DemoThreads` («Câu hỏi mới trong Threads») | GREP |
| TC-UX07-9 | AC4 | giảng viên | Tạo thông báo `AI_PENDING` / `POST_REPORTED` (SV đăng câu có AI chờ; báo cáo bài) rồi mở chuông | Chữ thông báo nói «Hỏi đáp công khai», không «Threads» | E2E |
| TC-UX07-10 | AC4 | sinh viên | `/` khối hoạt động («Chat, Threads, Luyện đề…») | Dùng tên mới | E2E |
| TC-UX07-11 | AC5 | giảng viên / SV | Mở trực tiếp `/threads`, `/insights`, `/analytics` | Vào đúng route, 200 | E2E |
| TC-UX07-12 | AC6 | giảng viên, 1440 + thu gọn sidebar 72 px | Sidebar thu gọn | Không cắt / không đẩy tràn; có tên đầy đủ qua `aria-label` / tooltip | E2E |
| TC-UX07-13 | AC6 | SV, 375 | Thanh dưới 5 mục (có «Hỏi đáp công khai») | Nhãn không bị cắt (`scrollWidth ≤ clientWidth` mỗi nhãn); nếu dùng nhãn rút gọn thì «Hỏi đáp» ≤ 8 ký tự; ảnh 375 px | E2E + MAC |
| TC-UX07-14 | AC7 | — | `git diff 1b72f54... -- frontend/e2e/shell.spec.ts` | Chỉ đổi chuỗi nhãn (dòng 120–124), không xoá / nới assertion; spec xanh | GREP |
| TC-UX07-15 | AC8 | STUDENT, TA | So số mục nav theo vai trước / sau | SV 8 mục, TA 12, GV 15 (hoặc số hiện hành), admin 6; chỉ đổi nhãn | E2E |
| TC-UX07-16 | AC9 | — | `rg -n "Threads\|Insights\|Analytics" docs/design/DESIGN.md` ở §2, §14.3, §14.21, §14.25 | Tên mới thay đủ ở 4 mục; (chỗ còn chữ cũ phải là ghi chú đổi tên có chủ ý) | GREP |
| TC-UX07-17 | AC9 | — | Chụp `threads`, `home`, `chat`, `inbox`, `gradebook`, `settings-llm` trước / sau | Sidebar có nhãn mới; các thứ khác không đổi | MAC |
| TC-UX07-PQ | (kiểm chéo) | STUDENT | Mở `/insights`, `/analytics` trực tiếp | 403; nhãn nav không lộ mục không có quyền | E2E |

---
## Bảng phủ AC đợt 2
| Story | AC | TC |
| --- | --- | --- |
| US-01 | AC1 | 01-1, 2, 3, 4 |
| | AC2 | 01-5, 6 |
| | AC3 | 01-3, 7, 8 |
| | AC4 | 01-9, 10 |
| | AC5 | 01-11, 12 |
| | AC6 | 01-13, 14, 15 |
| | AC7 | 01-16 |
| | AC8 | 01-17 |
| | AC9 | 01-18 |
| | AC10 | 01-19 |
| US-05 | AC1 | 05-1, 2, 3 |
| | AC2 | 05-4 |
| | AC3 | 05-5, 6 |
| | AC4 | 05-7, 8 |
| | AC5 | 05-9 |
| | AC6 | 05-10 |
| | AC7 | 05-11 |
| | AC8 | 05-12 |
| US-06 | AC1 | 06-1, 2 |
| | AC2 | 06-3, 4, 5 |
| | AC3 | 06-6, 7 |
| | AC4 | 06-8 |
| | AC5 | 06-9 |
| | AC6 | 06-10 |
| US-07 | AC1 | 07-1, 2, 3, 4 |
| | AC2 | 07-5, 6 |
| | AC3 | 07-7 |
| | AC4 | 07-8, 9, 10 |
| | AC5 | 07-11 |
| | AC6 | 07-12, 13 |
| | AC7 | 07-14 |
| | AC8 | 07-15 |
| | AC9 | 07-16, 17 |
