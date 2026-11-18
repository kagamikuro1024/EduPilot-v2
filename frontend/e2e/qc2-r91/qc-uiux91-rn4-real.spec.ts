/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkS (qc2, lượt 7): ca đợt 4 chưa có kịch bản — UX14-8, UX16-13/14, UX04-19, UX16-17, UX12-16, UX12-17. Lớp sao QR91A (dữ liệu sạch), QR91D (chốt thử), QR91E (không bài nộp).
import { test, type Page } from "@playwright/test";
import { api, courseId, expect, go, login, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
const A = process.env.QC_CLASS ?? "QR91A";
const D_ = process.env.QC_CLASS_D ?? "QR91D";
const E = process.env.QC_CLASS_E ?? "QR91E";

// ---- UX14-8: Chấm tất cả khoá khi lớp chưa có bài nộp, bấm được khi có bài ----
test("TC-UX14-8 «Chấm tất cả»: bài chưa có bài nộp khoá + lý do; bài có bài nộp bấm được", async ({ browser }, info) => {
  D(info); test.setTimeout(180_000);
  const o = await open(browser, "teacher");
  const c = await courseId(o.page, A);
  await go(o.page, "/grading", c, 3500);
  const btn = o.page.getByRole("button", { name: "Chấm tất cả" }).first();
  const read = async (tag: string) => {
    await o.page.waitForTimeout(2500);
    const dis = await btn.getAttribute("aria-disabled"); const dd = await btn.isDisabled();
    await btn.hover({ force: true }); await o.page.waitForTimeout(700);
    const tip = (await o.page.locator("[role=tooltip]").allInnerTexts()).join(" | ");
    console.log(`INFO UX14-8 ${tag}: aria-disabled=${dis} disabled=${dd} tooltip="${tip}"`);
    return { off: dis === "true" || dd, tip };
  };
  const pickBt = async (n: string) => { const opts = await o.page.getByLabel("Bài tập").locator("option").allInnerTexts(); await o.page.getByLabel("Bài tập").selectOption({ label: opts.find((x) => x.includes(n))! }); };
  await pickBt("Bài tập 04");
  const none = await read("Bài tập 04 (0 bài nộp)");
  expect(none.off, "bài chưa có bài nộp: khoá").toBe(true);
  expect(none.tip, "lý do").toMatch(/Chưa có bài nộp/);
  await pickBt("Bài tập 03");
  const has = await read("Bài tập 03 (có bài)");
  expect(has.off, "bài có bài nộp: bấm được").toBe(false);
  // lớp chưa có bài tập nào: khoá, lý do chọn bài
  const e = await courseId(o.page, E);
  await go(o.page, "/grading", e, 3500);
  const dis = await btn.getAttribute("aria-disabled");
  console.log("INFO UX14-8 lớp rỗng:", dis);
  expect(dis === "true" || (await btn.isDisabled())).toBe(true);
  await o.ctx.close();
});

// ---- UX16-13/14 + UX04-19: ô sổ điểm — Enter xuống dòng, Tab sang ô; nháy xanh nhạt khi lưu; reduce ----
async function sampleSave(page: Page, act: () => Promise<void>) {
  await page.evaluate(() => {
    const w = window as unknown as { __r: Array<{ t: number; bg: string; tf: string; an: string }>; __t0: number };
    w.__r = []; w.__t0 = performance.now();
    const f = document.querySelector("main tbody input") as HTMLElement;
    const td = f.closest("td") as HTMLElement;
    const tick = () => {
      const anims = document.getAnimations().map((a) => `${(a as CSSAnimation).animationName ?? (a as CSSTransition).transitionProperty}:${a.effect?.getTiming().duration}`).filter((x) => /saved|background/i.test(x));
      const cs = getComputedStyle(f), ct = getComputedStyle(td);
      w.__r.push({ t: Math.round(performance.now() - w.__t0), bg: `${cs.backgroundColor}|${ct.backgroundColor}`, tf: `${cs.transform}|${ct.transform}`, an: anims.join(",") });
      if (performance.now() - w.__t0 < 1500) setTimeout(tick, 25);
    };
    tick();
  });
  await act();
  await page.waitForTimeout(1800);
  return page.evaluate(() => (window as unknown as { __r: Array<{ t: number; bg: string; tf: string; an: string }> }).__r);
}
for (const reduce of [false, true]) {
  test(`TC-${reduce ? "UX16-14 TC-UX04-19" : "UX16-13"} nháy lưu ô sổ điểm, Enter/Tab (reduce=${reduce})`, async ({ browser }, info) => {
    D(info); test.setTimeout(240_000);
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, reducedMotion: reduce ? "reduce" : "no-preference", locale: "vi-VN" });
    const page = await ctx.newPage();
    await login(page, "teacher");
    const c = await courseId(page, A);
    await go(page, "/gradebook", c, 4000);
    const cells = page.locator("main tbody input");
    const first = cells.first();
    const old = await first.inputValue();
    // Enter: lưu rồi xuống ô dưới (cùng cột)
    const col = await first.evaluate((e) => [...(e.closest("tr")?.children ?? [])].indexOf(e.closest("td") as Element));
    const nv = old.replace(",00", "") === "7" ? "6,5" : "7";
    const tl = await sampleSave(page, async () => { await first.focus(); await first.fill(nv); await first.press("Enter"); });
    const base = tl[0].bg;
    const changed = tl.filter((s) => s.bg !== base);
    const t0 = changed[0]?.t, t1 = changed[changed.length - 1]?.t;
    console.log(`INFO reduce=${reduce}: nền gốc ${base} | đổi nền từ ${t0} ms đến ${t1} ms (${changed.length}/${tl.length} mẫu) | transform khác none: ${tl.some((s) => !/^none\|none$|^matrix\(1, 0, 0, 1, 0, 0\)\|/.test(s.tf) && !/^none/.test(s.tf))} | anim: ${[...new Set(tl.map((s) => s.an).filter(Boolean))].join(";")}`);
    console.log("INFO mẫu:", JSON.stringify(tl.filter((_, i) => i % 6 === 0).map((s) => [s.t, s.bg.split("|")[0].replace("rgba", "a").slice(0, 22), s.bg.split("|")[1].replace("rgba", "a").slice(0, 22)])));
    const active = await page.evaluate(() => { const a = document.activeElement as HTMLInputElement; const td = a.closest("td"); const tr = a.closest("tr"); return { tag: a.tagName, col: td ? [...(tr?.children ?? [])].indexOf(td) : -1, row: tr ? [...(tr.parentElement?.children ?? [])].indexOf(tr) : -1 }; });
    console.log("INFO sau Enter focus:", JSON.stringify(active), "cột ô gốc", col);
    if (!reduce) {
      expect(active.col, "Enter: giữ cột").toBe(col);
      expect(active.row, "Enter: xuống dòng kế").toBe(1);
      expect(changed.length, "có nháy nền khi lưu").toBeGreaterThan(0);
    } else {
      expect(changed.length, "reduce: vẫn đổi màu nền").toBeGreaterThan(0);
      expect(tl.some((s) => !/^none/.test(s.tf.split("|")[0]) || !/^none/.test(s.tf.split("|")[1])), "reduce: không transform").toBe(false);
    }
    // Tab: sang ô kế bên phải
    await first.focus(); await first.fill(old); await first.press("Tab");
    await page.waitForTimeout(1200);
    const tab = await page.evaluate(() => { const a = document.activeElement as HTMLElement; const td = a.closest("td"); const tr = a.closest("tr"); return { tag: a.tagName, col: td ? [...(tr?.children ?? [])].indexOf(td) : -1, row: tr ? [...(tr.parentElement?.children ?? [])].indexOf(tr) : -1 }; });
    console.log("INFO sau Tab focus:", JSON.stringify(tab));
    expect(tab.row, "Tab: giữ dòng").toBe(0);
    expect(tab.col, "Tab: sang ô kế").toBeGreaterThan(col);
    // trả giá trị gốc
    await go(page, "/gradebook", c, 3000);
    const f2 = page.locator("main tbody input").first();
    if ((await f2.inputValue()) !== old) { await f2.fill(old); await f2.press("Enter"); await page.waitForTimeout(1500); }
    await ctx.close();
  });
}

