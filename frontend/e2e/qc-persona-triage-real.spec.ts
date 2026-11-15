import { test, type Browser, type Page } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC sprint 11 (P-11-12): tái hiện 8 ứng viên lỗi từ PoC persona-eval, CHỈ ĐỌC, lớp 761987 (không đụng KC106).
const PW = process.env.QC_PW ?? "";
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-persona";
const BT03 = process.env.QC_BT03 ?? "";
mkdirSync(OUT, { recursive: true });
const info = (k: string, v: unknown) => console.log(`QCINFO ${k}: ${typeof v === "string" ? v : JSON.stringify(v)}`);

async function login(browser: Browser, user: string, w: number, h: number) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  if ((await page.locator("[data-part=topbar]").innerText()).includes("761988") || (await page.locator("[data-part=topbar]").innerText()).includes("KC106")) {
    await page.locator("[data-part=topbar]").getByText(/761988|KC106/).first().click();
    await page.getByText("761987").first().click();
    await page.waitForTimeout(1500);
  }
  return { ctx, page };
}
const main = async (p: Page) => (await p.locator("main").innerText()).replace(/\n+/g, " | ");

test("sinh viên 375: Hôm nay, thanh điều hướng, trang bài tập, Kết quả của tôi", async ({ browser }, ti) => {
  test.skip(ti.project.name !== "desktop");
  test.setTimeout(150_000);
  const { ctx, page } = await login(browser, "sv.kha", 375, 812);
  await page.goto(`${BASE}/`); await page.waitForTimeout(3000);
  const t = await main(page);
  info("C6 Hôm nay (SV) 375 — 600 ký tự đầu", t.slice(0, 700));
  const btns = await page.locator("main button, main a").allInnerTexts();
  info("C6 nút / liên kết trong main", btns.map((s) => s.trim().replace(/\s+/g, " ")).filter(Boolean).slice(0, 24));
  await page.screenshot({ path: `${OUT}/sv-homnay-375.png`, fullPage: true });
  const nav = await page.locator("nav a, nav button, [data-part=bottom-nav] a").allInnerTexts();
  info("C5 nhãn điều hướng 375", nav.map((s) => s.trim()).filter(Boolean));
  await page.getByText("Thêm", { exact: true }).last().click().catch(() => {});
  await page.waitForTimeout(800);
  info("C5 nhãn sau «Thêm»", (await page.locator("[role=dialog] a, [role=menu] a, aside a, nav a").allInnerTexts()).map((s) => s.trim()).filter(Boolean));
  await page.screenshot({ path: `${OUT}/sv-them-375.png` });
  await page.keyboard.press("Escape");
  if (BT03) {
    await page.goto(`${BASE}/assignments/${BT03}`); await page.waitForTimeout(3500);
    const a = await main(page);
    info("C3 trang bài tập SV: có '# ' / '## ' thô", { thoDauThang: /(^|\s)#{1,3}\s\S/.test(a), mau: (a.match(/#{1,3}\s[^|]{0,40}/g) ?? []).slice(0, 4) });
    info("C4 mốc giờ trong trang", (a.match(/[^|]*(quá hạn|Nộp muộn|Đã nộp|Hạn)[^|]*/g) ?? []).map((s) => s.trim()).slice(0, 8));
    await page.screenshot({ path: `${OUT}/sv-bai-tap-375.png`, fullPage: true });
  }
  await page.goto(`${BASE}/me`); await page.waitForTimeout(3000);
  info("Kết quả của tôi (/me) — công thức thô?", (await main(page)).slice(0, 900));
  await page.screenshot({ path: `${OUT}/sv-ket-qua-375.png`, fullPage: true });
  await ctx.close();
});

test("giảng viên 1440: hàng chờ, tab Bài tập, lọc, nút Tải bài lên, Hôm nay", async ({ browser }, ti) => {
  test.skip(ti.project.name !== "desktop");
  test.setTimeout(180_000);
  const { ctx, page } = await login(browser, "teacher", 1440, 900);
  await page.goto(`${BASE}/`); await page.waitForTimeout(3000);
  const home = await page.locator("main").innerText();
  info("C? Hôm nay (GV): tiêu đề + số mục", { tieuDe: (home.match(/\d+ việc[^\n]*/) ?? [""])[0], soDong: await page.locator("main li, main [data-part=today-item]").count() });
  await page.screenshot({ path: `${OUT}/gv-homnay-1440.png`, fullPage: true });
  await page.goto(`${BASE}/grading`); await page.waitForTimeout(3500);
  const q = await main(page);
  info("C1 hàng chờ mặc định", { chuaCoBaiNop: q.includes("Chưa có bài nộp"), mau: q.slice(0, 500) });
  info("C7 số nút «Tải bài lên»", await page.getByRole("button", { name: /Tải bài lên/ }).count() + await page.getByRole("link", { name: /Tải bài lên/ }).count());
  const cta = page.getByRole("button", { name: /Chấm tất cả/ });
  info("C? «Chấm tất cả»", { co: await cta.count(), khoa: await cta.first().isDisabled().catch(() => null), title: await cta.first().getAttribute("title").catch(() => null), aria: await cta.first().getAttribute("aria-describedby").catch(() => null) });
  await page.screenshot({ path: `${OUT}/gv-hang-cho-1440.png`, fullPage: true });
  // C2: lọc bài tập ở hàng chờ → đổi tab → quay lại
  const sel = page.locator("select").filter({ has: page.locator("option", { hasText: "Bài tập 03" }) }).first();
  if (await sel.count()) {
    await sel.selectOption({ label: (await sel.locator("option", { hasText: "Bài tập 03" }).first().innerText()).trim() });
    await page.waitForTimeout(1200);
    const v1 = await sel.inputValue();
    await page.getByRole("tab", { name: /Bài tập/ }).first().click(); await page.waitForTimeout(1500);
    const tabBT = await main(page);
    info("C8 tab Bài tập — 'Đã công bố' (số lần) + ngữ cảnh", { lan: (tabBT.match(/Đã công bố/g) ?? []).length, mau: (tabBT.match(/[^|]*Đã công bố[^|]*/g) ?? []).map((s) => s.trim()).slice(0, 6) });
    info("C? tab Bài tập — Đã nộp / Tổng", (tabBT.match(/[^|]*Đã nộp[^|]*/g) ?? []).map((s) => s.trim()).slice(0, 4));
    await page.screenshot({ path: `${OUT}/gv-tab-bai-tap-1440.png`, fullPage: true });
    await page.getByRole("tab", { name: /Hàng chờ/ }).first().click(); await page.waitForTimeout(1500);
    const sel2 = page.locator("select").filter({ has: page.locator("option", { hasText: "Bài tập 03" }) }).first();
    info("C2 lọc bài tập sau khi đổi tab rồi quay lại", { truoc: v1, sau: await sel2.inputValue().catch(() => null), giuNguyen: v1 === (await sel2.inputValue().catch(() => null)) });
    await page.screenshot({ path: `${OUT}/gv-quay-lai-hang-cho-1440.png`, fullPage: true });
  } else info("C2", "không thấy ô lọc «Bài tập 03»");
  await ctx.close();
});
