import { appendFileSync, readFileSync } from "node:fs";
import { test, type Browser } from "@playwright/test";
import { request } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl";
const D = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/ux60.json", "utf8")) as { a: string };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 380)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 20_000 });
const CID = () => sql(`select id from courses where class_code='QD9G'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
test("UX-S21/S23", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher");
  const urls: string[] = []; page.on("request", (r) => { if (/api\/v1\/courses\/.*(submissions|queue|grading)/.test(r.url()) && r.method() === "GET") urls.push(r.url().replace(/^.*\/api\/v1/, "").replace(/[0-9a-f-]{36}/g, "·").slice(0, 110)); });
  await page.goto(`/grading?tab=queue&assignment=${D.a}`); await page.waitForTimeout(5000);
  const seen = new Set<string>(); const collect = async () => { for (const t of await page.locator("main").getByText(/^QCG\d{5}$/).allInnerTexts()) seen.add(t); };
  await collect(); const first = seen.size; const hdr = (await page.locator("main").innerText()).match(/\d+\+? bài/g);
  for (let i = 0; i < 30; i++) { await page.evaluate(() => { for (const el of Array.from(document.querySelectorAll("*"))) { const e = el as HTMLElement; if (e.scrollHeight > e.clientHeight + 20 && getComputedStyle(e).overflowY !== "visible") e.scrollTop = e.scrollHeight; } window.scrollTo(0, document.body.scrollHeight); }); await page.waitForTimeout(500); await collect(); const b = page.getByRole("button", { name: /Tải thêm|Xem thêm/ }); if (await b.count()) { await b.first().click(); await page.waitForTimeout(800); } }
  const hdr2 = (await page.locator("main").innerText()).match(/\d+\+? bài/g);
  console.log("VERDICT S23", JSON.stringify({ first, seen: seen.size, hdr, hdr2, urls: [...new Set(urls)].slice(0, 8), dbn: sql(`select count(*) from submissions where assignment_id='${D.a}'`)[0][0] }));
  await ctx.close();
});

test("UX-S21b", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher");
  await page.goto(`/grading?tab=queue&assignment=${D.a}`); await page.waitForTimeout(5000);
  const sel = page.getByLabel("Trạng thái"); const v0 = await sel.inputValue().catch(async () => (await sel.innerText()));
  await page.getByRole("button", { name: /Chấm tất cả/ }).first().click(); await page.waitForTimeout(700);
  const dlg = await page.getByRole("alertdialog").or(page.getByRole("dialog")).count(); if (dlg) await page.getByRole("button", { name: /Chấm|Xác nhận|Bắt đầu/ }).last().click().catch(() => {});
  const samples: string[] = []; for (let i = 0; i < 12; i++) { await page.waitForTimeout(1500); const t = (await page.locator("main").innerText()).replace(/\n+/g, " ¦ "); samples.push((t.match(/(Đang chấm|Đã chấm|\d+\/\d+)[^¦]{0,30}/g) ?? []).slice(0, 3).join("|")); }
  await sel.selectOption({ label: "Chưa duyệt" }).catch(() => {}); await page.waitForTimeout(4000); const v1 = await sel.inputValue().catch(async () => (await sel.innerText()));
  console.log("VERDICT S21b", JSON.stringify({ v0, dlg, samples: [...new Set(samples)], v1, graded: sql(`select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}'`)[0][0] }));
  await ctx.close();
});

test("UX-S22", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto("/grading?tab=assignments"); await page.waitForTimeout(5000);
  const row = page.getByRole("row").filter({ hasText: "QC106 ux60" }).first(); const t = (await row.innerText().catch(() => "")).replace(/\n+/g, " ¦ ");
  const src = sql(`select source, count(*) from submissions where assignment_id='${D.a}' group by 1`);
  console.log("VERDICT S22", JSON.stringify({ t, src }));
  await ctx.close();
});
test("UX-S23-375", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher", 375); await page.goto(`/grading?tab=queue&assignment=${D.a}`); await page.waitForTimeout(5000);
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1); const rows = await page.getByRole("row").count(); const small = await page.getByRole("link").or(page.getByRole("button")).evaluateAll((els) => els.filter((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return r.width > 0 && r.height > 0 && r.height < 40 && (e.textContent ?? "").trim().length > 0; }).map((e) => (e.textContent ?? "").trim().slice(0, 14)).slice(0, 8));
  await page.screenshot({ path: `${OUT}/s106-uxs23-375.png` }); console.log("VERDICT S23-375", JSON.stringify({ ov, rows, small })); await ctx.close();
});

