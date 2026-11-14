import { test, type Browser } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.6 L6: mục «Hôm nay» của «Lưu ý chấm» (teacher, ta) ở 1440 và 375.
const PW = process.env.QC_PW ?? "";
const OUT = process.env.QC_OUT ?? "/tmp/qc-shots";
mkdirSync(OUT, { recursive: true });

async function login(browser: Browser, user: string, w: number, h: number) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  if ((await page.locator("[data-part=topbar]").innerText()).includes("761988")) {
    await page.locator("[data-part=topbar]").getByText("761988").first().click();
    await page.getByText("761987").first().click();
    await page.waitForTimeout(1500);
  }
  return { ctx, page };
}

for (const user of ["teacher", "ta"]) {
  for (const [w, h] of [[1440, 900], [375, 812]] as const) {
    test(`hôm nay ${user} @${w}`, async ({ browser }, info) => {
      test.skip(info.project.name !== "desktop");
      test.setTimeout(90_000);
      const { ctx, page } = await login(browser, user, w, h);
      await page.goto("/");
      await page.waitForTimeout(3000);
      const t = await page.locator("main").innerText();
      const lines = t.split("\n").filter((l) => /Lưu ý chấm|bản cũ/.test(l));
      const ov = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);
      console.log(`QCINFO ${user}@${w} Hôm nay: ${JSON.stringify(lines)} tràn ngang=${ov}`);
      await page.screenshot({ path: `${OUT}/today-${user}-${w}.png`, fullPage: true });
      await ctx.close();
    });
  }
}
