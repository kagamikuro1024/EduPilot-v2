import { appendFileSync, readFileSync } from "node:fs";
import { test, request, type Browser } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
import { writeFileSync } from "node:fs";
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl";
const D = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/ux60.json", "utf8")) as { a: string };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 380)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 20_000 });
const CID = () => sql(`select id from courses where class_code='QD105A'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
test("D06", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin"); const out: string[] = []; const reqs: string[] = [];
  page.on("response", (r) => { if (/admin\/llm\/(decision|providers)/.test(r.url()) && r.request().method() !== "GET") reqs.push(`${r.request().method()} ${r.url().replace(/^.*\/admin\/llm\//, "")} ${r.status()}`); });
  sql(`UPDATE decision_gates SET enabled=false`); await page.goto("/settings/llm"); await page.waitForTimeout(6000);
  const sec = page.locator("section, div").filter({ has: page.getByText("Quyết định nhanh", { exact: true }) }).last(); const blk = page.locator("main");
  const gateRow = (name: string) => blk.getByText(name, { exact: true }).locator("xpath=ancestor::*[.//button[normalize-space()='Theo lớp']][1]");
  const g1 = gateRow("Kiểm duyệt Hỏi đáp công khai"); const sw = g1.getByRole("switch").first();
  // D06-09 / 11
  await g1.getByRole("button", { name: "Theo lớp" }).click(); await page.waitForTimeout(2000); const tbl = await page.getByRole("table").count();
  const rowSw = page.locator("table").last().getByRole("switch"); const nsw = await rowSw.count(); const dis = nsw ? await rowSw.first().isDisabled().catch(() => null) : null; const ariaDis = nsw ? await rowSw.first().getAttribute("aria-disabled") : null;
  if (nsw) await rowSw.first().hover({ force: true }).catch(() => {}); await page.waitForTimeout(700); const tip1 = await page.getByRole("tooltip").allInnerTexts().catch(() => []);
  out.push(`09: bảng theo lớp ${tbl}, công tắc lớp ${nsw}, disabled ${dis}/${ariaDis}, tooltip ${JSON.stringify([...new Set(tip1)])}`);
  await g1.getByRole("button", { name: "Theo lớp" }).click().catch(() => {}); await page.waitForTimeout(500);
  await sw.hover(); await page.waitForTimeout(700); const tip2 = await page.getByRole("tooltip").allInnerTexts().catch(() => []); out.push(`11: tooltip trên công tắc cổng (bản giả): ${JSON.stringify([...new Set(tip2)])}; title ${await sw.getAttribute("title")}`);
  // D06-13
  const v0 = sql(`select version from decision_gates where gate='CHAT_ROUTING' or true order by gate limit 3`).map((r: string[]) => r[0]).join(","); const a0 = sql(`select count(*) from audit_log where entity='decision_gate'`)[0][0];
  await sw.click(); await page.waitForTimeout(1200); const undo = await page.getByRole("button", { name: /Hoàn tác/ }).count(); const dlg = await page.getByRole("dialog").or(page.getByRole("alertdialog")).count(); const toast = await page.getByText(/Thành công/).count();
  const on1 = await sw.getAttribute("aria-checked"); await page.waitForTimeout(800); const a1 = sql(`select count(*) from audit_log where entity='decision_gate'`)[0][0];
  if (undo) await page.getByRole("button", { name: /Hoàn tác/ }).first().click(); await page.waitForTimeout(1500); const on2 = await sw.getAttribute("aria-checked"); const a2 = sql(`select count(*) from audit_log where entity='decision_gate'`)[0][0];
  out.push(`13: sau bật aria-checked ${on1}; «Hoàn tác» ${undo}; hộp thoại ${dlg}; toast «Thành công» ${toast}; sau hoàn tác ${on2}; audit_log decision_gate ${a0} → ${a1} → ${a2}`);
  // D06-04
  const chip = blk.getByText("gemini-decision-1", { exact: true }).first(); const m0 = sql(`select model from llm_models where provider_id='01a1deef-8e87-74a9-b3d2-aad764ed3dc8'`).map((r: string[]) => r[0]).join(",");
  await chip.click().catch(() => {}); await page.waitForTimeout(1500); const msg = (await blk.innerText()).match(/Chưa được phép[^¦\n]*/)?.[0] ?? "(không thấy)";
  const sel = await page.locator("main").getByRole("radio", { checked: true }).allInnerTexts().catch(() => []);
  out.push(`04: bấm «gemini-decision-1» → thông báo «${msg}»; mô hình trong DB ${m0}; đang chọn ${JSON.stringify(sel)}`);
  // D06-03
  const dec = blk.getByText("Quyết định", { exact: true }).first().locator("xpath=ancestor::*[.//button[normalize-space()='Test kết nối']][1]"); await dec.getByRole("button", { name: "Test kết nối" }).first().click(); await page.waitForTimeout(2500);
  out.push(`03: yêu cầu ${JSON.stringify(reqs.slice(-3))}; chữ sau test: ${((await dec.innerText()).match(/(Kết nối|Thành công|OK|lỗi)[^¦\n]*/i) ?? [""])[0]}`);
  console.log("VERDICT D06", JSON.stringify(out)); await page.screenshot({ path: `${OUT}/s105-d06b.png`, fullPage: true }); await ctx.close();
});

test("D06-08", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin"); const codes: string[] = []; page.on("response", (r) => { if (/decision\/gates/.test(r.url()) && r.request().method() === "PUT") codes.push(String(r.status())); });
  await page.goto("/settings/llm"); await page.waitForTimeout(6000); const blk = page.locator("main"); const out: string[] = [];
  const g = blk.getByText("Định tuyến câu hỏi chat", { exact: true }).locator("xpath=ancestor::*[.//button[normalize-space()='Cài đặt nâng cao']][1]");
  await g.getByRole("button", { name: "Cài đặt nâng cao" }).click(); await page.waitForTimeout(1000);
  const th = page.locator("input[placeholder='0,70–0,99']"); const to = page.locator("input[placeholder='200–5000']");
  const tryv = async (inp: any, v: string, label: string) => { codes.length = 0; await inp.fill(v); await page.getByRole("button", { name: "Lưu", exact: true }).last().click(); await page.waitForTimeout(1500); const err = (await page.locator("[role=alert], [aria-invalid=true], [class*=error]").allInnerTexts().catch(() => [])).filter((x: string) => x.trim()).slice(0, 2); const toast = await page.locator("[class*=toast], [role=status]").allInnerTexts().catch(() => []); out.push(`${label}=${v}: PUT ${JSON.stringify(codes)}; lỗi tại ô ${JSON.stringify(err)}; toast ${JSON.stringify(toast.filter((x: string) => x.trim()).slice(0, 2))}`); };
  await tryv(th, "0,001", "ngưỡng"); await tryv(to, "5001", "thời hạn"); await tryv(th, "0,9", "ngưỡng hợp lệ"); await tryv(th, "0,95", "ngưỡng trả lại"); await tryv(to, "800", "thời hạn trả lại");
  console.log("VERDICT D06-08", JSON.stringify(out)); await page.screenshot({ path: `${OUT}/s105-d0608.png` }); await ctx.close();
});