test("UX-a", async ({ browser }) => {
  test.setTimeout(280_000);
  const subs = sql(`select s.id, r.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT' and r.run_state='DONE' order by s.created_at limit 21`);
  const { ctx, page } = await pageAs(browser, "teacher"); const st: number[] = []; const reqs: { t: number; m: string; u: string }[] = [];
  page.on("response", (r) => { if (/grading-results\/[0-9a-f-]+(\/review)?$/.test(r.url()) && r.request().method() !== "GET") st.push(r.status()); });
  page.on("request", (r) => { if (/grading-results\/[0-9a-f-]+(\/review)?$/.test(r.url()) && r.method() !== "GET") reqs.push({ t: Date.now(), m: r.method(), u: r.url().split("/").slice(-1)[0] }); });
  let ok = 0, saved = 0, conf = 0; const bad: string[] = [];
  for (let i = 0; i < 20; i++) {
    const [sid, rid] = subs[i]; const txt = `Nhận xét QC ${i} ${Date.now()}`; await page.goto(`/grading/${sid}`); await page.getByLabel("Nhận xét chung").waitFor(); await page.waitForTimeout(600);
    const ta = page.getByLabel("Nhận xét chung"); await ta.fill(txt); await page.getByRole("button", { name: "Duyệt bài", exact: true }).click();
    await page.waitForTimeout(1800); const r = sql(`select status, coalesce(overall_feedback,''), version from grading_results where id='${rid}'`)[0];
    if (r[0] === "REVIEWED") ok++; else bad.push(`${i}:${r[0]}`); if (r[1] === txt) saved++; else bad.push(`${i}:fb=${r[1].slice(0, 20)}`);
  }
  conf = st.filter((x) => x === 409).length; const overlap = reqs.length;
  console.log("VERDICT UX-a", JSON.stringify({ ok, saved, st: [...new Set(st)], conf, nWrites: overlap, bad: bad.slice(0, 5) })); await ctx.close();
});

test("UX-a2", async ({ browser }) => {
  const subs = sql(`select s.id, r.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT' and r.run_state='DONE' order by s.created_at limit 3`);
  const { ctx, page } = await pageAs(browser, "teacher"); const out: string[] = [];
  // Tab → Enter
  { const [sid, rid] = subs[0]; const txt = "Tab Enter " + Date.now(); await page.goto(`/grading/${sid}`); await page.getByLabel("Nhận xét chung").waitFor(); await page.waitForTimeout(600); await page.getByLabel("Nhận xét chung").fill(txt);
    let n = 0; for (; n < 12; n++) { await page.keyboard.press("Tab"); if ((await page.evaluate(() => document.activeElement?.textContent?.trim() ?? "")) === "Duyệt bài") break; } await page.keyboard.press("Enter"); await page.waitForTimeout(1800);
    const r = sql(`select status, overall_feedback from grading_results where id='${rid}'`)[0]; out.push(`TabEnter(${n} Tab): ${r[0]} ${r[1] === txt}`); }
  // blur → sửa điểm → Duyệt liền
  { const [sid, rid] = subs[1]; const v0 = Number(sql(`select version from grading_results where id='${rid}'`)[0][0]); await page.goto(`/grading/${sid}`); await page.getByLabel("Nhận xét chung").waitFor(); await page.waitForTimeout(800);
    const txt = "Blur rồi sửa điểm " + Date.now(); await page.getByLabel("Nhận xét chung").fill(txt); await page.getByLabel("Nhận xét chung").blur(); await page.waitForTimeout(300);
    const inp = page.getByLabel(/^Điểm Xác định/); const cnt = await inp.count(); if (cnt) { await inp.fill("1,5"); await inp.blur(); }
    await page.getByRole("button", { name: "Duyệt bài", exact: true }).click(); await page.waitForTimeout(2500);
    const r = sql(`select status, overall_feedback, version from grading_results where id='${rid}'`)[0]; const sc = sql(`select (select e->>'score'||'/'||(e->>'edited') from jsonb_array_elements(criteria) e where e->>'id'='c1') from grading_results where id='${rid}'`)[0][0];
    out.push(`blur+điểm(${cnt} ô): ${r[0]} fb=${r[1] === txt} version ${v0}→${r[2]} điểm1.50=${sc}`); }
  console.log("VERDICT UX-a2", JSON.stringify(out)); await ctx.close();
});

