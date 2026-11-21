/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkS lượt 8 (qc2, 2026-11-21): TC-UX17-8/9, TC-UX18-4/6/8/PQ có assertion thật (bản rw5g chỉ in INFO). Chạy: run8.sh qc-uiux91-ru1-real
import { test } from "@playwright/test";
import { execSync } from "node:child_process";
import { api, courseId, expect, go, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
test.describe.configure({ mode: "serial" });
const psql = (q: string) => execSync(`docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -At -F '|' -c "${q.replace(/"/g, '\\"')}"`).toString().trim();
const setCourse = async (page: any, id: string) => { await page.evaluate((c: string) => localStorage.setItem("ep:ui:course", c), id); };

test("TC-UX17-9 lớp chưa có báo cáo Insights: EmptyState + đúng một nút «Tạo báo cáo mới»", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop"); test.setTimeout(120_000);
  const code = psql("select class_code from courses c where class_code like 'QR91%' and not exists (select 1 from insight_reports i where i.course_id=c.id) order by class_code limit 1");
  console.log("INFO UX17-9 lớp không có insight_reports:", code);
  expect(code).not.toBe("");
  const o = await open(browser, "teacher"); const c = await courseId(o.page, code);
  await setCourse(o.page, c);
  await go(o.page, "/insights", c, 4500);
  const main = (await o.page.locator("main").innerText()).replace(/\s+/g, " ");
  const btns = await o.page.locator("main button").allInnerTexts();
  const create = await o.page.getByRole("button", { name: "Tạo báo cáo mới" }).count();
  console.log("INFO UX17-9 main:", JSON.stringify(main.slice(0, 200)), "| nút trong main:", JSON.stringify(btns), "| «Tạo báo cáo mới»:", create);
  expect(main).toContain("Chưa có báo cáo");
  expect(create, "đúng một nút «Tạo báo cáo mới»").toBe(1);
  expect(await o.page.locator("main [data-ep-empty], main [data-part=empty-state]").locator("button").count() + 0).toBeLessThanOrEqual(1);
  await o.ctx.close();
});

test("TC-UX17-8 admin xem nội dung prompt + audit_log mới", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop"); test.setTimeout(200_000);
  const before = +psql("select count(*) from audit_log where action='VIEW_DETAIL' and entity='llm_request'");
  const o = await open(browser, "admin");
  await o.page.goto("/observability"); await o.page.waitForTimeout(5000);
  const rows = o.page.locator("table", { has: o.page.locator("caption", { hasText: "Yêu cầu gửi tới model" }) }).locator("tbody tr");
  const n = await rows.count();
  console.log("INFO UX17-8 số hàng yêu cầu:", n, "| audit VIEW_DETAIL trước:", before);
  expect(n).toBeGreaterThan(0);
  let seen = false, tries = 0, title = "";
  for (let i = 0; i < Math.min(n, 12) && !seen; i++) {
    await rows.nth(i).click(); tries++;
    const dlg = o.page.getByRole("dialog");
    await dlg.waitFor({ timeout: 8000 });
    title = (await dlg.locator("h2, [role=heading]").first().innerText().catch(() => "")).replace(/\s+/g, " ");
    await dlg.getByRole("textbox").fill("QC SmkS kiểm tra nội dung đã che (TC-UX17-8)");
    await dlg.getByRole("button", { name: "Xem chi tiết" }).click();
    await o.page.waitForTimeout(2500);
    const t = (await dlg.innerText()).replace(/\s+/g, " ");
    if (/Nội dung đã che/.test(t)) { seen = true; console.log("INFO UX17-8 thấy nội dung:", JSON.stringify(t.slice(0, 260))); }
    else { console.log("INFO UX17-8 hàng", i, "không có nội dung:", JSON.stringify(t.slice(0, 160))); await o.page.keyboard.press("Escape"); await o.page.waitForTimeout(600); }
  }
  const after = +psql("select count(*) from audit_log where action='VIEW_DETAIL' and entity='llm_request'");
  const last = psql("select actor_id::text||'|'||entity_id||'|'||after::text from audit_log where action='VIEW_DETAIL' and entity='llm_request' order by created_at desc limit 1");
  console.log("INFO UX17-8 audit sau:", after, "| số lần mở:", tries, "| dòng cuối:", last, "| tiêu đề ngăn:", title);
  expect(seen, "admin xem được nội dung prompt đã che").toBe(true);
  expect(after - before, "mỗi lần mở một dòng audit_log mới").toBe(tries);
  await o.ctx.close();
});

