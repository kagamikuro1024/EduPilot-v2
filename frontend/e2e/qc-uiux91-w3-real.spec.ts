import { expect, test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC đợt 3 trên stack thật ep-qc: Hôm nay của giảng viên / sinh viên với dữ liệu seed; ảnh sau (K7) ở /out/shots-w3.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
async function login(page: Page, user: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
  await page.locator("[data-part=topbar]").waitFor({ timeout: 15_000 });
  await page.waitForTimeout(2500);
}
const info = (page: Page) => page.evaluate(() => ({
  h1: document.querySelector("main h1")?.textContent?.trim(),
  h3: [...document.querySelectorAll("main h3")].map((h) => (h as HTMLElement).innerText.replace(/\s+/g, " ").trim()),
  rows: [...document.querySelectorAll("main li[data-kind]")].map((l) => `${(l as HTMLElement).dataset.kind}: ${(l as HTMLElement).innerText.replace(/\s+/g, " ").slice(0, 70)}`),
}));

test("TC-UX10 thật: GV / TA / admin chia nhóm; ảnh sau 1440 + 375", async ({ browser }, info_) => {
  test.skip(info_.project.name !== "desktop");
  for (const user of ["teacher", "ta", "admin"]) {
    for (const [w, h] of [[1440, 900], [375, 812]] as const) {
      const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
      const page = await ctx.newPage();
      await login(page, user);
      const i = await info(page);
      if (w === 1440) console.log(`INFO ${user}:`, JSON.stringify(i).slice(0, 900));
      await page.screenshot({ path: `/out/shots-w3/hom-nay-${user}-${w}.png` });
      if (user === "teacher" && w === 1440) {
        const red = await page.evaluate(() => { const cv = document.createElement("canvas"); cv.width = cv.height = 1; const x = cv.getContext("2d")!; return [...document.querySelectorAll("main a, main button")].filter((e) => { const r = e.getBoundingClientRect(); return r.width > 30 && r.top >= 0 && r.top < 900; }).filter((e) => { x.clearRect(0, 0, 1, 1); x.fillStyle = getComputedStyle(e).backgroundColor; x.fillRect(0, 0, 1, 1); const d = x.getImageData(0, 0, 1, 1).data; return d[0] > 170 && d[1] < 70 && d[2] < 80; }).length; });
        console.log("INFO teacher nút đỏ:", red);
        expect(red).toBe(1);
        expect(i.h3.length, "GV thật có ≥ 1 nhóm").toBeGreaterThan(0);
        await page.evaluate(() => { const w2 = window as unknown as { __a: string[] }; w2.__a = []; const seen = new Set<Animation>(); const tick = () => { for (const a of document.getAnimations()) if (!seen.has(a)) { seen.add(a); w2.__a.push(`${(a as CSSAnimation).animationName ?? (a as CSSTransition).transitionProperty}:${a.effect?.getTiming().duration}`); } requestAnimationFrame(tick); }; tick(); });
        await page.locator("main li[data-kind]").first().locator("a, button").last().click();
        await page.waitForTimeout(1500);
        console.log("INFO sau khi bấm hàng đầu:", page.url(), JSON.stringify(await page.evaluate(() => [...new Set((window as unknown as { __a: string[] }).__a)])));
      }
      await ctx.close();
    }
  }
});

test("TC-UX11 thật: SV sv.gioi thứ tự + sv.moi + ảnh sau", async ({ browser }, info_) => {
  test.skip(info_.project.name !== "desktop");
  for (const user of ["sv.gioi", "sv.moi"]) {
    for (const [w, h] of [[1440, 900], [375, 812]] as const) {
      const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
      const page = await ctx.newPage();
      await login(page, user);
      const heads = await page.evaluate(() => [...document.querySelectorAll("main h1, main h2, main h3")].map((e) => `${(e as HTMLElement).innerText.trim().slice(0, 30)}@${Math.round(e.getBoundingClientRect().top + scrollY)}`));
      if (w === 1440) console.log(`INFO ${user}:`, heads.join(" | "));
      const cont = await page.getByRole("link", { name: /Lớp \d+/ }).allInnerTexts().catch(() => []);
      if (w === 1440 && user === "sv.gioi") console.log("INFO tiếp tục học:", JSON.stringify(cont.map((c) => c.replace(/\s+/g, " ")).slice(0, 5)));
      await page.screenshot({ path: `/out/shots-w3/hom-nay-${user}-${w}.png` });
      await ctx.close();
    }
  }
});
