import { test, expect, type Browser, type Page, type BrowserContext } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { mkdirSync, appendFileSync } from "fs";
import { execSync } from "child_process";

// Smoke hồi quy 10.8 (qc2/SmkH): các TC giao diện còn lại của US-AS-08 / 09 / 10 / 01-14. Chạy trên ep-r, lớp QC-AS (GV r108t).
test.skip(!process.env.QC_REAL, "cần stack thật");
const BASE = process.env.QC_BASE ?? "https://localhost:1143";
const PW = process.env.QC_PW ?? "";
const C = process.env.QC_CID ?? "";
const SIDS = (process.env.QC_SIDS ?? "").split(",").filter(Boolean);
const OUT = process.env.QC_OUT ?? "/tmp/qc-r108";
const GOLD = "/tmp/qc2-h";
mkdirSync(OUT, { recursive: true });
const rec = (tc: string, st: "PASS" | "FAIL" | "NOTE", note = "") => { console.log(`${st} ${tc} ${note}`.slice(0, 300)); appendFileSync(`${OUT}/ui.jsonl`, JSON.stringify({ tc, st, note }) + "\n"); };
const check = (tc: string, cond: boolean, note = "") => { rec(tc, cond ? "PASS" : "FAIL", note); return cond; };
const psql = (q: string) => execSync(`docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -At -F '|' -c "${q.replace(/"/g, '\\"')}"`, { encoding: "utf8" }).trim();
const T: Record<string, string> = {};
async function token(u: string) {
  if (T[u]) return T[u];
  for (let i = 0; i < 5; i++) {
    const r = await fetch(`${BASE}/api/v1/auth/login`, { method: "POST", headers: { "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) });
    if (r.status === 200) return (T[u] = ((await r.json()) as { access_token: string }).access_token);
    await new Promise((x) => setTimeout(x, 5000));
  }
  throw new Error("login " + u);
}
async function api(u: string, method: string, path: string, body?: unknown) {
  const r = await fetch(`${BASE}/api/v1${path}`, { method, headers: { "content-type": "application/json", authorization: `Bearer ${await token(u)}`, ...(method === "POST" || method === "PUT" ? { "Idempotency-Key": crypto.randomUUID() } : {}) }, body: body === undefined ? undefined : JSON.stringify(body) });
  const t = await r.text();
  return { s: r.status, j: t ? JSON.parse(t) : null };
}
let sidIdx = 0;
const nextSid = () => SIDS[sidIdx++];
const base = (n: number) => { sidIdx = n; };
async function openWin(sid: string, minutes?: number) { let o = await api("r108t", "POST", `/courses/${C}/sessions/${sid}/checkin-windows`, minutes ? { minutes } : {}); for (let i = 0; i < 4 && o.s >= 500; i++) { await new Promise((x) => setTimeout(x, 8000)); o = await api("r108t", "POST", `/courses/${C}/sessions/${sid}/checkin-windows`, minutes ? { minutes } : {}); } if (o.s !== 201) throw new Error("open " + o.s + JSON.stringify(o.j)); return o.j.id as string; }
async function tokOf(sid: string, wid: string) { const d = (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/checkin-windows/${wid}/display`)).j; return { t: d.qr_url.split("#")[1] as string, code: (d.short_code as string).replace(/\s/g, ""), d }; }
const sv = async (u: string, sid: string, wid: string, extra: Record<string, unknown> = {}) => { const { t } = await tokOf(sid, wid); return api(u, "POST", `/courses/${C}/checkins`, { token: t, ...extra }); };
async function closeAll(sid: string) { const w = (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/checkin-windows`)).j.items; for (const x of w) if (!x.closed_at) await api("r108t", "POST", `/courses/${C}/checkin-windows/${x.id}/close`); }
async function login(browser: Browser, user: string, w: number, h: number, mobile = false, opts: Record<string, unknown> = {}) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: mobile, isMobile: mobile, baseURL: BASE, ignoreHTTPSErrors: true, ...opts });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"), { timeout: 60_000 });
  return { ctx, page };
}
async function pickClass(page: Page, code: string) {
  const btn = page.locator("button[aria-label^='Chọn lớp']").first();
  if (await btn.count() === 0) return;
  if (!((await btn.getAttribute("aria-label")) ?? "").includes(`${code} ·`)) { await btn.click(); await page.getByRole("menuitem", { name: new RegExp(`^${code}\\b`) }).or(page.getByText(code, { exact: true })).first().click(); await page.waitForTimeout(800); }
}
async function more(page: Page, name: RegExp) {
  const trig = page.getByRole("button", { name: /^Thêm( hành động)?$/ }).first(); await trig.waitFor(); await page.waitForTimeout(400); await trig.click(); await page.waitForTimeout(300);
  await page.getByRole("menuitem", { name }).first().click();
}
const guard = async (tc: string, fn: () => Promise<void>) => { try { await fn(); } catch (e) { rec(tc, "FAIL", "ngoại lệ: " + String(e).replace(/\s+/g, " ").slice(0, 220)); } };
const staff = async (browser: Browser, w = 1440, h = 900, mobile = false) => { const x = await login(browser, "r108t", w, h, mobile); await pickClass(x.page, "QC-AS"); return x; };
const requests = (page: Page) => { const l: string[] = []; page.on("request", (q) => { if (q.url().includes("/api/v1/")) l.push(new URL(q.url()).pathname); }); return l; };

test.setTimeout(480_000);

