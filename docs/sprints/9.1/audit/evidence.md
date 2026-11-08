# Bằng chứng UI/UX — sprint 9.1, pha 1 (QC)

Chụp ngày 2026-11-08 trên bản demo VPS (`https://v2202609419695521098.happysrv.de`), trình duyệt Chromium trên Mac (`playwright-core` đi kèm `playwright-cli`), ngôn ngữ vi-VN. Chỉ quan sát, không đề xuất (đề xuất là việc của BA). **483 ảnh** ở `shots/`, số liệu thô ở `data/*.json`, mã chụp ở `scripts/` (chạy lại được).

## 1. Cách chụp và giới hạn

- **Tên ảnh:** `<route>__<vai>__<cỡ>.png` (cỡ 1440 = 1440×900, 375 = 375×812), ảnh dài toàn trang. Trạng thái khác: `<route>@loading|error|empty__…`; hộp thoại `<route>@dlg-<tên>__…`; focus bàn phím `<route>@focus__…`. `ID` trong tên = id thật của một bản ghi seed.
- **Lớp:** mọi màn của giảng viên / trợ giảng / sinh viên mở bằng `?course=` lớp **761987** (đầy dữ liệu) trừ khi ghi khác. Ảnh `@empty` dùng lớp **761988** (ít dữ liệu). Không truyền `?course=` thì giảng viên và sinh viên `sv.gioi` rơi vào lớp 761988 (lớp ít dữ liệu).
- **Tải** (`@loading`): mọi lời gọi `/api/v1/**` (trừ `/auth/`) bị giữ 8 s; ảnh chụp khung nhìn lúc ~1,8 s sau khi tải trang. **Lỗi** (`@error`): mọi lời gọi `/api/v1/**` (trừ `/auth/`) bị huỷ như mất mạng; trang vẫn nạp được. **Rỗng**: lớp 761988 và tài khoản `sv.moi`.
- **Đo:** *nội dung* = ms từ lúc điều hướng tới khi `<main>` có chữ và hết khung xương (ước lượng); *LCP* theo `PerformanceObserver`; axe 4.11 với thẻ wcag2a/2aa/21aa/22aa, cột `axe` = nghiêm trọng/cao/trung bình/nhẹ (số nút). Nhiều trình duyệt chạy song song nên LCP và nội dung dao động ±vài trăm ms.
- **Ảnh dài:** không dùng `fullPage` của Playwright vì nó đôi khi vẽ lệch nội dung ~24 px sang trái (đã kiểm: vị trí tiêu đề trong DOM luôn 240 px, `data/extra-layout.json`, 34 route × 2 lần; ảnh `fullPage` lệch ở ~1/5 lượt). Thay vào đó khung nhìn được kéo dài bằng chiều cao trang (tối đa 9000 px), nên sidebar và thanh điều hướng dưới nằm sát đáy ảnh. Bảng có khung cuộn riêng (Sổ điểm, Sinh viên) chỉ hiện phần trong khung.
- **Không chạm dữ liệu demo:** không bấm Lưu / Duyệt / Công bố / Chốt / Xoá / Nhận / Mời / Tạo. Hộp thoại chỉ mở rồi đóng bằng Esc. Mọi request không phải GET trong lúc mở hộp thoại: xem đầu mục 4 (0 request).

## 2. Tổng hợp

| Chỉ số | Giá trị |
| --- | --- |
| Lượt chụp | 419 (đầy 158, tải 116, lỗi 116, rỗng 29) |
| Lượt không chụp được | 0 |
| axe ở trạng thái đầy (nút) | nghiêm trọng 0 · cao 0 · trung bình 0 · nhẹ 0 |
| Lượt có lỗi / cảnh báo console (đầy) | 109 |
| Lượt có lời gọi API ≥ 400 (đầy) | 16 |
| Lượt tràn ngang ở 375 | 0 |

**Luật axe** (số lượt chụp trạng thái đầy có vi phạm): không có vi phạm nào

**Thông điệp console lặp nhiều nhất** (trạng thái đầy): «warning: The resource ‹url› was preloaded using link preload but not used within a few seconds from the window» ×108; «error: Failed to load resource: the server responded with a status of 401 ()» ×16; «error: Failed to load resource: the server responded with a status of 410 ()» ×2

**Phản hồi HTTP ≥ 400** (trạng thái đầy): 401 POST /api/v1/auth/refresh ×16; 410 POST /api/v1/auth/tokens/preview ×2

## 3. Từng màn, theo vai

### Khách (chưa đăng nhập)

**Đầy dữ liệu** (lớp 761987)

| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 1440 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 1225 | 1476 | [ảnh](shots/home__anon__1440.png) | 2 thẻ h1 · bản mới: video có nút ▶ và phụ đề in sẵn trong poster che phần dưới ảnh Chấm bài; sơ đồ 3 bước, 3 số đo, tab Giảng viên / Sinh viên với ảnh ứng dụng bên phải (ảnh vẫn nhỏ, chữ trong ảnh không đọc được); ảnh chụp lúc 19:28, sau khi demo cập nhật |
| `/` | 375 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 335 | 312 | [ảnh](shots/home__anon__375.png) | 2 thẻ h1 · 3 điều khiển < 24 px · ẢNH DÀI BỊ TRẮNG phía dưới hero: hiệu ứng hiện theo cuộn không chạy khi khung nhìn được kéo dài hết trang (cách chụp), nên không dùng ảnh này để đánh giá nội dung; xem `landing-scroll-motion-375-*` (chụp khung nhìn thật khi cuộn) |
| `/login` | 1440 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 831 | 836 | [ảnh](shots/login__anon__1440.png) | thẻ đăng nhập rộng ~440 px ở giữa trên nền trống 1440×900; dòng «Cần giúp? Liên hệ giảng viên hoặc quản trị viên của bạn.» ở đáy trang, chữ nhỏ |
| `/login` | 375 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 341 | 372 | [ảnh](shots/login__anon__375.png) |  |
| `/register` | 1440 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 774 | 528 | [ảnh](shots/register__anon__1440.png) | biểu mẫu 4 ô, hai dòng gợi ý chữ nhỏ (dưới ô mã số sinh viên và dưới ô mật khẩu); nút «Tạo tài khoản» hồng nhạt (khoá) khi chưa nhập |
| `/register` | 375 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 313 | 272 | [ảnh](shots/register__anon__375.png) |  |
| `/forgot-password` | 1440 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 779 | 548 | [ảnh](shots/forgot-password__anon__1440.png) |  |
| `/forgot-password` | 375 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 313 | 300 | [ảnh](shots/forgot-password__anon__375.png) |  |
| `/reset-password` | 1440 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 795 | 804 | [ảnh](shots/reset-password__anon__1440.png) | mở không kèm mã nhận ngay «Liên kết không dùng được» + nút đỏ «Yêu cầu liên kết mới» |
| `/reset-password` | 375 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 778 | 780 | [ảnh](shots/reset-password__anon__375.png) |  |
| `/verify-email` | 1440 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 842 | 848 | [ảnh](shots/verify-email__anon__1440.png) | mở không kèm mã nhận «Liên kết không dùng được» + ô email gửi lại (nút xám khoá) |
| `/verify-email` | 375 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 783 | 784 | [ảnh](shots/verify-email__anon__375.png) |  |
| `/invite/bogus-token-000` | 1440 | error: Failed to load resource: the server responded with a status of 401 () ¦ error: Failed to load resource: the server responded with a status of 410 () | 401 POST /api/v1/auth/refresh ¦ 410 POST /api/v1/auth/tokens/preview | 0/0/0/0 | 1124 | 1144 | [ảnh](shots/invite-bogus-token-000__anon__1440.png) |  |
| `/invite/bogus-token-000` | 375 | error: Failed to load resource: the server responded with a status of 401 () ¦ error: Failed to load resource: the server responded with a status of 410 () | 401 POST /api/v1/auth/refresh ¦ 410 POST /api/v1/auth/tokens/preview | 0/0/0/0 | 1039 | 1040 | [ảnh](shots/invite-bogus-token-000__anon__375.png) |  |
| `/khong-ton-tai` | 1440 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 1350 | 1344 | [ảnh](shots/khong-ton-tai__anon__1440.png) |  |
| `/khong-ton-tai` | 375 | error: Failed to load resource: the server responded with a status of 401 () | 401 POST /api/v1/auth/refresh | 0/0/0/0 | 1294 | 1292 | [ảnh](shots/khong-ton-tai__anon__375.png) |  |

**Đang tải** (API chậm 8 s)

| Route | Ảnh | Ghi nhận |
| --- | --- | --- |
| `/` | [ảnh](shots/home@loading__anon__1440.png) |  |
| `/login` | [ảnh](shots/login@loading__anon__1440.png) |  |
| `/register` | [ảnh](shots/register@loading__anon__1440.png) |  |
| `/forgot-password` | [ảnh](shots/forgot-password@loading__anon__1440.png) |  |
| `/reset-password` | [ảnh](shots/reset-password@loading__anon__1440.png) |  |
| `/verify-email` | [ảnh](shots/verify-email@loading__anon__1440.png) |  |
| `/invite/bogus-token-000` | [ảnh](shots/invite-bogus-token-000@loading__anon__1440.png) |  |
| `/khong-ton-tai` | [ảnh](shots/khong-ton-tai@loading__anon__1440.png) |  |

**Lỗi** (mất mạng)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/home@error__anon__1440.png) | 2 thẻ h1 |
| `/login` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/login@error__anon__1440.png) |  |
| `/register` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/register@error__anon__1440.png) |  |
| `/forgot-password` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/forgot-password@error__anon__1440.png) |  |
| `/reset-password` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/reset-password@error__anon__1440.png) |  |
| `/verify-email` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/verify-email@error__anon__1440.png) |  |
| `/invite/bogus-token-000` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/invite-bogus-token-000@error__anon__1440.png) |  |
| `/khong-ton-tai` | error: Failed to load resource: the server responded with a status of 401 () | 0/0/0/0 | [ảnh](shots/khong-ton-tai@error__anon__1440.png) |  |

### Giảng viên `teacher@`

**Đầy dữ liệu** (lớp 761987)

| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 1440 | 0 | 0 | 0/0/0/0 | 319 | 268 | [ảnh](shots/home__teacher__1440.png) | «Hôm nay» giảng viên: 9 dòng việc, dòng nào cũng một chữ «Mở» đỏ nhỏ ở mép phải; 6/9 dòng cùng chấm xám giống nhau · ba khối (Việc cần xử lý / Lớp cần chú ý / Sắp tới) đều là Panel trắng dài ~960 px, chừa trống ~240 px bên phải ở 1440 |
| `/` | 375 | 0 | 0 | 0/0/0/0 | 285 | 272 | [ảnh](shots/home__teacher__375.png) | nút hành động mỗi dòng là chữ đỏ «Mở» / «Xem bài» cỡ nhỏ nằm riêng một dòng dưới mô tả, vùng bấm thấp (~20 px) |
| `/students` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 2013 | 1200 | [ảnh](shots/students__teacher__1440.png) | bảng sinh viên cuộn trong khung riêng; cột «Phát biểu» hiện «1 · +0,25» không có nhãn đơn vị · cột «Hoạt động học» lặp «14 phút · 0 câu hỏi» ở hầu hết hàng |
| `/attendance` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 2305 | 2276 | [ảnh](shots/attendance__teacher__1440.png) | 30 hàng × 5 cột nút tròn; cả 30 dòng đã chọn sẵn «Có mặt» (chấm đỏ đặc) nên cột «Có mặt» là một cột đỏ liền; nút «+ phát biểu · +0,25» lặp 30 lần cùng chữ · tiêu đề cột có ký hiệu phím tắt «1 2 3 4 P» chữ rất nhỏ; trạng thái «Chưa có thay đổi» ở góc phải panel; «Lưu điểm danh» (đỏ) ở góc trên phải ngoài tầm mắt khi cuộn xuống hàng cuối |
| `/attendance` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1948 | 1948 | [ảnh](shots/attendance__teacher__375.png) | 5 cột lựa chọn thu thành một chữ «Có mặt» xám bên phải mỗi dòng; không thấy điều khiển nào để đổi sang Muộn / Vắng phép / Vắng, không thấy nút «+ phát biểu»; «Lưu điểm danh» nằm trên đầu danh sách 30 dòng |
| `/gradebook` | 1440 | 0 | 0 | 0/0/0/0 | 737 | 504 | [ảnh](shots/gradebook__teacher__1440.png) | bảng điểm nằm trong khung cuộn riêng cao ~560 px; hàng cuối bị cắt giữa chừng, trang dưới bảng trống · ô điểm là ô nhập số ngay trong bảng; cột «Kiểm tra hằng tuần» toàn «–» · «Chốt điểm» (đỏ) cạnh «Xem công thức» và nút «…» ở góc trên phải |
| `/grading` | 1440 | 0 | 0 | 0/0/0/0 | 1061 | 1604 | [ảnh](shots/grading__teacher__1440.png) | chữ bị cắt «…»: "" · «Chấm tất cả» hồng nhạt (trông bị khoá) khi hàng chờ có 4 bài; không thấy lý do khoá · cột «Bài tập» ngắt 3 dòng cho cùng một tên bài ở mọi hàng; nửa dưới khung trống · «Điểm» 0,00 đậm cho bài trống / lạc đề; thời gian nộp chữ rất nhỏ, xám nhạt |
| `/questions` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1467 | 1224 | [ảnh](shots/questions__teacher__1440.png) | có câu mô tả dưới tiêu đề «Câu hỏi dùng cho bài thi và luyện đề của lớp 761987. Chỉ giảng viên và trợ giảng thấy.» + dòng nhỏ «5 câu chờ duyệt» · 5 câu đều tên «mẫu» chờ duyệt; 30 hàng liền, không phân trang; nút «Tạo câu hỏi» đỏ có dấu + |
| `/exams` | 1440 | 0 | 0 | 0/0/0/0 | 1473 | 1232 | [ảnh](shots/exams__teacher__1440.png) | có câu mô tả dưới tiêu đề «Bài kiểm tra tính điểm của lớp 761987: soạn từ ngân hàng câu hỏi, đặt khung giờ rồi lên lịch.» · chỉ có nhóm «Đã đóng»; hàng bài thi có «Xem kết quả» dạng chữ không viền và dòng nhỏ «6 sinh viên đã bắt đầu» |
| `/threads` | 1440 | 0 | 0 | 0/0/0/0 | 738 | 504 | [ảnh](shots/threads__teacher__1440.png) | tiêu đề nhiều thread là đoạn chữ cắt từ tài liệu (tiếng Anh, bắt đầu bằng «·», «- -», «• »), cả tiêu đề lẫn đoạn xem trước cùng cỡ chữ trên nhiều dòng · mỗi dòng có ngày «00:10 02-11» rất nhỏ, tên tác giả + «Tuần 3» + trạng thái xếp ngang một dòng nhỏ |
| `/documents` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1749 | 1208 | [ảnh](shots/documents__teacher__1440.png) | chữ trong ô chọn «Loại» bị cắt («Quy chế m…», «Đề tham k…») · cột «Cập nhật» ghi «00:10 02-11» (giờ trước, ngày-tháng sau), khác định dạng «Thứ Năm, 29/10 · 00:10» ở Chấm bài · hai cột công tắc «Dùng cho AI» / «Hiện cho sinh viên» ở mọi dòng; dòng «Đáp án…» có hai dòng chữ nhỏ giải thích dưới tên |
| `/calendar` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1762 | 1224 | [ảnh](shots/calendar__teacher__1440.png) | ba nút ngang hàng ở góc trên phải («Thêm vào lịch», «Thêm sự kiện», «Tạo lịch buổi học») tên gần giống nhau · ô «Chủ nhật» (hôm nay) tô nền hồng nhạt cả cột; ngày trống hiện «—»; chỉ 1 tuần, phần dưới trang trống |
| `/insights` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 999 | 1016 | [ảnh](shots/insights__teacher__1440.png) | mỗi tiêu đề khối có câu mô tả xám bên dưới («Câu hỏi của lớp, đã bỏ tên và mã sinh viên…», «Câu hỏi trong 7 ngày liền trước đó, đã ẩn danh · xếp theo mức độ…») và thêm một biểu ngữ xanh về ẩn danh · năm thẻ chủ đề cao ~250 px mỗi thẻ nằm trong Panel lớn (thẻ trong thẻ); mỗi thẻ có đoạn «Nên làm:» nhãn đỏ + hai nút «Tạo thread ghim» / «Tạo buổi ôn tập» lặp 5 lần; trang dài > 2000 px · cột nội dung hẹp hơn các trang khác, nút «Tạo báo cáo mới» (đỏ) nằm ngoài lề phải của cột |
| `/analytics` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1009 | 1024 | [ảnh](shots/analytics__teacher__1440.png) | mỗi khối có tiêu đề + một dòng mô tả xám, rồi một câu tóm tắt in đậm một phần trong Panel (ba lớp chữ trước khi tới số) · biểu đồ đường: nhãn «47 câu» chồng lên đường ở đầu phải; nhãn trục ngày T6 T7 CN T2… không ghi ngày |
| `/class/members` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1486 | 1256 | [ảnh](shots/class-members__teacher__1440.png) | có câu mô tả dưới tiêu đề: «Lớp 761987. Duyệt yêu cầu vào lớp và quản lý người học.» · cột «Vào lớp» mọi hàng cùng «Danh sách đầu lớp» chữ rất nhỏ xám; email dưới tên cũng rất nhỏ · 30 hàng liền, không phân trang; trang dài ~1470 px |
| `/class/settings` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1522 | 1520 | [ảnh](shots/class-settings__teacher__1440.png) | dưới gần mọi ô nhập có một dòng giải thích nhỏ («Để trống nếu không giới hạn» lặp 2 lần, ví dụ miền email, mô tả công tắc) · mã lớp cỡ rất lớn chữ đơn cách; «Tạo lại mã» là nút chữ không viền cạnh hai nút có viền · 5 thanh trượt đỏ liền nhau cùng màu đỏ + công tắc đen; đỏ xuất hiện ở thanh trượt và mọi nút chính |
| `/gradebook/scheme` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1529 | 1540 | [ảnh](shots/gradebook-scheme__teacher__1440.png) | cột phải «Quy chế học phần» gần như trống (chỉ có tiêu đề, không hiện đoạn quy chế); mỗi mục có huy hiệu «Trang 1» / «Trang 2» · mỗi mục hiện hai lớp: dòng in đậm tóm tắt và đoạn giải thích trong khung xám bên dưới |
| `/settings` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1530 | 1068 | [ảnh](shots/settings__teacher__1440.png) | «Thiết bị đang đăng nhập» liệt kê ~25 phiên liền nhau, không phân trang, nhãn «Chrome trên macOS» / «Trình duyệt khác» giống nhau (phần lớn là các phiên đăng nhập của đợt chụp QC này) · «Đổi mật khẩu» hồng nhạt (khoá) khi chưa nhập; ba ô mật khẩu rộng ~370 px trong Panel rộng ~850 px |
| `/students/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1759 | 1744 | [ảnh](shots/students-ID__teacher__1440.png) |  |
| `/exams/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1494 | 1500 | [ảnh](shots/exams-ID__teacher__1440.png) | biểu mẫu dài ~1000 px với 5 nhóm; bài đã lên lịch nên hầu hết ô bị khoá (xám) chỉ có một biểu ngữ ở đầu trang giải thích; «Lưu thông tin» xám · dưới gần mọi ô có dòng chữ nhỏ giải thích («Hỗ trợ Markdown cơ bản…», «5–300, không dài hơn khung giờ.», «0 = không nhận yêu cầu xem lại.») · ngày giờ viết «02/11/2026, 00:07» trong ô; ở tiêu đề phụ viết «00:07 02/11 – 00:09 02/11» |
| `/exams/‹id›/results` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1557 | 1032 | [ảnh](shots/exams-ID-results__teacher__1440.png) | ô «Tìm theo tên hoặc MSSV» chỉ rộng ~110 px, nhãn nằm trên và không có biểu tượng tìm; hai nút trắng «Xuất CSV» / «Chấm lại» cạnh bộ lọc · cột «Nộp lúc» ghi «00:07 02/11» (khác «00:07 02-11» ở Tài liệu); dòng «Bùi Khánh Việt» in đậm (không rõ vì sao) |
| `/exams/‹id›/similarity` | 1440 | 0 | 0 | 0/0/0/0 | 1267 | 1032 | [ảnh](shots/exams-ID-similarity__teacher__1440.png) | có câu giải thích dài trong Panel «Độ giống chỉ là gợi ý — nhiều bài đúng cùng một cách làm…»; chỉ 1 dòng kết quả; nửa dưới trang trống |
| `/grading/‹id›` | 1440 | 0 | 0 | 0/0/0/0 | 1514 | 1524 | [ảnh](shots/grading-ID__teacher__1440.png) | bài làm bên trái, khung chấm bên phải bị cắt giữa tiêu chí thứ hai (khung chấm cuộn riêng, cao ~600 px); hộp «Duyệt bài» (đỏ) và «Đã duyệt 0/22» dính đáy khung · tên tiêu chí hiện hai lần cho mỗi dòng («Xác định tác nhân…» bên trái và «Điểm Xác định tác nhân…» phía trên ô điểm); hai khung «Đoạn trích từ bài làm · bấm để xem trong bài» nền hồng nhạt |
| `/threads/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1723 | 1732 | [ảnh](shots/threads-ID__teacher__1440.png) | câu trả lời AI hiện chuỗi «<ngữ_cảnh>» nguyên văn trong nội dung; «Độ tin cậy 0,80» hiện cho giảng viên · nội dung bài đăng hiện nguyên văn «- - Unstructured threats» (gạch đầu dòng lặp); nút «Đăng» hồng nhạt (khoá) dưới ô bình luận |
| `/assignments/‹id›` | 1440 | 0 | 0 | 0/0/0/0 | 1018 | 1020 | [ảnh](shots/assignments-ID__teacher__1440.png) | giảng viên mở /assignments/‹id› nhận «Bạn không có quyền xem màn này — Trang này dành cho sinh viên»; không thấy trang bài tập từ phía giảng viên ở route này |
| `/library` | 1440 | 0 | 0 | 0/0/0/0 | 1010 | 1016 | [ảnh](shots/library__teacher__1440.png) |  |
| `/library/‹id›` | 1440 | 0 | 0 | 0/0/0/0 | 1059 | 1040 | [ảnh](shots/library-ID__teacher__1440.png) |  |
| `/inbox?tab=resolved` | 1440 | 0 | 0 | 0/0/0/0 | 751 | 1988 | [ảnh](shots/inbox-tab-resolved__teacher__1440.png) |  |
| `/class/members?tab=pending` | 1440 | 0 | 0 | 0/0/0/0 | 1102 | 832 | [ảnh](shots/class-members-tab-pending__teacher__1440.png) |  |
| `/threads?view=moderation` | 1440 | 0 | 0 | 0/0/0/0 | 740 | 500 | [ảnh](shots/threads-view-moderation__teacher__1440.png) |  |
| `/questions?review_status=PENDING` | 1440 | 0 | 0 | 0/0/0/0 | 1463 | 1232 | [ảnh](shots/questions-review-status-PENDING__teacher__1440.png) |  |
| `/khong-ton-tai` | 1440 | 0 | 0 | 0/0/0/0 | 1018 | 1028 | [ảnh](shots/khong-ton-tai__teacher__1440.png) | màn «Không tìm thấy trang»: nút «Về Hôm nay» nằm xa mép phải, cách xa dòng giải thích; phần còn lại của trang trống |
| `/observability` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1047 | 1036 | [ảnh](shots/observability__teacher__1440.png) | hai khối cùng số đo (dải 5 chỉ số «Yêu cầu hôm nay 1.284 · p95 3,1 s…» và bảng «Lớp 761987 trong 7 ngày» p95 2,9 s…) liền nhau · danh sách yêu cầu hiện tên model kỹ thuật (gpt-4o-mini, qwen2.5-14b-instruct) và «Lớp int1006-1» dù lớp đang xem là 761987 |
| `/settings/llm` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1742 | 488 | [ảnh](shots/settings-llm__teacher__1440.png) | tài khoản giảng viên thấy trang cấu hình nhưng mọi ô xám và chỉ có dòng «Chỉ quản trị viên được thay đổi cấu hình.»; sáu liên kết đỏ «Cài đặt nâng cao» giống nhau |
| `/settings/integrations` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 996 | 1216 | [ảnh](shots/settings-integrations__teacher__1440.png) | giá trị kỹ thuật thô hiện nguyên văn: «mailpit:1025», «none», «mock-imap:1143», Tenant / Client ID dạng GUID, «Bật · Có»; ba khối SMTP / IMAP / Teams cùng kiểu bảng hai cột, panel chỉ rộng ~640 px (nửa trang) |
| `/inbox` | 1440 | 0 | 0 | 0/0/0/0 | 520 | 1752 | [ảnh](shots/inbox__teacher__1440.png) | «Đang chờ» lặp ở mỗi hàng của danh sách, ở bộ lọc, và ở dòng «Trạng thái» trong chi tiết; chi tiết hiện cả câu trả lời nháp của AI với «Độ tin cậy 0,90» cho giảng viên · hai hành động «Nhận» (nút đỏ) và «Đóng không trả lời» (chữ đỏ không viền) cạnh nhau; thời gian chờ «14 ngày», «6 ngày» chữ nhỏ ở góc phải mỗi hàng |
| `/inbox` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 287 | 292 | [ảnh](shots/inbox__teacher__375.png) | chỉ có danh sách (không có khung chi tiết); hàng được chọn nền hồng + vạch đỏ bên trái; «Đang chờ» lặp mỗi hàng; ba chip trạng thái bộ lọc nằm trong một khung chữ nhỏ |

**Đang tải** (API chậm 8 s)

| Route | Ảnh | Ghi nhận |
| --- | --- | --- |
| `/` | [ảnh](shots/home@loading__teacher__1440.png) |  |
| `/inbox` | [ảnh](shots/inbox@loading__teacher__1440.png) |  |
| `/students` | [ảnh](shots/students@loading__teacher__1440.png) | sau 1,8 s vùng nội dung trống hoàn toàn (không tiêu đề «Sinh viên», không khung xương) |
| `/attendance` | [ảnh](shots/attendance@loading__teacher__1440.png) |  |
| `/gradebook` | [ảnh](shots/gradebook@loading__teacher__1440.png) | sau 1,8 s vùng nội dung hoàn toàn trống: không tiêu đề, không khung xương, không dấu hiệu đang tải (khác trang sinh viên có khung xương) |
| `/grading` | [ảnh](shots/grading@loading__teacher__1440.png) |  |
| `/questions` | [ảnh](shots/questions@loading__teacher__1440.png) |  |
| `/exams` | [ảnh](shots/exams@loading__teacher__1440.png) | có tiêu đề «Bài thi» và khung xương 6 vạch trong Panel (khác Sổ điểm / Sinh viên không có gì) |
| `/threads` | [ảnh](shots/threads@loading__teacher__1440.png) |  |
| `/documents` | [ảnh](shots/documents@loading__teacher__1440.png) |  |
| `/calendar` | [ảnh](shots/calendar@loading__teacher__1440.png) |  |
| `/insights` | [ảnh](shots/insights@loading__teacher__1440.png) |  |
| `/analytics` | [ảnh](shots/analytics@loading__teacher__1440.png) |  |
| `/class/members` | [ảnh](shots/class-members@loading__teacher__1440.png) |  |
| `/class/settings` | [ảnh](shots/class-settings@loading__teacher__1440.png) |  |
| `/gradebook/scheme` | [ảnh](shots/gradebook-scheme@loading__teacher__1440.png) |  |
| `/settings` | [ảnh](shots/settings@loading__teacher__1440.png) |  |
| `/students/‹id›` | [ảnh](shots/students-ID@loading__teacher__1440.png) |  |
| `/exams/‹id›` | [ảnh](shots/exams-ID@loading__teacher__1440.png) |  |
| `/exams/‹id›/results` | [ảnh](shots/exams-ID-results@loading__teacher__1440.png) |  |
| `/exams/‹id›/similarity` | [ảnh](shots/exams-ID-similarity@loading__teacher__1440.png) |  |
| `/grading/‹id›` | [ảnh](shots/grading-ID@loading__teacher__1440.png) |  |
| `/threads/‹id›` | [ảnh](shots/threads-ID@loading__teacher__1440.png) |  |
| `/library/‹id›` | [ảnh](shots/library-ID@loading__teacher__1440.png) |  |
| `/inbox?tab=resolved` | [ảnh](shots/inbox-tab-resolved@loading__teacher__1440.png) |  |
| `/class/members?tab=pending` | [ảnh](shots/class-members-tab-pending@loading__teacher__1440.png) |  |
| `/threads?view=moderation` | [ảnh](shots/threads-view-moderation@loading__teacher__1440.png) |  |
| `/questions?review_status=PENDING` | [ảnh](shots/questions-review-status-PENDING@loading__teacher__1440.png) |  |
| `/khong-ton-tai` | [ảnh](shots/khong-ton-tai@loading__teacher__1440.png) |  |
| `/observability` | [ảnh](shots/observability@loading__teacher__1440.png) |  |
| `/settings/llm` | [ảnh](shots/settings-llm@loading__teacher__1440.png) |  |
| `/settings/integrations` | [ảnh](shots/settings-integrations@loading__teacher__1440.png) |  |
| `/assignments/‹id›` | [ảnh](shots/assignments-ID@loading__teacher__1440.png) |  |
| `/library` | [ảnh](shots/library@loading__teacher__1440.png) |  |

**Lỗi** (mất mạng)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/home@error__teacher__1440.png) |  |
| `/inbox` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/inbox@error__teacher__1440.png) | không có h1 |
| `/students` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/students@error__teacher__1440.png) |  |
| `/attendance` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/attendance@error__teacher__1440.png) | không có h1 · nội dung không hiện sau 25 s |
| `/gradebook` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/gradebook@error__teacher__1440.png) | không có h1 |
| `/grading` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/grading@error__teacher__1440.png) |  |
| `/questions` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/questions@error__teacher__1440.png) | nội dung không hiện sau 25 s |
| `/exams` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams@error__teacher__1440.png) | nội dung không hiện sau 25 s |
| `/threads` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads@error__teacher__1440.png) |  |
| `/documents` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/documents@error__teacher__1440.png) | chữ bị cắt «…»: "" |
| `/calendar` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/calendar@error__teacher__1440.png) |  |
| `/insights` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/insights@error__teacher__1440.png) |  |
| `/analytics` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/analytics@error__teacher__1440.png) |  |
| `/class/members` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/class-members@error__teacher__1440.png) | nội dung không hiện sau 25 s |
| `/class/settings` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/class-settings@error__teacher__1440.png) | nội dung không hiện sau 25 s |
| `/gradebook/scheme` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/gradebook-scheme@error__teacher__1440.png) | không có h1 · nội dung không hiện sau 25 s |
| `/settings` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/settings@error__teacher__1440.png) |  |
| `/students/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/students-ID@error__teacher__1440.png) |  |
| `/exams/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID@error__teacher__1440.png) | nội dung không hiện sau 25 s |
| `/exams/‹id›/results` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID-results@error__teacher__1440.png) |  |
| `/exams/‹id›/similarity` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID-similarity@error__teacher__1440.png) |  |
| `/grading/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/grading-ID@error__teacher__1440.png) | không có h1 · nội dung không hiện sau 25 s |
| `/threads/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads-ID@error__teacher__1440.png) |  |
| `/assignments/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/assignments-ID@error__teacher__1440.png) |  |
| `/library` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/library@error__teacher__1440.png) |  |
| `/library/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/library-ID@error__teacher__1440.png) |  |
| `/inbox?tab=resolved` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/inbox-tab-resolved@error__teacher__1440.png) | không có h1 |
| `/class/members?tab=pending` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/class-members-tab-pending@error__teacher__1440.png) | nội dung không hiện sau 25 s |
| `/threads?view=moderation` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads-view-moderation@error__teacher__1440.png) |  |
| `/questions?review_status=PENDING` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/questions-review-status-PENDING@error__teacher__1440.png) | nội dung không hiện sau 25 s |
| `/khong-ton-tai` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/khong-ton-tai@error__teacher__1440.png) |  |
| `/observability` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/observability@error__teacher__1440.png) |  |
| `/settings/llm` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/settings-llm@error__teacher__1440.png) |  |
| `/settings/integrations` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/settings-integrations@error__teacher__1440.png) |  |

**Rỗng / ít dữ liệu** (lớp 761988)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | 0 | 0/0/0/0 | [ảnh](shots/home@empty__teacher__1440.png) |  |
| `/inbox` | 0 | 0/0/0/0 | [ảnh](shots/inbox@empty__teacher__1440.png) |  |
| `/students` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/students@empty__teacher__1440.png) |  |
| `/attendance` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/attendance@empty__teacher__1440.png) |  |
| `/gradebook` | 0 | 0/0/0/0 | [ảnh](shots/gradebook@empty__teacher__1440.png) |  |
| `/grading` | 0 | 0/0/0/0 | [ảnh](shots/grading@empty__teacher__1440.png) | chữ bị cắt «…»: "" |
| `/questions` | 0 | 0/0/0/0 | [ảnh](shots/questions@empty__teacher__1440.png) |  |
| `/exams` | 0 | 0/0/0/0 | [ảnh](shots/exams@empty__teacher__1440.png) |  |
| `/threads` | 0 | 0/0/0/0 | [ảnh](shots/threads@empty__teacher__1440.png) |  |
| `/documents` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/documents@empty__teacher__1440.png) |  |
| `/calendar` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/calendar@empty__teacher__1440.png) |  |
| `/insights` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/insights@empty__teacher__1440.png) |  |
| `/analytics` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/analytics@empty__teacher__1440.png) |  |
| `/class/members` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/class-members@empty__teacher__1440.png) |  |
| `/class/settings` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/class-settings@empty__teacher__1440.png) |  |
| `/gradebook/scheme` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/gradebook-scheme@empty__teacher__1440.png) |  |
| `/settings` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/settings@empty__teacher__1440.png) |  |
| `/observability` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/observability@empty__teacher__1440.png) |  |
| `/settings/llm` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/settings-llm@empty__teacher__1440.png) |  |
| `/settings/integrations` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/settings-integrations@empty__teacher__1440.png) |  |

### Trợ giảng `ta@`

**Đầy dữ liệu** (lớp 761987)

| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 1440 | 0 | 0 | 0/0/0/0 | 319 | 276 | [ảnh](shots/home__ta__1440.png) |  |
| `/` | 375 | 0 | 0 | 0/0/0/0 | 303 | 292 | [ảnh](shots/home__ta__375.png) |  |
| `/inbox` | 1440 | 0 | 0 | 0/0/0/0 | 765 | 2056 | [ảnh](shots/inbox__ta__1440.png) |  |
| `/inbox` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 303 | 308 | [ảnh](shots/inbox__ta__375.png) |  |
| `/students` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 2146 | 1268 | [ảnh](shots/students__ta__1440.png) |  |
| `/attendance` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 2596 | 2604 | [ảnh](shots/attendance__ta__1440.png) |  |
| `/attendance` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1602 | 1604 | [ảnh](shots/attendance__ta__375.png) |  |
| `/gradebook` | 1440 | 0 | 0 | 0/0/0/0 | 763 | 524 | [ảnh](shots/gradebook__ta__1440.png) |  |
| `/grading` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1032 | 1564 | [ảnh](shots/grading__ta__1440.png) | chữ bị cắt «…»: "" |
| `/questions` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1546 | 1292 | [ảnh](shots/questions__ta__1440.png) |  |
| `/exams` | 1440 | 0 | 0 | 0/0/0/0 | 1563 | 1280 | [ảnh](shots/exams__ta__1440.png) |  |
| `/threads` | 1440 | 0 | 0 | 0/0/0/0 | 767 | 520 | [ảnh](shots/threads__ta__1440.png) |  |
| `/documents` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1798 | 1268 | [ảnh](shots/documents__ta__1440.png) |  |
| `/calendar` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1820 | 1272 | [ảnh](shots/calendar__ta__1440.png) |  |
| `/insights` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1040 | 1056 | [ảnh](shots/insights__ta__1440.png) |  |
| `/analytics` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1044 | 1056 | [ảnh](shots/analytics__ta__1440.png) |  |
| `/class/members` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1552 | 1292 | [ảnh](shots/class-members__ta__1440.png) |  |
| `/class/settings` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1540 | 1536 | [ảnh](shots/class-settings__ta__1440.png) | trợ giảng thấy toàn bộ cài đặt nhưng mọi ô, công tắc, thanh trượt đều xám khoá; lý do chỉ là dòng chữ rất nhỏ «Chỉ giảng viên được thay đổi.» ở cuối khung đầu tiên; mã lớp vẫn hiện |
| `/gradebook/scheme` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1573 | 1560 | [ảnh](shots/gradebook-scheme__ta__1440.png) |  |
| `/settings` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1306 | 1088 | [ảnh](shots/settings__ta__1440.png) |  |
| `/students/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1833 | 1840 | [ảnh](shots/students-ID__ta__1440.png) |  |
| `/exams/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1554 | 1556 | [ảnh](shots/exams-ID__ta__1440.png) |  |
| `/exams/‹id›/results` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1364 | 1064 | [ảnh](shots/exams-ID-results__ta__1440.png) |  |
| `/exams/‹id›/similarity` | 1440 | 0 | 0 | 0/0/0/0 | 1103 | 1076 | [ảnh](shots/exams-ID-similarity__ta__1440.png) |  |
| `/grading/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as ¦ warning: The resource https://v2202609419695521098.happysrv.de/_next/static/chunks/2k36rc48tuz-5.css was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appr | 0 | 0/0/0/0 | 1575 | 1568 | [ảnh](shots/grading-ID__ta__1440.png) |  |
| `/threads/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1865 | 1876 | [ảnh](shots/threads-ID__ta__1440.png) |  |
| `/assignments/‹id›` | 1440 | 0 | 0 | 0/0/0/0 | 1089 | 1096 | [ảnh](shots/assignments-ID__ta__1440.png) |  |
| `/library` | 1440 | 0 | 0 | 0/0/0/0 | 1063 | 1060 | [ảnh](shots/library__ta__1440.png) |  |
| `/library/‹id›` | 1440 | 0 | 0 | 0/0/0/0 | 1044 | 1052 | [ảnh](shots/library-ID__ta__1440.png) |  |
| `/inbox?tab=resolved` | 1440 | 0 | 0 | 0/0/0/0 | 769 | 2084 | [ảnh](shots/inbox-tab-resolved__ta__1440.png) |  |
| `/class/members?tab=pending` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1657 | 1404 | [ảnh](shots/class-members-tab-pending__ta__1440.png) |  |
| `/threads?view=moderation` | 1440 | 0 | 0 | 0/0/0/0 | 873 | 660 | [ảnh](shots/threads-view-moderation__ta__1440.png) |  |
| `/questions?review_status=PENDING` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1555 | 1296 | [ảnh](shots/questions-review-status-PENDING__ta__1440.png) |  |
| `/khong-ton-tai` | 1440 | 0 | 0 | 0/0/0/0 | 1201 | 1196 | [ảnh](shots/khong-ton-tai__ta__1440.png) |  |

**Đang tải** (API chậm 8 s)

| Route | Ảnh | Ghi nhận |
| --- | --- | --- |
| `/` | [ảnh](shots/home@loading__ta__1440.png) |  |
| `/inbox` | [ảnh](shots/inbox@loading__ta__1440.png) |  |
| `/students` | [ảnh](shots/students@loading__ta__1440.png) |  |
| `/attendance` | [ảnh](shots/attendance@loading__ta__1440.png) |  |
| `/gradebook` | [ảnh](shots/gradebook@loading__ta__1440.png) |  |
| `/grading` | [ảnh](shots/grading@loading__ta__1440.png) |  |
| `/questions` | [ảnh](shots/questions@loading__ta__1440.png) |  |
| `/exams` | [ảnh](shots/exams@loading__ta__1440.png) |  |
| `/threads` | [ảnh](shots/threads@loading__ta__1440.png) |  |
| `/documents` | [ảnh](shots/documents@loading__ta__1440.png) |  |
| `/calendar` | [ảnh](shots/calendar@loading__ta__1440.png) |  |
| `/insights` | [ảnh](shots/insights@loading__ta__1440.png) |  |
| `/class/settings` | [ảnh](shots/class-settings@loading__ta__1440.png) |  |
| `/gradebook/scheme` | [ảnh](shots/gradebook-scheme@loading__ta__1440.png) |  |
| `/settings` | [ảnh](shots/settings@loading__ta__1440.png) |  |
| `/students/‹id›` | [ảnh](shots/students-ID@loading__ta__1440.png) |  |
| `/exams/‹id›` | [ảnh](shots/exams-ID@loading__ta__1440.png) |  |
| `/exams/‹id›/results` | [ảnh](shots/exams-ID-results@loading__ta__1440.png) |  |
| `/exams/‹id›/similarity` | [ảnh](shots/exams-ID-similarity@loading__ta__1440.png) |  |
| `/grading/‹id›` | [ảnh](shots/grading-ID@loading__ta__1440.png) |  |
| `/threads/‹id›` | [ảnh](shots/threads-ID@loading__ta__1440.png) |  |
| `/assignments/‹id›` | [ảnh](shots/assignments-ID@loading__ta__1440.png) |  |
| `/library` | [ảnh](shots/library@loading__ta__1440.png) |  |
| `/library/‹id›` | [ảnh](shots/library-ID@loading__ta__1440.png) |  |
| `/inbox?tab=resolved` | [ảnh](shots/inbox-tab-resolved@loading__ta__1440.png) |  |
| `/class/members?tab=pending` | [ảnh](shots/class-members-tab-pending@loading__ta__1440.png) |  |
| `/threads?view=moderation` | [ảnh](shots/threads-view-moderation@loading__ta__1440.png) |  |
| `/questions?review_status=PENDING` | [ảnh](shots/questions-review-status-PENDING@loading__ta__1440.png) |  |
| `/analytics` | [ảnh](shots/analytics@loading__ta__1440.png) |  |
| `/class/members` | [ảnh](shots/class-members@loading__ta__1440.png) |  |
| `/khong-ton-tai` | [ảnh](shots/khong-ton-tai@loading__ta__1440.png) |  |

**Lỗi** (mất mạng)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/home@error__ta__1440.png) |  |
| `/inbox` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/inbox@error__ta__1440.png) | không có h1 |
| `/students` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/students@error__ta__1440.png) |  |
| `/attendance` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/attendance@error__ta__1440.png) | không có h1 · nội dung không hiện sau 25 s |
| `/gradebook` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/gradebook@error__ta__1440.png) | không có h1 |
| `/grading` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/grading@error__ta__1440.png) |  |
| `/questions` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/questions@error__ta__1440.png) | nội dung không hiện sau 25 s |
| `/exams` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams@error__ta__1440.png) | nội dung không hiện sau 25 s |
| `/threads` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads@error__ta__1440.png) |  |
| `/documents` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/documents@error__ta__1440.png) |  |
| `/calendar` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/calendar@error__ta__1440.png) |  |
| `/insights` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/insights@error__ta__1440.png) |  |
| `/analytics` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/analytics@error__ta__1440.png) |  |
| `/class/members` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/class-members@error__ta__1440.png) | nội dung không hiện sau 25 s |
| `/class/settings` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/class-settings@error__ta__1440.png) | nội dung không hiện sau 25 s |
| `/gradebook/scheme` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/gradebook-scheme@error__ta__1440.png) | không có h1 · nội dung không hiện sau 25 s |
| `/settings` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/settings@error__ta__1440.png) |  |
| `/students/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/students-ID@error__ta__1440.png) |  |
| `/exams/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID@error__ta__1440.png) | nội dung không hiện sau 25 s |
| `/exams/‹id›/results` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID-results@error__ta__1440.png) |  |
| `/exams/‹id›/similarity` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID-similarity@error__ta__1440.png) |  |
| `/grading/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/grading-ID@error__ta__1440.png) |  |
| `/threads/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads-ID@error__ta__1440.png) |  |
| `/assignments/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/assignments-ID@error__ta__1440.png) |  |
| `/library` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/library@error__ta__1440.png) |  |
| `/library/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/library-ID@error__ta__1440.png) |  |
| `/inbox?tab=resolved` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/inbox-tab-resolved@error__ta__1440.png) | không có h1 |
| `/class/members?tab=pending` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/class-members-tab-pending@error__ta__1440.png) | nội dung không hiện sau 25 s |
| `/threads?view=moderation` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads-view-moderation@error__ta__1440.png) |  |
| `/questions?review_status=PENDING` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/questions-review-status-PENDING@error__ta__1440.png) | nội dung không hiện sau 25 s |
| `/khong-ton-tai` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/khong-ton-tai@error__ta__1440.png) |  |

### Quản trị `admin@`

**Đầy dữ liệu** (lớp 761987)

| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 1440 | 0 | 0 | 0/0/0/0 | 530 | 272 | [ảnh](shots/home__admin__1440.png) |  |
| `/observability` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1306 | 1056 | [ảnh](shots/observability__admin__1440.png) |  |
| `/admin/courses` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1250 | 1020 | [ảnh](shots/admin-courses__admin__1440.png) | có câu mô tả dưới tiêu đề «Mở lớp, phân công giảng viên và lưu trữ lớp đã kết thúc.» + dòng nhỏ «2 lớp đang chạy»; chỉ 2 hàng, nửa dưới trang trống |
| `/admin/users` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1269 | 1020 | [ảnh](shots/admin-users__admin__1440.png) | có câu mô tả hai dòng dưới tiêu đề «Mời giảng viên và trợ giảng, khoá hoặc mở khoá tài khoản. Sinh viên tự đăng ký rồi vào lớp bằng mã tham gia.» · danh sách 30+ người dùng liền nhau không phân trang; cột «Lần cuối» ghi «00:06 · 02-11», «18:26», «Chưa đăng nhập» ba dạng khác nhau; thanh trên ghi «Toàn hệ thống · INT1006» thay cho chọn lớp |
| `/settings/llm` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1857 | 492 | [ảnh](shots/settings-llm__admin__1440.png) |  |
| `/settings/integrations` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1522 | 1236 | [ảnh](shots/settings-integrations__admin__1440.png) |  |
| `/settings` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1269 | 1036 | [ảnh](shots/settings__admin__1440.png) |  |
| `/class/members` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1001 | 1004 | [ảnh](shots/class-members__admin__1440.png) |  |
| `/gradebook` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 809 | 1084 | [ảnh](shots/gradebook__admin__1440.png) |  |

**Đang tải** (API chậm 8 s)

| Route | Ảnh | Ghi nhận |
| --- | --- | --- |
| `/` | [ảnh](shots/home@loading__admin__1440.png) |  |
| `/observability` | [ảnh](shots/observability@loading__admin__1440.png) |  |
| `/admin/courses` | [ảnh](shots/admin-courses@loading__admin__1440.png) |  |
| `/admin/users` | [ảnh](shots/admin-users@loading__admin__1440.png) |  |
| `/settings/llm` | [ảnh](shots/settings-llm@loading__admin__1440.png) |  |
| `/settings/integrations` | [ảnh](shots/settings-integrations@loading__admin__1440.png) |  |
| `/settings` | [ảnh](shots/settings@loading__admin__1440.png) |  |
| `/class/members` | [ảnh](shots/class-members@loading__admin__1440.png) |  |
| `/gradebook` | [ảnh](shots/gradebook@loading__admin__1440.png) |  |

**Lỗi** (mất mạng)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/home@error__admin__1440.png) |  |
| `/observability` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/observability@error__admin__1440.png) |  |
| `/admin/courses` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/admin-courses@error__admin__1440.png) |  |
| `/admin/users` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/admin-users@error__admin__1440.png) |  |
| `/settings/llm` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/settings-llm@error__admin__1440.png) |  |
| `/settings/integrations` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/settings-integrations@error__admin__1440.png) |  |
| `/settings` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/settings@error__admin__1440.png) |  |
| `/class/members` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/class-members@error__admin__1440.png) |  |
| `/gradebook` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/gradebook@error__admin__1440.png) |  |

### Sinh viên `sv.gioi@` (lớp 761987 + 761988)

**Đầy dữ liệu** (lớp 761987)

| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 1440 | 0 | 0 | 0/0/0/0 | 317 | 272 | [ảnh](shots/home__student__1440.png) |  |
| `/` | 375 | 0 | 0 | 0/0/0/0 | 312 | 296 | [ảnh](shots/home__student__375.png) | tiêu đề «Hôm nay» của khối thứ hai trùng tên mục «Hôm nay» ở thanh dưới; thanh điều hướng dưới có 5 mục (4 + «Thêm») |
| `/threads` | 1440 | 0 | 0 | 0/0/0/0 | 767 | 548 | [ảnh](shots/threads__student__1440.png) | từ kỹ thuật hiện cho sinh viên: AI |
| `/threads` | 375 | 0 | 0 | 0/0/0/0 | 308 | 1588 | [ảnh](shots/threads__student__375.png) | từ kỹ thuật hiện cho sinh viên: AI · «Bộ lọc» là một dòng chữ không có dấu hiệu bấm được; phần lớn tiêu đề thread là đoạn chữ tiếng Anh cắt từ tài liệu, dài 4–5 dòng ở 375 px |
| `/practice` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1353 | 1324 | [ảnh](shots/practice__student__1440.png) |  |
| `/practice` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 902 | 876 | [ảnh](shots/practice__student__375.png) | có câu mô tả dưới tiêu đề «Luyện theo chủ đề bạn hay sai, hoặc thi thử với đề cũ.» và chân trang «Kết quả luyện đề không tính vào điểm quá trình, trừ các bài ghi “tính điểm”.» · mỗi chủ đề lặp «Bạn chưa luyện chủ đề này» + «N câu · phản hồi ngay sau mỗi câu» (4 lần giống hệt chữ) |
| `/exams` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1530 | 1244 | [ảnh](shots/exams__student__1440.png) |  |
| `/exams` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1512 | 1268 | [ảnh](shots/exams__student__375.png) | có câu mô tả dưới tiêu đề «Bài kiểm tra tính điểm của lớp. Đồng hồ chạy khi bạn bấm Bắt đầu làm bài.»; mỗi bài thi lặp «Đã có điểm» hai lần (nhóm «Đã có điểm» và dòng nhỏ dưới tên) |
| `/library` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1841 | 1272 | [ảnh](shots/library__student__1440.png) | ô «Tìm tài liệu» tự focus (viền đỏ dày) khi mở trang; mép trên Panel danh sách chạm vào đáy ô tìm kiếm · dòng «Đề tham khảo tuần 5» không có nút «…» nên nút «Xem» lệch phải so với bốn dòng còn lại |
| `/library` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1318 | 1072 | [ảnh](shots/library__student__375.png) | ô «Tìm tài liệu» có viền đỏ (đang focus) ngay khi mở trang, chưa chạm gì · mỗi tài liệu có nhãn «PDF» + nút «Xem» + nút «…» trên cùng một hàng; ba lớp bộ lọc (tìm, Loại, Tuần) chiếm hơn nửa màn đầu |
| `/calendar` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1766 | 1248 | [ảnh](shots/calendar__student__1440.png) | tuần hiện tại chỉ có dữ liệu ở T2 (3 mục) còn 6 cột còn lại «—»; cột Chủ nhật tô nền hồng nhạt; phần dưới trang trống |
| `/calendar` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1509 | 1520 | [ảnh](shots/calendar__student__375.png) | mặc định ở 375 là kiểu «Danh sách»; mỗi mục lặp nhãn loại («Buổi học», «Bài thi») bằng chữ nhỏ xám dưới giờ |
| `/me` | 1440 | 0 | 0 | 0/0/0/0 | 748 | 2008 | [ảnh](shots/me__student__1440.png) | cột nội dung rộng ~660 px bên trái, 60% bề ngang 1440 để trống; «Phát biểu +0,25» lặp 12 dòng với cùng ngày «02/11» |
| `/me` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 290 | 1552 | [ảnh](shots/me__student__375.png) | 3 điều khiển < 24 px · công thức tính điểm hiện nguyên dạng biểu thức: «min(3; 1) = 1», «(20 × 9 + 20 × 10) ÷ 40», «min(10; max(0; 9,5 + 1 − 0))», kèm «Căn cứ · tr. 1» · ô «Nếu cuối kỳ được [8,0]» là ô nhập giữa đoạn văn, không có nút; trang dài ~2000 px |
| `/practice/history` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1310 | 1296 | [ảnh](shots/practice-history__student__1440.png) |  |
| `/practice/history` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 882 | 864 | [ảnh](shots/practice-history__student__375.png) |  |
| `/threads/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1879 | 1868 | [ảnh](shots/threads-ID__student__1440.png) | từ kỹ thuật hiện cho sinh viên: AI |
| `/threads/‹id›` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1876 | 1856 | [ảnh](shots/threads-ID__student__375.png) | từ kỹ thuật hiện cho sinh viên: AI · sinh viên thấy chuỗi «<ngữ_cảnh>» trong câu trả lời AI; nội dung bài đăng có «- - Unstructured threats»; dấu «···» đứng một dòng riêng dưới tên tác giả |
| `/library/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1815 | 1776 | [ảnh](shots/library-ID__student__1440.png) | từ kỹ thuật hiện cho sinh viên: AI |
| `/library/‹id›` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1897 | 1884 | [ảnh](shots/library-ID__student__375.png) | từ kỹ thuật hiện cho sinh viên: AI |
| `/assignments/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1276 | 1276 | [ảnh](shots/assignments-ID__student__1440.png) | từ kỹ thuật hiện cho sinh viên: AI · chữ bị cắt «…»: "" |
| `/assignments/‹id›` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1327 | 1296 | [ảnh](shots/assignments-ID__student__375.png) | từ kỹ thuật hiện cho sinh viên: AI · chữ bị cắt «…»: "" · 1 điều khiển < 24 px · đề bài hiện ký hiệu Markdown thô: «# Bài tập 03 — Phân tích…», «## Đề bài», «## Yêu cầu nộp» (in đậm và danh sách thì dựng đúng); tiêu đề trang là «Bài tập 04» còn đề bài ghi «Bài tập 03» · trang dài hơn ~3 màn hình ở 375 px trước khi tới «Tiêu chí chấm» và ô nộp bài |
| `/practice/at-symmetric` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1318 | 1288 | [ảnh](shots/practice-at-symmetric__student__1440.png) |  |
| `/practice/at-symmetric` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1333 | 1336 | [ảnh](shots/practice-at-symmetric__student__375.png) | «Kiểm tra» hồng nhạt (khoá) kèm dòng «Chọn một đáp án để kiểm tra»; nửa dưới màn hình trống |
| `/join` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1022 | 1032 | [ảnh](shots/join__student__1440.png) |  |
| `/join` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1021 | 1056 | [ảnh](shots/join__student__375.png) |  |
| `/join/AN7K2MQ` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1061 | 1072 | [ảnh](shots/join-AN7K2MQ__student__1440.png) | chuyển hướng → /join |
| `/join/AN7K2MQ` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1036 | 1048 | [ảnh](shots/join-AN7K2MQ__student__375.png) | chuyển hướng → /join · hai câu hướng dẫn mã cùng ý («Mã gồm 7 ký tự, không phân biệt chữ hoa chữ thường.» và «7 ký tự, không có O, 0, 1, I, L.»); đã là thành viên nên chỉ hiện «Bạn đã ở trong lớp này.» và nút «Về Hôm nay» |
| `/exams/‹id›` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1049 | 1056 | [ảnh](shots/exams-ID__student__1440.png) |  |
| `/exams/‹id›` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v ¦ warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1023 | 1032 | [ảnh](shots/exams-ID__student__375.png) |  |
| `/exams/‹id›/results` | 1440 | 0 | 0 | 0/0/0/0 | 718 | 732 | [ảnh](shots/exams-ID-results__student__1440.png) | sinh viên mở route của giảng viên nhận «Bạn không có quyền xem màn này»; mục «Bài thi» ở sidebar vẫn sáng như đang ở đúng màn |
| `/exams/‹id›/results` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v | 0 | 0/0/0/0 | 1245 | 1256 | [ảnh](shots/exams-ID-results__student__375.png) |  |
| `/gradebook` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 784 | 1052 | [ảnh](shots/gradebook__student__1440.png) |  |
| `/gradebook` | 375 | 0 | 0 | 0/0/0/0 | 297 | 564 | [ảnh](shots/gradebook__student__375.png) | màn chặn quyền: «Bạn không có quyền xem màn này» + «Trang này dành cho giảng viên và trợ giảng.»; nút «Về Hôm nay» là nút trắng, trong khi màn «Không tìm thấy trang» dùng nút đỏ cho cùng việc |
| `/inbox` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 829 | 1096 | [ảnh](shots/inbox__student__1440.png) |  |
| `/inbox` | 375 | 0 | 0 | 0/0/0/0 | 301 | 572 | [ảnh](shots/inbox__student__375.png) | cùng màn chặn quyền, nửa dưới màn hình trống |
| `/admin/users` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1022 | 1024 | [ảnh](shots/admin-users__student__1440.png) |  |
| `/admin/users` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v | 0 | 0/0/0/0 | 570 | 572 | [ảnh](shots/admin-users__student__375.png) | cùng màn chặn quyền («Trang này dành cho quản trị viên.») |
| `/khong-ton-tai` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1023 | 1036 | [ảnh](shots/khong-ton-tai__student__1440.png) |  |
| `/khong-ton-tai` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v | 0 | 0/0/0/0 | 1033 | 1040 | [ảnh](shots/khong-ton-tai__student__375.png) |  |
| `/chat` | 1440 | 0 | 0 | 0/0/0/0 | 761 | 524 | [ảnh](shots/chat__student__1440.png) | khung chat gần như trống: dòng «Hỏi bất cứ điều gì về lớp này.», một chip gợi ý «Quy chế thi cuối kỳ nói gì về tài liệu?», ô nhập sát đáy; nút «Phiên mới» hồng nhạt (khoá) ở góc trên phải · danh sách «Phiên trước» có 4 phiên, mỗi phiên tên bằng cả câu hỏi (3–4 dòng) cùng ngày giờ «23:10 30-10»; một phiên là câu hỏi ngoài chủ đề («Địa điểm du lịch đẹp ở miền Trung?») |
| `/chat` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 295 | 304 | [ảnh](shots/chat__student__375.png) | ẢNH LÀ TRANG ĐĂNG NHẬP — phiên bị đá ra giữa lúc chụp; chụp lại ở lượt sau (xem ghi chú mục cuối) · «Phiên trước (4)» là một ô thu gọn; khung chat chiếm cả màn, ô nhập ở đáy khung; nút «Gửi» hồng nhạt (khoá) khi chưa nhập |

