import { test } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { go, login, EV } from "./lib";
test.setTimeout(240_000);
const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "sh", "-c", `psql -U edupilot -d edupilot -At -c "${q}"; exit 0`], { encoding: "utf8" }).trim();
test("[06-31] LOAD1000 ảo hoá hàng", async ({ browser }) => {
  const id = sql("select id from courses where class_code='LOAD1000'"); const teacher = sql(`select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='${id}' and e.role_in_course='TEACHER' limit 1`).split("@")[0];
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage();
  await p.addInitScript(() => { (window as any).__lt = []; try { new PerformanceObserver((l) => { for (const e of l.getEntries()) (window as any).__lt.push(e.duration); }).observe({ type: "longtask", buffered: true }); } catch {} });
  await login(p, teacher);
  const reqs: string[] = []; p.on("request", (r) => { if (/gradebook\?/.test(r.url())) reqs.push(r.url().replace(/.*\/gradebook\?/, "")); });
  await go(p, `/gradebook?course=${id}`); await p.waitForTimeout(4000);
  const rowsSel = "main tbody tr"; let max = 0; const count = () => p.locator(rowsSel).count();
  max = Math.max(max, await count()); const first = await count();
  const scroller = await p.evaluateHandle(() => { const t = document.querySelector("main table"); let e: HTMLElement | null = t as HTMLElement; while (e && e.scrollHeight <= e.clientHeight + 5) e = e.parentElement; return e; });
  await p.evaluate(() => { (window as any).__lt.length = 0; });
  for (let i = 0; i < 40; i++) { await p.evaluate((e: any) => { e.scrollTop += 1200; }, scroller); await p.waitForTimeout(250); max = Math.max(max, await count()); }
  await p.evaluate((e: any) => { e.scrollTop = e.scrollHeight; }, scroller); await p.waitForTimeout(2500); max = Math.max(max, await count());
  const lt = await p.evaluate(() => (window as any).__lt as number[]); const ltMax = Math.max(0, ...lt);
  const lastText = await p.locator(rowsSel).last().innerText().catch(() => ""); const scrollTop = await p.evaluate((e: any) => [e.scrollTop, e.scrollHeight], scroller);
  await p.screenshot({ path: `${EV}/shots/s8r-06-31.png` });
  const ok = max <= 60 && max > 0 && ltMax < 50 && reqs.length >= 2;
  const line = `LOAD1000 ${teacher}: hàng trong DOM lúc đầu ${first}, tối đa khi cuộn ${max} (≤ 60); cuộn tới cuối ${JSON.stringify(scrollTop)}: hàng cuối «${lastText.replace(/\s+/g, " ").slice(0, 40)}»; ${reqs.length} lần gọi gradebook (cursor: ${reqs.slice(0, 3).join(" | ")}); long task ${lt.length} (lớn nhất ${Math.round(ltMax)} ms, yêu cầu < 50)`;
  appendFileSync("/tmp/qc2-s8r/ui0631.tsv", `TC-P6-06-31\t${ok ? "P" : "F"}\t${line}\n`); console.log(`${ok ? "P" : "F"} TC-P6-06-31 ${line}`);
});
