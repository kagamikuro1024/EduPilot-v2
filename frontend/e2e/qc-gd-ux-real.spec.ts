import { test, expect, type Browser, type Page } from "@playwright/test";
import { execSync } from "child_process";
import { mkdirSync } from "fs";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");
// QC 10.6 — kiểm lại danh mục UX S2-1…3, a…o (P-106-13) trên lớp 761987 (không KC106). Pha 1: DRAFT=0; pha 2: có DRAFT.
const PW = process.env.QC_PW ?? "";
const BASE = process.env.QC_BASE ?? "http://localhost:580";
const OUT = process.env.QC_OUT ?? "/tmp/qc-ux";
const BT = process.env.QC_BT03 ?? "";
mkdirSync(OUT, { recursive: true });
const info = (k: string, v: unknown) => console.log(`QCINFO ${k}: ${typeof v === "string" ? v : JSON.stringify(v)}`);
const sql = (q: string) => execSync(`docker exec ep-qc-postgres-1 psql -U edupilot -d edupilot -Atc "${q.replace(/"/g, '\\"')}"`).toString().trim();

async function login(browser: Browser, user: string, w = 1440, h = 900) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto(`${BASE}/login`);
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  const tb = page.locator("[data-part=topbar]");
  if (!(await tb.innerText()).includes("761987")) {
    await tb.locator("button").filter({ hasText: /\d{3,}|KC106|QC/ }).first().click();
    await page.getByText("761987", { exact: false }).first().click();
    await page.waitForTimeout(1500);
  }
  return { ctx, page };
}
const mainText = async (p: Page) => (await p.locator("main").innerText()).replace(/\n+/g, " | ");

