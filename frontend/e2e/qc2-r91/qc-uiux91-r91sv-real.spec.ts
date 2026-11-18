/* eslint-disable @typescript-eslint/no-explicit-any */
import { test } from "@playwright/test";
import { api, courseId, expect, go, mainText, noOverflow, open, rect } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const BAD = /\b(RAG|PII|provider|fallback|trace|confidence|token|LLM|embedding|prompt|model)\b/i;
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");
test.describe.configure({ mode: "serial" });

// ---------------- US-UX91-20 (/me, sinh viên) ----------------
test("TC-UX20-1/2/3/4/9 /me sinh viên", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "sv.gioi"); const c = await courseId(o.page, "761987");
  await go(o.page, "/me", c, 3000);
  const t = await mainText(o.page);
  // UX20-1: một khối giải trình; dòng «Tên · trọng số % · điểm» kèm «Căn cứ · tr. N» cùng dòng; không khối liệt kê thành phần riêng
  const gm = o.page.getByText(/^Giữa kỳ · 20 %/).first();
  await expect(gm, "dòng Giữa kỳ · 20 %").toBeVisible();
  const row = await gm.evaluate((e) => { let n: HTMLElement | null = e as HTMLElement; for (let i = 0; i < 4 && n; i++) { if (/Giữa kỳ/.test(n.innerText) && /\d/.test(n.innerText.replace(/Giữa kỳ · 20 %/, ""))) return n.innerText.replace(/\s+/g, " "); n = n.parentElement; } return (e as HTMLElement).innerText; });
  console.log("INFO UX20-1 dòng:", JSON.stringify(row));
  expect(row).toMatch(/Giữa kỳ · 20 %.*Căn cứ · tr\. \d/);
  expect((t.match(/Giữa kỳ/g) ?? []).length, "«Giữa kỳ» xuất hiện đúng một lần (một khối)").toBe(1);
  expect(/Thành phần điểm|Các thành phần/.test(t), "không còn khối liệt kê thành phần riêng").toBe(false);
  expect(t, "không cột đóng góp").not.toMatch(/đóng góp/i);
  // UX20-2
  await expect(o.page.getByRole("heading", { level: 1 })).toHaveText("Kết quả của tôi");
  const head = await o.page.evaluate(() => (document.querySelector("[data-ep-header-meta]") as HTMLElement | null)?.innerText ?? "");
  console.log("INFO UX20-2 meta dưới h1:", JSON.stringify(head));
  expect(head.trim()).not.toBe("761987");
  expect(t, "dòng nhãn + giá trị (P-11-6a): «Điểm quá trình X / 10» hoặc, khi đã chốt, «Điểm học phần X (chữ)»").toMatch(/Điểm quá trình\s*[\d,.]+\s*\/\s*10|Điểm học phần\s*[\d,.]+\s*\([A-F][+]?\)/);
  expect(t).not.toContain("Điểm quá trình hiện tại của bạn là");
  // UX20-3: số khớp API
  const g = await api(o.page, `/courses/${c}/me/grades`);
  expect(g.status).toBe(200);
  const nums = new Set<string>(); const walk = (v: any) => { if (v == null) return; if (typeof v === "number") nums.add(String(+v.toFixed(4))); else if (typeof v === "string") { for (const m of v.matchAll(/\d+(?:[.,]\d+)?/g)) nums.add(String(+(+m[0].replace(",", ".")).toFixed(4))); } else if (typeof v === "object") Object.values(v).forEach(walk); };
  walk(g.body);
  const lines = (await o.page.locator("main").innerText()).split("\n").filter((l) => /Giữa kỳ|Kiểm tra hằng tuần|Thi cuối kỳ|Điểm học phần|Điểm quá trình|Điểm cuối kỳ/.test(l));
  const bad: string[] = [];
  for (const l of lines) for (const m of l.replace(/\d+ ?%/g, "").matchAll(/\d+(?:,\d+)?/g)) { const v = String(+m[0].replace(",", ".")); if (!nums.has(v) && !["10", "0", "40", "60", "20"].includes(v)) bad.push(`${l.slice(0, 40)}→${m[0]}`); }
  console.log("INFO UX20-3 số không có trong API:", JSON.stringify(bad));
  expect(bad).toEqual([]);
  // UX20-4: không lộ nháp / ghi chú / rủi ro
  expect(t).not.toMatch(/bản nháp|nháp|ghi chú quan sát|nguy cơ|AT_RISK|rủi ro/i);
  // UX20-5 (nút «Yêu cầu xem lại») nằm ở /assignments/‹id›, kiểm trong qc-uiux91-rs0misc-real.spec.ts
  await o.ctx.close();
  // UX20-9: 375
  const m = await open(browser, "sv.gioi", 375, 812); await go(m.page, "/me", c, 3500);
  const h = await m.page.evaluate(() => document.documentElement.scrollHeight);
  console.log("INFO UX20-9 scrollHeight 375:", h);
  expect(h).toBeLessThanOrEqual(1600);
  expect(await noOverflow(m.page)).toBe(true);
  await m.ctx.close();
});

