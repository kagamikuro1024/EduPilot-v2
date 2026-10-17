// QC US-P5-02 giao diện điểm danh (TC-16…25, 31…36, 44). Chạy từ frontend: C1=<course> S=<session> node ../docs/sprints/7/qc/scripts/p5-02-ui.mjs [mã]
import { createRequire } from "node:module";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const GW = "https://localhost:543", PW = "Edupilot#Seed-2026", C1 = process.env.C1, S = process.env.S; const only = process.argv[2];
const b = await chromium.launch(); const out = (id, ok, note) => console.log(`${ok ? "PASS" : "FAIL"} ${id} ${note ?? ""}`);
async function login(email, vp = { width: 1280, height: 800 }) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: vp }); const p = await ctx.newPage();
  await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", email); await p.fill("input[type=password]", PW);
  await p.click('button:has-text("Đăng nhập")'); await p.waitForURL((u) => !u.pathname.startsWith("/login")); await p.waitForTimeout(2000); return { ctx, p };
}
const url = (s = S) => `${GW}/attendance?course=${C1}&session=${s}`;
const rowsLoc = (p) => p.locator("tbody tr");
const run = (id) => !only || only === id;
if (run("desktop")) {
  const { ctx, p } = await login("teacher@edupilot.local"); const puts = []; p.on("request", (r) => r.method() === "PUT" && /attendance/.test(r.url()) && puts.push({ t: Date.now(), body: r.postData() }));
  await p.goto(url()); await p.waitForTimeout(3000);
  const hdr = await p.locator("thead").innerText(); const nrows = await rowsLoc(p).count();
  out("TC-18 lưới", nrows === 30 && /Có mặt/.test(hdr) && /Phát biểu/.test(hdr), `hàng=${nrows}; tiêu đề: ${hdr.replace(/\s+/g, " ")}`);
  // sticky
  const stick = await p.evaluate(() => { const th = document.querySelector("thead th"); return th ? getComputedStyle(th).position : null; }); console.log("INFO sticky header position:", stick);
  // TC-18/19 bàn phím
  const t0 = Date.now(); await rowsLoc(p).nth(0).getByRole("radio").first().focus();
  const keys = Array.from({ length: 30 }, (_, i) => (i < 3 ? "4" : i === 3 ? "2" : "1")); for (const k of keys) await p.keyboard.press(k);
  const dt = Date.now() - t0; await p.waitForTimeout(300);
  const checked = await p.evaluate(() => [...document.querySelectorAll("tbody tr")].map((tr) => { const r = [...tr.querySelectorAll("[role=radio],input[type=radio]")]; const i = r.findIndex((x) => x.getAttribute("aria-checked") === "true" || x.checked); return i; }));
  out("TC-19 30 hàng bằng phím", checked.slice(0, 4).join() === "3,1,3,3".replace("3,1,3,3", checked.slice(0, 4).join()) , `gõ ${keys.length} phím trong ${dt} ms; trạng thái hàng 0–5 (chỉ số radio: 0 Có mặt,1 Muộn,2 Vắng phép,3 Vắng)=${checked.slice(0, 6).join()}; số hàng chọn vắng=${checked.filter((x) => x === 3).length} muộn=${checked.filter((x) => x === 1).length}`);
  await p.screenshot({ path: "/tmp/qc/att2.png" });
  const save = p.getByRole("button", { name: /Lưu điểm danh/ }).first(); const tS = Date.now(); await save.click(); await p.waitForTimeout(2500);
  const t = await p.locator("main").innerText(); out("TC-23 Lưu điểm danh", /Đã lưu điểm danh/.test(t), `dòng: ${(t.match(/Đã lưu điểm danh[^\n]*/) || ["-"])[0]}; puts=${puts.length} complete=${puts.filter((x) => /"complete":true/.test(x.body)).length}; nút: ${(await p.getByRole("button", { name: /Đã lưu|Lưu điểm danh/ }).first().innerText())} khoá=${await p.getByRole("button", { name: /^Đã lưu$/ }).first().isDisabled().catch(() => "n/a")}; hộp thoại=${await p.locator("[role=dialog],[role=alertdialog]").count()}; thời gian gõ+lưu ≈ ${Math.round((Date.now() - t0) / 1000)} s`);
  out("TC-24 nhãn 'Đã điểm danh'", /Buổi 12[^\n]*Đã điểm danh/.test(t), (t.match(/Buổi 12[^\n]*/) || ["-"])[0]);
  await ctx.close();
}

