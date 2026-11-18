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
    const toast = await page.locator("[role=status],[class*=toast]").allInnerTexts();
    // nguồn không tài liệu: lớp RI (mới, rỗng) làm nguồn cho lớp R
    await page.goto(`/class/settings?course=${S.R.id}`); await settle(page, 3500);
    const rowE = page.locator("li, tr, [role=listitem]").filter({ hasText: S.RI.class_code }).filter({ has: page.getByRole("button", { name: "Dùng lại tài liệu" }) }).first(); const hasE = await rowE.count(); let msg2 = "";
    if (hasE) { await rowE.getByRole("button", { name: "Dùng lại tài liệu" }).click(); await page.waitForTimeout(2500); msg2 = (await text(page)).match(/Lớp [^.]*chưa có tài liệu để dùng lại\./)?.[0] ?? ""; }
    // GV khác học phần: lớp QCOTH duy nhất
    await page.goto(`/class/settings?course=${S.OTH.id}`); await settle(page, 3500); const secOther = await page.getByRole("heading", { name: "Dùng lại nội dung từ lớp khác" }).count();
    res("TC-P210-32", has === 1 && /Đã dùng lại \d+ tài liệu từ lớp 761987\./.test(msg1) && /chưa có tài liệu để dùng lại/.test(msg2) && secOther === 0 && toast.every((x) => !/dùng lại/i.test(x)), `lớp cùng học phần: mục «Dùng lại nội dung từ lớp khác» hiện=${has}; bấm nguồn 761987 → tại chỗ «${msg1}»; nguồn chưa có tài liệu (${S.RI.class_code}) → «${msg2}»; toast=${JSON.stringify(toast)}; GV có lớp học phần duy nhất QCOTH → mục hiện=${secOther} (ẩn)`);
  });
  test("[P210-34] Nhập danh sách (UI)", async ({ page, context }) => {
    await login(page, "teacher"); await page.goto(`/class/members?course=${S.RI.id}`); await settle(page, 3500);
    await page.getByRole("tab", { name: "Nhập danh sách" }).click(); await settle(page, 1000); await shot(page, "p210-34-import-tab");
    const inModal = await page.getByRole("dialog").count();
    const invitesDefault = await page.getByLabel("Gửi thư mời cho sinh viên chưa có tài khoản").isChecked();
    const reqs: string[] = []; page.on("request", (r) => { if (/roster\/import/.test(r.url())) reqs.push(r.method() + " " + r.url().split("?")[1]); });
    await page.locator("input[type=file]").setInputFiles("/tmp/qc2-b/fx/loi-2.csv"); await page.waitForTimeout(500);
    await page.getByRole("button", { name: "Xem trước" }).click(); await page.waitForTimeout(3000); const pv = await text(page); await shot(page, "p210-34-preview");
    const btn = await page.getByRole("button", { name: /^Nhập \d+ sinh viên$/ }).innerText().catch(() => "");
    // tệp giả đuôi + 3 MiB
    await page.locator("input[type=file]").setInputFiles("/tmp/qc2-b/fx/fake-ext.csv"); await page.getByRole("button", { name: "Xem trước" }).click().catch(() => undefined); await page.waitForTimeout(2500); const fake = await text(page);
    writeFileSync("/tmp/qc2-b/fx/big3m.csv", "Email,Họ và tên,MSSV\n" + "a@b.vn,x,1\n".repeat(280000));
    await page.locator("input[type=file]").setInputFiles("/tmp/qc2-b/fx/big3m.csv"); await page.getByRole("button", { name: "Xem trước" }).click().catch(() => undefined); await page.waitForTimeout(2500); const big = await text(page);
    // nhập thật loi-2.csv
    await page.locator("input[type=file]").setInputFiles("/tmp/qc2-b/fx/loi-2.csv"); await page.getByRole("button", { name: "Xem trước" }).click(); await page.waitForTimeout(2500);
    await page.getByRole("button", { name: /^Nhập \d+ sinh viên$/ }).click(); await page.waitForTimeout(4000); const done = await text(page); await shot(page, "p210-34-done");
    const n = sql(`select count(*) from enrollments where course_id='${S.RI.id}' and role_in_course='STUDENT'`);
    res("TC-P210-34", inModal === 0 && invitesDefault && /Dòng 7/.test(pv) && /Dòng 19/.test(pv) && btn === "Nhập 28 sinh viên" && n === "28", `tab «Nhập danh sách» tại chỗ (dialog=${inModal}); tuỳ chọn mời mặc định bật=${invitesDefault}; Xem trước loi-2.csv gọi ${JSON.stringify(reqs.slice(0, 2))}: «Dòng 7»=${/Dòng 7/.test(pv)}, «Dòng 19»=${/Dòng 19/.test(pv)}; nút «${btn}»; tệp ELF đuôi .csv → «${fake.slice(fake.search(/không|Không|định dạng|hỗ trợ/), fake.search(/không|Không|định dạng|hỗ trợ/) + 90)}»; tệp 3 MiB → «${big.slice(big.search(/quá|lớn|2 MiB|MiB/), big.search(/quá|lớn|2 MiB|MiB/) + 90)}»; nhập thật → «${done.slice(done.search(/Đã|đã nhập|Nhập xong/), done.search(/Đã|đã nhập|Nhập xong/) + 100)}»; DB ${n} SV (28)`);
    // ngoại tuyến giữa chừng
    const f = "/tmp/qc2-b/fx/ok-30.csv"; await page.locator("input[type=file]").setInputFiles(f).catch(() => undefined);
  });
});
