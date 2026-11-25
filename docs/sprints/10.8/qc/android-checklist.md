# Checklist Android (US-AS-11 / US-AS-12, chạy tay) — sprint 10.8

Người chạy: QC + chủ dự án (máy thật). Mỗi bước ghi **PASS / FAIL / KHÔNG KIỂM ĐƯỢC** (+ lý do) vào bảng «Kết quả» cuối tệp. Không dán khoá, mã mời, token, toạ độ thật vào báo cáo. Ảnh chụp màn hình dùng tài khoản mẫu (`sv.gioi@edupilot.local`), đặt tên `shots-android/AC-n-<mô tả>.png`.

## Chuẩn bị
| Mục | Yêu cầu |
| --- | --- |
| Máy | (A) **máy thật** Android ≥ 9 có bootloader khoá + StrongBox / TEE (ghi hãng, model, bản Android, bản WebView); (B) máy thật thứ hai nếu có; (C) **emulator** Android Studio (API 34, Google APIs) |
| Mạng | Điện thoại và máy dựng cùng mạng, **hoặc** đường hầm HTTPS (Cloudflare quick tunnel) trỏ tới `ep-qc` (HTTPS bắt buộc cho `getUserMedia`, Keystore attest, WebCrypto) |
| Dựng APK | `mobile/README.md`: một lệnh «đặt `APP_SERVER_URL` → `cap sync` → dựng → `adb install`» (P-108-9a); `APP_SERVER_URL` = gốc web hiện tại; **đổi host tunnel ⇒ dựng lại + cài lại APK** |
| Khoá ký | APK ký bằng khoá của chủ dự án (ngoài repo). QC không đọc kho khoá; chỉ so **vân tay chứng chỉ** với `ATTESTATION_CERT_SHA256` trên gateway |
| Dữ liệu | Lớp QC-AS có `self_checkin` bật, `rotate_seconds=10`; một buổi; TEACHER mở màn chiếu trên laptop (1280×720 trở lên); SV `sv.gioi`, `sv.kha`; lớp B bật `require=APP`; lớp C bật `geofence` |
| Truy vết | Mỗi bước ghi vào DB `attendance_checkins` (đọc bằng `psql`): `channel`, `assurance`, `flags`, `device_id` |

