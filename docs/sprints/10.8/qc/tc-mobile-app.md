# QC test case — ỨNG DỤNG ĐIỆN THOẠI (app Android + PWA iPhone) — US-AS-09…15 + `android-checklist.md`

Viết 25/11/2026 (qc3) theo lệnh PM / chủ dự án: testcase ĐẦY ĐỦ cho app điện thoại, nhất là UI/UX; **CHẠY sau cùng**. **Chưa chạy gì** (không có kết quả P/F trong tệp này). Nguồn: `docs/specs/FEAT-attendance-selfcheck/{US,SRS}.md` (US-AS-09…15; US-AS-15 do BA `3018e3ce`, P-11-130 CHẤP NHẬN; AC16, AC17 và nav SV 10 mục do BA `c57b94b9`, P-11-137 CHẤP NHẬN), `docs/design/DESIGN.md` §14.8.2, §16, §23, `docs/sprints/10.8/qc/android-checklist.md`, `docs/sprints/10.8/proposals.md` (P-108-1…12), mã `mobile/` và `frontend/src/features/checkin/` @`2c0dbe83` (nhóm Y + PWA-17…23 theo `697854ef`) (đọc để biết chữ, nhãn, kích thước — ca vẫn viết hộp đen theo AC). Tệp này BỔ SUNG, không thay `tc-US-AS-09.md`, `tc-US-AS-10-12-13.md`, `tc-US-AS-11-14.md` (các tệp đó phủ API / web 375 px; ở đây phủ **thiết bị thật / giả lập** và mọi khía cạnh di động).

**Tổng: 345 TC** trong 25 nhóm; 25 ca đánh dấu «chỉ máy thật / điều kiện đặc biệt» (cột cuối).

## 1. Quy ước
| Mục | Quy tắc |
| --- | --- |
| Kết quả | **P** (đạt, có bằng chứng), **F** (trượt, ghi DEF), **N** (KHÔNG KIỂM ĐƯỢC: nêu lý do, không tính F). Không ghi P khi chưa chạy. Ca ghi «dự kiến F» là quan sát tĩnh từ mã trước khi chạy, **không** phải kết quả |
| Mã TC | `TC-MOB-<nhóm>-<số>`; cột «AC» ghi AC của US hoặc bước `AC-n` của `android-checklist.md` |
| MT (môi trường) | **A** = Android emulator AVD `ep-smoke` API 35; **AR** = Android máy thật; **I** = iOS Simulator 27.2, Safari; **P** = iOS Simulator, PWA đã «Thêm vào Màn hình chính»; **IR** = iPhone thật; **W** = Playwright (thiết bị di động giả, `isMobile`, `hasTouch`); **API** = gọi `curl`/python thẳng vào gateway, không cần thiết bị |
| Kiểm | **TAY** (người thao tác trên thiết bị), **E2E** (Playwright), **ADB** (lệnh `adb`), **SIMCTL** (lệnh `xcrun simctl`), **AXE** (`@axe-core/playwright`), **SQL** (psql chỉ đọc), **DIFF** (đọc mã / `go test`), ghép bằng `+` |
| Cột cuối | Trống = chạy được trên emulator / simulator / Playwright. Có chữ = cần thứ thiếu (máy thật, TalkBack, hai thiết bị, quyền chặn egress…) → nếu không có thì ghi **N** kèm lý do |
| Bằng chứng | Ảnh chụp: `shots-mobile/<mã TC>-<thiết bị>.png`; video ngắn `.mp4`/`.mov` cho ca có thời gian; trích log đã bỏ bí mật. KHÔNG dán token, khoá, mật khẩu, toạ độ thật, IP vào báo cáo |
| Thời gian chờ | Mã hết hạn thật chờ ≥ 12 s với `rotate_seconds=5`; cửa sổ hết giờ `minutes=1`; không đổi giờ máy chủ. Đổi giờ máy điện thoại chỉ ở ERR-16 rồi trả về tự động |

## 2. Môi trường và thiết bị
| Mã | Thiết bị | Ghi chú |
| --- | --- | --- |
| A | Android Virtual Device `ep-smoke`, API 35 (ghi loại image: Google APIs hoặc Play Store), camera sau `virtualscene` (poster QR) hoặc `webcam0`; ghi bản WebView, bản Chrome | APK release dựng từ `mobile/README.md`; `APP_SERVER_URL` = HTTPS thật của VPS (dev đang dựng). Emulator **không** đạt mức 3 và **không** là lỗi |
| AR | Máy Android thật (ghi hãng, model, bản Android, bản WebView, bootloader khoá/mở, StrongBox/TEE) | Chủ dự án KHÔNG có máy thật (P-108-12) → ca chỉ máy thật ghi **N** trừ khi có máy |
| I | iOS Simulator 27.2 (qc2 đã dựng): iPhone SE (3rd gen) 375×667, iPhone 13 mini 375×812, iPhone 15 390×844, iPhone 15 Pro Max 430×932 | Safari. Simulator KHÔNG có camera: quét QR không kiểm được ở iOS (và iOS web không quét, chỉ camera hệ thống mở liên kết) |
| P | iOS Simulator, Safari → Chia sẻ → «Thêm vào Màn hình chính» | PWA standalone; kho lưu trữ tách khỏi Safari |
| IR | iPhone thật nếu có | VoiceOver, ITP 7 ngày, biểu tượng thật |
| W | Playwright `Pixel 7` (412×915), `iPhone 15` (393×852) và các viewport 360/375/390/411/430 × 640–932 và ngang 667×375, 844×390, 915×412 | chạy trên bản build (`next start`) hoặc stack thử |

| Độ rộng (CSS px) | Android (R-SIZE) | iOS | Playwright |
| --- | --- | --- | --- |
| 360 × 640–800 | `wm size 1080x2400`, `wm density 480` | – (không có) | 360×640, 360×800 |
| 375 × 667–812 | `wm size 1125x2436`, `wm density 480` | iPhone SE 375×667, iPhone 13 mini 375×812 | 375×667, 375×812 |
| 390 × 844 | `wm size 1170x2532`, `wm density 480` | iPhone 15 | 390×844 |
| 411 × 891 | `wm size 1080x2340`, `wm density 420` | – | 411×891 |
| 430 × 932 | `wm size 1290x2796`, `wm density 480` | iPhone 15 Pro Max | 430×932 |
| Ngang | xoay `R-ROT` ở 667×375, 844×390, 915×412 | Simulator ⌘← / ⌘→ | cùng ba kích thước |

## 3. Công thức lệnh (tham chiếu trong các bước)
| Tên | Lệnh / thao tác |
| --- | --- |
| R-ENV | `emulator -avd ep-smoke -no-snapshot-load -camera-back virtualscene`; cài APK: `cd mobile && npm ci && APP_SERVER_URL=https://{VPS} npm run install:device` |
| R-CAM | poster QR cho camera ảo: thêm ảnh PNG vào `$ANDROID_HOME/emulator/resources/Toren1BD.posters` (mục `poster wall` → `default <ảnh>.png`); QR động của màn chiếu: `-camera-back webcam0` hướng vào màn laptop; ảnh poster dựng sẵn: Q-OK (đúng, đang hạn), Q-EXPIRED, Q-ALIEN, Q-OTHER-ORIGIN, Q-PATH, Q-NOUUID, Q-NOHASH, Q-OTHERCLASS, Q-CLOSED |
| R-SIZE-w | `adb shell wm size <W×H px>`; `adb shell wm density <dpi>` theo bảng ở mục 2; trả: `wm size reset`; `wm density reset` |
| R-DENSITY | `adb shell wm density 320` (hiển thị lớn) và `wm density 560` (nhỏ) |
| R-FONT-x | `adb shell settings put system font_scale 1.3` rồi `2.0`; trả `1.0` |
| R-DARK-on/off | `adb shell cmd uimode night yes` / `no` |
| R-ROT-land | `adb shell settings put system accelerometer_rotation 0; adb shell settings put system user_rotation 1` (ngang); `0` (dọc) |
| R-CUT-tall/corner/double/hole | `adb shell cmd overlay enable com.android.internal.display.cutout.emulation.<tall|corner|double|hole|narrow|wide>`; tắt: `overlay disable …` |
| R-NAV-gesture / R-NAV-3btn | `adb shell cmd overlay enable com.android.internal.systemui.navbar.gestural` / `…navbar.threebutton` |
| R-PERM | `adb shell pm revoke vn.edupilot.app android.permission.CAMERA`; `pm grant …`; `pm reset-permissions`; mở Cài đặt ứng dụng: `adb shell am start -a android.settings.APPLICATION_DETAILS_SETTINGS -d package:vn.edupilot.app` |
| R-NET-off | `adb shell cmd connectivity airplane-mode enable` / `disable` |
| R-NET-slow | `adb emu network speed 400:400` và `adb emu network delay 1000` (≈ RTT 2 s); trả: `network speed full`, `network delay none` |
| R-NET-flaky | vòng lặp `adb shell svc wifi disable; sleep 2; adb shell svc wifi enable; sleep 2` ×5 (kèm `svc data`) |
| R-NET-block-host | chặn máy chủ app bằng proxy phía máy chủ chạy (mitmproxy rule) hoặc nhờ dev tạm đóng cổng 443 của VPS thử; trả lại sau khi xong |
| R-KILL | `am force-stop vn.edupilot.app`; nền: `input keyevent KEYCODE_HOME` rồi `am kill vn.edupilot.app`; ép bộ nhớ thấp: `am send-trim-memory vn.edupilot.app COMPLETE`; Doze: `dumpsys deviceidle force-idle` / `unforce` |
| R-LINK | `adb shell am start -a android.intent.action.VIEW -d "$U" com.android.chrome` (như camera hệ thống mở liên kết) |
| R-GEO | `adb emu geo fix <kinh độ> <vĩ độ>`; vị trí giả: `adb shell appops set <gói app GPS giả> android:mock_location allow` |
| R-SHOT | `adb exec-out screencap -p > x.png`; quay: `adb shell screenrecord --time-limit 30 /sdcard/x.mp4`; chụp liên tiếp: vòng lặp `screencap` ≥ 5 ảnh/s |
| R-DUMP | `adb shell uiautomator dump /sdcard/ui.xml` rồi `adb pull`; đọc `bounds`, `content-desc`, `text` |
| R-LOGCAT | `adb logcat -d -v time`; lọc bí mật: `grep -iE 'access_token|refresh|ep_rt|signature|public_key|Bearer|token='` |
| R-CALL | `adb emu gsm call 0901234567`; `adb emu gsm cancel 0901234567` |
| S-SIM | `xcrun simctl list devices available`; `xcrun simctl boot "iPhone 15"` (và `iPhone SE (3rd generation)`, `iPhone 13 mini`, `iPhone 15 Pro Max`) |
| S-URL | `xcrun simctl openurl booted "$U"` (như camera hệ thống mở liên kết; luôn mở Safari) |
| S-DARK | `xcrun simctl ui booted appearance dark` / `light` |
| S-CONTENT-SIZE | `xcrun simctl ui booted content_size accessibility-extra-extra-extra-large`; trả `large` |
| S-PRIVACY | `xcrun simctl privacy booted grant|revoke|reset location com.apple.mobilesafari` |
| S-LOC | `xcrun simctl location booted set <vĩ độ>,<kinh độ>` |
| S-SHOT | `xcrun simctl io booted screenshot x.png`; quay: `xcrun simctl io booted recordVideo x.mov`; thanh trạng thái sạch: `xcrun simctl status_bar booted override --time 9:41 --batteryLevel 100` |
| S-KILL | `xcrun simctl terminate booted com.apple.mobilesafari` |
| S-NET | Network Link Conditioner trên máy Mac (hồ sơ 3G/100 % mất gói), hoặc Safari → Develop → Network → Throttling |
| S-CA | chỉ khi VPS dùng CA nội bộ: `xcrun simctl keychain booted add-root-cert <gốc>.pem` (HTTPS thật của VPS thì không cần) |
| E2E | `pnpm -C frontend exec playwright test e2e/attendance-checkin.spec.ts --project=mobile` (cần project `mobile` hoặc thêm dự án `isMobile`); script kiểm kích thước riêng: `e2e/mobile-ui.spec.ts` (QC viết khi chạy) |

## 4. Dữ liệu
| Tên | Nội dung |
| --- | --- |
| VPS | `https://{VPS}` = HTTPS thật do dev dựng (ghi host khi chạy; đổi host ⇒ dựng lại và cài lại APK). Mật khẩu seed lấy từ biến môi trường của đợt chạy, không ghi vào báo cáo |
| Tài khoản | SV: `sv.gioi` (SV1), `sv.kha` (SV2), `sv.nguyco` (ngoài lớp), `sv.moi` (chưa vào lớp nào); Staff: `teacher@`, `ta@`; `admin@` (đều `@edupilot.local`) |
| Lớp | **QCMOB1**: `self_checkin` bật, không geofence; **QCMOB2**: `geofence {lat,lng,radius_m}` bật + `campus_cidrs`; **QCMOB3**: `require=APP`; **QCMOB4**: lưu trữ; **QCMOB5**: `self_checkin` tắt. SV1, SV2 thuộc QCMOB1–3; `sv.nguyco` không thuộc lớp nào trong số đó. Không đụng 761987/761988 (trừ DAT-01) |
| Cửa sổ | W: cửa sổ đang mở, `rotate_seconds=10` (đủ cho quét tay) hoặc `5` (cho ca hết hạn); W60: `rotate_seconds=60` (poster QR sống ≤ 120 s); W-closed: đã đóng; W-old: token cũ > 2 bước; vòng 2 mở sau vòng 1 |
| QR / liên kết | `U = https://{VPS}/checkin?c={course_id}#{token}`; mã ngắn 6 ký tự hiện dưới QR của màn chiếu (`/attendance/display` trên laptop ≥ 1280×720) |
| Phụ trợ | app GPS giả (cho GEO-10), ảnh QR dựng sẵn (R-CAM), proxy chậm/chặn (R-NET-*), `psql` chỉ đọc vào `ep-qc` / stack thử |

## 5. Quan sát tĩnh TRƯỚC khi chạy (đọc mã / tài nguyên ngày 25/11; chưa phải kết quả)
Những điểm sau suy ra từ tệp trong repo @`2c0dbe83` (nhóm Y + PWA-17…23 theo `697854ef`); ca tương ứng sẽ xác nhận hay bác bỏ khi chạy. Nếu xác nhận → mở DEF.
| # | Quan sát | Ca xác nhận |
| --- | --- | --- |
| 1 | Biểu tượng launcher (`mipmap-*/ic_launcher*.png`) và màn splash (`drawable*/splash.png`) là bộ mặc định của Capacitor (chữ X xanh), không phải logo EduPilot đỏ – trắng (đã mở ảnh đọc) | INS-05, INS-06, THM-05 |
| 2 | `AndroidManifest.xml` chỉ khai `ACCESS_FINE_LOCATION`, không khai `ACCESS_COARSE_LOCATION`; Android 12+ xin cả hai cùng lúc nên hộp xin vị trí có thể bị từ chối ngay trong app | GEO-05, GEO-08, PRM-10 |
| 3 | `layout.tsx` không đặt `viewport-fit=cover`, không `apple-touch-icon`, không `appleWebApp` / `theme-color` metadata (chỉ `manifest.ts` có icon PNG 192/512 và `theme_color`) | SAF-05, SAF-06, PWA-03, PWA-05, THM-07 |
| 4 | Toàn app chỉ giao diện sáng (`color-scheme: light`, D59); `error.html` không đặt nền (chữ `#1a1a1a`) — dễ chữ tối trên nền tối nếu WebView ép tối | THM-01…05 |
| 5 | `targetSdk 36`: từ Android 15 giao diện vẽ tràn dưới thanh trạng thái / thanh điều hướng (edge-to-edge) — mọi `env(safe-area-inset-*)` phải đúng | SAF-01…04 |
| 6 | Manifest không đặt `windowSoftInputMode`; dock `/checkin` là `position: fixed` — bàn phím có thể che nút `Điểm danh` | KBD-01, KBD-02, TXT-08 |
| 7 | Ngưỡng 44 px chỉ bật ở `(max-width: 719px), (pointer: coarse)`; điện thoại ngang ≥ 720 px vẫn có `pointer: coarse` nên đạt, nhưng khung chuyển sang thanh bên | TGT-06, UIW-17 |
| 8 | `nav.ts` @2c0dbe83 ghi `/checkin` «ngoài thanh điều hướng, mở từ QR»; «Hôm nay» dẫn sang Lịch; «Thêm» / «Tìm nhanh» không có «Điểm danh» → PWA / app chưa có lối vào (BA đã vá spec ở P-11-137; ca ENT-* xác nhận khi dev làm xong) | ENT-01…22 |
| 9 | iOS Safari không có `BarcodeDetector` và Simulator không có camera: quét QR chỉ kiểm được trên Android; iOS dùng ô nhập mã + camera hệ thống mở Safari | FLW-04, PWA-08, PWA-09 |

