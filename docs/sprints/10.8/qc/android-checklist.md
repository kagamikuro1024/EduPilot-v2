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
