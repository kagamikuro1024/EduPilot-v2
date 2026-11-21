import { expect, test } from "@playwright/test";
// Bản sửa selector của TC-UX04-1/2 (qc-uiux91-w1.spec.ts:261): bản dựng cổng băm tên lớp CSS nên `class*=Button-module` không còn khớp; dùng data-variant.
test("TC-UX04-1/2 :active thu 0,97 ≤ 100 ms rồi trở lại (selector data-variant)", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop");
  await page.goto("/dev/ui");
  const bs = page.locator("main button[data-variant]:not([aria-disabled=true]):not([disabled])");
  await bs.first().waitFor();
  const n = Math.min(await bs.count(), 60);
  const bad: string[] = []; const seen = new Set<string>();
  for (let i = 0; i < n; i++) {
    const b = bs.nth(i);
    if (!(await b.isVisible())) continue;
    const sig = await b.evaluate((e) => `${e.getAttribute("data-variant")}|${e.className}`);
    if (seen.has(sig)) continue; seen.add(sig);
    await b.scrollIntoViewIfNeeded(); await b.hover(); await page.mouse.down(); await page.waitForTimeout(160);
    const t = await b.evaluate((e) => ({ tr: getComputedStyle(e).transform, d: getComputedStyle(e).transitionDuration, tp: getComputedStyle(e).transitionProperty }));
    if (!/matrix\(0\.97/.test(t.tr)) bad.push(`${sig} → transform=${t.tr} dur=${t.d} prop=${t.tp}`);
    { const ps = t.tp.split(",").map((x) => x.trim()); const ds = t.d.split(",").map((x) => parseFloat(x)); const i = ps.indexOf("transform"); const dur = i < 0 ? NaN : ds[i % ds.length]; if (!(dur <= 0.1)) bad.push(`${sig} transform ${t.d} / ${t.tp} > 100 ms`); }
    await page.mouse.up(); await page.mouse.move(0, 0); await page.waitForTimeout(220);
    if (!/none|matrix\(1,/.test(await b.evaluate((e) => getComputedStyle(e).transform))) bad.push(`${sig} không trở lại`);
  }
  console.log("INFO biến thể:", seen.size, bad.slice(0, 5).join(" ; "));
  expect(seen.size).toBeGreaterThan(2);
  expect(bad).toEqual([]);
});
