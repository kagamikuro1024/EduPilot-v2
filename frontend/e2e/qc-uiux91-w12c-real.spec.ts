import { expect, test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC vòng kiểm lại đợt 1+2 (stack ep-qc): TC-UX01-4/5/11/16, UX06-6/7/PQ, UX07-9.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
async function login(page: Page, user: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
  await page.locator("[data-part=topbar]").waitFor({ timeout: 15_000 });
  await page.waitForTimeout(1500);
}
const api = (page: Page, path: string) =>
  page.evaluate(async (p) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${p}`, { headers: { authorization: `Bearer ${a.access_token}` } });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, path);
const settle = async (p: Page, u: string) => { await p.goto(u); await p.waitForTimeout(2500); };
const tip = (b: ReturnType<Page["locator"]>) => b.evaluate((e) => ({ title: e.getAttribute("title"), desc: (e.getAttribute("aria-describedby") ?? "").split(" ").map((i) => document.getElementById(i)?.textContent ?? "").join(" ").trim(), label: e.getAttribute("aria-label") }));
const words = (s: string) => s.trim().split(/\s+/).filter(Boolean).length;

test("TC-UX01-4/11 tooltip thay câu mô tả: có chữ, ≤ 12 từ, aria-describedby (hoặc title) trỏ tới", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const out: string[] = [];
  const ctxA = await browser.newContext({ ignoreHTTPSErrors: true });
  const a = await ctxA.newPage();
  await login(a, "admin");
  await settle(a, "/admin/courses");
  const openBtn = a.getByRole("button", { name: /Mở lớp mới|Mở lớp/ }).first();
  if (await openBtn.count()) { await openBtn.click(); await a.waitForTimeout(600); }
  const submit = a.getByRole("button", { name: /^Mở lớp$/ }).last();
  const t3 = (await submit.count()) ? await tip(submit) : null;
  out.push(`#3 Mở lớp: ${JSON.stringify(t3)}`);
  await settle(a, "/admin/users");
  const inv = a.getByRole("button", { name: /Mời|Gửi lời mời/ }).first();
  if (await inv.count()) { await inv.click().catch(() => undefined); await a.waitForTimeout(600); }
  const send = a.getByRole("button", { name: /Gửi lời mời|Mời$/ }).last();
  const t4 = (await send.count()) ? await tip(send) : null;
  out.push(`#4 gửi lời mời: ${JSON.stringify(t4)}`);
  await ctxA.close();
  const ctxT = await browser.newContext({ ignoreHTTPSErrors: true });
  const t = await ctxT.newPage();
  await login(t, "teacher");
  await settle(t, "/class/settings");
  const sw = t.getByText("Cần giảng viên duyệt").first();
  const lbl = (await sw.count()) ? await sw.evaluate((e) => ({ title: e.getAttribute("title") ?? e.closest("[title]")?.getAttribute("title") ?? null, desc: (e.closest("label,div")?.querySelector("[aria-describedby]")?.getAttribute("aria-describedby") ?? null) })) : null;
  const swEl = t.getByRole("switch", { name: /Cần giảng viên duyệt/ }).first();
  const t34 = (await swEl.count()) ? await tip(swEl) : null;
  out.push(`#34 nhãn: ${JSON.stringify(lbl)} switch: ${JSON.stringify(t34)}`);
  await ctxT.close();
  const ctxS = await browser.newContext({ ignoreHTTPSErrors: true });
  const s = await ctxS.newPage();
  await login(s, "sv.moi");
  await settle(s, "/join");
  const ph = await s.locator("input").first().getAttribute("placeholder");
  out.push(`#29 placeholder: ${ph}`);
  await ctxS.close();
  console.log("INFO " + out.join("\nINFO "));
  const txt = (x: { title: string | null; desc: string } | null) => (x ? x.desc || x.title || "" : "");
  expect.soft(txt(t3), "#3 tooltip nút Mở lớp").toMatch(/mã tham gia/);
  expect.soft(words(txt(t3)), "#3 ≤ 12 từ").toBeLessThanOrEqual(12);
  expect.soft(txt(t4), "#4 tooltip gửi lời mời").toMatch(/72 giờ/);
  expect.soft(t34?.desc ?? "", "#34 aria-describedby trỏ tới chữ").toMatch(/chờ bạn duyệt/);
  expect(ph, "#29 placeholder").toBe("AN7K2MQ");
});