**Đang tải** (API chậm 8 s)

| Route | Ảnh | Ghi nhận |
| --- | --- | --- |
| `/` | [ảnh](shots/home@loading__student__1440.png) | khung chờ của «Hôm nay» cho tài khoản sinh viên hiện tiêu đề kiểu giảng viên «Việc cần xử lý hôm nay» + câu mô tả «Việc của bạn và các buổi học sắp tới hiện ở đây.»; sidebar chưa có mục nào, thanh trên không có chọn lớp / tìm kiếm |
| `/chat` | [ảnh](shots/chat@loading__student__1440.png) |  |
| `/threads` | [ảnh](shots/threads@loading__student__1440.png) |  |
| `/practice` | [ảnh](shots/practice@loading__student__1440.png) |  |
| `/exams` | [ảnh](shots/exams@loading__student__1440.png) |  |
| `/library` | [ảnh](shots/library@loading__student__1440.png) |  |
| `/calendar` | [ảnh](shots/calendar@loading__student__1440.png) |  |
| `/me` | [ảnh](shots/me@loading__student__1440.png) |  |
| `/practice/history` | [ảnh](shots/practice-history@loading__student__1440.png) |  |
| `/threads/‹id›` | [ảnh](shots/threads-ID@loading__student__1440.png) |  |
| `/library/‹id›` | [ảnh](shots/library-ID@loading__student__1440.png) |  |
| `/assignments/‹id›` | [ảnh](shots/assignments-ID@loading__student__1440.png) |  |
| `/practice/at-symmetric` | [ảnh](shots/practice-at-symmetric@loading__student__1440.png) |  |
| `/join` | [ảnh](shots/join@loading__student__1440.png) |  |
| `/join/AN7K2MQ` | [ảnh](shots/join-AN7K2MQ@loading__student__1440.png) |  |
| `/exams/‹id›` | [ảnh](shots/exams-ID@loading__student__1440.png) |  |
| `/exams/‹id›/results` | [ảnh](shots/exams-ID-results@loading__student__1440.png) |  |
| `/admin/users` | [ảnh](shots/admin-users@loading__student__1440.png) |  |
| `/khong-ton-tai` | [ảnh](shots/khong-ton-tai@loading__student__1440.png) |  |
| `/gradebook` | [ảnh](shots/gradebook@loading__student__1440.png) |  |
| `/inbox` | [ảnh](shots/inbox@loading__student__1440.png) |  |

