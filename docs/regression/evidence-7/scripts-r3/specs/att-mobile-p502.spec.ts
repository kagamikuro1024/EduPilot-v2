import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, psql, newUser, expect } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p502", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const F = FX("fx_att") as { C: string; C0: string; CE: string; S: string[] };
const ses = (C: string) => psql(`insert into class_sessions(course_id,session_no,starts_at,ends_at,room) select '${C}',coalesce(max(session_no),0)+1,now()-interval '30 minutes'+random()*interval '1 second',now()+interval '60 minutes','P.QC' from class_sessions where course_id='${C}' returning id`).split("\n")[0];
const arows = (p: Page) => p.locator("[data-part=att-row]");
const open = async (p: Page, S: string) => { await go(p, `/attendance?course=${F.C}&session=${S}`); await arows(p).first().waitFor({ timeout: 30000 }); };
async function mark(p: Page, i: number, label: string) { await arows(p).nth(i).getByRole("button").first().click(); await p.getByRole("radiogroup").getByRole("radio", { name: label, exact: true }).click(); }

test("16 375 px", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t", { width: 375, height: 812 }); const S = ses(F.C); await open(page, S);
  const geo = await page.evaluate(() => {
    const rows = Array.from(document.querySelectorAll("[data-part=att-row]")) as HTMLElement[]; const r0 = rows[0]; const cs = (e: Element) => getComputedStyle(e);
    // hàng ngăn đường kẻ: tìm phần tử cha trực tiếp của hàng có border-bottom 1px
    let el: HTMLElement | null = r0; let border = ""; let shadow = ""; let radius = ""; for (let i = 0; i < 4 && el; i++) { const s = cs(el); if (parseFloat(s.borderBottomWidth) > 0 || parseFloat(s.borderTopWidth) > 0) { border = `${s.borderBottomWidth}/${s.borderTopWidth} ${s.borderBottomStyle}`; shadow = s.boxShadow; radius = s.borderRadius; break; } el = el.parentElement; }
    return { n: rows.length, txt: r0.innerText.replace(/\n+/g, " | "), status0: (r0.querySelector("[data-part=att-status]") as HTMLElement)?.innerText, border, shadow, radius, sw: document.documentElement.scrollWidth, iw: window.innerWidth };
  });
  await arows(page).nth(3).getByRole("button").first().click();
  const marks = await page.getByRole("radiogroup").getByRole("radio").evaluateAll((els) => els.map((e) => ({ t: (e as HTMLElement).innerText, h: Math.round(e.getBoundingClientRect().height), w: Math.round(e.getBoundingClientRect().width) })));
  await page.getByRole("radiogroup").getByRole("radio", { name: "Vắng", exact: true }).click(); // 2 chạm
  const st3 = await arows(page).nth(3).locator("[data-part=att-status]").innerText(); const others = await arows(page).evaluateAll((els) => els.filter((_, i) => i !== 3).every((e) => (e.querySelector("[data-part=att-status]") as HTMLElement).innerText === "Có mặt"));
  const dlg = await page.getByRole("dialog").count(); const abs3 = await arows(page).nth(3).locator("[data-part=att-status]").evaluate((e) => getComputedStyle(e).color); const pres = await arows(page).nth(0).locator("[data-part=att-status]").evaluate((e) => getComputedStyle(e).color);
  await shot(page, "p502-16");
  await page.waitForTimeout(3000);
  // zoom 200%: viewport 188 px ≈ 375/2
  await page.setViewportSize({ width: 188, height: 406 }); await page.waitForTimeout(800);
  await arows(page).nth(5).getByRole("button").first().click(); const zm = await page.getByRole("radiogroup").getByRole("radio").evaluateAll((els) => els.map((e) => ({ t: (e as HTMLElement).innerText, vis: e.getBoundingClientRect().width > 0 && e.getBoundingClientRect().right <= window.innerWidth + 1 })));
  await page.getByRole("radiogroup").getByRole("radio", { name: "Muộn", exact: true }).click(); const st5 = await arows(page).nth(5).locator("[data-part=att-status]").innerText(); const swz = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1);
  await shot(page, "p502-16-zoom");
  const labelsOk = marks.map((m) => m.t).join("|") === "Có mặt|Muộn|Vắng phép|Vắng"; const hOk = marks.every((m) => m.h >= 44);
  R("TC-P5-02-16", geo.n === 30 && geo.sw <= geo.iw && labelsOk && hOk && st3 === "Vắng" && others && dlg === 0 && st5 === "Muộn" && swz && zm.every((m) => m.vis), `375×812: ${geo.n} hàng; hàng đầu «${geo.txt}»; trạng thái mặc định «${geo.status0}»; đường kẻ ${geo.border || "(không thấy border ở 4 cấp cha)"}, bóng ${geo.shadow}, bo góc ${geo.radius}; bấm hàng mở 4 lựa chọn ${JSON.stringify(marks)} (cao ≥44 = ${hOk}); đổi 1 người = 2 chạm → «${st3}», 29 người còn lại «Có mặt»=${others}; hộp thoại=${dlg}; scrollWidth ${geo.sw} ≤ innerWidth ${geo.iw}; màu chữ Vắng ${abs3} vs Có mặt ${pres}; zoom 200 % (viewport 188 px): 4 lựa chọn đều trong khung=${zm.every((m) => m.vis)}, đánh Muộn → «${st5}», không tràn ngang=${swz}`);
  await ctx.close();
});

test("29 đo 60 s trên điện thoại (kịch bản tay → tự động hoá có bấm giờ)", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t", { width: 375, height: 812 }); const S = ses(F.C); const reqs: any[] = []; page.on("request", (r) => { if (r.method() === "PUT" && /\/attendance$/.test(r.url())) reqs.push(JSON.parse(r.postData() ?? "{}")); });
  await open(page, S); await page.waitForTimeout(1000);
  const t0 = Date.now(); for (const i of [2, 5, 8]) await mark(page, i, "Vắng"); await mark(page, 10, "Muộn");
  await arows(page).nth(12).getByRole("button", { name: /^Ghi phát biểu/ }).click();
  await page.getByRole("button", { name: /^Lưu điểm danh/ }).last().click(); await page.getByText(/Đã lưu điểm danh · 30 sinh viên · vắng 3/).waitFor({ timeout: 20000 }); const dt = (Date.now() - t0) / 1000;
  const dlg = await page.getByRole("dialog").count(); const comp = reqs.filter((b) => b.complete); const last = reqs[reqs.length - 1];
  const abs = psql(`select count(*) from attendance_records where session_id='${S}' and status='ABSENT'`), late = psql(`select count(*) from attendance_records where session_id='${S}' and status='LATE'`), sp = psql(`select count(*) from participation_events where session_id='${S}'`);
  R("TC-P5-02-29", dt <= 60 && dlg === 0 && comp.length === 1 && last.complete && last.records.length === 30 && abs === "3" && late === "1" && sp === "1", `375×812, lớp 30 SV: 3 Vắng + 1 Muộn + 1 phát biểu + Lưu điểm danh trong ${dt.toFixed(1)} s (≤ 60 s; bấm bằng Playwright, không phải thiết bị thật: ngoài tầm của máy chủ chạy); hộp thoại=${dlg}; PUT tự lưu giữa chừng=${reqs.length - comp.length}, PUT complete:true=${comp.length} (cuối cùng, ${last?.records?.length} bản ghi = cả lưới); DB: ABSENT=${abs}, LATE=${late}, phát biểu=${sp}`);
  await ctx.close();
});
