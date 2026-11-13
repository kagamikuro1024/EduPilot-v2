/* eslint-disable @typescript-eslint/no-explicit-any */
import { test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}
const api = (page: Page, path: string) =>
  page.evaluate(async (p) => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await fetch(`/api/v1${p}`, { headers: { authorization: `Bearer ${a.access_token}` } });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, path);
const cid = async (page: Page, code: string) => (await api(page, "/me/courses")).body.items.find((i: any) => i.course.class_code === code).course.id as string;

const geom = (page: Page) =>
  page.evaluate(() => {
    const m = document.querySelector("main")!;
    const panels = [...m.querySelectorAll("[class*=Panel-module][class*=panel]")] as HTMLElement[];
    const top = panels.filter((p) => !p.parentElement?.closest("[class*=Panel-module][class*=panel]"));
    const info = top.map((p) => { const r = p.getBoundingClientRect(); return { y: Math.round(r.y + scrollY), b: Math.round(r.bottom + scrollY), w: Math.round(r.width), h: (p.previousElementSibling as HTMLElement | null)?.tagName ?? "", prevText: (p.previousElementSibling as HTMLElement | null)?.innerText.slice(0, 30) ?? "" }; });
    // phần tử con vừa có viền vừa bo góc trong Panel (khung lồng)
    const nested: string[] = [];
    for (const p of top) for (const e of p.querySelectorAll("*")) { const c = getComputedStyle(e); if (parseFloat(c.borderTopWidth) > 0 && c.borderTopStyle !== "none" && parseFloat(c.borderTopLeftRadius) >= 4 && !["BUTTON", "INPUT", "SELECT", "TEXTAREA"].includes(e.tagName) && (e as HTMLElement).getBoundingClientRect().width > 120 && !e.closest("[role=tablist], button, th, td")) nested.push(`${e.tagName}.${e.className.toString().slice(0, 28)}:${Math.round((e as HTMLElement).getBoundingClientRect().width)}`); }
    const hs = [...m.querySelectorAll("h1,h2,h3")].map((h) => { const r = h.getBoundingClientRect(); return { t: (h as HTMLElement).innerText.slice(0, 34), tag: h.tagName, y: Math.round(r.y + scrollY), b: Math.round(r.bottom + scrollY), next: Math.round((h.nextElementSibling?.getBoundingClientRect().y ?? 0) + scrollY) }; });
    return { panels: info, nested: nested.slice(0, 8), nestedN: nested.length, hs };
  });

test("TC-UX17 Insights thật 761988: Panel, khung lồng, chữ, khoảng cách khối", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "teacher@edupilot.local");
  await page.goto("/insights");
  await page.waitForTimeout(5000);
  const g = await geom(page);
  console.log("INFO UX17-1 Panel cấp trên:", JSON.stringify(g.panels));
  console.log("INFO UX17-2 khung lồng:", g.nestedN, JSON.stringify(g.nested));
  console.log("INFO UX17-6 tiêu đề:", JSON.stringify(g.hs));
  const gaps = g.panels.slice(1).map((p, i) => p.y - g.panels[i].b);
  console.log("INFO UX17-6 khoảng giữa Panel:", JSON.stringify(gaps));
  const lines = await page.locator("main").evaluate((m) => [...m.querySelectorAll("h1 ~ *, h2 ~ p, h2 + *, p")].filter((e) => e.tagName === "P" || /meta|sub|note|hint/i.test(e.className.toString())).slice(0, 12).map((e) => { const t = (e as HTMLElement).innerText.replace(/\s+/g, " ").trim(); return `${t.split(/\s+/).filter((w) => /\p{L}|\d/u.test(w)).length}t: ${t.slice(0, 90)}`; }));
  console.log("INFO UX17-3 các dòng phụ:", JSON.stringify(lines));
  await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/insights-1440-full.png", fullPage: true });
  // :active và hoạt cảnh
  const btn = page.getByRole("button", { name: "Tạo báo cáo mới" });
  const bb = (await btn.boundingBox())!;
  await page.mouse.move(bb.x + bb.width / 2, bb.y + bb.height / 2);
  await page.mouse.down();
  await page.waitForTimeout(250);
  console.log("INFO UX17-13 :active transform:", await btn.evaluate((e) => getComputedStyle(e).transform), "| nút Tạo báo cáo mới:", await btn.count());
  await page.mouse.move(5, 5); await page.mouse.up();
  await ctx.close();
  // lớp chưa có báo cáo
  const c2 = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const p2 = await c2.newPage();
  await login(p2, "teacher@edupilot.local");
  const c761987 = await cid(p2, "761987");
  const rep = await api(p2, `/courses/${c761987}/insights`);
  console.log("INFO UX17-9 API insights 761987:", rep.status, JSON.stringify(rep.body).slice(0, 160));
  await p2.evaluate((id) => localStorage.setItem("ep:ui:course", id), c761987);
  await p2.goto("/insights");
  await p2.waitForTimeout(4500);
  const t2 = (await p2.locator("main").innerText()).replace(/\n+/g, " ⏎ ");
  console.log("INFO UX17-9 /insights lớp không có báo cáo:", JSON.stringify(t2.slice(0, 400)), "| nút:", JSON.stringify(await p2.locator("main button").allInnerTexts()));
  await p2.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/insights-empty-1440.png" });
  // UX17-10: tạo báo cáo mới
  const create = p2.getByRole("button", { name: "Tạo báo cáo mới" });
  if (await create.count()) {
    await create.first().click();
    const seen: string[] = [];
    for (let i = 0; i < 16; i++) { await p2.waitForTimeout(400); const t = (await p2.locator("main").innerText()).replace(/\s+/g, " "); const m = t.match(/Đang [^.·⏎]{0,60}/); if (m && !seen.includes(m[0])) seen.push(m[0]); }
    console.log("INFO UX17-10 chỉ báo giai đoạn:", JSON.stringify(seen));
    await p2.goto("/"); await p2.waitForTimeout(1500); await p2.goto("/insights"); await p2.waitForTimeout(4500);
    console.log("INFO UX17-10 quay lại:", JSON.stringify((await p2.locator("main").innerText()).replace(/\n+/g, " ⏎ ").slice(0, 300)));
  }
  await c2.close();
});

