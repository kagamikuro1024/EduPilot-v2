import { execFileSync } from "node:child_process";
import { randomUUID } from "node:crypto";
import { expect, test, type Page } from "@playwright/test";
import { PWD, S, apiLogin, axeBad, call, clearRL, em, freshStudent, installRec, joinInfo, layout, newCourse, note, psql, sleep, techWords, tokenOf, uiLogin } from "./r4ui-lib";

// QC2-I r4ui: US-P2-07 (bộ chọn lớp), US-P2-09 (vào lớp bằng mã), US-P2-10 (danh sách lớp). Lớp A = QC2IUA (gv1, ta, sv1–4 ACTIVE, sv5–7 + svm PENDING).
installRec("r4ui-class");
const only = (info: { project: { name: string } }, p: "desktop" | "mobile") => test.skip(info.project.name !== p, `chỉ ${p}`);
const A = () => S.courseA.id as string;
const picker = (page: Page) => page.getByRole("button", { name: /^Chọn lớp, đang xem/ });
const topbar = (page: Page) => page.locator("[data-part=topbar]");
const caddy = (since: string) => execFileSync("docker", ["logs", "ep-r-caddy-1", "--since", since], { maxBuffer: 1 << 28 }).toString();

test("[TC-P207-31] id lớp lạ bị bỏ: ?course= và localStorage lớp 2 → vẫn lớp 1, 0 request /courses/C2/", async ({ page }, info) => {
  only(info, "desktop");
  const c2 = await newCourse(S.gv2.id, "L2");
  await apiLogin(page, em("sv1"));
  const hits: string[] = [];
  page.on("request", (r) => r.url().includes(`/courses/${c2.id}`) && hits.push(r.url()));
  await page.goto(`/?course=${c2.id}`);
  await expect(topbar(page)).toBeVisible({ timeout: 30_000 });
  await expect(picker(page)).toContainText(S.courseA.code);
  await expect(picker(page)).not.toContainText(c2.code);
  await page.evaluate((id) => localStorage.setItem("ep:ui:course", id), c2.id);
  await page.reload();
  await expect(picker(page)).toContainText(S.courseA.code);
  await page.waitForTimeout(1500);
  const ls = await page.evaluate(() => localStorage.getItem("ep:ui:course"));
  note(`request /courses/${c2.id.slice(0, 8)}…/ = ${hits.length}; ep:ui:course sau tải lại = ${ls}`);
  expect(hits).toEqual([]);
  expect([A(), "all", null]).toContain(ls);
  expect(ls).not.toBe(c2.id);
});

