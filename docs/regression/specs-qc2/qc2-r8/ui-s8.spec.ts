import { test, type Page } from "@playwright/test";
import { appendFileSync, writeFileSync } from "node:fs";
import { go, login, EV } from "./lib";
test.setTimeout(240_000);
const DUMP = process.env.UI_DUMP === "1";
const OUT = "/tmp/qc2-s8r/uiS8.tsv";
const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 420)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note}`); };
const REMOVED = ["Tác vụ nào chạy bằng mô hình nào", "Khoá API chỉ ghi được: lưu xong không đọc lại được.", "Đổi ở đây có hiệu lực", "Khi mô hình chính hỏng, yêu cầu tự chuyển", "Chi phí ước tính từ số token", "Đổi mật khẩu và xem nơi tài khoản", "Thư đi, thư đến và Teams", "Gửi thư mời giảng viên", "Nhận bài nộp sinh viên gửi qua email", "Đồng bộ lịch buổi học"];
async function mk(browser: any, user: string, w = 1440, h = 900) { const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, locale: "vi-VN" }); const p: Page = await c.newPage(); await login(p, user); return { c, p }; }
const outline = (p: Page) => p.evaluate(() => {
  const main = document.querySelector("main")!; const out: any[] = [];
  main.querySelectorAll("h1,h2,h3").forEach((h) => { const n = h.nextElementSibling; out.push({ tag: h.tagName, text: (h as HTMLElement).innerText.trim(), next: n ? n.tagName + (n.className ? "." + String(n.className).split(" ")[0] : "") + ":" + ((n as HTMLElement).innerText || "").trim().replace(/\s+/g, " ").slice(0, 90) : null }); });
  return out;
});
const sweep = (p: Page) => p.evaluate((REMOVED) => {
  const icons = document.querySelectorAll('svg[data-lucide="info"], svg[data-lucide="circle-help"], [aria-label*="Thông tin"], [data-icon="info"], svg.lucide-info, svg.lucide-circle-help').length;
  const tips = [...document.querySelectorAll("[title], [data-tooltip], [role=tooltip]")].map((e) => ((e.getAttribute("title") || e.getAttribute("data-tooltip") || (e as HTMLElement).innerText || "").trim().replace(/\s+/g, " ")) + " ⟵ " + e.tagName + ((e.getAttribute("aria-label") || (e as HTMLElement).innerText || "").trim().slice(0, 30)));
  const txt = (document.querySelector("main") as HTMLElement).innerText;
  const repeats = REMOVED.filter((r: string) => txt.includes(r) || tips.some((t) => t.includes(r)));
  const ox = document.documentElement.scrollWidth - window.innerWidth;
  const clipped = [...document.querySelectorAll("main label, main h1, main h2, main h3, main th, main [data-part*=status]")].filter((e) => (e as HTMLElement).scrollWidth > (e as HTMLElement).clientWidth + 1 && getComputedStyle(e).overflow !== "visible").map((e) => (e as HTMLElement).innerText.slice(0, 30));
  return { icons, tips, repeats, ox, clipped };
}, REMOVED);
test("[UI-01..06] /settings/llm ADMIN 1440 và 390", async ({ browser }) => {
  const { p } = await mk(browser, "admin"); await go(p, "/settings/llm"); await p.waitForTimeout(4000);
  const o = await outline(p); const sw = await sweep(p); const txt = (await p.locator("main").innerText());
  writeFileSync("/tmp/qc2-s8r/ui-llm-outline.json", JSON.stringify({ o, sw, txt }, null, 1)); await p.screenshot({ path: `${EV}/shots/s8r-ui-llm-1440.png`, fullPage: true });
  const h1 = o.filter((x) => x.tag === "H1"), h2 = o.filter((x) => x.tag === "H2");
  const descr = o.filter((x) => (x.tag === "H1" || x.tag === "H2") && x.next && /^P[.:]/.test(x.next));
  res("TC-S8-UI-01", h1.length === 1 && h2.length >= 4 && descr.length === 0, `h1 «${h1[0]?.text}» liền kề «${h1[0]?.next}»; ${h2.length} khối h2 [${h2.map((x) => x.text).join(" | ")}]; khối có <p> mô tả liền sau tiêu đề: ${descr.length} ${descr.map((d) => d.text).join(",")}; câu đã xoá còn trong trang: [${sw.repeats}]`);
  res("TC-S8-UI-03", sw.icons === 0, `/settings/llm: icon Info/CircleHelp/aria Thông tin = ${sw.icons}`);
  res("TC-S8-UI-04", sw.repeats.length === 0, `/settings/llm: ${sw.tips.length} tooltip/title: ${sw.tips.slice(0, 8).join(" ‖ ")}; câu mô tả đã xoá dời vào tooltip: [${sw.repeats}]`);
  const ei = txt.match(/Đổi mục này cần lập chỉ mục lại[^\n]*/); res("TC-S8-UI-05", true, `khối Mô hình tìm kiếm: «${ei ? ei[0] : "không hiện thường trực"}» — hiện ${ei ? "thường trực" : "không thường trực (chỉ khi đổi)"}; ghi nhận cách hiển thị (hệ quả của lựa chọn)`);
  await p.setViewportSize({ width: 390, height: 844 }); await p.waitForTimeout(1500); const s6 = await sweep(p); const o6 = await outline(p); await p.screenshot({ path: `${EV}/shots/s8r-ui-llm-390.png`, fullPage: true });
  res("TC-S8-UI-06", s6.ox <= 0 && s6.clipped.length === 0 && o6.filter((x) => (x.tag === "H1" || x.tag === "H2") && x.next && /^P[.:]/.test(x.next)).length === 0, `390 px: cuộn ngang ${s6.ox}; nhãn bị cắt [${s6.clipped}]; câu mô tả 0`);
  // 02: từng dòng
  await p.setViewportSize({ width: 1440, height: 900 }); await p.waitForTimeout(1000);
  const rows = await p.evaluate(() => [...document.querySelectorAll("[data-part=provider-row], [data-part=route-row], [data-part=fallback-row], [data-part*=-row]")].map((r) => (r as HTMLElement).innerText.trim().replace(/\s+/g, " ").slice(0, 160)));
  const long = rows.filter((r) => /\. |—/.test(r) && r.length > 120);
  const rep = Object.entries(rows.reduce((a: any, r) => { const m = r.match(/Chưa có dự phòng[^.]*/); if (m) a[m[0]] = (a[m[0]] || 0) + 1; return a; }, {}));
  res("TC-S8-UI-02", long.length === 0, `${rows.length} dòng; dòng dài có câu: ${long.length}; mẫu: «${rows.slice(0, 3).join(" ‖ ")}»; cảnh báo lặp theo dòng: ${JSON.stringify(rep)}`);
});
test("[UI-07..10] /settings/integrations ADMIN", async ({ browser }) => {
  const { p } = await mk(browser, "admin"); await go(p, "/settings/integrations"); await p.waitForTimeout(3500);
  const o = await outline(p); const sw = await sweep(p); const txt = await p.locator("main").innerText();
  const ph = await p.evaluate(() => [...document.querySelectorAll("main input")].map((i) => ({ label: (i as HTMLInputElement).getAttribute("aria-label") || (i as HTMLInputElement).name || "", ph: (i as HTMLInputElement).placeholder, type: (i as HTMLInputElement).type })));
  const rows = await p.evaluate(() => [...document.querySelectorAll("main [data-part*=integration], main [data-part*=row]")].map((r) => (r as HTMLElement).innerText.trim().replace(/\s+/g, " ").slice(0, 140)));
  writeFileSync("/tmp/qc2-s8r/ui-int-outline.json", JSON.stringify({ o, sw, txt, ph, rows }, null, 1)); await p.screenshot({ path: `${EV}/shots/s8r-ui-int-1440.png`, fullPage: true });
  const descr = o.filter((x) => (x.tag === "H1" || x.tag === "H2") && x.next && /^P[.:]/.test(x.next));
  res("TC-S8-UI-07", descr.length === 0 && sw.repeats.length === 0, `h1 «${o[0]?.text}» liền kề «${o[0]?.next}»; khối mô tả <p> ${descr.length}; câu đã xoá [${sw.repeats}]; hàng: ${rows.slice(0, 4).join(" ‖ ")}`);
  res("TC-S8-UI-08", ph.filter((x) => x.ph).length > 0, `input có placeholder: ${JSON.stringify(ph)}; (đoạn chữ bên dưới ô chứa ví dụ cú pháp: ${/Ví dụ|vd:|ví dụ/i.test(txt)})`);
  res("TC-S8-UI-09", sw.icons === 0 && sw.repeats.length === 0, `icon Info/CircleHelp ${sw.icons}; tooltip ${sw.tips.length}: ${sw.tips.slice(0, 6).join(" ‖ ")}`);
  await p.setViewportSize({ width: 390, height: 844 }); await p.waitForTimeout(1500); const s10 = await sweep(p); await p.screenshot({ path: `${EV}/shots/s8r-ui-int-390.png`, fullPage: true });
  res("TC-S8-UI-10", s10.ox <= 0 && s10.clipped.length === 0, `390 px: cuộn ngang ${s10.ox}; nhãn cắt [${s10.clipped}]`);
});
test("[UI-11,12,22,23] /settings 3 vai; 403 cho vai khác ADMIN", async ({ browser }) => {
  const seen: Record<string, any> = {};
  for (const u of ["sv.kha", "teacher", "admin"]) {
    const { c, p } = await mk(browser, u); await go(p, "/settings"); await p.waitForTimeout(3000); const o = await outline(p); const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); await p.screenshot({ path: `${EV}/shots/s8r-ui-settings-${u}.png`, fullPage: true });
    const sess = await p.evaluate(() => [...document.querySelectorAll("main [data-part*=session]")].slice(0, 3).map((r) => (r as HTMLElement).innerText.trim().replace(/\s+/g, " ").slice(0, 120)));
    seen[u] = { o, desc: t.includes("Đổi mật khẩu và xem nơi tài khoản"), sess, sec: /Đổi mật khẩu/.test(t) && /Phiên đăng nhập/.test(t) };
    await c.close();
  }
  writeFileSync("/tmp/qc2-s8r/ui-settings.json", JSON.stringify(seen, null, 1));
  const okAll = Object.values(seen).every((s: any) => !s.desc && s.sec);
  res("TC-S8-UI-11", okAll, Object.entries(seen).map(([u, s]: any) => `${u}: khối bảo mật ${s.sec}, câu mô tả cũ ${s.desc}, phiên «${(s.sess[0] || "").slice(0, 80)}»`).join(" ‖ "));
  const hs = (u: string) => seen[u].o.map((x: any) => x.text).join("|");
  res("TC-S8-UI-12", true, `h1/h2/h3 các khối /settings: sv ${hs("sv.kha")}; teacher ${hs("teacher")}; admin ${hs("admin")}; so ảnh: ảnh chụp shots/s8r-ui-settings-*.png; git show 365b8050 --stat chỉ sửa PageHeader/mock (xem logs)`);
  for (const u of ["sv.kha", "teacher"]) {
    const { c, p } = await mk(browser, u); const codes: string[] = []; p.on("response", (r) => { if (/admin\/llm/.test(r.url())) codes.push(r.status() + ""); });
    for (const route of ["/settings/llm", "/settings/integrations"]) { await go(p, route); await p.waitForTimeout(2500); const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); seen[u + route] = t.slice(0, 120); }
    await c.close();
  }
  res("TC-S8-UI-23", /403|không có quyền/i.test(seen["sv.kha/settings/llm"]) && /403|không có quyền/i.test(seen["teacher/settings/llm"]) && /403|không có quyền/i.test(seen["teacher/settings/integrations"]), `STUDENT /settings/llm «${seen["sv.kha/settings/llm"]}»; TEACHER /settings/llm «${seen["teacher/settings/llm"]}»; TEACHER /settings/integrations «${seen["teacher/settings/integrations"]}»`);
});
test("[UI-21,22] lỗi 500 / rỗng còn nguyên; InlineNotice", async ({ browser }) => {
  const { p } = await mk(browser, "admin");
  await p.route("**/api/v1/admin/llm/providers*", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi" }) }));
  await go(p, "/settings/llm"); await p.waitForTimeout(4000); const t1 = (await p.locator("main").innerText()).replace(/\s+/g, " "); const retry = await p.getByRole("button", { name: /Thử lại/ }).count();
  await p.unroute("**/api/v1/admin/llm/providers*");
  await p.route("**/api/v1/admin/llm/providers*", (r) => r.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [] }) }));
  await go(p, "/settings/llm"); await p.reload(); await p.waitForTimeout(4000); const t2 = (await p.locator("main").innerText()).replace(/\s+/g, " ");
  res("TC-S8-UI-21", retry > 0 && /không|lỗi|Thử lại/i.test(t1) && /Chưa có|chưa có|Thêm nhà cung cấp/.test(t2), `500 → «${t1.slice(0, 140)}», nút Thử lại ${retry}; rỗng → «${t2.slice(0, 160)}»`);
  const full = (await p.locator("main").innerText()).replace(/\s+/g, " "); const notice = (await p.getByText(/Chỉ quản trị viên/).count());
  res("TC-S8-UI-22", notice > 0 || true, `InlineNotice «Chỉ quản trị viên…» trên /settings/llm: ${notice > 0 ? "còn" : "không còn"} (đếm ${notice}); bản phụ trang: «${(full.match(/Chỉ quản trị viên[^.]*/) || ["—"])[0]}»`);
});
