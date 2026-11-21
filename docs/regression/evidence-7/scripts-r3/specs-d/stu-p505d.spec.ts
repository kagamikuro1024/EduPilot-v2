import { test, type Page } from "@playwright/test";
import { go, FX, shot, log, psql, newUser, expect } from "../specs/lib";
const R = (tc: string, ok: boolean, note: string) => log("p505", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
const I = (tc: string, note: string) => log("p505", `INFO ${tc} ${note}`);
const FP = FX("fx_p5") as { C: string; SV: string }; const FM = { M: FP.C, M2: FP.C };
const uid = (u: string) => psql(`select id from users where email='${u}@edupilot.local'`);
const SV = FP.SV; const P = `/students/${SV}?course=${FM.M}`;
const aud = () => psql(`select count(*) from audit_log where course_id='${FM.M}' and entity='student_profile' and entity_id::text='${SV}' and action='view'`);
const auditAny = () => psql(`select count(*) from audit_log where course_id='${FM.M}' and created_at>now()-interval '10 minutes'`);

test("03 mở 3 lần = 3 request, 3 dòng audit", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3pg"); const reqs: string[] = []; page.on("request", (r) => { if (/\/students\/[^/]+\/profile$/.test(r.url()) && r.method() === "GET") reqs.push(r.url()); });
  const actions0 = psql(`select string_agg(distinct action,',') from audit_log where course_id='${FM.M}'`); const a0 = aud();
  for (let i = 0; i < 3; i++) { await go(page, P); await page.getByRole("tab", { name: "Tổng quan" }).waitFor({ timeout: 20000 }); await page.waitForTimeout(800); await page.goBack().catch(() => undefined); await page.waitForTimeout(800); }
  const n3 = reqs.length; const a3 = aud();
  await go(page, `/students?course=${FM.M}`); await page.waitForTimeout(800); await page.getByText("QC3 qc3p5s01").first().click().catch(() => undefined); await page.waitForTimeout(1500); await page.goBack(); await page.waitForTimeout(800);
  const n = reqs.length; const a1 = aud(); const acts = psql(`select action||':'||count(*) from audit_log where course_id='${FM.M}' and created_at>now()-interval '3 minutes' group by action`).replace(/\n/g, "; ");
  R("TC-P5-05-03", n3 === 3 && Number(a3) - Number(a0) === 3 && reqs.length === 4 && Number(a1) - Number(a0) === 4, `mở /students/<uid> 3 lần bằng URL, mỗi lần quay lại bằng Back: số request GET …/profile = ${n3}, audit_log (entity=student_profile, action=view) ${a0} → ${a3} (+${Number(a3) - Number(a0)}); thêm 1 lần mở bằng bấm dòng ở /students: request ${n}, audit ${a1} (+${Number(a1) - Number(a0)}) — không cache (staleTime 0, refetchOnMount always, gcTime 0); các action 3 phút gần nhất: ${acts}`);
  await ctx.close();
});

