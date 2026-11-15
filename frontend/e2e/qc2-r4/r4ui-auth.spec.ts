import http from "node:http";
import { expect, test, type Page, type Route } from "@playwright/test";
import { API, PWD, S, apiLogin, axeBad, call, clearRL, em, installRec, layout, linkToken, mailTo, note, psql, techWords } from "./r4ui-lib";

// QC2-I r4ui: US-P2-02 (đăng nhập thật) + US-P2-03 (đăng ký / xác minh). Đăng nhập thật qua gateway ep-r bằng tài khoản qc2iu.*.
installRec("r4ui-auth");
const only = (info: { project: { name: string } }, p: "desktop" | "mobile") => test.skip(info.project.name !== p, `chỉ ${p}`);
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
const err = (code: string, extra: Record<string, unknown> = {}) => ({ code, message: "m", trace_id: "b".repeat(32), ...extra });
const topbar = (page: Page) => page.locator("[data-part=topbar]");
const sessions = (email: string) => psql(`select count(*) filter (where revoked_at is null), count(*) filter (where revoked_at is not null) from auth_sessions s join users u on u.id=s.user_id where u.email='${email}'`);

test("[TC-P202-23] CSRF: trang giả localhost:9999 gọi refresh + form POST, phiên nạn nhân còn nguyên", async ({ page, context }, info) => {
  only(info, "desktop");
  await apiLogin(page, em("sv1"));
  await page.goto("/");
  await expect(topbar(page)).toBeVisible();
  const c0 = (await context.cookies()).find((c) => c.name === "ep_rt");
  expect(c0?.sameSite).toBe("Lax");
  const srv = http.createServer((_q, res) => {
    res.setHeader("content-type", "text/html");
    res.end(`<html><body><form id=f method=POST action="${API}/auth/refresh" target=ifr></form><iframe name=ifr></iframe><script>
      fetch("${API}/auth/refresh",{method:"POST",credentials:"include"}).then(r=>window.fres="ok "+r.status).catch(e=>window.fres="err "+e);
      document.getElementById("f").submit();</script></body></html>`);
  });
  await new Promise<void>((r) => srv.listen(9999, r));
  try {
    const evil = await context.newPage();
    const st: number[] = [];
    evil.on("response", (r) => r.url().includes("/auth/refresh") && st.push(r.status()));
    await evil.goto("http://localhost:9999/");
    await expect.poll(() => st.length, { timeout: 15_000 }).toBeGreaterThanOrEqual(1);
    await evil.waitForTimeout(1500);
    note(`trạng thái refresh từ trang giả: ${st.join(",")}; cookie ep_rt SameSite=${c0?.sameSite}`);
    expect(st.every((s) => s >= 400)).toBe(true);
    await evil.close();
  } finally {
    srv.close();
  }
  const c1 = (await context.cookies()).find((c) => c.name === "ep_rt");
  expect(c1?.value).toBe(c0?.value); // cookie không xoay ⇒ máy chủ không làm mới theo yêu cầu giả
  await page.reload();
  await expect(topbar(page)).toBeVisible();
  const [active, revoked] = sessions(em("sv1")).split("|");
  note(`auth_sessions của nạn nhân: còn hiệu lực ${active}, bị thu hồi ${revoked}`);
  expect(Number(active)).toBeGreaterThanOrEqual(1);
});

