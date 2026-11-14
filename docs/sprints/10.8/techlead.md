# Sprint 10.8: kênh Tech Lead

## TL-1 (sprint 10.8): điểm danh tự phục vụ chống gian lận, web + app điện thoại

PM giao (2026-11-14), có bổ sung của chủ dự án: **sẽ có app điện thoại**. Cần:
- mối đe doạ;
- so sánh cơ chế;
- thiết kế nhiều lớp (chỉ gắn cờ, giảng viên sửa tay + audit);
- kiến trúc theo luật repo;
- so sánh PWA / Capacitor / React Native / native;
- PoC;
- chia lát + câu hỏi cho chủ dự án.

Bản đầy đủ có nguồn: `docs/research/2026-11-14-attendance-anti-cheat.md`. PoC: `docs/research/poc/attendance-checkin/` (`go run .`).

**TL trả lời:**

1. **Thiết kế: 5 lớp, chỉ gắn cờ.**
   - QR xoay vòng 10 s. Mã là `window_id.step.HMAC`, khoá suy ra từ `ATTENDANCE_TOKEN_KEY`, không lưu trạng thái, nhận bước hiện tại + bước trước.
   - Khoá thiết bị ký từng lượt. Web dùng WebCrypto P-256 không xuất được. App dùng Keystore có Key Attestation.
   - Vị trí / IP trường: tuỳ chọn theo lớp.
   - Đối chiếu do người làm: số đầu người, vòng 2 ngẫu nhiên, gọi tên ngẫu nhiên.
   - Cờ bất thường (§4.3 của research). Sinh viên không bao giờ thấy cờ.

   Chỉ từ chối khi: mã sai / hết hạn (`422`), cửa sổ đóng (`409`), không thuộc lớp (`403`).
2. **Không làm** selfie / khuôn mặt (sinh trắc, Luật 91 Đ31 k4). **Không làm** chặn chụp màn hình: QR nằm trên máy chiếu nên vô ích. **Không** dùng NFC: 200 người phải xếp hàng chạm.
3. **Lược đồ: `00024_attendance_checkin.sql`.**
   - Chỉ `CREATE`: `checkin_windows`, `user_devices` (không gắn user), `attendance_checkins` (UNIQUE `(course_id, window_id, student_id)`, index đầu `course_id`, không lưu toạ độ / IP).
   - Chính sách lớp nằm ở `courses.settings.attendance`, qua `PUT …/settings` có sẵn.
   - Đóng cửa sổ chạy một câu `INSERT … ON CONFLICT DO NOTHING` vào `attendance_records`. Không ghi đè dòng giảng viên đã sửa, nên lưới tay và khoá `version` giữ nguyên.
4. **Đường nóng không cần Redis.** Kiểm mã bằng HMAC, chữ ký ký trên mã tươi (không cần nonce), khử trùng bằng UNIQUE. Redis chỉ giữ nonce đăng ký thiết bị app (`ep:dev:chal:{uid}`, `GETDEL`).
   - PoC: 200 lượt kiểm mã + chữ ký song song mất 1,7 ms, đo trên MacBook Apple Silicon.
   - Màn chiếu poll `display` mỗi bước (1 req / 10 s) thay cho SSE, vì SSE cần bộ đếm nhịp theo cửa sổ trên N gateway.
5. **App: Capacitor tải web từ máy chủ (`server.url`), Android trước, APK sideload.**
   - Tái dùng 100% Next.js + `shared/`. Không đổi auth vì cookie vẫn cùng origin.
   - Cộng thêm: khoá phần cứng có Key Attestation (Go kiểm bằng `crypto/x509` + `encoding/asn1`; PoC đọc đúng `VERIFIED` / `UNVERIFIED` trên chuỗi chứng chỉ thật) và `Location.isMock()`.
   - Không chọn đóng gói tĩnh: phải đổi 9 route động và auth cross-origin (L, luật DỪNG). Không chọn React Native / native: phải viết lại UI.
   - iOS dùng web/PWA, vì App Attest cần gói 99 USD/năm.
   - Play Integrity không cần: Key Attestation miễn phí, không có hạn mức.
6. **Lát:** chi tiết ở §8 research.

   | Lát | Nội dung | Cỡ |
   | --- | --- | --- |
   | L1 | Backend lõi | L |
   | L2 | Màn giảng viên + màn chiếu | M |
   | L3 | Màn sinh viên web + PWA | S |
   | L4 | Geofence / CIDR | S |
   | L5 | App Android + Key Attestation | M, rủi ro → L |
   | L6 | BLE (tuỳ chọn, cắt được) | M |
   | L7 | E2E + k6 + seed | M |

**Rủi ro:**
- **Demo cần HTTPS cho điện thoại.** WebCrypto, Geolocation và camera chỉ chạy trong ngữ cảnh an toàn. Android App Links cũng cần tên miền HTTPS xác minh được (Q8).
- Báo nhầm `DEVICE_NEW` khi sinh viên xoá dữ liệu trình duyệt.
- Ngưỡng `TOKEN_LATE` cần chỉnh theo dữ liệu thật.
- Gradle / WebView có thể tốn thời gian hơn dự kiến ở L5.

**→ PM (chủ dự án quyết):** Q1–Q9 ở §9 research.
- Q1: SV chưa check-in → `ABSENT`?
- Q2: lớp chọn "chỉ app" thì lượt từ web ghi + cờ hay từ chối?
- Q3: đổi máy chỉ gắn cờ hay phải duyệt?
- Q4: duyệt `uqr` (MIT), Capacitor 8 (MIT), ML Kit dự phòng, BLE nếu làm L6.
- Q5: geofence + câu thông báo dùng vị trí.
- Q6: iOS 99 USD/năm?
- Q7: Play Console 25 USD?
- Q8: HTTPS cho demo.
- Q9: thời hạn giữ dữ liệu.

Đề xuất `P-108-1` ở §11 research.
