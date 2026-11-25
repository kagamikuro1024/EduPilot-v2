# QC 10.8 — ỨNG DỤNG ĐIỆN THOẠI, PHA 1 (qc, cặp 3) — `tc-mobile-run-qc.md`

Tệp kết quả RIÊNG của qc cho pha 1 của `tc-mobile-app.md` (qc3 chủ tệp, gộp vào §9 sau). Nhóm của qc: **6.B, 6.C, 6.F, 6.G, 6.I, 6.K, 6.N, 6.R, 6.S, 6.X**. **Mỗi mã một dòng** — bảng dưới là nguồn gộp duy nhất; bản máy đọc: `tc-mobile-run-qc.tsv` cùng thư mục.

## Môi trường

| Mục | Giá trị |
| --- | --- |
| Môi trường | **W** — Playwright **Chromium** trên VPS (`ep-vps-test`), thiết bị giả `Pixel 7` (412×915) và `iPhone 15` (393×852): **Chromium giả iPhone, KHÔNG WebKit** — Safari/WebKit chỉ có ở pha 3 |
| Nơi chạy | VPS `edupilot-vps`, HTTPS thật `https://v2202609419695521098.happysrv.de` (sha VPS khi chạy chính: f6113582; nghiệm lại bản sửa: ae069d85) |
| Lớp QC | `QCMOB1` (self_checkin bật, rotate 10 s, 41 buổi), `QCMOB2` (lớp thứ hai, bật), `QCMOB4B` (đã lưu trữ **qua giao diện Admin** thay QCMOB4), `QCMOB5` (self_checkin tắt), `QCMOB6` (sv.qc không thuộc) — không đụng 761987/761988, QCVPSA–D |
| Tài khoản | `sv.qc` (SV duy nhất qc dùng; **không** dùng sv.qc2/sv.qc2b của qc2), teacher, ta, admin |
| Số lượt | Cửa sổ điểm danh mở bằng API GV (`/sessions/{sid}/checkin-windows`), tối đa 3 vòng/buổi; bản ghi chỉ hiện ở lưới sau khi **đóng cửa sổ** |
| Kết quả | **P** đạt · **F** trượt (kèm DEF) · **N** không kiểm được (nêu lý do) |

## 6.B Phiên, vai, tài khoản

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| SES-01 | **P** | Đăng nhập qua giao diện 500 ms; URL/localStorage không chứa token |
| SES-02 | **P** | 5/5 lần mở lại vào thẳng, không màn đăng nhập/trắng |
| SES-03 | **N** | cần `am force-stop` rồi mở lại (native, pha 2) |
| SES-04 | **P** | Đăng xuất → /login; Back ra about:blank; `sessionStorage.ep.checkin` rỗng |
| SES-05 | **P** | teacher + ta bị chặn ở /checkin (không ô nhập, 0 POST); /attendance vẫn dùng được |
| SES-06 | **P** | admin bị chặn, 0 POST |
| SES-07 | **P** | sv.qc + sv.qc-b CÙNG context (IndexedDB `edupilot.device.v1` = một khoá): cả hai điểm danh được trong cùng cửa sổ; SQL `attendance_checkins` — cùng `device_id` 01a21458-a7df-744f-8ee3-8c168346378d, cả hai `flags` có `DEVICE_SHARED`; SV không thấy cờ trong app |
| SES-08 | **N** | native: thanh thông báo hệ thống |
| SES-09 | **N** | cần access token hết hạn thật (TTL); không đổi giờ máy chủ |
| SES-10 | **N** | cần thu hồi phiên từ thiết bị khác (thiết bị thật) |
| SES-11 | **P** | Chưa đăng nhập: chuyển /login, cất {c,token,at} vào sessionStorage (không localStorage) |
| SES-12 | **N** | cần Back của Android / vuốt iOS (native) |
| SES-13 | **N** | cần `pm clear` (native) |
| SES-14 | **P** | Đăng xuất KHÔNG xoá khoá thiết bị: IndexedDB `edupilot.device.v1` còn `{id:web,pub:124}` trước và sau; sv.qc-b đăng nhập lại trong CÙNG context → lượt thứ hai cùng `device_id` (SQL) → DB ghi `DEVICE_SHARED` |

