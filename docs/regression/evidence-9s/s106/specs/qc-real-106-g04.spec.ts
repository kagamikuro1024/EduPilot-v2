import { appendFileSync } from "node:fs";
import { test, type Browser, type Page } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
// QC smoke 10.6 (GD-04 UI) trên ep-qc, lớp QD9G. QC_AID = bài tập có đủ trạng thái; QC_AID2 = bài tập chưa có lưu ý (cờ bật).
const BASE = process.env.QC_BASE ?? "https://localhost:943";
const RES = process.env.QC_RES ?? `${OUT}/results_106.jsonl`;
const AID = process.env.QC_AID ?? "", AID2 = process.env.QC_AID2 ?? "";
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 330)); };
test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
test.use({ actionTimeout: 20_000 });
const CID = () => sql(`select id from courses where class_code='QD9G'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: BASE, ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage();
  await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page };
}
const panel = (p: Page) => p.locator("[data-part=guidance-panel]");
const open = async (p: Page, aid = AID) => { await p.goto(`/grading?tab=assignments&open=${aid}`); await p.waitForTimeout(4500); };
const openSV = async (p: Page) => { await p.goto(`/assignments/${AID}`); await p.waitForTimeout(3500); };
const BAD = /\b(AI|RAG|PII|nháp|prompt|LLM)\b/i;

test("TC-G04-01 khối «Lưu ý chấm» dưới rubric, dòng bản + Rút lưu ý phụ", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page);
  const n = await panel(page).count(); const t = n ? await panel(page).innerText() : "";
  const order = await page.evaluate(() => { const p = document.querySelector("[data-part=guidance-panel]"); const all = [...document.querySelectorAll("main *")]; const rub = all.find((e) => /Tiêu chí|rubric/i.test(e.textContent ?? "") && e.children.length === 0); return { panelAfterRubric: !!(p && rub && (rub.compareDocumentPosition(p) & Node.DOCUMENT_POSITION_FOLLOWING)) }; });
  const bt = await panel(page).getByRole("button", { name: "Rút lưu ý" }).evaluate((e) => getComputedStyle(e).backgroundColor).catch(() => "");
  const ap = await panel(page).getByRole("button", { name: /^Áp dụng/ }).evaluate((e) => getComputedStyle(e).backgroundColor).catch(() => "");
  const head = t.split("\n").slice(0, 4).join(" ¦ "); await page.screenshot({ path: `${OUT}/s106-g0401.png`, fullPage: true });
  verdict("TC-G04-01", n === 1 && /Bản v\d/.test(t) && /\d+ chờ duyệt/.test(t) && order.panelAfterRubric && bt !== ap, `cờ bật: ${n} khối; đầu khối «${head}»; có «N chờ duyệt» ${/\d+ chờ duyệt/.test(t)}; khối nằm sau rubric ${order.panelAfterRubric}; nền Rút lưu ý ${bt} vs Áp dụng ${ap}`);
  await ctx.close();
});
test("TC-G04-01b cờ tắt → khối ẩn", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page); const n = await panel(page).count(); const t = await page.locator("main").innerText();
  verdict("TC-G04-01", n === 0 && !/Lưu ý chấm/.test(t), `[cờ tắt] khối guidance-panel=${n}; chữ «Lưu ý chấm» trong màn=${/Lưu ý chấm/.test(t)}`); await ctx.close();
});
test("TC-G04-02 danh sách mục: sáu trạng thái, ngăn 1 px", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page);
  const items = panel(page).locator("[data-part=guidance-item]"); const n = await items.count(); const txts = await items.allInnerTexts();
  const st = await items.evaluateAll((els) => els.map((e) => { const c = getComputedStyle(e); return { bt: c.borderTopWidth, bb: c.borderBottomWidth, rad: c.borderTopLeftRadius, sh: c.boxShadow !== "none", st: e.getAttribute("data-status") }; }));
  const all = txts.join("\n"); const labels = ["Chờ duyệt", "Chờ áp dụng", "Đã áp dụng", "Chờ nghỉ", "Đã bỏ", "Đã nghỉ"]; const has = labels.map((l) => `${l}=${all.includes(l)}`);
  const drafts = txts.filter((x) => /Chờ duyệt/.test(x)); const d0 = drafts[0] ?? "";
  await page.screenshot({ path: `${OUT}/s106-g0402.png`, fullPage: true });
  verdict("TC-G04-02", labels.every((l) => all.includes(l)) && /lần sửa/.test(d0) && /(Gắt hơn|Nhẹ hơn|Làm rõ)/.test(d0) && /Duyệt/.test(d0) && /Bỏ/.test(d0), `${n} mục hiển thị (DB có 3 Chờ duyệt, 1 Chờ áp dụng, 1 Đang dùng v2, 1 Chờ nghỉ, 1 Đã bỏ, 1 Đã nghỉ); nhãn trạng thái: ${has.join(", ")}; mục Chờ duyệt: «${d0.replace(/\n/g, " ¦ ").slice(0, 150)}»; có «lần sửa» ${/lần sửa/.test(d0)}; ngăn: ${JSON.stringify(st.slice(0, 2))}; trạng thái thật trên UI: ${JSON.stringify([...new Set(txts.map((x) => (x.match(/Chờ duyệt|Chờ áp dụng|Đang dùng · v\d|Đã áp dụng|Chờ nghỉ|Đã bỏ|Đã nghỉ/) ?? ["?"])[0]))])}`);
  await ctx.close();
});
test("TC-G04-12 TA: có danh sách, không Duyệt/Bỏ/Áp dụng", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "ta"); await open(page); const n = await panel(page).count(); const items = await panel(page).locator("[data-part=guidance-item]").count();
  const btn = await panel(page).getByRole("button").allInnerTexts(); const bad = btn.filter((b) => /^(Duyệt|Bỏ|Áp dụng|Cho nghỉ|Rút lưu ý|Sửa|Đặt lại)/.test(b.trim()));
  await page.screenshot({ path: `${OUT}/s106-g0412.png`, fullPage: true });
  verdict("TC-G04-12", n === 1 && items > 0 && bad.length === 0, `TA: khối=${n}, ${items} mục; nút trong khối: ${JSON.stringify(btn)}; nút cấm: ${JSON.stringify(bad)}`); await ctx.close();
});
test("TC-G04-18 SV không thấy gì về lưu ý", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "sv.kha"); const reqs: string[] = []; page.on("request", (r) => { if (/guidance|regrade|reflect/.test(r.url())) reqs.push(r.method() + " " + r.url().replace(/^.*\/api\/v1/, "")); });
  await openSV(page); const n = await panel(page).count(); const html = await page.content(); const t = await page.locator("body").innerText();
  verdict("TC-G04-18", n === 0 && reqs.length === 0 && !/Lưu ý chấm|guidance/i.test(html.replace(/_next\/[^"']*/g, "")) && !/Lưu ý chấm/.test(t), `SV mở /assignments/{id}: khối=${n}; request guidance/regrade/reflect=${reqs.length} ${JSON.stringify(reqs.slice(0, 3))}; «Lưu ý chấm» trong chữ=${/Lưu ý chấm/.test(t)}; «guidance» trong DOM=${/guidance/i.test(html.replace(/_next\/[^"']*/g, ""))}; url ${page.url().replace(BASE, "")}`); await ctx.close();
});
test("TC-G04-24+14 Áp dụng: hộp xác nhận, chỉ POST sau xác nhận, không Hoàn tác; apply trả stale; /grading hiện bài bản cũ", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page); const posts: string[] = []; let applyBody: any = null;
  page.on("request", (r) => { if (r.method() === "POST" && /guidance\/apply/.test(r.url())) posts.push(new Date().toISOString()); });
  page.on("response", async (r) => { if (r.request().method() === "POST" && /guidance\/apply/.test(r.url())) applyBody = await r.json().catch(() => null); });
  const before = sql(`select guidance_version from assignments where id='${AID}'`)[0][0];
  await panel(page).getByRole("button", { name: /^Áp dụng/ }).click(); await page.waitForTimeout(800);
  const dlg = page.getByRole("alertdialog").or(page.getByRole("dialog")).last(); const dt = (await dlg.innerText().catch(() => "")).replace(/\n/g, " ¦ "); const preCount = posts.length; await page.screenshot({ path: `${OUT}/s106-g0424-dlg.png` });
  const waiting = sql(`select count(*) filter (where approved_in_version is null and status='APPROVED' and not pending_retire), count(*) filter (where pending_retire) from grading_guidance_items where assignment_id='${AID}'`)[0];
  await dlg.getByRole("button", { name: "Áp dụng" }).click(); await page.waitForTimeout(2500);
  const after = sql(`select guidance_version from assignments where id='${AID}'`)[0][0]; const pt = await panel(page).innerText(); const undo = await panel(page).getByText("Hoàn tác").count(); const toast = await page.getByRole("status").filter({ hasText: "Thành công" }).count();
  // đếm lại bằng SQL
  const cnt = sql(`select count(*) filter (where r.status='DRAFT' and r.run_state='DONE' and not exists (select 1 from jsonb_array_elements(r.criteria) e where (e->>'edited')::boolean)), count(*) filter (where r.status='DRAFT' and r.run_state='DONE' and exists (select 1 from jsonb_array_elements(r.criteria) e where (e->>'edited')::boolean)), count(*) filter (where r.status='REVIEWED') from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${AID}' and r.guidance_version < ${after}`)[0];
  const stale = applyBody?.stale ?? {};
  await panel(page).screenshot({ path: `${OUT}/s106-g0424-after.png` });
  verdict("TC-G04-24", /Áp dụng bản v\d+\?/.test(dt) && /Thêm \d+ lưu ý, cho nghỉ \d+/.test(dt) && preCount === 0 && posts.length === 1 && Number(after) === Number(before) + 1 && undo === 0 && toast === 0, `bấm «Áp dụng»: hộp «${dt.slice(0, 160)}»; POST apply trước xác nhận=${preCount}, sau xác nhận=${posts.length}; DB chờ áp dụng ${waiting[0]} / chờ nghỉ ${waiting[1]}; guidance_version ${before}→${after}; dòng «Hoàn tác»=${undo}; toast «Thành công»=${toast}; chữ sau áp dụng: «${pt.split("\n").slice(0, 3).join(" ¦ ")}»`);
  await ctx.close();
});
test("TC-G04-14 /grading hiện «N bài chấm theo bản cũ · Chấm lại» ngay sau apply; apply trả stale đúng SQL", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); const after = sql(`select guidance_version from assignments where id='${AID}'`)[0][0];
  const stale = JSON.parse(process.env.QC_STALE ?? "{}");
  const cnt = sql(`select count(*) filter (where r.status='DRAFT' and r.run_state='DONE' and not exists (select 1 from jsonb_array_elements(r.criteria) e where (e->>'edited')::boolean)), count(*) filter (where r.status='DRAFT' and r.run_state='DONE' and exists (select 1 from jsonb_array_elements(r.criteria) e where (e->>'edited')::boolean)), count(*) filter (where r.status='REVIEWED') from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${AID}' and r.guidance_version < ${after}`)[0];
  await page.goto(`/grading?assignment=${AID}`); await page.waitForTimeout(4000);
  const sel = page.locator("select").first(); let gtxt = "";
  if (false) { const opts = await sel.locator("option").allInnerTexts(); const k = opts.findIndex((o) => o.includes("QC106 g04ui")); console.log("opts", JSON.stringify(opts.slice(0, 12))); if (k >= 0) await sel.selectOption({ index: k }); await page.waitForTimeout(3000); }
  gtxt = await page.locator("main").innerText(); const m = gtxt.match(/(\d+) bài chấm theo bản cũ[^\n]*/); const regrade = await page.getByRole("button", { name: "Chấm lại" }).count();
  await page.screenshot({ path: `${OUT}/s106-g0414.png` }); const jobs = sql(`select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='${AID}' and r.run_state in ('QUEUED','RUNNING')`)[0][0];
  verdict("TC-G04-14", stale.count === Number(cnt[0]) && stale.editing === Number(cnt[1]) && stale.reviewed === Number(cnt[2]) && !!m && Number(m[1]) === Number(cnt[0]) && regrade > 0 && jobs === "0", `apply trả stale=${JSON.stringify(stale)}; SQL đếm lại: nháp chưa sửa ${cnt[0]}, nháp đã sửa ${cnt[1]}, đã duyệt ${cnt[2]}; /grading: «${m?.[0] ?? "không thấy dòng bản cũ"}», nút Chấm lại ×${regrade}; không tự chấm lại (kết quả QUEUED/RUNNING=${jobs})`); await ctx.close();
});