test("TC-UX17 Quan sát AI thật: số liệu đúng một chỗ, bảng 7 ngày, khoảng cách, quyền", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(180000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "teacher@edupilot.local");
  await page.goto("/observability");
  await page.waitForTimeout(5000);
  const txt = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  const cnt = (re: RegExp) => (txt.match(re) ?? []).length;
  console.log("INFO UX17-4 số lần xuất hiện: yêu cầu hôm nay", cnt(/Yêu cầu hôm nay/g), "| Yêu cầu 7 ngày", cnt(/Yêu cầu 7 ngày/g), "| p95", cnt(/p95/g), "| Tỷ lệ lỗi", cnt(/Tỷ lệ lỗi/g), "| chuyển giảng viên", cnt(/chuyển giảng viên|Câu chuyển/g), "| che thông tin", cnt(/che thông tin|Lần che/g));
  const strip = await page.evaluate(() => { const m = document.querySelector("main")!; return [...m.querySelectorAll("[class*=today], [class*=strip], [class*=stat]")].map((e) => e.className.toString().slice(0, 40) + ":" + e.children.length); });
  console.log("INFO UX17-4 dải ô:", JSON.stringify(strip));
  const g = await geom(page);
  console.log("INFO UX17-1 Panel:", JSON.stringify(g.panels), "| lồng:", g.nestedN, JSON.stringify(g.nested));
  console.log("INFO UX17-6 tiêu đề:", JSON.stringify(g.hs));
  const tbl = await page.locator("main table").first().evaluate((t) => [...t.querySelectorAll("tr")].slice(0, 3).map((r) => [...r.children].map((c) => (c as HTMLElement).innerText.trim()).join(" | ")));
  console.log("INFO UX17-5 bảng 7 ngày:", JSON.stringify(tbl));
  // UX17-7: chọn một hàng yêu cầu
  const rows = page.locator("main table").nth(1).locator("tbody tr");
  console.log("INFO UX17-7 số hàng yêu cầu:", await rows.count());
  await rows.first().click().catch(() => undefined);
  await page.waitForTimeout(1200);
  console.log("INFO UX17-7 sau chọn hàng: dialog", await page.getByRole("dialog").count(), "| text:", JSON.stringify((await page.locator("body").innerText()).replace(/\s+/g, " ").match(/(Nội dung|prompt|Prompt|câu hỏi của sinh viên)[^.]{0,80}/g)?.slice(0, 3)));
  await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-5/observability-1440-full.png", fullPage: true });
  await ctx.close();
  // quyền
  const c761988 = await (async () => { const c = await browser.newContext({ ignoreHTTPSErrors: true }); const p = await c.newPage(); await login(p, "teacher@edupilot.local"); const id = await cid(p, "761988"); await c.close(); return id; })();
  for (const [who, mail] of [["STUDENT", "sv.kha@edupilot.local"], ["TA", "ta@edupilot.local"]] as const) {
    const c = await browser.newContext({ ignoreHTTPSErrors: true }); const p = await c.newPage(); await login(p, mail);
    const ids = (await api(p, "/me/courses")).body.items.map((i: any) => i.course.id) as string[];
    const target = ids.includes(c761988) ? c761988 : ids[0];
    const out: string[] = [];
    for (const path of ["insights", "analytics", "observability", "observability/summary", "observability/requests"]) out.push(`${path}:${(await api(p, `/courses/${target}/${path}`)).status}`);
    console.log(`INFO UX17-11 ${who}:`, JSON.stringify(out));
    for (const u of ["/insights", "/analytics", "/observability"]) { await p.goto(u); await p.waitForTimeout(2500); console.log(`INFO UX17-11 ${who} ${u}:`, JSON.stringify((await p.locator("main").innerText().catch(() => "")).replace(/\s+/g, " ").slice(0, 100))); }
    await c.close();
  }
  // 375
  for (const u of ["/insights", "/observability"]) {
    const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true }); const p = await c.newPage(); await login(p, "teacher@edupilot.local");
    await p.goto(u); await p.waitForTimeout(4500);
    console.log(`INFO UX17-12 ${u} 375: tràn ngang`, await p.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth), "| cao", await p.evaluate(() => document.documentElement.scrollHeight));
    await c.close();
  }
});
