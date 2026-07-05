import type { Metadata } from "next";
import { Toaster } from "sonner";
import "./globals.css";
import PreferenceSyncProvider from "@/components/theme/PreferenceSyncProvider";

export const metadata: Metadata = {
  title: "EduPilot — Trợ giảng AI thông minh",
  description: "Hệ thống Trợ giảng AI EduPilot giúp sinh viên học tập hiệu quả hơn với công nghệ RAG tiên tiến.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className="h-full antialiased"
      suppressHydrationWarning
    >
      <body className="min-h-full flex flex-col font-sans">
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                try {
                  var prefs = JSON.parse(localStorage.getItem('user-preferences') || '{}');
                  var state = prefs.state || {};
                  var p = state.preferences || {};
                  var theme = p.theme || 'SYSTEM';
                  var dark = theme === 'DARK' || (theme === 'SYSTEM' && window.matchMedia('(prefers-color-scheme: dark)').matches);
                  if (dark) {
                    document.documentElement.classList.add('dark');
                  } else {
                    document.documentElement.classList.remove('dark');
                  }
                  var fontSize = p.font_size || 'DEFAULT';
                  document.documentElement.setAttribute('data-font-size', fontSize);
                  if (p.reduce_motion) {
                    document.documentElement.classList.add('reduce-motion');
                  } else {
                    document.documentElement.classList.remove('reduce-motion');
                  }
                } catch (e) {}
              })();
            `
          }}
        />
        <PreferenceSyncProvider>
          {children}
        </PreferenceSyncProvider>
        <Toaster richColors position="top-right" />
      </body>
    </html>
  );
}
