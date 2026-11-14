import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { settleGoto } from "./support/hydrate";
import { mockStudentToday } from "./support/screens";
import { asDemo } from "./support/session";

// OPS-VPS (landing): `/` chưa đăng nhập → trang giới thiệu tĩnh; đã đăng nhập → "Hôm nay"; route sâu vẫn về /login?next=.
test.beforeEach(({ page }) => settleGoto(page));
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const anonymous = (page: Page) => page.route("**/api/v1/auth/refresh", (r) => r.fulfill({ status: 401, contentType: "application/json", headers: cors, body: JSON.stringify({ code: "UNAUTHENTICATED", message: "x", trace_id: "c".repeat(32) }) }));

test("landing anonymous: tiêu đề, nút Đăng nhập / Tạo tài khoản, chân trang, không gọi API nào (kể cả refresh), không tài khoản demo", async ({ page }) => {
  const api: string[] = [];
  page.on("request", (q) => q.url().includes("/api/v1/") && api.push(new URL(q.url()).pathname));
  await anonymous(page);
  await page.goto("/");
  await expect(page).toHaveURL(`${BASE_URL}/`);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Vận hành lớp học có trợ lý AI");
  await expect(page.locator("[data-part=landing]").getByRole("link", { name: "Đăng nhập" }).first()).toBeVisible();
  await expect(page.getByRole("link", { name: "Tạo tài khoản" }).first()).toHaveAttribute("href", "/register");
  await expect(page.getByText("Đồ án tốt nghiệp · Lê Văn Quang Trung")).toBeVisible();
  await expect(page.getByRole("link", { name: "Mã nguồn trên GitHub" })).toHaveAttribute("href", /github\.com\/kagamikuro1024\/EduPilot-v2$/);
  await expect(page.getByText("Dữ liệu trên bản demo là dữ liệu mô phỏng.")).toBeVisible();
  const text = await page.locator("body").innerText();
  expect(text).not.toMatch(/@edupilot\.local|Edupilot#|mật khẩu mặc định/i);
  await expect(page.locator("[data-part=landing] h2")).toHaveText(["Bài nộp đi qua ba bước", "EduPilot trong ba màn", "Ba số đo trên dữ liệu mô phỏng", "Việc của giảng viên và sinh viên", "Lớp của bạn trên EduPilot"]);
  await expect(page.locator("[data-part=sidebar]:visible, [data-part=topbar]:visible")).toHaveCount(0); // không khung ứng dụng
  expect(api).toEqual([]); // US-21: máy chưa từng đăng nhập thì `/` không gọi làm mới phiên
  await page.getByRole("link", { name: "Đăng nhập" }).first().click();
  await expect(page).toHaveURL(`${BASE_URL}/login`);
});

test("landing có cờ 'đã từng đăng nhập' ⇒ vẫn gọi làm mới phiên (US-21 AC2); mất cờ vẫn thấy giới thiệu", async ({ page }) => {
  const refresh: string[] = [];
  page.on("request", (q) => q.url().includes("/api/v1/auth/refresh") && refresh.push(q.method()));
  await anonymous(page);
  await page.addInitScript(() => localStorage.setItem("ep_returning", "1"));
  await page.goto("/");
  await expect.poll(() => refresh.length).toBeGreaterThan(0);
});

test("landing mất cờ nhưng còn cookie: bấm Đăng nhập (chuyển trang trong ứng dụng) thì thử làm mới phiên bằng cookie (US-21 AC3)", async ({ page }) => {
  const refresh: string[] = [];
  page.on("request", (q) => q.url().includes("/api/v1/auth/refresh") && refresh.push(q.method()));
  await anonymous(page);
  await page.goto("/");
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  expect(refresh).toEqual([]); // ở `/` chưa có cờ: không gọi
  await page.getByRole("link", { name: "Đăng nhập" }).first().click();
  await expect(page).toHaveURL(`${BASE_URL}/login`);
  await expect.poll(() => refresh.length).toBe(1);
});

test("landing signed in: đã đăng nhập thì `/` là Hôm nay, không có trang giới thiệu", async ({ page, context }) => {
  await asDemo(context, "student");
  await mockStudentToday(page);
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible();
  await expect(page.getByRole("heading", { level: 1 })).not.toHaveText(/Vận hành lớp học/);
  await expect(page.locator("[data-part=landing]")).toHaveCount(0);
  await expect(page).toHaveTitle("Hôm nay · EduPilot");
});

test("landing deep link: /gradebook chưa đăng nhập vẫn về /login?next=", async ({ page }) => {
  await anonymous(page);
  await page.goto("/gradebook");
  await expect(page).toHaveURL(/\/login\?next=%2Fgradebook$/);
});

test("landing back from login: logo ở /login quay về trang giới thiệu", async ({ page }) => {
  await anonymous(page);
  await page.goto("/login");
  await page.getByRole("link", { name: /trang giới thiệu/ }).click();
  await expect(page).toHaveURL(`${BASE_URL}/`);
  await expect(page.locator("[data-part=landing]")).toBeVisible();
});

test("landing stale hint: dấu 'đã từng đăng nhập' nhưng phiên hết hạn → trang giới thiệu hiện, dấu bị gỡ", async ({ page }) => {
  await anonymous(page);
  await page.addInitScript(() => localStorage.setItem("ep_returning", "1"));
  await page.goto("/");
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  await expect.poll(() => page.evaluate(() => localStorage.getItem("ep_returning"))).toBeNull();
});

for (const width of [1440, 375]) {
  test(`landing responsive ${width}: không cuộn ngang, axe không lỗi nghiêm trọng, ảnh có kích thước`, async ({ page }) => {
    await anonymous(page);
    await page.setViewportSize({ width, height: width === 375 ? 812 : 900 });
    await page.emulateMedia({ reducedMotion: "reduce" }); // axe đo hình đã xong, không đo giữa chừng của chuyển động (hết chập chờn color-contrast)
    await page.goto("/");
    await expect(page.locator("[data-part=landing]")).toBeVisible();
    await page.evaluate(async () => { for (let y = 0; y < document.body.scrollHeight; y += 600) { window.scrollTo(0, y); await new Promise((r) => setTimeout(r, 40)); } window.scrollTo(0, 0); }); // nạp hết ảnh lười
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    for (const img of await page.locator("[data-part=landing] img[src^='/landing/']").all()) {
      expect(await img.getAttribute("width")).toBeTruthy();
      expect(await img.getAttribute("height")).toBeTruthy();
      expect((await img.getAttribute("alt"))?.length).toBeGreaterThan(10);
    }
    const res = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
    expect(res.violations.filter((v) => v.impact === "critical" || v.impact === "serious").map((v) => `${v.id}: ${v.nodes.length}`)).toEqual([]);
  });
}

test("landing video đầy đủ: không tải khi chưa bấm; `Xem video` mở hộp thoại, Esc đóng; có lời thay cho video không tiếng", async ({ page }) => {
  const media: string[] = [];
  page.on("request", (q) => /\/landing\/gioi-thieu-v1\.(mp4|webm)/.test(q.url()) && media.push(new URL(q.url()).pathname));
  await anonymous(page);
  await page.goto("/");
  await page.waitForLoadState("networkidle");
  expect(media).toEqual([]);
  const dlg = page.locator("[data-part=landing] dialog");
  const video = dlg.locator("video");
  await expect(video).toHaveAttribute("preload", "none");
  expect(await video.evaluate((v: HTMLVideoElement) => v.autoplay || !v.paused)).toBe(false);
  await expect(dlg).toBeHidden();
  const open = page.getByRole("button", { name: "Xem video" });
  await open.click();
  await expect(dlg).toBeVisible();
  await expect(video).toHaveAttribute("poster", "/landing/gioi-thieu-v1.webp");
  expect(media).toEqual([]); // mở hộp thoại vẫn chưa tải video
  await page.keyboard.press("Escape");
  await expect(dlg).toBeHidden();
  await expect(open).toBeFocused();
  // US-UX91-08 AC5: đóng thì dừng và bỏ nguồn; mở lại có nguồn trở lại
  await expect(video.locator("source")).toHaveCount(0);
  expect(await video.evaluate((v: HTMLVideoElement) => v.currentSrc)).toBe("");
  await open.click();
  await expect(video.locator("source")).toHaveCount(2);
  await page.keyboard.press("Escape");
  await expect(page.locator("#video-loi")).toContainText("AI không tự công bố");
});

test("landing đoạn sản phẩm: chạy khi có chuyển động, Tạm dừng / Phát dừng hẳn và nhớ trong phiên; reduce thì đứng ở hình đã xong, không m-on, không nút", async ({ page, browser }) => {
  await anonymous(page);
  await page.goto("/");
  const demo = page.locator("[data-part=landing] [data-demo]");
  const running = () => demo.evaluate((e) => e.getAnimations({ subtree: true }).filter((a) => a.playState === "running").length);
  await expect.poll(running).toBeGreaterThan(0);
  expect(await page.evaluate(() => document.documentElement.classList.contains("m-on"))).toBe(true);
  const pause = page.getByRole("button", { name: "Tạm dừng" });
  await expect(pause).toHaveAttribute("aria-pressed", "false");
  await pause.click();
  await expect(page.getByRole("button", { name: "Phát tiếp" })).toHaveAttribute("aria-pressed", "true");
  expect(await running()).toBe(0);
  await page.reload();
  await expect(page.getByRole("button", { name: "Phát tiếp" })).toBeVisible(); // nhớ trong phiên
  await page.getByRole("button", { name: "Phát tiếp" }).click();
  await expect.poll(running).toBeGreaterThan(0);

  const reduced = await browser.newContext({ reducedMotion: "reduce" });
  const p2 = await reduced.newPage();
  await anonymous(p2);
  await p2.goto("/");
  await expect(p2.locator("[data-part=landing]")).toBeVisible();
  expect(await p2.evaluate(() => document.documentElement.classList.contains("m-on"))).toBe(false);
  expect(await p2.evaluate(() => document.getAnimations().length)).toBe(0);
  await expect(p2.getByRole("button", { name: /Tạm dừng|Phát tiếp/ })).toBeHidden();
  await expect(p2.locator("[data-part=landing] [data-demo]").getByText("Đã duyệt").first()).toHaveCSS("opacity", "1"); // hình đã xong
  await reduced.close();
});

test("landing hiện theo cuộn: ba số đo ẩn tới khi cuộn tới; 146 đếm lên rồi đứng; reduce hiện đủ ngay (AC9, AC14, AC15)", async ({ page }) => {
  await anonymous(page);
  await page.goto("/");
  const stat = page.locator("[data-part=landing] li").filter({ hasText: "yêu cầu gửi AI" }).first();
  const num = stat.locator("p").first();
  const opacity = () => num.evaluate((e) => Number(getComputedStyle(e).opacity));
  await expect(page.locator("html")).toHaveClass(/m-armed/); // bộ quan sát đã gắn: từ đây khối mới bị ẩn (AC15)
  await expect.poll(opacity).toBe(0); // ẩn (có chuyển tiếp 1 → 0 ngay sau khi gắn)
  await stat.scrollIntoViewIfNeeded();
  await expect.poll(opacity).toBe(1);
  await expect(num).toHaveText("146");

  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();
  await expect(page.locator("[data-part=landing]")).toBeVisible();
  expect(await opacity()).toBe(1);
  await expect(num).toHaveText("146");
});

test("landing số liệu: ba số đo nguyên văn nguồn, mỗi số có hình (AC9, AC24)", async ({ page }) => {
  await anonymous(page);
  await page.goto("/");
  const sec = page.locator("[data-part=landing] section[aria-labelledby=proof-title]");
  await expect(sec.getByRole("heading", { level: 3 })).toHaveText([
    "yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp",
    "sinh viên mẫu có điểm học phần khớp tính tay",
    "Chấm hàng loạt, thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 %",
  ]);
  await expect(sec.getByText("Bộ phát hiện đạt recall 0,960 trên 200 mẫu tự soạn.")).toBeVisible();
  await expect(sec.locator("li [aria-hidden=true]")).toHaveCount(3); // mỗi số một hình
});

test("landing vai: hai tab Giảng viên / Sinh viên (bấm, mũi tên, ảnh đổi); không script → hai danh sách xếp dọc", async ({ page, browser }) => {
  await anonymous(page);
  await page.goto("/");
  const sec = page.locator("[data-part=landing] section[aria-labelledby=roles-title]");
  const tabs = sec.getByRole("tab");
  await expect(tabs).toHaveText(["Giảng viên", "Sinh viên"]);
  await expect(tabs.first()).toHaveAttribute("aria-selected", "true");
  await expect(sec.locator("dt")).toHaveCount(8);
  await expect(sec.getByText("Hỏi đáp công khai", { exact: true })).toBeAttached();
  expect(await page.locator("[data-part=landing]").innerText()).not.toMatch(/Threads/);
  await expect(sec.getByRole("tabpanel")).toHaveCount(1); // tab bị ẩn khỏi cây truy cập
  await tabs.nth(1).click();
  await expect(tabs.nth(1)).toHaveAttribute("aria-selected", "true");
  await expect(sec.getByText("Chat riêng", { exact: true })).toBeVisible();
  await expect(sec.locator("[data-role-shots] img.is-on")).toHaveAttribute("alt", /điện thoại/); // ảnh đổi theo vai
  await tabs.nth(1).press("ArrowLeft");
  await expect(tabs.first()).toHaveAttribute("aria-selected", "true");
  await expect(sec.getByText("Chấm bài", { exact: true })).toBeVisible();

  const noJs = await browser.newContext({ javaScriptEnabled: false });
  const p2 = await noJs.newPage();
  await anonymous(p2);
  await p2.goto("/");
  const s2 = p2.locator("[data-part=landing] section[aria-labelledby=roles-title]");
  await expect(s2.getByRole("tab")).toHaveCount(0);
  await expect(s2.getByRole("heading", { level: 3 })).toHaveText(["Giảng viên", "Sinh viên"]);
  await expect(s2.getByText("Sổ điểm", { exact: true })).toBeVisible();
  await expect(s2.getByText("Nộp bài tập", { exact: true })).toBeVisible();
  await noJs.close();
});

test("landing khung ghim: ở 1440 khung dính và ba màn đổi theo cuộn; ở 1024 và reduce xếp dọc không ghim", async ({ page }) => {
  await anonymous(page);
  const story = page.locator("[data-part=landing] section[aria-labelledby=story-title]");
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/");
  await expect(story).toHaveClass(/is-pinned/);
  expect(await story.locator(":scope > div").evaluate((e) => getComputedStyle(e).position)).toBe("sticky");
  const current = () => story.locator("[aria-current=true] h3").textContent();
  await story.scrollIntoViewIfNeeded();
  const box = (await story.boundingBox())!;
  await page.evaluate((y) => window.scrollTo(0, y), box.y + (await page.evaluate(() => window.scrollY)) + 1.4 * 900);
  await expect.poll(current).toBe("Sổ điểm có giải trình");
  await page.setViewportSize({ width: 1024, height: 800 });
  await expect(story).not.toHaveClass(/is-pinned/);
  await expect(story.locator("img")).toHaveCount(3);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();
  await expect(story).not.toHaveClass(/is-pinned/);
});
