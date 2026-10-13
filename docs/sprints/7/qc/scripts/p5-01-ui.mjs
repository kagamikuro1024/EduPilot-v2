// QC US-P5-01 TC-25/26/28/33 (ngăn kéo "Lịch buổi học"). Chạy từ frontend: CN=<course uuid> node ../docs/sprints/7/qc/scripts/p5-01-ui.mjs
import { createRequire } from "node:module";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const GW = "https://localhost:543", PW = "Edupilot#Seed-2026", CN = process.env.CN; const b = await chromium.launch();
const out = (id, ok, note) => console.log(`${ok ? "PASS" : "FAIL"} ${id} ${note ?? ""}`);
async function login(email, vp = { width: 1280, height: 900 }) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: vp }); const p = await ctx.newPage();
  await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", email); await p.fill("input[type=password]", PW);
  await p.click('button:has-text("Đăng nhập")'); await p.waitForURL((u) => !u.pathname.startsWith("/login")); await p.waitForTimeout(2000); return { ctx, p };
}
{ // vai
  for (const [e, who, want] of [["sv.gioi", "STUDENT", 0], ["ta", "TA", 1], ["teacher", "TEACHER", 1]]) {
    const { ctx, p } = await login(`${e}@edupilot.local`); await p.goto(`${GW}/calendar`); await p.waitForTimeout(2500);
    const n = await p.getByRole("button", { name: /Tạo lịch buổi học/ }).count(); out(`TC-25 nút theo vai ${who}`, n === want, `thấy nút=${n}`); await ctx.close(); }
}
const { ctx, p } = await login("teacher@edupilot.local"); const reqs = [];
p.on("request", (r) => /sessions\/generate/.test(r.url()) && reqs.push({ m: r.method(), dry: /dry_run=true/.test(r.url()), key: r.headers()["idempotency-key"] ?? null, t: Date.now() }));
await p.goto(`${GW}/calendar?course=${CN}`); await p.waitForTimeout(2500);
const btn = p.getByRole("button", { name: /Tạo lịch buổi học/ }).first(); const bb = await btn.boundingBox();
await btn.click(); await p.waitForTimeout(1000); const d = p.locator("[role=dialog]").first();
await d.getByRole("button", { name: /^T2$/ }).first().click();
const tin = d.locator("input[type=time]"); await tin.nth(0).fill("09:00"); await tin.nth(1).fill("11:30");
await d.locator('input[placeholder="P.302"]').fill("P.302");
const din = d.locator("input[type=date]"); await din.nth(0).fill("2026-10-12"); await din.nth(1).fill("2026-12-28"); await din.nth(2).fill("2026-11-02"); await d.getByRole("button", { name: /^Thêm$/ }).click();
const createBtn = () => d.getByRole("button", { name: /^Tạo \d+ buổi$|^Tạo buổi$/ }).first();
const disabledBefore = await createBtn().isDisabled();
await d.getByRole("button", { name: /Xem trước/ }).click(); await p.waitForTimeout(2000);
const t = await d.innerText(); const rows = await d.locator("tbody tr, [role=row]").count();
out("TC-25 Xem trước", /11 buổi/.test(t) && /bỏ 1 ngày nghỉ|1 ngày nghỉ/.test(t), `Tạo trước khi xem trước bị khoá=${disabledBefore}; chữ: ${t.split("\n").filter((l) => /buổi|nghỉ/i.test(l)).slice(0, 4).join(" | ")}; dòng=${rows}`);
console.log("INFO nút:", (await createBtn().innerText()), "| mô tả dưới tiêu đề:", t.split("\n").slice(0, 3).join(" | "));
// offline rồi thử lại (TC-33 + TC-26)
await ctx.setOffline(true); await createBtn().click(); await p.waitForTimeout(1500); const tOff = (await d.innerText()).split("\n").filter((l) => /lỗi|mạng|thử lại|kết nối|Chưa/i.test(l)).join(" | ");
const btnBack = await createBtn().isEnabled().catch(() => null); await ctx.setOffline(false); await p.waitForTimeout(500);
await createBtn().click(); await p.waitForTimeout(3000);
const posts = reqs.filter((r) => !r.dry); const prev = reqs.filter((r) => r.dry);
out("TC-26 khoá", prev.every((r) => !r.key) && posts.length >= 1 && posts.every((r) => r.key), `xem trước key=${JSON.stringify(prev.map((r) => r.key))}; tạo key=${JSON.stringify(posts.map((r) => r.key?.slice(0, 8)))} (số lần gửi=${posts.length})`);
out("TC-33 offline", btnBack !== false, `nút bật lại=${btnBack}; chữ lỗi="${tOff}"; sau online: ${(await d.innerText()).split("\n").filter((l) => /đã tạo|Đã tạo|11 buổi|Buổi 1/.test(l)).slice(0, 3).join(" | ")}`);
const uniq = new Set(posts.map((r) => r.key)).size; out("TC-33 cùng khoá khi thử lại", posts.length >= 2 ? uniq === 1 : true, `lần gửi=${posts.length} khoá khác nhau=${uniq}`);
await p.screenshot({ path: "/tmp/qc/p501-after.png" });
const list = await d.innerText(); console.log("INFO sau tạo:", list.split("\n").filter(Boolean).slice(0, 10).join(" | ").slice(0, 400));
await ctx.close();
{ const { ctx, p } = await login("teacher@edupilot.local", { width: 375, height: 812 }); await p.goto(`${GW}/calendar?course=${CN}`); await p.waitForTimeout(2500);
  await p.getByRole("button", { name: /Tạo lịch buổi học|Lịch buổi học/ }).first().click(); await p.waitForTimeout(1000); const d = p.locator("[role=dialog]").first(); const bx = await d.boundingBox();
  const ov = await p.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);
  const small = await d.locator("button, input, [role=button]").evaluateAll((els) => els.filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && (r.height < 44 || r.width < 44); }).map((e) => (e.innerText || e.getAttribute("aria-label") || e.tagName).slice(0, 20) + ":" + Math.round(e.getBoundingClientRect().width) + "x" + Math.round(e.getBoundingClientRect().height)));
  out("TC-28 375 px", bx.width >= 370 && ov, `ngăn kéo ${Math.round(bx.width)}×${Math.round(bx.height)} không tràn=${ov}; phần tử <44px: ${JSON.stringify(small).slice(0, 400)}`); await ctx.close(); }
await b.close();
