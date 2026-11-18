import { test, type Page, type BrowserContext } from "@playwright/test";
import { login, go, FX, shot, log, psql, newUser, expect } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p502", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("p502", `INFO ${tc} ${note}`);
const F = FX("fx_att") as { C: string; C0: string; CE: string; S: string[]; teacher: string; ta: string };
const ses = (C: string, startMin = -30, durMin = 90) =>
  psql(`insert into class_sessions(course_id,session_no,starts_at,ends_at,room) select '${C}',coalesce(max(session_no),0)+1,now()+interval '${startMin} minutes'+random()*interval '1 second',now()+interval '${startMin + durMin} minutes','P.QC' from class_sessions where course_id='${F.C}' returning id`.replace("where course_id='" + F.C + "'", `where course_id='${C}'`)).split("\n")[0];
const rows = (p: Page) => p.locator("[data-part=student-row]");
const rad = (p: Page, i: number, st: string) => rows(p).nth(i).locator(`input[data-status=${st}]`);
const stateTxt = (p: Page) => p.locator("[data-part=att-save-state]").innerText();
const openAtt = async (p: Page, C: string, S?: string) => { await go(p, `/attendance?course=${C}${S ? `&session=${S}` : ""}`); await p.locator("[data-part=student-row]").first().waitFor({ timeout: 30000 }); };
const puts = (p: Page) => { const a: Array<{ body: any; t: number }> = []; p.on("request", (r) => { if (r.method() === "PUT" && /\/sessions\/[^/]+\/attendance$/.test(r.url())) a.push({ body: JSON.parse(r.postData() ?? "{}"), t: Date.now() }); }); return a; };
const count = (S: string, st: string) => psql(`select count(*) from attendance_records where session_id='${S}' and status='${st}'`);
const nameOf = async (p: Page, i: number) => (await rows(p).nth(i).locator(".ep-item-title").first().innerText()).trim();

test("18 lưới ≥1100px", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const S = ses(F.C); await openAtt(page, F.C, S);
  const info = await page.evaluate(() => {
    const th = Array.from(document.querySelectorAll("[data-part=attendance-grid] thead th")).map((e) => (e as HTMLElement).innerText.replace(/\s+/g, " ").trim());
    const t = document.querySelector("[data-part=attendance-grid] thead th") as HTMLElement; const td = document.querySelector("[data-part=attendance-grid] tbody td") as HTMLElement;
    const sc = Array.from(document.querySelectorAll("[data-part=attendance-grid] *")).find((e) => { const s = getComputedStyle(e); return /(auto|scroll)/.test(s.overflowY) && e.scrollHeight > e.clientHeight + 5; }) as HTMLElement | undefined;
    return { th, thPos: getComputedStyle(t).position, tdPos: getComputedStyle(td).position, hasScroll: !!sc, scrollH: sc?.scrollHeight, clientH: sc?.clientHeight, radios: document.querySelectorAll("tbody tr:first-child input[type=radio]").length };
  });
  const thTop0 = await page.locator("[data-part=attendance-grid] thead th").first().evaluate((e) => e.getBoundingClientRect().top);
  await page.evaluate(() => { const sc = Array.from(document.querySelectorAll("[data-part=attendance-grid] *")).find((e) => { const s = getComputedStyle(e); return /(auto|scroll)/.test(s.overflowY) && e.scrollHeight > e.clientHeight + 5; }) as HTMLElement | undefined; if (sc) sc.scrollTop = 400; else window.scrollTo(0, 400); });
  await page.waitForTimeout(300);
  const thTop1 = await page.locator("[data-part=attendance-grid] thead th").first().evaluate((e) => e.getBoundingClientRect().top);
  const nameVisible = await rows(page).first().locator("td").first().isVisible();
  await page.evaluate(() => { const sc = Array.from(document.querySelectorAll("[data-part=attendance-grid] *")).find((e) => { const s = getComputedStyle(e); return /(auto|scroll)/.test(s.overflowY) && e.scrollHeight > e.clientHeight + 5; }) as HTMLElement | undefined; if (sc) sc.scrollTop = 0; else window.scrollTo(0, 0); });
  const grid = page.getByLabel(/Bảng điểm danh/); await grid.focus();
  const ring = await grid.evaluate((e) => { const s = getComputedStyle(e); return { fv: e.matches(":focus-visible"), outline: s.outlineStyle + " " + s.outlineWidth, shadow: s.boxShadow }; });
  await page.keyboard.press("4"); const a1 = await rad(page, 0, "ABSENT").isChecked(); const c1 = await rad(page, 1, "PRESENT").isChecked();
  await page.keyboard.press("Shift+3"); const e1 = await rad(page, 1, "EXCUSED").isChecked(); await page.keyboard.press("Shift+2"); const l1 = await rad(page, 1, "LATE").isChecked();
  await page.keyboard.press("2"); const l2 = await rad(page, 1, "LATE").isChecked(); const c2 = await rad(page, 2, "PRESENT").isChecked();
  await page.keyboard.press("ArrowDown"); await page.keyboard.press("ArrowUp"); await page.keyboard.press("1"); const p2 = await rad(page, 2, "PRESENT").isChecked();
  await page.keyboard.press("p");
  const sp = await rows(page).nth(3).locator("[data-part=speak-count]").count(); const sp2 = await rows(page).nth(2).locator("[data-part=speak-count]").count();
  await shot(page, "p502-18");
  const okCols = ["Có mặt", "Muộn", "Vắng phép", "Vắng", "Phát biểu"].every((x) => info.th.some((t) => t.startsWith(x)));
  R("TC-P5-02-18", okCols && info.radios === 4 && info.thPos === "sticky" && Math.abs(thTop0 - thTop1) < 2 && nameVisible && (ring.fv || ring.outline.includes("solid") || ring.shadow !== "none") && a1 && e1 && l1 && l2 && p2 && c1, `1280×800: cột ${JSON.stringify(info.th)}; 4 radio/hàng=${info.radios}; th position=${info.thPos}, td đầu=${info.tdPos}; cuộn bảng ${info.hasScroll ? `(scrollH ${info.scrollH} > ${info.clientH})` : "(không có vùng cuộn)"}: tiêu đề top ${thTop0.toFixed(0)} → ${thTop1.toFixed(0)} (dính), cột tên vẫn thấy=${nameVisible}; vòng focus ${JSON.stringify(ring)}; phím «4» hàng1=Vắng(${a1}) rồi xuống hàng 2 (hàng 2 vẫn Có mặt ${c1}); Shift+3 giữ hàng → Vắng phép(${e1}), Shift+2 → Muộn(${l1}) vẫn ở hàng 2; «2» đặt Muộn(${l2}) rồi xuống hàng 3; ↓↑ rồi «1» hàng 3 Có mặt(${p2}); «P» ghi phát biểu hàng đang chọn: hàng 4 có bộ đếm=${sp}, hàng 3=${sp2}`);
  await ctx.close();
});

