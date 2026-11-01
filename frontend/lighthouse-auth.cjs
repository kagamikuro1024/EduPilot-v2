// Đặt cookie `lh_role` theo vai của từng URL trước khi Lighthouse đo (US-PU-05 AC5). Cookie này CHỈ do gateway giả `lighthouse-api.mjs`
// đọc để trả phiên JWT giả cho `POST /auth/refresh`; ứng dụng không còn đọc cookie phiên mô phỏng nào (US-P2-12 AC10).
const ORIGIN = "http://localhost:3310";
const ROLE = { "/": "student", "/chat": "student", "/threads": "student", "/me": "student", "/inbox": "teacher", "/gradebook": "teacher", "/settings/llm": "admin" };
// Route động: tra theo tiền tố (US-P7-03: `/assignments/<id>` là màn của sinh viên).
const PREFIX = [["/assignments/", "student"]];
const roleFor = (path) => ROLE[path] ?? PREFIX.find(([p]) => path.startsWith(p))?.[1];

module.exports = async (browser, context) => {
  const path = new URL(context.url).pathname;
  const role = roleFor(path);
  const page = await browser.newPage();
  await page.deleteCookie(...(await page.cookies(ORIGIN)));
  if (role) await page.setCookie({ name: "lh_role", value: role, url: ORIGIN });
  await page.close();
};
