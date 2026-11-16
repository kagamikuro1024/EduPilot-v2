import { test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { cid, go, login, IDS } from "./lib";
const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], { encoding: "utf8" }).trim();
test("explore 08", async ({ browser }) => {
  test.setTimeout(170000);
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage(); await login(p, "teacher");
  const kha = IDS["QD8R6U"].students["sv.kha"];
  await go(p, `/students/${kha}?course=${cid("QD8R6U")}`); await p.waitForTimeout(3000);
  console.log("S1 URL", p.url(), "TABS", await p.getByRole("tab").allInnerTexts());
  const tab = p.getByRole("tab", { name: /Điểm/ }); if (await tab.count()) { await tab.first().click(); await p.waitForTimeout(2500); console.log("S1 TEXT", (await p.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 700)); }
  // thiếu điểm: lớp QD8R6Y
  const Y = cid("QD8R6Y"); const stu = IDS["QD8R6Y"].students["sv.gioi"]; const it = IDS["QD8R6Y"].items.final;
  console.log("backup", sql(`select id from grade_entries where course_id='${Y}' and student_id='${stu}' and item_id='${it}'`));
});
