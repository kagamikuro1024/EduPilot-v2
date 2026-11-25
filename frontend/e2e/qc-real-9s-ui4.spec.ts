import { readFileSync, writeFileSync } from "node:fs";
import AxeBuilder from "@axe-core/playwright";
import { test } from "@playwright/test";
import { needReal, noOverflow, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; aid: string; aid2: string };
const out: Record<string, unknown> = {};
const save = () => writeFileSync("/tmp/q9s/ui4.json", JSON.stringify(out, null, 1));
const BAD = ["ai_scores", "độ lệch", "confidence", "guidance", "đáp án chuẩn", "CANARY", "NEEDS_REVIEW", "lượt 2", "hai lượt"];
test("@real UI4 sinh viên B: kết quả đã công bố 1440 + 375", async ({ page }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "sv.kha");
  for (const vp of [{ w: 1440, h: 900 }, { w: 375, h: 812 }]) {
    await page.setViewportSize({ width: vp.w, height: vp.h });
    await page.goto(`/assignments/${S.aid2}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
    const t = await page.locator("main").innerText(); out[`B${vp.w}`] = t.slice(0, 1800);
    out[`B${vp.w}x`] = { label: t.includes("đã được giảng viên duyệt"), reviewer: ["Lê Thu Hà", "Phạm Quốc Bảo"].filter((n) => t.includes(n)), others: ["Nguyễn Cô", "sv.gioi", "20229002"].filter((n) => t.includes(n)), lines: t.split("\n").filter((l) => /duyệt|Nhận xét|Trừ/.test(l)).slice(0, 8) };
    out[`B${vp.w}bad`] = BAD.filter((w) => t.toLowerCase().includes(w.toLowerCase()));
    out[`B${vp.w}btn`] = await page.getByRole("button", { name: /Yêu cầu xem lại/ }).count();
    await shot(page, `p709-B-published-${vp.w}`);
    await noOverflow(page, true);
    out[`B${vp.w}small`] = await page.locator("main a[href], main button").evaluateAll((els) => els.filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && r.height < 44; }).map((e) => (e.textContent ?? "").trim().slice(0, 24)));
    if (vp.w === 375) { const ax = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze(); out.axeB375 = ax.violations.map((v) => ({ id: v.id, n: v.nodes.length })); }
  }
  save();
});
test.skip("@real UI4 sinh viên C: chưa nộp, kiểm tệp, nộp, 375 px", async ({ page, context }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "sv.nguyco");
  await page.goto(`/assignments/${S.aid}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3500);
  const t = await page.locator("main").innerText(); out.C1 = t.slice(0, 1500); out.Cbad = BAD.filter((w) => t.toLowerCase().includes(w.toLowerCase()));
  out.sidebarHasAssignmentEntry = (await page.locator("[data-part=sidebar]").innerText()).split("\n").map((s) => s.trim()).filter((s) => s.includes("Phân tích một vụ")).length;
  await shot(page, "p703-C-before");
  const presigns: string[] = []; page.on("request", (r) => { if (/presign/.test(r.url())) presigns.push(r.url()); });
  const inp = page.locator("input[type=file]").first();
  const cases: [string, Buffer, string][] = [["bai.exe", Buffer.from("MZ" + "x".repeat(50)), "exe"], ["bai.pdf", Buffer.alloc(0), "rong"], ["bai.PDF", Buffer.from("%PDF-1.4\n" + "x".repeat(200)), "PDF hoa"]];
  const errs: Record<string, string> = {};
  for (const [name, buf, tag] of cases) { await inp.setInputFiles({ name, mimeType: "application/octet-stream", buffer: buf }); await page.waitForTimeout(800); errs[tag] = (await page.locator("main").innerText()).split("\n").filter((l) => /Chỉ nhận|lớn hơn|rỗng|một tệp|không hợp lệ/i.test(l)).join(" / "); }
  await inp.setInputFiles({ name: "big.pdf", mimeType: "application/pdf", buffer: Buffer.alloc(20 * 1024 * 1024 + 1, 65) }); await page.waitForTimeout(1500); errs.big = (await page.locator("main").innerText()).split("\n").filter((l) => /Chỉ nhận|lớn hơn|rỗng|một tệp/i.test(l)).join(" / ");
  out.errs = errs; out.presignsBeforeValid = presigns.length;
  await shot(page, "p703-C-errors");
  // nộp hợp lệ .md
  await inp.setInputFiles({ name: "bai.md", mimeType: "text/markdown", buffer: Buffer.from("# Bài\n" + "Nội dung bài làm. SUNBURST chờ 10 ngày. ".repeat(10)) }); await page.waitForTimeout(1000);
  const sub = page.getByRole("button", { name: /Nộp bài/ }); out.submitBtn = await sub.count();
  if (await sub.count()) { await sub.first().click(); await page.waitForTimeout(5000); }
  out.C2 = (await page.locator("main").innerText()).slice(0, 900); out.presignsAfterValid = presigns.length;
  await shot(page, "p703-C-after");
  // 375 px
  await page.setViewportSize({ width: 375, height: 812 }); await page.goto(`/assignments/${S.aid}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(3000);
  out.drop375 = await page.evaluate(() => { const e = document.querySelector("input[type=file]")?.closest("label, div") as HTMLElement | null; const r = e?.getBoundingClientRect(); return { h: r ? Math.round(r.height) : null, scrollW: document.documentElement.scrollWidth, innerW: window.innerWidth, aria: e?.getAttribute("aria-label") ?? document.querySelector("input[type=file]")?.getAttribute("aria-label"), live: document.querySelectorAll("[aria-live]").length }; });
  const ax = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa"]).analyze(); out.axeC375 = ax.violations.map((v) => ({ id: v.id, n: v.nodes.length }));
  await shot(page, "p703-C-375");
  save();
});