## 6.C Luồng điểm danh chính

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| FLW-01 | **P** | Token QR tự gửi 1 lần, «Đã điểm danh 00:11» sau 171 ms; `#token` mất khỏi URL; chỉ dấu Check đỏ (svg/path) |
| FLW-02 | **N** | pha 1 (Chromium giả iPhone) không có rotate mới để mở liên kết bằng `simctl openurl`; **qc3 đã P ở pha 3 iOS** với cửa sổ `rotate_seconds=60` (token còn hạn khi trang tải, đúng 1 POST, DB `channel=WEB`) — khi gộp lấy **P** của qc3 |
| FLW-03 | **P** | Ba kiểu `5mk9v5` / `5MK 9V5` / `5MK-9V5` đều nhận, cùng giờ; cửa sổ 1 lượt |
| FLW-04 | **P** | Ô «Mã điểm danh» + «Lớp QCMOB1» + nút Điểm danh; không nút quét (Chromium không BarcodeDetector) |
| FLW-05 | **P** | Mở lại cùng liên kết: hiện lại đúng giờ lượt đầu, không lỗi |
| FLW-06 | **P** | Chạm đúp < 300 ms: 1 POST, nút khoá khi đang gửi, cửa sổ 1 lượt |
| FLW-07 | **P** | Xoay 412×915 → 915×412 giữa lúc gửi: 1 lượt, «Đã điểm danh» hiện sau xoay |
| FLW-08 | **P** | 31 yêu cầu ghi lại: token không có trong URL/Referer/yêu cầu nào |
| FLW-09 | **P** | Mã QCMOB2 khi đang chọn QCMOB1 → «Mã không đúng hoặc đã hết hạn…»; mã đúng lớp → «Đã điểm danh» |
| FLW-10 | **N** | không còn tài khoản nào không thuộc lớp (sv.qc* đều đã ghi danh) |
| FLW-11 | **P** | Vòng 2 tạo lượt mới, cửa sổ ghi 1 lượt |
| FLW-12 | **P** | Lớp tắt self_checkin: «Lớp này không điểm danh bằng mã.», không còn ô mã, 0 POST |
| FLW-13 | **P** | Sửa 578e809e, nghiệm lại ae069d85: «Lớp này đã được lưu trữ.», không POST (DEF-MOB-01) |
| FLW-14 | **N** | cần `pnpm build:gate` + `attendance-checkin.spec.ts` trên build (lượt riêng) |
| FLW-15 | **N** | cần hai AVD (native) |
| FLW-16 | **N** | cần camera hệ thống (native) |
| FLW-17 | **N** | cần LAN HTTP (native) |
| FLW-18 | **N** | cần chặn IndexedDB (native) |

## 6.F Nhánh lỗi (ERR)

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| ERR-01 | **P** | `c=khong-phai-uuid` → «Liên kết không đúng.», 0 POST |
| ERR-02 | **P** | Token cũ > 2 bước rotate → «Mã đã hết hạn. Quét lại mã trên màn chiếu.» |
| ERR-03 | **P** | Sửa ký tự cuối → «Mã không đúng. Quét lại mã trên màn chiếu.» |
| ERR-04 | **P** | Cửa sổ đã đóng → «Đã đóng điểm danh.» |
| ERR-05 | **P** | sv.qc mở QCMOB6 → «Bạn không thuộc lớp này.», checked_in=0 |
| ERR-06 | **P** (web) | Chromium, VPS 8d2ec456: để màn đăng nhập > 60 s → «Mã đã hết hạn…», 0 POST; kho cất `ep.checkin` nay **rỗng** sau 60 s. qc3 tái hiện **F trên iOS @40281db4** (về ô nhập, không có câu) — khác nền tảng, PM quyết |
| ERR-07 | **P** | teacher: màn chặn, 0 POST, token không lộ trong DOM |
| ERR-08 | **P** | ta + admin: như ERR-07, 0 POST, không lộ token |
| ERR-09 | **P** | 6 mã sai liên tiếp: 5×422 rồi 429 (`Retry-After: 60`); UI «Thử lại sau 33 giây.» đếm ngược 33→32, nút Điểm danh khoá, **ô mã bị gỡ** trong lúc đếm; hết giờ về ô nhập. Lý do hiện trong panel, KHÔNG gắn aria lên nút (quan sát nhỏ, không chấm) |
| ERR-10 | **N** | cần tải 11 lượt/phút rồi quét ngay (native+API) |
| ERR-11 | **P** | Mã sai → «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» dưới ô, aria-invalid=true, không hộp thoại |
| ERR-12 | **P** | Ô rỗng → nút khoá; ký tự lạ `😀@#$` → cùng một câu, 0 yêu cầu, không crash |
| ERR-13 | **P** | Dán 40 ký tự → cắt còn 12; khoảng trắng đầu/cuối vẫn nhận |
| ERR-14 | **P** | 500 JSON / 200 HTML / 500 rỗng → 3 câu tiếng Việt của SRS ui-foundation, không mã HTTP/JSON, nút còn dùng (TC cần chỉnh theo SRS) |
| ERR-15 | **N** | offline lần đầu mở liên kết (Safari iOS, pha 3) |
| ERR-16 | **N** | cần đổi giờ máy (native) |
| ERR-17 | **N** | cần đổi múi giờ máy (native) |
| ERR-18 | **P** | POST {code, student_id} → 422 VALIDATION_FAILED, details[].code=unknown, không ghi lượt |
| ERR-19 | **P** | Lớp không tồn tại → cùng một câu, không lộ khác biệt |
| ERR-20 | **N** | cửa sổ đóng > 15 phút — chờ dài, lượt sau |

