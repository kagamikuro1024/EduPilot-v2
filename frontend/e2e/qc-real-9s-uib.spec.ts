import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string; aid: string; aid2: string };
const out: Record<string, unknown> = {};
const save = () => writeFileSync("/tmp/q9s/uib.json", JSON.stringify(out, null, 1));
test("@real UIB 02-18 Drawer tạo bài tập: nháp, offline, không rò đáp án, bấm đúp", async ({ page, context }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  const reqBodies: string[] = []; page.on("request", (r) => { if (r.method() !== "GET") reqBodies.push(`${r.method()} ${r.url().split("/api/v1")[1] ?? r.url()} ${(r.postData() ?? "").includes("reference_answer") ? "[reference_answer]" : ""}`); });
  const open = async () => { await page.goto(`/grading?course=${S.course}`); await page.getByText("Chấm bài").first().waitFor(); await page.waitForTimeout(2000); await page.getByRole("tab", { name: "Bài tập" }).click(); await page.waitForTimeout(1200); await page.getByRole("button", { name: /Tạo bài tập/ }).click(); await page.waitForTimeout(1200); };
  await open();
  const T = "Bài QC offline " + Date.now(), D = "Mô tả nháp QC", REF = "ĐÁP-ÁN-CHUẨN-NHÁP-QC";
  await page.getByLabel(/Tiêu đề/).fill(T); await page.getByLabel(/^Mô tả/).fill(D); await page.getByLabel(/Đáp án chuẩn/).fill(REF);
  await context.setOffline(true); await page.waitForTimeout(3500); await context.setOffline(false);
  out.reqsBeforeSave = reqBodies.filter((r) => !r.startsWith("GET")).slice(0, 10);
  await page.reload(); await page.getByText("Chấm bài").first().waitFor(); await page.waitForTimeout(2000);
  await page.getByRole("tab", { name: "Bài tập" }).click(); await page.waitForTimeout(1200);
  const drawerOpen = await page.getByLabel(/Tiêu đề/).count();
  if (!drawerOpen) { await page.getByRole("button", { name: /Tạo bài tập/ }).click(); await page.waitForTimeout(1200); }
  out.drawerReopenedAuto = !!drawerOpen;
  out.restored = { title: (await page.getByLabel(/Tiêu đề/).inputValue()) === T, desc: (await page.getByLabel(/^Mô tả/).inputValue()) === D, ref: (await page.getByLabel(/Đáp án chuẩn/).inputValue()) === REF };
  out.restoredNote = await page.getByText(/khôi phục/).allInnerTexts();
  await shot(page, "p702-drawer-restored");
  save();
});
test("@real UIB 02-18 bấm đúp Tạo + xung đột 409", async ({ page, playwright }) => {
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  const posts: string[] = []; page.on("response", (r) => { if (r.request().method() === "POST" && /\/assignments(\?|$)/.test(r.url())) posts.push(String(r.status())); });
  await page.goto(`/grading?course=${S.course}`); await page.getByText("Chấm bài").first().waitFor(); await page.waitForTimeout(2000);
  await page.getByRole("tab", { name: "Bài tập" }).click(); await page.waitForTimeout(1200);
  await page.getByRole("button", { name: /Tạo bài tập/ }).click(); await page.waitForTimeout(800);
  const T = "Bài QC bấm đúp " + Date.now(); await page.getByLabel(/Tiêu đề/).fill(T);
  await page.getByLabel(/Hạn nộp/).fill("2026-12-30T08:00"); await page.getByLabel("Mã nộp thư").fill("QX" + String(Date.now()).slice(-6));
  await page.getByRole("button", { name: "Thêm tiêu chí" }).click(); await page.waitForTimeout(400);
  for (const [n, sc] of [["1", "5"], ["2", "5"]]) { await page.getByLabel(`Tên tiêu chí ${n}`).fill("Tiêu chí " + n); await page.getByLabel(`Điểm tiêu chí ${n}`).fill(sc); await page.getByLabel(`Hướng dẫn chấm tiêu chí ${n}`).fill("Hướng dẫn " + n); }
  out.total = (await page.locator("dialog[open]").innerText()).match(/Tổng tiêu chí[\s\S]{0,20}/)?.[0]?.replace(/\n/g, " ");
  const create = page.getByRole("button", { name: /^Tạo$/ }); await create.dblclick(); await page.waitForTimeout(3500);
  out.posts = posts; out.created = await page.getByText(T).count(); out.dlgAfter = (await page.locator("dialog[open]").innerText().catch(() => "")).replace(/\n+/g, " | ").slice(-400); save();
  // xung đột: mở bài vừa tạo, sửa mô tả, GV khác (API) sửa trước, rồi Lưu
  const desc = page.getByLabel(/^Mô tả/); await desc.fill("Mô tả của tôi " + Date.now());
  const api = await playwright.request.newContext({ baseURL: process.env.QC_BASE, ignoreHTTPSErrors: true });
  const lg = await api.post("/api/v1/auth/login", { data: { email: "teacher@edupilot.local", password: process.env.QC_PW } }); const tk = (await lg.json()).access_token;
  const H = { Authorization: "Bearer " + tk };
  const list = await (await api.get(`/api/v1/courses/${S.course}/assignments`, { headers: H })).json(); const row = (list.items ?? list.assignments ?? list).find((x: { title: string }) => x.title === T);
  const one = await (await api.get(`/api/v1/courses/${S.course}/assignments/${row.id}`, { headers: H })).json(); const A = one.assignment ?? one;
  const body: Record<string, unknown> = {}; for (const k of ["title", "description", "max_score", "due_at", "allow_late", "late_penalty_per_day", "late_max_days", "appeal_days", "channels", "submission_code", "reference_answer", "rubric", "teams_class_id", "teams_assignment_id", "version"]) body[k] = A[k];
  body.description = "Mô tả của GV khác"; const put = await api.put(`/api/v1/courses/${S.course}/assignments/${row.id}`, { headers: { ...H, "Idempotency-Key": "k" + Date.now() }, data: body }); out.otherPut = put.status();
  const saveBtn = page.getByRole("button", { name: /^Lưu$/ }); const putCodes: string[] = []; page.on("response", (r) => { if (r.request().method() === "PUT" && /assignments\//.test(r.url())) putCodes.push(String(r.status())); });
  await saveBtn.click(); await page.waitForTimeout(2500);
  out.putCodes = putCodes; out.conflictNotice = (await page.locator("dialog[open]").innerText()).split("\n").filter((l) => /vừa được|Giữ của tôi|Dùng của họ/.test(l));
  out.conflictBtns = await page.locator("dialog[open] button").allInnerTexts();
  out.gradeLine = (await page.locator("dialog[open]").innerText()).split("\n").filter((l) => /tính vào điểm|Tính vào/i.test(l));
  await shot(page, "p702-conflict"); save();
});
