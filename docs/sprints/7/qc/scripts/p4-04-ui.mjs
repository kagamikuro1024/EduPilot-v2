// QC US-P4-04 TC-26/27/31/32/33. Chạy từ frontend: node ../docs/sprints/7/qc/scripts/p4-04-ui.mjs
import { createRequire } from "node:module";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const GW = "https://localhost:543", PW = "Edupilot#Seed-2026";
const b = await chromium.launch(); const out = (id, ok, note) => console.log(`${ok ? "PASS" : "FAIL"} ${id} ${note ?? ""}`);
async function login(email, vp = { width: 1280, height: 900 }) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: vp }); const p = await ctx.newPage();
  await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", email); await p.fill("input[type=password]", PW);
  await p.click('button:has-text("Đăng nhập")'); await p.waitForURL((u) => !u.pathname.startsWith("/login")); await p.waitForTimeout(2000); return { ctx, p };
}
const sw = (p, re) => p.getByRole("switch", { name: re }).or(p.getByRole("checkbox", { name: re })).first();
for (const [email, who] of [["teacher", "TEACHER"], ["ta", "TA"], ["sv20", "STUDENT"]]) {
  const { ctx, p } = await login(`${email}@edupilot.local`); await p.goto(`${GW}/settings`); await p.waitForTimeout(2500);
  const t = await p.locator("main").innerText(); const ticket = await sw(p, /Mail báo ticket mới/).count(); const ans = await sw(p, /Mail khi giảng viên trả lời/).count();
  const st = ticket ? await sw(p, /Mail báo ticket mới/).isChecked().catch(() => null) : ans ? await sw(p, /Mail khi giảng viên trả lời/).isChecked().catch(() => null) : null;
  out(`TC-26 ${who}`, who === "STUDENT" ? ans === 1 && ticket === 0 : ticket === 1 && ans === 0, `công tắc ticket=${ticket} (bật=${who !== "STUDENT" ? st : "-"}) công tắc trả lời=${ans} (bật=${who === "STUDENT" ? st : "-"})`);
  if (who === "TEACHER") {
    const reqs = []; p.on("request", (r) => r.method() === "PUT" && /me\/settings/.test(r.url()) && reqs.push(r.postData()));
    const toasts0 = await p.locator("[role=status],[role=alert]").count(); const was = await sw(p, /Mail báo ticket mới/).isChecked();
    await sw(p, /Mail báo ticket mới/).click(); await p.waitForTimeout(100); const nowChecked = await sw(p, /Mail báo ticket mới/).isChecked(); await p.waitForTimeout(1500);
    const toast = (await p.locator("[role=status],[role=alert]").allInnerTexts()).join(" | ");
    out("TC-27 lạc quan", nowChecked !== was && reqs.length === 1 && /version/.test(reqs[0] ?? ""), `trước=${was} sau 100 ms=${nowChecked}; PUT=${JSON.stringify(reqs)}; thông báo="${toast.replace(/\n/g, " ")}"`);
    await sw(p, /Mail báo ticket mới/).click(); await p.waitForTimeout(1200);
  }
  await ctx.close();
}
{ const { ctx, p } = await login("admin@edupilot.local"); await p.goto(`${GW}/observability`); await p.waitForTimeout(3000);
  const t = await p.locator("main").innerText(); const i = t.indexOf("Thư chưa gửi được");
  out("TC-31 mục ở đầu trang", i >= 0 && i < 400, `vị trí chữ=${i}; đoạn: ${t.slice(Math.max(0, i), i + 260).replace(/\n/g, " | ")}`);
  out("TC-31 che người nhận / không thân thư", /\*\*\*@/.test(t) || /[a-z]\*+@/.test(t), `người nhận đã che=${/\*+@/.test(t)}; lộ email đầy đủ=${/sv04@edupilot/.test(t.slice(Math.max(0, i), i + 800))}`);
  await ctx.close(); }
{ const { ctx, p } = await login("teacher@edupilot.local"); await p.goto(`${GW}/observability`); await p.waitForTimeout(2500);
  const t = await p.locator("body").innerText(); out("TC-32 giảng viên", !/Thư chưa gửi được/.test(t), `url=${p.url()} thấy mục=${/Thư chưa gửi được/.test(t)}; đoạn đầu: ${t.slice(0, 120).replace(/\n/g, " | ")}`); await ctx.close(); }
await b.close();
