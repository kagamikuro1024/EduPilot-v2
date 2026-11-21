import { test } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { cid, go, login } from "./lib";
test.setTimeout(240_000);
const OUT = "/tmp/qc2-s8r/uitech.tsv";
test("SV: lỗi 500 trên 7 route sinh viên — có «Chi tiết kỹ thuật» / trace_id / mã lỗi không", async ({ browser }) => {
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage(); await login(p, "sv.kha");
  const C = cid("QD8R6U"); const rows: string[] = [];
  let fail = false;
  await p.route(/\/api\/v1\/(?!auth\/|me\/courses|me\/notifications|events|me\/preferences)/, async (r) => { if (fail && r.request().method() === "GET") return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi", trace_id: "0af7651916cd43dd8448eb211c80319c" }) }); return r.continue(); });
  for (const path of ["/", "/chat", "/threads", "/practice", "/library", "/calendar", "/me"]) {
    fail = false; await go(p, `${path}?course=${C}`); await p.waitForTimeout(1500);
    fail = true; await p.reload(); await p.waitForTimeout(3500);
    const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); const det = await p.locator("main details").count();
    const hit = /Chi tiết kỹ thuật|trace_id|INTERNAL|NOT_FOUND|0af7651916/.test(t) || det > 0;
    rows.push(`${path}\t${hit ? "LỘ" : "sạch"}\t${t.slice(0, 160)}`);
  }
  appendFileSync(OUT, rows.join("\n") + "\n"); console.log(rows.join("\n"));
});