test("19 30 SV bằng phím ≤ 60 s", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const S = ses(F.C); await openAtt(page, F.C, S);
  const n = await rows(page).count(); await page.getByLabel(/Bảng điểm danh/).focus();
  const t0 = Date.now(); for (let i = 0; i < n; i++) await page.keyboard.press([4, 12, 20].includes(i) ? "4" : i === 8 ? "2" : "1");
  await page.getByRole("button", { name: /^Lưu điểm danh/ }).first().click(); await page.getByText(/Đã lưu điểm danh · \d+ sinh viên/).waitFor({ timeout: 30000 }); const dt = (Date.now() - t0) / 1000;
  const msg = await page.getByText(/Đã lưu điểm danh · \d+ sinh viên/).innerText();
  await page.waitForTimeout(500);
  const abs = count(S, "ABSENT"), late = count(S, "LATE"), pres = count(S, "PRESENT"), done = psql(`select attendance_done_at is not null from class_sessions where id='${S}'`);
  R("TC-P5-02-19", dt <= 60 && abs === "3" && late === "1" && pres === String(n - 4) && done === "t", `lớp ${n} SV: gõ 30 phím (3 vắng ở hàng 5/13/21, 1 muộn hàng 9) + Lưu điểm danh trong ${dt.toFixed(1)} s (≤ 60 s); dòng «${msg}»; DB: ABSENT=${abs}, LATE=${late}, PRESENT=${pres}, session done=${done}`);
  await ctx.close();
});

