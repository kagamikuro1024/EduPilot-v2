// QC2 smoke sprint 5 (SmkM) — giao diện thật trên ep-r. Chạy: ONLY=<stage> node ui_m.cjs
// stage: s27 (PE06-27) | s42 (PE06-42) | s13 (PE06-13) | s37 (PE05-37) | s39 (PE06-39) | s18 (PE07-18) | s27t (PE07-27/28)
const L = require('./ui_lib.cjs'); const { chromium, login, B, R, sql, SH, metrics } = L; const fs = require('fs');
const FX = JSON.parse(fs.readFileSync('/tmp/qc2L/uix2.json')); const ONLY = process.env.ONLY || 's27';
const SUM = '#include <stdio.h>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n';
const startM = async (p, eid) => { await p.goto(`${B}/exams/${eid}/take?course=${FX.c1}`); await p.waitForTimeout(2500); await L.dismiss(p); const n = p.locator('[data-part=integrity-notice]'); if (await n.count()) await n.scrollIntoViewIfNeeded(); const b = p.getByRole('button', { name: /Bắt đầu làm bài/ }); if (await b.count()) { await b.first().click(); await p.waitForTimeout(3000); } };
const toCode = async (p) => { for (let i = 1; i <= 3; i++) { if (await p.locator('textarea').count()) return true; await p.getByRole('button', { name: new RegExp('^Câu ' + i) }).first().click().catch(() => {}); await p.waitForTimeout(700); } return (await p.locator('textarea').count()) > 0; };
const aidOf = (eid, u) => sql(`select id from exam_attempts where exam_id='${eid}' and student_id=(select id from users where email='${u}@edupilot.local')`);
const main = (p) => p.locator('main').innerText();
const TECH = /QUEUED|RUNNING|\bDONE\b|verdict|sandbox|go-judge|\bSSE\b|\bRAG\b|\bPII\b|provider|trace/i;
(async () => { process.on('unhandledRejection', (e) => console.log('ERR', String(e).slice(0, 300)));
  const br = await chromium.launch({ headless: true });
  if (ONLY === 's27') { // PE06-27: trạng thái nộp code theo thời gian
    const { ctx, p } = await login(br, 'r5s31', { width: 1440, height: 900 }); await startM(p, FX.xcode); await toCode(p);
    const ta = p.locator('textarea').first(); await ta.fill(SUM); await p.waitForTimeout(3000);
    const seq = []; const note = (t) => { if (!seq.length || seq[seq.length - 1].t !== t) seq.push({ t, at: Date.now() - t0 }); };
    const t0 = Date.now(); await p.getByRole('button', { name: /^Nộp lời giải$/ }).click();
    let last = ''; for (let i = 0; i < 160; i++) { const t = await main(p); last = t; const m = []; for (const k of ['Đã nhận, đang xếp hàng', 'Đang chấm', 'Đang chạy']) if (t.includes(k)) m.push(k); const done = /Lần nộp tính điểm|Đúng|Đạt|Sai|Lỗi biên dịch|AC/.test(t) && !m.length; note(m.length ? m.join(' + ') : done ? 'KẾT QUẢ' : '…'); if (done && i > 5) break; await p.waitForTimeout(150); }
    await p.getByRole('button', { name: /^Lần nộp$/ }).click().catch(() => {}); await p.waitForTimeout(800); const subs = await main(p);
    await p.screenshot({ path: SH + '/m-pe06-27.png' });
    const order = seq.map((x) => x.t).filter((x) => x !== '…'); const ok = order.includes('KẾT QUẢ') && order.indexOf('KẾT QUẢ') === order.length - 1 && !TECH.test(subs) && new Set(order).size === order.length;
    R('TC-PE06-27', ok ? 'P' : 'F', `nộp code (r5s31): chuỗi trạng thái theo thời gian ${JSON.stringify(seq.map((x) => x.t + '@' + (x.at / 1000).toFixed(1) + 's'))}; mỗi trạng thái hiện một lần (không lặp)=${new Set(order).size === order.length}; từ kỹ thuật (QUEUED/RUNNING/DONE/verdict/sandbox…) trong màn=${TECH.test(subs) || TECH.test(last)}; tab Lần nộp: "${subs.replace(/\n/g, ' | ').slice(subs.indexOf('Lần nộp'), subs.indexOf('Lần nộp') + 160)}"`);
    await ctx.close(); }
  if (ONLY === 's27q') { // PE06-27 (nhãn QUEUED): ép trạng thái trả về = QUEUED để xem chữ mà màn hiện
    const { ctx, p } = await login(br, 'r5s33', { width: 1440, height: 900 }); await startM(p, FX.xcode); await toCode(p); const ta = p.locator('textarea').first(); await ta.fill(SUM); await p.waitForTimeout(3000);
    let force = 'QUEUED'; await p.route(/\/code\/[^/]+\/submissions(\?.*)?$/, async (r) => { const resp = await r.fetch(); let j = await resp.json(); if (force && j.items) j.items = j.items.map((x) => ({ ...x, status: force, compile_ok: null, samples: [] })); await r.fulfill({ response: resp, json: j }); });
    await p.getByRole('button', { name: /^Nộp lời giải$/ }).click(); await p.waitForTimeout(3500); await p.getByRole('button', { name: /^Lần nộp$/ }).click().catch(() => {}); await p.waitForTimeout(800); const q = (await main(p)).replace(/\n/g, ' | '); force = null; await p.waitForTimeout(3500); const d = (await main(p)).replace(/\n/g, ' | ');
    console.log('QUEUED ->', q.slice(q.indexOf('Lần nộp'), q.indexOf('Lần nộp') + 200)); console.log('DONE ->', d.slice(d.indexOf('Lần nộp'), d.indexOf('Lần nộp') + 200)); await p.screenshot({ path: SH + '/m-pe06-27q.png' }); await ctx.close(); }
  if (ONLY === 's42') { // PE06-42: chặn SSE → hỏi lại mỗi 2 s
    const { ctx, p } = await login(br, 'r5s32', { width: 1440, height: 900 }); const evs = []; const subReq = [];
    await p.route('**/api/v1/events', (r) => { evs.push(Date.now()); r.abort('failed'); });
    p.on('request', (r) => { if (/\/submissions\//.test(r.url()) || /\/runs\//.test(r.url())) subReq.push(Date.now()); });
    await startM(p, FX.xcode); await toCode(p); const ta = p.locator('textarea').first(); await ta.fill(SUM); await p.waitForTimeout(3000);
    const t0 = Date.now(); await p.getByRole('button', { name: /^Nộp lời giải$/ }).click(); let done = null; let body = '';
    for (let i = 0; i < 90; i++) { body = await main(p); if (/Lần nộp tính điểm|Đúng|Đạt|Sai|Lỗi biên dịch/.test(body) && !/Đang chấm|xếp hàng/.test(body)) { done = (Date.now() - t0) / 1000; break; } await p.waitForTimeout(1000); }
    await p.getByRole('button', { name: /^Lần nộp$/ }).click().catch(() => {}); await p.waitForTimeout(1000); const subs = await main(p);
    const nSubs = Number(sql(`select count(*) from code_submissions where attempt_id='${aidOf(FX.xcode, 'r5s32')}' and kind='SUBMIT'`)); const verdict = sql(`select verdict from code_submissions where attempt_id='${aidOf(FX.xcode, 'r5s32')}' and kind='SUBMIT' order by created_at desc limit 1`);
    const gaps = subReq.slice(1).map((t, i) => Math.round((t - subReq[i]) / 100) / 10).filter((g) => g > 0.3);
    await p.unroute('**/api/v1/events'); await p.waitForTimeout(6000); const after = await main(p); const nSubs2 = Number(sql(`select count(*) from code_submissions where attempt_id='${aidOf(FX.xcode, 'r5s32')}' and kind='SUBMIT'`));
    R('TC-PE06-42', done !== null && nSubs === 1 && nSubs2 === 1 && evs.length <= 8 ? 'P' : 'F', `chặn /api/v1/events (route abort): kết quả hiện sau ${done && done.toFixed(1)} s nhờ hỏi lại GET submissions (khoảng cách giữa các lần hỏi ${JSON.stringify(gaps.slice(0, 6))} s ≈ 2); số lần thử nối /events = ${evs.length} (thời điểm tương đối so với lúc bấm nộp, s: ${JSON.stringify(evs.map((x) => Math.round((x - t0) / 100) / 10))} — một kết nối duy nhất tự thử lại, không mở thêm kênh); lần nộp trong DB = ${nSubs} (verdict ${verdict}), sau bật lại kênh = ${nSubs2}; chữ kết quả "Đúng/Đạt" xuất hiện ${(subs.match(/Lần nộp tính điểm/g) || []).length} lần trong tab Lần nộp`);
    await ctx.close(); }
  if (ONLY === 's13') { // PE06-13: xung đột bản nháp
    const u = 'r5s27'; const { ctx, p: A } = await login(br, u, { width: 1440, height: 900 }); const tabIds = [];
    A.on('request', (r) => { const h = r.headers()['x-exam-tab']; if (h && !tabIds.includes(h)) tabIds.push(h); });
    await startM(A, FX.xcode); await toCode(A); const taA = A.locator('textarea').first(); await taA.fill('// A1 bản đầu\n' + SUM); await A.waitForTimeout(3500);
    const B2 = await ctx.newPage(); B2.on('request', (r) => { const h = r.headers()['x-exam-tab']; if (h && !tabIds.includes(h)) tabIds.push(h); });
    await B2.goto(`${B}/exams/${FX.xcode}/take?course=${FX.c1}`); await B2.waitForTimeout(4000); await L.dismiss(B2); await toCode(B2);
    const bannerB = (await main(B2)).match(/[^\n]*Bài đang mở ở nơi khác[^\n]*/g) || []; const claim = B2.getByRole('button', { name: /Làm tiếp ở đây/ });
    // A đang chỉ đọc? (tab B mở sau: B là nơi viết). Lấy chữ gõ ở B, rồi A claim lại và lưu bản mới
    const stA = (await main(A)).match(/Bài đang mở ở nơi khác/g) || []; let steps = { bannerB: bannerB.length, claimB: await claim.count(), bannerA: stA.length };
    // B viết bản B1
    const taB = B2.locator('textarea').first(); if (await claim.count()) { await claim.first().click(); await B2.waitForTimeout(2500); }
    await taB.fill('// B1 viết ở tab 2\n' + SUM); await B2.waitForTimeout(3500);
    // A (đã cũ) claim lại và gõ với bản cũ
    await A.waitForTimeout(3000); const claimA = A.getByRole('button', { name: /Làm tiếp ở đây/ }); steps.claimA = await claimA.count(); if (await claimA.count()) { await claimA.first().click(); await A.waitForTimeout(2500); }
    await taA.fill('// A2 gõ ở tab 1 với bản cũ\n' + SUM); await A.waitForTimeout(4500);
    const txt = await main(A); const conflict = txt.match(/Bản trên máy chủ mới hơn[^\n]*/)?.[0]; const keepBtn = await A.getByRole('button', { name: /Dùng bản trên máy này/ }).count(); const srvBtn = await A.getByRole('button', { name: /Dùng bản đã lưu/ }).count();
    const typedKept = (await taA.inputValue()).includes('A2 gõ ở tab 1'); await A.screenshot({ path: SH + '/m-pe06-13.png' });
    const aid = aidOf(FX.xcode, u); const srvDraft = sql(`select source from code_drafts where attempt_id='${aid}' order by updated_at desc limit 1`).split('\n')[0];
    let r1 = null, r2 = null;
    if (keepBtn) { await A.getByRole('button', { name: /Dùng bản trên máy này/ }).click(); await A.waitForTimeout(4500); r1 = sql(`select source from code_drafts where attempt_id='${aid}' order by updated_at desc limit 1`).split('\n')[0]; }
    // lần hai: tab 2 (bản cũ) gõ → xung đột; chọn "Dùng bản đã lưu" → ô hiện bản máy chủ (dòng A2)
    let r2txt = null, conflictB = null;
    if (keepBtn) { await B2.bringToFront(); const cB = B2.getByRole('button', { name: /Làm tiếp ở đây/ }); if (await cB.count()) { await cB.first().click(); await B2.waitForTimeout(2500); } await B2.locator('textarea').first().fill('// B2 gõ với bản cũ\n' + SUM); await B2.waitForTimeout(4500);
      conflictB = (await main(B2)).match(/Bản trên máy chủ mới hơn[^\n]*/)?.[0] || null; const keptB = (await B2.locator('textarea').first().inputValue()).includes('B2 gõ với bản cũ');
      if (await B2.getByRole('button', { name: /Dùng bản đã lưu/ }).count()) { await B2.getByRole('button', { name: /Dùng bản đã lưu/ }).click(); await B2.waitForTimeout(3000); r2txt = (await B2.locator('textarea').first().inputValue()).split('\n')[0] + ' (chữ B2 còn lại: ' + keptB + ' trước khi chọn)'; } }
    R('TC-PE06-13', conflict && keepBtn && srvBtn && typedKept && r1 && r1.includes('A2') && conflictB && r2txt && r2txt.startsWith('// A2') ? 'P' : 'F', `2 tab cùng phiên (r5s27): bước ${JSON.stringify(steps)}; sau khi tab 1 gõ với bản cũ: "${conflict}" nút [Dùng bản trên máy này]=${keepBtn} [Dùng bản đã lưu]=${srvBtn}; chữ đang gõ giữ nguyên=${typedKept}; bản nháp máy chủ lúc xung đột: "${srvDraft}"; chọn "Dùng bản trên máy này" → máy chủ "${r1}"; lần hai (tab 2 gõ với bản cũ: "${conflictB}") chọn "Dùng bản đã lưu" → ô hiện "${r2txt}" (kỳ vọng dòng // A2…)`);
    await ctx.close(); }
  if (ONLY === 's37') { // PE05-37: hai tab, tab đầu chuyển chỉ đọc (tab 1 mất mạng bằng CDP riêng, tab 2 vẫn có mạng)
    const u = 'r5s30'; const { ctx, p: A } = await login(br, u, { width: 1440, height: 900 }); await startM(A, FX.xcode); const opt = A.locator('input[type=radio],input[type=checkbox]'); await opt.nth(0).click({ force: true }).catch(() => {}); await A.waitForTimeout(3000); await toCode(A); const ta = A.locator('textarea').first(); await ta.fill('// tab1 đã lưu\n' + SUM); await A.waitForTimeout(3500);
    const cdp = await ctx.newCDPSession(A); const net = async (off) => cdp.send('Network.emulateNetworkConditions', { offline: off, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
    await net(true); await A.getByRole('button', { name: /^Câu 1/ }).first().click().catch(() => {}); await A.waitForTimeout(500); await opt.nth(1).click({ force: true }).catch(() => {}); await A.waitForTimeout(800); await toCode(A); await ta.fill('// tab1 gõ lúc mất mạng CHƯA LƯU\n' + SUM); await A.waitForTimeout(1500);
    const B2 = await ctx.newPage(); await B2.goto(`${B}/exams/${FX.xcode}/take?course=${FX.c1}`); await B2.waitForTimeout(4000); await L.dismiss(B2); const bB = (await main(B2)).match(/[^\n]*(Bài đang mở ở nơi khác|Làm tiếp ở đây|Tab này chỉ để xem)[^\n]*/g) || [];
    const info = { bannerB_afterOpen: bB.length }; const roB = await B2.locator('input[type=radio],input[type=checkbox]').first().isDisabled().catch(() => null); info.radioB_disabled = roB; const claimB = B2.getByRole('button', { name: /Làm tiếp ở đây/ }); info.claimB = await claimB.count(); const t0b = Date.now(); if (await claimB.count()) { await claimB.first().click(); await B2.waitForTimeout(2500); }
    await net(false); let roA = null; for (let i = 0; i < 40; i++) { const tA = await main(A); if (/Bài đang mở ở nơi khác|Tab này chỉ để xem/.test(tA)) { roA = (Date.now() - t0b) / 1000; break; } await A.waitForTimeout(1000); }
    const tA = await main(A); const unsavedMsg = /Có thay đổi chưa lưu — chép ra nếu cần/.test(tA); const keep = (await ta.inputValue()).includes('CHƯA LƯU'); const dis = await ta.evaluate((e) => e.readOnly || e.disabled); const aid = aidOf(FX.xcode, u);
    const claimA = A.getByRole('button', { name: /Làm tiếp ở đây/ }); const cA = await claimA.count(); let back = null; await A.screenshot({ path: SH + '/m-pe05-37.png' }); if (cA) { await claimA.first().click(); await A.waitForTimeout(3500); back = await ta.evaluate((e) => !(e.readOnly || e.disabled)); }
    const srv = sql(`select source from code_drafts where attempt_id='${aid}' order by updated_at desc limit 1`).split('\n')[0];
    R('TC-PE05-37', roA !== null && unsavedMsg && keep && dis && cA > 0 && info.claimB > 0 ? 'P' : 'F', `2 tab cùng phiên (r5s30): tab 1 mất mạng (CDP riêng) chọn đáp án + gõ mã chưa lưu, tab 2 mở sau → ${JSON.stringify(info)} (dải "Bài đang mở ở nơi khác" + chỉ đọc + nút Làm tiếp ở đây); bấm nút ở tab 2, tab 1 có mạng lại → chuyển chỉ đọc sau ${roA && roA.toFixed(1)} s; chữ gõ chưa lưu của tab 1 còn trong ô=${keep}, ô chỉ đọc=${dis}; thông báo "Có thay đổi chưa lưu — chép ra nếu cần"=${unsavedMsg}; tab 1 có nút Làm tiếp ở đây=${cA}, bấm → ô ghi lại được=${back}; bản nháp máy chủ: "${srv}" (không bị tab 1 cũ ghi đè)`);
    await ctx.close(); }
  if (ONLY === 's39') { // PE06-39: hết giờ khi đang gõ
    const u = 'r5s20'; const { ctx, p } = await login(br, u, { width: 1440, height: 900 }); const puts = []; p.on('request', (r) => { if (r.method() === 'PUT' && /\/draft/.test(r.url())) puts.push({ t: Date.now(), body: r.postData() }); });
    await startM(p, FX.xcode1); await toCode(p); const ta = p.locator('textarea').first(); const aid = aidOf(FX.xcode1, u); const d = Number(sql(`select extract(epoch from deadline_at)*1000 from exam_attempts where id='${aid}'`));
    let n = 0, lastTyped = ''; while (Date.now() < d - 1500) { lastTyped = '// ' + (n++) + ' ' + 'k'.repeat(n % 7) + '\n' + SUM; await ta.fill(lastTyped).catch(() => {}); await p.waitForTimeout(250); }
    await p.waitForTimeout(500); const finalVal = await ta.inputValue(); const until = (d + 14000) - Date.now(); await p.waitForTimeout(Math.max(0, until));
    const ro = await ta.evaluate((e) => e.readOnly || e.disabled).catch(() => null); const val = await ta.inputValue().catch(() => ''); const draft = sql(`select source from code_drafts where attempt_id='${aid}' order by updated_at desc limit 1`); const st = sql(`select status,submit_reason from exam_attempts where id='${aid}'`); const sub = sql(`select count(*),max(auto::text),left(max(source),30) from code_submissions where attempt_id='${aid}' and kind='SUBMIT'`);
    const last = puts[puts.length - 1]; const lastBody = last ? JSON.parse(last.body) : {}; const srcOfLast = (lastBody.source || '').split('\n')[0];
    const draftHead = draft.split('\n')[0]; const valHead = val.split('\n')[0]; const typedHead = lastTyped.split('\n')[0];
    await p.screenshot({ path: SH + '/m-pe06-39.png' });
    R('TC-PE06-39', ro === true && draft.trim() === val.trim() && /^1\|true\|/.test(sub) && val.length > 0 ? 'P' : 'F', `bài code 1 phút (r5s20), gõ liên tục tới deadline−1,5 s (gõ cuối: "${typedHead}"): ô chỉ đọc sau hạn=${ro}; chữ còn trong ô="${valHead}" (=chữ gõ cuối: ${valHead === typedHead}); PUT draft cuối cách deadline ${last ? ((d - last.t) / 1000).toFixed(1) : '?'} s (nội dung PUT cuối "${srcOfLast}"; tổng ${puts.length} PUT); bản nháp máy chủ "${draftHead}" (=ô: ${draft.trim() === val.trim()}); lượt ${st}; SUBMIT tự động: ${sub}`);
    await ctx.close(); }
  if (ONLY === 's18') { // PE07-18: gửi gộp sự kiện liêm chính
    const u = 'r5s23'; const { ctx, p } = await login(br, u, { width: 1440, height: 900 }); const reqs = []; p.on('request', (r) => { if (/\/attempts\/[^/]+\/events$/.test(r.url()) && r.method() === 'POST') reqs.push({ t: Date.now(), body: r.postData() }); });
    await startM(p, FX.xcode); await toCode(p); const aid = aidOf(FX.xcode, u); const t0 = Date.now(); const ta = p.locator('textarea').first(); const SECRET = 'BIMAT_DAN_VAO_' + Date.now();
    const paste = async (txt) => { await ta.focus(); await p.evaluate((t) => { const dt = new DataTransfer(); dt.setData('text', t); document.activeElement.dispatchEvent(new ClipboardEvent('paste', { clipboardData: dt, bubbles: true, cancelable: true })); }, txt); };
    await paste(SECRET + ' ' + 'x'.repeat(300)); await p.evaluate(() => { Object.defineProperty(document, 'visibilityState', { get: () => 'hidden', configurable: true }); document.dispatchEvent(new Event('visibilitychange')); }); await p.waitForTimeout(2500);
    await p.evaluate(() => { Object.defineProperty(document, 'visibilityState', { get: () => 'visible', configurable: true }); document.dispatchEvent(new Event('visibilitychange')); }); await ctx.setOffline(true); await p.waitForTimeout(800); await ctx.setOffline(false);
    const before = reqs.length; await p.waitForTimeout(17000); const t_first = reqs.length ? (reqs[0].t - t0) / 1000 : null;
    const batch1 = reqs[0] ? JSON.parse(reqs[0].body).events.map((e) => e.type) : [];
    // gửi khi rời trang: thêm dán rồi đóng trang
    await paste('RỜI_TRANG_' + SECRET.slice(-6) + 'y'.repeat(50)); const nBefore = Number(sql(`select count(*) from exam_events where attempt_id='${aid}'`)); await p.evaluate(() => { window.location.href = 'about:blank'; }); await new Promise((r) => setTimeout(r, 2500));
    const nAfter = Number(sql(`select count(*) from exam_events where attempt_id='${aid}'`)); const rows = sql(`select type,meta::text from exam_events where attempt_id='${aid}' order by occurred_at`);
    const leak = rows.includes(SECRET) || rows.includes('RỜI_TRANG'); const pasteRows = rows.split('\n').filter((l) => l.startsWith('PASTE'));
    R('TC-PE07-18', batch1.length > 0 && t_first !== null && t_first >= 10 && t_first <= 20 && !leak && pasteRows.every((l) => /"chars"/.test(l) && /"item_id"/.test(l)) && nAfter > nBefore ? 'P' : 'F', `r5s23: dán 300+ ký tự (ô mã), ẩn/hiện tab giả lập, mất/có mạng: lô đầu gửi sau ${t_first && t_first.toFixed(1)} s (≈15 s), gồm ${JSON.stringify(batch1)}; số yêu cầu /events ${reqs.length}; dán thêm rồi rời trang (about:blank): sự kiện trong DB ${nBefore}→${nAfter} (${nAfter > nBefore ? 'gửi được khi rời trang' : 'KHÔNG gửi được khi rời trang'}); PASTE chỉ có chars+item_id: ${JSON.stringify(pasteRows.map((l) => l.split('|')[1]).slice(0, 3))}; nội dung dán xuất hiện trong DB: ${leak}`);
    await ctx.close(); }
  if (ONLY === 's27t') { // PE07-27/28: chữ trước khi bắt đầu + dải khi làm
    const { ctx, p } = await login(br, 'r5s25', { width: 1440, height: 900 }); await p.goto(`${B}/exams/${FX.xcode}/take?course=${FX.c1}`); await p.waitForTimeout(3000); await L.dismiss(p);
    const pre = await main(p); const EXACT = 'Trong giờ làm bài, chat AI tạm khoá. Hệ thống ghi lại số lần bạn rời trang hoặc dán nội dung để giảng viên xem khi cần; đây không phải giám thị và không tự trừ điểm của bạn.';
    const sm = p.locator('summary', { hasText: 'Tìm hiểu thêm' }); let learn = ''; let lnk = await sm.count(); if (lnk) { await sm.first().click(); await p.waitForTimeout(800); learn = (await p.locator('details[open]').first().innerText()).replace(/\n/g, ' | ').slice(0, 700); }
    await p.screenshot({ path: SH + '/m-pe07-27-learn.png' }); 
    const chk = await p.locator('input[type=checkbox]').count(); const startB = p.getByRole('button', { name: /Bắt đầu làm bài/ }); const enabledBefore = await startB.first().isEnabled(); const preTxt = await main(p); const tech = preTxt.match(TECH);
    const bgn = await startB.first().boundingBox(); const vh = 900; await startB.first().click(); await p.waitForTimeout(3000); const during = await main(p); const bar = during.includes(EXACT);
    const mm = ['trước khi bắt đầu', 'trong lúc làm'].map(() => 0);
    R('TC-PE07-27', pre.includes(EXACT) && bar && /Tìm hiểu thêm/.test(pre) && /(loại sự kiện|rời trang|thời điểm|độ dài)/i.test(learn) && /(không.*nội dung|nội dung.*không)/i.test(learn) ? 'P' : 'F', `r5s25: màn trước khi bắt đầu có nguyên văn câu minh bạch=${pre.includes(EXACT)}, mục mở rộng "Tìm hiểu thêm" (<summary>, ${lnk}); dải trong lúc làm có nguyên văn=${bar}; trang "Tìm hiểu thêm": "${learn}"`);
    R('TC-PE07-28', enabledBefore && chk === 0 && !tech ? 'P' : 'F', `nút "Bắt đầu làm bài" bấm được ngay (enabled=${enabledBefore}; ô tích=${chk} — không bắt tích); thấy được sau khi cuộn tới khối thông báo; từ cấm RAG/PII/provider/trace trong màn: ${tech ? tech[0] : 'không'}`);
    await ctx.close(); }
  await br.close(); })();
