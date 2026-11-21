/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkR (qc2, lượt 6): TC-UX02-6 (chuông, thời gian tương đối) và TC-UX04-4 (chạm không để lại hover) trên stack thật.
import { test } from "@playwright/test";
import { courseId, expect, go, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
test.describe.configure({ mode: "serial" });

const ict = (d: Date) => new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Ho_Chi_Minh", year: "numeric", month: "2-digit", day: "2-digit" }).format(d);
const hhmm = (d: Date) => new Intl.DateTimeFormat("vi-VN", { timeZone: "Asia/Ho_Chi_Minh", hour: "2-digit", minute: "2-digit", hour12: false }).format(d);

test("TC-UX02-6 chuông thông báo: 30 s → «vừa xong», 5 phút → «5 phút trước», 3 giờ → «3 giờ trước» (cùng ngày) hoặc «Hôm qua, HH:mm», 26 giờ → «Hôm qua, HH:mm»", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher");
  const now = Date.now();
  const mk = (i: number, ageMs: number) => ({ id: `00000000-0000-4000-8000-00000000c0${i}`, type: "TICKET_NEW", title: `Mục thử ${i}`, body: null, link: "/inbox", course_id: null, read_at: null, created_at: new Date(now - ageMs).toISOString() });
  const ages = [30_000, 5 * 60_000, 3 * 3600_000, 26 * 3600_000];
  await o.page.route(/\/api\/v1\/notifications(\?.*)?$/, (r: any) => (r.request().method() === "GET" ? r.fulfill({ json: { items: ages.map((a, i) => mk(i + 1, a)), unread_count: 4, next_cursor: null } }) : r.continue()));
  await o.page.goto("/"); await o.page.waitForTimeout(3000);
  await o.page.getByRole("button", { name: /Thông báo/ }).first().click();
  const pop = o.page.locator("[role=dialog], [data-part=notifications], [role=menu], [popover]").filter({ hasText: "Mục thử 1" }).first();
  await expect(pop).toBeVisible({ timeout: 15_000 });
  const rows = await o.page.getByText(/^Mục thử \d$/).evaluateAll((els: Element[]) => els.map((e) => { let n: HTMLElement | null = e as HTMLElement; for (let i = 0; i < 4 && n && !/vừa xong|trước|Hôm qua|Thứ|Chủ Nhật|Hôm nay/.test(n.innerText); i++) n = n.parentElement; return (n?.innerText ?? "").replace(/\s+/g, " "); }));
  console.log("INFO UX02-6 hàng chuông:", JSON.stringify(rows));
  const t3 = new Date(now - 3 * 3600_000), t26 = new Date(now - 26 * 3600_000);
  const sameDay3 = ict(t3) === ict(new Date(now));
  const exp = ["vừa xong", "5 phút trước", sameDay3 ? "3 giờ trước" : `Hôm qua, ${hhmm(t3)}`, `Hôm qua, ${hhmm(t26)}`];
  // 26 giờ trước có thể rơi vào «hôm kia» (> 1 ngày lịch) khi mới qua nửa đêm: dùng quy ước thứ
  const calDiff26 = Math.round((Date.parse(ict(new Date(now)) + "T00:00:00Z") - Date.parse(ict(t26) + "T00:00:00Z")) / 86400000);
  if (calDiff26 >= 2) exp[3] = "Thứ";
  for (let i = 0; i < 4; i++) expect(rows[i], `hàng ${i + 1}: «${exp[i]}»`).toContain(exp[i]);
  console.log("INFO UX02-6 mong đợi:", JSON.stringify(exp));
  expect(rows.join(" "), "không dạng cũ HH:mm dd-MM").not.toMatch(/\d\d:\d\d \d\d-\d\d/);
  await o.ctx.close();
});

test("TC-UX04-4 375 hasTouch: chạm một dòng danh sách rồi không giữ nền hover", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher", 375, 812); const c = await courseId(o.page, "761988");
  expect(await o.page.evaluate(() => matchMedia("(hover: hover)").matches), "thiết bị cảm ứng: (hover: hover) = false").toBe(false);
  await go(o.page, "/threads", c, 3500);
  const rows = o.page.locator("[data-part=thread-list] > li");
  await rows.first().waitFor({ timeout: 30_000 });
  const bg = (i: number) => rows.nth(i).evaluate((e: Element) => { const cs = getComputedStyle(e); const a = e.querySelector("a") as HTMLElement | null; return [cs.backgroundColor, a ? getComputedStyle(a).backgroundColor : ""]; });
  const before = await bg(1);
  // chạm không điều hướng đi: chạm vào ô meta (không phải liên kết)
  const meta = rows.nth(0).locator("[class*=meta]");
  await meta.tap(); await o.page.waitForTimeout(600);
  const after = await bg(0), other = await bg(1);
  console.log("INFO UX04-4 nền dòng 2 (không chạm):", JSON.stringify(before), "| dòng 1 sau chạm:", JSON.stringify(after));
  expect(after, "dòng vừa chạm không giữ nền hover (cùng nền dòng không chạm)").toEqual(other);
  await o.ctx.close();
});

