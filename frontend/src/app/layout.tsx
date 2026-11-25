import type { Metadata, Viewport } from "next";
import { Be_Vietnam_Pro } from "next/font/google";
import "@/shared/styles/tokens.css";
import "@/shared/styles/base.css";
import { OfflineBanner } from "@/shared/data/OfflineBanner";
import { QueryProvider } from "@/shared/data/queryClient";
import { AuthProvider } from "@/shared/session/AuthProvider";
import { EARLY_REFRESH_SCRIPT } from "@/shared/data/earlyRefresh";
import { EARLY_PAINT_SCRIPT } from "@/shared/data/earlyPaint";
import { BRAND } from "@/shared/styles/brand";

// next/font tự lưu font cùng máy chủ và phát @font-face "Be Vietnam Pro" (không gọi Google lúc chạy). KHÔNG đặt font-family qua
// className: --ep-font ở tokens.css (có "Noto Sans" dự phòng) mới là chuỗi font duy nhất của trang.
const beVietnamPro = Be_Vietnam_Pro({
  weight: ["400", "600", "700"], // bỏ 500 để nhẹ ~25 KB phông (góp ý #26): chữ 500 hiển thị bằng 400
  subsets: ["vietnamese", "latin"],
  display: "block", // US-PU-06 (đề xuất #14): swap làm chữ rộng ra sau khi phông về ⇒ ứng viên LCP mới
  variable: "--font-be-vietnam-pro",
});

export const metadata: Metadata = {
  title: { default: "EduPilot", template: "%s · EduPilot" },
  icons: { icon: "/brand/favicon.svg", apple: [{ url: "/brand/icon-192.png", sizes: "192x192", type: "image/png" }] },
  // PWA iPhone «Thêm vào Màn hình chính» (US-AS-15 AC11): tên, chạy toàn màn, thanh trạng thái sáng khớp thanh trên trắng
  appleWebApp: { capable: true, title: "EduPilot", statusBarStyle: "default" },
};

// US-AS-15 AC3: nội dung tràn dưới tai thỏ / thanh cử chỉ, CSS chừa bằng env(safe-area-inset-*); bàn phím ảo co khung nhìn (Chrome Android) để ô nhập + dock thấy cùng lúc (AC6)
export const viewport: Viewport = { width: "device-width", initialScale: 1, viewportFit: "cover", interactiveWidget: "resizes-content", themeColor: BRAND.canvas };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="vi" className={beVietnamPro.variable} suppressHydrationWarning>
      <head>
        {/* U1 (US-P10-08, TL-2): làm mới phiên chạy NGAY khi HTML được phân tích, song song với tải JS (≈ 0,6 s trên đường LCP) thay vì sau khi React sẵn sàng.
            Cùng điều kiện với AuthProvider (`/` chưa từng đăng nhập thì không gọi), cùng khoá `ep-refresh` giữa các tab; `authSession.doRefresh` dùng lại đúng MỘT lần
            rồi xoá (cookie xoay vòng: dùng lại token là bị thu hồi phiên). Token vẫn chỉ nằm trong bộ nhớ. */}
        <script
          dangerouslySetInnerHTML={{
            __html: EARLY_REFRESH_SCRIPT,
          }}
        />
        <script dangerouslySetInnerHTML={{ __html: EARLY_PAINT_SCRIPT }} />
      </head>
      <body>
        <QueryProvider>
          <OfflineBanner />
          <AuthProvider>{children}</AuthProvider>
        </QueryProvider>
      </body>
    </html>
  );
}