test("TC-UX01-5 dòng «Giữ»: chữ dữ liệu vẫn hiện, bỏ mã lớp ở dòng phụ (#11, #47, #48)", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const p = await ctx.newPage();
  await login(p, "teacher");
  await settle(p, "/students");
  const t1 = await p.locator("main").innerText();
  const sub = t1.match(/\d+ sinh viên[^\n]*/)?.[0] ?? "";
  console.log("INFO #48 dòng phụ /students:", sub);
  expect(sub).toMatch(/^\d+ sinh viên$/);
  expect(t1).toMatch(/\b\d{8}\b/);
  await settle(p, "/attendance");
  const t2 = (await p.locator("main").innerText()).replace(/\n/g, " ");
  const line = t2.match(/buổi \d+ · [^·]*\d{2}:\d{2}[–-]\d{2}:\d{2} · P\.\d+/i)?.[0] ?? "";
  console.log("INFO #11 /attendance:", line || t2.slice(0, 160));
  expect(line, "dòng phụ buổi học").not.toBe("");
  expect(line).not.toMatch(/\b76198[78]\b/);
  await ctx.close();
});

test("TC-UX01-16 meta rỗng: CLS ≤ 0,05 khi chuyển khung máy chủ → màn thật (/students, /class/members)", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
  const p = await ctx.newPage();
  await login(p, "teacher");
  for (const u of ["/students", "/class/members"]) {
    await p.addInitScript(() => { const w = window as unknown as { __cls: number }; w.__cls = 0; new PerformanceObserver((l) => l.getEntries().forEach((e) => { const x = e as unknown as { value: number; hadRecentInput: boolean }; if (!x.hadRecentInput) w.__cls += x.value; })).observe({ type: "layout-shift", buffered: true }); });
    await p.goto(u);
    await p.waitForTimeout(3000);
    const cls = await p.evaluate(() => (window as unknown as { __cls: number }).__cls);
    const gap = await p.evaluate(() => { const h = document.querySelector("main h1"); const n = h?.nextElementSibling as HTMLElement | null; return n ? { h: Math.round(n.getBoundingClientRect().height), t: n.innerText.trim().slice(0, 30) } : null; });
    console.log(`INFO ${u}: CLS=${cls.toFixed(3)} phần tử kế h1:`, JSON.stringify(gap));
    expect(cls, `${u} CLS`).toBeLessThanOrEqual(0.05);
    if (gap && gap.t === "") expect(gap.h, "không còn dòng trống").toBeLessThanOrEqual(1);
  }
  await ctx.close();
});

test("TC-UX06-6/7/PQ ep:ui:course là lớp không còn / không thuộc mình → bỏ giá trị, chọn lớp theo AC2; SV chỉ lớp của mình", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  for (const user of ["teacher", "sv.gioi"]) {
    for (const stale of ["00000000-0000-7000-8000-00000000dead"]) {
      const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
      const p = await ctx.newPage();
      const errs: string[] = [];
      p.on("pageerror", (e) => errs.push(e.message));
      await p.addInitScript((id) => localStorage.setItem("ep:ui:course", id), stale);
      await login(p, user);
      await p.waitForTimeout(2500);
      const v = await p.evaluate(() => localStorage.getItem("ep:ui:course"));
      const mine = ((await api(p, "/me/courses")).body as { items: Array<{ course: { id: string; class_code: string } }> }).items;
      console.log(`INFO ${user}: ep:ui:course sau đăng nhập = ${v}; lớp của tôi = ${mine.map((m) => m.course.class_code).join(",")}`);
      expect(errs, `${user} không pageerror`).toEqual([]);
      expect(v, `${user} giá trị lạ phải bị xoá hoặc thay`).not.toBe(stale);
      if (v) expect(mine.map((m) => m.course.id), `${user} chỉ lớp của mình`).toContain(v);
      expect((await p.locator("main").innerText()).trim().length, "không trang trống").toBeGreaterThan(20);
      await ctx.close();
    }
  }
  // lớp có thật nhưng SV không thuộc: lấy từ admin
  const ca = await browser.newContext({ ignoreHTTPSErrors: true });
  const pa = await ca.newPage();
  await login(pa, "admin");
  const all = ((await api(pa, "/admin/courses?limit=100")).body as { items?: Array<{ id: string; class_code: string }> } | null)?.items ?? [];
  await ca.close();
  const cs = await browser.newContext({ ignoreHTTPSErrors: true });
  const ps = await cs.newPage();
  await login(ps, "sv.gioi");
  const mine = ((await api(ps, "/me/courses")).body as { items: Array<{ course: { id: string } }> }).items.map((m) => m.course.id);
  const other = all.find((c) => !mine.includes(c.id));
  console.log("INFO lớp SV không thuộc:", other ? other.class_code : "không có trong seed", "| tất cả:", all.map((c) => c.class_code).join(","));
  if (other) {
    expect((await api(ps, `/courses/${other.id}/today`)).status, "GET /courses/‹lớp khác›/today").toBe(403);
    await ps.evaluate((id) => localStorage.setItem("ep:ui:course", id), other.id);
    await ps.reload();
    await ps.waitForTimeout(2500);
    const v = await ps.evaluate(() => localStorage.getItem("ep:ui:course"));
    console.log("INFO SV với lớp lạ → ep:ui:course =", v);
    expect(v).not.toBe(other.id);
  }
  await cs.close();
});