## Bước kiểm
| Mã | Bước | Mong đợi | Ghi chú truy vết |
| --- | --- | --- | --- |
| **AC-0** | Dựng + cài APK theo `mobile/README.md` (máy A, rồi emulator) | Lệnh một bước chạy; app mở được màn đăng nhập của web (`server.url`); không cảnh báo «ứng dụng không an toàn»; quyền hỏi: **chỉ** camera (và vị trí khi lớp bật) | US-AS-11 AC1; P-108-9 |
| AC-1 | Đăng nhập `sv.gioi` trong app | Đăng nhập thường (cookie WebView); đóng / mở lại app giữ phiên; không lưu mật khẩu ngoài cơ chế cookie | AC7 |
| AC-2 | TA / TEACHER đăng nhập trong app, mở `/checkin` | Màn chặn chuẩn như web; `/attendance` dùng bình thường | AC7 |
| **AC-3** | SV mở `/checkin` (không `#token`), bấm **«Quét mã điểm danh»**, quét QR trên màn chiếu | Xin quyền camera lần đầu; khung camera vuông; thấy QR → «Đã điểm danh hh:mm» trong ≤ 3 s; camera tắt (đèn camera tắt); DB: `channel=APP`, `assurance=3` (máy A) | AC3, AC5 |
| AC-3b | Quét QR **lạ** (QR bất kỳ, QR trang khác, QR của lớp khác không hợp lệ) trong khung quét | Bỏ qua im lặng, tiếp tục quét; không gọi API | AC3 |
| AC-3c | Quét QR bằng **camera hệ thống** của máy (không qua nút trong app) | Liên kết mở **trình duyệt**, không vào app; lượt là lượt web (`channel=WEB`, mức 1–2; `NOT_APP` nếu lớp `require=APP`); `mobile/README.md` có dòng «Dùng nút Quét mã điểm danh trong app» | AC3b |
| **AC-4** | Máy **không có** `BarcodeDetector`/`qr_code` (WebView cũ; hoặc emulator không hỗ trợ) | Không hiện nút quét; hiện ô nhập mã ngắn ngay; gõ mã ngắn trên màn chiếu → «Đã điểm danh»; DB `channel=CODE` | AC3a |
| AC-4b | Từ chối quyền camera (Cài đặt → Quyền) | Ô nhập mã + «Chưa cấp quyền camera.»; điểm danh bằng mã thành công | AC3a |
| AC-4c | Camera đang dùng bởi app khác / `getUserMedia` lỗi | Rơi về ô nhập mã, không treo | AC6 |
| **AC-5** | Lần đầu mở app: đăng ký thiết bị (nonce → chuỗi chứng thực) — máy thật A, APK ký đúng | `POST /me/devices` `201`; `user_devices` có hàng `kind=ANDROID_KEY`, `attested=true`, `boot_verified=true`, `security_level` StrongBox/TEE | US-AS-12 AC1–2 |
| AC-5b | Máy A **mở khoá bootloader** (nếu chủ dự án đồng ý, hoặc máy thử riêng) hoặc máy có `boot_verified=false` | Đăng ký `201` với `boot_verified=false`; điểm danh: mức 2 + `DEVICE_UNTRUSTED` (không chặn) | AC4–5 |
| **AC-6** | Điểm danh trên máy A (đã đăng ký) | `assurance=3`, không cờ; Staff không thấy chip «Chưa xác minh» | AC5 |
| **AC-7** | **Emulator**: đăng ký thiết bị | `422 ATTESTATION_FAILED` (gốc phần mềm); app **vẫn ký bằng khoá Keystore** và gửi `WEB_KEY` (DER); điểm danh `201`, **mức 2**, không `DEVICE_UNTRUSTED` (P-108-5); không báo lỗi cho SV | AC4; P-108-5 |
| AC-7b | Ngắt mạng tới Google (danh sách thu hồi) lúc đăng ký lần đầu | `503 ATTESTATION_UNAVAILABLE`; app vẫn điểm danh (mức 2); thử đăng ký lại sau | US-AS-12 AC3 |
| **AC-8** | APK **ký bằng khoá khác** (dựng debug rồi cài) | Đăng ký `422 ATTESTATION_FAILED` (gói/vân tay lệch); điểm danh vẫn `201` (mức 2 nhờ `WEB_KEY`); **không** đạt mức 3 | US-AS-12 AC4 |
| AC-9 | Lớp B `require=APP`: SV dùng **web trên điện thoại** (Chrome) | Lượt `NOT_APP`, vẫn `201` (SV không thấy gì khác); máy A trong app: không `NOT_APP` | P-108-6 |
| AC-9b | Máy A + lớp B: **bootloader mở / `GEO_MOCK`** | Không `NOT_APP` (đã có `DEVICE_UNTRUSTED` / `GEO_MOCK`) | P-108-6 |
| **AC-10** | Lớp C có geofence; SV trong vùng (đứng thật) và ngoài vùng (cách xa ≥ bán kính+100 m); lần 3 từ chối quyền vị trí | Trong vùng: không cờ; ngoài: `GEO_OUT`; từ chối: `GEO_DENIED`; mọi ca vẫn «Đã điểm danh»; dòng «Lớp này kiểm tra vị trí khi điểm danh.» hiện **trước** hộp xin quyền | US-AS-10 |
| **AC-11** | Bật «vị trí giả» (Cài đặt nhà phát triển → Chọn ứng dụng vị trí giả, dùng app GPS giả) | Plugin báo `isMock=true` → cờ `GEO_MOCK`, `assurance ≤ 2`, vẫn `201`; SV không thấy | US-AS-12 AC5 |
| AC-12 | Hai tài khoản (`sv.gioi`, `sv.kha`) điểm danh **trên cùng một điện thoại** (đăng xuất / đăng nhập trong app) | Cùng `device_id` ⇒ `DEVICE_SHARED` cả hai lượt (Staff thấy ở lưới) | US-AS-04 AC4 |
| AC-13 | Đổi điện thoại (máy A → máy B) cho `sv.gioi` | Lượt đầu máy B: `DEVICE_NEW`; lượt sau: không | US-AS-04 AC5 |
| AC-14 | Mã đã hết hạn (quét trễ > 2 bước) trong app | «Mã đã hết hạn…»; nút `Quét mã điểm danh` ở lại; **không** bị đăng xuất, không refresh phiên (P-108-11a) | US-AS-09 AC4 |
| AC-15 | Mất mạng giữa lúc điểm danh (bật chế độ máy bay) rồi bật lại | «Chưa có mạng.» + `Thử lại`; gửi lại an toàn; một lượt trong DB | US-AS-03 AC4 |
| AC-16 | Xoay màn hình, nút Back, đưa app xuống nền khi đang quét | Camera dừng khi ra nền (`visibilitychange`); trở lại không treo; Back thoát khung quét | AC3 |
| AC-17 | Đo nhanh: từ lúc quét tới «Đã điểm danh» (stopwatch ×5) | Ghi trung vị / lớn nhất; mục tiêu ≤ 3 s (DESIGN §14.8.2) | hiệu năng |
| AC-18 | Giao diện 375 px: chụp mọi trạng thái trong app (quét, xong, nhập mã, lỗi, 429) | Một màn; nút ≥ 44 px; không tràn ngang; chữ UI theo luật cứng PO | US-AS-09 AC7 |
| AC-19 | Gỡ app, cài lại | Khoá Keystore mới → `DEVICE_NEW` một lần; khoá cũ không dùng được | US-AS-04 |