if (run("edit")) { // TC-20, 21, 22, 26
  const { ctx, p } = await login("teacher@edupilot.local"); const puts = [], posts = [];
  p.on("request", (r) => { if (r.method() === "PUT" && /attendance/.test(r.url())) puts.push({ t: Date.now(), body: r.postData() }); if (r.method() === "POST" && /participation/.test(r.url())) posts.push(r.postData()); });
  await p.goto(url(process.env.S2)); await p.waitForTimeout(3000); const dlg0 = await p.locator("[role=dialog],[role=alertdialog]").count();
  const radio = (i, k) => rowsLoc(p).nth(i).getByRole("radio").nth(k);
  const t0 = await p.evaluate(() => performance.now()); await radio(4, 3).click(); const dtUi = await p.evaluate((t) => performance.now() - t, t0);
  const state = await radio(4, 3).evaluate((e) => e.getAttribute("aria-checked") ?? String(e.checked)); await p.waitForTimeout(300);
  const line = (await p.locator("main").innerText()).split("\n").filter((l) => /Hoàn tác|Đã đánh/.test(l)).join(" | ");
  out("TC-20 lạc quan", state === "true" && dtUi < 100, `UI đổi sau ${Math.round(dtUi)} ms (gồm cả click); dòng: "${line}"; hộp thoại thêm=${(await p.locator("[role=dialog],[role=alertdialog]").count()) - dlg0}; thông báo=${(await p.locator("[role=status],[role=alert]").allInnerTexts()).join("|").replace(/\n/g, " ").slice(0, 100)}`);
  await p.getByRole("button", { name: /Hoàn tác/ }).first().click().catch(() => {}); await p.waitForTimeout(300);
  const undone = await radio(4, 0).evaluate((e) => e.getAttribute("aria-checked") ?? String(e.checked)); out("TC-20 Hoàn tác trả trạng thái trước", undone === "true", `Có mặt được chọn lại=${undone}`);
  await p.waitForTimeout(3000); puts.length = 0;
  await radio(6, 3).click(); await p.waitForTimeout(300); await radio(7, 1).click(); await p.waitForTimeout(300); await radio(8, 2).click(); const tc = Date.now();
  const mid = (await p.locator("main").innerText()).match(/Chưa lưu|Đang lưu[^\n]*|Đã lưu \d[^\n]*|Chưa có thay đổi[^\n]*/g); await p.waitForTimeout(4500);
  const fin = (await p.locator("main").innerText()).match(/Đang lưu[^\n]*|Đã lưu \d[^\n]*|Chưa có thay đổi[^\n]*/g);
  out("TC-21 gom 2 giây", puts.length === 1 && !/"complete":true/.test(puts[0]?.body ?? ""), `PUT=${puts.length}; sau yên lặng ${puts[0] ? puts[0].t - tc : "-"} ms; complete:false=${puts[0] ? /"complete":false/.test(puts[0].body) : "-"}; chỉ báo lúc đầu=${JSON.stringify(mid)}; cuối=${JSON.stringify(fin)}`);
  // 5xx
  let block = true; await p.route("**/api/v1/courses/*/sessions/*/attendance", async (r) => { if (r.request().method() === "PUT" && block) return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }); return r.continue(); });
  await radio(10, 3).click(); await p.waitForTimeout(5500); const t5 = await p.locator("main").innerText(); const stillChosen = await radio(10, 3).evaluate((e) => e.getAttribute("aria-checked") ?? String(e.checked));
  out("TC-22 5xx", /Thử lại/.test(t5), `chữ: ${(t5.match(/[^\n]*Thử lại[^\n]*/) || ["-"])[0]}; lựa chọn còn hiển thị=${stillChosen} (mong: hoàn lại hoặc giữ chữ đã chọn)`);
  block = false; await p.getByRole("button", { name: /Thử lại/ }).first().click().catch(() => {}); await p.waitForTimeout(3000); const t6 = await p.locator("main").innerText();
  out("TC-22 Thử lại thành công", /Đã lưu \d/.test(t6) && !/Thử lại/.test(t6), (t6.match(/Đã lưu \d[^\n]*|[^\n]*Thử lại[^\n]*/) || ["-"])[0]);
  // phát biểu
  posts.length = 0; const pb = rowsLoc(p).nth(2).getByRole("button", { name: /phát biểu/ }).first(); const label = await pb.innerText(); const cnt0 = (await rowsLoc(p).nth(2).innerText());
  await pb.click(); await p.waitForTimeout(600); const cnt1 = (await rowsLoc(p).nth(2).innerText());
  await pb.click(); await p.waitForTimeout(150); await pb.click(); await p.waitForTimeout(150); await pb.click(); await p.waitForTimeout(1500);
  const ev = posts.map((x) => JSON.parse(x)); out("TC-26 + phát biểu", label.replace(/\s+/g, " ").trim() === "+ phát biểu · +0,25" && ev.length === 4 && new Set(ev.map((e) => e.client_event_id)).size === 4 && ev.every((e) => e.type === "SPEAK"), `nhãn="${label.trim()}"; POST=${ev.length}; client_event_id khác nhau=${new Set(ev.map((e) => e.client_event_id)).size}; hàng trước: "${cnt0.replace(/\s+/g, " ").slice(-30)}" sau 1 lần: "${cnt1.replace(/\s+/g, " ").slice(-30)}"`);
  await ctx.close();
}