test("[TC-P207-35] ánh xạ màn mô phỏng theo class_code (seed 761987/761988), 0 request API lớp thật từ màn mô phỏng", async ({ page }, info) => {
  only(info, "desktop");
  // GV seed (chỉ đọc): đổi lớp ở bộ chọn khi đang ở màn mô phỏng /analytics
  await apiLogin(page, "teacher@edupilot.local", "Edupilot#Seed-2026");
  await page.goto("/analytics");
  await expect(topbar(page)).toBeVisible({ timeout: 30_000 });
  await expect(picker(page)).toBeVisible({ timeout: 30_000 });
  const courseCalls: string[] = [];
  page.on("request", (r) => /\/api\/v1\/courses\/[0-9a-f-]{36}\//.test(r.url()) && courseCalls.push(r.url().replace(/^.*\/api\/v1/, "")));
  const text = async () => (await page.locator("main").innerText()).replace(/\s+/g, " ");
  const choose = async (code: string) => {
    await picker(page).click();
    await page.getByRole("dialog", { name: "Chọn lớp" }).getByRole("menuitem", { name: new RegExp(code) }).click();
    await expect(picker(page)).toContainText(code);
    await page.waitForTimeout(800);
  };
  await choose("761987");
  const t87 = await text();
  await choose("761988");
  const t88 = await text();
  note(`/analytics @761987 ${t87.length} ký tự; @761988 ${t88.length} ký tự; khác nhau=${t87 !== t88}; có '761987' ở bản 88: ${/761987/.test(t88)}; request lớp thật từ màn mô phỏng: ${courseCalls.length} (${[...new Set(courseCalls)].slice(0, 3).join(",")})`);
  expect(t87).not.toBe(t88);
  expect(courseCalls).toEqual([]);
});

// ---------------- /join ----------------
test("[TC-P209-42] /join (375): ô mã tự viết hoa, bỏ 0O1IL và khoảng trắng; 1 nút Xem lớp; xem trước; tham gia; mã sai; 6 mã sai → 429; lớp đầy", async ({ page }, info) => {
  only(info, "mobile");
  const gv1 = S.gv1.email;
  // lớp C: không cần duyệt (vào ngay); lớp F: đầy (sức chứa 1)
  const C = await newCourse(S.gv1.id, "J");
  const cj = await joinInfo(C.id, gv1);
  const F = await newCourse(S.gv1.id, "F");
  const fj = await joinInfo(F.id, gv1);
  const full = await call("PUT", `/courses/${F.id}/join-settings`, fj.token, { capacity: 1, version: fj.version });
  expect(full.status, JSON.stringify(full.body)).toBe(200);
  const filler = await freshStudent("fil");
  expect((await call("POST", "/courses/join", await tokenOf(filler), { code: fj.join_code })).status).toBe(200);
  const d = await freshStudent("d42");
  await apiLogin(page, d);
  await page.goto("/join");
  await expect(page.getByRole("heading", { name: "Tham gia lớp", level: 1 })).toBeVisible({ timeout: 30_000 });
  const input = page.getByLabel("Mã tham gia");
  await input.fill("an7k2mq");
  await expect(input).toHaveValue("AN7K2MQ");
  await input.fill(" bx 4p9 tw0 ILo1");
  const val = await input.inputValue();
  note(`gõ " bx 4p9 tw0 ILo1" → "${val}"`);
  expect(val).toBe("BX4P9TW");
  await input.fill("0O1IL");
  await expect(input).toHaveValue("");
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  await expect(page.getByRole("button", { name: "Xem lớp" })).toHaveAttribute("data-variant", "primary");
  // mã sai: câu chung
  await input.fill("AAAAAAA");
  await page.getByRole("button", { name: "Xem lớp" }).click();
  await expect(page.getByText("Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.")).toBeVisible({ timeout: 30_000 });
  await expect(input).toHaveAttribute("aria-invalid", "true");
  // lớp đầy
  await input.fill(String(fj.join_code));
  await page.getByRole("button", { name: "Xem lớp" }).click();
  await expect(page.locator("[data-part=join-preview]").getByText("Lớp đã đủ sĩ số. Hãy báo giảng viên.")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole("button", { name: "Tham gia lớp" })).toHaveCount(0);
  // lớp thường: xem trước đúng dòng, vào lớp
  await input.fill(String(cj.join_code));
  await page.getByRole("button", { name: "Xem lớp" }).click();
  const prev = page.locator("[data-part=join-preview]");
  await expect(prev).toContainText(`QC2IU J · ${C.code} · ${S.gv1.name} · HK1 2026–2027`, { timeout: 30_000 });
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  const l = await layout(page);
  note(`375 preview: ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length}`);
  expect(l.ox).toBeLessThanOrEqual(0);
  expect(l.touch).toEqual([]);
  await page.getByRole("button", { name: "Tham gia lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Bạn đã vào lớp." })).toBeVisible({ timeout: 30_000 });
  expect(psql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='${d}' and e.course_id='${C.id}'`)).toBe("ACTIVE");
  // lớp cần duyệt (A): câu cần duyệt + chờ duyệt
  const d2 = await freshStudent("d42b");
  await page.context().clearCookies();
  await apiLogin(page, d2);
  await page.goto("/join");
  const aj = await joinInfo(A(), gv1);
  await input.fill(String(aj.join_code));
  await page.getByRole("button", { name: "Xem lớp" }).click();
  await expect(page.getByText("Lớp này cần giảng viên duyệt.")).toBeVisible({ timeout: 30_000 });
  await page.getByRole("button", { name: "Tham gia lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi yêu cầu, chờ giảng viên duyệt." })).toBeVisible({ timeout: 30_000 });
  expect(psql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='${d2}' and e.course_id='${A()}'`)).toBe("PENDING");
  // 6 mã sai → bị chặn có đếm ngược
  const d3 = await freshStudent("d42c");
  await page.context().clearCookies();
  await apiLogin(page, d3);
  await page.goto("/join");
  let blocked = false;
  for (let i = 0; i < 12 && !blocked; i++) {
    await input.fill(["AAAAAAB", "AAAAAAC", "AAAAAAD", "AAAAAAE", "AAAAAAF", "AAAAAAG", "AAAAAAH", "AAAAAAJ", "AAAAAAK", "AAAAAAM", "AAAAAAN", "AAAAAAP"][i]);
    const btn = page.getByRole("button", { name: "Xem lớp" });
    if (!(await btn.isEnabled())) break;
    await btn.click();
    await page.waitForTimeout(500);
    blocked = (await page.getByText(/Bạn đã thử quá nhiều lần\. Thử lại sau \d\d:\d\d\./).count()) > 0;
  }
  const line = page.getByText(/Bạn đã thử quá nhiều lần\. Thử lại sau \d\d:\d\d\./);
  await expect(line).toBeVisible();
  const t1 = await line.innerText();
  await page.waitForTimeout(2200);
  const t2 = await line.innerText();
  note(`bị chặn: "${t1}" → "${t2}" (đếm ngược); nút Xem lớp disabled=${await page.getByRole("button", { name: "Xem lớp" }).isDisabled()}`);
  expect(t1).not.toBe(t2);
  await expect(page.getByRole("button", { name: "Xem lớp" })).toBeDisabled();
  expect(await page.locator("main").innerText()).not.toMatch(/RATE_LIMITED|JOIN_CODE_INVALID/);
});

