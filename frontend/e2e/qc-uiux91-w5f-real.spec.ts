/* eslint-disable @typescript-eslint/no-explicit-any */
import { test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}
const api = (page: Page, path: string) =>
  page.evaluate(async (p) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${p}`, { headers: { authorization: `Bearer ${a.access_token}` } });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, path);

test("TC-UX19-6/12 SV: danh sách không lộ câu trả lời AI chưa xác nhận; hộp thoại PII", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(150000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "sv.kha@edupilot.local");
  await page.goto("/threads");
  await page.waitForTimeout(4500);
  const li = page.locator("[data-part=thread-list] > li");
  console.log("INFO UX19-6 SV hàng:", await li.count(), JSON.stringify((await li.allInnerTexts()).slice(0, 4).map((t) => t.replace(/\s+/g, " ").slice(0, 170))));
  await page.getByRole("button", { name: "Đặt câu hỏi" }).first().click();
  const form = page.locator("[data-part=thread-form]");
  const boxes = form.getByRole("textbox");
  console.log("INFO UX19-12 ô nhập:", await boxes.count());
  await boxes.first().fill("Hỏi về điểm của em");
  await boxes.nth(1).fill("Em là 20229002, email sv.kha@edupilot.local, xin hỏi điểm của em");
  console.log("INFO UX19-12 nút:", JSON.stringify(await form.getByRole("button").allInnerTexts()));
  await page.waitForTimeout(1500);
  console.log("INFO UX19-12 thông báo:", JSON.stringify(await page.locator("[data-part=pii-notice]").allInnerTexts()));
  await form.getByRole("button", { name: /Đăng|Gửi/ }).first().click({ timeout: 8000 });
  await page.waitForTimeout(2500);
  console.log("INFO UX19-12 hộp thoại:", await page.getByRole("dialog").count(), JSON.stringify((await page.getByRole("dialog").first().innerText().catch(() => "")).replace(/\s+/g, " ").slice(0, 400)));
  await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/threads-pii-1440.png" });
  await ctx.close();
});

test("TC-UX19-11 SV không thuộc lớp: 403", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(100000);
  const c1 = await browser.newContext({ ignoreHTTPSErrors: true }); const p1 = await c1.newPage();
  await login(p1, "teacher@edupilot.local");
  const cs = (await api(p1, "/me/courses")).body.items.find((i: any) => i.course.class_code === "761988").course.id;
  const tid = (await api(p1, `/courses/${cs}/threads?limit=3`)).body.items[0].id;
  await c1.close();
  const c4 = await browser.newContext({ ignoreHTTPSErrors: true }); const p4 = await c4.newPage();
  await login(p4, "sv.moi@edupilot.local");
  const r1 = await api(p4, `/courses/${cs}/threads/${tid}`);
  const r2 = await api(p4, `/courses/${cs}/threads?limit=5`);
  console.log("INFO UX19-11 SV không thuộc lớp:", r1.status, r2.status);
  await c4.close();
});