**Lỗi** (mất mạng)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/home@error__student__1440.png) | dải báo mất mạng ở đầu trang bị sidebar che phần đầu câu (đọc được «…n đã nhập vẫn được giữ; hệ thống sẽ gửi lại khi có mạng.») · sau ~2 s vẫn chỉ có 4 vạch khung xương, chưa có thông báo lỗi hay nút thử lại trong vùng nội dung; chuông thông báo mất số; chân sidebar hiện «Bản mô phỏng · dữ liệu giả» (chữ này không có ở trạng thái bình thường) |
| `/chat` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/chat@error__student__1440.png) | từ kỹ thuật hiện cho sinh viên: AI |
| `/threads` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads@error__student__1440.png) |  |
| `/practice` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/practice@error__student__1440.png) |  |
| `/exams` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams@error__student__1440.png) | nội dung không hiện sau 25 s |
| `/library` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/library@error__student__1440.png) |  |
| `/calendar` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/calendar@error__student__1440.png) |  |
| `/me` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/me@error__student__1440.png) |  |
| `/practice/history` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/practice-history@error__student__1440.png) |  |
| `/threads/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads-ID@error__student__1440.png) |  |
| `/library/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/library-ID@error__student__1440.png) |  |
| `/assignments/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/assignments-ID@error__student__1440.png) |  |
| `/practice/at-symmetric` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/practice-at-symmetric@error__student__1440.png) |  |
| `/join` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/join@error__student__1440.png) |  |
| `/join/AN7K2MQ` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/join-AN7K2MQ@error__student__1440.png) |  |
| `/exams/‹id›` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID@error__student__1440.png) |  |
| `/exams/‹id›/results` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams-ID-results@error__student__1440.png) |  |
| `/gradebook` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/gradebook@error__student__1440.png) |  |
| `/inbox` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/inbox@error__student__1440.png) |  |
| `/admin/users` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/admin-users@error__student__1440.png) |  |
| `/khong-ton-tai` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/khong-ton-tai@error__student__1440.png) |  |

**Rỗng / ít dữ liệu** (lớp 761988)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | 0 | 0/0/0/0 | [ảnh](shots/home@empty__student__1440.png) |  |
| `/chat` | 0 | 0/0/0/0 | [ảnh](shots/chat@empty__student__1440.png) |  |
| `/threads` | 0 | 0/0/0/0 | [ảnh](shots/threads@empty__student__1440.png) |  |
| `/practice` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/practice@empty__student__1440.png) |  |
| `/exams` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/exams@empty__student__1440.png) |  |
| `/library` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/library@empty__student__1440.png) |  |
| `/calendar` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/calendar@empty__student__1440.png) |  |
| `/me` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/me@empty__student__1440.png) |  |
| `/practice/history` | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0/0/0/0 | [ảnh](shots/practice-history@empty__student__1440.png) |  |

### Sinh viên `sv.kha@` (lớp 761987)

**Đầy dữ liệu** (lớp 761987)

| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 1440 | 0 | 0 | 0/0/0/0 | 336 | 292 | [ảnh](shots/home__student2__1440.png) |  |
| `/chat` | 1440 | 0 | 0 | 0/0/0/0 | 793 | 536 | [ảnh](shots/chat__student2__1440.png) |  |
| `/me` | 1440 | 0 | 0 | 0/0/0/0 | 803 | 2668 | [ảnh](shots/me__student2__1440.png) |  |
| `/exams` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1674 | 1420 | [ảnh](shots/exams__student2__1440.png) |  |
| `/threads` | 1440 | 0 | 0 | 0/0/0/0 | 794 | 548 | [ảnh](shots/threads__student2__1440.png) | từ kỹ thuật hiện cho sinh viên: AI |
| `/practice` | 1440 | 0 | 0 | 0/0/0/0 | 1327 | 1336 | [ảnh](shots/practice__student2__1440.png) |  |

**Đang tải** (API chậm 8 s)

| Route | Ảnh | Ghi nhận |
| --- | --- | --- |
| `/` | [ảnh](shots/home@loading__student2__1440.png) |  |
| `/chat` | [ảnh](shots/chat@loading__student2__1440.png) |  |
| `/me` | [ảnh](shots/me@loading__student2__1440.png) |  |
| `/exams` | [ảnh](shots/exams@loading__student2__1440.png) |  |
| `/threads` | [ảnh](shots/threads@loading__student2__1440.png) |  |
| `/practice` | [ảnh](shots/practice@loading__student2__1440.png) |  |

**Lỗi** (mất mạng)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/home@error__student2__1440.png) |  |
| `/chat` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/chat@error__student2__1440.png) | từ kỹ thuật hiện cho sinh viên: AI |
| `/me` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/me@error__student2__1440.png) |  |
| `/exams` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams@error__student2__1440.png) | nội dung không hiện sau 25 s |
| `/threads` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/threads@error__student2__1440.png) |  |
| `/practice` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/practice@error__student2__1440.png) |  |

### Sinh viên mới `sv.moi@` (chưa vào lớp)

**Đầy dữ liệu** (lớp 761987)

| Route | Cỡ | Console | HTTP ≥400 | axe C/S/M/m | Nội dung ms | LCP ms | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 1440 | 0 | 0 | 0/0/0/0 | 730 | 492 | [ảnh](shots/home__newbie__1440.png) |  |
| `/` | 375 | 0 | 0 | 0/0/0/0 | 312 | 1068 | [ảnh](shots/home__newbie__375.png) | dải hồng «Bản mô phỏng · dữ liệu giả» ở đầu mọi trang của sinh viên chưa có lớp (không có ở `sv.gioi`) · ý «bạn chưa vào lớp nào» lặp 3 lần trong một màn (dòng ngày, tiêu đề khối «Nhập mã tham gia lớp», câu mô tả bên dưới) + «Mã gồm 7 chữ và số.» dưới ô nhập · ô «Mã tham gia lớp» hiện sẵn chữ xám «AN7K2MQ» (là mã thật của lớp 761987) trông như đã điền; «Tiếp tục» hồng nhạt (khoá) |
| `/join` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1063 | 1076 | [ảnh](shots/join__newbie__1440.png) |  |
| `/join` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v | 0 | 0/0/0/0 | 1005 | 1008 | [ảnh](shots/join__newbie__375.png) | trang «Tham gia lớp» có hai câu hướng dẫn mã (trên thẻ «Nhập mã tham gia giảng viên gửi cho bạn. Mã gồm 7 ký tự…» và dưới ô «7 ký tự, không có O, 0, 1, I, L.»); nút «Xem lớp» hồng nhạt khi ô đã có chữ xám placeholder |
| `/chat` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 765 | 540 | [ảnh](shots/chat__newbie__1440.png) |  |
| `/chat` | 375 | 0 | 0 | 0/0/0/0 | 295 | 308 | [ảnh](shots/chat__newbie__375.png) | trạng thái chưa có lớp: tiêu đề «Bạn chưa vào lớp nào» + nút đỏ «Tham gia lớp bằng mã»; nửa dưới màn hình trống; thanh dưới chỉ có «Hôm nay»; chữ chọn lớp «Chưa có lớp» ở thanh trên |
| `/library` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1036 | 1044 | [ảnh](shots/library__newbie__1440.png) |  |
| `/library` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v | 0 | 0/0/0/0 | 566 | 572 | [ảnh](shots/library__newbie__375.png) | cùng khung trống «Bạn chưa vào lớp nào» như Chat riêng |
| `/me` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 766 | 1056 | [ảnh](shots/me__newbie__1440.png) |  |
| `/me` | 375 | 0 | 0 | 0/0/0/0 | 296 | 564 | [ảnh](shots/me__newbie__375.png) |  |
| `/exams` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1011 | 1020 | [ảnh](shots/exams__newbie__1440.png) |  |
| `/exams` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v | 0 | 0/0/0/0 | 555 | 576 | [ảnh](shots/exams__newbie__375.png) |  |
| `/calendar` | 1440 | warning: The resource https://v2202609419695521098.happysrv.de/landing/gioi-thieu-v1.webp was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as | 0 | 0/0/0/0 | 1007 | 1020 | [ảnh](shots/calendar__newbie__1440.png) |  |
| `/calendar` | 375 | warning: The resource https://v2202609419695521098.happysrv.de/brand/logo-edupilot.svg was preloaded using link preload but not used within a few seconds from the window's load event. Please make sure it has an appropriate `as` v | 0 | 0/0/0/0 | 558 | 564 | [ảnh](shots/calendar__newbie__375.png) |  |

**Đang tải** (API chậm 8 s)

| Route | Ảnh | Ghi nhận |
| --- | --- | --- |
| `/` | [ảnh](shots/home@loading__newbie__1440.png) |  |
| `/join` | [ảnh](shots/join@loading__newbie__1440.png) |  |
| `/chat` | [ảnh](shots/chat@loading__newbie__1440.png) |  |
| `/library` | [ảnh](shots/library@loading__newbie__1440.png) |  |
| `/me` | [ảnh](shots/me@loading__newbie__1440.png) |  |
| `/exams` | [ảnh](shots/exams@loading__newbie__1440.png) |  |
| `/calendar` | [ảnh](shots/calendar@loading__newbie__1440.png) |  |

**Lỗi** (mất mạng)

| Route | Console | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- |
| `/` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/home@error__newbie__1440.png) |  |
| `/join` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/join@error__newbie__1440.png) |  |
| `/chat` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/chat@error__newbie__1440.png) |  |
| `/library` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/library@error__newbie__1440.png) |  |
| `/me` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/me@error__newbie__1440.png) |  |
| `/exams` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/exams@error__newbie__1440.png) |  |
| `/calendar` | error: Failed to load resource: net::ERR_INTERNET_DISCONNECTED | 0/0/0/0 | [ảnh](shots/calendar@error__newbie__1440.png) |  |

## 4. Hộp thoại, drawer, menu, focus bàn phím

Request không phải GET (ngoài `/auth/`) trong cả đợt mở hộp thoại / focus: **0**.

| Vai | Route | Cỡ | Mở | Esc đóng | Chuyển động lúc mở | axe C/S/M/m | Ảnh | Ghi nhận |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| teacher | `/` | 1440 | có lớp phủ / menu (bell) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/home@dlg-bell__teacher__1440.png) | popover thông báo / menu tài khoản / chọn lớp vào bằng «pop» 200 ms; đóng được bằng Esc |
| teacher | `/` | 1440 | có lớp phủ / menu (usermenu) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/home@dlg-usermenu__teacher__1440.png) |  |
| teacher | `/` | 1440 | có lớp phủ / menu (courseswitch) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/home@dlg-courseswitch__teacher__1440.png) |  |
| teacher | `/` | 1440 | có lớp phủ / menu (cmdk) | true | không có | 0/16/0/0 | [ảnh](shots/home@dlg-cmdk__teacher__1440.png) | hộp tìm nhanh (⌘K): axe «nested-interactive» mức cao ×16 (mỗi kết quả `#_r_3_-opt-N` chứa điều khiển lồng nhau) · hộp tìm nhanh gồm danh sách mục điều hướng cao ~330 px, mục đầu có phím ↵; không có nhóm hay nhãn phân loại |
| teacher | `/inbox` | 1440 | không có lớp phủ (ticket) — mở nội tuyến hoặc đổi trang, xem ảnh | true | background-color:140ms | 0/0/0/0 | [ảnh](shots/inbox@dlg-ticket__teacher__1440.png) |  |
| teacher | `/inbox` | 375 | không có lớp phủ (ticket) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/inbox@dlg-ticket__teacher__375.png) | mở một yêu cầu thì thay danh sách bằng trang chi tiết (có «← Hộp thư»); hai nút «Nhận» / «Đóng không trả lời» nằm sát thanh điều hướng dưới, phải cuộn mới thấy đủ |
| teacher | `/gradebook` | 1440 | có lớp phủ / menu (chotdiem) | true | Dialog-module__rUw9Ha__fadeIn:250ms | 0/0/0/0 | [ảnh](shots/gradebook@dlg-chotdiem__teacher__1440.png) | hộp thoại xác nhận «Chốt điểm» mở bằng hiệu ứng fadeIn 250 ms (dài hơn 200 ms); chỉ mở rồi đóng bằng Esc, không bấm xác nhận · hộp xác nhận «Chốt điểm» nói rõ phân bố điểm chữ và «1 sinh viên liệt»; nút huỷ là «Để sau» (các biểu mẫu nội tuyến khác dùng «Huỷ»); nền phủ mờ cả sidebar |
| teacher | `/gradebook` | 1440 | có lớp phủ / menu (themhanhdong) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/gradebook@dlg-themhanhdong__teacher__1440.png) |  |
| teacher | `/gradebook` | 1440 | có lớp phủ / menu (hanghd) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/gradebook@dlg-hanghd__teacher__1440.png) |  |
| teacher | `/grading` | 1440 | có lớp phủ / menu (chamtatca-khoa) | true | không có | 0/0/0/0 | [ảnh](shots/grading@dlg-chamtatca-khoa__teacher__1440.png) |  |
| teacher | `/questions` | 1440 | có lớp phủ / menu (taocauhoi) | true | background-color:140ms, Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/questions@dlg-taocauhoi__teacher__1440.png) |  |
| teacher | `/threads` | 1440 | không có lớp phủ (datcauhoi) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/threads@dlg-datcauhoi__teacher__1440.png) | «Đặt câu hỏi» mở khung soạn nội tuyến đẩy danh sách thread xuống ~330 px (không phải hộp thoại); ô «Đăng» hồng nhạt (khoá) đến khi nhập |
| teacher | `/documents` | 1440 | không có lớp phủ (taitailieu) — mở nội tuyến hoặc đổi trang, xem ảnh | true | background-color:140ms | 0/0/0/0 | [ảnh](shots/documents@dlg-taitailieu__teacher__1440.png) |  |
| teacher | `/documents` | 1440 | có lớp phủ / menu (thaotac) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/documents@dlg-thaotac__teacher__1440.png) |  |
| teacher | `/calendar` | 1440 | không có lớp phủ (themsukien) — mở nội tuyến hoặc đổi trang, xem ảnh | true | border-bottom-color:140ms, border-left-color:140ms, border-right-color:140ms, border-top-color:140ms, border-bottom-color:140ms, border-left-color:140ms, border-right-color:140ms, border-top-color:140ms | 0/0/0/0 | [ảnh](shots/calendar@dlg-themsukien__teacher__1440.png) | «Thêm sự kiện» mở biểu mẫu nội tuyến 5 ô phía trên tuần; ô đầu tự focus (viền đỏ dày); «Lưu» hồng nhạt (khoá); chuyển viền 140 ms ở 4 cạnh |
| teacher | `/calendar` | 1440 | có lớp phủ / menu (taolichbuoihoc) | true | Dialog-module__rUw9Ha__slideIn:250ms, Feedback-module__VRpUaW__shimmer:1400ms, Feedback-module__VRpUaW__shimmer:1400ms, Feedback-module__VRpUaW__shimmer:1400ms, Feedback-module__VRpUaW__shimmer:1400ms | 0/0/0/0 | [ảnh](shots/calendar@dlg-taolichbuoihoc__teacher__1440.png) | ngăn kéo / hộp «Tạo lịch buổi học» vào bằng slideIn 250 ms và có 4 khung xương shimmer 1400 ms chạy lặp trong lúc mở |
| teacher | `/class/members` | 1440 | không có lớp phủ (nhapds) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/class-members@dlg-nhapds__teacher__1440.png) |  |
| teacher | `/class/members` | 1440 | có lớp phủ / menu (thaotac) | true | Menu-module__W6EquG__pop:200ms | 0/1/0/0 | [ảnh](shots/class-members@dlg-thaotac__teacher__1440.png) | axe «target-size» mức cao ×1: nút «…» (thao tác cho từng thành viên) nhỏ hơn 24×24 px |
| teacher | `/class/settings` | 1440 | có lớp phủ / menu (taolaima) | true | Dialog-module__rUw9Ha__fadeIn:250ms | 0/0/0/0 | [ảnh](shots/class-settings@dlg-taolaima__teacher__1440.png) |  |
| teacher | `/insights` | 1440 | có lớp phủ / menu (cauhoiandanh) | false | không có | 0/0/0/0 | [ảnh](shots/insights@dlg-cauhoiandanh__teacher__1440.png) |  |
| teacher | `/settings/llm` | 1440 | không có lớp phủ (nangcao) — mở nội tuyến hoặc đổi trang, xem ảnh | true | background-color:140ms, color:140ms | 0/0/0/0 | [ảnh](shots/settings-llm@dlg-nangcao__teacher__1440.png) |  |
| teacher | `/settings` | 1440 | có lớp phủ / menu (thietbi) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/settings@dlg-thietbi__teacher__1440.png) |  |
| teacher | `/grading/‹id›` | 1440 | không có lớp phủ (trichdoan) — mở nội tuyến hoặc đổi trang, xem ảnh | true | background-color:140ms, border-left-color:140ms | 0/0/0/0 | [ảnh](shots/grading-ID@dlg-trichdoan__teacher__1440.png) |  |
| admin | `/admin/users` | 1440 | không có lớp phủ (moigv) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/admin-users@dlg-moigv__admin__1440.png) | «Mời giảng viên» mở biểu mẫu nội tuyến ngay dưới tiêu đề kèm thêm một câu giải thích («Người được mời nhận mật khẩu… dùng trong 72 giờ…») và đẩy bảng người dùng xuống |
| admin | `/admin/users` | 1440 | có lớp phủ / menu (thaotac) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/admin-users@dlg-thaotac__admin__1440.png) |  |
| admin | `/admin/courses` | 1440 | không có lớp phủ (molop) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/admin-courses@dlg-molop__admin__1440.png) | «Mở lớp» mở biểu mẫu nội tuyến 6 ô + «Trợ giảng» + 2 nút; mỗi ô có dòng gợi ý nhỏ dưới (ví dụ «Không bắt buộc. 1–1.000»); nút «Mở lớp» hồng nhạt (khoá) |
| admin | `/admin/courses` | 1440 | có lớp phủ / menu (hanhdong) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/admin-courses@dlg-hanhdong__admin__1440.png) |  |
| admin | `/settings/llm` | 1440 | không có lớp phủ (themncc) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/settings-llm@dlg-themncc__admin__1440.png) |  |
| admin | `/settings/llm` | 1440 | không có lớp phủ (nangcao) — mở nội tuyến hoặc đổi trang, xem ảnh | true | background-color:140ms, color:140ms | 0/0/0/0 | [ảnh](shots/settings-llm@dlg-nangcao__admin__1440.png) | «Cài đặt nâng cao» mở nội tuyến ngay trong hàng (ô nhiệt độ, số token), không đổi vị trí nút; mỗi ô có dòng đơn vị / mặc định nhỏ |
| student | `/` | 1440 | có lớp phủ / menu (bell) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/home@dlg-bell__student__1440.png) |  |
| student | `/` | 1440 | có lớp phủ / menu (usermenu) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/home@dlg-usermenu__student__1440.png) |  |
| student | `/` | 1440 | có lớp phủ / menu (courseswitch) | true | Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/home@dlg-courseswitch__student__1440.png) |  |
| student | `/` | 375 | có lớp phủ / menu (them) | true | không có | 0/0/0/0 | [ảnh](shots/home@dlg-them__student__375.png) | «Thêm» mở tờ đáy (bottom sheet) với 4 mục (Bài thi, Thư viện, Lịch, Kết quả của tôi) và nút đóng ×; nền phủ xám cả thanh trên |
| student | `/threads` | 1440 | không có lớp phủ (datcauhoi) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/threads@dlg-datcauhoi__student__1440.png) |  |
| student | `/threads` | 375 | không có lớp phủ (datcauhoi) — mở nội tuyến hoặc đổi trang, xem ảnh | true | không có | 0/0/0/0 | [ảnh](shots/threads@dlg-datcauhoi__student__375.png) | biểu mẫu «Đặt câu hỏi» gồm Tiêu đề, Nội dung (cả hai «bắt buộc»), Thẻ (placeholder «quy-che, tcp»), Tuần; «Đăng» hồng nhạt (khoá) cạnh «Huỷ»; cao hơn một màn hình ở 375 |
| student | `/library` | 1440 | có lớp phủ / menu (them) | true | border-bottom-color:140ms, border-left-color:140ms, border-right-color:140ms, border-top-color:140ms, Menu-module__W6EquG__pop:200ms | 0/0/0/0 | [ảnh](shots/library@dlg-them__student__1440.png) |  |
| student | `/calendar` | 1440 | không có lớp phủ (themvaolich) — mở nội tuyến hoặc đổi trang, xem ảnh | true | border-bottom-color:140ms, border-left-color:140ms, border-right-color:140ms, border-top-color:140ms | 0/0/0/0 | [ảnh](shots/calendar@dlg-themvaolich__student__1440.png) |  |
| student | `/chat` | 1440 | có lớp phủ / menu (soanthao) | true | border-bottom-color:140ms, border-left-color:140ms, border-right-color:140ms | 0/0/0/0 | [ảnh](shots/chat@dlg-soanthao__student__1440.png) |  |
| student | `/chat` | 375 | có lớp phủ / menu (soanthao) | true | border-bottom-color:140ms, border-left-color:140ms, border-right-color:140ms | 0/0/0/0 | [ảnh](shots/chat@dlg-soanthao__student__375.png) | ô nhập chat khi focus có khung đỏ dày bao cả vùng ô nhập + nút «Gửi» ở góc dưới phải |

