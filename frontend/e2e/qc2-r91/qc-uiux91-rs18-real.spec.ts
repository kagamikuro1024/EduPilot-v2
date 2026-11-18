/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkR (qc2, lượt 6): US-UX91-18 /documents — khẳng định thật cho TC-UX18-1/2/3/5/7/9/ID.
import { test } from "@playwright/test";
import { api, courseId, expect, go, noOverflow, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
test.describe.configure({ mode: "serial" });
const rows = (p: any) => p.locator("main tbody tr");

test("TC-UX18-1/2 nhãn «Loại» đủ chữ ở 1440 và 1100; không «Chia sẻ từ» với tài liệu của chính lớp", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  for (const w of [1440, 1100]) {
    const o = await open(browser, "teacher", w, 800); const c = await courseId(o.page, "761987");
    await go(o.page, "/documents", c, 3000);
    await rows(o.page).first().waitFor({ timeout: 30_000 });
    const cells = await rows(o.page).evaluateAll((trs: Element[]) => trs.map((tr) => {
      const tds = [...tr.querySelectorAll("td")] as HTMLElement[];
      const sel = tds[1].querySelector("select") as HTMLSelectElement | null;
      const el = (sel ?? tds[1]) as HTMLElement; const cs = getComputedStyle(el);
      const cv = document.createElement("canvas").getContext("2d")!; cv.font = `${cs.fontStyle} ${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`;
      const txt = sel ? sel.selectedOptions[0]?.text ?? "" : tds[1].innerText.trim();
      const need = cv.measureText(txt).width; const have = el.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight);
      return { txt, need: Math.round(need), have: Math.round(have), title: tds[0].innerText.replace(/\n/g, " | ") };
    }));
    console.log(`INFO UX18-1 @${w}:`, JSON.stringify(cells.map((x: any) => `${x.txt}:${x.need}/${x.have}`)));
    expect(cells.length).toBeGreaterThan(0);
    for (const x of cells) expect(x.need, `@${w} nhãn «${x.txt}» đủ chỗ`).toBeLessThanOrEqual(x.have);
    // UX18-2: tài liệu của chính lớp 761987 — API shared_from rỗng hoặc = 761987
    const api1 = await api(o.page, `/courses/${c}/documents?limit=100`);
    const own = (api1.body.items as any[]).filter((d) => !d.shared_from || d.shared_from === "761987").map((d) => String(d.title).slice(0, 12));
    const shown = cells.filter((x: any) => /Chia sẻ từ/.test(x.title));
    console.log(`INFO UX18-2 @${w} hàng có «Chia sẻ từ»:`, shown.length, "/", cells.length, "(API shared_from khác rỗng:", (api1.body.items as any[]).filter((d) => d.shared_from && d.shared_from !== "761987").length, ")");
    if (w === 1440) {
      const sharedHere = (api1.body.items as any[]).filter((d) => d.shared_from && d.shared_from !== "761987").length;
      expect(shown.length, "số hàng «Chia sẻ từ» = số tài liệu chia sẻ từ lớp khác").toBe(sharedHere);
      expect(own.length).toBeGreaterThan(0);
    }
    await o.ctx.close();
  }
});

test("TC-UX18-3 tài liệu chia sẻ từ lớp khác hiện «Chia sẻ từ {lớp}»", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher"); const c = await courseId(o.page, "761988");
  const list = await api(o.page, `/courses/${c}/documents?limit=100`);
  const items = list.body.items as any[];
  const shared = items.filter((d) => d.shared_from);
  console.log("INFO UX18-3 API:", JSON.stringify(items.map((d) => [String(d.title).slice(0, 14), d.shared_from || "∅"])));
  expect(shared.length, "761988 có ít nhất một tài liệu chia sẻ từ lớp khác").toBeGreaterThan(0);
  await go(o.page, "/documents", c, 3000);
  await rows(o.page).first().waitFor({ timeout: 30_000 });
  const titles = await rows(o.page).evaluateAll((trs: Element[]) => trs.map((tr) => (tr.querySelector("td") as HTMLElement).innerText.replace(/\n/g, " | ")));
  for (const d of shared) {
    const t = titles.find((x) => x.includes(String(d.title).slice(0, 14)));
    expect(t, `hàng «${d.title}»`).toBeTruthy();
    expect(t, `«Chia sẻ từ ${d.shared_from}»`).toContain(`Chia sẻ từ ${d.shared_from}`);
  }
  for (const d of items.filter((x) => !x.shared_from)) {
    const t = titles.find((x) => x.includes(String(d.title).slice(0, 14)));
    if (t) expect(t, "tài liệu của chính lớp không có dòng «Chia sẻ từ»").not.toContain("Chia sẻ từ");
  }
  await o.ctx.close();
});

