import { readFileSync } from "node:fs";
import { BASE_URL } from "./support/env";
import { expect, test, type Page } from "@playwright/test";
import { MOBILE_PRIMARY, navFor } from "../src/shared/shell/nav";
import { loadAudit } from "./support/audit";
import { asDemo, asJwt, sessionBody, type DemoRole } from "./support/session";

// US-PU-04: khung ứng dụng. Chỉ chạy ở dự án desktop (tự đặt bề rộng); bản dựng `build:gate` (mock bật, DEV_AUTH bật).
// Ca `no-backend` chạy ở bản dựng NEXT_PUBLIC_MOCK_SCREENS=0; các ca còn lại bị bỏ qua ở bản đó.
const MOCK_OFF = process.env.NEXT_PUBLIC_MOCK_SCREENS === "0";
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});
const mockOnly = () => test.skip(MOCK_OFF, "bản dựng MOCK_SCREENS=0");

const ROLES: DemoRole[] = ["student", "ta", "teacher", "admin"];
const size = (page: Page, width: number, height = 900) => page.setViewportSize({ width, height });
const box = (page: Page, sel: string) => page.locator(sel).first().evaluate((e) => ({ w: e.getBoundingClientRect().width, h: e.getBoundingClientRect().height }));

test("metrics: sidebar 216 / 72 / ẩn; header 56; thu gọn nhớ ở ep:ui:sidebar", async ({ page, context }) => {
  mockOnly();
  await asDemo(context, "teacher");
  for (const [w, side] of [[1440, 216], [1100, 216], [1099, 72], [900, 72], [720, 72]] as const) {
    await size(page, w);
    await page.goto("/");
    await expect.poll(async () => (await box(page, "[data-part=sidebar]")).w, { message: `sidebar ở ${w}` }).toBe(side);
    expect((await box(page, "header")).h, `header ở ${w}`).toBe(56);
  }
  for (const w of [719, 375]) {
    await size(page, w, 800);
    await page.goto("/");
    await expect(page.locator("[data-part=sidebar]")).toHaveCount(0);
    await expect(page.locator("[data-part=bottom-nav]")).toBeVisible();
  }
  // thu gọn / mở lại ở ≥ 1100 và nhớ qua tải lại
  await size(page, 1440);
  await page.goto("/");
  const btn = page.getByRole("button", { name: /Thu gọn thanh bên/ });
  await expect(btn).toHaveAttribute("aria-expanded", "true");
  await btn.click();
  await expect.poll(async () => (await box(page, "[data-part=sidebar]")).w).toBe(72);
  await page.reload();
  await expect.poll(async () => (await box(page, "[data-part=sidebar]")).w).toBe(72);
  const v = await page.evaluate(() => localStorage.getItem("ep:ui:sidebar"));
  expect(v).toBe("collapsed");
  await page.getByRole("button", { name: /Mở rộng thanh bên/ }).click();
  await expect(page.getByRole("button", { name: /Thu gọn thanh bên/ })).toHaveAttribute("aria-expanded", "true");
  // nhãn có tên truy cập khi thu gọn
  await size(page, 900);
  await expect(page.locator("[data-part=sidebar] nav a").first()).toHaveAttribute("aria-label", /.+/);
});

test("metrics: thanh dưới không che nút cuối nội dung (375)", async ({ page, context }) => {
  mockOnly();
  await asDemo(context, "student");
  await size(page, 375, 800);
  await page.goto("/threads");
  await page.locator("main").waitFor();
  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  await page.waitForTimeout(200);
  const r = await page.evaluate(() => {
    const nav = document.querySelector("[data-part=bottom-nav]")!.getBoundingClientRect();
    const main = document.querySelector("main")!;
    const last = Array.from(main.querySelectorAll("button, a, input, textarea")).filter((e) => (e as HTMLElement).offsetParent !== null).pop()!.getBoundingClientRect();
    return { navTop: nav.top, lastBottom: last.bottom, pad: parseFloat(getComputedStyle(main).paddingBottom) };
  });
  expect(r.lastBottom).toBeLessThanOrEqual(r.navTop + 0.5);
  expect(r.pad).toBeGreaterThanOrEqual(56);
});

