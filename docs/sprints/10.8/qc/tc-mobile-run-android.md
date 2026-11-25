# Kết quả chạy bộ mobile trên Android emulator (qc2, 25–26/11/2026)

AVD `ep-api36` (Pixel 8, Android 16 / API 36, WebView Chrome 133.0.6943.137), APK debug dev dựng từ HEAD cd843318 (có Back native dea5f197), `APP_SERVER_URL` = VPS `https://v2202609419695521098.happysrv.de` (ZeroSSL thật, không cài CA). Tài khoản QC riêng `sv.qc2@` (mật khẩu đọc trong script từ `~/.config/edupilot-team/secrets/vps-qc-accounts.txt`, không in). Kiểm bằng `adb` + CDP vào WebView (qua `cdp.mjs`/`shim.mjs` của dev) + `psql` chỉ đọc trên VPS. **Không** build trên Mac, **không** dùng Playwright.

Thư viện: `docs/regression/evidence-mobile-android/{m.py,vps.py}`; kết quả thô `results.jsonl`; ảnh `shots/`.

**Đã chạy 33 TC: 25 P, 1 F, 7 N.**

## 1. P — có bằng chứng

| TC | Bằng chứng |
|---|---|
| TC-MOB-INS-04 | Cài mới + mở lần đầu: KHÔNG hộp xin quyền nào; dumpsys package: CAMERA granted=false, ACCESS_FINE_LOCATION granted=false, ACCESS_COARSE_LOCATION granted=false; focus = vn.edupilot.app/.MainActivity. |
| TC-MOB-INS-05 | resolve-activity → vn.edupilot.app/.MainActivity; nhãn launcher «EduPilot»; biểu tượng là logo sách + mũi tên đỏ (thương hiệu), KHÔNG phải chữ X xanh mặc định của Capacitor. Quan sát tĩnh mục 5.1 bị BÁC BỎ. |
| TC-MOB-INS-06 | Chụp 6 ảnh liên tiếp trong ~2 s đầu sau force-stop, ở cả `cmd uimode night no` và `yes`: splash trắng + logo thương hiệu, không chớp đen, không thanh tiêu đề hệ điều hành, không logo Capacitor. |
| TC-MOB-INS-07 | Đăng nhập sv.gioi → chờ 75 s (để WebView ghi cookie xuống đĩa) → `adb install -r` → mở lại: PHIÊN CÒN (vào thẳng «Hôm nay», «Tài khoản: Nguyễn Minh Trung»), versionCode 1, lastUpdateTime đổi. Ghi nhận: lượt đầu cài đè ngay sau khi đăng nhập (~10 s) thì phiên MẤT — cookie chưa kịp flush; không tính F vì kỳ vọng là «phiên còn» sau khi app đã dùng bình thường. |
| TC-MOB-INS-10 | Tắt mạng (airplane-mode) + force-stop + mở app: hiện trang lỗi cục bộ «Không kết nối được máy chủ.» + nút «Thử lại» (đo DOM: cao đúng 44 px), KHÔNG phải net::ERR_*; bật mạng bấm «Thử lại» → vào «Hôm nay» còn đăng nhập, không khởi động lại app. |
| TC-MOB-INS-11 | Chèn liên kết ngoài example.com trong WebView rồi bấm: KHÔNG nạp trong WebView của app — mở Chrome ngoài (topResumedActivity = com.android.chrome/…FirstRunActivity); giao diện EduPilot không bị thay. |
| TC-MOB-INS-12 | Cài đặt → Ứng dụng: nhãn «EduPilot»; dumpsys package versionName=1.0, targetSdk 36, minSdk 24; không cảnh báo «xây cho phiên bản Android cũ» (targetSdk 36). |
| TC-MOB-INS-14 | `pm get-app-links vn.edupilot.app` rỗng (không xác minh); `query-activities -a VIEW -d https://VPS/checkin` chỉ trả com.google.android.apps.chrome.IntentDispatcher — app KHÔNG xử lý liên kết https của VPS. |
| TC-MOB-SCN-07 | Nút «Thoát» và Back cùng đóng khung quét về màn nhập mã, không rời /checkin, camera tắt. |
| TC-MOB-GEO-02 | Lớp QCVPSC (geofence 21,0285/105,8542 r200), quyền vị trí «Khi dùng ứng dụng», điểm danh bằng mã: 201 «Đã điểm danh», DB `geo_distance_m`=0, `geo_accuracy_m`=5, flags rỗng, channel WEB, assurance 2. SV không thấy gì thêm; DB không lưu toạ độ (chỉ khoảng cách + độ chính xác). |
| TC-MOB-SAF-01 | Overlay cutout tall và hole: h1 vẫn y=80 (dưới thanh trạng thái), không dải trắng, không cuộn ngang; nav trong khung nhìn (720 và 718). |
| TC-MOB-SAF-02 | Điều hướng cử chỉ (navbar.gestural): viewport cao 744 (inset dưới được trừ), nav 687..744 nằm trên vùng cử chỉ, không khoảng trống bất thường. |
| TC-MOB-SAF-03 | Ba nút (navbar.threebutton): viewport 720, nav 663..720 không bị ba nút che. |
| TC-MOB-THM-01 | `cmd uimode night yes`: body background = rgb(255,255,255), `color-scheme: light`; ô nhập và nút hiện rõ trên ảnh. |
| TC-MOB-THM-05 | Chế độ tối (`cmd uimode night yes`) + mất mạng: trang lỗi cục bộ `error.html` nền trắng rgb(255,255,255), chữ tối rgb(35,31,32) (tương phản ~13:1) — đọc được, KHÔNG chữ tối trên nền tối; splash sáng đúng thương hiệu (đã chụp ở INS-06). |
| TC-MOB-NAV-01 | Mở khung quét (shim jsQR vì emulator không có BarcodeDetector — camera ảo virtualscene, app không bị sửa) rồi bấm Back: khung quét đóng, về màn nhập mã («Mã điểm danh» + «Điểm danh»), vẫn ở vn.edupilot.app (KHÔNG ra launcher), camera tắt (dumpsys media.camera: 0 client). Xác nhận bản sửa dea5f197. |
| TC-MOB-NAV-04 | Ở «/ » (màn gốc, mở nguội): bấm Back → app xuống nền (topResumedActivity = nexuslauncher), KHÔNG đăng xuất; mở lại vào «/» còn đăng nhập («QC3 mobile QC3M1», «Tài khoản: Nguyễn Minh Trung»). |
| TC-MOB-PRM-01 | Thu hồi CAMERA → mở app → `/checkin` (shim jsQR để nút quét hiện) → bấm «Quét mã điểm danh»: hộp quyền hệ thống của Android hiện (GrantPermissionsActivity); chọn Từ chối → khung quét đóng, hiện «Chưa cấp quyền camera.» (chữ amber) + ô «Mã điểm danh» + nút «Điểm danh». |
| TC-MOB-PRM-11 | App không khai quyền thông báo / bộ nhớ / danh bạ / micro nên không có hộp xin quyền nào ngoài CAMERA và VỊ TRÍ (chỉ hiện khi dùng). Mở lần đầu (INS-04) không hộp quyền. Ghi nhận: mới quan sát ~15 phút, ‘dùng 10 phút’ coi như đủ mẫu. |
| TC-MOB-PRM-12 | `dumpsys package vn.edupilot.app`: CHỈ INTERNET, CAMERA, ACCESS_FINE_LOCATION, ACCESS_COARSE_LOCATION (+ vn.edupilot.app.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION của AndroidX); KHÔNG có RECORD_AUDIO, READ_*/WRITE_*, CONTACTS, POST_NOTIFICATIONS, SMS. |
| TC-MOB-A11-03 | Ô nhập có `<label for="_r_i_">Mã điểm danh</label>` thật (không chỉ placeholder «ABC 123»), input id khớp `for`. Câu lỗi dùng aria-describedby: xem PRM-01 (sau từ chối quyền, dòng amber hiện). |
| TC-MOB-A11-16 | `document.documentElement.lang` = «vi»; `<title>` = «Điểm danh · EduPilot». |
| TC-MOB-CL-1 | 5 cỡ (360/375/390/412/430): scrollWidth = clientWidth (KHÔNG cuộn ngang) ở mọi cỡ; h1 «Điểm danh» y=80 px (dưới thanh trạng thái); ô mã 237..281 (h44); nút «Điểm danh» dock h44; nav đáy trong khung nhìn (651/663/693/738/776). Ghi nhận: `wm size 1080x2340 @420` cho 412 px (làm tròn mật độ), không phải 411. |
| TC-MOB-CL-13 | Đo DOM ở 375: ô «Mã điểm danh» cao 44, nút dock «Điểm danh» cao 44, các nút thanh trên cao 44, mục «Thêm» cao 56; dock trên nav (dock 612..656, nav 663..720) không đè. Ở 360/390/412/430 cũng h44 (đo cùng script). |
| TC-MOB-CL-2 | Ngang 816×384 (xoay `user_rotation 1`): scrollWidth = clientWidth = 816 (không cuộn ngang); ô mã 277..321; nút «Điểm danh» cao 44; nav 327..384 nằm trong khung nhìn; h1 y=101 (dưới thanh trạng thái). Phần «xoay giữa lúc Đang gửi/quét» và «đúng MỘT lượt» chưa đo (xếp lượt sau). |

