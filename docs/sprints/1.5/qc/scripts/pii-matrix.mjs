// QC — bảng kiểm bộ nhận diện thông tin cá nhân ở Threads (SRS 4.3.3, US 01-AC18). Hộp đen, theo bảng "Khớp / KHÔNG khớp" và câu P1–P4.
//   const m = await import('/abs/docs/sprints/1.5/qc/scripts/pii-matrix.mjs');
//   const rows = await m.default(browser, { base: 'http://localhost:3000', out: '/abs/…/shots/pii' });
//   console.log(m.table(rows)); console.log(m.summary(rows));
// Mỗi ca: SV B ở `/threads`, mở form, điền Tiêu đề + Nội dung (+ chủ đề "Mật mã đối xứng"), bấm `Đăng câu hỏi`.
//   want = null  → KHÔNG có dialog, thread được tạo (URL sang /threads/<id>)
//   want = {...} → có dialog "Bài này có thông tin cá nhân", câu đếm khớp `count`, đúng 2 nút hành động, không in lại giá trị; chưa có thread nào được tạo.
// Chỉ chạy khi dev báo xong 4.3.3; mỗi ca bắt đầu từ `ep_demo_state` sạch. Không dùng Playwright (không có trong frontend).
export const CASES = [
  // P1–P4 cố định của SRS
  { id: 'P1', n: 2, title: 'Hỏi về bài tập 03', body: 'Mail của em là uyen.tt229002@sv.edupilot.test, SĐT 0912345678.', count: '1 địa chỉ email, 1 số điện thoại', hide: 'Mail của em là [đã ẩn], SĐT [đã ẩn].' },
  { id: 'P2', n: 2, title: 'SĐT em', body: '0912 345 678 / +84 912 345 678', count: '2 số điện thoại' },
  { id: 'P3', title: 'Hỏi về khoá RSA', body: 'Em nghĩ 2048 bit an toàn đến 2030 theo NIST SP 800-57, phiên bản 1.3.1, cổng 8443, mỗi lần phát biểu +0,25 điểm', count: null },
  // email
  { id: 'mail-hoa', n: 1, title: 'Hỏi', body: 'Mail UYEN.TT229002@SV.EDUPILOT.TEST giúp em', count: '1 địa chỉ email' },
  { id: 'mail-cham-cuoi', n: 1, title: 'Hỏi', body: 'Thầy gửi tới uyen.tt229002@sv.edupilot.test.', count: '1 địa chỉ email' },
  { id: 'mail-@AI', title: 'Hỏi', body: '@AI giải thích giúp em về IV', count: null },
  { id: 'mail-lab@2', title: 'Hỏi', body: 'Bài lab@2 chạy chưa ổn ạ', count: null },
  // số điện thoại — khớp
  { id: 'sdt-10', n: 1, title: 'Hỏi', body: 'Gọi 0912345678 nhé', count: '1 số điện thoại' },
  { id: 'sdt-cach', n: 1, title: 'Hỏi', body: 'Gọi 0912 345 678 nhé', count: '1 số điện thoại' },
  { id: 'sdt-cham', n: 1, title: 'Hỏi', body: 'Gọi 091.234.5678 nhé', count: '1 số điện thoại' },
  { id: 'sdt-gach', n: 1, title: 'Hỏi', body: 'Gọi 091-234-5678 nhé', count: '1 số điện thoại' },
  { id: 'sdt-+84', n: 1, title: 'Hỏi', body: 'Gọi +84 912 345 678 nhé', count: '1 số điện thoại' },
  { id: 'sdt-84', n: 1, title: 'Hỏi', body: 'Gọi 84912345678 nhé', count: '1 số điện thoại' },
  // số điện thoại — KHÔNG khớp
  ...['2048', '0,25', '8443', '1.3.1', '800-57', 'QUIZ01'].map((v) => ({ id: 'khong-sdt-' + v, title: 'Hỏi về ' + v, body: `Em thấy con số ${v} trong tài liệu, nghĩa là gì ạ?`, count: null })),
  // MSSV
  { id: 'mssv', n: 1, title: 'Hỏi', body: 'MSSV của em là 20229002 ạ', count: /mã số sinh viên|MSSV/i },
  { id: 'khong-mssv', title: 'Hỏi', body: 'Khoá 2048 bit dùng đến năm 2030 ạ', count: null },
  // họ tên
  { id: 'ten-day-du', n: 1, title: 'Hỏi', body: 'Bạn Lê Quang Huy nghỉ mấy buổi ạ', count: '1 họ tên' },
  { id: 'ten-hoa', n: 1, title: 'Hỏi', body: 'Bạn LÊ QUANG HUY nghỉ mấy buổi ạ', count: '1 họ tên' },
  { id: 'ten-NFD', n: 1, title: 'Hỏi', body: 'Bạn ' + 'Lê Quang Huy'.normalize('NFD') + ' nghỉ mấy buổi ạ', count: '1 họ tên' },
  { id: 'ten-chinh-minh', n: 1, title: 'Hỏi', body: 'Em là Trần Thu Uyên, em hỏi về IV', count: '1 họ tên' },
  { id: 'khong-ten-don', title: 'Hỏi', body: 'Huy có hỏi về IV không ạ', count: null },
  { id: 'khong-ten-TA', title: 'Hỏi', body: 'Anh Phạm Quốc Bảo và cô Lê Thu Hà đã trả lời rồi ạ', count: null },
  // điểm gắn danh tính
  { id: 'diem-em', n: 1, title: 'Hỏi', body: 'em được 8,5 điểm Bài tập 03', count: /điểm gắn với một người/ },
  { id: 'diem-ten', n: 2, title: 'Hỏi', body: 'điểm của Lê Quang Huy là 4,9', count: '1 họ tên, 1 điểm gắn với một người' },
  { id: 'khong-diem-1', title: 'Hỏi', body: '+0,25 điểm mỗi lần phát biểu', count: null },
  { id: 'khong-diem-2', title: 'Hỏi', body: 'Bài tập 03 chấm 4 tiêu chí × 2,5 điểm', count: null },
];