## 6.S Câu chữ (MSG)

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| MSG-01 | **P** | «Mã đã hết hạn. Quét lại mã trên màn chiếu.» (ERR-02) |
| MSG-02 | **P** | «Mã không đúng. Quét lại mã trên màn chiếu.» (ERR-03) |
| MSG-03 | **P** | «Đã đóng điểm danh.» (ERR-04) |
| MSG-04 | **P** | «Lớp này không điểm danh bằng mã.» (FLW-12) |
| MSG-05 | **P** | «Bạn không thuộc lớp này.» (ERR-05) |
| MSG-06 | **P** | «Thử lại sau 33 giây.» ({n} nguyên ≥1, đếm ngược) — cùng bằng chứng ERR-09; gateway trả 429 `RATE_LIMITED` + `Retry-After: 60` |
| MSG-07 | **P** | «Chưa có mạng.» + nút Thử lại (NET-02, sửa ae069d85) |
| MSG-08 | **N** | quyền camera bị từ chối — native (pha 2) |
| MSG-09 | **N** | camera đang bị app khác dùng — native (pha 2) |
| MSG-10 | **P** | Không BarcodeDetector: không câu lỗi, ô nhập + nút hiện ngay |
| MSG-11 | **N** | WebView không tải được trang — native |
| MSG-12 | **P** | «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» (ERR-11/12/13) |
| MSG-13 | **P** | «Liên kết không đúng.» (ERR-01) |
| MSG-14 | **P** | 5xx/HTML/thân rỗng → câu tiếng Việt của SRS (ERR-14) |
| MSG-15 | **P** | Quét DOM `innerText` + `aria-label`/`aria-description`/`alt` của CẢ 6 trạng thái (entry, đang gửi, 429, lỗi 5xx, mất mạng, đã điểm danh): 0 chuỗi cấm (`CHECKIN_`, `NotAllowedError`, `ERR_`, `net::`, `undefined`, `null`, `[object`, `Error`, `Failed`…), 0 mã HTTP |
| MSG-16 | **N** | cỡ chữ 2,0× (native: Android font scale / iOS content size) |

## 6.G Độ rộng, xoay ngang, khung app

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| UIW-01 | **P** | 360: 4 trạng thái web không tràn ngang (scrollWidth ≤ innerWidth) |
| UIW-02 | **P** | 375: 4 trạng thái không tràn ngang |
| UIW-03 | **P** | 390: 4 trạng thái không tràn ngang |
| UIW-04 | **P** | 411: 4 trạng thái không tràn ngang |
| UIW-05 | **P** | 430: 4 trạng thái không tràn ngang |
| UIW-06 | **P** | Bảng 5×5 đủ; ô «đang quét» = native → pha 2 |
| UIW-07 | **P** | Câu lỗi dài 360: xuống dòng trong panel, không tràn (sw=360/360), nút chính thấy |
| UIW-08 | **P** | 360 và 430: «Đã điểm danh 01:04» một dòng; phần tử đỏ duy nhất là svg/path dấu Check |
| UIW-09 | **N** | khung camera (native) |
| UIW-10 | **P** | /attendance 5 bề rộng: không cuộn ngang; thanh lưu trên thanh dưới |
| UIW-11 | **P** | 667×375: 4 trạng thái không tràn, nút chính không bị che |
| UIW-12 | **P** | 844×390: 4 trạng thái không tràn, nút chính không bị che |
| UIW-13 | **P** | 915×412: 4 trạng thái không tràn, nút chính không bị che |
| UIW-14 | **N** | xoay máy giữa lúc gửi (native) |
| UIW-15 | **N** | xoay máy khi đang giữ mã (native) |
| UIW-16 | **N** | xoay máy khi camera mở (native) |
| UIW-17 | **P** | 375: 5 mục thanh đáy 56 px, không cắt chữ, không quá đáy (812); 844×1000: thanh bên thu gọn 72 px, 10 mục, không tràn. Ghi chú: ngang thấp + pointer:coarse dùng thanh đáy theo M2 — TC cần cập nhật |
| UIW-18 | **P** | 4 màn SV (/, /chat, /threads, /practice) ở 360 và 430: không cuộn ngang |
| UIW-19 | **P** | CLS = 0,000 ở 360 và 430 (≤ 0,1) |
| UIW-20 | **P** | 360×640 nhập mã: scrollHeight=640, không cuộn dọc, ô + nút thấy |
| UIW-21 | **N** | thanh điều hướng hệ thống (native) |
| UIW-22 | **N** | mật độ màn hình (native) |

