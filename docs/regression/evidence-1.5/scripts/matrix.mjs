// Smoke 1.5 trên app thật (ep-r :3700): ma trận vai × route theo SRS PROTO §2 + redirect chưa đăng nhập + 404 + tràn ngang 375/1440.
import { chromium } from '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@playwright/test/index.mjs';
import fs from 'fs';
const F = 'https://localhost:1143', PW = 'Edupilot#Seed-2026';
const ACC = { student: 'sv.kha@edupilot.local', ta: 'ta@edupilot.local', teacher: 'teacher@edupilot.local', admin: 'admin@edupilot.local' };
const G = {
  ALL: ['/'],
  SV: ['/chat','/practice','/practice/history','/library','/me','/join'],
  SV_TA_GV: ['/threads','/calendar'],
  TA_GV: ['/inbox','/students','/attendance','/class/members','/gradebook','/gradebook/scheme','/grading','/questions','/documents','/insights','/analytics'],
  GV_AD: ['/observability','/settings/llm','/settings/integrations'],
  ADMIN: ['/admin/courses','/admin/users'],
};
const roleOf = { student: 'SV', ta: 'TA', teacher: 'GV', admin: 'AD' };
const allowed = (v, g) => ({ ALL: 1, SV: roleOf[v]==='SV', SV_TA_GV: ['SV','TA','GV'].includes(roleOf[v]), TA_GV: ['TA','GV'].includes(roleOf[v]), GV_AD: ['GV','AD'].includes(roleOf[v]), ADMIN: roleOf[v]==='AD' }[g]);
const out = { redirect: [], matrix: [], overflow: [], notfound: [] };
const b = await chromium.launch({ executablePath: '/Users/kuro/Library/Caches/ms-playwright/chromium_headless_shell-1247/chrome-headless-shell-mac-arm64/chrome-headless-shell' });
// 1) chưa đăng nhập
if (!process.env.ROLES || process.env.ROLES==='student') { const c = await b.newContext({ ignoreHTTPSErrors: true }); const p = await c.newPage();
  for (const r of ['/inbox','/chat','/gradebook','/me','/']) { await p.goto(F + r, { waitUntil: 'domcontentloaded' }).catch(()=>{}); await p.waitForURL(u => u.pathname.startsWith('/login'), { timeout: 6000 }).catch(()=>{}); out.redirect.push({ r, url: p.url().replace(F,'') }); }
  await p.goto(F + '/khong-co-trang', { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(500);
  out.notfound.push({ r: '/khong-co-trang (chưa đăng nhập)', url: p.url().replace(F,''), text: (await p.innerText('body')).slice(0,120).replace(/\s+/g,' ') });
  await c.close(); }
async function login(p, v) {
  await p.goto(F + '/login', { waitUntil: 'domcontentloaded' });
  await p.getByLabel(/Email/i).first().fill(ACC[v]); await p.getByLabel(/Mật khẩu/i).first().fill(PW);
  await p.getByRole('button', { name: /Đăng nhập/i }).first().click(); await p.waitForURL(u => !u.pathname.startsWith('/login'), { timeout: 25000 }).catch(()=>{});
  return !p.url().includes('/login'); }
async function visit(p, v, r) {
  await p.goto(F + r, { waitUntil: 'domcontentloaded' }).catch(()=>{}); await p.waitForTimeout(1200);
  if (p.url().includes('/login')) { if (await login(p, v)) { await p.goto(F + r, { waitUntil: 'domcontentloaded' }).catch(()=>{}); await p.waitForTimeout(1200); } } }
for (const v of (process.env.ROLES ? process.env.ROLES.split(',') : Object.keys(ACC))) {
  const c = await b.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } }); const p = await c.newPage();
  await p.goto(F + '/login', { waitUntil: 'domcontentloaded' });
  await p.getByLabel(/Email/i).first().fill(ACC[v]); await p.getByLabel(/Mật khẩu/i).first().fill(PW);
  await p.getByRole('button', { name: /Đăng nhập/i }).first().click(); await p.waitForURL(u => !u.pathname.startsWith('/login'), { timeout: 20000 }).catch(()=>{});
  const loginOk = !p.url().includes('/login'); out.matrix.push({ v, r: '(login)', ok: loginOk, url: p.url().replace(F,'') });
  if (!loginOk) { await c.close(); continue; }
  for (const w of [1440, 375]) {
    await p.setViewportSize({ width: w, height: 900 });
    for (const g of Object.keys(G)) for (const r of G[g]) {
      const exp = allowed(v, g) ? 'MO' : 'CHAN';
      await visit(p, v, r);
      const t = (await p.innerText('body').catch(()=>'')).replace(/\s+/g,' ');
      const blocked = /Bạn không có quyền|Trang này dành cho/.test(t), nf = /Không tìm thấy trang/.test(t);
      const got = blocked ? 'CHAN' : nf ? 'NF' : 'MO';
      const ov = await p.evaluate(() => document.documentElement.scrollWidth - window.innerWidth).catch(()=>-1);
      if (w === 1440) out.matrix.push({ v, r, exp, got, url: p.url().replace(F,''), h1: (await p.locator('h1').first().innerText().catch(()=>'')).slice(0,40) });
      out.overflow.push({ v, r, w, ov });
    }
    if (w === 1440) { await visit(p, v, '/khong-co-trang');
      out.notfound.push({ r: '/khong-co-trang ' + v, text: (await p.innerText('body')).replace(/\s+/g,' ').slice(0,100) }); }
  }
  await c.close();
}
await b.close(); fs.writeFileSync(process.argv[2] || '/tmp/qc2-A-matrix.json', JSON.stringify(out, null, 1)); console.log('done');
