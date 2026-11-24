import { chromium, newCtx, text, BASE } from "./lib.mjs";
const NAME = process.argv[2] || "QC3-UI-OC";
const b = await chromium.launch();
const { ctx, page } = await newCtx(b, "admin");
await page.goto(`${BASE}/settings/llm`); await page.waitForTimeout(4000);
const row = page.locator("div").filter({ has: page.getByText(NAME, { exact: true }) }).filter({ has: page.getByRole("button", { name: "Test kết nối" }) }).last();
const t0 = Date.now(); await row.getByRole("button", { name: "Test kết nối" }).click();
for (let i = 0; i < 30; i++) { await page.waitForTimeout(1000); const x = await row.innerText(); if (/Đã kết nối|quá chậm|không được|Không kết nối|OK|Kết nối lỗi/.test(x.split("Test kết nối").slice(1).join(" ")) || /role=alert/.test("")) { } if (await row.locator("[role=alert],[role=status]").count()) break; }
console.log("sau", Math.round((Date.now() - t0) / 1000), "s:", (await row.innerText()).replace(/\n+/g, " | ").slice(0, 300));
await row.screenshot({ path: `/tmp/qc3oc/ui/test-${NAME}.png` });
await b.close();
