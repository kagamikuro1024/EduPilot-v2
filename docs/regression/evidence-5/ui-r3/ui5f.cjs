// qc3 s5 PE09-25: 5 route × 5 bề rộng: AUDIT (ox/cut/ell), TOUCH (≤390, SV), axe serious/critical, đủ 1 h1. ep-r, lớp R5C1, bài F19b (PUBLISHED).
const L = require('./ui_lib.cjs'); const { chromium, B, SH, rec, AxeBuilder, AUDIT_SRC, TOUCH_SRC } = L; const fs = require('fs');
const eid = JSON.parse(fs.readFileSync('/tmp/qc3e/f19b.json')).eid; const C = JSON.parse(fs.readFileSync('/tmp/qc3e/f19.json')).c1;
const WS = [1440, 1024, 720, 719, 375]; const rows = [];
const PL = [['gv', 'r5t', ['/questions', '/exams', `/exams/${eid}`, `/exams/${eid}/results`]], ['sv', 'r5s19', [`/exams/${eid}/take`, '/exams']]];
(async () => { const br = await chromium.launch();
  for (const [role, u, routes] of PL) { const { ctx, p } = await L.login(br, u, { width: 1440, height: 900 });
    for (const w of WS) { await p.setViewportSize({ width: w, height: 800 }); for (const r of routes) { try { await p.goto(B + r + (r.includes('?') ? '&' : '?') + 'course=' + C, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(3000); await L.dismiss(p);
      if (r.endsWith('/take')) { const sums = p.locator('[data-part=student-result] [data-part=result-item] summary'); for (let i = 0; i < await sums.count(); i++) await sums.nth(i).click(); await p.waitForTimeout(500); }
      const a = await p.evaluate(`(${AUDIT_SRC.trim()})`); const opts = await p.evaluate(() => [...document.querySelectorAll('option')].map((o) => o.textContent.trim())); a.cut = a.cut.filter((c) => !opts.some((o) => o.slice(0, 30) === c)); const t = w <= 390 && role === 'sv' ? await p.evaluate(TOUCH_SRC) : []; const ax = (await new AxeBuilder({ page: p }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze()).violations.filter((v) => ['serious', 'critical'].includes(v.impact)).map((v) => v.id + ':' + v.nodes.length); const h1 = await p.locator('h1').count();
      const pass = a.ox <= 0 && !a.cut.length && !a.ell.length && !t.length && !ax.length && h1 === 1; rows.push({ role, r: r.replace(eid, '[id]'), w, ox: a.ox, cut: a.cut.slice(0, 2), ell: a.ell.slice(0, 2), touch: t.slice(0, 2), ax, h1, pass }); if (!pass) console.log('FAIL', JSON.stringify(rows.at(-1)).slice(0, 300));
      if (w === 375 || w === 1440) await p.screenshot({ path: `${SH}/pe09-25-${role}-${r.replace(eid, 'id').replace(/\W+/g, '_')}-${w}.png` });
    } catch (e) { rows.push({ role, r, w, pass: false, err: String(e).slice(0, 100) }); console.log('ERR', r, w, String(e).slice(0, 100)); } } }
    await ctx.close(); }
  fs.writeFileSync(SH + '/../pe09-25-rows.json', JSON.stringify(rows, null, 1)); const f = rows.filter((x) => !x.pass); console.log('rows', rows.length, 'FAIL', f.length);
  await br.close(); })();
