import { test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
const BAD = /\b(RAG|PII|provider|fallback|trace|confidence|token|LLM|embedding|prompt|model|Threads|Insights|Analytics)\b/;

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}

test("TC-UX91-X4 SV 375: quét chữ mọi route, nhãn điều hướng", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(240000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const page = await ctx.newPage();
  page.setDefaultTimeout(10000);
  await login(page, "sv.kha@edupilot.local");
  const links = await page.evaluate(() => [...new Set([...document.querySelectorAll("nav a[href^='/'], aside a[href^='/']")].map((a) => a.getAttribute("href")!))]);
  console.log("INFO X4 liên kết điều hướng:", JSON.stringify(links));
  const labels = await page.evaluate(() => [...document.querySelectorAll("nav a[href^='/']")].map((a) => (a as HTMLElement).innerText.trim()).filter(Boolean));
  console.log("INFO X4 nhãn thanh dưới:", JSON.stringify(labels));
  const more = page.getByRole("link", { name: "Thêm" }).or(page.getByRole("button", { name: "Thêm" }));
  if (await more.count()) { await more.first().tap(); await page.waitForTimeout(700); console.log("INFO X4 nhãn «Thêm»:", JSON.stringify((await page.getByRole("dialog").first().innerText().catch(() => "")).replace(/\n+/g, " | ").slice(0, 400))); await page.keyboard.press("Escape"); }
  const routes = [...new Set(["/", "/chat", "/threads", "/exams", "/library", "/calendar", "/me", ...links.filter((l) => !/logout|login/.test(l))])];
  const hits: string[] = [];
  for (const r of routes) {
    try {
      await page.goto(r); await page.waitForTimeout(2500);
      const t = (await page.locator("body").innerText()).replace(/\s+/g, " ");
      const m = t.match(new RegExp(BAD.source, "g"));
      if (m) hits.push(`${r}: ${[...new Set(m)].join(",")}`);
      console.log(`INFO X4 ${r}: ${m ? "TỪ CẤM " + [...new Set(m)].join(",") : "sạch"} (${t.length} ký tự)`);
    } catch (e) { console.log(`INFO X4 ${r}: lỗi ${(e as Error).message.slice(0, 80)}`); }
  }
  console.log("INFO X4 tổng hợp:", JSON.stringify(hits));
  await ctx.close();
});
