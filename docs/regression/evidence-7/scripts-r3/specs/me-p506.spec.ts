import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, psql, newUser, expect } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p506", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("p506", `INFO ${tc} ${note}`);
const FM = FX("fx_me") as { M: string; M2: string; U: string[] };
const FA = FX("fx_att") as { C: string; S: string[] };
const txt = (p: Page) => p.locator("main").innerText();
const openMe = async (p: Page, c: string) => { await go(p, `/me?course=${c}`); await p.getByText("Kết quả của tôi").first().waitFor({ timeout: 30000 }); await p.waitForTimeout(1500); };

test("07 duyệt các trang của C", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3me03"); const out: string[] = []; const bad: string[] = [];
  for (const u of ["/", "/me", "/chat", "/threads", "/settings", "/calendar"]) { await go(page, `${u}?course=${FM.M}`); await page.waitForTimeout(1200); const t = await page.evaluate(() => document.body.innerText); out.push(`${u}:${t.length}`); const m = t.match(/AT_RISK|WATCH|nguy cơ|rủi ro|cần chú ý|có nguy|bỏ học|canary|CANARY|Insight|risk/gi); if (m) bad.push(`${u}:${[...new Set(m)].join(",")}`); }
  R("TC-P5-06-07", bad.length === 0, `SV C (qc3me03, tuần học thấp) duyệt / , /me, /chat, /threads, /settings, /calendar (độ dài chữ ${out.join(" ")}): từ khoá nhận định rủi ro / canary tìm thấy ở: ${JSON.stringify(bad)}`);
  await ctx.close();
});

test("08/09/10 nội dung /me", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3me01", { width: 1280, height: 800 }); await openMe(page, FM.M);
  const t = await txt(page); const heads = await page.locator("main h1,main h2,main h3").allInnerTexts(); const sel = await page.evaluate(() => Array.from(document.querySelectorAll("[data-part=topbar] select,[data-part=topbar] button")).map((e) => (e as HTMLElement).innerText.trim()).filter(Boolean).slice(0, 8));
  const chart = await page.locator("main svg").first().evaluate((e) => e.getBoundingClientRect().height).catch(() => 0); const chartBox = await page.locator("main [class*=trend]").first().evaluate((e) => e.getBoundingClientRect().height).catch(() => 0);
  const order = ["Chuyên cần", "Điểm cộng đã ghi", "Thời gian học"].map((x) => t.indexOf(x)); await shot(page, "p506-08");
  const ok = /Kết quả của tôi/.test(t) && /Bạn có mặt 7\/9 buổi, vắng 2\./.test(t) && /Buổi 3/.test(t) && /Buổi 6/.test(t) && /Phát biểu 2 lần · \+0,50\./.test(t) && /Tuần này: 45 phút/.test(t) && order[0] >= 0 && order[0] < order[1] && order[1] < order[2] && !/\d\.\d{2}\b(?! phút)/.test(t.replace(/\d{2}\/\d{2}\/\d{4}/g, ""));
  R("TC-P5-06-08", ok, `/me?course=M (qc3me01, 2 lớp): tiêu đề «Kết quả của tôi»; thứ tự khối Chuyên cần(${order[0]}) < Điểm cộng(${order[1]}) < Thời gian học(${order[2]}); câu «${(t.match(/Bạn có mặt[^\n]*/) ?? [""])[0]}»; danh sách buổi vắng: ${(t.match(/Buổi \d+ · [^\n]*\n[^\n]*/g) ?? []).join(" ; ").replace(/\n/g, " ")}; «${(t.match(/Phát biểu \d+ lần[^\n]*/) ?? [""])[0]}»; sự kiện: ${(t.match(/Phát biểu \+[^\n]*\n?[^\n]*/) ?? [""])[0].replace(/\n/g, " ")}; biểu đồ svg cao ${chart.toFixed(0)} px (khung ${chartBox.toFixed(0)}), «${(t.match(/Tuần này[^\n]*/) ?? [""])[0]}»; bộ chọn lớp ở topbar: ${JSON.stringify(sel)}; dấu thập phân dùng phẩy`);
  // 09
  const fake = t.match(/điểm thành phần|giải trình|nếu cuối kỳ|việc sắp tới|mô phỏng|dữ liệu giả|demo|mock|lorem/gi) ?? [];
  R("TC-P5-06-09", fake.length === 0, `tìm điểm thành phần / giải trình điểm / «nếu cuối kỳ được…» / việc sắp tới có điểm / mô phỏng / giả: ${JSON.stringify(fake)}; các tiêu đề thấy: ${JSON.stringify(heads)}`);
  // 10
  const all = await page.evaluate(() => document.body.innerText + "\n" + document.body.innerHTML); const g = all.match(/streak|rank|progress ring|confetti|xếp hạng|huy hiệu|badge|cố lên|tuyệt vời/gi) ?? [];
  const pb = await page.evaluate(() => ({ progress: document.querySelectorAll("progress").length, role: document.querySelectorAll("[role=progressbar]").length, meter: document.querySelectorAll("meter,[role=meter]").length, kpi: Array.from(document.querySelectorAll("main *")).filter((e) => { const s = getComputedStyle(e); return s.display === "grid" && s.gridTemplateColumns.split(" ").length >= 3 && e.children.length >= 3 && Array.from(e.children).every((c) => /^\s*[\d.,]+\s*%?\s*$/.test((c as HTMLElement).innerText.split("\n")[0] ?? "")); }).length }));
  R("TC-P5-06-10", g.length === 0 && pb.progress === 0 && pb.role === 0 && pb.kpi === 0, `grep -ciE 'streak|rank|progress ring|confetti|xếp hạng|huy hiệu|badge|cố lên|tuyệt vời' trong innerText+innerHTML = ${g.length} ${JSON.stringify(g.slice(0, 3))}; <progress>=${pb.progress}, role=progressbar=${pb.role}, meter=${pb.meter}, lưới ≥3 ô số lớn (thẻ KPI)=${pb.kpi}; không so sánh với bạn học: «so với|xếp|lớp trung bình» = ${(t.match(/so với|trung bình lớp|bạn học/gi) ?? []).length}`);
  await ctx.close();
});