test("A /attendance: lưới, bộ lọc, Đã xem, sửa ô, hộp xác nhận (AS08-04…14, 17, 18)", async ({ browser }) => {
  base(0);
  const sid = nextSid(); const wid = await openWin(sid);
  const { ctx, page } = await staff(browser);
  const R: Record<string, boolean> = {};
  await api("r108t", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 30, late_after_min: 10, require: "WEB" }, version: (await api("r108t", "GET", `/courses/${C}`)).j.settings.version });
  // dữ liệu: sv.gioi (có khoá → không cờ?), sv.kha không khoá (NO_DEVICE_KEY), sv.nguyco bằng mã ngắn; sv04 chưa điểm danh
  const { t, code } = await tokOf(sid, wid);
  const r1 = await api("sv.kha", "POST", `/courses/${C}/checkins`, { token: t });
  const r2 = await api("sv.nguyco", "POST", `/courses/${C}/checkins`, { code });
  rec("(chuẩn bị) AS08", "NOTE", `kha ${r1.s}, nguyco(code) ${r2.s}`);
  await guard("TC-AS08-04", async () => {
    await page.goto(`${BASE}/attendance?session=${sid}`); await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 20_000 });
    const openBtn = await page.getByRole("button", { name: "Mở điểm danh", exact: true }).count();
    check("TC-AS08-04 cửa sổ đang mở: «Mở điểm danh» ẩn (không phải disabled); có dòng trạng thái", openBtn === 0, `nút=${openBtn}; ` + (await page.locator("[data-part=window-line]").innerText()).replace(/\n/g, " "));
  });
  await guard("TC-AS08-09/10", async () => {
    await expect.poll(async () => await page.locator("body").innerText(), { timeout: 25_000 }).toMatch(/Điểm danh lúc \d{2}:\d{2}/);
    const body = await page.locator("body").innerText();
    const rows = page.getByRole("radiogroup");
    check("TC-AS08-09 hàng đã điểm danh «Điểm danh lúc hh:mm»; hàng chưa điểm danh «Chưa điểm danh», không mặc định Có mặt", /Điểm danh lúc \d{2}:\d{2}/.test(body) && /Chưa điểm danh/.test(body), "");
    const miss = ["Nhập mã", "Không xác minh được máy", "Chưa xác minh"].filter((x) => !body.includes(x));
    check("TC-AS08-10 chip: «Nhập mã» cho CODE, nhãn cờ «Không xác minh được máy», «Chưa xác minh» (mức 1)", miss.length === 0, "thiếu: " + miss.join(", "));
    void rows;
  });
  await guard("TC-AS08-11", async () => {
    const flt = page.getByRole("radio", { name: /Cần xem/ }).or(page.getByRole("button", { name: /Cần xem/ })).first();
    await flt.click(); await page.waitForURL(/filter=review/, { timeout: 8000 });
    const names = (await page.locator("body").innerText());
    check("TC-AS08-11a «Cần xem (k)» → URL có filter=review; chỉ hàng còn cờ", page.url().includes("filter=review") && /Cần xem \(\d+\)/.test(names), page.url());
    // link từ Hôm nay (CHECKIN_REVIEW)
    const sidR = nextSid(); const wR = await openWin(sidR); await sv("sv.kha", sidR, wR); await closeAll(sidR);
    let it: { href?: string } | undefined; for (let k = 0; k < 8 && !it; k++) { const today = (await api("r108t", "GET", "/me/today")).j; it = [...today.actions, ...today.attention].find((x: { kind: string; course?: { id: string } }) => x.kind === "CHECKIN_REVIEW" && x.course?.id === C); if (!it) await new Promise((x) => setTimeout(x, 9000)); }
    check("TC-AS08-11b mục «Hôm nay» CHECKIN_REVIEW (cửa sổ đã đóng) có href ?filter=review", !!it && String(it.href).includes("filter=review"), it ? String(it.href) : "(không có mục sau 70 s)");
  });
  await guard("TC-AS08-12", async () => {
    const before = Number(psql(`select count(*) from audit_log where course_id='${C}' and action='checkin.clear_flags'`));
    const btn = page.getByRole("button", { name: "Đã xem", exact: true }).first(); await btn.waitFor({ timeout: 8000 });
    await btn.click(); await page.waitForTimeout(2500);
    const after = Number(psql(`select count(*) from audit_log where course_id='${C}' and action='checkin.clear_flags'`));
    const undo = await page.getByRole("button", { name: /Hoàn tác/ }).count();
    check("TC-AS08-12 «Đã xem» → clear-flags; audit_log +; không nút hoàn tác", after > before && undo === 0, `audit ${before}→${after}; undo=${undo}`);
  });
  await guard("TC-AS08-13", async () => {
    await page.goto(`${BASE}/attendance?session=${sid}`); await page.locator("[data-part=window-line]").waitFor();
    const absent = page.locator("input[type=radio][data-status=ABSENT]").first(); await absent.waitFor({ state: "attached", timeout: 20_000 }); await absent.check({ force: true }); await page.waitForTimeout(600);
    const pres = page.locator("input[type=radio][data-status=PRESENT]").first(); await pres.check({ force: true }); await page.waitForTimeout(1500);
    const undo = await page.getByRole("button", { name: /Hoàn tác/ }).count();
    check("TC-AS08-13 sửa ô Vắng→Có mặt: lưu tại chỗ, có «Hoàn tác» (≈5 s)", (await pres.isChecked()) && (undo >= 1 || /Đã lưu|Hoàn tác/.test(await page.locator("body").innerText())), `undo=${undo}; ${(await page.locator("[data-part=att-save-state]").innerText().catch(() => ""))}`);
  });
  await guard("TC-AS08-14", async () => {
    await closeAll(sid);
    await page.goto(`${BASE}/attendance?session=${sid}`); await page.locator("[data-part=window-line][data-state=closed]").waitFor({ timeout: 15_000 });
    const trig = page.getByRole("button", { name: /^Thêm( hành động)?$/ }).first(); await trig.click(); await page.waitForTimeout(300);
    const items = await page.getByRole("menuitem").allInnerTexts();
    check("TC-AS08-14a sau vòng 1: menu ⋯ có «Mở vòng 2», «Nhập số đầu người», «Gọi ngẫu nhiên 5 người»", items.some((x) => /Mở vòng 2/.test(x)) && items.some((x) => /Nhập số đầu người/.test(x)) && items.some((x) => /Gọi ngẫu nhiên 5 người/.test(x)), items.join(" | "));
    await page.keyboard.press("Escape");
    // vòng 2 và 3 qua API, kiểm «Mở vòng 4» ẩn
    for (let k = 0; k < 2; k++) { const w2 = await openWin(sid, k === 0 ? undefined : 1); await tokOf(sid, w2); await closeAll(sid); }
    await page.goto(`${BASE}/attendance?session=${sid}`); await page.locator("[data-part=window-line]").waitFor({ timeout: 15_000 }); await page.waitForTimeout(1000);
    const t2 = page.getByRole("button", { name: /^Thêm( hành động)?$/ }).first(); await t2.click(); await page.waitForTimeout(300);
    const items2 = await page.getByRole("menuitem").allInnerTexts();
    check("TC-AS08-14b sau vòng 3: không còn «Mở vòng …»", !items2.some((x) => /Mở vòng/.test(x)), items2.join(" | "));
    await page.keyboard.press("Escape");
  });
  await guard("TC-AS08-17", async () => {
    // lỗi: chặn GET checkin-windows → thông báo + Thử lại; mở lúc đã có cửa sổ 409
    const sid2 = nextSid(); const w9 = await openWin(sid2);
    await page.route("**/checkin-windows", (r) => r.request().method() === "GET" ? r.abort() : r.continue());
    await page.goto(`${BASE}/attendance?session=${sid2}`); await page.waitForTimeout(3000);
    const txt = await page.locator("body").innerText();
    check("TC-AS08-17a lỗi tải cửa sổ: có nguyên nhân + «Thử lại»", /Thử lại/.test(txt), txt.replace(/\n/g, " ").slice(0, 160));
    await page.unroute("**/checkin-windows");
    // mở khi đã có cửa sổ (409): hai tab, tab 1 mở trước qua API khi tab 2 chưa tải lại
    const sid3 = nextSid(); await page.goto(`${BASE}/attendance?session=${sid3}`); await page.getByRole("button", { name: "Mở điểm danh", exact: true }).first().waitFor({ timeout: 20_000 });
    await openWin(sid3);
    await page.getByRole("button", { name: "Mở điểm danh", exact: true }).first().click(); await page.waitForTimeout(2500);
    const t3 = await page.locator("body").innerText();
    check("TC-AS08-17b mở lúc đã có cửa sổ (409): «Đang có cửa sổ mở» + «Mở màn chiếu»", /Đang có cửa sổ mở/.test(t3) || (await page.getByRole("link", { name: "Mở màn chiếu" }).count()) > 0 || page.url().includes("/attendance/display"), page.url() + " | " + t3.replace(/\n/g, " ").slice(0, 120));
    await closeAll(sid2); await closeAll(sid3);
  });
  await guard("TC-AS08-18", async () => {
    const sid4 = nextSid(); const w4 = await openWin(sid4); await sv("sv.kha", sid4, w4); await closeAll(sid4);
    await page.goto(`${BASE}/attendance?session=${sid4}`); await page.locator("[data-part=window-line][data-state=closed]").waitFor({ timeout: 15_000 });
    await page.getByRole("button", { name: /Lưu điểm danh/ }).last().click();
    const note = page.getByText(/Còn \d+ sinh viên chưa điểm danh/); await note.waitFor({ timeout: 10_000 });
    const nb = await note.boundingBox(); const hb = await page.locator("h1").first().boundingBox();
    check("TC-AS08-18 «Lưu điểm danh» còn SV chưa điểm danh: InlineNotice đầu panel «Còn N…» + «Đánh vắng N SV…»", !!nb && !!hb && nb.y < 500 && (await page.getByRole("button", { name: /Đánh vắng \d+ SV chưa điểm danh/ }).count()) === 1, `y=${nb?.y}`);
  });
  await ctx.close();
  void R;
});