## 6.A Cài đặt và vỏ app Android (APK, quyền, ký, biểu tượng, splash)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-INS-01 | US-AS-11 AC1; AC-0 | A | APK release dựng từ `mobile/README.md` (HTTPS thật của VPS) | `aapt2 dump badging app-release.apk`; `aapt2 dump xmltree --file AndroidManifest.xml`; `apksigner verify --print-certs` | package `vn.edupilot.app`; `uses-permission` chỉ INTERNET, CAMERA, ACCESS_FINE_LOCATION (ghi nếu có thêm); không `autoVerify`; không `usesCleartextTraffic="true"`; `allowBackup=false`; `debuggable` không có; vân tay chứng chỉ = `ATTESTATION_CERT_SHA256` của gateway | ADB |  |
| TC-MOB-INS-02 | US-AS-11 AC1; AC-0 | A | Như INS-01; gỡ app cũ `adb uninstall vn.edupilot.app` | Chạy đúng MỘT lệnh `APP_SERVER_URL=https://{VPS} npm run install:device`; sau đó `adb shell pm list packages vn.edupilot.app`; `adb shell am start -W -n vn.edupilot.app/.MainActivity` | lệnh thoát 0, không cần bước tay nào khác; gói có trong danh sách; `am start -W` in `Status: ok`; màn đăng nhập của web hiện trong ≤ 10 s (cold start đo ở PRF-01); không hộp cảnh báo «ứng dụng không an toàn» hay «bị chặn» | ADB |  |
| TC-MOB-INS-03 | US-AS-11 AC1; P-108-9a | A | `APP_SERVER_URL` sai (http:// hoặc rỗng) | Chạy `npm run install:device` với `APP_SERVER_URL=http://10.0.2.2:3500`, rồi với biến rỗng | cả hai lần dừng ngay, thông báo tiếng Việt nêu «HTTPS» / «APP_SERVER_URL», không dựng APK, không cài; không để lại `www/` dở dang | ADB |  |
| TC-MOB-INS-04 | US-AS-11 AC1 | A | App đã cài, host VPS đúng | Quan sát lần mở đầu tiên: hộp quyền hệ thống; `adb shell dumpsys package vn.edupilot.app \| grep -A12 'requested permissions'` | khi mở app KHÔNG có hộp xin quyền nào (quyền chỉ xin khi dùng camera / vị trí); `granted=false` cho CAMERA và ACCESS_FINE_LOCATION trước khi dùng | ADB |  |
| TC-MOB-INS-05 | US-AS-15 AC11 (thương hiệu); DESIGN §10 | A | App đã cài | `adb shell cmd package resolve-activity --brief vn.edupilot.app`; chụp màn hình launcher (R-SHOT) và đối chiếu `mobile/android/app/src/main/res/mipmap-*/ic_launcher*.png` với `frontend/public/brand/icon-192.png` | tên dưới biểu tượng là «EduPilot»; biểu tượng là logo EduPilot (đỏ – trắng) khớp `frontend/public/brand`, KHÔNG phải biểu tượng Capacitor mặc định (chữ X xanh). Quan sát tĩnh trước khi chạy: `ic_launcher*.png` hiện là biểu tượng Capacitor mặc định (đọc ảnh 25/11) → dự kiến F | TAY+ADB |  |
| TC-MOB-INS-06 | US-AS-15 AC5; AC-18 | A | App đã cài, máy sáng rồi tối (R-DARK) | Đóng hẳn app (`am force-stop`), mở lại, chụp liên tiếp 6 ảnh trong 2 s đầu (R-SHOT loop) ở chế độ sáng, rồi chế độ tối | màn khởi động (splash) đúng thương hiệu EduPilot (không logo Capacitor mặc định), nền trắng ở cả hai chế độ, không chớp đen, không thanh tiêu đề của hệ điều hành; chuyển sang web không có khung trắng giật > 300 ms. Quan sát tĩnh: `drawable/splash.png` hiện là biểu tượng Capacitor mặc định → dự kiến F | TAY+ADB |  |
| TC-MOB-INS-07 | US-AS-11 AC1 | A | App đã cài, đăng nhập `sv.gioi` | Cài đè bản mới cùng chữ ký (`adb install -r`); mở lại | phiên đăng nhập còn (cookie WebView giữ); không mất dữ liệu; `versionCode` tăng hợp lệ; cài đè bản ký khác khoá → `INSTALL_FAILED_UPDATE_INCOMPATIBLE` (ghi nhận, không lỗi) | ADB |  |
| TC-MOB-INS-08 | US-AS-11 AC1; AC-19 | A | App đã cài, đã điểm danh một lần | Gỡ app (`adb uninstall`), cài lại, đăng nhập `sv.gioi`, điểm danh lại | khoá Keystore mới; lượt đầu có `DEVICE_NEW` ở lưới Staff (đúng một lần), lượt sau không; không dữ liệu cũ sót (không đăng nhập sẵn) | TAY+SQL |  |
| TC-MOB-INS-09 | US-AS-11 AC1 | A | Máy API 24–26 nếu có AVD phụ (minSdk 24) | Cài và mở app trên AVD API 24 hoặc 28 | app mở được; nếu WebView cũ không có `BarcodeDetector` thì hiện ô nhập mã ngay (không lỗi); ghi bản WebView. Không có AVD phụ → KHÔNG KIỂM ĐƯỢC | TAY | AVD phụ |
| TC-MOB-INS-10 | US-AS-11 AC1; US-AS-15 AC14 | A | App đã cài, VPS tắt (hoặc `APP_SERVER_URL` trỏ cổng chết) | Mở app lần đầu khi không tới được máy chủ | hiện trang lỗi chuẩn cục bộ (`error.html`): «Không kết nối được máy chủ.» + nút `Thử lại` cao ≥ 44 px; KHÔNG trang lỗi mặc định của WebView (net::ERR_*); bật máy chủ rồi bấm `Thử lại` → vào màn đăng nhập, không phải khởi động lại app | TAY |  |
| TC-MOB-INS-11 | US-AS-11 AC1; AC5 | A | App đã cài | Trong app mở liên kết ngoài (ví dụ dán `https://example.com` vào ô tìm / ô mã hoặc điều hướng từ trang có liên kết ngoài nếu có); `adb logcat` lọc `Capacitor` | `allowNavigation` chỉ host của ứng dụng: trang lạ KHÔNG nạp trong WebView của app (mở trình duyệt ngoài hoặc bị chặn), không thấy giao diện EduPilot bị thay bằng trang ngoài; log không in token | TAY+ADB |  |
| TC-MOB-INS-12 | US-AS-11 AC1 | A | App đã cài, máy API 35 | Kiểm tên và biểu tượng ở Cài đặt → Ứng dụng; kiểm thông tin phiên bản | tên «EduPilot»; `versionName 1.0` hiển thị; không hiện cảnh báo «ứng dụng được xây cho phiên bản Android cũ» | TAY+ADB |  |
| TC-MOB-INS-13 | US-AS-11 AC3b; AC-3c | A + Chrome của emulator | App đã cài; cửa sổ mở, liên kết quét `U` = `https://{VPS}/checkin?c={C1}#{T}` | `adb shell am start -a android.intent.action.VIEW -d "$U"` | liên kết mở Chrome (hoặc trình duyệt mặc định), KHÔNG vào app (không `autoVerify`, không hộp «Mở bằng»); điểm danh thành công ở trình duyệt; DB `channel=WEB`; `mobile/README.md` có dòng «Dùng nút Quét mã điểm danh trong app» | ADB+SQL |  |
| TC-MOB-INS-14 | US-AS-11 AC1; P-108-2 | A | App đã cài | `adb shell cmd package query-activities -a android.intent.action.VIEW -d https://{VPS}/checkin`; `adb shell pm get-app-links vn.edupilot.app` | app KHÔNG nằm trong danh sách xử lý liên kết `https://{VPS}`; trạng thái liên kết `none`/`undefined` (không xác minh) | ADB |  |

## 6.B Phiên đăng nhập, vai và tài khoản trên điện thoại
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-SES-01 | US-AS-11 AC7; AC-1 | A | `sv.gioi` | Mở app → đăng nhập bằng email + mật khẩu → vào «Hôm nay» | đăng nhập thường, ≤ 3 s sau bấm; không hộp «Lưu mật khẩu» của app (WebView không lưu); URL không chứa token (đọc thanh địa chỉ không có; `logcat` không có `access_token`) | TAY+ADB |  |
| TC-MOB-SES-02 | US-AS-11 AC7; AC-1; US-AS-15 AC10 | A | SV đã đăng nhập | `am force-stop vn.edupilot.app`; mở lại; lặp 5 lần liên tiếp, mỗi lần chờ ≥ 5 s trong app | cả 5 lần vào thẳng «Hôm nay» (phiên còn hạn), không màn đăng nhập, không màn trắng; cookie làm mới xoay vòng không làm mất phiên (không thu hồi do dùng lại token) | TAY+ADB |  |
| TC-MOB-SES-03 | US-AS-15 AC10 | A | SV vừa đăng nhập | Đăng nhập xong RỒI `am force-stop` ngay (≤ 2 s, chưa kịp đồng bộ cookie); mở lại | vào «Hôm nay» hoặc màn đăng nhập — KHÔNG màn trắng, không vòng lặp chuyển trang; nếu về đăng nhập thì đăng nhập lại thành công; ghi hành vi thực tế (cookie flush) | TAY+ADB |  |
| TC-MOB-SES-04 | US-AS-11 AC7 | A | SV đã đăng nhập | Menu người dùng → Đăng xuất; kiểm: nút Back; mở lại app | về `/login`; Back không quay về màn đã đăng nhập; mở lại app vẫn ở đăng nhập (cookie bị xoá); `sessionStorage.ep.checkin` rỗng | TAY |  |
| TC-MOB-SES-05 | US-AS-11 AC7; AC-2 | A | `ta@edupilot.local`, `teacher@edupilot.local` | Đăng nhập TA, mở `/checkin` (gõ đường dẫn qua thanh trên hoặc liên kết `adb am start`); sau đó GV | cả hai: màn chặn chuẩn như web (không giao diện điểm danh, không ô nhập mã); `/attendance` dùng bình thường (mở được, mở cửa sổ được với GV/TA); gateway KHÔNG nhận `POST /checkins` | TAY+SQL |  |
| TC-MOB-SES-06 | US-AS-09 AC8; US-AS-15 AC15 | A | `admin@edupilot.local` | Đăng nhập admin, mở `/checkin` | màn chặn chuẩn, không gọi API điểm danh (`docker logs` gateway không có `POST …/checkins`) | TAY+ADB |  |
| TC-MOB-SES-07 | AC-12; US-AS-04 AC4 | A | `sv.gioi` rồi `sv.kha` cùng điện thoại; cửa sổ mở | SV1 điểm danh → đăng xuất → SV2 đăng nhập → điểm danh (cùng cửa sổ) | SQL: hai lượt cùng `device_id`; cả hai có `DEVICE_SHARED`; SV không thấy cờ nào trong app; Staff thấy chip ở lưới `/attendance` | TAY+SQL |  |
| TC-MOB-SES-08 | US-AS-11 AC7 | A | `sv.gioi` | Trong cùng app: đăng nhập, kéo thả thanh thông báo hệ thống (có thông báo) → quay lại; đổi tài khoản Google/hệ thống (nếu có) | không tự đăng xuất, không đổi người dùng; email hiển thị đúng `sv.gioi` | TAY |  |
| TC-MOB-SES-09 | US-AS-15 AC10 | A | SV đăng nhập, access token hết hạn (chờ theo TTL hoặc đổi giờ máy +1 h chỉ để thử, rồi trả lại) | Mở lại app sau khi token truy cập hết hạn nhưng cookie làm mới còn | phiên tự làm mới (≤ 2 s), không màn đăng nhập; không hộp lỗi; đúng MỘT `POST /auth/refresh` (gateway log) | TAY+ADB |  |
| TC-MOB-SES-10 | US-AS-15 AC10 | A | Cookie làm mới hết hạn (xoá cookie bằng đăng xuất ở thiết bị khác để thu hồi phiên) | Mở lại app | về màn đăng nhập, câu tiếng Việt (nếu có); không màn trắng; không vòng lặp /login ↔ / | TAY |  |
| TC-MOB-SES-11 | US-AS-09 AC2 | A | SV chưa đăng nhập; cửa sổ mở | `am start -a VIEW -d $U` (Chrome) khi chưa đăng nhập; đăng nhập trong 20 s | Chrome chuyển `/login?next=/checkin…`; sau đăng nhập tự gửi một lần → «Đã điểm danh»; `sessionStorage.ep.checkin` xoá sau khi dùng; URL không còn `#token`; DB đúng một lượt | TAY+SQL |  |
| TC-MOB-SES-12 | US-AS-09 AC2; US-AS-15 AC7(c) | A, I | Như SES-11 | Sau đăng nhập từ liên kết QR, bấm Back (Android) / vuốt cạnh trái (iOS) | KHÔNG về `/login` rồi vòng lại `/checkin` lần hai; không lượt hai; không lỗi cũ hiện lại | TAY+SQL |  |
| TC-MOB-SES-13 | US-AS-11 AC7 | A | Hai tài khoản trên hai AVD (hoặc xoá dữ liệu app giữa lần) | Xoá dữ liệu app (`pm clear vn.edupilot.app`), mở lại | về đăng nhập; `pm clear` xoá cả khoá Keystore app (khoá mới lần sau → `DEVICE_NEW` một lần) | ADB+SQL |  |
| TC-MOB-SES-14 | US-AS-09 AC3; P-108-4 | A, I | Một trình duyệt, hai SV lần lượt | Điểm danh SV1, đăng xuất, SV2 điểm danh (web) | khoá thiết bị thuộc TRÌNH DUYỆT, không theo tài khoản: đăng xuất không xoá khoá; hai lượt cùng `device_id` → `DEVICE_SHARED` (Staff thấy) | TAY+SQL |  |

## 6.C Luồng điểm danh chính (liên kết, quét, nhập mã, lặp, chọn lớp)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-FLW-01 | US-AS-09 AC1; AC-3 | A | W1 đang mở (rotate 10 s); SV đã đăng nhập trong app; QR hiện trên laptop (hoặc ảnh poster của camera ảo, R-CAM) | Trong app vào `/checkin` bằng mục «Điểm danh» (thanh điều hướng / «Thêm» / «Tìm nhanh»; US-AS-15 AC16), bấm `Quét mã điểm danh`, quét QR trên màn chiếu | xin quyền camera lần đầu; thấy QR → «Đã điểm danh hh:mm» trong ≤ 3 s (đo 5 lần, PRF-02); dấu `Check` đỏ là chỗ đỏ DUY NHẤT; camera tắt (đèn camera tắt, `dumpsys media.camera` không còn client); DB `channel=APP` | TAY+SQL |  |
| TC-MOB-FLW-02 | US-AS-09 AC1 | I | W1 mở, iPhone Simulator Safari đã đăng nhập `sv.gioi` | `xcrun simctl openurl booted "$U"` (như camera hệ thống mở liên kết) | trang tự gửi MỘT `POST`, hiện «Đã điểm danh hh:mm» ≤ 3 s; `#token` biến khỏi thanh địa chỉ; không bấm gì; `channel=WEB` | SIMCTL+SQL |  |
| TC-MOB-FLW-03 | US-AS-09 AC10; US-AS-03 AC3a | A | W1 mở; app đã đăng nhập; không dùng quét | `/checkin` → bấm `Nhập mã` (hoặc từ chối camera) → gõ mã ngắn đang hiện trên màn chiếu theo ba kiểu: `abc123`, `ABC 123`, `abc-123` → `Điểm danh` | cả ba kiểu được nhận (kiểu đầu tiên thành công; hai kiểu sau trả lượt cũ `200` — vẫn «Đã điểm danh hh:mm» cùng giờ); DB `channel=CODE`, đúng một dòng | TAY+SQL |  |
| TC-MOB-FLW-04 | US-AS-09 AC10 | I, P | Safari và PWA đã đăng nhập; W1 mở | Mở `/checkin` không `#token` → hiện ô «Mã điểm danh» + «Lớp {mã lớp}» + nút `Điểm danh` → gõ mã đúng | «Đã điểm danh hh:mm»; chỉ ô nhập (không nút quét, không `BarcodeDetector` ở iOS); DB `channel=CODE` | TAY+SQL |  |
| TC-MOB-FLW-05 | US-AS-09 AC5; AC4 | A, I | Đã điểm danh xong ở W1 | Mở lại CÙNG liên kết `U` (Back rồi quét lại / `openurl` lần hai) | hiện lại «Đã điểm danh hh:mm» (giờ lượt đầu), không lỗi; DB vẫn đúng 1 dòng (`UNIQUE`) | TAY+SQL |  |
| TC-MOB-FLW-06 | US-AS-03 AC4; US-AS-15 AC6 | A | W1 mở | Chạm `Điểm danh` hai lần nhanh (< 300 ms) với mã đúng | đúng MỘT `POST` rời máy (gateway log); nút và ô khoá khi đang gửi; kết quả một lần | TAY+ADB |  |
| TC-MOB-FLW-07 | US-AS-03 AC1; US-AS-15 AC2 | A | W1 mở; mã ngắn đúng; xoay giữa lúc gửi (mạng chậm R-NET-slow) | Bấm `Điểm danh` rồi xoay máy dọc→ngang trong lúc «Đang gửi» | đúng MỘT lượt trong DB; không «Đang gửi» lặp; sau khi xong vẫn «Đã điểm danh hh:mm» (không mất kết quả khi tạo lại bố cục) | TAY+SQL |  |
| TC-MOB-FLW-08 | US-AS-09 AC1; AC9 | A | W1 mở; bật ghi `adb logcat` + proxy quan sát hoặc `chrome://inspect` KHÔNG dùng (release) | Điểm danh bằng quét; lọc log thiết bị và log gateway tìm token | token không có trong URL, `Referer`, logcat hay gateway log; chỉ nằm trong thân `POST` | ADB |  |
| TC-MOB-FLW-09 | US-AS-03 AC1; US-AS-15 AC15 | A | `sv.gioi` thuộc lớp QCMOB1 và QCMOB2; mở W1 ở QCMOB1 | Chọn lớp bằng bộ chọn trên thanh trên: QCMOB2 rồi QCMOB1; điểm danh bằng mã | dòng «Lớp {mã lớp}» đổi theo lớp đang chọn; mã của QCMOB1 nhập khi chọn QCMOB2 → lỗi «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.»; chọn đúng lớp thì thành công | TAY |  |
| TC-MOB-FLW-10 | US-AS-09 AC10 | A, I | SV chưa có lớp nào (`sv.moi` chưa vào lớp) | Mở `/checkin` | màn «Bạn chưa vào lớp nào.» chuẩn; không ô nhập mã; không lỗi | TAY |  |
| TC-MOB-FLW-11 | US-AS-02; US-AS-05 | A | Vòng 2 mở sau vòng 1 (rotate 10 s) | SV đã điểm danh vòng 1 điểm danh tiếp vòng 2 trong app | lượt mới (`round=2`), «Đã điểm danh hh:mm» đúng giờ lượt 2; hai dòng khác nhau trong DB | TAY+SQL |  |
| TC-MOB-FLW-12 | US-AS-03 AC8; US-AS-15 AC16(d) | A | Lớp QCMOB5 `self_checkin` tắt | Vào `/checkin` từ mục «Điểm danh» | màn «Lớp này không điểm danh bằng mã.» (không ô nhập, không nút gửi theo US-AS-15 AC16(d); nếu app vẫn hiện ô mã rồi báo lỗi ở ô thì ghi nhận lệch AC16(d)); không hộp lỗi kỹ thuật; không gọi `POST /checkins` | TAY+ADB |  |
| TC-MOB-FLW-13 | US-AS-03 AC8 | A | Lớp lưu trữ QCMOB4 | Mở `/checkin` ở lớp đó, thử điểm danh | câu tiếng Việt một dòng (lớp đã lưu trữ), không mã `COURSE_ARCHIVED`; không lượt | TAY |  |
| TC-MOB-FLW-14 | US-AS-14 AC2 (di động) | W | Playwright thiết bị di động `Pixel 7` và `iPhone 15` | Chạy `attendance-checkin.spec.ts` (các ca web) với `isMobile`, `hasTouch`, viewport 375×812 và 412×915 | tất cả ca xanh; không cuộn ngang; không lỗi 5xx; đối chiếu với TC-AS09-01…39 | E2E |  |
| TC-MOB-FLW-15 | US-AS-09 AC1; US-AS-11 AC3 | A | Hai SV (`sv.gioi`, `sv.kha`) trên hai AVD cùng quét một QR | Quét gần như đồng thời | mỗi người đúng 1 lượt; không lỗi 429 (hạn mức theo người); `round` giống nhau | TAY+SQL | hai AVD |
| TC-MOB-FLW-16 | US-AS-11 AC3b | A | Chrome của emulator có đăng nhập | Quét QR bằng ứng dụng Camera của emulator (poster ảo) rồi bấm liên kết gợi ý | Chrome mở `/checkin?c=…`; lượt web `channel=WEB`; app EduPilot không tự bật | TAY+SQL |  |
| TC-MOB-FLW-17 | US-AS-09 AC7 | A, I | Stack thử HTTP thường trong LAN (`http://<ip-lan>:580`), không phải ngữ cảnh an toàn; W mở | Mở liên kết `U` bản HTTP bằng Chrome emulator (`am start VIEW`) và Safari Simulator (`openurl`) | trang vẫn tự điểm danh «Đã điểm danh hh:mm»; KHÔNG khoá thiết bị (không `device`); DB cờ `NO_DEVICE_KEY`; không lỗi kỹ thuật; không hộp cảnh báo của app | TAY+SQL | stack HTTP thường |
| TC-MOB-FLW-18 | US-AS-09 AC3 | I | Safari chế độ riêng tư (không IndexedDB bền) hoặc chặn WebCrypto | Mở liên kết `U` / nhập mã | điểm danh được; không `device`; cờ `NO_DEVICE_KEY`; không lỗi hiển thị cho SV | TAY+SQL |  |

## 6.D Quét QR trong app (WebView, camera)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-SCN-01 | US-AS-11 AC3 | A | App đã đăng nhập; chưa cấp quyền camera (`pm reset-permissions`) | `/checkin` → `Quét mã điểm danh` | hộp quyền hệ thống hiện đúng một lần; chọn «Khi dùng ứng dụng» → khung camera vuông hiện ≤ 1.5 s; tiêu đề `Điểm danh` và `Thoát` thấy; khung rộng bằng panel (đo: video.width = panel.width ±2 px) | TAY+ADB |  |
| TC-MOB-SCN-02 | US-AS-11 AC3 | A | Camera ảo `virtualscene`, poster = QR đúng (Q-OK) | Quét | nhận QR → camera dừng ngay (`dumpsys media.camera` hết client trong ≤ 1 s), «Đang gửi» ≤ 300 ms, rồi «Đã điểm danh hh:mm» | TAY+ADB |  |
| TC-MOB-SCN-03 | US-AS-11 AC3; AC-3b | A | Poster lần lượt: Q-ALIEN (QR URL bất kỳ), Q-OTHER-ORIGIN, Q-PATH (path ≠ /checkin), Q-NOUUID (c không UUID), Q-NOHASH, QR không phải URL (chữ thường) | Mỗi poster giữ trước camera 5 s | cả 6: bỏ qua im lặng, khung quét vẫn chạy, KHÔNG câu lỗi, KHÔNG `POST` (gateway log trống); sau đó đổi sang Q-OK vẫn quét được | TAY+ADB |  |
| TC-MOB-SCN-04 | US-AS-11 AC3; AC-14 | A | Poster QR đúng nhưng đã hết hạn (chụp mã cũ > 2 bước) | Quét | «Mã đã hết hạn. Quét lại mã trên màn chiếu.»; nút `Quét mã điểm danh` ở lại; KHÔNG đăng xuất, không refresh phiên | TAY+ADB |  |
| TC-MOB-SCN-05 | US-AS-09 AC4 | A | Poster QR hợp lệ của lớp mà SV không thuộc (Q-OTHERCLASS) | Quét | «Bạn không thuộc lớp này.»; không lượt; phiên còn | TAY+SQL |  |
| TC-MOB-SCN-06 | US-AS-11 AC3 | A | Cửa sổ đã đóng; QR của cửa sổ đóng | Quét | «Đã đóng điểm danh.» | TAY |  |
| TC-MOB-SCN-07 | US-AS-11 AC3; US-AS-15 AC7(a) | A | Đang ở khung quét | Bấm `Thoát`; lặp bằng Back | về màn nhập mã (nút `Quét mã điểm danh` + `Nhập mã`), camera tắt; không rời `/checkin` | TAY+ADB |  |
| TC-MOB-SCN-08 | US-AS-11 AC3; AC-16; US-AS-15 AC10 | A | Đang quét | Home (đưa app xuống nền), chờ 10 s, mở lại | camera tắt khi ra nền (`visibilitychange`); mở lại về màn nhập mã có nút `Quét mã điểm danh`; không treo, không màn trắng; bấm quét lại hoạt động | TAY+ADB |  |
| TC-MOB-SCN-09 | US-AS-15 AC2 | A | Đang quét | Xoay dọc ↔ ngang 4 lần | không tạo lại Activity (`configChanges` có `orientation`; `dumpsys activity` cùng `ActivityRecord`); camera chạy tiếp hoặc dựng lại ≤ 2 s, không treo; không `POST` lặp | TAY+ADB |  |
| TC-MOB-SCN-10 | US-AS-09 AC9 (quét) | A | Poster Q-OK; mạng chậm (R-NET-slow) | Quét | «Đang gửi» ≤ 300 ms sau nhận QR; quét tiếp lần nữa trong lúc chờ KHÔNG gửi thêm; đúng một POST/token | TAY+ADB |  |
| TC-MOB-SCN-11 | US-AS-11 AC3a; AC-4 | A | AVD / WebView KHÔNG hỗ trợ `qr_code` (hoặc chặn `BarcodeDetector` bằng cờ Chrome) | Mở `/checkin` | không nút quét; ô nhập mã hiện ngay; điểm danh bằng mã thành công, `channel=CODE`; không câu «Chưa cấp quyền camera.» | TAY+SQL |  |
| TC-MOB-SCN-12 | US-AS-11 AC3a; AC-4c; US-AS-15 AC14 | A | Camera đang bị app khác dùng (mở Camera hệ thống và giữ), hoặc `hw.camera.back=none` | Bấm `Quét mã điểm danh` | «Không mở được camera.» + ô nhập mã; không treo; không `NotReadableError`/`NotFoundError` | TAY |  |
| TC-MOB-SCN-13 | US-AS-11 AC3 | A | Poster Q-OK | Quét; ngay lúc «Đang gửi» bật chế độ máy bay (R-NET-off) rồi tắt | «Chưa có mạng.» + `Thử lại` sau ≤ 15 s; sau khi có mạng `Thử lại` gửi lại đúng token còn hạn; một lượt trong DB | TAY+SQL |  |
| TC-MOB-SCN-14 | US-AS-11 AC3 | A | Hai QR trong khung hình (Q-ALIEN + Q-OK) | Quét | chọn đúng QR hợp lệ (Q-OK), bỏ QR lạ; một POST | TAY |  |
| TC-MOB-SCN-15 | US-AS-11 AC3 | A | Ánh sáng yếu / QR nghiêng 30° / QR xa (poster đổi kích thước) | Quét ở 3 điều kiện | nhận được trong ≤ 5 s ở điều kiện thường; điều kiện khắc nghiệt không crash, không POST sai, ô `Nhập mã` luôn dùng được | TAY |  |
| TC-MOB-SCN-16 | US-AS-11 AC3; AC-17 | AR | Máy thật, camera sau | Quét thật màn chiếu (laptop) 5 lần | ghi trung vị / lớn nhất từ lúc đưa vào khung tới «Đã điểm danh»; mục tiêu ≤ 3 s; camera sau được chọn (không phải camera trước) | TAY | máy thật |

## 6.E Vị trí / mạng trường trên điện thoại (US-AS-10)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-GEO-01 | US-AS-10 AC4; AC-10 | A | Lớp QCMOB2 `geofence` bật; W mở; app chưa cấp vị trí (`pm reset-permissions`) | Điểm danh bằng mã; quay video / chụp liên tiếp (R-SHOT loop 5 ảnh/s) từ lúc bấm | dòng «Lớp này kiểm tra vị trí khi điểm danh.» hiện TRƯỚC hộp xin quyền vị trí (ảnh dòng xuất hiện trước ảnh hộp quyền); chỉ một dòng; sau đó «Đã điểm danh hh:mm» | TAY |  |
| TC-MOB-GEO-02 | US-AS-10 AC2 | A | QCMOB2; vị trí giả của emulator đặt TRONG bán kính (`adb emu geo fix {lng} {lat}`); cấp quyền «Khi dùng ứng dụng» | Điểm danh bằng mã | 201; DB `geo_distance_m` ≤ bán kính, không cờ `GEO_OUT`/`GEO_DENIED`; SV không thấy gì thêm; không có toạ độ trong DB | TAY+SQL |  |
| TC-MOB-GEO-03 | US-AS-10 AC2; AC5 | A | QCMOB2; vị trí đặt NGOÀI vùng (cách ≥ bán kính + 100 m) | Điểm danh | 201 «Đã điểm danh hh:mm»; DB cờ `GEO_OUT`; Staff thấy chip ở lưới; SV không thấy | TAY+SQL |  |
| TC-MOB-GEO-04 | US-AS-10 AC4; US-AS-15 AC8 | A | QCMOB2; từ chối quyền vị trí (lần 1) | Điểm danh | vẫn «Đã điểm danh hh:mm»; DB cờ `GEO_DENIED`; không câu lỗi cho SV; thời gian từ bấm tới kết quả ≤ 3 s + hộp quyền | TAY+SQL |  |
| TC-MOB-GEO-05 | US-AS-10 AC2; Android 12+ | A | QCMOB2; API 35 (hộp «Chính xác / Gần đúng») | Bấm điểm danh; ở hộp quyền chọn «Gần đúng» rồi lần khác «Chính xác» | hộp quyền hiện đủ hai lựa chọn Chính xác / Gần đúng (nếu hộp không hiện hoặc luôn bị từ chối → ghi: manifest hiện chỉ khai ACCESS_FINE_LOCATION, không khai ACCESS_COARSE_LOCATION; Capacitor xin cả hai nên có thể bị từ chối ngay trên Android 12+ → dự kiến F nếu quyền không bao giờ cấp được); «Gần đúng» ⇒ `GEO_WEAK` hoặc `GEO_OUT`, vẫn 201 | TAY+ADB+SQL |  |
| TC-MOB-GEO-06 | US-AS-10 AC4 | A | QCMOB2; Vị trí hệ thống TẮT (Cài đặt → Vị trí → tắt) | Điểm danh | không treo quá 8 s (timeout định vị); «Đã điểm danh hh:mm»; cờ `GEO_DENIED`; không câu lỗi | TAY+SQL |  |
| TC-MOB-GEO-07 | US-AS-10 AC1 | A, I | Lớp QCMOB1 KHÔNG khai geofence / campus | Điểm danh trên lớp đó | không dòng «Lớp này kiểm tra vị trí…», không hộp xin quyền vị trí; DB không cờ `GEO_*`/`OFF_CAMPUS` | TAY+SQL |  |
| TC-MOB-GEO-08 | US-AS-15 AC8 | A | QCMOB2; vị trí đã bị từ chối ở GEO-04 | Cấp lại ở Cài đặt → Ứng dụng → EduPilot → Quyền → Vị trí → «Khi dùng ứng dụng»; quay lại app; điểm danh vòng 2 | KHÔNG cần đăng nhập lại / tải lại thủ công; lượt sau gửi vị trí bình thường (DB không `GEO_DENIED`, có `geo_distance_m`) | TAY+SQL |  |
| TC-MOB-GEO-09 | US-AS-15 AC8 | A | QCMOB2; từ chối hai lần (hệ thống chặn vĩnh viễn) | Điểm danh 3 lần ở 3 vòng / 3 lần nhập | hộp xin quyền KHÔNG hiện lại (đếm = 0 ở lần 3); không vòng lặp xin quyền; điểm danh vẫn được với `GEO_DENIED` | TAY+SQL |  |
| TC-MOB-GEO-10 | US-AS-12 AC5; AC-11 | AR hoặc A + app GPS giả | QCMOB2; bật «Chọn ứng dụng vị trí giả» (`adb shell appops set <gps-giả> android:mock_location allow`) | Điểm danh trong app | plugin `isMockLocation` = true → cờ `GEO_MOCK`, `assurance ≤ 2`; vẫn 201; SV không thấy | TAY+SQL | máy thật (hoặc emulator + app GPS giả) |
| TC-MOB-GEO-11 | US-AS-10 AC4; US-AS-15 AC8 | I, P | QCMOB2; Safari rồi PWA; `simctl privacy booted reset location` | Điểm danh bằng mã; từ chối; rồi `simctl privacy booted grant location` (hoặc Cài đặt → Quyền riêng tư → Dịch vụ định vị); mở lại trang | dòng «Lớp này kiểm tra vị trí khi điểm danh.» hiện trước hộp; từ chối → `GEO_DENIED` + vẫn thành công; sau khi cấp lại, tải/hiện lại trang không cần đăng nhập lại; lượt sau có vị trí | SIMCTL+SQL |  |
| TC-MOB-GEO-12 | US-AS-10 AC2; luật cứng 5 | A, I | Sau GEO-02…11 | `adb logcat -d`, `docker logs` gateway, `pg_dump` bảng `attendance_checkins`, `audit_log` — tìm toạ độ đã đặt (lat/lng), IP máy | 0 lần xuất hiện toạ độ, IP, tên máy ở log thiết bị, gateway, DB, `audit_log`; chỉ có `geo_distance_m`, `geo_accuracy_m`, `in_campus` | ADB+SQL |  |
| TC-MOB-GEO-13 | US-AS-10 AC3 | A | Lớp có `campus_cidrs`; SV dùng dữ liệu di động (ngoài CIDR) rồi Wi-Fi trường (nếu có) | Điểm danh ở hai mạng | ngoài CIDR → `OFF_CAMPUS`; trong CIDR → không cờ; IP không vào DB/log; Wi-Fi trường không có → ghi KHÔNG KIỂM ĐƯỢC phần trong CIDR | TAY+SQL | mạng trường |

## 6.F Nhánh lỗi, QR lạ, hết hạn, lớp khác, chưa đăng nhập, phân quyền
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-ERR-01 | US-AS-09 AC8; AC4 | I, A | (a) `https://{VPS}/checkin?c=khong-phai-uuid#<token thật, đúng cú pháp>`; (b) `…?c=khong-phai-uuid#tokenbatky` | Mở bằng `simctl openurl` / `am start VIEW` | (a) «Liên kết không đúng.» một dòng, không gọi API, không đăng xuất; (b) rơi về ô nhập mã, **0 POST**, URL giữ nguyên | TAY+ADB | P-11-145: (a) `ERR-01c` P; (b) `ERR-01b` P |
| TC-MOB-ERR-02 | US-AS-09 AC4 | I, A | Liên kết có token đã hết hạn (> 2 bước, giữ liên kết cũ) | Mở liên kết | «Mã đã hết hạn. Quét lại mã trên màn chiếu.»; web: không nút; app: nút `Quét mã điểm danh` | TAY |  |
| TC-MOB-ERR-03 | US-AS-09 AC4 | I, A | Liên kết token có MAC sai (sửa một ký tự cuối) | Mở liên kết | «Mã không đúng. Quét lại mã trên màn chiếu.»; không phân biệt cửa sổ lạ / MAC sai | TAY |  |
| TC-MOB-ERR-04 | US-AS-09 AC4 | I, A | Cửa sổ đã đóng; liên kết còn trong 2 bước | Mở liên kết | «Đã đóng điểm danh.» | TAY |  |
| TC-MOB-ERR-05 | US-AS-09 AC4; US-AS-15 AC15 | I, A | Liên kết hợp lệ của lớp mà SV KHÔNG thuộc | Mở liên kết | «Bạn không thuộc lớp này.»; không lượt; không lộ cửa sổ có tồn tại | TAY+SQL |  |
| TC-MOB-ERR-06 | US-AS-09 AC2 | I, A | SV chưa đăng nhập; mở liên kết; để màn đăng nhập > 60 s rồi đăng nhập | Đăng nhập chậm | token bị bỏ khỏi `sessionStorage` quá 60 s; hiện «Mã đã hết hạn. Quét lại mã trên màn chiếu.»; không `POST` | TAY+ADB |  |
| TC-MOB-ERR-07 | US-AS-09 AC8; US-AS-15 AC15 | I, A | `teacher@` đăng nhập trong Safari / app | Mở liên kết điểm danh `U` | màn chặn chuẩn; không ô mã; gateway không nhận `POST /checkins`; không lộ token | TAY+ADB |  |
| TC-MOB-ERR-08 | US-AS-09 AC8 | I, A | `ta@`, `admin@` | Như ERR-07 | như ERR-07 | TAY+ADB |  |
| TC-MOB-ERR-09 | US-AS-03 AC3b; AC10 | A, I | W mở; mã đúng đang hiện | Gõ 6 mã SAI liên tiếp trong 60 s rồi gõ mã ĐÚNG | đến lần sai thứ 6: «Thử lại sau {n} giây.» đếm ngược từng giây; nút `Điểm danh` khoá, lý do «Thử lại sau {n} giây.»; mã đúng trong lúc 429 vẫn bị 429; hết giờ quay về ô nhập; gateway trả 429 kèm `Retry-After` | TAY+ADB |  |
| TC-MOB-ERR-10 | US-AS-09 AC4 | A | Giả lập 429 ở đường `token` (qua tải 11 lượt/phút từ script cùng tài khoản) | Quét QR ngay sau đó | «Thử lại sau {n} giây.»; hết giờ tự gửi lại đúng token nếu còn hạn, nếu hết hạn → «Mã đã hết hạn. Quét lại mã trên màn chiếu.» | TAY+API |  |
| TC-MOB-ERR-11 | US-AS-09 AC10 | A, I | W mở | Nhập mã sai (đúng cú pháp) và mã hết hạn (mã cũ) | cả hai cùng câu «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» ở dưới ô (không phân biệt); ô có viền lỗi + `aria-invalid`; không hộp thoại | TAY |  |
| TC-MOB-ERR-12 | US-AS-09 AC10; US-AS-15 AC6 | A, I | Ô mã rỗng; ký tự lạ (emoji, `@#$`, dấu tiếng Việt) | Quan sát nút; gõ ký tự lạ rồi `Điểm danh` | nút `Điểm danh` khoá khi rỗng; ký tự lạ → cùng câu «Mã không đúng hoặc đã hết hạn. …», không gọi API; không crash | TAY+ADB |  |
| TC-MOB-ERR-13 | US-AS-09 AC10 | A, I | Ô mã | Dán chuỗi 40 ký tự; gõ 12+ ký tự | ô giới hạn 12 ký tự (`maxlength`), phần dư bị cắt; không tràn ngang; mã dán có khoảng trắng đầu/cuối vẫn xử lý | TAY |  |
| TC-MOB-ERR-14 | US-AS-15 AC14 | W | Playwright `route` trả 500 / HTML / rỗng cho `POST …/checkins` | Gửi mã ngắn và token | Câu theo ĐÚNG loại lỗi ở SRS `FEAT-ui-foundation` bảng 3.7: 500 → «Có lỗi xảy ra. Dữ liệu của bạn vẫn an toàn. Thử lại sau ít phút.»; HTML/502 → «Máy chủ chưa phản hồi đúng. Dữ liệu của bạn vẫn an toàn.»; thân rỗng/không đọc được → «Phản hồi từ máy chủ không đọc được. Thử lại sau.» Mỗi ca một câu, `role=alert`, có «Thử lại»; không JSON, `undefined`, mã HTTP; nút còn dùng được | E2E |  |
| TC-MOB-ERR-15 | US-AS-15 AC14 | I | iPhone Simulator Safari, tắt mạng trước khi mở liên kết lần đầu | Mở liên kết khi offline | trang lỗi của Safari (do hệ điều hành, ngoài tầm app) — ghi nhận nội dung; sau khi có mạng, mở lại liên kết còn hạn mã → điểm danh được; KHÔNG kỳ vọng chữ tiếng Việt của app ở bước offline đầu tiên | TAY |  |
| TC-MOB-ERR-16 | US-AS-09 AC4; điều kiện biên | A | Đồng hồ máy lệch +10 phút (tắt giờ tự động) rồi trả lại | Điểm danh bằng quét và bằng mã | vẫn «Đã điểm danh hh:mm» (hạn mã do máy chủ tính; riêng cất `sessionStorage` 60 s dùng giờ máy: ghi nhận); giờ hiển thị khớp `checked_in_at` đổi sang giờ Việt Nam; trả lại giờ tự động sau test | TAY+SQL |  |
| TC-MOB-ERR-17 | US-AS-09 AC1; múi giờ | A, I | Múi giờ máy đặt UTC rồi Asia/Ho_Chi_Minh | Điểm danh, đọc «Đã điểm danh hh:mm» | ghi hai giá trị; kỳ vọng giờ khớp `checked_in_at` đổi sang giờ Việt Nam; nếu hiển thị theo múi giờ máy khi máy đặt UTC thì ghi nhận cho PM quyết (spec chỉ nói hh:mm) | TAY+SQL |  |
| TC-MOB-ERR-18 | US-AS-03 AC8 | A, I | Thân yêu cầu có khoá lạ (qua `curl` bằng token SV, không từ giao diện) | `POST …/checkins {code, student_id}` | 422 `VALIDATION_FAILED` `details[].code=unknown`; không ghi gì (đối chiếu luật cứng 4); giao diện di động không thể gửi khoá này (đọc log) | API |  |
| TC-MOB-ERR-19 | US-AS-09 AC8 | A, I | SV `sv.nguyco` chưa ghi danh lớp QCMOB1, mở liên kết lớp đó | Mở | «Bạn không thuộc lớp này.»; không phân biệt lớp tồn tại hay không | TAY+SQL |  |
| TC-MOB-ERR-20 | US-AS-09 AC4 | A | App; cửa sổ đã đóng hơn 15 phút; nhập mã ngắn cũ | Nhập mã | «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» (khớp cửa sổ đóng > 15 phút không trả `WINDOW_CLOSED`); không lộ khác biệt | TAY |  |

## 6.G Độ rộng 360 / 375 / 390 / 411 / 430 px và xoay ngang (US-AS-15 AC1, AC2)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-UIW-01 | US-AS-15 AC1 | W, A (R-SIZE-360), AR | `sv.gioi`; W đang mở; viewport 360×640 (nhỏ nhất), rồi 360×800 | Với từng trạng thái trong [đang gửi, đã điểm danh, nhập mã, đang quét, lỗi (một câu), 429] mở `/checkin` (đẩy trạng thái bằng route giả hoặc dữ liệu thật); chụp ảnh từng ô; đo `document.documentElement.scrollWidth` và `innerWidth`; đo vị trí nút chính | 6 trạng thái × 2 chiều cao: `scrollWidth ≤ innerWidth`; không cuộn dọc ở `/checkin` (nội dung + nút chính trong MỘT màn); nút chính (`Điểm danh` / `Quét mã điểm danh` / `Thử lại`) nằm trọn trong khung nhìn; chữ không cắt dấu, không chồng; khung camera vuông (tỷ lệ 1:1 ±2 %) rộng bằng panel | E2E+TAY |  |
| TC-MOB-UIW-02 | US-AS-15 AC1 | W, A (R-SIZE-375), I (iPhone 13 mini), IR | Như UIW-01, viewport 375×667 và 375×812 | Như UIW-01 | như UIW-01; riêng 375×667 (iPhone SE) nút chính vẫn trong khung nhìn khi KHÔNG bàn phím | E2E+TAY |  |
| TC-MOB-UIW-03 | US-AS-15 AC1 | W, A (R-SIZE-390), I (iPhone 15) | 390×844 | Như UIW-01 | như UIW-01 | E2E+TAY |  |
| TC-MOB-UIW-04 | US-AS-15 AC1 | W, A (R-SIZE-411), AR | 411×891 (Pixel) | Như UIW-01 | như UIW-01 | E2E+TAY |  |
| TC-MOB-UIW-05 | US-AS-15 AC1 | W, A (R-SIZE-430), I (iPhone 15 Pro Max) | 430×932 | Như UIW-01 | như UIW-01; không chữ quá nhỏ ở panel hẹp rộng (khoảng trắng cân đối, nội dung căn giữa cột đọc) | E2E+TAY |  |
| TC-MOB-UIW-06 | US-AS-15 AC1 | W | Viewport 360 / 375 / 390 / 411 / 430 × 640 | Chạy kiểm tự động 5 × 6 + `/attendance` Staff; đính 35 ảnh vào `shots-mobile/uiw/` | 35/35 ô: `scrollWidth ≤ innerWidth`; nút chính trong khung nhìn (`getBoundingClientRect().bottom ≤ innerHeight`); báo cáo bảng 5×6 điền P/F | E2E |  |
| TC-MOB-UIW-07 | US-AS-15 AC1 | W, A | `sv.gioi`; chuỗi dài nhất: lỗi «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.»; tên lớp dài | Gây lỗi mã ngắn ở 360 px | câu lỗi xuống dòng trong panel, không tràn ngang, không cắt dấu (kiểm ảnh phóng 200 %); không chồng lên nút; nút chính vẫn trong khung nhìn | E2E+TAY |  |
| TC-MOB-UIW-08 | US-AS-15 AC1; US-AS-09 AC7 | W, A | Trạng thái «Đã điểm danh hh:mm» | Đo ở 360 và 430 | chữ `Đã điểm danh 07:03` một dòng (hoặc xuống dòng gọn), dấu `Check` đỏ cạnh chữ; đây là chỗ đỏ DUY NHẤT (đếm phần tử có màu đỏ thương hiệu trong `main` = 1) | E2E |  |
| TC-MOB-UIW-09 | US-AS-15 AC1 | A | Đang quét, 360 / 411 / 430 | Đo khung camera | khung vuông rộng bằng panel (sai số ≤ 2 px); nút `Thoát` thấy dưới khung không cuộn; không méo hình (object-fit cover) | TAY+ADB |  |
| TC-MOB-UIW-10 | US-AS-15 AC1 | W, A | `/attendance` của GV `teacher@` ở 5 độ rộng | Mở danh sách buổi, lưới điểm danh, thanh lưu | không cuộn ngang ở trang (lưới có thể cuộn trong khung riêng, ghi nhận); thanh lưu nằm trên thanh dưới (`56px + safe-area`), không che hàng cuối; hành động chính thấy | E2E+TAY |  |
| TC-MOB-UIW-11 | US-AS-15 AC2 | W, A (R-ROT-land), I (xoay), AR | 667×375 (iPhone SE ngang) | Mở `/checkin` ở 6 trạng thái | nút chính, ô nhập, `Thoát` không bị thanh dưới / tai thỏ che; khung camera co theo chiều cao khả dụng (không tràn ra ngoài khung nhìn); không cuộn ngang | E2E+TAY |  |
| TC-MOB-UIW-12 | US-AS-15 AC2 | W, A, I | 844×390 (iPhone 15 ngang) | Như UIW-11 | như UIW-11; lề trái/phải lớn hơn phần tai thỏ (Dynamic Island) — đo `safe-area-inset-left/right` | E2E+TAY |  |
| TC-MOB-UIW-13 | US-AS-15 AC2 | W, A, AR | 915×412 (Pixel ngang) | Như UIW-11 | như UIW-11 | E2E+TAY |  |
| TC-MOB-UIW-14 | US-AS-15 AC2 | A | Đang gửi (mạng chậm) | Xoay máy giữa lúc «Đang gửi» | đúng một lượt trong DB; kết quả hiện sau xoay; không «Đang gửi» lặp | TAY+SQL |  |
| TC-MOB-UIW-15 | US-AS-15 AC2 | A | Ô mã đã gõ `ABC12` | Xoay dọc → ngang → dọc | mã vẫn `ABC12`; focus giữ hoặc trả lại hợp lý; không gửi | TAY |  |
| TC-MOB-UIW-16 | US-AS-15 AC2 | A | Đang quét | Xoay máy | camera chạy tiếp hoặc dựng lại ≤ 2 s, không treo; Activity không tạo lại (cùng `ActivityRecord`) | TAY+ADB |  |
| TC-MOB-UIW-17 | US-AS-15 AC1; AC2 (shell) | A, I | Khung app `AppShell` ở dọc và ngang | Quan sát thanh trên, thanh dưới (SV: 5 mục), chọn lớp, menu «Thêm» | dọc: thanh dưới 5 mục thấy đủ, không cắt chữ. Ngang điện thoại (cao < 500 px CSS, `pointer: coarse`) → **vẫn thanh đáy** theo M2 (`@media (max-width: 719px), (max-height: 500px) and (pointer: coarse)` trong `AppShell.module.css`), đủ 5 mục, không cắt. Thanh bên chỉ khi RỘNG ≥ 720 px VÀ cao ≥ 500 px (hoặc trỏ chuột chính xác), ghi nhận, không mất mục; không cuộn ngang | TAY |  |
| TC-MOB-UIW-18 | US-AS-15 AC1 | W, A | Mọi màn trong khung app đầu SV: Hôm nay, Chat riêng, Hỏi đáp, Luyện đề, Thêm | Mở từng màn ở 360 và 430 | không cuộn ngang (`scrollWidth ≤ innerWidth`) ở mọi màn; thanh dưới không che nội dung cuối (`padding-bottom` 56 px + safe area) | E2E |  |
| TC-MOB-UIW-19 | US-AS-15 AC1 | W | Tải `/checkin` mỗi độ rộng; đo CLS (Lighthouse mobile / `PerformanceObserver`) | Chạy 3 lượt mỗi độ rộng | CLS ≤ 0,1 (ngưỡng của `lighthouserc.json`); chuyển trạng thái sending → done không nhảy bố cục (khung xương cùng kích thước kết quả) | E2E |  |
| TC-MOB-UIW-20 | US-AS-15 AC1 | W, A | Độ cao thấp 640 px ở 360 px, có thanh dưới 56 px | Trạng thái nhập mã không bàn phím | ô, nhãn «Lớp {mã lớp}», nút `Điểm danh` (dock) thấy hết, không cuộn dọc | E2E+TAY |  |
| TC-MOB-UIW-21 | US-AS-15 AC1 | A | Hai chế độ điều hướng: ba nút và cử chỉ | Mở `/checkin` ở 411 | thanh điều hướng hệ thống không che thanh dưới app (xem SAF-02/03) | TAY |  |
| TC-MOB-UIW-22 | US-AS-15 AC1; AC13 | W, A | Hiển thị «Kích thước màn hình» Android nhỏ nhất và lớn nhất (R-DENSITY) | Mở `/checkin` | nội dung không tràn; nút ≥ 44 px (CSS px theo mật độ); không chữ chồng | TAY+ADB |  |

## 6.H Vùng an toàn: tai thỏ, đục lỗ, thanh cử chỉ, ba nút, Home (US-AS-15 AC3)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-SAF-01 | US-AS-15 AC3 | A (R-CUT-tall, R-CUT-corner, R-CUT-double, R-CUT-hole) | Emulator có cutout; API 35 (edge-to-edge) | Với từng kiểu cutout mở `/checkin` và «Hôm nay»; chụp ảnh | tiêu đề `Điểm danh` và thanh trên KHÔNG nằm dưới thanh trạng thái / cutout (đo: y đầu của h1 ≥ chiều cao status bar); không dải trắng ngoài ý muốn trên đầu | TAY+ADB |  |
| TC-MOB-SAF-02 | US-AS-15 AC3 | A (R-NAV-gesture) | Điều hướng cử chỉ | Mở `/checkin` ở trạng thái nhập mã (dock) và «Hôm nay» | thanh dưới của app và dock nằm TRÊN vùng cử chỉ (không dưới thanh cử chỉ); không khoảng trống bất thường dưới đáy; nút cuối chạm được | TAY+ADB |  |
| TC-MOB-SAF-03 | US-AS-15 AC3 | A (R-NAV-3btn) | Điều hướng ba nút | Như SAF-02 | thanh dưới và dock không bị ba nút che; khoảng cách đáy đúng | TAY+ADB |  |
| TC-MOB-SAF-04 | US-AS-15 AC3 | A (R-ROT-land + R-CUT-tall) | Ngang có cutout bên trái hoặc phải | Mở `/checkin` ngang | lề trái/phải lớn hơn phần cutout; chữ và nút không bị cutout che; thanh bên không dưới cutout | TAY+ADB |  |
| TC-MOB-SAF-05 | US-AS-15 AC3 | I (iPhone 15 Dynamic Island), IR | Safari (không PWA) | Mở `/checkin` dọc; chụp | tiêu đề dưới đảo; thanh dưới (nav) trên thanh Home (không dưới); không dải trắng đáy; ghi: `viewport-fit=cover` chưa đặt trong `layout.tsx` (đọc mã 25/11) — kết quả nhìn thấy mới là tiêu chí | SIMCTL+TAY |  |
| TC-MOB-SAF-06 | US-AS-15 AC3; AC11 | P | PWA standalone iPhone 15 | Mở `/checkin`, «Hôm nay»; chụp | thanh trạng thái không che tiêu đề; thanh dưới và dock trên thanh Home; nền khoảng trên/dưới cùng màu canvas (không đen/ trắng lệch); không chữ dưới Dynamic Island | SIMCTL+TAY |  |
| TC-MOB-SAF-07 | US-AS-15 AC3 | I, P | iPhone 15 ngang (844×390) | Mở `/checkin` ngang (Safari và PWA) | lề trái/phải ≥ phần tai thỏ; nút chính, ô nhập, `Thoát` không bị che; không dải trắng | SIMCTL+TAY |  |
| TC-MOB-SAF-08 | US-AS-15 AC3 | I | iPhone SE (không tai thỏ, có nút Home) | Mở `/checkin` | bố cục đúng, không khoảng đệm thừa dưới đáy (safe-area 0) | TAY |  |
| TC-MOB-SAF-09 | US-AS-15 AC3 | W | Playwright giả `env(safe-area-inset-*)` bằng `page.addStyleTag(':root{--sai:...}')` hoặc thiết bị `iPhone 15` + CSS | Đo `dock.bottom` và `nav.top` ở inset đáy 34 px | dock nằm ngay trên thanh dưới, thanh dưới cao `56px + 34px`; không chồng nhau (khoảng ≥ 0) | E2E |  |
| TC-MOB-SAF-10 | US-AS-15 AC3 | A, I | Mọi trạng thái `/checkin` với cutout | Rà 6 trạng thái | trong cả 6 trạng thái không phần tử tương tác nào nằm dưới vùng bị che (so ảnh với đường biên vùng an toàn) | TAY |  |

## 6.I Cỡ chữ lớn và phóng (US-AS-15 AC4)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-TXT-01 | US-AS-15 AC4 | A (R-FONT-1.3) | Cỡ chữ 1,3×; 360 px | Duyệt 6 trạng thái `/checkin` | không cuộn ngang; không chữ chồng / cắt; nút chính chạm được; câu lỗi đọc đủ một câu | TAY+ADB |  |
| TC-MOB-TXT-02 | US-AS-15 AC4 | A (R-FONT-2.0) | Cỡ chữ lớn nhất của hệ thống (2,0× hoặc tối đa của API 35); 360 và 411 px | Duyệt 6 trạng thái + ô nhập khi bàn phím mở | không cuộn ngang; cho phép cuộn dọc; dock KHÔNG che quá 1/3 màn và KHÔNG che thông báo lỗi; nút vẫn chạm được; ghi: WebView có theo cỡ chữ hệ thống không (đo cỡ chữ `h1` thực tế) | TAY+ADB |  |
| TC-MOB-TXT-03 | US-AS-15 AC4 | A | Cỡ hiển thị (density) lớn nhất + cỡ chữ lớn nhất | Duyệt `/checkin` | không vỡ bố cục; nút ≥ 44 px (CSS px) | TAY+ADB |  |
| TC-MOB-TXT-04 | US-AS-15 AC4 | I (Safari, `aA` → 200 %) | Safari phóng trang 200 % | Duyệt 6 trạng thái `/checkin` | không cuộn ngang; không chữ chồng; nút chính chạm được; dock không che thông báo lỗi | TAY |  |
| TC-MOB-TXT-05 | US-AS-15 AC4 | I (S-CONTENT-SIZE accessibility-extra-extra-extra-large) | iOS «Chữ lớn hơn» tối đa | Duyệt `/checkin` Safari và PWA | ghi nhận WebKit có theo cỡ chữ hệ thống không; dù theo hay không: không vỡ bố cục, không cuộn ngang | SIMCTL+TAY |  |
| TC-MOB-TXT-06 | US-AS-15 AC4; DESIGN §16 | W | Playwright `zoom=2` / `deviceScaleFactor` + `page.setViewportSize(180×…)` (tương đương zoom 200 % ở 360 px) | Duyệt 6 trạng thái | không cuộn ngang; chữ không bị cắt (kiểm `overflow` của phần tử); nút trong khung nhìn sau cuộn dọc | E2E |  |
| TC-MOB-TXT-07 | US-AS-15 AC4 | A | Cỡ chữ 2,0×, trạng thái lỗi 429 đếm ngược | Gây 429 | câu «Thử lại sau {n} giây.» đọc đủ, không cắt; nút khoá có lý do thấy | TAY |  |
| TC-MOB-TXT-08 | US-AS-15 AC4 | A | Cỡ chữ 2,0× + bàn phím mở | Gõ mã | ô nhập thấy trên bàn phím; có thể cuộn tới nút `Điểm danh`; không hộp lỗi bị che | TAY |  |

## 6.J Chế độ sáng / tối của máy (US-AS-15 AC5)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-THM-01 | US-AS-15 AC5 | A (R-DARK-on) | Android tối (`cmd uimode night yes`) | Mở app và `/checkin` (6 trạng thái) | giao diện VẪN sáng (nền trắng, chữ tối); không nền tối chữ tối; ô nhập, dấu `Check` đỏ, nút hiện rõ | TAY+ADB |  |
| TC-MOB-THM-02 | US-AS-15 AC5 | A, Chrome emulator | Chrome với cờ «Auto Dark Mode for Web Contents» bật (`chrome://flags/#enable-force-dark`) + máy tối | Mở `/checkin` bằng liên kết `am start VIEW` | trang vẫn sáng hoặc nếu bị tối ép buộc thì chữ vẫn đọc được (AA), ô nhập và `Check` không biến mất — ghi nhận Chrome có tôn trọng `color-scheme: light` | TAY |  |
| TC-MOB-THM-03 | US-AS-15 AC5 | A | Developer options «Override force-dark» bật (WebView của app) | Mở app → `/checkin` | giao diện không bị đảo màu; ô nhập và `Check` còn; tương phản chữ ≥ 4,5:1 (đo bằng ảnh chụp) | TAY+ADB |  |
| TC-MOB-THM-04 | US-AS-15 AC5 | I (S-DARK), P | iOS tối (`simctl ui booted appearance dark`) | Mở `/checkin` (Safari, PWA) | giao diện vẫn sáng; thanh trạng thái PWA đọc được (chữ hệ thống tương phản với `theme_color`/nền) | SIMCTL+TAY |  |
| TC-MOB-THM-05 | US-AS-15 AC5 | A | Splash khi máy tối | Khởi động nguội ở chế độ tối | splash sáng đúng thương hiệu, không chớp đen rồi trắng; trang lỗi chuẩn (`error.html`) ở chế độ tối vẫn đọc được (chữ tối trên nền sáng) | TAY |  |
| TC-MOB-THM-06 | US-AS-15 AC5 | W | Playwright `colorScheme: dark` + `forcedColors: active` | Mở `/checkin` 6 trạng thái; axe `color-contrast` | axe: không lỗi tương phản nghiêm trọng; ảnh chụp vẫn sáng | E2E+AXE |  |
| TC-MOB-THM-07 | US-AS-15 AC5 | A, I | Thanh trạng thái | Đọc ảnh: biểu tượng pin/giờ so với nền thanh (Android: màu thanh trạng thái; iOS PWA: kiểu thanh) | biểu tượng hệ thống đọc được (không trắng trên trắng, không đen trên đen); ghi màu `theme_color` đang dùng (BRAND.red) | TAY |  |
| TC-MOB-THM-08 | US-AS-15 AC5; US-AS-15 phạm vi | A | Ban ngày ngoài nắng (độ sáng 100 %) và ban đêm (độ sáng thấp) | Đọc `Đã điểm danh hh:mm` và lỗi | đạt AA (đo bằng ảnh chụp: chữ chính ≥ 4,5:1, chữ lớn ≥ 3:1); trạng thái không chỉ bằng màu (có chữ) | TAY |  |

## 6.K Bàn phím che ô nhập (US-AS-15 AC6)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-KBD-01 | US-AS-15 AC6 | A | Nhập mã, 360×640 và 411×891, dọc | Chạm ô «Mã điểm danh» | bàn phím chữ mở (chữ hoa); ô nhập VÀ nút `Điểm danh` (dock) thấy cùng lúc trên bàn phím; đo: `dock.bottom ≤ keyboardTop` | TAY+ADB |  |
| TC-MOB-KBD-02 | US-AS-15 AC6 | A, I | Như KBD-01, ngang (667×375) | Chạm ô | ô nhập thấy trước; nếu không đủ chỗ thì cuộn tới được nút `Điểm danh`; không cả hai bị che | TAY |  |
| TC-MOB-KBD-03 | US-AS-15 AC6 | A, I | Mã sai → lỗi dưới ô | Gõ mã sai, bấm Điểm danh khi bàn phím mở | thông báo lỗi của ô KHÔNG bị bàn phím che (thấy hoặc cuộn tới trong ≤ 1 thao tác) | TAY |  |
| TC-MOB-KBD-04 | US-AS-15 AC6; US-AS-03 AC3a | A, I | W mở | Gõ `abc 123`, `ABC-123`, `abc123` bằng bàn phím | cả ba nhận; ô hiển thị chữ hoa; không gợi ý sửa, không viết hoa tự động lạ, không gạch đỏ chính tả (`autocorrect=off`, `spellcheck=false`) | TAY+SQL |  |
| TC-MOB-KBD-05 | US-AS-15 AC6 | A, I | Bàn phím có phím «Đi» (`enterkeyhint=go`) | Gõ mã đúng, bấm «Đi» / Enter | gửi như bấm nút; một POST; nhãn phím đúng «Đi»/«Go» | TAY+ADB |  |
| TC-MOB-KBD-06 | US-AS-15 AC6 | A | Đã gõ mã | Đóng bàn phím (Back/vuốt xuống) rồi xoay máy | mã đã gõ không mất; trang không rời `/checkin` (Back lần đầu chỉ đóng bàn phím) | TAY |  |
| TC-MOB-KBD-07 | US-AS-15 AC6 | A | Mạng chậm; mã đúng | Bấm `Điểm danh` rồi chạm tiếp ô và nút | ô và nút khoá khi đang gửi; không gửi hai lần (1 POST) | TAY+ADB |  |
| TC-MOB-KBD-08 | US-AS-15 AC6 | A | Gboard tiếng Việt (Telex/VNI) hoặc bàn phím có dự đoán | Gõ mã `abc123` | không chèn dấu tiếng Việt, không dự đoán thay chữ; ô nhận đúng ký tự | TAY |  |
| TC-MOB-KBD-09 | US-AS-15 AC6 | A, I | Clipboard có `ABC 123` (mã đọc to) và có `abc-123` | Dán vào ô | ô nhận (cắt khoảng trắng/gạch khi gửi); nút mở khoá; không tràn | TAY |  |
| TC-MOB-KBD-10 | US-AS-15 AC6 | A | Autofill/trình quản lý mật khẩu của Android bật | Chạm ô mã | không hộp gợi ý mật khẩu/địa chỉ (`autocomplete=off`) | TAY |  |
| TC-MOB-KBD-11 | US-AS-15 AC6 | A | Bàn phím phần cứng của emulator | Tab đến ô, gõ, Enter | Enter gửi; Tab theo thứ tự nhìn; focus thấy rõ | TAY |  |
| TC-MOB-KBD-12 | US-AS-15 AC6 | W | Playwright thiết bị `Pixel 7` (viewport thu khi bàn phím giả, `page.setViewportSize` 411×500) | Chạm ô mã | `scrollIntoView({block:center})` đưa ô vào giữa; nút `Điểm danh` còn trong khung nhìn thu hẹp; không lỗi | E2E |  |

## 6.L Back, vuốt quay lại (US-AS-15 AC7)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-NAV-01 | US-AS-15 AC7(a) | A (Back + cử chỉ) | Đang quét | Back (nút ba nút) rồi lần khác Back cử chỉ vuốt cạnh | thoát khung quét về màn nhập mã; camera tắt (đèn tắt); KHÔNG rời `/checkin` | TAY+ADB |  |
| TC-MOB-NAV-02 | US-AS-15 AC7(b) | A, I | Vừa «Đã điểm danh hh:mm» | Back / vuốt cạnh trái | về trang trước hoặc «Hôm nay»; KHÔNG lượt hai (1 dòng DB); KHÔNG lỗi cũ; địa chỉ không còn `#token`, `sessionStorage.ep.checkin` rỗng | TAY+SQL |  |
| TC-MOB-NAV-03 | US-AS-15 AC7(c) | A, I | Đăng nhập từ liên kết QR (SES-11) | Back sau đăng nhập | không về `/login` rồi vòng lại; ở Back lần hai thoát màn trước | TAY |  |
| TC-MOB-NAV-04 | US-AS-15 AC7(d) | A | Ở «Hôm nay» (màn gốc) | Back | đưa app xuống nền; KHÔNG đăng xuất; mở lại vào «Hôm nay» đã đăng nhập | TAY |  |
| TC-MOB-NAV-05 | US-AS-15 AC7(d) | A | Đang «Đang gửi» (mạng chậm) | Back ở màn gốc | không thoát hẳn; POST không bị hủy trước khi xong; mở lại thấy kết quả | TAY+SQL |  |
| TC-MOB-NAV-06 | US-AS-15 AC7 | A | Predictive back (Android 14+ nếu bật) | Vuốt Back chậm giữa chừng rồi thả | không bay màn lạ, không thoát nhầm; hành vi như Back thường | TAY |  |
| TC-MOB-NAV-07 | US-AS-15 AC7 | I | iPhone Safari | Vuốt từ cạnh trái khi ở `/checkin` sau điểm danh | về trang trước; không lượt hai; không lỗi cũ | TAY+SQL |  |
| TC-MOB-NAV-08 | US-AS-15 AC7; AC11 | P | PWA (không có nút Back của trình duyệt) | Từ `/checkin` đã điểm danh tìm đường quay lại | mọi màn có đường quay lại bằng điều hướng của khung (thanh dưới / thanh trên); không màn cụt | TAY |  |
| TC-MOB-NAV-09 | US-AS-15 AC7 | A | Nhập mã rồi Back | Back khi bàn phím mở, rồi khi bàn phím đóng | Back đầu đóng bàn phím; Back sau rời màn; mã không gửi | TAY |  |
| TC-MOB-NAV-10 | US-AS-15 AC7 | A, I | Đã điểm danh; mở lại cùng liên kết bằng Back | Back về trang QR rồi tiến | hiện «Đã điểm danh hh:mm» (lặp, không lỗi) | TAY |  |
| TC-MOB-NAV-11 | US-AS-15 AC7 | A | Mở `/checkin` từ thanh dưới / «Thêm» | Chuyển Hôm nay → Lịch → `/checkin` → Back ×3 | thứ tự Back đúng ngăn xếp; không vòng; thoát gốc → nền | TAY |  |
| TC-MOB-NAV-12 | US-AS-15 AC7 | W | Playwright (điều hướng lịch sử) | Sau «Đã điểm danh» `page.goBack()` | không POST lần hai (đếm request); không lỗi cũ; URL không có hash | E2E |  |

## 6.M Quyền camera / vị trí: từ chối rồi cấp lại (US-AS-15 AC8)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-PRM-01 | US-AS-11 AC3a; US-AS-15 AC8 | A | `pm reset-permissions` | Bấm `Quét mã điểm danh` → hộp quyền → «Từ chối» | về màn nhập mã; «Chưa cấp quyền camera.» (chữ amber) + ô «Mã điểm danh» + `Điểm danh`; nhập mã đúng → thành công, `channel=CODE` | TAY+SQL |  |
| TC-MOB-PRM-02 | US-AS-15 AC8 | A | Sau PRM-01 | Cài đặt hệ thống → Ứng dụng → EduPilot → Quyền → Máy ảnh → «Khi dùng ứng dụng»; quay lại app (Back / Recents) | KHÔNG đăng nhập lại, KHÔNG tải lại thủ công: khi trang hiện lại nút `Quét mã điểm danh` trở lại và dòng «Chưa cấp quyền camera.» biến mất ≤ 2 s; quét thành công | TAY+ADB |  |
| TC-MOB-PRM-03 | US-AS-15 AC8 | A | `pm reset-permissions`; API 35 | Bấm quét → Từ chối; lặp bấm quét 10 lần (nếu nút còn) | hộp hệ thống hiện ≤ 2 lần tổng cộng (Android chặn sau 2 lần); không vòng lặp; sau chặn chỉ còn ô nhập mã; mỗi lần bấm không treo | TAY+ADB |  |
| TC-MOB-PRM-04 | US-AS-15 AC8 | A | Camera bị chặn vĩnh viễn | Mở app lại; vào `/checkin` | chỉ ô nhập mã, không hiện lại hộp xin quyền; điểm danh bằng mã được; hướng dẫn cấp lại không dài dòng (luật chữ UI: không câu giải thích) — ghi nhận chữ hiện ra | TAY |  |
| TC-MOB-PRM-05 | US-AS-15 AC8; AC10 | A | App đang mở ở `/checkin` | Từ Cài đặt thu hồi quyền Máy ảnh khi app chạy (Android khởi động lại tiến trình) | mở lại app: không màn trắng, đăng nhập còn hạn, không crash; quét lại hỏi quyền hoặc về ô nhập mã | TAY+ADB |  |
| TC-MOB-PRM-06 | US-AS-15 AC8 | A | Chọn «Chỉ lần này» (one-time) cho camera | Quét xong; để app nền > 1 phút; quay lại quét lần hai | quyền hết hạn → hộp xin lại hoặc ô nhập mã (không treo, không lỗi kỹ thuật) | TAY |  |
| TC-MOB-PRM-07 | US-AS-15 AC8 | I, P | QCMOB2 geofence; `simctl privacy booted reset location` | Điểm danh → «Không cho phép»; vào Cài đặt → Quyền riêng tư → Dịch vụ định vị → Safari → «Khi dùng ứng dụng»; quay lại Safari | dòng «Lớp này kiểm tra vị trí khi điểm danh.» trước hộp; từ chối → `GEO_DENIED`, vẫn thành công; sau khi cấp lại, lượt sau có vị trí, không cần đăng nhập lại | TAY+SQL |  |
| TC-MOB-PRM-08 | US-AS-15 AC8 | I | Vị trí bị chặn vĩnh viễn (iOS) | Điểm danh 3 lần | không hộp lặp; `GEO_DENIED`; vẫn thành công | TAY+SQL |  |
| TC-MOB-PRM-09 | US-AS-15 AC8 | A | Android 12+: tắt công tắc quyền riêng tư «Truy cập máy ảnh» toàn máy (Cài đặt → Quyền riêng tư) | Bấm `Quét mã điểm danh` | «Không mở được camera.» + ô nhập mã; không treo | TAY |  |
| TC-MOB-PRM-10 | US-AS-10 AC1 | A | Quyền vị trí đã cấp; lớp tắt geofence | Điểm danh | không xin lại, không dòng thông báo vị trí, không cờ `GEO_*` | TAY+SQL |  |
| TC-MOB-PRM-11 | US-AS-11 AC1 | A | API 33+ | Mở app lần đầu và dùng 10 phút | KHÔNG có hộp xin quyền thông báo, lưu trữ, danh bạ, micro | TAY+ADB |  |
| TC-MOB-PRM-12 | US-AS-11 AC1; AC5 | A | Đã cấp camera | `adb shell dumpsys package vn.edupilot.app \| grep -E 'RECORD_AUDIO\|READ_\|WRITE_'` | không có quyền micro / bộ nhớ / danh bạ (chỉ INTERNET, CAMERA, ACCESS_FINE_LOCATION) | ADB |  |

## 6.N Mạng yếu, mất mạng, khôi phục (US-AS-15 AC9)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-NET-01 | US-AS-15 AC9 | A | SV đăng nhập; mạng chậm R-NET-slow (≈ 400 kbit/s, RTT 2 s) | Mở `/checkin` (nav) | Có phản hồi ngay khi chạm: khung xương/chờ (`Skeleton`, `aria-busy`) hiện thay vì màn trắng; nội dung cuối cùng hiện ra hoặc câu lỗi có «Thử lại» (không treo vô hạn); không cuộn ngang. **Không chấm theo ngưỡng thời gian** (bỏ 3 s/8 s — P-11-143, US-AS-15 AC9 chỉ yêu cầu hành vi) | TAY |  |
| TC-MOB-NET-02 | US-AS-09 AC4; US-AS-15 AC9 | A | Đang ở ô nhập mã đã gõ mã đúng | Bật chế độ máy bay → `Điểm danh` | ≤ 15 s hiện «Chưa có mạng.» + `Thử lại`; dải offline (OfflineBanner) hiện; mã đã gõ giữ nguyên | TAY |  |
| TC-MOB-NET-03 | US-AS-15 AC9 | A | Sau NET-02 | Tắt chế độ máy bay; bấm `Thử lại` | dải offline biến mất ≤ 3 s sau khi có mạng; `Thử lại` thành công «Đã điểm danh hh:mm»; một lượt; không tải lại trang | TAY+SQL |  |
| TC-MOB-NET-04 | US-AS-15 AC9 | A | Mạng chập chờn (R-NET-flaky: bật/tắt 5 lần trong 20 s) khi quét QR | Quét Q-OK | không đăng xuất, không màn lỗi, không lượt thứ hai; cuối cùng thành công hoặc «Chưa có mạng.» + `Thử lại` | TAY+SQL |  |
| TC-MOB-NET-05 | US-AS-15 AC9 | A | Đang gửi (mạng chậm) | Chuyển Wi-Fi → dữ liệu di động giữa chừng (`svc wifi disable`) | kết nối lại, kết quả đúng một lượt; không hộp lỗi kỹ thuật | TAY+SQL |  |
| TC-MOB-NET-06 | US-AS-11 AC6; US-AS-15 AC14 | A | Máy chủ `server.url` tắt hoặc chặn bằng R-NET-block-host khi app đang mở và điều hướng sang trang khác | Chạm sang «Lịch» | trang lỗi chuẩn cục bộ tiếng Việt + `Thử lại` (không trang lỗi WebView); sau khi chặn gỡ, `Thử lại` vào lại | TAY |  |
| TC-MOB-NET-07 | US-AS-15 AC9 | A | Proxy làm chậm trả lời `POST /checkins` 20 s | Gửi mã | sau 15 s: «Chưa có mạng.» + `Thử lại`; nếu máy chủ đã ghi lượt, `Thử lại` trả `200` «Đã điểm danh hh:mm»; một dòng DB | TAY+SQL |  |
| TC-MOB-NET-08 | US-AS-15 AC9 | A | Quét QR; mạng cắt NGAY sau khi thấy QR; token hết hạn trong lúc chờ | Chờ > 2 bước rotate rồi bật mạng, bấm `Thử lại` | «Mã đã hết hạn. Quét lại mã trên màn chiếu.» + nút `Quét mã điểm danh` ở lại; không đăng xuất | TAY |  |
| TC-MOB-NET-09 | US-AS-15 AC9 | I | Safari đã mở `/checkin`; cắt mạng host (Network Link Conditioner 100 % loss) | Nhập mã → `Điểm danh` | «Chưa có mạng.» + `Thử lại`; bật mạng, `Thử lại` thành công | TAY+SQL |  |
| TC-MOB-NET-10 | US-AS-15 AC9; AC11 | P | PWA đã đăng nhập, mạng host cắt | Mở PWA từ biểu tượng khi offline; rồi bật mạng | ghi nhận màn hiện ra (iOS không có service worker); không treo vô hạn quá 15 s; có mạng lại thì mở lại vào thẳng; không mất phiên | TAY |  |
| TC-MOB-NET-11 | US-AS-15 AC9 | A | Android «Tiết kiệm dữ liệu» bật | Điểm danh bằng quét và mã | hoạt động bình thường; camera không bị chặn | TAY |  |
| TC-MOB-NET-12 | US-AS-15 AC9 | A, I | Mạng yếu (R-NET-slow) và quét nhiều lần cùng QR | Quét lại QR ngay sau khi nhận (trong lúc chờ) | đúng 1 POST cho mỗi token (đếm ở gateway log và ở DB); lượt sau là `200` cùng kết quả | TAY+SQL |  |

## 6.O Nền → mở lại, hệ thống giết tiến trình, vòng đời (US-AS-15 AC10)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-LIF-01 | US-AS-15 AC10 | A | Mạng chậm R-NET-slow; vừa quét xong, đang «Đang gửi» | Home (nền) → mở lại trong 20 s | hiện đúng kết quả («Đã điểm danh hh:mm» hoặc câu lỗi), KHÔNG gửi lại (đúng 1 POST) | TAY+SQL |  |
| TC-MOB-LIF-02 | US-AS-15 AC10 | A | Đang gửi (chậm); Staff đóng cửa sổ trong lúc app nền | Mở lại sau khi cửa sổ đóng | hiện «Đã đóng điểm danh.» (hoặc «Đã điểm danh hh:mm» nếu lượt kịp ghi); không treo | TAY+SQL |  |
| TC-MOB-LIF-03 | US-AS-15 AC6; AC10 | A | Đã gõ nửa mã ở ô | Khoá màn hình 30 s, mở khoá | mã đã gõ còn nguyên; ô và nút dùng được; không màn trắng | TAY |  |
| TC-MOB-LIF-04 | US-AS-15 AC10 | A | Đang quét | Cuộc gọi đến (`adb emu gsm call 0901234567`), nghe rồi cúp (`gsm cancel`) | camera tắt khi cuộc gọi tới; quay lại về màn nhập mã với `Quét mã điểm danh`; không treo | TAY+ADB |  |
| TC-MOB-LIF-05 | US-AS-15 AC10 | A | App ở nền; phiên còn hạn | `adb shell am kill vn.edupilot.app` (hệ thống giết nền); mở lại từ Recents | vào «Hôm nay» hoặc màn đăng nhập với phiên còn hạn → vào thẳng; không màn trắng, không mất đăng nhập còn hạn | TAY+ADB |  |
| TC-MOB-LIF-06 | US-AS-15 AC10 | A | Đang ở `/checkin`, ô có mã dở | `am send-trim-memory vn.edupilot.app COMPLETE` rồi `am kill`; mở lại | không crash; về `/checkin` hoặc «Hôm nay»; phiên giữ; ghi nhận mã dở có mất không | TAY+ADB |  |
| TC-MOB-LIF-07 | US-AS-15 AC10 | A | Màn hình tắt 5 phút | `dumpsys deviceidle force-idle`; mở lại và điểm danh | phiên làm mới (một `POST /auth/refresh`); điểm danh bình thường | TAY+ADB |  |
| TC-MOB-LIF-08 | US-AS-15 AC10 | A | Mở `/checkin`, chuyển sang app khác 10 phút | Quay lại bằng Recents rồi nhập mã mới | phiên còn hoặc tự làm mới; điểm danh được; không màn trắng | TAY |  |
| TC-MOB-LIF-09 | US-AS-15 AC10 | I | Safari tab nền 5 phút | Quay lại tab, nhập mã | trang còn hoặc tải lại êm; không lượt hai; không màn trắng | TAY+SQL |  |
| TC-MOB-LIF-10 | US-AS-15 AC10; AC11 | P | PWA đã đăng nhập | Vuốt lên thoát PWA, mở lại từ biểu tượng | vào thẳng đã đăng nhập hoặc về đăng nhập (ghi nhận); không màn trắng/đen; không vòng lặp | TAY |  |
| TC-MOB-LIF-11 | US-AS-15 AC10 | A | Chế độ tiết kiệm pin bật | Quét QR | quét vẫn chạy (≥ 3 khung/giây ghi nhận), nhận QR ≤ 5 s | TAY |  |
| TC-MOB-LIF-12 | US-AS-15 AC14 | A, I | Đổi ngôn ngữ hệ thống sang English | Mở app / Safari tới `/checkin` | giao diện vẫn tiếng Việt (`lang=vi`); không hộp «dịch trang»; bố cục không vỡ | TAY |  |
| TC-MOB-LIF-13 | US-AS-15 AC10 | A | Máy bộ nhớ thấp (`am send-trim-memory … COMPLETE` lặp, nhiều app nền) | Mở camera quét | không crash; nếu camera không mở → «Không mở được camera.» + ô nhập mã | TAY+ADB |  |

## 6.P PWA trên iPhone Safari (US-AS-09 AC6, US-AS-15 AC11)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-PWA-01 | US-AS-09 AC6; US-AS-15 AC11 | W, I | `https://{VPS}/manifest.webmanifest` | `curl -s` + `jq`; tải từng icon | JSON hợp lệ: `name="EduPilot"`, `short_name`, `start_url="/"`, `scope="/"`, `display="standalone"`, `lang="vi"`, `theme_color`, `background_color`; `icons` 192, 512 PNG trả 200 và đúng kích thước; không `serviceWorker` đăng ký (`navigator.serviceWorker.getRegistrations()` rỗng) | E2E |  |
| TC-MOB-PWA-02 | US-AS-15 AC11 | I | Safari iPhone 15 Simulator; trang `https://{VPS}/` | Chia sẻ → «Thêm vào Màn hình chính» (xem hộp tên) | tên đề xuất «EduPilot»; không lời mời cài đặt tự hiện (không banner, không hộp) khi duyệt | TAY |  |
| TC-MOB-PWA-03 | US-AS-15 AC11 | I, IR | Sau PWA-02 | Quan sát biểu tượng trên Màn hình chính (chụp) | biểu tượng là logo EduPilot (không phải ảnh chụp trang/chữ cái đầu); nền đúng, không viền trắng thừa, không bị cắt; nếu iOS lấy ảnh chụp trang vì không có `apple-touch-icon`, ghi F (kiểm `<link rel=apple-touch-icon>` trong HTML: quan sát tĩnh 25/11: `layout.tsx` chỉ khai `icons.icon` = favicon.svg, manifest có icon PNG 192/512) | SIMCTL+TAY |  |
| TC-MOB-PWA-04 | US-AS-15 AC11 | P | Mở PWA từ biểu tượng | Quan sát | chạy TOÀN MÀN HÌNH: không thanh địa chỉ, không thanh công cụ Safari; `matchMedia('(display-mode: standalone)').matches = true` (hoặc `navigator.standalone = true`) | TAY |  |
| TC-MOB-PWA-05 | US-AS-15 AC11; AC3 | P | PWA mở ở «Hôm nay» và `/checkin` | Đọc ảnh vùng trên | thanh trạng thái đọc được (giờ, pin không chìm vào nền) và không che tiêu đề; không khoảng trống đen/trắng lệch màu canvas | SIMCTL+TAY |  |
| TC-MOB-PWA-06 | US-AS-15 AC11 | P | PWA lần mở đầu | Quan sát màn khởi động và màn đầu | KHÔNG màn đen/trắng kéo dài > 1,5 s trước khung; nếu có splash do iOS sinh thì không méo/không logo lạ; ghi thời gian từ chạm → thấy giao diện | TAY |  |
| TC-MOB-PWA-07 | US-AS-15 AC11 | P | Safari đã đăng nhập `sv.gioi`; PWA vừa thêm | Mở PWA | phải đăng nhập lại trong PWA (kho lưu trữ tách khỏi Safari); không lỗi lạ; sau đăng nhập vào «Hôm nay» | TAY |  |
| TC-MOB-PWA-08 | US-AS-15 AC11; US-AS-09 AC10 | P | PWA đã đăng nhập; W mở | Mở `/checkin` trong PWA | KHÔNG nút quét (web không quét trong trang); chỉ ô «Mã điểm danh» + `Điểm danh`; nhập mã đúng → «Đã điểm danh hh:mm» | TAY+SQL |  |
| TC-MOB-PWA-09 | US-AS-15 AC11 | P + Safari | W mở; liên kết quét `U` | Mở `U` bằng `simctl openurl` khi PWA đã cài | liên kết mở SAFARI (không vào PWA); lượt web; `channel=WEB`; PWA không tự nổi lên | SIMCTL+SQL |  |
| TC-MOB-PWA-10 | US-AS-15 AC17; US-AS-04 AC5 | P + Safari | `sv.gioi`: lượt 1 trong PWA (ô nhập mã), lượt 2 bằng QR camera hệ thống (Safari) | Điểm danh hai lần ở hai vòng; đọc `user_devices`, `attendance_checkins`, lưới Staff | hai `device_id` khác nhau (kho PWA ≠ kho Safari); mỗi khoá `DEVICE_NEW` MỘT lần; cùng `user_id` ⇒ KHÔNG `DEVICE_SHARED`; cờ chỉ Staff thấy, SV không thấy, không chặn | TAY+SQL |  |
| TC-MOB-PWA-11 | US-AS-15 AC11 | P | PWA | Tìm nút Back / vuốt cạnh trái ở nhiều màn | không nút Back của trình duyệt; mọi màn có đường quay lại bằng điều hướng của khung; không màn cụt | TAY |  |
| TC-MOB-PWA-12 | US-AS-09 AC6 | P, I | Duyệt Safari 10 phút ở `/checkin` | Quan sát: banner «Thêm vào Màn hình chính», `beforeinstallprompt` (Android Chrome) | KHÔNG lời mời cài đặt nào do app đưa ra (đếm = 0) | TAY |  |
| TC-MOB-PWA-13 | US-AS-09 AC6; US-AS-15 AC17 | P (+ iPhone thật) | PWA đã cài; `sv.gioi` đã điểm danh MỘT lần trong PWA (có `device_id` trong DB); chờ hơn 7 ngày KHÔNG mở Safari nhưng có mở PWA (hoặc iPhone thật dùng sẵn) | Điểm danh lần nữa TRONG PWA (nhập mã); đối chiếu `device_id` | khoá trong PWA còn: cùng `device_id`, KHÔNG `DEVICE_NEW` (kể cả sau > 7 ngày); không kiểm được trong lượt chạy ngắn (Simulator không đổi giờ) → KHÔNG KIỂM ĐƯỢC (thời gian), không tính F; WebKit Playwright chỉ kiểm `navigator.storage.persist` được gọi (PWA-22) | TAY+SQL | 8 ngày / iPhone thật |
| TC-MOB-PWA-14 | US-AS-15 AC11; AC3 | P | PWA, `viewport-fit` chưa đặt | Cuộn trang dài (Hôm nay) và `/checkin` có bàn phím | thanh dưới và dock trên thanh Home; bàn phím không đẩy bố cục vỡ; overscroll (kéo xuống) không để lộ nền đen | TAY |  |
| TC-MOB-PWA-15 | US-AS-15 AC11 | P | PWA; xoay ngang | Mở `/checkin` ngang | bố cục dùng được; nút không bị Dynamic Island/Home che (xem SAF-07) | TAY |  |
| TC-MOB-PWA-16 | US-AS-09 AC6 | W | Phản hồi header của `/manifest.webmanifest` | `curl -sI` | `Content-Type` manifest hợp lệ; không `Cache-Control` làm kẹt bản cũ quá 1 ngày; HTTPS | E2E |  |
| TC-MOB-PWA-17 | US-AS-15 AC17 | P | PWA mới cài + đăng nhập, CHƯA điểm danh trong PWA | Safari Web Inspector (Develop → Simulator → PWA) → Storage → IndexedDB; sau đó điểm danh bằng ô nhập mã | trước khi điểm danh: KHÔNG có khoá thiết bị trong IndexedDB của PWA; sau lượt đầu: có khoá ECDSA (không xuất được), `DEVICE_NEW` một lần | TAY+SQL |  |
| TC-MOB-PWA-18 | US-AS-15 AC17 | P | Sau PWA-17 | Điểm danh vòng 2 trong PWA | cùng `device_id` với lượt đầu; không `DEVICE_NEW` | TAY+SQL |  |
| TC-MOB-PWA-19 | US-AS-15 AC17 | P + Safari | PWA đã có khoá (PWA-18) | Quét QR bằng camera hệ thống → Safari điểm danh (vòng sau) | dùng khoá của SAFARI (kho khác): `DEVICE_NEW` một lần cho khoá đó, lượt Safari sau không; khoá PWA không bị dùng; khoá Safari không chuyển vào PWA | TAY+SQL |  |
| TC-MOB-PWA-20 | US-AS-15 AC17; US-AS-04 AC4 | P + Safari | Sau PWA-19 (hai khoá cùng `user_id`) | Staff xem lưới `/attendance` và danh sách lượt | KHÔNG `DEVICE_SHARED` cho hai khoá cùng một sinh viên; chỉ `DEVICE_NEW` (cờ Staff, không chặn) | TAY+SQL |  |
| TC-MOB-PWA-21 | US-AS-15 AC17 | P + Safari | Điểm danh ở Safari TRƯỚC khi cài PWA (khoá Safari có), sau đó cài PWA | Mở PWA, đăng nhập, điểm danh | PWA tạo khoá MỚI (không thừa hưởng khoá Safari): `DEVICE_NEW` một lần cho PWA; hai `device_id` khác nhau | TAY+SQL |  |
| TC-MOB-PWA-22 | US-AS-15 AC17; US-AS-09 AC3 | W (WebKit) | Playwright dự án WebKit | Mở `/checkin`, điểm danh, kiểm `navigator.storage.persist` đã được gọi (bọc hàm bằng `addInitScript`) | `persist()` được gọi đúng một lần sau khi tạo khoá; không lỗi nếu bị từ chối | E2E |  |
| TC-MOB-PWA-23 | US-AS-15 AC17 | IR | iPhone thật: Safari không mở > 7 ngày | Quét QR (Safari) điểm danh | có thể `DEVICE_NEW` lại ở Safari (iOS xoá kho sau 7 ngày không mở Safari); không chặn, vẫn 201; PWA không bị ảnh hưởng (PWA-18) | TAY+SQL | iPhone thật, 8 ngày |

## 6.Q Trợ năng: TalkBack, VoiceOver, axe (US-AS-15 AC12)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-A11-01 | US-AS-15 AC12 | W | Playwright + `@axe-core/playwright` ở 360 và 430 px | Chạy axe trên `/checkin` ở 6 trạng thái và `/attendance` Staff | 0 lỗi nghiêm trọng / nghiêm trọng cao (critical, serious); báo cáo mọi lỗi nhẹ | AXE |  |
| TC-MOB-A11-02 | US-AS-15 AC12 | A (TalkBack) | TalkBack bật (nếu AVD có; không có → thay bằng `uiautomator dump` ghi `content-desc`/`text` và kiểm tay sau) | Mở `/checkin` (nhập mã); vuốt phải qua mọi phần tử | `h1` «Điểm danh» đọc ĐẦU; thứ tự đọc theo thứ tự nhìn (tiêu đề → ô mã → «Lớp {mã lớp}» → nút `Điểm danh`); không phần tử thừa đọc hai lần | TAY | TalkBack |
| TC-MOB-A11-03 | US-AS-15 AC12 | A (TalkBack) | Ô «Mã điểm danh» | Chạm ô khi TalkBack | đọc «Mã điểm danh» (nhãn thật, không chỉ placeholder «ABC 123»); báo «ô soạn thảo» đúng; khi lỗi đọc câu lỗi (`aria-describedby`) | TAY | TalkBack |
| TC-MOB-A11-04 | US-AS-15 AC12 | A (TalkBack) | Nút `Quét mã điểm danh`, `Thoát`, `Nhập mã`, `Điểm danh`, `Thử lại` | Chạm từng nút | mỗi nút có tên nói được («Quét mã điểm danh», «Thoát», …); nút khoá đọc «không khả dụng» kèm lý do «Thử lại sau {n} giây.» | TAY | TalkBack |
| TC-MOB-A11-05 | US-AS-15 AC12 | A (TalkBack) | Đang quét | Chạm khung camera; kích hoạt `Thoát` bằng cử chỉ hai lần chạm | khung đọc «Camera quét mã điểm danh»; `Thoát` dùng được; camera tắt | TAY | TalkBack |
| TC-MOB-A11-06 | US-AS-15 AC12 | A (TalkBack) | Gửi mã đúng | Quan sát lời đọc | thông báo MỘT lần «Đang gửi» (`polite`), rồi «Đã điểm danh hh:mm» (`polite`); không đọc trùng | TAY | TalkBack |
| TC-MOB-A11-07 | US-AS-15 AC12 | A (TalkBack) | Mã sai | Gây lỗi | câu lỗi được đọc ngay (`role=alert`), một lần; không đọc lại khi cuộn | TAY | TalkBack |
| TC-MOB-A11-08 | US-AS-15 AC12 | A (TalkBack) | 429 đếm ngược | Gây 429 | thông báo lúc bắt đầu «Thử lại sau {n} giây.» và lúc hết giờ; KHÔNG đọc mỗi giây (đếm số lời đọc trong 10 s ≤ 2) | TAY | TalkBack |
| TC-MOB-A11-09 | US-AS-15 AC12 | W, A | Dấu `Check` đỏ | Kiểm DOM | `Check` có `aria-hidden`; chữ «Đã điểm danh hh:mm» đọc đủ; trạng thái không chỉ truyền bằng màu (lỗi có chữ + biểu tượng) | E2E+TAY |  |
| TC-MOB-A11-10 | US-AS-15 AC12 | I (VoiceOver — simulator bật VoiceOver bằng phím tắt hoặc iPhone thật) | VoiceOver bật | Safari `/checkin` nhập mã: duyệt bằng vuốt phải | như A11-02…08 (`h1` đầu; ô có nhãn; nút có tên; kết quả đọc một lần; 429 không đọc mỗi giây) | TAY | iPhone thật (Simulator giới hạn) |
| TC-MOB-A11-11 | US-AS-15 AC12 | P | VoiceOver trong PWA | Duyệt `/checkin` và thanh dưới | như A11-10; thanh dưới có tên «Hôm nay», «Chat riêng», «Hỏi đáp», «Luyện đề», «Thêm» | TAY | iPhone thật |
| TC-MOB-A11-12 | US-AS-15 AC12 | A, I | Thanh dưới của khung | Duyệt bằng trình đọc | mỗi mục có tên chữ; mục đang chọn được đọc là «đã chọn»/`aria-current`; không mục chỉ là biểu tượng không nhãn | TAY |  |
| TC-MOB-A11-13 | US-AS-15 AC12 | A | Bàn phím phần cứng / Switch Access | Tab qua `/checkin` | thứ tự focus theo thứ tự nhìn; viền focus thấy rõ; không bẫy focus ở khung quét (Esc/`Thoát` thoát được) | TAY |  |
| TC-MOB-A11-14 | US-AS-15 AC12 | W | `prefers-reduced-motion: reduce` | Mở `/checkin` các trạng thái | không hoạt ảnh gây chóng mặt; khung xương không nhấp nháy mạnh; chuyển trạng thái tức thời | E2E |  |
| TC-MOB-A11-15 | US-AS-15 AC12; AC4 | A | Android «Hiển thị có độ tương phản cao» / «Thu phóng» bật | Duyệt `/checkin` | không vỡ bố cục; chữ đọc được | TAY |  |
| TC-MOB-A11-16 | US-AS-15 AC12 | W | `lang`, tiêu đề tài liệu | Đọc `<html lang>` và `<title>` | `lang=vi`; `<title>` có «Điểm danh · EduPilot»; trình đọc phát âm tiếng Việt | E2E |  |

## 6.R Vùng chạm ≥ 44 px (US-AS-15 AC13)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-TGT-01 | US-AS-15 AC13 | W | 360 px và 430 px; 6 trạng thái `/checkin` | Đo `getBoundingClientRect()` mọi `button, a, input` trong `main`, thanh trên và thanh dưới | mọi điều khiển cao và rộng ≥ 44 px (đo hộp bấm, kể cả `Thoát`, `Nhập mã`, `Thử lại`); báo cáo bảng tên → w×h | E2E |  |
| TC-MOB-TGT-02 | US-AS-15 AC13 | W | 360 px | Đo khoảng cách giữa hai điều khiển kề nhau (nút + nút, mục nav + mục nav) | khoảng cách ≥ 8 px; không hai điều khiển chạm hoặc đè nhau | E2E |  |
| TC-MOB-TGT-03 | US-AS-15 AC13 | W, A | Dock (nút chính) và thanh dưới | Đo `dock.bottom` và `nav.top` | dock không đè lên thanh dưới (dock.bottom ≤ nav.top); cả hai ≥ 44 px | E2E+TAY |  |
| TC-MOB-TGT-04 | US-AS-15 AC13 | A | Android `uiautomator dump` | Dump cây giao diện ở 5 màn; đọc `bounds` của mọi nút | ≥ 44 × 44 px thực (đổi sang CSS px theo mật độ); danh sách vi phạm nếu có | ADB |  |
| TC-MOB-TGT-05 | US-AS-15 AC13 | W | `/attendance` Staff ở 360 px: hàng có hành động (đánh vắng, mở cửa sổ, lọc) | Đo mọi hàng có hành động | vùng bấm ≥ 44 × 44 px; hàng kề không chồng | E2E |  |
| TC-MOB-TGT-06 | US-AS-15 AC13 | W, A (R-ROT-land) | Ngang ≥ 720 px (khung desktop với thiết bị chạm) | Đo `Thoát`, `Nhập mã`, `Thử lại`, mục thanh bên | vẫn ≥ 44 × 44 khi `pointer: coarse` (theo `Button.module.css`); báo cáo nếu < 44 (dự đoán từ CSS: đã phủ bằng `(pointer: coarse)`) | E2E+TAY |  |

## 6.S Chữ lỗi tiếng Việt, không từ kỹ thuật (US-AS-15 AC14)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-MSG-01 | US-AS-15 AC14; US-AS-09 AC4 | A, I, W | Mã hết hạn | Gây lỗi bằng liên kết hết hạn | «Mã đã hết hạn. Quét lại mã trên màn chiếu.» (đúng từng chữ, dấu) | E2E+TAY |  |
| TC-MOB-MSG-02 | US-AS-15 AC14 | A, I, W | MAC sai | Gây lỗi | «Mã không đúng. Quét lại mã trên màn chiếu.» | E2E+TAY |  |
| TC-MOB-MSG-03 | US-AS-15 AC14 | A, I, W | Cửa sổ đóng | Gây lỗi | «Đã đóng điểm danh.» | E2E+TAY |  |
| TC-MOB-MSG-04 | US-AS-15 AC14 | A, I, W | Lớp tắt `self_checkin` | Gây lỗi | «Lớp này không điểm danh bằng mã.» | E2E+TAY |  |
| TC-MOB-MSG-05 | US-AS-15 AC14 | A, I, W | SV ngoài lớp | Gây lỗi | «Bạn không thuộc lớp này.» | E2E+TAY |  |
| TC-MOB-MSG-06 | US-AS-15 AC14 | A, I, W | Quá hạn mức | Gây 429 | «Thử lại sau {n} giây.» ({n} số nguyên ≥ 1, đếm xuống) | E2E+TAY |  |
| TC-MOB-MSG-07 | US-AS-15 AC14 | A, I, W | Mất mạng | Gây lỗi | «Chưa có mạng.» + nút `Thử lại` | E2E+TAY |  |
| TC-MOB-MSG-08 | US-AS-15 AC14; US-AS-11 AC3a | A | Quyền camera bị từ chối | Từ chối | «Chưa cấp quyền camera.» (không `NotAllowedError`) | TAY |  |
| TC-MOB-MSG-09 | US-AS-15 AC14 | A | Camera lỗi hoặc đang bị app khác dùng | Gây lỗi (SCN-12) | «Không mở được camera.» + ô nhập mã (không `NotReadableError`/`NotFoundError`) | TAY |  |
| TC-MOB-MSG-10 | US-AS-15 AC14 | A | Không có `BarcodeDetector` | Mở `/checkin` | KHÔNG câu nào; ô nhập mã hiện ngay | TAY |  |
| TC-MOB-MSG-11 | US-AS-15 AC14 | A | WebView không tải được trang (máy chủ tắt / không mạng) | Mở app khi máy chủ tắt | trang lỗi chuẩn tiếng Việt «Không kết nối được máy chủ.» + `Thử lại` (hoặc câu BA chốt trong US-AS-15); KHÔNG trang lỗi WebView (ERR_*) | TAY |  |
| TC-MOB-MSG-12 | US-AS-15 AC14; US-AS-09 AC10 | A, I, W | Mã ngắn sai / hết hạn | Gây lỗi | «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» | E2E+TAY |  |
| TC-MOB-MSG-13 | US-AS-15 AC14; US-AS-09 AC4 | A, I, W | Liên kết không phải UUID | Gây lỗi | «Liên kết không đúng.» | E2E+TAY |  |
| TC-MOB-MSG-14 | US-AS-15 AC14 | W | Lỗi 5xx / thân rỗng / HTML từ máy chủ | Chặn route trả 500 | «Chưa điểm danh được. Thử lại.»; không mã HTTP, JSON, `undefined` | E2E |  |
| TC-MOB-MSG-15 | US-AS-15 AC14 | W, A, I | Quét văn bản của mọi trạng thái `/checkin` (DOM `innerText` + `aria-label`) | Duyệt 6 trạng thái và mọi lỗi ở MSG-01…14; tìm chuỗi cấm: `CHECKIN_`, `NotAllowedError`, `NotFoundError`, `NotReadableError`, `ERR_`, `net::`, `undefined`, `null`, `[object`, `{`, mã HTTP 3 chữ số, chữ tiếng Anh như `Error`, `Failed`, `Forbidden` | 0 lần xuất hiện; 100 % câu có dấu tiếng Việt, mỗi lỗi MỘT câu; không `title`/tooltip tiếng Anh | E2E |  |
| TC-MOB-MSG-16 | US-AS-15 AC14 | A | Cỡ chữ lớn | Xem MSG-01…14 ở chữ 2,0× | các câu không bị cắt cụt giữa chữ | TAY |  |

## 6.T Bảo mật: token, khoá, log, WebView, liên kết (luật cứng 4, 5; US-AS-11 AC1, AC5, AC7)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-SEC-01 | US-AS-09 AC9; US-AS-03 AC10 | A, I | W mở; điểm danh qua quét và liên kết | Ghi mọi request (proxy / `chrome://inspect` KHÔNG dùng — dùng gateway log + `adb logcat`); tìm `T` | `T` không có trong URL, `Referer`, header; chỉ trong thân `POST` | ADB |  |
| TC-MOB-SEC-02 | US-AS-09 AC2; AC9 | A, I | Sau điểm danh và sau 61 s | Đọc `sessionStorage.ep.checkin`, `localStorage`, `IndexedDB`, cookie (Web Inspector Safari / `chrome://inspect` ở build debug hoặc Playwright) | `ep.checkin` xoá sau khi dùng hoặc sau 60 s; `localStorage` KHÔNG chứa token truy cập/làm mới/QR (chỉ `ep.device.android` = id khoá); IndexedDB chỉ chứa khoá thiết bị web (không xuất được) | E2E+TAY |  |
| TC-MOB-SEC-03 | US-AS-11 AC1 | A | App release | `adb logcat -d \| grep -iE 'access_token\|refresh\|ep_rt\|signature\|public_key\|Bearer\|token='` | 0 dòng chứa bí mật; log `chromium`/`Capacitor` không in thân yêu cầu | ADB |  |
| TC-MOB-SEC-04 | US-AS-11 AC1 | A | App release | `adb shell cat /proc/net/unix \| grep webview_devtools_remote`; thử `chrome://inspect` từ Chrome desktop | không socket gỡ lỗi WebView (APK không `debuggable`, WebView debugging tắt); không kết nối được | ADB |  |
| TC-MOB-SEC-05 | US-AS-11 AC1 | A | App release | `adb shell run-as vn.edupilot.app ls`; `adb backup` (nếu còn) / `dumpsys package \| grep ALLOW_BACKUP` | `run-as` bị từ chối; `ALLOW_BACKUP` không có; không thể sao lưu dữ liệu app | ADB |  |
| TC-MOB-SEC-06 | US-AS-11 AC1 | A | App | `APP_SERVER_URL` HTTPS; thử trang `http://…` (đổi thử một APK debug với `cleartext` mặc định hoặc mở `http://10.0.2.2` trong app qua liên kết) | HTTP thường KHÔNG tải (`cleartext=false`, `allowMixedContent=false`); hiện trang lỗi tiếng Việt; không cảnh báo chứng chỉ bị bỏ qua | TAY+ADB |  |
| TC-MOB-SEC-07 | US-AS-11 AC5 | W | Header của `https://{VPS}/checkin` | `curl -sI` | `Content-Security-Policy` không `unsafe-eval` và không nguồn script ngoài tên miền; `Permissions-Policy: camera=(self), geolocation=(self)`; `Referrer-Policy: no-referrer`; `Cache-Control: no-store`/`private, no-cache`; HSTS có | E2E |  |
| TC-MOB-SEC-08 | US-AS-11 AC1 | A | VPS HTTPS thật | `openssl s_client -connect {VPS}:443 -servername {VPS}`; Chrome emulator mở `https://{VPS}` | chuỗi chứng chỉ hợp lệ (CA công khai, chưa hết hạn, tên khớp); app không báo lỗi SSL; TLS ≥ 1.2 | ADB |  |
| TC-MOB-SEC-09 | US-AS-11 AC1 | A | App | `adb shell am start -a android.intent.action.VIEW -d vn.edupilot.app://checkin` | KHÔNG có trình xử lý (manifest không có intent filter cho scheme); không mở app theo liên kết lạ (tránh chiếm quyền liên kết) | ADB |  |
| TC-MOB-SEC-10 | US-AS-11 AC1 | A | Manifest | `aapt2 dump xmltree` — kiểm `exported` | chỉ `MainActivity` exported (launcher); `FileProvider` `exported=false`; không service/receiver exported thừa | ADB |  |
| TC-MOB-SEC-11 | US-AS-12 AC7; US-AS-04 | A | Sau khi đăng ký thiết bị | SQL `user_devices` | chỉ khoá công khai + băm; không IMEI, serial, nhãn; `label` NULL | SQL |  |
| TC-MOB-SEC-12 | US-AS-12 AC7; US-AS-09 AC3 | I, P | Safari / PWA | Thử `crypto.subtle.exportKey` với khoá thiết bị (qua bản kiểm của dev hoặc Playwright cho Chromium) | khoá ECDSA `extractable=false` → xuất bị từ chối; khoá lưu trong IndexedDB (một khoá cho cả trình duyệt) | E2E |  |
| TC-MOB-SEC-13 | US-AS-11 AC4 | A, AR | Plugin `EduPilotAttest` | Gọi `sign` hai lần rồi đọc chữ ký DER; kiểm khoá Keystore (`KeyInfo.isInsideSecureHardware` qua log kiểm của dev hoặc chuỗi chứng thực) | khoá không xuất được (`PURPOSE_SIGN` duy nhất); trên máy thật nằm trong TEE/StrongBox; emulator: phần mềm (ghi nhận, không F) | TAY | Keystore phần cứng chỉ máy thật |
| TC-MOB-SEC-14 | US-AS-11 AC7; US-AS-12 AC8 | A | `sv.gioi`; token SV | `curl` `POST /me/devices/challenge` bằng token TA/GV/Admin và không token | TA/GV/Admin → 403; không JWT → 401; SV chỉ cho CHÍNH mình (nonce theo `uid` từ JWT) | API |  |
| TC-MOB-SEC-15 | US-AS-03 AC9; luật cứng 4 | A | `sv.gioi` | `POST …/checkins` với `student_id`/`user_id` trong thân | 422 `unknown`, không ghi; giao diện không thể gửi | API |  |
| TC-MOB-SEC-16 | luật cứng 5 | A, I | Sau khi chạy mọi nhóm GEO/ERR/FLW | `pg_dump` `attendance_checkins`; `grep` log gateway 24 h | không toạ độ, IP, tên máy trong DB; log không có `token`, toạ độ, IP, `public_key`, `signature` | SQL+ADB |  |
| TC-MOB-SEC-17 | US-AS-09 AC9 | I, A | Nhấn giữ liên kết/QR; chia sẻ liên kết điểm danh qua tin nhắn tới SV khác | SV khác (không có mã) mở liên kết cũ đã hết hạn | chỉ «Mã đã hết hạn. …»; không điểm danh hộ (đúng ngữ nghĩa: token ngắn hạn, danh tính từ JWT người mở) | TAY+SQL |  |
| TC-MOB-SEC-18 | US-AS-11 AC7 | A | Cookie làm mới | Kiểm `Set-Cookie` của `/auth/login` (`curl -i`) và `document.cookie` trong trang | cookie `ep_rt` có `HttpOnly; Secure; SameSite=Lax`; `document.cookie` KHÔNG thấy `ep_rt`; token truy cập chỉ trong bộ nhớ (không `localStorage`) | API+E2E |  |

## 6.U Thiết bị và chứng thực khoá (US-AS-12; chỉ máy thật mới chứng minh mức 3)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-DEV-01 | US-AS-12 AC1; AC-5 | AR | Máy thật Android ≥ 9, bootloader khoá, APK ký đúng khoá của chủ dự án | Lần đầu mở app điểm danh; đọc gateway log và `user_devices` | `POST /me/devices/challenge` 200 → `POST /me/devices` 201; `user_devices` có `kind=ANDROID_KEY`, `attested=true`, `boot_verified=true`, `security_level` TEE/StrongBox | TAY+SQL | máy thật |
| TC-MOB-DEV-02 | US-AS-12 AC5; AC-6 | AR | Sau DEV-01 | Điểm danh trong app (không vị trí) | `assurance=3`; không cờ; Staff không thấy chip «Chưa xác minh» | TAY+SQL | máy thật |
| TC-MOB-DEV-03 | US-AS-12 AC4; AC-7; US-AS-11 AC4 | A | Emulator (khoá phần mềm) | Điểm danh lần đầu (đăng ký thiết bị chạy ngầm) | đăng ký `422 ATTESTATION_FAILED`; app VẪN ký bằng khoá Keystore và gửi `WEB_KEY` (DER); điểm danh 201; `assurance=2`; không `DEVICE_UNTRUSTED`; SV không thấy lỗi | TAY+SQL |  |
| TC-MOB-DEV-04 | US-AS-12 AC3; AC-7b | A | Máy chủ không tải được danh sách thu hồi Google (dev chặn egress `android.googleapis.com` ở VPS test) | Đăng ký lần đầu | `503 ATTESTATION_UNAVAILABLE`; app vẫn điểm danh (mức 2); thử đăng ký lại sau khi gỡ chặn thì 201 (hoặc 422 trên emulator) | TAY+SQL | cần dev chặn egress |
| TC-MOB-DEV-05 | US-AS-12 AC4; AC-8 | A, AR | APK debug ký khoá khác (dựng `assembleDebug`, cài đè sau khi gỡ) | Điểm danh | đăng ký `422 ATTESTATION_FAILED` (gói/vân tay lệch); điểm danh vẫn 201, mức 2 nhờ `WEB_KEY`; KHÔNG đạt mức 3 | TAY+SQL |  |
| TC-MOB-DEV-06 | US-AS-12 AC5; AC-9 | A, AR | Lớp QCMOB3 `require=APP`; SV dùng WEB trên Chrome điện thoại | Điểm danh bằng liên kết | lượt web có cờ `NOT_APP`; vẫn 201; SV không thấy; trong app (máy A) không `NOT_APP` | TAY+SQL |  |
| TC-MOB-DEV-07 | US-AS-12 AC5; AC-9b | AR | QCMOB3; bootloader mở hoặc `GEO_MOCK` | Điểm danh trong app | không `NOT_APP` (đã có `DEVICE_UNTRUSTED`/`GEO_MOCK`); vẫn 201 | TAY+SQL | máy thật |
| TC-MOB-DEV-08 | US-AS-04 AC5; AC-13 | AR | `sv.gioi` đổi máy A → B | Điểm danh lần đầu ở máy B, rồi lần sau | lượt đầu máy B `DEVICE_NEW`; lượt sau không | TAY+SQL | hai máy thật |
| TC-MOB-DEV-09 | US-AS-12 AC1 | A | Gateway log, `adb` mở/đóng app 5 lần | Đếm `POST /me/devices` và `/challenge` | chỉ đăng ký MỘT lần (key_id đã cache `localStorage.ep.device.android`); trên emulator thất bại → không thử lại mỗi lượt trong cùng phiên (`sessionStorage.ep.device.android.failed`); ≤ 5 challenge/phút | ADB |  |
| TC-MOB-DEV-10 | US-AS-12 AC1 | A | Script gọi `challenge` 6 lần/phút bằng token SV | `curl` ×6 | lần thứ 6 → 429; không rò nonce | API |  |
| TC-MOB-DEV-11 | US-AS-12 AC4 | API | Nonce sai / hết hạn / đã dùng | `POST /me/devices` với `attestation_chain` mẫu | `422 DEVICE_CHALLENGE_INVALID`; chuỗi quá lớn (> 6 chứng chỉ hoặc > 16 KB) → 422; chuỗi tự ký → `ATTESTATION_FAILED` | API |  |
| TC-MOB-DEV-12 | US-AS-12 AC9 | API | `go test ./internal/attendance -run 'TestAttestRealChains\|TestAttestRejects'` | Chạy test với chuỗi mẫu `caiman…` (StrongBox, VERIFIED) và `akita…` (TEE, UNVERIFIED) | `boot_verified` đúng 2 ca; chứng chỉ giả → `ATTESTATION_FAILED` (thay cho máy thật khi không có) | DIFF |  |
| TC-MOB-DEV-13 | US-AS-04 AC4 | A | Gỡ app rồi cài lại; điểm danh | SQL `user_devices` | khoá cũ không dùng được; khoá mới → `DEVICE_NEW` một lần | TAY+SQL |  |
| TC-MOB-DEV-14 | US-AS-12 AC6; US-AS-11 AC6 | A | Plugin lỗi giả (đổi bản kiểm: `sign` ném lỗi) hoặc máy không Keystore | Điểm danh | app không gửi `device` (mức 1, `NO_DEVICE_KEY`); không treo; vẫn «Đã điểm danh» | TAY+SQL | bản kiểm của dev |

## 6.V Hiệu năng và nhịp tải
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-PRF-01 | US-AS-11; nhịp tải | A, AR | App đã cài; 5 lần khởi động nguội (`am force-stop`) | `adb shell am start -W -n vn.edupilot.app/.MainActivity` ×5; ghi `TotalTime` | ghi trung vị/lớn nhất; đề xuất ≤ 2 s tới khung (PM chốt); không ANR; từ khởi động tới thấy «Hôm nay» đã đăng nhập ≤ 5 s | ADB |  |
| TC-MOB-PRF-02 | US-AS-11; AC-17 | A | W mở, poster QR; app đã quét sẵn quyền | Đo stopwatch ×5 từ lúc QR vào khung tới «Đã điểm danh» (quay màn hình `adb shell screenrecord`) | trung vị ≤ 3 s (DESIGN §14.8.2); ghi lớn nhất | TAY |  |
| TC-MOB-PRF-03 | US-AS-15 AC9; nhịp tải | A | R-NET-slow | Từ nhận QR tới «Đang gửi» | ≤ 300 ms (đo trên quay màn hình 60 fps) | TAY |  |
| TC-MOB-PRF-04 | US-AS-14 AC4 | W | Lighthouse mobile (`lighthouserc.json` dạng mobile) cho `/checkin` | Chạy 3 lượt | LCP ở trạng thái «đã quay lại» ≤ ngưỡng của cổng 10.7; TBT ≤ 200 ms; CLS ≤ 0,1; JS ≤ 256000 B | E2E |  |
| TC-MOB-PRF-05 | US-AS-11 | A | Quét liên tục 10 lần (đóng/mở khung) | `dumpsys meminfo vn.edupilot.app` đầu và cuối | bộ nhớ không tăng tuyến tính (> 30 MB sau 10 lần ghi nhận); không crash; camera luôn tắt sau mỗi lần | ADB |  |
| TC-MOB-PRF-06 | US-AS-11 | A | Quét 60 s | `dumpsys gfxinfo vn.edupilot.app` | tỷ lệ khung giật ghi nhận; không ANR; pin giảm không bất thường (ghi nhận) | ADB |  |
| TC-MOB-PRF-07 | nhịp tải | W | Chuyển trạng thái sending → done | Quay `trace` ghi thời gian | khung xương → kết quả không nhấp nháy, không chớp trắng; chuyển ≤ 300 ms sau phản hồi | E2E |  |
| TC-MOB-PRF-08 | US-AS-09 AC1 | I | iPhone 15 Simulator | `openurl` → «Đã điểm danh» ×5 | trung vị ≤ 3 s; ghi lớn nhất | SIMCTL |  |

## 6.W Seed, E2E, tải, dữ liệu 30 ngày nhìn từ điện thoại (US-AS-13, US-AS-14)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-DAT-01 | US-AS-14 AC1 | A, I | Seed `pnpm dev` trên stack thử; lớp 761988; cửa sổ vòng 1 đóng sẵn | `sv.gioi` điểm danh trên emulator ở buổi demo mới (mở tay) | điểm danh được; 270 `attendance_records` của 761987 và `seed/expected_crm.csv` không đổi | TAY+SQL |  |
| TC-MOB-DAT-02 | US-AS-13 AC2; US-AS-09 AC5 | A | Cửa sổ đã purge (dev chạy `gateway admin purge-checkins` ở dev/test) | Mở lại liên kết cũ của cửa sổ đã purge | KHÔNG «Đã điểm danh hh:mm» (lượt đã xoá); hiện câu lỗi đúng (đóng/hết hạn); `GET /me/attendance.checked_in_at=null`; trạng thái cuối còn nguyên | TAY+SQL | stack thử |
| TC-MOB-DAT-03 | US-AS-14 AC2 | W | `attendance-checkin.spec.ts` ở viewport 375×812 | Chạy với `--project=mobile` | xanh; kịch bản: quét đúng/lạ/không hỗ trợ, nhập mã đúng/sai/hết hạn/đóng/vượt 5 sai, `DEVICE_SHARED`, 375 px | E2E |  |
| TC-MOB-DAT-04 | US-AS-14 AC3 | W | k6 `benchmarks/load/checkin.js` | Chạy 200 SV/30 s + 200 lượt/5 s | `POST …/checkins` p95 ≤ 500 ms, lỗi 0 %, đúng 200 hàng (liên quan trải nghiệm: hết thời gian điểm danh di động ≤ 3 s) | E2E |  |
| TC-MOB-DAT-05 | US-AS-13 AC5 | A | Sau nhiều lượt trên điện thoại | Kiểm lược đồ `attendance_checkins` | không cột toạ độ, IP, tên máy; `user_devices.label` NULL | SQL |  |
| TC-MOB-DAT-06 | US-AS-14 AC4 | W | `/checkin` axe mobile | Chạy axe ở 375 | không lỗi nghiêm trọng (trùng A11-01, ghi chéo) | AXE |  |
| TC-MOB-DAT-07 | US-AS-14 AC5 | W | Luật | `bash scripts/ui-antipatterns.sh` | sạch (0 vi phạm) cho `/checkin` và khung app trên di động | DIFF |  |

## 6.X Staff trên điện thoại: `/attendance` (US-AS-15 phạm vi; US-AS-08)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-ATT-01 | US-AS-15 AC1; US-AS-08 | A, I | `teacher@` trên điện thoại; lớp QCMOB1 | Mở `/attendance`, chọn buổi, mở cửa sổ điểm danh bằng nút chính | mở được; thấy mã/ nút «Mở màn chiếu»; mỗi vùng một hành động chính; không cuộn ngang | TAY |  |
| TC-MOB-ATT-02 | US-AS-15 AC13 | W, A | `teacher@` | Đo hộp bấm mọi nút/hàng ở `/attendance` | ≥ 44 × 44 px; hai điều khiển kề ≥ 8 px | E2E |  |
| TC-MOB-ATT-03 | US-AS-15 AC1 | W, A, I | `ta@` | Mở lưới điểm danh | lưới cuộn trong khung riêng, trang không cuộn ngang; cột chính thấy; thanh lưu trên thanh dưới | E2E+TAY |  |
| TC-MOB-ATT-04 | US-AS-15 AC3 | A (cutout, gesture), I | GV | Mở `/attendance` | thanh lưu và nút chính không dưới vùng bị che | TAY |  |
| TC-MOB-ATT-05 | US-AS-15 AC4 | A (R-FONT-2.0) | GV | Duyệt `/attendance` | không cuộn ngang; nút chạm được; chữ không chồng | TAY |  |
| TC-MOB-ATT-06 | US-AS-15 AC5 | A, I | Máy tối | Mở `/attendance` | giao diện vẫn sáng | TAY |  |
| TC-MOB-ATT-07 | US-AS-15 AC12 | W, A | GV | axe + TalkBack trên `/attendance` | 0 lỗi nghiêm trọng; nút có tên; lưới có nhãn hàng/cột | AXE+TAY |  |
| TC-MOB-ATT-08 | US-AS-15 AC15; US-AS-08 | A | SV mở `/attendance` (không phải Staff) | Gõ `/attendance` | màn chặn chuẩn (không dữ liệu lớp); không gọi API Staff | TAY+ADB |  |
| TC-MOB-ATT-09 | US-AS-15 AC2 | W, A | GV ngang 844×390 | Mở `/attendance` | thanh bên (≥ 720 px) không che nội dung; nút chính thấy; không cuộn ngang | E2E+TAY |  |
| TC-MOB-ATT-10 | US-AS-08 | I | GV iPhone mở `/attendance/display` (màn chiếu) | Mở trên điện thoại | màn chiếu hiển thị mã/QR (dùng để chiếu); không crash; Esc/`Thoát` có; không dùng để SV quét trên chính máy | TAY |  |
| TC-MOB-ATT-11 | US-AS-10 AC7 | A, I | QCMOB2 có lượt `GEO_OUT`, `OFF_CAMPUS`; `teacher@`/`ta@` rồi `sv.gioi` | Staff xem danh sách lượt ở `/attendance`; SV mở mọi màn của mình | Staff thấy `geo_distance_m`, `in_campus`, cờ; SV KHÔNG thấy khoảng cách, cờ, mức tin cậy, thiết bị ở bất kỳ màn nào (đọc cả `/me/attendance`) | TAY+SQL |  |

## 6.Y Lối vào Điểm danh trong app / PWA: nav, «Thêm», «Tìm nhanh», «Hôm nay» (US-AS-15 AC16, P-11-137)
| Mã | AC | MT | Dữ liệu / tiền điều kiện | Bước | Kỳ vọng (đo được) | Kiểm | Chỉ máy thật / điều kiện |
| --- | --- | --- | --- | --- | --- | --- | --- |
| TC-MOB-ENT-01 | US-AS-15 AC16(a); US-AS-09 AC10 | W, A (R-ROT-land, ≥ 720 px), I (iPad-like không có; dùng Playwright 1024) | `sv.gioi` thuộc QCMOB1; viewport 1024×768 và 844×390 | Mở «Hôm nay» và đọc thanh điều hướng (bên) | có mục `Điểm danh` → `/checkin` (nhãn đúng «Điểm danh»), vị trí sau «Bài thi» theo SRS `FEAT-ui-foundation` 7.5; tổng mục SV = 10; bấm → `/checkin` hiện ô «Mã điểm danh» | E2E+TAY |  |
| TC-MOB-ENT-02 | US-AS-15 AC16(a) | A (360/411), I (iPhone 15), P | Như ENT-01, điện thoại dọc (thanh dưới 5 mục) | Mở menu «Thêm» | thanh dưới giữ nguyên (Hôm nay, Chat riêng, Hỏi đáp, Luyện đề, Thêm); trong «Thêm» có `Điểm danh`; chạm → `/checkin`; vùng chạm ≥ 44 px; nhãn tiếng Việt; đúng thứ tự theo `nav.ts` | TAY+E2E |  |
| TC-MOB-ENT-03 | US-AS-15 AC16(a) | W, A, I, P | SV có lớp | Mở «Tìm nhanh» (nút tìm trên thanh trên / phím tắt), gõ `diem danh` rồi `Điểm danh` | kết quả `Điểm danh` hiện (cả không dấu); Enter / chạm → `/checkin`; không kết quả lạ; bàn phím ảo không che danh sách | E2E+TAY |  |
| TC-MOB-ENT-04 | US-AS-15 AC16(a); Q-A-7 | W, A | Ba lớp của SV: QCMOB1 (bật, cửa sổ ĐÃ đóng), QCMOB5 (`self_checkin` tắt), QCMOB2 (bật, chưa từng mở cửa sổ) | Chọn lần lượt từng lớp; đọc nav, «Thêm», «Tìm nhanh» | mục `Điểm danh` có ở cả ba lớp (không phụ thuộc bật/tắt hay cửa sổ); giao diện mục giống hệt nhau (không biểu tượng chấm, số, màu khác) | E2E+TAY |  |
| TC-MOB-ENT-05 | US-AS-15 AC16(a); Q-A-7 | A, I | QCMOB1; SV mở sẵn app ở «Hôm nay» | Staff mở cửa sổ vòng 1; chờ 70 s (một vòng quét); chụp nav, «Thêm», «Hôm nay», trung tâm thông báo; Staff mở vòng 2 | KHÔNG có báo «điểm danh đang mở» ở bất kỳ đâu (không thông báo, huy hiệu, chấm, dòng chữ); nav / «Thêm» DOM không đổi so với trước khi mở; vòng 2 không báo trước | TAY+E2E |  |
| TC-MOB-ENT-06 | US-AS-15 AC16(b) | W, A | QCMOB1 (bật); buổi hôm nay A: bắt đầu sau now+16 phút (ngoài khoảng) | Mở «Hôm nay» của SV | hàng buổi A KHÔNG có liên kết `Điểm danh` | E2E |  |
| TC-MOB-ENT-07 | US-AS-15 AC16(b) | W, A | Buổi B: bắt đầu sau now+14 phút (trong 15 phút trước giờ) | Mở «Hôm nay» | hàng buổi B có liên kết `Điểm danh` → `/checkin`; chạm ≥ 44 px; không tràn ngang ở 360 px | E2E+TAY |  |
| TC-MOB-ENT-08 | US-AS-15 AC16(b) | W, A | Buổi C: đang diễn ra (start = now−30', end = now+30'); buổi C2: end = now+1' | Mở «Hôm nay»; chờ 2 phút ở C2 (hoặc đặt end sát mốc) rồi tải lại | C: có liên kết; C2: có liên kết đến hết giờ buổi; sau end KHÔNG còn (ghi giờ chuyển mất ±60 s) | E2E |  |
| TC-MOB-ENT-09 | US-AS-15 AC16(b) | W, A | Buổi D: end = now−1' (vừa hết giờ) | Mở «Hôm nay» | KHÔNG có liên kết `Điểm danh` (hoặc buổi không còn trong timeline); ghi rõ hành vi | E2E |  |
| TC-MOB-ENT-10 | US-AS-15 AC16(b); (d) | W, A | Buổi E của lớp QCMOB5 (`self_checkin` tắt) đang diễn ra | Mở «Hôm nay» | KHÔNG có liên kết `Điểm danh` ở hàng buổi E; mục nav vẫn có (ENT-04) | E2E |  |
| TC-MOB-ENT-11 | US-AS-15 AC16(b) | API | `go test ./internal/today -run 'TestTimelineCheckinFlag\|TestTodayQueryBudget'` chạy trên VPS (`ep-vps-test`); `GET /me/today` bằng token SV | So `timeline[]` và danh sách mục `today` | không mục `today` mới; hợp đồng `timeline[]` chỉ thêm MỘT cờ boolean (không dữ liệu cá nhân); SV giữ ≤ 8 truy vấn; `openapi.yaml` khớp | DIFF+API |  |
| TC-MOB-ENT-12 | US-AS-15 AC16(c); US-AS-09 AC10 | A, I, P | SV có lớp bật; không `#token` | Vào `/checkin` từ mục `Điểm danh` ở app Android, Safari, PWA | app Android: nút chính `Quét mã điểm danh` + `Nhập mã`; Safari và PWA: CHỈ ô «Mã điểm danh» + `Điểm danh` (không nút quét); dòng «Lớp {mã lớp}» | TAY |  |
| TC-MOB-ENT-13 | US-AS-15 AC16(d) | A, I, P | Lớp QCMOB5 (`self_checkin` tắt), SV chọn lớp đó | Vào `/checkin` từ mục `Điểm danh` | màn «Lớp này không điểm danh bằng mã.» — KHÔNG ô nhập, KHÔNG nút gửi, KHÔNG gọi `POST …/checkins` (gateway log trống) | TAY+ADB |  |
| TC-MOB-ENT-14 | US-AS-15 AC16(e); US-AS-09 AC4; US-AS-03 AC3a | A, I, P | Lớp bật; (i) cửa sổ vừa đóng ≤ 15 phút, có mã ngắn của bước cuối; (ii) lớp chưa từng mở cửa sổ; (iii) có cửa sổ đã đóng > 15 phút; (iv) liên kết `#token` của cửa sổ đã đóng | Vào `/checkin` từ mục «Điểm danh»: (i) nhập mã ngắn của cửa sổ đã đóng; (ii) nhập mã bất kỳ đúng cú pháp; (iii) nhập mã của cửa sổ đóng > 15 phút; (iv) mở liên kết `#token` của cửa sổ đã đóng | (i) «Đã đóng điểm danh.» (mã khớp cửa sổ ĐÃ ĐÓNG, 409 `WINDOW_CLOSED`); (ii) «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» (chưa từng mở cửa sổ — không khớp cửa sổ nào trong 15 phút; không lộ cửa sổ có tồn tại, Q-A-7); (iii) như (ii); (iv) «Đã đóng điểm danh.»; không câu mới ngoài hai câu này (BA chốt `ab9cb871`) | TAY+SQL |  |
| TC-MOB-ENT-15 | US-AS-15 AC16(f) | W, A, I | `sv.moi` (chưa vào lớp nào) | Đăng nhập, đọc nav, «Thêm», «Tìm nhanh»; gõ `/checkin` | mục `Điểm danh` KHÔNG hiện (thanh điều hướng chỉ `Hôm nay`, SRS `FEAT-ui-foundation` 4.3.2); gõ `/checkin` → màn «Bạn chưa vào lớp nào.» | E2E+TAY |  |
| TC-MOB-ENT-16 | US-AS-15 AC16(g); US-AS-09 AC8 | W, A, I | `teacher@`, `ta@`, `admin@` | Đọc nav, «Thêm», «Tìm nhanh»; gõ `/checkin` | KHÔNG có mục `Điểm danh` ở cả ba vai; `/checkin` → màn chặn chuẩn, không gọi API điểm danh | E2E+TAY |  |
| TC-MOB-ENT-17 | US-AS-15 AC16(h); AC13 | W, A | Mục `Điểm danh` ở nav, «Thêm», «Tìm nhanh», liên kết «Hôm nay» | Đo `getBoundingClientRect()`; đọc chữ | ≥ 44 × 44 px (đo hộp bấm); nhãn đúng «Điểm danh», tiếng Việt có dấu; KHÔNG câu giải thích, tooltip, huy hiệu (luật chữ UI 09/10) | E2E+TAY |  |
| TC-MOB-ENT-18 | US-AS-15 AC16(a); AC12 | W, A | `/checkin` đang mở | Quan sát mục trong nav; duyệt bằng TalkBack/VoiceOver | mục `Điểm danh` ở trạng thái đang chọn (`aria-current=page`); trình đọc đọc «Điểm danh» + «đã chọn» | E2E+TAY |  |
| TC-MOB-ENT-19 | US-AS-15 AC16(a) | W | SV 2 lớp (một bật, một tắt); chọn «Tất cả lớp của tôi» (nếu có) | Vào mục `Điểm danh` | mục hiện; `/checkin` yêu cầu chọn một lớp cụ thể hoặc hiện «Lớp {mã lớp}» của lớp đang chọn (ghi hành vi); không lỗi kỹ thuật | E2E+TAY |  |
| TC-MOB-ENT-20 | US-AS-15 AC16(b); AC1 | W, A | «Hôm nay» có buổi + liên kết ở 360/375/430 px | Chụp, đo | hàng buổi + liên kết không cuộn ngang; liên kết thấy không cần cuộn nhiều; không chồng chữ | E2E+TAY |  |
| TC-MOB-ENT-21 | US-AS-15 AC16; luật cứng 1 | API | Token SV | So `GET /me/today` trước/sau khi có cờ; tìm trường cá nhân thêm | không `student_id`, cờ, mức tin cậy, thiết bị, vị trí trong `timeline[]`; chỉ một boolean thêm | API |  |
| TC-MOB-ENT-22 | US-AS-15 AC16(a); AC11 | P | PWA đã cài, đã đăng nhập | Từ «Hôm nay» trong PWA vào `/checkin` chỉ bằng điều hướng của khung | đi tới và quay lại được không cần trình duyệt; không màn cụt (PWA không có nút Back) | TAY |  |

## 7. Truy vết AC → TC
| AC | TC thiết bị trong tệp này |
| --- | --- |
| US-AS-09 AC1 | ERR-17, FLW-01, FLW-02, FLW-08, FLW-15, PRF-08 |
| US-AS-09 AC2 | ERR-06, SEC-02, SES-11, SES-12 |
| US-AS-09 AC3 | FLW-18, PWA-22, SEC-12, SES-14 |
| US-AS-09 AC4 | ENT-14, ERR-01, ERR-02, ERR-03, ERR-04, ERR-05, ERR-10, ERR-16, ERR-20, FLW-05, MSG-01, MSG-13, NET-02, SCN-05 |
| US-AS-09 AC5 | DAT-02, FLW-05 |
| US-AS-09 AC6 | PWA-01, PWA-12, PWA-13, PWA-16 |
| US-AS-09 AC7 | FLW-17, UIW-08 |
| US-AS-09 AC8 | ENT-16, ERR-01, ERR-07, ERR-08, ERR-19, SES-06 |
| US-AS-09 AC9 | FLW-08, SCN-10, SEC-01, SEC-02, SEC-17 |
| US-AS-09 AC10 | ENT-01, ENT-12, ERR-11, ERR-12, ERR-13, FLW-03, FLW-04, FLW-10, MSG-12, PWA-08 |
| US-AS-10 AC1 | GEO-07, PRM-10 |
| US-AS-10 AC2 | GEO-02, GEO-03, GEO-05, GEO-12 |
| US-AS-10 AC3 | GEO-13 |
| US-AS-10 AC4 | GEO-01, GEO-04, GEO-06, GEO-11 |
| US-AS-10 AC5 | GEO-03 |
| US-AS-10 AC6 | — (API (422 hình dạng toạ độ/geofence): `tc-US-AS-10-12-13.md` TC-AS10-xx; ở điện thoại không tạo được thân sai từ giao diện) |
| US-AS-10 AC7 | ATT-11 |
| US-AS-11 AC1 | INS-01, INS-02, INS-03, INS-04, INS-07, INS-08, INS-09, INS-10, INS-11, INS-12, INS-14, PRM-11, PRM-12, SEC-03, SEC-04, SEC-05, SEC-06, SEC-08, SEC-09, SEC-10 |
| US-AS-11 AC2 | — (DIFF: `tc-US-AS-11-14.md` TC-AS11-04; ở điện thoại: DEV-03/05 (cùng route)) |
| US-AS-11 AC3 | FLW-15, SCN-01, SCN-02, SCN-03, SCN-04, SCN-06, SCN-07, SCN-08, SCN-13, SCN-14, SCN-15, SCN-16 |
| US-AS-11 AC3a | MSG-08, PRM-01, SCN-11, SCN-12 |
| US-AS-11 AC3b | FLW-16, INS-13 |
| US-AS-11 AC4 | DEV-03, SEC-13 |
| US-AS-11 AC5 | INS-11, PRM-12, SEC-07 |
| US-AS-11 AC6 | DEV-14, NET-06 |
| US-AS-11 AC7 | SEC-14, SEC-18, SES-01, SES-02, SES-04, SES-05, SES-08, SES-13 |
| US-AS-11 AC8 | — (chính checklist này (`android-checklist.md`) + DEV-01…14) |
| US-AS-12 AC1 | DEV-01, DEV-09, DEV-10 |
| US-AS-12 AC2 | — (API/Go `TestAttestRejects`: `tc-US-AS-10-12-13.md`; ở điện thoại DEV-01, DEV-03, DEV-12) |
| US-AS-12 AC3 | DEV-04 |
| US-AS-12 AC4 | DEV-03, DEV-05, DEV-11 |
| US-AS-12 AC5 | DEV-02, DEV-06, DEV-07, GEO-10 |
| US-AS-12 AC6 | DEV-14 |
| US-AS-12 AC7 | SEC-11, SEC-12 |
| US-AS-12 AC8 | SEC-14 |
| US-AS-12 AC9 | DEV-12 |
| US-AS-13 AC1 | — (server-only (worker purge): `tc-US-AS-10-12-13.md` TC-AS13-01…) |
| US-AS-13 AC2 | DAT-02 |
| US-AS-13 AC3 | — (server-only: `tc-US-AS-10-12-13.md`) |
| US-AS-13 AC4 | — (server-only (`ForgetStudent`): `tc-US-AS-10-12-13.md`) |
| US-AS-13 AC5 | DAT-05 |
| US-AS-13 AC6 | — (server-only (log purge): `tc-US-AS-10-12-13.md`) |
| US-AS-13 AC7 | — (server-only: `tc-US-AS-10-12-13.md`) |
| US-AS-14 AC1 | DAT-01 |
| US-AS-14 AC2 | DAT-03, FLW-14 |
| US-AS-14 AC3 | DAT-04 |
| US-AS-14 AC4 | DAT-06, PRF-04 |
| US-AS-14 AC5 | DAT-07 |
| US-AS-14 AC6 | — (bảng truy vết SRS mục 11: `tc-US-AS-11-14.md`; ở điện thoại: ERR + SEC) |
| US-AS-15 AC1 | ATT-01, ATT-03, ENT-20, UIW-01, UIW-02, UIW-03, UIW-04, UIW-05, UIW-06, UIW-07, UIW-08, UIW-09, UIW-10, UIW-17, UIW-18, UIW-19, UIW-20, UIW-21, UIW-22 |
| US-AS-15 AC2 | ATT-09, FLW-07, SCN-09, UIW-11, UIW-12, UIW-13, UIW-14, UIW-15, UIW-16, UIW-17 |
| US-AS-15 AC3 | ATT-04, PWA-05, PWA-14, SAF-01, SAF-02, SAF-03, SAF-04, SAF-05, SAF-06, SAF-07, SAF-08, SAF-09, SAF-10 |
| US-AS-15 AC4 | A11-15, ATT-05, TXT-01, TXT-02, TXT-03, TXT-04, TXT-05, TXT-06, TXT-07, TXT-08 |
| US-AS-15 AC5 | ATT-06, INS-06, THM-01, THM-02, THM-03, THM-04, THM-05, THM-06, THM-07, THM-08 |
| US-AS-15 AC6 | ERR-12, FLW-06, KBD-01, KBD-02, KBD-03, KBD-04, KBD-05, KBD-06, KBD-07, KBD-08, KBD-09, KBD-10, KBD-11, KBD-12, LIF-03 |
| US-AS-15 AC7 | NAV-01, NAV-02, NAV-03, NAV-04, NAV-05, NAV-06, NAV-07, NAV-08, NAV-09, NAV-10, NAV-11, NAV-12, SCN-07, SES-12 |
| US-AS-15 AC8 | GEO-04, GEO-08, GEO-09, GEO-11, PRM-01, PRM-02, PRM-03, PRM-04, PRM-05, PRM-06, PRM-07, PRM-08, PRM-09 |
| US-AS-15 AC9 | NET-01, NET-02, NET-03, NET-04, NET-05, NET-07, NET-08, NET-09, NET-10, NET-11, NET-12, PRF-03 |
| US-AS-15 AC10 | LIF-01, LIF-02, LIF-03, LIF-04, LIF-05, LIF-06, LIF-07, LIF-08, LIF-09, LIF-10, LIF-11, LIF-13, PRM-05, SCN-08, SES-02, SES-03, SES-09, SES-10 |
| US-AS-15 AC11 | ENT-22, INS-05, LIF-10, NAV-08, NET-10, PWA-01, PWA-02, PWA-03, PWA-04, PWA-05, PWA-06, PWA-07, PWA-08, PWA-09, PWA-11, PWA-14, PWA-15, SAF-06 |
| US-AS-15 AC12 | A11-01, A11-02, A11-03, A11-04, A11-05, A11-06, A11-07, A11-08, A11-09, A11-10, A11-11, A11-12, A11-13, A11-14, A11-15, A11-16, ATT-07, ENT-18 |
| US-AS-15 AC13 | ATT-02, ENT-17, TGT-01, TGT-02, TGT-03, TGT-04, TGT-05, TGT-06, UIW-22 |
| US-AS-15 AC14 | ERR-14, ERR-15, INS-10, LIF-12, MSG-01, MSG-02, MSG-03, MSG-04, MSG-05, MSG-06, MSG-07, MSG-08, MSG-09, MSG-10, MSG-11, MSG-12, MSG-13, MSG-14, MSG-15, MSG-16, NET-06, SCN-12 |
| US-AS-15 AC15 | ATT-08, ERR-05, ERR-07, FLW-09, SES-06 |
| US-AS-15 AC16 | ENT-01, ENT-02, ENT-03, ENT-04, ENT-05, ENT-06, ENT-07, ENT-08, ENT-09, ENT-10, ENT-11, ENT-12, ENT-13, ENT-14, ENT-15, ENT-16, ENT-17, ENT-18, ENT-19, ENT-20, ENT-21, ENT-22, FLW-12 |
| US-AS-15 AC17 | PWA-10, PWA-13, PWA-17, PWA-18, PWA-19, PWA-20, PWA-21, PWA-22, PWA-23 |

### Bước `AC-n` của `android-checklist.md` → TC
| Bước | TC |
| --- | --- |
| AC-0 | INS-01, INS-02, INS-04 |
| AC-1 | SES-01, SES-02 |
| AC-2 | SES-05 |
| AC-3 | FLW-01, SCN-01, SCN-02 |
| AC-3b | SCN-03 |
| AC-3c | INS-13, FLW-16 |
| AC-4 | SCN-11 |
| AC-4b | PRM-01 |
| AC-4c | SCN-12 |
| AC-5 | DEV-01 |
| AC-5b | DEV-07 |
| AC-6 | DEV-02 |
| AC-7 | DEV-03 |
| AC-7b | DEV-04 |
| AC-8 | DEV-05 |
| AC-9 | DEV-06 |
| AC-9b | DEV-07 |
| AC-10 | GEO-01, GEO-02, GEO-03, GEO-04 |
| AC-11 | GEO-10 |
| AC-12 | SES-07 |
| AC-13 | DEV-08 |
| AC-14 | SCN-04 |
| AC-15 | SCN-13, NET-02, NET-03 |
| AC-16 | SCN-08 |
| AC-17 | SCN-16, PRF-02 |
| AC-18 | UIW-01, UIW-02, TGT-01 |
| AC-19 | INS-08, DEV-13 |

## 8. Thứ tự chạy gợi ý (PM xếp lượt cuối)
1. **W + API + DIFF** (không cần thiết bị, ~1 giờ): E2E 5 độ rộng × 6 trạng thái, đo vùng chạm, axe, MSG-15, SEC-07/14/15/18, PWA-01/16, DEV-10…12, PRF-04, DAT-03…07.
2. **Emulator A theo nhóm trạng thái** để đỡ đổi cấu hình: (a) cài đặt, phiên, luồng, quét, lỗi; (b) đổi độ rộng 360 → 375 → 390 → 411 → 430 (UIW, SAF, TXT, TGT); (c) ngang; (d) tối/sáng; (e) bàn phím; (f) quyền; (g) mạng; (h) vòng đời; (i) bảo mật; (j) thiết bị/chứng thực. Sau mỗi nhóm trả cấu hình (R-SIZE reset, font 1.0, night no, rotation 0, network full).
3. **iOS Simulator I và P**: liên kết, nhập mã, vị trí, an toàn vùng, PWA, trợ năng (VoiceOver nếu bật được).
4. **Máy thật** (nếu có): DEV-01/02/07/08, GEO-10, SEC-13, SCN-16, PWA-13, A11-10/11.
Ước lượng: W ~1 giờ; emulator ~6 giờ; iOS ~2 giờ; máy thật ~2 giờ. Số ca: 345.

## 9. Bảng kết quả (điền khi chạy)

### 9.1 Pha 1 — Playwright trên VPS (Chromium), 25/11/2026
Môi trường: VPS `f6113582`/`913aada5`, Chromium 1.63 (không WebKit), 390×844 (`isMobile`, `hasTouch`) và 1024×768; tài khoản `sv.qc3@edupilot.local` (QC3), lớp **QC3M1** (bật điểm danh, `rotate_seconds=10`) và **QC3M2** (tắt); Staff `teacher@`.
Chạy bằng `ep-vps-test ~/Documents/TA_Agent_qc3 -- 'node docs/regression/evidence-mobile/<script>'`; script: `mrun.mjs` (ent/axe/sec/nav), `ent_probe.mjs`, `sec2.mjs`, `nav2.mjs`, `ent_fix.mjs`. Ảnh + JSON: `docs/regression/evidence-mobile/run1/`.

| Mã | KQ | Ghi chú (số đo) | Ảnh |
| --- | --- | --- | --- |
| TC-MOB-ENT-01 | P | Chromium 1024×768: 10 mục nav, có mục /checkin nhãn «Điểm danh» (thanh bên mở nhãn) | run1/shots/ENT-01-nav-1024.png |
| TC-MOB-ENT-02 | P | 390×844: thanh dưới 5 mục; «Thêm» có «Điểm danh»; hộp chạm 342×48 | run1/shots/ENT-02-more-390.png |
| TC-MOB-ENT-03 | P | «Tìm nhanh» gõ «diem danh» → có «Điểm danh» | run1/shots/ENT-03-palette-390.png |
| TC-MOB-ENT-04 | P | QC3M1 (bật) / QCMOB5 (tắt) / QCMOB2 (bật): mục giống nhau (href/HTML) | — |
| TC-MOB-ENT-05 | P | Mở cửa sổ vòng 1 trên buổi thật rồi tải lại: nav/«Thêm»/body không đổi, không câu «đang mở» | run1/shots/ENT-05-after-open.png |
| TC-MOB-ENT-06 | P | API: buổi bắt đầu sau 40 phút → checkin_link=false; UI: hàng «Sắp tới» không liên kết (đo 23:19–23:45 VN) | run1/shots/dbg3-today.png |
| TC-MOB-ENT-07 | P | API: buổi trong 15 phút trước giờ → checkin_link=true; UI: hàng có «Điểm danh» | run1/shots/dbg3-today.png |
| TC-MOB-ENT-08 | P | API: buổi đang diễn ra → true; UI có liên kết | run1/shots/dbg3-today.png |
| TC-MOB-ENT-09 | P | API: buổi vừa hết giờ → false; UI không liên kết | run1/shots/dbg3-today.png |
| TC-MOB-ENT-10 | P | QC3M2 self_checkin=false: API=false, UI hàng không liên kết | run1/shots/TC-MOB-ENT-10.png |
| TC-MOB-ENT-11 | P | `go test ./internal/today -run 'TestTimelineCheckinFlag|TestTodayQueryBudget'` trên VPS: ok (1,47 s) | — |
| TC-MOB-ENT-12 | P | Chromium 390: ô «Mã điểm danh» + «Lớp QC3M1», KHÔNG nút quét (web) — phần app/PWA để pha 2/3 | run1/shots/ENT-12-checkin-web-390.png |
| TC-MOB-ENT-13 | P | Lớp tắt: «Lớp này không điểm danh bằng mã.», 0 ô nhập | run1/shots/ENT-13-class-off-390.png |
| TC-MOB-ENT-14 | P | Lớp chưa từng mở cửa sổ, mã bịa → «Mã không đúng hoặc đã hết hạn…»; không câu «Đã đóng» | run1/shots/ENT-14-code-nomatch-390.png |
| TC-MOB-ENT-15 | P | sv.moi (chưa có lớp): nav 1 mục «Hôm nay», «Thêm» không có Điểm danh | run1/shots/ENT-15-noclass-390.png |
| TC-MOB-ENT-16 | P | teacher/ta/admin: 0 mục «Điểm danh» ở nav | — |
| TC-MOB-ENT-17 | P | 390: không điều khiển nav nào < 44 px; mục trong «Thêm» 342×48 | — |
| TC-MOB-ENT-18 | P | Ở /checkin: mục nav /checkin có aria-current=page | run1/shots/ENT-18-checkin-active.png |
| TC-MOB-ENT-19 | P | Bộ chọn lớp mở (105 mục); không có mục «Tất cả lớp» (ghi hành vi, chờ BA) | run1/shots/ENT-19-course-select-390.png |
| TC-MOB-ENT-20 | P | «Hôm nay» 390: hàng buổi + liên kết không tràn ngang | run1/shots/ENT-06-10-today-390.png |
| TC-MOB-A11-01 | P | axe /checkin Chromium 390: 0 vi phạm (0 critical/serious) | — |
| TC-MOB-A11-16 | P | /checkin: lang=vi, title «Điểm danh · EduPilot» | — |
| TC-MOB-SEC-01 | P | Luồng token 63 ký tự: 0/31 request chứa token; hash rỗng sau khi tải | run1/shots/SEC-01-token-flow.png |
| TC-MOB-SEC-07 | P | CSP không unsafe-eval; Permissions-Policy camera/geolocation (self); HSTS có; Referrer-Policy không đặt cho /checkin (chỉ trang liên kết một lần — ghi nhận, không phải lỗi theo AC5) | — |
| TC-MOB-SEC-18 | P | localStorage không có token (chỉ ep_demo_state/ep_returning); document.cookie không thấy ep_rt | — |
| TC-MOB-NAV-12 | P | Token flow: 1 POST /checkins; Back/Forward không gửi lại, không lặp lỗi | run1/shots/NAV-12-back-forward.png |
| TC-MOB-THM-06 | P | `color-scheme: dark` (Chromium): nền body vẫn rgb(255,255,255), color-scheme=light; axe WCAG AA 0 vi phạm | run1/shots/THM-06-dark.png |
| TC-MOB-PRM-10 | P | QC3M1 geo_check=false: không dòng «Lớp này kiểm tra vị trí…», 0 request định vị | — |
| TC-MOB-NAV-02 | P | Token flow: Back/Forward sau «Đã điểm danh» không gửi lại (POST=1), không hiện lỗi cũ | run1/shots/NAV-02-back.png |
| TC-MOB-NAV-03 | P | Mở liên kết khi chưa đăng nhập → /login → tự gửi 1 lần → «Đã điểm danh»; Back về «/» (không vòng /login) | run1/shots/NAV-03-login-back.png |
| TC-MOB-PRF-07 | P | POST→phản hồi 12 ms; phản hồi→«Đã điểm danh» 12 ms (VPS, ngưỡng QC ≤ 2000 ms) | — |
| TC-MOB-A11-09 | P | Ô «Mã điểm danh» có nhãn thật «Mã điểm danh» (h1 «Điểm danh»); 18 svg, 0 svg không aria-hidden | run1/shots/A11-09-label.png |
| TC-MOB-DAT-07 | P | `bash scripts/ui-antipatterns.sh` trên VPS: toàn bộ phép ✓ (0 vi phạm) | — |
| TC-MOB-DAT-03 | P | `e2e/attendance-checkin.spec.ts` dự án `mobile` trên VPS: **14/14 passed** (22,0 s) — gồm 429 đếm ngược, lớp kiểm tra vị trí, chưa đăng nhập, giảng viên bị chặn | — |
| TC-MOB-NAV-11 | P | Mở / → /calendar → /checkin: Back 1 = /calendar, Back 2 = / (ngăn xếp đúng, không vòng) | run1/shots/NAV-11-stack.png |
| TC-MOB-PRF-04 | P | Mobile 4G chậm (1,6 Mbps, RTT 150 ms) + CPU 4× bằng CDP trên Chromium (giữ phiên đăng nhập — lhci không mang được Bearer): LCP=780 ms (trần 2500), CLS=0,000 (trần 0,1), long task=0 (TBT≈0, trần 200), load=653 ms | run1/shots/PRF-04-throttled.png |
| TC-MOB-DAT-01 | P | Stack thử `ep-dat` (seed thật): 761987 = **270** `attendance_records` (đúng kỳ vọng), `seed/expected_crm.csv` sha256 `e20b4f7f…` — đo trước và sau. Mở tay vòng 1 cho buổi mới `01a213ae-d2a1-7539…` (26/11 06:30Z) bằng teacher: `POST …/checkin-windows` → **201**, `rotate_seconds`=10. Lấy mã màn chiếu (`XXP ART`), `sv.gioi` điểm danh trên UI mobile 390×844 → «Đã điểm danh 00:22»; DB `attendance_checkins` +1 (`channel=CODE`, `assurance=2`, `device_id` NULL, không geo, `flags` rỗng). Sau: 761987 vẫn **270**, sha file CRM **không đổi** | `run1/shots/DAT-01-after.png` |
| TC-MOB-DAT-02 | P | `ep-dat`: dev lùi mốc 2 cửa sổ (`01a213b7…`, `01a213ba…`) 31 ngày (`UPDATE checkin_windows` ×2, `purge rc=0`, 00:25:39 +07) rồi purge: **23 lượt, 0 thiết bị**. Mở lại `/checkin` bằng mã cũ `XXP ART`: KHÔNG «Đã điểm danh», hiện «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» `GET /courses/{id}/me/attendance`: buổi 4 → `checked_in_at=null`, `status=PRESENT` (trạng thái cuối nguyên). DB sau purge: `attendance_checkins` 44→21 (−23), `attendance_records` **293** (761987 = 270, 761988 = 23) không đổi, `checkin_windows` = 3 | `run1/shots/DAT-02-after.png` |
| TC-MOB-DAT-05 | P | `ep-dat` sau 44 lượt: `attendance_checkins` KHÔNG có cột toạ độ (`lat`/`lng`), IP, hay tên máy — chỉ `geo_distance_m`/`geo_accuracy_m` (khoảng cách/độ chính xác, không phải vị trí thô); quét toàn bộ bảng `attendance%` theo `lat|lng|lon|ip|host|machine|fingerprint` = **0** cột. `user_devices` 21 hàng, `count(label)`=**0** (toàn NULL) | — |
| TC-MOB-DAT-04 | P | `ep-dat` T1 (DB `edupilot_t1`: 20 lớp, 1.000 SV, 3 GV, 5 TA), k6 trên VPS (`/usr/local/bin/k6`), `benchmarks/load/checkin.js`. **spread** (200 SV/30 s): POST `/checkins` p95 **7,9 ms** (trần 500), lỗi **0/200**, `checkin_rows_mismatch`=**0**, display p95 5,0 ms, background p95 5,1 ms (< 300). **burst** (200 lượt/5 s): p95 **9,4 ms**, lỗi **0/200**, mismatch **0**, display 8,3 ms, background 10,8 ms. DB sau 2 lượt: `attendance_checkins` = **400** (200 + 200, mỗi cửa sổ 50) — đúng số lượt gửi. Tệp: `run1/dat04-spread.json`, `run1/dat04-burst.json` | `run1/dat04-*.json` |

**Chưa chạy trong pha 1** (không tính F): 6.J `THM-01…05,07,08` (thiết bị); 6.L `NAV-01…11` (phần Back trên thiết bị + bàn phím); 6.M `PRM-01…09,11,12` (quyền trên thiết bị); 6.Q `A11-02…08,10…15` (TalkBack/VoiceOver thiết bị); 6.T còn `SEC-02…06,08…17` (logcat/Keystore/APK); 6.V `PRF-01…03,05,06,08` (thiết bị); 6.W `DAT-05` (lược đồ) + `DAT-06` (axe — đã P ở A11-01).

**Quan sát (chưa phải lỗi, chờ PM/BA):**
- Thanh bên 1024 px thu gọn nhãn: mục nav chỉ còn biểu tượng 55×34 px (nhãn nằm ở `aria-label`). Luật DESIGN «điều hướng chính trên desktop luôn có nhãn chữ» — nếu buộc áp cho 1024 thì cần quyết định thiết kế; AC13 chỉ áp 360–430 nên **không** tính F.
- Bộ chọn lớp không có mục «Tất cả lớp của tôi» (105 mục, đều là lớp cụ thể) trong khi ENT-19 giả định có thể có.
- `GET /me/today` giới hạn timeline 8 mục: sinh viên nhiều lớp có thể không thấy buổi của lớp cần điểm danh (đã gặp với sv.gioi 9 lớp). AC16(b) không nói tới giới hạn này.
- `/checkin` không đặt `Referrer-Policy` (chỉ các trang liên kết một lần) — token nằm ở fragment và đã bị xoá nên rủi ro thấp.


### 9.2 Pha 2 — Android emulator (qc2, 25–26/11/2026)
Nguồn: `docs/sprints/10.8/qc/tc-mobile-run-android.md` (qc2; AVD `ep-api36` Android 16/API 36, WebView Chrome 133, APK debug HEAD `cd843318`, `APP_SERVER_URL` = VPS; số mới nhất theo lô 5, qc2 `eb88c798`). **34 TC = 28 P, 0 F, 6 N** — chi tiết từng ca + ảnh ở tệp đó và `docs/regression/evidence-mobile-android/`.

- **Đã đóng sau khi chạy lại (lô 5, VPS `f6113582`):**
  - `TC-MOB-CL-4` = **P**: cỡ chữ 2,0× dock 695..748, nav 756..816 (không đè, cách 8 px), nhãn nav một dòng ⇒ **DEF-R10.8-07 ĐÓNG** (sửa `f6113582`). Còn quan sát: nhãn nav bị ellipsis («Hôm …», «Luyệ…») — **đã hết** ở `ffad9d57` (PR #8, nhãn thanh đáy một dòng; xem `docs/design/review/2026-11-25-mobile.md`).
  - `TC-MOB-GEO-03` = **P**: mock 21,1000/105,9000 + geofence 21,0285/105,8542 r200 ⇒ `GEO_OUT`, `geo_distance_m`=9263, accuracy 5, `channel=CODE`, kèm `GEO_MOCK` (đúng US-AS-12 AC5).
- **N (6):** `INS-01`, `INS-02`, `INS-03`, `INS-09`, `INS-13` (chỉ có APK debug, không Gradle/SDK trên VPS, không AVD API 24–28, Chrome chưa qua màn chào), `GEO-05` (mock hết hạn giữa lượt ⇒ geolocation timeout, không có dòng DB; chạy lại lượt sau).
- **Chưa chạy (chờ lượt sau):** 6.D `SCN-01…16`, 6.E `GEO-01,04,06…13`, 6.H `SAF-04…10`, 6.J `THM-02…08`, 6.L `NAV-02,03,05…12`, 6.M `PRM-02…10`, 6.O `LIF-01…13`, 6.Q `A11-01,02,04…15` (TalkBack chưa bật), 6.U `DEV-01…14`, checklist `CL-3,5…12,14…16`, `INS-08,13`.

**Cộng dồn pha 1 + pha 2 (khử trùng `A11-16`):** 71 TC có kết quả — **63 P, 0 F, 8 N**.

### 9.3 Pha 3 — iOS Simulator (qc3, 26/11/2026, máy `qc3-iphone15`)

Công cụ: XCUITest headless (`docs/regression/evidence-ios/uitest/`, scheme `Qc2UITests` từ qc2) + `xcrun simctl`; ảnh trong `docs/regression/evidence-ios/shots/`. Máy: iPhone 15 (iOS 27.2, 393×852 CSS).

**Lô A — cấu hình, khung, sáng/tối: 7 TC = 7 P, 0 F.**

| TC | KQ | Bằng chứng |
|---|---|---|
| TC-MOB-PWA-01 | P | `manifest.webmanifest` 200 `application/manifest+json`: `name`/`short_name` «EduPilot», `start_url`/`scope` `/`, `display` `standalone`, `lang` `vi`, `theme_color` `#FFFFFF`, `background_color` `#FFFFFF`; icon `/brand/icon-192.png` 200 (192×192), `/brand/icon-512.png` 200 (512×512), `logo-edupilot-mark.svg` 200. Không có `serviceWorker` trong mã `frontend/src` (0 lần) ⇒ không đăng ký SW |
| TC-MOB-PWA-02 | P | Safari `/` → Chia sẻ → «Thêm vào Màn hình chính»: hộp tên đề xuất «EduPilot»; duyệt trang không hiện banner/hộp mời cài đặt |
| TC-MOB-PWA-03 | P | Biểu tượng trên Màn hình chính: logo EduPilot (sách + bút đỏ) trên nền trắng bo góc, không phải ảnh chụp trang/chữ cái đầu, không viền lạ; `apple-touch-icon` = `/brand/icon-192.png` (đã có trong HTML) |
| TC-MOB-SAF-05 | P | `/checkin` dọc trong Safari: đo khung (XCUITest) — cửa sổ 393×852; «Mã điểm danh» y=261 (dưới đảo động), ô nhập 296..340, nút «Điểm danh» 645..689, thanh đáy 728..745 (trên thanh công cụ Safari và trên vùng Home 818); không dải trắng đáy; HTML có `viewport-fit` |
| TC-MOB-SAF-06 | P | PWA standalone: thanh trạng thái trắng chữ đen, tiêu đề «Hôm nay»/«Điểm danh» dưới đảo; thanh đáy 5 mục nằm trên thanh Home; nền trên/dưới cùng màu canvas trắng, không dải đen/trắng lệch |
| TC-MOB-THM-04 | P | `simctl ui appearance dark`: trang vẫn SÁNG ở cả Safari và PWA (chữ đen trên nền trắng), thanh trạng thái PWA đọc được (chữ đen, nền trắng `theme_color`) |
| TC-MOB-THM-07 | P | Ảnh: giờ/pin/sóng màu đen trên nền trắng — đọc rõ. `theme_color` đang dùng `#FFFFFF` (không phải BRAND.red); đối chiếu US-AS-15 AC11/D59: nền sáng, ghi nhận là quan sát |

### 9.4 Soát tổng theo MÃ TC (26/11/2026 — qc chốt lại `bb052c71`)

Nguồn từng mã: qc3 iOS 59 mã (bảng §9.1–§9.7 + `tc-mobile-run-ios.md`), qc2 `results.jsonl` 68 mã, qc `tc-mobile-run-qc.tsv` 139 mã. 17 mã trùng (12 qc∩qc3, 5 qc∩qc2, 5 qc2∩qc3), không mã nào lệch nhãn trừ `TC-MOB-FLW-02` (qc **N** vì pha 1 không có dữ liệu cấu hình `rotate_seconds`; qc3 **P** trên iOS với cửa sổ 60 s — giữ P).

| Mã | P | F | N |
|---|---|---|---|
| qc3 iOS | 59 | 0 | 0 |
| qc2 Android | 57 | 0 | 11 |
| qc pha 1 | 82 | 2 | 55 |
| **Hợp (khử trùng)** | **176** | **2** | **66** |
| TC | 345 | | |

2 F của qc: `TC-MOB-TGT-02`, `TC-MOB-ATT-02` — cùng bằng chứng `/attendance` bề rộng 360 px, `.mHead` ↔ `.mPlus` cách 4 px (44×44 vẫn đạt). qc giữ mức **Thấp**.

**Trùng mã lỗi cần PM đổi số:** qc đặt việc trên là `DEF-MOB-05`, nhưng `DEF-MOB-05` đã dùng cho lỗi stash /checkin của qc3 (đã sửa `ce30065e`+`e039b61f`, PM đóng). Đề nghị đánh số lại thành `DEF-MOB-06`.

### 9.5 Pha 3 — iOS, lô B (một phần; 26/11/2026)

Máy `qc3-iphone15`; lớp `QC3M1` (`self_checkin` bật, `rotate_seconds` 60 để token sống đủ lâu qua bước tải trang), buổi 6/7.

| TC | KQ | Bằng chứng |
|---|---|---|
| TC-MOB-FLW-02 | P | `simctl openurl` liên kết `#token` khi Safari đã đăng nhập: trang tự gửi MỘT `POST`, hiện «Đã điểm danh 01:28», không bấm gì; DB `attendance_checkins` `channel=WEB`, đúng 1 dòng/cửa sổ |
| TC-MOB-ERR-02 | P | Mở liên kết của bước đã quay: «Mã đã hết hạn. Quét lại mã trên màn chiếu.» một dòng, không nút |
| TC-MOB-ERR-11 | P | Gõ mã sai `ZZZ999` + Enter: «Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu.» dưới ô, viền lỗi, không hộp thoại |
| TC-MOB-ERR-12 | P | Ô rỗng: nút «Điểm danh» ở trạng thái khoá (`isEnabled=false`); ký tự lạ đi cùng một câu lỗi chung ở ERR-11 |
| TC-MOB-ERR-13 | P | Dán 40 ký tự: ô chỉ giữ **12** (`value.count=12`), không tràn ngang |
| TC-MOB-ERR-18 | P | `POST …/checkins {code, student_id}` bằng token SV: **422** `VALIDATION_FAILED`, `details[0].{field:"student_id", code:"unknown"}`; không ghi gì |

**DEF-MOB-05 (Thấp, đề nghị mở):** mở một liên kết QR MỚI trong tab Safari đang còn `sessionStorage.ep.checkin` cũ (> 60 s) → trang hiện «Mã đã hết hạn. Quét lại mã trên màn chiếu.» và **không** điểm danh, dù token mới hợp lệ. Tái hiện 4 lần; cùng token đó `POST` trực tiếp trả **201**. Sau khi tắt hẳn Safari (xoá sessionStorage) thì liên kết mới chạy đúng («Đã điểm danh 01:28»). Gốc: `/checkin` giữ stash cũ trong `sessionStorage` khi trang được phục hồi (bfcache/SPA) nên không ghi đè stash mới; stash cũ quá 60 s ⇒ nhánh `stashExpired` trả câu hết hạn.

**Còn lại của lô B (chưa chạy):** `FLW-04` (PWA gõ mã), `ERR-01` (`c` không phải UUID), `ERR-03` (MAC sai), `ERR-04` (cửa sổ đã đóng), `ERR-05/19` (SV không thuộc lớp), `ERR-06` (đăng nhập chậm > 60 s), `ERR-07/08` (teacher/TA/admin), `ERR-09` (429 sau 6 mã sai), `ERR-15` (offline lần đầu — cần cắt mạng host), `ERR-17` (múi giờ), `NET-09/12`.

### 9.6 Nghiệm DEF-MOB-05 trên iOS (VPS `40281db4`) — ĐẠT

- Tab Safari đã có trạng thái cũ (> 60 s, màn «Đã điểm danh» của lượt trước) + mở liên kết QR MỚI trong cùng tab: trang gửi **đúng 1** `POST /checkins` (gateway log `19:20:45Z` 200, `19:23:36Z` 200 — 1 lần cho mỗi lần mở), hiện «Đã điểm danh 02:12», không còn «Mã đã hết hạn» oan.
- Liên kết còn token bước cũ: server trả `CHECKIN_TOKEN_EXPIRED` (đo bằng API: token 5 phút tuổi → 422, token mới → 201), trang rơi về nhánh hết hạn / ô nhập mã — đúng ý AC4.
- `#main` (skip link) không bị cất làm token (e039b61f): URL giữ nguyên, 0 POST.
- Giới hạn bằng chứng: `simctl openurl` bỏ qua lệnh mở lại y nguyên URL đang đứng (không phát `hashchange`), nên đường "đổi hash trên trang đang dựng" chỉ nghiệm được qua e2e của dev (`frontend/e2e/checkin-net.spec.ts`, techlead-ui ĐẠT); trên máy thật cần quét QR bằng camera.

### 9.7 Lô B — thêm 26/11 (VPS `40281db4`)

| TC | KQ | Bằng chứng |
|---|---|---|
| TC-MOB-ERR-03 | P | Sửa 1 ký tự MAC: **422** `CHECKIN_TOKEN_INVALID` «Mã không đúng. Quét lại mã trên màn chiếu.» |
| TC-MOB-ERR-05 | P | SV không thuộc lớp (`sv.moi`): **403** `FORBIDDEN`, `details.reason="course"` |
| TC-MOB-ERR-07 | P | TEACHER `POST /checkins`: **403** |
| TC-MOB-ERR-08 | P | TA `POST /checkins`: **403** |
| TC-MOB-ERR-09 | P | 5 lần mã sai liên tiếp = 422, lần thứ **6** trở đi = **429** `RATE_LIMITED` |
| TC-MOB-ERR-01 | P | P-11-145. (a) `?c=khong-phai-uuid#<token thật>` → «Liên kết không đúng.» một dòng, không gọi API, không đăng xuất (`ERR-01c`). (b) `?c=khong-phai-uuid#tokenbatky` → ô nhập mã «Mã điểm danh» + lớp QC3M1, **0 POST** (gateway log 90 s chỉ có 1 POST của lớp khác), URL giữ nguyên (`ERR-01b2`). |

## 10. Ghi chú
- Chữ UI theo luật cứng PO (09/10/2026): ca KHÔNG đòi câu giải thích mới; chữ lỗi chỉ theo bảng US-AS-09 AC4/AC10 và US-AS-15 AC14.
- Không có ca nào kiểm Play Integrity, App Attest, thông báo đẩy, đa cửa sổ, máy tính bảng, màn gập, chế độ tối thật (ngoài phạm vi US-AS-15).
- Ca chỉ máy thật / điều kiện: Keystore phần cứng (SEC-13, DEV-01/02/07/08), `GEO_MOCK` thật (GEO-10), PWA 8 ngày (PWA-13), VoiceOver thật (A11-10/11), TalkBack (A11-02…08 nếu AVD không có), hai thiết bị (FLW-15), danh sách thu hồi Google bị chặn (DEV-04), AVD API cũ (INS-09), mạng trường (GEO-13).
