import { execSync } from "node:child_process";
import { type APIRequestContext, expect, test } from "@playwright/test";
import { call, courseId, login, needReal, shot, sql, uiLogin } from "./qc-real-lib";

// QC sprint 11 — công tắc nhà cung cấp 9router ở /settings/llm (ADMIN): tắt → tác vụ INSIGHT rơi về fake (env); bật → 9router.
// Tiền điều kiện: `python3 llm_route.py real` đã đặt tuyến CHAT/CLASSIFY/UTILITY… sang «QC 9router».
needReal();

// Chat chỉ gọi LLM khi truy xuất có ngữ cảnh (embedding giả không có ngữ nghĩa) → dùng tác vụ INSIGHT (đặt tên chủ đề) để chứng minh tuyến theo công tắc.
const used = new Set<number>();
async function gen(request: APIRequestContext) {
  execSync("docker exec ep-qc-redis-1 sh -c 'redis-cli --scan --pattern \"ep:rl:insight-generate:*\" | xargs -r redis-cli del'");
  const teacher = await login(request, "teacher");
  const cid = await courseId(request, "teacher");
  const t0 = new Date().toISOString();
  let r = await call(request, teacher, "POST", `/courses/${cid}/insights/knowledge-gap/generate`, { period_days: 120 });
  for (const period of [120, 30, 14, 7]) {
    if (used.has(period)) continue;
    r = await call(request, teacher, "POST", `/courses/${cid}/insights/knowledge-gap/generate`, { period_days: period });
    if (r.status === 202) { used.add(period); break; }
  }
  expect(r.status, r.text.slice(0, 200)).toBe(202);
  await expect.poll(() => sql(`select count(*) from llm_audit where task='INSIGHT' and created_at > '${t0}'`)[0][0], { timeout: 180_000, intervals: [3000] }).not.toBe("0");
  return sql(`select provider, model, status from llm_audit where task='INSIGHT' and created_at > '${t0}' order by created_at desc limit 1`)[0];
}

test("@real LLM-TOGGLE tắt 9router → chat chạy bản ghi fake; bật lại → chat qua 9router", async ({ request, browser }) => {
  const adm = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: process.env.QC_BASE ?? "https://localhost:943" });
  const ap = await adm.newPage();
  await uiLogin(ap, "admin");
  await ap.goto("/settings/llm");
  const row = ap.getByText("QC 9router").first();
  await expect(row).toBeVisible();
  await shot(ap, "llm-toggle-0-initial");
  // vị trí công tắc của hàng «QC 9router»
  const holder = ap.locator("li, tr, div").filter({ hasText: "QC 9router" }).filter({ has: ap.getByRole("switch") }).last();
  const toggle = holder.getByRole("switch");
  await expect(toggle).toBeChecked();

  const on1 = await gen(request);
  expect(on1[1], `INSIGHT khi 9router bật: ${on1}`).toBe("ag/gemini-3.8-flash");

  await toggle.click();
  await expect(holder.getByText("Đang tắt")).toBeVisible();
  await shot(ap, "llm-toggle-1-off");
  await ap.waitForTimeout(3000); // nạp lại nóng
  const off = await gen(request);
  expect(off[1], `INSIGHT khi 9router tắt (kỳ vọng fake): ${off}`).not.toBe("ag/gemini-3.8-flash");
  expect(off[0]).toMatch(/fake/);

  await toggle.click();
  await expect(holder.getByText("Đang bật")).toBeVisible();
  await shot(ap, "llm-toggle-2-on");
  await ap.waitForTimeout(3000);
  const on2 = await gen(request);
  expect(on2[1], `INSIGHT sau khi bật lại: ${on2}`).toBe("ag/gemini-3.8-flash");
  await adm.close();
});
