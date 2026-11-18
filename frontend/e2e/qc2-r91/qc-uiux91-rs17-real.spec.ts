/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkR (qc2, lượt 6): US-UX91-17 /insights + /observability — khẳng định thật cho TC-UX17-1…13, PQ.
import { test } from "@playwright/test";
import { api, courseId, expect, go, mainText, noOverflow, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
test.describe.configure({ mode: "serial" });

const topic = (rank: number, name: string) => ({ rank, name, why: "Nhiều sinh viên hỏi lại", score: 0.8 - rank / 20, signals: { students: 9, questions: 14, low_confidence_rate: 0.3, not_helpful_rate: 0.1, practice_wrong_rate: 0.2 }, recommendation: "Ôn lại ví dụ trên lớp", examples: ["Câu hỏi mẫu một", "Câu hỏi mẫu hai"], named_by: "llm", chapter: "Chương 3", week: 3, keywords: ["a", "b"] });
const FIX = {
  report: { id: "00000000-0000-4000-8000-0000000000a1", period_from: null, period_to: null, data_watermark: "2026-11-15T01:00:00Z", created_at: "2026-11-15T02:00:00Z", payload: { topics: [topic(1, "Mã hoá khoá công khai"), topic(2, "Chữ ký số"), topic(3, "Tường lửa")], uncovered: [{ text: "Quy trình khôi phục sự cố", count: 5 }, { text: "Kiểm thử xâm nhập", count: 3 }], counts: { questions: 120, students: 28, clusters: 9 }, period_days: 14 } },
  has_new_data: false,
  previous: { id: "00000000-0000-4000-8000-0000000000a0", created_at: "2026-11-08T02:00:00Z", new_topics: ["Tường lửa"], moved: [{ name: "Chữ ký số", from: 3, to: 2 }] },
};
const LATEST = /\/insights\/knowledge-gap\/latest(\?.*)?$/;

const geom = (page: any) => page.evaluate(() => {
  const m = document.querySelector("main")!;
  const all = [...m.querySelectorAll("[data-ep-panel]")] as HTMLElement[];
  const top = all.filter((p) => !p.parentElement?.closest("[data-ep-panel]"));
  const nested: string[] = [];
  for (const p of top) for (const e of p.querySelectorAll("*")) { const c = getComputedStyle(e); if (parseFloat(c.borderTopWidth) > 0 && c.borderTopStyle !== "none" && parseFloat(c.borderTopLeftRadius) >= 4 && !["BUTTON", "INPUT", "SELECT", "TEXTAREA", "SUMMARY"].includes(e.tagName) && (e as HTMLElement).getBoundingClientRect().width > 120 && !e.closest("[role=tablist], button, th, td")) nested.push(`${e.tagName}.${e.className.toString().slice(0, 30)}`); }
  const sections = ["Chủ đề nên dạy lại", "Tài liệu chưa đề cập", "So với báo cáo ngày"].map((t) => {
    const h = [...m.querySelectorAll("h2")].find((x) => (x as HTMLElement).innerText.startsWith(t)) as HTMLElement | undefined;
    if (!h) return null;
    const sec = (h.closest("[data-ep-section]") ?? h.parentElement) as HTMLElement; const ps = [...sec.querySelectorAll("[data-ep-panel]")].filter((p) => !p.parentElement?.closest("[data-ep-panel]"));
    const hb = h.getBoundingClientRect(); const pb = ps[0]?.getBoundingClientRect();
    return { t, panels: ps.length, hTop: hb.top + scrollY, hBottom: hb.bottom + scrollY, pTop: pb ? pb.top + scrollY : -1, pBottom: pb ? pb.bottom + scrollY : -1, hInPanel: !!h.closest("[data-ep-panel]") };
  });
  return { topN: top.length, nested, sections };
});

test("TC-UX17-1/2/3/6/13 Insights (báo cáo giả có đủ ba khối): một Panel mỗi khối, không khung lồng, chữ ngắn, khoảng cách, chuyển động", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher"); const c = await courseId(o.page, "761988");
  await o.page.route(LATEST, (r: any) => r.fulfill({ json: FIX }));
  await go(o.page, "/insights", c, 3500);
  await expect(o.page.getByRole("heading", { name: "Chủ đề nên dạy lại" })).toBeVisible({ timeout: 30_000 });
  const g = await geom(o.page);
  console.log("INFO UX17-1 geom:", JSON.stringify(g));
  // UX17-1: mỗi khối đúng MỘT Panel; tổng Panel cấp trên = 3 (không Panel lồng Panel)
  for (const s of g.sections) { expect(s, "khối có mặt").not.toBeNull(); expect(s!.panels, `«${s!.t}» đúng một Panel`).toBe(1); }
  expect(g.topN, "ba Panel cấp trên").toBe(3);
  // UX17-2
  expect(g.nested, "không phần tử vừa viền vừa bo góc bên trong Panel").toEqual([]);
  // UX17-3: không câu giải thích; meta ngắn ≤ 12 từ
  const metas: string[] = await o.page.locator("[data-ep-header-meta] > *").allInnerTexts();
  console.log("INFO UX17-3 meta:", JSON.stringify(metas));
  for (const m of metas) expect(m.trim().split(/\s+/).filter((w) => /\p{L}|\d/u.test(w)).length, `«${m}» ≤ 12 từ`).toBeLessThanOrEqual(12);
  expect(await mainText(o.page)).not.toMatch(/Báo cáo này|Hệ thống (sẽ|đã) |Bạn có thể (xem|dùng)/);
  // UX17-6: đáy Panel → tiêu đề khối kế ≥ 32, lớn hơn khoảng tiêu đề → Panel
  const S = g.sections as any[];
  for (let i = 1; i < S.length; i++) {
    const gap = S[i].hTop - S[i - 1].pBottom; const below = S[i].pTop - S[i].hBottom;
    console.log(`INFO UX17-6 khối ${i}: khoảng trên ${Math.round(gap)} / dưới tiêu đề ${Math.round(below)}`);
    expect(gap, "≥ 32 px").toBeGreaterThanOrEqual(32); expect(gap, "lớn hơn khoảng dưới tiêu đề").toBeGreaterThan(below);
  }
  // UX17-13: không hoạt cảnh mới khi mở; :active 0,97
  await o.page.waitForTimeout(1500);
  const anims = await o.page.evaluate(() => document.getAnimations().filter((a) => a.playState === "running" && !!(a.effect as any)?.target?.closest?.("main")).map((a) => (a as any).animationName ?? (a as any).transitionProperty ?? a.constructor.name));
  console.log("INFO UX17-13 hoạt cảnh đang chạy trong main:", JSON.stringify(anims));
  expect(anims).toEqual([]);
  const btn = o.page.getByRole("button", { name: "Tạo báo cáo mới" }).first();
  const bb = (await btn.boundingBox())!;
  await o.page.mouse.move(bb.x + bb.width / 2, bb.y + bb.height / 2); await o.page.mouse.down(); await o.page.waitForTimeout(250);
  const tf = await btn.evaluate((e: Element) => getComputedStyle(e).transform);
  await o.page.mouse.move(5, 5); await o.page.mouse.up();
  console.log("INFO UX17-13 :active transform:", tf);
  expect(tf).toBe("matrix(0.97, 0, 0, 0.97, 0, 0)");
  await o.ctx.close();
});