**Focus bàn phím** (Tab 10 lần từ đầu trang; `vòng` = có outline / shadow trên phần tử đang focus)

- `/` · teacher · 1440 px · [ảnh](shots/home@focus__teacher__1440.png)
  1. a «Bỏ qua điều hướng» vòng=shadow 8  
  2. button «Chọn lớp, đang xem 761987 · An» vòng=shadow 10  
  3. button «Tìm nhanh hoặc đi đến» vòng=shadow 11  
  4. button «Thông báo, 8 chưa đọc» vòng=shadow 10  
  5. button «Tài khoản: TS. Lê Thu Hà» vòng=shadow 10  
  6. a «EduPilot — Hôm nay» vòng=shadow 0  
  7. a «Hôm nay» vòng=shadow 96  
  8. a «Hộp thư hỗ trợ» vòng=shadow 131  
  9. a «Sinh viên» vòng=shadow 166  
  10. a «Điểm danh» vòng=shadow 201
- `/inbox` · teacher · 1440 px · [ảnh](shots/inbox@focus__teacher__1440.png)
  1. a «Bỏ qua điều hướng» vòng=shadow 8  
  2. button «Chọn lớp, đang xem 761987 · An» vòng=shadow 10  
  3. button «Tìm nhanh hoặc đi đến» vòng=shadow 11  
  4. button «Thông báo, 8 chưa đọc» vòng=shadow 10  
  5. button «Tài khoản: TS. Lê Thu Hà» vòng=shadow 10  
  6. a «EduPilot — Hôm nay» vòng=shadow 0  
  7. a «Hôm nay» vòng=shadow 96  
  8. a «Hộp thư hỗ trợ» vòng=shadow 131  
  9. a «Sinh viên» vòng=shadow 166  
  10. a «Điểm danh» vòng=shadow 201
