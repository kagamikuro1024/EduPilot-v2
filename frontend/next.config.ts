import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

// Cờ thời điểm dựng (SRS FEAT-ui-foundation 8.1): công cụ dev (`page.dev.tsx`, /dev/ui, /dev/data) chỉ là route khi `next dev`
// hoặc NEXT_PUBLIC_DEV_TOOLS=1; bản như production không biên dịch chúng.
const base = ["tsx", "ts", "jsx", "js"];

// Cổng dán token dev (US-PU-04 AC9): bản không bật NEXT_PUBLIC_DEV_AUTH=1 thay mô-đun bằng mô-đun rỗng ngay lúc phân giải
// (lazy + nhánh chết không đủ: chunk vẫn được phát ra), nên build thường không chứa chữ nào của cổng.
const gate = process.env.NEXT_PUBLIC_DEV_AUTH === "1" ? "./src/shared/session/TokenGate.tsx" : "./src/shared/session/TokenGate.off.ts";

const config = (phase: string): NextConfig => ({
  turbopack: { resolveAlias: { "@ep/token-gate": gate } },
  output: "standalone",
  // Không để Next tự sinh AGENTS.md / CLAUDE.md trong frontend/ (luật agent nằm ở CLAUDE.md gốc).
  agentRules: false,
  devIndicators: false,
  pageExtensions: phase === PHASE_DEVELOPMENT_SERVER || process.env.NEXT_PUBLIC_DEV_TOOLS === "1" ? [...base, "dev.tsx"] : base,
});

export default config;
