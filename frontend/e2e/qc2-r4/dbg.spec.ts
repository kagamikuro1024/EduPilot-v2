import { test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { S } from "./lib";
test.use({ viewport: { width: 375, height: 812 } });
test("dbg login", async ({ page }) => {
  execFileSync("docker", ["exec", "ep-r-redis-1", "sh", "-c", "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del"]);
  const resp: string[] = []; page.on("response", (r) => { if (/auth\/login/.test(r.url())) resp.push(r.status() + ""); });
  page.on("console", (m) => { if (m.type() === "error") console.log("CONSOLE", m.text().slice(0, 200)); });
  await page.goto("/login"); await page.waitForLoadState("networkidle").catch(() => undefined); await page.waitForTimeout(1500);
  await page.locator("input[type=email]").fill(S.D3); await page.locator("input[type=password]").fill(S.pwd);
  console.log("VALUES", await page.locator("input[type=email]").inputValue(), (await page.locator("input[type=password]").inputValue()).length);
  await page.keyboard.press("Enter"); await page.waitForTimeout(6000);
  console.log("URL", page.url(), "resp", resp.join(","), (await page.locator("body").innerText()).replace(/\s+/g, " ").slice(0, 400));
  await page.screenshot({ path: "/tmp/qc2-dbg-login.png" });
});
