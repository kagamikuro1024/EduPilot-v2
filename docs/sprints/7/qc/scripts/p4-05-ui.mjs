// QC US-P4-05 TC-11/27/28 (chat sinh viên) + TC-42 (hộp tri thức). Chạy từ frontend: STATES=/tmp/qc/states.txt node ../docs/sprints/7/qc/scripts/p4-05-ui.mjs
import { createRequire } from "node:module"; import fs from "node:fs";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const GW = "https://localhost:543", PW = "Edupilot#Seed-2026"; const b = await chromium.launch();
const out = (id, ok, note) => console.log(`${ok ? "PASS" : "FAIL"} ${id} ${note ?? ""}`);
async function login(email, vp = { width: 1280, height: 900 }) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: vp }); const p = await ctx.newPage();
  await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", email); await p.fill("input[type=password]", PW);
  await p.click('button:has-text("Đăng nhập")'); await p.waitForURL((u) => !u.pathname.startsWith("/login")); await p.waitForTimeout(2000); return { ctx, p };
}
const st = fs.readFileSync(process.env.STATES ?? "/tmp/qc/states.txt", "utf8").trim().split("\n").map((l) => l.split(" "));
{ const { ctx, p } = await login("sv.kha@edupilot.local");
  for (const [state, sid] of st) {
    await p.goto(`${GW}/chat?session=${sid}`); await p.waitForTimeout(2500);
    const t = await p.locator("main").innerText(); const lines = t.split("\n").filter((l) => /giảng viên|Đang chờ|Đã đóng|vừa gửi|đã chờ|Hỏi tiếp|Đã rõ/i.test(l)).map((l) => l.trim());
    const bad = /ticket|SLA|độ tin cậy|confidence/i.test(t);
    const btns = await p.getByRole("button").filter({ hasText: /Hỏi tiếp|Đã rõ|Nhờ giảng viên/ }).allInnerTexts();
    out(`TC-27 ${state}`, !bad, `dòng: ${JSON.stringify(lines).slice(0, 300)} nút: ${JSON.stringify(btns)} từ cấm=${bad}`);
  }
  const ans = st.find((s) => s[0] === "ANSWERED"); await p.goto(`${GW}/chat?session=${ans[1]}`); await p.waitForTimeout(2500);
  const dlg0 = await p.locator("[role=dialog],[role=alertdialog]").count(); let dialogSeen = false; p.on("dialog", () => (dialogSeen = true));
  await p.getByRole("button", { name: /Đã rõ/ }).first().click(); await p.waitForTimeout(400);
  const t1 = await p.locator("main").innerText(); const dlg1 = await p.locator("[role=dialog],[role=alertdialog]").count(); const toast = (await p.locator("[role=status],[role=alert]").allInnerTexts()).join("|");
  out("TC-11 Đã rõ lạc quan", /Đã đóng/.test(t1) && /Hoàn tác/.test(t1) && dlg1 === dlg0 && !dialogSeen, `đóng=${/Đã đóng/.test(t1)} hoàn tác=${/Hoàn tác/.test(t1)} hộp thoại=${dlg1 - dlg0} thông báo="${toast.replace(/\n/g, " ")}"`);
  await p.waitForTimeout(5500); const t2 = await p.locator("main").innerText();
  out("TC-11 hết 5 s bỏ Hoàn tác", !/Hoàn tác/.test(t2), `còn Hoàn tác=${/Hoàn tác/.test(t2)}`);
  await ctx.close(); }
await b.close();