## 6.I Cỡ chữ, phóng

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| TXT-01 | **N** | cỡ chữ hệ thống Android (native) |
| TXT-02 | **N** | cỡ chữ hệ thống Android (native) |
| TXT-03 | **N** | phóng Safari aA (native, pha 3) |
| TXT-04 | **N** | iOS content size (native, pha 3) |
| TXT-05 | **N** | phóng trình duyệt khác (native) |
| TXT-06 | **P** | 180 px (≈200 % ở 360): 4 trạng thái không cuộn ngang, 0 phần tử chữ bị cắt |
| TXT-07 | **N** | cỡ chữ lớn + camera (native) |
| TXT-08 | **N** | cỡ chữ lớn trên iOS (native, pha 3) |

## 6.K Bàn phím che ô nhập (phần web)

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| KBD-01 | **N** | bàn phím thật Android (native) |
| KBD-02 | **N** | bàn phím thật iOS (native, pha 3) |
| KBD-03 | **N** | Gboard Telex (native) |
| KBD-04 | **P** | autocapitalize=characters, autocomplete=off, autocorrect=off, spellcheck=false, enterkeyhint=go, maxlength=12, placeholder «ABC 123» |
| KBD-05 | **P** | Enter gửi như nút, đúng 1 POST; mã có khoảng trắng được nhận |
| KBD-06 | **N** | phím cứng / bàn phím ngoài (native) |
| KBD-07 | **P** | Đang gửi: dock gỡ/nút khoá (data-state=sending), không gửi hai lần |
| KBD-08 | **N** | bàn phím chiếm chỗ trên iOS (native, pha 3) |
| KBD-09 | **P** | Dán «abc-123» vẫn điểm danh được |
| KBD-10 | **P** | autocomplete=off: không gợi ý mật khẩu/địa chỉ (cùng bằng chứng KBD-04) |
| KBD-11 | **N** | bàn phím thật (native) |
| KBD-12 | **P** | Khung thu 411×500: ô 228–272, nút Điểm danh 391–435 — cả hai trong khung |

## 6.N Mạng yếu, mất mạng

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| NET-01 | **P** | 400 kbit/s + RTT 2 s (CDP): khung 5 360 ms, không màn trắng, không cuộn ngang. DEF-MOB-04 RÚT (P-11-143: ngưỡng TC không neo AC) |
| NET-02 | **P** | Sửa ae069d85: offline khi gửi mã → «Chưa có mạng.» + Thử lại, mã giữ nguyên; retry xong checked_in=1 (DEF-MOB-02) |
| NET-03 | **P** | Có mạng lại: bấm Điểm danh → «Đã điểm danh», checked_in=1, không tải lại trang |
| NET-04 | **N** | mạng chập chờn (native) |
| NET-05 | **N** | đổi Wi-Fi ↔ 4G (native) |
| NET-06 | **N** | WebView chặn host (native) |
| NET-07 | **P** | Sửa ae069d85: POST chậm 20 s → sau 16 s có «Chưa có mạng.», trạng thái về entry (DEF-MOB-03) |
| NET-08 | **N** | tiết kiệm dữ liệu (native) |
| NET-09 | **N** | PWA offline (native) |
| NET-10 | **N** | chế độ máy bay (native) |
| NET-11 | **N** | mất mạng khi đang mở màn chiếu (native) |
| NET-12 | **N** | mất mạng khi camera mở (native) |

