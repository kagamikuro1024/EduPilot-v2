// Đặt phiên MÔ PHỎNG (cookie ep_demo_*) theo vai của từng URL trước khi Lighthouse đo (US-PU-05 AC5).
// `/settings/llm`: token dev chỉ ở bộ nhớ nên không thể nạp trước khi Lighthouse tải lại trang → đo màn "Dán token quản trị" của Admin.
const ORIGIN = "http://localhost:3310";
const ROLE = { "/": "student", "/chat": "student", "/threads": "student", "/inbox": "teacher", "/gradebook": "teacher", "/settings/llm": "admin" };

module.exports = async (browser, context) => {
  const path = new URL(context.url).pathname;
  const role = ROLE[path];
  const page = await browser.newPage();
  await page.deleteCookie(...(await page.cookies(ORIGIN)));
  if (role) {
    await page.setCookie({ name: "ep_demo_role", value: role, url: ORIGIN });
    if (role === "student") await page.setCookie({ name: "ep_demo_person", value: "sv-2", url: ORIGIN });
  }
  await page.close();
};