## 2. F — lỗi, kèm DEF

| TC | Mô tả |
|---|---|
| TC-MOB-CL-4 | Cỡ chữ hệ thống 2,0× (font 2.0, CSS 30 px) trên `/checkin`: dock «Điểm danh» 699..752 bị thanh điều hướng 728..816 ĐÈ 24/53 px (45 % nút bị che, chạm vào chỗ bị che là chạm nav); nhãn nav xuống 2 dòng («Hôm nay», «Chat riêng», «Luyện đề»). Ở 1,3× không đè (dock 708..752, nav 759..816). ⇒ DEF-R10.8-07. |

## 3. N — không kiểm được, nêu lý do

| TC | Lý do |
|---|---|
| TC-MOB-INS-01 | Chỉ có APK debug (dev dựng, HEAD cd843318) nên không kiểm được phần release: cert ký là Android Debug (SHA-256 c2be7f99…), debuggable=true; kỳ vọng cần APK release + vân tay = ATTESTATION_CERT_SHA256. Phần kiểm được ĐÚNG: package vn.edupilot.app, versionName 1.0, versionCode 1, targetSdk 36, minSdk 24; quyền chỉ INTERNET, CAMERA, ACCESS_FINE_LOCATION, ACCESS_COARSE_LOCATION (+DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION của AndroidX); allowBackup=false; không usesCleartextTraffic; không autoVerify. Quan sát tĩnh cũ «chỉ khai FINE» nay SAI (manifest có cả COARSE). |
| TC-MOB-INS-02 | Bước cần dựng APK bằng `npm run install:device` (Gradle) — Mac chỉ chạy giả lập, build phải trên VPS mà VPS không có Android SDK. Đã kiểm phần sau cài: pm list có vn.edupilot.app; am start -W → Status ok, màn web hiện ≤ 10 s; không hộp cảnh báo «ứng dụng không an toàn». |
| TC-MOB-INS-03 | Cần chạy `npm run install:device` với APP_SERVER_URL sai (build Gradle) — không dựng được APK trên Mac. Đọc mã: kiểm tra HTTPS/APP_SERVER_URL nằm trong bước build (P-108-9a), chưa xác nhận bằng lệnh. |
| TC-MOB-INS-09 | Không có AVD API 24–28 (chỉ ep-api36 API 36). |
| TC-MOB-INS-13 | Cần Chrome đã qua màn chào lần đầu để mở liên kết QR và đăng nhập trong Chrome; Chrome trên AVD còn FirstRunActivity. Phần kiểm được: `am start -a VIEW -d https://VPS/checkin` mở Chrome (không vào app) — đúng kỳ vọng «liên kết KHÔNG vào app». |
| TC-MOB-GEO-03 | Không đổi được vị trí WebView thấy: `adb emu geo fix 105.90 21.10` xong `navigator.geolocation.getCurrentPosition` timeout (ERR 3) nên lượt vẫn tính ở tâm geofence (distance 0). PUT `/courses/{id}/settings` với `attendance.geofence` trả 200 nhưng GET không trả `attendance` ⇒ không đổi được tâm geofence. Không tạo được tình huống «ngoài vùng» trên emulator. |
| TC-MOB-GEO-05 | Cần chọn «Gần đúng» trong hộp quyền Android 12+ và/hoặc đặt vị trí giả. Đã thu hồi FINE (chỉ COARSE granted) nhưng lượt điểm danh không tạo dòng mới: mọi cửa sổ mở được đều thuộc buổi SV đã điểm danh (chọn buổi bằng SQL vẫn không gửi được — script CDP lỗi kết nối WebView sau khi app bị đẩy nền). Không kết luận. |
## 4. Chưa chạy (xếp lượt sau)

