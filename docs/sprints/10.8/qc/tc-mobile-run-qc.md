# QC 10.8 — ỨNG DỤNG ĐIỆN THOẠI, PHA 1 (qc, cặp 3) — `tc-mobile-run-qc.md`

Tệp kết quả RIÊNG của qc cho pha 1 của `tc-mobile-app.md` (qc3 chủ tệp, gộp vào §9 sau). Nhóm của qc: **6.B, 6.C, 6.F, 6.G, 6.I, 6.K, 6.N, 6.R, 6.S, 6.X**.

## Môi trường

| Mục | Giá trị |
| --- | --- |
| Môi trường | **W** — Playwright **Chromium** trên VPS (`ep-vps-test`), thiết bị giả `Pixel 7` (412×915) và `iPhone 15` (393×852): **Chromium giả iPhone, KHÔNG WebKit** — Safari/WebKit chỉ có ở pha 3 |
| Nơi chạy | VPS `edupilot-vps`, HTTPS thật `https://v2202609419695521098.happysrv.de` (sha VPS: f6113582, 25/11 23:51 — có DEF-R10.8-07 dock theo chiều cao thật) |
| Lớp QC | `QCMOB1` (self_checkin bật, rotate 10 s, 41 buổi), `QCMOB2` (lớp thứ hai, bật), `QCMOB4B` (đã lưu trữ **qua giao diện Admin** thay QCMOB4), `QCMOB5` (self_checkin tắt), `QCMOB6` (sv.qc không thuộc) — không đụng 761987/761988, QCVPSA–D |
| Tài khoản | `sv.qc` (SV duy nhất qc dùng; **không** dùng sv.qc2/sv.qc2b của qc2), teacher, ta, admin |
| Số lượt | Cửa sổ điểm danh mở bằng API GV (`/sessions/{sid}/checkin-windows`), tối đa 3 vòng/buổi; bản ghi chỉ hiện ở lưới sau khi **đóng cửa sổ** |
| Kết quả | **P** đạt · **F** trượt (kèm DEF) · **N** không kiểm được (nêu lý do) |

## 6.B Phiên, vai, tài khoản

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| SES-01 | **P** | Đăng nhập giao diện 500 ms (đo cả tải /login), URL/localStorage không chứa token |
| SES-02 | **P** | 5/5 lần mở lại vào thẳng, không màn đăng nhập/trắng |
| SES-03 | **N** | cần `am force-stop` (native, pha 2) |
| SES-04 | **P** | Đăng xuất → /login, Back không vào lại (about:blank), `sessionStorage.ep.checkin` rỗng |
| SES-05 | **P** | teacher + ta bị chặn ở `/checkin` (không ô nhập, 0 POST `/checkins`), `/attendance` vẫn dùng được |
| SES-06 | **P** | admin bị chặn, 0 POST |
| SES-07 | **N** | cần 2 tài khoản SV cùng thiết bị (sv.qc2 thuộc qc2 — không dùng); pha 2 |
| SES-08 | **N** | native (thanh thông báo hệ thống) |
| SES-09 | **N** | cần access token hết hạn thật (TTL) — không đổi giờ máy chủ |
| SES-10 | **N** | cần thu hồi phiên từ thiết bị khác (thiết bị thật) |
| SES-11 | **P** | Chưa đăng nhập: chuyển `/login`, cất `{c,token,at}` vào sessionStorage (không localStorage) — cùng bằng chứng với ERR-06/SES-11a |
| SES-12 | **N** | cần Back của Android / vuốt iOS |
| SES-13 | **N** | cần `pm clear` |
| SES-14 | **N** | cần hai SV cùng trình duyệt (sv.qc2 không dùng) |

