import { expect, test } from "@playwright/test";
import { needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// QC sprint 11 — P-11-14: /assignments (STUDENT) trên ep-qc thật.
needReal();

test("@real ASG-1 SV: nhóm, một hành động mỗi hàng, khớp DB, nav (TC-A1…A3)", async ({ page }) => {
  await uiLogin(page, "sv.kha");
  await page.goto("/assignments");
  await expect(page.getByRole("heading", { name: "Bài tập", level: 1 })).toBeVisible();
  await expect(page.getByText("Cần nộp", { exact: true })).toBeVisible();
  await shot(page, "asg-1440");
  const nav = (await page.locator("[data-part=sidebar]").innerText()).split("\n").map((s) => s.trim()).filter(Boolean);
  const i = nav.indexOf("Bài tập");
  expect(nav[i - 1], "«Bài tập» sau «Hỏi đáp công khai»").toBe("Hỏi đáp công khai");
  expect(nav[i + 1], "«Bài tập» trước «Luyện đề»").toBe("Luyện đề");
  const rows = page.locator("main li, main [role=row], main article").filter({ hasText: /Bài tập 0\d/ });
  const n = await rows.count();
  const db = Number(sql(`select count(*) from assignments a join courses c on c.id=a.course_id where c.class_code='761987' and a.published_at is not null`)[0][0]);
  expect.soft(n, `số hàng trên màn (${n}) = bài tập đã công bố của lớp (${db})`).toBe(db);
  for (let k = 0; k < n; k++) {
    const acts = await rows.nth(k).locator("a[href], button").evaluateAll((els) => els.filter((e) => /^(Nộp bài|Nộp lại|Xem|Xem điểm|Xem kết quả)/.test((e.textContent ?? "").trim())).length);
    expect.soft(acts, `hàng ${k + 1}: đúng một hành động chính`).toBe(1);
  }
});

test("@real ASG-2 Staff / SV ngoài lớp bị chặn, không gọi API; 375 px (TC-A4…A6)", async ({ page, browser }) => {
  for (const u of ["teacher", "ta"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: process.env.QC_BASE ?? "https://localhost:943" });
    const p = await ctx.newPage();
    const calls: string[] = [];
    await uiLogin(p, u);
    p.on("request", (r) => { if (/\/api\/v1\/.*assignments/.test(r.url()) && !/\/assignments\/[0-9a-f-]{36}\//.test(r.url())) calls.push(r.url()); });
    await p.goto("/assignments");
    await expect(p.getByText("Bạn không có quyền xem màn này")).toBeVisible();
    expect(calls, `${u}: không gọi API danh sách bài tập SV`).toEqual([]);
    await ctx.close();
  }
  await page.setViewportSize({ width: 375, height: 812 });
  await uiLogin(page, "sv.kha");
  await page.goto("/assignments");
  await expect(page.getByText("Cần nộp", { exact: true })).toBeVisible();
  await noOverflow(page);
  await shot(page, "asg-375");
  const small = await page.locator("main a[href], main button").evaluateAll((els) => els.filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && r.height < 44; }).map((e) => `${(e.textContent ?? "").trim().slice(0, 20)}:${Math.round(e.getBoundingClientRect().height)}`));
  expect.soft(small, "vùng chạm ≥ 44 px").toEqual([]);
});