test("20/21/22/23/26 thao tác trên lưới", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const S = ses(F.C); const reqs = puts(page);
  const posts: any[] = []; page.on("request", (r) => { if (r.method() === "POST" && /\/participation$/.test(r.url())) posts.push(JSON.parse(r.postData() ?? "{}")); });
  await openAtt(page, F.C, S);
  // 20
  const nm = await nameOf(page, 5);
  const m = await page.evaluate(async () => { const el = document.querySelectorAll("[data-part=student-row]")[5].querySelector("input[data-status=ABSENT]") as HTMLInputElement; const t0 = performance.now(); el.click(); await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))); return { dt: performance.now() - t0, checked: el.checked }; });
  const undoLine = page.getByText(new RegExp(`Đã đánh Vắng · ${nm}`)); const vis = await undoLine.isVisible().catch(() => false);
  const dlg = await page.getByRole("dialog").count(); const toast = await page.evaluate(() => document.querySelectorAll("[class*=toast],[data-part=toast],[role=alert]").length);
  const undoBtn = page.getByRole("button", { name: "Hoàn tác" }); const ub = await undoBtn.count();
  await undoBtn.first().click(); await page.waitForTimeout(300); const back = await rad(page, 5, "PRESENT").isChecked();
  await rad(page, 6, "ABSENT").click(); await page.getByText(/Đã đánh Vắng ·/).waitFor(); const t1 = Date.now(); await page.getByText(/Đã đánh Vắng ·/).waitFor({ state: "detached", timeout: 9000 }); const gone = (Date.now() - t1) / 1000;
  R("TC-P5-02-20", m.checked && m.dt <= 100 && vis && dlg === 0 && toast === 0 && ub >= 1 && back && gone <= 6, `đổi hàng 6 sang Vắng: UI đổi sau ${m.dt.toFixed(0)} ms (≤100); dòng tĩnh «Đã đánh Vắng · ${nm}» hiện=${vis} + nút Hoàn tác=${ub}; hộp thoại=${dlg}, toast/alert=${toast}; Hoàn tác → hàng 6 về Có mặt=${back}; dòng tự mất sau ≈${gone.toFixed(1)} s (5 s)`);
  // 21: gom 2 s
  await page.waitForTimeout(2500); reqs.length = 0;
  const seq: string[] = []; const t00 = Date.now(); await rad(page, 7, "ABSENT").click(); await page.waitForTimeout(250); await rad(page, 8, "LATE").click(); await page.waitForTimeout(250); await rad(page, 9, "EXCUSED").click();
  while (Date.now() - t00 < 5500) { const s = (await stateTxt(page)).trim(); if (seq[seq.length - 1] !== s) seq.push(s); await page.waitForTimeout(120); }
  const pr = reqs.filter((x) => !x.body.complete);
  R("TC-P5-02-21", pr.length === 1 && pr[0].body.records.length === 3 && seq.some((x) => /Đang lưu/.test(x)) && /Đã lưu \d\d:\d\d/.test(seq[seq.length - 1]) && pr[0].t - t00 >= 2300, `3 thay đổi trong ≈0,5 s: số PUT=${pr.length} (complete:false, ${pr[0]?.body.records.length} bản ghi) gửi sau ${((pr[0]?.t ?? 0) - t00)} ms (2 s yên lặng sau thay đổi cuối ≈ 500 ms ⇒ ≈2500); chỉ báo: ${JSON.stringify(seq)}`);
  // 22: 5xx
  await page.route(/\/sessions\/[^/]+\/attendance$/, (r) => (r.request().method() === "PUT" ? r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi máy chủ" }) }) : r.continue()));
  await rad(page, 10, "ABSENT").click(); await page.getByText(/Chưa lưu được/).waitFor({ timeout: 15000 });
  const keep = await rad(page, 10, "ABSENT").isChecked(); const retry = await page.getByRole("button", { name: "Thử lại" }).count();
  await page.unroute(/\/sessions\/[^/]+\/attendance$/); await page.getByRole("button", { name: "Thử lại" }).first().click(); await page.getByText(/Đã lưu \d\d:\d\d/).waitFor({ timeout: 15000 });
  await page.waitForTimeout(400); const dbA = psql(`select count(*) from attendance_records where session_id='${S}' and status='ABSENT' and student_id=(select (array_agg(user_id order by user_id))[1] from enrollments where false) or false`);
  R("TC-P5-02-22", keep && retry >= 1, `PUT trả 500: dòng «Chưa lưu được. Thử lại» hiện, nút Thử lại=${retry}, lựa chọn Vắng vẫn đánh dấu=${keep} (UI giữ lựa chọn của người dùng, không hoàn về Có mặt); bỏ chặn → Thử lại → «Đã lưu HH:MM»; ABSENT trong DB cho buổi=${count(S, "ABSENT")}`);
  // 26: phát biểu
  const sp = rows(page).nth(12).getByRole("button", { name: /Ghi phát biểu/ });
  const t2 = Date.now(); await sp.click(); await page.waitForTimeout(150); const c1 = (await rows(page).nth(12).locator("[data-part=speak-count]").innerText()).trim(); const lat = Date.now() - t2;
  await page.waitForTimeout(1200); const first = posts[0];
  for (let i = 0; i < 3; i++) { await sp.click(); await page.waitForTimeout(200); }
  await page.waitForTimeout(1500); const c4 = (await rows(page).nth(12).locator("[data-part=speak-count]").innerText()).trim();
  const ids = new Set(posts.map((p) => p.client_event_id)); const lab = await sp.evaluate((e) => ({ aria: e.getAttribute("aria-label"), title: e.getAttribute("title"), txt: (e as HTMLElement).innerText }));
  const db = psql(`select count(*) from participation_events where session_id='${S}' and type='SPEAK'`);
  R("TC-P5-02-26", posts.length === 4 && ids.size === 4 && first.type === "SPEAK" && first.session_id === S && !!first.student_id && c1.startsWith("+1") && c4.startsWith("+4") && db === "4", `bấm «+» 1 lần: bộ đếm «${c1}» sau ${lat} ms; POST …/participation ${JSON.stringify({ ...first, client_event_id: String(first?.client_event_id).slice(0, 8) + "…" })}; bấm thêm 3 lần: bộ đếm «${c4}», số POST=${posts.length}, client_event_id khác nhau=${ids.size}, DB participation_events SPEAK=${db}; nhãn nút: aria-label «${lab.aria}», tooltip «${lab.title}» (TC gốc: «+ phát biểu · +0,25»: nhãn hiện là biểu tượng + với aria «Ghi phát biểu», bộ đếm dạng «+n · +0,25×n» khi công thức đã có điểm cộng)`);
  // 23: Lưu điểm danh
  await rad(page, 13, "ABSENT").click(); await rad(page, 14, "ABSENT").click(); await rad(page, 15, "ABSENT").click();
  reqs.length = 0; await page.getByRole("button", { name: /^Lưu điểm danh/ }).first().click();
  const msg = page.getByText(/Đã lưu điểm danh · \d+ sinh viên · vắng \d+/); await msg.waitFor({ timeout: 20000 }); const mt = await msg.innerText();
  const comp = reqs.filter((x) => x.body.complete); const dlg2 = await page.getByRole("dialog").count(); const toast2 = await page.evaluate(() => document.querySelectorAll("[class*=toast],[data-part=toast]").length);
  const btn = page.locator("[data-part=att-save-bar] button"); const bt = (await btn.innerText()).trim(); const dis = await btn.isDisabled();
  const opt = (await page.locator("[data-part=att-session-select] option:checked").innerText()).trim();
  await rad(page, 20, "LATE").click(); await page.waitForTimeout(300); const dis2 = await btn.isDisabled();
  const absN = count(S, "ABSENT");
  R("TC-P5-02-23", comp.length === 1 && comp[0].body.records.length === 30 && /vắng \d+/.test(mt) && dlg2 === 0 && toast2 === 0 && dis && /^Đã lưu/.test(bt) && /Đã điểm danh/.test(opt) && !dis2, `bấm Lưu điểm danh: PUT complete:true=${comp.length} lần, ${comp[0]?.body.records.length} bản ghi (cả lưới); dòng «${mt}» (số vắng theo dữ liệu, DB ABSENT=${absN}); hộp thoại=${dlg2}, toast=${toast2}; nút «${bt}» khoá=${dis}; bộ chọn buổi «${opt}»; thêm một thay đổi → nút mở lại=${!dis2}`);
  await shot(page, "p502-23"); await ctx.close();
});