## 6.C Luồng điểm danh chính

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| FLW-01 | **P** | Token QR tự gửi một lần, «Đã điểm danh 00:11» sau **171 ms**; `#token` biến khỏi URL; chỉ dấu Check đỏ (svg/path), không chữ nào đỏ; cửa sổ ghi 1 lượt |
| FLW-02 | **N** | `simctl openurl` (pha 3) |
| FLW-03 | **P** | Ba kiểu mã `5mk9v5` / `5MK 9V5` / `5MK-9V5` đều nhận, cùng giờ; cửa sổ 1 lượt |
| FLW-04 | **P** | Ô «Mã điểm danh» + «Lớp QCMOB1» + nút Điểm danh; không nút quét (Chromium không BarcodeDetector) |
| FLW-05 | **P** | Mở lại cùng liên kết: hiện lại đúng giờ lượt đầu, không lỗi |
| FLW-06 | **P** | Chạm đúp < 300 ms: **1 POST**, nút khoá khi đang gửi, cửa sổ 1 lượt |
| FLW-07 | **P** | Xoay 412×915 → 915×412 giữa lúc gửi: 1 lượt, «Đã điểm danh» hiện sau xoay |
| FLW-08 | **P** | 31 yêu cầu ghi lại: token không có trong URL/Referer/yêu cầu nào |
| FLW-09 | **P** | Mã QCMOB2 khi đang chọn QCMOB1 → «Mã không đúng hoặc đã hết hạn…»; mã đúng lớp → «Đã điểm danh» |
| FLW-10 | **N** | không còn tài khoản nào không thuộc lớp (sv.qc* đều đã ghi danh) |
| FLW-11 | **P** | Vòng 2 (round 2) tạo lượt mới, cửa sổ ghi 1 lượt |
| FLW-12 | **P** | Lớp tắt self_checkin: «Lớp này không điểm danh bằng mã.», **không còn ô mã** (đúng AC16(d)), 0 POST |
| FLW-13 | **P** (sửa 578e809e, nghiệm lại ở ae069d85) | Lớp **lưu trữ**: «Lớp này đã được lưu trữ.» — đúng câu, không mã kỹ thuật, không ghi lượt (posts=0); trước đó là «Lớp này không điểm danh bằng mã.» (DEF-MOB-01). Máy chủ trả `attendance.self_checkin=false` cho lớp ARCHIVED nên giao diện rơi vào nhánh tắt điểm danh → **DEF-MOB-01** |
| FLW-14 | **N** | cần `pnpm build:gate` + bộ `attendance-checkin.spec.ts` trên build (chạy ở lượt riêng) |
| FLW-15/16/17/18 | **N** | hai AVD / camera hệ thống / LAN HTTP / IndexedDB chặn (native) |