## Bảng «Kết quả» (điền khi chạy)
| Mã | Máy / bản Android / WebView | Kết quả | Ghi chú / ảnh |
| --- | --- | --- | --- |
| AC-0 | | | |
| (tiếp tục cho mọi mã trên) | | | |

## Ghi chú cho QC
- Mức 3 **chỉ** chứng minh được bằng chuỗi chứng chỉ thật (máy thật hoặc `testdata` của Go test); emulator không đạt mức 3 và **không** là lỗi.
- Không có máy thật ⇒ AC-5, AC-6, AC-9b, AC-11, AC-13, AC-19 ghi **KHÔNG KIỂM ĐƯỢC** (không tính FAIL), kèm lý do; US-AS-11 vẫn có thể PASS theo AC8 của chính nó.
- iOS / Play Store / ML Kit / BLE / NFC ngoài phạm vi.

## Chốt (P-108-12, PM)
Chủ dự án **không có máy Android thật**: AC-5, AC-5b, AC-6, AC-9b, AC-11, AC-13, AC-17, AC-19 ghi **KHÔNG KIỂM ĐƯỢC** (không FAIL). Mức 3 kiểm bằng fixture `user_devices` + chuỗi chứng thực mẫu ở `docs/research/poc/attendance-checkin/testdata` (Go test); emulator vẫn kiểm mức 2 (AC-0, 1–4, 7, 8, 9, 12, 14–16, 18).

## Bổ sung P-11-130 (US-AS-15): bước tay trên điện thoại (qc3, 25/11/2026)

Bộ ca đầy đủ ở `tc-mobile-app.md` (mã `TC-MOB-*`); bảng dưới là **trình tự bước tay** để người chạy đi một lượt trên emulator (Pixel, API 35) và iPhone (Simulator / thật), mỗi bước trỏ tới ca có số liệu đo. Ghi **PASS / FAIL / KHÔNG KIỂM ĐƯỢC** vào bảng «Kết quả US-AS-15» cuối mục. Công thức lệnh `R-*` / `S-*` ở `tc-mobile-app.md` mục 3.

