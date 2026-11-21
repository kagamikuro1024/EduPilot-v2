import { test } from "@playwright/test";
import { go, FX, shot, log, newUser } from "./lib";
const F = FX("fx_ui8") as { C: string; T: string }; const FA = FX("fx_att") as { C: string; teacher: string };
const txt = async (p: any) => (await p.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 140);
test("38 chẩn đoán trạng thái rỗng", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.T);
  await go(page, `/inbox?course=${F.C}`); await page.waitForTimeout(1500);
  log("r7-ui", `INFO 38dbg A lớp QC3T1 mới tải: URL=${page.url().replace(/^https:\/\/[^/]+/, "")} chi tiết hiện=${await page.locator("[data-part=inbox-detail]").count()} :: ${await txt(page)}`);
  await page.getByRole("radio", { name: "Đã nhận" }).click(); await page.waitForTimeout(1500);
  log("r7-ui", `INFO 38dbg B bấm «Đã nhận» (rỗng) lần đầu: URL=${page.url().replace(/^https:\/\/[^/]+/, "")} inbox-list=${await page.locator("[data-part=inbox-list]").count()} :: ${await txt(page)}`);
  await shot(page, "r7-38dbg-B");
  await page.getByRole("radio", { name: "Đã trả lời" }).click(); await page.waitForTimeout(1500);
  await page.locator("[data-part=inbox-list] [data-ticket-id]").first().click(); await page.waitForTimeout(1000);
  log("r7-ui", `INFO 38dbg C chọn một phiếu «Đã trả lời»: URL=${page.url().replace(/^https:\/\/[^/]+/, "")}`);
  await page.getByRole("radio", { name: "Đã nhận" }).click(); await page.waitForTimeout(1500);
  log("r7-ui", `INFO 38dbg D rồi bấm «Đã nhận» (rỗng) khi có phiếu đang chọn: URL=${page.url().replace(/^https:\/\/[^/]+/, "")} inbox-list=${await page.locator("[data-part=inbox-list]").count()} :: ${await txt(page)}`);
  await shot(page, "r7-38dbg-D");
  await ctx.close();
});
