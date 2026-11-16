// Bản sao frontend/lighthouse-auth.cjs của QC2: origin 3330 (bản dựng cổng của checkout pq), thêm /exams/<id>/take theo vai student (cookie lh_role).
const ORIGIN = "http://localhost:3330";
const ROLE = { "/": "student", "/chat": "student", "/threads": "student", "/me": "student", "/checkin": "student", "/inbox": "teacher", "/gradebook": "teacher", "/settings/llm": "admin" };
const PREFIX = [["/assignments/", "student"], ["/exams/", "student"]];
const roleFor = (path) => ROLE[path] ?? PREFIX.find(([p]) => path.startsWith(p))?.[1];
module.exports = async (browser, context) => {
  const path = new URL(context.url).pathname;
  const role = roleFor(path);
  const page = await browser.newPage();
  await page.deleteCookie(...(await page.cookies(ORIGIN)));
  if (role) await page.setCookie({ name: "lh_role", value: role, url: ORIGIN });
  await page.close();
};
