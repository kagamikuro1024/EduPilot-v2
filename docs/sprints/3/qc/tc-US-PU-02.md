# QC test case — US-PU-02 (primitive đủ 8 trạng thái + `/dev/ui`)
Nguồn: `docs/specs/FEAT-ui-foundation/US.md` US-PU-02 AC1–AC16 + `SRS.md` 7.2 (bảng 24 khối × 8 trạng thái, ô N/A), 7.3 (quy ước trạng thái), 7.4 (chuỗi mẫu tiếng Việt dài), 8.1 (cờ build). Hộp đen: chỉ dùng `data-part` / `data-name` / `data-state` / `data-na` / `data-reason` ghi trong US + chữ hiển thị. Nền: `audit-baseline.md`.

Tiền điều kiện chung: `gbuild && $FE exec next start -p 3300` (bản gate, `/dev/ui` mở); `pbuild` cho TC-PU02-40. Công cụ: **P** = `$PW dev-ui.spec.ts -g …` (Playwright của dev), **A** = Eval trình duyệt thật (`AUDIT_SRC`, `TOUCH_SRC` nguyên văn của `audit.mjs`), **S** = shell, **T** = tay (QC). Thiếu `frontend/e2e/dev-ui.spec.ts` → TC loại **P** FAIL "KHÔNG KIỂM ĐƯỢC". Với mỗi TC loại **P**, QC **cũng** tự đo lại bằng **A** ở các ô đã nêu (không chỉ tin test của dev).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PU02-01 | AC1 | `/dev/ui` 1440 | **A** `$$('[data-part=primitive]').length`; liệt kê `data-name` và so với bảng SRS 7.2 | `24`; đúng tên 22 primitive + `CitationList` + `VerificationState`, không thừa / thiếu |
| TC-PU02-02 | AC1 | – | **A** `$$('[data-part=state-cell]:not([data-na])').length`; mỗi ô có `data-state` | `117`; mọi `data-state` ∈ {`default`,`hover`,`focus`,`selected`,`disabled`,`loading`,`empty`,`error`}; mọi ô có nội dung nhìn thấy (không rỗng) |
| TC-PU02-03 | AC1 | – | **A** `$$('[data-part=state-cell][data-na]').length`; ô nào thiếu `data-reason`; so danh sách (khối, trạng thái) với bảng N/A SRS 7.2 | `75`; 0 ô thiếu / rỗng `data-reason`; danh sách khớp SRS 7.2 từng cặp (117 + 75 = 24 × 8 = 192 — kiểm phép nhân); không thêm bớt |
| TC-PU02-04 | AC1 | – | **P** `$PW dev-ui.spec.ts -g 'matrix'; echo rc=$?` | `rc=0`, 0 test skip |
| TC-PU02-05 | AC2 (hover) | – | **A** với mỗi primitive tương tác (16 khối): `mouse.move` vào phần tử ở ô `default` rồi so `getComputedStyle` (nền, `text-decoration`, màu chữ) với ô `hover` | ≥ 1 thuộc tính khác nhau ở **mọi** khối; khối không đổi gì → liệt kê, FAIL |
| TC-PU02-06 | AC2 (focus) | – | **A** `Tab` tới từng phần tử focus được ở ô `focus`; đọc `boxShadow` tính ra | Khớp `--ep-focus` (đỏ 2 px + khoảng 2 px); **mọi** 16 khối |
| TC-PU02-07 | AC2 (selected) | – | **A** ô `selected` của ActionList, DataTable, Tabs, SegmentedControl, FilterChips, Menu: đọc `aria-selected`/`aria-pressed`/`aria-checked`/`aria-current`; so ảnh chụp với ô default | Có đúng một thuộc tính aria nêu trên **và** một dấu hiệu không chỉ là màu (vạch / chữ đậm / icon): đo `font-weight` hoặc có phần tử vạch/icon khác default — mắt xác nhận bằng ảnh `shots/pu02-selected-*.png` |
| TC-PU02-08 | AC2 (disabled) | – | **A** ở ô `disabled` của mọi khối có trạng thái đó: `disabled`/`aria-disabled`, `cursor`, đăng ký `click` bộ đếm rồi `click({force:true})`, `aria-describedby` khi có lý do | `disabled` hoặc `aria-disabled="true"`; `cursor: not-allowed`; số lần handler = `0`; nếu có lý do thì `aria-describedby` trỏ phần tử có chữ |
| TC-PU02-09 | AC2 | – | **P** `$PW dev-ui.spec.ts -g 'states'; echo rc=$?` | `rc=0` |
| TC-PU02-10 | AC3 (loading) | – | **A** mọi ô `data-state=loading`: có `[aria-busy=true]`; đếm `.animate-spin` / `svg` quay / `role=progressbar` ngoài nút; so khung `header`/`nav` của shell trước-sau | Có `aria-busy=true`; không spinner giữa trang (chỉ khung xương); shell không đổi |
| TC-PU02-11 | AC3 (empty) | – | **A** mọi ô `empty`: đếm `button, a` hành động trong ô; đọc lời | Đúng **1** hành động; có câu nói *vì sao rỗng* (≥ 1 mệnh đề, không "Không có dữ liệu" trơ trọi) |
| TC-PU02-12 | AC3 (error) | – | **A** mọi ô `error`: `role=alert`, nút `Thử lại`, kiểm nội dung với `/^[A-Z_]{3,}$\|Something went wrong\|undefined\|\[object/` và có đủ ba ý (vấn đề, dữ liệu an toàn?, cách khắc phục) | Có `role=alert` + `Thử lại`; không khớp regex cấm; mắt đọc: có câu "dữ liệu của bạn vẫn an toàn / chưa lưu" |
| TC-PU02-13 | AC3 | – | **P** `$PW dev-ui.spec.ts -g 'loading\|empty\|error'; echo rc=$?` | `rc=0` |
| TC-PU02-14 | AC4 | `/dev/ui` + chuỗi mẫu SRS 7.4 | **A** ở 375, 900, 1280, 1440, **640**: chạy `AUDIT_SRC` | Mỗi bề rộng `{ ox: 0, cut: [], ell: [] }` (không cuộn ngang, không chữ cắt, không `…` thiếu `title`) |
| TC-PU02-15 | AC4 | 375 cảm ứng | **A** `TOUCH_SRC` | `[]` (vùng chạm ≥ 44 × 44) |
| TC-PU02-16 | AC4 | – | **A** đo `scrollHeight ≤ clientHeight + 1` cho chuỗi "Ặ Ế Ộ Ử Ữ Ầ" và tiêu đề 120 ký tự, tên dài, URL dài; ở 5 bề rộng; chụp ảnh | 0 phần tử bị cắt dấu; URL dài xuống dòng hoặc cuộn trong khối của nó (không phá bố cục trang) |
| TC-PU02-17 | AC4 | – | **P** `$PW dev-ui.spec.ts -g 'overflow'; echo rc=$?` | `rc=0` |
| TC-PU02-18 | AC5 | `DataTable` 1.000 dòng ở `/dev/ui` | **A** cuộn dần tới cuối bằng `mouse.wheel`; mỗi 100 ms đếm `tbody tr` | Luôn **< 80** `<tr>` trong DOM; cuộn tới dòng cuối ≤ **1 s** (đo đồng hồ) |
| TC-PU02-19 | AC5 | – | **A** `th` `position: sticky` khi cuộn; chiều cao dòng bằng `getBoundingClientRect().height` cho 20 dòng ngẫu nhiên | Tiêu đề cột dính (vẫn thấy sau khi cuộn 5.000 px); mọi dòng cao ∈ [44, 52] |
| TC-PU02-20 | AC5 | – | **A** bàn phím: chọn dòng, `↓` ×5, `↑`, `End`, `Home`, `Enter`, `Esc` | Dòng đang chọn đổi đúng; `End` tới dòng 1000 (cuộn theo); `Enter` kích hoạt; `Esc` thoát chọn |
| TC-PU02-21 | AC5 | – | **A** cuộn tới `scrollTop` S, chọn dòng k, mở `Drawer` chi tiết, đóng | `scrollTop` = S (±0) và dòng k vẫn đang chọn |
| TC-PU02-22 | AC5 | – | **A** đo p95 thời gian khung (rAF) trong lúc cuộn tự động 5 s | Ghi số vào report (CI chỉ ghi, không chặn); nếu > 25 ms ghi "lưu ý" |
| TC-PU02-23 | AC6 | `page.route` giả 3 trang | **P** `$PW dev-ui.spec.ts -g 'cursor'`; **A** tay: cuộn nhanh gần cuối nhiều lần | Số request = 3, mỗi `cursor` khác nhau, không gọi trùng khi cuộn nhanh; sau `next_cursor:null` không gọi nữa |
| TC-PU02-24 | AC6 | – | **A** đo CLS (PerformanceObserver) lúc dòng khung xương "đang tải thêm" xuất hiện / biến mất | ≤ `0,02`; khung xương nằm đúng chỗ (ngay dưới dòng cuối) |
| TC-PU02-25 | AC6 (nhánh lỗi) | trang 2 trả 503 | **A** cuộn tới trang 2 | Các dòng trang 1 **còn nguyên**; hiện `InlineNotice` ở đáy với `Thử lại`; bấm `Thử lại` → trang 2 tải được (lần này 200) |
| TC-PU02-26 | AC7 (bẫy focus) | – | **A** mở `Dialog`, `Drawer`; `Tab` ×20 và `Shift+Tab` ×20 | `document.activeElement` luôn trong hộp thoại; phần còn lại có `inert`/`aria-modal`; nền đứng yên (không cuộn) |
| TC-PU02-27 | AC7 (đóng + trả focus) | – | **A** `Esc` khi đóng; trả focus về nút đã mở; riêng `ConfirmIrreversible` đang `loading` bấm `Esc` | Đóng và trả focus đúng nút mở; `ConfirmIrreversible` đang `loading`: `Esc` **không** đóng |
| TC-PU02-28 | AC7 (Drawer rộng) | – | **A** Drawer ở 900, 1440 px → `width`; ở 719, 375 px | ∈ [420, 520] ở ≥ 720 px; ở < 720 px = `innerWidth` (toàn màn) |
| TC-PU02-29 | AC7 (Popover/Menu) | – | **A** mở Menu: `↓ ↓ Enter`; mở Popover: bấm ngoài; mở lại: `Esc` | `↑ ↓` đổi mục; `Enter` chọn; bấm ngoài và `Esc` đều đóng; `Dialog` chỉ xuất hiện ở khối được phép (không dùng Dialog cho thao tác đảo ngược được) |
| TC-PU02-30 | AC7 | – | **P** `$PW dev-ui.spec.ts -g 'overlay'; echo rc=$?` | `rc=0` |
| TC-PU02-31 | AC8 | Composer ở `/dev/ui` | **A** 1440: `width`; đếm `button.primary` trong vùng; nút phụ có lớp `ghost`; ô trống → nút gửi `disabled` | `width ≤ 820`; đúng `1` nút primary; nút phụ `ghost`; ô trống: gửi disabled |
| TC-PU02-32 | AC8 | – | **A** gõ khi ô ở `loading`; gửi lỗi (`page.route` 500) | Gõ tiếp được, chữ không mất; sau lỗi giữ nguyên nội dung và hiện `Gửi lại` |
| TC-PU02-33 | AC8 | – | **A** `setViewportSize({375, 330})` (bàn phím ảo giả lập), focus ô nhập | Ô nhập `isIntersectingViewport` = true, không bị thanh dưới che |
| TC-PU02-34 | AC9 | – | **A** PerformanceObserver `layout-shift` (không `hadRecentInput`) khi `DataTable`, `ActionList`, `Section`, `PageHeader` chuyển loading → đã tải ở `/dev/ui`; đo chênh chiều cao | Tổng CLS ≤ `0,05`; chiều cao đã tải lệch khung xương ≤ 8 px |
| TC-PU02-35 | AC10 | – | **A** `CitationList` có n nguồn: đọc "Nguồn tham khảo (n)", `aria-expanded`, bấm một nguồn; đếm `window.open`/điều hướng | Nhãn đúng n; bấm mở đoạn trích **tại chỗ** (`aria-expanded` false→true), URL không đổi, không tab mới; rỗng: "Chưa có nguồn tham khảo cho câu trả lời này." |
| TC-PU02-36 | AC10 | `?as=teacher` | **A** `VerificationState` bốn trạng thái: đọc chữ, chấm màu (computed), nền khối, `borderLeftWidth` | Đúng bốn chuỗi: `Chờ xác nhận`; `Đã được giảng viên xác nhận · <tên>`; `Đã được giảng viên sửa & xác nhận` + `Xem câu trả lời AI gốc`; `Đang chờ giảng viên`; khối "đã xác nhận": nền trong suốt, `borderLeftWidth = 1px` xanh |
| TC-PU02-37 | AC10 (phân quyền) | `?as=student` | **A** đếm nút duyệt (`Xác nhận`, `Chỉnh sửa`, `Loại khỏi tri thức`) ở `VerificationState` ba vai | SV: **0** nút duyệt; TA / GV / Admin: có đủ ba nút, nằm **ngay cạnh** nháp (khoảng cách ≤ 24 px) |
| TC-PU02-38 | AC11 | – | **S** `grep -rnE "[\"'\`>][^\"'\`<]*\b(RAG\|PII\|fallback\|trace\|provider\|redaction\|confidence\|embedding\|prompt\|LLM)\b" frontend/src/shared/ui frontend/src/shared/domain --include=*.tsx \| grep -v 'ui-allow:' \| wc -l`; `ui-antipatterns.sh` | `0`; sạch |
| TC-PU02-39 | AC12 | bản `gbuild` | **A** `audit.mjs` bốn vai + spec (như baseline); `sweep.mjs only:'student'`; so `data-part` của QC 1.5 (`page-title`, `brand`, `chat-thread`, `chat-composer`, `chat-history`, `provider-status`, `provider-action`, `bell-dot`, `settings-section`, `nav-badge-*`, `thread-row`, `thread-form`, `col-qt`, `col-status`, `student-row`, `inbox-reply`, `chart-axis-label`, `chart-point`) bằng cách truy ra từ DOM các route tương ứng | Mỗi lượt FAIL 0, số hàng ≥ nền (680); sweep `FORBIDDEN`=0; cả 18 `data-part` còn có mặt ở route dùng chúng; `proto-curl.sh all` ≥ 497 PASS |
| TC-PU02-40 | AC16 | `pbuild` | **S** `curl -s -o /dev/null -w '%{http_code}\n' $BASE/dev/ui` và `/dev/data`; `grep -rl 'data-part="primitive"' frontend/.next/static \| wc -l`; rồi với `gbuild` lại `/dev/ui` | Production: `404` cả hai; `0` tệp bundle chứa khối; `gbuild`: `/dev/ui` `200` |
| TC-PU02-41 | AC13 | `/dev/ui` Composer + Field | **A** gõ chuỗi dài vào Composer và một `Field`; `context.setOffline(true)`; bấm gửi | Giá trị ô **không đổi** (`toHaveValue`); thông báo dễ hiểu (không mã lỗi trần); nút bấm lại được; sau `setOffline(false)` `Gửi lại` gửi được |
| TC-PU02-42 | AC13 | – | **P** `$PW dev-ui.spec.ts -g 'offline-input'; echo rc=$?` | `rc=0` |
| TC-PU02-43 | AC14 | – | **A** ở `/dev/ui`: với mỗi `[data-part=work-region]` đếm `.primary` hiện; đếm hành động phụ hiện sẵn | ≤ 1 primary; ≤ 2 phụ hiện sẵn (còn lại trong `OverflowMenu`) |
| TC-PU02-44 | AC14 | build gate | **A** quét mọi route × vai (cookie `ep_demo_*`) và các hộp thoại / ngăn mở được bằng một thao tác: đếm `.primary` đang hiện trong `main > section`, `[role=dialog]`, `[data-part=work-region]`, `form`; `jq length frontend/e2e/primary-allow.json` | Không vùng nào > 1; `primary-allow.json` ≤ `5` mục, mỗi mục có route + vùng + lý do không rỗng |
| TC-PU02-45 | AC14 | – | **P** `$PW dev-ui.spec.ts -g 'one-primary'` và `$PW ui-foundation.spec.ts -g 'one-primary sweep'` | `rc=0` |
| TC-PU02-46 | AC15 | `/dev/ui` + `docs/design/edupilot-ui-v3.html` | **T** trả lời 10 điều kiện `DESIGN.md` §22 cho `/dev/ui`; đếm card lồng card, nút đỏ đặc / vùng, nghĩa của chỗ đỏ, diện tích đỏ (đo bằng canvas) | Cả 10 "có"; 0 card lồng card; ≤ 1 đỏ đặc / vùng; mọi chỗ đỏ có nghĩa; diện tích đỏ < 8 %; bảng ghi vào `report-US-PU-02.md`, ảnh 1440 + 375 đính kèm |
| TC-PU02-47 | AC11 (mắt) | – | **T** đọc to mọi chuỗi của `/dev/ui` mà SV có thể thấy | Không từ kỹ thuật (bảng dịch `DESIGN.md` §13); nút là động từ |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Trạng thái N/A thêm bớt tuỳ ý | 03 |
| Disabled vẫn kích hoạt handler | 08 |
| Spinner giữa trang / mã lỗi trần | 10, 12 |
| Tràn chữ Việt, zoom 200 % (640 px) | 14–17 |
| Cuộn nhanh gọi trùng trang; lỗi trang kế xoá dòng cũ | 23, 25 |
| Bẫy focus; `Esc` khi đang xử lý không đảo ngược | 26, 27 |
| Bàn phím ảo che ô nhập | 33 |
| Mạng đứt giữa lúc gõ | 41 |
| `/dev/*` lọt vào bản production | 40 |
| Vỡ màn mock | 39 |

## Câu hỏi cho BA / PM
- **Q-QC-PU02-1** — AC2 "dấu hiệu không chỉ là màu" của `selected`: máy đo được `aria-*` nhưng không chứng minh "không chỉ màu"; TC-PU02-07 dùng ảnh + mắt. Chấp nhận kết luận tay cho phần này? — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Không cần kết luận bằng mắt nữa: AC2 đã sửa (v1.1) thành hai phép đo — ít nhất một thuộc tính tính toán (`font-weight`, `text-decoration`, độ rộng viền / vạch, icon `svg`) khác `default`, và ảnh thang xám của `selected` khác `default` > 0,5 % điểm ảnh. Phần "mắt" chỉ bổ sung.
- **Q-QC-PU02-2** — AC5 đo p95 khung 25 ms "CI chỉ ghi, không chặn": QC cũng chỉ ghi số (không FAIL). Đúng ý? — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Đúng ý: p95 khung 25 ms chỉ **ghi số** vào báo cáo, không FAIL (AC5 ghi "CI chỉ ghi, không chặn").

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md (FEAT-ui-foundation, APPROVED 2026-10-03).

Tổng: 47 TC.