test("active indicator: vạch đỏ 2px, nền không đỏ, khung vỏ nền đặc", async ({ page, context }) => {
  mockOnly();
  await asDemo(context, "teacher");
  await page.goto("/attendance");
  const cur = page.locator("[data-part=sidebar] a[aria-current=page]");
  await expect(cur).toHaveCount(1);
  const st = await cur.evaluate((e) => {
    const b = getComputedStyle(e, "::before");
    const red = getComputedStyle(document.documentElement).getPropertyValue("--ep-red").trim();
    const probe = document.createElement("i");
    probe.style.color = red;
    document.body.append(probe);
    const redRgb = getComputedStyle(probe).color;
    probe.remove();
    return { width: b.width, bg: b.backgroundColor, redRgb, itemBg: getComputedStyle(e).backgroundColor };
  });
  expect(st.width).toBe("2px");
  expect(st.bg).toBe(st.redRgb);
  expect(st.itemBg).not.toBe(st.redRgb);
  const info = await page.evaluate(() => {
    const redProbe = document.createElement("i");
    redProbe.style.color = getComputedStyle(document.documentElement).getPropertyValue("--ep-red");
    document.body.append(redProbe);
    const red = getComputedStyle(redProbe).color;
    redProbe.remove();
    const els = Array.from(document.querySelectorAll("[data-part=sidebar], [data-part=sidebar] *, header, header *"));
    const redBg = els.filter((e) => getComputedStyle(e).backgroundColor === red).length;
    const marks = document.querySelectorAll("[data-part=bell-dot], [data-part^=nav-badge-]").length;
    const shell = ["[data-part=sidebar]", "header"].map((q) => {
      const cs = getComputedStyle(document.querySelector(q)!);
      return { bf: cs.backdropFilter, bg: cs.backgroundColor };
    });
    return { redBg, marks, shell };
  });
  expect(info.redBg).toBeLessThanOrEqual(info.marks);
  for (const sh of info.shell) {
    expect(sh.bf === "none" || sh.bf === "").toBe(true);
    expect(sh.bg).not.toMatch(/rgba\(.*, 0(\.\d+)?\)$/);
  }
});

type Row = [string, string];
const NAV: Record<string, Row[]> = {
  student: [["Hôm nay", "/"], ["Chat riêng", "/chat"], ["Threads", "/threads"], ["Luyện đề", "/practice"], ["Thư viện", "/library"], ["Lịch", "/calendar"], ["Kết quả của tôi", "/me"]],
  "student-no-course": [["Hôm nay", "/"]],
  ta: [["Hôm nay", "/"], ["Hộp thư hỗ trợ", "/inbox"], ["Sinh viên", "/students"], ["Điểm danh", "/attendance"], ["Sổ điểm", "/gradebook"], ["Chấm bài", "/grading"], ["Ngân hàng câu hỏi", "/questions"], ["Threads", "/threads"], ["Tài liệu", "/documents"], ["Lịch", "/calendar"], ["Insights", "/insights"], ["Analytics", "/analytics"]],
  teacher: [],
  admin: [["Hôm nay", "/"], ["Quan sát AI", "/observability"], ["Lớp học", "/admin/courses"], ["Người dùng", "/admin/users"], ["Cấu hình LLM", "/settings/llm"], ["Tích hợp", "/settings/integrations"]],
};
NAV.teacher = [...NAV.ta, ["Quan sát AI", "/observability"], ["Cấu hình LLM", "/settings/llm"], ["Tích hợp", "/settings/integrations"]];

