// Đăng nhập thật vào stack đo (ep-*) trước mỗi URL: cookie refresh do gateway đặt, app tự đổi ra phiên. Vai theo route như frontend/lighthouse-auth.cjs.
// Biến: LH_ORIGIN (https://localhost:843), SEED_DEFAULT_PASSWORD.
const ORIGIN = process.env.LH_ORIGIN;
const USER = { student: "t1sv0001@edupilot.local", teacher: "t1gv1@edupilot.local", admin: "admin@edupilot.test" };
const ROLE = { "/": "student", "/chat": "student", "/threads": "student", "/me": "student", "/inbox": "teacher", "/gradebook": "teacher", "/settings/llm": "admin" };
module.exports = async (browser, context) => {
  const path = new URL(context.url).pathname;
  const role = ROLE[path] ?? (path.startsWith("/assignments/") ? "student" : "student");
  const page = await browser.newPage();
  await page.deleteCookie(...(await page.cookies(ORIGIN)));
  await page.goto(`${ORIGIN}/login`, { waitUntil: "domcontentloaded" });
  const status = await page.evaluate(async (email, password) => {
    const r = await fetch("/api/v1/auth/login", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password }) });
    return r.status;
  }, USER[role], process.env.SEED_DEFAULT_PASSWORD);
  if (status !== 200) throw new Error(`đăng nhập ${role} → ${status}`);
  await page.close();
};
