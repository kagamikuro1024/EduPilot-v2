import { test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { cid, go, login } from "./lib";
const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], { encoding: "utf8" }).trim();
test("explore 06-46", async ({ browser }) => {
  test.setTimeout(150000);
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage(); await login(p, "teacher");
  p.on("response", (r) => { if (r.status() >= 400) console.log("HTTP", r.status(), r.url().split("/api/v1")[1]?.slice(0, 120)); });
  const stu = sql(`select e.user_id from enrollments e join courses c on c.id=e.course_id where c.class_code='QD8A2' order by 1 limit 1`);
  await go(p, `/students/${stu}?course=${cid("QD8A2")}`); await p.waitForTimeout(4000);
  console.log("TEXT", (await p.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 300));
  const stu2 = sql(`select e.user_id from enrollments e join courses c on c.id=e.course_id where c.class_code='QD8B1' and e.user_id<>(select id from users where email='teacher@edupilot.local') order by 1 limit 1`);
  await go(p, `/students/${stu2}?course=${cid("QD8A2")}`); await p.waitForTimeout(3000);
  console.log("STU1 is teacher?", sql(`select email from users where id='${stu}'`));
});
