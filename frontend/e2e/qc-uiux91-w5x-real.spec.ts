import { test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}

const PAGES: [string, string, string][] = [
  ["chat", "sv.kha@edupilot.local", "/chat"],
  ["threads-sv", "sv.kha@edupilot.local", "/threads"],
  ["threads-gv", "teacher@edupilot.local", "/threads"],
  ["me", "sv.kha@edupilot.local", "/me"],
  ["me-nguyco", "sv.nguyco@edupilot.local", "/me"],
  ["documents", "teacher@edupilot.local", "/documents"],
  ["insights", "teacher@edupilot.local", "/insights"],
  ["observability", "teacher@edupilot.local", "/observability"],
];

for (const [name, email, url] of PAGES) {
  test(`X ${name}`, async ({ browser }) => {
    test.setTimeout(120000);
    for (const [w, h] of [[1440, 900], [375, 812]] as const) {
      const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h } });
      const page = await ctx.newPage();
      await login(page, email);
      await page.goto(url);
      await page.waitForTimeout(4500);
      await page.screenshot({ path: `/out/shots-w5/${name}-${w}.png` });
      await page.screenshot({ path: `/out/shots-w5/${name}-${w}-full.png`, fullPage: true });
      const t = (await page.locator("main").innerText()).replace(/\n+/g, " ⏎ ");
      console.log(`INFO ${name}@${w} H=${await page.evaluate(() => document.documentElement.scrollHeight)} :: ${t.slice(0, w === 1440 ? 1800 : 400)}`);
      await ctx.close();
    }
  });
}
