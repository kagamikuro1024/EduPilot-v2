# QC test case — US-AS-08 (`/attendance` mở điểm danh, màn chiếu `/attendance/display`)
Nguồn: US.md + SRS §7 + `DESIGN.md` §14.8, §14.8.1 (màn chiếu), §23.1 (P-108-8), P-108-7. Hộp đen; quy ước: `tc-00-chung.md`. Playwright thật trên `ep-qc` (QC_REAL), TEACHER `teacher@`, TA `ta@`, SV `sv.gioi`. Mỗi TC UI chạy **1440×900 + 375×812**, trừ màn chiếu (1280×720 + 375). Lớp QC-AS, `rotate_seconds=5`.

## `/attendance` — nhân viên
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS08-01 | AC1 | Lớp bật `self_checkin`; buổi chưa có cửa sổ | TEACHER mở `/attendance`, chọn buổi | Hành động chính `Lưu điểm danh`; một hành động phụ `Mở điểm danh`; ≤ 1 nút primary (cổng một-primary) | UI |
| TC-AS08-02 | AC1 | Lớp **tắt** `self_checkin` | Mở `/attendance` | Không có `Mở điểm danh`, không dòng cửa sổ; trang y nguyên trước sprint (so ảnh với bản chụp 10.7) | UI |
| TC-AS08-03 | AC1 | – | Bấm `Mở điểm danh` | `POST …/checkin-windows` (có `Idempotency-Key`) rồi chuyển **cùng tab** tới `/attendance/display?course=…&session=…` | E2E |
| TC-AS08-04 | AC1, AC6 | Cửa sổ đang mở | Quay lại `/attendance` | `Mở điểm danh` **ẩn** (không phải disabled); có dòng trạng thái | UI |
| TC-AS08-05 | AC3 | Cửa sổ mở, 3/30 SV đã điểm danh | Quan sát dòng cửa sổ | Một dòng tĩnh dưới thanh công cụ trong panel (không panel thứ hai, không toast): «Đang mở · 3/30 · còn mm:ss» + `Mở màn chiếu`, `Đóng`; vòng ≥ 2 thêm «Vòng 2» | UI |
| TC-AS08-06 | AC3 | – | SV điểm danh thêm khi staff đang xem | Dòng và hàng làm mới sau ≤ `rotate_seconds`; tab ẩn → ngừng làm mới (quan sát số request) | E2E |
| TC-AS08-07 | AC3 | Cửa sổ mở | Bấm `Đóng` | Không hộp xác nhận; dòng thành «Đã ghi {a} có mặt · {b} muộn · {u} chưa điểm danh» tại chỗ, **không toast**; kèm `Đánh vắng {u} SV chưa điểm danh` khi u>0 | UI |
| TC-AS08-08 | AC3 | – | Bấm `Đánh vắng {u} SV…` | Hộp `ConfirmIrreversible` chuẩn: tiêu đề «Đánh vắng {u} sinh viên?», hệ quả «{u} sinh viên chưa điểm danh thành Vắng.», xác nhận `Đánh vắng`; sau đó dòng cập nhật; có thể sửa từng hàng bằng lưới | UI |
| TC-AS08-09 | AC4 | Buổi có cửa sổ, SV A đã điểm danh 07:03, SV B chưa | Đọc lưới | Hàng A: «Điểm danh lúc 07:03»; hàng B: ô trạng thái **rỗng**, «Chưa điểm danh» (không mặc định `Có mặt`, khác buổi không có cửa sổ); chip tối đa 2 rồi `+n`; không nền đỏ cho cờ (chữ hổ phách + chấm vàng) | UI |
| TC-AS08-10 | AC4 | Có SV `channel=CODE`, SV có cờ, SV mức 1 | Đọc chip | «Nhập mã» (trung tính) cho `CODE`; nhãn cờ đúng SRS 4.4.1; «Chưa xác minh» cho mức 1 | UI |
| TC-AS08-11 | AC4 | – | Bộ lọc `Cần xem ({k})` | `filter=review` vào URL; chỉ hàng còn cờ chưa bỏ; link từ `Hôm nay` (`CHECKIN_REVIEW`) mở đúng bộ lọc | E2E |
| TC-AS08-12 | AC4 | Hàng có cờ | Bấm `Đã xem` | Gọi `clear-flags` cho các lượt của người đó trong buổi; cờ biến khỏi bộ lọc; `audit_log`; không có nút hoàn tác | E2E+DB |
| TC-AS08-13 | AC4 | Sửa một ô trạng thái | Sửa `Vắng`→`Có mặt` | Lưu tại chỗ, `version`, «Hoàn tác» 5 giây như trước sprint | UI |
| TC-AS08-14 | AC5 | Vòng 1 đã đóng | Menu `⋯` | `Mở vòng 2`, `Nhập số đầu người`, `Gọi ngẫu nhiên 5 người`; vòng 3 xong → `Mở vòng …` ẩn | UI |
| TC-AS08-15 | AC5 | – | `Nhập số đầu người` | Đuôi dòng thành ô số `Số đầu người` (0–2000), **tự lưu** (không nút lưu); lớn hơn dung sai → «Số điểm danh nhiều hơn số người đếm» (cảnh báo, không chặn) | UI |
| TC-AS08-16 | AC5 | – | `Gọi ngẫu nhiên 5 người` | `Drawer` «Gọi ngẫu nhiên» 5 hàng (tên, MSSV, «Có cờ»/«Ngẫu nhiên»); không gì khác | UI |
| TC-AS08-17 | AC6 | – | Trạng thái | Tải = khung xương; lỗi = nguyên nhân + `Thử lại`; mở lúc đã có cửa sổ (`409`) → «Đang có cửa sổ mở» + `Mở màn chiếu` | UI |
| TC-AS08-18 | AC6 | `PUT …/attendance complete:true` còn SV chưa dòng | Bấm `Lưu điểm danh` | `InlineNotice` cảnh báo **đầu panel** «Còn {n} sinh viên chưa điểm danh» + `Đánh vắng {n} SV chưa điểm danh` (cùng hộp xác nhận); các ô đã sửa còn nguyên | UI |
| TC-AS08-19 | AC6 | 375 px | Mở `/attendance` có cửa sổ | Lưới là danh sách hàng; dòng cửa sổ xuống 2 dòng (chữ, rồi hai nút ≥ 44 px); `Lưu điểm danh` dock dưới (≥ 720 px không dock); `Mở điểm danh` nằm trong «Thêm»; không tràn ngang | UI |
| TC-AS08-20 | AC7 | Mất mạng khi `Đóng` (Playwright `context.setOffline(true)`) | Bấm `Đóng` rồi bật mạng | Dòng «Chưa đóng» (cảnh báo), không nút; tự thử lại cùng `Idempotency-Key` (kiểm: một cửa sổ đóng, không lần ghi đôi) | E2E |
| TC-AS08-21 | AC7 | Hai tab cùng đóng (đua) | Đóng ở tab 1 và tab 2 | Tab 2 `409`/`200` → tải lại trạng thái; ô đã nhập không mất | E2E |
| TC-AS08-22 | AC8 | – | STUDENT mở `/attendance` và `/attendance/display`; ADMIN cũng vậy | Màn chặn chuẩn; **0 request** tới `/api/v1` từ trang (đếm bằng `page.on('request')`) | E2E |
| TC-AS08-23 | AC8 | TA | Mở `/attendance` | Thấy đủ nút như TEACHER; `PUT …/settings` bằng TA `403` | E2E+API |
| TC-AS08-24 | AC9 | – | `bash scripts/ui-antipatterns.sh` + `--selftest` | Sạch; chỉ token `--ep-*`; không chữ giải thích thừa (luật cứng PO) | DIFF |
| TC-AS08-25 | – | Audit khung 10.7 | `qc-ui-audit-real.spec.ts` thêm route `/attendance` có cửa sổ mở (data) | Không lệch E/H/P/D mới (1440 + 375) | UI |