test("UX-bc", async ({ browser }) => {
  test.setTimeout(200_000);
  const sid = sql(`select s.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT' and r.run_state='DONE' order by s.created_at limit 1`)[0][0];
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=queue&assignment=${D.a}&status=DRAFT`); await page.waitForTimeout(4500);
  await page.getByRole("link", { name: /Duyệt/ }).or(page.getByRole("button", { name: /^Duyệt$/ })).first().click().catch(() => {}); await page.waitForTimeout(2500);
  const seq: string[] = []; const urls: string[] = [];
  for (let i = 0; i < 5; i++) {
    await page.getByLabel("Nhận xét chung").waitFor(); await page.waitForTimeout(500); const head = (await page.locator("main").innerText()).replace(/\n+/g, " ¦ "); seq.push((head.match(/Đã duyệt[^¦]{0,18}|\d+\s*\/\s*\d+[^¦]{0,14}/g) ?? []).slice(0, 2).join(" | ")); urls.push(page.url().replace(/^.*\/grading/, "").replace(/[0-9a-f-]{36}/, "·").slice(0, 60));
    await page.getByRole("button", { name: "Duyệt bài", exact: true }).click(); await page.waitForTimeout(2500);
  }
  const reviewed = sql(`select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='REVIEWED'`)[0][0];
  console.log("VERDICT UX-bc", JSON.stringify({ seq, urls, reviewed })); await page.screenshot({ path: `${OUT}/s106-uxbc.png` }); await ctx.close();
});

test("UX-c2", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=queue&assignment=${D.a}&status=DRAFT`); await page.waitForTimeout(6000);
  const hdr = (await page.locator("main").innerText()).match(/\d+\+? bài/g); const nr = await page.getByRole("row").count();
  const links = await page.getByRole("link", { name: /Duyệt bài của/ }).evaluateAll((els) => els.slice(0, 2).map((e) => (e as HTMLAnchorElement).getAttribute("href")?.replace(/[0-9a-f-]{36}/g, "·")));
  console.log("VERDICT UX-c2", JSON.stringify({ hdr, nr, links, dbDraft: sql(`select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT'`)[0][0] })); await ctx.close();
});

test("UX-b", async ({ browser }) => {
  const rows = sql(`select s.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT' and r.run_state='DONE' order by s.created_at limit 2`);
  const { ctx, page } = await pageAs(browser, "teacher"); const out: string[] = [];
  const idOf = (u: string) => (u.match(/\/grading\/([0-9a-f-]{36})/) ?? [])[1] ?? "";
  // 1) mở bài cụ thể không bộ lọc
  await page.goto(`/grading/${rows[0][0]}`); await page.getByLabel("Nhận xét chung").waitFor(); await page.waitForTimeout(600); await page.getByRole("button", { name: "Duyệt bài", exact: true }).click(); await page.waitForTimeout(3000);
  const u1 = page.url(); const next1 = idOf(u1); const inA = next1 ? sql(`select count(*) from submissions where id='${next1}' and assignment_id='${D.a}'`)[0][0] : "-"; out.push(`bài cụ thể: sau duyệt URL ${u1.replace(/^.*\/\/[^/]+/, "").replace(/[0-9a-f-]{36}/g, "·").slice(0, 70)}; bài kế cùng bài tập ${inA}`);
  // 2) lọc Đã duyệt
  const rv = sql(`select s.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='REVIEWED' order by s.created_at limit 1`)[0][0];
  await page.goto(`/grading/${rv}?n=0&m=3&assignment=${D.a}&status=REVIEWED`); await page.waitForTimeout(2500); const btn = page.getByRole("button", { name: /Duyệt bài|Lưu|Đã duyệt/ }); const t = await page.locator("main").innerText();
  out.push(`lọc Đã duyệt: nút ${JSON.stringify(await btn.allInnerTexts())}; chữ «Đã duyệt bài trước»=${/Đã duyệt bài trước/.test(t)}`);
  console.log("VERDICT UX-b", JSON.stringify(out)); await ctx.close();
});

