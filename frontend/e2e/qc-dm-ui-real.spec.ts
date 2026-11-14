import { test } from "@playwright/test";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: QC_REAL=1 playwright test -c playwright.qc-real.config.ts");
// QC sprint 10.5 L5: /settings/llm (khối «Quyết định nhanh») và /observability (bảng) trên stack thật, 1440 + 375, ADMIN và TEACHER. Ảnh ghi ra docs/sprints/10.5/qc/shots.
const PW = process.env.QC_PW ?? "";
const OUT = process.env.QC_OUT ?? "/tmp/qc-shots";
mkdirSync(OUT, { recursive: true });

async function login(browser: any, user: string, w: number, h: number) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page };
}

for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  test(`admin settings/llm + observability @${w}`, async ({ browser }, info) => {
    test.skip(info.project.name !== "desktop");
    test.setTimeout(120_000);
    const { ctx, page } = await login(browser, "admin", w, h);
    await page.goto("/settings/llm"); await page.waitForTimeout(2500);
    const txt = await page.locator("main").innerText();
    const i1 = txt.indexOf("Mô hình theo tác vụ"), i2 = txt.indexOf("Quyết định nhanh"), i3 = txt.indexOf("Mô hình tìm kiếm tài liệu");
    console.log(`QCINFO settings@${w} vị trí: tác vụ=${i1} quyết định=${i2} tìm kiếm=${i3} đúng thứ tự=${i1 >= 0 && i2 > i1 && (i3 < 0 || i2 < i3)}`);
    for (const label of ["Kiểm duyệt Threads", "Cờ chèn lệnh khi chấm", "Định tuyến câu hỏi chat"]) console.log(`QCINFO settings@${w} có «${label}»: ${txt.includes(label)}`);
    console.log(`QCINFO settings@${w} từ cấm: ${/\b(RAG|PII|fallback|trace|cascade|decision)\b/i.test(txt.slice(i2 >= 0 ? i2 : 0, (i2 >= 0 ? i2 : 0) + 1500))}`);
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
    console.log(`QCINFO settings@${w} tràn ngang: ${overflow}`);
    const sw = await page.locator("[role=switch]").evaluateAll((els: any[]) => els.map((e) => { const r = e.getBoundingClientRect(); return `${Math.round(r.width)}x${Math.round(r.height)}`; }));
    console.log(`QCINFO settings@${w} switch sizes: ${sw.join(",")}`);
    await page.screenshot({ path: `${OUT}/settings-llm-admin-${w}.png`, fullPage: true });
    await page.goto("/observability"); await page.waitForTimeout(3000);
    const o = await page.locator("main").innerText();
    console.log(`QCINFO observability@${w} có «Quyết định nhanh»: ${o.includes("Quyết định nhanh")}; vị trí sau bảng 7 ngày: ${o.indexOf("Quyết định nhanh") > o.indexOf("7 ngày")}`);
    const ov = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
    console.log(`QCINFO observability@${w} tràn ngang: ${ov}`);
    await page.screenshot({ path: `${OUT}/observability-admin-${w}.png`, fullPage: true });
    await ctx.close();
  });
}

test("teacher không thấy khối / bảng decision", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const { ctx, page } = await login(browser, "teacher", 1440, 900);
  await page.goto("/settings/llm"); await page.waitForTimeout(2000);
  const s = await page.locator("body").innerText();
  console.log(`QCINFO teacher settings: có «Quyết định nhanh»: ${s.includes("Quyết định nhanh")}; url=${page.url()}`);
  await page.goto("/observability"); await page.waitForTimeout(2000);
  const o = await page.locator("body").innerText();
  console.log(`QCINFO teacher observability: có «Quyết định nhanh»: ${o.includes("Quyết định nhanh")}; url=${page.url()}`);
  await ctx.close();
});