// ---- UX16-17: Chốt điểm — xác nhận bấm đúp (lớp thử QR91D) ----
test("TC-UX16-17 Chốt điểm: bấm đúp xác nhận chỉ chốt một lần", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher");
  const c = await courseId(o.page, D_);
  const posts: string[] = [];
  o.page.on("request", (r) => { if (r.method() === "POST" && /finalize/.test(r.url())) posts.push(`${r.headers()["idempotency-key"] ?? "-"}`); });
  const before = await api(o.page, `/courses/${c}/grade-scheme/status`);
  console.log("INFO trước:", JSON.stringify(before.body).slice(0, 200));
  await go(o.page, "/gradebook", c, 3500);
  await o.page.getByRole("button", { name: "Chốt điểm" }).first().click();
  const dlg = o.page.getByRole("dialog");
  await expect(dlg).toBeVisible();
  await dlg.getByRole("button", { name: /^Chốt điểm$/ }).dblclick();
  await o.page.waitForTimeout(4000);
  const after = await api<any>(o.page, `/courses/${c}/grade-scheme/status`);
  console.log("INFO sau:", JSON.stringify(after.body).slice(0, 220), "| POST finalize:", JSON.stringify(posts));
  const gb = (after.body as any).gradebook;
  expect(gb?.status, "sổ đã chốt").toBe("LOCKED");
  expect(gb?.snapshot_version, "đúng một bản chốt").toBe(1);
  expect(posts.length, "một lần ghi (lần hai bị chặn hoặc cùng khoá idempotency)").toBeLessThanOrEqual(2);
  if (posts.length === 2) expect(posts[0], "cùng Idempotency-Key").toBe(posts[1]);
  const snaps = await api<any>(o.page, `/courses/${c}/gradebook/snapshots`);
  console.log("INFO snapshots:", snaps.status, JSON.stringify(snaps.body).slice(0, 200));
  await o.ctx.close();
});

