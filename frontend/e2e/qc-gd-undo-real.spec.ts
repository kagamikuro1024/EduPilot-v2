import { test, type Browser, type Page } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: QC_REAL=1 playwright test -c playwright.qc-gd.config.ts");
// QC sprint 10.6 P-106-8: Hoàn tác Duyệt / Bỏ (TC-G04-19…28) và Áp dụng qua ConfirmIrreversible (TC-G04-24) trên stack thật.
const PW = process.env.QC_PW ?? "";
const OUT = process.env.QC_OUT ?? "/tmp/qc-shots";
mkdirSync(OUT, { recursive: true });

async function open(browser: Browser, w: number, h: number) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  if ((await page.locator("[data-part=topbar]").innerText()).includes("761988")) {
    await page.locator("[data-part=topbar]").getByText("761988").first().click();
    await page.getByText("761987").first().click();
    await page.waitForTimeout(1500);
  }
  await page.goto("/grading?tab=assignments");
  await page.waitForTimeout(2500);
  if (w < 500) await page.getByText("Bài tập 03 —").first().dispatchEvent("click");
  else await page.locator("[data-part=assignment-row]", { hasText: "Bài tập 03" }).first().click();
  await page.waitForTimeout(2500);
  return { ctx, page, panel: page.locator("[data-part=guidance-panel]") };
}
const toasts = (page: Page) => page.getByRole("status").filter({ hasText: "Thành công" }).count();

for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  test(`undo Duyệt / Bỏ @${w}`, async ({ browser }, info) => {
    test.skip(info.project.name !== "desktop");
    test.setTimeout(180_000);
    const { ctx, page, panel } = await open(browser, w, h);
    const draft = panel.locator("[data-part=guidance-item]").filter({ has: page.getByRole("button", { name: "Duyệt" }) });
    console.log(`QCINFO @${w} số mục chờ duyệt: ${await draft.count()}`);
    const dialogs0 = await page.getByRole("dialog").count();
    const first = draft.first();
    const text0 = (await first.innerText()).split("\n")[0];
    await first.getByRole("button", { name: "Duyệt" }).click();
    await page.waitForTimeout(600);
    const undo = panel.getByRole("button", { name: "Hoàn tác" });
    console.log(`QCINFO @${w} G04-19 sau Duyệt: Hoàn tác=${await undo.count()} toast Thành công=${await toasts(page)} dialog thêm=${(await page.getByRole("dialog").count()) - dialogs0}`);
    if (await undo.count()) {
      const b = await undo.first().boundingBox();
      console.log(`QCINFO @${w} G04-28 vùng chạm Hoàn tác: ${Math.round(b?.width ?? 0)}x${Math.round(b?.height ?? 0)} tràn ngang=${await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)}`);
      await panel.screenshot({ path: `${OUT}/undo-after-approve-${w}.png` });
      await undo.first().click();
      await page.waitForTimeout(1200);
      const row = panel.locator("[data-part=guidance-item]", { hasText: text0.slice(0, 30) }).first();
      console.log(`QCINFO @${w} G04-20 sau Hoàn tác: nút Duyệt trở lại=${await row.getByRole("button", { name: "Duyệt" }).count()} «Chờ áp dụng»=${await row.getByText("Chờ áp dụng").count()}`);
    }
    if (w > 500) {
      // Bỏ → Hoàn tác
      const second = draft.first();
      const t1 = (await second.innerText()).split("\n")[0];
      await second.getByRole("button", { name: "Bỏ" }).click();
      await page.waitForTimeout(600);
      console.log(`QCINFO @${w} G04-21 sau Bỏ: Hoàn tác=${await panel.getByRole("button", { name: "Hoàn tác" }).count()}`);
      await panel.getByRole("button", { name: "Hoàn tác" }).first().click();
      await page.waitForTimeout(1200);
      console.log(`QCINFO @${w} G04-21 sau Hoàn tác Bỏ: Duyệt trở lại=${await panel.locator("[data-part=guidance-item]", { hasText: t1.slice(0, 30) }).first().getByRole("button", { name: "Duyệt" }).count()}`);
      // G04-22: Duyệt, quá 5 s, tải lại
      const third = draft.first();
      const t2 = (await third.innerText()).split("\n")[0];
      await third.getByRole("button", { name: "Duyệt" }).click();
      await page.waitForTimeout(6500);
      console.log(`QCINFO @${w} G04-22 sau 6,5 s: Hoàn tác=${await panel.getByRole("button", { name: "Hoàn tác" }).count()}`);
      await page.reload();
      await page.waitForTimeout(2500);
      if (!(await page.locator("[data-part=guidance-panel]").count())) await page.locator("[data-part=assignment-row]", { hasText: "Bài tập 03" }).first().click();
      await page.waitForTimeout(2000);
      const row = page.locator("[data-part=guidance-item]", { hasText: t2.slice(0, 30) }).first();
      console.log(`QCINFO @${w} G04-22 sau tải lại: «Chờ áp dụng»=${await row.getByText("Chờ áp dụng").count()} menu Đặt lại / nút: ${JSON.stringify(await row.getByRole("button").allInnerTexts())}`);
      await page.screenshot({ path: `${OUT}/undo-after-reload-${w}.png`, fullPage: true });
      // G04-24: Áp dụng → ConfirmIrreversible (huỷ)
      await page.locator("[data-part=guidance-panel]").getByRole("button", { name: /^Áp dụng/ }).click();
      await page.waitForTimeout(800);
      const pt2 = await panel.innerText();
      const conf = panel.locator("[data-part=confirm-irreversible], [role=alertdialog], [role=dialog]");
      console.log(`QCINFO @${w} G04-24 sau bấm Áp dụng: có «Để sau»=${/Để sau/.test(pt2)} chữ=${JSON.stringify((pt2.match(/[^\n]*Áp dụng bản[^\n]*/) ?? [""])[0])} khối xác nhận=${await conf.count()}`);
      await panel.screenshot({ path: `${OUT}/apply-confirm-${w}.png` });
      const later = panel.getByRole("button", { name: "Để sau" });
      if (await later.count()) await later.first().click();
      await page.waitForTimeout(600);
      // G04-27: offline khi Hoàn tác
      const fourth = draft.first();
      if (await fourth.count()) {
        await fourth.getByRole("button", { name: "Duyệt" }).click();
        await page.waitForTimeout(500);
        await ctx.setOffline(true);
        await panel.getByRole("button", { name: "Hoàn tác" }).first().click();
        await page.waitForTimeout(1500);
        const pt = await panel.innerText();
        console.log(`QCINFO @${w} G04-27 offline: Thử lại=${/Thử lại/.test(pt)} thông báo=${JSON.stringify((pt.match(/[^\n]*(lỗi|Không|mạng)[^\n]*/i) ?? [""])[0])}`);
        await page.screenshot({ path: `${OUT}/undo-offline-${w}.png` });
        await ctx.setOffline(false);
      }
    }
    await ctx.close();
  });
}