test("TC-UX05-7 sổ điểm lớp chưa có công thức: EmptyState đúng một nút «Thiết lập công thức», không câu «Điểm chính thức …»", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher");
  const mine = ((await (await import("./r91lib")).api(o.page, "/me/courses?limit=100")).body.items as any[]);
  let cid = "";
  for (const i of mine) {
    const st = await (await import("./r91lib")).api(o.page, `/courses/${i.course.id}/grade-scheme/status`);
    if (st.status === 200 && st.body.state !== "CONFIRMED" && !st.body.confirmed_at) { cid = i.course.id; console.log("INFO UX05-7 lớp", i.course.class_code, "state", st.body.state); break; }
  }
  expect(cid, "có lớp chưa xác nhận công thức").not.toBe("");
  await go(o.page, "/gradebook", cid, 3500);
  const main = o.page.locator("main");
  await expect(main.getByText("Chưa có công thức điểm", { exact: true })).toBeVisible({ timeout: 20_000 });
  const acts = await main.getByRole("link", { name: "Thiết lập công thức" }).count() + await main.getByRole("button", { name: "Thiết lập công thức" }).count();
  expect(acts, "đúng một nút chính «Thiết lập công thức»").toBe(1);
  expect(await main.innerText()).not.toContain("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.");
  await o.ctx.close();
});

test("TC-UX01-5 dòng «Giữ»: /students «N sinh viên», /attendance dòng phụ buổi học không mã lớp (#11, #47, #48)", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher"); const c = await courseId(o.page, "761987");
  await go(o.page, "/students", c, 3000);
  const t1 = await o.page.locator("main").innerText();
  const sub = t1.match(/\d+ sinh viên[^\n]*/)?.[0] ?? "";
  console.log("INFO #48 dòng phụ /students:", sub);
  expect(sub).toMatch(/^\d+ sinh viên$/);
  expect(t1).toMatch(/\b\d{8}\b/);
  await go(o.page, "/attendance", c, 3500);
  const t2 = (await o.page.locator("main").innerText()).replace(/\n/g, " ");
  const line = t2.match(/buổi \d+ · [^·]*\d{2}:\d{2}[–-]\d{2}:\d{2} · P\.\d+/i)?.[0] ?? "";
  console.log("INFO #11 /attendance:", line || t2.slice(0, 200));
  expect(line, "dòng phụ buổi học").not.toBe("");
  expect(line).not.toMatch(/\b76198[78]\b/);
  await o.ctx.close();
});

test("TC-UX20-5 điểm đã công bố ⇒ đúng một nút phụ «Yêu cầu xem lại», bấm mở biểu mẫu như cũ", async ({ browser }, info) => {
  D(info); test.setTimeout(300_000);
  const o = await open(browser, "sv.kha"); const c = await courseId(o.page, "QD9A"); // QC SmkS: 761987 không có bài đã công bố cho SV; sv.kha có 1 kết quả công bố ở QD9A (chỉ đọc)
  await go(o.page, "/assignments", c, 3000);
  const hrefs: string[] = await o.page.locator("main a[href^='/assignments/']").evaluateAll((as: Element[]) => [...new Set(as.map((a) => a.getAttribute("href") as string))]);
  console.log("INFO UX20-5 số bài:", hrefs.length);
  let found = "";
  for (const h of hrefs.slice(0, 14)) {
    await go(o.page, h, c, 2500);
    const b = o.page.getByRole("button", { name: "Yêu cầu xem lại" });
    if (await b.count()) { found = h; break; }
  }
  expect(found, "có bài đã công bố điểm với nút «Yêu cầu xem lại»").not.toBe("");
  const btn = o.page.getByRole("button", { name: "Yêu cầu xem lại" });
  console.log("INFO UX20-5 trang:", found, "| số nút:", await btn.count(), "| variant:", await btn.first().getAttribute("data-variant"));
  expect(await btn.count(), "đúng một nút").toBe(1);
  expect(await btn.first().getAttribute("data-variant")).not.toBe("primary");
  await btn.first().click(); await o.page.waitForTimeout(800);
  const form = (await o.page.locator("main").innerText()).replace(/\s+/g, " ");
  expect(form, "mở biểu mẫu (tiêu chí + lý do)").toMatch(/lý do/i);
  await o.ctx.close();
});
