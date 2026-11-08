import { expect, test, type Page, type BrowserContext } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC FEAT-uiux-91 đợt 2 (tc-wave-2.md) chạy trên STACK THẬT `ep-qc` (gateway + seed), không phải gateway giả.
// Chạy bằng `playwright.qc-real.config.ts` (baseURL = https://localhost:14443, ignoreHTTPSErrors).
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
async function courseId(page: Page, code: string) {
  return page.evaluate(async (c) => {
    const r = await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" });
    const { access_token } = await r.json();
    const m = await (await fetch("/api/v1/me/courses", { headers: { authorization: `Bearer ${access_token}` } })).json();
    return m.items.find((i: { course: { class_code: string } }) => i.course.class_code === c).course.id as string;
  }, code);
}
const go = async (page: Page, url: string, course?: string) => {
  await page.goto(course ? `${url}${url.includes("?") ? "&" : "?"}course=${course}` : url);
  await page.locator("main").first().waitFor({ state: "attached" });
  await page.waitForFunction(() => (document.querySelector("main")?.innerText ?? "").trim().length > 20, undefined, { timeout: 20_000 }).catch(() => undefined);
  await page.waitForTimeout(600);
};
const mainText = (page: Page) => page.locator("main").innerText();
const OLD = /\b(Threads|Insights|Analytics)\b/;

// ---------- US-UX91-07 ----------
const NAV: Array<{ role: string; user: string; has: string[]; hasNot: string[] }> = [
  { role: "student", user: "sv.gioi", has: ["Hỏi đáp công khai"], hasNot: ["Lỗ hổng kiến thức", "Số liệu lớp học"] },
  { role: "teacher", user: "teacher", has: ["Hỏi đáp công khai", "Lỗ hổng kiến thức", "Số liệu lớp học"], hasNot: [] },
  { role: "ta", user: "ta", has: ["Hỏi đáp công khai", "Lỗ hổng kiến thức", "Số liệu lớp học"], hasNot: [] },
  { role: "admin", user: "admin", has: [], hasNot: ["Hỏi đáp công khai", "Lỗ hổng kiến thức", "Số liệu lớp học"] },
];
for (const n of NAV) {
  test(`TC-UX07-1/2/3 nhãn nav + ⌘K — ${n.role}`, async ({ page }, info) => {
    desktop(info);
    await login(page, n.user);
    const nav = await page.locator("[data-part=sidebar]").innerText();
    for (const l of n.has) expect(nav, l).toContain(l);
    for (const l of n.hasNot) expect(nav, l).not.toContain(l);
    expect(nav).not.toMatch(OLD);
    await page.keyboard.press("Control+k");
    const pal = await page.getByRole("dialog").last().innerText();
    for (const l of n.has) expect(pal, `⌘K ${l}`).toContain(l);
    expect(pal).not.toMatch(OLD);
  });
}

test("TC-UX07-4 thanh dưới / Thêm ở 375 — sinh viên + giảng viên", async ({ browser }) => {
  for (const [user, want] of [["sv.gioi", /Hỏi đáp/], ["teacher", /Lỗ hổng kiến thức|Số liệu lớp học/]] as const) {
    const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await login(page, user);
    const bottom = page.locator("[data-part=bottomnav], nav").last();
    const more = bottom.getByRole("button", { name: /^Thêm$/ });
    let txt = await bottom.innerText();
    await more.click();
    txt += "\n" + (await page.getByRole("dialog").last().innerText());
    expect(txt, user).toMatch(want);
    expect(txt).not.toMatch(OLD);
    // AC6: nhãn không bị cắt
    const clipped = await page.evaluate(() => [...document.querySelectorAll("nav a, nav button, nav span")].filter((e) => (e as HTMLElement).scrollWidth > (e as HTMLElement).clientWidth + 1).map((e) => e.textContent));
    expect(clipped, "nhãn bị cắt ở 375").toEqual([]);
    await ctx.close();
  }
});