export default async function piiMatrix(browser, { base = 'http://localhost:3000', out, only } = {}) {
  const tab = await browser.open({ name: 'qc-pii', url: base + '/login', viewport: { width: 1440, height: 900 } });
  const rows = await tab.run(async ({ page }, a) => {
    const { base, out, only, CASES } = a; const rows = [];
    const fs = await import('node:fs'); if (out) fs.mkdirSync(out, { recursive: true });
    const sleep = (ms) => new Promise((s) => setTimeout(s, ms));
    await page.setCookie(...[['ep_demo_role', 'student'], ['ep_demo_person', 'sv-2'], ['ep_demo_course', 'int1006-1']].map(([name, value]) => ({ name, value, url: base })));
    const textOf = () => page.evaluate(() => document.body.innerText);
    for (const c of CASES) {
      if (only && !only.includes(c.id)) continue;
      const count = c.count && c.count.source ? new RegExp(c.count.source, c.count.flags) : c.count;
      await page.goto(base + '/threads', { waitUntil: 'networkidle0' }); await page.evaluate(() => localStorage.removeItem('ep_demo_state')); await page.goto(base + '/threads', { waitUntil: 'networkidle0' }); await sleep(250);
      await page.evaluate(() => { const b = [...document.querySelectorAll('button')].find((x) => /^Đặt câu hỏi$/.test(x.innerText.trim())); if (b) b.click(); }); await sleep(300);
      const filled = await page.evaluate((title, body) => {
        const f = document.querySelector('[data-part=thread-form]') || document; const set = (el, v) => { const p = el.tagName === 'TEXTAREA' ? HTMLTextAreaElement.prototype : el.tagName === 'SELECT' ? HTMLSelectElement.prototype : HTMLInputElement.prototype; Object.getOwnPropertyDescriptor(p, 'value').set.call(el, v); el.dispatchEvent(new Event('input', { bubbles: true })); el.dispatchEvent(new Event('change', { bubbles: true })); };
        const ti = f.querySelector('input[type=text],input:not([type]):not([type=checkbox])'), ta = f.querySelector('textarea'), sel = f.querySelector('select'); if (!ti || !ta) return false;
        set(ti, title); set(ta, body); if (sel) { const o = [...sel.options].find((x) => x.text.trim() === 'Mật mã đối xứng'); if (o) set(sel, o.value); } return true;
      }, c.title, c.body);
      await sleep(200);
      const typedHint = (await textOf()).includes('Có vẻ bài có thông tin cá nhân. Bạn sẽ được hỏi trước khi đăng.');
      const urlBefore = page.url();
      await page.evaluate(() => { const b = [...document.querySelectorAll('button')].find((x) => /^Đăng câu hỏi$/.test(x.innerText.trim())); if (b && !b.disabled) b.click(); }); await sleep(700);
      const d = await page.evaluate(() => { const dl = document.querySelector('[role=dialog]'); if (!dl) return null; const btns = [...dl.querySelectorAll('button')].filter((b) => b.getAttribute('aria-label') !== 'Đóng' && !/^[×x]$/i.test(b.innerText.trim())).map((b) => b.innerText.trim()); return { text: dl.innerText.replace(/\s+/g, ' '), btns }; });
      const url = page.url(); const created = /\/threads\/[^/]+$/.test(url.replace(base, '')) && url !== urlBefore;
      let ok, detail;
      if (!count) { ok = !d && created && !typedHint; detail = `dialog=${!!d} created=${created} nhắc=${typedHint}`; }
      else {
        const countOk = d && (typeof count === 'string' ? d.text.includes(count) : count.test(d.text));
        const twoBtns = d && d.btns.length === 2 && d.btns.includes('Chuyển sang chat riêng') && d.btns.includes('Ẩn thông tin rồi đăng');
        const noValueEcho = d && !/uyen\.tt229002@sv\.edupilot\.test|0912345678|20229002/.test(d.text);
        ok = !!(d && /Bài này có thông tin cá nhân/.test(d.text) && countOk && twoBtns && noValueEcho && !created && typedHint); detail = d ? `count=${countOk} 2nút=${twoBtns} khôngInLại=${noValueEcho} chưaTạo=${!created} nhắc=${typedHint} [${d.text.slice(0, 90)}]` : 'không có dialog';
        // 3b: ẩn rồi đăng — mọi ca khớp: dòng "Đã ẩn n thông tin cá nhân" (PM #20c) + bản gốc không còn ở DOM / ep_demo_state
        if (d && c.n) {
          await page.evaluate(() => { const b = [...document.querySelector('[role=dialog]').querySelectorAll('button')].find((x) => /Ẩn thông tin rồi đăng/.test(x.innerText)); b && b.click(); }); await sleep(900);
          const t = await textOf(); const st = await page.evaluate(() => localStorage.getItem('ep_demo_state') || '');
          const hideOk = !c.hide || t.includes(c.hide); const nOk = t.includes(`Đã ẩn ${c.n} thông tin cá nhân`); const clean = hideOk && nOk && !/uyen\.tt229002@|0912345678|20229002/.test(t) && !/uyen\.tt229002@|0912345678|20229002/.test(st);
          rows.push({ id: c.id + ' · 3b ẩn rồi đăng', ok: clean && /\/threads\/[^/]+$/.test(page.url().replace(base, '')), detail: `"Đã ẩn ${c.n} thông tin cá nhân"=${nOk} bản ẩn đúng=${hideOk} bản gốc không còn trong DOM/state=${!/uyen\.tt229002@|0912345678|20229002/.test(t + st)}`, file: '' });
        }
      }
      let file = ''; if (out && !ok) { file = `${out}/${c.id.replace(/[^\w-]/g, '_')}.png`; await page.screenshot({ path: file }); }
      rows.push({ id: c.id, ok: !!ok, detail, file });
    }
    return rows;
  }, { args: [{ base, out, only, CASES: CASES.map((c) => ({ ...c, count: c.count instanceof RegExp ? { source: c.count.source, flags: c.count.flags } : c.count })) }] });
  await tab.close();
  return rows;
}
export function table(rows) { return ['| ca | kết quả | chi tiết | ảnh |', '| --- | --- | --- | --- |'].concat(rows.map((r) => `| ${r.id} | ${r.ok ? 'PASS' : '**FAIL**'} | ${String(r.detail).replace(/\|/g, '\\|').slice(0, 200)} | ${r.file ? r.file.split('/').pop() : ''} |`)).join('\n'); }
export function summary(rows) { const bad = rows.filter((r) => !r.ok); return `PII matrix: ${rows.length - bad.length}/${rows.length} PASS\n` + bad.map((r) => `  FAIL ${r.id} · ${r.detail}`).join('\n'); }