test("nav per role: nhãn + href + thứ tự khớp SRS 7.5", async ({ page, context }) => {
  mockOnly();
  for (const [ctx, rows] of Object.entries(NAV)) {
    await context.clearCookies();
    const noCourse = ctx === "student-no-course";
    await asDemo(context, ctx.startsWith("student") ? "student" : (ctx as DemoRole), { person: noCourse ? "sv-4" : undefined });
    await page.goto("/");
    const got = await page.locator("[data-part=sidebar] nav a").evaluateAll((as) => as.map((a) => [(a.textContent ?? "").replace(/\d+$/, "").trim(), a.getAttribute("href")]));
    expect(got, ctx).toEqual(rows);
    if (ctx === "student") expect(await page.locator("body").innerText()).not.toContain("Cấu hình LLM");
    // nhóm
    if (ctx === "teacher") {
      const groups = await page.locator("[data-part=sidebar] nav p").allTextContents();
      expect(groups).toEqual(["Làm việc", "Đánh giá", "Nội dung", "Hiểu lớp học", "Hệ thống"]);
    }
    if (ctx === "ta") expect(await page.locator("[data-part=sidebar] nav p").allTextContents()).not.toContain("Hệ thống");
  }
  expect(navFor("teacher").flatMap((g) => g.items)).toHaveLength(15);
});

test("badges: chỉ inbox / grading; không viết cứng số trong nav.ts", async ({ page, context }) => {
  mockOnly();
  await asDemo(context, "teacher");
  await page.goto("/");
  const keys = await page.locator("[data-part^=nav-badge-]").evaluateAll((es) => es.map((e) => e.getAttribute("data-part")!.replace("nav-badge-", "")));
  for (const k of keys) expect(["inbox", "grading"]).toContain(k);
  expect(readFileSync("src/shared/shell/nav.ts", "utf8").split("\n").filter((l) => /badge: [0-9]/.test(l))).toHaveLength(0);
  // mọi huy hiệu hiện đều khác 0
  for (const t of await page.locator("[data-part^=nav-badge-]").allTextContents()) expect(Number(t)).toBeGreaterThan(0);
});

test("mobile: ≤ 5 đích, Thêm đủ mục còn lại, đóng bằng Esc trả focus, không tràn ngang", async ({ page, context }) => {
  mockOnly();
  for (const role of ROLES) {
    for (const w of [375, 390]) {
      await context.clearCookies();
      await asDemo(context, role);
      await size(page, w, 844);
      await page.goto("/");
      const items = page.locator("[data-part=bottom-nav] a, [data-part=bottom-nav] button");
      expect(await items.count(), `${role}@${w}`).toBeLessThanOrEqual(5);
      for (const h of await items.evaluateAll((es) => es.map((e) => e.getBoundingClientRect().height))) expect(h).toBeGreaterThanOrEqual(44);
      const total = navFor(role, true).flatMap((g) => g.items).length;
      const more = page.locator("[data-part=bottom-nav]").getByRole("button", { name: "Thêm" });
      await more.click();
      const dlg = page.locator("dialog[open]");
      expect(await dlg.locator("a").count(), `${role} Thêm`).toBe(total - MOBILE_PRIMARY[role].length);
      await page.keyboard.press("Escape");
      await expect(page.locator("dialog[open]")).toHaveCount(0);
      await expect(more).toBeFocused();
      expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBe(0);
    }
  }
});

test("topbar: không h1, 4 nhóm điều khiển, hồ sơ có aria-label Tài khoản", async ({ page, context }) => {
  mockOnly();
  await asDemo(context, "teacher");
  await page.goto("/");
  await expect(page.locator("header h1")).toHaveCount(0);
  await expect(page.locator("header > button, header > div > button, header > div > div > button")).not.toHaveCount(0);
  const groups = await page.evaluate(() => ({
    course: !!document.querySelector('header button[aria-label^="Chọn lớp"]'),
    search: !!document.querySelector('header button[aria-label="Tìm nhanh hoặc đi đến"]'),
    bell: !!document.querySelector('header button[aria-label^="Thông báo"]'),
    profile: !!document.querySelector('header button[aria-label^="Tài khoản: "]'),
  }));
  expect(groups).toEqual({ course: true, search: true, bell: true, profile: true });
  await page.getByRole("button", { name: /^Tài khoản: / }).click();
  await expect(page.getByText("Đổi vai")).toBeVisible();
});