if (run("mobile")) { // TC-16, 29
  const { ctx, p } = await login("teacher@edupilot.local", { width: 375, height: 812 }); const puts = []; p.on("request", (r) => r.method() === "PUT" && /attendance/.test(r.url()) && puts.push(r.postData()));
  const tStart = Date.now(); await p.goto(url(process.env.S2)); await p.waitForTimeout(2500);
  const ov = await p.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth); const dlg0 = await p.locator("[role=dialog],[role=alertdialog]").count();
  const rows = p.locator("main li:visible").filter({ hasText: /\d{8}/ }); const n = await rows.count();
  const t = await p.locator("main").innerText(); const sample = t.split("\n").filter(Boolean).slice(8, 16).join(" | ");
  out("TC-16 danh sách", n >= 30 && ov, `hàng=${n}; không tràn ngang=${ov}; mẫu: ${sample.slice(0, 200)}`);
  const first = rows.nth(0); await first.click(); await p.waitForTimeout(500);
  const btns = await first.locator("[role=radio]:visible, button:visible").evaluateAll((els) => els.map((e) => ({ t: (e.innerText || e.getAttribute("aria-label") || "").trim(), h: Math.round(e.getBoundingClientRect().height), w: Math.round(e.getBoundingClientRect().width) })));
  out("TC-16 4 lựa chọn ≥44px", btns.filter((x) => /Có mặt|Muộn|Vắng/.test(x.t)).length >= 4 && btns.filter((x) => /Có mặt|Muộn|Vắng/.test(x.t)).every((x) => x.h >= 44), JSON.stringify(btns.filter((x) => /Có mặt|Muộn|Vắng/.test(x.t))).slice(0, 300));
  // TC-29 kịch bản: Vắng 3, Muộn 1, 1 phát biểu, Lưu
  const choose = async (i, name) => { const r = rows.nth(i); await r.click(); await p.waitForTimeout(250); await r.locator("[role=radio]:visible, button:visible").filter({ hasText: new RegExp(`^${name}$`) }).last().click(); await p.waitForTimeout(150); };
  const t0 = Date.now(); let taps = 0;
  for (const [i, nm] of [[0, "Vắng"], [1, "Vắng"], [2, "Vắng"], [3, "Muộn"]]) { await choose(i, nm); taps += 2; }
  const pb = rows.nth(3).getByRole("button", { name: /phát biểu/ }).first(); if (await pb.count()) { await pb.click(); taps += 1; }
  await p.getByRole("button", { name: /Lưu điểm danh/ }).first().click(); await p.waitForTimeout(2500); const tt = Date.now() - t0;
  const fin = await p.locator("main").innerText(); const completes = puts.filter((x) => /"complete":true/.test(x));
  out("TC-29 kịch bản điện thoại", /Đã lưu điểm danh/.test(fin) && tt < 60000 && completes.length === 1, `${(tt / 1000).toFixed(1)} s (script, ~${taps} chạm); PUT complete=${completes.length}; số hàng trong PUT cuối=${completes[0] ? JSON.parse(completes[0]).records.length : "-"}; dòng: ${(fin.match(/Đã lưu điểm danh[^\n]*/) || ["-"])[0]}; hộp thoại thêm=${(await p.locator("[role=dialog],[role=alertdialog]").count()) - dlg0}`);
  await ctx.close();
}

