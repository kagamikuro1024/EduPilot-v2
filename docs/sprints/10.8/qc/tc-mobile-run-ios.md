# Kế hoạch chạy pha 3 — iOS (chuẩn bị sẵn, chưa bật máy)

Viết 26/11/2026 (qc3) theo lệnh PM. **Chưa boot simulator nào** (Android emulator `ep-api36` đang chạy cho qc2 — một máy một người). Pha 3 do qc3 chạy khi qc2 tắt emulator.

Nguồn TC: `tc-mobile-app.md` (bộ 345 TC). Đích: mọi TC mà cột MT có `I`, `P` hoặc `IR`.

## 1. Phạm vi

**117 TC** đụng iOS, chia theo nhóm:

| Nhóm | Số TC | Tiêu biểu |
|---|---|---|
| 6.P PWA (Safari iPhone) | 21 | PWA-01…23 (trừ 16/22 chờ dữ liệu) |
| 6.E lỗi/trạng thái | 16 | ERR-01…09,11,12,13,15,17,18,19 |
| 7 thông báo | 10 | MSG-01…07,12,13,15 |
| 6.N lối vào | 10 | ENT-01,02,03,05,12,13,14,15,16,22 |
| 6.C luồng | 6 | FLW-02,04,05,10,17,18 |
| 6.G giao diện | 6 | UIW-02,03,05,11,12,17 |
| 6.B điểm danh | 6 | ATT-01,03,04,06,10,11 |
| 6.H vùng an toàn | 5 | SAF-05,06,07,08,10 |
| 6.K bàn phím | 5 | KBD-02,03,04,05,09 |
| 6.L Back/vuốt | 5 | NAV-02,03,07,08,10 |
| 6.T bảo mật | 5 | SEC-01,02,12,16,17 |
| 6.E vị trí / mạng / vòng đời / chữ / tối / trợ năng / hiệu năng / DAT / phiên | 22 | GEO-07,11,12; NET-09,10,12; LIF-09,10,12; TXT-04,05; THM-04,07; PRM-07,08; A11-10,11,12; PRF-08; SES-12,14; DAT-01 |
| **Cộng** | **117** | |

Riêng `TC-AS09-34 AC17` (PWA-17…23) nằm trong nhóm 6.P.

## 2. Thiết bị và runtime

Máy này có: Xcode 27.2, runtime **iOS 27.2** (2 biến thể, biến thể thứ hai chỉ cho iPhone 19,4). Không có runtime cũ hơn.

| Vai trong TC | Thiết bị | UDID | Cỡ CSS | Ghi chú |
|---|---|---|---|---|
| iPhone 15 (TC SAF-05, UIW-03…, PWA mặc định) | `qc3-iphone15` (tạo mới từ devicetype iPhone 15) | — (tạo ở bước chạy) | 390×844 | Có Dynamic Island ⇒ SAF-05 dùng máy này |
| Máy nhỏ (UIW-02, TXT, KBD) | `qc3-se3` (iPhone SE 3rd gen) | — | 375×667 | Không tai thỏ; kiểm 375 px như TC yêu cầu |
| Máy lớn (UIW-05) | `qc3-iphone15pm` (iPhone 15 Pro Max) | — | 430×932 | |
| Máy của qc2 (dùng lại cho ca cần máy đã có dữ liệu) | `qc2-iphone17` | `F60A8A75-1402-428D-9C8A-E1703583985D` | 402×874 | Chỉ dùng khi qc2 bàn giao |
| Máy nhỏ đã dựng sẵn | `iPhone 17e` | `3EA903D3-5650-45E5-A415-CBD825086464` | ~390×844 | Dự phòng |

**Lệch so với TC (ghi trong kết quả):** runtime chỉ có iOS 27.2 nên không có iPhone 13 mini (375×812) và không có máy iOS cũ hơn; thay bằng SE 3 (375×667) cho ca 375 px. `IR` (iPhone thật) và ca ITP 7 ngày (`PWA-13`, `PWA-23`) sẽ là **N** nếu PM không cấp máy thật.

Tạo máy (một lần, không tốn CPU):
```bash
python3 docs/regression/evidence-ios/ios.py create qc3-iphone15 "iPhone 15"
python3 docs/regression/evidence-ios/ios.py create qc3-se3 "iPhone SE (3rd generation)"
python3 docs/regression/evidence-ios/ios.py create qc3-iphone15pm "iPhone 15 Pro Max"
```

