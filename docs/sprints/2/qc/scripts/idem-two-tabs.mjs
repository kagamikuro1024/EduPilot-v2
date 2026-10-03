// QC GATE-PG TC-GATE-27 — gửi đúp một request có Idempotency-Key bằng HAI TAB trình duyệt thật cùng lúc (PG.md "Bạn tự kiểm" mục 4).
// Kỳ vọng: đúng 1 bản ghi `_test_items`; hai response giống hệt (status, thân, Content-Type); đúng một response mang `Idempotent-Replayed: true`
// (header đọc được từ trình duyệt ⇒ kiểm luôn Access-Control-Expose-Headers); nếu một tab trúng 409 IDEMPOTENCY_IN_PROGRESS thì gửi lại sau Retry-After.
// Chế độ test. Chạy bằng eval:
//   const m = await import('/abs/docs/sprints/2/qc/scripts/idem-two-tabs.mjs');
//   console.log(m.table(await m.default(browser, { base: 'https://localhost', token: '<tok STUDENT U1 --ttl 10m>', cwd: '/abs/worktree' })));
export default async function run(browser, { base = 'https://localhost', token, cwd = process.cwd(), origin = 'http://localhost:3000' } = {}) {
  if (!token) throw new Error('thiếu token');
  const key = 'qc-two-tabs-' + Date.now(), name = 'two-tabs-' + Date.now();
  const A = await browser.open({ name: 'qc-idem-a', url: origin + '/', viewport: { width: 900, height: 600 } });
  const B = await browser.open({ name: 'qc-idem-b', url: origin + '/', viewport: { width: 900, height: 600 } });
  try {
    const prep = async ({ page }) => { const c = page.createCDPSession ? await page.createCDPSession() : await page.target().createCDPSession(); await c.send('Security.setIgnoreCertificateErrors', { ignore: true }); return true; };
    await A.run(prep, {}); await B.run(prep, {});
    const send = async ({ page }, a) => page.evaluate(async (base, token, key, name) => {
      const r = await fetch(base + '/api/v1/_test/items', { method: 'POST', headers: { Authorization: 'Bearer ' + token, 'Content-Type': 'application/json', 'Idempotency-Key': key }, body: JSON.stringify({ name }) });
      return { status: r.status, body: await r.text(), replayed: r.headers.get('idempotent-replayed'), ct: r.headers.get('content-type'), retry: r.headers.get('retry-after') };
    }, a.base, a.token, a.key, a.name);
    const arg = { base, token, key, name };
    let [ra, rb] = await Promise.all([A.run(send, { args: [arg] }), B.run(send, { args: [arg] })]);
    const first = { a: ra.status, b: rb.status };
    const sleep = (ms) => new Promise((s) => setTimeout(s, ms));
    if (ra.status === 409) { await sleep(((+ra.retry) || 1) * 1000 + 200); ra = await A.run(send, { args: [arg] }); }
    if (rb.status === 409) { await sleep(((+rb.retry) || 1) * 1000 + 200); rb = await B.run(send, { args: [arg] }); }
    const cp = await import('node:child_process');
    const psql = `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot exec -T postgres psql -U edupilot -d edupilot -At -c "select count(*) from _test_items where name='${name}'"`;
    const count = cp.execSync(psql, { cwd }).toString().trim();
    const rows = []; const push = (id, ok, detail) => rows.push({ id, ok: !!ok, detail: detail || '' });
    push('I1 đúng 1 bản ghi trong DB', count === '1', 'count=' + count);
    push('I2 hai status giống nhau và 2xx', ra.status === rb.status && /^2/.test(String(ra.status)), `lần đầu ${first.a}/${first.b} · cuối ${ra.status}/${rb.status}`);
    push('I3 hai thân giống hệt', ra.body === rb.body && ra.body.length > 0, ra.body.slice(0, 80));
    push('I4 Content-Type giống nhau', ra.ct === rb.ct, `${ra.ct} / ${rb.ct}`);
    push('I5 đúng một response có Idempotent-Replayed: true (và đọc được từ trình duyệt)', [ra.replayed, rb.replayed].filter((x) => x === 'true').length === 1, `${ra.replayed} / ${rb.replayed}`);
    return rows;
  } finally { for (const t of [A, B]) { try { await t.close(); } catch (e) {} } }
}
export const table = (rows) => rows.map((r) => `${r.ok ? 'PASS' : 'FAIL'} ${r.id} — ${r.detail}`).join('\n') + '\n=> ' + (rows.every((r) => r.ok) ? 'ok=true' : 'ok=false');