// ---------------- US-UX91-13 (/chat, sinh viên) ----------------
test("TC-UX13-1/5/9/11/12/13/17 /chat 1440 (phiên trống, lịch sử)", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "sv.gioi"); const c = await courseId(o.page, "761987");
  const posts: string[] = []; o.page.on("request", (r) => { if (r.method() !== "GET" && /\/chat\//.test(r.url())) posts.push(r.method() + " " + new URL(r.url()).pathname); });
  await go(o.page, "/chat", c, 4000);
  const rows = o.page.locator("[data-part=chat-history] li");
  console.log("INFO số phiên:", await rows.count());
  expect(await rows.count()).toBeGreaterThanOrEqual(3);
  const inner = await rect(o.page.locator("[data-part=chat-inner]")); const hist = await rect(o.page.locator("[data-part=chat-history]")); const main = await rect(o.page.locator("main"));
  const L = inner.x - hist.r, R = main.r - inner.r;
  console.log("INFO UX13-1 inner", JSON.stringify(inner), "hist", JSON.stringify(hist), "main", JSON.stringify({ x: main.x, r: main.r }), "L", Math.round(L), "R", Math.round(R));
  expect(inner.w).toBeLessThanOrEqual(841);
  expect(Math.abs((L - 24) - R), "lề trái (sau 24 px) = lề phải ±3").toBeLessThanOrEqual(3);
  const hs = await o.page.locator("[data-part=chat-history]").evaluate((e) => { const c = getComputedStyle(e); return { br: c.borderRadius, sh: c.boxShadow, bg: c.backgroundColor, bl: c.borderLeftWidth, brt: c.borderRightWidth }; });
  console.log("INFO UX13-1 kiểu cột lịch sử", JSON.stringify(hs));
  expect(hs.br).toBe("0px"); expect(hs.sh).toBe("none");
  expect(await o.page.locator("[data-part=chat-history]").evaluate((e) => !!e.closest("[data-ep-panel]")), "lịch sử không nằm trong Panel").toBe(false);
  // UX13-5: xoá nằm trong menu ⋯
  const trash = await rows.first().locator("button[aria-label*=Xoá]").count();
  await rows.first().hover();
  const more = rows.first().getByRole("button", { name: /Thao tác với phiên/ });
  expect(trash, "không nút thùng rác trực tiếp trên hàng").toBe(0);
  await more.click(); await expect(o.page.getByRole("menuitem", { name: "Xoá" })).toBeVisible(); await o.page.keyboard.press("Escape");
  // UX13-9/11/12/13/17 phiên trống: mở «Cuộc trò chuyện mới»/phiên trống
  const emptySug = o.page.locator("[data-part=chat-suggestions] li");
  if (!(await emptySug.count())) { await o.page.goto(`/chat?course=${c}`); await o.page.waitForTimeout(3500); }
  console.log("INFO UX13-9 nút Phiên mới khi trống:", await o.page.getByRole("button", { name: "Phiên mới" }).count());
  expect(await o.page.getByRole("button", { name: "Phiên mới" }).count()).toBe(0);
  const sugs = await emptySug.allInnerTexts(); console.log("INFO UX13-11 gợi ý:", JSON.stringify(sugs));
  expect(sugs.length).toBeGreaterThanOrEqual(2); expect(sugs.length).toBeLessThanOrEqual(3);
  expect(sugs.join("|")).toContain("Điểm quá trình của tôi hiện là bao nhiêu?"); expect(sugs.join("|")).toContain("Tuần này tôi có những buổi học nào?");
  const sst = await emptySug.first().evaluate((e) => { const b = (e.querySelector("button") ?? e) as HTMLElement; const c = getComputedStyle(b); return { br: c.borderRadius, bg: c.backgroundColor, bd: c.borderTopWidth }; });
  console.log("INFO UX13-11 kiểu", JSON.stringify(sst));
  expect(parseFloat(sst.br)).toBeLessThanOrEqual(2); expect(sst.bg === "rgba(0, 0, 0, 0)" || sst.bg === "transparent").toBe(true);
  const before = posts.length;
  await emptySug.nth(sugs.findIndex((s) => /Điểm quá trình/.test(s)) >= 0 ? sugs.findIndex((s) => /Điểm quá trình/.test(s)) : 0).locator("button, a, span").first().click();
  await o.page.waitForTimeout(700);
  const val = await o.page.getByRole("textbox", { name: "Câu hỏi của bạn" }).inputValue();
  console.log("INFO UX13-12 ô nhập:", JSON.stringify(val), "POST mới:", posts.length - before);
  expect(val.length).toBeGreaterThan(5); expect(posts.length - before).toBe(0);
  await o.page.getByRole("textbox", { name: "Câu hỏi của bạn" }).fill("");
  const lastSug = await rect(emptySug.last()); const comp = await rect(o.page.locator("[data-part=chat-composer]"));
  console.log("INFO UX13-13 khoảng trống", Math.round(comp.y - lastSug.b), "/", Math.round(inner.h));
  expect((comp.y - lastSug.b) / inner.h).toBeLessThanOrEqual(1 / 3 + 0.005);
  const intro = await o.page.locator("main").innerText();
  expect(intro).toContain("chỉ bạn đọc được");
  expect(intro.match(BAD), "từ kỹ thuật AI").toBeNull();
  await o.ctx.close();
});

test("TC-UX13-6/7/8/10/15 gửi tin, Phiên mới, PII, xoá có hoàn tác", async ({ browser }, info) => {
  D(info); test.setTimeout(280_000);
  const o = await open(browser, "sv.gioi"); const c = await courseId(o.page, "761987");
  const reqs: string[] = []; o.page.on("request", (r) => { if (r.method() !== "GET" && /\/chat\//.test(r.url())) reqs.push(r.method() + " " + new URL(r.url()).pathname.replace(/[0-9a-f-]{36}/g, "‹id›")); });
  await go(o.page, "/chat", c, 4000);
  const composer = o.page.getByRole("textbox", { name: "Câu hỏi của bạn" });
  const send = async (s: string) => { await composer.fill(s); await o.page.getByRole("button", { name: "Gửi", exact: true }).click(); };
  const tag = `r91-${Date.now() % 100000}`;
  await send(`Em tên Nguyễn Minh Trung, MSSV 20229002, email sv.kha@edupilot.local, hỏi điểm giữa kỳ ${tag}`);
  await o.page.waitForTimeout(9000);
  // UX13-10: nút Phiên mới là nút phụ (không đỏ)
  const pm = o.page.getByRole("button", { name: "Phiên mới" });
  expect(await pm.count(), "«Phiên mới» hiện sau khi gửi 1 tin").toBeGreaterThan(0);
  const pmv = await pm.first().evaluate((e) => ({ v: e.getAttribute("data-variant"), bg: getComputedStyle(e).backgroundColor }));
  console.log("INFO UX13-10 Phiên mới", JSON.stringify(pmv)); expect(pmv.v).not.toBe("primary"); expect(pmv.bg).not.toBe("rgb(206, 22, 40)");
  // UX13-15: dòng che PII
  const txt = (await o.page.locator("main").innerText()).replace(/\s+/g, " ");
  const m = txt.match(/Đã ẩn (\d+) thông tin cá nhân trước khi gửi cho AI/);
  console.log("INFO UX13-15", JSON.stringify(m));
  expect(m, "dòng che PII").not.toBeNull(); expect(+m![1]).toBeGreaterThanOrEqual(2);
  // UX13-6/7/8: xoá phiên vừa tạo
  const rows = o.page.locator("[data-part=chat-history] li"); const n0 = await rows.count();
  const dels = () => reqs.filter((x) => x.startsWith("DELETE")).length;
  await rows.first().hover();
  await rows.first().getByRole("button", { name: /Thao tác với phiên/ }).click();
  const dlg0 = await o.page.getByRole("dialog").count();
  await o.page.getByRole("menuitem", { name: "Xoá" }).click();
  await o.page.waitForTimeout(500);
  const n1 = await rows.count(); const undo = o.page.getByRole("button", { name: "Hoàn tác" });
  console.log("INFO UX13-6 hàng", n0, "→", n1, "hộp thoại", dlg0, await o.page.getByRole("dialog").count(), "Hoàn tác", await undo.count());
  expect(n1).toBe(n0 - 1); expect(await o.page.getByRole("dialog").count()).toBe(0); expect(await undo.count()).toBeGreaterThan(0);
  expect(await o.page.getByText(/Đã xoá/).count()).toBeGreaterThan(0);
  await undo.first().click(); await o.page.waitForTimeout(1500);
  console.log("INFO UX13-7 hàng", await rows.count(), "DELETE", dels());
  expect(await rows.count()).toBe(n0); expect(dels(), "Hoàn tác: không DELETE").toBe(0);
  await rows.first().getByRole("button", { name: /Thao tác với phiên/ }).click();
  await o.page.getByRole("menuitem", { name: "Xoá" }).click();
  const t0 = Date.now(); let at = -1;
  for (let i = 0; i < 40 && at < 0; i++) { await o.page.waitForTimeout(250); if (dels() > 0) at = Date.now() - t0; }
  console.log("INFO UX13-8 DELETE sau", at, "ms");
  expect(at).toBeGreaterThan(3500); expect(at).toBeLessThan(7500);
  await o.page.reload(); await o.page.waitForTimeout(4000);
  expect((await o.page.locator("[data-part=chat-history]").innerText())).not.toContain(tag.slice(0, 3) === "r91" ? "r91-" + tag.split("-")[1] : tag);
  await o.ctx.close();
});