test("D06-375", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin", 375); await page.goto("/settings/llm"); await page.waitForTimeout(6000);
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1); const sec = page.locator("main").getByRole("heading", { name: "Quyết định nhanh" }).first(); await sec.scrollIntoViewIfNeeded();
  const sw = await page.locator("main").getByRole("switch").evaluateAll((els) => els.map((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return [Math.round(r.width), Math.round(r.height), Math.round(r.top + window.scrollY)]; }));
  const y0 = await sec.evaluate((e) => e.getBoundingClientRect().top + window.scrollY); const gateSw = sw.filter((x) => x[2] > y0 && x[2] < y0 + 400);
  const tgt = await page.locator("main").getByRole("button", { name: /Theo lớp|Cài đặt nâng cao/ }).evaluateAll((els) => els.slice(-6).map((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return [Math.round(r.width), Math.round(r.height)]; }));
  await page.locator("main").getByRole("button", { name: "Theo lớp" }).first().click(); await page.waitForTimeout(1500); const ov2 = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
  console.log("VERDICT D06-375", JSON.stringify({ ov, ov2, gateSwitches: gateSw, buttons: tgt })); await page.screenshot({ path: `${OUT}/s105-d06-375.png` }); await ctx.close();
});

test("D07", async ({ browser }) => {
  const out: string[] = [];
  { const { ctx, page } = await pageAs(browser, "admin", 375); await page.goto("/observability"); await page.waitForTimeout(6000); const ov = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
    const tb = await page.locator("table").evaluateAll((els) => els.map((e) => { const p = e.parentElement as HTMLElement; return [e.scrollWidth, p.clientWidth, getComputedStyle(p).overflowX]; })); const t = (await page.locator("main").innerText()); out.push(`375: tràn ngang trang ${ov}; bảng [scrollWidth, khung, overflow-x] ${JSON.stringify(tb.slice(0, 6))}; có «Quyết định»: ${/Quyết định/.test(t)}`); await page.screenshot({ path: `${OUT}/s105-d07-375.png`, fullPage: true }); await ctx.close(); }
  { const { ctx, page } = await pageAs(browser, "admin"); let n = 0; await page.route(/observability\/decision/, (r) => { n++; return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }); });
    await page.goto("/observability"); await page.waitForTimeout(6000); const retry = await page.getByRole("button", { name: /Thử lại/ }).count(); out.push(`12 lỗi: yêu cầu ${n}, nút «Thử lại» ${retry}`); if (retry) { await page.unroute(/observability\/decision/); await page.getByRole("button", { name: /Thử lại/ }).first().click(); await page.waitForTimeout(3000); out.push(`12 sau thử lại: «Thử lại» còn ${await page.getByRole("button", { name: /Thử lại/ }).count()}`); } await ctx.close(); }
  console.log("VERDICT D07", JSON.stringify(out));
});