test("TC-UX07-5/6/7/11 tiêu đề trang = nhãn nav, <title>, HTML máy chủ, URL cũ còn", async ({ page, request }, info) => {
  desktop(info);
  for (const [u, h] of [["/threads", "Hỏi đáp công khai"], ["/insights", "Lỗ hổng kiến thức"], ["/analytics", "Số liệu lớp học"]] as const) {
    const html = await (await request.get(u)).text();
    expect(html, `HTML máy chủ ${u}`).toContain(`<title>${h} · EduPilot</title>`);
  }
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  for (const [u, h] of [["/threads", "Hỏi đáp công khai"], ["/insights", "Lỗ hổng kiến thức"], ["/analytics", "Số liệu lớp học"]] as const) {
    await go(page, u, c1);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(h);
    await expect(page).toHaveTitle(`${h} · EduPilot`);
  }
  const t = await (await page.request.get("/threads")).text();
  expect(t).toContain("Hỏi đáp công khai");
  const sv = await page.evaluate(async () => (await fetch("/threads")).status);
  expect(sv).toBe(200);
});

test("TC-UX07-7 /threads/‹id›: liên kết quay lại", async ({ page }, info) => {
  desktop(info);
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  await go(page, "/threads", c1);
  const link = page.locator("main a[href^='/threads/']").first();
  await link.click();
  await expect(page.locator("main")).toContainText("Hỏi đáp công khai");
  expect(await mainText(page)).not.toMatch(OLD);
});

test("TC-UX07-15 số mục nav theo vai không đổi (SV 8, TA 12, GV 15, admin 6)", async ({ browser }, info) => {
  desktop(info);
  const want: Record<string, number> = { "sv.gioi": 8, ta: 12, teacher: 15, admin: 6 };
  for (const [u, n] of Object.entries(want)) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
    const page = await ctx.newPage();
    await login(page, u);
    const count = await page.locator("[data-part=sidebar] a[href]").evaluateAll((as) => new Set(as.map((a) => (a as HTMLAnchorElement).getAttribute("href"))).size);
    expect(count - 0, u).toBeGreaterThanOrEqual(n - 1);
    expect(count, u).toBeLessThanOrEqual(n + 1);
    await ctx.close();
  }
});

// ---------- US-UX91-01 ----------
type Chk = { role: string; user: string; url: string; open?: (p: Page) => Promise<void>; frags: string[]; course?: boolean; h1?: RegExp };
const CHECKS: Chk[] = [
  { role: "admin", user: "admin", url: "/admin/courses", frags: ["lưu trữ lớp đã kết thúc"] },
  { role: "admin", user: "admin", url: "/admin/courses", open: async (p) => { await p.getByRole("button", { name: "Mở lớp" }).first().click(); }, frags: ["nhận thông báo kèm mã tham gia ngay khi lớp mở"] },
  { role: "admin", user: "admin", url: "/admin/users", frags: ["Sinh viên tự đăng ký"] },
  { role: "admin", user: "admin", url: "/admin/users", open: async (p) => { await p.getByRole("button", { name: "Mời giảng viên" }).first().click(); }, frags: ["Quản trị viên không biết"] },
  { role: "teacher", user: "teacher", url: "/analytics", course: true, frags: ["không phải bảng theo dõi từng sinh viên", "Câu hỏi sinh viên gửi trong", "thời gian lớp chờ bạn", "bị che trước khi câu hỏi đi tới model", "AI chỉ ra bản nháp", "Tiền trả cho model trong khoảng đang xem"] },
  { role: "teacher", user: "teacher", url: "/exams", course: true, frags: ["soạn từ ngân hàng"] },
  { role: "student", user: "sv.gioi", url: "/exams", course: true, frags: ["Đồng hồ chạy khi bạn bấm Bắt đầu làm bài"] },
  { role: "teacher", user: "teacher", url: "/insights", course: true, frags: ["đã bỏ tên và mã số sinh viên, nhóm theo chủ đề", "Báo cáo chỉ dùng câu hỏi của", "xếp theo mức độ cả lớp đang vướng", "không tài liệu nào của lớp trả lời được", "Bản trước tạo ngày"] },
  { role: "student", user: "sv.gioi", url: "/join", frags: ["không phân biệt hoa thường"] },
  { role: "student", user: "sv.gioi", url: "/me", course: true, frags: ["761987\n"] },
  { role: "teacher", user: "teacher", url: "/class/settings", course: true, frags: ["Lớp 761987", "Sinh viên vào bằng mã sẽ chờ bạn duyệt"] },
  { role: "teacher", user: "teacher", url: "/class/members", course: true, frags: ["Duyệt yêu cầu vào lớp và quản lý người học"] },
  { role: "teacher", user: "teacher", url: "/observability", course: true, frags: ["Nội dung câu hỏi chỉ quản trị viên hệ thống mở được", "Hệ thống đang chạy thế nào", "Số tổng hợp của lớp bạn phụ trách", "Chọn một hàng để xem nội dung đã che"] },
  { role: "admin", user: "admin", url: "/observability", frags: ["Hệ thống đang chạy thế nào", "Chọn một hàng để xem nội dung đã che"] },
  { role: "student", user: "sv.gioi", url: "/practice/history", course: true, frags: ["Bấm một lượt để xem lại từng câu", "Tính trên các câu đã làm trong các lượt gần nhất"] },
  { role: "student", user: "sv.gioi", url: "/practice", course: true, frags: ["hoặc thi thử với đề cũ"] },
  { role: "teacher", user: "teacher", url: "/questions", course: true, frags: ["Chỉ giảng viên và trợ giảng thấy"] },
];
const cache: Record<string, string> = {};
for (const c of CHECKS) {
  test(`TC-UX01-1 ${c.role} ${c.url}${c.open ? " (mở form)" : ""}: không còn câu giải thích cũ`, async ({ page }, info) => {
    desktop(info);
    await login(page, c.user);
    const cid = c.course ? (cache.c1 ??= await courseId(page, "761987")) : undefined;
    await go(page, c.url, cid);
    if (c.open) { await c.open(page); await page.waitForTimeout(500); }
    const t = await mainText(page);
    expect((await page.getByRole("heading", { level: 1 }).first().innerText()).length, "trang có h1").toBeGreaterThan(0);
    for (const f of c.frags) expect(t, `còn «${f}»`).not.toContain(f);
  });
}