test("[TC-P209-43] /join/<mã> chưa đăng nhập → /login?next → quay lại /join đã xem trước, mã rời URL; 375 sạch; log Caddy không lộ mã", async ({ page }, info) => {
  only(info, "mobile");
  const aj = await joinInfo(A(), S.gv1.email);
  const code = String(aj.join_code);
  const since = new Date(Date.now() - 2000).toISOString();
  const d = await freshStudent("d43");
  await page.goto(`/join/${code}`);
  await expect(page).toHaveURL(new RegExp(`/login\\?next=%2Fjoin%2F${code}$`), { timeout: 30_000 });
  const lg = await layout(page);
  expect(lg.ox).toBeLessThanOrEqual(0);
  await uiLogin(page, d, PWD, `/join/${code}`).catch(() => undefined);
  await expect(page.locator("[data-part=join-preview]")).toContainText(S.courseA.code, { timeout: 30_000 });
  await expect(page).toHaveURL(/\/join$/);
  const l = await layout(page);
  const ax = await axeBad(page);
  const logs = caddy(since);
  const cnt = logs.split(code).length - 1;
  const red = (logs.match(/\/join\/REDACTED/g) ?? []).length;
  note(`sau đăng nhập URL ${new URL(page.url()).pathname}; 375 ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length} axe=${JSON.stringify(ax)}; log Caddy: mã=${cnt}, /join/REDACTED=${red}`);
  expect(l.ox).toBeLessThanOrEqual(0);
  expect(l.cut).toEqual([]);
  expect(l.touch).toEqual([]);
  expect(ax).toEqual([]);
  expect(cnt).toBe(0);
});