test("[TC-P202-30] hai tab cùng context tải cùng lúc: refresh tuần tự, ≤ 2, phiên không bị thu hồi", async ({ page, context }, info) => {
  only(info, "desktop");
  const email = em("sv2");
  await apiLogin(page, email);
  const other = await context.newPage();
  const iv: Array<{ a: number; b: number }> = [];
  for (const p of [page, other]) {
    const open = new Map<unknown, number>();
    p.on("request", (r) => r.url().endsWith("/auth/refresh") && open.set(r, Date.now()));
    p.on("requestfinished", (r) => open.has(r) && iv.push({ a: open.get(r)!, b: Date.now() }));
  }
  clearRL();
  await Promise.all([page.goto("/"), other.goto("/")]);
  await expect(topbar(page)).toBeVisible();
  await expect(topbar(other)).toBeVisible();
  await page.waitForTimeout(1500);
  iv.sort((x, y) => x.a - y.a);
  note(`refresh: ${iv.length} lần; khoảng ${iv.map((i) => `${i.a % 100000}-${i.b % 100000}`).join(" | ")}`);
  expect(iv.length).toBeLessThanOrEqual(2);
  for (let i = 1; i < iv.length; i++) expect(iv[i].a).toBeGreaterThanOrEqual(iv[i - 1].b - 5); // không chồng thời gian (±5 ms đo)
  const [active, revoked] = sessions(email).split("|");
  note(`auth_sessions: còn ${active}, thu hồi ${revoked}`);
  expect(Number(active)).toBeGreaterThanOrEqual(1);
  const reuse = psql(`select count(*) from auth_sessions s join users u on u.id=s.user_id where u.email='${email}' and s.revoked_reason ilike '%reuse%'`);
  expect(reuse).toBe("0");
});

test("[TC-P202-31] 5 request song song nhận 401 TOKEN_EXPIRED: đúng 1 refresh, mỗi cái phát lại đúng 1 lần; POST giữ Idempotency-Key", async ({ page }, info) => {
  only(info, "desktop");
  await apiLogin(page, em("adm"));
  await page.goto("/admin/users");
  await expect(page.getByRole("heading", { name: "Người dùng", level: 1 })).toBeVisible();
  const refreshes: string[] = [];
  const seen = new Map<string, number>();
  const hits: Array<{ k: string; status: number; idem?: string }> = [];
  let armed = true;
  await page.route(/\/api\/v1\/(?!auth\/)/, async (route) => {
    const r = route.request();
    const k = `${r.method()} ${new URL(r.url()).pathname}${new URL(r.url()).search}`;
    if (!armed || r.url().includes("/events")) return route.fallback();
    const n = (seen.get(k) ?? 0) + 1;
    seen.set(k, n);
    if (n === 1) { hits.push({ k, status: 401, idem: r.headers()["idempotency-key"] }); return json(route, 401, err("TOKEN_EXPIRED")); }
    hits.push({ k, status: -1, idem: r.headers()["idempotency-key"] });
    return route.fallback();
  });
  page.on("request", (r) => r.url().endsWith("/auth/refresh") && refreshes.push(r.url()));
  const t0 = refreshes.length;
  // làm mới dữ liệu cùng lúc: tab quay lại tiêu điểm
  await page.evaluate(() => {
    document.dispatchEvent(new Event("visibilitychange"));
    window.dispatchEvent(new Event("focus"));
  });
  await page.waitForTimeout(3000);
  const expired = hits.filter((h) => h.status === 401);
  const replays = hits.filter((h) => h.status === -1);
  note(`401 TOKEN_EXPIRED ép: ${expired.length} request (${expired.map((e) => e.k.slice(0, 40)).join("; ")}); refresh ${refreshes.length - t0}; phát lại ${replays.length}`);
  expect(expired.length).toBeGreaterThanOrEqual(5);
  expect(refreshes.length - t0).toBe(1);
  for (const e of expired) expect([...seen.entries()].find(([k]) => k === e.k)?.[1]).toBe(2); // mỗi cái đúng 1 lần phát lại
  // POST: khung mời bị 401 một lần rồi thành công, cùng khoá
  armed = false;
  hits.length = 0;
  seen.clear();
  armed = true;
  await page.getByRole("button", { name: "Mời giảng viên" }).click();
  await page.getByLabel("Email", { exact: true }).fill(`qc2iu.inv${Date.now()}@edupilot.local`);
  await page.getByLabel("Họ và tên").fill("QC2IU Mời Thử");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi link mời, hạn 72 giờ." })).toBeVisible({ timeout: 20_000 });
  const posts = hits.filter((h) => h.k.startsWith("POST /api/v1/admin/users"));
  note(`POST /admin/users: ${posts.length} lần, khoá ${posts.map((p) => p.idem?.slice(0, 12)).join(",")}; refresh tổng ${refreshes.length - t0}`);
  expect(posts.length).toBe(2);
  expect(posts[0].idem).toBeTruthy();
  expect(posts[0].idem).toBe(posts[1].idem);
});