test("TC-UX17-12 375: ba khối xếp dọc, không tràn ngang (Insights giả + Quan sát AI)", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher", 375, 812); const c = await courseId(o.page, "761988");
  await o.page.route(LATEST, (r: any) => r.fulfill({ json: FIX }));
  await go(o.page, "/insights", c, 3500);
  await expect(o.page.getByRole("heading", { name: "Chủ đề nên dạy lại" })).toBeVisible({ timeout: 30_000 });
  const g = await geom(o.page);
  const S = g.sections as any[];
  for (let i = 1; i < S.length; i++) expect(S[i].hTop, "khối sau nằm dưới khối trước").toBeGreaterThanOrEqual(S[i - 1].pBottom);
  expect(await noOverflow(o.page), "/insights 375 không tràn ngang").toBe(true);
  await go(o.page, "/observability", c, 3500);
  expect(await noOverflow(o.page), "/observability 375 không tràn ngang").toBe(true);
  await o.ctx.close();
});

test("TC-UX17-10 Tạo báo cáo mới: chỉ báo giai đoạn, rời trang được, quay lại báo cáo cũ vẫn xem được", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher"); const c = await courseId(o.page, "761988");
  await o.page.route(LATEST, (r: any) => r.fulfill({ json: FIX }));
  await o.page.route(/\/insights\/knowledge-gap\/generate$/, (r: any) => r.fulfill({ status: 202, json: { job_id: "00000000-0000-4000-8000-0000000000b1" } }));
  await o.page.route(/\/jobs\/00000000-0000-4000-8000-0000000000b1$/, (r: any) => r.fulfill({ json: { id: "00000000-0000-4000-8000-0000000000b1", status: "RUNNING", progress: 10 } }));
  await go(o.page, "/insights", c, 3000);
  await o.page.getByRole("button", { name: "Tạo báo cáo mới" }).first().click();
  const stage = o.page.getByRole("status").filter({ hasText: /Đang gom câu hỏi của lớp/ });
  await expect(stage, "chỉ báo giai đoạn").toBeVisible({ timeout: 15_000 });
  console.log("INFO UX17-10 chỉ báo:", JSON.stringify(await stage.first().innerText()));
  // rời trang giữa chừng không bị chặn (không hộp thoại «rời trang»)
  let dlg = 0; o.page.on("dialog", (d: any) => { dlg++; void d.dismiss(); });
  await o.page.goto("/"); await o.page.waitForTimeout(1500);
  expect(dlg, "không hộp thoại chặn rời trang").toBe(0);
  await go(o.page, "/insights", c, 3000);
  await expect(o.page.getByRole("heading", { name: "Chủ đề nên dạy lại" }), "báo cáo cũ vẫn xem được").toBeVisible({ timeout: 20_000 });
  await expect(o.page.getByText("Mã hoá khoá công khai").first()).toBeVisible();
  await o.ctx.close();
});