test("UX-dlj", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); const out: string[] = [];
  await page.goto(`/grading?tab=queue&assignment=${D.a}&status=DRAFT`); await page.waitForTimeout(4000);
  const chip = page.getByRole("button", { name: /Cần xem kỹ/ }); const st = async () => (await chip.getAttribute("aria-pressed")) ?? (await chip.getAttribute("data-active")) ?? (await chip.getAttribute("class"))?.slice(0, 40);
  const s0 = await st(); await chip.click(); await page.waitForTimeout(800); const s1 = await st(); await chip.click(); await page.waitForTimeout(800); const s2 = await st();
  await page.getByRole("tab", { name: "Bài tập" }).click(); await page.waitForTimeout(1500); await page.getByRole("tab", { name: "Hàng chờ" }).click(); await page.waitForTimeout(2500); const s3 = await st();
  await page.goto(`/grading?tab=queue&assignment=${D.a}&status=DRAFT`); await page.waitForTimeout(3500); const s4 = await st();
  out.push(`d: ban đầu ${s0}; bật ${s1}; tắt ${s2}; đổi tab rồi quay lại ${s3}; vào lại bằng URL ${s4}`);
  await page.goto(`/grading?tab=queue&assignment=${D.a}&status=REVIEWED`); await page.waitForTimeout(4000); const pub = page.getByRole("button", { name: "Công bố", exact: true }).first();
  out.push(`l: nút Công bố thấy ${await pub.isVisible()}, bật ${await pub.isEnabled()}, title ${await pub.getAttribute("title")}, aria-describedby ${await pub.getAttribute("aria-describedby")}, aria-disabled ${await pub.getAttribute("aria-disabled")}`);
  await pub.hover().catch(() => {}); await page.waitForTimeout(700); out.push(`l tooltip: ${((await page.getByRole("tooltip").allInnerTexts().catch(() => [])) as string[]).join("|")}`); await page.screenshot({ path: `${OUT}/s106-uxl.png` });
  await page.goto("/grading?tab=assignments"); await page.waitForTimeout(4500); const row = page.getByRole("row").filter({ hasText: "QC106 ux60" }).first(); const href = await row.locator("a[href]").first().getAttribute("href").catch(() => null); const asRow = await row.getAttribute("role"); const cls = await row.getAttribute("href");
  out.push(`j: hàng có <a href> ${href?.replace(/[0-9a-f-]{36}/g, "·")}; thuộc tính href trên hàng ${cls}`);
  console.log("VERDICT UX-dlj", JSON.stringify(out)); await ctx.close();
  const m = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, locale: "vi-VN" }); await m.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID()); const p2 = await m.newPage();
  await p2.addLocatorHandler(p2.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); }); await p2.goto("/login"); await p2.locator("input[type=email]").fill("teacher@edupilot.local"); await p2.locator("input[type=password]").fill(PW); await p2.keyboard.press("Enter"); await p2.waitForURL((u) => !u.pathname.startsWith("/login")); await p2.goto("/grading?tab=assignments"); await p2.waitForTimeout(4500);
  const l2 = p2.locator("a[href*='open=']").first(); const all2 = await p2.locator("a[href*='open=']").count(); const bb = await p2.locator("a[href*='open=']").evaluateAll((els) => els.map((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return [Math.round(r.width), Math.round(r.height)]; }).filter((x) => x[0] > 0).slice(0, 4)); const card = await p2.locator("a[href*='open=']").first().evaluate((a) => { const c = a.closest("li, [role=row], tr, article, div[class]") as HTMLElement; const r = c.getBoundingClientRect(); return { x: r.x, y: r.y, w: r.width, h: r.height, tag: c.tagName }; }); let navd = "-"; { await p2.mouse.click(card.x + card.w - 12, card.y + card.h - 8); await p2.waitForTimeout(1500); navd = p2.url().replace(/^.*grading/, "").replace(/[0-9a-f-]{36}/, "·"); } console.log("VERDICT UX-j375c", JSON.stringify({ card, navd }));
  console.log("VERDICT UX-j375", JSON.stringify({ box: bb, all2, ov: await p2.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1) })); await m.close();
});