## 6.R Vùng chạm ≥ 44 px

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| TGT-01 | **P** | 360 và 430: mọi button/a/input trong main, topbar, thanh dưới ≥ 44×44 |
| TGT-02 | **P** | Sửa 8d2ec456: 4 phép đo (checkin/attendance × 360/375) min **8 px**, 0 cặp <8 — trước sửa: attendance 7 cặp 4 px (DEF-MOB-06) |
| TGT-03 | **P** | dock đáy 683 = mép trên thanh dưới 683 (không đè); dock 61 px, thanh dưới 57 px |
| TGT-04 | **N** | uiautomator dump (native) |
| TGT-05 | **P** | /attendance 375 (GV): mọi nút/hàng ≥ 44×44 |
| TGT-06 | **N** | ngang thấp + pointer:coarse dùng thanh đáy (M2) nên không đo thanh bên |

## 6.X Staff trên điện thoại — /attendance

| Mã | Kết quả | Lý do / bằng chứng |
| --- | --- | --- |
| ATT-01 | **P** | 375: mở được buổi, cửa sổ, có «Mở màn chiếu», không cuộn ngang |
| ATT-02 | **P** | 44×44 ĐẠT ở 375; vế «hai điều khiển kề ≥ 8 px» nay đạt (min 8 px ở 360 và 375 sau 8d2ec456) |
| ATT-03 | **P** | TA 360: lưới cuộn trong khung riêng, trang không cuộn ngang |
| ATT-04 | **N** | safe-area (native) |
| ATT-05 | **N** | cỡ chữ 2,0× (native) |
| ATT-06 | **N** | máy tối/native |
| ATT-07 | **P** | axe /attendance 360: 0 lỗi critical/serious; TalkBack = native |
| ATT-08 | **P** | SV mở /attendance: màn chặn «Bạn không có quyền xem màn này», không dữ liệu lớp |
| ATT-09 | **P** | 844×1000: thanh bên hiện, không cuộn ngang |
| ATT-10 | **P** | /attendance/display?course&session ở 393: QR/mã/vòng/Thoát hiện, không crash |
| ATT-11 | **N** | geofence + SQL (pha 2) |

### Nghiệm lại bản sửa (VPS ae069d85)

| TC | DEF | Kết quả |
| --- | --- | --- |
| FLW-13 | DEF-MOB-01 (578e809e) | **P** — «Lớp này đã được lưu trữ.», không POST |
| NET-02 + MSG-07 | DEF-MOB-02 (ae069d85) | **P** — «Chưa có mạng.» + `Thử lại`, mã giữ nguyên, retry → `checked_in=1` |
| NET-07 | DEF-MOB-03 (ae069d85) | **P** — sau 16 s có «Chưa có mạng.», trạng thái `entry` |
| TGT-02 + ATT-02 | DEF-MOB-06 (8d2ec456) | **P** — `.mHead` ↔ `.mPlus` min 8 px ở 360 và 375, cả /checkin và /attendance |

### Chạy bù 6 mã trước để N vì thiếu dữ kiện (tài khoản `sv.qc-b`)

| Mã | Trước | Nay | Ghi chú |
| --- | --- | --- | --- |
| SES-07 | N | **P** | hai SV cùng context → cùng `device_id`, cả hai `DEVICE_SHARED` (SQL) |
| SES-14 | N | **P** | khoá thiết bị (IndexedDB) sống qua đăng xuất; lượt thứ hai cùng `device_id` |
| ERR-09 | N | **P** | 429 ở lần sai thứ 6 + `Retry-After: 60`, đếm ngược, ô mã bị gỡ, hết giờ về ô nhập |
| MSG-06 | N | **P** | «Thử lại sau {n} giây.» đếm ngược |
| TGT-02 | N | **F** | /checkin đạt; /attendance kề 4 px (`.mHead` ↔ `.mPlus`) → DEF-MOB-06 |
| MSG-15 | N | **P** | 6 trạng thái: 0 chuỗi cấm, 0 mã HTTP |

## Tổng (phần qc)