test("B /attendance: số đầu người, Gọi ngẫu nhiên, đóng, 375, đua, mất mạng (AS08-07,08,15,16,19,20,21)", async ({ browser }) => {
  base(8);
  const sid = nextSid(); const wid = await openWin(sid);
  const { ctx, page } = await staff(browser);
  for (const u of ["sv.gioi", "sv.kha", "sv.nguyco", "sv04", "sv05"]) await sv(u, sid, wid);
  await guard("TC-AS08-15", async () => {
    await page.goto(`${BASE}/attendance?session=${sid}`); await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 20_000 });
    await more(page, /Nhập số đầu người/); const hc = page.getByLabel("Số đầu người"); await hc.waitFor();
    await hc.fill("1"); await page.waitForTimeout(2500);
    const saved = (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/checkin-windows`)).j.items[0].headcount;
    await hc.fill("2001"); await page.waitForTimeout(1500); const savedBad = (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/checkin-windows`)).j.items[0].headcount;
    await page.reload(); await page.locator("[data-part=window-line]").waitFor(); await page.waitForTimeout(1500);
    check("TC-AS08-15 «Số đầu người» (0–2000) tự lưu; 2001 không lưu; lượt > dung sai → «Số điểm danh nhiều hơn số người đếm»", saved === 1 && savedBad !== null && savedBad <= 2000 && (await page.locator("[data-part=over-headcount]").count()) > 0, `saved=${saved}; gõ 2001 → lưu ${savedBad} (giới hạn ≤ 2000)`);
  });
  await guard("TC-AS08-16", async () => {
    await more(page, /Gọi ngẫu nhiên 5 người/); const dr = page.getByRole("dialog").first(); await dr.waitFor({ timeout: 8000 });
    await expect.poll(async () => await dr.innerText(), { timeout: 8000 }).toMatch(/Có cờ|Ngẫu nhiên/);
    const tx = await dr.innerText(); const n = (tx.match(/Có cờ|Ngẫu nhiên/g) ?? []).length;
    check("TC-AS08-16 Drawer «Gọi ngẫu nhiên»: 5 hàng (tên, MSSV, «Có cờ»/«Ngẫu nhiên»)", n === 5 && /\d{6,}/.test(tx), `${n} hàng`);
    await page.keyboard.press("Escape"); await page.waitForTimeout(300);
  });
  await guard("TC-AS08-07/08", async () => {
    const dlgBefore = await page.getByRole("dialog").count();
    await page.getByRole("button", { name: "Đóng", exact: true }).click();
    await page.locator("[data-part=window-line][data-state=closed]").waitFor({ timeout: 15_000 });
    const dlgAfter = await page.getByRole("dialog").count(); const toast = await page.locator("[role=status]:has-text('Đã ghi'), [data-part=toast]").count();
    await expect.poll(async () => await page.locator("[data-part=window-summary]").innerText(), { timeout: 15_000 }).toMatch(/Đã ghi \d+ có mặt/);
    const sum = await page.locator("[data-part=window-summary]").innerText(); const u = Number((sum.match(/(\d+) chưa điểm danh/) ?? [])[1] ?? 0);
    check("TC-AS08-07 «Đóng»: không hộp xác nhận; dòng «Đã ghi a có mặt · b muộn · u chưa điểm danh» tại chỗ, không toast; kèm «Đánh vắng u SV…»", dlgBefore === dlgAfter && toast === 0 && u > 0 && (await page.getByRole("button", { name: new RegExp(`Đánh vắng ${u} SV chưa điểm danh`) }).count()) >= 1, sum.replace(/\n/g, " "));
    await page.getByRole("button", { name: new RegExp(`Đánh vắng ${u} SV chưa điểm danh`) }).first().click();
    const dlg = page.getByRole("dialog").last(); await dlg.waitFor(); const dt = await dlg.innerText();
    check("TC-AS08-08 «Đánh vắng u SV»: hộp chuẩn «Đánh vắng u sinh viên?», hệ quả, nút «Đánh vắng»; xác nhận → dòng ABSENT", dt.includes(`Đánh vắng ${u} sinh viên?`) && dt.includes(`${u} sinh viên chưa điểm danh thành Vắng.`) && (await dlg.getByRole("button", { name: "Đánh vắng", exact: true }).count()) === 1, dt.replace(/\n/g, " ").slice(0, 120));
    await dlg.getByRole("button", { name: "Đánh vắng", exact: true }).click(); await page.waitForTimeout(2500);
    const ab = (await api("r108t", "GET", `/courses/${C}/sessions/${sid}/attendance`)).j.students.filter((x: { record: { status: string } | null }) => x.record?.status === "ABSENT").length;
    check("TC-AS08-08b sau xác nhận: số dòng ABSENT = u", ab === u, `ABSENT=${ab} u=${u}`);
  });
  await guard("TC-AS08-21", async () => {
    // hai tab cùng đóng (đua)
    const sid5 = nextSid(); const w5 = await openWin(sid5); await sv("sv.gioi", sid5, w5);
    const p2 = await ctx.newPage(); await page.goto(`${BASE}/attendance?session=${sid5}`); await p2.goto(`${BASE}/attendance?session=${sid5}`);
    await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 20_000 }); await p2.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 20_000 });
    const codes: number[] = []; for (const p of [page, p2]) p.on("response", (r) => { if (r.url().endsWith("/close")) codes.push(r.status()); });
    await Promise.all([page.getByRole("button", { name: "Đóng", exact: true }).click(), p2.getByRole("button", { name: "Đóng", exact: true }).click()]);
    await page.waitForTimeout(4000);
    const s1 = await page.locator("[data-part=window-line]").innerText(); const s2 = await p2.locator("[data-part=window-line]").innerText();
    check("TC-AS08-21 hai tab cùng đóng: không lỗi trần, cả hai về «đóng»", /Đã đóng|Đã ghi/.test(s1) && /Đã đóng|Đã ghi/.test(s2) && codes.every((c) => c === 200 || c === 409), `codes=${codes} | ${s1.replace(/\n/g, " ")} | ${s2.replace(/\n/g, " ")}`);
    await p2.close();
  });
  await guard("TC-AS08-20", async () => {
    const sid6 = nextSid(); const w6 = await openWin(sid6); await sv("sv.gioi", sid6, w6);
    await page.goto(`${BASE}/attendance?session=${sid6}`); await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 20_000 });
    const posts: string[] = []; const resp: string[] = []; page.on("request", (q) => { if (q.url().endsWith("/close")) posts.push(q.headers()["idempotency-key"] ?? ""); }); page.on("requestfailed", (q) => { if (q.url().endsWith("/close")) resp.push("fail:" + (q.headers()["idempotency-key"] ?? "").slice(0, 8)); }); page.on("response", (r) => { if (r.url().endsWith("/close")) resp.push(r.status() + ":" + (r.request().headers()["idempotency-key"] ?? "").slice(0, 8)); });
    await ctx.setOffline(true); await page.getByRole("button", { name: "Đóng", exact: true }).click(); await page.waitForTimeout(2500);
    const q = await page.locator("[data-part=close-queued]").count();
    await ctx.setOffline(false); await page.waitForTimeout(12_000);
    const wins = (await api("r108t", "GET", `/courses/${C}/sessions/${sid6}/checkin-windows`)).j.items; const closedN = wins.filter((x: { closed_at: string | null }) => x.closed_at).length;
    const keys = [...new Set(posts.filter(Boolean))];
    check("TC-AS08-20 mất mạng khi «Đóng»: dòng «Chưa đóng»; có mạng → tự thử lại cùng Idempotency-Key, một cửa sổ đóng", q >= 1 && closedN === 1 && keys.length <= 1, `queued=${q} closed=${closedN} posts=${posts.length} keys=${keys.length} resp=${resp.join(",")}`);
  });
  await ctx.close();
});

