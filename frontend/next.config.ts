import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

// Cờ thời điểm dựng (SRS FEAT-ui-foundation 8.1): công cụ dev (`page.dev.tsx`, /dev/ui, /dev/data) chỉ là route khi `next dev`
// hoặc NEXT_PUBLIC_DEV_TOOLS=1; bản như production không biên dịch chúng.
const base = ["tsx", "ts", "jsx", "js"];

const config = (phase: string): NextConfig => ({
  output: "standalone",
  // Trang nhận liên kết một lần (xác minh email, đặt lại mật khẩu, lời mời): không gửi Referer, không cache (SRS FEAT-account-security 6.3).
  async headers() {
    const h = [{ key: "Referrer-Policy", value: "no-referrer" }, { key: "Cache-Control", value: "no-store" }];
    return ["/verify-email", "/reset-password", "/invite/:path*", "/join", "/join/:path*"].map((source) => ({ source, headers: h }));
  },
  // Không để Next tự sinh AGENTS.md / CLAUDE.md trong frontend/ (luật agent nằm ở CLAUDE.md gốc).
  agentRules: false,
  devIndicators: false,
  pageExtensions: phase === PHASE_DEVELOPMENT_SERVER || process.env.NEXT_PUBLIC_DEV_TOOLS === "1" ? [...base, "dev.tsx"] : base,
});

export default config;
