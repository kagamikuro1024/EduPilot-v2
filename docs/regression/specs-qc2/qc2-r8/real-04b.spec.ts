import { test, type Page } from "@playwright/test";
import { appendFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { go, login, EV } from "./lib";
test.setTimeout(240_000);
const OUT = "/tmp/qc2-s8r/ui04b.tsv";
const res = (tc: string, ok: boolean, note: string) => { appendFileSync(OUT, `${tc}\t${ok ? "P" : "F"}\t${note.replace(/\s+/g, " ").slice(0, 420)}\n`); console.log(`${ok ? "P" : "F"} ${tc} ${note}`); };
const sql = (q: string) => execFileSync("docker", ["exec", "ep-r-postgres-1", "sh", "-c", `psql -U edupilot -d edupilot -At -c "${q.replace(/"/g, '\\"')}"; exit 0`], { encoding: "utf8" }).trim();
const PDF = process.env.QC_PDF ?? process.env.QC_REPO + "/seed/demo/quy-che-lop1-v2.pdf";
const CODE = "QD8FM5";
async function open(browser: any) { const c = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" }); const p: Page = await c.newPage(); await login(p, "gv.s8f"); const id = sql(`select id from courses where class_code='${CODE}'`); await go(p, `/gradebook/scheme?course=${id}`); await p.waitForTimeout(3000); return { p, id }; }
const state = (id: string) => sql(`select (select count(*) from documents where course_id='${id}'), (select string_agg(status||':'||version, ',' order by version) from grade_schemes where course_id='${id}')`);
async function pick(p: Page) { const [fc] = await Promise.all([p.waitForEvent("filechooser"), p.getByRole("button", { name: /Tải quy chế/ }).first().click()]); await fc.setFiles(PDF); }
test("[04-31] tải lên hỏng giữa chừng", async ({ browser }) => {
  const { p, id } = await open(browser); const before = state(id);
  await p.route((u) => !/\/api\//.test(u.pathname) && /minio|edupilot|\.pdf|X-Amz/i.test(u.toString()), (r) => (r.request().method() === "PUT" ? r.abort("connectionreset") : r.continue()));
  await pick(p); await p.waitForTimeout(6000);
  const t = (await p.locator("main").innerText()).replace(/\s+/g, " "); const retry = await p.getByRole("button", { name: /^Tải lại$/ }).count(); const after = state(id);
  await p.screenshot({ path: `${EV}/shots/s8r-04-31.png` });
  res("TC-P6-04-31", /quy-che-lop1-v2\.pdf/.test(t) && retry > 0 && before === after, `PUT lên kho bị ngắt: thông báo «${(t.match(/quy-che-lop1-v2\.pdf:[^A-Z]{0,90}/) || ["(không thấy)"])[0]}», nút «Tải lại» ${retry}, tên tệp được giữ ${/quy-che-lop1-v2\.pdf/.test(t)}; DB trước ${before} = sau ${after} (không tài liệu/bản nháp rác)`);
});
test("[04-34] trích FAILED: khối lỗi + Thử lại, bản nháp cũ nguyên vẹn", async ({ browser }) => {
  const { p, id } = await open(browser); const before = state(id); const hadDraft = /Giữa kỳ/.test(await p.locator("main").innerText());
  await pick(p);
  for (let i = 0; i < 20; i++) { await p.waitForTimeout(2000); if (await p.getByRole("button", { name: /Thay bản nháp/ }).count()) break; }
  const replaced = (await p.getByRole("button", { name: /Thay bản nháp/ }).count()) > 0; if (replaced) await p.getByRole("button", { name: /Thay bản nháp/ }).first().click();
  let t = ""; for (let i = 0; i < 60; i++) { await p.waitForTimeout(3000); t = (await p.locator("main").innerText()).replace(/\s+/g, " "); if (await p.getByRole("button", { name: /^Thử lại$/ }).count() && !/Bản nháp đang có chỉnh sửa/.test(t)) break; }
  const retry = await p.getByRole("button", { name: /^Thử lại$/ }).count(); const after = state(id); await p.screenshot({ path: `${EV}/shots/s8r-04-34.png` });
  const err = (t.match(/(quy-che-lop1-v2|qc2-pdf34)\.pdf:[^]{0,140}/) || [""])[0];
  const draftStill = /Giữa kỳ/.test(t) && /Xác nhận công thức|Điền|Trọng số/.test(t);
  res("TC-P6-04-34", retry > 0 && draftStill && hadDraft && /[a-zê]/i.test(err), `trích FAILED (stub JSON hỏng; bấm «Thay bản nháp» ${replaced}): khối lỗi «${err.slice(0, 160)}» + nút Thử lại ${retry}; bản nháp cũ còn hiển thị ${draftStill} (trước ${hadDraft}); DB scheme trước ${before} → sau ${after}`);
});