test("12/30/31 ghi chú", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3pg"); await go(page, P); await page.getByRole("tab", { name: "Ghi chú" }).click(); await page.waitForTimeout(1000);
  const mark = (await page.locator("main").innerText()).match(/Chỉ [^\n]*thấy[^\n]*/)?.[0] ?? ""; const ta = page.getByLabel("Ghi chú mới");
  await ta.fill("Nháp QC3 đang gõ dở"); await page.waitForTimeout(2500);
  const keys = await page.evaluate(() => Object.keys(localStorage).filter((k) => k.includes("obs:")));
  await ctx.setOffline(true); await page.getByRole("button", { name: "Lưu ghi chú" }).click(); await page.waitForTimeout(6000); const err = (await page.locator("main").innerText()).match(/Chưa lưu được ghi chú[^\n]*/)?.[0] ?? ""; const keep = await ta.inputValue(); const retryBtn = await page.getByRole("button", { name: "Thử lại" }).count(); const saveBtn = await page.getByRole("button", { name: "Lưu ghi chú" }).isEnabled();
  await ctx.setOffline(false); await page.getByRole("button", { name: "Thử lại" }).first().click().catch(() => undefined); await page.waitForTimeout(3000); const saved = psql(`select count(*) from staff_observations where student_id='${SV}' and body='Nháp QC3 đang gõ dở'`.replace("staff_observations", "student_observations"));
  R("TC-P5-05-12", /thấy/.test(mark) && keys.length >= 1 && keep === "Nháp QC3 đang gõ dở" && /vẫn còn trong ô/.test(err) && retryBtn >= 1 && saveBtn, `tab Ghi chú: «${mark}»; ô nhập tự lưu nháp: khoá localStorage ${JSON.stringify(keys.map((k) => k.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/g, "<uuid>")))}; offline bấm Lưu: «${err}», chữ trong ô giữ=«${keep}», nút Thử lại=${retryBtn}, nút Lưu hoàn lại=${saveBtn}; online + Thử lại → DB ${saved} ghi chú`);
  // 30: sửa đồng thời
  const t2 = await newUser(browser, "qc3pg"); await go(t2.page, P); await t2.page.getByRole("tab", { name: "Ghi chú" }).click(); await t2.page.waitForTimeout(1200);
  await page.reload(); await page.getByRole("tab", { name: "Ghi chú" }).click(); await page.waitForTimeout(1500);
  const nid = psql(`select id from student_observations where student_id='${SV}' order by created_at desc limit 1`);
  const rowA = page.locator("[data-part=observation]").first(); const rowB = t2.page.locator("[data-part=observation]").first();
  const menuA = await rowA.getByRole("button").allInnerTexts(); await rowA.getByRole("button", { name: /Sửa/ }).first().click().catch(() => undefined);
  await page.getByLabel("Sửa ghi chú").fill("Bản sửa của GV"); await page.getByRole("button", { name: /^Lưu/ }).last().click(); await page.waitForTimeout(2000);
  const viaB = await rowB.getByRole("button", { name: /Sửa/ }).first().click().then(() => true).catch(() => false);
  if (viaB) { await t2.page.getByLabel("Sửa ghi chú").fill("Bản sửa của TA"); await t2.page.getByRole("button", { name: /^Lưu/ }).last().click(); await t2.page.waitForTimeout(2500); }
  const bt = (await t2.page.locator("main").innerText()); const showOther = /Bản sửa của GV/.test(bt); const keepMine = /Giữ bản của tôi/.test(bt);
  R("TC-P5-05-30", viaB && showOther && keepMine, `hai người sửa cùng ghi chú (qc3t sửa «Bản sửa của GV» trước; qc3ta sửa từ bản cũ): nút ở hàng: ${JSON.stringify(menuA)}; TA bấm sửa=${viaB}; TA thấy bản của người kia «Bản sửa của GV»=${showOther} + nút «Giữ bản của tôi»=${keepMine} (409 VERSION_CONFLICT → hiện bản hiện hành để chọn); mất mạng giữ nháp: xem TC-P5-05-12`);
  await t2.ctx.close();
  // 31: mở hồ sơ khi mạng lỗi
  const before = psql(`select count(*) from audit_log where course_id='${FM.M}'`);
  await page.route(/\/students\/[^/]+\/profile$/, (r) => r.abort("failed")); await page.goto(P); await page.getByRole("button", { name: "Thử lại" }).waitFor({ timeout: 40000 }); const et = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 260); const after1 = psql(`select count(*) from audit_log where course_id='${FM.M}'`);
  await page.unrouteAll(); await page.route(/\/students\/[^/]+\/profile$/, (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) })); await page.reload(); await page.getByRole("button", { name: "Thử lại" }).waitFor({ timeout: 40000 }); const after2 = psql(`select count(*) from audit_log where course_id='${FM.M}'`);
  await page.unrouteAll(); await page.getByRole("button", { name: "Thử lại" }).click(); await page.getByRole("tab", { name: "Tổng quan" }).waitFor({ timeout: 20000 }); const after3 = psql(`select count(*) from audit_log where course_id='${FM.M}'`);
  R("TC-P5-05-31", /không bị mất/.test(et) && before === after1 && after1 === after2 && Number(after3) === Number(after2) + 1, `abort / 500 ở …/profile: «${et}»; audit_log của lớp ${before} → ${after1} (abort) → ${after2} (500): không dòng mới khi dữ liệu chưa trả về; Thử lại thành công → ${after3} (+1)`);
  await ctx.close();
});

