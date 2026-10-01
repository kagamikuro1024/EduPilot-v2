import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // Không để Next tự sinh AGENTS.md / CLAUDE.md trong frontend/ (luật agent nằm ở CLAUDE.md gốc).
  agentRules: false,
  devIndicators: false,
};

export default nextConfig;
