import { test } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.7 (P-107-15) — documents: công tắc khoá (aria-disabled) nhận Tab / chạm, mở lý do, Esc đóng, bấm không đổi giá trị; hàng chia sẻ «Chỉ đổi được ở lớp gốc».
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-docs";
const CLS = process.env.QC_DOC_CLASS ?? "761987"; // 761988 = lớp nhận tài liệu chia sẻ (fixture QC: document_courses)
mkdirSync(OUT, { recursive: true });
const ok = (n: string, v: boolean, d = "") => console.log(`QCAC ${n} ${v ? "PASS" : "FAIL"} ${d}`);
for (const w of [1440, 375]) {
  test(`documents công tắc khoá @${w}`, async ({ browser }) => {
    test.setTimeout(150_000);
    const touch = w < 500;
    const ctx = await browser.newContext({ viewport: { width: w, height: touch ? 812 : 900 }, locale: "vi-VN", hasTouch: touch, isMobile: touch, ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    const patches: string[] = [];
    page.on("request", (q) => { if (q.method() === "PATCH" && /documents/.test(q.url())) patches.push(q.url()); });
    await page.goto(`${BASE}/login`);
    await page.locator("input[type=email]").fill("teacher@edupilot.local");
    await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
    await page.keyboard.press("Enter");
    await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
    const tb = page.locator("[data-part=topbar]"); await tb.waitFor();
    if (!(await tb.innerText()).includes(CLS)) { await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click(); await page.getByText(CLS).first().click(); await page.waitForTimeout(1500); }
    await page.goto(`${BASE}/documents`, { waitUntil: "domcontentloaded" });
    await page.waitForTimeout(3500);
    const locked = page.locator("[role=switch][aria-disabled=true]").filter({ visible: true });
    const nLocked = await locked.count();
    const info = await page.evaluate(() => {
      const main = document.querySelector("main")!;
      const sw = Array.from(main.querySelectorAll<HTMLElement>("[role=switch][aria-disabled=true]"));
      return { n: sw.length, truly: sw.filter((e) => (e as HTMLButtonElement).disabled).length, tips: sw.map((e) => ({ name: e.getAttribute("aria-label") ?? e.closest("label")?.textContent?.trim() ?? "", d: e.getAttribute("aria-describedby") ? document.getElementById(e.getAttribute("aria-describedby")!)?.textContent : null })).slice(0, 6), icons: Array.from(main.querySelectorAll("svg.lucide-info, svg.lucide-circle-help")).filter((x) => !x.closest("[role=status],[role=alert]")).length, shared: Array.from(main.querySelectorAll("tr, li")).filter((r) => /Chia sẻ từ/.test(r.textContent ?? "")).length };
    });
    console.log(`QCINFO documents[${CLS}]@${w} ${JSON.stringify(info)}`);
    ok(`DOC-${w}-aria-disabled`, nLocked >= 0 && (nLocked > 0 || CLS !== "761987") && info.truly === 0, `${nLocked} công tắc aria-disabled; thuộc tính disabled thật = ${info.truly} (phải 0 để nhận focus)`);
    ok(`DOC-${w}-icons`, info.icons === 0, `icon info/circle-help = ${info.icons}`);
    if (nLocked > 0) {
      const first = locked.first();
      await first.scrollIntoViewIfNeeded();
      const before = await first.getAttribute("aria-checked");
      const tipOf = async () => first.evaluate((e) => { const id = e.getAttribute("aria-describedby"); const t = id ? document.getElementById(id) : null; return t ? { text: t.textContent, vis: getComputedStyle(t).opacity !== "0" && getComputedStyle(t).visibility !== "hidden" && (t as HTMLElement).offsetParent !== null } : null; });
      if (!touch) {
        await first.focus();
        const focused = await first.evaluate((e) => e === document.activeElement);
        await page.waitForTimeout(400);
        const t1 = await tipOf();
        ok(`DOC-${w}-focus`, focused && !!t1?.vis, `focus=${focused}; lý do hiện=${t1?.vis} «${t1?.text}»`);
        await page.keyboard.press("Escape"); await page.waitForTimeout(300);
        const t2 = await tipOf();
        ok(`DOC-${w}-esc`, !t2?.vis, `sau Esc lý do hiện=${t2?.vis}`);
        await page.keyboard.press("Space"); await page.keyboard.press("Enter"); await page.waitForTimeout(500);
        ok(`DOC-${w}-no-toggle-kbd`, (await first.getAttribute("aria-checked")) === before && patches.length === 0, `aria-checked ${before}→${await first.getAttribute("aria-checked")}; PATCH=${patches.length}`);
        // Tab tới được (đi từ công tắc liền trước)
        await page.keyboard.press("Shift+Tab"); await page.keyboard.press("Tab");
        ok(`DOC-${w}-tab`, await first.evaluate((e) => e === document.activeElement), "Shift+Tab rồi Tab quay lại công tắc khoá");
        await page.mouse.move(0, 0);
      } else {
        const bb = (await first.boundingBox())!;
        await page.touchscreen.tap(bb.x + bb.width / 2, bb.y + bb.height / 2); await page.waitForTimeout(600);
        const t1 = await tipOf();
        ok(`DOC-${w}-tap`, !!t1?.vis, `chạm: lý do hiện=${t1?.vis} «${t1?.text}»`);
        ok(`DOC-${w}-no-toggle-tap`, (await first.getAttribute("aria-checked")) === before && patches.length === 0, `aria-checked ${before}→${await first.getAttribute("aria-checked")}; PATCH=${patches.length}`);
        const h = (await page.locator("main h1").first().boundingBox())!;
        await page.touchscreen.tap(h.x + 4, h.y + h.height / 2); await page.waitForTimeout(600);
        const t2 = await tipOf();
        ok(`DOC-${w}-tap-outside`, !t2?.vis, `chạm ra ngoài: lý do hiện=${t2?.vis}`);
        await page.screenshot({ path: `${OUT}/documents-tap-375.png` });
      }
    }
    // hàng chia sẻ từ lớp khác
    const sharedRow = page.locator("tr, li").filter({ hasText: /Chia sẻ từ/ }).first();
    if ((await sharedRow.count()) > 0) {
      const sw = sharedRow.locator("[role=switch]").first();
      const d = await sw.evaluate((e) => ({ dis: e.getAttribute("aria-disabled"), tip: e.getAttribute("aria-describedby") ? document.getElementById(e.getAttribute("aria-describedby")!)?.textContent : null }));
      ok(`DOC-${w}-shared`, d.dis === "true" && /Chỉ đổi được ở lớp gốc/.test(d.tip ?? ""), `aria-disabled=${d.dis}; lý do «${d.tip}»`);
    } else ok(`DOC-${w}-shared`, false, `không có hàng «Chia sẻ từ» trong lớp ${CLS} (KHÔNG KIỂM ĐƯỢC)`);
    await page.screenshot({ path: `${OUT}/documents-${w}.png` });
    await ctx.close();
  });
}