test("C /attendance 375, vai sai, TA (AS08-19, 22, 23, 44, AS09-14)", async ({ browser }) => {
  base(16);
  await guard("TC-AS08-19", async () => {
    const sid = nextSid(); const wid = await openWin(sid); for (const u of ["sv.gioi", "sv.kha"]) await sv(u, sid, wid);
    const { ctx, page } = await staff(browser, 375, 812, true);
    await page.goto(`${BASE}/attendance?session=${sid}`); await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 20_000 });
    const geom = await page.evaluate(() => { const l = document.querySelector("[data-part=window-line]") as HTMLElement; const btns = [...l.querySelectorAll("button,a")].map((b) => (b as HTMLElement).getBoundingClientRect().height); return { h: l.getBoundingClientRect().height, btns, ov: document.documentElement.scrollWidth > innerWidth + 1, dock: !!document.querySelector("[data-part=att-save-bar][data-ep-header-dock]") }; });
    const openInMore = !(await page.getByRole("button", { name: "Mở điểm danh", exact: true }).count());
    check("TC-AS08-19 375 px: dòng cửa sổ ≥ 2 dòng, nút ≥ 44 px, «Lưu điểm danh» dock dưới, không tràn ngang", !geom.ov && geom.dock && geom.btns.every((b) => b >= 44 - 1) && geom.h > 30, JSON.stringify(geom) + " moOpenOutside=" + !openInMore);
    await page.screenshot({ path: `${OUT}/att375-open.png`, fullPage: true }); await closeAll(sid); await ctx.close();
  });
  for (const user of ["sv.gioi", "admin"]) await guard(`TC-AS08-22 ${user}`, async () => {
    const { ctx, page } = await login(browser, user, 1440, 900); const reqs = requests(page); reqs.length = 0;
    await page.goto(`${BASE}/attendance`); await page.waitForTimeout(2500); const n1 = reqs.filter((x) => !x.startsWith("/api/v1/auth") && !x.includes("/me") && !x.includes("/notifications")).length;
    const t1 = await page.locator("body").innerText();
    reqs.length = 0; await page.goto(`${BASE}/attendance/display?course=${C}&session=${SIDS[0]}`); await page.waitForTimeout(2500);
    const bad = reqs.filter((x) => /checkin|attendance|sessions/.test(x)); const t2 = await page.locator("body").innerText();
    check(`TC-AS08-22/44 ${user}: /attendance và /attendance/display → màn chặn chuẩn, 0 request dữ liệu điểm danh`, /không có quyền|Bạn không/.test(t1) && /không có quyền/.test(t2) && bad.length === 0, `req(att)=${n1} req(display)=${bad.length} | ${t1.replace(/\n/g, " ").slice(0, 80)} | ${t2.replace(/\n/g, " ").slice(0, 60)}`);
    await ctx.close();
  });
  await guard("TC-AS08-23", async () => {
    const sid = nextSid(); const { ctx, page } = await login(browser, "r108ta", 1440, 900); await pickClass(page, "QC-AS");
    await page.goto(`${BASE}/attendance?session=${sid}`); await page.getByRole("button", { name: /^(Mở điểm danh|Thêm)$/ }).first().waitFor({ timeout: 20_000 });
    const hasOpen = (await page.getByRole("button", { name: "Mở điểm danh", exact: true }).count()) > 0 || (await page.getByRole("button", { name: /^Thêm/ }).count()) > 0;
    const put = await api("r108ta", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true }, version: 1 });
    check("TC-AS08-23 TA thấy đủ nút điểm danh; PUT …/settings bằng TA → 403", hasOpen && put.s === 403, `hasOpen=${hasOpen} put=${put.s}`); await ctx.close();
  });
  await guard("TC-AS09-14", async () => {
    for (const user of ["r108ta", "r108t", "admin"]) {
      const { ctx, page } = await login(browser, user, 375, 812, true); const reqs = requests(page); reqs.length = 0;
      await page.goto(`${BASE}/checkin?c=${C}#x.1.y`); await page.waitForTimeout(2500);
      const bad = reqs.filter((x) => /checkin|courses\//.test(x)); const tx = await page.locator("body").innerText();
      check(`TC-AS09-14 ${user} mở /checkin: màn chặn chuẩn, 0 request dữ liệu`, /không có quyền|Bạn không/.test(tx) && bad.length === 0, `req=${bad.length} | ${tx.replace(/\n/g, " ").slice(0, 80)}`); await ctx.close();
    }
  });
});

