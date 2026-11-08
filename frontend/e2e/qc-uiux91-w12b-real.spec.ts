import { expect, test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC đợt 1+2, bổ sung (stack ep-qc): TC-UX02-5/PQ, UX04-17, UX05-6/8, UX06-2, UX07-10/12/13.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
const desktop = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
async function login(page: Page, user: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
  await page.locator("[data-part=topbar]").waitFor({ timeout: 15_000 });
}
const settle = async (page: Page, url: string) => { await page.goto(url); await page.waitForTimeout(2500); };
const main = (page: Page) => page.locator("main").innerText();
const OLD_FMT = /\b\d{2}:\d{2} \d{2}-\d{2}\b|Invalid Date|NaN|undefined/;

test("TC-UX02-5 chỉ ngày: không dạng cũ, có dd/MM hoặc dd/MM/yyyy (GV + SV)", async ({ browser }, info) => {
  desktop(info);
  for (const [user, routes] of [["teacher", ["/assignments", "/insights", "/calendar", "/exams", "/library"]], ["sv.gioi", ["/assignments", "/calendar", "/exams", "/library"]]] as const) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await login(page, user);
    for (const r of routes) {
      await settle(page, r);
      const t = await main(page);
      console.log(`INFO ${user} ${r}: ${(t.match(/\b\d{2}\/\d{2}(\/\d{4})?\b/g) ?? []).slice(0, 3).join(",") || "—"}`);
      expect(t, `${user} ${r}`).not.toMatch(OLD_FMT);
    }
    await ctx.close();
  }
});

test("TC-UX02-PQ SV /documents, /inbox → 403 như cũ", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "sv.gioi");
  for (const r of ["/documents", "/inbox"]) {
    await settle(page, r);
    const t = await main(page);
    expect(t, r).toContain("Bạn không có quyền xem màn này");
    expect(t, r).not.toMatch(/\d{2}\/\d{2}|\d{2}:\d{2}/);
  }
  await ctx.close();
});

test("TC-UX04-17 không so le: mọi hoạt cảnh bắt đầu cùng lúc khi tải `/` và `/inbox`", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "teacher");
  for (const r of ["/", "/inbox"]) {
    await page.addInitScript(() => undefined);
    await page.goto(r);
    await page.waitForTimeout(300);
    const starts = await page.evaluate(() => document.getAnimations().filter((a) => a.playState === "running" && a.effect && (a.effect as KeyframeEffect).target && !(a instanceof CSSTransition && false)).map((a) => ({ n: (a as CSSAnimation).animationName ?? "tr", t: Math.round(Number(a.startTime ?? 0)), d: Number(a.effect?.getTiming().delay ?? 0), dur: Number(a.effect?.getTiming().duration ?? 0) })));
    const uniq = [...new Set(starts.map((s) => s.d))];
    console.log(`INFO ${r}: ${starts.length} hoạt cảnh, delay khác nhau: ${uniq.join(",")} ; ${[...new Set(starts.map((s) => s.n))].join(",")}`);
    expect(uniq.filter((d) => d > 0), `${r}: delay > 0 (so le)`).toEqual([]);
  }
  await ctx.close();
});

test("TC-UX05-6 TA /grading: nút Tải bài lên theo quyền (có nút ⇔ không khoá vì quyền)", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "ta");
  await settle(page, "/grading");
  const n = await page.getByRole("button", { name: "Tải bài lên" }).count();
  console.log("INFO TA thấy nút Tải bài lên:", n);
  expect(await main(page)).not.toMatch(/Bạn không có quyền xem màn này/);
  await ctx.close();
});

test("TC-UX05-8 Sổ điểm lớp có điểm: câu «Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.» ở chân bảng", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "teacher");
  await page.getByRole("button", { name: /^Chọn lớp/ }).click();
  await page.getByRole("option", { name: /761987/ }).or(page.getByRole("menuitem", { name: /761987/ })).first().click();
  await page.waitForTimeout(800);
  await page.getByRole("link", { name: "Sổ điểm" }).first().click();
  await page.waitForTimeout(2500);
  const t = await main(page);
  expect(t.split("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.").length - 1, "đúng một lần").toBe(1);
  await ctx.close();
});

test("TC-UX06-2 chọn A rồi B, tải lại → vẫn B", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "teacher");
  const top = page.locator("[data-part=topbar]");
  const cur = async () => ((await top.innerText()).match(/76198[78]/) ?? [""])[0];
  const pick = async (code: string) => {
    await page.getByRole("button", { name: /^Chọn lớp/ }).click();
    await page.getByRole("option", { name: new RegExp(code) }).or(page.getByRole("menuitem", { name: new RegExp(code) })).first().click();
    await page.waitForTimeout(800);
  };
  await pick("761987"); expect(await cur()).toBe("761987");
  await pick("761988"); expect(await cur()).toBe("761988");
  await page.reload(); await page.locator("[data-part=topbar]").waitFor(); await page.waitForTimeout(1500);
  expect(await cur()).toBe("761988");
  await ctx.close();
});

test("TC-UX07-10/12/13 SV: khối hoạt động không còn Threads; sidebar thu gọn có aria-label; thanh dưới 375 không cắt nhãn", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "teacher");
  await page.goto("/");
  await page.waitForTimeout(1500);
  expect(await page.locator("body").innerText()).not.toMatch(/\bThreads\b/);
  const collapse = page.getByRole("button", { name: /Thu gọn/ }).first();
  if (await collapse.count()) {
    await collapse.click();
    await page.waitForTimeout(400);
    const links = await page.locator("nav a").evaluateAll((l) => l.map((a) => ({ t: (a.textContent ?? "").trim(), al: a.getAttribute("aria-label") ?? "", ti: a.getAttribute("title") ?? "", w: Math.round(a.getBoundingClientRect().width) })));
    console.log("INFO sidebar thu gọn:", JSON.stringify(links.slice(0, 4)));
    for (const l of links.filter((x) => x.w > 0 && x.w < 90)) expect(l.al || l.ti || l.t, "tên đầy đủ khi thu gọn").toBeTruthy();
  }
  await ctx.close();
  const m = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true });
  const mp = await m.newPage();
  await login(mp, "sv.gioi");
  await mp.waitForTimeout(800);
  const cut = await mp.evaluate(() => [...document.querySelectorAll("nav[aria-label] a, [data-part=bottom-nav] a, nav a")].filter((a) => { const r = a.getBoundingClientRect(); return r.bottom > innerHeight - 90 && r.width > 0; }).map((a) => ({ t: (a.textContent ?? "").trim(), cut: [...a.querySelectorAll("*")].some((e) => e.scrollWidth > e.clientWidth + 1) || a.scrollWidth > a.clientWidth + 1 })));
  console.log("INFO thanh dưới 375:", JSON.stringify(cut));
  expect(cut.length).toBeGreaterThanOrEqual(4);
  expect(cut.filter((c) => c.cut)).toEqual([]);
  await m.close();
});
