import { expect, test } from "@playwright/test";
import { needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// QC smoke 9 — TC-P7-06-15 / TC-P7-07-06 (DEF-R9S-02): hàng chờ /grading nêu nguyên nhân bài «Chấm lỗi».
needReal();
const WANT: Record<string, string> = {
  LLM_BUDGET_EXHAUSTED: "Hết ngân sách AI hôm nay — chấm lại ngày mai hoặc chấm tay",
  LLM_UNAVAILABLE: "AI tạm thời không phản hồi",
  GRADING_OUTPUT_INVALID: "AI trả kết quả không dùng được",
  EXTRACT_FAILED: "Không đọc được nội dung tệp",
  TEXT_TOO_LONG: "Bài quá dài để chấm",
};
for (const vp of [{ w: 1440, h: 900 }, { w: 390, h: 844 }]) {
  test(`@real FAILED nguyên nhân ${vp.w}px`, async ({ page }) => {
    await page.setViewportSize({ width: vp.w, height: vp.h });
    const cid = sql(`select id from courses where class_code='QD9G'`)[0][0];
    await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), cid);
    await uiLogin(page, "teacher");
    await page.goto(`/grading?course=${cid}&status=FAILED`);
    await page.locator("main").waitFor();
    await page.waitForTimeout(3000);
    const txt = await page.locator("main").innerText();
    await shot(page, `failed-${vp.w}`);
    const codes = sql(`select distinct error_code from grading_results where course_id='${cid}' and run_state='FAILED'`).map((r) => r[0]);
    const have: Record<string, boolean> = {};
    for (const c of codes) if (WANT[c]) have[c] = txt.includes(WANT[c]);
    console.log("CAUSES", vp.w, JSON.stringify(have), "chấm lỗi:", (txt.match(/Chấm lỗi/g) ?? []).length);
    for (const [c, ok] of Object.entries(have)) expect.soft(ok, `${c} hiện «${WANT[c]}»`).toBe(true);
    await noOverflow(page);
  });
}
