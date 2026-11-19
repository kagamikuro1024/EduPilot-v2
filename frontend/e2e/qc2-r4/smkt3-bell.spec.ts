import { expect, test, type Page } from "@playwright/test";
import { S, apiLogin, installRec, newCourse, note, psql } from "./r4ui-lib";

// SmkT3: TC-P208-34 — hai thứ tự: (A) GV mở sẵn trang rồi mới được gán lớp (chuông sống); (B) được gán xong rồi mới đăng nhập (đúng chữ TC "vừa được gán; đăng nhập GV2").
installRec("smkt3-bell");
const picker = (p: Page) => p.getByRole("button", { name: /^Chọn lớp, đang xem/ });

async function clickBellItem(page: Page, code: string) {
  await page.getByRole("button", { name: /^Thông báo, \d+ chưa đọc/ }).click();
  const item = page.locator("[data-part=notifications]").getByRole("link", { name: new RegExp(`Bạn được phân công lớp .*${code}`) });
  await expect(item).toBeVisible({ timeout: 20_000 });
  await item.click();
}

test("[TC-P208-34] B: được gán rồi đăng nhập → bấm thông báo → đúng đường dẫn, bộ chọn lớp đúng lớp, unread giảm 1", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  const gv2 = S.gv2;
  const c = await newCourse(gv2.id, "BB");
  await apiLogin(page, gv2.email);
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  const t0 = Date.now();
  await expect(page.locator("[data-part=bell-dot]")).toHaveCount(1, { timeout: 40_000 });
  const dt = (Date.now() - t0) / 1000;
  const bell = page.getByRole("button", { name: /^Thông báo, \d+ chưa đọc/ });
  const unread0 = Number((await bell.getAttribute("aria-label"))!.match(/(\d+) chưa đọc/)![1]);
  await clickBellItem(page, c.code);
  const link = psql(`select link from notifications where user_id='${gv2.id}' and course_id='${c.id}' order by created_at desc limit 1`);
  await expect.poll(() => new URL(page.url()).pathname, { timeout: 15_000 }).toBe(link.split("?")[0]);
  const sel = await picker(page).innerText();
  const search = new URL(page.url()).search;
  note(`B: chấm sau ${dt.toFixed(1)} s; link ${link}; sau bấm search="${search}"; bộ chọn lớp="${sel.replace(/\s+/g, " ")}" (mong ${c.code}); unread ${unread0}`);
  await expect(picker(page)).toContainText(c.code);
  expect(search).not.toContain("course=");
  await expect.poll(() => psql(`select count(*) from notifications where user_id='${gv2.id}' and course_id='${c.id}' and read_at is null`), { timeout: 15_000 }).toBe("0");
  const after = Number(((await page.getByRole("button", { name: /^Thông báo/ }).getAttribute("aria-label")) ?? "").match(/(\d+) chưa đọc/)?.[1] ?? 0);
  expect(after).toBe(unread0 - 1);
});

test("[TC-P208-34] A: GV mở sẵn trang rồi mới được gán lớp (chuông sống) → bấm thông báo → bộ chọn lớp", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  const gv2 = S.gv2;
  await apiLogin(page, gv2.email);
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  await page.waitForTimeout(3000);
  const reqs: string[] = [];
  page.on("request", (r) => /\/me\/courses/.test(r.url()) && reqs.push(r.url().slice(-40)));
  const c = await newCourse(gv2.id, "AA");
  const t0 = Date.now();
  await expect(page.locator("[data-part=bell-dot]")).toHaveCount(1, { timeout: 40_000 });
  const dt = (Date.now() - t0) / 1000;
  await clickBellItem(page, c.code);
  await page.waitForTimeout(3000);
  const sel1 = (await picker(page).innerText()).replace(/\s+/g, " ");
  const url1 = page.url();
  await page.reload();
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  const opts = await picker(page).click().then(() => page.getByRole("option").allInnerTexts().catch(() => [] as string[]));
  note(`A: chấm sau ${dt.toFixed(1)} s; sau bấm url=${url1} bộ chọn="${sel1}" (mong ${c.code}); /me/courses gọi lại sau gán: ${reqs.length}; sau tải lại, danh sách chọn lớp chứa ${c.code}: ${opts.join("|").includes(c.code)}`);
  expect(sel1).toContain(c.code);
});
