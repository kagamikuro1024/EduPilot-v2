# Kết quả chạy bộ mobile trên Android emulator (qc2, 25–26/11/2026)

AVD `ep-api36` (Pixel 8, Android 16 / API 36, WebView Chrome 133.0.6943.137), APK debug dev dựng từ HEAD cd843318 (có Back native dea5f197), `APP_SERVER_URL` = VPS `https://v2202609419695521098.happysrv.de` (ZeroSSL thật, không cài CA). Mật khẩu đọc từ `~/.config/edupilot-team/secrets/vps-seed.txt`, không ghi vào đây. Kiểm bằng `adb` + CDP vào WebView (qua `cdp.mjs`/`shim.mjs` của dev) — **không** dùng Playwright, **không** build trên Mac.

Thư viện chạy: `docs/regression/evidence-mobile-android/{m.py,vps.py}`; kết quả thô: `results.jsonl` (chép cùng thư mục); ảnh: `shots/` (25 ảnh).

**Đã chạy 26 TC: 21 P, 0 F, 5 N.** Không có F trong lượt này. Phần chưa chạy liệt kê ở §3 (không ghi P).

## 1. P — có bằng chứng

| TC | Kết quả | Bằng chứng |
|---|---|---|
| TC-MOB-INS-04 | P | Cài mới + mở lần đầu: KHÔNG hộp xin quyền nào; dumpsys package: CAMERA granted=false, ACCESS_FINE_LOCATION granted=false, ACCESS_COARSE_LOCATION granted=false; focus = vn.edupilot.app/.MainActivity. |
| TC-MOB-INS-05 | P | resolve-activity → vn.edupilot.app/.MainActivity; nhãn launcher «EduPilot»; biểu tượng là logo sách + mũi tên đỏ (thương hiệu), KHÔNG phải chữ X xanh mặc định của Capacitor. Quan sát tĩnh mục 5.1 bị BÁC BỎ. |
| TC-MOB-INS-06 | P | Chụp 6 ảnh liên tiếp trong ~2 s đầu sau force-stop, ở cả `cmd uimode night no` và `yes`: splash trắng + logo thương hiệu, không chớp đen, không thanh tiêu đề hệ điều hành, không logo Capacitor. |
| TC-MOB-INS-07 | P | Đăng nhập sv.gioi → chờ 75 s (để WebView ghi cookie xuống đĩa) → `adb install -r` → mở lại: PHIÊN CÒN (vào thẳng «Hôm nay», «Tài khoản: Nguyễn Minh Trung»), versionCode 1, lastUpdateTime đổi. Ghi nhận: lượt đầu cài đè ngay sau khi đăng nhập (~10 s) thì phiên MẤT — cookie chưa kịp flush; không tính F vì kỳ vọng là «phiên còn» sau khi app đã dùng bình thường. |
| TC-MOB-INS-10 | P | Tắt mạng (airplane-mode) + force-stop + mở app: hiện trang lỗi cục bộ «Không kết nối được máy chủ.» + nút «Thử lại» (đo DOM: cao đúng 44 px), KHÔNG phải net::ERR_*; bật mạng bấm «Thử lại» → vào «Hôm nay» còn đăng nhập, không khởi động lại app. |
| TC-MOB-INS-11 | P | Chèn liên kết ngoài example.com trong WebView rồi bấm: KHÔNG nạp trong WebView của app — mở Chrome ngoài (topResumedActivity = com.android.chrome/…FirstRunActivity); giao diện EduPilot không bị thay. |
| TC-MOB-INS-12 | P | Cài đặt → Ứng dụng: nhãn «EduPilot»; dumpsys package versionName=1.0, targetSdk 36, minSdk 24; không cảnh báo «xây cho phiên bản Android cũ» (targetSdk 36). |
| TC-MOB-INS-14 | P | `pm get-app-links vn.edupilot.app` rỗng (không xác minh); `query-activities -a VIEW -d https://VPS/checkin` chỉ trả com.google.android.apps.chrome.IntentDispatcher — app KHÔNG xử lý liên kết https của VPS. |
| TC-MOB-SCN-07 | P | Nút «Thoát» và Back cùng đóng khung quét về màn nhập mã, không rời /checkin, camera tắt. |
| TC-MOB-SAF-01 | P | Overlay cutout tall và hole: h1 vẫn y=80 (dưới thanh trạng thái), không dải trắng, không cuộn ngang; nav trong khung nhìn (720 và 718). |
| TC-MOB-SAF-02 | P | Điều hướng cử chỉ (navbar.gestural): viewport cao 744 (inset dưới được trừ), nav 687..744 nằm trên vùng cử chỉ, không khoảng trống bất thường. |
| TC-MOB-SAF-03 | P | Ba nút (navbar.threebutton): viewport 720, nav 663..720 không bị ba nút che. |
| TC-MOB-THM-01 | P | `cmd uimode night yes`: body background = rgb(255,255,255), `color-scheme: light`; ô nhập và nút hiện rõ trên ảnh. |
| TC-MOB-NAV-01 | P | Mở khung quét (shim jsQR vì emulator không có BarcodeDetector — camera ảo virtualscene, app không bị sửa) rồi bấm Back: khung quét đóng, về màn nhập mã («Mã điểm danh» + «Điểm danh»), vẫn ở vn.edupilot.app (KHÔNG ra launcher), camera tắt (dumpsys media.camera: 0 client). Xác nhận bản sửa dea5f197. |
| TC-MOB-NAV-04 | P | Ở «/ » (màn gốc, mở nguội): bấm Back → app xuống nền (topResumedActivity = nexuslauncher), KHÔNG đăng xuất; mở lại vào «/» còn đăng nhập («QC3 mobile QC3M1», «Tài khoản: Nguyễn Minh Trung»). |
| TC-MOB-PRM-01 | P | Thu hồi CAMERA → mở app → `/checkin` (shim jsQR để nút quét hiện) → bấm «Quét mã điểm danh»: hộp quyền hệ thống của Android hiện (GrantPermissionsActivity); chọn Từ chối → khung quét đóng, hiện «Chưa cấp quyền camera.» (chữ amber) + ô «Mã điểm danh» + nút «Điểm danh». |
| TC-MOB-A11-03 | P | Ô nhập có `<label for="_r_i_">Mã điểm danh</label>` thật (không chỉ placeholder «ABC 123»), input id khớp `for`. Câu lỗi dùng aria-describedby: xem PRM-01 (sau từ chối quyền, dòng amber hiện). |
| TC-MOB-A11-16 | P | `document.documentElement.lang` = «vi»; `<title>` = «Điểm danh · EduPilot». |
| TC-MOB-CL-1 | P | 5 cỡ (360/375/390/412/430): scrollWidth = clientWidth (KHÔNG cuộn ngang) ở mọi cỡ; h1 «Điểm danh» y=80 px (dưới thanh trạng thái); ô mã 237..281 (h44); nút «Điểm danh» dock h44; nav đáy trong khung nhìn (651/663/693/738/776). Ghi nhận: `wm size 1080x2340 @420` cho 412 px (làm tròn mật độ), không phải 411. |
| TC-MOB-CL-13 | P | Đo DOM ở 375: ô «Mã điểm danh» cao 44, nút dock «Điểm danh» cao 44, các nút thanh trên cao 44, mục «Thêm» cao 56; dock trên nav (dock 612..656, nav 663..720) không đè. Ở 360/390/412/430 cũng h44 (đo cùng script). |
| TC-MOB-CL-2 | P | Ngang 816×384 (xoay `user_rotation 1`): scrollWidth = clientWidth = 816 (không cuộn ngang); ô mã 277..321; nút «Điểm danh» cao 44; nav 327..384 nằm trong khung nhìn; h1 y=101 (dưới thanh trạng thái). Phần «xoay giữa lúc Đang gửi/quét» và «đúng MỘT lượt» chưa đo (xếp lượt sau). |