test("TC-UX18-4/6/8/PQ công tắc tài liệu, thư viện SV, quyền", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop"); test.setTimeout(200_000);
  const t = await open(browser, "teacher"); const c = await courseId(t.page, "761987");
  await setCourse(t.page, c);
  await go(t.page, "/documents", c, 5000);
  // UX18-4: tên truy cập «Dùng cho AI: bật/tắt», «Hiện cho sinh viên: bật/tắt» (getByRole tính tên truy cập thật)
  const grp = { aiOn: /^Dùng cho AI: bật/, aiOff: /^Dùng cho AI: tắt/, svOn: /^Hiện cho sinh viên: bật/, svOff: /^Hiện cho sinh viên: tắt/ };
  const total = await t.page.getByRole("switch").count();
  const cnt: Record<string, number> = {}; const colors: Record<string, string[]> = {};
  for (const [k, re] of Object.entries(grp)) {
    const l = t.page.getByRole("switch", { name: re }); cnt[k] = await l.count();
    colors[k] = [...new Set(await l.evaluateAll((els) => els.map((e) => getComputedStyle(e, "::before").backgroundColor) /* đường ray vẽ bằng ::before (Field.module.css .switch::before) */))];
  }
  console.log("INFO UX18-4 tổng công tắc:", total, "| theo tên:", JSON.stringify(cnt), "| màu:", JSON.stringify(colors));
  expect(total).toBeGreaterThan(0);
  expect(cnt.aiOn + cnt.aiOff + cnt.svOn + cnt.svOff, "mọi công tắc có tên «…: bật/tắt»").toBe(total);
  expect(cnt.aiOn + cnt.svOn, "có công tắc bật").toBeGreaterThan(0); expect(cnt.aiOff + cnt.svOff, "có công tắc tắt").toBeGreaterThan(0);
  const cOn = [...new Set([...colors.aiOn, ...colors.svOn])], cOff = [...new Set([...colors.aiOff, ...colors.svOff])];
  expect(cOn.length, "màu bật đồng nhất mọi hàng").toBe(1); expect(cOff.length, "màu tắt đồng nhất mọi hàng").toBe(1);
  expect(cOn[0], "bật khác tắt về màu").not.toBe(cOff[0]);
  // UX18-6: tài liệu «Hiện cho sinh viên: tắt» không có trong /library của SV
  const hidden = [...new Set(await t.page.getByRole("switch", { name: grp.svOff }).evaluateAll((els) => els.map((e) => (e.closest("tr")?.querySelector("td")?.textContent ?? "").trim()).filter(Boolean)))];
  console.log("INFO UX18-6 tài liệu ẩn với SV:", JSON.stringify(hidden));
  expect(hidden.length, "có ít nhất 1 tài liệu ẩn với SV (vd «Đáp án»)").toBeGreaterThan(0);
  const s = await open(browser, "sv.kha");
  await s.page.goto("/library"); await s.page.waitForTimeout(4500);
  const lib = (await s.page.locator("main").innerText()).replace(/\s+/g, " ");
  for (const h of hidden) expect(lib.includes(h), `Thư viện SV không chứa «${h}»`).toBe(false);
  // UX18-8: SV vào /documents bị chặn; /library được; API 403
  await s.page.goto("/documents"); await s.page.waitForTimeout(3000);
  const dtxt = (await s.page.locator("main").innerText()).replace(/\s+/g, " ");
  console.log("INFO UX18-8 SV /documents:", JSON.stringify(dtxt.slice(0, 120)));
  expect(dtxt).toMatch(/không có quyền|dành cho giảng viên/);
  const apiDoc = await api(s.page, `/courses/${await courseId(s.page, "761987")}/documents?limit=5`);
  console.log("INFO UX18-8 API:", apiDoc.status, "| /library dài", lib.length);
  expect(apiDoc.status).toBe(403); expect(lib.length).toBeGreaterThan(200);
  await s.ctx.close();
  // UX18-PQ: giảng viên không phụ trách lớp lạ
  const other = psql("select id from courses c where not exists (select 1 from enrollments e join users u on u.id=e.user_id where e.course_id=c.id and u.email='teacher@edupilot.local') limit 1");
  const r = await api(t.page, `/courses/${other}/documents?limit=5`);
  console.log("INFO UX18-PQ GV lớp lạ", other, "→", r.status);
  expect([403, 404]).toContain(r.status);
  await t.ctx.close();
});