// ---- UX12-16: mất mạng → hàng đợi cục bộ, «Chờ mạng», bật mạng tự đồng bộ ----
test("TC-UX12-16 điểm danh ngoại tuyến: hàng đợi giữ thay đổi, dải mất mạng, bật mạng tự đồng bộ lên máy chủ", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher");
  const c = await courseId(o.page, A);
  let gridUrl = "";
  o.page.on("response", (r) => { if (/\/sessions\/[0-9a-f-]{36}\/attendance$/.test(r.url()) && r.request().method() === "GET") gridUrl = r.url(); });
  const puts: string[] = [];
  o.page.on("requestfinished", (r) => { if (r.method() === "PUT" && /attendance/.test(r.url())) puts.push(r.postData() ?? ""); }); // chỉ tính yêu cầu thành công (offline: sự kiện `request` vẫn bắn rồi lỗi)
  await go(o.page, "/attendance", c, 4000);
  expect(gridUrl, "đọc lưới buổi").not.toBe("");
  const rows = o.page.locator("main tbody tr[data-part=student-row]");
  // QC SmkS3: dữ liệu giữ qua các lượt ⇒ chọn 3 hàng đang «Có mặt» (đổi thật mới có hàng đợi)
  const presIdx = await rows.evaluateAll((els) => els.map((e, i) => ((e.querySelector("input[data-status=PRESENT]") as HTMLInputElement | null)?.checked ? i : -1)).filter((i) => i >= 0));
  const pick = presIdx.slice(-3);
  console.log("INFO UX12-16 hàng chọn:", JSON.stringify(pick));
  await o.ctx.setOffline(true);
  for (const i of pick) await rows.nth(i).locator("input[data-status=ABSENT]").check({ force: true });
  await o.page.waitForTimeout(1500);
  const bar = o.page.locator("main").innerText().then((t) => t.replace(/\s+/g, " "));
  const txt = await bar;
  console.log("INFO ngoại tuyến:", txt.slice(0, 220), "| dải:", await o.page.locator("[data-part=offline-banner]").allInnerTexts());
  await expect(o.page.locator("[data-part=offline-banner]")).toContainText(/Chưa có mạng|Mất kết nối mạng/);
  const saveBtn = o.page.getByRole("button", { name: /Lưu điểm danh|Chờ mạng|Đã lưu/ }).first();
  console.log("INFO thanh lưu:", JSON.stringify(await saveBtn.innerText()), "| số PUT lúc offline:", puts.length);
  await saveBtn.click({ timeout: 4000 }).catch((e) => console.log("INFO bấm lưu khi offline:", String(e).slice(0, 120)));
  await o.page.waitForTimeout(800);
  expect(await o.page.getByRole("button", { name: /Chờ mạng/ }).count() + (/Chờ mạng/.test(await o.page.locator("main").innerText()) ? 1 : 0), "thanh lưu «Chờ mạng»").toBeGreaterThan(0);
  expect(puts.length, "chưa gửi gì lúc offline").toBe(0);
  await o.ctx.setOffline(false);
  await expect.poll(() => puts.length, { timeout: 30_000, intervals: [1000] }).toBeGreaterThan(0);
  await o.page.waitForTimeout(2500);
  await o.page.reload(); await o.page.waitForTimeout(4500);
  const ab = await Promise.all(pick.map((i) => rows.nth(i).locator("input[data-status=ABSENT]").isChecked()));
  console.log("INFO sau tải lại (từ máy chủ): ABSENT ở 3 hàng =", JSON.stringify(ab), "| PUT thành công:", puts.length);
  expect(ab, "3 SV Vắng đã lên máy chủ (còn sau tải lại)").toEqual([true, true, true]);
  // trả lại
  for (const i of pick) await rows.nth(i).locator("input[data-status=PRESENT]").check({ force: true });
  await o.page.waitForTimeout(4000);
  await o.ctx.close();
});