## 2. N — không kiểm được, nêu lý do

| TC | Lý do |
|---|---|
| TC-MOB-INS-01 | Chỉ có APK debug (dev dựng, HEAD cd843318) nên không kiểm được phần release: cert ký là Android Debug (SHA-256 c2be7f99…), debuggable=true; kỳ vọng cần APK release + vân tay = ATTESTATION_CERT_SHA256. Phần kiểm được ĐÚNG: package vn.edupilot.app, versionName 1.0, versionCode 1, targetSdk 36, minSdk 24; quyền chỉ INTERNET, CAMERA, ACCESS_FINE_LOCATION, ACCESS_COARSE_LOCATION (+DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION của AndroidX); allowBackup=false; không usesCleartextTraffic; không autoVerify. Quan sát tĩnh cũ «chỉ khai FINE» nay SAI (manifest có cả COARSE). |
| TC-MOB-INS-02 | Bước cần dựng APK bằng `npm run install:device` (Gradle) — Mac chỉ chạy giả lập, build phải trên VPS mà VPS không có Android SDK. Đã kiểm phần sau cài: pm list có vn.edupilot.app; am start -W → Status ok, màn web hiện ≤ 10 s; không hộp cảnh báo «ứng dụng không an toàn». |
| TC-MOB-INS-03 | Cần chạy `npm run install:device` với APP_SERVER_URL sai (build Gradle) — không dựng được APK trên Mac. Đọc mã: kiểm tra HTTPS/APP_SERVER_URL nằm trong bước build (P-108-9a), chưa xác nhận bằng lệnh. |
| TC-MOB-INS-09 | Không có AVD API 24–28 (chỉ ep-api36 API 36). |
| TC-MOB-INS-13 | Cần Chrome đã qua màn chào lần đầu để mở liên kết QR và đăng nhập trong Chrome; Chrome trên AVD còn FirstRunActivity. Phần kiểm được: `am start -a VIEW -d https://VPS/checkin` mở Chrome (không vào app) — đúng kỳ vọng «liên kết KHÔNG vào app». |
## 3. Chưa chạy trong lượt này