test("[TC-P202-32] refresh 401 → /login?next=; phát lại vẫn 401 → báo lỗi, không làm mới nữa; SESSION_REVOKED → không làm mới, /login kèm thông báo", async ({ page }, info) => {
  only(info, "desktop");
  const email = em("sv3");
  // (a) refresh hỏng giữa chừng
  await apiLogin(page, email);
  let n = 0;
  await page.route("**/api/v1/auth/refresh", (r) => { n++; return n === 1 ? r.fallback() : json(r, 401, err("TOKEN_INVALID")); });
  await page.route("**/api/v1/me/sessions", (r) => json(r, 401, err("TOKEN_EXPIRED")));
  await page.goto("/settings");
  await expect(page).toHaveURL(/\/login\?next=%2Fsettings/);
  note(`(a) refresh gọi ${n} lần (1 lúc tải + 1 khi 401) → ${new URL(page.url()).pathname}${new URL(page.url()).search}`);
  expect(n).toBe(2);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  // (b) phát lại vẫn 401 (refresh thật thành công): không vòng lặp
  await apiLogin(page, email);
  let rf = 0, gets = 0;
  page.on("request", (r) => r.url().endsWith("/auth/refresh") && rf++);
  await page.route("**/api/v1/me/sessions", (r) => { gets++; return json(r, 401, err("TOKEN_EXPIRED")); });
  await page.goto("/settings");
  await page.waitForTimeout(4000);
  note(`(b) refresh ${rf} lần; GET /me/sessions ${gets} lần; url ${new URL(page.url()).pathname}`);
  expect(rf).toBeLessThanOrEqual(3); // tải trang 1 + khi 401 1 (+ 1 dự phòng); không vòng lặp
  expect(gets).toBeLessThanOrEqual(2);
  await page.waitForTimeout(3000);
  const rf2 = rf;
  await page.waitForTimeout(3000);
  expect(rf).toBe(rf2);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  // (c) SESSION_REVOKED: không làm mới
  await apiLogin(page, email);
  let rc = 0;
  page.on("request", (r) => r.url().endsWith("/auth/refresh") && rc++);
  await page.route("**/api/v1/me/sessions", (r) => json(r, 401, err("SESSION_REVOKED", { details: { reason: "refresh_reuse" } })));
  await page.goto("/settings");
  await expect(page).toHaveURL(/\/login\?.*revoked=refresh_reuse/);
  await expect(page.getByText("Bạn đã bị đăng xuất. Hãy đăng nhập lại.")).toBeVisible();
  note(`(c) refresh ${rc} lần (chỉ lúc tải trang); url ${new URL(page.url()).search}`);
  expect(rc).toBe(1);
});

