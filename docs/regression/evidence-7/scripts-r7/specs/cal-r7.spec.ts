import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, psql, newUser, expect } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("r7-cal", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("r7-cal", `INFO ${tc} ${note}`);
const F = FX("fx_cal") as { CE: string; CF: string; tE: string; tF: string; S: string[] };
const nS = (c: string) => psql(`select count(*) from class_sessions where course_id='${c}'`);
async function plan(p: Page, o: { day: string; from: string; to: string; skip?: string }) {
  const d = p.getByRole("dialog");
  await d.getByRole("button", { name: o.day, exact: true }).click();
  await d.getByLabel("Bắt đầu").fill("09:00"); await d.getByLabel("Kết thúc").fill("11:30"); await d.getByLabel("Phòng").fill("P.302");
  await d.getByLabel("Từ ngày").fill(o.from); await d.getByLabel("Đến ngày").fill(o.to);
  if (o.skip) { await d.getByLabel("Ngày nghỉ").fill(o.skip); await d.getByRole("button", { name: "Thêm", exact: true }).click(); }
}
const track = (p: Page) => { const a: Array<{ url: string; key: string | undefined; body: any }> = []; p.on("request", (r) => { if (r.method() === "POST" && /\/sessions\/generate/.test(r.url())) a.push({ url: r.url(), key: r.headers()["idempotency-key"], body: JSON.parse(r.postData() ?? "{}") }); }); return a; };

test("26 Idempotency-Key: xem trước không có, tạo thật khoá mới mỗi lần", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, F.tE); const calls = track(page);
  await go(page, `/calendar?course=${F.CE}`);
  const before = nS(F.CE);
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click(); const d = page.getByRole("dialog"); await d.waitFor();
  await plan(page, { day: "T3", from: "2027-09-01", to: "2027-09-30" });
  for (let i = 0; i < 3; i++) { await d.getByRole("button", { name: "Xem trước" }).click(); await d.getByText(/Sẽ tạo \d+ buổi/).waitFor(); await page.waitForTimeout(700); }
  const nPrev = calls.filter((c) => /dry_run=true/.test(c.url)).length;
  const mid = nS(F.CE);
  await d.getByRole("button", { name: /^Tạo \d+ buổi$/ }).click(); await d.getByText(/Đã tạo \d+ buổi/).waitFor({ timeout: 20000 }); await page.waitForTimeout(500);
  const c1 = nS(F.CE);
  if ((await page.getByRole("dialog").count()) === 0) await page.getByRole("button", { name: "Tạo lịch buổi học" }).click();
  await d.waitFor(); await plan(page, { day: "T5", from: "2027-09-01", to: "2027-09-30" });
  await d.getByRole("button", { name: "Xem trước" }).click(); await d.getByText(/Sẽ tạo \d+ buổi/).waitFor();
  await d.getByRole("button", { name: /^Tạo \d+ buổi$/ }).click(); await d.getByText(/Đã tạo \d+ buổi/).waitFor({ timeout: 20000 }); await page.waitForTimeout(500);
  const c2 = nS(F.CE);
  const prevs = calls.filter((c) => /dry_run=true/.test(c.url)); const reals = calls.filter((c) => !/dry_run/.test(c.url));
  const rk = reals.map((c) => c.key); const pk = prevs.map((c) => c.key ?? null);
  await shot(page, "r7-p501-26");
  R("TC-P5-01-26", prevs.length >= 4 && prevs.every((c) => !c.key) && reals.length === 2 && !!rk[0] && !!rk[1] && rk[0] !== rk[1] && mid === before && Number(c1) > Number(before) && Number(c2) > Number(c1), `[DEF-R7-10, 01b3f6d9] lớp QC3U9: ${nPrev} lần «Xem trước» liên tiếp rồi tổng ${prevs.length} (dry_run=true), Idempotency-Key = ${JSON.stringify(pk)} (không khoá); buổi trong lớp trước/sau xem trước: ${before}/${mid} (xem trước không ghi); ${reals.length} lần «Tạo N buổi» thật: khoá «${rk[0]}» ≠ «${rk[1]}», không trùng khoá xem trước=${rk.every((k) => !pk.includes(k ?? null))}; số buổi ${before} → ${c1} → ${c2}`);
  await ctx.close();
});
