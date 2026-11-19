import { execFileSync, spawnSync } from "node:child_process";
import { resolve } from "node:path";
import { expect, test, type Page, type Route } from "@playwright/test";
import { API, PWD, apiLogin, axeBad, call, clearRL, installRec, layout, linkToken, mailTo, note, psql, techWords } from "./r4ui-lib";

// QC2-I r4ui: US-P2-04 (quên / đặt lại mật khẩu, thiết bị), US-P2-05 (giới hạn đăng nhập).
installRec("r4ui-acct");
const only = (info: { project: { name: string } }, p: "desktop" | "mobile") => test.skip(info.project.name !== p, `chỉ ${p}`);
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
const FORGOT_MSG = "Nếu email này có tài khoản, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu.";

/** Tài khoản mới (đăng ký + xác minh qua Mailpit). */
async function freshUser(tag: string) {
  clearRL();
  const t0 = Date.now();
  const email = `qc2iu.${tag}${t0}@edupilot.local`;
  const r = await call("POST", "/auth/register", null, { email, password: PWD, full_name: `QC2IU ${tag}` });
  expect(r.status).toBe(202);
  const tok = linkToken(await mailTo(email, /Xác minh/i, t0), /verify-email/);
  expect((await call("POST", "/auth/verify-email", null, { token: tok })).status).toBe(200);
  return email;
}
async function resetToken(email: string) {
  clearRL();
  const t0 = Date.now();
  const r = await call("POST", "/auth/forgot-password", null, { email });
  expect(r.status).toBe(202);
  return linkToken(await mailTo(email, /mật khẩu|Đặt lại/i, t0), /reset-password/);
}