| Nhóm | P | F | N | Mã |
| --- | --- | --- | --- | --- |
| 6.B Phiên, vai, tài khoản | 8 | 0 | 6 | 14 |
| 6.C Luồng điểm danh chính | 11 | 0 | 7 | 18 |
| 6.F Nhánh lỗi (ERR) | 15 | 0 | 5 | 20 |
| 6.S Câu chữ (MSG) | 12 | 0 | 4 | 16 |
| 6.G Độ rộng, xoay ngang, khung app | 16 | 0 | 6 | 22 |
| 6.I Cỡ chữ, phóng | 1 | 0 | 7 | 8 |
| 6.K Bàn phím che ô nhập (phần web) | 6 | 0 | 6 | 12 |
| 6.N Mạng yếu, mất mạng | 4 | 0 | 8 | 12 |
| 6.R Vùng chạm ≥ 44 px | 6 | 0 | 0 | 6 |
| 6.X Staff trên điện thoại — /attendance | 7 | 0 | 4 | 11 |
| **Cộng** | **84** | **0** | **55** | **139** |

## DEF mới

| DEF | Mức | Nội dung | Bằng chứng |
| --- | --- | --- | --- |
| DEF-MOB-01 | ~~Thấp~~ **ĐÃ SỬA** (578e809e, nghiệm lại ae069d85: «Lớp này đã được lưu trữ.») | Lớp đã lưu trữ hiện «Lớp này không điểm danh bằng mã.» thay vì câu về lớp lưu trữ (FLW-13) | Máy chủ: lớp ARCHIVED trả `attendance.self_checkin=false`; giao diện chỉ có nhánh "tắt điểm danh" |
| DEF-MOB-02 | ~~Trung bình~~ **ĐÃ SỬA** (ae069d85: có `Thử lại`, mã giữ nguyên, retry xong `checked_in=1`) | Mất mạng khi gửi **mã ngắn**: có «Chưa có mạng.» nhưng **không có nút `Thử lại`** (đường token thì có) (NET-02, MSG-07) | `CheckinScreen.describe` → `viaCode` chỉ đặt lỗi dưới ô, không gắn `retry` |
| DEF-MOB-03 | ~~Trung bình~~ **ĐÃ SỬA** (ae069d85: sau 16 s có «Chưa có mạng.», về `entry`) | Máy chủ trả lời POST chậm 20 s: sau 16 s vẫn «đang gửi», không có «Chưa có mạng.» / hết giờ (NET-07) | Gọi `apiClient.post` không có hạn giờ; chỉ đổi trạng thái khi promise kết thúc |
| DEF-MOB-06 | ~~Thấp~~ **ĐÃ SỬA** (8d2ec456, `gap: var(--ep-space-2)`; nghiệm lại 4 phép đo min 8 px ở 360/375) | `/attendance` dọc: nút mở hàng `.mHead` và nút ghi phát biểu `.mPlus` cách **4 px** (< 8 px) ở 360 px — 7 cặp (TGT-02, ATT-02) | `AttendanceGrid.module.css` `.mTop { gap: var(--ep-space-1) }` = 4 px |
| DEF-MOB-04 | **RÚT** (P-11-143) | Mạng 400 kbit/s + RTT 2 s: khung 5 360 ms — **TC đặt ngưỡng 3 s không neo AC nên TC sai**, không phải lỗi sản phẩm; số đo giữ ở dòng NET-01 làm tham chiếu | — |

## Ghi chú

- Quan sát tĩnh §5 của `tc-mobile-app.md` không chấm ở pha 1 trừ ca W tương ứng.
- Mọi ca dựng trạng thái bằng `page.route` (gửi/đã gửi/lỗi/429) chỉ đo bố cục, không tạo lượt trong DB.
- Số đếm đúng của phần qc là **139 mã = 84 P · 0 F · 55 N** sau khi chạy bù + nghiệm lại DEF-MOB-06 6 mã thiếu dữ kiện (trước đó 78 P/0 F/61 N; lượt đầu báo 77 P do cột 6.N ghi 1 P trong khi thực tế có 4).
- Gộp §9 của qc3 (dedup theo mã, commit 582e0240): **244 mã = 177 P · 2 F · 65 N**; FLW-02 lấy P của qc3 (pha 3 iOS), phần tôi là N.
- Cách dựng cho 6 mã chạy bù: `e2e/qc-mob-pha1-gap.spec.ts` (SES/ERR/TGT/MSG-15) + `e2e/qc-mob-pha1-err09api.spec.ts` (429/Retry-After qua API); SQL đối chiếu trên Postgres VPS (chỉ đọc).
