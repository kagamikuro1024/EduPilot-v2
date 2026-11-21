// QC SmkS: probe UX08-28 — Firefox launch lỗi gì; WebKit: khối chữ ẩn nào, tổ tiên nào có opacity 0
import { firefox, webkit, test } from "@playwright/test";
test.skip(!process.env.QC_REAL, "QC_REAL");
const BASE = process.env.QC_BASE ?? "https://localhost:1143";
for (const [name, bt] of [["firefox", firefox], ["webkit", webkit]] as const) {
  test(`probe ${name}`, async () => {
    test.setTimeout(120_000);
    let b;
    try { b = await bt.launch(); } catch (e) { console.log(`PROBE ${name} LAUNCH LỖI:`, String(e).replace(/\s+/g, " ").slice(0, 700)); return; }
    const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, baseURL: BASE });
    const page = await ctx.newPage();
    await page.goto("/", { waitUntil: "load" }); await page.waitForTimeout(1500);
    const h = await page.evaluate(() => document.documentElement.scrollHeight);
    for (let y = 0; y < h; y += 500) { await page.evaluate((v) => window.scrollTo(0, v), y); await page.waitForTimeout(150); }
    await page.waitForTimeout(800);
    const r = await page.evaluate(() => {
      const out: string[] = [];
      for (const el of document.querySelectorAll("h3")) {
        if (!(el as HTMLElement).innerText.trim()) continue;
        const chain: string[] = [];
        for (let n: Element | null = el; n && n !== document.body; n = n.parentElement) { const cs = getComputedStyle(n); if (cs.opacity !== "1" || cs.visibility !== "visible" || cs.display === "none") chain.push(`${n.tagName}.${n.className.toString().slice(0, 20)}#${n.id} op=${cs.opacity} vis=${cs.visibility} disp=${cs.display} hidden=${(n as HTMLElement).hidden} role=${n.getAttribute("role")} data=${[...n.attributes].filter((a) => a.name.startsWith("data-")).map((a) => a.name + "=" + a.value).join(",")}`); }
        out.push(`${(el as HTMLElement).innerText.trim().slice(0, 30)} :: ${chain.join(" <- ") || "hiện"}`);
      }
      return out;
    });
    console.log(`PROBE ${name} h3:`, JSON.stringify(r, null, 1));
    console.log(`PROBE ${name} supports animation-timeline:`, await page.evaluate(() => CSS.supports("animation-timeline: view()")), await page.evaluate(() => navigator.userAgent.slice(-50)));
    await b.close();
  });
}