test("[TC-P202-36] JWT chỉ trong bộ nhớ: không biến toàn cục, không data-*, URL sạch, storage sạch", async ({ page }, info) => {
  only(info, "desktop");
  await apiLogin(page, em("gv1"));
  await page.goto("/");
  await expect(topbar(page)).toBeVisible();
  const res = await page.evaluate(() => {
    const JWT = /eyJ[\w-]{10,}\.[\w-]{10,}\.[\w-]*/;
    const out: string[] = [];
    if (JWT.test(document.documentElement.outerHTML)) out.push("outerHTML");
    for (const k of Object.getOwnPropertyNames(window)) {
      if (/token|jwt|access/i.test(k) && !/^(on|webkit)/.test(k)) out.push("window." + k);
      try { const v = (window as unknown as Record<string, unknown>)[k]; if (typeof v === "string" && JWT.test(v)) out.push("window." + k + "=jwt"); } catch { /* getter */ }
    }
    document.querySelectorAll("*").forEach((e) => { for (const a of Array.from(e.attributes)) if (a.name.startsWith("data-") && JWT.test(a.value)) out.push(e.tagName + "." + a.name); });
    for (const st of [localStorage, sessionStorage]) for (const [k, v] of Object.entries(st)) if (/token|jwt|refresh|access/i.test(k) || JWT.test(v)) out.push("storage." + k);
    if (JWT.test(location.href)) out.push("url");
    return { out, w: (window as unknown as Record<string, unknown>).__token === undefined };
  });
  note(`vi phạm: ${JSON.stringify(res.out)}`);
  expect(res.out).toEqual([]);
  expect(res.w).toBe(true);
});

