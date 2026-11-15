import { test, type Browser } from "@playwright/test";
import { mkdirSync } from "fs";
// QC2 nhóm G · 10.6 · TC-G01-13/14/15: /class/settings, 4 công tắc «Học từ lần sửa» (1440 + 375)
const PW = process.env.QC_PW ?? "", BASE = process.env.QC_BASE ?? "https://localhost:1143", OUT = process.env.QC_OUT ?? "/tmp/qc2g-shots";
mkdirSync(OUT, { recursive: true });
const info = (k: string, v: unknown) => console.log(`QCINFO ${k}: ${typeof v === "string" ? v : JSON.stringify(v)}`);
async function login(browser: Browser, user: string, w: number, h: number) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  const tb = page.locator("[data-part=topbar]");
  const cb = tb.locator("button").filter({ hasText: /\d{3,}/ });
  if (!(await tb.innerText()).includes("761987") && (await cb.count()) > 0) { await tb.locator("button").filter({ hasText: /\d{3,}/ }).first().click(); await page.getByText("761987", { exact: false }).first().click(); await page.waitForTimeout(1500); }
  return { ctx, page };
}
for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  test(`teacher /class/settings @${w}`, async ({ browser }) => {
    test.setTimeout(120_000);
    const { ctx, page } = await login(browser, "teacher", w, h);
    await page.goto("/class/settings"); await page.waitForTimeout(2500);
    const txt = await page.locator("main").innerText();
    for (const l of ["Lưu ý chấm", "Ca kiểm thử", "Gợi ý câu hỏi tương tự", "Gợi ý khi sinh câu hỏi"]) info(`@${w} nhãn «${l}»`, txt.includes(l));
    info(`@${w} nhóm «Học từ lần sửa»`, txt.includes("Học từ lần sửa"));
    const sw = page.locator("[role=switch]");
    const sizes = await sw.evaluateAll((els: Element[]) => els.map((e) => { const r = e.getBoundingClientRect(); const lab = (e.getAttribute("aria-label") ?? e.closest("[data-part]")?.textContent ?? "").slice(0, 40); return { lab, w: Math.round(r.width), h: Math.round(r.height), dis: e.getAttribute("aria-disabled") ?? e.getAttribute("disabled") }; }));
    info(`@${w} công tắc`, sizes);
    info(`@${w} tràn ngang`, await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth));
    info(`@${w} câu mô tả (dòng > 60 ký tự dưới nhóm)`, (txt.split("\n").filter((l) => l.length > 60 && !/Hồ sơ|Hoàn tác/.test(l))).slice(0, 4));
    await page.screenshot({ path: `${OUT}/class-settings-${w}.png`, fullPage: true });
    if (w === 1440) {
      // G01-14: đổi công tắc «Gợi ý khi sinh câu hỏi» rồi Hoàn tác
      const t = page.getByRole("switch", { name: /Gợi ý khi sinh câu hỏi/ }); const before = await t.getAttribute("aria-checked");
      await t.click(); await page.waitForTimeout(500);
      const after = await t.getAttribute("aria-checked"); const body = await page.locator("body").innerText();
      info("G01-14 đổi công tắc", { before, after, hoanTac: /Hoàn tác/.test(body), toastThanhCong: /Thành công|thành công/.test(body), hopXacNhan: (await page.getByRole("dialog").count()) > 0 });
      const undo = page.getByRole("button", { name: /Hoàn tác/ }).first();
      if (await undo.count()) { await undo.click(); await page.waitForTimeout(800); info("G01-14 sau Hoàn tác", await t.getAttribute("aria-checked")); }
      else { await t.click(); await page.waitForTimeout(500); info("G01-14 trả cờ thủ công", await t.getAttribute("aria-checked")); }
    }
    await ctx.close();
  });
}
test("SV không vào /class/settings, không thấy công tắc", async ({ browser }) => {
  const { ctx, page } = await login(browser, "sv.kha", 1440, 900);
  await page.goto("/class/settings"); await page.waitForTimeout(2000);
  const t = await page.locator("body").innerText();
  info("SV /class/settings", { url: page.url(), coHocTuLanSua: /Học từ lần sửa|Lưu ý chấm/.test(t) });
  await ctx.close();
});
