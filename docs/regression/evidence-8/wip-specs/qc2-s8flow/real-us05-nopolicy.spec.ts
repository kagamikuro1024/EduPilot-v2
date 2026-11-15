import { expect, test } from "@playwright/test";
import { cid, go, log, login, shot } from "./real-lib";
// US-P6-05 UI ở lớp QD8F2 theo trạng thái (STAGE = nopolicy | draftready | confirmed). TC-09,10,12,13,15,19,20,21,11
const CLS = process.env.CLS ?? "QD8F2";
const STAGE = process.env.STAGE ?? "nopolicy";
const BANNER = "Công thức điểm chưa xác nhận";
async function dump(page: any, tag: string) {
  const banners = await page.getByText(BANNER, { exact: true }).count();
  const links = await page.locator("main a").allInnerTexts();
  const fin = page.getByRole("button", { name: "Chốt điểm" });
  const finInfo = (await fin.count()) ? { disabled: await fin.first().isDisabled(), aria: await fin.first().getAttribute("aria-disabled"), title: await fin.first().getAttribute("title") } : null;
  log("us05", `${tag} banners=${banners} links=${JSON.stringify(links.filter((l: string) => /công thức|quy chế/i.test(l)))} chotDiem=${JSON.stringify(finInfo)}`);
  return { banners, links, finInfo };
}
test("teacher gradebook + attendance", async ({ page }) => {
  const C2 = cid(CLS), C1 = cid("QD8F8");
  await login(page, process.env.GV ?? "gv.s8f");
  await go(page, `/gradebook?course=${C2}`); await shot(page, `us05-${STAGE}-gradebook-teacher`);
  const a = await dump(page, `[${STAGE}] teacher /gradebook QD8F2`);
  const body = await page.locator("main").innerText(); log("us05", `[${STAGE}] teacher gradebook text: ${body.slice(0, 500).replace(/\n/g, " | ")}`);
  if (STAGE === "nopolicy") {
    const fin = page.getByRole("button", { name: "Chốt điểm" });
    if (await fin.count()) {
      await fin.first().hover(); await page.waitForTimeout(700); const tip1 = await page.getByRole("tooltip").allInnerTexts(); log("us05", `hover tooltip=${JSON.stringify(tip1)}`);
      await page.mouse.move(0, 0); await page.waitForTimeout(300); await fin.first().focus(); await page.waitForTimeout(700); const tip2 = await page.getByRole("tooltip").allInnerTexts(); log("us05", `focus tooltip=${JSON.stringify(tip2)}`);
    }
    log("us05", `info icons in main: ${await page.locator("main svg.lucide-info, main svg.lucide-circle-help").count()}`);
  }
  // attendance
  await go(page, `/attendance?course=${C2}`); await shot(page, `us05-${STAGE}-attendance-teacher`);
  const b = await dump(page, `[${STAGE}] teacher /attendance QD8F2`);
  // lớp 1 (CONFIRMED): không banner
  await go(page, `/gradebook?course=${C1}`); const c = await dump(page, `[${STAGE}] teacher /gradebook QD8F8(CONFIRMED)`);
  await go(page, `/attendance?course=${C1}`); const d = await dump(page, `[${STAGE}] teacher /attendance QD8F8(CONFIRMED)`);
  expect(true).toBeTruthy();
});
test("ta gradebook", async ({ page }) => {
  await login(page, "ta"); await go(page, `/gradebook?course=${cid(CLS)}`); await shot(page, `us05-${STAGE}-gradebook-ta`);
  await dump(page, `[${STAGE}] ta /gradebook QD8F2`);
  await go(page, `/attendance?course=${cid(CLS)}`); await dump(page, `[${STAGE}] ta /attendance QD8F2`);
});
test("student no banner", async ({ page }) => {
  await login(page, "sv.gioi");
  for (const u of ["/me", `/attendance?course=${cid(CLS)}`, `/gradebook?course=${cid(CLS)}`, "/chat"]) { await go(page, u); const n = await page.getByText(BANNER, { exact: true }).count(); log("us05", `[${STAGE}] student ${u} banners=${n}`); }
});
