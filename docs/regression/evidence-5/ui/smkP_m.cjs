// smkP (qc2) — rerun TC-PE06-27 (QUEUED thật bằng hàng chấm đầy), TC-PE07-18 (đóng tab thật), R5-10 (nhãn «Chấm theo bộ test cũ»). Chạy: ONLY=s27|s18|stale node smkP_m.cjs
const L = require('./ui_lib.cjs'); const { chromium, login, B, R, sql, SH } = L; const fs = require('fs'); const { spawn } = require('child_process');
const FX = JSON.parse(fs.readFileSync('/tmp/qc2L/uix2.json')); const ONLY = process.env.ONLY || 's27';
const SUM = '#include <stdio.h>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n';
const startM = async (p, eid) => { await p.goto(`${B}/exams/${eid}/take?course=${FX.c1}`); await p.waitForTimeout(2500); await L.dismiss(p); const n = p.locator('[data-part=integrity-notice]'); if (await n.count()) await n.scrollIntoViewIfNeeded(); const b = p.getByRole('button', { name: /Bắt đầu làm bài/ }); if (await b.count()) { await b.first().click(); await p.waitForTimeout(3000); } };
const toCode = async (p) => { for (let i = 1; i <= 3; i++) { if (await p.locator('textarea').count()) return true; await p.getByRole('button', { name: new RegExp('^Câu ' + i) }).first().click().catch(() => {}); await p.waitForTimeout(700); } return (await p.locator('textarea').count()) > 0; };
const aidOf = (eid, u) => sql(`select id from exam_attempts where exam_id='${eid}' and student_id=(select id from users where email='${u}@edupilot.local')`);
const main = (p) => p.locator('main').innerText();
const TECH = /QUEUED|RUNNING|\bDONE\b|verdict|sandbox|go-judge|\bSSE\b|\bRAG\b|\bPII\b|provider|trace/i;
(async () => { process.on('unhandledRejection', (e) => console.log('ERR', String(e).slice(0, 300)));
  const br = await chromium.launch({ headless: true });
  if (ONLY === 's27') {
    const u = process.env.U || 'r5s31'; const { ctx, p } = await login(br, u, { width: 1440, height: 900 }); await startM(p, FX.xcode); await toCode(p);
    const ta = p.locator('textarea').first(); await ta.fill(SUM); await p.waitForTimeout(2000);
    const fl = spawn('python3', ['../scripts/smkP_flood.py', FX.xcode, process.env.N || '30'], { stdio: 'inherit' }); await new Promise((r) => setTimeout(r, 6000));
    const seq = []; const t0 = Date.now(); const note = (t) => { if (!seq.length || seq[seq.length - 1].t !== t) seq.push({ t, at: Date.now() - t0 }); };
    await p.getByRole('button', { name: /^Nộp lời giải$/ }).click(); let last = '';
    for (let i = 0; i < 400; i++) { const t = await main(p); last = t; const m = []; for (const k of ['Đã nhận, đang xếp hàng', 'Đang chấm', 'Đang chạy']) if (t.includes(k)) m.push(k); const done = /Lần nộp tính điểm|Đúng|Đạt|Sai|Lỗi biên dịch|Biên dịch được/.test(t) && !m.length; note(m.length ? m.join(' + ') : done ? 'KẾT QUẢ' : '…'); if (done && i > 5) break; await p.waitForTimeout(150); }
    await p.getByRole('button', { name: /^Lần nộp$/ }).click().catch(() => {}); await p.waitForTimeout(800); const subs = await main(p); await p.screenshot({ path: SH + '/smkP-pe06-27.png' });
    const order = seq.map((x) => x.t).filter((x) => x !== '…'); const queued = order.includes('Đã nhận, đang xếp hàng');
    const ok = queued && order.includes('KẾT QUẢ') && order.indexOf('KẾT QUẢ') === order.length - 1 && !TECH.test(subs) && new Set(order).size === order.length;
    R('TC-PE06-27', ok ? 'P' : 'F', `smkP @634d59c5: ${u} nộp code khi 30 SV khác đang xếp hàng chấm (smkP_flood.py): chuỗi trạng thái ${JSON.stringify(seq.map((x) => x.t + '@' + (x.at / 1000).toFixed(1) + 's'))}; nhãn «Đã nhận, đang xếp hàng» hiện=${queued}; mỗi trạng thái một lần=${new Set(order).size === order.length}; từ kỹ thuật trong màn=${TECH.test(subs) || TECH.test(last)}`);
    await ctx.close(); }
  if (ONLY === 's18') {
    const res = {};
    for (const [mode, u] of [['navigate', 'r5s23'], ['close', 'r5s24'], ['ctxclose', 'r5s25']]) {
      const { ctx, p } = await login(br, u, { width: 1440, height: 900 }); await startM(p, FX.xcode); await toCode(p); const aid = aidOf(FX.xcode, u); const ta = p.locator('textarea').first(); const SECRET = 'BIMAT_' + Date.now();
      const paste = async (txt) => { await ta.focus(); await p.evaluate((t) => { const dt = new DataTransfer(); dt.setData('text', t); document.activeElement.dispatchEvent(new ClipboardEvent('paste', { clipboardData: dt, bubbles: true, cancelable: true })); }, txt); };
      await p.waitForTimeout(16500); // để lô đầu (nếu có) đi xong, bộ đệm rỗng
      const nBefore = Number(sql(`select count(*) from exam_events where attempt_id='${aid}'`)); const pBefore = Number(sql(`select count(*) from exam_events where attempt_id='${aid}' and type='PASTE'`));
      await paste(SECRET + ' ' + 'x'.repeat(300)); await p.waitForTimeout(500); // bộ đệm có 1 PASTE, chưa tới nhịp 15 s
      if (mode === 'navigate') await p.evaluate(() => { window.location.href = 'about:blank'; }); else if (mode === 'close') await p.close({ runBeforeUnload: true }); else await ctx.close();
      await new Promise((r) => setTimeout(r, 3000)); const nAfter = Number(sql(`select count(*) from exam_events where attempt_id='${aid}'`)); const pAfter = Number(sql(`select count(*) from exam_events where attempt_id='${aid}' and type='PASTE'`));
      const rows = sql(`select meta::text from exam_events where attempt_id='${aid}' and type='PASTE'`); res[mode] = { u, nBefore, nAfter, pBefore, pAfter, leak: rows.includes(SECRET), meta: rows.split('\n').slice(-1)[0] };
      await ctx.close().catch(() => {});
    }
    const ok = Object.values(res).every((x) => x.pAfter > x.pBefore && !x.leak);
    R('TC-PE07-18', ok ? 'P' : 'F', `smkP @634d59c5 (d7fecdc2 R5-07): dán 328 ký tự vào ô mã rồi ĐÓNG THẬT ngay (trước nhịp 15 s), đối chiếu exam_events trong DB sau 3 s: ${JSON.stringify(res)}; Chrome headless thật, close = page.close({runBeforeUnload:true}) / browserContext.close()`);
  }
  if (ONLY === 'stale') {
    const [e, aid] = [process.env.E, process.env.A]; const { ctx, p } = await login(br, 'r5t', { width: 1440, height: 900 }); await p.goto(`${B}/exams/${e}/results?course=${FX.c1}`); await p.waitForTimeout(3000);
    await p.locator('tbody tr').first().click(); await p.waitForTimeout(2000); const dr = (await p.locator('dialog[open]').allInnerTexts()).join(' ');
    const sum = p.locator('dialog[open] summary'); const n = await sum.count(); const txt = []; for (let i = 0; i < n; i++) txt.push((await sum.nth(i).innerText()).replace(/\s+/g, ' '));
    R('TC-PE03-33', /Chấm theo bộ test cũ/.test(dr) ? 'P' : 'F', `smkP UI: GV mở Drawer lượt của bài đã công bố sau khi sửa test ẩn: tóm tắt các lần nộp = ${JSON.stringify(txt)}; có «Chấm theo bộ test cũ»=${/Chấm theo bộ test cũ/.test(dr)}`);
    await ctx.close(); }
  await br.close();
})();
