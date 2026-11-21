import { test, expect } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { writeFileSync } from "node:fs";
import AxeBuilder from "@axe-core/playwright";
import { S, login, res, settle, text, shot, sql, TECH } from "./lib";
test.setTimeout(300_000);
test.describe("GV 1440", () => {
  test.use({ viewport: { width: 1440, height: 900 } });
  test("[P210-15] GV thấy cảnh báo + Hôm nay", async ({ page }) => {
    await login(page, "teacher"); await page.goto(`/class/members?course=${S.R.id}`); await settle(page, 3000);
    await page.getByRole("tab", { name: /Chờ duyệt/ }).click(); await settle(page, 1200); const mem = await text(page); await shot(page, "p210-15-members");
    const rows = await page.getByText("Email chưa khớp MSSV").count();
    await page.goto(`/?course=${S.R.id}`); await settle(page, 3000); let t = await text(page);
    if (!t.includes(S.R.class_code)) { const pk = page.getByRole("button", { name: /^Chọn lớp, đang xem/ }); await pk.click(); await page.getByRole("menuitem", { name: new RegExp(S.R.class_code) }).click(); await settle(page, 3000); t = await text(page); }
    await shot(page, "p210-15-today");
    const act = await page.locator("main li, main [role=listitem]").filter({ hasText: /chưa khớp/i }).allInnerTexts();
    res("TC-P210-15", rows >= 1 && act.length >= 1, `GV tab Chờ duyệt lớp ${S.R.class_code}: ${rows} hàng nhãn «Email chưa khớp MSSV»; «Hôm nay» (chọn lớp ${S.R.class_code}) có ${act.length} mục về email chưa khớp: «${(act[0] ?? "").replace(/\s+/g, " ").slice(0, 150)}» (nguồn việc F14 EMAIL_MISMATCH)`);
  });
  test("[P210-32] Dùng lại nội dung từ lớp khác", async ({ page }) => {
    await login(page, "teacher"); await page.goto(`/class/settings?course=${S.O.id}`); await settle(page, 3500);
    const sec = page.getByRole("heading", { name: "Dùng lại nội dung từ lớp khác" }); const has = await sec.count(); await shot(page, "p210-32-settings");
    // nguồn đã có tài liệu: 761987
    const row = page.locator("li, tr, [role=listitem]").filter({ hasText: "761987" }).filter({ has: page.getByRole("button", { name: "Dùng lại tài liệu" }) }).first(); const hasRow = await row.count();
    let msg1 = ""; if (hasRow) { await row.getByRole("button", { name: "Dùng lại tài liệu" }).click(); await page.waitForTimeout(2500); msg1 = (await text(page)).match(/Đã dùng lại[^.]*\./)?.[0] ?? ""; }
    const toast = await page.locator("[data-part=toast],[class*=toast],[class*=Toast]").allInnerTexts();
    // nguồn không tài liệu: lớp RI (mới, rỗng) làm nguồn cho lớp R
    await page.goto(`/class/settings?course=${S.R.id}`); await settle(page, 3500);
    const rowE = page.locator("li, tr, [role=listitem]").filter({ hasText: S.RI.class_code }).filter({ has: page.getByRole("button", { name: "Dùng lại tài liệu" }) }).last(); const hasE = await rowE.count(); let msg2 = "";
    if (hasE) { await rowE.getByRole("button", { name: "Dùng lại tài liệu" }).click(); await page.waitForTimeout(2500); msg2 = (await text(page)).match(new RegExp(`Lớp ${S.RI.class_code} chưa có tài liệu để dùng lại\\.`))?.[0] ?? ""; }
    // GV khác học phần: lớp QCOTH duy nhất
    await page.goto(`/class/settings?course=${S.OTH2.id}`); await settle(page, 3500); const secOther = await page.getByRole("heading", { name: "Dùng lại nội dung từ lớp khác" }).count();
    res("TC-P210-32", has === 1 && /Đã dùng lại \d+ tài liệu từ lớp 761987\./.test(msg1) && /chưa có tài liệu để dùng lại/.test(msg2) && secOther === 0 && toast.every((x) => !/dùng lại/i.test(x)), `lớp cùng học phần: mục «Dùng lại nội dung từ lớp khác» hiện=${has}; bấm nguồn 761987 → tại chỗ «${msg1}»; nguồn chưa có tài liệu (${S.RI.class_code}) → «${msg2}»; toast=${JSON.stringify(toast)}; GV có lớp học phần duy nhất QCOTH → mục hiện=${secOther} (ẩn)`);
  });
  test("[P210-34] Nhập danh sách (UI)", async ({ page, context }) => {
    const FX = "../docs/regression/evidence-4/fx"; const T = { timeout: 4000 };
    await login(page, "teacher"); await page.goto(`/class/members?course=${S.RI.id}`); await settle(page, 3500);
    await page.getByRole("tab", { name: "Nhập danh sách" }).click(); await settle(page, 1000); await shot(page, "p210-34-import-tab");
    const inModal = await page.getByRole("dialog").count();
    const invitesDefault = await page.getByLabel("Gửi thư mời cho sinh viên chưa có tài khoản").isChecked().catch(() => false);
    const reqs: Array<{ q: string; key: string }> = []; page.on("request", (r) => { if (/roster\/import/.test(r.url()) && r.method() === "POST") reqs.push({ q: r.url().split("?")[1] ?? "", key: r.headers()["idempotency-key"] ?? "" }); });
    const file = page.locator("input[type=file]"); const preview = () => page.getByRole("button", { name: "Xem trước" });
    await file.setInputFiles(`${FX}/loi-2.csv`); await preview().click(T); await page.waitForTimeout(3000); const pv = await text(page); await shot(page, "p210-34-preview");
    const btn = await page.getByRole("button", { name: /^Nhập \d+ sinh viên$/ }).innerText().catch(() => "");
    const prim = await page.locator("main button").evaluateAll((bs) => bs.filter((b) => b.getAttribute("data-variant") === "primary" && (b as HTMLElement).offsetParent !== null).length);
    const dry = reqs.filter((r) => /dry_run=true/.test(r.q)).length;
    // tệp ELF đuôi .csv
    await file.setInputFiles(`${FX}/fake-ext.csv`); await preview().click(T).catch(() => undefined); await page.waitForTimeout(2500); const fake = await text(page);
    const fakeAlert = (await page.getByRole("alert").allInnerTexts()).join(" | ");
    // tệp 3 MiB
    writeFileSync(`${FX}/big3m.csv`, "Email,Họ và tên,MSSV\n" + "a@b.vn,x,1\n".repeat(280000));
    await file.setInputFiles(`${FX}/big3m.csv`); await page.waitForTimeout(1500); const big = await text(page); const bigAlert = (await page.getByRole("alert").allInnerTexts()).join(" | ");
    // nhập thật loi-2.csv
    await file.setInputFiles(`${FX}/loi-2.csv`); await preview().click(T); await page.waitForTimeout(3000);
    await page.getByRole("button", { name: /^Nhập \d+ sinh viên$/ }).click(T); await page.waitForTimeout(4000); const done = await text(page); await shot(page, "p210-34-done");
    const n1 = sql(`select count(*) from enrollments where course_id='${S.RI.id}' and role_in_course='STUDENT'`);
    // ngoại tuyến giữa chừng
    const before = reqs.length;
    await file.setInputFiles(`${FX}/ok-30.csv`); await preview().click(T); await page.waitForTimeout(3000);
    await context.setOffline(true); await page.getByRole("button", { name: /^Nhập \d+ sinh viên$/ }).click(T); await page.waitForTimeout(3000);
    const off = await text(page); const fileKept = (await page.getByRole("button", { name: /^Nhập \d+ sinh viên$/ }).count()) > 0 || /ok-30\.csv/.test(off); const resend = page.getByRole("button", { name: "Gửi lại" }); const hasResend = await resend.count(); await shot(page, "p210-34-offline");
    await context.setOffline(false); await page.waitForTimeout(800);
    await resend.click(T); await page.waitForTimeout(5000); const after = await text(page);
    const posts = reqs.slice(before).filter((r) => !/dry_run/.test(r.q)); const keys = posts.map((r) => r.key); const sameKey = keys.length >= 2 && keys[0] === keys[keys.length - 1] && keys[0] !== "";
    const n2 = sql(`select count(*) from enrollments where course_id='${S.RI.id}' and role_in_course='STUDENT'`);
    // TA: không có tab
    const ctxTA = await page.context().browser()!.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const pTA = await ctxTA.newPage(); await login(pTA, "ta"); await pTA.goto(`/class/members?course=${S.R.id}`); await settle(pTA, 3000);
    const taTab = await pTA.getByRole("tab", { name: "Nhập danh sách" }).count(); const taTabs = (await pTA.getByRole("tab").allInnerTexts()).join("/").replace(/\s+/g, " "); await ctxTA.close();
    const ok = inModal === 0 && invitesDefault && /Dòng 7/.test(pv) && /Dòng 19/.test(pv) && /Email không đúng dạng/.test(pv) && btn === "Nhập 28 sinh viên" && prim === 1 && dry >= 1 && /MiB|2 MB/.test(bigAlert + big) && n1 === "28" && hasResend >= 1 && sameKey && n2 === "58" && taTab === 0;
    res("TC-P210-34", ok, `tab «Nhập danh sách» tại chỗ (dialog=${inModal}); tuỳ chọn mời mặc định bật=${invitesDefault}; Xem trước loi-2.csv gọi ${dry} POST dry_run=true; bảng có «Dòng 7»=${/Dòng 7/.test(pv)}, «Email không đúng dạng»=${/Email không đúng dạng/.test(pv)}, «Dòng 19»=${/Dòng 19/.test(pv)}; nút «${btn}»; nút chính hiện=${prim}; ELF .csv → alert «${fakeAlert.slice(0, 90)}»; 3 MiB → «${bigAlert.slice(0, 60)}»; nhập thật → «${done.slice(done.search(/Đã nhập/), done.search(/Đã nhập/) + 60)}» (DB ${n1}/28); ngoại tuyến → «${off.slice(off.search(/Mất kết nối/), off.search(/Mất kết nối/) + 70)}», nút Gửi lại=${hasResend}, tệp giữ=${fileKept}; Gửi lại: Idempotency-Key lần 1 và lần gửi lại giống nhau=${sameKey} (${keys.map((k) => k.slice(0, 8)).join(",")}), kết quả «${after.slice(after.search(/Đã nhập/), after.search(/Đã nhập/) + 40)}», sổ SV ${n2} (28+30=58, không trùng); TA (lớp R): tab Nhập danh sách=${taTab} (các tab: ${taTabs})`);
  });
});
