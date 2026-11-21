import { test, expect } from "@playwright/test";
import { execFileSync } from "node:child_process";
import AxeBuilder from "@axe-core/playwright";
import { S, login, res, settle, text, shot, sql, TECH } from "./lib";
test.setTimeout(240_000);
const clearJoin = () => execFileSync("docker", ["exec", "ep-r-redis-1", "sh", "-c", "redis-cli --scan --pattern 'ep:join:fail:*' | xargs -r redis-cli del"]);
test.use({ viewport: { width: 375, height: 812 }, hasTouch: true });
test("[P209-42] /join 375", async ({ page }) => {
  await login(page, S.D1, S.pwd); await page.goto("/join"); await settle(page);
  const inp = page.getByLabel("Mã tham gia"); const btn = page.getByRole("button", { name: "Xem lớp" });
  await inp.pressSequentially("an7k2mq"); const v1 = await inp.inputValue();
  await inp.fill(""); await inp.pressSequentially("0O1IL"); const v2 = await inp.inputValue();
  await inp.fill(""); await inp.pressSequentially(" a n 7 k "); const v3 = await inp.inputValue();
  const prim = await page.locator("main button").evaluateAll((bs) => bs.filter((b) => /primary/.test(b.className) || b.getAttribute("data-variant") === "primary").length);
  const dis0 = await btn.isDisabled();
  await inp.fill(S.O.code); await expect(btn).toBeEnabled(); await btn.click(); await settle(page, 1500);
  const pv = await text(page, "[data-part=join-preview]"); await shot(page, "join-preview-375");
  const joinBtn = page.getByRole("button", { name: "Tham gia lớp", exact: true }); const hasJ = await joinBtn.count();
  const ox = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  await joinBtn.click(); await settle(page, 1500); const done = await text(page);
  res("TC-P209-42a", v1 === "AN7K2MQ" && !/[01OIL]/.test(v2) && v3 === "AN7K" && prim <= 1 && hasJ === 1, `ô mã 375 px: gõ «an7k2mq» → «${v1}» (tự viết hoa); gõ «0O1IL» → «${v2}» (không nhận); « a n 7 k » → «${v3}»; nút chính Xem lớp (≤ 1 primary: ${prim}; khoá khi chưa đủ 7 ký tự: ${dis0}); xem trước «${pv}»; nút «Tham gia lớp» ${hasJ}; tràn ngang ${ox}; sau tham gia: «${done.slice(0, 80)}»`);
});
test("[P209-42b] lớp cần duyệt + 6 mã sai", async ({ page }) => {
  await login(page, S.D3, S.pwd); await page.goto("/join"); await settle(page); await page.getByLabel("Mã tham gia").fill(S.F.code); await page.getByRole("button", { name: "Xem lớp" }).click(); await settle(page, 1500);
  const pvF = await text(page, "[data-part=join-preview]"); await page.getByRole("button", { name: "Tham gia lớp", exact: true }).click(); await settle(page, 1500); const doneF = await text(page);
  res("TC-P209-42b", /Lớp này cần giảng viên duyệt/.test(pvF) && /chờ giảng viên duyệt/i.test(doneF), `lớp cần duyệt: xem trước «${pvF}», sau tham gia «${doneF.slice(0, 90)}»`);
});
test("[P209-42c] 6 mã sai", async ({ page }) => {
  await login(page, S.D2, S.pwd); clearJoin(); await page.goto("/join"); await settle(page); let err = "";
  for (let i = 0; i < 6; i++) { await page.getByLabel("Mã tham gia").fill(["ZZZZZ22", "ZZZZZ23", "ZZZZZ24", "ZZZZZ25", "ZZZZZ26", "ZZZZZ27"][i]); const b = page.getByRole("button", { name: "Xem lớp" }); if (await b.isEnabled()) await b.click(); await page.waitForTimeout(900); if (i === 0) err = await text(page); }
  await settle(page, 800); const lock = await text(page); await shot(page, "join-locked-375");
  res("TC-P209-42c", /Thử lại sau\s*\d\d:\d\d/.test(lock) && !TECH.test(lock), `mã sai lần 1: «${err.slice(0, 120)}»; sau 6 mã sai: «${lock.slice(lock.indexOf("Bạn đã thử"), lock.indexOf("Bạn đã thử") + 70)}» (đếm ngược mm:ss, ô khoá); không từ kỹ thuật`);
});
test("[P209-43] /join/[code] chưa đăng nhập", async ({ page, context }) => {
  const code = S.R.code; await context.clearCookies(); await page.goto(`/join/${code}`); await settle(page, 2000);
  const u1 = page.url(); await page.waitForTimeout(500);
  await page.locator("input[type=email]").fill(S.D2); await page.locator("input[type=password]").fill(S.pwd);
  execFileSync("docker", ["exec", "ep-r-redis-1", "sh", "-c", "(redis-cli --scan --pattern 'ep:rl:auth*'; redis-cli --scan --pattern 'ep:rl:ip:*') | xargs -r redis-cli del"]);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 }).catch(() => undefined); await settle(page, 2500);
  const u2 = page.url(); const body = await text(page); const hist = await page.evaluate(() => document.referrer + "|" + location.href);
  const caddy = execFileSync("bash", ["-c", `docker logs ep-r-caddy-1 2>&1 | grep -c "${code}" || true`], { encoding: "utf8" }).trim();
  const caddyNonPath = execFileSync("bash", ["-c", `docker logs ep-r-caddy-1 2>&1 | grep "${code}" | grep -v '"uri":"/join/${code}"' | grep -vc 'invite' || true`], { encoding: "utf8" }).trim();
  const ox = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  res("TC-P209-43", /\/login\?next=%2Fjoin%2F/i.test(u1) && /\/join$/.test(u2) && /VFC|Lớp|lớp/.test(body) && ox === 0, `chưa đăng nhập mở /join/${code} → ${u1.replace(code, "<mã>")}; sau đăng nhập → ${u2} (mã bị xoá khỏi thanh địa chỉ bằng history.replaceState); xem trước hiện «${body.slice(body.indexOf("Tham gia lớp"), body.indexOf("Tham gia lớp") + 110)}»; tràn ngang ${ox}; log Caddy: ${caddy} dòng chứa mã (mã nằm trong đường dẫn /join/<mã> — ghi nhận Q-QC), dòng ngoài đường dẫn/invite: ${caddyNonPath}`);
});