test("D03-12", async ({ browser }) => {
  const label = process.env.D3_LABEL ?? "x"; const body = process.env.D3_BODY ?? ""; const { ctx, page } = await pageAs(browser, "sv.kha"); const codes: string[] = [];
  page.on("response", (r) => { if (/\/threads$/.test(r.url()) && r.request().method() === "POST") codes.push(String(r.status())); });
  await page.goto("/threads"); await page.waitForTimeout(2500); await page.getByRole("button", { name: "Đặt câu hỏi" }).click(); await page.waitForTimeout(700);
  const f = page.locator("[data-part=thread-form]"); await f.getByLabel("Tiêu đề").fill("QC105 hộp thoại " + label + " " + Date.now()); await f.getByLabel("Nội dung").fill(body); await page.waitForTimeout(1500);
  await f.getByRole("button", { name: "Đăng", exact: true }).click(); await page.waitForTimeout(2500);
  const dlg = page.getByRole("dialog").or(page.getByRole("alertdialog")).last(); const dt = (await dlg.innerText().catch(() => "")).replace(/\n+/g, " ¦ "); const btns = await dlg.getByRole("button").allInnerTexts().catch(() => []);
  const bad = /(decision|cascade|embedding|fallback|FAIL|RAG|PII|classif|AI)/i.test(dt);
  console.log("VERDICT D03-12", JSON.stringify({ label, codes, dt: dt.slice(0, 420), btns, forbidden: bad })); await page.screenshot({ path: `${OUT}/s105-d0312-${label}.png` }); await ctx.close();
});

