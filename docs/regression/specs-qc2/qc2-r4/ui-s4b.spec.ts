import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { S, login, res, settle, text, shot, sql, TECH } from "./lib";
test.setTimeout(300_000);
const R = () => S.R.id as string;
test.describe("GV 1440", () => {
  test.use({ viewport: { width: 1440, height: 900 } });
  test("[P209-45] /class/settings", async ({ page }) => {
    await login(page, "teacher"); await page.goto(`/class/settings?course=${R()}`); await settle(page, 3500);
    const t0 = await text(page); await shot(page, "class-settings-1440");
    const code = sql(`select join_code from courses where id='${R()}'`);
    const big = t0.includes(code);
    await page.context().grantPermissions(["clipboard-read", "clipboard-write"], { origin: "https://localhost:1143" });
    await page.getByRole("button", { name: "Sao chép mã" }).click(); await page.waitForTimeout(800); const afterCopy = await text(page);
    const clipCode = await page.evaluate(() => navigator.clipboard.readText()).catch(() => "");
    const linkBtn = page.getByRole("button", { name: "Sao chép liên kết" }); const hasLink = await linkBtn.count(); let clipLink = ""; if (hasLink) { await linkBtn.click(); await page.waitForTimeout(800); clipLink = await page.evaluate(() => navigator.clipboard.readText()).catch(() => ""); }
    const toast = await page.locator("[data-part=toast],[class*=toast],[class*=Toast]").allInnerTexts();
    const cnt = Number(sql(`select count(*) from enrollments where course_id='${R()}' and role_in_course='STUDENT' and status='ACTIVE'`));
    const reg = page.getByRole("button", { name: "Tạo lại mã" }).first(); const regVariant = await reg.evaluate((b) => b.className + "|" + (b.getAttribute("data-variant") ?? ""));
    await reg.click(); await page.waitForTimeout(700); const dlg = (await page.getByRole("dialog").first().innerText().catch(() => "")).replace(/\s+/g, " "); await shot(page, "class-settings-regen-dialog");
    const later = await page.getByRole("dialog").getByRole("button", { name: "Để sau" }).count(); await page.keyboard.press("Escape"); await page.waitForTimeout(500);
    const codeAfterEsc = sql(`select join_code from courses where id='${R()}'`);
    const sw = await Promise.all(["Cho phép tham gia bằng mã", "Hết hạn", "Duyệt"].map(async (n) => [n, await page.getByText(n, { exact: false }).count()]));
    // TA và SV
    const ctxTA = await page.context().browser()!.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const pTA = await ctxTA.newPage(); await login(pTA, "ta"); await pTA.goto(`/class/settings?course=${R()}`); await settle(pTA, 3000);
    const taRegen = await pTA.getByRole("button", { name: "Tạo lại mã" }).count(); const taSw = await pTA.locator("input[role=switch],button[role=switch]").evaluateAll((els) => els.map((e) => (e as HTMLInputElement).disabled || e.getAttribute("aria-disabled") === "true")); const taTxt = await text(pTA);
    const ctxSV = await page.context().browser()!.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const pSV = await ctxSV.newPage(); await login(pSV, "sv.gioi"); await pSV.goto(`/class/settings?course=${R()}`); await settle(pSV, 3000); const svTxt = await text(pSV);
    res("TC-P209-45", big && clipCode === code && (!hasLink || clipLink.includes(`/join/${code}`)) && /Đã sao chép/.test(afterCopy) && toast.every((x) => !/sao chép/i.test(x)) && /Mã cũ sẽ ngừng hoạt động ngay/.test(dlg) && later === 1 && codeAfterEsc === code && taRegen === 0, `GV: mã lớn hiện «${code}»=${big}; Sao chép mã → bộ nhớ tạm «${clipCode}» (khớp mã=${clipCode === code}), «Đã sao chép» tại chỗ=${/Đã sao chép/.test(afterCopy)}; Sao chép liên kết (${hasLink} nút) → «${clipLink}» (toast: ${JSON.stringify(toast)}); «Tạo lại mã» (nút: ${regVariant.slice(0, 60)}) mở hộp «${dlg.slice(0, 190)}», «Để sau»=${later}, Esc không đổi mã=${codeAfterEsc === code}; mục cài đặt ${JSON.stringify(sw)}; ${cnt} SV đang học; TA: nút Tạo lại mã=${taRegen}, công tắc khoá=${JSON.stringify(taSw)}; SV vào /class/settings thấy «${svTxt.slice(0, 80)}»`);
    // 46: bàn phím + axe + từ kỹ thuật (GV 1440)
    await page.goto(`/class/settings?course=${R()}`); await settle(page, 3000);
    let reached = ""; let opened = false;
    for (let i = 0; i < 40; i++) { await page.keyboard.press("Tab"); const lab = await page.evaluate(() => (document.activeElement?.textContent || document.activeElement?.getAttribute("aria-label") || "").trim().slice(0, 30)); if (lab === "Tạo lại mã") { reached = lab; await page.keyboard.press("Enter"); await page.waitForTimeout(600); opened = (await page.getByRole("dialog").count()) > 0; await page.keyboard.press("Escape"); break; } }
    const axe = await new AxeBuilder({ page }).analyze(); const ser = axe.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? "")).map((v) => v.id);
    await page.setViewportSize({ width: 375, height: 812 }); await page.goto(`/class/settings?course=${R()}`); await settle(page, 3000); const ox = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth); const t375 = await text(page);
    res("TC-P209-46", reached === "Tạo lại mã" && opened && ser.length === 0 && ox === 0 && !TECH.test(t375), `bàn phím: Tab tới «Tạo lại mã» (${reached}) + Enter mở hộp xác nhận=${opened}, Esc đóng; axe serious/critical=${JSON.stringify(ser)} (tổng vi phạm ${axe.violations.length}); 375 px: tràn ngang ${ox}; từ kỹ thuật trong trang: ${TECH.test(t375) ? t375.match(TECH)![0] : "không"}`);
  });
  test("[P209-48] /class/members", async ({ page }) => {
    await login(page, "teacher"); await page.goto(`/class/members?course=${R()}`); await settle(page, 3500);
    const tab = page.getByRole("tab", { name: /Chờ duyệt/ }); const tabTxt = await tab.innerText().catch(() => ""); await tab.click(); await settle(page, 1500);
    const rows0 = await text(page); await shot(page, "members-pending-1440");
    const boxes = page.locator("tbody").getByRole("checkbox"); const nb = await boxes.count();
    const mismatchRow = /Email chưa khớp MSSV/.test(rows0);
    // chọn 3 hàng thường (không phải mismatch) rồi Duyệt 3
    let picked = 0; for (let i = 0; i < nb && picked < 3; i++) { const row = boxes.nth(i).locator("xpath=ancestor::tr[1]"); const rt = await row.innerText().catch(() => ""); if (/Email chưa khớp/.test(rt) || !rt.trim()) continue; await boxes.nth(i).check().catch(() => undefined); picked++; }
    const approveBtn = page.getByRole("button", { name: /^Duyệt \d+$/ }); const btnTxt = await approveBtn.innerText().catch(() => "");
    const pend0 = Number(sql(`select count(*) from enrollments where course_id='${R()}' and status='PENDING'`));
    await approveBtn.click(); await page.waitForTimeout(600); const after = await text(page); const undo = page.getByRole("button", { name: "Hoàn tác" }).first(); const hasUndo = await undo.count();
    const act1 = Number(sql(`select count(*) from enrollments where course_id='${R()}' and status='PENDING'`));
    await undo.click().catch(() => undefined); await page.waitForTimeout(1500); const pendBack = Number(sql(`select count(*) from enrollments where course_id='${R()}' and status='PENDING'`));
    res("TC-P209-48a", picked === 3 && /Duyệt 3/.test(btnTxt) && /Đã duyệt 3/.test(after) && hasUndo > 0 && pendBack === pend0, `tab «${tabTxt}»; chọn ${picked} hàng → «${btnTxt}»; sau bấm: «${after.slice(after.indexOf("Đã duyệt") , after.indexOf("Đã duyệt") + 60)}» có Hoàn tác=${hasUndo}; DB PENDING trước ${pend0} → sau duyệt ${act1} → sau Hoàn tác ${pendBack}`);
    // hàng EMAIL_MISMATCH
    const mm = page.getByText("Email chưa khớp MSSV").first(); const rowMM = mm.locator("xpath=ancestor::tr[1]"); const mmBtn = rowMM.getByRole("button", { name: /Duyệt/ }); const nMM = await mmBtn.count();
    let confirmTxt = ""; if (nMM) { await mmBtn.first().click(); await page.waitForTimeout(600); confirmTxt = await text(page); await page.getByRole("button", { name: "Để sau" }).first().click().catch(() => undefined); }
    res("TC-P209-48b", mismatchRow && nMM > 0 && /Duyệt .*|Để sau/.test(confirmTxt), `hàng nhãn «Email chưa khớp MSSV»=${mismatchRow}; nút Duyệt riêng ở hàng=${nMM}; xác nhận ngắn tại chỗ có «Duyệt <tên>» và «Để sau»=${/Để sau/.test(confirmTxt)}`);
    // mời ra + Hoàn tác (chỉ GV): tab Thành viên
    const tabAll = page.getByRole("tab", { name: /Đang học|Thành viên|Tất cả/ }).first(); await tabAll.click().catch(() => undefined); await settle(page, 1200);
    const menu = page.getByRole("button", { name: /^Thao tác cho / }).first(); const hasMenu = await menu.count(); let removed = "", undo2 = 0;
    if (hasMenu) { const nm = (await menu.getAttribute("aria-label"))!.replace("Thao tác cho ", ""); await menu.click(); await page.getByRole("menuitem", { name: "Mời ra khỏi lớp" }).click(); await page.waitForTimeout(700); const t = await text(page); removed = t.slice(t.indexOf("Đã mời"), t.indexOf("Đã mời") + 60); undo2 = await page.getByRole("button", { name: "Hoàn tác" }).count(); await page.getByRole("button", { name: "Hoàn tác" }).first().click().catch(() => undefined); await page.waitForTimeout(1200); }
    const ctxTA = await page.context().browser()!.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const pTA = await ctxTA.newPage(); await login(pTA, "ta"); await pTA.goto(`/class/members?course=${R()}`); await settle(pTA, 3000);
    const taMenu = await pTA.getByRole("button", { name: /^Thao tác cho / }).count(); const taMM = pTA.getByText("Email chưa khớp MSSV").first().locator("xpath=ancestor::tr[1]"); const taMMbtn = await taMM.getByRole("button", { name: /Duyệt/ }).count();
    res("TC-P209-48c", hasMenu > 0 && /Đã mời/.test(removed) && undo2 > 0 && taMenu === 0, `GV mời ra: «${removed}» + Hoàn tác=${undo2}; TA: nút Mời ra (menu thao tác)=${taMenu}, nút Duyệt ở hàng EMAIL_MISMATCH=${taMMbtn}`);
  });
});
test.describe("375", () => {
  test.use({ viewport: { width: 375, height: 812 }, hasTouch: true });
  test("[P209-49] /class/members 375", async ({ page }) => {
    await login(page, "teacher"); await page.goto(`/class/members?course=${R()}`); await settle(page, 3500); const ox = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    const small = await page.locator("button, a, [role=tab], input").evaluateAll((els) => els.filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && (r.height < 44 || r.width < 44) && getComputedStyle(e).visibility !== "hidden"; }).length);
    const axe = await new AxeBuilder({ page }).analyze(); const ser = axe.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? "")).map((v) => v.id);
    // bảng ảo hoá 200+ dòng: lớp 761987 hoặc lớp có nhiều SV
    const big = sql("select id from courses c where (select count(*) from enrollments e where e.course_id=c.id and e.role_in_course='STUDENT' and e.status='ACTIVE')>=200 order by 1 limit 1");
    let tr = -1, total = 0; if (big) { total = Number(sql(`select count(*) from enrollments e where e.course_id='${big}' and e.status='ACTIVE' and e.role_in_course='STUDENT'`)); await page.goto(`/class/members?course=${big}`); await settle(page, 5000); tr = await page.locator("tbody tr, [role=row]").count(); await shot(page, "members-200-375"); }
    let kb = ""; for (let i = 0; i < 25; i++) { await page.keyboard.press("Tab"); const l = await page.evaluate(() => document.activeElement?.tagName + ":" + (document.activeElement?.textContent || "").trim().slice(0, 16)); kb += l + " | "; }
    res("TC-P209-49", ox === 0 && ser.length === 0 && (tr >= 0 && tr < 80), `375 px: tràn ngang ${ox}; mục chạm < 44 px (đếm phần tử): ${small}; axe serious/critical ${JSON.stringify(ser)}; lớp ${total} SV đang học → số hàng trong DOM = ${tr} (< 80 ⇒ ảo hoá); Tab đi qua: ${kb.slice(0, 200)}`);
  });
});
