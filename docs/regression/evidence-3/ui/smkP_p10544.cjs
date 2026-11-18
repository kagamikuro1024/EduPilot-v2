// smkP: TC-P105-44 (AC11 không từ kỹ thuật ở /settings/llm) sau R3-01 (ffb46656). node smkP_p10544.cjs
const L = require('../../evidence-5/ui/ui_lib.cjs'); const { chromium, login, B, fs } = L;
const RE = /\b(RAG|PII|trace|fallback|embedding|prompt)\b/i;
(async () => { const br = await chromium.launch({ headless: true }); const out = {};
  for (const [u, w] of [['admin', 1440], ['admin', 390], ['teacher', 1440]]) {
    const { ctx, p } = await login(br, u, { width: w, height: 900 }); await p.goto(B + '/settings/llm'); await p.waitForTimeout(4000);
    const txt = []; const grab = async (tag) => { txt.push([tag, await p.evaluate(() => document.body.innerText)]); };
    await grab('trang');
    for (const d of await p.locator('details:not([open]) > summary').all()) { await d.click().catch(() => {}); }
    await p.waitForTimeout(600); await grab('mở details');
    for (const b of await p.getByRole('button', { name: /nâng cao|Hiện thêm|Mở rộng/i }).all()) { await b.click().catch(() => {}); }
    await p.waitForTimeout(600); await grab('nâng cao');
    const names = await p.locator('button:visible, [role=button]:visible').evaluateAll((es) => [...new Set(es.map((e) => (e.getAttribute('aria-label') || e.innerText).trim()).filter(Boolean))]);
    for (const n of names.filter((n) => /^(Thêm|Sửa|Chỉnh|Đổi|Xoá|Tạo|Kiểm tra|Test)/.test(n)).slice(0, 8)) { const b = p.getByRole('button', { name: n, exact: true }).first(); if (!(await b.count())) continue; await b.click({ timeout: 3000 }).catch(() => {}); await p.waitForTimeout(700); if (await p.locator('dialog[open],[role=dialog],[aria-modal=true]').count()) { await grab('hộp thoại ' + n); await p.keyboard.press('Escape'); await p.waitForTimeout(300); } }
    const hits = []; for (const [tag, t] of txt) { const m = t.match(new RegExp(RE.source, 'gi')); if (m) hits.push([tag, [...new Set(m)]]); }
    const usage = /Mức dùng theo tác vụ/.test(txt[0][1]); const idx = (txt[0][1].match(/Đánh chỉ mục[^\n]*/) || [null])[0]; out[u+w+'_idx'] = idx; const status = await p.locator('[role=status]').allInnerTexts();
    out[u + w] = { hits, usage, names, statusSuccess: status.filter((s) => /Thành công/.test(s)).length, dialogs: txt.map((x) => x[0]) };
    const verbs = names.filter((n) => !/^[A-ZĐÀ-Ỵa-zà-ỹ]/.test(n)); out[u + w].nonVerbSample = names.slice(0, 30);
    await ctx.close(); }
  console.log(JSON.stringify(out, null, 1)); fs.writeFileSync(__dirname + '/../logs/smkP-p10544.json', JSON.stringify(out, null, 1)); await br.close(); })();
