import { test } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 — documents: dòng ghi chú đáp án bị bỏ; lý do công tắc «Hiện cho sinh viên» bị khoá nằm ở nhãn / tooltip của công tắc, không có icon riêng.
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-docs";
mkdirSync(OUT, { recursive: true });
for (const w of [1440, 375]) {
  test(`documents công tắc bị khoá @${w}`, async ({ browser }) => {
    test.setTimeout(120_000);
    const ctx = await browser.newContext({ viewport: { width: w, height: w === 1440 ? 900 : 812 }, locale: "vi-VN", hasTouch: w < 500, ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await page.goto(`${BASE}/login`);
    await page.locator("input[type=email]").fill("teacher@edupilot.local");
    await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
    await page.keyboard.press("Enter");
    await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
    await page.locator("[data-part=topbar]").waitFor();
    const tb = page.locator("[data-part=topbar]");
    if (!(await tb.innerText()).includes("761987")) { await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click(); await page.getByText("761987").first().click(); await page.waitForTimeout(1500); }
    await page.goto(`${BASE}/documents`, { waitUntil: "domcontentloaded" });
    await page.waitForTimeout(3000);
    const r = await page.evaluate(() => {
      const main = document.querySelector("main")!;
      const switches = Array.from(main.querySelectorAll<HTMLElement>("[role=switch], input[type=checkbox]")).filter((e) => e.offsetParent !== null || e.closest("label"));
      const rows = Array.from(main.querySelectorAll("tr, li, [role=row]"));
      const info = switches.map((sw) => {
        const dis = (sw as HTMLInputElement).disabled || sw.getAttribute("aria-disabled") === "true";
        if (!dis) return null;
        const row = sw.closest("tr, li, [role=row]");
        const rect = sw.getBoundingClientRect();
        const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
        const tipEl = hit?.closest("[title]") ?? sw.closest("[title]");
        const lab = sw.getAttribute("aria-label") || sw.closest("label")?.textContent?.trim() || "";
        const desc = sw.getAttribute("aria-describedby") ? document.getElementById(sw.getAttribute("aria-describedby")!)?.textContent : null;
        return { row: row?.textContent?.trim().slice(0, 60), label: lab.slice(0, 50), title: tipEl?.getAttribute("title") ?? null, tipOn: tipEl ? tipEl.tagName + (tipEl === sw ? "(chính công tắc)" : "(vỏ)") : null, desc, tag: sw.tagName };
      }).filter(Boolean);
      const icons = main.querySelectorAll("svg.lucide-info, svg.lucide-circle-help, svg.lucide-circle-question-mark").length;
      const keyNote = rows.filter((x) => /Không hiển thị cho sinh viên/.test(x.textContent ?? "")).length;
      const dupStatic = (main.innerText.match(/Không dùng cho AI của sinh viên/g) ?? []).length;
      return { switches: switches.length, disabled: info, icons, keyNote, dupStatic };
    });
    console.log(`QCINFO documents@${w} ${JSON.stringify(r)}`);
    await page.screenshot({ path: `${OUT}/documents-${w}.png`, fullPage: false });
    await ctx.close();
  });
}
