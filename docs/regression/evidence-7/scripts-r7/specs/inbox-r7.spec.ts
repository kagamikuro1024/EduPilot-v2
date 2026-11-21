import { test } from "@playwright/test";
import { go, FX, shot, log, psql, newUser } from "./lib";
// chạy lại sau 425396a7: TC-P4-03-33 / 36 (DEF-R7-05 / DEF-R7-04, sha 24036aae). Lớp QC3G, GV qc3t5 (fx_ui5.json do setup_inbox.py dựng)
const R = (tc: string, ok: boolean, note: string) => log("r7-inbox", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const rows = (p: any) => p.locator("[data-part=inbox-list] [data-ticket-id]");
const f = FX("fx_ui5") as { G: string; T: string; open: string[]; claimed: string[] };
const sel = (p: any) => p.url().split("ticket=")[1];

test("36 J/K R Ctrl+Enter, bộ lọc Đang chờ", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, f.T);
  await go(page, `/inbox?course=${f.G}`); await page.waitForTimeout(1200);
  const filt = await page.getByRole("radio", { name: "Đang chờ" }).first().getAttribute("aria-checked");
  const ids0 = await rows(page).evaluateAll((els) => els.map((e) => e.getAttribute("data-ticket-id")));
  await rows(page).first().click(); await page.waitForTimeout(800);
  const s0 = sel(page); await page.keyboard.press("j"); await page.waitForTimeout(700); const s1 = sel(page); await page.keyboard.press("k"); await page.waitForTimeout(700); const s2 = sel(page);
  await page.getByRole("button", { name: "Nhận", exact: true }).click(); await page.waitForTimeout(1500);
  await page.keyboard.press("r"); const focused = await page.evaluate(() => document.activeElement?.tagName + ":" + (document.activeElement?.getAttribute("aria-label") ?? ""));
  await page.keyboard.type("jjkk không đổi ticket"); const s3 = sel(page); const val = await page.getByLabel("Câu trả lời gửi sinh viên").inputValue();
  const title = await page.getByRole("button", { name: "Gửi trả lời" }).getAttribute("title");
  await page.keyboard.press("Control+Enter"); await page.waitForTimeout(3000); const s4 = sel(page);
  const st = psql(`select status from escalation_tickets where id='${s2}'`); const stNext = psql(`select status from escalation_tickets where id='${s4}'`);
  const ids1 = await rows(page).evaluateAll((els) => els.map((e) => e.getAttribute("data-ticket-id")));
  await shot(page, "r7-p403-36");
  R("TC-P4-03-36", filt === "true" && s1 !== s0 && s2 === s0 && /TEXTAREA/.test(focused) && s3 === s2 && val.startsWith("jjkk") && st === "ANSWERED" && !!s4 && s4 !== s2 && s4 === ids0[1] && stNext === "OPEN", `bộ lọc «Đang chờ» checked=${filt}, ${ids0.length} phiếu; J → ${s0?.slice(-6)}→${s1?.slice(-6)}, K → ${s2?.slice(-6)}; Nhận; «R» focus ${focused}; gõ «jjkk…» không đổi phiếu (${s3 === s2}), ô «${val}»; tooltip «Gửi trả lời»=«${title}»; Ctrl+Enter → phiếu ${s2?.slice(-6)} ${st}, URL sang ${s4?.slice(-6)} (kỳ vọng phiếu kế ${ids0[1]?.slice(-6)}; trạng thái ${stNext}); danh sách sau: ${ids1.length} phiếu, còn ${ids1.includes(s2) ? "CÓ" : "không có"} phiếu vừa trả lời`);
  // 36b: bộ lọc «Tất cả»: phiếu CLAIMED + Ctrl+Enter
  await page.getByRole("radio", { name: "Tất cả" }).click(); await page.waitForTimeout(1200);
  const all = await rows(page).evaluateAll((els) => els.map((e) => e.getAttribute("data-ticket-id")));
  const cand = f.claimed[0]; const ci = all.indexOf(cand);
  await page.locator(`[data-ticket-id="${cand}"]`).click(); await page.waitForTimeout(800);
  await page.getByLabel("Câu trả lời gửi sinh viên").fill("Trả lời thử phím tắt (Tất cả)"); await page.keyboard.press("Control+Enter"); await page.waitForTimeout(3000);
  const b = sel(page);
  R("TC-P4-03-36b", ci >= 0 && !!b && b !== cand && b === all[ci + 1], `bộ lọc «Tất cả» (${all.length} dòng): phiếu đã nhận ${cand.slice(-6)} (vị trí ${ci}) + Ctrl+Enter → URL ${b?.slice(-6)} (kỳ vọng phiếu kế ${all[ci + 1]?.slice(-6)}), DB ${psql(`select status from escalation_tickets where id='${cand}'`)}`);
  await ctx.close();
});

