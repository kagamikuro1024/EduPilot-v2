import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, psql, newUser, expect } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p501", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("p501", `INFO ${tc} ${note}`);
const F = FX("fx_att") as { C: string; C0: string; CE: string; CF: string; S: string[] };
const nS = (c: string) => psql(`select count(*) from class_sessions where course_id='${c}'`);
async function plan(p: Page, o: { day: string; from: string; to: string; skip?: string }) {
  const d = p.getByRole("dialog");
  await d.getByRole("button", { name: o.day, exact: true }).click();
  await d.getByLabel("Bắt đầu").fill("09:00"); await d.getByLabel("Kết thúc").fill("11:30"); await d.getByLabel("Phòng").fill("P.302");
  await d.getByLabel("Từ ngày").fill(o.from); await d.getByLabel("Đến ngày").fill(o.to);
  if (o.skip) { await d.getByLabel("Ngày nghỉ").fill(o.skip); await d.getByRole("button", { name: "Thêm", exact: true }).click(); }
}
const track = (p: Page) => { const a: Array<{ url: string; key: string | undefined; body: any }> = []; p.on("request", (r) => { if (r.method() === "POST" && /\/sessions\/generate/.test(r.url())) a.push({ url: r.url(), key: r.headers()["idempotency-key"], body: JSON.parse(r.postData() ?? "{}") }); }); return a; };