test("14/15 rỗng, tắt, tuần thấp", async ({ browser }) => {
  const a = await newUser(browser, "qc3me04"); await openMe(a.page, FM.M); const t4 = await txt(a.page); const has4 = /Thời gian học/.test(t4); const zero = /0 phút/.test(t4);
  psql(`insert into user_settings(user_id,track_study_time) select id,false from users where email='qc3me02@edupilot.local' on conflict (user_id) do update set track_study_time=false`); const rds = (k: string) => k;
  const b = await newUser(browser, "qc3me02"); await openMe(b.page, FM.M); const t2 = await txt(b.page); const off = /Đang tắt/.test(t2);
  psql(`update user_settings set track_study_time=true where user_id=(select id from users where email='qc3me02@edupilot.local')`);
  R("TC-P5-06-14", !has4 && !zero && off, `SV mới (qc3me04, chưa có dữ liệu): khối «Thời gian học» hiển thị=${has4}, chữ «0 phút»=${zero}; SV tắt đo (qc3me02, track_study_time=false, lớp cho tắt): «${(t2.match(/Thời gian học[\s\S]{0,40}/) ?? [""])[0].replace(/\n/g, " | ")}» (có «Đang tắt»=${off}, không biểu đồ)`);
  await a.ctx.close(); await b.ctx.close();
  const c = await newUser(browser, "qc3me03"); await openMe(c.page, FM.M); const tc = await txt(c.page); const weeks = (await c.page.evaluate(() => { const s = document.querySelector("main svg"); if (!s) return []; return Array.from(s.querySelectorAll("rect,circle,path,line")).map((e) => { const cs = getComputedStyle(e); return `${e.tagName}:${cs.fill}:${cs.stroke}`; }); }));
  const colors = [...new Set(weeks)]; const reds = colors.filter((x) => /rgb\((2[0-5]\d|1[89]\d), ?\d{1,2}, ?\d{1,2}\)/.test(x));
  const forbidden = tc.match(/mục tiêu|so với|trung bình|thấp|giảm|nên học|cố gắng/gi) ?? []; await shot(c.page, "p506-15");
  R("TC-P5-06-15", reds.length === 0 && forbidden.length === 0 && colors.length >= 1, `SV C (qc3me03; tuần này 50/45/30 phút < 60): tập màu fill/stroke trong biểu đồ ${colors.length} giá trị: ${JSON.stringify(colors.slice(0, 6))}; màu đỏ=${reds.length}; chữ «mục tiêu|so với|trung bình|thấp|giảm|nên học»=${JSON.stringify(forbidden)}; «${(tc.match(/Tuần này[^\n]*/) ?? [""])[0]}»`);
  await c.ctx.close();
});

