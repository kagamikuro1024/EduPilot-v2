import { expect, test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { appendFileSync } from "node:fs";
import { S, SEEDPW, login, res, settle, sql } from "./lib";
import { call, clearRL, linkToken, mailTo, tokenOf } from "./r4ui-lib";
// DEF-R4-05 chạy lại: token mời / token đặt lại / mã tham gia không còn trong log Caddy (kể cả Next-Url, Next-Router-State-Tree)
test.setTimeout(240_000);
test("[P201-11] quét log sau lời mời, đặt lại mật khẩu, /join/<mã>", async ({ page, browser }) => {
  const since = new Date(Date.now() - 5000).toISOString();
  const adm = await tokenOf("admin@edupilot.local", SEEDPW);
  const t0 = Date.now(); const email = `qc2s4.cad${t0}@edupilot.local`;
  const r = await call("POST", "/admin/users", adm, { email, full_name: "QC2S4 Caddy " + t0, role: "TEACHER" });
  expect(r.status, JSON.stringify(r.body)).toBe(201);
  const inv = linkToken(await mailTo(email, /mời|Lời mời/i, t0), /invite/);
  clearRL(); await page.goto(`/invite/${inv}`); await settle(page, 2500);
  await page.getByLabel("Mật khẩu", { exact: true }).fill("Qc2s4!Pass-Cad9x"); await page.getByLabel("Nhập lại mật khẩu", { exact: true }).fill("Qc2s4!Pass-Cad9x");
  await page.getByRole("button", { name: "Đặt mật khẩu và vào" }).click(); await page.waitForURL(/\/$/, { timeout: 30_000 }); await settle(page, 3000);
  // sau khi vào: điều hướng RSC thêm (đổi trang) để Next gửi Next-Url của trang trước
  await page.goto("/settings"); await settle(page, 2000); await page.goto("/"); await settle(page, 2000);
  // đặt lại mật khẩu: tài khoản SV mô phỏng D1
  clearRL(); const t1 = Date.now(); const fr = await call("POST", "/auth/forgot-password", null, { email: S.D1 }); expect(fr.status).toBe(202);
  const rtok = linkToken(await mailTo(S.D1, /đặt lại|Đặt lại/i, t1), /reset/);
  const ctx2 = await browser.newContext({ ignoreHTTPSErrors: true }); const p2 = await ctx2.newPage();
  await p2.goto(`/reset-password?token=${rtok}`.replace(/^/, "https://localhost:1143")); await settle(p2, 2500);
  await p2.goto("https://localhost:1143/login"); await settle(p2, 1500); await ctx2.close();
  // mã tham gia /join/<mã> khi đã đăng nhập (SV mới D2), rồi điều hướng đi trang khác
  await page.context().clearCookies(); await login(page, S.D2, S.pwd); await page.goto(`/join/${S.F.code}`); await settle(page, 3000); await page.goto("/"); await settle(page, 2500);
  const toks = [inv, rtok, S.F.code];
  const logs = execFileSync("docker", ["logs", "--since", since, "ep-r-caddy-1"], { encoding: "utf8", maxBuffer: 1 << 28, stdio: ["ignore", "pipe", "pipe"] });
  const other: Record<string, number> = {};
  for (const c of ["ep-r-gateway-1", "ep-r-worker-1", "ep-r-pgbouncer-1", "ep-r-redis-1", "ep-r-postgres-1"]) {
    const l = execFileSync("docker", ["logs", "--since", since, c], { encoding: "utf8", maxBuffer: 1 << 28, stdio: ["ignore", "pipe", "pipe"] }) + "";
    other[c] = toks.filter((t) => l.includes(t)).length;
  }
  const hits = toks.map((t) => logs.split("\n").filter((l) => l.includes(t)).length);
  const lines = logs.split("\n"); const nuHdr = lines.filter((l) => /Next-Url|Next-Router-State-Tree/.test(l)).length;
  const red = (logs.match(/(invite|join)\/REDACTED/g) ?? []).length; const tokR = (logs.match(/token=REDACTED/g) ?? []).length;
  const poor = lines.filter((l) => /\/invite\/|\/join\/|token=/.test(l) && !/REDACTED/.test(l) && l.trim()).length;
  appendFileSync("../docs/regression/evidence-4/logs/s4n-caddy-scan.txt", `since ${since}; tokens: invite(43)=${hits[0]} reset=${hits[1]} join=${hits[2]} dòng Caddy; Next-Url/Next-Router-State-Tree trong log=${nuHdr}; /invite|/join REDACTED=${red} token=REDACTED=${tokR}; dòng chứa đường dẫn bí mật chưa che=${poor}; container khác=${JSON.stringify(other)}; tổng dòng Caddy=${lines.length}\n`);
  const ok = hits.every((h) => h === 0) && Object.values(other).every((v) => v === 0) && poor === 0 && red >= 1;
  res("TC-P201-11", ok, `DEF-R4-05 @f5f8a8be: lời mời mới (token 43 ký tự, mở /invite/<token>, đặt mật khẩu, đi /settings và /), đặt lại mật khẩu (?token=), /join/<mã> khi đã đăng nhập → quét docker logs ep-r-caddy-1 từ ${since}: dòng chứa token mời=${hits[0]}, token đặt lại=${hits[1]}, mã tham gia=${hits[2]}; tiêu đề Next-Url/Next-Router-State-Tree trong log=${nuHdr}; /invite|/join REDACTED=${red}; token=REDACTED=${tokR}; dòng còn lộ đường dẫn bí mật=${poor}; gateway/worker/pgbouncer/redis/postgres chứa token=${JSON.stringify(other)}; bằng chứng logs/s4n-caddy-scan.txt`);
});
