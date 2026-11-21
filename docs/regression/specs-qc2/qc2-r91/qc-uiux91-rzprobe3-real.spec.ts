// QC SmkS: probe UX16-20 — TA gọi POST finalize (api qua page.evaluate) có treo không
import { test } from "@playwright/test";
import { open, go, courseId, api } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
test("probe TA finalize", async ({ browser }) => {
  test.setTimeout(90_000);
  const { page } = await open(browser, "ta");
  const c = await courseId(page, process.env.QC_CLASS ?? "761987");
  await go(page, `/gradebook?course=${c}`, undefined, 3500);
  const t0 = Date.now();
  const r = await Promise.race([api(page, `/courses/${c}/gradebook/finalize`, { method: "POST", body: {} }), new Promise((res) => setTimeout(() => res("TREO>30s"), 30_000))]);
  console.log("PROBE finalize", JSON.stringify(r).slice(0, 200), Date.now() - t0, "ms");
});