- `/gradebook` · teacher · 1440 px · [ảnh](shots/gradebook@focus__teacher__1440.png)
  1. a «Bỏ qua điều hướng» vòng=shadow 8  
  2. button «Chọn lớp, đang xem 761987 · An» vòng=shadow 10  
  3. button «Tìm nhanh hoặc đi đến» vòng=shadow 11  
  4. button «Thông báo, 8 chưa đọc» vòng=shadow 10  
  5. button «Tài khoản: TS. Lê Thu Hà» vòng=shadow 10  
  6. a «EduPilot — Hôm nay» vòng=shadow 0  
  7. a «Hôm nay» vòng=shadow 96  
  8. a «Hộp thư hỗ trợ» vòng=shadow 131  
  9. a «Sinh viên» vòng=shadow 166  
  10. a «Điểm danh» vòng=shadow 201
- `/chat` · student · 1440 px · [ảnh](shots/chat@focus__student__1440.png)
  1. textarea «Câu hỏi của bạn» vòng=KHÔNG 771  
  2. (body)  
  3. a «Bỏ qua điều hướng» vòng=shadow 8  
  4. button «Chọn lớp, đang xem 761987 · An» vòng=shadow 10  
  5. button «Tìm nhanh hoặc đi đến» vòng=shadow 11  
  6. button «Thông báo, 8 chưa đọc» vòng=shadow 10  
  7. button «Tài khoản: Nguyễn Minh Trung» vòng=shadow 10  
  8. a «EduPilot — Hôm nay» vòng=shadow 0  
  9. a «Hôm nay» vòng=shadow 65  
  10. a «Chat riêng» vòng=shadow 100
