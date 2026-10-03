import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

// Cờ thời điểm dựng (SRS FEAT-ui-foundation 8.1): công cụ dev (`page.dev.tsx`, /dev/ui, /dev/data) chỉ là route khi `next dev`
// hoặc NEXT_PUBLIC_DEV_TOOLS=1; bản như production không biên dịch chúng.
const base = ["tsx", "ts", "jsx", "js"];

// Bộ chọn "Tài khoản mẫu" của /login chỉ có ở bản NEXT_PUBLIC_DEV_TOOLS=1: bản khác thay mô-đun bằng mô-đun rỗng ngay lúc phân giải
// (nhánh chết + import động vẫn phát chunk), nên build thường không chứa chữ nào của nó.
const loginChoices = process.env.NEXT_PUBLIC_DEV_TOOLS === "1" ? "./src/app/login/LoginChoices.tsx" : "./src/app/login/LoginChoices.off.ts";

const config = (phase: string): NextConfig => ({
  turbopack: { resolveAlias: { "@ep/login-choices": loginChoices } },
  output: "standalone",
  // Không để Next tự sinh AGENTS.md / CLAUDE.md trong frontend/ (luật agent nằm ở CLAUDE.md gốc).
  agentRules: false,
  devIndicators: false,
  pageExtensions: phase === PHASE_DEVELOPMENT_SERVER || process.env.NEXT_PUBLIC_DEV_TOOLS === "1" ? [...base, "dev.tsx"] : base,
});

export default config;