## 3. Công cụ và cách chạy

Không có Appium/idb/WDA trên máy; dùng đúng bộ có sẵn:

| Việc | Cách | Tự động? |
|---|---|---|
| Mở URL như camera hệ thống | `xcrun simctl openurl booted "$U"` | có |
| Ảnh chụp | `xcrun simctl io booted screenshot` | có |
| Đọc chữ trên ảnh (kiểm «Đã điểm danh», câu lỗi) | Vision (macOS) qua `ocr.swift` → `./ocr <ảnh>`; tesseract trên máy **không có gói `vie`** nên chỉ là dự phòng `eng` | có |
| Sáng/tối | `simctl ui booted appearance dark` | có |
| Cỡ chữ hệ thống | `simctl ui booted content_size accessibility-extra-extra-extra-large` | có |
| Quyền vị trí | `simctl privacy booted grant\|revoke\|reset location` | có |
| Vị trí giả | `simctl location booted set <lat>,<lng>` | có |
| Thanh trạng thái cố định (THM-07) | `simctl status_bar booted override --time 9:41 --batteryLevel 100` | có |
| Xoay ngang | Simulator ⌘→ (hoặc `osascript` phím tắt) | có |
| Chạm / vuốt / gõ | `osascript` + System Events (Accessibility đã bật — đã thử `click at`) | có, toạ độ theo **tỉ lệ khung cửa sổ** nên không phụ thuộc scale |
| DOM / IndexedDB (PWA-17, SEC-12, LIF) | Safari → Develop → Simulator → trang → Console/Storage | **tay** (Apple không có WebDriver cho Safari trong Simulator nếu không có Appium/WDA) |
| VoiceOver (A11-10) | Simulator: Settings → Accessibility → VoiceOver, hoặc ⌥⌘F5 | **tay** (nghe đọc) |
| Mạng chậm/mất mạng (NET-09/10, FLW-17) | Network Link Conditioner trên Mac, hoặc tắt Wi-Fi của Mac khi Simulator dùng mạng host | **tay** |
| Thêm vào Màn hình chính (PWA-02) | Chia sẻ → «Thêm vào Màn hình chính»: **tap theo toạ độ** một lần/thiết bị, xong PWA mở bằng tap biểu tượng | bán tự động |

Script: `docs/regression/evidence-ios/ios.py` (lệnh `boot/open/shot/ocr/ui/size/loc/perm/tap/type/batch1`). Ảnh + JSON kết quả vào `docs/regression/evidence-ios/shots/` và `results.tsv`.

Mật khẩu seed đọc từ `~/.config/edupilot-team/secrets/seed-password.txt` trong lệnh chạy; **không echo**, không ghi vào evidence (kết quả phải qua `grep` = 0 lần xuất hiện).

## 4. Thứ tự chạy (8 lô)

Chạy từ lô ít rủi ro nhất, tăng dần; mỗi lô commit + báo PM.

| Lô | Việc | TC | Cần gì |
|---|---|---|---|
| A | Cấu hình/khung: manifest, theme-color, icon, standalone, status bar, sáng/tối, vùng an toàn dọc | PWA-01,02,03; SAF-05,06; THM-04,07; 6.P phần metadata | Boot `qc3-iphone15` |
| B | `/checkin` Safari: mã đúng/sai, lỗi trạng thái, offline, quay lại | FLW-02,04; ERR-01…09,11,12,13,15,17,18,19; NET-09,12 | Cửa sổ mở + mã trên VPS |
| C | Ngang + máy nhỏ + cỡ chữ (TXT-04/05) + bàn phím (KBD) | UIW-02,03,05,11,12,17; SAF-07,08,10; TXT-04,05; KBD-02,03,04,05,09 | Boot thêm `qc3-se3`, `qc3-iphone15pm` |
| D | Vị trí (GEO-07,11,12; PRM-07,08) trên lớp QCMOB2 | — | `simctl privacy`/`location`; lớp geofence |
| E | PWA thật: cài, mở standalone, deep link, hai vòng, device_id, IndexedDB | PWA-04…12,14,15,17…21; NAV-08; LIF-10; NET-10; FLW-10 | Tap cài PWA 1 lần; Web Inspector cho PWA-17 |
| F | Bảo mật/phiên/ENT: cookie, `device_id`, lối vào | SEC-01,02,12,16,17; SES-12,14; ENT-01,02,03,05,12,13,14,15,16,22; ATT-01,03,04,06,10,11; MSG ×10; DAT-01 | Vị trí + ảnh |
| G | Back/vuốt cạnh trái, đổi ngôn ngữ, hiệu năng | NAV-02,03,07,10; LIF-09,12; PRF-08 | `osascript` phím tắt, `simctl` |
| H | VoiceOver + iPhone thật | A11-10,11,12; PWA-13,23; SAF-05 phần IR | Bật VoiceOver tay; IR = N nếu không có máy |