test("pha1 S2-1/S2-2/S2-3/d/j/l/n — hàng chờ, tab Bài tập, Công bố", async ({ browser }, ti) => {
  test.skip(ti.project.name !== "desktop");
  test.setTimeout(150_000);
  const { ctx, page } = await login(browser, "teacher");
  page.setDefaultTimeout(6000);
  await page.goto(`${BASE}/grading`); await page.waitForTimeout(4000);
  const sels = () => page.locator("main select");
  const optIn = async (word: string) => { const n = await sels().count(); for (let i = 0; i < n; i++) if ((await sels().nth(i).locator("option", { hasText: word }).count()) > 0) return sels().nth(i); return null; };
  const stSel = await optIn("Chưa duyệt");
  info("S2-1 lọc trạng thái lúc mở (DRAFT=0, còn bài chưa chấm)", { chon: stSel ? await stSel.evaluate((e) => (e as HTMLSelectElement).selectedOptions[0].text) : null });
  let t = await mainText(page);
  info("S2-1 «Chưa có bài nộp» có không", t.includes("Chưa có bài nộp"));
  info("S2-3 dòng tổng", (t.match(/\d+\+? bài/g) ?? []).slice(0, 2));
  // chọn BT03 + lọc «Chưa duyệt» (rỗng) → «Chấm tất cả» phải bấm được
  const btSel = await optIn("Bài tập 03");
  await btSel?.selectOption({ index: 1 }).catch(() => {});
  const bt03 = await btSel?.locator("option", { hasText: "Bài tập 03" }).first().getAttribute("value");
  if (bt03) await btSel!.selectOption(bt03);
  await stSel?.selectOption({ label: "Chưa duyệt" });
  await page.waitForTimeout(1500);
  const cta = page.getByRole("button", { name: /Chấm tất cả/ }).first();
  info("S2-1 «Chấm tất cả» khi đã chọn BT03 + xem «Chưa duyệt» (rỗng)", { khoa: await cta.isDisabled(), danhSach: ((await mainText(page)).match(/Chưa có bài nộp[^|]*/) ?? [""])[0] });
  await page.screenshot({ path: `${OUT}/s21-chua-duyet-bt03.png` });
  // d: giữ bộ lọc khi rời / vào lại
  const chip = page.getByRole("button", { name: /Cần xem kỹ/ }).first();
  const state = async () => (await chip.getAttribute("aria-pressed")) ?? (await chip.getAttribute("aria-checked")) ?? "?";
  const before = await state();
  const keepBT = await btSel!.inputValue();
  await page.goto(`${BASE}/`); await page.waitForTimeout(1500);
  await page.goto(`${BASE}/grading`); await page.waitForTimeout(2500);
  const bt2 = await optIn("Bài tập 03");
  info("d bộ lọc sau khi rời / vào lại", { canXemKy: { truoc: before, sau: await state() }, baiTapGiu: (await bt2?.inputValue()) === keepBT, trangThai: await (await optIn("Chưa duyệt"))?.evaluate((e) => (e as HTMLSelectElement).selectedOptions[0].text) });
  // S2-2 / j
  await page.getByRole("tab", { name: /Bài tập/ }).first().click(); await page.waitForTimeout(2500);
  t = await mainText(page);
  info("S2-2 tab Bài tập (đầu bảng)", t.slice(t.indexOf("Tiêu đề"), t.indexOf("Tiêu đề") + 330));
  info("S2-2 SQL bài MATCHED mọi nguồn BT03", sql(`select count(*) from submissions where assignment_id='${BT}' and match_status='MATCHED'`));
  const link = page.locator("main a[href*='open=']").first();
  info("j liên kết thật", { soLienKet: await page.locator("main a[href*='open=']").count(), href: await link.getAttribute("href") });
  const popup = await Promise.all([ctx.waitForEvent("page", { timeout: 4000 }).catch(() => null), link.click({ button: "middle" }).catch(() => null)]).then((r) => r[0]);
  info("j chuột giữa mở tab mới", { tabMoi: !!popup, url: popup ? popup.url() : null });
  if (popup) await popup.close();
  await link.click(); await page.waitForTimeout(1500);
  info("j bấm thường: mở ngăn, URL", { url: page.url().replace(BASE, ""), ngan: await page.locator("[data-part=assignment-drawer], [role=dialog]").count() });
  await page.screenshot({ path: `${OUT}/j-tab-bai-tap.png` });
  // l / n
  await page.goto(`${BASE}/grading`); await page.waitForTimeout(2500);
  const pub = page.getByRole("button", { name: /^Công bố/ }).first();
  info("l «Công bố» khi chưa chọn", { khoa: await pub.isDisabled(), title: await pub.getAttribute("title"), aria: await pub.getAttribute("aria-describedby") });
  await (await optIn("Đã duyệt"))?.selectOption({ label: "Đã duyệt" });
  await page.waitForTimeout(1500);
  const allBox = page.getByText(/Chọn tất cả/).first();
  info("l ô «Chọn tất cả»", await allBox.count());
  await allBox.click().catch(() => {});
  await page.waitForTimeout(600);
  info("l «Công bố» sau khi chọn", { khoa: await pub.isDisabled() });
  await pub.click().catch(() => {});
  await page.waitForTimeout(1200);
  const dlg = page.getByRole("dialog").or(page.getByRole("alertdialog")).first();
  info("n hộp Công bố — nguyên văn", ((await dlg.innerText().catch(() => "")) || "").replace(/\n+/g, " | "));
  await page.screenshot({ path: `${OUT}/n-hop-cong-bo.png` });
  const later = page.getByRole("button", { name: /Để sau|Huỷ|Đóng/ }).first();
  info("n nút huỷ", await later.innerText().catch(() => null));
  await later.click().catch(() => {});
  await ctx.close();
});

