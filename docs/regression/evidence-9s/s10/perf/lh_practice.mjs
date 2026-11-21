// TC-12-17: Lighthouse (performance) /practice và /practice/[id] trên bản build production ep-qc (https://localhost:943), SV sv.kha, lớp QD10A.
// Dùng Chromium của Playwright + lighthouse 12.6.1 của @lhci (qc5); phiên đăng nhập giữ trong cùng profile; LCP: throttling devtools, TBT: simulate (như CI .github/workflows/ci.yml); ngưỡng không nới.
import { createRequire } from "node:module";
import { writeFileSync } from "node:fs";
const req = createRequire("/Users/kuro/Documents/TA_Agent_qc5/frontend/package.json");
const { chromium } = req("@playwright/test");
const lh = (await import("/Users/kuro/Documents/TA_Agent_qc5/node_modules/.pnpm/lighthouse@12.6.1_supports-color@7.2.0/node_modules/lighthouse/core/index.js")).default;
const BASE = "https://localhost:943", CID = process.env.CID, AID = process.env.AID, PW = process.env.QC_PW, RUNS = Number(process.env.RUNS || 3);
const ctx = await chromium.launchPersistentContext(process.env.PROFILE ?? "/Users/kuro/.config/edupilot-team/qc3-work/lh-profile", { headless: true, ignoreHTTPSErrors: true, args: ["--remote-debugging-port=9333", "--ignore-certificate-errors", "--no-sandbox"] });
await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID);
const page = await ctx.newPage();
console.log("launch ok"); await page.goto(BASE + "/login"); await page.locator("input[type=email]").fill("sv.kha@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
const b = page.getByRole("button", { name: "Đã hiểu" }); if (!process.env.NOACK && await b.count()) await b.click().catch(() => {});
console.log("login ok", page.url()); await page.goto(BASE + "/practice"); await page.waitForTimeout(2000); console.log("practice ok");
const med = (a) => a.slice().sort((x, y) => x - y)[Math.floor(a.length / 2)];
const out = [];
for (const path of (process.env.PATHS ? process.env.PATHS.split(",") : ["/practice", `/practice/${AID}`])) {
  for (const method of ["devtools", "simulate"]) {
    const lcp = [], tbt = [], scr = [];
    for (let i = 0; i < RUNS; i++) {
      console.log("lh", path, method, i); const r = await lh(BASE + path, { port: 9333, logLevel: "error", output: "json", onlyCategories: ["performance"], throttlingMethod: method, disableStorageReset: true });
      const a = r.lhr.audits; if (process.env.DIAGFILE && i === 0 && method === "devtools") { const nr = (a["network-requests"].details?.items ?? []).map((x) => ({ url: x.url.replace(BASE, "").slice(0, 90), start: Math.round(x.networkRequestTime), end: Math.round(x.networkEndTime), status: x.statusCode, size: x.transferSize })); const t0 = Math.min(...nr.map((x) => x.start)); writeFileSync(process.env.DIAGFILE, JSON.stringify({ lcp: a["largest-contentful-paint"].numericValue, lcpElement: a["largest-contentful-paint-element"].details, phases: a["lcp-breakdown-insight"]?.details ?? a["lcp-phases"]?.details ?? null, fcp: a["first-contentful-paint"].numericValue, tti_main: a["main-thread-tasks"]?.details?.items?.slice(0, 5) ?? null, network: nr.map((x) => ({ ...x, start: x.start - t0, end: x.end - t0 })).sort((p, q) => p.start - q.start) }, null, 1)); } if (process.env.DIAG) { const le = a["largest-contentful-paint-element"]; console.log("LCP el", String(JSON.stringify(le.details?.items?.[0]?.items?.[0]?.node?.snippet ?? le.details?.items?.slice(0, 2))).slice(0, 300)); console.log("LCP phases", String(JSON.stringify(a["lcp-breakdown-insight"]?.details?.items ?? a["largest-contentful-paint"].details ?? "")).slice(0, 600)); } lcp.push(a["largest-contentful-paint"].numericValue); tbt.push(a["total-blocking-time"].numericValue);
      scr.push((a["resource-summary"].details.items.find((x) => x.resourceType === "script")?.transferSize ?? 0) / 1024);
    }
    out.push({ path: path.replace(AID, "[id]"), method, lcp_ms: Math.round(med(lcp)), tbt_ms: Math.round(med(tbt)), script_kb: Math.round(med(scr)), lcp_all: lcp.map(Math.round), tbt_all: tbt.map(Math.round) });
    console.log(JSON.stringify(out[out.length - 1]));
  }
}
writeFileSync(process.env.OUTF || "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s10/perf/lh-1217.json", JSON.stringify(out, null, 1));
await ctx.close();
