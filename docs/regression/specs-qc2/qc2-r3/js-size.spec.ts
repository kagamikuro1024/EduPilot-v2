import { test } from "@playwright/test";
import { appendFileSync, writeFileSync } from "node:fs";
import { asDemo, asJwt } from "../../../../frontend/e2e/support/session";
import { acquireFakeApi, releaseFakeApi } from "../../../../frontend/e2e/support/fake-api-lock";
import { base, reset, script } from "../../../../frontend/e2e/support/llm-fixtures";
import { gradebookApi } from "../../../../frontend/e2e/support/gradebook-fixtures";
test.beforeEach(async ({}, info) => { test.skip(info.project.name !== "desktop", "1 lần"); });
const R: Array<[string, string, string, boolean?]> = [["/", "/", "student"], ["/chat", "/chat", "student"], ["/threads", "/threads", "student"], ["/inbox", "/inbox", "teacher"], ["/gradebook", "/gradebook", "teacher"], ["/me", "/me", "student"], ["/settings/llm", "/settings/llm", "admin", true], ["/dev/ui", "/dev/ui", ""]];
test("JS truyền (nén) mỗi route lần đầu", async ({ browser }) => {
  const out: Record<string, number> = {};
  for (const [name, path, role, adm] of R) {
    if (adm) await acquireFakeApi();
    try {
      const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const page = await ctx.newPage();
      if (adm) { await reset(page); await script(page, base()); await asJwt(page, "ADMIN", { email: "admin@ptit.edu.vn", fullName: "Đỗ Hoàng Nam", expIn: 366 * 86400 }); }
      else if (role) await asDemo(ctx, role as "student" | "teacher", { expIn: 366 * 86400 });
      if (path === "/gradebook") await gradebookApi(page);
      let total = 0; const pend: Promise<void>[] = [];
      page.on("response", (r) => { if (r.request().resourceType() === "script" && r.url().includes("/_next/")) pend.push(r.request().sizes().then((s) => { total += s.responseBodySize; }).catch(() => undefined)); });
      await page.goto(path); await page.locator("main").first().waitFor(); await page.waitForLoadState("networkidle"); await page.waitForTimeout(500); await Promise.all(pend);
      out[name] = total; await ctx.close();
    } finally { if (adm) releaseFakeApi(); }
  }
  writeFileSync("../docs/regression/evidence-3/gate-self/js-size.json", JSON.stringify(out, null, 1)); console.log("JS " + JSON.stringify(out));
});
