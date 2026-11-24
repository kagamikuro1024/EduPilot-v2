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
  const { status, token } = await page.evaluate(async (email, password) => {
    const r = await fetch("/api/v1/auth/login", { method: "POST", credentials: "include", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password }) });
    return { status: r.status, token: r.ok ? (await r.json()).access_token : "" };
  }, process.env.LH_FIRST_VISIT === "1" && role === "student" ? "t1sv0002@edupilot.local" : USER[role], process.env.SEED_DEFAULT_PASSWORD);
  if (status !== 200) throw new Error(`đăng nhập ${role} → ${status}`);
  // P-11-5: đo ở trạng thái người dùng quay lại — bấm «Đã hiểu» của hộp thoại thời gian học (idempotent, không đổi dữ liệu T1 khác). LH_FIRST_VISIT=1 bỏ bước này
  // (lần đầu; chạy riêng trên tài khoản chưa từng ack, vd. t1sv0002, ghi báo cáo nhãn «lần đầu» và không chấm ngưỡng).
  if (role === "student" && process.env.LH_FIRST_VISIT !== "1") {
    const ack = await page.evaluate(async (t) => (await fetch("/api/v1/me/tracking-notice/ack", { method: "POST", credentials: "include", headers: { Authorization: `Bearer ${t}` } })).status, token);
    if (ack >= 300) throw new Error(`ack hộp thoại thời gian học → ${ack}`);
  }
  await page.close();
};
