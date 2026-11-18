import { test, type Page } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { cid, go, login } from "./lib";
test.setTimeout(180_000);
const OUT = "/tmp/qc2-s8r/ui09.tsv";
const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 420)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note}`); };
test("[09-23] what-if offline giữa lúc gõ: không mất số, có mạng tính lại", async ({ browser }) => {
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1280, height: 900 }, locale: "vi-VN" }); const p: Page = await c.newPage(); await login(p, "sv.kha");
  await go(p, `/me?course=${cid("QD8R6U")}`); await p.waitForTimeout(3500);
  const input = p.getByLabel("Điểm cuối kỳ giả định"); const has = (await input.count()) > 0;
  if (!has) { res("TC-P6-09-23", false, "QD8R6U: không có ô «Điểm cuối kỳ giả định» (cột cuối kỳ đã công bố?): " + (await p.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 160)); return; }
  await input.fill("8"); await p.waitForTimeout(1500); const r1 = await p.locator("[data-part=what-if-result]").innerText().catch(() => "");
  await c.setOffline(true); await input.fill("9"); await p.waitForTimeout(2500);
  const kept = await input.inputValue(); const err = (await p.getByRole("button", { name: "Thử lại" }).count()) > 0 || (await p.getByRole("alert").count()) > 0;
  await c.setOffline(false); const retry = p.getByRole("button", { name: "Thử lại" }); if (await retry.count()) await retry.first().click(); else { await input.fill("9"); }
  await p.waitForTimeout(3000); const r2 = await p.locator("[data-part=what-if-result]").innerText().catch(() => ""); const kept2 = await input.inputValue();
  res("TC-P6-09-23", /8/.test(r1) && kept === "9" && kept2 === "9" && err && /9/.test(r2), `gõ 8 → «${r1}»; mất mạng, gõ 9: ô còn «${kept}», báo lỗi/Thử lại ${err}; có mạng lại: ô «${kept2}», kết quả «${r2}»`);
});