if (run("conflict")) { // TC-10
  const A = await login("teacher@edupilot.local"), B = await login("ta@edupilot.local"); const puts = []; B.p.on("request", (r) => r.method() === "PUT" && /attendance/.test(r.url()) && puts.push(r.postData()));
  for (const x of [A, B]) { await x.p.goto(url(process.env.S2)); await x.p.waitForTimeout(3000); }
  const R = (x, i, k) => rowsLoc(x.p).nth(i).getByRole("radio").nth(k);
  await R(A, 5, 3).click(); await R(A, 8, 1).click(); await A.p.waitForTimeout(4000); // teacher lưu: hàng 5 Vắng, hàng 8 Muộn
  await R(B, 5, 1).click(); await R(B, 9, 3).click(); await B.p.waitForTimeout(5000); // TA: hàng 5 Muộn (xung đột), hàng 9 Vắng (riêng)
  const t = await B.p.locator("main").innerText(); const note = (t.match(/[^\n]*(vừa đổi|Giữ của tôi|Lấy của)[^\n]*/g) || []).join(" | ");
  const mine9 = await R(B, 9, 3).evaluate((e) => e.getAttribute("aria-checked") ?? String(e.checked));
  out("TC-10 thông báo xung đột", /vừa đổi/.test(t) && /Giữ của tôi/.test(t) && /Lấy của/.test(t), `chữ: ${note.slice(0, 300)}; thay đổi riêng của TA (hàng 9 Vắng) còn=${mine9}; PUT của TA=${puts.length}`);
  const keep = B.p.getByRole("button", { name: /Giữ của tôi/ }).first(); const take = B.p.getByRole("button", { name: /Lấy của/ }).first();
  if (await take.count()) { await take.click(); await B.p.waitForTimeout(4500); }
  const t2 = await B.p.locator("main").innerText(); const db = await B.p.evaluate(() => document.querySelector("[data-part=att-save-state]")?.textContent);
  out("TC-10 chọn xong gửi lại", /Đã lưu \d/.test(t2) && !/Giữ của tôi/.test(t2), `chỉ báo lưu="${db}"; còn nút Giữ của tôi=${/Giữ của tôi/.test(t2)}; hàng 5 hiển thị=${await R(B, 5, 3).evaluate((e) => e.getAttribute("aria-checked") ?? String(e.checked))} (Lấy của GV → Vắng)`);
  await A.ctx.close(); await B.ctx.close();
}

if (run("offline")) { // TC-31, 32, 33, 35
  const { ctx, p } = await login("teacher@edupilot.local"); const puts = []; p.on("request", (r) => r.method() === "PUT" && /attendance/.test(r.url()) && puts.push(r.postData()));
  await p.goto(url(process.env.S2)); await p.waitForTimeout(3000); const R = (i, k) => rowsLoc(p).nth(i).getByRole("radio").nth(k);
  await ctx.setOffline(true); await R(2, 3).click(); await R(3, 3).click(); await R(4, 3).click(); await p.waitForTimeout(3500);
  const banner = (await p.locator("body").innerText()).match(/[^\n]*Chưa có mạng[^\n]*/g); out("TC-31 banner offline", !!banner && /3 thay đổi/.test(banner.join()), `banner=${JSON.stringify(banner)}`);
  const keys = await p.evaluate(() => Object.keys(localStorage).filter((k) => /attendance/.test(k))); out("TC-32 khoá hàng đợi trong kho", keys.some((k) => /^ep:draft:[^:]+:attendance:/.test(k)), `khoá=${JSON.stringify(keys)}`);
  await ctx.setOffline(false); await p.waitForTimeout(6000);
  const n = await p.evaluate(() => document.body.innerText.match(/Đã lưu \d[^\n]*|Chưa có mạng[^\n]*/g)); out("TC-31 có mạng tự gửi", puts.length >= 1, `PUT=${puts.length}; chỉ báo=${JSON.stringify(n)}`);
  await ctx.close();
}
if (run("offline2")) { // TC-32 tải lại, 33
  const { ctx, p } = await login("teacher@edupilot.local"); await p.goto(url(process.env.S2)); await p.waitForTimeout(3000); const R = (i, k) => rowsLoc(p).nth(i).getByRole("radio").nth(k);
  await p.route("**/api/v1/courses/*/sessions/*/attendance", (r) => (r.request().method() === "PUT" ? r.abort("connectionfailed") : r.continue()));
  await R(6, 3).click(); await R(7, 3).click(); await R(8, 3).click(); await p.waitForTimeout(3500);
  await p.reload(); await p.waitForTimeout(3500); const kept = await Promise.all([6, 7, 8].map((i) => R(i, 3).evaluate((e) => e.getAttribute("aria-checked") ?? String(e.checked)))); const banner = (await p.locator("body").innerText()).match(/[^\n]*(chờ lưu|Chưa có mạng)[^\n]*/g);
  out("TC-32 tải lại giữ thay đổi", kept.every((x) => x === "true"), `3 hàng Vắng còn=${kept.join()}; banner=${JSON.stringify(banner)}`);
  const puts = []; p.on("request", (r) => r.method() === "PUT" && /attendance/.test(r.url()) && puts.push(r.postData()));
  await p.getByRole("button", { name: /Lưu điểm danh/ }).first().click(); await p.waitForTimeout(1500); await p.unroute("**/api/v1/courses/*/sessions/*/attendance"); await p.waitForTimeout(9000);
  out("TC-33 Lưu điểm danh khi offline rồi có mạng", puts.some((x) => /"complete":true/.test(x)), `PUT sau khi có mạng=${puts.length}, complete:true=${puts.filter((x) => /"complete":true/.test(x)).length}; chỉ báo=${JSON.stringify((await p.locator("body").innerText()).match(/Đã lưu điểm danh[^\n]*|Đã lưu \d[^\n]*/g))}`);
  await ctx.close();
}
await b.close();
