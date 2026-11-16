import { test } from "@playwright/test";
import { cid, go, login } from "./lib";
test("explore esc", async ({ browser }) => {
  test.setTimeout(100000);
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage();
  await login(p, "teacher"); await go(p, `/gradebook/scheme?course=${cid("QD8R6W")}`); await p.waitForTimeout(3000);
  const L = () => p.evaluate(() => JSON.stringify([...document.querySelectorAll("dialog[open]")].map((e) => e.tagName + "#" + (e.className.match(/[A-Za-z]+$/)?.[0] ?? "") + ":" + (e.textContent ?? "").slice(0, 30))));
  console.log("DLG0", await L());
  await p.getByRole("button", { name: /^Xác nhận công thức$/ }).first().click(); await p.waitForTimeout(1500); console.log("DLG1", await L());
  await p.keyboard.press("Escape"); await p.waitForTimeout(800); console.log("DLG2 after Esc", await L());
  console.log("FOCUS", await p.evaluate(() => (document.activeElement as HTMLElement | null)?.textContent?.slice(0, 30) ?? document.activeElement?.tagName));
  await p.getByRole("button", { name: /^Xác nhận công thức$/ }).first().click(); await p.waitForTimeout(1200);
  await p.getByRole("button", { name: /^Để sau$/ }).click(); await p.waitForTimeout(800); console.log("DLG3 after Để sau", await L());
});
