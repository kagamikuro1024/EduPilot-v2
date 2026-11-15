import { expect, test } from "@playwright/test";
import { cid, go, log, login, shot } from "./real-lib";
test("attendance banner + 30 students timing", async ({ page }) => {
  const C = cid(process.env.ATT ?? "QD8F2");
  await login(page, process.env.GV ?? "gv.s8f"); await go(page, `/gradebook?course=${C}`); await page.getByRole("link", { name: "Điểm danh" }).first().click(); await page.waitForTimeout(6000); await page.locator("[data-part=att-row]").first().waitFor({ timeout: 40000 }).catch(() => undefined); await page.waitForTimeout(1500); log("us05att", `url=${page.url()} course=${await page.locator("[data-part=topbar]").innerText().then(t=>t.slice(0,80).replace(/\n/g," "))}`); await shot(page, `us05-attendance-${process.env.ATT ?? "QD8F2"}`);
  const t = await page.locator("main").innerText(); log("us05att", `text: ${t.slice(0, 600).replace(/\n/g, " | ")}`);
  log("us05att", `banners=${await page.getByText("Công thức điểm chưa xác nhận", { exact: true }).count()} links=${JSON.stringify(await page.locator("main a").allInnerTexts())}`);
  const bn = page.getByText("Công thức điểm chưa xác nhận", { exact: true });
  if (await bn.count()) { const bb = await bn.first().boundingBox(); const hh = await page.locator("main h1").first().boundingBox(); log("us05att", `banner box=${JSON.stringify(bb)} h1=${JSON.stringify(hh)}`); }
  const rows = page.locator("[data-part=attendance-row], tbody tr"); log("us05att", `rows=${await rows.count()}`);
  const html = await page.locator("main").innerHTML(); log("us05att", `data-parts: ${[...new Set(html.match(/data-part="[^"]+"/g) ?? [])].join(",")}`);
});
