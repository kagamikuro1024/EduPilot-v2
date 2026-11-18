/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkR (qc2, lượt 6): các ca phân quyền (PQ) của 9.1: TC-UX01-18/PQ, UX04-PQ, UX07-PQ, UX15-PQ, UX21-PQ.
import { test } from "@playwright/test";
import { api, courseId, expect, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
test.describe.configure({ mode: "serial" });
const DENY = "Bạn không có quyền xem màn này";
const body = async (p: any) => (await p.locator("main").innerText().catch(() => "")).replace(/\s+/g, " ");
const hrefs = (p: any) => p.locator("[data-part=sidebar] a, nav a").evaluateAll((as: Element[]) => as.map((a) => a.getAttribute("href")));

test("TC-UX01-18 TC-UX07-PQ TC-UX04-PQ SV: /observability /analytics /insights /gradebook /attendance → 403 nguyên văn, không lộ nav, không chuyển động mới", async ({ browser }, info) => {
  D(info); test.setTimeout(300_000);
  const o = await open(browser, "sv.gioi");
  for (const [u, who] of [["/observability", "giảng viên và quản trị viên"], ["/analytics", "giảng viên và trợ giảng"], ["/insights", "giảng viên và trợ giảng"], ["/gradebook", "giảng viên và trợ giảng"], ["/attendance", "giảng viên và trợ giảng"]] as const) {
    await o.page.goto(u); await o.page.waitForTimeout(2500);
    const t = await body(o.page);
    console.log(`INFO SV ${u}:`, JSON.stringify(t.slice(0, 140)));
    expect(t, `${u}: 403`).toContain(DENY);
    expect(t, `${u}: dòng «Trang này dành cho …» y nguyên`).toContain(`Trang này dành cho ${who}.`);
    const run = await o.page.evaluate(() => document.getAnimations().filter((a) => a.playState === "running" && a.effect && ((a.effect as any).target as Element | null)?.closest?.("main")).length);
    expect(run, `${u}: không hoạt cảnh đang chạy ở màn 403`).toBe(0);
  }
  const h = (await hrefs(o.page)).filter(Boolean) as string[];
  console.log("INFO SV nav:", JSON.stringify(h));
  for (const bad of ["/insights", "/analytics", "/observability", "/gradebook", "/attendance", "/grading", "/inbox"]) expect(h, `nav SV không có ${bad}`).not.toContain(bad);
  await o.ctx.close();
});

test("TC-UX01-PQ TA: /observability /settings/llm → 403; SV /insights → 403 (đã ở trên); chữ bị xoá không lộ dữ liệu", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "ta");
  for (const u of ["/observability", "/settings/llm"]) {
    await o.page.goto(u); await o.page.waitForTimeout(2500);
    const t = await body(o.page);
    console.log(`INFO TA ${u}:`, JSON.stringify(t.slice(0, 140)));
    expect(t, `TA ${u}`).toContain(DENY);
    expect(t, "không lộ số liệu / khoá").not.toMatch(/\d+ yêu cầu|p95|sk-|API key|masked/i);
  }
  await o.ctx.close();
});

test("TC-UX21-PQ SV đăng nhập → `/` là Hôm nay của sinh viên (không phải giảng viên)", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "sv.gioi");
  await o.page.goto("/"); await o.page.waitForTimeout(3500);
  const t = await body(o.page);
  const h = (await hrefs(o.page)).filter(Boolean) as string[];
  console.log("INFO UX21-PQ:", new URL(o.page.url()).pathname, JSON.stringify(t.slice(0, 160)), "| nav:", JSON.stringify(h));
  expect(new URL(o.page.url()).pathname).toBe("/");
  expect(await o.page.locator("main h1").first().innerText()).toMatch(/Hôm nay|Chào/);
  expect(h).toEqual(expect.arrayContaining(["/chat", "/me"]));
  for (const bad of ["/gradebook", "/grading", "/inbox", "/students", "/documents"]) expect(h, `nav SV không có ${bad}`).not.toContain(bad);
  expect(t, "không việc của giảng viên").not.toMatch(/Chấm bài|Duyệt|câu hỏi cần xử lý/i);
  await o.ctx.close();
});

test("TC-UX15-PQ giảng viên khác lớp: GET /courses/‹lớp lạ›/submissions/‹sid› → 403/404", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const t = await open(browser, "teacher"); const c = await courseId(t.page, "761987");
  const subs = await api(t.page, `/courses/${c}/submissions?limit=5`);
  const sid = subs.body.items[0].id as string;
  expect((await api(t.page, `/courses/${c}/submissions/${sid}`)).status, "chủ lớp đọc được").toBe(200);
  await t.ctx.close();
  const g = await open(browser, "gv.s8f");
  const gc = ((await api(g.page, "/me/courses?limit=100")).body.items as any[]).map((i) => i.course.id);
  expect(gc.includes(c), "gv.s8f không dạy 761987").toBe(false);
  const r = await api(g.page, `/courses/${c}/submissions/${sid}`);
  console.log("INFO UX15-PQ gv.s8f:", r.status, JSON.stringify(r.body).slice(0, 120));
  expect([403, 404]).toContain(r.status);
  const s = await open(browser, "sv.kha");
  const r2 = await api(s.page, `/courses/${c}/submissions/${sid}`);
  console.log("INFO UX15-PQ SV:", r2.status);
  expect([403, 404]).toContain(r2.status);
  await s.ctx.close(); await g.ctx.close();
});
