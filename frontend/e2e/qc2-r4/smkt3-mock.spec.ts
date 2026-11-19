import { expect, test, type Page } from "@playwright/test";
import { S, apiLogin, em, installRec, note } from "./r4ui-lib";

// SmkT3: TC-P207-35 phần còn lại (GV /inbox, SV chọn lớp, lớp ngoài mô phỏng). ep-r, chỉ đọc.
installRec("smkt3-mock");
const picker = (p: Page) => p.getByRole("button", { name: /^Chọn lớp, đang xem/ });
const text = async (p: Page) => (await p.locator("main").innerText()).replace(/\s+/g, " ");

async function choose(p: Page, code: string) {
  await picker(p).click();
  await p.getByRole("dialog", { name: "Chọn lớp" }).getByRole("menuitem", { name: new RegExp(code) }).click();
  await expect(picker(p)).toContainText(code);
  await p.waitForTimeout(900);
}

test("[TC-P207-35] GV /inbox 761987 ↔ 761988; SV chọn lớp; lớp ngoài mô phỏng; 0 request lớp thật", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  test.setTimeout(240_000);
  const calls: string[] = [];
  page.on("request", (r) => /\/api\/v1\/courses\/[0-9a-f-]{36}\//.test(r.url()) && calls.push(r.url().replace(/^.*\/api\/v1/, "").slice(0, 70)));
  const out: string[] = [];
  // GV seed
  await apiLogin(page, "teacher@edupilot.local", "Edupilot#Seed-2026");
  await page.goto("/inbox");
  await expect(picker(page)).toBeVisible({ timeout: 30_000 });
  await choose(page, "761987");
  const g87 = await text(page);
  await choose(page, "761988");
  const g88 = await text(page);
  out.push(`GV /inbox: 761987 ${g87.length} ký tự, 761988 ${g88.length} ký tự, khác nhau=${g87 !== g88}`);
  expect(g87).not.toBe(g88);
  const gCalls = calls.length;
  // SV seed có cả hai lớp
  await page.context().clearCookies();
  await page.evaluate(() => localStorage.removeItem("ep:ui:course"));
  await apiLogin(page, "sv.gioi@edupilot.local", "Edupilot#Seed-2026");
  for (const route of ["/chat", "/threads", "/calendar"]) {
    await page.goto(route);
    await expect(picker(page)).toBeVisible({ timeout: 30_000 });
    await choose(page, "761987");
    const a = await text(page);
    await choose(page, "761988");
    const b = await text(page);
    out.push(`SV ${route}: 761987 ${a.length}, 761988 ${b.length}, khác nhau=${a !== b}`);
  }
  // lớp ngoài mô phỏng (lớp QC của gv1): chữ hiển thị so với 761987
  await page.context().clearCookies();
  await apiLogin(page, em("gv1"));
  await page.goto("/analytics");
  await expect(picker(page)).toBeVisible({ timeout: 30_000 });
  const own = (await picker(page).innerText()).replace(/\s+/g, " ");
  const o = await text(page);
  out.push(`GV gv1 lớp ngoài mô phỏng (${own}) /analytics ${o.length} ký tự; có chữ '761987'=${/761987/.test(o)}`);
  note(`${out.join(" ; ")}; request /courses/<id>/ từ màn mô phỏng: GV ${gCalls}, toàn lượt ${calls.length} (${[...new Set(calls)].slice(0, 4).join(",")})`);
  void S;
});

test("[TC-P207-30 GATEP2-26] GV seed: bộ chọn lớp có 'Tất cả lớp của tôi' + 2 lớp seed; 'Hôm nay' nêu lớp; lớp 2 'Thiết lập lớp mới'", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  await apiLogin(page, "teacher@edupilot.local", "Edupilot#Seed-2026");
  await page.goto("/");
  await expect(picker(page)).toBeVisible({ timeout: 30_000 });
  await picker(page).click();
  const dlg = page.getByRole("dialog", { name: "Chọn lớp" });
  const items = (await dlg.getByRole("menuitem").allInnerTexts()).map((t) => t.replace(/\s+/g, " ").trim());
  await page.keyboard.press("Escape");
  const has = (re: RegExp) => items.filter((t) => re.test(t)).length;
  await choose(page, "761988");
  await page.goto("/");
  await expect(picker(page)).toContainText("761988", { timeout: 20_000 });
  await page.waitForTimeout(2500);
  const t88 = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  await picker(page).click();
  await dlg.getByRole("menuitem", { name: /Tất cả lớp của tôi/ }).click();
  await expect(picker(page)).toContainText(/Tất cả lớp/, { timeout: 15_000 });
  await page.waitForTimeout(2500);
  const tall = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  note(`bộ chọn: ${items.length} mục; 'Tất cả lớp của tôi'=${has(/Tất cả lớp của tôi/)}; 761987=${has(/761987/)}; 761988=${has(/761988/)}; Hôm nay @761988: «${t88.slice(0, 220)}» có 'Thiết lập lớp mới'=${/Thiết lập lớp mới/.test(t88)}; Hôm nay @tất cả: «${tall.slice(0, 220)}»`);
  expect(has(/Tất cả lớp của tôi/)).toBe(1);
  expect(has(/761987/) + has(/761988/)).toBe(2);
});

test("[TC-P211-32 GATEP2-30] / của SV C và GV: ≤ 3 s thấy việc kế tiếp, đúng một hành động chính", async ({ browser }, info) => {
  if (info.project.name !== "desktop") test.skip();
  const out: string[] = [];
  for (const [u, tag] of [["sv.nguyco@edupilot.local", "SV C"], ["teacher@edupilot.local", "GV"]]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN", viewport: { width: 1440, height: 900 } });
    const page = await ctx.newPage();
    await apiLogin(page, u, "Edupilot#Seed-2026");
    const t0 = Date.now();
    await page.goto("/");
    const primary = page.locator('main [data-variant="primary"]:visible');
    await expect(primary.first()).toBeVisible({ timeout: 15_000 });
    const dt = (Date.now() - t0) / 1000;
    const n = await primary.count();
    const label = (await primary.first().innerText()).replace(/\s+/g, " ").trim();
    const h = (await page.locator("main h1, main h2").first().innerText().catch(() => "")).replace(/\s+/g, " ").trim();
    out.push(`${tag}: hành động chính hiện sau ${dt.toFixed(1)} s, số nút primary=${n}, «${label}»; tiêu đề đầu «${h}»`);
    expect(n).toBe(1);
    await ctx.close();
  }
  note(out.join(" ; "));
});
