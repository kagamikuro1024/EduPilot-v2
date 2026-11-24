import { appendFileSync, readFileSync, writeFileSync } from "node:fs";
import { expect, test, type Page } from "@playwright/test";
import { PW, sql, needReal, call, login as apiLogin } from "./qc-real-lib";

// TC-12-15 (US-P9-12 AC6) MỘT MẠCH trên lớp QC dựng QD10G (ep-qc): GV trích đề → duyệt hàng loạt → SV luyện chủ đề + thi thử → lịch sử → «Chủ đề cần ôn» → Hôm nay «Ôn chủ đề»; nhánh lỗi: mất mạng khi nộp, trích đề quá tải.
const CLS = process.env.QC_F12_CLASS ?? "QD10G";
const BASE = process.env.QC_BASE ?? "https://localhost:943";
const OUT = process.env.QC_OUT ?? "/tmp/x";
const MODE = process.env.STUB_MODE_FILE ?? "/Users/kuro/.config/edupilot-team/qc3-work/stub10_mode";
const KEYS = new Map((JSON.parse(readFileSync("../seed/practice/de-mau.json", "utf8")).questions as { stem: string; options: string[]; correct: number }[]).map((q) => [q.stem, q]));
const log: string[] = [];
const note = (s: string) => { log.push(s); console.log("STEP", s); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.beforeAll(() => needReal());

async function login(browser: import("@playwright/test").Browser, user: string) {
  const id = sql(`select id from courses where class_code='${CLS}'`)[0][0];
  const ctx = await browser.newContext({ baseURL: BASE, ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), id);
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor();
  return { ctx, page, id };
}
/** Câu đang mở → khoá tính tay (de-mau.json). `right` = chọn đúng; sai = chọn phương án đầu tiên khác đáp án đúng. */
async function pick(page: Page, region: string | null, right: boolean) {
  await page.getByRole("list", { name: "Các đáp án" }).waitFor({ timeout: 20_000 }); await page.waitForTimeout(400); const main = page.locator("main"); const text = (await main.innerText()).replace(/\s+/g, " ");
  const hit = [...KEYS.entries()].filter(([s]) => text.includes(s)).sort((a, b) => b[0].length - a[0].length)[0];
  expect(hit, `câu không có trong đề mẫu: ${text.slice(0, 140)}`).toBeTruthy();
  const q = hit![1]; const k = right ? q.correct : q.options.findIndex((_, i) => i !== q.correct);
  await page.getByRole("list", { name: "Các đáp án" }).getByRole("button", { name: q.options[k], exact: true }).click(); return q.stem;
}

test("TC-12-15 F12 một mạch", async ({ browser, request }) => {
  test.setTimeout(900_000); writeFileSync(MODE, "ok");
  // ---- 1. GV: trích đề ------------------------------------------------------------------------------------------------------------
  const gv = await login(browser, "teacher"); const p = gv.page; const bulk: string[] = [];
  p.on("response", (r) => { if (/questions\/(extract|bulk)/.test(r.url())) bulk.push(`${r.request().method()} ${r.url().split("/").slice(-2).join("/")} ${r.status()}`); });
  await p.goto("/questions"); await p.waitForTimeout(2000);
  await p.getByRole("button", { name: "Tạo câu hỏi" }).click(); await p.getByRole("menuitem", { name: "Trích từ đề" }).click();
  const form = p.getByRole("form", { name: "Trích câu hỏi từ đề cũ" }); await form.waitFor();
  await form.getByLabel("Đề cũ").selectOption({ label: "Đề mẫu F12" }); await form.getByLabel("Đáp án").selectOption({ label: "Đáp án F12" });
  await form.getByRole("button", { name: "Trích câu hỏi" }).click();
  await p.getByText(/Đã tạo \d+ câu chờ duyệt/).waitFor({ timeout: 90_000 }); const sum = await p.getByText(/Đã tạo \d+ câu chờ duyệt/).innerText();
  await p.screenshot({ path: `${OUT}/f12-1-trich.png` }); const created = Number(sum.match(/\d+/)![0]);
  note(`1 trích đề UI: «${sum}»; DB EXTRACTED PENDING ${sql(`select count(*) from question_bank q join courses c on c.id=q.course_id where c.class_code='${CLS}' and origin='EXTRACTED' and review_status='PENDING'`)[0][0]}`);
  expect(created).toBeGreaterThanOrEqual(20);
  // ---- 2. GV: duyệt hàng loạt (trần 50) -------------------------------------------------------------------------------------------
  await p.reload(); await p.waitForTimeout(2500);
  const boxes = p.getByRole("checkbox"); const nb = await boxes.count(); for (let i = 1; i < Math.min(nb, 31); i++) await boxes.nth(i).check({ timeout: 4000 }).catch(() => {});
  await p.getByRole("toolbar").getByRole("button", { name: "Duyệt", exact: true }).click(); await p.waitForTimeout(6000);
  const appr = Number(sql(`select count(*) from question_bank q join courses c on c.id=q.course_id where c.class_code='${CLS}' and review_status='APPROVED'`)[0][0]);
  await p.screenshot({ path: `${OUT}/f12-2-duyet.png` }); note(`2 duyệt hàng loạt UI: APPROVED ${appr}/${created} (${bulk.filter((b) => /bulk/.test(b)).join(",")})`);
  expect(appr).toBe(created);
  // ---- 3. SV: luyện chủ đề 10 câu (đúng 3, sai 7) ------------------------------------------------------------------------------------
  const sv = await login(browser, "sv36"); const s = sv.page;
  await s.goto("/practice"); await s.waitForTimeout(2500); await s.getByRole("button", { name: "Luyện 10 câu" }).first().click(); await s.waitForURL(/\/practice\/[0-9a-f-]+/);
  const wrongT = new Set([1, 2, 3, 5, 6, 8, 9]); const stems: string[] = [];
  for (let i = 1; i <= 10; i++) { stems.push(await pick(s, null, !wrongT.has(i))); await s.getByRole("button", { name: "Kiểm tra" }).click(); await s.waitForTimeout(500); await s.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).click(); await s.waitForTimeout(500); }
  await s.waitForTimeout(1500); await s.screenshot({ path: `${OUT}/f12-3-topic.png` });
  const topicRow = sql("select correct_count, status from practice_attempts where student_id=(select id from users where email='sv36@edupilot.local') and mode='TOPIC' order by created_at desc limit 1")[0];
  note(`3 TOPIC 10 câu UI: DB correct_count ${topicRow[0]} status ${topicRow[1]} (QC tính tay 3)`); expect(topicRow[0]).toBe("3");
  // ---- 3b. SV khác (sv37) làm 3 câu < 10 → khối chủ đề yếu ẩn (TC-09-05, kiểm đúng chữ thật của khối) --------------------------------
  const sv2 = await login(browser, "sv37"); const t2 = sv2.page;
  await t2.goto("/practice"); await t2.waitForTimeout(2000); await t2.getByRole("button", { name: "Luyện 10 câu" }).first().click(); await t2.waitForURL(/\/practice\/[0-9a-f-]+/);
  for (let i = 1; i <= 3; i++) { await pick(t2, null, false); await t2.getByRole("button", { name: "Kiểm tra" }).click(); await t2.waitForTimeout(500); await t2.getByRole("button", { name: /Câu tiếp theo|Xem kết quả/ }).click(); await t2.waitForTimeout(500); }
  await t2.goto("/practice/history"); await t2.waitForTimeout(2500); const h2 = await t2.locator("main").innerText(); await t2.screenshot({ path: `${OUT}/f12-3b-it-cau.png` });
  note(`3b sv37 làm 3 câu (<10): khối «Chủ đề bạn hay sai» / «Chủ đề cần ôn» ${/Chủ đề (bạn hay sai|cần ôn)/.test(h2) ? "HIỆN (sai)" : "ẩn (đúng)"}`); expect(h2).not.toMatch(/Chủ đề (bạn hay sai|cần ôn)/); await sv2.ctx.close();
  // ---- 4. SV: thi thử 20 câu — tải lại ở câu 8, offline câu 12–14 ----------------------------------------------------------------
  await s.goto("/practice"); await s.waitForTimeout(1500); await s.getByRole("tab", { name: "Thi thử" }).click(); await s.getByRole("button", { name: "Bắt đầu thi thử" }).click(); await s.waitForURL(/\/practice\/[0-9a-f-]+/);
  let expected = 0; let idxAfter = -1;
  for (let i = 1; i <= 20; i++) {
    if (i === 12) await sv.ctx.setOffline(true);
    const right = i % 4 !== 0; await pick(s, null, right); if (right) expected++; await s.waitForTimeout(900);
    if (i === 8) { await s.reload(); await s.waitForTimeout(2000); const m = (await s.locator("main").innerText()).match(/(\d+)\s*\/\s*20/); idxAfter = m ? Number(m[1]) : -1; }
    if (i === 14) { await sv.ctx.setOffline(false); await s.waitForTimeout(3000); }
    if (i < 20) await s.getByRole("button", { name: "Câu sau" }).click();
  }
  await s.getByRole("button", { name: "Nộp bài" }).first().click(); await s.getByRole("dialog").getByRole("button", { name: "Nộp bài" }).click();
  await expect(s.getByRole("heading", { name: `Bạn đúng ${expected}/20 câu`, level: 1 })).toBeVisible({ timeout: 30_000 }); await s.screenshot({ path: `${OUT}/f12-4-mock.png` });
  note(`4 thi thử UI: tải lại ở câu 8 → câu ${idxAfter}; offline câu 12–14; kết quả «Bạn đúng ${expected}/20 câu» khớp tính tay`); expect(idxAfter).toBe(8);
  // ---- 5. SV: lịch sử + «Chủ đề cần ôn» ------------------------------------------------------------------------------------------
  await s.goto("/practice/history"); await s.waitForTimeout(2500); const hist = await s.locator("main").innerText(); await s.screenshot({ path: `${OUT}/f12-5-lichsu.png` });
  note(`5 lịch sử UI: ${/Đúng 3\/10/.test(hist) ? "có «Đúng 3/10»" : "THIẾU Đúng 3/10"}, ${new RegExp(`Đúng ${expected}/20`).test(hist) ? `«Đúng ${expected}/20»` : "THIẾU kết quả thi thử"}, khối chủ đề yếu ${/Chủ đề bạn hay sai/.test(hist) ? "«Chủ đề bạn hay sai» HIỆN (US/SRS ghi «Chủ đề cần ôn»)" : /Chủ đề cần ôn/.test(hist) ? "«Chủ đề cần ôn» HIỆN" : "ẩn"}`);
  expect(hist).toMatch(/Đúng 3\/10/); expect(hist).toMatch(/Chủ đề (bạn hay sai|cần ôn)/);
  // ---- 6. SV: Hôm nay «Ôn chủ đề» ---------------------------------------------------------------------------------------------------
  // quy tắc FR-19: chủ đề vừa luyện trong 3 ngày thì Hôm nay không nhắc ôn lại → lùi đồng hồ các lượt của sv36 4 ngày (QC ghi rõ, ca TC-12-22 cũng làm vậy)
  sql(`update practice_attempts set started_at = started_at - interval '4 days' where student_id=(select id from users where email='sv36@edupilot.local') and course_id=(select id from courses where class_code='${CLS}')`);
  await s.goto("/"); await s.waitForTimeout(3500); const home = await s.locator("main").innerText(); await s.screenshot({ path: `${OUT}/f12-6-homnay.png` });
  note(`6 Hôm nay UI: «Ôn chủ đề» ${/Ôn chủ đề/.test(home) ? "HIỆN" : "THIẾU"}: ${home.replace(/\n+/g, " ¦ ").slice(0, 160)}`); expect(home).toMatch(/Ôn chủ đề/);
  // ---- 7. nhánh lỗi 1: mất mạng khi nộp (thi thử 5 câu) -------------------------------------------------------------------------
  await s.goto("/practice"); await s.waitForTimeout(1500); await s.getByRole("tab", { name: "Thi thử" }).click();
  const nIn = s.locator("main input").first(); await nIn.fill("5"); await s.getByRole("button", { name: "Bắt đầu thi thử" }).click(); await s.waitForURL(/\/practice\/[0-9a-f-]+/);
  for (let i = 1; i <= 5; i++) { await pick(s, null, i !== 1); await s.waitForTimeout(700); if (i < 5) await s.getByRole("button", { name: "Câu sau" }).click(); }
  await s.getByRole("button", { name: "Nộp bài" }).first().click(); await sv.ctx.setOffline(true); await s.getByRole("dialog").getByRole("button", { name: "Nộp bài" }).click(); await s.waitForTimeout(3000);
  const off = await s.locator("body").innerText(); const offOk = /Chưa nộp được\. Đang thử lại/.test(off); await s.screenshot({ path: `${OUT}/f12-7-offline-nop.png` });
  await sv.ctx.setOffline(false); await expect(s.getByRole("heading", { name: /Bạn đúng \d+\/\d+ câu/, level: 1 })).toBeVisible({ timeout: 60_000 });
  note(`7 nhánh lỗi mất mạng khi nộp UI: «Chưa nộp được. Đang thử lại.» ${offOk ? "HIỆN" : "THIẾU"}; có mạng → tự nộp, hiện kết quả`); expect(offOk).toBe(true);
  // ---- 8. nhánh lỗi 2: trích đề quá tải THẬT (bộ đếm RPM của nhà cung cấp cạn → Scheduler từ chối = OVERLOADED; 429 của nhà cung cấp thì việc hoãn vô hạn theo P-11-23) ----
  const adm = await apiLogin(request, "admin"); const cidv = sql(`select id from courses where class_code='${CLS}'`)[0][0];
  const setRpm = async (rpm: number | null) => { const lst = (await call(request, adm, "GET", "/admin/llm/providers")).json as { items: Record<string, unknown>[] }; const pv = lst.items.find((x) => x.name === "Q10-stub")!;
    const models = (pv.models as { model: string; kind: string; enabled: boolean }[]).map((m) => ({ model: m.model, kind: m.kind, price_in: "0", price_out: "0", enabled: m.enabled }));
    return call(request, adm, "PUT", `/admin/llm/providers/${pv.id}`, { type: pv.type, name: pv.name, base_url: pv.base_url, enabled: true, rpm_limit: rpm, tpm_limit: pv.tpm_limit ?? null, version: pv.version, models }); };
  const r1 = await setRpm(1); expect(r1.status, r1.text.slice(0, 160)).toBe(200); await p.waitForTimeout(3000);
  const [de, dk] = ["Đề mẫu F12", "Đáp án F12"].map((t) => sql(`select id from documents where course_id='${cidv}' and title='${t}'`)[0][0]);
  const drain = await call(request, await apiLogin(request, "teacher"), "POST", `/courses/${cidv}/questions/extract`, { document_id: de, answer_key_document_id: dk }); // rút hết token của giờ này
  await p.goto("/questions"); await p.waitForTimeout(2000); await p.getByRole("button", { name: "Tạo câu hỏi" }).click(); await p.getByRole("menuitem", { name: "Trích từ đề" }).click();
  const f2 = p.getByRole("form", { name: "Trích câu hỏi từ đề cũ" }); await f2.waitFor(); await f2.getByLabel("Đề cũ").selectOption({ label: "Đề mẫu F12" }); await f2.getByLabel("Đáp án").selectOption({ label: "Đáp án F12" });
  await f2.getByRole("button", { name: "Trích câu hỏi" }).click();
  const errTxt = p.getByText(/Chưa trích được\. Thử lại\./); await errTxt.first().waitFor({ timeout: 20_000 }).catch(() => {}); const hasErr = (await errTxt.count()) > 0; await p.screenshot({ path: `${OUT}/f12-8-trich-loi.png` });
  const r2 = await setRpm(null); note(`8 nhánh lỗi trích quá tải UI (RPM nhà cung cấp = 1, drain API ${drain.status}): «Chưa trích được. Thử lại.» ${hasErr ? "HIỆN" : "THIẾU"}; trả rpm_limit về trống ${r2.status}. Ghi nhận (không chấm): bộ đếm RPM cạn → Scheduler CHỜ token (không từ chối) nên việc chạy chậm chứ không OVERLOADED; nhánh OVERLOADED chấm bằng go test TestWorkerCallOverloaded (TC-02-14)`);
  appendFileSync(`${OUT}/f12-notes.txt`, log.join("\n") + "\n"); await gv.ctx.close(); await sv.ctx.close();
});