test("TC-UX18-7/ID công tắc: lỗi → hoàn lại + Thử lại; bấm nhanh 5 lần khớp lần cuối", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher"); const c = await courseId(o.page, "761987");
  await go(o.page, "/documents", c, 3000);
  await o.page.locator("main tbody tr [role=switch]").first().waitFor({ timeout: 30_000 });
  const en = await o.page.locator("main tbody tr [role=switch]").evaluateAll((els: Element[]) => els.map((e, i) => (e as HTMLButtonElement).disabled || e.getAttribute("aria-disabled") === "true" ? -1 : i).filter((i) => i >= 0));
  console.log("INFO UX18 công tắc bật được (chỉ số):", JSON.stringify(en));
  const sw = (i: number) => o.page.locator("main tbody tr [role=switch]").nth(en[i]);
  const before = await sw(0).getAttribute("aria-checked");
  const pat = /\/api\/v1\/courses\/[0-9a-f-]+\/documents\/[0-9a-f-]+/;
  await o.page.route(pat, (r: any) => (["PATCH", "PUT", "POST"].includes(r.request().method()) ? r.abort() : r.continue()));
  await sw(0).click(); await o.page.waitForTimeout(2500);
  const after = await sw(0).getAttribute("aria-checked");
  const alert = (await o.page.getByRole("alert").allInnerTexts()).join(" ");
  console.log("INFO UX18-7 công tắc:", before, "→", after, "| lỗi:", JSON.stringify(alert), "| Thử lại:", await o.page.getByRole("button", { name: "Thử lại" }).count());
  expect(after, "công tắc hoàn lại khi lưu lỗi").toBe(before);
  expect(alert).toContain("Chưa lưu được");
  expect(await o.page.getByRole("button", { name: "Thử lại" }).count()).toBeGreaterThan(0);
  await o.page.unroute(pat);
  await o.page.getByRole("button", { name: "Thử lại" }).first().click(); await o.page.waitForTimeout(2500);
  expect(await sw(0).getAttribute("aria-checked"), "Thử lại áp dụng giá trị mới").not.toBe(before);
  await o.page.reload(); await rows(o.page).first().waitFor(); await o.page.waitForTimeout(2500);
  expect(await sw(0).getAttribute("aria-checked"), "máy chủ đã lưu").not.toBe(before);
  await sw(0).click(); await o.page.waitForTimeout(2000); // trả lại
  // ID
  const b2 = await sw(1).getAttribute("aria-checked");
  for (let i = 0; i < 5; i++) await sw(1).click({ delay: 10, timeout: 8000 });
  await o.page.waitForTimeout(4000);
  const ui = await sw(1).getAttribute("aria-checked");
  await o.page.reload(); await rows(o.page).first().waitFor(); await o.page.waitForTimeout(2500);
  const srv = await sw(1).getAttribute("aria-checked");
  console.log("INFO UX18-ID", b2, "→ UI", ui, "→ máy chủ", srv);
  expect(ui, "5 lần bấm (lẻ) = đảo").not.toBe(b2); expect(srv, "máy chủ khớp giao diện (lần bấm cuối)").toBe(ui);
  if (srv !== b2) { await sw(1).click(); await o.page.waitForTimeout(2000); }
  await o.ctx.close();
});

test("TC-UX18-5/9 375: vùng chạm công tắc ≥ 44 px; hàng thành thẻ xếp dọc; không tràn ngang", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher", 375, 812); const c = await courseId(o.page, "761987");
  await go(o.page, "/documents", c, 3500);
  const vis = o.page.locator("[role=switch]:visible");
  await vis.first().waitFor({ timeout: 30_000 });
  const hit = await vis.evaluateAll((els: Element[]) => els.slice(0, 8).map((e) => {
    e.scrollIntoView({ block: "center" }); const r = e.getBoundingClientRect(); const cx = r.x + r.width / 2, cy = r.y + r.height / 2;
    let ok = 0; for (const [dx, dy] of [[-21, 0], [21, 0], [0, -21], [0, 21], [-21, -21], [21, 21]]) { const t = document.elementFromPoint(cx + dx, cy + dy); if (t && (t === e || e.contains(t) || t.contains(e))) ok++; }
    return { w: Math.round(r.width), h: Math.round(r.height), ok };
  }));
  console.log("INFO UX18-5 vùng chạm:", JSON.stringify(hit));
  for (const h of hit) expect(h.w >= 44 && h.h >= 44, `công tắc ${h.w}×${h.h} ≥ 44`).toBe(true);
  expect(await noOverflow(o.page)).toBe(true);
  // UX18-9: bảng thành thẻ: không <table> nhìn thấy; các thẻ rộng ≈ bề rộng khung và xếp dọc
  expect(await o.page.locator("main table:visible").count(), "không bảng ngang ở 375").toBe(0);
  const cards = await o.page.locator("main [class*=itemMeta]:visible").evaluateAll((els: Element[]) => els.map((e) => { const r = (e.closest("li, article, [class*=item]") ?? e).getBoundingClientRect(); return [Math.round(r.top + scrollY), Math.round(r.bottom + scrollY), Math.round(r.width)]; }));
  console.log("INFO UX18-9 thẻ (top,bottom,width):", JSON.stringify(cards.slice(0, 5)));
  expect(cards.length).toBeGreaterThanOrEqual(2);
  for (let i = 1; i < cards.length; i++) expect(cards[i][0], `thẻ ${i} nằm dưới thẻ ${i - 1}`).toBeGreaterThanOrEqual(cards[i - 1][1] - 1);
  for (const cd of cards) expect(cd[2]).toBeLessThanOrEqual(375);
  await o.ctx.close();
});
