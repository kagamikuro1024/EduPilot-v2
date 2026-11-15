# EduPilot — app Android (điểm danh bằng mã)

Vỏ Capacitor 8 nạp ứng dụng web từ `APP_SERVER_URL` (không đóng gói tĩnh, không đổi auth, không đổi route Next.js) và thêm plugin Kotlin `EduPilotAttest` (khoá Keystore + chứng thực).
Dùng nút **Quét mã điểm danh** trong app; quét QR bằng camera hệ thống mở trình duyệt (lượt web).

## Yêu cầu

- JDK 21, Android SDK (`platforms;android-36`, `build-tools;35.0.0`), `ANDROID_HOME` trỏ tới SDK, `adb` khi cài lên máy.
- Kho khoá ký APK của chủ dự án, **ngoài repo**:

```bash
export EDUPILOT_KEYSTORE=~/.edupilot/edupilot-dev.jks   # đường dẫn tới .jks
export EDUPILOT_KEYSTORE_PASSWORD=...                    # + EDUPILOT_KEY_ALIAS (mặc định edupilot), EDUPILOT_KEY_PASSWORD (mặc định = mật khẩu kho)
```

## Một lệnh: đặt `APP_SERVER_URL` → `cap sync` → dựng → `adb install`

```bash
cd mobile && npm ci
APP_SERVER_URL=https://<host-demo> npm run install:device
```

`APP_SERVER_URL` bị đóng vào APK: đổi host thì phải chạy lại lệnh trên (dựng lại APK, cài lại). Chỉ nhận HTTPS (`getUserMedia` và WebCrypto cần ngữ cảnh an toàn; APK không bật cleartext), kể cả khi thử trên emulator: dùng tunnel HTTPS (`cloudflared tunnel --url http://localhost:<cổng>`).

Chỉ dựng: `APP_SERVER_URL=… npm run build` → `android/app/build/outputs/apk/release/app-release.apk`.

## Gắn APK với máy chủ

Máy chủ chỉ chấp nhận chứng thực từ APK có đúng tên gói và vân tay chứng chỉ ký:

```bash
ATTESTATION_PACKAGE=vn.edupilot.app
ATTESTATION_CERT_SHA256=<SHA-256 của chứng chỉ ký, 64 hex>   # apksigner verify --print-certs app-release.apk
```

Để trống thì mọi lượt đăng ký thiết bị trả `ATTESTATION_FAILED` và app điểm danh bằng khoá Keystore ở mức 2.

## Kiểm

- Manifest: `aapt2 dump xmltree --file AndroidManifest.xml app-release.apk` — có `CAMERA`, không `autoVerify`, không `usesCleartextTraffic="true"`.
- Chữ ký: `apksigner verify --print-certs app-release.apk`.
- Kịch bản tay trên emulator / máy thật: `docs/sprints/10.8/qc/android-checklist.md`.
