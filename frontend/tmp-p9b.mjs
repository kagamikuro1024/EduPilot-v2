import { chromium } from "@playwright/test";
const b = await chromium.launch();
const ctx = await b.newContext({ viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
const p = await ctx.newPage();
const errs = [];
p.on("console", (m) => m.type() === "error" && errs.push(m.text().slice(0, 160)));
p.on("pageerror", (e) => errs.push("PAGEERR " + e.message.slice(0, 200)));
const U = "http://localhost:3098";
await p.goto(U + "/login");
await p.getByLabel(/email/i).fill("sv1@p9.local");
await p.getByLabel(/mật khẩu/i).first().fill("Dev@12345");
await p.getByRole("button", { name: /đăng nhập/i }).click();
await p.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 20000 });
await p.goto(U + "/practice");
await p.getByRole("button", { name: "Đã hiểu" }).click({ timeout: 4000 }).catch(() => {});
await p.getByRole("button", { name: /^Luyện 10 câu$/ }).first().click();
await p.waitForURL(/\/practice\/[0-9a-f-]{36}/);
await p.waitForTimeout(1200);
await p.screenshot({ path: "/tmp/p9shots/topic-q.png" });
for (let i = 0; i < 10; i++) {
  const type = await p.locator("ul[aria-label]").first().getAttribute("aria-label").catch(() => null);
  const opt = p.locator("main ul button").first();
  if (await opt.count()) await opt.click(); else await p.getByRole("textbox").first().fill("thử");
  await p.getByRole("button", { name: /^(Kiểm tra|Trả lời)$/ }).click();
  await p.waitForTimeout(400);
  if (i === 0) {
    await p.getByRole("button", { name: "Giải thích thêm" }).click().catch(() => {});
    await p.waitForTimeout(1500);
    await p.screenshot({ path: "/tmp/p9shots/topic-feedback.png" });
  }
  const finish = p.getByRole("button", { name: "Xem kết quả" });
  if (await finish.count()) { await finish.click(); break; }
  await p.getByRole("button", { name: "Câu tiếp theo" }).click();
}
await p.waitForTimeout(1500);
await p.screenshot({ path: "/tmp/p9shots/topic-result.png", fullPage: true });
console.log("H1:", await p.locator("h1").first().innerText());
console.log(errs.join("\n") || "no errors");
await b.close();
