# QC test case — US-PU-04 (shell thật: sidebar 216/72, topbar 56, bottom nav < 720, điều hướng theo vai, phiên JWT/demo)
Nguồn: `docs/specs/FEAT-ui-foundation/US.md` US-PU-04 AC1–AC14 + `SRS.md` 7.1 (route PU), 7.5 (điều hướng theo vai + ma trận quyền mở route, khớp commit `307bfd2`), 7.6 (route → backend → phase), 3.3 (hai nguồn phiên), 8.1 (cờ build). Nền: `audit-baseline.md`.

Tiền điều kiện chung: bản `gbuild` (`/dev/*` mở, mock bật, cổng dán token bật) chạy `next start -p 3300`; ngữ cảnh vai = cookie `ep_demo_role/person/course` như 1.5 (SV B `sv-2`, SV D chưa vào lớp `sv-4`, TA, GV, Admin) hoặc JWT thật (`tok` = `go run ./cmd/gateway token --role … --ttl 30m`, secret của `.env.local`). Công cụ: **A** = Eval trình duyệt thật (chuột thật; `AUDIT_SRC`, `TOUCH_SRC`, `LEFT` nguyên văn của `audit.mjs`), **P** = `$PW shell.spec.ts -g …` của dev, **S** = shell, **T** = tay. Thiếu `shell.spec.ts` → TC loại **P** FAIL "KHÔNG KIỂM ĐƯỢC". QC tự đo bằng **A** cho mọi AC, không chỉ tin **P**.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PU04-01 | AC1 | GV `/` | **A** đo `[data-part=sidebar]` rộng và `header` cao ở 1440, 1100 | Sidebar **216**, header **56**; nhãn chữ hiện |
| TC-PU04-02 | AC1 | – | **A** ở 1099, 900, 720 | Sidebar **72** (icon + `aria-label`); header 56; nhãn hiện khi hover / focus (`mouse.move`, `Tab`) |
| TC-PU04-03 | AC1 | – | **A** ở 719, 390, 375 | Không sidebar (ẩn / không tồn tại); có `[data-part=bottom-nav]` cố định đáy |
| TC-PU04-04 | AC1 | 1440 | **A** bấm nút thu gọn sidebar (đọc `aria-expanded`), tải lại; đọc `localStorage` | `aria-expanded` đổi; sidebar 72 ↔ 216; nhớ qua tải lại; khoá `ep:ui:sidebar` giá trị không phải token; ở 1100 vẫn thu gọn / mở được |
| TC-PU04-05 | AC1 | 375 | **A** cuộn `main` tới cuối trang dài (ví dụ `/threads`) | Nút cuối của nội dung `isIntersectingViewport`, không bị thanh dưới che (đệm dưới ≥ chiều cao thanh + safe-area) |
| TC-PU04-06 | AC1 | – | **P** `$PW shell.spec.ts -g 'metrics'; echo rc=$?` | `rc=0` |
| TC-PU04-07 | AC2 | GV, từng mục đang chọn | **A** đọc `aria-current` và `borderLeftWidth` / `::before` của mục chọn; `backgroundColor` | `aria-current="page"` có đúng 1 mục; vạch **2 px** màu `--ep-red` (hoặc chữ đậm hơn); nền không đỏ |
| TC-PU04-08 | AC2 | mọi vai | **A** đếm phần tử trong `[data-part=sidebar], header` có nền đỏ (computed `background-color` thuộc họ đỏ); đếm `bell-dot` + huy hiệu hiện; đo độ trong suốt của khung vỏ | Số phần tử nền đỏ ≤ số chấm / huy hiệu đang hiện; khung vỏ **nền đặc** (không `backdrop-filter`, `rgba` alpha < 1 ở nền header/sidebar) |
| TC-PU04-09 | AC2 | – | **S** `bash scripts/ui-antipatterns.sh \| grep 'Khung vỏ dùng nền đặc'` | Dòng bắt đầu `✓` |
| TC-PU04-10 | AC3 | SV B | **A** đọc `nav a` theo thứ tự: nhãn + `href` | 7 mục đúng thứ tự SRS 7.5: Hôm nay `/`, Chat riêng `/chat`, Threads `/threads`, Luyện đề `/practice`, Thư viện `/library`, Lịch `/calendar`, Kết quả của tôi `/me` |
| TC-PU04-11 | AC3 | SV D chưa vào lớp (`sv-4`, không `ep_demo_course`) | **A** đọc `nav a` | **Chỉ** `Hôm nay` (1 mục) |
| TC-PU04-12 | AC3 | TA | **A** đọc `nav a` + tên nhóm | 12 mục theo nhóm Làm việc 4 / Đánh giá 3 / Nội dung 3 / Hiểu lớp học 2, đúng thứ tự và `href`; **không** nhóm "Hệ thống" |
| TC-PU04-13 | AC3 | GV | **A** đọc `nav a` + nhóm | 12 mục như TA **+** nhóm "Hệ thống": Quan sát AI `/observability`, Cấu hình LLM `/settings/llm`, Tích hợp `/settings/integrations` (tổng 15) |
| TC-PU04-14 | AC3 | Admin | **A** đọc `nav a` | 6 mục theo nhóm: Vận hành (Hôm nay, Quan sát AI), Quản trị (Lớp học `/admin/courses`, Người dùng `/admin/users`), Cấu hình (Cấu hình LLM, Tích hợp) |
| TC-PU04-15 | AC3 | – | **S** `proto-curl.sh`-kiểu: với từng vai, `visible student / \| grep -c 'Cấu hình LLM'` = 0; TA không có "Cấu hình LLM"; `curl` các route cấm, đối chiếu `tc_00_matrix` | SV: 0 lần "Cấu hình LLM"; TA không mục Hệ thống; mục nav của từng vai ⊆ route vai đó mở được (khớp ma trận SRS 7.5); `tc_00_matrix` ≥ 128 hàng PASS |
| TC-PU04-16 | AC3 | – | **P** `$PW shell.spec.ts -g 'nav per role'; echo rc=$?` | `rc=0` |
| TC-PU04-17 | AC4 | GV, TA | **A** liệt kê `[data-part^=nav-badge-]` và mục mang huy hiệu | Hậu tố chỉ `inbox` (Hộp thư hỗ trợ) và `grading` (Chấm bài); Tài liệu, Threads… **không** có số |
| TC-PU04-18 | AC4 | – | **A** (với nguồn mock) duyệt hết bài / nhận hết phiếu → huy hiệu về 0; mục "chưa biết" (chưa có dữ liệu thật) | Khi bằng 0 hoặc chưa biết: phần tử **không tồn tại** (không hiện "0") |
| TC-PU04-19 | AC4 | – | **S** `grep -nE 'badge: [0-9]' frontend/src/shared/shell/nav.ts \| wc -l` | `0` (giá trị lấy từ nguồn dữ liệu, không viết cứng) |
| TC-PU04-20 | AC5 | SV, TA, GV, Admin ở 375 và 390 | **A** đếm `[data-part=bottom-nav] a, [data-part=bottom-nav] button` | ≤ **5** mỗi vai = 4 đích chính + `Thêm`; mỗi đích cao ≥ 44 px |
| TC-PU04-21 | AC5 | – | **A** bấm `Thêm`: đếm mục trong bảng trượt; đóng bằng `Esc`, bấm ngoài, vuốt xuống (touch); đọc focus | Bảng liệt kê **đủ** các mục còn lại của vai (số mục nav − 4), cùng nhóm, đủ nhãn; cả ba cách đóng đều đóng và **trả focus** về nút `Thêm` |
| TC-PU04-22 | AC5 | 375 | **A** thanh trên: bộ chọn lớp, tên trang (`h1` của trang), đúng **một** hành động ngữ cảnh; `h1` không lặp ở `header` ≥ 720 | Có đủ ba thứ; đúng 1 hành động; không tràn ngang ở 375 và 390 (`AUDIT_SRC` `ox:0`) |
| TC-PU04-23 | AC5 | – | **A** `TOUCH_SRC` và `AUDIT_SRC` ở 375/390 trên `/` của từng vai | `[]`; `{ ox:0, cut:[], ell:[] }` |
| TC-PU04-24 | AC5 | – | **P** `$PW shell.spec.ts -g 'mobile'; echo rc=$?` | `rc=0` |
| TC-PU04-25 | AC6 | ≥ 720 | **A** `header h1` đếm; đếm nhóm điều khiển trong `header`; `aria-label` nút hồ sơ | `0` `h1` trong `header`; đúng **4** nhóm (bộ chọn lớp, ô bảng lệnh, chuông, hồ sơ); nút hồ sơ `aria-label` khớp `^Tài khoản: <tên người>` (không tên vai); dòng 1 tên, dòng 2 vai |
| TC-PU04-26 | AC6 | – | **P** `$PW shell.spec.ts -g 'topbar'; echo rc=$?` | `rc=0` |
| TC-PU04-27 | AC7 | GV | **A** `Ctrl K` (và `⌘K`, và bấm ô tìm nhanh); đọc focus; gõ "diem danh" | Palette mở, focus ở ô nhập, bẫy focus (Tab ×10 vẫn trong palette); mục "Điểm danh" **đứng đầu** (không phân biệt dấu/hoa thường); `↑ ↓` đổi mục (`aria-activedescendant`), `Enter` đi tới `/attendance` |
| TC-PU04-28 | AC7 | SV B | **A** gõ "diem danh"; gõ "zzzz"; liệt kê toàn bộ mục khi ô trống | SV: không thấy "Điểm danh"; "zzzz": "Không thấy mục nào khớp." + gợi ý; danh sách trống ô = đúng 7 mục của SV (không route vai khác) |
| TC-PU04-29 | AC7 | – | **A** `Esc`; kiểm `document.activeElement` | Palette đóng; focus trở về phần tử đã mở |
| TC-PU04-30 | AC8 | `/dev/data` (`items` giả) | **A** `unread=2`, rồi `0`; mở popover khi `unread>0` rồi đọc chấm | `[data-part=bell-dot]` có mặt khi `unread>0`, biến khi `0`; **mở khung không xoá chấm**; bấm một mục → điều hướng + đánh dấu đã đọc; chấm biến khi hết chưa đọc |
| TC-PU04-31 | AC8 | – | **A** `items=[]`; đọc chuỗi; mỗi hàng: tiêu đề, một dòng ngữ cảnh, thời điểm, đường kẻ ngăn | Rỗng: "Chưa có thông báo. Khi có việc cần bạn, nó sẽ hiện ở đây."; hàng đủ ba trường |
| TC-PU04-32 | AC8 | – | **A** `unread` 0→1 rồi hiển thị lại nhiều lần; đếm cập nhật của `[aria-live=polite]` | Đúng **một lần** "Có 1 thông báo mới"; không đọc lại mỗi lần hiển thị lại |
| TC-PU04-33 | AC8 | – | **P** `$PW shell.spec.ts -g 'notifications'; echo rc=$?` | `rc=0` |
| TC-PU04-34 | AC9 | `tok ADMIN` | **A** dán token ADMIN hợp lệ ở cổng dev (hoặc `tokenStore`), mở `/` | Phiên `jwt`: nav của Admin (6 mục); **không** bộ đổi vai mô phỏng; menu hồ sơ có `Đăng xuất`; vai / `sub` / `email` lấy từ claim (đối chiếu giải mã tay JWT) |
| TC-PU04-35 | AC9 | – | **A** bấm `Đăng xuất`; mở `/` | Token bị xoá; trở về phiên `demo` (cookie `ep_demo_*`) hoặc cổng token |
| TC-PU04-36 | AC9 | token `exp` quá khứ; `tok STUDENT` với claim `role` đổi thành `SUPERUSER` (ký đúng hoặc sai) | **A** dán từng token | Hết hạn → `InlineNotice` "Phiên đã hết hạn"; role lạ → "Token không hợp lệ"; token **bị bỏ** (không áp phiên) |
| TC-PU04-37 | AC9 | build gate, chưa token | **A** vào `/settings/llm` (backend thật) | Cổng "Dán token quản trị để tiếp tục": ô `type="password"`, `autocomplete="off"` |
| TC-PU04-38 | AC9 | – | **A** dán `abc`, dán chuỗi 3 đoạn không phải JWT, dán chuỗi rỗng; so `Object.keys(localStorage)` trước / sau | "Token không hợp lệ" mỗi lần; **không** lưu (`localStorage`/`sessionStorage` không đổi); không hiện token trong DOM sau khi dán (ô password) |
| TC-PU04-39 | AC9 | `pbuild` (không cờ dev) | **S**+**A** mở `/settings/llm`; `grep -c 'type="password"'` ở HTML; `curl` | Màn "Cần đăng nhập — tính năng đăng nhập sẽ có ở bản sau"; **không** `input[type=password]`; không ô dán |
| TC-PU04-40 | AC9 | – | **P** `$PW shell.spec.ts -g 'session'; echo rc=$?` | `rc=0` |
| TC-PU04-41 | AC10 | SV B, rồi TA, đo request mạng | **A** mở `/settings/llm`, `/observability`, `/admin/users`; đếm request `/api/v1/` | Màn "Bạn không có quyền xem màn này" + câu "Trang này dành cho …" + một nút `Về Hôm nay`; **0** request tới `/api/v1/*` (kể cả `/api/v1/admin/*`); không nháy nội dung đích |
| TC-PU04-42 | AC10 | GV | **A** `/admin/courses` (chặn); `/settings/llm` (mở, chỉ xem) | `/admin/courses`: màn chặn; `/settings/llm`: mở được, không có thao tác sửa (xem US-P1-05) |
| TC-PU04-43 | AC10 | SV D chưa vào lớp | **A** mở `/chat`, `/threads` | Màn "Bạn chưa vào lớp nào" (hành vi 1.5) |
| TC-PU04-44 | AC10 (phân quyền thật) | **R** gateway thật; token STUDENT | **S** `curl -sk -H "Authorization: Bearer $(tok STUDENT)" $GW/api/v1/admin/llm/providers` (khi US-P1-04 có) và route ADMIN khác | `403 FORBIDDEN` — chặn thật ở gateway, không chỉ ở khung (khung là phòng thủ lớp ngoài) |
| TC-PU04-45 | AC10 | – | **S** `proto-curl.sh tc_00_matrix` (`F=:3300`) | Ma trận route × vai: ≥ 128 hàng PASS, không route nào mở cho vai không có trong SRS 7.5 (TA vào `/settings/llm` bị chặn) |
| TC-PU04-46 | AC11 | build `NEXT_PUBLIC_MOCK_SCREENS=0 DEV_TOOLS=1 DEV_AUTH=1` | **A** với mỗi route `backend=mock` của từng vai (bảng SRS 7.6): mở, đếm `[data-part=empty-no-backend]`, nút, tên phase, mục nav tương ứng | Có `empty-no-backend`; chứa đúng **tên phase** theo bảng (P2…P10); đúng **1** nút (`Về Hôm nay`); mục nav **vẫn có** trong `nav`; `/settings/llm` và `/dev/*` không bị ảnh hưởng |
| TC-PU04-47 | AC11 | – | **A** đọc câu của SV: không từ kỹ thuật (`RAG`, `PII`, `trace`, `provider`…) | Câu tiếng Việt thường theo vai |
| TC-PU04-48 | AC11 | build `MOCK_SCREENS=1` (mặc định) | **A** mở `/inbox`, `/chat`… | Màn mock chạy như 1.5 (không `empty-no-backend`) |
| TC-PU04-49 | AC11 | – | **P** `$PW shell.spec.ts -g 'no-backend'; echo rc=$?` | `rc=0` |
| TC-PU04-50 | AC12 | bất kỳ trang | **A** `Tab` lần đầu: `document.activeElement.textContent`; `Enter`; đếm landmark | Focus đầu = `Bỏ qua điều hướng` (hiện khi focus); `Enter` → focus trong `main`; có `header`, `nav` có `aria-label`, đúng **1** `main` |
| TC-PU04-51 | AC12 | – | **A** thứ tự Tab: ghi chuỗi 12 phần tử focus đầu tiên; `Esc` ở palette, popover chuông, menu hồ sơ | Thứ tự: bỏ qua → thanh trên → sidebar → nội dung; mỗi lớp phủ đóng bằng `Esc` và trả focus; vòng focus nhìn thấy ở mọi mục khung (ảnh 3 mục) |
| TC-PU04-52 | AC12 | – | **P** `$PW shell.spec.ts -g 'keyboard'; echo rc=$?` | `rc=0` |
| TC-PU04-53 | AC13 | bản gate | **A** `audit.mjs` bốn vai + spec (như baseline) | Mỗi lượt FAIL 0; hàng PASS ≥ nền (SV 165, GV 170, TA 106, Admin 54, spec 185) |
| TC-PU04-54 | AC13 | – | **A** `LEFT` của `[data-part=page-title]` ở 1440 / 390; cao `[data-part=brand]`, logo đầy đủ, mark thanh trên điện thoại | `240` / `16`; brand cao **56**, logo đầy đủ cao **32**, mark **28** ở thanh trên điện thoại |
| TC-PU04-55 | AC13 | – | **A** badge và chuông prototype: hành vi chuỗi duyệt bài / nhận phiếu như `report-v5-round2` (02-61/62), D3 → chuông | Không đổi hành vi so với 1.5 (đối chiếu `TC-02-61/62`, `TC-01-138`) |
| TC-PU04-56 | AC13 | – | **P** `$PW shell.spec.ts -g 'regression-1.5'; echo rc=$?` | `rc=0` |
| TC-PU04-57 | AC14 | 4 vai × 375/390 | **A** `TOUCH_SRC` ở `/` và ở từng route mà thanh dưới dẫn tới; `ox` | `[]` mọi ca; không tràn ngang |
| TC-PU04-58 | AC14 | – | **P** `$PW shell.spec.ts -g 'touch'; echo rc=$?` | `rc=0` |
| TC-PU04-59 | tổng | – | **A** `sweep.mjs only:'student'`; `proto-curl.sh all`; `$FE lint`; `ui-antipatterns.sh` | `FORBIDDEN`=0; ≥ 497 PASS 0 FAIL; lint rc=0; ✓ mọi dòng |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| SV chưa vào lớp thấy nav đầy đủ | 11, 43 |
| TA mở được Cấu hình LLM | 12, 15, 41, 45 |
| Số huy hiệu viết cứng / hiện "0" | 18, 19 |
| Thanh dưới > 5 đích; che nội dung | 05, 20 |
| Token hết hạn / vai lạ; token lưu / lộ | 36, 38 |
| Build thường lộ cổng dán token | 39 |
| Màn chặn vẫn gọi API | 41 |
| Chặn thật ở gateway (khung chỉ phòng thủ) | 44 |
| Route chưa có backend: mục nav biến mất | 46 |
| Chuông đọc lại thông báo mỗi lần | 32 |

## Câu hỏi cho BA / PM
- **Q-QC-PU04-1** — AC3 "Giảng viên 12 mục + nhóm Hệ thống" nhưng bảng SRS 7.5 ghi tổng **15** (12 + 3). QC chấm theo bảng (15) — *chờ xác nhận*.
- **Q-QC-PU04-2** — AC10 yêu cầu TC-PU04-44 trên gateway thật, nhưng route `/api/v1/admin/llm/*` chỉ có sau US-P1-04: TC chạy khi story đó bàn giao, trước đó ghi N/A (không FAIL). Đồng ý? — *chờ trả lời*.
- **Q-QC-PU04-3** — AC9 "phiên Student JWT" vào màn mock `/chat`: khung dùng cookie demo hay JWT cho màn mock? QC chưa kiểm hành vi này (ngoài AC). — *chờ trả lời*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md (FEAT-ui-foundation, APPROVED 2026-10-03).

Tổng: 59 TC.