## `/attendance/display` — màn chiếu (DESIGN §14.8.1; P-108-7, P-108-8)
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-AS08-30 | AC2 | Cửa sổ mở | TEACHER mở màn chiếu 1280×720 | Toàn màn hình, **ngoài khung ứng dụng** (không sidebar / topbar / bottom nav / `Page`); `h1` ẩn «Màn chiếu điểm danh»; QR SVG (`uqr`) vuông bên trái; bên phải: nhãn «Mã điểm danh», mã ngắn 6 ký tự dạng `ABC 123`, số đã điểm danh «87 / 120» (`/ {headcount}` chỉ khi có), «còn 07:42 · Vòng 2» (vòng chỉ khi ≥ 2), `Đóng điểm danh`, `Thoát`; **không đỏ** ở đâu cả | UI |
| TC-AS08-31 | AC2 | – | Đọc tính toán CSS của mã ngắn / số đếm / meta | Ba token `--ep-text-projector-code/count/meta` ≈ 14vmin / 8vmin / 4vmin (≈ 101 / 58 / 29 px ở 720): sai số ≤ 2 px | UI |
| TC-AS08-32 | AC2 | – | Quét QR bằng `jsQR` / `zxing` (Node) từ ảnh chụp màn chiếu 1280×720 và 1920×1080 | Giải ra đúng `qr_url`; sửa URL `…#token` xác thực với API | E2E |
| TC-AS08-33 | AC2 | `rotate_seconds=5` | Quan sát 30 s | QR và mã ngắn **đổi cùng lúc** mỗi 5 s (không lệch), đổi tức thời (không transition); ≤ 12 request `display`/phút (6/phút ở 10 s); không SSE / WebSocket | E2E |
| TC-AS08-34 | AC2 | Lệch đồng hồ máy: đặt `Date` của trang lệch +60 s (Playwright `clock`) | Quan sát | Vẫn đổi đúng nhịp máy chủ (bù `server_time`); đồng hồ đếm ngược đúng giờ máy chủ | E2E |
| TC-AS08-35 | AC2 | – | Quan sát `navigator.wakeLock` (bắt lời gọi) | Gọi `wakeLock.request("screen")` khi cửa sổ mở; thất bại được bỏ qua (không lỗi hiển thị) | E2E |
| TC-AS08-36 | P-108-7 | Cửa sổ mở, `rotate_seconds=5` | Mất mạng (`setOffline`) | Giữ QR + mã ngắn **tới** `step_expires_at + rotate_seconds` theo giờ máy chủ (≤ ~10 s), **sau đó** thay bằng ô vuông **cùng cỡ** + «Mất kết nối» (kiểu cảnh báo, không đỏ); không giữ QR cũ quá hạn | E2E |
| TC-AS08-37 | P-108-7 | – | Bật mạng lại | Thử lại sau 1, 2, 4 s (trần `rotate_seconds`); QR mới trở lại, «Mất kết nối» biến mất | E2E |
| TC-AS08-38 | AC2 | – | Đóng cửa sổ từ lưới / hết giờ | QR thành «Đã đóng» (cỡ số đếm) kèm số cuối; chỉ còn `Thoát` | E2E |
| TC-AS08-39 | AC2 | – | `Esc`; `Thoát` | Về `/attendance?course=&session=` | E2E |
| TC-AS08-40 | AC2 | – | `Đóng điểm danh` | `close`; chuyển «Đã đóng» | E2E |
| TC-AS08-41 | AC6 | Tải đầu / lỗi đầu | Chặn `display` bằng `route.abort` lúc tải đầu | `Skeleton` vuông cỡ QR → `InlineNotice` «Không tải được mã» + `Thử lại` | UI |
| TC-AS08-42 | AC6 | Hai máy cùng mở màn chiếu (T10) | Mở trên hai trình duyệt | Cả hai hiện cùng QR (không chặn; rủi ro chấp nhận ghi ở SRS 8) | E2E |
| TC-AS08-43 | AC2 | Portrait / 375 px | Mở màn chiếu | Các khối xếp dọc cùng thứ tự; dùng được; không tràn ngang | UI |
| TC-AS08-44 | AC8 | STUDENT / ADMIN | Mở `/attendance/display` | Màn chặn chuẩn, 0 request tới `/api/v1` | E2E |
| TC-AS08-45 | P-108-8 | Mã nguồn | `grep -rn "projector" frontend/src \| grep -v features/attendance/display \| grep -v shared/styles` | 0 chỗ dùng ba token ngoài thư mục `features/attendance/display/` (và `shared/styles/` nơi định nghĩa); `ui-antipatterns.sh` kiểm 32 bắt khi chèn thử ở nơi khác (`--selftest`) | DIFF |
| TC-AS08-46 | P-108-8 | – | `frame.spec.ts` | Có `/checkin` (loại `read`) trong danh sách đo; `/attendance/display` được liệt kê là **route trình chiếu ngoài khung** (DESIGN §23.1), không bị đo khung chuẩn | E2E |
| TC-AS08-47 | AC9 | Lighthouse / axe | `/attendance/display` ở trạng thái mở | axe 0 lỗi nghiêm trọng; LCP theo cổng 10.7 (trạng thái «đã quay lại») | UI |