test("24/25 bộ chọn buổi, rỗng, đổi buổi giữ thay đổi", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t");
  const sPast = ses(F.C0, -60 * 48, 90), sNow = ses(F.C0, -20, 90), sFut = ses(F.C0, 60 * 24, 90);
  psql(`update class_sessions set attendance_done_at=now(), attendance_done_by=(select id from users where email='qc3t@edupilot.local') where id='${sPast}'`);
  await go(page, `/attendance?course=${F.C0}`); await page.locator("[data-part=att-session-select]").waitFor();
  const sel = page.locator("[data-part=att-session-select]"); const v0 = await sel.inputValue(); const opts = await sel.locator("option").allInnerTexts();
  await go(page, `/attendance?course=${F.C0}&session=${sPast}`); await page.locator("[data-part=att-session-select]").waitFor(); const v1 = await page.locator("[data-part=att-session-select]").inputValue(); const url1 = page.url();
  await sel.selectOption(sFut).catch(() => undefined); await page.waitForTimeout(1500); const url2 = page.url();
  R("TC-P5-02-24", v0 === sNow && opts.length === 3 && /Chưa$/.test(opts[0]) && /Chưa$/.test(opts[1]) && /Đã điểm danh$/.test(opts[2]) && v1 === sPast && url2.includes(sFut), `mở /attendance không ?session= → mặc định buổi đang diễn ra (${v0 === sNow}); thứ tự gần nhất trước: ${JSON.stringify(opts)}; ?session=<buổi đã điểm danh> chọn đúng (${v1 === sPast}); đổi buổi ghi ?session= vào URL (${url2.replace(/^https:\/\/[^/]+/, "")})`);
  // 25
  await go(page, `/attendance?course=${F.CE}`); await page.getByText("Lớp chưa có lịch buổi học").waitFor({ timeout: 20000 });
  const link = await page.getByRole("link", { name: "Tạo lịch buổi học" }).getAttribute("href"); const emptyTxt = (await page.locator("main").innerText()).replace(/\n+/g, " | ");
  const sA = ses(F.C, -30, 90), sB = ses(F.C, -25, 90); await openAtt(page, F.C, sA);
  await rad(page, 2, "ABSENT").click(); await page.locator("[data-part=att-session-select]").selectOption(sB); await page.waitForTimeout(3500);
  const kept = psql(`select count(*) from attendance_records where session_id='${sA}' and status='ABSENT'`);
  R("TC-P5-02-25", link === "/calendar" && kept === "1", `lớp chưa có buổi: «Lớp chưa có lịch buổi học» + liên kết «Tạo lịch buổi học» → ${link}; nội dung trang: ${emptyTxt.slice(0, 140)}; đổi buổi khi còn 1 thay đổi chưa gửi (≈0 s sau khi bấm Vắng): sau 3,5 s DB buổi cũ có ${kept} bản ghi ABSENT (tự lưu trước khi đổi, không mất)`);
  await ctx.close();
});

