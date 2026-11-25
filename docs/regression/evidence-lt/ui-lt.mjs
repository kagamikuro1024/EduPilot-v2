// LT-rerun (qc3) trên ep-qc 3f91ed90: PE04-48 (nav) + PE04-44 (hành động theo trạng thái, TA / GV)
import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs";
const X = JSON.parse(readFileSync("/tmp/qc3lt/exams.json", "utf8"));
const only = process.argv[2] ?? "all";
const b = await chromium.launch({ headless: true });
const navItems = (p) => p.locator("nav a:visible").evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean));
try {
  if (only === "all" || only === "nav") {
    const res = {};
    for (const [u, k] of [["sv.gioi", "SV"], ["ta", "TA"], ["teacher", "GV"], ["admin", "ADMIN"]]) {
      const { ctx, page } = await newCtx(b, u, 1440, 900); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), X.course);
      await page.goto(`${BASE}/?course=${X.course}`); await page.waitForTimeout(3000); res[k] = await navItems(page); await ctx.close();
    }
    const mob = {};
    for (const [u, k] of [["sv.gioi", "SV"], ["ta", "TA"]]) {
      const { ctx, page } = await newCtx(b, u, 375, 800); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), X.course);
      await page.goto(`${BASE}/?course=${X.course}`); await page.waitForTimeout(3000);
      const bar = await page.locator("nav a:visible,nav button:visible").evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean));
      const more = page.getByRole("button", { name: /Thêm/ }).first(); let moreItems = [];
      if (await more.count()) { await more.click(); await page.waitForTimeout(800); moreItems = await page.locator("[role=menu] a:visible,[role=dialog] a:visible,dialog[open] a:visible").evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean)); }
      mob[k] = { bar, moreItems }; await ctx.close();
    }
    console.log(JSON.stringify({ res, mob }, null, 1));
    const n = Object.fromEntries(Object.entries(res).map(([k, v]) => [k, v.length])); const has = Object.fromEntries(Object.entries(res).map(([k, v]) => [k, v.includes("Bài thi")]));
    out("TC-PE04-48", has.SV && has.TA && has.GV && !has.ADMIN && mob.SV.moreItems.includes("Bài thi"), JSON.stringify({ counts: n, hasExams: has, mobileSV: mob.SV, mobileTA: mob.TA, items: res }));
  }
  if (only === "all" || only === "exam44") {
    const state = { DRAFT: X.DRAFT, OPEN: X.OPEN, CLOSED: X.CLOSED }; const r = {};
    for (const who of ["ta", "teacher"]) {
      const { ctx, page } = await newCtx(b, who, 1440, 900); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), X.course);
      for (const [st, id] of Object.entries(state)) {
        await page.goto(`${BASE}/exams/${id}?course=${X.course}`); await page.waitForTimeout(3500);
        const t = await text(page); const btns = await page.locator("main button:visible, main a[role=button]:visible, main a:visible").evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean));
        r[`${who}/${st}`] = { schedule: /Lên lịch/.test(t) && btns.includes("Lên lịch"), extend: btns.includes("Gia hạn"), results: btns.some((x) => /Xem kết quả/.test(x)), note: /Chỉ giảng viên lên lịch/.test(t), btns: btns.slice(0, 14) };
        await page.screenshot({ path: `shots/pe44-${who}-${st}.png` });
      }
      await ctx.close();
    }
    console.log(JSON.stringify(r, null, 1));
    const ok = !r["ta/DRAFT"].schedule && !r["ta/DRAFT"].note && !r["ta/OPEN"].note && !r["ta/CLOSED"].note && r["teacher/DRAFT"].schedule && r["teacher/OPEN"].extend && r["teacher/CLOSED"].results;
    out("TC-PE04-44", ok, JSON.stringify(r));
  }
} finally { await b.close(); }
