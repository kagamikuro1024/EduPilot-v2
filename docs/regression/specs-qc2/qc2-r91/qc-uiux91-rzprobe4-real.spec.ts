// QC SmkS: probe UX21-PQ — SV đã đăng nhập mở `/`: mỗi 1 s ghi h1, tiêu đề tab, số lần gọi /auth/refresh
import { test } from "@playwright/test";
import { open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
for (const user of ["sv.gioi", "teacher"]) test(`probe / sau đăng nhập ${user}`, async ({ browser }) => {
  test.setTimeout(120_000);
  const { page } = await open(browser, user);
  const reqs: string[] = [];
  page.on("request", (r) => { if (/auth\/refresh|\/me\b|today/.test(r.url())) reqs.push(r.method() + " " + new URL(r.url()).pathname); });
  await page.goto("/");
  for (let i = 1; i <= 14; i++) {
    await page.waitForTimeout(1000);
    console.log(`PROBE ${user} t=${i}s url=${new URL(page.url()).pathname} h1="${await page.locator("h1").first().innerText().catch(() => "—")}" topbar=${await page.locator("[data-part=topbar]").count()} req=${reqs.length}`);
  }
  console.log("PROBE reqs", JSON.stringify(reqs));
});