test("31/32/33/35/36 mất mạng", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const S = ses(F.C); const reqs = puts(page); await openAtt(page, F.C, S);
  const userId = await page.evaluate(() => Object.keys(localStorage).find((k) => k.startsWith("ep:draft:")) ?? "");
  await ctx.setOffline(true); await page.waitForTimeout(500);
  const t0 = Date.now(); await rad(page, 3, "ABSENT").click(); await rad(page, 4, "ABSENT").click(); await rad(page, 5, "ABSENT").click();
  const banner = page.locator("[data-part=offline-banner]"); let bt = ""; for (let i = 0; i < 20 && !/3 thay đổi/.test(bt); i++) { bt = (await banner.innerText()).trim(); await page.waitForTimeout(150); } const bl = Date.now() - t0;
  const st = (await stateTxt(page)).trim();
  const keys = await page.evaluate(() => Object.keys(localStorage).filter((k) => k.includes(":attendance:")));
  const saveBtn = page.getByRole("button", { name: /^Lưu điểm danh|^Chờ mạng/ }).first(); const saveDis = await page.locator("[data-part=att-save-bar] button").isDisabled(); const saveTxt = (await page.locator("[data-part=att-save-bar] button").innerText()).trim();
  // 35: đăng xuất hỏi trước (còn thay đổi chờ lưu)
  await page.locator("[data-part=topbar]").getByRole("button").last().click().catch(() => undefined);
  const menu = page.getByRole("menuitem", { name: "Đăng xuất" }).or(page.getByRole("button", { name: "Đăng xuất" }));
  if (!(await menu.first().isVisible().catch(() => false))) { const trg = page.locator("[data-part=topbar] button[aria-haspopup]").last(); await trg.click().catch(() => undefined); }
  await menu.first().click(); await page.getByText("Còn điểm danh chưa lưu").waitFor({ timeout: 8000 }).catch(() => undefined); const askTxt = (await page.getByRole("dialog").innerText().catch(() => "")).replace(/\n+/g, " | ");
  await page.getByRole("button", { name: "Ở lại" }).click().catch(() => undefined); await page.waitForTimeout(500); const stay = page.url().includes("/attendance");
  R("TC-P5-02-35", /Còn điểm danh chưa lưu/.test(askTxt) && stay, `offline, 3 thay đổi chờ lưu → Đăng xuất: hộp «${askTxt}» (hỏi trước, nút «Ở lại»/«Đăng xuất»); bấm Ở lại → vẫn ở ${page.url().replace(/^https:\/\/[^/]+/, "")} (${stay})`);
  await ctx.setOffline(false);
  await page.getByText(/Đã lưu \d\d:\d\d/).waitFor({ timeout: 30000 }); await page.waitForTimeout(500);
  const abs = count(S, "ABSENT"); const sent = reqs.filter((x) => !x.body.complete).map((x) => x.body.records.map((r: any) => r.student_id.slice(0, 4))); const order = reqs.length;
  R("TC-P5-02-31", /Chưa có mạng · 3 thay đổi chờ lưu/.test(bt) && abs === "3" && keys.length === 1, `setOffline(true); đánh Vắng 3 người; banner «${bt}» hiện sau ${bl} ms; chỉ báo ở lưới «${st}»; kho: ${JSON.stringify(keys)}; setOffline(false) → tự gửi (PUT ${order} lần) → «Đã lưu»; DB ABSENT=${abs}`);
  // 33: Lưu điểm danh offline
  await ctx.setOffline(true); await page.waitForTimeout(400); await rad(page, 8, "LATE").click(); await page.waitForTimeout(300);
  const bar = page.locator("[data-part=att-save-bar] button"); const barDis = await bar.isDisabled(); const barTxt = (await bar.innerText()).trim();
  const hdr = page.getByRole("button", { name: /^Lưu điểm danh/ }); const hdrDis = (await hdr.count()) ? await hdr.first().isDisabled() : null;
  let queued = false; if (hdrDis === false) { await hdr.first().click(); queued = true; }
  await ctx.setOffline(false); await page.waitForTimeout(6000); const doneAt = psql(`select attendance_done_at is not null from class_sessions where id='${S}'`);
  R("TC-P5-02-33", queued && doneAt === "t", `offline: nút thanh dưới «${barTxt}» khoá=${barDis}; nút đầu trang «Lưu điểm danh» khoá=${hdrDis}; ${queued ? "đã bấm khi offline" : "KHÔNG bấm được khi offline (nút bị khoá)"}; sau khi có mạng 6 s: session.attendance_done_at đặt=${doneAt} (kỳ vọng AC13: xếp hàng complete:true rồi done_at)`);
  await shot(page, "p502-33");
  // 32: tải lại giữ hàng đợi (PUT bị chặn bằng route abort, trang tải được)
  const S2 = ses(F.C); await ctx.route(/\/sessions\/[^/]+\/attendance$/, (r) => (r.request().method() === "PUT" ? r.abort("failed") : r.continue()));
  await openAtt(page, F.C, S2); await rad(page, 1, "ABSENT").click(); await rad(page, 2, "ABSENT").click(); await rad(page, 3, "ABSENT").click(); await page.waitForTimeout(3000);
  const key = await page.evaluate(() => Object.keys(localStorage).filter((k) => k.includes(":attendance:")));
  await page.reload(); await page.locator("[data-part=student-row]").first().waitFor({ timeout: 30000 }); await page.waitForTimeout(1500);
  const afterReload = [await rad(page, 1, "ABSENT").isChecked(), await rad(page, 2, "ABSENT").isChecked(), await rad(page, 3, "ABSENT").isChecked()];
  const state = await ctx.storageState(); const ctx2 = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: "https://localhost:1143", viewport: { width: 1280, height: 800 }, storageState: state });
  await ctx2.route(/\/sessions\/[^/]+\/attendance$/, (r) => (r.request().method() === "PUT" ? r.abort("failed") : r.continue()));
  const p2 = await ctx2.newPage(); await go(p2, `/attendance?course=${F.C}&session=${S2}`); await p2.locator("[data-part=student-row]").first().waitFor({ timeout: 30000 }); await p2.waitForTimeout(1500);
  const reopen = [await rad(p2, 1, "ABSENT").isChecked(), await rad(p2, 2, "ABSENT").isChecked(), await rad(p2, 3, "ABSENT").isChecked()];
  const pat = /^ep:draft:[0-9a-f-]{36}:attendance:[0-9a-f-]{36}:[0-9a-f-]{36}/.test(key[0] ?? "");
  R("TC-P5-02-32", afterReload.every(Boolean) && reopen.every(Boolean) && key.length === 1 && pat && count(S2, "ABSENT") === "0", `PUT bị chặn (mạng đứt giả): 3 thay đổi chờ; kho localStorage ${JSON.stringify(key.map((k) => k.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/g, "<uuid>")))} khớp ep:draft:<userId>:attendance:<course>:<session>=${pat}; tải lại trang: 3 hàng vẫn Vắng ${JSON.stringify(afterReload)}; đóng trình duyệt + mở lại (newContext cùng storageState): ${JSON.stringify(reopen)}; DB chưa có bản ghi (${count(S2, "ABSENT")})`);
  await ctx2.close(); await ctx.close();
});