test("UX-d2", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); const out: string[] = [];
  await page.goto(`/grading?tab=queue`); await page.waitForTimeout(4000);
  const chip = page.getByRole("button", { name: /Cần xem kỹ/ }); const st = async () => await chip.getAttribute("aria-pressed");
  out.push("vào hàng chờ: " + (await st()));
  for (let r = 0; r < 2; r++) { const a = await st(); if (a === "true") { await chip.click(); await page.waitForTimeout(700); } out.push(`vòng ${r}: tắt=${await st()}`); await page.goto("/grading?tab=assignments"); await page.waitForTimeout(2000); await page.goto("/grading?tab=queue"); await page.waitForTimeout(3000); out.push(`vòng ${r}: sau khi rời/vào lại (goto) = ${await st()}`);
    await page.getByRole("tab", { name: "Bài tập" }).click(); await page.waitForTimeout(1500); await page.getByRole("tab", { name: "Hàng chờ" }).click(); await page.waitForTimeout(2500); out.push(`vòng ${r}: sau đổi tab = ${await st()}; url ${page.url().replace(/^.*grading/, "").slice(0, 80)}`); }
  console.log("VERDICT UX-d2", JSON.stringify(out)); await ctx.close();
});

test("UX-g", async ({ browser }) => {
  const aid = sql(`select id from assignments where title='QC106 g04ui' order by created_at desc limit 1`)[0][0]; const out: string[] = [];
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=assignments&open=${aid}`); await page.waitForTimeout(5500);
  const panel = page.locator("[data-part=guidance-panel]"); const items = panel.locator("[data-part=guidance-item]");
  const draft = items.filter({ hasText: "Chờ duyệt" }).first(); const iid0 = await draft.count(); const txt0 = (await draft.innerText()).split("\n")[0];
  await draft.getByRole("button", { name: "Duyệt", exact: true }).click(); await page.waitForTimeout(600);
  const undo1 = await page.getByRole("button", { name: /Hoàn tác/ }).count(); const toast = await page.getByText(/Thành công/).count(); await page.waitForTimeout(5800); const undo2 = await page.getByRole("button", { name: /Hoàn tác/ }).count();
  const reset = panel.getByRole("button", { name: "Đặt lại" }); const nreset = await reset.count(); const inMenu = await panel.getByRole("button", { name: /Thêm|Khác|⋯|…/ }).count();
  out.push(`Duyệt «${txt0.slice(0, 40)}»: Hoàn tác lúc 0,6 s = ${undo1}, sau 6,4 s = ${undo2}; toast «Thành công» ${toast}; nút «Đặt lại» thấy được ×${nreset} (menu ⋯ ×${inMenu})`);
  console.log("VERDICT UX-g", JSON.stringify(out)); await page.screenshot({ path: `${OUT}/s106-uxg.png` }); await ctx.close();
});

test("UX-g2", async ({ browser }) => {
  const aid = sql(`select id from assignments where title='QC106 g04ui' order by created_at desc limit 1`)[0][0];
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=assignments&open=${aid}`); await page.waitForTimeout(5500);
  const it = page.locator("[data-part=guidance-item]").filter({ hasText: "Cần nêu rõ tác nhân" }).first(); const before = sql(`select status, text, text_original from grading_guidance_items where assignment_id='${aid}' and text like 'Cần nêu rõ tác nhân%'`)[0];
  await it.getByRole("button", { name: "Đặt lại" }).click(); await page.waitForTimeout(2500); const after = sql(`select status, text from grading_guidance_items where assignment_id='${aid}' and text like 'Cần nêu rõ tác nhân%'`)[0]; const toast = await page.getByText(/Thành công/).count();
  console.log("VERDICT UX-g2", JSON.stringify({ before: before.map((x: string) => x.slice(0, 30)), after: after.map((x: string) => x.slice(0, 30)), toast })); await ctx.close();
});

