# TC-AS11-17 + 6 kiểm dev3 — AVD ep-api36 (Pixel 8, Android 16 / API 36, WebView của Google APIs), VPS e8b2162d, APK debug build từ HEAD (APP_SERVER_URL = VPS, cert ZeroSSL thật)
| Mã | Kết quả | Ghi chú / ảnh (shots-vps/) |
| AC-0 | PASS | app mở trang landing của VPS, không cảnh báo chứng chỉ — AC-0-app-open |
| AC-1 | PASS | đăng nhập sv.gioi; force-stop sau >30 s + mở lại vẫn đăng nhập — AC-1-mo-lai-giu-phien |
| AC-3 | PASS (jsQR) | emulator: BarcodeDetector «không hoạt động» ⇒ app KHÔNG hiện nút quét (đúng sau sửa P-11-131 của dev2). Với shim jsQR cài TRƯỚC khi tải trang: «Đã điểm danh» 1,7 s; DB channel=WEB assurance=2 — AC-3-ket-qua |
| AC-3b | PASS | QR chữ / origin khác / cùng origin khác đường dẫn: tiếp tục quét, 0 POST /checkins — AC-3b-* |
| AC-4 | PASS | không BarcodeDetector ⇒ không nút quét, ô nhập mã sẵn; gõ mã ngắn → «Đã điểm danh» 631 ms; DB channel=CODE — AC-4-* |
| AC-4b | PASS | từ chối camera → «Chưa cấp quyền camera.» + ô mã — AC-4b-tu-choi-camera |
| AC-7 | PASS | POST /me/devices/challenge 200 → POST /me/devices 422 (attest từ chối, gốc phần mềm); điểm danh 201 mức 2 (FINDING 2 đã hết: nonce base64url được chấp nhận, tới bước chứng thực) |
| AC-9 | KHÔNG KIỂM ĐƯỢC phần «trong app không NOT_APP» | lớp B require=APP: NOT_APP vì checkin.go:330 `!attestedOK`; emulator không chứng thực được ⇒ luôn NOT_APP (đúng thiết kế) |
| AC-14 | PASS | mã cũ >2 bước: «Mã đã hết hạn. Quét lại mã trên màn chiếu.», nút «Quét mã điểm danh» ở lại, ở /checkin, không đăng xuất — AC-14-het-han |
| AC-16 | PASS 3/4, FAIL 1/4 | ra nền (Home): khung quét đóng, camera tắt, mở lại không treo; xoay ngang/dọc khi quét: vẫn quét, camera live — AC-16-quet-xoay-ngang. **FAIL: Back (phím, 3 nút, cả tap thật vào nút «Quét») khi đang quét đưa cả APP ra launcher**, trái US-AS-15 AC7(a) «thoát khung quét về ô nhập mã, không rời /checkin». history.state có epScan:true, history.length 3; `history.back()` qua CDP đóng khung quét đúng ⇒ web đúng, vỏ native không gọi webView.goBack() |
| a launcher / splash | PASS | icon launcher là logo thương hiệu (vòng nền xanh nhạt là màu icon chủ đề API 36, không phải nền thương hiệu); khung đầu ≈0,5 s là splash trắng + logo quyển sách mũi tên đỏ, rồi trang web — a-launcher, a-splash-c1 |
| b cutout + cử chỉ/3 nút + xoay | PASS (quan sát) | cutout «tall» + 3 nút: tiêu đề, thanh trên, dock «Điểm danh», thanh dưới nằm dưới status bar và trên thanh 3 nút; ngang: lề trái/phải đệm theo tai thỏ + thanh nút, sidebar hiện ≥720 px CSS. Không đo số bằng env(safe-area-*) — b-portrait-3button-cutout, b-landscape-3button-cutout |
| c chế độ tối | PASS | `cmd uimode night yes`: prefers-color-scheme trong WebView = light, nền #fff, khung sáng — c-dark-checkin |
| d bàn phím /checkin | PASS | Gboard mở: ô + «Điểm danh» thấy cùng lúc (visualViewport 530/867 px), phím «Đi» gửi, lỗi «Mã không đúng…» thấy, mã ABC123 giữ nguyên — d-keyboard-checkin, d-keyboard-go. Ghi chú: Gboard mở chế độ «Try out your stylus» của emulator, tắt bằng settings stylus_handwriting_enabled=0 |
| e mất mạng | PASS | khởi động app khi tắt wifi+data: «Không kết nối được máy chủ.» + nút «Thử lại»; bật mạng + «Thử lại» → vào app còn đăng nhập — e-offline-start, e-retry-ok |
| f quyền Xấp xỉ | PASS | lớp C geofence, chỉ ACCESS_COARSE_LOCATION: lượt CODE assurance=1 flags GEO_WEAK, geo_accuracy_m=2000 — f-geo-approx*. Ghi chú: lần đầu (provider mạng của emulator chưa có vị trí) getCurrentPosition hết 8 s ⇒ GEO_DENIED; app coi hết giờ lấy vị trí là từ chối |
| Chưa chạy | – | AC-15 (mất mạng GIỮA lúc gửi), AC-18 (chụp 375 mọi trạng thái), AC-2 (GV trong app; đã PASS ở dd943280), AC-8, CL-8 cấp lại quyền ở Cài đặt (pm grant làm app khởi động lại) |