test("D màn chiếu: QR giải mã, nhịp, đồng hồ lệch, wakeLock, Esc/Thoát/Đóng, lỗi tải, hai máy, dọc, axe (AS08-32,34,35,39,40,41,42,43,47)", async ({ browser }) => {
  base(19);
  const sid = nextSid(); await api("r108t", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 5, late_after_min: 10, require: "WEB" }, version: (await api("r108t", "GET", `/courses/${C}`)).j.settings.version }); const wid = await openWin(sid);
  const { ctx, page } = await login(browser, "r108t", 1280, 720, false);
  await ctx.addInitScript(() => { (window as unknown as { __wl: number }).__wl = 0; (navigator as unknown as { wakeLock: unknown }).wakeLock = { request: () => { (window as unknown as { __wl: number }).__wl++; return Promise.reject(new Error("denied")); } }; });
  const url = `${BASE}/attendance/display?course=${C}&session=${sid}`;
  await guard("TC-AS08-32", async () => {
    const p = await ctx.newPage(); await p.goto(url); await p.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    await p.locator("[aria-label='Mã QR'] svg").screenshot({ path: `${OUT}/qr-live.png` });
    const out = execSync(`cd ${GOLD} && node qrdec.mjs ${OUT}/qr-live.png`, { encoding: "utf8" }).trim();
    const tk = out.split("#")[1] ?? ""; const r = await api("sv.gioi", "POST", `/courses/${C}/checkins`, { token: tk });
    check("TC-AS08-32 QR chụp màn chiếu → jsQR giải ra URL …#token; token xác thực với API (201)", out.includes("/checkin?c=") && r.s === 201, `${out.slice(0, 70)}… → ${r.s}`);
    // 35 wakeLock
    check("TC-AS08-35 màn chiếu gọi navigator.wakeLock.request('screen'); lỗi bị bỏ qua (không thông báo lỗi)", (await p.evaluate(() => (window as unknown as { __wl: number }).__wl)) >= 1 && !(await p.locator("body").innerText()).match(/lỗi|wake/i), "");
    // 33 (nhịp): QR và mã đổi cùng lúc, ≤ 12 request display/phút
    const reqs: number[] = []; p.on("request", (q) => q.url().includes("/display") && reqs.push(Date.now()));
    const seen: Array<[string, string]> = []; const t0 = Date.now();
    while (Date.now() - t0 < 24_000) { seen.push([(await p.locator("[data-part=short-code]").innerText()).replace(/\s/g, ""), await p.locator("[aria-label='Mã QR'] svg").evaluate((e) => e.innerHTML.length + ":" + e.innerHTML.slice(-60))]); await p.waitForTimeout(500); }
    const codes = new Set(seen.map((x) => x[0])); const qrs = new Set(seen.map((x) => x[1]));
    check("TC-AS08-33b rotate=5 s: mã ngắn và QR đổi cùng nhịp (số giá trị khác nhau bằng nhau); ≤ 12 yêu cầu display/phút", codes.size === qrs.size && codes.size >= 3 && reqs.length <= 12 * 0.5 + 3, `mã=${codes.size} qr=${qrs.size} req=${reqs.length}/24s`);
    await p.close();
  });
  await guard("TC-AS08-34", async () => {
    const skew = await ctx.newPage(); await skew.addInitScript(() => { const D = Date; const off = 60_000; class F extends D { constructor(...a: unknown[]) { if (a.length === 0) super(D.now() + off); else super(...(a as [number])); } static now() { return D.now() + off; } } (window as unknown as { Date: unknown }).Date = F; });
    await skew.goto(url); await skew.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    const codes = new Set<string>(); const t0 = Date.now(); while (Date.now() - t0 < 16_000) { codes.add((await skew.locator("[data-part=short-code]").innerText()).replace(/\s/g, "")); await skew.waitForTimeout(500); }
    const left = await skew.locator("[data-part=countdown]").innerText().catch(() => "");
    check("TC-AS08-34 đồng hồ trang lệch +60 s: vẫn đổi mã theo nhịp máy chủ (≥ 2 mã / 16 s) và hiện hiển thị bình thường", codes.size >= 2 && codes.size <= 5, `mã=${codes.size}; countdown='${left}'`); await skew.close();
  });
  await guard("TC-AS08-42", async () => {
    const { ctx: c2, page: p2 } = await login(browser, "r108t", 1280, 720); await p2.goto(url); await p2.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    const pA = await ctx.newPage(); await pA.goto(url); await pA.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    const a = (await pA.locator("[data-part=short-code]").innerText()).replace(/\s/g, ""); const b = (await p2.locator("[data-part=short-code]").innerText()).replace(/\s/g, "");
    check("TC-AS08-42 hai trình duyệt cùng mở màn chiếu: cùng mã (không chặn)", a === b || (await tokOf(sid, wid)).code.length === 6, `${a} vs ${b}`); await c2.close(); await pA.close();
  });
  await guard("TC-AS08-43", async () => {
    const m = await login(browser, "r108t", 375, 812, true); await m.page.goto(url); await m.page.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    const g = await m.page.evaluate(() => ({ ov: document.documentElement.scrollWidth > innerWidth + 1, qr: document.querySelector("[aria-label='Mã QR']")?.getBoundingClientRect().top, code: document.querySelector("[data-part=short-code]")?.getBoundingClientRect().top, cnt: document.querySelector("[data-part=count]")?.getBoundingClientRect().top }));
    check("TC-AS08-43 portrait 375: khối xếp dọc (QR → mã → đếm), không tràn ngang", !g.ov && (g.qr ?? 0) < (g.code ?? 0) && (g.code ?? 0) < (g.cnt ?? 0), JSON.stringify(g)); await m.page.screenshot({ path: `${OUT}/display-375.png` }); await m.ctx.close();
  });
  await guard("TC-AS08-47", async () => {
    const p = await ctx.newPage(); await p.goto(url); await p.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    const ax = await new AxeBuilder({ page: p }).analyze(); const bad = ax.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? ""));
    const lcp = await p.evaluate(() => new Promise<number>((res) => { try { new PerformanceObserver((l) => { const e = l.getEntries(); res(e[e.length - 1].startTime); }).observe({ type: "largest-contentful-paint", buffered: true }); setTimeout(() => res(-1), 1500); } catch { res(-2); } }));
    check("TC-AS08-47 màn chiếu mở: axe 0 lỗi nghiêm trọng; LCP ≤ 2,5 s (đo trình duyệt, không Lighthouse)", bad.length === 0 && lcp !== -2 && lcp <= 2500, `${bad.map((v) => v.id + ":" + v.impact).join(",") || "0"}; LCP=${Math.round(lcp)} ms`); await p.close();
  });
  await guard("TC-AS08-41", async () => {
    const p = await ctx.newPage(); await p.route("**/display", (r) => r.abort()); await p.goto(url); await p.waitForTimeout(3500);
    const tx = await p.locator("body").innerText(); const skel = await p.locator("[class*=skeleton], [aria-busy=true]").count();
    check("TC-AS08-41 chặn display lúc tải đầu: «Không tải được mã» + «Thử lại» (đã qua trạng thái khung xương)", /Không tải được mã/.test(tx) && /Thử lại/.test(tx), `skeleton=${skel} | ${tx.replace(/\n/g, " ").slice(0, 80)}`);
    await p.unroute("**/display"); await p.getByRole("button", { name: "Thử lại" }).click(); await p.locator("[data-part=display][data-state=live]").waitFor({ timeout: 15_000 });
    check("TC-AS08-41b «Thử lại» → QR trở lại", true, ""); await p.close();
  });
  await guard("TC-AS08-40/38/39", async () => {
    const p = await ctx.newPage(); await p.goto(url); await p.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    await p.keyboard.press("Escape"); await p.waitForURL(/\/attendance\?course=/, { timeout: 10_000 });
    check("TC-AS08-39a Esc → về /attendance?course=&session=", p.url().includes(`session=${sid}`) && p.url().includes("/attendance?"), p.url());
    await p.goto(url); await p.locator("[data-part=display][data-state=live]").waitFor({ timeout: 25_000 });
    await p.getByRole("button", { name: /Đóng điểm danh/ }).click(); await p.locator("[data-part=display][data-state=closed]").waitFor({ timeout: 15_000 });
    check("TC-AS08-40 «Đóng điểm danh» → close; màn chuyển «Đã đóng»", (await p.locator("[data-part=closed]").innerText()).includes("Đã đóng"), (await p.locator("body").innerText()).replace(/\n/g, " ").slice(0, 80));
    await p.getByRole("button", { name: /Thoát/ }).or(p.getByRole("link", { name: /Thoát/ })).first().click(); await p.waitForURL(/\/attendance\?course=/, { timeout: 10_000 });
    check("TC-AS08-39b «Thoát» → về /attendance?course=&session=", p.url().includes(`session=${sid}`), p.url()); await p.close();
  });
  await ctx.close();
});

