import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

// Cờ thời điểm dựng (SRS FEAT-ui-foundation 8.1): công cụ dev (`page.dev.tsx`, /dev/ui, /dev/data) chỉ là route khi `next dev`
// hoặc NEXT_PUBLIC_DEV_TOOLS=1; bản như production không biên dịch chúng.
const base = ["tsx", "ts", "jsx", "js"];

const config = (phase: string): NextConfig => ({
  output: "standalone",
  // CSS vào thẳng HTML: lần vẽ đầu không chờ 3 tệp CSS (US-PU-06: LCP / FCP).
  experimental: { inlineCss: true },
  // Trang nhận liên kết một lần (xác minh email, đặt lại mật khẩu, lời mời): không gửi Referer, không cache (SRS FEAT-account-security 6.3).
  async headers() {
    // 10.8 (ARCHITECTURE §11.7): CSP + Permissions-Policy cho MỌI route (plugin app gọi được từ mọi trang cùng tên miền). Chỉ bản dựng production: `next dev` cần `unsafe-eval`.
    // `unsafe-inline` vì App Router chèn script nội tuyến khi hydrate (nonce buộc mọi trang kết xuất theo request); KHÔNG `unsafe-eval`, KHÔNG nguồn script ngoài.
    const api = (process.env.NEXT_PUBLIC_API_URL ?? "").replace(/\/+$/, "");
    const security =
      phase === PHASE_DEVELOPMENT_SERVER
        ? []
        : [
            {
              key: "Content-Security-Policy",
              value: `default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'${api ? ` ${api}` : ""}; media-src 'self' blob:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'`,
            },
            { key: "Permissions-Policy", value: "camera=(self), geolocation=(self)" },
          ];
    const h = [{ key: "Referrer-Policy", value: "no-referrer" }, { key: "Cache-Control", value: "no-store" }];
    return [
      ...(security.length ? [{ source: "/:path*", headers: security }] : []),
      // Trang ứng dụng (US-PU-06 AC2): Next tiền kết xuất tĩnh nên tự gắn `s-maxage=31536000` (cache chung). HTML chỉ là khung chưa có dữ liệu người dùng,
      // nhưng ứng dụng đăng nhập không được để CDN / proxy giữ bản nào: `private, no-cache`. Tệp tĩnh `_next/` và tệp có đuôi giữ cache dài của Next.
      { source: "/((?!_next/|.*\\..*).*)", headers: [{ key: "Cache-Control", value: "private, no-cache" }] },
      ...["/verify-email", "/reset-password", "/invite/:path*", "/join", "/join/:path*"].map((source) => ({ source, headers: h })),
    ];
  },
  // Không để Next tự sinh AGENTS.md / CLAUDE.md trong frontend/ (luật agent nằm ở CLAUDE.md gốc).
  agentRules: false,
  devIndicators: false,
  pageExtensions: phase === PHASE_DEVELOPMENT_SERVER || process.env.NEXT_PUBLIC_DEV_TOOLS === "1" ? [...base, "dev.tsx"] : base,
});

export default config;