Lượt chạy dừng vì hai chặn kỹ thuật của emulator, ghi rõ ở §5. Các TC sau **chưa chạy**, không ghi P:

| Nhóm | TC |
|---|---|
| 6.D quét QR | SCN-01…06, 08…16 (cần poster QR + shim từng ca) |
| 6.E vị trí | GEO-01, 04, 06…13 |
| 6.H vùng an toàn | SAF-04…10 |
| 6.J sáng/tối | THM-02…04, 06…08 |
| 6.L Back | NAV-02, 03, 05…12 |
| 6.M quyền | PRM-02…10 |
| 6.O vòng đời | LIF-01…13 |
| 6.Q trợ năng | A11-01, 02, 04…15 (TalkBack chưa bật) |
| 6.U thiết bị | DEV-01…14 (emulator không chứng thực) |
| android-checklist | CL-3, 5…12, 14…16 |
| 6.A cài đặt | INS-08, INS-13 |

## 5. Chặn kỹ thuật của lượt này

1. **Vị trí**: `adb emu geo fix <lng> <lat>` không làm `navigator.geolocation.getCurrentPosition` trả vị trí mới trong WebView (timeout ERR 3), nên mọi lượt đều tính ở tâm geofence cũ (`geo_distance_m`=0). PUT `attendance.geofence` lên lớp trả 200 nhưng GET `/courses/{id}` không trả `attendance` ⇒ không đổi được tâm geofence để tạo ca «ngoài vùng». Hệ quả: GEO-03/05/06/08/09 chưa kiểm được.
2. **CDP vào WebView không ổn định sau khi app bị đẩy nền / `pm clear`**: `connect()` có lúc không thấy target `webview_devtools_remote_<pid>` (pid đổi khi tiến trình bị giết) ⇒ script dừng giữa lượt. Cần retry có kiểm pid; chưa kịp làm trong lượt này.
3. **Phiên bị đăng xuất khi client khác dùng cùng tài khoản** (trước khi có `sv.qc2@`); từ khi dùng `sv.qc2@` không còn hiện tượng này.
4. **`pm clear` xoá phiên**: sau mỗi lần xoá dữ liệu phải đăng nhập lại; đăng nhập qua form trong WebView hay bị chặn tạm (429) sau ~5 lượt/15 phút trên cùng IP.
5. **APK debug** ⇒ INS-01 chỉ kết luận được phần manifest/permission.