test("25/26 calendar vi tạo lịch", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const calls = track(page);
  await go(page, `/calendar?course=${F.CE}`);
  const btn = await page.getByRole("button", { name: "Tạo lịch buổi học" }).count();
  const stu = await newUser(browser, "qc3at100"); await go(stu.page, `/calendar?course=${F.CE}`); const btnStu = await stu.page.getByRole("button", { name: "Tạo lịch buổi học" }).count(); await stu.ctx.close();
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click(); const d = page.getByRole("dialog"); await d.waitFor();
  const head = (await d.innerText()).split("\n").slice(0, 3).join(" | ");
  const dis0 = [await d.getByRole("button", { name: "Tạo buổi" }).isDisabled(), await d.getByRole("button", { name: "Xem trước" }).isDisabled()];
  const ph = await d.getByLabel("Phòng").getAttribute("placeholder"); const types = [await d.getByLabel("Từ ngày").getAttribute("type"), await d.getByLabel("Đến ngày").getAttribute("type"), await d.getByLabel("Ngày nghỉ").getAttribute("type")];
  await plan(page, { day: "T2", from: "2027-03-01", to: "2027-05-17", skip: "2027-03-22" });
  await d.getByRole("button", { name: "Xem trước" }).click(); const sum = d.getByText(/Sẽ tạo \d+ buổi/); await sum.waitFor(); const st = (await sum.innerText()).trim();
  const rowsN = await d.getByRole("row").count(); const cols = await d.getByRole("columnheader").allInnerTexts();
  const afterPrev = nS(F.CE); const pv = calls.find((c) => /dry_run=true/.test(c.url))!;
  const tbtn = d.getByRole("button", { name: /^Tạo \d+ buổi$/ }); const tTxt = (await tbtn.innerText()).trim(); const tEn = await tbtn.isEnabled();
  await tbtn.click(); await d.getByText(/Đã tạo \d+ buổi/).waitFor({ timeout: 20000 }); const k1 = calls.filter((c) => !/dry_run/.test(c.url));
  const menu = d.getByRole("button", { name: /^Thêm với buổi 1/ }); const hasMenu = await menu.count(); let items: string[] = []; if (hasMenu) { await menu.first().click(); items = await page.getByRole("menuitem").allInnerTexts(); await page.keyboard.press("Escape"); }
  const created = nS(F.CE);
  R("TC-P5-01-25", btn === 1 && btnStu === 0 && dis0[0] && dis0[1] && ph === "P.302" && types.every((t) => t === "date") && st === "Sẽ tạo 11 buổi · bỏ 1 ngày nghỉ" && rowsN <= 61 && rowsN === 12 && cols.join("|").replace(/\s/g, "") === "Buổi|Ngày|Giờ" && afterPrev === "0" && tTxt === "Tạo 11 buổi" && tEn && created === "11" && hasMenu === 1 && items.some((x) => /Sửa/.test(x)) && items.some((x) => /Xoá/.test(x)), `nút «Tạo lịch buổi học»: GV=${btn}, SV=${btnStu}; ngăn kéo đầu: «${head}»; trước xem trước «Tạo buổi»/«Xem trước» khoá=${dis0}; placeholder phòng «${ph}», ô Từ/Đến/Ngày nghỉ type=${types}; xem trước: «${st}», bảng ${rowsN - 1} dòng (cột ${cols.join("/")}); xem trước không ghi (DB ${afterPrev} buổi); nút «${tTxt}» bật=${tEn}; tạo → DB ${created} buổi; danh sách buổi có menu …: ${JSON.stringify(items)}`);
  // 422
  await d.getByLabel("Đến ngày").fill("2030-01-01"); await d.getByRole("button", { name: "Xem trước" }).click(); await page.waitForTimeout(2500); const err = (await d.innerText()).replace(/\n+/g, " | "); const inv = await d.getByLabel("Đến ngày").getAttribute("aria-invalid"); const m422 = err.match(/Khoảng ngày[^|]*/)?.[0] ?? err.slice(-160);
  R("TC-P5-01-25b", true, `lỗi 422 hiện tại ô: «${m422}», Đến ngày aria-invalid=${inv}`);
  // 26: khoá
  await d.getByLabel("Đến ngày").fill("2027-05-17"); await d.getByRole("button", { name: "T2", exact: true }).click(); await d.getByRole("button", { name: "T4", exact: true }).click(); await d.getByRole("button", { name: "Xem trước" }).click(); await d.getByText(/Sẽ tạo \d+ buổi/).waitFor();
  await d.getByRole("button", { name: /^Tạo \d+ buổi$/ }).click(); await d.getByText(/Đã tạo \d+ buổi/).waitFor({ timeout: 20000 });
  const prevs = calls.filter((c) => /dry_run=true/.test(c.url)); const reals = calls.filter((c) => !/dry_run/.test(c.url));
  R("TC-P5-01-26", prevs.length >= 2 && prevs.every((c) => !c.key) && reals.length === 2 && !!reals[0].key && !!reals[1].key && reals[0].key !== reals[1].key, `mạng: ${prevs.length} lần xem trước, Idempotency-Key = ${JSON.stringify(prevs.map((c) => c.key ?? null))} (không có khoá); ${reals.length} lần tạo thật, khoá lần 1 «${reals[0]?.key}» ≠ lần 2 «${reals[1]?.key}»`);
  await ctx.close();
});

test("28 375 px", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t", { width: 375, height: 812 }); await go(page, `/calendar?course=${F.CE}`);
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click(); const d = page.getByRole("dialog"); await d.waitFor(); await page.waitForTimeout(600);
  const g = await page.evaluate(() => { const dl = document.querySelector("[role=dialog]") as HTMLElement; const r = dl.getBoundingClientRect(); const small = Array.from(dl.querySelectorAll("button,input,select,a")).filter((e) => { const b = e.getBoundingClientRect(); const s = getComputedStyle(e); return b.width > 0 && b.height > 0 && b.height < 44 && s.visibility !== "hidden" && !(e as HTMLInputElement).hidden; }).map((e) => `${(e as HTMLElement).innerText?.slice(0, 14) || e.getAttribute("aria-label") || (e as HTMLInputElement).type}:${Math.round(e.getBoundingClientRect().height)}`); return { l: r.left, w: r.width, h: r.height, iw: window.innerWidth, ih: window.innerHeight, sw: document.documentElement.scrollWidth, dsw: dl.scrollWidth, dcw: dl.clientWidth, small }; });
  await shot(page, "p501-28");
  R("TC-P5-01-28", g.w >= g.iw - 1 && g.h >= g.ih - 2 && g.sw <= g.iw && g.dsw <= g.dcw + 1 && g.small.length === 0, `375×812: ngăn kéo rộng ${g.w.toFixed(0)}/${g.iw}, cao ${g.h.toFixed(0)}/${g.ih} (toàn màn hình); scrollWidth trang ${g.sw} ≤ ${g.iw}, ngăn kéo ${g.dsw} ≤ ${g.dcw}; phần tử tương tác cao < 44 px: ${JSON.stringify(g.small)}`);
  await ctx.close();
});

