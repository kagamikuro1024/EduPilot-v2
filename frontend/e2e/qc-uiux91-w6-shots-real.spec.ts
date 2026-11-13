import { test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC đợt 6: ảnh mốc trang giới thiệu (TC-UX08-56). Ghi vào /out/shots-w6 (bind mount /srv/work/qc/out).
const OUT = "../docs/sprints/9.1/qc/shots/wave-6";
const H2 = ["Xem EduPilot làm việc", "AI soạn nháp, giảng viên quyết, mã tính điểm", "Khác biệt, có số đo", "Cho giảng viên và sinh viên", "Bắt đầu với lớp của bạn"];
test("TC-UX08-56 ảnh mốc 1440 / 390", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  for (const [tag, vp, mobile] of [["1440", { width: 1440, height: 900 }, false], ["390", { width: 390, height: 844 }, true]] as const) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: vp, hasTouch: mobile, isMobile: mobile, locale: "vi-VN" });
    const page = await ctx.newPage();
    await page.goto("/", { waitUntil: "load" });
    await page.waitForTimeout(1500);
    await page.screenshot({ path: `${OUT}/landing-${tag}-0-dau.png` });
    const top = async (i: number, dy = 0) => {
      await page.evaluate(([h, d]) => { const e = [...document.querySelectorAll("h2")].find((x) => (x as HTMLElement).innerText.trim() === h)!; window.scrollTo(0, e.getBoundingClientRect().top + scrollY - 40 + (d as number)); }, [H2[i], dy] as const);
      await page.waitForTimeout(700);
    };
    if (!mobile) {
      await top(0, 0); await page.screenshot({ path: `${OUT}/landing-1440-1-ghim1.png` });
      await top(0, 1000); await page.screenshot({ path: `${OUT}/landing-1440-2-ghim2.png` });
      await top(0, 1800); await page.screenshot({ path: `${OUT}/landing-1440-3-ghim3.png` });
    } else {
      await top(0, 300); await page.screenshot({ path: `${OUT}/landing-390-1-buoc.png` });
    }
    await top(1); await page.screenshot({ path: `${OUT}/landing-${tag}-4-so-do.png` });
    await top(2); await page.screenshot({ path: `${OUT}/landing-${tag}-5-so-do-dem.png` });
    await top(3); await page.screenshot({ path: `${OUT}/landing-${tag}-6-vai.png` });
    await top(4); await page.screenshot({ path: `${OUT}/landing-${tag}-7-cta.png` });
    await ctx.close();
  }
});