- `/` · student · 375 px · [ảnh](shots/home@focus__student__375.png)
  1. a «Bỏ qua điều hướng» vòng=shadow 8  
  2. a «EduPilot — Hôm nay» vòng=shadow 6  
  3. button «Chọn lớp, đang xem 761987 · An» vòng=shadow 6  
  4. button «Tìm nhanh hoặc đi đến» vòng=shadow 6  
  5. button «Thông báo, 8 chưa đọc» vòng=shadow 6  
  6. button «Tài khoản: Nguyễn Minh Trung» vòng=shadow 6  
  7. a «Bài tập 04 — Phân tích một vụ » vòng=shadow 239  
  8. a «Mở bài tập» vòng=shadow 364  
  9. a «Buổi 11 · An ninh mạng Thứ 2, » vòng=shadow 550  
  10. a «Hôm nay» vòng=shadow 756

## 5. Trang giới thiệu — chuyển động và ngân sách (khách)

| Cỡ | `prefers-reduced-motion` | JS nén KB | JS giải nén KB | Tổng KB | LCP ms | Luật reduced-motion trong CSS | Scroll-driven CSS | Video | Số animation ngay sau khi cuộn → sau 1 s (mỗi bước ~800 px) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1440 | no-preference | 216 | 679 | 558 | 1468 | 2 | có | 1 | 29→29, 29→29, 29→29, 29→29, 29→29 |
| 1440 | reduce | 216 | 679 | 558 | 1464 | 2 | có | 1 | 0→0, 0→0, 0→0, 0→0, 0→0 |
| 375 | no-preference | 216 | 679 | 466 | 1448 | 2 | có | 1 | 29→29, 29→29, 29→29, 29→29, 29→29, 29→29, 29→29 |
| 375 | reduce | 216 | 679 | 466 | 1468 | 2 | có | 1 | 0→0, 0→0, 0→0, 0→0, 0→0, 0→0, 0→0 |