test("[TC-P204-27] forgot / reset: một ô, một nút, câu chung, gửi lại khoá; reset hợp lệ / không hợp lệ / sau thành công, token rời URL", async ({ page }, info) => {
  only(info, "desktop");
  const email = await freshUser("rst");
  clearRL();
  await page.goto("/forgot-password");
  await expect(page.getByRole("heading", { name: "Quên mật khẩu" })).toBeVisible();
  await expect(page.getByLabel("Email")).toHaveCount(1);
  const send = page.getByRole("button", { name: "Gửi hướng dẫn", exact: true });
  await expect(send).toHaveCount(1);
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  await page.getByLabel("Email").fill(email);
  await send.click();
  await expect(page.getByText(FORGOT_MSG)).toBeVisible({ timeout: 30_000 });
  const again = page.getByRole("button", { name: /^Gửi lại hướng dẫn/ });
  await expect(again).toBeDisabled();
  await expect(again).toContainText(/\(\d+ giây\)/);
  // email không có: cùng câu
  clearRL();
  await page.goto("/forgot-password");
  await page.getByLabel("Email").fill("khong.co.qc2iu@edupilot.local");
  await page.getByRole("button", { name: "Gửi hướng dẫn", exact: true }).click();
  await expect(page.getByText(FORGOT_MSG)).toBeVisible({ timeout: 30_000 });
  // reset hợp lệ
  const token = await resetToken(email);
  await page.goto(`/reset-password?token=${token}`);
  await expect(page.getByRole("heading", { name: "Đặt mật khẩu mới" })).toBeVisible({ timeout: 30_000 });
  expect(await page.evaluate(() => location.search)).toBe("");
  expect(page.url()).not.toContain(token);
  const a = page.getByLabel("Mật khẩu mới", { exact: true });
  const b = page.getByLabel("Nhập lại mật khẩu", { exact: true });
  await expect(a).toHaveAttribute("autocomplete", "new-password");
  await expect(a).toHaveAttribute("type", "password");
  await expect(b).toHaveAttribute("type", "password");
  await expect(b).toHaveAttribute("autocomplete", "new-password");
  await expect(page.getByText("Ít nhất 10 ký tự, không phải mật khẩu phổ biến.")).toBeVisible();
  const NEW = "Qc2iu!Moi-2026y";
  await a.fill(NEW);
  await b.fill(NEW);
  clearRL();
  await page.getByRole("button", { name: "Đổi mật khẩu", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Mật khẩu đã được đổi. Hãy đăng nhập lại." })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole("link", { name: "Đăng nhập" })).toHaveAttribute("href", "/login");
  expect(page.url()).not.toContain(token);
  // sau thành công: dùng lại = không dùng được
  await page.goto(`/reset-password?token=${token}`);
  await expect(page.getByText("Liên kết đã hết hạn hoặc đã được dùng.")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole("link", { name: "Yêu cầu liên kết mới" })).toHaveAttribute("href", "/forgot-password");
  // token rác / thiếu token
  await page.goto(`/reset-password?token=${"Z".repeat(43)}`);
  await expect(page.getByText("Liên kết đã hết hạn hoặc đã được dùng.")).toBeVisible({ timeout: 30_000 });
  await page.goto("/reset-password");
  await expect(page.getByRole("link", { name: "Yêu cầu liên kết mới" })).toBeVisible();
  // mật khẩu mới đăng nhập được, mật khẩu cũ không
  clearRL();
  expect((await call("POST", "/auth/login", null, { email, password: NEW })).status).toBe(200);
  clearRL();
  expect((await call("POST", "/auth/login", null, { email, password: PWD })).status).toBe(401);
  note("forgot 1 ô + 1 nút; câu chung cho email có/không; reset hợp lệ/không hợp lệ/đã dùng đúng chữ; token rời URL; mật khẩu mới dùng được");
});

test("[TC-P204-28] /reset-password: Referrer-Policy no-referrer; token không rò qua Referer và log Caddy", async ({ page, request }, info) => {
  only(info, "desktop");
  const email = await freshUser("rf");
  const token = await resetToken(email);
  const since = new Date(Date.now() - 2000).toISOString();
  const seen: string[] = [];
  page.on("request", (q) => seen.push(`${q.url()} ref=${q.headers()["referer"] ?? ""}`));
  await page.goto(`/reset-password?token=${token}`);
  await expect(page.getByRole("heading", { name: "Đặt mật khẩu mới" })).toBeVisible({ timeout: 30_000 });
  await page.waitForTimeout(1500);
  const hd = (await request.get("/reset-password")).headers();
  const leak = seen.filter((s) => s.split(" ref=")[1]?.includes(token));
  const external = seen.filter((s) => !s.startsWith("https://localhost:1143"));
  // docker logs ghi log Caddy ra stderr: gom cả stdout và stderr
  const dl = spawnSync("docker", ["logs", "ep-r-caddy-1", "--since", since], { maxBuffer: 1 << 28 });
  const logs = dl.stdout.toString() + dl.stderr.toString();
  const cnt = logs.split(token).length - 1;
  const redacted = (logs.match(/reset-password\?token=REDACTED/g) ?? []).length;
  note(`Referrer-Policy=${hd["referrer-policy"]}; Cache-Control=${hd["cache-control"]}; request mang token ở Referer: ${leak.length}; request ra ngoài: ${external.length}; log Caddy đếm token=${cnt}, dòng REDACTED=${redacted}`);
  expect(hd["referrer-policy"]).toBe("no-referrer");
  expect(leak).toEqual([]);
  expect(external).toEqual([]);
  expect(cnt).toBe(0);
  expect(redacted).toBeGreaterThanOrEqual(1);
});

/** Hộp "Thời gian học trên EduPilot" chặn SV vừa vào lớp (nhất quán với US-P5-03): xác nhận để làm tiếp. */
async function ackNotice(page: Page) {
  const ok = page.getByRole("button", { name: "Đã hiểu" });
  if (await ok.isVisible({ timeout: 4000 }).catch(() => false)) await ok.click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
}

/** Tạo n phiên (đăng nhập API với UA khác nhau) cho tài khoản mới; trả email. */
async function userWithSessions(tag: string) {
  const email = await freshUser(tag);
  // /settings của SV chỉ mở khi đã có lớp (nav.ts needsCourse, SRS 4.3.2): ghi danh vào một lớp ACTIVE để có đủ khung app
  psql(`insert into enrollments(course_id,user_id,role_in_course,status,joined_via) select c.id,u.id,'STUDENT','ACTIVE','CODE' from users u, (select id from courses where status='ACTIVE' order by created_at limit 1) c where u.email='${email}' on conflict do nothing`);
  for (const ua of ["Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0"]) {
    clearRL();
    const r = await call("POST", "/auth/login", null, { email, password: PWD }, { "user-agent": ua });
    expect(r.status).toBe(200);
  }
  return email;
}

test("[TC-P204-30] /settings: Đổi mật khẩu primary; Thiết bị này; đăng xuất 1 thiết bị ngay + dòng tĩnh; mọi thiết bị khác có số; rỗng", async ({ page }, info) => {
  only(info, "desktop");
  const email = await userWithSessions("dv");
  await apiLogin(page, email);
  await page.goto("/settings");
  await expect(page.getByRole("heading", { name: "Tài khoản và bảo mật" })).toBeVisible({ timeout: 30_000 });
  await ackNotice(page);
  await expect(page.getByRole("heading", { name: "Mật khẩu", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Thiết bị đang đăng nhập" })).toBeVisible();
  const change = page.getByRole("button", { name: "Đổi mật khẩu", exact: true });
  await expect(change).toHaveAttribute("data-variant", "primary");
  const rows = page.locator("ul li").filter({ hasText: /Lần cuối/ });
  await expect(rows).toHaveCount(3);
  await expect(rows.filter({ hasText: "Thiết bị này" })).toHaveCount(1);
  const others = page.getByRole("button", { name: "Đăng xuất mọi thiết bị khác" });
  await expect(others).not.toHaveAttribute("data-variant", "primary");
  // mở xác nhận: nêu số 2, huỷ
  await others.click();
  await expect(page.getByRole("dialog")).toContainText("Đăng xuất 2 thiết bị khác. Họ sẽ phải đăng nhập lại.");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  // đăng xuất một thiết bị: ngay, không hộp thoại, dòng tĩnh
  const other = rows.filter({ hasNotText: "Thiết bị này" }).first();
  await other.getByRole("button", { name: /Thao tác cho/ }).click();
  await page.getByRole("menuitem", { name: "Đăng xuất thiết bị này" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByText("Đã đăng xuất thiết bị này")).toBeVisible();
  await expect(rows).toHaveCount(2);
  await others.click();
  await expect(page.getByRole("dialog")).toContainText("Đăng xuất 1 thiết bị khác. Họ sẽ phải đăng nhập lại.");
  await page.getByRole("dialog").getByRole("button", { name: "Đăng xuất", exact: true }).click();
  await expect(page.getByText("Không có thiết bị nào khác.")).toBeVisible({ timeout: 20_000 });
  await expect(others).toHaveCount(0);
  const active = psql(`select count(*) from auth_sessions s join users u on u.id=s.user_id where u.email='${email}' and s.revoked_at is null`);
  note(`phiên còn hiệu lực trong DB sau khi đăng xuất hết: ${active}`);
  expect(active).toBe("1");
});

test("[TC-P204-31] /settings: chỉ bàn phím, 375 px, ui-antipatterns.sh, axe, không từ kỹ thuật", async ({ page }, info) => {
  const email = await userWithSessions("dk");
  await apiLogin(page, email);
  await page.goto("/settings");
  await expect(page.getByRole("heading", { name: "Thiết bị đang đăng nhập" })).toBeVisible({ timeout: 30_000 });
  await ackNotice(page);
  await expect(page.locator("ul li").filter({ hasText: /Lần cuối/ })).toHaveCount(3);
  if (info.project.name === "desktop") {
    // bàn phím: Tab tới "Đăng xuất mọi thiết bị khác" → Enter mở hộp xác nhận → Esc đóng
    let reached = false;
    for (let i = 0; i < 150 && !reached; i++) {
      await page.keyboard.press("Tab");
      reached = await page.evaluate(() => /Đăng xuất mọi thiết bị khác/.test(document.activeElement?.textContent ?? ""));
    }
    expect(reached).toBe(true);
    await page.keyboard.press("Enter");
    await expect(page.getByRole("dialog")).toContainText("Đăng xuất 2 thiết bị khác");
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    // menu của một hàng bằng phím
    let menu = false;
    for (let i = 0; i < 150 && !menu; i++) {
      await page.keyboard.press("Shift+Tab");
      menu = await page.evaluate(() => /Thao tác cho/.test(document.activeElement?.getAttribute("aria-label") ?? ""));
    }
    expect(menu).toBe(true);
    await page.keyboard.press("Enter");
    await expect(page.getByRole("menuitem", { name: "Đăng xuất thiết bị này" })).toBeVisible();
    await page.keyboard.press("Escape");
    const out = execFileSync("bash", ["scripts/ui-antipatterns.sh"], { cwd: resolve(__dirname, "../../.."), maxBuffer: 1 << 24 }).toString();
    note(`ui-antipatterns.sh exit 0; dòng cuối: ${out.trim().split("\n").slice(-1)[0].slice(0, 120)}`);
  } else {
    const l = await layout(page);
    note(`375: ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length}`);
    expect(l.ox).toBeLessThanOrEqual(0);
    expect(l.cut).toEqual([]);
    expect(l.touch).toEqual([]);
  }
  const ax = await axeBad(page);
  const t = await page.locator("main").innerText();
  note(`axe=${JSON.stringify(ax)}; "session"/"IP" trần: ${/\bsession\b/i.test(t)}/${/\bIP\b/.test(t)}`);
  expect(ax).toEqual([]);
  expect(await techWords(page, ["token", "session", "refresh", "jwt", "bcrypt"])).toEqual([]);
  expect(/\bIP\b/.test(t)).toBe(false);
});

test("[TC-P205-33] /login bị giới hạn: 429 LOGIN_THROTTLED retry_after 3 (page.route) rồi 429 thật sau các lần sai", async ({ page }, info) => {
  only(info, "desktop");
  clearRL();
  // (1) mô phỏng
  let calls = 0;
  await page.route("**/api/v1/auth/login", (r) => {
    calls++;
    return calls === 1 ? r.fulfill({ status: 429, contentType: "application/json", headers: { "Retry-After": "3" }, body: JSON.stringify({ code: "LOGIN_THROTTLED", message: "m", trace_id: "b".repeat(32), retry_after: 3 }) }) : r.fallback();
  });
  await page.goto("/login");
  await page.getByLabel("Email").fill("Khong.Biet@Sv.Example");
  await page.getByLabel("Mật khẩu", { exact: true }).fill("mat-khau-dang-go");
  const submit = page.getByRole("button", { name: "Đăng nhập", exact: true });
  const t0 = Date.now();
  await submit.click();
  const line = page.getByRole("status").filter({ hasText: /^Bạn đã thử quá nhiều lần\. Thử lại sau 00:0[123]\.$/ });
  await expect(line).toBeVisible();
  const texts = new Set<string>();
  const rec = async () => { for (let i = 0; i < 30 && (await line.count()); i++) { texts.add((await line.innerText().catch(() => "")).trim()); await page.waitForTimeout(250); } };
  await expect(submit).toBeDisabled();
  await expect(submit).toHaveAttribute("aria-describedby", "login-wait");
  await expect(page.getByLabel("Email")).toHaveValue("Khong.Biet@Sv.Example");
  await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveValue("mat-khau-dang-go");
  await expect(page.getByRole("link", { name: "Quên mật khẩu?" })).toBeVisible();
  expect(await page.locator("body").innerText()).not.toMatch(/LOGIN_THROTTLED|RATE_LIMITED/);
  await rec();
  await expect(submit).toBeEnabled({ timeout: 4_000 });
  const dt = Date.now() - t0;
  note(`(1) khoá ${dt} ms (≈3,1 s); các chữ đếm ngược: ${[...texts].join(" | ")}`);
  expect(dt).toBeLessThan(5500);
  expect(texts.size).toBeGreaterThanOrEqual(2); // đếm ngược đổi mỗi giây
  await expect(line).toHaveCount(0);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  // (2) thật: sai liên tiếp tới khi máy chủ trả 429 (bước chờ tăng dần từ lần sai thứ 5), chỉ ảnh hưởng email này
  clearRL();
  await page.goto("/login");
  const email = "qc2iu.sv4@edupilot.local";
  let hit429: { retry: number | undefined; body: string } | null = null;
  page.on("response", async (r) => {
    if (r.url().endsWith("/auth/login") && r.status() === 429) hit429 = { retry: Number(r.headers()["retry-after"]), body: JSON.stringify(await r.json().catch(() => ({}))).slice(0, 120) };
  });
  for (let i = 0; i < 9 && !hit429; i++) {
    await page.getByLabel("Email").fill(email);
    await page.getByLabel("Mật khẩu", { exact: true }).fill(`sai-mat-khau-${i}-xyz`);
    if (await submit.isEnabled()) await submit.click();
    await page.waitForTimeout(400);
  }
  await expect.poll(() => hit429, { timeout: 15_000 }).not.toBeNull();
  const real = page.getByRole("status").filter({ hasText: /^Bạn đã thử quá nhiều lần\. Thử lại sau \d\d:\d\d\.$/ });
  await expect(real).toBeVisible();
  const realText = await real.innerText();
  await expect(submit).toBeDisabled();
  expect(await page.locator("body").innerText()).not.toMatch(/LOGIN_THROTTLED|RATE_LIMITED/);
  note(`(2) 429 thật: ${JSON.stringify(hit429)}; dòng UI "${realText.trim()}"; nút đăng nhập disabled`);
  clearRL(); // gỡ khoá email thử
});
void (null as unknown as Page);
void API;