test("palette: Ctrl K, bỏ dấu, activedescendant, Esc trả focus, vai SV không thấy Điểm danh", async ({ page, context }) => {
  mockOnly();
  await asDemo(context, "teacher");
  await page.goto("/");
  await page.locator("main").waitFor();
  await page.waitForTimeout(300); // chờ hydrate trước khi bấm phím tắt
  const opener = page.getByRole("button", { name: "Tìm nhanh hoặc đi đến" });
  await opener.focus();
  await page.keyboard.press("Control+k");
  const input = page.getByRole("combobox", { name: "Tìm nhanh hoặc đi đến" });
  await expect(input).toBeFocused();
  await input.fill("diem danh");
  await expect(page.getByRole("option").first()).toHaveText(/Điểm danh/);
  const id = await input.getAttribute("aria-activedescendant");
  expect(id).toBeTruthy();
  await input.fill("d");
  await page.keyboard.press("ArrowDown");
  expect(await input.getAttribute("aria-activedescendant")).not.toBe(id);
  await input.fill("diem danh");
  for (let i = 0; i < 10; i++) await page.keyboard.press("Tab");
  expect(await page.evaluate(() => !!document.activeElement?.closest("dialog"))).toBe(true);
  await page.keyboard.press("Escape");
  await expect(opener).toBeFocused();
  await expect(page.locator("dialog[open]")).toHaveCount(0);
  await page.keyboard.press("Control+k");
  await expect(page.locator("dialog[open]")).toHaveCount(1);
  await page.getByRole("combobox").fill("diem danh");
  await expect(page.getByRole("option")).toHaveCount(1);
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/attendance$/);

  await context.clearCookies();
  await asDemo(context, "student");
  await page.goto("/");
  await page.locator("main").waitFor();
  await page.waitForTimeout(300); // chờ hydrate trước khi bấm phím tắt
  await page.keyboard.press("Control+k");
  await page.getByRole("combobox").fill("diem danh");
  await expect(page.getByRole("option")).toHaveCount(0);
  await page.getByRole("combobox").fill("zzzz");
  await expect(page.getByText("Không thấy mục nào khớp.")).toBeVisible();
  await page.getByRole("combobox").fill("");
  await expect(page.getByRole("option")).toHaveCount(7);
});

test("notifications: chấm theo unread, mở không xoá chấm, rỗng, thông báo đọc một lần", async ({ page }) => {
  await page.goto("/dev/data");
  const demo = page.locator("[data-part=bell-demo]");
  const dot = demo.locator("[data-part=bell-dot]");
  await expect(dot).toHaveCount(1);
  await demo.getByRole("button", { name: /^Thông báo/ }).click();
  await expect(dot).toHaveCount(1); // mở khung không xoá chấm
  const rows = page.locator("[data-part=notifications] li");
  await expect(rows).toHaveCount(2);
  await expect(rows.first()).toContainText("Thông báo số 1");
  await expect(rows.first()).toContainText("Hộp thư hỗ trợ · 2 phút trước");
  await page.keyboard.press("Escape");
  await demo.getByRole("button", { name: "Đọc hết" }).click();
  await expect(dot).toHaveCount(0);
  const live = demo.locator("[data-part=bell-live]");
  await expect(live).toHaveText("");
  await demo.getByRole("button", { name: "Thêm một thông báo" }).click();
  await expect(live).toHaveText("Có 1 thông báo mới");
  await demo.getByRole("button", { name: "Hiển thị lại" }).click();
  await expect(live).toHaveText("Có 1 thông báo mới"); // không đọc lại: nội dung không đổi ⇒ không có cập nhật mới
  const mutations = await live.evaluate(
    (el) =>
      new Promise<number>((res) => {
        let n = 0;
        new MutationObserver(() => n++).observe(el, { childList: true, characterData: true, subtree: true });
        document.querySelectorAll("button").forEach((b) => b.textContent === "Hiển thị lại" && b.click());
        setTimeout(() => res(n), 300);
      }),
  );
  expect(mutations).toBe(0);
  await demo.getByRole("button", { name: "Xoá hết" }).click();
  await demo.getByRole("button", { name: /^Thông báo/ }).click();
  await expect(page.getByText("Chưa có thông báo. Khi có việc cần bạn, nó sẽ hiện ở đây.")).toBeVisible();
});