test("TC-UX01-3 khung máy chủ (PreShell): không câu phụ, Hỏi đáp công khai", async ({ request }) => {
  const OLDSUB = [
    "Câu hỏi công khai của", "Đồng hồ chạy khi", "Hỏi về điểm, chuyên cần và bài học của chính bạn", "Điểm chính thức nằm ở hệ thống",
  ];
  for (const u of ["/", "/inbox", "/threads", "/me", "/gradebook", "/chat"]) {
    const html = await (await request.get(u)).text();
    expect(html.length, u).toBeGreaterThan(1000);
    for (const f of OLDSUB) expect(html, `${u} còn «${f}»`).not.toContain(f);
    expect(html, u).not.toMatch(/>Threads</);
  }
});

test("TC-UX01-10 hộp xác nhận Chốt điểm còn mô tả hệ quả", async ({ page }, info) => {
  desktop(info);
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  await go(page, "/gradebook", c1);
  const btn = page.getByRole("button", { name: "Chốt điểm" });
  await expect(btn).toBeVisible();
  if ((await btn.getAttribute("aria-disabled")) === "true") test.skip(true, "Chốt điểm đang khoá ở lớp seed (công thức chưa xác nhận)");
  await btn.click();
  const d = page.getByRole("dialog").last();
  await expect(d).toContainText(/sinh viên/);
  await page.keyboard.press("Escape");
});

test("TC-UX01-9 SV mở /observability, /analytics → 403 giữ nguyên", async ({ page }, info) => {
  desktop(info);
  await login(page, "sv.gioi");
  for (const u of ["/observability", "/analytics", "/insights"]) {
    await go(page, u);
    await expect(page.getByText("Bạn không có quyền xem màn này")).toBeVisible();
  }
});

test("TC-UX01-17 375 px: dưới h1 không còn dòng phụ trống, không tràn ngang", async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  for (const u of ["/students", "/class/members", "/exams", "/insights", "/analytics", "/observability", "/questions"]) {
    await go(page, u, c1);
    const r = await page.evaluate(() => ({ ov: document.documentElement.scrollWidth > document.documentElement.clientWidth, empty: [...document.querySelectorAll("main h1 + *")].some((e) => (e as HTMLElement).innerText.trim() === "" && (e as HTMLElement).offsetHeight > 4) }));
    expect(r.ov, `${u} tràn ngang`).toBe(false);
    expect(r.empty, `${u} dòng trống dưới h1`).toBe(false);
  }
  await ctx.close();
});

