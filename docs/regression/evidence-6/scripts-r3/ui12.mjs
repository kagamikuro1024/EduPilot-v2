import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8")); const E = JSON.parse(readFileSync("/tmp/qc3e/p8e2.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv.slice(2); const b = await chromium.launch({ headless: true });
const mk = async (user, course, w = 1440, h = 900) => { const { ctx, page } = await newCtx(b, user, w, h); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), course); return { ctx, page }; };
const cal = async (p, course = C.U) => { const code = psql(`select class_code from courses where id='${course}'`); for (let i = 0; i < 4; i++) { await p.goto(`${BASE}/calendar?course=${course}`); await p.waitForTimeout(3500); if ((await text(p)).slice(0, 140).includes(code)) return; await p.evaluate((c) => localStorage.setItem("ep:ui:course", c), course); } console.log("WARN không xác nhận được lớp", code); };
try {
  if (which.includes("11")) {
    const { page } = await mk("gv.s8f", C.U); await cal(page); const cnt = (id) => psql(`select count(*) from calendar_events where id='${id}'`); const id1 = E["QC3 sửa xoá"], id2 = E["QC3 việc 5h"];
    const c0 = cnt(id1), c00 = cnt(id2);
    await page.getByRole("button", { name: "Thêm với QC3 sửa xoá" }).first().click(); await page.waitForTimeout(500); const dlg0 = await page.locator("dialog[open]").count(); await page.getByRole("menuitem", { name: "Xoá" }).click(); await page.waitForTimeout(800);
    const t1 = await text(page); const has = /Đã xoá sự kiện/.test(t1) && t1.includes("Hoàn tác"); const c1 = cnt(id1); await page.waitForTimeout(6500); const t2 = await text(page); const hidden = !t2.includes("Hoàn tác"); const c2 = cnt(id1);
    await page.getByRole("button", { name: "Thêm với QC3 việc 5h" }).first().click(); await page.getByRole("menuitem", { name: "Xoá" }).click(); await page.waitForTimeout(800); await page.getByRole("button", { name: "Hoàn tác" }).first().click(); await page.waitForTimeout(7000); const c3 = cnt(id2); const back = await page.getByText("QC3 việc 5h").count();
    out("TC-P8-03-11", c0 === "1" && has && hidden && c2 === "0" && c3 === "1" && back > 0 && dlg0 === 0, `xoá tại chỗ (menu «Thêm với …» → Xoá) trên ep-r @6201b665 (có eb8f16a0): trước ${c0}; dòng «Đã xoá sự kiện · Hoàn tác»=${has}; DB count ${c1} (trong 5 s) → ${c2} sau 6,5 s; dòng biến mất=${hidden}; sự kiện thứ hai xoá rồi Hoàn tác ≤5 s → DB ${c3}, hiện lại=${back > 0}; hộp thoại modal (<dialog open>)=${dlg0}; menu «Thêm với…» là popover role=dialog, không phải modal`);
  }
} finally { await b.close(); }