test.describe("session", () => {
  test("phiên ADMIN thật: nav Admin, không đổi vai, có Đăng xuất; token không lộ ra storage / DOM / URL", async ({ page, context }) => {
    await context.clearCookies();
    const calls = await asJwt(page, "ADMIN", { email: "admin@ptit.edu.vn", fullName: "Quản Trị Thử" });
    await page.goto("/");
    await expect(page.locator("[data-part=sidebar] nav a").first()).toBeVisible();
    expect(calls.refresh).toBe(1);
    const nav = await page.locator("[data-part=sidebar] nav a").allTextContents();
    expect(nav.map((t) => t.trim())).toEqual(NAV.admin.map((r) => r[0]));
    const tok = sessionBody("ADMIN", { email: "admin@ptit.edu.vn" }).access_token.split(".")[1];
    expect(await page.evaluate((t) => document.body.innerHTML.includes(t) || JSON.stringify({ ...localStorage }).includes(t) || document.cookie.includes(t) || location.href.includes(t), tok)).toBe(false);
    expect(await page.evaluate(() => JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage }))).not.toContain("eyJ");
    await page.getByRole("button", { name: /^Tài khoản: / }).click();
    await expect(page.getByText("Đổi vai")).toHaveCount(0);
    await expect(page.getByText("admin@ptit.edu.vn").first()).toBeVisible();
    await page.getByRole("menuitem", { name: "Đăng xuất" }).click();
    await expect(page).toHaveURL(/\/login$/);
    expect(calls.logout).toBe(1);
  });

  test("phiên STUDENT thật: màn mock dùng sv-2 + tên thật, SV bị chặn /settings/llm", async ({ page }) => {
    mockOnly();
    await asJwt(page, "STUDENT", { email: "sv@ptit.edu.vn", fullName: "Sinh Viên Thử" });
    await page.goto("/settings/llm");
    await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
    await page.getByRole("link", { name: "Về Hôm nay" }).click();
    expect((await page.locator("[data-part=sidebar] nav a").allTextContents()).map((t) => t.trim())).toEqual(NAV.student.map((r) => r[0]));
    await page.locator("[data-part=sidebar]").getByRole("link", { name: "Chat riêng" }).click(); // điều hướng trong trang: token ở bộ nhớ không mất
    await expect(page.getByRole("button", { name: /^Tài khoản: Sinh Viên Thử/ })).toBeVisible();
  });
});

