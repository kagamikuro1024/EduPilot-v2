import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { EXAMCOL, FINAL, WEEKLY, gradebookApi, student as gb, type Opts } from "./support/gradebook-fixtures";
import { settleGoto } from "./support/hydrate";
import { asDemo } from "./support/session";

// QC FEAT-uiux-91 đợt 4 trên harness e2e (gateway giả, dữ liệu cố định): Sổ điểm UX16-1/8/9/10/18.
const desktop = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
test.beforeEach(({ page }) => settleGoto(page));
async function sheet(page: Page, context: BrowserContext, opts: Opts = {}, preload?: (api: Awaited<ReturnType<typeof gradebookApi>>) => void) {
  await asDemo(context, "teacher");
  const api = await gradebookApi(page, opts);
  preload?.(api);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/gradebook");
  await expect(page.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
  await page.locator("[data-part=grade-row]").first().waitFor();
  return api;
}
const E = (score: string) => ({ score, version: 1, source: "MANUAL" });
const key = (i: number, item: string) => `${gb(i).id}:${item}`;
const colIndex = async (page: Page, name: RegExp) => (await page.locator("thead th").allInnerTexts()).findIndex((t) => name.test(t.trim()));
const cellTexts = async (page: Page, col: number, n = 3) => {
  const out: string[] = [];
  for (let i = 0; i < n; i++) {
    const td = page.locator("[data-part=grade-row]").nth(i).locator("td").nth(col);
    const inp = td.locator("input");
    out.push((await inp.count()) ? await inp.inputValue() : (await td.innerText()).trim());
  }
  return out;
};

test("TC-UX16-1 cột Quá trình: 6 / 5,5 / 7,915 → 6,000 / 5,500 / 7,915; ô nhập giữ nguyên chữ người dùng", async ({ page, context }, info) => {
  desktop(info);
  await sheet(page, context, {}, (api) => {
    [["6", "7"], ["5.5", "8.5"], ["7.915", "9"]].forEach(([w, f], k) => { api.entries.set(key(k + 1, WEEKLY), E(w)); api.entries.set(key(k + 1, FINAL), E(f)); });
  });
  const qc = await colIndex(page, /Quá trình/), hp = await colIndex(page, /Điểm học phần/), wc = await colIndex(page, /Kiểm tra tuần/), fc = await colIndex(page, /Thi cuối kỳ/);
  const q = await cellTexts(page, qc), h = await cellTexts(page, hp), w = await cellTexts(page, wc), f = await cellTexts(page, fc);
  console.log("INFO Quá trình:", JSON.stringify(q), "| Điểm học phần:", JSON.stringify(h), "| ô nhập tuần:", JSON.stringify(w), "| ô nhập cuối kỳ:", JSON.stringify(f));
  expect(q, "AC1 ví dụ của spec").toEqual(["6,000", "5,500", "7,915"]);
  expect(h.every((x) => /^\d+,\d{2,}$/.test(x)), "Điểm học phần ≥ 2 chữ số lẻ").toBe(true);
  expect.soft(f, "TC-UX16-1: cột Thi cuối kỳ 7,00 / 8,50 / 9,00").toEqual(["7,00", "8,50", "9,00"]);
  expect.soft(w, "cột Kiểm tra tuần đệm tới 3 chữ số lẻ").toEqual(["6,000", "5,500", "7,915"]);
  const style = await page.locator("[data-part=grade-row]").first().locator("td").nth(qc).evaluate((e) => ({ a: getComputedStyle(e).textAlign, n: getComputedStyle(e).fontVariantNumeric }));
  expect(style.a).toBe("right");
  expect(style.n).toContain("tabular-nums");
});

test("TC-UX16-8 1.000 sinh viên: DOM chỉ giữ ≤ 80 hàng (ảo hoá), cuộn xuống vẫn có hàng", async ({ page, context }, info) => {
  desktop(info);
  await sheet(page, context, { students: 1000 });
  const n0 = await page.locator("[data-part=grade-row]").count();
  const sc = await page.evaluate(() => { const e = [...document.querySelectorAll("main *")].find((x) => /(auto|scroll)/.test(getComputedStyle(x).overflowY) && x.scrollHeight > x.clientHeight) as HTMLElement; e.scrollTop = e.scrollHeight / 2; return Math.round(e.scrollHeight); });
  await page.waitForTimeout(800);
  const n1 = await page.locator("[data-part=grade-row]").count();
  console.log("INFO số hàng DOM:", n0, "→", n1, "| scrollHeight:", sc);
  expect(n0).toBeLessThanOrEqual(80);
  expect(n1).toBeLessThanOrEqual(80);
  expect(n1).toBeGreaterThan(0);
});

test("TC-UX16-9/10/11 Thiếu điểm: hàng thiếu có chữ, hàng đủ trống, liên kết lọc; không hàng thiếu thì ẩn cột; không có «Đủ»", async ({ page, context }, info) => {
  desktop(info);
  await sheet(page, context, { students: 6 }, (api) => {
    for (const i of [1, 2, 3, 4]) { api.entries.set(key(i, WEEKLY), E("7")); api.entries.set(key(i, FINAL), E("8")); }
    api.entries.set(key(5, WEEKLY), E("7")); // thiếu cuối kỳ; SV 6 thiếu cả hai
  });
  const text = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  const thieu = text.match(/Thiếu điểm: (\d+) sinh viên/);
  console.log("INFO tiêu đề phụ:", thieu?.[0], "| trạng thái hàng:", JSON.stringify(await page.locator("[data-part=row-status]").allInnerTexts()));
  expect(thieu?.[1], "N đúng").toBe("2");
  expect(await page.getByText("Đủ", { exact: true }).count(), "UX16-11").toBe(0);
  const st = await page.locator("[data-part=row-status]").allInnerTexts();
  expect(st.slice(0, 4).every((s) => s.trim() === ""), "hàng đủ để trống").toBe(true);
  expect(st.slice(4).every((s) => /Thiếu điểm/.test(s)), "hàng thiếu có «Thiếu điểm»").toBe(true);
  const link = page.getByRole("link", { name: /Thiếu điểm: 2/ }).or(page.getByRole("button", { name: /Thiếu điểm: 2/ })).first();
  await link.click();
  await page.waitForTimeout(600);
  console.log("INFO sau lọc: số hàng", await page.locator("[data-part=grade-row]").count());
  expect(await page.locator("[data-part=grade-row]").count()).toBe(2);
  // UX16-10
  const p2 = await context.newPage();
  settleGoto(p2);
  const ctx2 = await p2.context();
  void ctx2;
  await sheet(p2, context, { students: 4 }, (api) => { for (const i of [1, 2, 3, 4]) { api.entries.set(key(i, WEEKLY), E("7")); api.entries.set(key(i, FINAL), E("8")); } });
  const heads = await p2.locator("thead th").allInnerTexts();
  console.log("INFO cột khi không ai thiếu:", JSON.stringify(heads));
  expect(heads.some((h) => /Trạng thái/.test(h)), "cột Trạng thái ẩn").toBe(false);
  expect(await p2.getByText(/Thiếu điểm:/).count()).toBe(0);
  void EXAMCOL;
});

test("TC-UX16-18 lỗi tải: ApiErrorNotice + Thử lại; sổ đã chốt chỉ đọc", async ({ page, context }, info) => {
  desktop(info);
  await asDemo(context, "teacher");
  let fail = true;
  await gradebookApi(page, { h: { "GET gradebook": (r, next) => (fail ? r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x", trace_id: "t" }) }) : next(r)) } });
  await page.goto("/gradebook");
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible({ timeout: 15_000 });
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await page.locator("[data-part=grade-row]").first().waitFor();
  const p2 = await context.newPage();
  settleGoto(p2);
  await gradebookApi(p2, { locked: true });
  await p2.goto("/gradebook");
  await p2.locator("[data-part=grade-row]").first().waitFor();
  console.log("INFO sổ đã chốt: số input", await p2.locator("main tbody input").count());
  expect(await p2.locator("main tbody input").count(), "chỉ đọc").toBe(0);
});