test("UX-h", async ({ browser }) => {
  const aid = sql(`select id from assignments where title='QC106 g04ui' order by created_at desc limit 1`)[0][0];
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=assignments&open=${aid}`); await page.waitForTimeout(5500);
  const t = (await page.locator("[data-part=guidance-panel]").innerText()).replace(/\n+/g, " ¦ "); const lines = t.match(/Bản v\d[^¦]*(¦[^¦]*)?/g) ?? []; const tips = await page.locator("[data-part=guidance-panel] [role=tooltip], [data-part=guidance-panel] [title]").count();
  const sqlv = sql(`select r.guidance_version, count(*) filter (where r.status in ('REVIEWED','PUBLISHED')) n, count(*) filter (where r.status in ('REVIEWED','PUBLISHED') and exists (select 1 from jsonb_array_elements(r.criteria) e where e->>'edited'='true')) ed from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${aid}' group by 1 order by 1`);
  console.log("VERDICT UX-h", JSON.stringify({ lines, tips, sqlv })); await ctx.close();
});

test("UX-f", async ({ browser }) => {
  const aid = sql(`select id from assignments where title='QC106 g04ui' order by created_at desc limit 1`)[0][0];
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=assignments&open=${aid}`); await page.waitForTimeout(5500);
  const it = page.locator("[data-part=guidance-item]").filter({ hasText: "Chờ duyệt" }).first(); const orig = (await it.innerText()).split("\n")[0];
  await it.getByRole("button", { name: "Sửa", exact: true }).click(); await page.waitForTimeout(500); const ta = it.locator("textarea").or(page.locator("textarea").last()).first(); const newTxt = "Chữ đã sửa QC " + Date.now(); await ta.fill(newTxt);
  await it.getByRole("button", { name: "Duyệt", exact: true }).click().catch(async () => { await page.getByRole("button", { name: /Lưu|Duyệt/ }).last().click(); });
  const seen: string[] = []; for (let i = 0; i < 8; i++) { await page.waitForTimeout(400); seen.push((await page.locator("[data-part=guidance-panel]").innerText()).includes(newTxt) ? "mới" : (await page.locator("[data-part=guidance-panel]").innerText()).includes(orig.slice(0, 25)) ? "GỐC" : "-"); }
  await page.reload(); await page.waitForTimeout(5000); const after = (await page.locator("[data-part=guidance-panel]").innerText()).includes(newTxt); const db = sql(`select status, text, left(text_original, 40) from grading_guidance_items where assignment_id='${aid}' and text like 'Chữ đã sửa QC%'`);
  console.log("VERDICT UX-f", JSON.stringify({ seen, afterReload: after, db: db.map((r: string[]) => [r[0], r[1].slice(0, 24), r[2]]) })); await ctx.close();
});

test("UX-m", async ({ browser }) => {
  const aid = sql(`select id from assignments where title='QC106 g12b' order by created_at desc limit 1`)[0][0];
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=assignments&open=${aid}`); await page.waitForTimeout(5500);
  const panel = page.locator("[data-part=guidance-panel]"); const n0 = await panel.locator("[data-part=guidance-item]").count();
  await page.getByRole("button", { name: "Rút lưu ý" }).click(); const seen: string[] = [];
  for (let i = 0; i < 24; i++) { await page.waitForTimeout(250); const b = page.getByRole("button", { name: /Rút lưu ý|Đang rút/ }).first(); const bt = ((await b.innerText().catch(() => "")) + "|" + (await b.isDisabled().catch(() => false))); const line = ((await panel.innerText()).match(/Đã rút[^¦\n]*|Đang[^¦\n]*|bị loại[^¦\n]*/g) ?? []).join(","); seen.push(bt + (line ? " / " + line : "")); }
  const n1 = await panel.locator("[data-part=guidance-item]").count(); await page.goto("/grading?tab=queue"); await page.waitForTimeout(2500); await page.goto(`/grading?tab=assignments&open=${aid}`); await page.waitForTimeout(5000); const n2 = await page.locator("[data-part=guidance-panel] [data-part=guidance-item]").count();
  console.log("VERDICT UX-m", JSON.stringify({ n0, n1, n2, seen: [...new Set(seen)] })); await ctx.close();
});