test("TC-UX17-4/5/7 Quan sát AI (admin): số hôm nay một dòng, số 7 ngày một bảng, hàng đầu có số; chi tiết yêu cầu chỉ nội dung đã che", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "admin");
  await go(o.page, "/observability", undefined, 4500);
  const m = await o.page.evaluate(() => {
    const main = document.querySelector("main") as HTMLElement;
    const line = [...main.querySelectorAll("p, span, div")].filter((e) => e.children.length === 0 && /^\s*24 giờ qua:/.test((e as HTMLElement).innerText)).map((e) => (e as HTMLElement).innerText);
    const h2 = [...main.querySelectorAll("h2")].find((h) => /7 ngày qua/.test((h as HTMLElement).innerText)) as HTMLElement;
    const tbl = (h2?.closest("[data-ep-section]") ?? h2?.parentElement)?.querySelector("table");
    const head = tbl ? [...tbl.querySelectorAll("thead th")].map((t) => (t as HTMLElement).innerText.trim()) : [];
    const body = tbl ? [...tbl.querySelectorAll("tbody tr")].map((r) => [...r.children].map((c) => (c as HTMLElement).innerText.trim())) : [];
    const tiles = [...main.querySelectorAll("[class*=tile], [class*=kpi], [data-ep-stat]")].length;
    return { line, head, body, tiles };
  });
  console.log("INFO UX17-4/5:", JSON.stringify(m));
  expect(m.line.length, "đúng một dòng chữ «24 giờ qua: …»").toBe(1);
  expect(m.line[0]).toMatch(/yêu cầu/); expect(m.line[0]).toMatch(/p95/); expect(m.line[0]).toMatch(/lỗi/);
  expect(m.tiles, "không dải ô số cùng cỡ").toBe(0);
  expect(m.head.length, "bảng 7 ngày có cột").toBeGreaterThanOrEqual(5);
  expect(m.body.length, "bảng 7 ngày có hàng").toBeGreaterThanOrEqual(5);
  for (const c of m.body[0]) expect(c.trim().length, "hàng đầu không ô trống").toBeGreaterThan(0);
  for (const col of ["p95", "Yêu cầu"]) expect(m.head.filter((h: string) => h === col).length, `cột «${col}» đúng một lần`).toBe(1);
  // UX17-7: mở một hàng yêu cầu
  const reqTbl = o.page.locator("main table").last();
  const rows = reqTbl.locator("tbody tr"); const n = await rows.count();
  expect(n).toBeGreaterThan(0);
  await rows.first().click();
  const dlg = o.page.getByRole("dialog");
  await expect(dlg).toBeVisible({ timeout: 10_000 });
  const before = (await dlg.innerText()).replace(/\s+/g, " ");
  console.log("INFO UX17-7 trước khi nêu lý do:", JSON.stringify(before.slice(0, 200)));
  expect(before, "chưa có nội dung trước khi nêu lý do").not.toContain("Nội dung đã che");
  expect(before).toContain("Nội dung là câu hỏi của sinh viên");
  await dlg.getByRole("textbox").fill("Kiểm tra QC hồi quy sprint 9.1");
  await dlg.getByRole("button", { name: "Xem chi tiết" }).click();
  await o.page.waitForTimeout(2500);
  const after = (await dlg.innerText()).replace(/\s+/g, " ");
  console.log("INFO UX17-7 sau:", JSON.stringify(after.slice(0, 400)));
  if (after.includes("Nội dung đã che")) {
    expect(after, "không email thật").not.toMatch(/[\w.+-]+@(?!example\.invalid)[\w-]+\.[a-z]{2,}/i);
    expect(after, "không MSSV 8 chữ số").not.toMatch(/\b\d{8}\b/);
  } else expect(after, "nội dung hết hạn lưu hoặc lỗi chuẩn").toMatch(/hết hạn|Không|Chưa/);
  await o.ctx.close();
});