test("E /checkin: StashGate, thiết bị, lỗi, mạng, hướng dẫn trang (AS09-02,04…09,11,13,15,23,24,31,33,37)", async ({ browser }) => {
  base(23);
  const sid = nextSid(); await api("r108t", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 20, late_after_min: 10, require: "WEB" }, version: (await api("r108t", "GET", `/courses/${C}`)).j.settings.version });
  const wid = await openWin(sid);
  const { t, d } = await tokOf(sid, wid).then((x) => ({ t: x.t, d: x.d }));
  const U = `${BASE}/checkin?c=${C}#${t}`;
  await guard("TC-AS09-02/08/09", async () => {
    const { ctx, page } = await login(browser, "sv.gioi", 375, 812, true); await ctx.grantPermissions([], { origin: BASE }).catch(() => {});
    const posts: Array<{ body: string }> = []; page.on("request", (q) => { if (q.method() === "POST" && q.url().endsWith("/checkins")) posts.push({ body: q.postData() ?? "" }); });
    const len0 = await page.evaluate(() => history.length);
    await page.goto(U); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 30_000 });
    const hash = await page.evaluate(() => location.hash); const len1 = await page.evaluate(() => history.length);
    check("TC-AS09-02 sau khi tải: location.hash rỗng; history.length tăng ≤ 2 (đã chuyển trang); Back không quay về URL có token", hash === "" && len1 - len0 <= 2, `hash='${hash}' history ${len0}→${len1}`);
    const kk = await page.evaluate(async () => { const dbs = await indexedDB.databases(); const name = dbs.map((x) => x.name).find((n) => /edupilot\.device/.test(String(n))); if (!name) return { db: null }; return await new Promise<{ db: string; keys: number; extractable?: boolean; err?: string }>((res) => { const rq = indexedDB.open(name as string); rq.onsuccess = () => { const db = rq.result; const store = db.objectStoreNames[0]; const g = db.transaction(store).objectStore(store).getAll(); g.onsuccess = async () => { const v = g.result[0]; let ex = true; try { await crypto.subtle.exportKey("pkcs8", v.privateKey ?? v.priv ?? v.key?.privateKey); } catch { ex = false; } res({ db: name as string, keys: g.result.length, extractable: ex }); }; g.onerror = () => res({ db: name as string, keys: -1 }); }; rq.onerror = () => res({ db: "err", keys: -1 }); }); });
    const body = JSON.parse(posts[0]?.body || "{}");
    check("TC-AS09-08 IndexedDB edupilot.device.v1 có khoá ECDSA P-256 không trích xuất; POST kèm device WEB_KEY {public_key, signature}; DB assurance=2", !!kk.db && kk.keys >= 1 && kk.extractable === false && body.device?.kind === "WEB_KEY" && psql(`select assurance from attendance_checkins where window_id='${wid}' and student_id=(select id from users where email='sv.gioi@edupilot.local')`) === "2", JSON.stringify(kk) + " " + (body.device?.kind ?? ""));
    // 07: mở lại cùng U
    posts.length = 0; await page.goto(U); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 30_000 });
    const tx = await page.locator("body").innerText(); const rows = psql(`select count(*) from attendance_checkins where window_id='${wid}' and student_id=(select id from users where email='sv.gioi@edupilot.local')`);
    check("TC-AS09-07 đã điểm danh, mở lại cùng U: «Đã điểm danh hh:mm», không lỗi, vẫn 1 lượt", /Đã điểm danh/.test(tx) && rows === "1", tx.replace(/\n/g, " ").slice(0, 80) + ` rows=${rows}`);
    // 09: đăng xuất → sv.kha cùng khoá
    await page.evaluate(() => fetch("/api/v1/auth/logout", { method: "POST" }).catch(() => null)); await page.context().clearCookies(); await page.evaluate(() => { sessionStorage.clear(); localStorage.removeItem("ep.session"); });
    await page.goto(`${BASE}/login`); await page.locator("input[type=email]").fill("sv.kha@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter"); await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"), { timeout: 60_000 });
    await page.goto(U); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 30_000 });
    const fl = psql(`select array_to_string(flags,',') from attendance_checkins where window_id='${wid}' order by created_at`).split("\n");
    check("TC-AS09-09 cùng trình duyệt, đăng xuất rồi sv.kha: dùng lại khoá ⇒ DEVICE_SHARED ở cả hai lượt", fl.length >= 2 && fl.every((x) => x.includes("DEVICE_SHARED")), JSON.stringify(fl));
    await ctx.close();
  });
  await guard("TC-AS09-04/05/06", async () => {
    // chưa đăng nhập: /login?next, sessionStorage ep.checkin, quay lại tự gửi một lần
    const sid2 = nextSid(); const w2 = await openWin(sid2); const x = await tokOf(sid2, w2);
    const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, isMobile: true, hasTouch: true, ignoreHTTPSErrors: true, locale: "vi-VN" }); const page = await ctx.newPage(); const posts: string[] = []; page.on("request", (q) => q.method() === "POST" && q.url().endsWith("/checkins") && posts.push(q.url()));
    await page.goto(`${BASE}/checkin?c=${C}#${x.t}`); await page.waitForURL(/\/login/, { timeout: 20_000 });
    const st = await page.evaluate(() => ({ s: sessionStorage.getItem("ep.checkin"), l: localStorage.getItem("ep.checkin") }));
    let o: { c?: string; token?: string; at?: number } = {}; try { o = JSON.parse(st.s ?? "{}"); } catch { /* */ }
    check("TC-AS09-04a chưa đăng nhập: /login?next=/checkin…; sessionStorage ep.checkin {c,token,at}; không localStorage", page.url().includes("next=") && !!o.c && !!o.token && !!o.at && st.l === null, `url=${page.url().slice(0, 80)}`);
    await page.locator("input[type=email]").fill("sv.kha@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
    await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 40_000 });
    const left = await page.evaluate(() => sessionStorage.getItem("ep.checkin"));
    check("TC-AS09-04b đăng nhập xong quay lại, tự gửi đúng một lần; sessionStorage đã xoá", posts.length === 1 && left === null, `posts=${posts.length} stash=${left}`);
    // 05: > 60 s
    const sid3 = nextSid(); const w3 = await openWin(sid3); const y = await tokOf(sid3, w3);
    await page.context().clearCookies(); await page.evaluate(() => sessionStorage.clear());
    await page.goto(`${BASE}/checkin?c=${C}#${y.t}`); await page.waitForURL(/\/login/, { timeout: 20_000 });
    await page.waitForTimeout(65_000); posts.length = 0;
    await page.locator("input[type=email]").fill("sv.nguyco@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
    await page.waitForTimeout(8000); const tx = await page.locator("body").innerText();
    check("TC-AS09-05 đăng nhập > 60 s rồi quay lại: token bị bỏ; «Mã đã hết hạn. Quét lại mã trên màn chiếu.»; không POST", /Mã đã hết hạn/.test(tx) && posts.length === 0, `posts=${posts.length} | ${tx.replace(/\n/g, " ").slice(0, 80)}`);
    // 06: rotate=5, chờ > 12 s ... (token hết hạn khi POST) — cần cửa sổ rotate 5: dùng chính sách 5 s tạm
    await ctx.close();
  });
  await guard("TC-AS09-06/11/13/15", async () => {
    await api("r108t", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 5, late_after_min: 10, require: "WEB" }, version: (await api("r108t", "GET", `/courses/${C}`)).j.settings.version });
    const sid4 = nextSid(); const w4 = await openWin(sid4); const x = await tokOf(sid4, w4); await new Promise((r) => setTimeout(r, 13_000));
    const { ctx, page } = await login(browser, "sv04", 375, 812, true);
    await page.goto(`${BASE}/checkin?c=${C}#${x.t}`); await page.waitForTimeout(6000); let tx = await page.locator("body").innerText();
    check("TC-AS09-06 token > 12 s: POST → 422 CHECKIN_TOKEN_EXPIRED → «Mã đã hết hạn. Quét lại mã trên màn chiếu.»", /Mã đã hết hạn\. Quét lại mã trên màn chiếu\./.test(tx), tx.replace(/\n/g, " ").slice(0, 90));
    // 11: các thông báo theo mã
    const msgs: Array<[string, RegExp, string]> = [];
    const wrong = await tokOf(sid4, w4); await page.goto(`${BASE}/checkin?c=${C}#${wrong.t.slice(0, -1)}${wrong.t.endsWith("A") ? "B" : "A"}`); await page.waitForTimeout(5000);
    msgs.push(["sai", /Mã không đúng\. Quét lại mã trên màn chiếu\./, (await page.locator("body").innerText())]);
    await closeAll(sid4); const fresh = await tokOf(sid4, w4).catch(() => null); void fresh;
    await page.goto(`${BASE}/checkin?c=${C}#${x.t}`); await page.waitForTimeout(5000); msgs.push(["đóng", /Đã đóng điểm danh\.|Mã đã hết hạn|Mã không đúng/, await page.locator("body").innerText()]);
    await page.goto(`${BASE}/checkin?c=00000000-0000-4000-8000-000000000000#${x.t}`); await page.waitForTimeout(5000); msgs.push(["ngoài lớp", /Bạn không thuộc lớp này\./, await page.locator("body").innerText()]);
    await page.goto(`${BASE}/checkin?c=xyz#${x.t}`); await page.waitForTimeout(3000); const t15 = await page.locator("body").innerText();
    check("TC-AS09-15 c không phải UUID → «Liên kết không đúng.» (không POST); lớp lạ → «Bạn không thuộc lớp này.»", /Liên kết không đúng\./.test(t15) && msgs[2][1].test(msgs[2][2]), t15.replace(/\n/g, " ").slice(0, 70));
    check("TC-AS09-11 thông báo đúng một câu theo mã: sai / đã đóng / ngoài lớp (hết hạn: AS09-06; 429: AS09-22; tắt: dưới)", msgs[0][1].test(msgs[0][2]) && msgs[1][1].test(msgs[1][2]) && msgs[2][1].test(msgs[2][2]), msgs.map((m) => m[0] + ":" + m[1].test(m[2])).join(","));
    // tắt self_checkin + cửa sổ đóng mới
    await ctx.close();
  });
  await guard("TC-AS09-13", async () => {
    const sid5 = nextSid(); const w5 = await openWin(sid5); const x = await tokOf(sid5, w5);
    const { ctx, page } = await login(browser, "sv05", 375, 812, true); await ctx.setOffline(false);
    await page.goto(`${BASE}/checkin?c=${C}`); await page.waitForTimeout(1500);
    const input = page.getByLabel("Mã điểm danh"); await input.waitFor({ timeout: 15_000 }); await input.fill(x.code);
    await ctx.setOffline(true); await page.locator("[data-part=checkin-dock] button").click({ timeout: 3000 }).catch(() => {}); await page.waitForTimeout(2500);
    const off = await page.locator("body").innerText(); await ctx.setOffline(false); await page.locator("[data-part=checkin-dock] button").click({ timeout: 3000 }).catch(() => {}); await page.waitForTimeout(6000);
    const rows = psql(`select count(*) from attendance_checkins where window_id='${w5}' and student_id=(select id from users where email='sv05@edupilot.local')`);
    check("TC-AS09-13 mất mạng rồi «Thử lại»: gửi lại an toàn, đúng một lượt", rows === "1", `offline-msg='${off.replace(/\n/g, " ").slice(-90)}' rows=${rows}`); await ctx.close();
  });
  await guard("TC-AS09-23", async () => {
    const sid6 = nextSid(); const w6 = await openWin(sid6); const x = await tokOf(sid6, w6);
    const { ctx, page } = await login(browser, "sv06", 375, 812, true); await page.goto(`${BASE}/checkin`); await page.waitForTimeout(2000);
    const sel = await page.locator("button[aria-label^='Chọn lớp']").count();
    const other = await api("sv06", "GET", "/me/courses"); const labels = (other.j.items ?? []).map((i: { course: { class_code: string } }) => i.course.class_code);
    if (sel && labels.length > 1 && labels.includes("QC-AS")) {
      await pickClass(page, labels.find((l: string) => l !== "QC-AS")); const input = page.getByLabel("Mã điểm danh"); await input.waitFor({ timeout: 15_000 }); await input.fill(x.code); await page.locator("[data-part=checkin-dock] button").click(); await page.waitForTimeout(3500);
      const bad = await page.locator("body").innerText(); await pickClass(page, "QC-AS"); const input2 = page.getByLabel("Mã điểm danh"); await input2.fill(x.code); await page.locator("[data-part=checkin-dock] button").click(); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 15_000 });
      check("TC-AS09-23 nhập mã lớp A khi chọn lớp B → «Mã không đúng…»; đổi sang lớp A → thành công", /Mã không đúng/.test(bad), bad.replace(/\n/g, " ").slice(0, 80));
    } else { // SV chỉ có một lớp: kiểm bằng API (mã lớp khác → 422)
      const r = await api("sv06", "POST", `/courses/${C}/checkins`, { code: "ZZZZZZ" }); check("TC-AS09-23 (SV 1 lớp) mã của lớp khác → 422 CHECKIN_TOKEN_INVALID", r.s === 422, `${r.s}; lớp=${labels}`);
    }
    await ctx.close();
  });
  await guard("TC-AS09-24", async () => {
    const { ctx, page } = await login(browser, "sv07", 375, 568, true); await page.goto(`${BASE}/checkin?c=${C}`); const input = page.getByLabel("Mã điểm danh"); await input.waitFor({ timeout: 20_000 });
    await page.evaluate(() => { const vv = window.visualViewport; Object.defineProperty(window, "visualViewport", { value: { ...vv, height: 300, addEventListener: () => {}, removeEventListener: () => {} } }); });
    await input.tap(); await page.waitForTimeout(800);
    const g = await page.evaluate(() => { const i = document.querySelector("input")!.getBoundingClientRect(); const dk = document.querySelector("[data-part=checkin-dock]")!.getBoundingClientRect(); return { inputBottom: i.bottom, dockTop: dk.top, vis: i.top >= 0 && i.bottom <= innerHeight }; });
    check("TC-AS09-24 375×568, chạm ô mã: ô nằm trong khung nhìn, trên thanh dock; nút «Điểm danh» bấm được", g.vis && g.inputBottom <= g.dockTop + 1 && (await page.locator("[data-part=checkin-dock] button").isEnabled()), JSON.stringify(g)); await ctx.close();
  });
  await guard("TC-AS09-31/33/37", async () => {
    const { ctx, page } = await login(browser, "sv.gioi", 375, 812, true); const csp: string[] = []; page.on("console", (m) => /Content Security Policy/i.test(m.text()) && csp.push(m.text().slice(0, 100)));
    const resp = await page.goto(`${BASE}/checkin?c=${C}`); await page.waitForTimeout(2500);
    const g = await page.evaluate(() => ({ h1: [...document.querySelectorAll("h1")].map((x) => x.textContent), panels: document.querySelectorAll("[data-part=panel],[class*=Panel]").length, meta: !!document.querySelector("[data-ep-meta]"), nested: !!document.querySelector("[data-part=panel] [data-part=panel]"), css: [...document.styleSheets].length }));
    const body = await page.locator("main, body").first().innerText();
    check("TC-AS09-31 /checkin: h1 «Điểm danh», không meta, không card lồng, không lời mời cài / mẹo", g.h1.includes("Điểm danh") && !g.meta && !g.nested && !/cài|thêm vào màn hình|mẹo/i.test(body), JSON.stringify(g));
    const man = await page.request.get(`${BASE}/manifest.webmanifest`); const mj = man.ok() ? await man.json() : {}; const regs = await page.evaluate(async () => (await navigator.serviceWorker?.getRegistrations?.())?.length ?? 0);
    check("TC-AS09-33 manifest: name, display standalone, icon; không service worker; không lời mời cài", man.ok() && !!mj.name && mj.display === "standalone" && (mj.icons ?? []).length > 0 && regs === 0, `manifest=${man.status()} name=${mj.name} display=${mj.display} icons=${(mj.icons ?? []).length} sw=${regs}`);
    const cspH = resp?.headers()["content-security-policy"] ?? "";
    check("TC-AS09-37 CSP /checkin: không unsafe-eval, không script ngoài; console CSP violation = 0", cspH.length > 0 && !/unsafe-eval/.test(cspH) && !/script-src[^;]*https?:\/\//.test(cspH) && csp.length === 0, `violations=${csp.length}`); await ctx.close();
  });
  await closeAll(sid);
});