test("33-35", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, f.T, { width: 375, height: 812 });
  await go(page, `/inbox?course=${f.G}`); await page.waitForTimeout(1500);
  const sw0 = await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]);
  const listVisible = await page.locator("[data-part=inbox-list]").isVisible(); const detailVisible = await page.locator("[data-part=inbox-detail]").isVisible();
  await page.getByRole("button", { name: "Đang chờ" }).click().catch(() => undefined); await page.waitForTimeout(500);
  await page.evaluate(() => window.scrollTo(0, 300)); const sc0 = await page.evaluate(() => window.scrollY);
  const rowsN = await rows(page).count();
  await page.evaluate(() => window.scrollTo(0, 300)); const sc0c = await page.evaluate(() => window.scrollY);
  await rows(page).nth(Math.min(rowsN - 1, 6)).click(); await page.waitForTimeout(1500);
  const url1 = page.url(); const dv = await page.locator("[data-part=inbox-detail]").isVisible(); const lv = await page.locator("[data-part=inbox-list]").isVisible();
  const replyBox = await page.locator("[data-part=inbox-reply]").boundingBox().catch(() => null);
  await shot(page, "p403-33-detail");
  const tgt = await page.evaluate(() => Array.from(document.querySelectorAll("button,a,[role=button],textarea")).filter((e) => (e as HTMLElement).offsetParent !== null).map((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return [((e as HTMLElement).innerText || e.getAttribute("aria-label") || e.tagName).slice(0, 20), Math.round(r.width), Math.round(r.height)]; }));
  const small = tgt.filter((x: any) => x[2] < 44 && x[1] > 0 && !/Bỏ qua điều hướng/.test(x[0]));
  await page.getByRole("button", { name: "Hộp thư" }).first().click(); await page.waitForTimeout(1200);
  const url2 = page.url(); const lv2 = await page.locator("[data-part=inbox-list]").isVisible(); const sc2 = await page.evaluate(() => window.scrollY); const filt = await page.getByRole("radio", { name: "Đang chờ" }).first().getAttribute("aria-checked").catch(() => null);
  await page.getByRole("radio", { name: "Đã nhận" }).click(); await page.waitForTimeout(1000); await page.locator(`[data-ticket-id="${f.claimed[1]}"]`).click(); await page.waitForTimeout(1500);
  const comp = await page.getByLabel("Câu trả lời gửi sinh viên").evaluate((e: HTMLElement) => { let n: HTMLElement | null = e; let pos = ""; while (n && n !== document.body) { const cs = getComputedStyle(n); if (cs.position === "sticky" || cs.position === "fixed") { pos = cs.position + " (đáy cách đáy màn hình " + (innerHeight - Math.round(n.getBoundingClientRect().bottom)) + " px)"; break; } n = n.parentElement; } return pos || "static: đáy ô tại y=" + Math.round(e.getBoundingClientRect().bottom); }).catch(() => "không có ô trả lời"); await shot(page, "p403-34-composer");
  const sw1 = await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]);
  R("TC-P4-03-33", /ticket=/.test(url1) && dv && !/ticket=/.test(url2) && lv2 && Math.abs(sc2 - sc0c) < 40 && /sticky|fixed/.test(comp), `375×812: danh sách toàn màn hình (danh sách ${listVisible}, chi tiết ẩn=${!detailVisible}); bấm dòng ${Math.min(rowsN - 1, 6) + 1}/${rowsN} → URL ${url1.split("/inbox")[1]} chi tiết ${dv}, danh sách ẩn=${!lv}; ô trả lời (phiếu đã nhận): vị trí ${comp} (viewport 812); Quay lại → ${url2.split("/inbox")[1]}, bộ lọc «Đang chờ» checked=${filt}, cuộn trước ${sc0c} sau ${sc2}`);
  R("TC-P4-03-34", small.length === 0 && sw0[0] <= sw0[1] && sw1[0] <= sw1[1], `vùng chạm: ${tgt.length} phần tử (bỏ liên kết ẩn «Bỏ qua điều hướng» — hiện khi focus), nhỏ hơn 44 px cao: ${JSON.stringify(small.slice(0, 8))}; scrollWidth ${sw0[0]}/${sw0[1]} (danh sách), ${sw1[0]}/${sw1[1]} (sau quay lại)`);
  await ctx.close();
  const d = await newUser(browser, f.T, { width: 1280, height: 800 }); await go(d.page, `/inbox?course=${f.G}`); await d.page.waitForTimeout(1200);
  const lw = await d.page.locator("[data-part=inbox-list]").evaluate((e: HTMLElement) => Math.round(e.getBoundingClientRect().width)); const dw = await d.page.locator("[data-part=inbox-detail]").evaluate((e: HTMLElement) => Math.round(e.getBoundingClientRect().width)).catch(() => 0);
  await shot(d.page, "p403-35");
  R("TC-P4-03-35", Math.abs(lw - 380) <= 6 && dw > 0, `1280×800: danh sách rộng ${lw} px (kỳ vọng 380), chi tiết ${dw} px cùng màn hình`);
  await d.ctx.close();
});

