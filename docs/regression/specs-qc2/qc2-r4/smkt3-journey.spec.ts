import { expect, test, type Page } from "@playwright/test";
import { S, apiLogin, call, clearRL, freshStudent, installRec, joinInfo, note, psql, tokenOf } from "./r4ui-lib";

// SmkT3: TC-GATEP2-24 (tự kiểm 5) — luồng F1+F2 qua UI: Admin mở lớp → gán GV → GV thấy chuông + mã → SV D 375 px /join/MÃ → xem trước → tham gia → GV thấy thành viên mới. ep-r (DB không trống; lớp và người mới tạo riêng cho ca).
installRec("smkt3-journey");
const SHOT = "/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-4/shots/gatep2-24-";
const stamp = () => Date.now().toString(36).toUpperCase();
const picker = (p: Page) => p.getByRole("button", { name: /^Chọn lớp, đang xem/ });

test("[TC-P212-24 GATEP2-24] Admin mở lớp → GV thấy chuông + mã → SV 375 /join/MÃ → xem trước → tham gia → GV thấy thành viên mới", async ({ browser }, info) => {
  if (info.project.name !== "desktop") test.skip();
  test.setTimeout(300_000);
  const steps: string[] = [];
  // 1. Admin mở lớp trên UI, gán GV2 (GV đã có tài khoản, chưa mở trang)
  const adm = await browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN", viewport: { width: 1440, height: 900 } });
  const ap = await adm.newPage();
  await apiLogin(ap, "qc2iu.adm@edupilot.local");
  await ap.goto("/admin/courses");
  await expect(ap.getByRole("heading", { name: "Lớp học", level: 1 })).toBeVisible({ timeout: 30_000 });
  await ap.getByRole("button", { name: "Mở lớp" }).first().click();
  const code = `QC2IUJ${stamp()}`.slice(0, 16);
  await ap.getByLabel("Mã lớp", { exact: true }).fill(code);
  await ap.getByLabel("Giảng viên").selectOption({ label: S.gv2.name });
  await ap.getByRole("button", { name: "Mở lớp" }).click();
  await expect(ap.getByRole("status").filter({ hasText: `Đã mở lớp. Đã gửi thông báo phân công cho ${S.gv2.name}.` })).toBeVisible({ timeout: 30_000 });
  await ap.screenshot({ path: SHOT + "1-admin-mo-lop.png" });
  const cid = psql(`select id from courses where class_code='${code}'`);
  steps.push(`1 Admin mở lớp ${code} qua UI (courses=${psql(`select count(*) from courses where class_code='${code}'`)}), gán ${S.gv2.name}`);
  await adm.close();
  // 2. GV đăng nhập: chuông + mã
  const gv = await browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN", viewport: { width: 1440, height: 900 } });
  const gp = await gv.newPage();
  await apiLogin(gp, S.gv2.email);
  await gp.goto("/");
  await expect(gp.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  await expect(gp.locator("[data-part=bell-dot]")).toHaveCount(1, { timeout: 40_000 });
  await gp.getByRole("button", { name: /^Thông báo, \d+ chưa đọc/ }).click();
  const item = gp.locator("[data-part=notifications]").getByRole("link", { name: new RegExp(`Bạn được phân công lớp .*${code}`) });
  await expect(item).toBeVisible({ timeout: 20_000 });
  await gp.screenshot({ path: SHOT + "2-gv-chuong.png" });
  await item.click();
  await expect(picker(gp)).toContainText(code, { timeout: 15_000 });
  const info2 = await joinInfo(cid, S.gv2.email);
  const joinCode = String(info2.join_code);
  const body = psql(`select body from notifications where user_id='${S.gv2.id}' and course_id='${cid}' order by created_at desc limit 1`);
  steps.push(`2 GV thấy chuông + «Bạn được phân công lớp …${code}»; thông báo nêu mã tham gia: ${body.includes(joinCode)}; bộ chọn lớp = ${code}`);
  expect(body).toContain(joinCode);
  // 3. SV D trên 375 px: /join/MÃ → xem trước → tham gia
  const d = await freshStudent("jr");
  const svc = await browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN", viewport: { width: 375, height: 812 }, hasTouch: true });
  const sp = await svc.newPage();
  clearRL();
  await apiLogin(sp, d);
  await sp.goto(`/join/${joinCode}`);
  await expect(sp.locator("[data-part=join-preview]")).toContainText(code, { timeout: 30_000 });
  const ox = await sp.evaluate(() => document.documentElement.scrollWidth - innerWidth);
  await sp.screenshot({ path: SHOT + "3-sv-xem-truoc-375.png" });
  const join = sp.getByRole("button", { name: "Tham gia lớp" });
  await join.click();
  await expect(sp.getByRole("status").filter({ hasText: /Bạn đã vào lớp\.|Đã gửi yêu cầu/ })).toBeVisible({ timeout: 30_000 });
  await sp.screenshot({ path: SHOT + "4-sv-da-vao-375.png" });
  const st = psql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='${d}' and e.course_id='${cid}'`);
  steps.push(`3 SV ${d.split("@")[0]} 375 px /join/${joinCode}: xem trước đúng lớp ${code}, tràn ngang=${ox}, tham gia → enrollments ${st}`);
  // 4. GV thấy thành viên mới (tải lại, vào danh sách thành viên)
  await gp.goto(`/class/members?course=${cid}`);
  await expect(picker(gp)).toContainText(code, { timeout: 30_000 });
  const dName = psql(`select full_name from users where email='${d}'`);
  await expect(gp.locator("main")).toContainText(dName, { timeout: 20_000 }).catch(() => undefined);
  const mem = (await gp.locator("main").innerText()).replace(/\s+/g, " ");
  await gp.screenshot({ path: SHOT + "5-gv-thanh-vien.png" });
  steps.push(`4 GV /class/members lớp ${code}: thấy «${dName}» = ${mem.includes(dName)} (hoặc ở hàng chờ duyệt nếu lớp cần duyệt; enrollments=${st})`);
  note(steps.join(" ; "));
  expect(st).toBe("ACTIVE");
  expect(mem).toContain(dName);
  void call;
  void tokenOf;
  await gv.close();
  await svc.close();
});
