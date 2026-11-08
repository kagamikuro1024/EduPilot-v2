import { test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC đợt 1+2: ảnh sau (K7) trên stack ep-qc. Ghi /out/shots-w12. Không dùng fullPage.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
const ROUTES: Array<[string, string, string]> = [["teacher", "/", "home"], ["teacher", "/inbox", "inbox"], ["teacher", "/gradebook", "gradebook"], ["teacher", "/threads", "threads"], ["teacher", "/grading", "grading"], ["sv.gioi", "/chat", "chat"], ["sv.gioi", "/threads", "threads-sv"], ["sv.gioi", "/", "home-sv"], ["ta", "/observability", "observability"]];
test("K7 ảnh sau đợt 1+2", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180_000);
  for (const [w, h] of [[1440, 900], [375, 812]] as const) {
    const done = new Set<string>();
    for (const [user, url, tag] of ROUTES) {
      const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
      const page = await ctx.newPage();
      await page.goto("/login");
      await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
      await page.locator("input[type=password]").fill(PW);
      await page.keyboard.press("Enter");
      await page.waitForURL((u) => !u.pathname.startsWith("/login"));
      await page.locator("[data-part=topbar]").waitFor();
      await page.goto(url);
      await page.waitForTimeout(2500);
      await page.screenshot({ path: `/out/shots-w12/${tag}-${w}.png` });
      done.add(tag);
      await ctx.close();
    }
  }
});