- **Bản cũ** (xem và đo trước 19:25; ảnh gốc không còn vì demo được cập nhật giữa lúc chụp): hero chữ + 2 nút, ảnh chụp ứng dụng tĩnh cắt ngang ở đáy khung, ba số đo «0 / 146», «30 / 30», «+0,05 %» là chữ tĩnh, hai danh sách 6 mục «Cho giảng viên» / «Cho sinh viên» với ảnh nhỏ không đọc được chữ, không video, không sơ đồ luồng, không chuyển động nào; JS nén 215 KB, 0 luật scroll-driven, 1 luật reduced-motion, 0 video; LCP đo được 1,4–1,6 s (một lần 6,9 s lúc 4 trình duyệt cùng chạy).
- **Bản mới** (demo đổi lúc ~19:25, commit `765dabfb`): video có poster (`preload=none`, có điều khiển, không tự phát, mp4 + webm, phụ đề in trong poster), sơ đồ 3 bước «AI soạn nháp → Giảng viên duyệt → Mã tính điểm» có đường nối đỏ, ba số đo đếm lên (đo được «19» đang chạy lên tới «146»), tab «Giảng viên | Sinh viên»; hiện dần khi cuộn bằng CSS scroll-timeline.
- Bản mới, `prefers-reduced-motion: reduce`: 0 animation đang chạy (so với 29 ở chế độ thường), số đo hiện ngay giá trị cuối «146»; có 2 luật reduced-motion trong CSS. Chế độ thường: 29 animation chạy liên tục sau mỗi lần cuộn (không về 0 sau 1 s).
- Bản mới: JS nén 216 KB (giải nén 679 KB) — sát ngưỡng 256 KB của sprint; LCP 1,47 s ở 1440 và 375; tổng truyền 558 KB (1440) / 466 KB (375), video chưa tải (media = 0 KB) trước khi bấm.

## 6. Ghi chú thêm

- **Thời gian thấy nội dung** (142 lượt đầy dữ liệu của 6 tài khoản đã đăng nhập, 1440 và 375, từ điều hướng tới khi `<main>` có chữ, máy Mac → VPS 12 vCPU qua Internet): trung vị 1,05 s, p90 1,77 s, lớn nhất 2,6 s (`/attendance` trợ giảng 1440). LCP trung vị 1,07 s, p90 1,74 s, lớn nhất 2,67 s. Không lời gọi API nào > 800 ms. Chậm nhất: `/attendance` (2,6 / 2,3 s), `/students` (2,1 / 2,0 s), `/library/‹id›` ở 375 (1,9 s).
- **Console:** mọi lần tải của khách (16 lượt) có lỗi `401` từ `POST /api/v1/auth/refresh` (cố làm mới phiên khi chưa có phiên); 110 / 158 lượt đầy có cảnh báo «resource … was preloaded using link preload but not used within a few seconds» (tệp CSS, `brand/logo-edupilot.svg`, và cả `landing/giang-vien-duyet-bai.webp` — ảnh của trang giới thiệu — bị preload ở các màn trong ứng dụng). Không có lỗi JS (`pageerror`) nào, axe ở trạng thái đầy = 0 trên cả 158 lượt; chỉ hai mục axe mức cao xuất hiện khi mở lớp phủ (⌘K `nested-interactive` ×16, nút «…» của thành viên lớp `target-size` ×1).
- **Mật khẩu seed trong prompt không đăng nhập được** (401 `INVALID_CREDENTIALS` cả 6 tài khoản). Demo dùng `SEED_DEFAULT_PASSWORD` do `deploy/vps-env.sh` sinh ngẫu nhiên; QC đọc đúng biến đó ở `/opt/edupilot/.env.local` qua ssh (chỉ đọc một dòng, không in ra, không ghi vào repo). Cần sửa dòng mật khẩu trong prompt / tài liệu sprint cho các vai sau.
- **Không chụp trên demo vì có tác dụng ghi:** `/exams/‹id›/take` và `/practice/‹attempt›` (mở là tạo lượt làm bài), `/join/‹mã›` với `sv.moi` (vào lớp thật), xung đột 409 (cần ghi hai lần), mọi hộp thoại xác nhận chưa mở tới bước bấm. Các trạng thái này cần stack test `-p ep-qc`; chưa chụp trong đợt này.
- **Mất mạng** được mô phỏng bằng huỷ mọi lời gọi `/api/v1/**` ngoại trừ `/auth/` (trang vẫn nạp được); chưa thử ngắt mạng giữa lúc đang nhập hay đang gửi.
- **Phiên bị đá ra** giữa chừng (về `/login`) ở 8 lượt chụp đầu (Hộp thư giảng viên, Chat riêng sinh viên và vài ảnh «đang tải»); nguyên nhân chưa rõ (nghi ngờ nhiều trình duyệt song song cùng một tài khoản). Đã đăng nhập lại và chụp lại; script `shoot.mjs` tự thử lại một lần khi gặp.
- **Dấu vết QC trên demo:** mỗi lần đăng nhập tạo một phiên mới nên mục «Thiết bị đang đăng nhập» của `teacher@`, `ta@`, `admin@`, `sv.gioi@`, `sv.kha@`, `sv.moi@` hiện thêm hàng chục dòng (Chrome trên macOS) cho tới khi hết hạn; QC không bấm «Đăng xuất mọi thiết bị khác» để khỏi đăng xuất phiên của chủ dự án. Video chat sinh viên (`sv.kha`) để lại 1 phiên chat mới và 1 lượt gọi LLM thật.
- **Lazy ảnh:** trang giới thiệu nạp ảnh chụp ứng dụng lười; ảnh chụp trang phải cuộn hết trang trước khi chụp mới thấy đủ ảnh (đã làm trong `shoot.mjs`).
- **Hộp thoại / drawer:** phần lớn «hộp thoại» của ứng dụng thực chất là biểu mẫu nội tuyến (không có lớp phủ, không bẫy focus); chỉ «Chốt điểm», «Tạo lại mã», «Tạo lịch buổi học», popover thông báo / tài khoản / chọn lớp, hộp tìm ⌘K và tờ đáy «Thêm» ở 375 là lớp phủ.
- **Video** (`video/*.webm`, 1280×720, 25 fps, ghi bằng Playwright): `landing.webm` 10 s (khách cuộn hết trang; quay lúc 19:27 khi demo đã là bản landing mới), `teacher-grading.webm` 18 s (Hôm nay → Chấm bài → mở một bài, bấm «Đoạn trích», cuộn; không bấm Duyệt), `student-chat.webm` 33 s (`sv.kha` gõ «Brute force mật khẩu là gì và phòng chống thế nào?» rồi gửi).
- **Chuyển động quan sát được trong video:** chuyển trang ở ứng dụng là cắt cứng (không chuyển cảnh); hàng chờ Chấm bài hiện khung xương trước khi có dữ liệu; chat chờ ~8 s ở dòng «Đang tìm trong tài liệu…» rồi chữ chảy dần; trong lúc chờ và lúc chữ chảy, tin nhắn của sinh viên hiện **hai lần** (hai bong bóng «Bạn» giống hệt, ~5 s đầu) rồi còn một; lúc chữ đang chảy hiện nguyên ký hiệu Markdown «**1. Brute-force mật khẩu là gì?**» và gạch «- » cho tới khi trả lời xong mới dựng đúng; nguồn tham khảo là 6 dòng cùng tên «Network Security Threats» khác số trang.
- **Tác dụng phụ trên demo của video chat:** câu trả lời kết thúc bằng «AI chưa đủ chắc chắn về câu này» + «Đang chờ giảng viên · vừa gửi», tức hệ thống đã chuyển câu hỏi thành **một yêu cầu mới trong Hộp thư hỗ trợ** của giảng viên 761987. QC không xoá được từ giao diện sinh viên; PM cần biết khi chủ dự án demo (số yêu cầu đang chờ tăng thêm 1).
- **Demo đổi giữa lúc chụp:** frontend khởi động lại lúc ~19:25 (+07) với trang giới thiệu mới (commit `765dabfb`). Chỉ trang giới thiệu được xác nhận là đổi; ảnh `*__anon__*` của trang giới thiệu và `landing-scroll-*` là bản mới (bản cũ chỉ còn mô tả chữ ở mục trang giới thiệu). Với các màn trong ứng dụng, QC không đối chiếu từng ảnh giữa lượt chụp đầu và lượt chụp lại nên không loại trừ chênh lệch nhỏ; thời điểm chụp từng ảnh không còn trong tên tệp (bước nén ghi đè mtime).