test("16 375 px", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3me01", { width: 375, height: 812 }); await openMe(page, FM.M);
  const g = await page.evaluate(() => { const sw = document.documentElement.scrollWidth, iw = window.innerWidth; const small = Array.from(document.querySelectorAll("main button,main a,main input,main [role=button]")).filter((e) => { const b = e.getBoundingClientRect(); return b.width > 0 && b.height > 0 && b.height < 44; }).map((e) => `${(e as HTMLElement).innerText.slice(0, 16) || e.getAttribute("aria-label")}:${Math.round(e.getBoundingClientRect().height)}`); const secs = Array.from(document.querySelectorAll("main section")).map((e) => Math.round(e.getBoundingClientRect().left)); const svg = document.querySelector("main svg"); return { sw, iw, small, secs: [...new Set(secs)], svgW: svg ? Math.round(svg.getBoundingClientRect().width) : null }; });
  const fold = await page.getByRole("button", { name: "Mở" }).count(); if (fold) await page.getByRole("button", { name: "Mở" }).last().click(); await page.waitForTimeout(500);
  const svgW = await page.locator("main svg").first().evaluate((e) => Math.round(e.getBoundingClientRect().width)).catch(() => null); const sw2 = await page.evaluate(() => document.documentElement.scrollWidth); await shot(page, "p506-16");
  R("TC-P5-06-16", g.sw <= g.iw && sw2 <= g.iw && g.secs.length === 1 && g.small.length === 0 && (svgW ?? 0) <= g.iw, `375×812: scrollWidth ${g.sw}/${g.iw} (sau khi mở khối gập ${sw2}); các khối cùng mép trái=${JSON.stringify(g.secs)} (một cột); phần tử bấm cao < 44 px: ${JSON.stringify(g.small)}; biểu đồ rộng ${svgW} ≤ ${g.iw}; nút Mở/Gập=${fold}`);
  await ctx.close();
});

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
  const b = await newUser(browser, "qc3me04"); await openMe(b.page, FM.M); const tb = await txt(b.page); const emptyOk = /Chưa có buổi nào được điểm danh\./.test(tb); const btns = await b.page.locator("main section").filter({ hasText: "Chuyên cần" }).getByRole("button").count();
  R("TC-P5-06-18", sk >= 1 && /Thử lại/.test(et) && /không bị mất/.test(et) && retryOk && /Bạn có mặt/.test(body) && /Mất kết nối|Chưa có mạng/.test(banner) && emptyOk && btns === 0, `tải chậm (3,5 s): khung xương=${sk}; lớp chưa điểm danh: «Chưa có buổi nào được điểm danh.»=${emptyOk}, nút trong khối=${btns}; 500: «${et}», Thử lại → tải lại được; offline sau khi tải: nội dung cũ vẫn còn=${/Bạn có mặt/.test(body)}, dải «${banner}»`);
  R("TC-P5-06-E1", /Thử lại/.test(et) && /không bị mất/.test(et), `mạng đứt/500 khi mở /me: PageState lỗi «${et}» + nút Thử lại; Thử lại → nội dung về`);
  await a.ctx.close(); await b.ctx.close();
  // E2: bị REMOVED trong lúc đang xem
  const c = await newUser(browser, "qc3at129"); await openMe(c.page, FA.C); const before = /Bạn có mặt|Chưa có buổi/.test(await txt(c.page));
  psql(`update enrollments set status='REMOVED',removed_at=now(),previous_status='ACTIVE' where course_id='${FA.C}' and user_id=(select id from users where email='qc3at129@edupilot.local')`);
  await c.page.reload(); await c.page.waitForTimeout(4000); const after = (await c.page.evaluate(() => document.body.innerText)).replace(/\n+/g, " | ").slice(0, 300); await shot(c.page, "p506-e2");
  psql(`update enrollments set status='ACTIVE',removed_at=null,previous_status=null where course_id='${FA.C}' and user_id=(select id from users where email='qc3at129@edupilot.local')`);
  R("TC-P5-06-E2", before && !/Bạn có mặt|Phát biểu \d+ lần|Tuần này/.test(after), `SV đang xem /me của lớp, bị REMOVED trong lúc xem → làm mới: ${after}; còn dữ liệu cũ (Bạn có mặt/Phát biểu/Tuần này)=${/Bạn có mặt|Phát biểu \d+ lần|Tuần này/.test(after)}`);
  await c.ctx.close();
});