const sizes = (p: Page) => panel(p).locator("button").evaluateAll((els) => els.map((e) => { const r = e.getBoundingClientRect(); return { t: (e.textContent || "").trim().slice(0, 12), w: Math.round(r.width), h: Math.round(r.height), x: Math.round(r.x), y: Math.round(r.y) }; }).filter((b) => b.w > 0));
const overflow = (p: Page) => p.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);
test("TC-G04-16 375 px: xếp dọc, nút ≥ 44 px, không tràn ngang", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher", 375); await open(page); const n = await panel(page).count();
  const b = await sizes(page); const small = b.filter((x) => x.h < 44 || x.w < 44); const ov = await overflow(page);
  const pr = await panel(page).boundingBox(); const vert = await panel(page).locator("[data-part=guidance-item]").first().evaluate((e) => { const k = e.querySelectorAll("p, button"); const a = k[0].getBoundingClientRect(), z = k[k.length - 1].getBoundingClientRect(); return z.top >= a.bottom - 1; }).catch(() => false);
  await panel(page).screenshot({ path: `${OUT}/s106-g0416.png` });
  verdict("TC-G04-16", n === 1 && small.length === 0 && !ov && !!pr && pr.width <= 375 && vert, `375 px: khối=${n} rộng ${pr?.width}; ${b.length} nút, nhỏ hơn 44 px: ${JSON.stringify(small)}; tràn ngang ${ov}; chữ mục trên, nút dưới (xếp dọc) ${vert}`); await ctx.close();
});
test("TC-G04-28 375 px: Duyệt → «Hoàn tác» vùng chạm ≥ 44 px, không tràn, không che nút kế bên", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher", 375); await open(page);
  const it0 = panel(page).locator("[data-part=guidance-item][data-status=DRAFT]").first(); const key = (await it0.locator("p").first().innerText()).slice(0, 25); await it0.getByRole("button", { name: "Duyệt" }).click(); await page.waitForTimeout(1200);
  const it = panel(page).locator("[data-part=guidance-item]").filter({ hasText: key }).first(); const undo = it.getByRole("button", { name: "Hoàn tác" }); const ub = await undo.boundingBox(); const others = await it.locator("button").evaluateAll((els) => els.map((e) => { const r = e.getBoundingClientRect(); return { t: (e.textContent || "").trim().slice(0, 10), x: r.x, y: r.y, w: r.width, h: r.height }; }));
  const ov = await overflow(page); const hit = ub && (await page.evaluate(([x, y]) => { const e = document.elementFromPoint(x, y); return (e?.textContent || "").trim().slice(0, 12); }, [ub.x + ub.width / 2, ub.y + ub.height / 2]));
  const inter = ub ? others.filter((o) => o.t !== "Hoàn tác" && !(o.x + o.w <= ub.x || ub.x + ub.width <= o.x || o.y + o.h <= ub.y || ub.y + ub.height <= o.y)) : [];
  await it.screenshot({ path: `${OUT}/s106-g0428.png` });
  await undo.click().catch(() => {}); await page.waitForTimeout(1500);
  verdict("TC-G04-28", !!ub && ub.height >= 44 && ub.width >= 44 && !ov && inter.length === 0 && hit === "Hoàn tác", `375 px sau «Duyệt»: «Hoàn tác» ${ub ? Math.round(ub.width) + "×" + Math.round(ub.height) : "không thấy"}; tràn ngang ${ov}; giao với nút khác: ${JSON.stringify(inter.map((x) => x.t))}; điểm giữa chạm trúng «${hit}»`); await ctx.close();
});
test("TC-G04-27 offline khi Hoàn tác: giữ trạng thái, báo tại chỗ + Thử lại, không gửi hai lần", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page); const resets: string[] = [];
  page.on("response", async (r) => { if (r.request().method() === "POST" && /items\/[^/]+\/decide/.test(r.url())) { const b = r.request().postData() ?? ""; if (b.includes("RESET")) resets.push(String(r.status())); } });
  const it = panel(page).locator("[data-part=guidance-item][data-status=DRAFT]").first(); const txt0 = (await it.locator("p").first().innerText()).slice(0, 30);
  await it.getByRole("button", { name: "Duyệt" }).click(); await page.waitForTimeout(1500);
  const itA = panel(page).locator("[data-part=guidance-item]").filter({ hasText: txt0 }).first(); const st1 = await itA.getAttribute("data-status"); const s1 = (await itA.innerText()).includes("Chờ áp dụng");
  await ctx.setOffline(true); await itA.getByRole("button", { name: "Hoàn tác" }).click(); await page.waitForTimeout(2500);
  const body = (await itA.innerText()).replace(/\n/g, " ¦ "); const retry = await panel(page).getByRole("button", { name: "Thử lại" }).count(); const keep = body.includes("Chờ áp dụng"); await page.screenshot({ path: `${OUT}/s106-g0427.png` });
  const during = resets.length; await ctx.setOffline(false); await page.waitForTimeout(500);
  const rb = panel(page).getByRole("button", { name: "Thử lại" }).or(itA.getByRole("button", { name: "Hoàn tác" })).first(); await rb.click(); await page.waitForTimeout(2500);
  const dbs = sql(`select status from grading_guidance_items where assignment_id='${AID}' and text like '${txt0.replace(/'/g, "''")}%' limit 1`)[0][0];
  verdict("TC-G04-27", st1 === "APPROVED" && s1 && keep && retry + (body.includes("Thử lại") ? 1 : 0) > 0 && during === 0 && dbs === "DRAFT" && resets.filter((x) => x === "200").length === 1, `Duyệt → «Chờ áp dụng»; offline + Hoàn tác: mục vẫn «Chờ áp dụng» ${keep}; nút «Thử lại» ×${retry}; chữ mục: «${body.slice(0, 200)}»; RESET tới máy chủ khi offline: ${during}; có mạng + Thử lại → DB status=${dbs}; số RESET thành công=${resets.filter((x) => x === "200").length} (mọi mã: ${JSON.stringify(resets)})`); await ctx.close();
});
test("TC-G04-15 rỗng một dòng; GET 500 → PageState + Thử lại", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page, AID2); const pt = (await panel(page).innerText()).replace(/\n/g, " ¦ "); const emp = await panel(page).locator("[data-part=empty-state], [role=status]").count();
  await page.route(/\/guidance(\?.*)?$/, (r) => (r.request().method() === "GET" ? r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Đã xảy ra lỗi." }) }) : r.continue()));
  await page.goto(`/grading?tab=assignments&open=${AID}`); await page.waitForTimeout(5000); const pn = await panel(page).count(); const drawer = await page.locator("[data-part=assignment-drawer], [role=dialog]").first().innerText().catch(() => ""); const retryN = await page.getByRole("button", { name: "Thử lại" }).count(); await page.screenshot({ path: `${OUT}/s106-g0415-500.png` });
  verdict("TC-G04-15", emp > 0 && retryN > 0, `AID2 chưa có lưu ý: khối chỉ có «${pt.slice(0, 120)}», phần tử EmptyState/status=${emp} (không có dòng «Chưa có lưu ý»); chặn GET /guidance → 500: khối guidance-panel=${pn} (biến mất, không PageState), nút «Thử lại» ×${retryN}; ngăn bài tập: «${drawer.replace(/\n/g, " ¦ ").slice(0, 120)}»`); await ctx.close();
});
test("TC-G04-15c chữ sửa còn sau tải lại", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page); const it = panel(page).locator("[data-part=guidance-item][data-status=DRAFT]").first(); const key = (await it.locator("p").first().innerText()).slice(0, 25);
  await it.getByRole("button", { name: "Sửa" }).click(); await page.waitForTimeout(500); await panel(page).getByRole("textbox").first().fill("Chữ sửa dở của QC106 giữ sau tải lại."); await page.waitForTimeout(1500);
  await page.reload(); await page.waitForTimeout(5000); const it2 = panel(page).locator("[data-part=guidance-item]").filter({ hasText: key }).first(); const shownRaw = await it2.innerText().catch(() => ""); await it2.getByRole("button", { name: "Sửa" }).click(); await page.waitForTimeout(500); const v = await it2.getByRole("textbox").inputValue().catch(() => "?");
  console.log("VERDICT-INFO", "TC-G04-15c", key, "→", JSON.stringify(v), JSON.stringify(shownRaw.slice(0, 60))); await ctx.close();
});
test("TC-G04-17 quét chữ + icon", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await open(page); const t = await panel(page).innerText(); const icons = await panel(page).locator("svg").evaluateAll((e) => e.map((x) => x.getAttribute("class") ?? ""));
  const info = icons.filter((c) => /info|circle-help|question/i.test(c)); const bad = t.match(BAD); const desc = await panel(page).locator("h3 ~ p, h3 + p").allInnerTexts();
  const tips = await panel(page).locator("[title]").evaluateAll((e) => e.map((x) => x.getAttribute("title")));
  verdict("TC-G04-17", !bad && info.length === 0 && desc.length === 0, `chữ khối: từ cấm ${bad?.[0] ?? "không"}; icon Info/Help ${info.length} (tổng svg ${icons.length}); dòng mô tả ngay dưới tiêu đề ${desc.length}; title/tooltip: ${JSON.stringify(tips)}; (script ui-antipatterns chạy riêng)`); await ctx.close();
});