test("34 409 khi mạng về", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const S = ses(F.C); const reqs = puts(page); await openAtt(page, F.C, S);
  // tạo bản ghi gốc để có version: TA đánh vắng hàng 6 trước, GV tải lại để có version
  const ta = await newUser(browser, "qc3ta"); await openAtt(ta.page, F.C, S); await rad(ta.page, 6, "LATE").click(); await ta.page.waitForTimeout(3500);
  await page.reload(); await page.locator("[data-part=student-row]").first().waitFor({ timeout: 30000 }); await page.waitForTimeout(800);
  await ctx.setOffline(true); await page.waitForTimeout(400);
  await rad(page, 6, "ABSENT").click(); await rad(page, 7, "ABSENT").click(); await page.waitForTimeout(300);
  await rad(ta.page, 6, "EXCUSED").click(); await ta.page.waitForTimeout(3500); // người khác đổi cùng hàng trong lúc GV offline
  reqs.length = 0; await ctx.setOffline(false); await page.getByText(/vừa đổi 1 dòng/).waitFor({ timeout: 40000 }).catch(() => undefined);
  await page.waitForTimeout(8000);
  const n1 = reqs.length; const notice = (await page.locator("main").innerText()).match(/[^\n]*vừa đổi[^\n]*/)?.[0] ?? "";
  const mine = await page.getByRole("button", { name: "Giữ của tôi" }).count(); const theirs = await page.getByRole("button", { name: /^Lấy của/ }).count();
  const st = (await stateTxt(page)).trim(); const stay7 = await rad(page, 7, "ABSENT").isChecked();
  await page.waitForTimeout(16000); const n2 = reqs.length; const db6 = psql(`select status from attendance_records where session_id='${S}' order by created_at limit 1`);
  await page.getByRole("button", { name: "Giữ của tôi" }).first().click().catch(() => undefined); await page.waitForTimeout(4000);
  R("TC-P5-02-34", /vừa đổi 1 dòng/.test(notice) && mine === 1 && theirs === 1 && n2 === n1 && stay7, `GV offline đổi hàng 7 + 8 (hàng đã có bản ghi của TA); TA đổi hàng 7 trong lúc đó; có mạng lại: PUT tự gửi ${n1} lần, 409 → «${notice}», nút «Giữ của tôi»=${mine}, «Lấy của …»=${theirs}, chỉ báo «${st}»; chờ thêm 16 s: số PUT ${n1} → ${n2} (hàng đợi dừng, không thử lại); thay đổi khác của GV (hàng 8 Vắng) giữ=${stay7}; sau «Giữ của tôi»: DB hàng 7=${count(S, "ABSENT")} ABSENT`);
  await ta.ctx.close(); await ctx.close();
});