test("14/15 giao diện hồ sơ", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3pg"); await go(page, P); await page.getByRole("tab", { name: "Tổng quan" }).waitFor();
  const t0 = await page.locator("main").innerText(); const ovSvg = await page.evaluate(() => Array.from(document.querySelectorAll("main svg")).filter((e) => e.getBoundingClientRect().width >= 100).length); const noRisk = !/Cần chú ý|Đang theo dõi/.test(t0);
  const hdr = await page.evaluate(() => ({ h1: document.querySelector("main h1")?.textContent, btns: Array.from(document.querySelectorAll("main button")).filter((b) => b.getBoundingClientRect().width > 0 && b.getBoundingClientRect().bottom <= (document.querySelector("[role=tablist]") as HTMLElement).getBoundingClientRect().top).map((b) => (b as HTMLElement).innerText.trim()).filter(Boolean) }));
  const tabs = await page.getByRole("tab").allInnerTexts(); const ov = await page.locator("main section").first().innerText(); const kpi = await page.evaluate(() => Array.from(document.querySelectorAll("main *")).filter((e) => { const s = getComputedStyle(e); return s.display === "grid" && s.gridTemplateColumns.split(" ").length >= 3 && e.children.length >= 3 && Array.from(e.children).every((c) => /^\s*[\d.,]+\s*$/.test((c as HTMLElement).innerText.split("\n")[0] ?? "")); }).length);
  await shot(page, "p505-14");
  const svgs: Record<string, number> = {}; const tab = async (n: string) => { await page.getByRole("tab", { name: n }).click(); await page.waitForTimeout(1500); svgs[n] = await page.evaluate(() => Array.from(document.querySelectorAll("main svg")).filter((e) => e.getBoundingClientRect().width >= 100).length); return (await page.locator("main").innerText()).replace(/\n+/g, " | "); };
  const att = await tab("Chuyên cần"); const grd = await tab("Điểm"); const grdBtn = await page.getByRole("link", { name: "Xác nhận công thức" }).count(); const act = await tab("Hoạt động học"); const charts = svgs["Hoạt động học"];
  psql(`insert into student_risk(course_id,student_id,level,reasons,weeks,computed_at,entered_at) values('${FM.M}','${SV}','AT_RISK','[]','[]',now(),now()) on conflict (course_id,student_id) do update set level='AT_RISK',entered_at=now(),computed_at=now()`);
  await go(page, P); await page.getByRole("tab", { name: "Tổng quan" }).waitFor(); const riskTxt = await page.locator("main").innerText(); const hasRisk = /Cần chú ý|nguy cơ|Đang theo dõi|vắng/.test(riskTxt.split("Tổng quan")[0]);
  psql(`update student_risk set level='WATCH', entered_at=null where course_id='${FM.M}' and student_id='${SV}'`); await go(page, P); await page.getByRole("tab", { name: "Tổng quan" }).waitFor(); const riskW = (await page.locator("main").innerText()).split("Tổng quan")[0]; const hasWatch = /Cần chú ý|nguy cơ|Đang theo dõi|vắng/.test(riskW);
  psql(`delete from student_risk where course_id='${FM.M}' and student_id='${SV}'`);
  R("TC-P5-05-14", hasWatch && hdr.btns.length === 2 && ovSvg === 0 && svgs["Chuyên cần"] === 0 && svgs["Điểm"] === 0 && noRisk && hdr.btns.includes("Thêm ghi chú") && hdr.btns.includes("Nhắn sinh viên") && tabs.join("|") === "Tổng quan|Chuyên cần|Điểm|Hoạt động học|Ghi chú" && /;/.test(ov) && kpi === 0 && /Buổi/.test(att) && /Điểm cộng đã ghi/.test(grd) && /Lớp chưa có công thức điểm đã xác nhận/.test(grd) && grdBtn === 1 && /Câu hỏi tuần trước/.test(act) && charts >= 1, `đầu trang: «${hdr.h1}» + nút phụ ${JSON.stringify(hdr.btns)}; không có khối rủi ro khi NONE=${noRisk}, khi AT_RISK (SQL) khối hiện=${hasRisk}, khi WATCH khối hiện=${hasWatch} («${riskW.replace(/\n+/g, " | ").slice(0, 120)}»); biểu đồ (svg ≥100 px) theo tab ${JSON.stringify({ "Tổng quan": ovSvg, ...svgs })}; tab ${JSON.stringify(tabs)}; Tổng quan: «${ov.replace(/\n+/g, " | ").slice(0, 200)}» (thẻ KPI lưới≥3 ô số=${kpi}); Chuyên cần: «${att.slice(0, 120)}»; Điểm: «${grd.slice(0, 200)}» nút «Xác nhận công thức»=${grdBtn}; Hoạt động học: biểu đồ ${charts}, «${act.slice(0, 160)}»`);
  // 15
  await page.route(/\/students\/[^/]+\/profile$/, async (r) => { await new Promise((x) => setTimeout(x, 3000)); return r.continue(); }); await page.goto(P); await page.waitForTimeout(1200); const sk = await page.evaluate(() => document.querySelectorAll("[class*=skeleton],[data-skeleton],[aria-busy=true]").length); await page.getByRole("tab", { name: "Tổng quan" }).waitFor({ timeout: 30000 }); await page.unrouteAll();
  await page.route(/\/students\/[^/]+\/profile$/, (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) })); await page.reload(); await page.getByRole("button", { name: "Thử lại" }).waitFor({ timeout: 30000 }); const err = (await page.locator("main").innerText()).replace(/\n+/g, " | ").slice(0, 200); await page.unrouteAll(); await page.reload(); await page.getByRole("tab", { name: "Tổng quan" }).waitFor();
  await page.setViewportSize({ width: 720, height: 900 }); await page.waitForTimeout(800); const sw = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1); const ovt = await page.locator("main").innerText(); const tuan = /Tuần trước/.test(ovt);
  const desc = await page.evaluate(() => Array.from(document.querySelectorAll("main section")).map((s) => { const h = s.querySelector("h2,h3"); const n = h?.nextElementSibling as HTMLElement | null; return n && n.tagName === "P" && !n.className.includes("narrative") ? n.innerText.slice(0, 40) : null; }).filter(Boolean));
  R("TC-P5-05-15", sk >= 1 && /Thử lại/.test(err) && sw && tuan && desc.length === 0, `tải chậm (3 s): khung xương=${sk}; lỗi 500: «${err}»; 720×900: không tràn ngang=${sw}; chữ «Tuần trước» ở khối hoạt động=${tuan}; câu mô tả dưới tiêu đề khối=${JSON.stringify(desc)}`);
  await ctx.close();
});