test("D06-14-16-19", async ({ browser }) => {
  const out: string[] = [];
  // 14: hai tab
  { const { ctx, page } = await pageAs(browser, "admin"); const codes: string[] = []; page.on("response", (r) => { if (/decision\/gates/.test(r.url()) && r.request().method() === "PUT") codes.push(String(r.status())); });
    await page.goto("/settings/llm"); await page.waitForTimeout(6000); const blk = page.locator("main");
    const g = blk.getByText("Định tuyến câu hỏi chat", { exact: true }).locator("xpath=ancestor::*[.//button[normalize-space()='Cài đặt nâng cao']][1]"); const sw = g.getByRole("switch").first(); const before = await sw.getAttribute("aria-checked");
    const api = await request.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true }); const lg = await api.post("/api/v1/auth/login", { data: { email: "admin@edupilot.local", password: PW } }); const tok = (await lg.json()).access_token; const H = { Authorization: `Bearer ${tok}` };
    const cur = await (await api.get("/api/v1/admin/llm/decision", { headers: H })).json(); const ver = cur.gates.find((x: any) => x.gate === "CHAT_ROUTING").version; const r2 = await api.put("/api/v1/admin/llm/decision/gates/CHAT_ROUTING", { headers: H, data: { enabled: true, version: ver } });
    await sw.click(); await page.waitForTimeout(2000); const after = await sw.getAttribute("aria-checked"); const txt = (await blk.innerText()).replace(/\n+/g, " ¦ "); const note = txt.match(/(đã đổi|xung đột|tải lại|Dữ liệu đã thay đổi|người khác)[^¦]*/i)?.[0] ?? "(không thấy)";
    out.push(`14: tab khác đổi cổng (PUT ${r2.status()}); tab này bấm với version cũ → PUT ${JSON.stringify(codes)}; công tắc ${before} → ${after}; thông báo «${note}»; DB enabled = ${sql("select enabled from decision_gates where gate='CHAT_ROUTING'")[0][0]}`);
    const cur2 = await (await api.get("/api/v1/admin/llm/decision", { headers: H })).json(); await api.put("/api/v1/admin/llm/decision/gates/CHAT_ROUTING", { headers: H, data: { enabled: false, version: cur2.gates.find((x: any) => x.gate === "CHAT_ROUTING").version } }); await ctx.close(); }
  // 16: tải chậm / 500 / chưa cấu hình
  { const { ctx, page } = await pageAs(browser, "admin"); await page.route(/admin\/llm\/decision$/, async (r) => { await new Promise((x) => setTimeout(x, 3000)); await r.continue(); });
    await page.goto("/settings/llm"); await page.waitForTimeout(1200); const skel = await page.locator("[aria-busy=true], [data-part*=skeleton], [class*=skeleton], [class*=Skeleton]").count(); const spin = await page.locator("[role=progressbar], [class*=spinner]").count(); await page.waitForTimeout(5000);
    out.push(`16 chậm 3 s: skeleton ${skel}, spinner riêng ${spin}`); await ctx.close(); }
  { const { ctx, page } = await pageAs(browser, "admin"); let n = 0; await page.route(/admin\/llm\/decision$/, (r) => { n++; return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }); });
    await page.goto("/settings/llm"); await page.waitForTimeout(5000); const retry = page.locator("main").getByRole("button", { name: /Thử lại/ }); const rc = await retry.count(); let after = -1;
    if (rc) { await page.unroute(/admin\/llm\/decision$/); await retry.first().click(); await page.waitForTimeout(3000); after = await page.locator("main").getByRole("button", { name: /Thử lại/ }).count(); }
    out.push(`16 lỗi 500: yêu cầu ${n}, «Thử lại» ${rc}, sau bấm còn ${after}`); await ctx.close(); }
  { const { ctx, page } = await pageAs(browser, "admin"); await page.route(/admin\/llm\/decision$/, async (r) => { const res = await r.fetch(); const j = await res.json(); j.provider = null; await r.fulfill({ response: res, json: j }); });
    await page.goto("/settings/llm"); await page.waitForTimeout(5000); const t = (await page.locator("main").innerText()).replace(/\n+/g, " ¦ "); const i = t.indexOf("Quyết định nhanh");
    out.push(`16 chưa cấu hình (provider=null): «${t.slice(i, i + 260)}»; nút «Thêm» ×${await page.locator("main").getByRole("button", { name: "Thêm", exact: true }).count()}`); await ctx.close(); }
  // 19: quét chữ
  { const { ctx, page } = await pageAs(browser, "admin"); await page.goto("/settings/llm"); await page.waitForTimeout(6000); const t = await page.locator("main").innerText(); const i = t.indexOf("Quyết định nhanh"); const blk = t.slice(i, t.indexOf("Mô hình tìm kiếm tài liệu"));
    out.push(`19: chữ khối «Quyết định nhanh» (${blk.length} ký tự) có RAG/PII/fallback/trace/cascade/decision(chữ thường) = ${/(\bRAG\b|\bPII\b|fallback|\btrace\b|cascade|\bdecision\b)/.test(blk.replace(/decision-fake|gemini-decision-1/g, ""))}`); await ctx.close(); }
  console.log("VERDICT D06b", JSON.stringify(out));
});

test("D06-16load", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin"); await page.route(/admin\/llm\/decision$/, async (r) => { await new Promise((x) => setTimeout(x, 3000)); await r.continue(); });
  await page.goto("/settings/llm"); await page.waitForTimeout(1500); const t = (await page.locator("main").innerText()).replace(/\n+/g, " ¦ "); const i = t.indexOf("Quyết định nhanh");
  const sk = await page.locator("main [class*=kel], main [aria-busy], main [role=status]").evaluateAll((els) => els.map((e) => `${e.tagName}.${(e as HTMLElement).className.toString().slice(0, 40)}|${e.getAttribute("aria-busy")}`).slice(0, 6));
  console.log("VERDICT D06-16load", JSON.stringify({ i, quyetdinh: i >= 0 ? t.slice(i, i + 160) : "(chưa có khối)", sk, head: t.slice(0, 160) })); await ctx.close();
  const c2 = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true }); await c2.close();
});
test("D06-19txt", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin"); await page.goto("/settings/llm"); await page.waitForTimeout(6000); const t = await page.locator("main").innerText(); const i = t.indexOf("Quyết định nhanh"); const blk = t.slice(i, t.indexOf("Mô hình tìm kiếm tài liệu")).replace(/decision-fake|gemini-decision-1/g, "");
  const hits = (blk.match(/(\bRAG\b|\bPII\b|fallback|\btrace\b|cascade|\bdecision\b)/gi) ?? []); console.log("VERDICT D06-19txt", JSON.stringify({ len: blk.length, hits, text: blk.replace(/\n+/g, " ¦ ").slice(0, 500) })); await ctx.close();
});

