import { appendFileSync, writeFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { expect, test, type Page, type Browser } from "@playwright/test";
import { call, login, needReal, sql, PW, OUT } from "./qc-real-lib";

// QC smoke S10 (11-19): UI giảng viên /questions trên ep-qc, lớp QD10A (không đụng 761987/761988).
const BASE = process.env.QC_BASE ?? "https://localhost:943";
const RES = process.env.QC_RES ?? `${OUT}/results_m.jsonl`;
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 220)); };
test.use({ actionTimeout: 30_000 });
test.beforeAll(() => { needReal(); });

async function gvPage(browser: Browser, w = 1440, code = "QD10A", user = "teacher") {
  const id = sql(`select id from courses where class_code='${code}'`)[0][0];
  const ctx = await browser.newContext({ baseURL: BASE, ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), id);
  const page = await ctx.newPage();
  await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page, id };
}

/** Các lệnh POST …/questions/bulk gateway ep-qc ghi nhận từ mốc `since` (ISO): [{t, status}]. */
function srvBulk(since: string) {
  const out = execFileSync("docker", ["logs", "--since", since, "ep-qc-gateway-1"], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
  return out.split("\n").filter((l) => l.includes("/questions/bulk") && l.includes('"POST"')).map((l) => { try { const j = JSON.parse(l); return { t: j.time as string, status: j.status as number }; } catch { return { t: "", status: 0 }; } });
}
const dbn = (q: string) => Number(sql(q)[0][0]);
const CID = () => sql(`select id from courses where class_code='QD10A'`)[0][0];
const pendIds = (n: number) => sql(`select id from question_bank where course_id='${CID()}' and review_status='PENDING' and type='MCQ_SINGLE' order by created_at limit ${n}`).map((r) => r[0]);
function resetPending(n = 12) { // đưa các câu PENDING về trạng thái chờ duyệt để chạy lại
  sql(`update question_bank set review_status='PENDING', reviewed_by=null, reviewed_at=null, reject_reason=null where course_id='${CID()}' and title ~ '^(Quản trị rủi ro|Tấn công và phòng thủ|Mật mã)' and review_status in ('APPROVED','REJECTED') and origin='MANUAL' and archived_at is null and updated_at > now() - interval '1 day' and false`);
}

test("TC-04-01 + TC-04-02 trang /questions và menu Tạo câu hỏi", async ({ browser }) => {
  const { ctx, page } = await gvPage(browser);
  await page.goto("/questions"); await page.waitForTimeout(2000);
  await page.screenshot({ path: `${OUT}/s10-0401-1440.png` });
  const h1 = await page.getByRole("heading", { level: 1 }).innerText();
  const filters = await page.locator("main label, main [role=combobox]").count();
  const labels = (await page.locator("main").innerText()).split("\n");
  const wanted = ["Trạng thái", "Chủ đề", "Độ khó", "Loại", "Nguồn"];
  const hasFilters = wanted.every((w) => labels.some((l) => l.trim() === w));
  const primary = await page.getByRole("button", { name: "Tạo câu hỏi" }).count();
  const body = await page.locator("main").innerText();
  const gen = (await page.locator("main").getByRole("button").or(page.locator("main").getByRole("link")).allInnerTexts()).filter((t) => /^(Sinh|.*Sinh câu|.*AI gợi|.*Gợi ý từ AI)/.test(t)); const genVisible = gen.length > 0; const aiText = body.match(/Sinh câu hỏi|Sinh từ tài liệu|AI soạn nháp|AI gợi ý/)?.[0] ?? "";
  verdict("TC-04-01", /Ngân hàng câu hỏi/.test(h1) && hasFilters && primary === 1 && !genVisible, `tiêu đề «${h1}»; bộ lọc ${wanted.join("/")} có ${hasFilters}; nút chính «Tạo câu hỏi» ×${primary}; điều khiển (nút/liên kết) sinh AI hiển thị mặc định: ${JSON.stringify(gen)}; chữ «AI» trong nội dung: «${aiText}»`);
  await page.getByRole("button", { name: "Tạo câu hỏi" }).first().click(); await page.waitForTimeout(500);
  const items = await page.getByRole("menuitem").allInnerTexts(); await page.screenshot({ path: `${OUT}/s10-0402-menu.png` });
  verdict("TC-04-02", items.some((x) => /Soạn tay/.test(x)) && items.some((x) => /Trích từ đề/.test(x)) && items.some((x) => /Sinh từ tài liệu/.test(x)), `menu: ${JSON.stringify(items)} (US-P9-03 đã làm nên có «Sinh từ tài liệu»)`);
  await ctx.close();
});

test("TC-04-04 chọn hàng loạt, trần 50", async ({ browser }) => {
  const { ctx, page } = await gvPage(browser);
  await page.goto("/questions"); await page.waitForTimeout(2000);
  const boxes = page.getByRole("checkbox");
  for (const i of [4, 5, 6]) await boxes.nth(i).check();
  await page.waitForTimeout(500);
  const bar = await page.getByRole("toolbar").allInnerTexts(); await page.screenshot({ path: `${OUT}/s10-0404-bar.png` });
  const n0 = await boxes.count();
  for (let i = 7; i < Math.min(n0, 70); i++) { await boxes.nth(i).check({ timeout: 4000 }).catch(() => {}); }
  const checked = await page.getByRole("checkbox", { checked: true }).count();
  const txt = (await page.getByRole("toolbar").allInnerTexts()).join(" | ");
  verdict("TC-04-04", /Duyệt/.test(bar.join(" ")) && /Loại/.test(bar.join(" ")) && checked <= 50, `3 hàng: thanh ${JSON.stringify(bar).slice(0, 200)}; thử chọn tới ${n0 - 1} hàng → tối đa ${checked} ô được chọn (trần 50); thanh: ${txt.slice(0, 140)}`);
  await ctx.close();
});

test("TC-04-16 rỗng / lỗi tải / đang tải", async ({ browser }) => {
  const { ctx, page } = await gvPage(browser, 1440, "QD10C");
  await page.goto("/questions"); await page.waitForTimeout(2000);
  const empty = await page.locator("main").innerText(); await page.screenshot({ path: `${OUT}/s10-0416-empty.png` });
  const { ctx: c2, page: p2 } = await gvPage(browser, 1440, "QD10A");
  const H = (r: import("@playwright/test").Route) => r.abort(); await p2.route("**/api/v1/courses/*/questions?*", H);
  await p2.goto("/questions"); await p2.waitForTimeout(9000);
  const err = await p2.locator("main").innerText(); await p2.screenshot({ path: `${OUT}/s10-0416-error.png` });
  const retry = await p2.getByRole("button", { name: "Thử lại" }).count();
  await p2.unroute("**/api/v1/courses/*/questions?*", H); if (retry) await p2.getByRole("button", { name: "Thử lại" }).first().click();
  await p2.waitForTimeout(2500);
  const rows = await p2.getByRole("row").count();
  const p3 = await c2.newPage(); await p3.route("**/api/v1/courses/*/questions?*", async (r) => { await new Promise((x) => setTimeout(x, 2500)); await r.continue(); });
  await p3.goto("/questions"); await p3.waitForTimeout(700);
  const sk = await p3.locator("[data-ep-skeleton], [class*=skeleton], [aria-busy=true]").count(); await p3.screenshot({ path: `${OUT}/s10-0416-loading.png` });
  verdict("TC-04-16", /Chưa có câu hỏi/.test(empty) && /Tạo câu hỏi/.test(empty) && retry > 0 && rows > 3 && sk > 0, `rỗng: «${empty.replace(/\n/g, " ").slice(0, 100)}»; lỗi tải: «${err.replace(/\n/g, " ").slice(0, 120)}» nút Thử lại ×${retry}, bấm → ${rows} hàng; đang tải: skeleton ×${sk}`);
  await ctx.close(); await c2.close();
});

async function openEdit(page: Page, title: string) {
  await page.getByPlaceholder("Tiêu đề…").fill(title); await page.waitForTimeout(1500);
  await page.getByRole("row").filter({ hasText: title }).first().click(); await page.waitForTimeout(800);
}
test("TC-04-09 + TC-04-10 Drawer sửa và duyệt; nháp khi đóng nhầm", async ({ browser }) => {
  const { ctx, page } = await gvPage(browser);
  const [qid] = pendIds(1); const t0 = sql(`select title,stem from question_bank where id='${qid}'`)[0];
  await page.goto("/questions"); await page.waitForTimeout(1500);
  await openEdit(page, t0[0]);
  await page.getByRole("dialog").getByRole("button", { name: "Chỉnh sửa" }).click(); await page.waitForTimeout(800);
  const stem = () => page.getByRole("dialog").getByRole("textbox").nth(2);
  await stem().click(); await stem().press("End"); await stem().type(" QCNHAP-20-KY-TU-DO-DANG");
  await page.waitForTimeout(800); await page.keyboard.press("Escape"); await page.waitForTimeout(600);
  const closed = await page.getByRole("dialog").count() === 0;
  await page.getByRole("row").filter({ hasText: t0[0] }).first().click(); await page.waitForTimeout(1000);
  let kept = (await page.getByRole("dialog").innerText()).includes("QCNHAP-20-KY-TU-DO-DANG");
  if (!kept && await page.getByRole("dialog").getByRole("button", { name: "Chỉnh sửa" }).count()) { await page.getByRole("dialog").getByRole("button", { name: "Chỉnh sửa" }).click(); await page.waitForTimeout(800); kept = ((await stem().inputValue().catch(() => "")) ?? "").includes("QCNHAP-20-KY-TU-DO-DANG"); }
  await page.screenshot({ path: `${OUT}/s10-0410-reopen.png` });
  verdict("TC-04-10", closed && kept, `gõ dở ≥20 ký tự vào đề, Esc (đóng ${closed}), mở lại cùng câu: bản gõ dở còn ${kept}`);
  // TC-04-09: sửa đề / giải thích / độ khó rồi Lưu và Duyệt trong Drawer
  if (!kept) { await page.getByRole("dialog").getByRole("button", { name: "Huỷ" }).click().catch(() => {}); await page.keyboard.press("Escape"); await openEdit(page, t0[0]); await page.getByRole("dialog").getByRole("button", { name: "Chỉnh sửa" }).click(); await page.waitForTimeout(600); }
  await stem().fill(t0[1] + " (QC sửa)");
  await page.getByRole("dialog").getByRole("button", { name: "Lưu thay đổi" }).click(); await page.waitForTimeout(2000);
  const afterSave = sql(`select review_status, stem from question_bank where id='${qid}'`)[0];
  const ap = page.getByRole("dialog").getByRole("button", { name: /^Duyệt$/ }); const nAp = await ap.count();
  if (nAp) { await ap.click(); await page.waitForTimeout(2500); }
  const st = sql(`select review_status, stem from question_bank where id='${qid}'`)[0];
  await page.screenshot({ path: `${OUT}/s10-0409-after.png` });
  verdict("TC-04-09", afterSave[1].includes("(QC sửa)") && nAp > 0 && st[0] === "APPROVED", `Drawer → Chỉnh sửa → đề sửa → «Lưu thay đổi»: DB đề đã sửa ${afterSave[1].includes("(QC sửa)")}; nút «Duyệt» trong Drawer ×${nAp}; sau Duyệt: ${st[0]}`);
  await ctx.close();
});

async function filterPending(page: Page) { // đưa nhóm «Quản trị rủi ro …» về Chờ duyệt (DB) rồi lọc theo tiêu đề; bảng chỉ còn các hàng này
  sql(`update question_bank set review_status='PENDING', reviewed_by=null, reviewed_at=null, reject_reason=null where course_id='${CID()}' and title like 'Quản trị rủi ro %' and type='MCQ_SINGLE'`);
  await page.getByPlaceholder("Tiêu đề…").fill("Quản trị rủi ro"); await page.waitForTimeout(1800);
}
async function selectRows(page: Page, n: number) {
  const titles: string[] = [];
  for (let i = 1; i <= n; i++) { const row = page.getByRole("row").nth(i); titles.push((await row.innerText()).split("\n").map((x) => x.trim()).filter(Boolean)[0]); await row.getByRole("checkbox").check(); }
  return titles;
}
const stOf = (titles: string[]) => titles.map((t) => sql(`select review_status from question_bank where course_id='${CID()}' and title='${t.replace(/'/g, "''")}' order by updated_at desc limit 1`)[0]?.[0] ?? `?${t}`);

test("TC-04-06b Loại: đóng tab trong 5 s vẫn gửi; Hoàn tác thì không", async ({ browser }) => {
  const { ctx, page } = await gvPage(browser);
  const bulk: { idem: string | undefined }[] = [];
  ctx.on("request", (r) => { if (r.method() === "POST" && /\/questions\/bulk/.test(r.url())) bulk.push({ idem: r.headers()["idempotency-key"] }); });
  const T0 = new Date().toISOString();
  await page.goto("/questions"); await page.waitForTimeout(1800); await filterPending(page);
  const undoTitles = await selectRows(page, 1);
  await page.getByRole("toolbar").getByRole("button", { name: "Loại", exact: true }).click(); await page.waitForTimeout(700);
  await page.getByRole("button", { name: /Hoàn tác/ }).first().click(); await page.waitForTimeout(500);
  await page.close({ runBeforeUnload: true }); await new Promise((r) => setTimeout(r, 6500));
  const nUndo = srvBulk(T0).length; const undoSt = stOf(undoTitles);
  const p2 = await ctx.newPage(); await p2.goto("/questions"); await p2.waitForTimeout(1800); await filterPending(p2);
  const titles = await selectRows(p2, 2);
  await p2.getByRole("toolbar").getByRole("button", { name: "Loại", exact: true }).click(); await p2.waitForTimeout(1500);
  const sentEarly = srvBulk(T0).length - nUndo;
  await p2.close({ runBeforeUnload: true }); await new Promise((r) => setTimeout(r, 4000));
  const st = stOf(titles); const all = srvBulk(T0); const after = all.length - nUndo;
  verdict("TC-04-06b", nUndo === 0 && undoSt[0] === "PENDING" && sentEarly === 0 && st.every((x) => x === "REJECTED") && after === 1 && all[all.length - 1].status === 200, `(theo log gateway) Hoàn tác trước khi rời: ${nUndo} lệnh bulk, câu giữ ${undoSt[0]}; Loại 2 câu rồi đóng tab ≤5 s: trước khi đóng ${sentEarly} lệnh, sau đóng ${after} lệnh bulk (status ${all.map((x) => x.status).join(",")}), DB ${st.join("/")} — gửi ngay khi pagehide, một lần`);
  await ctx.close();
});

test("TC-04-14 Duyệt hàng loạt khi mất mạng: gửi lại một lần cùng Idempotency-Key", async ({ browser }) => {
  const { ctx, page } = await gvPage(browser);
  const bulk: { idem: string | undefined; ok?: number }[] = [];
  ctx.on("request", (r) => { if (r.method() === "POST" && /\/questions\/bulk/.test(r.url())) bulk.push({ idem: r.headers()["idempotency-key"] }); });
  ctx.on("response", (r) => { if (/\/questions\/bulk/.test(r.url())) bulk.filter((b) => b.ok === undefined).forEach((b) => { b.ok = r.status(); }); });
  const T0 = new Date().toISOString();
  await page.goto("/questions"); await page.waitForTimeout(1800); await filterPending(page);
  const titles = await selectRows(page, 3); const before = stOf(titles);
  await ctx.setOffline(true);
  await page.getByRole("toolbar").getByRole("button", { name: "Duyệt", exact: true }).click(); await page.waitForTimeout(3000);
  const off = await page.locator("main").innerText(); const msg = /Chưa gửi được/.test(off);
  await page.screenshot({ path: `${OUT}/s10-0414-offline.png` });
  await ctx.setOffline(false); await page.waitForTimeout(8000);
    let st = stOf(titles);
  if (st.some((x) => x !== "APPROVED") && await page.getByRole("button", { name: /Thử lại/ }).count()) { await page.getByRole("button", { name: /Thử lại/ }).first().click(); await page.waitForTimeout(3000); st = stOf(titles); }
  await page.waitForTimeout(6000); st = stOf(titles);
  const sv = srvBulk(T0); const ok200 = sv.filter((b) => b.status === 200).length; const idems = new Set(bulk.map((b) => b.idem)).size;
  await page.screenshot({ path: `${OUT}/s10-0414-online.png` });
  verdict("TC-04-14", msg && sv.length === 1 && ok200 === 1 && idems === 1 && st.every((x) => x === "APPROVED"), `offline: «Chưa gửi được» ${msg}; trình duyệt thử gửi ${bulk.length} lần (khoá Idempotency-Key khác nhau: ${idems}); gateway nhận ${sv.length} lệnh bulk (200 ×${ok200}); DB ${before.join("/")} → ${st.join("/")}`);
  await ctx.close();
});

test("TC-04-12 câu EXTRACTED: nhãn nguồn + trích dẫn bấm được", async ({ browser }) => {
  const { ctx, page } = await gvPage(browser);
  const doc = sql(`select d.id from documents d join courses c on c.id=d.course_id where c.class_code='QD10A' and d.title like 'QC S10 Đề mẫu%' limit 1`)[0][0];
  const ch = sql(`select id from content_chunks where document_id='${doc}' limit 1`)[0]?.[0];
  const [qid] = pendIds(1); const t0 = sql(`select title from question_bank where id='${qid}'`)[0][0];
  sql(`update question_bank set origin='EXTRACTED', citations='[{"page":1,"document_id":"${doc}"${ch ? `,"chunk_id":"${ch}"` : ""}}]'::jsonb where id='${qid}'`);
  await page.goto("/questions"); await page.waitForTimeout(1500); await openEdit(page, t0);
  await page.screenshot({ path: `${OUT}/s10-0412-drawer.png` });
  const txt = await page.getByRole("dialog").innerText();
  const link = page.getByRole("dialog").getByRole("link").first(); const nl = await page.getByRole("dialog").getByRole("link").count();
  const href = nl ? await link.getAttribute("href") : null;
  let landed = ""; if (nl) { await link.click(); await page.waitForTimeout(2000); landed = new URL(page.url()).pathname + new URL(page.url()).search; }
  verdict("TC-04-12", /Nguồn|Tài liệu, trang/.test(txt) && nl > 0 && /library|documents/.test(landed), `Drawer câu EXTRACTED: chữ «${txt.replace(/\n/g, " ").slice(0, 200)}»; liên kết ×${nl} href=${href}; bấm → ${landed}`);
  await ctx.close();
});