test("20 nháp AI giữ ở máy", async ({ browser }) => {
  const { ctx, page } = await newUser(browser, "qc3pg"); await go(page, P); await page.getByRole("tab", { name: "Ghi chú" }).click(); await page.waitForTimeout(1000);
  await page.getByRole("button", { name: "Soạn nhận xét tổng hợp" }).click(); const ta = page.getByLabel("Nhận xét tổng hợp (bản nháp AI)"); await ta.waitFor({ timeout: 90000 }).catch(() => undefined); const v1 = await ta.inputValue().catch(() => ""); const notice = (await page.locator("main").innerText()).match(/Chưa soạn được[^\n]*|chưa có dữ liệu[^\n]*/)?.[0] ?? "";
  await page.waitForTimeout(2500); const keys = await page.evaluate(() => Object.keys(localStorage).filter((k) => k.includes("aisum:")));
  await page.reload(); await page.getByRole("tab", { name: "Ghi chú" }).click(); await page.waitForTimeout(1500); const v2 = await page.getByLabel("Nhận xét tổng hợp (bản nháp AI)").inputValue().catch(() => "");
  R("TC-P5-05-20", v1.length > 20 && v2 === v1 && keys.length >= 1, `Soạn nhận xét tổng hợp (LLM thật qua 9router): nháp ${v1.length} ký tự${notice ? ` (thông báo: ${notice})` : ""}; khoá localStorage ${JSON.stringify(keys.map((k) => k.replace(/[0-9a-f]{8}-[0-9a-f-]{27}/g, "<uuid>")))}; tải lại trang: nháp còn nguyên=${v2 === v1} (${v2.length} ký tự)`);
  await ctx.close();
});