test("forbidden: màn chặn, nút Về Hôm nay, 0 request /api/v1", async ({ page, context }) => {
  const api: string[] = [];
  page.on("request", (r) => /\/api\/v1\//.test(r.url()) && api.push(r.url()));
  const cases: Array<[DemoRole, string[]]> = [["student", ["/settings/llm", "/observability", "/admin/users"]], ["ta", ["/settings/llm", "/observability", "/admin/users"]], ["teacher", ["/admin/courses"]]];
  for (const [role, routes] of cases) {
    await context.clearCookies();
    await asDemo(context, role);
    for (const r of routes) {
      await page.goto(r);
      await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
      await expect(page.getByText(/^Trang này dành cho /)).toBeVisible();
      await expect(page.locator("main").getByRole("link", { name: "Về Hôm nay" })).toHaveCount(1);
      await expect(page.locator("main button")).toHaveCount(0);
    }
  }
  expect(api).toEqual([]);
  await context.clearCookies();
  await asDemo(context, "teacher");
  await page.goto("/settings/llm");
  await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toHaveCount(0);
  await context.clearCookies();
  await asDemo(context, "student", { person: "sv-4" });
  await page.goto("/chat");
  await expect(page.getByRole("heading", { name: "Bạn chưa vào lớp nào" })).toBeVisible();
});

test("no-backend: MOCK_SCREENS=0 → empty-no-backend đúng phase, 1 nút, nav còn", async ({ page, context }) => {
  test.skip(!MOCK_OFF, "chỉ ở bản dựng NEXT_PUBLIC_MOCK_SCREENS=0");
  const PHASE: Record<string, string> = { "/": "P2", "/admin/courses": "P2", "/admin/users": "P2", "/chat": "P3", "/threads": "P3", "/inbox": "P4", "/students": "P5", "/attendance": "P5", "/gradebook": "P6", "/me": "P6", "/grading": "P7", "/settings/integrations": "P7", "/documents": "P8", "/library": "P8", "/calendar": "P8", "/practice": "P9", "/questions": "P9", "/insights": "P10", "/analytics": "P10", "/observability": "P10" };
  for (const role of ROLES) {
    await context.clearCookies();
    await asDemo(context, role);
    for (const item of navFor(role, true).flatMap((g) => g.items)) {
      if (item.href === "/settings/llm") continue;
      await page.goto(item.href);
      const e = page.locator("[data-part=empty-no-backend]");
      await expect(e, `${role} ${item.href}`).toHaveCount(1);
      await expect(e).toContainText(`giai đoạn ${PHASE[item.href]} `);
      await expect(e.locator("a, button")).toHaveCount(1);
      await expect(page.locator(`[data-part=sidebar] nav a[href="${item.href}"]`)).toHaveCount(1);
      if (role === "student") expect(await e.innerText()).not.toMatch(/RAG|PII|trace|provider|fallback/i);
    }
  }
  await context.clearCookies();
  await asDemo(context, "admin");
  await page.goto("/settings/llm");
  await expect(page.locator("[data-part=empty-no-backend]")).toHaveCount(0);
  await page.goto("/dev/ui");
  await expect(page.locator("[data-part=empty-no-backend]")).toHaveCount(0);
});

test("keyboard: Bỏ qua điều hướng đầu tiên, vào main, thứ tự header → sidebar → nội dung, Esc trả focus", async ({ page, context }) => {
  mockOnly();
  await asDemo(context, "teacher");
  await page.goto("/");
  await page.locator("main").waitFor();
  await page.keyboard.press("Tab");
  expect(await page.evaluate(() => document.activeElement?.textContent)).toBe("Bỏ qua điều hướng");
  await page.keyboard.press("Enter");
  expect(await page.evaluate(() => !!document.activeElement?.closest("main") || document.activeElement?.id === "main")).toBe(true);
  expect(await page.locator("main").count()).toBe(1);
  expect(await page.locator("nav[aria-label]").count()).toBeGreaterThan(0);
  await page.goto("/");
  const seq: string[] = [];
  for (let i = 0; i < 40; i++) {
    await page.keyboard.press("Tab");
    seq.push(await page.evaluate(() => {
      const a = document.activeElement!;
      return a.closest("header") ? "header" : a.closest("aside") ? "sidebar" : a.closest("main") ? "main" : a.className.toString().includes("skip") ? "skip" : "other";
    }));
  }
  const collapsed = seq.filter((v, i) => i === 0 || v !== seq[i - 1]);
  expect(collapsed.slice(0, 4)).toEqual(["skip", "header", "sidebar", "main"]);
  // Esc trả focus: menu hồ sơ
  const prof = page.getByRole("button", { name: /^Tài khoản: / });
  await prof.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByText("Đổi vai")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(prof).toBeFocused();
});

test("regression-1.5: LEFT page-title 240 / 16, brand 56, logo 32, mark 28", async ({ page, context }) => {
  mockOnly();
  const { LEFT_SRC } = await loadAudit();
  await asDemo(context, "teacher");
  await size(page, 1440);
  await page.goto("/gradebook");
  expect(await page.evaluate(LEFT_SRC)).toBe(240);
  expect((await box(page, "[data-part=sidebar] [data-part=brand]")).h).toBe(56);
  expect((await box(page, "[data-part=sidebar] [data-part=brand] img")).h).toBe(32);
  await size(page, 390, 844);
  await page.goto("/gradebook");
  expect(await page.evaluate(LEFT_SRC)).toBe(16);
  expect(await box(page, "header [data-part=brand] img")).toMatchObject({ w: 28, h: 28 });
});

test("touch: vùng chạm và tràn ngang ở 4 vai × 375/390", async ({ browser }) => {
  mockOnly();
  test.setTimeout(90_000);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  for (const role of ROLES) {
    for (const w of [375, 390]) {
      // thiết bị cảm ứng thật (pointer: coarse) — vùng chạm 44 px chỉ áp dụng ở đó
      const context = await browser.newContext({ viewport: { width: w, height: 844 }, hasTouch: true, isMobile: true, locale: "vi-VN", baseURL: BASE_URL });
      await asDemo(context, role);
      const page = await context.newPage();
      for (const href of ["/", ...MOBILE_PRIMARY[role].slice(1, 3)]) {
        await page.goto(href);
        await page.locator("main").waitFor();
        await page.waitForTimeout(150);
        expect(await page.evaluate(TOUCH_SRC), `${role}@${w} ${href} touch`).toEqual([]);
        const a = (await page.evaluate(AUDIT_SRC)) as { ox: number };
        expect(a.ox, `${role}@${w} ${href} ox`).toBe(0);
      }
      await context.close();
    }
  }
});

test("BUG-PU04-1: bảng Thêm ở điện thoại là bảng trượt — đóng bằng bấm ngoài, vuốt xuống, Esc và trả focus về nút Thêm", async ({ browser }) => {
  mockOnly();
  test.setTimeout(90_000);
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true, locale: "vi-VN", baseURL: BASE_URL });
  await asDemo(context, "teacher");
  const page = await context.newPage();
  await page.goto("/");
  await page.locator("main").waitFor();
  await page.waitForTimeout(300);
  const more = page.locator("[data-part=bottom-nav]").getByRole("button", { name: "Thêm" });
  const dlg = page.locator("dialog[open]");
  const open = async () => {
    await more.tap();
    await expect(dlg).toHaveCount(1);
  };

  await open();
  const box = (await dlg.boundingBox())!;
  expect(box.height, "bảng không phủ kín màn").toBeLessThan(844 - 60);
  expect(box.y + box.height).toBeGreaterThanOrEqual(843); // dính đáy
  expect(await dlg.locator("a").count()).toBe(navFor("teacher").flatMap((g) => g.items).length - MOBILE_PRIMARY.teacher.length);

  // 1) bấm ngoài (vùng nền phía trên bảng)
  await page.touchscreen.tap(195, 20);
  await expect(dlg).toHaveCount(0);
  await expect(more).toBeFocused();

  // 2) vuốt xuống (cử chỉ thật qua CDP)
  await open();
  const cdp = await context.newCDPSession(page);
  const y0 = box.y + 40;
  await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [{ x: 195, y: y0 }] });
  for (let i = 1; i <= 6; i++) await cdp.send("Input.dispatchTouchEvent", { type: "touchMove", touchPoints: [{ x: 195, y: y0 + i * 30 }] });
  await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
  await expect(dlg).toHaveCount(0);
  await expect(more).toBeFocused();

  // 3) Esc
  await open();
  await page.keyboard.press("Escape");
  await expect(dlg).toHaveCount(0);
  await expect(more).toBeFocused();
  await context.close();
});