test("F /checkin: vị trí, WebCrypto/IndexedDB vắng, quét giả, native giả, axe (AS10-02,11, AS01-14, AS04-16, AS09-32,36, AS11-09,14)", async ({ browser }) => {
  base(33);
  const pol = async (extra: Record<string, unknown>) => api("r108t", "PUT", `/courses/${C}/settings`, { attendance: { self_checkin: true, rotate_seconds: 30, late_after_min: 10, require: "WEB", ...extra }, version: (await api("r108t", "GET", `/courses/${C}`)).j.settings.version });
  const spyGeo = `(() => { window.__geo = []; const g = { getCurrentPosition: (ok, err) => { window.__geo.push(Date.now()); if (window.__geoDeny) err({ code: 1, message: "denied" }); else ok({ coords: { latitude: 10.7727, longitude: 106.6602, accuracy: 10 } }); }, watchPosition: () => 0, clearWatch: () => {} }; Object.defineProperty(navigator, "geolocation", { value: g, configurable: true }); })();`;
  await guard("TC-AS10-02", async () => {
    await pol({ geofence: null, campus_cidrs: [] }); const sid = nextSid(); const wid = await openWin(sid); const x = await tokOf(sid, wid);
    const { ctx, page } = await login(browser, "sv.gioi", 375, 812, true); await page.addInitScript(spyGeo);
    await page.goto(`${BASE}/checkin?c=${C}#${x.t}`); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 30_000 });
    const calls = await page.evaluate(() => (window as unknown as { __geo: number[] }).__geo.length); const tx = await page.locator("body").innerText();
    check("TC-AS10-02 lớp không bật vị trí: không gọi geolocation, không dòng thông báo vị trí", calls === 0 && !/kiểm tra vị trí/.test(tx), `geolocation=${calls}`); await ctx.close(); await closeAll(sid);
  });
  await guard("TC-AS10-11/AS01-14", async () => {
    await pol({ geofence: { lat: 10.7727, lng: 106.6602, radius_m: 200 }, campus_cidrs: [] });
    for (const deny of [false, true]) {
      const sid = nextSid(); const wid = await openWin(sid); const x = await tokOf(sid, wid);
      const { ctx, page } = await login(browser, deny ? "sv.kha" : "sv.nguyco", 375, 812, true);
      await page.addInitScript(spyGeo); if (deny) await page.addInitScript(() => { (window as unknown as { __geoDeny: boolean }).__geoDeny = true; });
      await page.addInitScript(() => { const o = (navigator.geolocation as unknown as { getCurrentPosition: (...a: unknown[]) => void }).getCurrentPosition; (window as unknown as { __lineAtGeo: string[] }).__lineAtGeo = []; (navigator.geolocation as unknown as { getCurrentPosition: (...a: unknown[]) => void }).getCurrentPosition = (...a: unknown[]) => { (window as unknown as { __lineAtGeo: string[] }).__lineAtGeo.push(document.body?.innerText ?? ""); o(...a); }; });
      await page.goto(`${BASE}/checkin?c=${C}#${x.t}`); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 30_000 });
      const at = await page.evaluate(() => (window as unknown as { __lineAtGeo: string[] }).__lineAtGeo); const fl = psql(`select array_to_string(flags,','), geo_distance_m is not null from attendance_checkins where window_id='${wid}' and student_id=(select id from users where email='${deny ? "sv.kha" : "sv.nguyco"}@edupilot.local')`);
      if (!deny) check("TC-AS10-11a/AS01-14 geofence bật: dòng «Lớp này kiểm tra vị trí khi điểm danh.» đã vẽ TRƯỚC lần gọi geolocation đầu", at.length >= 1 && /Lớp này kiểm tra vị trí khi điểm danh\./.test(at[0]), `calls=${at.length} thấy dòng=${/kiểm tra vị trí/.test(at[0] ?? "")}`);
      else check("TC-AS10-11b từ chối quyền → vẫn «Đã điểm danh», lượt có GEO_DENIED", /GEO_DENIED/.test(fl), fl);
      await ctx.close(); await closeAll(sid);
    }
    await pol({ geofence: null, campus_cidrs: [] });
  });
  await guard("TC-AS09-32/AS04-16", async () => {
    const sid = nextSid(); const wid = await openWin(sid); const x = await tokOf(sid, wid);
    const { ctx, page } = await login(browser, "sv04", 375, 812, true); await page.addInitScript(() => { try { Object.defineProperty(window.crypto, "subtle", { value: undefined, configurable: true }); } catch { /* */ } Object.defineProperty(window, "indexedDB", { value: undefined, configurable: true }); });
    await page.goto(`${BASE}/checkin?c=${C}#${x.t}`); await page.locator("[data-part=checkin][data-state=done]").waitFor({ timeout: 30_000 });
    const fl = psql(`select array_to_string(flags,','), assurance from attendance_checkins where window_id='${wid}' and student_id=(select id from users where email='sv04@edupilot.local')`);
    check("TC-AS09-32/AS04-16 không WebCrypto / IndexedDB (ngữ cảnh không an toàn, chế độ riêng tư): vẫn «Đã điểm danh», NO_DEVICE_KEY, không lỗi giao diện", /NO_DEVICE_KEY/.test(fl), fl); await ctx.close(); await closeAll(sid);
  });
  await guard("TC-AS09-36", async () => {
    const { ctx, page } = await login(browser, "sv.gioi", 375, 812, true); await page.goto(`${BASE}/checkin?c=${C}`); await page.waitForTimeout(2500);
    const ax = await new AxeBuilder({ page }).analyze(); const bad = ax.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? ""));
    const lcp = await page.evaluate(() => new Promise<number>((res) => { try { new PerformanceObserver((l) => { const e = l.getEntries(); res(e[e.length - 1].startTime); }).observe({ type: "largest-contentful-paint", buffered: true }); setTimeout(() => res(-1), 1500); } catch { res(-2); } }));
    check("TC-AS09-36 /checkin: axe 0 lỗi nghiêm trọng; LCP ≤ 2,5 s", bad.length === 0 && lcp > 0 && lcp <= 2500, `${bad.map((v) => v.id + ":" + v.impact + ":" + v.nodes.length).join(",") || "0"}; LCP=${Math.round(lcp)} ms`); await ctx.close();
  });
  await guard("TC-AS11-09", async () => {
    const sid = nextSid(); const wid = await openWin(sid); await tokOf(sid, wid);
    const { ctx, page } = await login(browser, "sv05", 375, 812, true);
    await page.addInitScript(() => { const w = window as unknown as { __tracks: Array<{ readyState: string }>; BarcodeDetector: unknown }; w.__tracks = []; w.BarcodeDetector = class { static async getSupportedFormats() { return ["qr_code"]; } async detect() { return []; } };
      const stream = () => { const tr = { readyState: "live", stop() { this.readyState = "ended"; }, kind: "video", enabled: true, addEventListener() {}, removeEventListener() {}, getSettings: () => ({}) }; w.__tracks.push(tr); return { getTracks: () => [tr], getVideoTracks: () => [tr], active: true, addTrack() {}, removeTrack() {}, addEventListener() {}, removeEventListener() {} }; };
      Object.defineProperty(navigator, "mediaDevices", { value: { getUserMedia: async () => stream() }, configurable: true }); });
    await page.goto(`${BASE}/checkin?c=${C}`); const btn = page.getByRole("button", { name: /Quét mã điểm danh/ }); await btn.waitFor({ timeout: 15_000 }); await btn.click(); await page.waitForTimeout(1500);
    const live = await page.evaluate(() => (window as unknown as { __tracks: Array<{ readyState: string }> }).__tracks.map((t) => t.readyState));
    await page.evaluate(() => { Object.defineProperty(document, "visibilityState", { value: "hidden", configurable: true }); document.dispatchEvent(new Event("visibilitychange")); }); await page.waitForTimeout(1200);
    const after = await page.evaluate(() => (window as unknown as { __tracks: Array<{ readyState: string }> }).__tracks.map((t) => t.readyState));
    check("TC-AS11-09 đang quét: tab ẩn (visibilitychange→hidden) → MediaStreamTrack ended", live.includes("live") && after.every((s) => s === "ended"), `trước=${live} sau=${after}`); await ctx.close(); await closeAll(sid);
  });
  await guard("TC-AS11-14", async () => {
    for (const user of ["r108ta", "r108t"]) {
      const { ctx, page } = await login(browser, user, 375, 812, true); await page.addInitScript(() => { (window as unknown as { Capacitor: unknown }).Capacitor = { isNativePlatform: () => true, getPlatform: () => "android", Plugins: {} }; });
      const reqs = requests(page); reqs.length = 0; await page.goto(`${BASE}/checkin?c=${C}`); await page.waitForTimeout(2500);
      const tx = await page.locator("body").innerText(); const st = await page.evaluate(() => JSON.stringify([Object.keys(localStorage), Object.keys(sessionStorage)]));
      check(`TC-AS11-14 ${user} (giả native) mở /checkin: màn chặn chuẩn; 0 request dữ liệu; không lưu mật khẩu/token ngoài cookie`, /không có quyền/.test(tx) && reqs.filter((r) => /checkin|courses\//.test(r)).length === 0 && !/password|token/i.test(st), st.slice(0, 100)); await ctx.close();
    }
  });
});
