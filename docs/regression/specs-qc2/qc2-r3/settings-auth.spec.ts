import { test } from "@playwright/test";
import { writeFileSync } from "node:fs";
import { asJwt } from "../../../../frontend/e2e/support/session";
import { acquireFakeApi, releaseFakeApi } from "../../../../frontend/e2e/support/fake-api-lock";
import { base, reset, script } from "../../../../frontend/e2e/support/llm-fixtures";
test.beforeEach(async ({}, info) => { test.skip(info.project.name !== "desktop", "1 lần"); });
test("P105-51: /settings/llm không token vs token thật", async ({ browser }) => {
  await acquireFakeApi();
  try {
    const out: Record<string, unknown> = {};
    // (a) không phiên: cookie mô phỏng cũ không còn tác dụng
    const c1 = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); await c1.addCookies([{ name: "ep_demo_role", value: "admin", url: "http://localhost:3310" }]); const p1 = await c1.newPage();
    const reqA: string[] = []; p1.on("request", (r) => { if (/\/api\/v1\/admin\/llm/.test(r.url())) reqA.push(r.url()); });
    await p1.goto("/settings/llm"); await p1.waitForTimeout(3000); out.noToken = { url: p1.url(), text: (await p1.locator("body").innerText()).replace(/\s+/g, " ").slice(0, 200), llmRequests: reqA.length };
    // (b) token Admin thật (JWT giả ký) + gateway giả: màn gọi apiClient thật
    const c2 = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p2 = await c2.newPage();
    await reset(p2); await script(p2, base()); await asJwt(p2, "ADMIN", { email: "admin@ptit.edu.vn", fullName: "Đỗ Hoàng Nam", expIn: 366 * 86400 });
    const reqB: string[] = []; p2.on("request", (r) => { if (/\/api\/v1\/admin\/llm/.test(r.url())) reqB.push(r.method() + " " + r.url().replace(/^.*\/api\/v1/, "")); });
    await p2.goto("/settings/llm"); await p2.locator("main").first().waitFor(); await p2.waitForLoadState("networkidle"); await p2.waitForTimeout(800);
    out.withToken = { url: p2.url(), llmRequests: [...new Set(reqB)], h1: await p2.locator("h1").first().innerText(), hasProviders: /OpenAI/.test(await p2.locator("main").innerText()) };
    writeFileSync("../docs/regression/evidence-3/gate-self/settings-auth.json", JSON.stringify(out, null, 1)); console.log("AUTH " + JSON.stringify(out));
  } finally { releaseFakeApi(); }
});