test("UX-o", async ({ browser }) => {
  test.setTimeout(150_000);
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=queue&assignment=${D.a}&status=DRAFT`); await page.waitForTimeout(5000);
  const hdr = async () => ((await page.locator("main").innerText()).match(/\d+\+? bài/) ?? [""])[0]; const h0 = await hdr();
  const api = await request.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true }); const lg = await api.post("/api/v1/auth/login", { data: { email: "ta@edupilot.local", password: PW } }); const tok = (await lg.json()).access_token ?? (await lg.json()).token;
  const rows = sql(`select r.id, r.version from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT' and r.run_state='DONE' order by s.created_at limit 2`);
  const cid = sql(`select course_id from assignments where id='${D.a}'`)[0][0]; const codes: number[] = [];
  for (const [rid, ver] of rows) { const r = await api.post(`/api/v1/courses/${cid}/grading-results/${rid}/review`, { headers: { Authorization: `Bearer ${tok}` }, data: { version: Number(ver) } }); codes.push(r.status()); }
  await page.waitForTimeout(12000); const h1 = await hdr(); // không thao tác: SSE?
  await page.evaluate(() => { const set = (v: string) => Object.defineProperty(document, "visibilityState", { value: v, configurable: true }); set("hidden"); document.dispatchEvent(new Event("visibilitychange")); set("visible"); document.dispatchEvent(new Event("visibilitychange")); window.dispatchEvent(new Event("focus")); }); await page.waitForTimeout(4000); const h2 = await hdr();
  const dbn = sql(`select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT'`)[0][0];
  console.log("VERDICT UX-o", JSON.stringify({ ta: codes, h0, h1_after12s: h1, h2_after_focus: h2, dbDraft: dbn })); await ctx.close();
});

test("UX-17", async ({ browser }) => {
  const aid = sql(`select id from assignments where title='QC106 g04ui' order by created_at desc limit 1`)[0][0]; const sid = sql(`select s.id from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${D.a}' and r.status='DRAFT' and r.run_state='DONE' limit 1`)[0][0];
  const { ctx, page } = await pageAs(browser, "teacher", 375); const out: string[] = [];
  for (const [name, url] of [["hàng chờ", `/grading?tab=queue&assignment=${D.a}&status=DRAFT`], ["màn duyệt", `/grading/${sid}`], ["ngăn lưu ý", `/grading?tab=assignments&open=${aid}`]]) {
    await page.goto(url); await page.waitForTimeout(5000); const ov = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
    const small = await page.locator("button:visible, a:visible").evaluateAll((els) => els.filter((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return r.width > 0 && r.height > 0 && r.height < 44 && (e.textContent ?? "").trim().length > 0 && r.top < 2000; }).map((e) => `${(e.textContent ?? "").trim().slice(0, 16)}:${Math.round((e as HTMLElement).getBoundingClientRect().height)}`).slice(0, 8));
    out.push(`${name}: tràn ngang ${ov}; nút/liên kết cao <44 px: ${JSON.stringify(small)}`); await page.screenshot({ path: `${OUT}/s106-ux17-${name.replace(/ /g, "")}.png` }); }
  console.log("VERDICT UX-17", JSON.stringify(out)); await ctx.close();
});

test("UX-j2", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher", 375); await page.goto("/grading?tab=assignments"); await page.waitForTimeout(5000); const out: string[] = [];
  const a = page.locator("a[href*='open=']:visible").first(); await a.scrollIntoViewIfNeeded(); const b = (await a.boundingBox())!; const href = await a.getAttribute("href");
  const info = await a.evaluate((e) => { const cs = getComputedStyle(e, "::after"); return { after: cs.content, pos: cs.position, w: cs.width, h: cs.height }; });
  const hits = await page.evaluate(([x, y]) => { const el = document.elementFromPoint(x as number, y as number); return el ? `${el.tagName}.${(el as HTMLElement).className.toString().slice(0, 30)} ${(el.closest("a") as HTMLAnchorElement | null)?.getAttribute("href") ?? "-"}` : "none"; }, [200, b.y + b.height / 2 + 14] as [number, number]);
  out.push(`liên kết ${Math.round(b.width)}x${Math.round(b.height)} tại y=${Math.round(b.y)}; ::after ${JSON.stringify(info)}; điểm chạm cách 14 px dưới / x=200: ${hits}`);
  await page.mouse.click(200, b.y + b.height / 2 + 14); await page.waitForTimeout(1500); out.push("sau chạm: " + page.url().replace(/^.*grading/, "").replace(/[0-9a-f-]{36}/, "·"));
  console.log("VERDICT UX-j2", JSON.stringify(out)); await ctx.close();
});
