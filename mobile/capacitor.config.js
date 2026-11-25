// Cấu hình Capacitor. `APP_SERVER_URL` được đọc lúc `cap sync` và đóng vào APK: đổi host thì phải dựng lại APK (P-108-9).
const url = process.env.APP_SERVER_URL;
if (!url) throw new Error("Đặt APP_SERVER_URL (vd. https://edupilot.example.com) trước khi chạy cap sync.");
const host = new URL(url).host;

/** @type {import('@capacitor/cli').CapacitorConfig} */
module.exports = {
  appId: "vn.edupilot.app",
  appName: "EduPilot",
  webDir: "www",
  server: {
    url,
    cleartext: false, // không HTTP thường
    allowNavigation: [host], // chỉ host của ứng dụng
    errorPath: "error.html", // WebView không tải được server.url → trang lỗi cục bộ có «Thử lại»
  },
  android: { allowMixedContent: false },
  plugins: {
    // targetSdk 36 ⇒ edge-to-edge bắt buộc: "css" = decor view được đệm theo thanh trạng thái / điều hướng / cắt góc / bàn phím (trang không có viewport-fit=cover);
    // style LIGHT = nền sáng ⇒ biểu tượng thanh trạng thái và thanh điều hướng màu tối.
    SystemBars: { insetsHandling: "css", style: "LIGHT" },
  },
};