test("[TC-P202-37] /login: thuộc tính ô, một nút primary, liên kết, logo 40/32 px", async ({ page }, info) => {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Đăng nhập EduPilot" })).toBeVisible();
  const email = page.getByLabel("Email");
  await expect(email).toHaveAttribute("autocomplete", "username");
  await expect(email).toHaveAttribute("inputmode", "email");
  const pw = page.getByLabel("Mật khẩu", { exact: true });
  await expect(pw).toHaveAttribute("autocomplete", "current-password");
  await page.getByRole("button", { name: "Hiện mật khẩu" }).click();
  await expect(pw).toHaveAttribute("type", "text");
  await page.getByRole("button", { name: "Ẩn mật khẩu" }).click();
  const submit = page.getByRole("button", { name: "Đăng nhập", exact: true });
  await expect(submit).toHaveCount(1);
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  await expect(submit).toHaveAttribute("data-variant", "primary");
  await expect(page.getByRole("link", { name: "Quên mật khẩu?" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Chưa có tài khoản? Đăng ký" })).toBeVisible();
  const h = Math.round(await page.locator("[data-part=brand]").evaluate((e) => e.getBoundingClientRect().height));
  note(`logo cao ${h}px @${info.project.name}`);
  expect(h).toBe(info.project.name === "mobile" ? 32 : 40);
});

test("[TC-P202-38] đăng nhập sai: role=alert chung, giữ email, xoá mật khẩu, bấm đúp = 1 request", async ({ page }, info) => {
  only(info, "desktop");
  clearRL();
  await page.goto("/login");
  const reqs: string[] = [];
  page.on("request", (r) => r.url().endsWith("/auth/login") && r.method() === "POST" && reqs.push(r.url()));
  await page.getByLabel("Email").fill(em("sv4"));
  await page.getByLabel("Mật khẩu", { exact: true }).fill("sai-mat-khau-12345");
  const submit = page.getByRole("button", { name: "Đăng nhập", exact: true });
  await submit.click();
  await submit.click({ force: true, noWaitAfter: true }).catch(() => undefined);
  const alert = page.getByRole("alert").filter({ hasText: "Email hoặc mật khẩu không đúng." });
  await expect(alert).toBeVisible();
  await page.waitForTimeout(500);
  const sameMsg = await alert.innerText();
  note(`số request /auth/login: ${reqs.length}; lỗi "${sameMsg}"`);
  expect(reqs.length).toBe(1);
  await expect(page.getByLabel("Email")).toHaveValue(em("sv4"));
  await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveValue("");
  // email không tồn tại: cùng câu
  clearRL();
  await page.getByLabel("Email").fill("khong.ton.tai.qc2iu@edupilot.local");
  await page.getByLabel("Mật khẩu", { exact: true }).fill("sai-mat-khau-12345");
  await submit.click();
  await expect(page.getByRole("alert").filter({ hasText: "Email hoặc mật khẩu không đúng." })).toBeVisible();
  clearRL();
});

test("[TC-P202-39] /login 375: AUDIT/TOUCH, axe 0 serious/critical, không từ kỹ thuật", async ({ page }, info) => {
  only(info, "mobile");
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Đăng nhập EduPilot" })).toBeVisible();
  const l = await layout(page);
  const ax = await axeBad(page);
  note(`ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length} axe=${JSON.stringify(ax)}`);
  expect(l.ox).toBeLessThanOrEqual(0);
  expect(l.cut).toEqual([]);
  expect(l.touch).toEqual([]);
  expect(ax).toEqual([]);
  expect(await techWords(page)).toEqual([]);
});

const REGISTER_MSG = "Nếu email này dùng được, chúng tôi đã gửi thư xác nhận. Kiểm tra hộp thư của bạn.";

test("[TC-P203-31] /register: nhãn, autocomplete, chú thích MSSV, 1 nút primary, lỗi từng ô, màn Kiểm tra email bất kể email + gửi lại khoá 60 s", async ({ page }, info) => {
  only(info, "desktop");
  clearRL();
  await page.goto("/register");
  await expect(page.getByRole("heading", { name: "Tạo tài khoản EduPilot" })).toBeVisible();
  await expect(page.getByLabel("Họ và tên")).toBeVisible();
  await expect(page.getByLabel("Email")).toHaveAttribute("autocomplete", "username");
  await expect(page.getByLabel("Mã số sinh viên (không bắt buộc)")).toBeVisible();
  await expect(page.getByText("Chỉ để giảng viên đối chiếu; không dùng để vào lớp.")).toBeVisible();
  await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveAttribute("autocomplete", "new-password");
  await expect(page.getByText("Ít nhất 10 ký tự, không phải mật khẩu phổ biến.")).toBeVisible();
  const submit = page.getByRole("button", { name: "Tạo tài khoản", exact: true });
  await expect(submit).toHaveCount(1);
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  await expect(submit).toHaveAttribute("data-variant", "primary");
  // thiếu / sai: lỗi dưới từng ô (aria-invalid), giữ chữ
  await page.getByLabel("Họ và tên").fill("QC2IU Lỗi");
  await page.getByLabel("Email").fill("sai");
  await page.getByLabel("Mật khẩu", { exact: true }).fill("ngan");
  await submit.click();
  await expect(page.getByLabel("Email")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Mật khẩu", { exact: true })).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Họ và tên")).toHaveValue("QC2IU Lỗi");
  // gửi thật: email mới
  const fresh = `qc2iu.reg${Date.now()}@edupilot.local`;
  await page.getByLabel("Email").fill(fresh);
  await page.getByLabel("Mật khẩu", { exact: true }).fill(PWD);
  await submit.click();
  await expect(page.getByRole("heading", { name: "Kiểm tra email của bạn" })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText("Nếu email này dùng được, chúng tôi đã gửi thư xác nhận.")).toBeVisible();
  const msgNew = await page.locator("main").innerText();
  const resend = page.getByRole("button", { name: /^Gửi lại thư/ });
  await expect(resend).toBeDisabled();
  const lbl = await resend.innerText();
  note(`nút gửi lại: "${lbl}"`);
  expect(lbl).toMatch(/\((\d+) giây\)/);
  expect(Number(lbl.match(/\((\d+) giây\)/)![1])).toBeGreaterThan(50);
  // email đã có tài khoản: cùng câu
  clearRL();
  await page.goto("/register");
  await page.getByLabel("Họ và tên").fill("QC2IU Trùng");
  await page.getByLabel("Email").fill(em("sv1"));
  await page.getByLabel("Mật khẩu", { exact: true }).fill(PWD);
  await submit.click();
  await expect(page.getByRole("heading", { name: "Kiểm tra email của bạn" })).toBeVisible({ timeout: 30_000 });
  expect(await page.locator("main").innerText()).toBe(msgNew);
  expect(await techWords(page, ["token", "session", "refresh", "jwt", "bcrypt"])).toEqual([]);
  expect(REGISTER_MSG.length).toBeGreaterThan(10);
});

test("[TC-P203-32] /verify-email?token=<hợp lệ>: tự xác minh, token rời URL, Referrer-Policy no-referrer, token không rò qua Referer", async ({ page, request }, info) => {
  only(info, "desktop");
  clearRL();
  const t0 = Date.now();
  const email = `qc2iu.ver${t0}@edupilot.local`;
  const r = await call("POST", "/auth/register", null, { email, password: PWD, full_name: "QC2IU Xác minh" });
  expect(r.status).toBe(202);
  const token = linkToken(await mailTo(email, /Xác minh/i, t0), /verify-email/);
  const seen: string[] = [];
  page.on("request", (q) => seen.push(`${q.url()} ref=${q.headers()["referer"] ?? ""}`));
  await page.goto(`/verify-email?token=${token}`);
  await expect(page.getByRole("heading", { name: "Email đã được xác minh." })).toBeVisible({ timeout: 30_000 });
  expect(await page.evaluate(() => location.search)).toBe("");
  expect(page.url()).not.toContain(token);
  const withTok = seen.filter((s) => s.includes("ref=") && s.split("ref=")[1].includes(token));
  const external = seen.filter((s) => !s.startsWith(BASE_ORIGIN) && !s.startsWith("data:") && !s.startsWith("blob:"));
  const hd = (await request.get("/verify-email")).headers();
  note(`Referrer-Policy=${hd["referrer-policy"]}; request mang token ở Referer: ${withTok.length}; request ra ngoài: ${external.length}`);
  expect(hd["referrer-policy"]).toBe("no-referrer");
  expect(withTok).toEqual([]);
  expect(external).toEqual([]);
  expect(psql(`select status from users where email='${email}'`)).toBe("ACTIVE");
});
const BASE_ORIGIN = process.env.QC_BASE ?? "https://localhost:1143";

test("[TC-P203-33] /register và /verify-email 375: AUDIT/TOUCH/axe sạch, không từ kỹ thuật", async ({ page }, info) => {
  only(info, "mobile");
  clearRL();
  const t0 = Date.now();
  const email = `qc2iu.v33${t0}@edupilot.local`;
  await call("POST", "/auth/register", null, { email, password: PWD, full_name: "QC2IU Di động" });
  const token = linkToken(await mailTo(email, /Xác minh/i, t0), /verify-email/);
  await page.goto("/register");
  await expect(page.getByRole("heading", { name: "Tạo tài khoản EduPilot" })).toBeVisible();
  const a = await layout(page);
  const ax = await axeBad(page);
  note(`/register ox=${a.ox} cut=${a.cut.length} touch=${a.touch.length} axe=${JSON.stringify(ax)}`);
  expect(a.ox).toBeLessThanOrEqual(0);
  expect(a.cut).toEqual([]);
  expect(a.touch).toEqual([]);
  expect(ax).toEqual([]);
  expect(await techWords(page, ["token", "session", "refresh", "jwt", "bcrypt"])).toEqual([]);
  await page.goto(`/verify-email?token=${token}`);
  await expect(page.getByRole("heading", { name: "Email đã được xác minh." })).toBeVisible({ timeout: 30_000 });
  const b = await layout(page);
  const bx = await axeBad(page);
  note(`/verify-email ox=${b.ox} cut=${b.cut.length} touch=${b.touch.length} axe=${JSON.stringify(bx)}`);
  expect(b.ox).toBeLessThanOrEqual(0);
  expect(b.cut).toEqual([]);
  expect(b.touch).toEqual([]);
  expect(bx).toEqual([]);
  expect(await techWords(page, ["token", "session", "refresh", "jwt", "bcrypt"])).toEqual([]);
  void S;
});
