import { test } from "@playwright/test";
import { apiLogin, call, clearRL, freshStudent, linkToken, mailTo, psql, PWD } from "./r4ui-lib";
// SmkT2: soi DOM thật của /reset-password sau khi đổi và /settings (không kiểm, chỉ in)
test("[DOM] reset + settings", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  const email = await freshStudent("dom");
  psql(`insert into enrollments(course_id,user_id,role_in_course,status,joined_via) select c.id,u.id,'STUDENT','ACTIVE','CODE' from users u, (select id from courses where status='ACTIVE' order by created_at limit 1) c where u.email='${email}' on conflict do nothing`);
  clearRL();
  const t0 = Date.now();
  await call("POST", "/auth/forgot-password", null, { email });
  const token = linkToken(await mailTo(email, /mật khẩu|Đặt lại/i, t0), /reset-password/);
  await page.goto(`/reset-password?token=${token}`);
  await page.getByLabel("Mật khẩu mới", { exact: true }).waitFor({ timeout: 20000 });
  await page.getByLabel("Mật khẩu mới", { exact: true }).fill("Qc2iu!Moi-2026y");
  await page.getByLabel("Nhập lại mật khẩu", { exact: true }).fill("Qc2iu!Moi-2026y");
  clearRL();
  await page.getByRole("button", { name: "Đổi mật khẩu", exact: true }).click();
  await page.waitForTimeout(4000);
  const dump = async (tag: string) => console.log(tag, JSON.stringify({ url: page.url().replace(token, "<T>"), h: await page.locator("h1,h2,h3").allInnerTexts(), btn: await page.getByRole("button").allInnerTexts(), links: await page.getByRole("link").allInnerTexts(), alerts: await page.getByRole("alert").allInnerTexts(), main: (await page.locator("main").innerText()).slice(0, 600) }));
  await dump("AFTER-RESET");
  clearRL();
  for (const ua of ["Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile/15E148 Safari/604.1", "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0"]) await call("POST", "/auth/login", null, { email, password: "Qc2iu!Moi-2026y" }, { "user-agent": ua });
  await apiLogin(page, email, "Qc2iu!Moi-2026y");
  await page.goto("/settings");
  await page.waitForTimeout(3500);
  await dump("SETTINGS");
  const li = await page.locator("ul li").allInnerTexts();
  console.log("LI", JSON.stringify(li.slice(0, 6)));
});
