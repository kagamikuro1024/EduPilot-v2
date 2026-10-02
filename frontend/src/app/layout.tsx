import type { Metadata } from "next";
import { Be_Vietnam_Pro } from "next/font/google";
import "@/shared/styles/tokens.css";
import "@/shared/styles/base.css";
import "@/shared/styles/base.css";

// next/font phát @font-face "Be Vietnam Pro", khớp --ep-font trong tokens.css.
const beVietnamPro = Be_Vietnam_Pro({
  weight: ["400", "500", "600", "700"],
  subsets: ["vietnamese", "latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "EduPilot", template: "%s · EduPilot" },
  icons: { icon: "/brand/favicon.svg" },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="vi" className={beVietnamPro.className}>
      <body>{children}</body>
    </html>
  );
}