// ---- DEF-R9.1-06 (sửa 6d127b88): chấm đã chọn giữ đỏ khi rê chuột; luật rê không có trên cảm ứng ----
test("TC-UX12-2 chấm điểm danh đã chọn giữ đỏ khi rê (desktop) + luật rê không áp trên cảm ứng 375", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop"); test.setTimeout(180_000);
  const o = await open(browser, "teacher"); const c = await courseId(o.page, process.env.QC_CLASS ?? "QR91A");
  await go(o.page, `/attendance?course=${c}`, undefined, 3000);
  const row = o.page.locator("main tbody tr[data-part=student-row]").nth(4);
  const was = await row.locator("input:checked").first().getAttribute("data-status");
  const res: Record<string, { away: string; over: string }> = {};
  for (const st of ["LATE", "EXCUSED", "ABSENT"]) {
    const dot = row.locator(`input[data-status=${st}]`);
    await dot.check({ force: true });
    await o.page.mouse.move(5, 5); await o.page.waitForTimeout(300);
    const away = await dot.evaluate((e) => getComputedStyle(e).borderColor);
    await dot.hover(); await o.page.waitForTimeout(300);
    const over = await dot.evaluate((e) => getComputedStyle(e).borderColor);
    res[st] = { away, over }; console.log(`INFO UX12-2 ${st} chuột-xa=${away} chuột-trên-ô=${over}`);
  }
  await row.locator(`input[data-status=${was ?? "PRESENT"}]`).check({ force: true });
  for (const [st, v] of Object.entries(res)) { expect(v.away, `${st} đỏ khi chuột xa`).toBe("rgb(206, 22, 40)"); expect(v.over, `${st} vẫn đỏ khi rê lên ô`).toBe("rgb(206, 22, 40)"); }
  await o.ctx.close();
  // cảm ứng 375: mọi luật :hover của .radio nằm trong @media (hover: hover) và điều kiện đó sai ⇒ không áp
  const m = await open(browser, "teacher", 375, 812);
  await go(m.page, `/attendance?course=${c}`, undefined, 3000);
  const r = await m.page.evaluate(() => {
    const out: { sel: string; cond: string | null; applies: boolean | null }[] = [];
    const walk = (rules: CSSRuleList, cond: string | null) => { for (const ru of Array.from(rules)) {
      if (ru instanceof CSSMediaRule) walk(ru.cssRules, ru.conditionText);
      else if (ru instanceof CSSStyleRule && /radio/.test(ru.selectorText) && /:hover/.test(ru.selectorText)) out.push({ sel: ru.selectorText, cond, applies: cond === null ? true : matchMedia(cond).matches });
    } };
    for (const sh of Array.from(document.styleSheets)) { try { walk(sh.cssRules, null); } catch { /* tệp ngoài miền */ } }
    return { hoverHover: matchMedia("(hover: hover)").matches, rules: out };
  });
  console.log("INFO UX12-2 375:", JSON.stringify(r));
  expect(r.hoverHover, "375 + cảm ứng: (hover: hover) sai").toBe(false);
  expect(r.rules.length, "có luật :hover cho .radio").toBeGreaterThan(0);
  for (const x of r.rules) expect(x.applies, `luật ${x.sel} không áp trên cảm ứng`).toBe(false);
  await m.ctx.close();
});