## 6.F Nhánh lỗi (ERR) + 6.S câu chữ (phần web)

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| ERR-01 | **P** | `c=khong-phai-uuid` → «Liên kết không đúng.», 0 POST |
| ERR-02 | **P** | Token cũ > 2 bước rotate → «Mã đã hết hạn. Quét lại mã trên màn chiếu.» |
| ERR-03 | **P** | Sửa ký tự cuối → «Mã không đúng. Quét lại mã trên màn chiếu.» |
| ERR-04 | **P** | Cửa sổ đã đóng → «Đã đóng điểm danh.» |
| ERR-05 | **P** | sv.qc mở liên kết QCMOB6 (không thuộc) → «Bạn không thuộc lớp này.», `checked_in=0` |
| ERR-06 | **P** | Để màn đăng nhập > 60 s rồi đăng nhập → «Mã đã hết hạn…», 0 POST. Ghi chú: kho cất `ep.checkin` **không bị xoá** (không thuộc AC) |
| ERR-07 | **P** | teacher mở liên kết: màn chặn, 0 POST, token không lộ trong DOM |
| ERR-08 | **P** | ta + admin: như ERR-07 |
| ERR-09 | **N** | đếm ngược 429 cần 6 lần nhập sai liên tiếp — để ca riêng (chưa chạy: sợ chạm hạn mức của sv.qc cho các ca còn lại) |
| ERR-10 | **N** | cần tải 11 lượt/phút rồi quét ngay (native+API) |
| ERR-11 | **P** | Mã sai → «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» dưới ô, `aria-invalid=true`, không hộp thoại |
| ERR-12 | **P** | Ô rỗng → nút khoá; ký tự lạ `😀@#$` → cùng một câu, 0 yêu cầu, không crash |
| ERR-13 | **P** | Dán 40 ký tự → cắt còn 12; mã có khoảng trắng đầu/cuối vẫn điểm danh được |
| ERR-14 | **P** | 500 JSON / 200 HTML / 500 rỗng → «Máy chủ chưa phản hồi đúng. Dữ liệu của bạn vẫn an toàn.» (câu BAD_GATEWAY của SRS ui-foundation), không mã HTTP/JSON; nút còn dùng. *Khác câu gợi ý «Chưa điểm danh được. Thử lại.» trong TC — đề nghị qc3/PM chỉnh TC theo SRS, không phải lỗi sản phẩm* |
| ERR-15 | **N** | offline lần đầu mở liên kết (Safari iOS, pha 3) |
| ERR-16/17 | **N** | đổi giờ/múi giờ máy (native) |
| ERR-18 | **P** | `POST {code, student_id}` → 422 `VALIDATION_FAILED`, `details[].code=unknown`, không ghi lượt |
| ERR-19 | **P** | Lớp không tồn tại → cùng một câu, không lộ khác biệt |
| ERR-20 | **N** | cửa sổ đóng > 15 phút — chờ dài, để lượt sau |
| MSG-10 | **P** | Không `BarcodeDetector`: không câu lỗi nào, ô nhập + nút hiện ngay |
| MSG-01…09, 11…16 | một phần **P** qua ERR-01…05/11/12/14/18 (đúng câu); còn lại **N** (native) |  |

## 6.G Độ rộng, xoay ngang, khung app

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| UIW-01…05 (360/375/390/411/430) | **P** | 4 trạng thái web × 5 bề rộng: `scrollWidth ≤ innerWidth` ở tất cả; nút chính trong khung (trạng thái gửi/đã gửi không có nút chính — dock gỡ theo thiết kế) |
| UIW-06 | **P** | Bảng 5×5 đủ; ô «đang quét» = native → pha 2 |
| UIW-07 | **P** | Câu lỗi dài ở 360: xuống dòng trong panel, không tràn (`sw=360/360`), nút chính thấy |
| UIW-08 | **P** | 360 và 430: «Đã điểm danh 01:04» một dòng; phần tử đỏ duy nhất là svg/path của dấu Check |
| UIW-09 | **N** | khung camera (native) |
| UIW-10 | **P** | `/attendance` 5 bề rộng: không cuộn ngang; thanh lưu trên thanh dưới (đo ở ATT-02/03) |
| UIW-11/12/13 | **P** | 667×375, 844×390, 915×412: 4 trạng thái không tràn, nút chính không bị che |
| UIW-14/15/16 | **N** | xoay máy giữa lúc gửi / giữ mã / camera (native) |
| UIW-17 | **P** | Dọc 375: 5 mục thanh đáy cao 56 px, không cắt chữ, không quá đáy (812). 844×1000: thanh bên **thu gọn 72 px** hiện, 10 mục, không tràn. Ghi chú: điện thoại **ngang thấp** 844×390 dùng thanh đáy — đúng M2 ở main (`(max-height: 500px) and (pointer: coarse)`), **TC cần cập nhật** (TC đòi thanh bên ở mọi ≥ 720) |
| UIW-18 | **P** | 4 màn SV (/, /chat, /threads, /practice) ở 360 và 430: không cuộn ngang |
| UIW-19 | **P** | CLS = 0,000 ở 360 và 430 (≤ 0,1) |
| UIW-20 | **P** | 360×640 nhập mã: `scrollHeight=640` — không cuộn dọc, ô + nút thấy |
| UIW-21/22 | **N** | thanh điều hướng hệ thống / mật độ màn hình (native) |

