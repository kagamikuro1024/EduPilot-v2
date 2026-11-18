import { test } from "@playwright/test";
import { login, go, FX, shot, log, psql, newUser } from "./lib";
const R = (tc: string, ok: boolean, note: string) => log("p405", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
test("P4-03-18", async ({ browser }) => {
  const f = FX("fx_ui4").ans; const { ctx, page } = await newUser(browser, f.u); await go(page, `/chat?session=${f.sid}`); await page.waitForTimeout(1500);
  const info = await page.evaluate(() => {
    const els = Array.from(document.querySelectorAll("main *")).filter((e) => e.children.length === 0 && (e.textContent ?? "").includes("QCANS"));
    if (!els.length) return null;
    let node: Element = els[0];
    while (node.parentElement && !(node.parentElement as HTMLElement).innerText.includes("QCINB")) node = node.parentElement;
    return { text: (node as HTMLElement).innerText.replace(/\n+/g, " | "), buttons: Array.from(node.querySelectorAll("button")).map((b) => (b as HTMLElement).innerText) };
  });
  await shot(page, "p403-18");
  R("TC-P4-03-18", !!info && /Giảng viên · /.test(info.text) && !/Trợ lý AI/.test(info.text) && !info.buttons.some((b) => /Hữu ích|Không hữu ích/.test(b)), `tin trả lời của GV trong /chat của SV: «${info?.text}»; nút dưới tin: ${JSON.stringify(info?.buttons)} (mong nhãn «Giảng viên · {tên}», không «Trợ lý AI», không Hữu ích / Không hữu ích)`);
  await ctx.close();
});
test("P4-05-15", async ({ browser }) => {
  const f = FX("fx_ui6").auto; const { ctx, page } = await newUser(browser, f.u); await go(page, `/chat?session=${f.sid}`); await page.waitForTimeout(1500);
  const st = await page.locator("[data-part=ticket-status]").allInnerTexts(); const txt = await page.locator("main").innerText();
  const hoi = await page.getByRole("button", { name: /Hỏi tiếp/ }).count();
  await shot(page, "p405-15");
  R("TC-P4-05-15", st.some((x) => /Đã đóng/.test(x)) || /Đã đóng/.test(txt) ? hoi >= 1 : false, `ticket CLOSED/AUTO_7D, SV mở /chat: dòng trạng thái ${JSON.stringify(st)}, «Đã đóng» trong trang=${/Đã đóng/.test(txt)}, nút «Hỏi tiếp»=${hoi}; (≤30 ngày kể từ đóng: API followup 200 đã kiểm ở 05-15 phần API)`);
  await ctx.close();
});
test("P4-05-28", async ({ browser }) => {
  const f = FX("fx_ui6").claim; const { ctx, page } = await newUser(browser, f.u); await go(page, `/chat?session=${f.sid}`); await page.waitForTimeout(1500);
  const loc = page.locator("[data-part=ticket-status]").first();
  const t0 = await loc.innerText(); const b0 = await loc.boundingBox(); const m0 = await page.locator("main").evaluate((e: HTMLElement) => e.scrollHeight);
  await page.waitForTimeout(125000);
  const t1 = await loc.innerText(); const b1 = await loc.boundingBox(); const m1 = await page.locator("main").evaluate((e: HTMLElement) => e.scrollHeight);
  R("TC-P4-05-28", t0 !== t1 && Math.abs((b0?.y ?? 0) - (b1?.y ?? 0)) < 1 && Math.abs((b0?.height ?? 0) - (b1?.height ?? 0)) < 1 && m0 === m1, `giữ /chat mở 125 s: «${t0}» → «${t1}» (Age đổi); vị trí khối y ${b0?.y?.toFixed(1)} → ${b1?.y?.toFixed(1)}, cao ${b0?.height?.toFixed(1)} → ${b1?.height?.toFixed(1)}, scrollHeight main ${m0} → ${m1} (không nhảy layout)`);
  await ctx.close();
});
test("P4-05-49", async ({ browser }) => {
  const f = FX("fx_ui4"); const { ctx, page } = await newUser(browser, "qc3t"); await go(page, `/class/settings?course=${f.A}`); await page.waitForTimeout(1500);
  const group = page.locator("section,div", { has: page.getByText("Ngưỡng chuyển cho giảng viên", { exact: true }) }).last();
  const gtxt = (await group.innerText()).replace(/\n+/g, " | ");
  const slider = page.locator("input[type=range]").first(); const v0 = await slider.inputValue();
  await slider.focus(); const key = Number(v0) < 0.85 ? "ArrowRight" : "ArrowLeft"; for (let i = 0; i < 5; i++) await page.keyboard.press(key);
  const v1 = await slider.inputValue(); await page.waitForTimeout(3000);
  const gr = psql(`select escalation_threshold from courses where id='${f.A}'`);
  const au = psql(`select before::text||' => '||after::text from audit_log where course_id='${f.A}' and (before::text like '%escalation_threshold%' or after::text like '%escalation_threshold%') order by created_at desc limit 1`);
  await shot(page, "p405-49");
  R("TC-P4-05-49", v1 !== v0 && gr !== "" && au !== "" && /Ít chuyển/.test(gtxt) && /Chuyển nhiều/.test(gtxt), `khối «Ngưỡng chuyển cho giảng viên»: «${gtxt.slice(0, 200)}»; kéo thanh ${v0} → ${v1}; DB courses.escalation_threshold=${gr}; audit_log mới nhất: ${au.slice(0, 200)}; đầu «Ít chuyển», cuối «Chuyển nhiều», không câu mô tả khác`);
  await ctx.close();
});
