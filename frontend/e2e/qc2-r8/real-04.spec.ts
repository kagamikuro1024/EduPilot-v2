import { test, type Page, type BrowserContext } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { cid, go, login, shot } from "./lib";
// QC2 sprint 8 US-P6-04 (giao diện công thức) trên ep-r thật. Ghi kết quả: /tmp/qc2-s8r/ui04.tsv (TC \t P|F \t ghi chú)
test.setTimeout(240_000);
const OUT = "/tmp/qc2-s8r/ui04.tsv";
const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 400)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note}`); };
const mk = async (browser: any, user: string, w = 1440, h = 900) => { const c: BrowserContext = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN" }); const p: Page = await c.newPage(); await login(p, user); return p; };
const scheme = async (p: Page, code: string) => { await go(p, `/gradebook/scheme?course=${cid(code)}`); await p.waitForTimeout(2500); };
const mainText = (p: Page) => p.locator("main").innerText();
const confirmBtn = (p: Page) => p.getByRole("button", { name: /^Xác nhận công thức$/ }).first();

test("[04-01..07] bố cục, trích dẫn, 1023/1024, luật chữ, làm tròn", async ({ browser }) => {
  const p = await mk(browser, "teacher"); await scheme(p, "QD8B1");
  // 01 một Panel, không lồng, có cột trích dẫn
  const pan = await p.evaluate(() => { const ps = [...document.querySelectorAll("[data-ep-panel]")]; return { n: ps.length, nested: ps.filter((e) => e.parentElement?.closest("[data-ep-panel]")).length, aside: !!document.querySelector("[data-part=citation-pane]") }; });
  res("TC-P6-04-01", pan.nested === 0 && pan.aside && pan.n <= 2, `[data-ep-panel] ${pan.n}, lồng nhau ${pan.nested}, cột trích dẫn ${pan.aside}`); await shot(p, "04-01");
  // 02 click các khối → trang trích dẫn
  const pageOf = async (re: RegExp) => { await p.getByRole("button", { name: re }).first().click(); await p.waitForTimeout(700); return (await p.locator("[data-part=citation-page]").first().innerText().catch(() => "?")).trim(); };
  const g1 = await pageOf(/^Điểm quá trình gồm/), g2 = await pageOf(/^Mỗi lần phát biểu được/), g3 = await pageOf(/^Điểm chữ xác định/);
  const hl = await p.evaluate(() => !!document.querySelector("[data-part=citation-pane] mark, [data-part=citation-pane] [data-cited], [data-part=citation-pane] [aria-current]"));
  res("TC-P6-04-02", /1/.test(g1) && /2/.test(g2) && /2/.test(g3) && hl, `Giữa kỳ → «${g1}», Điểm cộng → «${g2}», Thang chữ → «${g3}»; đoạn trích được nhấn mạnh ${hl}`);
  // 03 1023 / 1024
  await p.setViewportSize({ width: 1024, height: 900 }); await p.waitForTimeout(800); const a1024 = await p.evaluate(() => [...document.querySelectorAll("[data-part=citation-pane]")].some((e) => !e.closest("dialog") && (e as HTMLElement).offsetParent !== null));
  await p.setViewportSize({ width: 1023, height: 900 }); await p.waitForTimeout(800); const a1023 = await p.evaluate(() => [...document.querySelectorAll("[data-part=citation-pane]")].some((e) => !e.closest("dialog") && (e as HTMLElement).offsetParent !== null));
  const D = "dialog[open]:has([data-part=citation-pane])";
  const dlg0 = await p.locator(D).count(); await p.keyboard.press("Escape"); await p.waitForTimeout(600); const dlgAfter = await p.locator(D).count();
  const chip = p.getByRole("button", { name: /^Trang \d$/ }).first(); await chip.focus(); await p.keyboard.press("Enter"); await p.waitForTimeout(900); const dlg = await p.locator(D).count(); await p.keyboard.press("Escape"); await p.waitForTimeout(600);
  const foc = await p.evaluate(() => { const a = document.activeElement as HTMLElement | null; return (a?.tagName === "BUTTON" ? (a.textContent ?? "") : "<" + a?.tagName + ">").slice(0, 30); }); const dlgEnd = await p.locator(D).count();
  res("TC-P6-04-03", a1024 && !a1023 && dlg0 >= 1 && dlgAfter === 0 && dlg >= 1 && dlgEnd === 0 && /Trang \d/.test(foc), `1024 px cột cố định ${a1024}; 1023 px cột ${a1023}, mục đang chọn → Drawer ${dlg0}; Esc đóng ${dlgAfter === 0}; chọn lại bằng nút Trang → Drawer ${dlg}, Esc đóng ${dlgEnd === 0}, focus về «${foc}»`);
  await p.setViewportSize({ width: 1440, height: 900 }); await p.waitForTimeout(800);
  // 04 luật chữ: không câu mô tả dưới tiêu đề, không icon Info
  const info = await p.locator('[data-icon="info"], svg[aria-label*="Thông tin"], [data-icon="circle-help"]').count();
  const t = await mainText(p); const lines = t.split("\n").map((x) => x.trim()).filter(Boolean);
  const h1i = lines.indexOf("Công thức điểm"); const under = lines.slice(h1i + 1, h1i + 3).join(" | ");
  res("TC-P6-04-04", info === 0, `icon Info/CircleHelp ${info}; hai dòng sau tiêu đề: «${under}»`);
  // 06 làm tròn trống, nút chính disabled kèm tooltip trên nút
  const btn = confirmBtn(p); const ad = await btn.getAttribute("aria-disabled"); const dis = await btn.isDisabled().catch(() => false);
  await btn.hover(); await p.waitForTimeout(700); let tip = await p.locator("[role=tooltip]").allInnerTexts(); await p.mouse.move(5, 5); await btn.focus(); await p.waitForTimeout(700); const tipF = await p.locator("[role=tooltip]").allInnerTexts();
  const modal = await p.locator("dialog[open]").count();
  res("TC-P6-04-06", (ad === "true" || dis) && /Còn 1 mục chưa rõ: làm tròn/.test((tip.concat(tipF, [(await btn.getAttribute("title")) ?? ""])).join(" ") || t) && modal === 0 && /Bước làm tròn/.test(t), `nút aria-disabled=${ad}; tooltip hover «${tip.join("/")}», focus «${tipF.join("/")}»; modal ${modal}; khối Làm tròn có ô điền`);
  res("TC-P6-04-44", tipF.join(" ").includes("Còn 1 mục chưa rõ"), `tooltip hiện khi focus bàn phím (không chỉ hover): «${tipF.join("/")}»`);
  await shot(p, "04-06");
  // 07 điền làm tròn 0,1
  await p.locator("main select").first().selectOption({ label: "Làm tròn đến 0,1" });
  await p.keyboard.press("Tab"); await p.waitForTimeout(2500);
  const t2 = await mainText(p); const ad2 = await confirmBtn(p).getAttribute("aria-disabled"); const dis2 = await confirmBtn(p).isDisabled().catch(() => false);
  res("TC-P6-04-07", !/Còn 1 mục chưa rõ/.test(t2) && ad2 !== "true" && !dis2, `sau điền «Làm tròn đến 0,1»: còn «chưa rõ» ${/Còn \d+ mục chưa rõ/.test(t2)}; nút aria-disabled=${ad2} disabled=${dis2}`); await shot(p, "04-07");
  // 17 xem thử lớp 2 chưa có điểm
  res("TC-P6-04-17", /Chưa đủ điểm để xem thử/.test(t2), `khối Xem thử: ${(t2.match(/Chưa đủ điểm để xem thử[^\n]*/) || ["(không thấy chữ)"])[0]}`);
  // 19 hộp thoại xác nhận (không bấm xác nhận cuối): nêu số, Esc huỷ, click ngoài không đóng
  let log19 = ""; await confirmBtn(p).click(); await p.waitForTimeout(1500);
  const dlgT = await p.locator("dialog[open]").first().innerText().catch(() => "");
  const DL = "dialog[open]";
  await p.mouse.click(5, 5); await p.waitForTimeout(500); const still = await p.locator(DL).count();
  await p.keyboard.press("Escape"); await p.waitForTimeout(600); let gone = await p.locator(DL).count();
  if (gone > 0) { await p.locator(DL).first().focus().catch(() => undefined); await p.keyboard.press("Escape"); await p.waitForTimeout(600); const g2 = await p.locator(DL).count(); log19 = `Esc sau click ngoài không đóng (còn ${gone}); Esc khi focus trong hộp thoại: còn ${g2}`; gone = g2; }
  res("TC-P6-04-19", /điểm học phần của 0 sinh viên/.test(dlgT) && /0 điểm đã nhập bị bỏ/.test(dlgT) && /Xác nhận công thức/.test(dlgT) && still >= 1 && gone === 0, `hộp thoại: «${dlgT.replace(/\s+/g, " ").slice(0, 260)}»; click ngoài vẫn mở ${still >= 1}; Esc đóng ${gone === 0} ${log19}`);
});

test("[04-20] lớp 1 CONFIRMED có điểm: số tác động", async ({ browser }) => {
  // lớp QD8R6W CONFIRMED v1 có điểm; tạo DRAFT đổi bước làm tròn bằng API trước khi chạy (xem tập lệnh s8r-04a.py)
  const p = await mk(browser, "teacher"); await scheme(p, "QD8R6W");
  const t = await mainText(p); await shot(p, "04-20-pre");
  const b = confirmBtn(p); const ok = (await b.count()) > 0 && !(await b.isDisabled().catch(() => true));
  if (!ok) { res("TC-P6-04-20", false, `không có nút Xác nhận công thức bật được; nội dung: ${t.slice(0, 160)}`); return; }
  await b.click(); await p.waitForTimeout(2500); const d = await p.locator("dialog[open], [role=dialog]").first().innerText().catch(() => "");
  res("TC-P6-04-20", /Áp cho \d+ sinh viên/.test(d), `hộp thoại: «${d.replace(/\s+/g, " ").slice(0, 300)}»`); await shot(p, "04-20"); await p.keyboard.press("Escape");
});

test("[04-23,45,46] phân quyền giao diện: TA chỉ đọc, SV 403", async ({ browser }) => {
  const ta = await mk(browser, "ta"); await scheme(ta, "QD8B1");
  const tt = await mainText(ta); const cf = await ta.getByRole("button", { name: /Xác nhận công thức/ }).count(), up = await ta.getByRole("button", { name: /Tải quy chế/ }).count(), inp = await ta.locator("main input, main select").count();
  res("TC-P6-04-45", cf === 0 && up === 0 && inp === 0 && /Giữa kỳ/.test(tt), `TA: Xác nhận ${cf}, Tải quy chế ${up}, ô điền ${inp}; thấy công thức/trích dẫn ${/Giữa kỳ/.test(tt)}`);
  const sv = await mk(browser, "sv.gioi"); await go(sv, `/gradebook/scheme?course=${cid("QD8R6W")}`); await sv.waitForTimeout(2500);
  const html = await sv.content(); const txt = await sv.locator("body").innerText();
  res("TC-P6-04-46", !/Thi cuối kỳ/.test(html) && /403|không có quyền/i.test(txt), `SV: trang ${(txt.match(/[^\n]*(403|không có quyền)[^\n]*/i) || ["?"])[0].slice(0, 80)}; DOM không chứa «Thi cuối kỳ» ${!/Thi cuối kỳ/.test(html)}`); await shot(sv, "04-46");
});

test("[04-38,39] trạng thái rỗng, skeleton, lỗi 500, 375 px", async ({ browser }) => {
  const g = await mk(browser, "gv.s8f"); await go(g, `/gradebook/scheme?course=${cid("QD8F3")}`); await g.waitForTimeout(2500);
  const t = await mainText(g); const up = await g.getByRole("button", { name: /Tải quy chế/ }).count();
  const state1 = /Lớp chưa có quy chế/.test(t) && up >= 1;
  // skeleton: chậm 3 s
  await g.route("**/grade-scheme", async (r) => { await new Promise((x) => setTimeout(x, 3000)); await r.continue().catch(() => undefined); });
  await g.reload(); await g.waitForTimeout(900); const sk = await g.locator('[data-skeleton], [aria-busy=true], [class*=keleton]').count();
  await g.waitForTimeout(3500); await g.unroute("**/grade-scheme");
  // lỗi 500 rồi Thử lại
  let fail = true; await g.route("**/grade-scheme", async (r) => { if (fail && r.request().method() === "GET") await r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi" }) }); else await r.continue(); });
  await g.reload(); await g.waitForTimeout(3500); const et = await g.locator("main").innerText(); const retry = g.getByRole("button", { name: /Thử lại/ }); const hasRetry = (await retry.count()) > 0;
  fail = false; if (hasRetry) { await retry.first().click(); await g.waitForTimeout(3000); }
  const t3 = await g.locator("main").innerText();
  res("TC-P6-04-38", state1 && sk > 0 && hasRetry && /Lớp chưa có quy chế/.test(t3), `rỗng «Lớp chưa có quy chế» + Tải quy chế ${state1}; skeleton khi chậm ${sk}; lỗi 500 hiện Thử lại ${hasRetry}; bấm Thử lại tải lại được ${/Lớp chưa có quy chế/.test(t3)}`);
  const m = await mk(browser, "teacher", 375, 812); await scheme(m, "QD8B1");
  const ox = await m.evaluate(() => document.documentElement.scrollWidth - window.innerWidth); const bb = await confirmBtn(m).boundingBox();
  await m.getByRole("button", { name: /^Giữa kỳ ·/ }).first().click(); await m.waitForTimeout(900); const dr = await m.locator("dialog[open], [role=dialog]").count();
  res("TC-P6-04-39", ox <= 0 && (bb?.height ?? 0) >= 44 && dr >= 1, `375 px: cuộn ngang ${ox}; chiều cao nút chính ${bb?.height}; trích dẫn mở bằng Drawer ${dr >= 1}`); await shot(m, "04-39");
});

test("[04-35] offline: sửa ô, rời ô, có mạng lại", async ({ browser }) => {
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage(); await login(p, "teacher"); await scheme(p, "QD8B1");
  const ev = p.locator("[data-part=edit-value]").first(); await ev.click(); await p.waitForTimeout(500);
  const inp = p.locator("main input:not([type=file])").nth(1); const orig = await inp.inputValue();
  await c.setOffline(true); await inp.fill("24"); await inp.press("Tab"); await p.waitForTimeout(2500);
  const off = (await p.locator("[data-part=save-state]").allInnerTexts()).join("/") || (await p.locator("main").innerText()).match(/[^\n]*(Chưa lưu|Mất mạng|offline)[^\n]*/i)?.[0] || "(không thấy nhãn)";
  const keep = await p.evaluate(() => [...document.querySelectorAll("main input")].some((i) => (i as HTMLInputElement).value === "24") || /24%/.test(document.querySelector("main")?.textContent ?? ""));
  await c.setOffline(false); await p.waitForTimeout(9000);
  const on = (await p.locator("[data-part=save-state]").allInnerTexts()).join("/");
  const db = execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", `select weight from grade_components gc join grade_schemes s on s.id=gc.scheme_id join courses c on c.id=s.course_id where c.class_code='QD8B1' and s.status='DRAFT' and gc.key='midterm'`], { encoding: "utf8" }).trim();
  res("TC-P6-04-35", /Chưa lưu|offline|Mất mạng/i.test(off) && keep && /^24/.test(db), `offline: nhãn «${off}», giữ chữ ${keep}; có mạng lại nhãn «${on}»; DB weight midterm=${db} (gốc ${orig})`);
  // khôi phục 25 để các ca xác nhận phía sau có tổng 100
  await ev.click().catch(() => undefined); const i2 = p.locator("main input:not([type=file])").nth(1); await i2.fill("25"); await i2.press("Tab"); await p.waitForTimeout(3500);
  const db2 = execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", `select weight from grade_components gc join grade_schemes s on s.id=gc.scheme_id join courses c on c.id=s.course_id where c.class_code='QD8B1' and s.status='DRAFT' and gc.key='midterm'`], { encoding: "utf8" }).trim(); console.log("restored", db2);
});

test("[04-22,41] xác nhận lớp 2: không toast, đổi tại chỗ, Hôm nay + banner biến mất", async ({ browser }) => {
  const p = await mk(browser, "teacher");
  await go(p, "/"); await p.waitForTimeout(3000); const today0 = await p.locator("main").innerText(); const has0 = /công thức/i.test(today0);
  await scheme(p, "QD8B1"); const g0 = await (async () => { await go(p, `/gradebook?course=${cid("QD8B1")}`); await p.waitForTimeout(2500); return (await mainText(p)); })(); const ban0 = /chưa xác nhận|Thiết lập công thức|công thức/i.test(g0);
  await scheme(p, "QD8B1"); await confirmBtn(p).click(); await p.waitForTimeout(1500);
  await p.locator("dialog[open]").getByRole("button", { name: /^Xác nhận công thức$/ }).click(); await p.waitForTimeout(2500);
  const toast = await p.locator("[role=status], [data-part=toast], [class*=oast]").allInnerTexts(); const tt = await mainText(p);
  const placed = (tt.match(/Đã xác nhận[^\n]*/) || [""])[0];
  await go(p, "/"); await p.waitForTimeout(3500); const today1 = await p.locator("main").innerText();
  await go(p, `/gradebook?course=${cid("QD8B1")}`); await p.waitForTimeout(2500); const g1 = await mainText(p);
  res("TC-P6-04-41", !toast.some((x) => /Thành công/i.test(x)) && /Đã xác nhận/.test(tt), `sau xác nhận: toast «${toast.join("/")}»; chữ tại chỗ «${placed}»`);
  res("TC-P6-04-22", has0 !== true || !/công thức điểm/i.test(today1), `Hôm nay trước: ${has0 ? "có" : "không"} mục công thức; sau: ${/công thức/i.test(today1) ? "còn «" + (today1.match(/[^\n]*công thức[^\n]*/i) || [""])[0] + "»" : "hết"}; /gradebook trước có chữ công thức/banner ${ban0}, sau «${(g1.match(/[^\n]*(chưa xác nhận|Thiết lập công thức)[^\n]*/i) || ["(không còn)"])[0]}»`); await shot(p, "04-22");
});

test("[04-03r,04-19r] lặp lại trên bản nháp có trích dẫn (A2) và hộp thoại xác nhận (R6W)", async ({ browser }) => {
  const p = await mk(browser, "teacher"); await scheme(p, "QD8A2");
  const D = "dialog[open]:has([data-part=citation-pane])";
  const pane = () => p.evaluate(() => [...document.querySelectorAll("[data-part=citation-pane]")].some((e) => !e.closest("dialog") && (e as HTMLElement).offsetParent !== null));
  await p.getByRole("button", { name: /^Điểm quá trình gồm/ }).first().click(); await p.waitForTimeout(500);
  await p.setViewportSize({ width: 1024, height: 900 }); await p.waitForTimeout(900); const a1024 = await pane();
  await p.setViewportSize({ width: 1023, height: 900 }); await p.waitForTimeout(900); const a1023 = await pane(); const d0 = await p.locator(D).count();
  await p.keyboard.press("Escape"); await p.waitForTimeout(600); const dEsc = await p.locator(D).count();
  const chip = p.getByRole("button", { name: /^Trang \d$/ }).first(); await chip.focus(); await p.keyboard.press("Enter"); await p.waitForTimeout(900); const d1 = await p.locator(D).count();
  await p.keyboard.press("Escape"); await p.waitForTimeout(600); const d2 = await p.locator(D).count();
  const foc = await p.evaluate(() => { const a = document.activeElement as HTMLElement | null; return a?.tagName === "BUTTON" ? (a.textContent ?? "").slice(0, 20) : "<" + a?.tagName + ">"; });
  res("TC-P6-04-03", a1024 && !a1023 && d0 >= 1 && dEsc === 0 && d1 >= 1 && d2 === 0 && /Trang \d/.test(foc), `1024 px: cột cố định ${a1024}; 1023 px: cột cố định ${a1023}, mục đang chọn mở Drawer ${d0}, Esc đóng ${dEsc === 0}; Enter trên nút Trang mở Drawer ${d1}, Esc đóng ${d2 === 0}, focus về «${foc}»`);
  const r = await mk(browser, "teacher"); await scheme(r, "QD8R6W");
  await r.getByRole("button", { name: /^Xác nhận công thức$/ }).first().click(); await r.waitForTimeout(1500);
  const t = await r.locator("dialog[open]").first().innerText().catch(() => "");
  await r.mouse.click(5, 5); await r.waitForTimeout(500); const still = await r.locator("dialog[open]").count();
  await r.keyboard.press("Escape"); await r.waitForTimeout(600); const gone = await r.locator("dialog[open]").count();
  res("TC-P6-04-19", still === 1 && gone === 0 && /sinh viên/.test(t) && /bị bỏ/.test(t), `hộp thoại nêu số: «${t.replace(/\s+/g, " ").slice(0, 200)}»; click ngoài vẫn mở ${still === 1}; Esc đóng ${gone === 0}; (lớp 2 chưa có điểm: 04-19 trên QD8B1 đã đo «Áp cho 3 sinh viên · … 0 sinh viên sẽ đổi · 0 điểm đã nhập bị bỏ»)`);
});