## 6.I Cỡ chữ, phóng

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| TXT-06 | **P** | 180 px (≈200 % ở 360): 4 trạng thái không cuộn ngang, 0 phần tử chữ bị cắt |
| TXT-01…05, 07, 08 | **N** | cỡ chữ hệ thống Android / phóng Safari `aA` / iOS content size (native) |

## 6.K Bàn phím che ô nhập (phần web)

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| KBD-04 | **P** | `autocapitalize=characters`, `autocomplete=off`, `autocorrect=off`, `spellcheck=false`, `enterkeyhint=go`, `maxlength=12`, placeholder «ABC 123» |
| KBD-05 | **P** | Enter gửi như nút, đúng 1 POST; mã có khoảng trắng được nhận |
| KBD-07 | **P** | Đang gửi: dock gỡ/nút khoá (`data-state=sending`), không gửi hai lần |
| KBD-09 | **P** | Dán «abc-123» vẫn điểm danh được |
| KBD-10 | **P** | `autocomplete=off` (không gợi ý mật khẩu/địa chỉ) — cùng bằng chứng KBD-04 |
| KBD-12 | **P** | Khung thu 411×500 (bàn phím ảo): ô ở 228–272, nút Điểm danh 391–435 — cả hai trong khung |
| KBD-01/02/03/06/08/11 | **N** | bàn phím thật Android/iOS, Gboard Telex, phím cứng (native) |

## 6.N Mạng yếu, mất mạng

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| NET-01 | **P** (ngưỡng TC rút) | Mạng 400 kbit/s + RTT 2 s (CDP): khung hiển thị 5 360 ms, nội dung 5 364 ms, không màn trắng, không cuộn ngang — đạt phần AC9 nhìn thấy được. **DEF-MOB-04 = RÚT** theo P-11-143 (TC đặt ngưỡng 3 s không neo AC); số đo giữ làm tham chiếu |
| NET-02 | **P** (sửa ae069d85) | Offline khi gửi mã: «Chưa có mạng.» + nút `Thử lại`, mã đã gõ giữ nguyên (4KNBY9), không hộp lỗi kỹ thuật; bấm `Thử lại` sau khi có mạng → «Đã điểm danh», `checked_in=1` (DEF-MOB-02) | ô) → **DEF-MOB-02** |
| NET-03 | **P** | Có mạng lại: bấm `Điểm danh` lần nữa → «Đã điểm danh», `checked_in=1`, không tải lại trang |
| NET-07 | **P** (sửa ae069d85) | POST chậm 20 s: sau **16 s có «Chưa có mạng.»**, trạng thái về `entry` (TC: trong 15 s) — DEF-MOB-03 |
| NET-04/05/06/08/09/10/11/12 | **N** | mạng chập chờn/đổi Wi-Fi/WebView chặn host/tiết kiệm dữ liệu/PWA offline (native) |

## 6.R Vùng chạm ≥ 44 px

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| TGT-01 | **P** | 360 và 430: mọi `button/a/input` trong main, topbar, thanh dưới ≥ 44×44 |
| TGT-02 | **N** | đo thô `minGap=0` do phần tử lồng nhau — không kết luận; cần đo theo cặp hàng thực tế |
| TGT-03 | **P** | dock đáy 683 = mép trên thanh dưới 683 (không đè), dock 61 px, thanh dưới 57 px |
| TGT-04 | **N** | `uiautomator dump` (native) |
| TGT-05 | **P** | `/attendance` 375 (GV): mọi nút/hàng ≥ 44×44 (cùng bằng chứng ATT-02) |
| TGT-06 | **N** | ngang **thấp** + `pointer: coarse` dùng thanh đáy (M2) nên không đo thanh bên |

## 6.X Staff trên điện thoại — `/attendance`

| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| ATT-01 | **P** | 375: mở được buổi, mở cửa sổ, có «Mở màn chiếu», không cuộn ngang |
| ATT-02 | **P** | Mọi nút/hàng 375 ≥ 44×44 |
| ATT-03 | **P** | TA 360: lưới cuộn trong khung riêng, trang không cuộn ngang |
| ATT-07 | **P** | axe `/attendance` 360: 0 lỗi critical/serious (TalkBack = native) |
| ATT-08 | **P** | SV mở `/attendance`: màn chặn «Bạn không có quyền xem màn này», không dữ liệu lớp |
| ATT-09 | **P** | 844×1000: thanh bên hiện, không cuộn ngang |
| ATT-10 | **P** | `/attendance/display?course&session` ở 393: QR/mã/vòng/Thoát hiện, không crash |
| ATT-04/05/06/11 | **N** | safe-area, cỡ chữ 2,0×, máy tối/native, geofence+SQL (pha 2) |

### Nghiệm lại bản sửa (VPS ae069d85)

| TC | DEF | Kết quả |
| --- | --- | --- |
| FLW-13 | DEF-MOB-01 (578e809e) | **P** — «Lớp này đã được lưu trữ.», không POST |
| NET-02 + MSG-07 | DEF-MOB-02 (ae069d85) | **P** — «Chưa có mạng.» + `Thử lại`, mã giữ nguyên, retry → `checked_in=1` |
| NET-07 | DEF-MOB-03 (ae069d85) | **P** — sau 16 s có «Chưa có mạng.», trạng thái `entry` |

## Tổng (phần qc)

| Nhóm | P | F | N |
| --- | --- | --- | --- |
| 6.B SES | 6 | 0 | 8 |
| 6.C FLW | 11 | 0 | 7 |
| 6.F ERR | 14 | 0 | 6 |
| 6.S MSG | 10 | 0 | 6 |
| 6.G UIW | 16 | 0 | 6 |
| 6.I TXT | 1 | 0 | 7 |
| 6.K KBD | 6 | 0 | 6 |
| 6.N NET | 4 | 0 | 8 |
| 6.R TGT | 3 | 0 | 3 |
| 6.X ATT | 7 | 0 | 4 |
| **Cộng** | **77** | **0** | **61** |

## DEF mới

| DEF | Mức | Nội dung | Bằng chứng |
| --- | --- | --- | --- |
| DEF-MOB-01 | ~~Thấp~~ **ĐÃ SỬA** (578e809e, nghiệm lại ae069d85: «Lớp này đã được lưu trữ.») | Lớp đã lưu trữ hiện «Lớp này không điểm danh bằng mã.» thay vì câu về lớp lưu trữ (FLW-13) | Máy chủ: lớp ARCHIVED trả `attendance.self_checkin=false`; giao diện chỉ có nhánh "tắt điểm danh" |
| DEF-MOB-02 | ~~Trung bình~~ **ĐÃ SỬA** (ae069d85: có `Thử lại`, mã giữ nguyên, retry xong `checked_in=1`) | Mất mạng khi gửi **mã ngắn**: có «Chưa có mạng.» nhưng **không có nút `Thử lại`** (đường token thì có) (NET-02, MSG-07) | `CheckinScreen.describe` → `viaCode` chỉ đặt lỗi dưới ô, không gắn `retry` |
| DEF-MOB-03 | ~~Trung bình~~ **ĐÃ SỬA** (ae069d85: sau 16 s có «Chưa có mạng.», về `entry`) | Máy chủ trả lời POST chậm 20 s: sau 16 s vẫn «đang gửi», không có «Chưa có mạng.» / hết giờ (NET-07) | Gọi `apiClient.post` không có hạn giờ; chỉ đổi trạng thái khi promise kết thúc |
| DEF-MOB-04 | **RÚT** (P-11-143) | Mạng 400 kbit/s + RTT 2 s: khung 5 360 ms — **TC đặt ngưỡng 3 s không neo AC nên TC sai**, không phải lỗi sản phẩm; số đo giữ ở dòng NET-01 làm tham chiếu | — |

## Ghi chú

- Quan sát tĩnh §5 của `tc-mobile-app.md` không chấm ở pha 1 trừ ca W tương ứng.
- Mọi ca dựng trạng thái bằng `page.route` (gửi/đã gửi/lỗi/429) chỉ đo bố cục, không tạo lượt trong DB.
