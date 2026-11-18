/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkR (qc2, lượt 6): US-UX91-06 lớp mặc định / ep:ui:course trên ep-r (giảng viên `teacher` có > 60 lớp: kỳ vọng tính từ /me/courses + /me/today thật, không cứng 761987/8).
import { test } from "@playwright/test";
import { api, courseId, expect, go, login, mainText, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
test.describe.configure({ mode: "serial" });
const KEY = "ep:ui:course";
const topbar = (p: any) => p.locator("[data-part=topbar]");

async function mine(page: any) {
  const r = await api(page, "/me/courses?limit=100");
  return (r.body.items as any[]).map((i) => i.course) as Array<{ id: string; class_code: string; status: string }>;
}
const todayOf = (ids: string[]) => ({ count: ids.length, actions: ids.map((id, i) => ({ id: `a${i}`, kind: "JOIN_REQUESTS", title: "Việc", reason: "r", urgency: "normal", href: "/class/members", course: { id, class_code: "x" } })) });

for (const [tc, name, pick] of [
  ["TC-UX06-3", "today có việc ở cả A và B ⇒ lớp mặc định = A (đầu danh sách có dữ liệu)", (a: string, b: string) => ({ ids: [a, b], want: a })],
  ["TC-UX06-4", "today chỉ có việc thuộc B ⇒ lớp mặc định = B", (_a: string, b: string) => ({ ids: [b], want: b })],
  ["TC-UX06-5", "today rỗng cả hai ⇒ lớp đầu danh sách (A)", (a: string) => ({ ids: [], want: a })],
] as const) {
  test(`${tc} ${name}`, async ({ browser }, info) => {
    D(info); test.setTimeout(240_000);
    const probe = await open(browser, "teacher");
    const act = (await mine(probe.page)).filter((c) => c.status === "ACTIVE");
    await probe.ctx.close();
    expect(act.length, "giảng viên có ≥ 2 lớp ACTIVE").toBeGreaterThanOrEqual(2);
    const [A, B] = act;
    const { ids, want } = pick(A.id, B.id);
    const wantCode = want === A.id ? A.class_code : B.class_code;
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
    const page = await ctx.newPage();
    await page.route("**/api/v1/me/today**", (r: any) => r.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(todayOf(ids)) }));
    await login(page, "teacher");
    await page.waitForTimeout(2500);
    console.log(`INFO ${tc} A=${A.class_code} B=${B.class_code} mong đợi ${wantCode}; thanh trên:`, JSON.stringify((await topbar(page).innerText()).slice(0, 60)));
    await expect(topbar(page), `${tc}: lớp mặc định = ${wantCode}`).toContainText(wantCode);
    await ctx.close();
  });
}

test("TC-UX06-6 ep:ui:course = UUID lạ ⇒ bỏ giá trị, chọn theo AC2, không pageerror, không trang trống", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const probe = await open(browser, "teacher");
  const all = await mine(probe.page);
  await probe.ctx.close();
  const act = all.filter((c) => c.status === "ACTIVE");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await ctx.newPage();
  const errs: string[] = []; page.on("pageerror", (e) => errs.push(e.message));
  await page.addInitScript((k) => localStorage.setItem(k, "00000000-0000-7000-8000-00000000dead"), KEY);
  let appToday: any = null; // QC SmkS3: /me/today đổi theo thời gian (401 việc, 50 dòng đầu) ⇒ lấy đúng phản hồi mà ứng dụng nhận
  page.on("response", async (r: any) => { if (/\/me\/today/.test(r.url()) && r.ok()) appToday = await r.json().catch(() => null); });
  await login(page, "teacher");
  await page.waitForTimeout(2500);
  const withData = new Set<string>(((appToday?.actions ?? []) as any[]).map((a: any) => a.course?.id).filter(Boolean));
  const want = (act.find((c) => withData.has(c.id)) ?? act[0]).class_code;
  console.log("INFO UX06-6 mong đợi", want, "| thanh trên:", JSON.stringify((await topbar(page).innerText()).slice(0, 50)));
  await expect(topbar(page)).toContainText(want);
  expect((await mainText(page)).trim().length, "không trang trống").toBeGreaterThan(20);
  expect(errs, "không pageerror").toEqual([]);
  expect(await page.evaluate((k) => localStorage.getItem(k), KEY)).not.toBe("00000000-0000-7000-8000-00000000dead");
  await ctx.close();
});

test("TC-UX06-7 TC-UX06-PQ lớp đã lưu trữ ⇒ bỏ giá trị; SV: lớp không thuộc ⇒ 403 và bỏ giá trị", async ({ browser }, info) => {
  D(info); test.setTimeout(300_000);
  const t0 = await open(browser, "teacher");
  const me = await api(t0.page, "/me/profile");
  const teacherId = (me.body.id ?? me.body.user?.id) as string;
  await t0.ctx.close();
  const adm = await open(browser, "admin");
  const post = (p: string, b: unknown) => adm.page.evaluate(async ([path, body]) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${path}`, { method: "POST", headers: { authorization: `Bearer ${a.access_token}`, "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify(body) });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [p, b] as const);
  const code = `QC${Date.now().toString().slice(-6)}`;
  const mk = await post("/admin/courses", { subject_code: "INT1006", class_code: code, name: "Lớp thử QC 9.1", semester: "2026-2027-HK1", teacher_id: teacherId });
  console.log("INFO tạo lớp:", mk.status, code);
  expect(mk.status).toBeLessThan(300);
  const id = (mk.body.id ?? mk.body.course?.id) as string;
  // SV không thuộc lớp
  const sv = await open(browser, "sv.gioi");
  await sv.page.evaluate(([k, v]) => localStorage.setItem(k, v), [KEY, id] as const);
  await sv.page.reload(); await sv.page.waitForTimeout(3000);
  const st = (await api(sv.page, `/courses/${id}/today`)).status;
  const val = await sv.page.evaluate((k) => localStorage.getItem(k), KEY);
  console.log("INFO PQ SV: /courses/‹id›/today =", st, "| ep:ui:course =", val);
  expect(st, "SV không đọc được dữ liệu lớp lạ").toBe(403);
  expect(val, "SV: bỏ giá trị").not.toBe(id);
  const svMine = await mine(sv.page);
  await expect(topbar(sv.page)).toContainText(svMine[0].class_code.slice(0, 3));
  await sv.ctx.close();
  // GV: lớp ARCHIVED
  const ar = await post(`/admin/courses/${id}/archive`, {});
  console.log("INFO lưu trữ:", ar.status);
  expect(ar.status).toBeLessThan(300);
  const t = await open(browser, "teacher");
  await t.page.evaluate(([k, v]) => localStorage.setItem(k, v), [KEY, id] as const);
  await t.page.reload(); await t.page.waitForTimeout(3500);
  const after = await t.page.evaluate((k) => localStorage.getItem(k), KEY);
  const mineT = await mine(t.page);
  console.log("INFO UX06-7 sau lưu trữ: ep:ui:course =", after, "| lớp", code, "trạng thái", mineT.find((c) => c.id === id)?.status);
  expect(after, "lớp ARCHIVED bị bỏ khỏi giá trị nhớ").not.toBe(id);
  expect(await topbar(t.page).innerText()).not.toContain(code);
  expect((await mainText(t.page)).trim().length).toBeGreaterThan(20);
  await t.ctx.close(); await adm.ctx.close();
  void courseId; void go;
});