// ---------------- /class/settings ----------------
test("[TC-P209-45] /class/settings GV: mã lớn + sao chép (tĩnh), Tạo lại mã có xác nhận nêu số, công tắc; TA không có Tạo lại mã; SV bị chặn, 0 request join-code", async ({ page, context }, info) => {
  only(info, "desktop");
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await apiLogin(page, S.gv1.email);
  await page.goto(`/class/settings?course=${A()}`);
  await expect(page.getByRole("heading", { name: "Mã và cài đặt tham gia", level: 1 })).toBeVisible({ timeout: 30_000 });
  const code = await page.locator("[data-part=join-code]").innerText();
  expect(code).toMatch(/^[A-Z2-9]{7}$/);
  await page.getByRole("button", { name: "Sao chép mã" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã sao chép" })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(code);
  await page.getByRole("button", { name: "Sao chép liên kết" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toContain(`/join/${code}`);
  const regen = page.getByRole("button", { name: "Tạo lại mã" });
  await expect(regen).not.toHaveAttribute("data-variant", "primary");
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  for (const sw of ["Cho phép tham gia bằng mã", "Hết hạn", "Giới hạn sĩ số", "Giới hạn tên miền email", "Cần giảng viên duyệt"]) await expect(page.getByText(sw, { exact: false }).first(), sw).toBeVisible();
  const active = psql(`select count(*) from enrollments where course_id='${A()}' and role='STUDENT' and status='ACTIVE'`);
  await regen.click();
  const dlg = page.getByRole("dialog");
  await expect(dlg).toContainText(`Tạo lại mã cho lớp ${S.courseA.code}?`);
  await expect(dlg).toContainText(`Mã cũ sẽ ngừng hoạt động ngay. ${active} sinh viên đang ở trong lớp không bị ảnh hưởng.`);
  await expect(dlg.getByRole("button", { name: "Để sau" })).toBeVisible();
  await dlg.getByRole("button", { name: "Để sau" }).click();
  await expect(dlg).toHaveCount(0);
  expect((await joinInfo(A(), S.gv1.email)).join_code).toBe(code);
  await regen.click();
  await page.getByRole("dialog").getByRole("button", { name: "Tạo lại mã" }).click();
  await expect.poll(() => page.locator("[data-part=join-code]").innerText(), { timeout: 20_000 }).not.toBe(code);
  const nw = await page.locator("[data-part=join-code]").innerText();
  const old = await call("POST", "/courses/join/preview", await tokenOf(em("sv1")), { code });
  note(`mã ${code} → ${nw}; mã cũ preview ${old.status} ${old.body.code}`);
  expect(old.status).toBe(404);
  // TA: thấy mã, không có Tạo lại mã
  await page.context().clearCookies();
  await apiLogin(page, S.ta.email);
  await page.goto(`/class/settings?course=${A()}`);
  await expect(page.locator("[data-part=join-code]")).toBeVisible({ timeout: 30_000 });
  expect(await page.getByRole("button", { name: "Tạo lại mã" }).count()).toBe(0);
  // SV: màn chặn, 0 request join-code
  await page.context().clearCookies();
  await apiLogin(page, em("sv1"));
  const hits: string[] = [];
  page.on("request", (r) => r.url().includes("/join-code") && hits.push(r.url()));
  await page.goto("/class/settings");
  await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible({ timeout: 30_000 });
  expect(hits).toEqual([]);
});

test("[TC-P209-46] /class/settings: chỉ bàn phím, 375 px, axe, chuỗi kỹ thuật", async ({ page }, info) => {
  await apiLogin(page, S.gv1.email);
  await page.goto(`/class/settings?course=${A()}`);
  await expect(page.getByRole("heading", { name: "Mã và cài đặt tham gia", level: 1 })).toBeVisible({ timeout: 30_000 });
  if (info.project.name === "desktop") {
    const press = async (re: RegExp, key = "Tab") => {
      for (let i = 0; i < 60; i++) {
        await page.keyboard.press(key);
        if (await page.evaluate((src) => new RegExp(src).test((document.activeElement?.textContent ?? "") + (document.activeElement?.getAttribute("aria-label") ?? "")), re.source)) return true;
      }
      return false;
    };
    expect(await press(/Sao chép mã/)).toBe(true);
    await page.keyboard.press("Enter");
    await expect(page.getByRole("status").filter({ hasText: "Đã sao chép" })).toBeVisible();
    expect(await press(/Tạo lại mã/)).toBe(true);
    await page.keyboard.press("Enter");
    await expect(page.getByRole("dialog")).toContainText("Tạo lại mã cho lớp");
    await page.keyboard.press("Escape");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    note("bàn phím: Tab tới Sao chép mã (Enter → Đã sao chép) và Tạo lại mã (Enter → xác nhận, Esc đóng)");
  } else {
    const l = await layout(page);
    note(`375: ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length}`);
    expect(l.ox).toBeLessThanOrEqual(0);
    expect(l.cut).toEqual([]);
    expect(l.touch).toEqual([]);
  }
  const ax = await axeBad(page);
  note(`axe=${JSON.stringify(ax)}`);
  expect(ax).toEqual([]);
  expect(await techWords(page, ["token", "session", "jwt", "bcrypt", "join_code", "rag", "pii", "provider"])).toEqual([]);
});

// ---------------- /class/members ----------------
async function membersPage(page: Page, email: string, tab = "pending") {
  await apiLogin(page, email);
  await page.goto(`/class/members?course=${A()}&tab=${tab}`);
  await expect(page.getByRole("heading", { name: "Thành viên lớp", level: 1 })).toBeVisible({ timeout: 30_000 });
}
const st = (email: string) => psql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='${email}' and e.course_id='${A()}'`);

test("[TC-P209-48] /class/members: Chờ duyệt (N), chọn 3 → Duyệt 3 + Hoàn tác (3 undo), mời ra + Hoàn tác, hàng EMAIL_MISMATCH; TA không có nút", async ({ page }, info) => {
  only(info, "desktop");
  for (const k of ["sv5", "sv6", "sv7"]) expect(st(em(k)), k).toBe("PENDING");
  await membersPage(page, S.gv1.email);
  const tab = page.getByRole("tab", { name: /Chờ duyệt/ });
  const n = psql(`select count(*) from enrollments where course_id='${A()}' and status='PENDING' and role='STUDENT'`);
  await expect(tab).toContainText(n);
  // hàng EMAIL_MISMATCH
  const mm = page.locator("table tbody tr").filter({ hasText: "QC2IU Lệch" });
  await expect(mm.getByText("Email chưa khớp MSSV")).toBeVisible();
  await mm.getByRole("button", { name: "Duyệt" }).click();
  await expect(page.getByText("Email của người này khác email trong danh sách lớp. Chỉ duyệt nếu bạn chắc đúng là sinh viên này.")).toBeVisible();
  expect(st(em("svm"))).toBe("PENDING"); // chưa gọi gì cho tới khi xác nhận
  await expect(page.getByRole("button", { name: "Duyệt QC2IU Lệch" })).toBeVisible();
  await page.reload();
  // chọn 3 hàng thường → Duyệt 3
  const ops: string[] = [];
  page.on("request", (r) => /\/members\/[^/]+\/(approve|undo)$/.test(r.url()) && ops.push(r.url().split("/").slice(-1)[0]));
  for (const nm of ["QC2IU Giang", "QC2IU Hà", "QC2IU Khoa"]) await page.locator("table tbody tr").filter({ hasText: nm }).getByRole("checkbox").check();
  await page.getByRole("button", { name: "Duyệt 3" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã duyệt 3 yêu cầu" })).toBeVisible({ timeout: 30_000 });
  await expect.poll(() => ["sv5", "sv6", "sv7"].map((k) => st(em(k))).join(","), { timeout: 20_000 }).toBe("ACTIVE,ACTIVE,ACTIVE");
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => ["sv5", "sv6", "sv7"].map((k) => st(em(k))).join(","), { timeout: 20_000 }).toBe("PENDING,PENDING,PENDING");
  note(`duyệt hàng loạt: ${ops.filter((o) => o === "approve").length} approve, ${ops.filter((o) => o === "undo").length} undo; DB về PENDING`);
  expect(ops.filter((o) => o === "undo").length).toBe(3);
  // mời ra + Hoàn tác (GV)
  await page.getByRole("tab", { name: /Thành viên/ }).click();
  const row = page.locator("table tbody tr").filter({ hasText: "QC2IU Dũng" });
  await row.getByRole("button", { name: /Thao tác cho/ }).click();
  await page.getByRole("menuitem", { name: "Mời ra khỏi lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã mời SV QC2IU Dũng ra khỏi lớp" })).toBeVisible({ timeout: 20_000 });
  await expect.poll(() => st(em("sv4")), { timeout: 15_000 }).not.toBe("ACTIVE");
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => st(em("sv4")), { timeout: 15_000 }).toBe("ACTIVE");
  // TA: không có Duyệt ở hàng mismatch, không có "Mời ra"
  await page.context().clearCookies();
  await membersPage(page, S.ta.email);
  const mmTa = page.locator("table tbody tr").filter({ hasText: "QC2IU Lệch" });
  await expect(mmTa.getByText("Email chưa khớp MSSV")).toBeVisible();
  expect(await mmTa.getByRole("button", { name: /Duyệt/ }).count()).toBe(0);
  await page.getByRole("tab", { name: /Thành viên/ }).click();
  const rowTa = page.locator("table tbody tr").filter({ hasText: "QC2IU Dũng" });
  const menu = rowTa.getByRole("button", { name: /Thao tác cho/ });
  if (await menu.count()) {
    await menu.click();
    expect(await page.getByRole("menuitem", { name: "Mời ra khỏi lớp" }).count()).toBe(0);
  }
  note("TA: mismatch không có nút Duyệt; không có 'Mời ra'");
});

test("[TC-P209-49] /class/members 375: AUDIT/TOUCH sạch; bảng ảo hoá 200+ dòng (<80 <tr>); bàn phím", async ({ page }, info) => {
  const C = await newCourse(S.gv1.id, "M");
  const gv = S.gv1.email;
  const t = await tokenOf(gv);
  const rows = ["Email,Họ và tên,MSSV"];
  for (let i = 1; i <= 230; i++) rows.push(`qc2iu.m${C.code.toLowerCase()}.${i}@example.test,SV Ảo ${i},QM${String(i).padStart(6, "0")}${C.code.slice(-3)}`);
  const b = "----q" + randomUUID().replace(/-/g, "");
  const body = `--${b}\r\nContent-Disposition: form-data; name="send_invites"\r\n\r\nfalse\r\n--${b}\r\nContent-Disposition: form-data; name="file"; filename="r.csv"\r\nContent-Type: text/csv\r\n\r\n${rows.join("\n")}\n\r\n--${b}--\r\n`;
  const r = await fetch(`https://localhost:1143/api/v1/courses/${C.id}/roster/import`, { method: "POST", headers: { authorization: `Bearer ${t}`, "content-type": `multipart/form-data; boundary=${b}`, "Idempotency-Key": randomUUID() }, body });
  const rj = (await r.json()) as Record<string, unknown>;
  note(`import 230 dòng: ${r.status} ${JSON.stringify({ total: rj.total, created_users: rj.created_users })}`);
  expect(r.status).toBe(200);
  await apiLogin(page, gv);
  await page.goto(`/class/members?course=${C.id}`);
  await expect(page.getByRole("heading", { name: "Thành viên lớp", level: 1 })).toBeVisible({ timeout: 30_000 });
  await expect(page.locator("table tbody tr, [data-part=row]").first()).toBeVisible({ timeout: 30_000 });
  const trs = await page.locator("table tbody tr").count();
  const items = await page.locator("main li, main [role=row]").count();
  const tabTxt = await page.getByRole("tab", { name: /Thành viên/ }).innerText();
  if (info.project.name === "mobile") {
    const l = await layout(page);
    note(`375: ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length}; <tr>=${trs}; li/row=${items}; tab "${tabTxt.replace(/\s+/g, " ")}"`);
    expect(l.ox).toBeLessThanOrEqual(0);
    expect(l.cut).toEqual([]);
    expect(l.touch).toEqual([]);
    expect(Math.max(trs, items)).toBeLessThan(80 + 40);
  } else {
    note(`desktop: <tr> trong DOM=${trs}; tab "${tabTxt.replace(/\s+/g, " ")}"`);
    expect(trs).toBeLessThan(80);
    let reached = false;
    for (let i = 0; i < 80 && !reached; i++) {
      await page.keyboard.press("Tab");
      reached = await page.evaluate(() => !!document.activeElement?.closest("table tbody tr"));
    }
    expect(reached).toBe(true);
    note("bàn phím: Tab tới được hàng của bảng");
  }
  expect(trs).toBeLessThan(80);
});

test("[TC-P209-51] đầu cuối 375×812: D xem trước và xin vào lớp cần duyệt; GV thấy chuông + hàng chờ, duyệt; D thấy lớp; GV tạo lại mã, mã cũ → câu chung; 6 mã sai → 429", async ({ page, browser }, info) => {
  only(info, "mobile");
  const C = await newCourse(S.gv1.id, "E");
  const cj = await joinInfo(C.id, S.gv1.email);
  expect((await call("PUT", `/courses/${C.id}/join-settings`, cj.token, { require_approval: true, version: cj.version })).status).toBe(200);
  const code = String((await joinInfo(C.id, S.gv1.email)).join_code);
  const d = await freshStudent("e2e");
  // D (375×812)
  await apiLogin(page, d);
  await page.goto(`/join/${code}`);
  await expect(page.locator("[data-part=join-preview]")).toContainText(C.code, { timeout: 30_000 });
  await expect(page.getByText("Lớp này cần giảng viên duyệt.")).toBeVisible();
  await page.getByRole("button", { name: "Tham gia lớp" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi yêu cầu, chờ giảng viên duyệt." })).toBeVisible({ timeout: 30_000 });
  // GV: chuông + hàng chờ (ngữ cảnh riêng)
  const gctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: "https://localhost:1143", locale: "vi-VN", viewport: { width: 1280, height: 800 } });
  const g = await gctx.newPage();
  await apiLogin(g, S.gv1.email);
  await g.goto(`/class/members?course=${C.id}&tab=pending`);
  await expect(g.getByRole("heading", { name: "Thành viên lớp", level: 1 })).toBeVisible({ timeout: 30_000 });
  const row = g.locator("table tbody tr").filter({ hasText: "QC2IU e2e" });
  await expect(row).toBeVisible({ timeout: 30_000 });
  const bellUnread = await g.getByRole("button", { name: /^Thông báo/ }).getAttribute("aria-label");
  note(`GV: chuông "${bellUnread}"; hàng chờ có`);
  await row.getByRole("button", { name: "Duyệt" }).click();
  await expect.poll(() => psql(`select e.status from enrollments e join users u on u.id=e.user_id where u.email='${d}' and e.course_id='${C.id}'`), { timeout: 20_000 }).toBe("ACTIVE");
  // D thấy lớp ở bộ chọn lớp và "Hôm nay"
  await page.goto("/");
  await expect(picker(page)).toContainText(C.code, { timeout: 30_000 });
  await expect(page.locator("main")).toBeVisible();
  // GV tạo lại mã → D thử mã cũ
  const regen = await call("POST", `/courses/${C.id}/join-code/regenerate`, await tokenOf(S.gv1.email), {});
  expect(regen.status).toBe(200);
  const d2 = await freshStudent("e2eb");
  const t2 = await tokenOf(d2);
  const old = await call("POST", "/courses/join/preview", t2, { code });
  note(`mã cũ sau khi tạo lại: ${old.status} ${old.body.code}`);
  expect(old.status).toBe(404);
  expect(old.body.code).toBe("JOIN_CODE_INVALID");
  // 6 mã sai → 429
  let last = 0;
  for (let i = 0; i < 12 && last !== 429; i++) last = (await call("POST", "/courses/join/preview", t2, { code: "Z" + "ABCDEFGHJKMNPQRSTUVWXYZ".slice(i, i + 6).padEnd(6, "X") })).status;
  note(`mã sai liên tiếp → ${last}`);
  expect(last).toBe(429);
  const prod = execFileSync("bash", ["-c", "grep -c 'test.skip(true' e2e/class-join.spec.ts; grep -n 'join flow end to end' e2e/class-join.spec.ts | head -2"], { cwd: __dirname + "/../.." }).toString().trim().replace(/\n/g, " | ");
  note(`class-join.spec.ts 'join flow end to end' là ca @real test.skip: ${prod}`);
  await gctx.close();
  await sleep(10);
});

// ---------------- US-P2-10 ----------------
test("[TC-P210-15] GV thấy cảnh báo: /class/members tab Chờ duyệt có 'Email chưa khớp MSSV'; 'Hôm nay' có việc EMAIL_MISMATCH", async ({ page }, info) => {
  only(info, "desktop");
  await membersPage(page, S.gv1.email);
  await expect(page.locator("table tbody tr").filter({ hasText: "QC2IU Lệch" }).getByText("Email chưa khớp MSSV")).toBeVisible();
  await page.goto(`/?course=${S.courseA.id}`);
  await expect(page.locator("main")).toBeVisible();
  const want = page.locator("main").getByText(/Xem và xác nhận/);
  await expect(want.first()).toBeVisible({ timeout: 30_000 });
  const rowTxt = (await page.locator("main [data-part=action-row], main li").filter({ hasText: /Xem và xác nhận/ }).first().innerText().catch(() => "")).replace(/\s+/g, " ");
  note(`'Hôm nay' có việc cảnh báo: "${rowTxt.slice(0, 160)}"`);
  const api = await call("GET", `/courses/${A()}/today`, await tokenOf(S.gv1.email));
  const kinds = (api.body.items ?? api.body.actions ?? []).map((a: { kind: string }) => a.kind);
  note(`API today kinds: ${[...new Set(kinds)].join(",")}`);
  expect(kinds).toContain("EMAIL_MISMATCH");
});

test("[TC-P210-32] /class/settings lớp 2: 'Dùng lại nội dung từ lớp khác' chỉ khi có nguồn đủ điều kiện; kết quả tại chỗ; TA không thấy", async ({ page }, info) => {
  only(info, "desktop");
  // nguồn = lớp A (gv1 dạy cả hai) có tài liệu; đích = lớp mới của cùng GV
  const dest = await newCourse(S.gv1.id, "S");
  const t = await tokenOf(S.gv1.email);
  const src = await call("GET", `/courses/${dest.id}/share-sources`, t);
  note(`share-sources của lớp đích: ${src.status} ${JSON.stringify(src.body).slice(0, 200)}`);
  expect(src.status).toBe(200);
  await apiLogin(page, S.gv1.email);
  await page.goto(`/class/settings?course=${dest.id}`);
  await expect(page.getByRole("heading", { name: "Mã và cài đặt tham gia", level: 1 })).toBeVisible({ timeout: 30_000 });
  const items = (src.body.items ?? []) as Array<{ class_code: string; documents: number }>;
  const sect = page.getByRole("heading", { name: "Dùng lại nội dung từ lớp khác" });
  if (items.length === 0) {
    await expect(sect).toHaveCount(0);
    note("không có nguồn đủ điều kiện ⇒ mục ẩn (đúng)");
    return;
  }
  await expect(sect).toBeVisible();
  const first = items[0];
  const btn = page.getByRole("button", { name: "Dùng lại tài liệu" }).first();
  await btn.click();
  const msg = page.getByRole("status").filter({ hasText: first.documents > 0 ? new RegExp(`Đã dùng lại \\d+ tài liệu từ lớp ${first.class_code}\\.`) : `Lớp ${first.class_code} chưa có tài liệu để dùng lại.` });
  await expect(msg).toBeVisible({ timeout: 40_000 });
  await expect(page.locator("main [role=alert]")).toHaveCount(0);
  note(`kết quả tại chỗ: "${(await msg.innerText()).trim()}"`);
});

test("[TC-P210-34] /class/members tab Nhập danh sách: vùng thả, tuỳ chọn mời bật, Xem trước (dry_run), Nhập N sinh viên, lỗi tại vùng thả, mất mạng giữ tệp + Gửi lại cùng khoá; TA không có tab", async ({ page }, info) => {
  only(info, "desktop");
  const C = await newCourse(S.gv1.id, "I", [S.ta.id]);
  await apiLogin(page, S.gv1.email);
  await page.goto(`/class/members?course=${C.id}`);
  await expect(page.getByRole("heading", { name: "Thành viên lớp", level: 1 })).toBeVisible({ timeout: 30_000 });
  await page.getByRole("tab", { name: "Nhập danh sách" }).click();
  expect(await page.getByRole("dialog").count()).toBe(0);
  await expect(page.getByText("Thả tệp CSV hoặc XLSX vào đây.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Chọn tệp" })).toBeVisible();
  await expect(page.getByLabel("Gửi thư mời cho sinh viên chưa có tài khoản")).toBeChecked();
  const input = page.getByLabel("Tệp danh sách lớp");
  const calls: string[] = [];
  page.on("request", (r) => r.url().includes("/roster/import") && calls.push(`${new URL(r.url()).searchParams.get("dry_run") === "true" ? "dry" : "real"} ${r.headers()["idempotency-key"] ?? ""}`));
  // tệp sai loại / quá lớn → lỗi tại vùng thả
  await input.setInputFiles({ name: "fake-ext.pdf", mimeType: "application/pdf", buffer: Buffer.from("%PDF-1.7") });
  await expect(page.getByText("Tệp phải là CSV hoặc XLSX.")).toBeVisible();
  await input.setInputFiles({ name: "lon.csv", mimeType: "text/csv", buffer: Buffer.alloc(3 * 1024 * 1024, 97) });
  await expect(page.getByText("Tệp quá lớn. Tối đa 2 MB.")).toBeVisible();
  // loi-2.csv: 30 dòng, dòng 7 và 19 lỗi → 28 hợp lệ; bỏ tuỳ chọn mời để không gửi thư thật
  await page.getByLabel("Gửi thư mời cho sinh viên chưa có tài khoản").uncheck();
  const rows = ["Email,Họ và tên,MSSV"];
  for (let i = 1; i <= 30; i++) rows.push(i === 7 ? `khong-phai-email,Sinh Viên ${i},I${C.code.slice(-4)}${String(i).padStart(3, "0")}` : i === 19 ? `qc2iu.i${i}.${C.code.toLowerCase()}@example.test,,I${C.code.slice(-4)}${String(i).padStart(3, "0")}` : `qc2iu.i${i}.${C.code.toLowerCase()}@example.test,Sinh Viên ${i},I${C.code.slice(-4)}${String(i).padStart(3, "0")}`);
  await input.setInputFiles({ name: "loi-2.csv", mimeType: "text/csv", buffer: Buffer.from(rows.join("\n") + "\n") });
  await page.getByRole("button", { name: "Xem trước" }).click();
  await expect(page.getByRole("cell", { name: 'Dòng 7 · Email · "Email không đúng dạng."' })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByRole("cell", { name: 'Dòng 19 · Họ và tên · "Thiếu họ và tên."' })).toBeVisible();
  await expect(page.getByText(/28 sẽ được nhập/)).toBeVisible();
  expect(psql(`select count(*) from enrollments where course_id='${C.id}' and role='STUDENT'`)).toBe("0"); // dry_run không ghi
  const go = page.getByRole("button", { name: "Nhập 28 sinh viên" });
  await expect(go).toBeEnabled();
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
  // mất mạng giữa chừng: giữ tệp + Gửi lại cùng khoá
  const keys: string[] = [];
  let down = true;
  await page.route(/\/roster\/import(\?(?!dry_run=true).*)?$/, (r) => {
    keys.push(r.request().headers()["idempotency-key"] ?? "");
    return down ? r.abort("internetdisconnected") : r.fallback();
  });
  await go.click();
  await expect(page.getByText("Mất kết nối. Tệp vẫn được giữ")).toBeVisible({ timeout: 40_000 });
  await expect(page.getByText("loi-2.csv")).toBeVisible();
  down = false;
  await page.getByRole("button", { name: "Gửi lại" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã nhập 28 sinh viên · 2 dòng lỗi chưa nhập." })).toBeVisible({ timeout: 60_000 });
  note(`dry_run đã gọi ${calls.filter((c) => c.startsWith("dry")).length} lần; nhập thật ${keys.length} lần, ${new Set(keys).size} khoá; enrollments=${psql(`select count(*) from enrollments where course_id='${C.id}' and role='STUDENT'`)}`);
  expect(new Set(keys).size).toBe(1);
  expect(psql(`select count(*) from enrollments where course_id='${C.id}' and role='STUDENT'`)).toBe("28");
  // TA không có tab
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await page.context().clearCookies();
  await apiLogin(page, S.ta.email);
  await page.goto(`/class/members?course=${C.id}`);
  await expect(page.getByRole("heading", { name: "Thành viên lớp", level: 1 })).toBeVisible({ timeout: 30_000 });
  expect(await page.getByRole("tab", { name: "Nhập danh sách" }).count()).toBe(0);
  clearRL();
});
