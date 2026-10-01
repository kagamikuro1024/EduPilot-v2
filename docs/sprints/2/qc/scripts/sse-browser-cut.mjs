// QC GATE-PG TC-GATE-25 — SSE thật trong Chrome: rút mạng 10 s rồi cắm lại, client tự nối lại bằng Last-Event-ID, không mất / không trùng.
// PG.md "Bạn tự kiểm" mục 3. Không có UI ở PG nên "trình duyệt" = fetch + ReadableStream với header Authorization (SRS 7) — mã client nằm trong chính script này.
// Hộp đen: chỉ dùng HTTP. Chạy bằng eval (như sweep.mjs của sprint 1.5), khi stack ở chế độ test (route thử `POST /api/v1/_test/events`):
//   const m = await import('/abs/docs/sprints/2/qc/scripts/sse-browser-cut.mjs');
//   const rows = await m.default(browser, { base: 'https://localhost', token: '<tok STUDENT U1 --ttl 10m>', cwd: '/abs/worktree' });
//   console.log(m.table(rows));
// Kịch bản: mở stream; phát n=1..30 mỗi 1 s từ tiến trình ngoài (curl, không qua trình duyệt); ở n=8 bật OFFLINE (CDP) và abort kết nối đang mở; ở n=18 tắt offline (= 10 s mất mạng).
// Kỳ vọng: client nhận đúng 30 giá trị n duy nhất theo thứ tự; ≥ 1 lần nối lại; lần nối lại gửi Last-Event-ID = id của sự kiện cuối trước khi mất mạng; không `resync`;
// có ≥ 1 lỗi fetch trong thời gian offline (chứng minh mạng thật sự đứt); preflight CORS cho Authorization + Last-Event-ID qua (nếu thiếu, lần nối lại sẽ lỗi).
export const CFG = { total: 30, cutAt: 8, restoreAt: 18, tailMs: 7000, origin: 'http://localhost:3000' };

