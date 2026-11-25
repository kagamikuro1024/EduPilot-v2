import { test, expect, type Browser, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { mkdirSync, appendFileSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.8 (đợt 3): tương tác /attendance của Staff trên stack thật: mở điểm danh, số đầu người, gọi ngẫu nhiên (Drawer), đóng, 422 CHECKIN_UNRESOLVED, ConfirmIrreversible «Đánh vắng», mở vòng 2. 1440 + 375.
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const PW = process.env.QC_PW ?? "";
const C = process.env.QC_CID ?? "";
const SIDS = (process.env.QC_SIDS ?? "").split(",").filter(Boolean);
const OUT = process.env.QC_OUT ?? "/tmp/qc-as-ui";
mkdirSync(OUT, { recursive: true });
const rec = (tc: string, st: "PASS" | "FAIL" | "NOTE", note = "") => { console.log(`${st} ${tc} ${note}`.slice(0, 280)); appendFileSync(`${OUT}/ui.jsonl`, JSON.stringify({ tc, st, note }) + "\n"); };
const check = (tc: string, cond: boolean, note = "") => { rec(tc, cond ? "PASS" : "FAIL", note); return cond; };
async function token(u: string) { const r = await fetch(`${BASE}/api/v1/auth/login`, { method: "POST", headers: { "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) }); return ((await r.json()) as { access_token: string }).access_token; }
async function api(u: string, method: string, path: string, body?: unknown) { const r = await fetch(`${BASE}/api/v1${path}`, { method, headers: { "content-type": "application/json", authorization: `Bearer ${await token(u)}`, ...(method === "POST" ? { "Idempotency-Key": crypto.randomUUID() } : {}) }, body: body ? JSON.stringify(body) : undefined }); const t = await r.text(); return { s: r.status, j: t ? JSON.parse(t) : null }; }
async function login(browser: Browser, w: number, h: number, mobile: boolean) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: mobile, isMobile: mobile, baseURL: BASE });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill("teacher@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  const btn = page.locator("button[aria-label^='Chọn lớp']").first();
  if (!((await btn.getAttribute("aria-label")) ?? "").includes("QC-AS ·")) { await btn.click(); await page.getByRole("menuitem", { name: /^QC-AS · / }).click(); await page.waitForTimeout(1000); }
  return { ctx, page };
}
async function more(page: Page, name: RegExp) {
  const trig = page.getByRole("button", { name: /^Thêm( hành động)?$/ }).first(); await trig.waitFor(); await page.waitForTimeout(500); await trig.click(); await page.waitForTimeout(400);
  await page.getByRole("menuitem", { name }).first().click();
}
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
let idx = 0;

for (const [w, h, mobile] of [[1440, 900, false], [375, 812, true]] as const) {
  test(`/attendance tương tác Staff ${w} px`, async ({ browser }) => {
    const sid = SIDS[idx++]; const tag = `${w}`;
    const { ctx, page } = await login(browser, w, h, mobile);
    await page.goto(`${BASE}/attendance?session=${sid}`);
    await page.getByRole("button", { name: /^(Mở điểm danh|Thêm)$/ }).first().waitFor();
    // mở điểm danh (mở xong tự chuyển sang màn chiếu)
    if (await page.getByRole("button", { name: "Mở điểm danh" }).count()) await page.getByRole("button", { name: "Mở điểm danh" }).first().click(); else await more(page, /^Mở điểm danh$/);
    await page.waitForURL(/\/attendance\/display/, { timeout: 15_000 });
    check(`TC-AS08-02a (${tag}) «Mở điểm danh» → mở cửa sổ rồi chuyển sang /attendance/display?course=…&session=…`, page.url().includes(`session=${sid}`) && page.url().includes(`course=${C}`), page.url());
    await page.goto(`${BASE}/attendance?session=${sid}`);
    await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 10_000 });
    const wins = (await api("teacher", "GET", `/courses/${C}/sessions/${sid}/checkin-windows`)).j.items;
    check(`TC-AS08-02 (${tag}) «Mở điểm danh» → dòng «Đang mở» + API có 1 cửa sổ vòng 1`, wins.length === 1 && wins[0].round === 1, JSON.stringify(wins.map((x: { round: number }) => x.round)));
    const href = await page.getByRole("link", { name: "Mở màn chiếu" }).getAttribute("href");
    check(`TC-AS08-03 (${tag}) liên kết «Mở màn chiếu» → /attendance/display?course=…&session=…`, !!href && href.includes("/attendance/display") && href.includes(`session=${sid}`) && href.includes(`course=${C}`), href ?? "");
    // 4 SV điểm danh
    const d = (await api("teacher", "GET", `/courses/${C}/sessions/${sid}/checkin-windows/${wins[0].id}/display`)).j; const t = d.qr_url.split("#")[1];
    for (const u of ["sv.gioi", "sv.kha", "sv.nguyco", "sv04"]) await api(u, "POST", `/courses/${C}/checkins`, { token: t });
    await expect.poll(async () => (await page.locator("[data-part=window-count]").innerText()).replace(/\s/g, ""), { timeout: 20_000 }).toMatch(/·4\/\d+/);
    check(`TC-AS08-05 (${tag}) đếm «k/n» cập nhật theo lượt (4/…) không tải lại`, true, await page.locator("[data-part=window-count]").innerText());
    const flagTxt = await page.locator("body").innerText();
    check(`TC-AS08-06 (${tag}) hàng đã điểm danh hiện «Điểm danh lúc» + chip «Không xác minh được máy»`, /Điểm danh lúc \d{2}:\d{2}/.test(flagTxt) && flagTxt.includes("Không xác minh được máy"), "");
    await page.screenshot({ path: `${OUT}/att-open-${tag}.png`, fullPage: true });
    // số đầu người
    await more(page, /Nhập số đầu người/);
    const hc = page.getByLabel("Số đầu người"); await hc.waitFor(); await hc.fill("1");
    await expect.poll(async () => (await api("teacher", "GET", `/courses/${C}/sessions/${sid}/checkin-windows`)).j.items[0].headcount, { timeout: 8000 }).toBe(1);
    check(`TC-AS07-10a (${tag}) «Nhập số đầu người» tự lưu (600 ms) → API headcount=1`, true, "");
    await page.reload(); await page.locator("[data-part=window-line]").waitFor(); await page.waitForTimeout(1500);
    check(`TC-AS07-01 (${tag}) 4 lượt > 1 + max(2,…)=3 ⇒ «Số điểm danh nhiều hơn số người đếm»`, (await page.locator("[data-part=over-headcount]").count()) > 0, (await page.locator("[data-part=window-line]").innerText()).replace(/\n/g, " "));
    // gọi ngẫu nhiên
    await more(page, /Gọi ngẫu nhiên 5 người/);
    const dr = page.getByRole("dialog").first(); await dr.waitFor({ timeout: 8000 });
    await expect.poll(async () => (await dr.innerText()), { timeout: 8000 }).toMatch(/Có cờ|Ngẫu nhiên/);
    const drTxt = await dr.innerText();
    check(`TC-AS07-10b (${tag}) Drawer «Gọi ngẫu nhiên»: 4 tên + MSSV + «Có cờ»`, (drTxt.match(/Có cờ/g) ?? []).length === 4 && /\d{8}/.test(drTxt), drTxt.replace(/\n/g, " ").slice(0, 120));
    await page.screenshot({ path: `${OUT}/att-spot-${tag}.png` });
    await page.keyboard.press("Escape"); await page.waitForTimeout(400);
    // đóng
    await page.getByRole("button", { name: "Đóng", exact: true }).click();
    await page.locator("[data-part=window-line][data-state=closed]").waitFor({ timeout: 10_000 });
    const t1 = Date.now(); await expect.poll(async () => await page.locator("[data-part=window-summary]").innerText(), { timeout: 10_000 }).toMatch(/Đã ghi [1-9]\d* có mặt|Đã ghi \d+ có mặt · [1-9]\d* muộn/).catch(() => undefined);
    const sum = await page.locator("[data-part=window-summary]").innerText(); rec(`TC-AS05-17a0 (${tag}) độ trễ cập nhật «Đã ghi…» sau khi bấm Đóng`, "NOTE", `${Date.now() - t1} ms → ${sum}`);
    check(`TC-AS05-17a (${tag}) đóng bằng lưới: «Đã ghi a có mặt · b muộn · u chưa điểm danh» tại chỗ`, /Đã ghi \d+ có mặt · \d+ muộn · \d+ chưa điểm danh/.test(sum), sum);
    // 422 CHECKIN_UNRESOLVED: Lưu điểm danh khi còn người chưa điểm danh
    await page.getByRole("button", { name: /Lưu điểm danh/ }).last().click();
    const note = page.getByText(/Còn \d+ sinh viên chưa điểm danh/); await note.waitFor({ timeout: 10_000 });
    const nUn = Number((await note.innerText()).match(/\d+/)![0]);
    check(`TC-AS05-10c (${tag}) «Lưu điểm danh» khi còn SV chưa điểm danh → 422 CHECKIN_UNRESOLVED hiện «Còn N sinh viên chưa điểm danh» + nút «Đánh vắng N SV…»`, nUn > 0 && (await page.getByRole("button", { name: new RegExp(`Đánh vắng ${nUn} SV chưa điểm danh`) }).count()) === 1, `N=${nUn}; số nút «Đánh vắng N SV…» cùng lúc: ${await page.getByRole("button", { name: new RegExp(`Đánh vắng ${nUn} SV chưa điểm danh`) }).count()}`);
    const bbn = await note.boundingBox(); const lh = await note.evaluate((e) => parseFloat(getComputedStyle(e).lineHeight) || 20);
    check(`TC-AS08-UX1 (${tag}) D-AS-05: «Còn N sinh viên chưa điểm danh» không xuống dòng từng chữ (≤ 2 dòng, rộng ≥ 150 px)`, !!bbn && bbn.width >= 150 && bbn.height <= lh * 2.5, JSON.stringify(bbn));
    const prim = await page.getByRole("button", { name: /^(Mở điểm danh|Lưu điểm danh)/ }).allInnerTexts();
    check(`TC-AS08-UX2 (${tag}) sau đóng, đã có vòng: không còn nút «Mở điểm danh» cạnh «Mở vòng 2»; chỉ một «Lưu điểm danh» chính`, !prim.some((x) => x.trim() === "Mở điểm danh"), prim.join(" | "));
    await page.screenshot({ path: `${OUT}/att-unresolved-${tag}.png`, fullPage: true });
    // ConfirmIrreversible
    await page.getByRole("button", { name: new RegExp(`Đánh vắng ${nUn} SV chưa điểm danh`) }).first().click();
    const dlg = page.getByRole("dialog").last(); await dlg.waitFor();
    const dt = await dlg.innerText();
    check(`TC-AS05-17b (${tag}) hộp xác nhận chuẩn nêu số người: «Đánh vắng ${nUn} sinh viên?» + hệ quả + nút «Đánh vắng»`, dt.includes(`Đánh vắng ${nUn} sinh viên?`) && dt.includes(`${nUn} sinh viên chưa điểm danh thành Vắng.`), dt.replace(/\n/g, " ").slice(0, 150));
    await page.screenshot({ path: `${OUT}/att-confirm-${tag}.png` });
    await dlg.getByRole("button", { name: "Đánh vắng", exact: true }).click();
    await expect.poll(async () => Number((await api("teacher", "GET", `/courses/${C}/sessions/${sid}/attendance`)).j.students.filter((x: { record: { status: string } | null }) => x.record?.status === "ABSENT").length), { timeout: 10_000 }).toBe(nUn);
    check(`TC-AS05-07c (${tag}) xác nhận → ${nUn} dòng ABSENT, thông báo «Còn…» biến mất`, (await page.getByText(/Còn \d+ sinh viên chưa điểm danh/).count()) === 0, "");
    // lưu hoàn tất
    await page.getByRole("button", { name: /Lưu điểm danh/ }).last().click();
    await expect(page.getByText(/Đã lưu điểm danh · \d+ sinh viên/)).toBeVisible({ timeout: 10_000 });
    check(`TC-AS05-11b (${tag}) sau đánh vắng, «Lưu điểm danh» hoàn tất («Đã lưu điểm danh · N sinh viên · vắng M»)`, true, await page.getByText(/Đã lưu điểm danh · \d+ sinh viên/).first().innerText());
    // vòng 2
    await more(page, /Mở vòng 2/);
    await page.waitForURL(/\/attendance\/display/, { timeout: 15_000 }); await page.goto(`${BASE}/attendance?session=${sid}`);
    await page.locator("[data-part=window-line][data-state=open]").waitFor({ timeout: 10_000 });
    check(`TC-AS07-10c (${tag}) «Mở vòng 2» → «Vòng 2»`, (await page.locator("[data-part=window-line]").innerText()).includes("Vòng 2"), "");
    const ax = await new AxeBuilder({ page }).analyze();
    check(`TC-AS08-25b (${tag}) axe /attendance có cửa sổ: 0 lỗi nghiêm trọng`, ax.violations.filter((v) => v.impact === "critical" || v.impact === "serious").length === 0, ax.violations.map((v) => `${v.id}:${v.impact}`).join(","));
    check(`TC-AS08-01b (${tag}) không tràn ngang`, await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), "");
    await page.screenshot({ path: `${OUT}/att-round2-${tag}.png`, fullPage: true });
    await api("teacher", "POST", `/courses/${C}/checkin-windows/${(await api("teacher", "GET", `/courses/${C}/sessions/${sid}/checkin-windows`)).j.items.find((x: { closed_at: string | null }) => !x.closed_at).id}/close`);
    await ctx.close(); void sleep;
  });
}
