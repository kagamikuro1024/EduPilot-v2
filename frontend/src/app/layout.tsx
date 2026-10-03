import type { Metadata } from "next";
import { Be_Vietnam_Pro } from "next/font/google";
import "@/shared/styles/tokens.css";
import "@/shared/styles/base.css";
import { OfflineBanner } from "@/shared/data/OfflineBanner";
import { QueryProvider } from "@/shared/data/queryClient";

// next/font tự lưu font cùng máy chủ và phát @font-face "Be Vietnam Pro" (không gọi Google lúc chạy). KHÔNG đặt font-family qua
// className: --ep-font ở tokens.css (có "Noto Sans" dự phòng) mới là chuỗi font duy nhất của trang.
const beVietnamPro = Be_Vietnam_Pro({
  weight: ["400", "600", "700"], // bỏ 500 để nhẹ ~25 KB phông (góp ý #26): chữ 500 hiển thị bằng 400
  subsets: ["vietnamese", "latin"],
  display: "swap",
  variable: "--font-be-vietnam-pro",
});

export const metadata: Metadata = {
  title: { default: "EduPilot", template: "%s · EduPilot" },
  icons: { icon: "/brand/favicon.svg" },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="vi" className={beVietnamPro.variable}>
      <body>
        <QueryProvider>
          <OfflineBanner />
          {children}
        </QueryProvider>
      </body>
    </html>
  );
}