test("pha2 a/b/c — Duyệt ngay sau gõ nhận xét, tự sang bài kế, bộ đếm", async ({ browser }, ti) => {
  test.skip(ti.project.name !== "desktop");
  test.setTimeout(280_000);
  const { ctx, page } = await login(browser, "teacher");
  page.setDefaultTimeout(5000);
  const ids = sql(`select r.id from grading_results r join submissions s on s.course_id=r.course_id and s.id=r.submission_id where s.assignment_id='${BT}' and r.status='DRAFT' order by r.id`).split("\n").filter(Boolean);
  info("DRAFT trước khi duyệt", ids.length);
  await page.goto(`${BASE}/grading`); await page.waitForTimeout(3000);
  await page.getByRole("combobox").filter({ has: page.locator("option", { hasText: "Chưa duyệt" }) }).first().selectOption({ label: "Chưa duyệt" }).catch(() => {});
  await page.waitForTimeout(1500);
  await page.locator("main a[aria-label^='Duyệt bài của']").first().click();
  await page.waitForURL(/\/grading\//); await page.waitForTimeout(3000);
  for (let i = 0; i < Math.min(ids.length, 3); i++) {
    const url = page.url();
    const prog = await page.locator("[data-part=review-progress]").innerText().catch(() => "");
    const note = `QC nhận xét ${i} ${Date.now()}`;
    const ta = page.getByLabel("Nhận xét cho sinh viên").first();
    await ta.fill(note);
    const t0 = Date.now();
    await page.getByRole("button", { name: /^Duyệt bài$/ }).click(); // ngay sau khi gõ, không blur chờ
    await page.waitForTimeout(3500);
    const sid = url.match(/grading\/([0-9a-f-]{36})/)?.[1] ?? "";
    const st = sql(`select status from grading_results where submission_id='${sid}' order by round desc limit 1`);
    const saved = sql(`select count(*) from grading_results where submission_id='${sid}' and criteria::text like '%${note.slice(0, 12)}%'`);
    info(`a lượt ${i}: bộ đếm trước`, prog);
    info(`a lượt ${i}: duyệt ngay lần bấm đầu?`, { statusDB: st, nhanXetLuu: saved, ms: Date.now() - t0, urlSauDoi: page.url() !== url });
    const from = await page.locator("[data-part=review-from]").innerText({ timeout: 2000 }).catch(() => "");
    info(`b lượt ${i}: chỉ báo bài trước`, { tuSang: page.url() !== url, from, quayLai: await page.getByRole("link", { name: /Quay lại/ }).count() });
    info(`c lượt ${i}: bộ đếm sau`, await page.locator("[data-part=review-progress]").innerText().catch(() => "—"));
    await page.screenshot({ path: `${OUT}/a-b-c-luot-${i}.png` });
  }
  await ctx.close();
});

test("pha1b l/n/C2 — Công bố và lọc bài tập qua đổi tab", async ({ browser }, ti) => {
  test.skip(ti.project.name !== "desktop");
  test.setTimeout(150_000);
  const { ctx, page } = await login(browser, "teacher");
  page.setDefaultTimeout(6000);
  await page.goto(`${BASE}/grading`); await page.waitForTimeout(3500);
  const sels = () => page.locator("main select");
  const optIn = async (word: string) => { const n = await sels().count(); for (let i = 0; i < n; i++) if ((await sels().nth(i).locator("option", { hasText: word }).count()) > 0) return sels().nth(i); return null; };
  const pub = page.getByRole("button", { name: /^Công bố/ }).first();
  const did = await pub.getAttribute("aria-describedby");
  info("l lý do khoá (aria-describedby)", did ? await page.locator(`[id="${did}"]`).innerText().catch(() => "(không đọc được)") : null);
  const btSel = await optIn("Bài tập 03");
  const v = await btSel!.locator("option", { hasText: "Bài tập 03" }).first().getAttribute("value");
  await btSel!.selectOption(v!);
  await (await optIn("Đã duyệt"))!.selectOption({ label: "Đã duyệt" });
  await page.waitForTimeout(2000);
  const boxes = page.locator("main input[type=checkbox]");
  info("l số checkbox", await boxes.count());
  info("l mẫu nhãn checkbox", await boxes.evaluateAll((els) => els.slice(0, 3).map((e) => (e as HTMLInputElement).getAttribute("aria-label") ?? (e.parentElement?.textContent ?? "").trim())));
  if (await boxes.count()) await boxes.first().check().catch(async () => { await boxes.first().click(); });
  await page.waitForTimeout(600);
  info("l «Công bố» sau khi chọn tất cả", { khoa: await pub.isDisabled(), nhan: await pub.innerText() });
  await pub.click().catch((e) => info("l click lỗi", String(e).slice(0, 100)));
  await page.waitForTimeout(1500);
  const dlg = page.getByRole("dialog").or(page.getByRole("alertdialog")).first();
  info("n hộp Công bố — nguyên văn", ((await dlg.innerText().catch(() => "")) || "").replace(/\n+/g, " | "));
  info("n nút trong hộp", await dlg.getByRole("button").allInnerTexts().catch(() => []));
  await page.screenshot({ path: `${OUT}/n-hop-cong-bo.png` });
  await page.keyboard.press("Escape");
  await page.waitForTimeout(500);
  // C2: lọc bài tập sau khi đổi tab rồi quay lại
  const sel1 = await optIn("Bài tập 03");
  const before = await sel1!.inputValue();
  await page.getByRole("tab", { name: /Bài tập/ }).first().click(); await page.waitForTimeout(1500);
  await page.getByRole("tab", { name: /Hàng chờ/ }).first().click(); await page.waitForTimeout(1500);
  const after = await (await optIn("Bài tập 03"))!.inputValue();
  info("C2 lọc «Bài tập» sau đổi tab rồi quay lại", { giuNguyen: before === after, truoc: before ? "BT03" : "", sau: after ? "có" : "(mặc định)" });
  await ctx.close();
});

test("pha2a h/i/m/k/n — ngăn lưu ý, Rút lưu ý, Chấm lại, hộp Công bố bản cũ", async ({ browser }, ti) => {
  test.skip(ti.project.name !== "desktop");
  test.setTimeout(420_000);
  const { ctx, page } = await login(browser, "teacher");
  page.setDefaultTimeout(8000);
  await page.goto(`${BASE}/grading?tab=assignments`); await page.waitForTimeout(3000);
  await page.locator("main a[href*='open=']").nth(1).click().catch(async () => { await page.getByText("Bài tập 03").first().click(); });
  await page.waitForTimeout(3000);
  const panel = page.locator("[data-part=guidance-panel]").first();
  const pt = (await panel.innerText().catch(() => "")).replace(/\n+/g, " | ");
  info("h/i ngăn «Lưu ý chấm»", pt.slice(0, 520));
  info("h/i nhãn", { doiDiem: /đổi điểm \d+ ?%/.test(pt), baiDoiDiem: /\d+ bài đổi điểm/.test(pt), BanVDangDung: /Bản v\d+ \(đang dùng\)/.test(pt), sua24: /sửa \d+ ?%/.test(pt), baiDaDuyet: /\d+ bài đã duyệt/.test(pt) });
  await panel.screenshot({ path: `${OUT}/hi-ngan-luu-y.png` }).catch(() => {});
  // m: Rút lưu ý
  const t0 = Date.now();
  await panel.getByRole("button", { name: /Rút lưu ý/ }).click();
  let msg = "";
  for (let i = 0; i < 60; i++) { await page.waitForTimeout(2000); const x = (await panel.innerText()).replace(/\n+/g, " | "); const m = x.match(/Đã rút[^|]*|Không có lưu ý mới[^|]*|\d+ lưu ý bị loại/g); if (m) { msg = m.join(" ; "); break; } }
  info("m sau Rút lưu ý", { giay: Math.round((Date.now() - t0) / 1000), thongBao: msg });
  await page.screenshot({ path: `${OUT}/m-rut-luu-y.png` });
  await page.keyboard.press("Escape");
  // k: Chấm lại (ở hàng chờ)
  await page.goto(`${BASE}/grading`); await page.waitForTimeout(3000);
  const sels = () => page.locator("main select");
  const n = await sels().count(); for (let i = 0; i < n; i++) { const o = sels().nth(i).locator("option", { hasText: "Bài tập 03" }); if (await o.count()) { await sels().nth(i).selectOption(await o.first().getAttribute("value") as string); break; } }
  await page.waitForTimeout(2000);
  const q = await mainText(page);
  info("k dòng «bản cũ»", (q.match(/[^|]*bản cũ[^|]*/) ?? [""])[0].trim());
  const rg = page.getByRole("button", { name: /^Chấm lại/ }).first();
  if (await rg.count()) {
    const t1 = Date.now(); await rg.click();
    const seen: string[] = []; let done = "";
    for (let i = 0; i < 100; i++) {
      await page.waitForTimeout(3000);
      const x = (await mainText(page));
      const prog = (x.match(/Đang chấm[^|]*\d+\/\d+[^|]*|Đã xếp[^|]*/) ?? [""])[0].trim();
      if (prog && !seen.includes(prog)) seen.push(prog);
      const dn = (x.match(/Đã chấm lại[^|]*|Chấm lại xong[^|]*|đã chấm lại[^|]*/) ?? [""])[0].trim();
      if (dn) { done = dn; break; }
      if (!/bản cũ/.test(x) && i > 3) { done = "(dòng bản cũ biến mất, không thấy dòng báo xong)"; break; }
    }
    info("k Chấm lại — tiến độ thấy được", seen.slice(0, 6));
    info("k Chấm lại — báo xong", { giay: Math.round((Date.now() - t1) / 1000), done });
    await page.screenshot({ path: `${OUT}/k-cham-lai.png` });
  } else info("k", "không có nút «Chấm lại»");
  // n: hộp Công bố với bài đã duyệt bản cũ
  const st = await (async () => { const nn = await sels().count(); for (let i = 0; i < nn; i++) if (await sels().nth(i).locator("option", { hasText: "Chưa duyệt" }).count()) return sels().nth(i); return null; })();
  await st?.selectOption({ label: "Đã duyệt" }); await page.waitForTimeout(2000);
  await page.locator("main input[type=checkbox]").first().check().catch(() => {});
  await page.getByRole("button", { name: /^Công bố/ }).first().click().catch(() => {});
  await page.waitForTimeout(1500);
  const dlg = page.getByRole("dialog").or(page.getByRole("alertdialog")).first();
  info("n hộp Công bố (có bài bản cũ) — nguyên văn", ((await dlg.innerText().catch(() => "")) || "").replace(/\n+/g, " | "));
  info("n nút đóng (aria-label)", await dlg.locator("button").first().getAttribute("aria-label").catch(() => null));
  await page.screenshot({ path: `${OUT}/n-hop-cong-bo-ban-cu.png` });
  await page.keyboard.press("Escape");
  await ctx.close();
});

test("P-106-14 — cookie ep_rt cũ/hết hạn, mở /login, đăng nhập ngay ⇒ vào được, không bị đẩy về", async ({ browser }, ti) => {
  test.skip(ti.project.name !== "desktop");
  test.setTimeout(240_000);
  const host = new URL(BASE).hostname;
  const results: unknown[] = [];
  for (const [label, delayMs, loops] of [["không trễ", 0, 6], ["refresh trễ 1,5 s (đua có chủ ý)", 1500, 4], ["refresh trễ 4 s", 4000, 2]] as const) {
    let okN = 0;
    for (let i = 0; i < loops; i++) {
      const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 }, locale: "vi-VN" });
      await ctx.addCookies([{ name: "ep_rt", value: "cookie-cu-het-han-" + i, domain: host, path: "/api/v1/auth", httpOnly: true, secure: true, sameSite: "Lax", expires: Math.floor(Date.now() / 1000) + 3600 }]);
      const page = await ctx.newPage();
      if (delayMs) await page.route("**/auth/refresh", async (r) => { await new Promise((res) => setTimeout(res, delayMs)); await r.continue(); });
      await page.goto(`${BASE}/login`, { waitUntil: "domcontentloaded" });
      await page.locator("input[type=email]").fill("teacher@edupilot.local");
      await page.locator("input[type=password]").fill(PW);
      await page.keyboard.press("Enter");
      const left = await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"), { timeout: 15000 }).then(() => true).catch(() => false);
      await page.waitForTimeout(delayMs + 2500); // chờ lượt refresh cũ trả lời
      const url = new URL(page.url());
      const stay = left && !url.pathname.startsWith("/login");
      const topbar = await page.locator("[data-part=topbar]").count();
      if (stay && topbar) okN++;
      else await page.screenshot({ path: `${OUT}/p14-fail-${delayMs}-${i}.png` });
      await ctx.close();
    }
    results.push({ kichBan: label, vaoDuocVaOLai: `${okN}/${loops}` });
  }
  info("P-106-14 kết quả", results);
});
