import { readFileSync } from "node:fs";
import { expect, test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// TC-UX16-24: điểm học phần 30 SV mẫu ở API sổ điểm khớp seed/expected_final_grades.csv (sprint chỉ đổi hiển thị).
test("TC-UX16-24 điểm học phần 30/30 khớp expected_final_grades.csv", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(process.env.QC_PW ?? "Edupilot#Seed-2026");
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  const rows = await page.evaluate(async () => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const h = { authorization: `Bearer ${a.access_token}` };
    const m = await (await fetch("/api/v1/me/courses", { headers: h })).json();
    const c = m.items.find((i: { course: { class_code: string } }) => i.course.class_code === "761987").course.id;
    const g = await (await fetch(`/api/v1/courses/${c}/gradebook?limit=100`, { headers: h })).json();
    return g.rows.map((r: { student: { student_code: string }; result: { hp?: string; letter?: string } | null }) => [r.student.student_code, r.result?.hp ?? "", r.result?.letter ?? ""]);
  });
  const csv = readFileSync("/repo/seed/expected_final_grades.csv", "utf8").trim().split("\n").slice(1).map((l) => l.split(","));
  const exp = new Map(csv.map((c) => [c[1], { hp: c[12], letter: c[13] }]));
  let ok = 0; const bad: string[] = [];
  for (const [code, hp, letter] of rows as string[][]) {
    const e = exp.get(code);
    if (!e) continue;
    if (Number(hp) === Number(e.hp) && letter === e.letter) ok++; else bad.push(`${code}: API ${hp}/${letter} ≠ CSV ${e.hp}/${e.letter}`);
  }
  console.log(`INFO khớp ${ok}/${csv.length}; lệch: ${bad.slice(0, 5).join(" ; ")}`);
  expect(bad).toEqual([]);
  expect(ok).toBe(30);
  await ctx.close();
});