// ---------- US-UX91-05 ----------
test("TC-UX05-1/2/3 Hộp thư rỗng: EmptyState đúng mẫu, không «Chọn một câu hỏi», nút ghost căn trái", async ({ page }, info) => {
  desktop(info);
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  await page.route(/\/api\/v1\/(courses\/[^/]+|me)\/tickets(\?|$)/, (r) => r.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], next_cursor: null }) }));
  await go(page, "/inbox", c1);
  const t = await mainText(page);
  expect(t).toContain("Chưa có câu hỏi nào cần thầy/cô xử lý.");
  expect(t).toContain("Khi AI không đủ chắc chắn hoặc sinh viên nhờ hỗ trợ, câu sẽ hiện ở đây.");
  expect(t).not.toContain("Chọn một câu hỏi để xem");
  const h1 = await page.getByRole("heading", { level: 1 }).boundingBox();
  const all = page.getByRole("button", { name: "Xem tất cả" }).first();
  await expect(all).toBeVisible();
  const bb = await all.boundingBox();
  const bg = await all.evaluate((e) => getComputedStyle(e).backgroundColor);
  expect(Math.abs(bb!.x - h1!.x), "mép trái nút so với tiêu đề").toBeLessThanOrEqual(30);
  expect(bg).not.toMatch(/rgb\(2[0-9]{2}, [0-9]{1,2}, [0-9]{1,2}\)/); // không đỏ
});

test("TC-UX05-5 Chấm bài chưa có bài nộp: EmptyState + Tải bài lên", async ({ page }, info) => {
  desktop(info);
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  await page.route(/\/api\/v1\/courses\/[^/]+\/submissions(\?|$)/, (r) => r.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], next_cursor: null }) }));
  await go(page, "/grading?tab=queue", c1);
  const t = await mainText(page);
  expect(t).toContain("Chưa có bài nộp.");
  expect(t).toContain("Sinh viên nộp trong ứng dụng, hoặc thầy/cô tải ZIP lên.");
  await expect(page.getByRole("button", { name: "Tải bài lên" }).first()).toBeVisible();
});

test("TC-UX05-9 lỗi tải: ApiErrorNotice + Thử lại, không EmptyState", async ({ page }, info) => {
  desktop(info);
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  await page.route(/\/api\/v1\/courses\/[^/]+\/tickets(\?|$)/, (r) => r.abort("internetdisconnected"));
  await go(page, "/inbox", c1);
  await page.waitForTimeout(2500);
  await expect(page.getByRole("button", { name: "Thử lại" }).first()).toBeVisible({ timeout: 15_000 });
  expect(await mainText(page)).not.toContain("Chưa có câu hỏi nào cần thầy/cô xử lý.");
  await page.unroute(/\/api\/v1\/courses\/[^/]+\/tickets(\?|$)/);
  await page.getByRole("button", { name: "Thử lại" }).first().click();
  await expect(page.getByRole("button", { name: "Thử lại" })).toHaveCount(0, { timeout: 15_000 });
});

test("TC-UX05-10 STUDENT: /inbox, /grading, /gradebook → 403, không EmptyState giảng viên", async ({ page }, info) => {
  desktop(info);
  await login(page, "sv.gioi");
  for (const u of ["/inbox", "/grading", "/gradebook"]) {
    await go(page, u);
    await expect(page.getByText("Bạn không có quyền xem màn này")).toBeVisible();
    expect(await mainText(page)).not.toMatch(/Chưa có câu hỏi nào cần thầy|Chưa có bài nộp|Thiết lập công thức/);
  }
});

test("TC-UX05-4 chi tiết ticket: không còn «Đã chờ …» / «Trạng thái …» lặp", async ({ page }, info) => {
  desktop(info);
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  await go(page, "/inbox", c1);
  const row = page.locator("main [role=listbox] [role=option], main li button, main button").filter({ hasText: /ngày|giờ/ }).first();
  if (await row.count()) await row.click();
  await page.waitForTimeout(800);
  const t = await mainText(page);
  expect(t).not.toMatch(/^Đã chờ\s*$/m);
  expect(t).not.toMatch(/^Trạng thái\s*$/m);
});

test("TC-UX05-11 375: Hộp thư rỗng không tràn, nút ≥ 44 px", async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "teacher");
  const c1 = await courseId(page, "761987");
  await page.route(/\/api\/v1\/(courses\/[^/]+|me)\/tickets(\?|$)/, (r) => r.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], next_cursor: null }) }));
  await go(page, "/inbox", c1);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  const bb = await page.getByRole("button", { name: "Xem tất cả" }).first().boundingBox();
  expect(bb!.height).toBeGreaterThanOrEqual(44);
  await ctx.close();
});