// ---- UX12-17: hai người cùng sửa một buổi, người sau lưu → 409, thấy giá trị của người trước + hai lựa chọn ----
test("TC-UX12-17 điểm danh xung đột: A (TA) và B (giảng viên) cùng sửa, B lưu sau → 409, hiện giá trị của A và hai lựa chọn", async ({ browser }, info) => {
  D(info); test.setTimeout(300_000);
  const a = await open(browser, "ta");
  const b = await open(browser, "teacher");
  const c = await courseId(b.page, A);
  await go(a.page, "/attendance", c, 4000);
  await go(b.page, "/attendance", c, 4000);
  const ra = a.page.locator("main tbody tr[data-part=student-row]");
  const rb = b.page.locator("main tbody tr[data-part=student-row]");
  const idx = 25;
  await ra.nth(idx).locator("input[data-status=ABSENT]").check({ force: true });
  await expect(a.page.getByText(/Đã lưu/).first()).toBeVisible({ timeout: 15_000 });
  await a.page.waitForTimeout(1500);
  await rb.nth(idx).locator("input[data-status=LATE]").check({ force: true });
  await b.page.waitForTimeout(6000);
  const t = (await b.page.locator("main").innerText()).replace(/\s+/g, " ");
  console.log("INFO B sau 409:", t.slice(0, 400));
  console.log("INFO nút:", JSON.stringify(await b.page.locator("main button").allInnerTexts()));
  expect(t, "báo người kia vừa đổi").toMatch(/vừa đổi/);
  const row = rb.nth(idx);
  await expect(row.getByRole("button", { name: /Lấy của/ })).toBeVisible();
  await expect(row.getByRole("button", { name: "Giữ của tôi" })).toBeVisible();
  console.log("INFO hàng xung đột:", (await row.innerText()).replace(/\s+/g, " "));
  expect(await row.locator("input[data-status=LATE]").isChecked(), "lựa chọn của B chưa mất").toBe(true);
  // dọn: lấy của người kia rồi đặt lại Có mặt
  await row.getByRole("button", { name: /Lấy của/ }).click();
  await b.page.waitForTimeout(1500);
  await rb.nth(idx).locator("input[data-status=PRESENT]").check({ force: true });
  await b.page.waitForTimeout(4000);
  await a.ctx.close(); await b.ctx.close();
});