test("10/E1/E2 xung đột hai người", async ({ browser }) => {
  for (const slow of [false, true]) {
    const A1 = await newUser(browser, "qc3t"); const B1 = await newUser(browser, "qc3ta"); const S = ses(F.C);
    if (slow) for (const x of [A1, B1]) { const c = await x.ctx.newCDPSession(x.page); await c.send("Network.enable"); await c.send("Network.emulateNetworkConditions", { offline: false, latency: 400, downloadThroughput: (50 * 1024) / 8, uploadThroughput: (50 * 1024) / 8 }); }
    await openAtt(A1.page, F.C, S); await openAtt(B1.page, F.C, S);
    await rad(A1.page, 3, "ABSENT").click(); await rad(B1.page, 3, "LATE").click(); await rad(B1.page, 4, "ABSENT").click();
    await A1.page.waitForTimeout(slow ? 7000 : 3500); // A lưu trước
    await B1.page.waitForTimeout(slow ? 9000 : 5000);
    const body = (await B1.page.locator("main").innerText()); const notice = body.match(/[^\n]*vừa đổi \d+ dòng[^\n]*/)?.[0] ?? "";
    const mine = await B1.page.getByRole("button", { name: "Giữ của tôi" }).count(); const theirs = await B1.page.getByRole("button", { name: /^Lấy của/ }).count(); const keep4 = await rad(B1.page, 4, "ABSENT").isChecked();
    await B1.page.getByRole("button", { name: "Giữ của tôi" }).first().click().catch(() => undefined); await B1.page.waitForTimeout(slow ? 9000 : 5000);
    const db3 = psql(`select r.status from attendance_records r join enrollments e on e.user_id=r.student_id and e.course_id=r.course_id where r.session_id='${S}' order by e.student_code_snapshot limit 1 offset 3`); const db4 = psql(`select r.status from attendance_records r join enrollments e on e.user_id=r.student_id and e.course_id=r.course_id where r.session_id='${S}' order by e.student_code_snapshot limit 1 offset 4`);
    const tc = slow ? "TC-P5-02-E1" : "TC-P5-02-10"; const ok = /vừa đổi 1 dòng/.test(notice) && mine === 1 && theirs === 1 && keep4;
    R(tc, ok && db3 !== "", `${slow ? "mạng 3G giả (độ trễ 400 ms, 50 kbps) ở cả hai ngữ cảnh: " : ""}GV (qc3t) và TA (qc3ta) cùng /attendance?session=…: cả hai đổi hàng 4 (GV Vắng, TA Muộn), TA còn đổi hàng 5 (Vắng); GV lưu trước, TA lưu: TA thấy «${notice}» + «Giữ của tôi»=${mine} / «Lấy của …»=${theirs}; thay đổi còn lại của TA (hàng 5 Vắng) giữ=${keep4}; chọn «Giữ của tôi» → gửi lại với version mới; DB hàng 4=${db3 || "—"}, hàng 5=${db4 || "—"}`);
    await shot(B1.page, slow ? "p502-e1" : "p502-10"); await A1.ctx.close(); await B1.ctx.close();
  }
});

