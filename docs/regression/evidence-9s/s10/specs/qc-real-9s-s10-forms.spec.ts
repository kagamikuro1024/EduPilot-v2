import { appendFileSync, writeFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import { needReal, sql, PW, OUT } from "./qc-real-lib";

// QC smoke S10: TC-11-19 UI Forms trên ep-qc / QD10A — hàng «Chưa khớp» có «Nhập ở Sổ điểm», không dòng giải thích, định danh đã che.
const BASE = process.env.QC_BASE ?? "https://localhost:943";
const RES = process.env.QC_RES ?? `${OUT}/results_m.jsonl`;
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 300)); };
test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
test.beforeAll(() => needReal());
test("TC-11-19 Forms: hàng Chưa khớp có Nhập ở Sổ điểm", async ({ browser }) => {
  test.setTimeout(180_000);
  const id = sql(`select id from courses where class_code='QD10A'`)[0][0];
  const stu = sql(`select u.email, u.student_code from enrollments e join users u on u.id=e.user_id where e.course_id='${id}' and e.role_in_course='STUDENT' and e.status='ACTIVE' and u.email='sv.kha@edupilot.local'`)[0];
  const stamp = new Date().toISOString().slice(0, 16).replace("T", " ") + `:${Math.floor(Math.random() * 59)}`;
  const csv = ["Email,Mã số sinh viên,Completion time,Câu 1,Câu 2", `${stu[0]},${stu[1]},${stamp},A,B`, `khong.co.trong.lop@example.test,99999999,${stamp},A,B`, `x@y.z,${stu[1]},${stamp},A,B`].join("\n");
  const fp = `${OUT}/forms-1119.csv`; writeFileSync(fp, csv);
  const ctx = await browser.newContext({ baseURL: BASE, ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), id);
  const page = await ctx.newPage();
  await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill("teacher@edupilot.local"); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  await page.goto("/gradebook"); await page.waitForTimeout(3000); await page.screenshot({ path: `${OUT}/s10-1119-0.png` });
  const more = page.getByRole("button", { name: /Thêm|Khác|Tuỳ chọn|Thao tác|Menu/ }).first();
  if (await more.count()) await more.click(); else await page.locator("[data-part=page-header] button").last().click();
  await page.getByRole("menuitem", { name: "Nhập từ Forms" }).click().catch(async () => { await page.getByText("Nhập từ Forms").first().click(); });
  await page.locator("[data-part=forms-input]").setInputFiles(fp); await page.waitForTimeout(500);
  await page.getByRole("button", { name: "Tiếp tục" }).click(); await page.waitForTimeout(2500); await page.screenshot({ path: `${OUT}/s10-1119-2.png` });
  // bước 2: dùng nguồn đáp án nhập tay: lấy cột còn lại; điền câu hỏi + đáp án
  const t2 = await page.locator("[data-part=forms-import]").innerText(); console.log("STEP2", t2.replace(/\n/g, " | ").slice(0, 400));
  await page.getByRole("button", { name: /Lấy hết cột còn lại/ }).click().catch(() => {});
  const qs = page.locator("select[aria-label^='Câu hỏi']"); const nq = await qs.count();
  for (let i = 0; i < nq; i++) { const opts = await qs.nth(i).locator("option").allInnerTexts(); if (opts.length > 1) await qs.nth(i).selectOption({ index: Math.min(i + 1, opts.length - 1) }); }
  const ans = page.locator("input[aria-label^='Đáp án câu']"); for (let i = 0; i < await ans.count(); i++) await ans.nth(i).fill("A");
  await page.getByRole("button", { name: "Xem trước" }).click(); await page.waitForTimeout(3000); await page.screenshot({ path: `${OUT}/s10-1119-3.png` });
  const sum = await page.locator("[data-part=forms-summary]").innerText().catch(() => ""); const li = page.locator("[data-part=forms-unmatched] li"); const n = await li.count();
  const texts = await li.allInnerTexts(); const links = await page.locator("[data-part=forms-unmatched] li").getByRole("button", { name: "Nhập ở Sổ điểm" }).count();
  const extraP = await page.locator("[data-part=forms-import] p").allInnerTexts();
  const unmasked = texts.some((t) => /khong\.co\.trong\.lop@example\.test/.test(t));
  verdict("TC-11-19", n >= 2 && links === n && !unmasked, `kết quả xem trước: «${sum}»; ${n} hàng chưa khớp, mỗi hàng có «Nhập ở Sổ điểm» ×${links}; định danh hiển thị: ${JSON.stringify(texts.map((t) => t.replace(/\n/g, " ")).slice(0, 3))} (email đầy đủ lộ: ${unmasked}); đoạn chữ trong hộp thoại: ${JSON.stringify(extraP).slice(0, 220)}`);
  await ctx.close();
});