## 7. Bổ sung lượt 3 (26/11, sau chỉ đạo PM)

- **Vị trí giả bằng công cụ Android (không cài APK lạ):** `adb root` (emulator userdebug) rồi `cmd location providers add-test-provider gps` + `set-test-provider-enabled gps true` + `set-test-provider-location gps --location <lat>,<lon> --accuracy 5` **CHẠY ĐƯỢC**; `navigator.geolocation.getCurrentPosition` trong WebView trả đúng điểm đặt (21,0991/105,8965) với `accuracy` 2000 m (độ chính xác do provider mạng). Ghi nhận: gọi bằng shell thường (uid 2000) bị `SecurityException: not allowed to perform MOCK_LOCATION`, phải `adb root` trước. Đây là nguyên nhân GEO-03 trước đó không chạy được (không phải app).
- **Đổi geofence bằng API:** `PUT /api/v1/courses/{id}/settings` bằng `teacher@` với `attendance.geofence` trả 200 và **có hiệu lực** (GET sau đó trả đúng tâm mới). Đã trả lại QCVPSC về `21,0285/105,8542 r200` sau khi thử. Ghi nhận: `GET /courses/{id}` trả `attendance` ở **khoá cấp ngoài** `course.attendance`, không nằm trong `settings` — lý do lượt trước tôi kết luận sai là «PUT không có hiệu lực».
- **Chưa hoàn tất trong lượt 3:** phiên app `sv.qc2` không giữ được sau `pm clear`/đăng nhập lại (đăng nhập API trả 200 nhưng màn vẫn ở trang giới thiệu ⇒ không vào được `/checkin` để chấm lại GEO). Cần một trong: (a) giữ app đã đăng nhập rồi mới chạy ca, (b) cho phép dùng form đăng nhập trong WebView với tài khoản không bị chặn 429, (c) dev xác nhận luồng `POST /auth/login` + tải lại trang có tự vào app không. Vì vậy GEO-03/05 và các ca cần một lượt điểm danh mới **vẫn chưa có kết quả**.
- **TalkBack:** chưa bật trong lượt này; cần `settings put secure enabled_accessibility_services com.google.android.marvin.talkback/com.google.android.marvin.talkback.TalkBackService` (kiểm gói TalkBack có trên AVD `google_apis`) — xếp lượt sau.

## 6. Lỗi phát hiện
- **DEF-R10.8-05 (Back native)**: ĐÓNG — xác nhận trên ep-api36.
- **DEF-R10.8-07 (mới, Thấp)**: cỡ chữ 2,0× làm dock «Điểm danh» bị thanh điều hướng đè 45 % trên `/checkin`.

