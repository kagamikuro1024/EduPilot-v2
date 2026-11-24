import { chromium, newCtx, text, BASE } from "./lib.mjs";
const NAME = "QC3-stub-slow";
const b = await chromium.launch();
const { ctx, page } = await newCtx(b, "admin");
await page.goto(`${BASE}/settings/llm`); await page.waitForTimeout(4000);
const row = page.locator("div").filter({ has: page.getByText(NAME, { exact: true }) }).filter({ has: page.getByRole("button", { name: "Test kết nối" }) }).last();
const btn = row.getByRole("button", { name: "Test kết nối" });
const t0 = Date.now(); await btn.click();
let early = "", st = [];
for (let s = 1; s <= 40; s++) {
  await page.waitForTimeout(1000);
  const full = await page.locator("body").innerText(); const k = full.indexOf(NAME); const seg = full.slice(k, k + 220).replace(/\n+/g, " | ");
  if (s === 12) early = seg;
  if (/Kết nối tốt|quá chậm|Kết nối lỗi/.test(seg) && s > 3) { console.log("kết quả sau", s, "s:", seg); await row.screenshot({ path: "/tmp/qc3oc/ui/ui63.png" }); break; }
}
console.log("ở giây 12 (đang chờ):", early);
console.log("tổng", Math.round((Date.now() - t0) / 1000), "s; nút còn bấm được:", await btn.isEnabled());
await b.close();
