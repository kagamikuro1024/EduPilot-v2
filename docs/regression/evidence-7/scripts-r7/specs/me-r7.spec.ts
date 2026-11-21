import { test } from "@playwright/test";
import { go, FX, shot, log, newUser } from "./lib";
// TC-P5-06-18 chạy lại; chữ rỗng theo US-P5-06 AC7 đã vá P-11-82 (sprint/11-p10-eval 0081082e): «Chưa có buổi nào được điểm danh» không dấu chấm
const R = (tc: string, ok: boolean, note: string) => log("r7-me", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const FM = FX("fx_me") as { M: string };
const txt = (p: any) => p.locator("main").innerText();
const openMe = async (p: any, c: string) => { await go(p, `/me?course=${c}`); await p.getByText("Kết quả của tôi").first().waitFor({ timeout: 30000 }); await p.waitForTimeout(1500); };

test("18/E1/E2 trạng thái", async ({ browser }) => {
  const a = await newUser(browser, "qc3me01"); const p = a.page;
  await p.route(/\/me\/(attendance|participation|study-time)/, async (r) => { await new Promise((x) => setTimeout(x, 3500)); return r.continue(); });
  await p.goto(`/me?course=${FM.M}`); await p.waitForTimeout(1500); const sk = await p.evaluate(() => document.querySelectorAll("[class*=skeleton],[data-skeleton],[aria-busy=true]").length); await p.getByText("Bạn có mặt").waitFor({ timeout: 30000 }); await p.unrouteAll();
  // 500
  let n = 0; await p.route(/\/me\/(attendance|participation)/, (r) => (++n <= 2 ? r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }) : r.continue()));
  await p.reload(); await p.getByRole("button", { name: "Thử lại" }).waitFor({ timeout: 30000 }); const et = (await txt(p)).replace(/\n+/g, " | ").slice(0, 220); await p.unrouteAll(); await p.getByRole("button", { name: "Thử lại" }).click(); await p.getByText("Bạn có mặt").waitFor({ timeout: 20000 }); const retryOk = true;
  // offline sau khi đã tải
  await a.ctx.setOffline(true); await p.waitForTimeout(1500); await p.evaluate(() => window.dispatchEvent(new Event("offline"))); const body = await txt(p); const banner = (await p.locator("[data-part=offline-banner]").innerText().catch(() => "")).trim();
  await a.ctx.setOffline(false);
  // rỗng
  const b = await newUser(browser, "qc3me04"); await openMe(b.page, FM.M); const tb = await txt(b.page); const emptyOk = /Chuyên cần[\s\S]{0,40}Chưa có buổi nào được điểm danh(?!\.)/.test(tb) && !/điểm danh\./.test(tb); await shot(b.page, "r7-me-empty-18"); const btns = await b.page.locator("main section").filter({ hasText: "Chuyên cần" }).getByRole("button").count();
  R("TC-P5-06-18", sk >= 1 && /Thử lại/.test(et) && /không bị mất/.test(et) && retryOk && /Bạn có mặt/.test(body) && /Mất kết nối|Chưa có mạng/.test(banner) && emptyOk && btns === 0, `tải chậm (3,5 s): khung xương=${sk}; lớp chưa điểm danh: khối Chuyên cần «Chưa có buổi nào được điểm danh» (không dấu chấm, P-11-82)=${emptyOk}, main: «${tb.replace(/\n+/g, " | ").slice(0, 260)}», nút trong khối=${btns}; 500: «${et}», Thử lại → tải lại được; offline sau khi tải: nội dung cũ vẫn còn=${/Bạn có mặt/.test(body)}, dải «${banner}»`);
  await a.ctx.close(); await b.ctx.close();
});