export default async function run(browser, { base = 'https://localhost', token, cwd = process.cwd(), origin = CFG.origin } = {}) {
  if (!token) throw new Error('thiếu token (tok STUDENT <U1> --ttl 10m)');
  const tab = await browser.open({ name: 'qc-sse-cut', url: origin + '/', viewport: { width: 1200, height: 800 } });
  try {
    return await tab.run(async ({ page }, a) => {
      const { base, token, cwd, CFG } = a;
      const cp = await import('node:child_process');
      const sleep = (ms) => new Promise((s) => setTimeout(s, ms));
      const rows = []; const push = (id, ok, detail) => rows.push({ id, ok: !!ok, detail: detail || '' });
      const cdp = page.createCDPSession ? await page.createCDPSession() : await page.target().createCDPSession();
      await cdp.send('Network.enable'); await cdp.send('Security.setIgnoreCertificateErrors', { ignore: true });
      const net = (offline) => cdp.send('Network.emulateNetworkConditions', { offline, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
      await net(false);

      await page.evaluate((base, token) => {
        const S = (window.__sse = { events: [], control: [], status: [], errors: [], sentLastIds: [], lastId: null, reconnects: 0, stop: false, ctl: null });
        const parse = (blk) => {
          let id = null, ev = 'message', data = [];
          for (const ln of blk.split('\n')) {
            if (ln.startsWith(':')) continue;
            if (ln.startsWith('id: ')) id = ln.slice(4); else if (ln.startsWith('event: ')) ev = ln.slice(7); else if (ln.startsWith('data: ')) data.push(ln.slice(6));
          }
          if (ev === 'ready' || ev === 'message' && !data.length) return;
          if (['reconnect', 'shutdown', 'resync'].includes(ev)) { S.control.push({ ev, data: data.join('\n') }); return; }
          if (id) { S.lastId = id; let d = null; try { d = JSON.parse(data.join('\n')); } catch (e) {} S.events.push({ id, ev, d }); }
        };
        (async () => {
          while (!S.stop) {
            const ctl = new AbortController(); S.ctl = ctl; const headers = { Authorization: 'Bearer ' + token };
            if (S.lastId) { headers['Last-Event-ID'] = S.lastId; S.sentLastIds.push(S.lastId); }
            try {
              const r = await fetch(base + '/api/v1/events', { headers, signal: ctl.signal });
              S.status.push(r.status); if (!r.ok) throw new Error('HTTP ' + r.status);
              const rd = r.body.getReader(), dec = new TextDecoder(); let buf = '';
              for (;;) { const { value, done } = await rd.read(); if (done) break; buf += dec.decode(value, { stream: true }); let i;
                while ((i = buf.indexOf('\n\n')) >= 0) { const blk = buf.slice(0, i); buf = buf.slice(i + 2); parse(blk); } }
            } catch (e) { S.errors.push(String((e && e.message) || e)); }
            if (S.stop) break; S.reconnects++; await new Promise((r) => setTimeout(r, 3000)); // retry: 3000 của server
          }
        })();
      }, base, token);

      for (let i = 0; i < 40 && !(await page.evaluate(() => window.__sse.status.length)); i++) await sleep(250);
      push('K1 stream mở được (HTTP 200)', await page.evaluate(() => window.__sse.status[0]) === 200, 'status=' + (await page.evaluate(() => window.__sse.status[0])));

      const pub = (n) => { try { return cp.execFileSync('curl', ['-sk', '--max-time', '10', '-o', '/dev/null', '-w', '%{http_code}', '-X', 'POST', '-H', 'Authorization: Bearer ' + token,
        '-H', 'Content-Type: application/json', '-d', JSON.stringify({ type: 'test.ping', data: { n } }), base + '/api/v1/_test/events'], { cwd }).toString(); } catch (e) { return 'ERR'; } };
      const pubCodes = [];
      for (let n = 1; n <= CFG.total; n++) {
        if (n === CFG.cutAt) { await net(true); await page.evaluate(() => window.__sse.ctl && window.__sse.ctl.abort()); }
        if (n === CFG.restoreAt) await net(false);
        pubCodes.push(pub(n)); await sleep(1000);
      }
      await net(false); await sleep(CFG.tailMs);
      const S = await page.evaluate(() => { window.__sse.stop = true; return JSON.parse(JSON.stringify({ ...window.__sse, ctl: null })); });

      const ns = S.events.filter((e) => e.ev === 'test.ping').map((e) => e.d && e.d.n);
      const want = Array.from({ length: CFG.total }, (_, i) => i + 1);
      push('K2 phát đủ 30 sự kiện (mọi POST 2xx)', pubCodes.every((c) => /^2/.test(c)), pubCodes.join(','));
      push('K3 nhận đúng 1..30 theo thứ tự (không mất)', JSON.stringify(ns) === JSON.stringify(want), 'nhận: ' + ns.join(' '));
      push('K4 không trùng', new Set(ns).size === ns.length, 'trùng=' + (ns.length - new Set(ns).size));
      push('K5 ≥ 1 lần nối lại', S.reconnects >= 1, 'reconnects=' + S.reconnects);
      const before = S.events.filter((e) => e.d && e.d.n === CFG.cutAt - 1 || e.d && e.d.n === CFG.cutAt).map((e) => e.id);
      push('K6 lần nối lại gửi Last-Event-ID là id sự kiện cuối trước khi mất mạng (n=' + (CFG.cutAt - 1) + ' hoặc ' + CFG.cutAt + ')', S.sentLastIds.length >= 1 && before.includes(S.sentLastIds[0]), 'gửi=' + S.sentLastIds.join(',') + ' · id n=7/8=' + before.join(','));
      push('K7 mạng thật sự đứt (≥ 1 lỗi fetch khi offline)', S.errors.length >= 1, 'lỗi: ' + S.errors.slice(0, 3).join(' | '));
      push('K8 không có resync', !S.control.some((c) => c.ev === 'resync'), JSON.stringify(S.control));
      push('K9 mọi lần nối lại HTTP 200 (không 401/429)', S.status.filter((s) => s !== 200).length === 0, 'status=' + S.status.join(','));
      return rows;
    }, { base, token, cwd, CFG });
  } finally { try { await tab.close(); } catch (e) {} }
}
export const table = (rows) => rows.map((r) => `${r.ok ? 'PASS' : 'FAIL'} ${r.id} — ${r.detail}`).join('\n') + '\n=> ' + (rows.every((r) => r.ok) ? 'ok=true' : 'ok=false');