test("D06-05-06-12", async ({ browser }) => {
  test.setTimeout(150000);
  const out: string[] = []; const KEY = "qc-ui-key-123";
  const { ctx, page } = await pageAs(browser, "admin"); const bodies: string[] = [];
  page.on("response", async (r) => { try { if (/\/api\//.test(r.url())) bodies.push(r.url() + " " + (await r.text())); } catch {} });
  const st = async () => (await page.locator("[data-part=decision-provider-status]").innerText()).replace(/\s+/g, " ").trim();
  await page.goto("/settings/llm"); await page.waitForTimeout(5000); const prov = page.locator("[data-part=decision-provider]");
  const s0 = await st(); out.push(`trạng thái đầu (decision-fake): «${s0}»`);
  await prov.getByRole("radio", { name: "gemini-decision-1" }).click(); await page.waitForTimeout(3000); const s1 = await st(); out.push(`chọn gemini-decision-1 chưa có khoá: «${s1}»`);
  await prov.getByRole("button", { name: "Nhập khoá" }).click(); await page.waitForTimeout(800);
  const inputs = await page.locator("input[type=password], input[name*=key i], input[name*=khoa i]").count(); const f = page.locator("input[type=password]").first();
  if (!(await f.count())) out.push("(không có input password — dump) " + (await prov.innerText()).replace(/\s+/g, " ").slice(0, 300)); else await f.fill(KEY);
  await page.screenshot({ path: process.env.QC_SHOT ? process.env.QC_SHOT + "/d06-key-form.png" : undefined }).catch(() => {});
  await prov.getByRole("button", { name: "Lưu" }).click(); await page.waitForTimeout(3000); const s2 = await st(); out.push(`sau lưu khoá: «${s2}»`);
  await page.reload(); await page.waitForTimeout(5000);
  const html = await page.content(); const vals = await page.locator("input").evaluateAll((els) => els.map((e) => (e as HTMLInputElement).value));
  const leakNet = bodies.filter((b) => b.includes(KEY)).map((b) => b.split(" ")[0].replace(/^https?:\/\/[^/]+/, "")); const s3 = await st();
  out.push(`tải lại: «${s3}»; DOM chứa khoá: ${html.includes(KEY)}; giá trị input chứa khoá: ${vals.some((v) => v.includes(KEY))}; phản hồi API chứa khoá (trừ phản hồi của chính PUT lưu? — liệt kê): ${JSON.stringify(leakNet)}; số phản hồi quan sát: ${bodies.length}`);
  // D06-12: công tắc cổng bản thật
  const row = page.locator("main").getByText("Kiểm duyệt Hỏi đáp công khai", { exact: true }).locator("xpath=ancestor::*[.//button[normalize-space()='Cài đặt nâng cao']][1]"); const sw = row.getByRole("switch").first();
  await sw.hover(); await page.waitForTimeout(1200); const tip = await page.locator("[role=tooltip]").allInnerTexts(); const ti = await sw.getAttribute("title"); const al = await sw.getAttribute("aria-label");
  const rowText = (await row.innerText()).replace(/\s+/g, " ").trim(); const icons = await page.locator("[data-part=decision-provider], main").first().locator("svg[class*=info i], svg[class*=help i], [data-icon=info], [aria-label*=\"Thông tin\"]").count();
  out.push(`D06-12: tooltip tại công tắc ${JSON.stringify(tip)} title=${ti} aria-label=${al}; hàng: «${rowText}»; icon Info/Help: ${icons}`);
  // D06-06: Tạm dừng
  const api = await request.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true }); const lg = await api.post("/api/v1/auth/login", { data: { email: "admin@edupilot.local", password: PW } }); const H = { Authorization: `Bearer ${(await lg.json()).access_token}` };
  const pr = (await (await api.get("/api/v1/admin/llm/providers", { headers: H })).json()).items.find((x: any) => x.type === "decision");
  const rp = await api.put(`/api/v1/admin/llm/providers/${pr.id}`, { headers: H, data: { type: "decision", name: pr.name, base_url: pr.base_url, enabled: false, rpm_limit: pr.rpm_limit, tpm_limit: pr.tpm_limit, version: pr.version, models: [{ model: "decision-fake", kind: "decision", price_in: "0", price_out: "0", enabled: false }, { model: "gemini-decision-1", kind: "decision", price_in: "0", price_out: "0", enabled: true }] } });
  await page.reload(); await page.waitForTimeout(5000); const s4 = await st(); out.push(`provider tạm dừng (PUT ${rp.status()}): «${s4}»`);
  writeFileSync("/Users/kuro/.config/edupilot-team/qc3-work/d06.txt", out.join("\n")); await ctx.close();
});

test("D06-22", async ({ browser }) => {
  const out: string[] = [];
  for (const w of [1440, 375]) {
    const { ctx, page } = await pageAs(browser, "admin"); await page.setViewportSize({ width: w, height: 900 }); await page.goto("/settings/llm"); await page.waitForTimeout(6000);
    const ov = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth })); const reds = await page.evaluate(() => { const r: Record<string, number> = {}; document.querySelectorAll("main *").forEach((e) => { const c = getComputedStyle(e as HTMLElement).color; if (/rgb\(\s*(1[89]\d|2\d\d),\s*(\d|[1-5]\d),\s*(\d|[1-7]\d)\)/.test(c) && (e as HTMLElement).children.length === 0 && (e.textContent || "").trim()) r[(e.textContent || "").trim().slice(0, 30)] = (r[(e.textContent || "").trim().slice(0, 30)] || 0) + 1; }); return r; });
    out.push(`${w}px: tràn ngang ${ov.sw > ov.cw + 1} (${ov.sw}/${ov.cw}); chữ màu đỏ trong main: ${JSON.stringify(reds)}`);
    if (w === 1440) { const stops: string[] = []; let noRing = 0; await page.keyboard.press("Tab"); for (let i = 0; i < 40; i++) { const info = await page.evaluate(() => { const e = document.activeElement as HTMLElement; if (!e) return null; const cs = getComputedStyle(e); return { t: (e.getAttribute("aria-label") || e.textContent || e.tagName).trim().slice(0, 24), ring: (cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0) || /rgb/.test(cs.boxShadow) }; }); if (!info) break; stops.push(info.t); if (!info.ring) noRing++; await page.keyboard.press("Tab"); } out.push(`phím Tab: ${stops.length} điểm dừng, không có viền focus: ${noRing}; đầu: ${JSON.stringify(stops.slice(0, 8))}`); }
    await page.screenshot({ path: `${process.env.QC_SHOT || "/tmp"}/d06-22-${w}.png`, fullPage: true }).catch(() => {}); await ctx.close();
  }
  writeFileSync("/Users/kuro/.config/edupilot-team/qc3-work/d0622.txt", out.join("\n"));
});