test("33 mất mạng khi tạo", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const calls = track(page); await go(page, `/calendar?course=${F.CF}`);
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click(); const d = page.getByRole("dialog"); await d.waitFor();
  await plan(page, { day: "T6", from: "2027-06-04", to: "2027-07-30" }); await d.getByRole("button", { name: "Xem trước" }).click(); const sum = await d.getByText(/Sẽ tạo \d+ buổi/).innerText(); const N = Number(/(\d+)/.exec(sum)![1]);
  // (a) mất mạng thật: request không tới máy chủ
  await ctx.setOffline(true); const btn = d.getByRole("button", { name: new RegExp(`^Tạo ${N} buổi$`) }); await btn.click(); await d.getByRole("alert").waitFor({ timeout: 40000 }).catch(() => undefined); const alert = (await d.getByRole("alert").innerText().catch(() => "")).replace(/\n+/g, " | "); const back = await btn.isEnabled(); const n0 = nS(F.CF);
  await ctx.setOffline(false); await btn.click(); await d.getByText(/Đã tạo \d+ buổi/).waitFor({ timeout: 30000 }); const n1 = nS(F.CF);
  const real = calls.filter((c) => !/dry_run/.test(c.url)); const keys = new Set(real.map((c) => c.key));
  R("TC-P5-01-33", back && n0 === "0" && n1 === String(N) && keys.size === 1, `xem trước «${sum}»; setOffline khi bấm «Tạo ${N} buổi»: ô lỗi «${alert}», nút trở lại=${back}, DB ${n0} buổi; online + bấm lại: DB ${n1} buổi (đúng N=${N}); ${real.length} yêu cầu tạo, số Idempotency-Key khác nhau=${keys.size} (thử lại dùng cùng khoá)`);
  // (b) phản hồi bị cắt sau khi máy chủ đã ghi: thử lại cùng khoá không tạo đôi
  await d.getByRole("button", { name: "T6", exact: true }).click(); await d.getByRole("button", { name: "T2", exact: true }).click(); await d.getByRole("button", { name: "Xem trước" }).click(); const sum2 = await d.getByText(/Sẽ tạo \d+ buổi/).innerText(); const N2 = Number(/(\d+)/.exec(sum2)![1]);
  let k = 0; await page.route(/\/sessions\/generate(\?|$)/, async (r) => { if (/dry_run/.test(r.request().url())) return r.continue(); k++; if (k === 1) { await r.fetch(); return r.abort("failed"); } return r.continue(); });
  const calls0 = calls.length; await d.getByRole("button", { name: new RegExp(`^Tạo ${N2} buổi$`) }).click(); await d.getByRole("alert").waitFor({ timeout: 40000 }).catch(() => undefined); const nMid = nS(F.CF);
  await d.getByRole("button", { name: new RegExp(`^Tạo ${N2} buổi$`) }).click().catch(() => undefined); await page.waitForTimeout(6000); const n2 = nS(F.CF);
  const ks = new Set(calls.slice(calls0).filter((c) => !/dry_run/.test(c.url)).map((c) => c.key));
  R("TC-P5-01-33b", nMid === String(N + N2) && n2 === String(N + N2) && ks.size === 1, `máy chủ đã ghi nhưng phản hồi bị cắt: DB ${nMid} buổi (= ${N}+${N2}); bấm thử lại cùng khoá: DB ${n2} buổi (không tạo đôi); số khoá khác nhau=${ks.size}`);
  await ctx.close();
});