| Mã | AC | Bước tay (tóm tắt) | Mong đợi | Ca |
| --- | --- | --- | --- | --- |
| **CL-1** | AC1 | Đặt cỡ máy 360, 375, 390, 411, 430 (R-SIZE); ở mỗi cỡ mở `/checkin` qua 6 trạng thái (đang gửi, đã điểm danh, nhập mã, đang quét, lỗi, 429) và `/attendance` của GV; chụp | không cuộn ngang; nút chính và nội dung trong MỘT màn; chữ không cắt dấu; khung camera vuông rộng bằng panel | UIW-01…10, 17…22 |
| **CL-2** | AC2 | Xoay ngang 667×375, 844×390, 915×412; mở `/checkin`; xoay giữa lúc «Đang gửi» và giữa lúc quét | nút / ô / `Thoát` không bị thanh dưới hay tai thỏ che; khung camera co theo chiều cao; đúng MỘT lượt trong DB; mã đã gõ còn; camera không treo | UIW-11…16, SCN-09, FLW-07 |
| **CL-3** | AC3 | Bật cutout (R-CUT tall / corner / double / hole), thanh cử chỉ và ba nút (R-NAV); iPhone 15 Safari và PWA, dọc và ngang | tiêu đề, thanh trên, thanh dưới, dock không dưới vùng bị che; lề ngang lớn hơn tai thỏ; không dải trắng dưới đáy | SAF-01…10 |
| **CL-4** | AC4 | Cỡ chữ 1,3× rồi lớn nhất (R-FONT); Safari phóng 200 %; iOS chữ lớn nhất | không cuộn ngang; không chữ chồng / cắt; dock không che quá 1/3 màn và không che lỗi; nút chạm được | TXT-01…08 |
| **CL-5** | AC5 | Máy tối (R-DARK-on, S-DARK) + «Override force-dark» + Chrome tối ép buộc | `/checkin` và khung vẫn sáng; AA; ô nhập và `Check` còn; thanh trạng thái đọc được | THM-01…08 |
| **CL-6** | AC6 | Chạm ô «Mã điểm danh» dọc và ngang; gõ `abc 123`; bấm «Đi»; đóng bàn phím; xoay; bấm hai lần | ô và `Điểm danh` thấy cùng lúc trên bàn phím; lỗi không bị che; mã không mất; một POST | KBD-01…12, FLW-06 |
| **CL-7** | AC7 | Back / vuốt khi đang quét; sau «Đã điểm danh»; sau đăng nhập từ liên kết QR; Back ở màn gốc | thoát khung quét (đèn camera tắt); không lượt hai, không lỗi cũ; không về `/login`; app xuống nền không đăng xuất | NAV-01…12 |
| **CL-8** | AC8 | Từ chối camera → cấp lại ở Cài đặt → quay lại; từ chối vị trí (lớp QCMOB2) → cấp lại; chặn vĩnh viễn | cấp lại không cần đăng nhập lại / tải lại; nút `Quét mã điểm danh` trở lại, «Chưa cấp quyền camera.» biến mất; không vòng lặp xin quyền; điểm danh bằng mã luôn được | PRM-01…12, GEO-04, GEO-08, GEO-09, GEO-11 |
| **CL-9** | AC9 | R-NET-slow (400 kbit/s, RTT 2 s) rồi máy bay ngay sau khi thấy QR; quét lại cùng QR; token hết hạn trong lúc chờ | «Đang gửi» ≤ 300 ms; camera đã dừng; một POST/token; ≤ 15 s «Chưa có mạng.» + `Thử lại`; hết hạn → câu hết hạn + nút quét; không đăng xuất | NET-01…12, SCN-10, SCN-13 |
| **CL-10** | AC10 | Home / khoá màn / cuộc gọi khi quét và khi gửi; `am kill`; mở lại sau khi cửa sổ đóng | camera tắt khi ra nền, quay lại màn nhập mã + `Quét mã điểm danh`; đang gửi → đúng kết quả, không gửi lại; đóng → «Đã đóng điểm danh.»; giết nền → phiên còn, không màn trắng | LIF-01…13, SCN-08, SES-02/03 |
| **CL-11** | AC11 | iPhone Safari → Thêm vào Màn hình chính; mở PWA; đăng nhập; thử quét QR bằng camera hệ thống; xoay; Back | tên «EduPilot» + icon thương hiệu; toàn màn hình; thanh trạng thái đọc được; chỉ ô nhập mã; lần đầu phải đăng nhập lại; liên kết quét mở Safari; có đường quay lại; không lời mời cài | PWA-01…16, SAF-06, NAV-08 |
| **CL-12** | AC12 | Bật TalkBack / VoiceOver; duyệt `/checkin` mọi trạng thái; quét; gây lỗi; gây 429 | `h1` đọc đầu; ô có nhãn; nút có tên; thứ tự đọc đúng; «Đang gửi» / «Đã điểm danh» đọc một lần; lỗi (`alert`); 429 không đọc mỗi giây | A11-01…16 |
| **CL-13** | AC13 | `uiautomator dump` hoặc đo Playwright ở 360 và 430 | mọi điều khiển ≥ 44 × 44 px, kề nhau ≥ 8 px; dock không đè thanh dưới | TGT-01…06, ATT-02 |
| **CL-14** | AC14 | Gây mọi lỗi: hết hạn, sai, đóng, tắt, lớp khác, 429, mất mạng, từ chối camera, camera lỗi, WebView không tải; quét chuỗi cấm trong chữ hiển thị | 100 % tiếng Việt có dấu, một câu; 0 mã kỹ thuật / JSON / `undefined` / trang lỗi tiếng Anh | MSG-01…16, INS-10, SCN-12 |
| **CL-15** | AC16 | SV có lớp bật / tắt điểm danh (QCMOB1, QCMOB5, QCMOB2) và `sv.moi`, GV, TA, Admin: đọc thanh điều hướng, «Thêm», «Tìm nhanh», «Hôm nay» (buổi ở 5 mốc giờ); vào `/checkin` từ mục | SV có lớp: mục `Điểm danh` ở cả ba nơi, không lộ trạng thái cửa sổ, không báo «đang mở»; liên kết ở «Hôm nay» từ 15 phút trước giờ đến hết giờ buổi, chỉ lớp bật; app có nút quét, PWA chỉ ô mã; lớp tắt → «Lớp này không điểm danh bằng mã.»; chưa có lớp và Staff không có mục | ENT-01…22 |
| **CL-16** | AC17 | iPhone: điểm danh trong PWA (ô mã), lần sau trong PWA, rồi bằng QR camera hệ thống (Safari); Staff xem lưới | khoá PWA tạo ở lượt đầu trong PWA, `DEVICE_NEW` một lần; lượt sau cùng `device_id`; Safari khoá khác, `DEVICE_NEW` một lần; không `DEVICE_SHARED`; cờ không chặn | PWA-10, 13, 17…23 |

### Kết quả US-AS-15 (điền khi chạy)
| Mã | Máy / bản Android / WebView / iOS | Kết quả | Ghi chú / ảnh |
| --- | --- | --- | --- |
| CL-1 … CL-16 | | | |

Ghi chú: AC15 (phân quyền, không đổi) kiểm bằng ERR-05/07/08, SES-05/06, ATT-08 trong `tc-mobile-app.md`. Ca chỉ máy thật / điều kiện đặc biệt ghi **KHÔNG KIỂM ĐƯỢC** (không tính FAIL) kèm lý do, theo P-108-12.
