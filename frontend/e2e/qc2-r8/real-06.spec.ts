import { expect, test, type Page, type BrowserContext } from "@playwright/test";
import { cid, go, log, login, shot, IDS } from "./lib";
// US-P6-06 UI thật trên lớp riêng QD8R6U (bản sao sạch lớp 1). Tên test bắt đầu bằng [TC-…] để gom kết quả.
test.setTimeout(100_000);
const C = "QD8R6U", I = IDS[C], MID = I.items.midterm, FIN = I.items.final, WK = I.items.weekly;
const nm = (a: string) => I.names[a] as string;
let page: Page, ctx: BrowserContext;
test.beforeAll(async ({ browser }) => { ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); page = await ctx.newPage(); await login(page, "teacher"); });
const open = () => go(page, `/gradebook?course=${cid(C)}`);
const rowL = (a: string) => page.locator("[data-part=grade-row]").filter({ hasText: nm(a) }).first();
const scrollBox = (top: number) => page.evaluate((t) => { const r = document.querySelector("[data-part=grade-row]"); let e: any = r; while (e && !(e.scrollHeight > e.clientHeight + 5 && /(auto|scroll)/.test(getComputedStyle(e).overflowY))) e = e.parentElement; if (e) e.scrollTop = t; return !!e; }, top);
async function rowOf(a: string) { for (let t = 0; t <= 4000; t += 250) { if (await rowL(a).count()) return rowL(a); await scrollBox(t); await page.waitForTimeout(250); } await scrollBox(0); return rowL(a); }
const row = rowOf;
const cellOf = (a: string, item: string) => page.locator(`input[data-cell="${I.students[a]}:${item}"]`);
const puts: any[] = [];
test("[TC-P6-06-01,TC-P6-06-02,TC-P6-06-04] bố cục, hàng A/B/C, ghi chú chính thức", async () => {
  await open(); await shot(page, "06-gradebook-1440");
  await expect(page.locator("main [data-ep-panel]")).toHaveCount(1);
  await expect(page.locator("h1")).toHaveText("Sổ điểm");
  await expect(page.getByRole("link", { name: "Xem công thức" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Chốt điểm" })).toBeVisible();
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  const items = await page.getByRole("menuitem").allInnerTexts(); log("06", `menu ⋯: ${JSON.stringify(items)}`);
  expect(items.join("|")).toContain("Nhập từ XLSX"); expect(items.join("|")).toContain("Xuất XLSX"); expect(items.join("|")).not.toContain("Mở khoá");
  await page.keyboard.press("Escape");
  expect(await page.locator("thead th").first().evaluate((e) => getComputedStyle(e).position)).toBe("sticky");
  expect(await page.locator("table").count()).toBeGreaterThan(0);
  const ta = (await (await row("sv.gioi")).innerText()).replace(/\s+/g, " "), tb = (await (await row("sv.kha")).innerText()).replace(/\s+/g, " "), tc = (await (await row("sv.nguyco")).innerText()).replace(/\s+/g, " "); await scrollBox(0);
  log("06", `A: ${ta}`); log("06", `B: ${tb}`); log("06", `C: ${tc}`);
  for (const [t, ex] of [[ta, ["1,00", "10,00", "9,5", "A+"]], [tb, ["0,50", "8,25", "7,0", "B"]], [tc, ["0,25", "1,50", "1,25", "3,0", "F"]]] as const) for (const e of ex) expect(t).toContain(e);
  const note = page.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường."); await expect(note).toHaveCount(1);
  const nb = await note.boundingBox(), tbx = await page.locator("table").first().boundingBox(); log("06", `note y=${nb?.y} table bottom=${(tbx?.y ?? 0) + (tbx?.height ?? 0)}`);
  expect(await page.locator("[title]:has-text('Điểm chính thức')").count()).toBe(0);
});
test("[TC-P6-06-06,TC-P6-06-07,TC-P6-06-08] Enter / Tab / Esc", async () => {
  await open(); const r: string[] = []; page.on("request", (q) => { if (q.method() === "PUT" && q.url().includes("grade-entries")) puts.push(q.postData()); });
  puts.length = 0;
  const first = nm("sv.gioi"); const rows = page.locator("[data-part=grade-row]"); const n0 = await rows.nth(0).innerText();
  const c0 = rows.nth(0).locator("input[data-cell]").first(), c1 = rows.nth(1).locator("input[data-cell]").first();
  const before = await c0.inputValue(); const n0v = before.startsWith("8,5") ? "7,5" : "8,5"; await c0.focus(); await c0.fill(n0v); await c0.press("Enter");
  await expect.poll(() => puts.length).toBe(1); log("06", `PUT body ${puts[0]}`);
  const b = JSON.parse(puts[0]); expect(b.score).toBe(n0v.replace(",", ".")); expect(typeof b.version).toBe("number");
  await expect(c1).toBeFocused(); await expect(c0).toHaveValue(n0v + "0");
  const b1 = await c1.inputValue(); const n1v = (parseFloat(b1.replace(",", ".")) + 0.25).toFixed(2).replace(".", ","); await c1.focus(); await c1.fill(n1v); await c1.press("Tab"); await expect.poll(() => puts.length).toBe(2);
  await expect(rows.nth(1).locator("input[data-cell]").nth(1)).toBeFocused(); await expect(c1).toHaveValue(n1v);
  const c2 = rows.nth(2).locator("input[data-cell]").first(); const v2 = await c2.inputValue(); await c2.focus(); await c2.fill("9"); await c2.press("Escape");
  await page.waitForTimeout(800); expect(puts.length).toBe(2); await expect(c2).toHaveValue(v2); log("06", `trước ${before}`);
});
test("[TC-P6-06-09,TC-P6-06-10,TC-P6-06-11,TC-P6-06-12] lỗi tại ô, chuỗi sai, xoá ô", async () => {
  await open(); const rows = page.locator("[data-part=grade-row]"); const a = rows.nth(3).locator("input[data-cell]").first();
  const r5xx: number[] = []; page.on("response", (r) => { if (r.url().includes("grade-entries") && r.status() >= 500) r5xx.push(r.status()); });
  const orig = await a.inputValue();
  await a.fill("10,5"); await a.press("Enter"); await expect(page.getByRole("alert").filter({ hasText: "Điểm ngoài thang 0–10" })).toBeVisible(); await expect(a).toHaveValue("10,5");
  await a.fill("8,333"); await a.press("Enter"); await expect(page.getByRole("alert").filter({ hasText: "Điểm có quá 2 chữ số thập phân" })).toBeVisible(); await expect(a).toHaveValue("8,333");
  for (const s of ["abc", "-1", "1e2", "8,,5", "1000"]) { await a.fill(s); await a.press("Enter"); await page.waitForTimeout(500); log("06", `chuỗi '${s}' → alert: ${JSON.stringify(await page.getByRole("alert").allInnerTexts())} value=${await a.inputValue()}`); expect(await a.inputValue()).toBe(s); }
  expect(r5xx).toEqual([]); await a.fill(orig === "" ? "" : orig); await a.press("Escape");
  // xoá ô: dùng hàng 4
  const b = rows.nth(4).locator("input[data-cell]").first(); const ov = (await b.inputValue()).replace(",", "."); await b.fill(""); await b.press("Enter");
  await expect(rows.nth(4).locator("[data-part=row-status]")).toHaveText("Thiếu điểm", { timeout: 15000 }); log("06", `xoá ô: status ${await rows.nth(4).locator("[data-part=row-status]").innerText()}`);
  await b.fill(ov); await b.press("Enter"); await expect(rows.nth(4).locator("[data-part=row-status]")).not.toHaveText("Thiếu điểm", { timeout: 15000 });
});
test("[TC-P6-06-13] ô EXAM không sửa được", async () => {
  await open(); const r = page.locator("[data-part=grade-row]").nth(1); const tds = r.locator("td"); const html = await tds.nth(2).innerHTML();
  log("06", `ô EXAM: ${html.slice(0, 200)}`); expect(await tds.nth(2).locator("input").count()).toBe(0);
  await tds.nth(2).hover(); await page.waitForTimeout(900); const tip = await page.getByRole("tooltip").allInnerTexts(); const title = await tds.nth(2).locator("[title]").first().getAttribute("title").catch(() => null);
  log("06", `tooltip ô EXAM: ${JSON.stringify(tip)} title=${title}`); expect(JSON.stringify(tip) + String(title) + html).toContain("Điểm lấy từ bài thi");
});
test("[TC-P6-06-15] ô tính hiện … trong lúc chờ", async () => {
  await open(); await page.route("**/grade-entries", async (rt) => { if (rt.request().method() === "PUT") await new Promise((s) => setTimeout(s, 1500)); await rt.continue(); });
  await scrollBox(0); const rb = await row("sv.kha"); const c = cellOf("sv.kha", FIN); await expect(c).toHaveValue("6,50"); await c.fill("8"); await c.press("Enter");
  await expect(rb.locator("[data-part=row-status]")).toHaveText("…"); const mid = (await rb.innerText()).replace(/\s+/g, " "); log("06", `đang chờ: ${mid}`);
  await expect(rb.locator("[data-part=row-status]")).not.toHaveText("…", { timeout: 10000 }); const end = (await rb.innerText()).replace(/\s+/g, " "); log("06", `xong: ${end}`);
  expect(end).toContain("8,25"); expect(end).toContain("8,0"); expect(end).toContain("B+");
  await page.unroute("**/grade-entries"); await c.fill("6,5"); await c.press("Enter"); await expect(rb.locator("[data-part=row-status]")).not.toHaveText("…", { timeout: 10000 });
});
test("[TC-P6-06-18,TC-P6-06-19,TC-P6-06-20] xung đột 409: Giữ của tôi / Dùng của họ", async ({ browser }) => {
  const mk = async () => { const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage(); await login(p, "teacher"); await go(p, `/gradebook?course=${cid(C)}`); await p.locator("[data-part=grade-row]").first().waitFor(); return p; };
  const p1 = await mk(), p2 = await mk(); const B = I.students["sv.kha"];
  const a1 = p1.locator(`input[data-cell="${B}:${FIN}"]`), a2 = p2.locator(`input[data-cell="${B}:${FIN}"]`); const pop = p2.locator("[data-part=cell-conflict]");
  await a1.focus(); await a1.fill("7,5"); await a1.press("Enter"); await p1.waitForTimeout(2500);
  await a2.focus(); await a2.fill("6"); await a2.press("Enter"); await expect(pop).toBeVisible({ timeout: 15000 });
  const txt = (await pop.innerText()).replace(/\s+/g, " "); log("06", `popover: ${txt}`); expect(txt).toContain("vừa nhập 7,5"); await expect(a2).not.toHaveValue(""); await shot(p2, "06-conflict");
  await pop.getByRole("button", { name: "Giữ của tôi" }).click(); await p2.waitForTimeout(2500); log("06", `sau Giữ của tôi: ô ${await a2.inputValue()}`); expect(await a2.inputValue()).toBe("6,00");
  // vòng 2: p2 đang cập nhật nhất, p1 cũ → p2 nhập 5, p1 (cũ) nhập 8 → xung đột ở p1
  const pop1 = p1.locator("[data-part=cell-conflict]");
  await a2.focus(); await a2.fill("5"); await a2.press("Enter"); await p2.waitForTimeout(2500); await a1.focus(); await a1.fill("8"); await a1.press("Enter"); await expect(pop1).toBeVisible({ timeout: 15000 });
  await pop1.getByRole("button", { name: "Dùng của họ" }).click(); await expect(pop1).toHaveCount(0); await expect(a1).toHaveValue("5,00");
  await a2.focus(); await a2.fill("6,5"); await a2.press("Enter"); await p2.waitForTimeout(1500);
});
test("[TC-P6-06-23,TC-P6-06-24] Giải trình B + Căn cứ", async () => {
  await open(); await page.getByRole("button", { name: `Hành động — ${nm("sv.kha")}` }).click(); await page.getByRole("menuitem", { name: "Giải trình" }).click();
  const calc = page.locator("[data-part=grade-calculation]"); await expect(calc).toBeVisible(); const t = (await calc.innerText()).replace(/\s+/g, " "); log("06", `giải trình B: ${t.slice(0, 900)}`); await shot(page, "06-explain-B");
  for (const e of ["Điểm quá trình trước điểm cộng / trừ", "= 7,75", "= 0,5", "= 8,25", "= 7,2", "do giảng viên đặt"]) expect(t).toContain(e);
  const btns = await calc.getByRole("button", { name: /Căn cứ/ }).allInnerTexts(); log("06", `Căn cứ: ${JSON.stringify(btns)}`);
  await calc.getByRole("button", { name: /Căn cứ · tr\. 2/ }).first().click(); await page.waitForTimeout(600); log("06", `quote: ${(await calc.innerText()).replace(/\s+/g, " ").slice(-300)}`);
  await page.keyboard.press("Escape");
});
test("[TC-P6-06-28] Lịch sử ở menu dòng", async () => {
  await open(); const hist = page.getByRole("button", { name: "Lịch lịch sử" }); expect(await page.getByRole("button", { name: /^Lịch sử$/ }).count()).toBe(0);
  await page.getByRole("button", { name: `Hành động — ${nm("sv.kha")}` }).click(); await expect(page.getByRole("menuitem", { name: "Lịch sử" })).toBeVisible(); await page.getByRole("menuitem", { name: "Lịch sử" }).click(); await page.waitForTimeout(1500);
  const t = (await page.getByRole("dialog").innerText().catch(() => "")).replace(/\s+/g, " "); log("06", `lịch sử B: ${t.slice(0, 400)}`); expect(t.length).toBeGreaterThan(5); await page.keyboard.press("Escape");
});
test("[TC-P6-06-47,TC-P6-06-48] công bố cột giữa kỳ (chấm tròn Đã công bố) / cột EXAM không có menu", async () => {
  await open(); const head = page.locator("thead th").filter({ hasText: "Giữa kỳ" }); const m = page.getByRole("button", { name: "Cột Giữa kỳ" });
  await expect(head.locator("[aria-label='Đã công bố']")).toHaveCount(1);
  await m.click(); expect(await page.getByRole("menuitem").allInnerTexts()).toEqual(["Gỡ công bố"]); await page.getByRole("menuitem", { name: "Gỡ công bố" }).click(); await expect(head.locator("[aria-label='Đã công bố']")).toHaveCount(0, { timeout: 10000 });
  await m.click(); await page.getByRole("menuitem", { name: "Công bố cho sinh viên" }).click(); await expect(head.locator("[aria-label='Đã công bố']")).toHaveCount(1, { timeout: 10000 });
  expect(await page.getByRole("button", { name: "Cột Kiểm tra hằng tuần" }).count()).toBe(0);
});
test("[TC-P6-06-39] offline: xếp hàng rồi tự gửi lại", async () => {
  await open(); const a = page.locator("[data-part=grade-row]").nth(0).locator("input[data-cell]").first(), b = cellOf("sv.kha", MID); const va = await a.inputValue(), vb = await b.inputValue();
  await ctx.setOffline(true); await a.fill("4"); await a.press("Enter"); await b.fill("4,5"); await b.press("Enter"); await page.waitForTimeout(1500);
  log("06", `offline: nhãn ${JSON.stringify(await page.getByText(/Chưa lưu/).allInnerTexts())}`); expect(await page.getByText(/Chưa lưu/).count()).toBeGreaterThan(0);
  await ctx.setOffline(false); await expect(page.getByText(/Chưa lưu/)).toHaveCount(0, { timeout: 30000 }); await page.waitForTimeout(1500);
  await a.fill(va.replace(",", ".")); await a.press("Enter"); await b.fill(vb.replace(",", ".")); await b.press("Enter"); await page.waitForTimeout(1500);
});
test("[TC-P6-06-41] lỗi 500 / chậm: skeleton rồi lỗi + Thử lại", async () => {
  await page.route("**/api/v1/courses/*/gradebook?*", async (rt) => { await new Promise((s) => setTimeout(s, 3000)); await rt.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi hệ thống." }) }); });
  await page.goto(`/gradebook?course=${cid(C)}`, { waitUntil: "commit" }); let sk = 0; for (let i = 0; i < 12 && !sk; i++) { await page.waitForTimeout(250); sk = await page.locator("[class*=skeleton],[data-skeleton],[aria-busy=true]").count(); } log("06", `skeleton khi tải: ${sk}`);
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible({ timeout: 20000 }); log("06", `lỗi: ${(await page.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 200)}`); expect(sk).toBeGreaterThan(0);
  await page.unroute("**/api/v1/courses/*/gradebook?*"); await page.getByRole("button", { name: "Thử lại" }).click(); await expect(page.locator("[data-part=grade-row]").first()).toBeVisible({ timeout: 20000 });
});
test("[TC-P6-06-60] TA: chỉ đọc", async ({ browser }) => {
  const c2 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c2.newPage(); await login(p, "ta"); await go(p, `/gradebook?course=${cid(C)}`);
  await expect(p.locator("[data-part=grade-row]").first()).toBeVisible({ timeout: 20000 });
  expect(await p.locator("input[data-cell]").count()).toBe(0);
  for (const nme of ["Chốt điểm", "Nhập từ XLSX", "Xuất XLSX", "Mở khoá", "Công bố cho sinh viên"]) expect(await p.getByRole("button", { name: nme }).count() + await p.getByRole("menuitem", { name: nme }).count()).toBe(0);
  expect(await p.getByRole("button", { name: /^Cột / }).count()).toBe(0);
  const bn = (await p.locator("[data-part=grade-row]").first().getByRole("button", { name: /^Hành động/ }).first().getAttribute("aria-label")) ?? ""; await p.getByRole("button", { name: bn }).click();
  const it = await p.getByRole("menuitem").allInnerTexts(); log("06", `TA menu dòng: ${JSON.stringify(it)}`); expect(it.join("|")).toContain("Giải trình"); expect(it.join("|")).toContain("Lịch sử"); await shot(p, "06-ta"); await c2.close();
});
