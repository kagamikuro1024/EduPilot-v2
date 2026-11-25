import { chromium } from 'playwright';
import fs from 'fs';
const BASE = 'https://v2202609419695521098.happysrv.de';
const PW = fs.readFileSync('/srv/work/techlead-ui/.pw', 'utf8').trim();
const [,, who, widths, routes, out] = process.argv; // who=email
const sizes = widths.split(',').map(s => s.split('x').map(Number));
fs.mkdirSync(out, { recursive: true });
const b = await chromium.launch();
const report = [];
for (const [w, h0] of sizes) {
  const h = h0 || 812;
  const ctx = await b.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, locale: 'vi-VN', timezoneId: 'Asia/Ho_Chi_Minh' });
  const p = await ctx.newPage();
  const errs = []; p.on('pageerror', e => errs.push(e.message));
  if (who !== 'anon') {
    await p.goto(BASE + '/login'); await p.getByLabel('Email').fill(who); await p.getByLabel('Mật khẩu', { exact: true }).fill(PW);
    await p.keyboard.press('Enter'); await p.waitForURL(u => !u.pathname.startsWith('/login'), { timeout: 20000 }).catch(() => {});
  }
  let first = true;
  for (let r of routes.split(',')) {
    errs.length = 0;
    if (r.endsWith('/@')) { // "/threads/@" = mục đầu tiên trong danh sách /threads
      const list = r.slice(0, -2);
      await p.goto(BASE + list); await p.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});
      const href = await p.evaluate(l => [...document.querySelectorAll('a[href]')].map(a => a.getAttribute('href')).find(h => new RegExp('^' + l + '/[^/?#]+$').test(h) && !h.endsWith('/history')), list);
      if (!href) { report.push({ route: r, w, h, missing: true }); continue; }
      r = href;
    }
    await p.goto(BASE + r, { waitUntil: 'domcontentloaded' }); await p.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {}); await p.waitForTimeout(1200);
    if (!first) await p.getByRole('button', { name: 'Đã hiểu' }).click({ timeout: 1000 }).catch(() => {});
    first = false;
    const m = await p.evaluate(() => {
      const vw = innerWidth, de = document.documentElement;
      const small = [...document.querySelectorAll('button,a[href],input,select,textarea,[role=button],[role=tab]')].filter(e => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && getComputedStyle(e).visibility !== 'hidden' && (r.height < 44 || r.width < 44) && e.type !== 'hidden'; }).map(e => { const r = e.getBoundingClientRect(); return `${e.tagName.toLowerCase()}「${(e.innerText || e.getAttribute('aria-label') || e.placeholder || '').trim().slice(0, 24)}」${Math.round(r.width)}×${Math.round(r.height)}`; });
      const over = [...document.querySelectorAll('body *')].filter(e => { const r = e.getBoundingClientRect(); return r.width > 0 && (r.right > vw + 1 || r.left < -1) && getComputedStyle(e).position !== 'fixed'; }).slice(0, 6).map(e => `${e.tagName.toLowerCase()}.${String(e.className).slice(0, 40)} ${Math.round(e.getBoundingClientRect().left)}–${Math.round(e.getBoundingClientRect().right)}`);
      const clipped = [...document.querySelectorAll('body *')].filter(e => e.children.length === 0 && e.innerText && e.scrollWidth > e.clientWidth + 1 && ['hidden', 'clip'].includes(getComputedStyle(e).overflowX)).slice(0, 8).map(e => e.innerText.trim().slice(0, 30));
      return { url: location.pathname + location.search, scrollW: de.scrollWidth, vw, docH: de.scrollHeight, small: small.slice(0, 25), smallN: small.length, over, clipped };
    });
    const name = `${r.replace(/\//g, '_').replace(/^_/, '').slice(0, 40) || 'root'}-${w}x${h}`;
    await p.screenshot({ path: `${out}/${name}.png` });
    if (m.docH > h + 4) await p.screenshot({ path: `${out}/${name}-full.png`, fullPage: true });
    report.push({ route: r, w, h, ...m, errs: [...errs] });
  }
  await ctx.close();
}
await b.close();
fs.writeFileSync(`${out}/report-${Date.now()}.json`, JSON.stringify(report, null, 1));
console.log(JSON.stringify(report.map(x => ({ r: x.route, w: x.w, url: x.url, sw: x.scrollW, h: x.docH, smallN: x.smallN, over: x.over?.length, errs: x.errs?.length, missing: x.missing }))));