test("33b ô trả lời dính đáy khi chi tiết dài", async ({ browser }) => {
  const fl = FX("fx_ui5") as { long: string };
  const { ctx, page } = await newUser(browser, f.T, { width: 375, height: 812 });
  await go(page, `/inbox?course=${f.G}`); await page.getByRole("radio", { name: "Đã nhận" }).click(); await page.waitForTimeout(1200); await page.locator(`[data-ticket-id="${fl.long}"]`).click(); await page.waitForTimeout(1500);
  const ta = page.getByLabel("Câu trả lời gửi sinh viên");
  const m = async () => ta.evaluate((e: HTMLElement) => { const r = e.closest("[data-part=inbox-reply]")!.getBoundingClientRect(); let n: HTMLElement | null = e; let sc = ""; while (n && n !== document.body) { const o = getComputedStyle(n).overflowY; if (/(auto|scroll)/.test(o) && n.scrollHeight > n.clientHeight) { sc = n.tagName + "." + n.className.toString().slice(0, 30) + " scrollTop=" + n.scrollTop + "/" + (n.scrollHeight - n.clientHeight); break; } n = n.parentElement; } return { top: Math.round(r.top), bottom: Math.round(r.bottom), ih: innerHeight, sy: scrollY, sc }; });
  const a0 = await m(); await shot(page, "r7-p403-33b-top");
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight)); await page.waitForTimeout(500); const a1 = await m();
  await page.evaluate(() => window.scrollTo(0, 0)); await page.waitForTimeout(300); const a2 = await m();
  const ok = (x: any) => x.bottom <= x.ih && x.top >= 0;
  R("TC-P4-03-33b", ok(a0) && ok(a1) && ok(a2), `phiếu có nội dung dài (CLAIMED), 375×812: ô trả lời ${JSON.stringify(a0)} lúc mở (cuộn 0); cuộn xuống đáy ${JSON.stringify(a1)}; cuộn lên đầu ${JSON.stringify(a2)}; ô nằm trong viewport (0 ≤ top, bottom ≤ ${a0.ih}) ở cả ba vị trí = ${ok(a0) && ok(a1) && ok(a2)}`);
  await ctx.close();
});