test("TC-UX07-9 thông báo của giảng viên / SV không còn «Threads» / «Insights» / «Analytics»", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  for (const user of ["teacher", "ta", "sv.gioi"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true });
    const p = await ctx.newPage();
    await login(p, user);
    const r = await api(p, "/notifications?limit=50");
    const items = ((r.body as { items?: Array<{ type?: string; title?: string; body?: string; message?: string }> } | null)?.items ?? []);
    const texts = items.map((i) => `${i.type ?? ""}|${i.title ?? ""}|${i.body ?? i.message ?? ""}`);
    console.log(`INFO ${user}: ${items.length} thông báo; loại: ${[...new Set(items.map((i) => i.type))].join(",")}`);
    expect(texts.filter((t) => /\b(Threads|Insights|Analytics)\b/.test(t)), `${user} chữ cũ trong thông báo`).toEqual([]);
    await p.getByRole("button", { name: /Thông báo|thông báo mới/ }).first().click().catch(() => undefined);
    await p.waitForTimeout(800);
    expect(await p.locator("body").innerText()).not.toMatch(/\b(Threads|Insights|Analytics)\b/);
    await ctx.close();
  }
});

const post = (page: Page, path: string, body: unknown) =>
  page.evaluate(async ([p, b]) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${p}`, { method: "POST", headers: { authorization: `Bearer ${a.access_token}`, "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify(b) });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [path, body] as const);

test("TC-UX06-7/PQ lớp đã lưu trữ và lớp SV không thuộc → bỏ giá trị ep:ui:course", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ca = await browser.newContext({ ignoreHTTPSErrors: true });
  const pa = await ca.newPage();
  await login(pa, "admin");
  const users = ((await api(pa, "/admin/users?limit=100")).body as { items: Array<{ id: string; email: string }> }).items;
  const teacher = users.find((u) => u.email === "teacher@edupilot.local")!;
  const code = `QC${Date.now().toString().slice(-6)}`;
  const mk = await post(pa, "/admin/courses", { subject_code: "INT1006", class_code: code, name: "Lớp thử QC", semester: "2026-2027-HK1", teacher_id: teacher.id });
  console.log("INFO tạo lớp:", mk.status, code);
  expect(mk.status).toBeLessThan(300);
  const id = (mk.body as { id?: string; course?: { id: string } }).id ?? (mk.body as { course: { id: string } }).course.id;
  // SV không thuộc lớp này
  const cs = await browser.newContext({ ignoreHTTPSErrors: true });
  const ps = await cs.newPage();
  await ps.addInitScript((v) => localStorage.setItem("ep:ui:course", v), id);
  await login(ps, "sv.gioi");
  await ps.waitForTimeout(2000);
  const sv403 = (await api(ps, `/courses/${id}/today`)).status;
  const svVal = await ps.evaluate(() => localStorage.getItem("ep:ui:course"));
  console.log("INFO SV với lớp không thuộc: /today =", sv403, "; ep:ui:course =", svVal);
  expect(sv403).toBe(403);
  expect(svVal, "SV: bỏ giá trị").not.toBe(id);
  await cs.close();
  // GV: lớp còn → lưu trữ → đăng nhập lại
  const ct = await browser.newContext({ ignoreHTTPSErrors: true });
  const pt = await ct.newPage();
  await pt.addInitScript((v) => localStorage.setItem("ep:ui:course", v), id);
  await login(pt, "teacher");
  const kept = await pt.evaluate(() => localStorage.getItem("ep:ui:course"));
  console.log("INFO GV lớp còn hoạt động: ep:ui:course =", kept);
  const ar = await post(pa, `/admin/courses/${id}/archive`, {});
  console.log("INFO lưu trữ:", ar.status);
  await pt.evaluate((v) => localStorage.setItem("ep:ui:course", v), id);
  await pt.context().clearCookies();
  await login(pt, "teacher");
  await pt.waitForTimeout(2000);
  const after = await pt.evaluate(() => localStorage.getItem("ep:ui:course"));
  const mine = ((await api(pt, "/me/courses")).body as { items: Array<{ course: { id: string; class_code: string; status: string } }> }).items;
  console.log("INFO GV sau lưu trữ: ep:ui:course =", after, "; /me/courses =", mine.map((m) => `${m.course.class_code}:${m.course.status}`).join(","));
  expect(after, "GV: lớp đã lưu trữ → bỏ giá trị").not.toBe(id);
  expect((await pt.locator("main").innerText()).trim().length).toBeGreaterThan(20);
  await ct.close();
  await ca.close();
});
