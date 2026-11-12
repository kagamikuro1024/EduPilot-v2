import { expect, test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC FEAT-uiux-91 đợt 4 (tc-wave-4.md): TC-UX15-14 — 409 THẬT khi lưu điểm (không giả phản hồi): người kia sửa trước, UI giữ bản cũ.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";

test("TC-UX15-14 xung đột phiên bản thật: hiện giá trị của người kia, hỏi giữ bên nào, không mất điểm đã nhập", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(120_000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  await page.waitForTimeout(1200);
  const api = (path: string, method = "GET", body?: unknown) =>
    page.evaluate(async ([p, m, b]) => {
      const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
      const h: Record<string, string> = { authorization: `Bearer ${a.access_token}`, "content-type": "application/json" };
      const r = await fetch(`/api/v1${p}`, { method: m as string, headers: h, body: b ? JSON.stringify(b) : undefined });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, [path, method, body ?? null] as const);
  const mine = await api("/me/courses");
  const c = mine.body.items.find((i: { course: { class_code: string } }) => i.course.class_code === "761987").course.id as string;
  const as = await api(`/courses/${c}/assignments?limit=50`);
  const aid = as.body.items.find((a: { title: string }) => /Bài tập 03/.test(a.title)).id as string;
  const dr = await api(`/courses/${c}/submissions?limit=100&assignment_id=${aid}&status=DRAFT`);
  expect(dr.body.items.length, "có bài DRAFT").toBeGreaterThan(0);
  const sid = dr.body.items[0].id as string;
  let rid = "";
  page.on("response", async (r) => { const m = r.url().match(/grading-results\/([0-9a-f-]{36})$/); if (m && r.request().method() === "GET") rid = m[1]; });
  await page.goto(`/grading/${sid}?course=${c}`);
  await page.waitForTimeout(3500);
  expect(rid, "bắt được id kết quả").not.toBe("");
  const inp = page.locator("main input[aria-label^='Điểm ']").nth(1);
  const old = await inp.inputValue();
  // người kia sửa điểm tiêu chí 2 trước
  const cur = await api(`/courses/${c}/grading-results/${rid}`);
  const crit = cur.body.criteria[1];
  const theirs = old === "1,50" ? "1.25" : "1.50";
  const put = await api(`/courses/${c}/grading-results/${cur.body.result.id}`, "PUT", { version: cur.body.result.version, criteria: [{ id: crit.id, score: theirs }] });
  console.log("INFO người kia PUT:", put.status, JSON.stringify(put.body).slice(0, 200), "rid", rid, "result.id", cur.body.result.id, "| version", cur.body.result.version, "→", put.body?.result?.version);
  expect(put.status).toBe(200);
  // tôi sửa bằng bản cũ (version cũ trong trang)
  const mine2 = old === "0,75" ? "1,00" : "0,75";
  await inp.fill(mine2);
  await inp.blur();
  await page.waitForTimeout(2500);
  const txt = (await page.locator("main").innerText()).replace(/\s+/g, " ");
  const hasNotice = /Người khác vừa sửa/.test(txt);
  const btns = await page.getByRole("button", { name: /Giữ của tôi|Dùng của họ/ }).allInnerTexts();
  console.log("INFO sau 409: thông báo:", hasNotice, "| nút:", JSON.stringify(btns), "| ô điểm:", await inp.inputValue(), "| bản của họ:", theirs);
  await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-4/review-409.png" });
  expect(hasNotice, "UX15-14: hiện thông báo xung đột").toBe(true);
  expect(btns.length, "hai lựa chọn giữ bên nào").toBe(2);
  const shown = (await inp.inputValue()).replace(",", ".");
  console.log("INFO ô điểm sau 409 (mong đợi: của tôi", mine2, "hoặc của họ", theirs, "):", shown, "| ban đầu:", old);
  expect.soft([mine2.replace(",", "."), theirs], "UX15-14: ô điểm hiện giá trị tôi nhập hoặc của người kia, không bản cũ").toContain(shown.replace(/0$/, "") === shown ? shown : shown);
  await page.getByRole("button", { name: "Giữ của tôi" }).click();
  await page.waitForTimeout(2500);
  const fin = await api(`/courses/${c}/grading-results/${cur.body.result.id}`);
  const got = fin.body.criteria[1].score as string;
  console.log("INFO sau «Giữ của tôi»: điểm lưu =", got, "| tôi đã nhập", mine2, "| version", fin.body.result.version);
  expect(Number(got.replace(",", ".")), "«Giữ của tôi» ghi đúng điểm tôi nhập").toBe(Number(mine2.replace(",", ".")));
  await ctx.close();
});

test("TC-UX16-12 xung đột ô sổ điểm thật: hiện giá trị người kia, Giữ của tôi / Dùng của họ, không mất số đã nhập", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  test.setTimeout(120_000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
  await page.waitForTimeout(1200);
  const api = (path: string, method = "GET", body?: unknown) =>
    page.evaluate(async ([p, m, b]) => {
      const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
      const h: Record<string, string> = { authorization: `Bearer ${a.access_token}`, "content-type": "application/json" };
      const r = await fetch(`/api/v1${p}`, { method: m as string, headers: h, body: b ? JSON.stringify(b) : undefined });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, [path, method, body ?? null] as const);
  const mine = await api("/me/courses");
  const c = mine.body.items.find((i: { course: { class_code: string } }) => i.course.class_code === "761987").course.id as string;
  let req: { item_id: string; student_id: string; score: string; version: number } | null = null;
  page.on("request", (r) => { if (r.method() === "PUT" && /grade-entries$/.test(r.url())) req = JSON.parse(r.postData() ?? "null"); });
  await page.goto(`/gradebook?course=${c}`);
  await page.waitForTimeout(4500);
  const inp = page.locator("main tbody tr").first().locator("input").first();
  const orig = (await inp.inputValue()).replace(",", ".");
  // lần sửa đầu để bắt item_id / student_id / version hiện tại của ô
  const first = orig === "5.25" ? "5,50" : "5,25";
  await inp.click(); await inp.fill(first); await page.keyboard.press("Tab"); await page.waitForTimeout(2500);
  expect(req, "bắt được PUT grade-entries").not.toBeNull();
  const r1 = req!;
  await page.goto(`/gradebook?course=${c}`); // UI giữ version r1.version + 1
  await page.waitForTimeout(4500);
  // người kia ghi trước (dùng đúng version UI đang giữ)
  const theirs = first === "5,25" ? "6.00" : "6.75";
  const other = await api(`/courses/${c}/grade-entries`, "PUT", { item_id: r1.item_id, student_id: r1.student_id, score: theirs, version: r1.version + 1 });
  console.log("INFO người kia PUT:", other.status, JSON.stringify(other.body).slice(0, 120));
  expect(other.status).toBe(200);
  const inp2 = page.locator("main tbody tr").first().locator("input").first();
  const mineVal = "9,50";
  await inp2.click(); await inp2.fill(mineVal); await page.keyboard.press("Tab"); await page.waitForTimeout(2500);
  const pop = page.locator("[data-part=cell-conflict]");
  const popTxt = (await pop.innerText().catch(() => "")).replace(/\s+/g, " ");
  console.log("INFO hộp xung đột:", popTxt, "| ô:", await inp2.inputValue());
  await page.screenshot({ path: "../docs/sprints/9.1/qc/shots/wave-4/gradebook-409.png" });
  expect(await pop.count(), "hiện hộp xung đột").toBeGreaterThan(0);
  expect(popTxt, "hiện giá trị của người kia").toMatch(/vừa nhập 6([,.]\d+)?(?!\d)/);
  await page.getByRole("button", { name: /Giữ của tôi/ }).first().click();
  await page.waitForTimeout(2500);
  const entry = async () => ((await api(`/courses/${c}/gradebook?limit=100`)).body.rows.find((x: { student: { id: string } }) => x.student.id === r1.student_id).entries[r1.item_id]) as { score: string; version: number };
  const cell = (await entry()).score;
  console.log("INFO sau «Giữ của tôi»: điểm lưu =", cell, "| tôi nhập", mineVal);
  const back = await api(`/courses/${c}/grade-entries`, "PUT", { item_id: r1.item_id, student_id: r1.student_id, score: orig, version: (await entry()).version }); // trả dữ liệu như seed
  console.log("INFO khôi phục ô về", orig, ":", back.status);
  expect(Number(String(cell).replace(",", ".")), "«Giữ của tôi» ghi đúng số tôi nhập").toBe(9.5);
  await ctx.close();
});