test("E2 PUT lặp sau mất mạng không audit trùng", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const S = ses(F.C); await openAtt(page, F.C, S);
  const before = psql(`select count(*) from audit_log where course_id='${F.C}' and entity_id::text='${S}'`);
  let n = 0; await page.route(/\/sessions\/[^/]+\/attendance$/, async (r) => { if (r.request().method() !== "PUT") return r.continue(); n++; if (n === 1) { await r.fetch(); return r.abort("failed"); } return r.continue(); });
  await rad(page, 2, "ABSENT").click(); await rad(page, 3, "ABSENT").click(); await page.waitForTimeout(2500); await page.getByText(/Đã lưu \d\d:\d\d/).waitFor({ timeout: 40000 }).catch(() => undefined); await page.waitForTimeout(1000);
  const rec = count(S, "ABSENT"); const aud = psql(`select action||'/'||count(*) from audit_log where course_id='${F.C}' and created_at>now()-interval '2 minutes' and (entity_id::text='${S}' or after::text like '%${S}%') group by action`).replace(/\n/g, "; ");
  const ver = psql(`select max(version) from attendance_records where session_id='${S}'`);
  R("TC-P5-02-E2", n >= 2 && rec === "2" && ver === "1", `PUT đầu tới máy chủ (đã ghi) nhưng phản hồi bị cắt → client gửi lại: số PUT=${n}; bản ghi ABSENT=${rec} (không nhân đôi), version lớn nhất=${ver} (lần gửi lặp không tăng version); audit_log 2 phút gần nhất: ${aud || "(không có dòng nào gắn buổi)"}`);
  await ctx.close();
});

test("44 trạng thái tải/lỗi", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3t"); const S = ses(F.C);
  await page.route(/\/sessions\/[^/]+\/attendance$/, async (r) => { if (r.request().method() === "GET") { await new Promise((x) => setTimeout(x, 3500)); return r.continue(); } return r.continue(); });
  await page.goto(`/attendance?course=${F.C}&session=${S}`); await page.waitForTimeout(1500);
  const sk = await page.evaluate(() => document.querySelectorAll("[class*=skeleton],[data-skeleton],[aria-busy=true]").length); await page.locator("[data-part=student-row]").first().waitFor({ timeout: 30000 });
  await page.unroute(/\/sessions\/[^/]+\/attendance$/); let k = 0;
  await page.route(/\/sessions\/[^/]+\/attendance$/, (r) => (r.request().method() === "GET" && ++k <= 2 ? r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }) : r.continue()));
  await page.reload(); await page.getByRole("button", { name: "Thử lại" }).waitFor({ timeout: 40000 }); const et = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 200); const sel = page.url().includes(S);
  await page.unroute(/\/sessions\/[^/]+\/attendance$/); await page.getByRole("button", { name: "Thử lại" }).click(); await page.locator("[data-part=student-row]").first().waitFor({ timeout: 30000 });
  R("TC-P5-02-44", sk >= 1 && /Thử lại/.test(et) && sel, `chậm (GET lưới trễ 3,5 s): khung xương=${sk} phần tử; GET 500: «${et}» + nút Thử lại; buổi đang chọn còn trong URL=${sel}; bấm Thử lại → lưới ${await rows(page).count()} hàng`);
  await ctx.close();
});
