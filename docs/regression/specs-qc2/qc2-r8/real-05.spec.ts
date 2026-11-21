import { test } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { go, login } from "./lib";
test.setTimeout(150_000);
const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], { encoding: "utf8" }).trim();
const OUT = "/tmp/qc2-s8r/ui05.tsv";
const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 400)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note}`); };
test("[05-42] UI tài liệu nguồn đã gỡ", async ({ browser }) => {
  const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage(); await login(p, "gv.s8f");
  const id = sql("select id from courses where class_code='QD8H3'");
  const errs: string[] = []; p.on("response", (r) => { if (r.status() >= 500) errs.push(r.status() + " " + r.url()); });
  await go(p, `/gradebook/scheme?course=${id}`); await p.waitForTimeout(3000);
  const t = (await p.locator("main").innerText()).replace(/\s+/g, " ");
  const btn = p.getByRole("button", { name: /^Trang \d$/ }).first(); let pane = "";
  if (await btn.count()) { await btn.click(); await p.waitForTimeout(1500); pane = (await p.locator("[data-part=citation-pane]").first().innerText().catch(() => "")).replace(/\s+/g, " "); }
  res("TC-P6-05-42-ui", /Tài liệu nguồn đã gỡ/.test(t + pane) && errs.length === 0, `QD8H3 sau khi xoá tài liệu: trang công thức không lỗi 5xx (${errs.length}); chữ «Tài liệu nguồn đã gỡ» ${/Tài liệu nguồn đã gỡ/.test(t + pane)}; cột trích dẫn «${pane.slice(0, 120)}»; main «${t.slice(0, 120)}»`);
});

test("[05-13,05-20,05-21] banner: hết sau xác nhận; điểm danh 30 SV ≤ 60 s khi banner hiện; lớp CONFIRMED không banner", async ({ browser }) => {
  const mk = async (u: string) => { const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p = await c.newPage(); await login(p, u); return p; };
  const h3 = sql("select id from courses where class_code='QD8H3'"), fm5 = sql("select id from courses where class_code='QD8FM5'"), r6 = sql("select id from courses where class_code='QD8R6U'");
  const BAN = /Công thức điểm chưa xác nhận/;
  const t = await mk("gv.s8f");
  await go(t, `/gradebook?course=${h3}`); await t.waitForTimeout(2500); const g13 = await t.locator("main").innerText();
  res("TC-P6-05-13", !BAN.test(g13), `QD8H3 sau xác nhận, tải lại /gradebook: banner «Công thức điểm chưa xác nhận» ${BAN.test(g13) ? "CÒN" : "đã biến mất"}`);
  const tt = await mk("teacher"); await go(tt, `/attendance?course=${r6}`); await tt.waitForTimeout(2500); const a21 = await tt.locator("main").innerText();
  res("TC-P6-05-21", !BAN.test(a21) && /Sinh Viên|sinh viên|Có mặt/i.test(a21), `/attendance lớp CONFIRMED (QD8R6U): banner ${BAN.test(a21) ? "CÓ" : "không"}; lưới điểm danh có ${/Có mặt/i.test(a21)}`);
  await go(t, `/attendance?course=${fm5}`); await t.waitForTimeout(3000);
  const banner = await t.getByText(BAN).first().isVisible().catch(() => false); const bb = await t.getByText(BAN).first().boundingBox().catch(() => null);
  const rows = t.locator("[data-part=student-row]"); const n = await rows.count(); const fr = await rows.first().boundingBox();
  let ms = -1; let marked = 0; let dlg = 0; let putOk = "";
  if (n > 0) {
    const puts: string[] = []; t.on("response", (r) => { if (r.request().method() === "PUT" && /attendance/.test(r.url())) puts.push(r.status() + ""); });
    await t.getByLabel(/Bảng điểm danh/).focus(); const t0 = Date.now();
    for (let i = 0; i < 30; i++) { await t.keyboard.press(String(1 + (i % 4))); marked++; }
    for (let k = 0; k < 40 && puts.length === 0; k++) await t.waitForTimeout(250);
    ms = Date.now() - t0; dlg = await t.locator("dialog[open]").count(); putOk = puts.join(",");
  }
  const covers = !!(bb && fr && bb.y + bb.height > fr.y);
  res("TC-P6-05-20", banner && n >= 30 && ms >= 0 && ms <= 60000 && /200/.test(putOk) && dlg === 0 && !covers, `QD8FM5 chưa xác nhận: banner hiện ${banner}; ${n} hàng; bấm phím 1–4 cho ${marked} SV trong ${ms} ms (≤ 60000); PUT điểm danh ${putOk || "không có"}; hộp thoại ${dlg}; banner che lưới ${covers}`);
});