Điều kiện chung trước khi chạy: VPS còn lớp/mã dùng được (`QCVPSA–D` từ pha 1), mật khẩu seed đọc được, cửa sổ điểm danh mở bằng teacher API. Lô D và F cần tạo lại lớp geofence (bản pha 1 đã dùng) và cần mở vòng 2 để kiểm `device_id` lặp lại.

## 5. Dữ liệu cần có (chuẩn bị trước, không phụ thuộc máy)

1. **Lớp không geofence** (như `QCVPSA`) + 2 SV: dùng cho A/B/C.
2. **Lớp geofence** (tâm 21,0285/105,8542 r200) + `campus_cidrs`: D.
3. **Lớp `require=APP`**: kiểm ERR-08 (PWA không phải app?) — xác nhận lại định nghĩa trong SRS trước khi chạy.
4. **Lượt đã purge** (dùng lại `01a213ba…` đã purge trên `ep-dat`? **không** — VPS mới là nơi iOS thao tác) ⇒ tạo 1 cửa sổ mới, purge sau khi kiểm để lặp ca câu lỗi.
5. **`sv.gioi` + `sv.nguyco`** trên VPS: đã có; kiểm mật khẩu còn đúng.

## 6. Rủi ro đã biết

| Rủi ro | Ảnh hưởng | Xử lý |
|---|---|---|
| Không có WebDriver cho Safari trong Simulator | Không assert DOM tự động được | Dùng ảnh + OCR cho mọi ca; DOM chỉ ở ca ghi rõ «tay qua Web Inspector» (PWA-17, SEC-12) |
| Simulator không có camera | Quét QR (PWA-09/10 phần camera, FLW-03…) chỉ kiểm được bằng ô mã | Ghi rõ trong kết quả; ca QR thật để `IR` |
| Runtime chỉ iOS 27.2 | Thiếu máy cũ (13 mini, iOS 16…) | Dùng SE 3 + 15 + 15 Pro Max; ghi lệch |
| Tap bằng System Events | Toạ độ lệch nếu cửa sổ đổi vị trí/scale | Toạ độ theo tỉ lệ; chụp ảnh ngay sau mỗi tap để chứng minh |
| PWA «Thêm vào Màn hình chính» | Mỗi thiết bị phải làm tay một lần | Làm ngay đầu lô E; chụp màn hình hộp tên |
| Android emulator đang chạy | Boot song song làm chậm cả hai | **Không boot tới khi qc2 tắt** |

## 7. Trạng thái chuẩn bị

- [x] Liệt kê 117 TC iOS + nhóm theo lô.
- [x] Kiểm runtime/thiết bị, danh sách devicetype tạo được.
- [x] Viết `evidence-ios/ios.py` (boot/openurl/screenshot/OCR/ui/size/privacy/location/tap/type).
- [x] Xác nhận `osascript` + System Events chạy được (đã gọi thử; cần Accessibility).
- [x] OCR tiếng Việt: dựng `evidence-ios/ocr.swift` bằng Vision, đã đọc đúng «Đã điểm danh 00:22» từ ảnh `DAT-01-after.png`.
- [x] Xác nhận repo `mobile/` chỉ có `android/` — **không có app iOS native**, pha 3 kiểm **PWA trên Safari**, đúng như cột MT của bộ TC.
- [ ] Tạo 3 simulator (chờ qc2 tắt emulator để không tranh CPU).
- [ ] Dựng lại dữ liệu lớp geofence trên VPS (bước đầu của lô D).
- [ ] Chạy lô A → H.