test("TC-UX17-4/7 giảng viên: không bảng yêu cầu, không nội dung, mỗi chỉ số một chỗ", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher"); const c = await courseId(o.page, "761988");
  await go(o.page, "/observability", c, 4500);
  const txt = await mainText(o.page);
  for (const k of ["Yêu cầu", "Tỷ lệ lỗi", "Chuyển giảng viên", "Lần che thông tin"]) expect((txt.match(new RegExp(k, "g")) ?? []).length, `«${k}» đúng một chỗ`).toBe(1);
  expect(await o.page.locator("main table").count(), "không bảng yêu cầu").toBe(0);
  expect(await o.page.getByRole("button", { name: "Xem chi tiết" }).count()).toBe(0);
  expect(txt).not.toMatch(/Nội dung đã che|masked_prompt/);
  await o.ctx.close();
});

test("TC-UX17-11/PQ phân quyền: SV 403 ba trang; TA /observability 403, /insights /analytics theo quyền; giảng viên lớp lạ 403/404", async ({ browser }, info) => {
  D(info); test.setTimeout(300_000);
  const t = await open(browser, "teacher"); const c88 = await courseId(t.page, "761988");
  const lib = await api(t.page, "/me/courses?limit=100");
  const mine = new Set((lib.body.items as any[]).map((i) => i.course.id));
  await t.ctx.close();
  const sv = await open(browser, "sv.moi");
  const svCourses = ((await api(sv.page, "/me/courses?limit=100")).body.items as any[]).map((i) => i.course.id);
  const foreign = svCourses.find((id: string) => !mine.has(id)) ?? c88; // lớp mà giảng viên `teacher` không dạy; nếu không có dùng 761988
  // SV: UI 403 ba trang
  for (const u of ["/insights", "/analytics", "/observability"]) {
    await sv.page.goto(u); await sv.page.waitForTimeout(2500);
    const tx = (await sv.page.locator("main").innerText().catch(() => "")).replace(/\s+/g, " ");
    console.log(`INFO UX17-11 SV ${u}:`, JSON.stringify(tx.slice(0, 120)));
    expect(tx, `SV ${u}`).toContain("Bạn không có quyền xem màn này");
  }
  for (const p of ["insights/knowledge-gap/latest", "analytics", "observability/summary"]) expect((await api(sv.page, `/courses/${c88}/${p}`)).status, `SV API ${p}`).toBeGreaterThanOrEqual(403);
  await sv.ctx.close();
  const ta = await open(browser, "ta");
  await ta.page.goto("/observability"); await ta.page.waitForTimeout(2500);
  const tt = (await ta.page.locator("main").innerText().catch(() => "")).replace(/\s+/g, " ");
  console.log("INFO UX17-11 TA /observability:", JSON.stringify(tt.slice(0, 120)));
  expect(tt).toContain("Bạn không có quyền xem màn này");
  const taCourses = ((await api(ta.page, "/me/courses?limit=100")).body.items as any[]).map((i) => i.course.id);
  const taC = taCourses[0];
  const insTA = await api(ta.page, `/courses/${taC}/insights/knowledge-gap/latest`);
  console.log("INFO UX17-11 TA insights API:", insTA.status, "| payload có câu hỏi chat riêng?", JSON.stringify(insTA.body).includes("private_chat"));
  expect([200, 403, 404]).toContain(insTA.status);
  expect(JSON.stringify(insTA.body)).not.toMatch(/masked_prompt|private_chat/);
  await ta.ctx.close();
  // giảng viên khác lớp: một giảng viên không dạy lớp đó
  const g = await open(browser, "gv.s8f");
  const gc = ((await api(g.page, "/me/courses?limit=100")).body.items as any[]).map((i) => i.course.id);
  const lopLa = [...mine].find((id) => !gc.includes(id)) as string;
  const r1 = await api(g.page, `/courses/${lopLa}/insights/knowledge-gap/latest`);
  const r2 = await api(g.page, `/courses/${lopLa}/analytics`);
  console.log("INFO UX17-PQ gv.s8f lớp lạ:", r1.status, r2.status);
  expect([403, 404]).toContain(r1.status); expect([403, 404]).toContain(r2.status);
  void foreign;
  await g.ctx.close();
});
