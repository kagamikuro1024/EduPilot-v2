// QC US-P4-02 AC10/AC11 (TC-P4-02-25…28). Chạy từ thư mục frontend: node ../docs/sprints/7/qc/scripts/p4-02-ui.mjs
import { createRequire } from "node:module";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const GW = process.env.GW ?? "https://localhost:543", PASSWORD = process.env.PASSWORD ?? "Edupilot#Seed-2026";
const b = await chromium.launch();
const out = (id, ok, note) => console.log(`${ok ? "PASS" : "FAIL"} ${id} ${note ?? ""}`);
async function login(email, viewport) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport }); const p = await ctx.newPage();
  await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", email); await p.fill("input[type=password]", PASSWORD);
  await p.click('button:has-text("Đăng nhập")'); await p.waitForURL((u) => !u.pathname.startsWith("/login")); await p.waitForTimeout(2500); return { ctx, p };
}
for (const [label, vp] of [["desktop", { width: 1280, height: 900 }], ["375", { width: 375, height: 812 }]]) {
  const { ctx, p } = await login("teacher@edupilot.local", vp);
  const bell = p.getByRole("button", { name: /Thông báo/ }).first(); const box = await bell.boundingBox();
  const lbl = await bell.getAttribute("aria-label"); const vis = (await bell.innerText()).trim();
  out(`TC-P4-02-25 số trên chuông (${label})`, /^(\d|9\+)/.test(vis), `nhãn="${lbl}" chữ hiển thị="${vis}" (kỳ vọng số, tối đa "9+")`);
  if (label === "375") out("TC-P4-02-26 vùng chạm chuông 375px", box.width >= 44 && box.height >= 44, `${box.width}×${box.height}`);
  await bell.click(); await p.waitForTimeout(1000);
  const items = p.locator('[role=dialog] button, [role=menu] button, [role=menuitem], [role=dialog] a').filter({ hasText: /QC thông báo|câu hỏi đang chờ/ });
  const n = await items.count(); const ib = n ? await items.first().boundingBox() : null;
  if (label === "375") out("TC-P4-02-26 vùng chạm mục 375px", !!ib && ib.height >= 44, ib ? `${ib.width}×${ib.height}` : "không tìm thấy mục");
  const sw = await p.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth); if (label === "375") out("TC-P4-02-26 không tràn ngang", sw);
  if (label === "desktop") {
    await items.filter({ hasText: "QC thông báo" }).first().click(); await p.waitForTimeout(1500);
    out("TC-P4-02-25 bấm mục → mở link", p.url().includes("/class/members"), p.url());
  }
  await ctx.close();
}
await b.close();