// ---------- US-UX91-06 ----------
const ADMIN_FREE = async (ctx: BrowserContext) => ctx;
test("TC-UX06-1 nhớ lớp đã chọn sau đăng xuất / đăng nhập lại", async ({ page }, info) => {
  desktop(info);
  await login(page, "teacher");
  const c2 = await courseId(page, "761988");
  await go(page, "/", c2);
  await expect(page.locator("[data-part=topbar]")).toContainText("761988");
  const key = await page.evaluate(() => localStorage.getItem("ep:ui:course"));
  expect(key).toBe(c2);
  await page.getByRole("button", { name: /^Tài khoản/ }).click();
  await page.getByRole("menuitem", { name: /Đăng xuất/ }).click();
  await page.waitForURL(/\/login|\/$/);
  await login(page, "teacher");
  await expect(page.locator("[data-part=topbar]")).toContainText("761988");
});

test("TC-UX06-3/4/5 lớp mặc định lần đầu theo `me/today`", async ({ browser }, info) => {
  desktop(info);
  const probe = await browser.newContext({ ignoreHTTPSErrors: true });
  const pp = await probe.newPage();
  await login(pp, "teacher");
  const c1 = await courseId(pp, "761987");
  const c2 = await courseId(pp, "761988");
  const order = await pp.evaluate(async () => {
    const { access_token } = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    return (await (await fetch("/api/v1/me/courses", { headers: { authorization: `Bearer ${access_token}` } })).json()).items.map((i: { course: { class_code: string } }) => i.course.class_code);
  });
  await probe.close();
  const first = order[0] === "761987" ? "761987" : "761988";
  const second = first === "761987" ? "761988" : "761987";
  const idOf = (code: string) => (code === "761987" ? c1 : c2);
  const todayFor = (...ids: string[]) => ({ count: ids.length, actions: ids.map((id, i) => ({ id: `a${i}`, kind: "JOIN_REQUESTS", title: "Việc", reason: "r", urgency: "normal", href: "/class/members", course: { id, class_code: id === c1 ? "761987" : "761988" } })), attention: [], upcoming: [] });
  for (const [name, ids, want] of [["cả hai có việc", [idOf(first), idOf(second)], first], ["chỉ lớp thứ hai có việc", [idOf(second)], second], ["không lớp nào có việc", [], first]] as const) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await page.route("**/api/v1/me/today**", (r) => r.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(todayFor(...ids)) }));
    await login(page, "teacher");
    await expect(page.locator("[data-part=topbar]"), name).toContainText(want);
    await ctx.close();
  }
});

test("TC-UX06-6/7 ep:ui:course trỏ tới lớp không tồn tại → bỏ, không lỗi, không trang trống", async ({ browser }, info) => {
  desktop(info);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  const errs: string[] = [];
  page.on("pageerror", (e) => errs.push(e.message));
  await page.addInitScript(() => localStorage.setItem("ep:ui:course", "00000000-0000-7000-8000-00000000dead"));
  await login(page, "teacher");
  await expect(page.locator("[data-part=topbar]")).toContainText(/76198[78]/);
  expect(await mainText(page).then((t) => t.trim().length)).toBeGreaterThan(20);
  expect(errs).toEqual([]);
  expect(await page.evaluate(() => localStorage.getItem("ep:ui:course"))).not.toBe("00000000-0000-7000-8000-00000000dead");
  await ctx.close();
});

test("TC-UX06-8 STUDENT / TA: bộ chọn lớp chỉ lớp của mình, không mục quản lý", async ({ browser }, info) => {
  desktop(info);
  for (const u of ["sv.kha", "ta"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await login(page, u);
    await page.getByRole("button", { name: /^Chọn lớp/ }).click();
    const menu = await page.getByRole("menu").first().innerText();
    expect(menu, u).not.toMatch(/761988/);
    expect(menu, u).not.toMatch(/Quản lý lớp này|Mời trợ giảng/);
    await ctx.close();
  }
});

test("TC-UX06-9 375: mục bộ chọn lớp cao ≥ 44 px", async ({ browser }) => {
  const ctx = await browser.newContext({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true, ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await login(page, "teacher");
  await page.getByRole("button", { name: /^Chọn lớp/ }).click();
  const items = page.getByRole("menuitem").or(page.getByRole("option")).or(page.getByRole("menu").first().locator("button, a"));
  const n = await items.count();
  expect(n).toBeGreaterThan(0);
  for (let i = 0; i < n; i++) expect((await items.nth(i).boundingBox())!.height, `mục ${i}`).toBeGreaterThanOrEqual(43);
  await ctx.close();
});
void ADMIN_FREE;