test("D06-22b", async ({ browser }) => {
  const out: string[] = [];
  for (const w of [1440, 375]) {
    const { ctx, page } = await pageAs(browser, "admin"); await page.setViewportSize({ width: w, height: 900 }); await page.goto("/settings/llm"); await page.waitForTimeout(6000);
    const ov = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth })); const cut = await page.evaluate(() => { const r: string[] = []; document.querySelectorAll("main *").forEach((e) => { const h = e as HTMLElement; if (h.children.length === 0 && h.scrollWidth > h.clientWidth + 1 && getComputedStyle(h).overflow !== "visible" && (h.textContent || "").length > 40) r.push((h.textContent || "").slice(0, 30)); const b = h.getBoundingClientRect(); if (b.right > window.innerWidth + 1 && b.width > 0) r.push("ngoài khung: " + (h.textContent || "").slice(0, 20)); }); return r.slice(0, 5); });
    out.push(`${w}px: tràn ngang ${ov.sw > ov.cw + 1} (${ov.sw}/${ov.cw}); phần tử bị cắt/ngoài khung: ${JSON.stringify(cut)}`); await page.screenshot({ path: `${process.env.QC_SHOT || "/tmp"}/d06-22b-${w}.png`, fullPage: false }).catch(() => {}); await ctx.close();
  }
  writeFileSync("/Users/kuro/.config/edupilot-team/qc3-work/d0622b.txt", out.join("\n"));
});
