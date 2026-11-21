const L = require('./ui_lib.cjs'); const { chromium, B, SH, rec } = L;
const C = '01a1df27-62b5-7775-a346-5c602a43e2f9', EM = '01a1df28-122f-7580-b4c2-b7eb0c3506e9';
(async () => { const br = await chromium.launch(); const { ctx, p } = await L.login(br, 'sv.gioi', { width: 1440, height: 900 });
  await p.goto(B + `/exams/${EM}/take?course=${C}`, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(3500); await L.dismiss(p);
  const sums = p.locator('[data-part=student-result] [data-part=result-item] summary'); const n = await sums.count(); for (let i = 0; i < n; i++) await sums.nth(i).click();
  await p.waitForTimeout(600); const t = await p.locator('[data-part=student-result]').innerText(); await p.screenshot({ path: SH + '/pe09-11-A-mcq-opened.png', fullPage: true });
  rec('s5-11-A2', n === 10 && /Đáp án đúng/.test(t) && /Bạn chọn/.test(t) && /10,00 \/ 10,00/.test(t), `A mở ${n} câu: «Bạn chọn» ${(t.match(/Bạn chọn/g) || []).length}, «Đáp án đúng» ${(t.match(/Đáp án đúng/g) || []).length}, giải thích ${(t.match(/Giải thích/g) || []).length}`);
  await ctx.close(); await br.close(); })();
