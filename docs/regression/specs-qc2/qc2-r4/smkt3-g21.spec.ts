import { expect, test } from "@playwright/test";
import { PWD, apiLogin, call, clearRL, freshStudent, installRec, linkToken, mailTo, note } from "./r4ui-lib";

// SmkT3: TC-GATEP2-21 (tự kiểm 2) / TC-P204-14 — đang đăng nhập ở máy B, quên mật khẩu và đặt lại ở máy A → máy B bị đăng xuất kèm câu giải thích. ep-r, hai context.
installRec("smkt3-g21");

test("[TC-P204-14 GATEP2-21] đặt lại mật khẩu ở máy A → máy B bị đăng xuất, thấy câu giải thích", async ({ browser }, info) => {
  if (info.project.name !== "desktop") test.skip();
  test.setTimeout(180_000);
  const email = await freshStudent("g21");
  const mk = () => browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN", viewport: { width: 1440, height: 900 } });
  const [cb, ca] = [await mk(), await mk()];
  const pb = await cb.newPage();
  await apiLogin(pb, email);
  await pb.goto("/threads");
  await expect(pb.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  const before = (await cb.cookies()).some((c) => c.name === "ep_rt");
  // máy A: cũng đang đăng nhập; quên mật khẩu → thư → đặt lại bằng giao diện
  const pa = await ca.newPage();
  await apiLogin(pa, email);
  clearRL();
  const t0 = Date.now();
  await pa.goto("/forgot-password");
  await pa.getByLabel("Email").fill(email);
  await pa.getByRole("button", { name: "Gửi hướng dẫn", exact: true }).click();
  await expect(pa.getByText("Nếu email này có tài khoản, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu.")).toBeVisible({ timeout: 30_000 });
  const token = linkToken(await mailTo(email, /mật khẩu|Đặt lại/i, t0), /reset-password/);
  await pa.goto(`/reset-password?token=${token}`);
  const NEW = "Qc2iu!Moi-G21z";
  await pa.getByLabel("Mật khẩu mới", { exact: true }).fill(NEW);
  await pa.getByLabel("Nhập lại mật khẩu", { exact: true }).fill(NEW);
  clearRL();
  const t1 = Date.now();
  await pa.getByRole("button", { name: "Đổi mật khẩu", exact: true }).click();
  await expect(pa.getByRole("heading", { name: "Mật khẩu đã được đổi. Hãy đăng nhập lại." })).toBeVisible({ timeout: 30_000 });
  // máy B: yêu cầu kế tiếp (tải lại) → bị đăng xuất
  await pb.reload();
  await expect(pb).toHaveURL(/\/login/, { timeout: 30_000 });
  const msg = "Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi.";
  const shown = await pb.getByRole("status").filter({ hasText: msg }).isVisible().catch(() => false);
  await pb.screenshot({ path: "/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-4/shots/gatep2-21-may-B.png" });
  // máy A: tải lại trang cũ → cũng phải đăng nhập lại; cookie ep_rt của hai máy vô hiệu
  await pa.goto("/threads");
  await expect(pa).toHaveURL(/\/login/, { timeout: 30_000 });
  const rt = async (c: typeof cb) => (await c.cookies()).find((k) => k.name === "ep_rt")?.value;
  const [tb, ta] = [await rt(cb), await rt(ca)];
  const ref = async (v?: string) => (v ? (await call("POST", "/auth/refresh", null, {}, { cookie: `ep_rt=${v}` })).status : "không có cookie");
  const refB = await ref(tb);
  const refA = await ref(ta);
  clearRL();
  const old = await call("POST", "/auth/login", null, { email, password: PWD });
  clearRL();
  const neo = await call("POST", "/auth/login", null, { email, password: NEW });
  note(`máy B có cookie ep_rt trước=${before}; sau ${((Date.now() - t1) / 1000).toFixed(1)} s từ lúc đổi: tải lại → ${new URL(pb.url()).pathname + new URL(pb.url()).search}; câu «${msg}» hiện=${shown}; máy A tải lại /threads → ${new URL(pa.url()).pathname}; refresh bằng cookie ep_rt cũ: máy B ${refB}, máy A ${refA}; mật khẩu cũ → ${old.status}, mới → ${neo.status}`);
  expect(shown).toBe(true);
  expect(old.status).toBe(401);
  expect(neo.status).toBe(200);
});
