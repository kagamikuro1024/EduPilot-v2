const L = require('./ui_lib.cjs'); const { chromium, login, B, sql } = L; const fs = require('fs'); const FX = JSON.parse(fs.readFileSync('/tmp/qc2L/uix2.json'));
(async () => { const br = await chromium.launch({ headless: true }); const u = 'r5s28';
  const { ctx, p: A } = await login(br, u, { width: 1440, height: 900 });
  await A.goto(`${B}/exams/${FX.xmcq}/take?course=${FX.c1}`); await A.waitForTimeout(2500); await L.dismiss(A); const n = A.locator('[data-part=integrity-notice]'); if (await n.count()) await n.scrollIntoViewIfNeeded(); await A.getByRole('button', { name: /Bắt đầu làm bài/ }).first().click(); await A.waitForTimeout(3000);
  const opt = A.locator('input[type=radio],input[type=checkbox]'); await opt.nth(0).click({ force: true }); await A.waitForTimeout(3000);
  const aid = sql(`select id from exam_attempts where exam_id='${FX.xmcq}' and student_id=(select id from users where email='${u}@edupilot.local')`);
  await ctx.setOffline(true); await opt.nth(1).click({ force: true }); await A.waitForTimeout(1500);
  console.log('LS', await A.evaluate((a) => localStorage.getItem('exam:' + a), aid));
  const B2 = await ctx.newPage(); await ctx.setOffline(false); await B2.goto(`${B}/exams/${FX.xmcq}/take?course=${FX.c1}`); await B2.waitForTimeout(4000); await L.dismiss(B2);
  console.log('B banner', (await B2.locator('main').innerText()).match(/[^\n]*(ở nơi khác|chỉ để xem)[^\n]*/g));
  const claim = B2.getByRole('button', { name: /Làm tiếp ở đây/ }); if (await claim.count()) { await claim.first().click(); await B2.waitForTimeout(2500); }
  for (let i = 0; i < 25; i++) { const t = await A.locator('main').innerText(); if (/Bài đang mở ở nơi khác/.test(t)) { console.log('A read-only after', i, 's'); console.log(t.match(/Bài đang mở[\s\S]{0,200}/)[0].replace(/\n/g, ' | ')); break; } await A.waitForTimeout(1000); }
  console.log('LS2', await A.evaluate((a) => localStorage.getItem('exam:' + a), aid)); console.log('srv', sql(`select count(*) from exam_answers where attempt_id='${aid}'`));
  await br.close(); })();