Lượt chạy bị cắt vì máy chỉ một giả lập và VPS đang chạy pha 1 song song (phiên `sv.gioi` bị client khác làm mất nhiều lần — xem ghi chú §4). Các nhóm sau **chưa chạy**, xếp lượt sau:

| Nhóm | TC chưa chạy | Vì sao |
|---|---|---|
| 6.D quét QR | SCN-01…06, 08…16 | Emulator không có `BarcodeDetector` ⇒ app ẩn nút quét (đúng thiết kế); chỉ kiểm được đường shim jsQR (đã dùng cho NAV-01/SCN-07). Cần poster QR + shim cho từng ca |
| 6.E vị trí | GEO-01…13 | Cần lớp geofence (QCVPSC) + `adb emu geo fix` + đọc cờ ở API Staff; chưa chạy |
| 6.H vùng an toàn | SAF-04…10 | SAF-04 (ngang + cutout) và các ca iOS/PWA (SAF-05…08) chưa chạy |
| 6.J sáng/tối | THM-02…08 | THM-01 đã P; force-dark, splash tối, iOS chưa chạy |
| 6.L Back | NAV-02, 03, 05…12 | NAV-01/04 đã P; các nhánh còn lại chưa chạy |
| 6.M quyền | PRM-02…12 | PRM-01 P; PRM-02 (cấp lại rồi quay lại) chưa đo được vì phiên bị đăng xuất giữa lượt |
| 6.O vòng đời | LIF-01…13 | chưa chạy |
| 6.Q trợ năng | A11-01, 02, 04…15 | A11-03/16 P; TalkBack chưa bật, axe chạy trên VPS (`ep-vps-test`) |
| 6.U thiết bị | DEV-01…14 | Emulator không chứng thực (DEV-03 một phần đã có từ lượt dev: 422 rồi mức 2) |
| android-checklist | CL-3…CL-12, CL-14…16 | CL-1, CL-2, CL-13 P; còn lại chưa chạy |
| 6.A cài đặt | INS-08 (xong một phần), INS-02/03/09/13 | INS-08 cần gỡ/cài lại + so `user_devices` (API Staff không trả `device_id`); INS-13 cần Chrome qua màn chào |

## 4. Ghi chú môi trường (không tính F)
1. **Phiên `sv.gioi` bị đăng xuất nhiều lần** giữa lượt (`/login?next=/checkin`, «Bạn đã bị đăng xuất») sau force-stop/đổi quyền. Nguyên nhân khả dĩ: một client khác trên VPS đăng nhập cùng tài khoản (pha 1 đang chạy song song) làm refresh token cũ bị thu hồi. INS-07 (cài đè sau 75 s) cho thấy phiên ĐƯỢC giữ khi không có client khác ⇒ không phải lỗi mã.
2. **Đăng nhập bị chặn tạm** («Bạn đã thử quá nhiều lần. Thử lại sau 00:19.») sau ~5 lượt đăng nhập/20 phút trên cùng IP — đúng thiết kế chống dò.
3. **Android Debug cert**: APK là bản debug (chữ ký «Android Debug», `debuggable=true`) nên INS-01 không kết luận được phần release.
4. Quan sát tĩnh §5.1 (`ic_launcher` là logo Capacitor) **SAI**: biểu tượng và splash là logo thương hiệu (ảnh `INS-05-launcher`, `INS-06-*-strip`). Quan sát §5.2 (manifest chỉ khai FINE) cũng **SAI**: có cả `ACCESS_COARSE_LOCATION`.

## 5. Lỗi phát hiện trong lượt này
Không có F mới. **DEF-R10.8-05 (Back native) XÁC NHẬN ĐÃ SỬA**: khung quét mở → Back → đóng khung, về màn nhập mã, vẫn trong app, camera tắt (`NAV-01-back-closes-scanner`).

