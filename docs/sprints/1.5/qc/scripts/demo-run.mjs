// QC — đi trọn `docs/DEMO_SCRIPT.md` bước 1–7 trên prototype (TC `qc/tc-US-PROTO-DEMO.md`).
// Chạy trong Eval (JS) có global `browser`:
//   const run = (await import('/abs/docs/sprints/1.5/qc/scripts/demo-run.mjs')).default;
//   const rows = await run(browser, { base: 'http://localhost:3400', out: '<abs>/docs/sprints/1.5/shots/qc-v5/v5qdemo' });
//   console.log(rows.map(r => `${r.ok ? 'PASS' : 'FAIL'} ${r.id} ${r.ms}ms · ${r.detail}`).join('\n'));
// Một trình duyệt, đổi vai bằng menu hồ sơ (không đặt cookie tay). Mỗi bước chụp một ảnh.
// Không đọc mã dev: mọi bộ chọn là chữ hiển thị / `aria-label` / `data-part` ghi trong US.md.
// Chú ý: nút của prototype cần chuột thật (`page.mouse.click`) — `el.click()` trong trang không kích hoạt.

const HELPERS = `({
  sleep: (ms) => new Promise((s) => setTimeout(s, ms)),
  box: async (page, src, flags, sel) => page.evaluate((src, flags, sel) => {
    const re = new RegExp(src, flags);
    const els = [...document.querySelectorAll(sel || 'button,a,[role=button],[role=menuitem],[role=tab],[role=option],summary,label,li')];
    const el = els.find((e) => { const t = ((e.innerText || '') + ' ' + (e.getAttribute('aria-label') || '')).trim(); const b = e.getBoundingClientRect(); return re.test(t) && b.width > 0 && b.height > 0 && !!e.offsetParent; });
    if (!el) return null; el.scrollIntoView({ block: 'center' });
    const b = el.getBoundingClientRect();
    return { x: b.x + Math.min(b.width / 2, 40), y: b.y + b.height / 2, t: (el.innerText || el.getAttribute('aria-label') || '').trim().slice(0, 60) };
  }, src, flags, sel),
  click: async function (page, src, flags, sel, wait) { const b = await this.box(page, src, flags || '', sel); if (!b) return null; await page.mouse.click(b.x, b.y); await this.sleep(wait || 700); return b.t; },
  // ô trong thẻ sinh viên (điểm danh): tìm thẻ nhỏ nhất chứa tên rồi bấm nhãn trong thẻ đó
  mark: async function (page, name, label) {
    const b = await page.evaluate((n, l) => {
      const vis = (e) => e.offsetParent && e.getBoundingClientRect().width > 0;
      const cards = [...document.querySelectorAll('li,div')].filter((e) => vis(e) && (e.innerText || '').includes(n) && (e.innerText || '').includes('Có mặt') && (e.innerText || '').length < 220).sort((x, y) => x.innerText.length - y.innerText.length);
      const el = cards[0] && [...cards[0].querySelectorAll('label,button')].find((e) => (e.innerText || '').trim() === l);
      if (!el) return null; el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 };
    }, name, label);
    if (!b) return null; await page.mouse.click(b.x, b.y); await this.sleep(350); return name + '/' + label;
  },
  type: async function (page, sel, text) { const el = await page.$(sel); if (!el) return false; const b = await page.evaluate((x) => { x.scrollIntoView({ block: 'center' }); const r = x.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; }, el); await page.mouse.click(b.x, b.y); await el.type(text, { delay: 8 }); return true; },
  role: async function (page, name) { await this.click(page, 'Tài khoản', 'i', 'header button', 600); const t = await this.click(page, name, '', 'button,a', 1800); await this.sleep(400); return t; },
  course: async function (page, code) { await this.click(page, '7619', '', 'header button', 700); const t = await this.click(page, code, '', 'button,a,[role=option],li', 1800); await this.sleep(400); return t; },
  vp: async function (page, w) { await page.setViewport({ width: w, height: w < 500 ? 844 : 900, deviceScaleFactor: 1, isMobile: w < 500, hasTouch: w < 500 }); await this.sleep(500); },
  // chờ một chuỗi hiện trong trang (mặc định 8 s) — tránh bấm khi khung chưa dựng xong
  waitFor: async function (page, src, ms) { const end = Date.now() + (ms || 8000); const re = new RegExp(src); while (Date.now() < end) { if (re.test(await this.txt(page))) return true; await this.sleep(250); } return false; },
  txt: (page) => page.evaluate(() => document.body.innerText),
  shot: async (page, out, name) => { await page.screenshot({ path: out + '/' + name + '.webp', type: 'webp', quality: 70 }); },
})`;

// Mỗi bước: id (TC), nhãn, hàm chạy trong trang trả về { ok, detail }.
const STEPS = ({ h, base }) => ({
  'TC-DEMO-01 admin /admin/courses': async (page, out) => {
    await h.role(page, 'Admin · Đỗ Hoàng Nam');
    await page.goto(base + '/admin/courses', { waitUntil: 'networkidle0' }); await h.sleep(400);
    const t = await h.txt(page); await h.shot(page, out, 'd01-admin-courses');
    return { ok: /761987/.test(t) && /761988/.test(t) && (t.match(/Lê Thu Hà/g) || []).length >= 2, detail: 'hai lớp + giảng viên' };
  },
  'TC-DEMO-02 GV chuông': async (page, out) => {
    await h.role(page, 'Giảng viên · Lê Thu Hà');
    await h.click(page, 'Thông báo', 'i', 'header button', 800);
    const t = await h.txt(page); await h.shot(page, out, 'd02-gv-chuong');
    await page.keyboard.press('Escape');
    return { ok: /BX4P9TW/.test(t) && /Thiết lập lớp mới/.test(t), detail: 'mã tham gia + việc thiết lập lớp 2' };
  },
  'TC-DEMO-03 SV D /join': async (page, out) => {
    await h.vp(page, 390); await h.role(page, 'Sinh viên D · Phạm Ngọc Linh');
    await page.goto(base + '/join/BX4P9TW', { waitUntil: 'networkidle0' }); await h.waitFor(page, 'Mã lớp');
    const prev = await h.txt(page); await h.shot(page, out, 'd03-join-preview');
    await h.click(page, '^Tham gia lớp$', '', 'button', 1500);
    const t = await h.txt(page); await h.shot(page, out, 'd03-join-cho-duyet');
    return { ok: /761988/.test(prev) && /chờ giảng viên duyệt/.test(t), detail: 'xem trước + chờ duyệt' };
  },
  'TC-DEMO-04 GV duyệt D': async (page, out) => {
    await h.vp(page, 1440); await h.role(page, 'Giảng viên · Lê Thu Hà');
    await h.course(page, '761988'); await h.click(page, '7619', '', 'header button', 700);
    await h.click(page, 'Quản lý lớp này', '', 'button,a,[role=menuitem],li', 1500); await h.waitFor(page, 'Yêu cầu chờ duyệt');
    const b = await page.evaluate(() => { const li = [...document.querySelectorAll('li,tr,div')].filter((e) => /Phạm Ngọc Linh/.test(e.innerText || '') && /Duyệt/.test(e.innerText || '')).pop(); const btn = li && [...li.querySelectorAll('button')].find((x) => (x.innerText || '').trim() === 'Duyệt'); if (!btn) return null; btn.scrollIntoView({ block: 'center' }); const r = btn.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; });
    if (b) { await page.mouse.click(b.x, b.y); await h.sleep(1200); }
    const t = await h.txt(page); await h.shot(page, out, 'd04-duyet-D');
    return { ok: /Yêu cầu chờ duyệt \(3\)/.test(t) && /Thành viên \(25\)/.test(t), detail: '3 yêu cầu seed còn chờ, D vào lớp' };
  },
  'TC-DEMO-05 SV D tải lại': async (page, out) => {
    await h.vp(page, 390); await h.role(page, 'Sinh viên D · Phạm Ngọc Linh');
    await page.goto(base + '/', { waitUntil: 'networkidle0' }); await page.reload({ waitUntil: 'networkidle0' }); await h.sleep(600);
    const t = await h.txt(page); await h.shot(page, out, 'd05-svD-home');
    return { ok: /761988/.test(t) && !/Nhập mã tham gia/.test(t), detail: 'bộ chọn lớp có 761988' };
  },
  'TC-DEMO-06..09 SV B chat D1/D2': async (page, out) => {
    await h.role(page, 'Sinh viên B · Trần Thu Uyên');
    await h.click(page, '^Chat riêng$', '', 'a,button', 1200); await h.waitFor(page, 'Phiên mới');
    await h.type(page, 'textarea', 'Em là Trần Thu Uyên, MSSV 20229002. Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?');
    await page.keyboard.press('Enter'); await h.sleep(6000);
    const d1 = await h.txt(page); await h.shot(page, out, 'd06-D1');
    await h.click(page, '^Hữu ích$', '', 'button', 800);
    const fb = await h.txt(page);
    await h.type(page, 'textarea', 'Bạn Lê Quang Huy nghỉ mấy buổi rồi ạ?');
    await page.keyboard.press('Enter'); await h.sleep(5000);
    const d2 = await h.txt(page); await h.shot(page, out, 'd08-D2');
    await h.click(page, 'Nguồn tham khảo', '', 'button,summary,a', 800); await h.shot(page, out, 'd09-nguon');
    return { ok: /Đã ẩn 2 thông tin cá nhân/.test(d1) && /vắng 2 buổi/i.test(d1) && /\+0,75|0,75 điểm/.test(d1) && !/\[\[SV_/.test(d1) && /Đã ghi nhận/.test(fb) && /chỉ trả lời được thông tin của chính bạn/.test(d2), detail: 'ẩn 2 PII, vắng 2 / +0,75, Hữu ích tại chỗ, D2 từ chối' };
  },
  'TC-DEMO-10 D3 chuyển GV': async (page, out) => {
    await h.type(page, 'textarea', 'Thi cuối kỳ có được mang một tờ A4 ghi chú viết tay vào phòng thi không ạ?');
    await page.keyboard.press('Enter'); await h.sleep(6500);
    const t = await h.txt(page); await h.shot(page, out, 'd10-D3');
    return { ok: /AI chưa đủ chắc chắn về câu này/.test(t) && /Đang chờ giảng viên · vừa gửi/.test(t), detail: 'tự chuyển giảng viên' };
  },
  'TC-DEMO-11..13 GV inbox D3 + D4': async (page, out) => {
    await h.vp(page, 1440); await h.role(page, 'Giảng viên · Lê Thu Hà'); await h.waitFor(page, 'việc cần xử lý hôm nay');
    await h.click(page, '^Trả lời$', '', 'button,a', 1500);
    await h.course(page, '761987'); // BƯỚC NGOÀI KỊCH BẢN: Hôm nay mở inbox theo lớp đang chọn (lớp 2) → rỗng
    await h.waitFor(page, 'Thi cuối kỳ có được mang một tờ A4');
    await h.click(page, 'Thi cuối kỳ có được mang một tờ A4', '', 'li,button,a,[role=button],[role=option]', 1200);
    const open = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd11-ticket');
    await h.click(page, '^Nhận$', '', 'button', 1200);
    const claimed = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd12-claimed');
    await h.type(page, 'main textarea', 'Được mang một tờ A4 viết tay, hai mặt, không dùng bản photo. Thầy sẽ nhắc lại trên lớp.');
    await h.click(page, '^Gửi trả lời$', '', 'button', 1600);
    const ans = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd13-answered');
    return { ok: /Đã chờ/.test(open) && /Độ tin cậy/.test(open) && /Đã nhận/.test(claimed) && /Đã trả lời/.test(ans) && /Đã gửi thư thông báo/.test(ans), detail: 'Open → Claimed → Answered + dòng thư (mô phỏng)' };
  },
  'TC-DEMO-14..15 SV B nhận trả lời': async (page, out) => {
    await h.vp(page, 390); await h.role(page, 'Sinh viên B · Trần Thu Uyên');
    await h.click(page, 'Thông báo', 'i', 'header button', 800);
    await h.click(page, 'Giảng viên đã trả lời', '', '[role=menu] a,[role=menu] button,a,button', 1500); await h.waitFor(page, 'Giảng viên Lê Thu Hà trả lời');
    const t = await h.txt(page); await h.shot(page, out, 'd14-tra-loi-GV');
    await h.click(page, '^Đã rõ$', '', 'button', 1200);
    const t2 = await h.txt(page); await h.shot(page, out, 'd15-da-dong');
    return { ok: /Giảng viên Lê Thu Hà trả lời/.test(t) && /Câu hỏi đã đóng/.test(t2), detail: 'nhãn giảng viên + đóng câu hỏi' };
  },
  'TC-DEMO-16..18 điểm danh buổi 10': async (page, out) => {
    await h.role(page, 'Giảng viên · Lê Thu Hà');
    await h.click(page, '^Điểm danh$', '', 'button,a', 1500); await h.waitFor(page, 'Mặc định mọi sinh viên có mặt');
    const url = page.url();
    const t0 = Date.now();
    await h.mark(page, 'Lê Quang Huy', 'Vắng'); await h.mark(page, 'Vũ Khánh Huy', 'Vắng');
    await h.mark(page, 'Lý Gia Thảo', 'Muộn'); await h.mark(page, 'Trần Thu Uyên', 'Phát biểu');
    const marks = Date.now() - t0; await h.shot(page, out, 'd17-danh-dau');
    await h.click(page, '^Lưu điểm danh$', '', 'button', 1500);
    const t = await h.txt(page); await h.shot(page, out, 'd18-da-luu');
    return { ok: /session=10/.test(url) && marks < 60000 && /Đã hoàn tất buổi 10 · 27 có mặt, 1 muộn, 2 vắng/.test(t), detail: `đánh dấu ${marks}ms; chốt buổi 10` };
  },
  'TC-DEMO-19 SV B /me 8,5': async (page, out) => {
    await h.vp(page, 390); await h.role(page, 'Sinh viên B · Trần Thu Uyên');
    await page.goto(base + '/me', { waitUntil: 'networkidle0' }); await h.sleep(600);
    const t = await h.txt(page); await h.shot(page, out, 'd19-me-85');
    return { ok: /Điểm quá trình hiện tại 8,5/.test(t) && /4 lần × 0,25/.test(t), detail: 'QT 8,5 · phát biểu +1,00' };
  },
  'TC-DEMO-21 SV B BT03 đang chấm': async (page, out) => {
    await page.goto(base + '/assignments/bt03', { waitUntil: 'networkidle0' }); await h.sleep(600);
    const t = await h.txt(page); await h.shot(page, out, 'd21-bt03-dang-cham');
    return { ok: /Đang chấm/.test(t) && /Nộp muộn 1 ngày/.test(t) && !/nháp/i.test(t), detail: 'không lộ điểm nháp' };
  },
  'TC-DEMO-22..25 chấm + công bố': async (page, out) => {
    await h.vp(page, 1440); await h.role(page, 'Giảng viên · Lê Thu Hà');
    await h.click(page, 'Chấm bài', '', 'nav a,a', 1800); await h.waitFor(page, 'bài khớp bộ lọc');
    const queue = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd22-hang-cho');
    const b = await page.evaluate(() => { const el = [...document.querySelectorAll('td span')].find((e) => (e.innerText || '').trim() === 'Trần Thu Uyên'); el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; });
    await page.mouse.click(b.x, b.y); await h.sleep(1600);
    for (let i = 0; i < 6; i++) { const t = await page.evaluate(() => { const el = document.querySelector('[aria-label="Tăng điểm Phân tích tấn công"]'); el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; }); await page.mouse.click(t.x, t.y); await h.sleep(220); }
    const detail = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd23-sua-diem');
    await h.click(page, '^Duyệt bài$', '', 'button', 1600);
    await h.click(page, '^Chưa duyệt', '', 'main button,main [role=tab],main label', 1200);
    const cb = await page.evaluate(() => { const tr = [...document.querySelectorAll('tr')].find((e) => /Trần Thu Uyên/.test(e.innerText || '')); const i = tr.querySelector('input[type=checkbox]'); i.scrollIntoView({ block: 'center' }); const r = i.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; });
    await page.mouse.click(cb.x, cb.y); await h.sleep(600);
    await h.click(page, '^Công bố$', '', 'main button', 1200);
    await h.click(page, '^Công bố điểm$', '', '[role=dialog] button,button', 1800);
    const after = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd25-da-cong-bo');
    return { ok: /Hai lượt chấm lệch 1,5 điểm/.test(queue) && /8,50 điểm rubric − 0,5 nộp muộn/.test(detail) && /Đã công bố/.test(after), detail: 'cờ lệch → sửa 2,5 → 8,0 → công bố' };
  },
  'TC-DEMO-26 SV B thấy 8,0 + QT 8,7': async (page, out) => {
    await h.vp(page, 390); await h.role(page, 'Sinh viên B · Trần Thu Uyên');
    await page.goto(base + '/assignments/bt03', { waitUntil: 'networkidle0' }); await page.reload({ waitUntil: 'networkidle0' }); await h.sleep(700);
    const a = await h.txt(page); await h.shot(page, out, 'd26-bt03-diem');
    await page.goto(base + '/me', { waitUntil: 'networkidle0' }); await h.sleep(600);
    const me = await h.txt(page); await h.shot(page, out, 'd26-me-87');
    return { ok: /Điểm bài này:\n8,0|8,0\n8,5 theo rubric/.test(a) && /Điểm quá trình hiện tại 8,7/.test(me), detail: 'BT03 8,0 · QT 8,7 (xem BUG-v5-DEMO-2 nếu ra 8,4)' };
  },
  'TC-DEMO-27..30 công thức điểm lớp 2': async (page, out) => {
    await h.vp(page, 1440); await h.role(page, 'Giảng viên · Lê Thu Hà');
    await h.click(page, '^Sổ điểm$', '', 'nav a,a', 1500); await h.course(page, '761988'); await h.waitFor(page, 'Công thức');
    const locked = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd27-gradebook-lop2');
    await h.click(page, 'Mở công thức điểm|Xem công thức', '', 'button,a', 1800);
    await h.click(page, '^Tải quy chế$', '', 'button,label', 3500);
    const draft = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd28-ban-nhap');
    await h.type(page, 'main input[type=text],main input:not([type])', 'Làm tròn đến 0,1'); await h.sleep(500);
    await h.click(page, '^Xác nhận công thức$', '', 'button', 1200);
    await h.click(page, '^Xác nhận công thức$', '', '[role=dialog] button', 1800);
    await h.shot(page, out, 'd29-da-xac-nhan');
    await h.click(page, '^Sổ điểm$', '', 'main a,nav a,a', 1600);
    const open = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd30-mo-khoa');
    return { ok: /Công thức chưa xác nhận/.test(locked) && /Chưa rõ: quy chế không nêu quy tắc làm tròn/.test(draft) && /Trang 2/.test(draft) && /Công thức đã xác nhận/.test(open), detail: 'banner → nháp có số trang → xác nhận → mở khoá' };
  },
  'TC-DEMO-31..32 sổ điểm lớp 1': async (page, out) => {
    await h.course(page, '761987'); await h.waitFor(page, 'Trần Thu Uyên');
    const b = await page.evaluate(() => { const el = document.querySelector('[aria-label="Hành động với Trần Thu Uyên"]'); el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; });
    await page.mouse.click(b.x, b.y); await h.sleep(700);
    await h.click(page, 'Xem giải trình điểm', '', '[role=menu] button,[role=menu] a', 1400);
    const gt = await page.evaluate(() => [...document.querySelectorAll('[role=dialog]')].map((e) => e.innerText).join('\n')); await h.shot(page, out, 'd31-giai-trinh');
    await page.keyboard.press('Escape'); await h.sleep(500);
    const m = await page.evaluate(() => { const el = document.querySelector('[aria-label="Thêm hành động với sổ điểm"]'); el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; });
    await page.mouse.click(m.x, m.y); await h.sleep(700);
    await h.click(page, 'XLSX', '', '[role=menu] button,[role=menu] a', 1600);
    const t = await h.txt(page); await h.shot(page, out, 'd32-xuat-xlsx');
    return { ok: /4 lần × 0,25/.test(gt) && /\(7,0 \+ 8,0 \+ 8,0\) ÷ 3/.test(gt) && /Điểm chính thức nằm ở hệ thống quản lý đào tạo/.test(gt) && /Đã tạo file sổ điểm \(mô phỏng\)/.test(t), detail: 'giải trình đủ nguồn + xuất XLSX' };
  },
  'TC-DEMO-33 What-if CK 8,0': async (page, out) => {
    await h.vp(page, 390); await h.role(page, 'Sinh viên B · Trần Thu Uyên');
    await page.goto(base + '/me', { waitUntil: 'networkidle0' }); await h.sleep(700);
    const bx = await page.evaluate(() => { const el = [...document.querySelectorAll('input')].find((e) => e.offsetParent && e.type === 'text'); el.scrollIntoView({ block: 'center' }); const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; });
    await page.mouse.click(bx.x, bx.y); await page.keyboard.press('End');
    for (let i = 0; i < 14; i++) await page.keyboard.press('Backspace');
    await page.keyboard.type('8,0', { delay: 60 }); await h.sleep(900);
    const t = await h.txt(page); await h.shot(page, out, 'd33-whatif');
    return { ok: /điểm học phần sẽ là\n8,3/.test(t), detail: '0,4×8,7 + 0,6×8,0 = 8,3' };
  },
  'TC-DEMO-34..36 insights lớp 1': async (page, out) => {
    await h.vp(page, 1440); await h.role(page, 'Giảng viên · Lê Thu Hà');
    await h.click(page, '^Insights$', '', 'nav a,a', 1600); await h.waitFor(page, 'Chủ đề nên dạy lại');
    await h.click(page, 'Tạo báo cáo mới', '', 'button', 5000);
    const rep = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd34-bao-cao');
    await h.click(page, 'Xem 5 câu hỏi đã ẩn danh', '', 'button,summary,a', 1000);
    const q = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd35-cau-mau');
    await h.click(page, '^Tạo thread ghim$', '', 'button', 1400);
    const pin = await page.evaluate(() => [...document.querySelectorAll('[role=status]')].map((e) => e.innerText).join(' ')); await h.shot(page, out, 'd36-ghim');
    return { ok: /Mật mã đối xứng \(AES, CBC\)/.test(rep) && /Hàm băm và chữ ký số/.test(rep) && (q.match(/“/g) || []).length >= 5 && /Dưới 3 sinh viên hỏi/.test(q) && /Đã ghim thread/.test(pin), detail: 'A,B đứng đầu · 5 câu ẩn danh · ghim thread' };
  },
  'TC-DEMO-37 insights lớp 2': async (page, out) => {
    await h.course(page, '761988'); await h.sleep(800);
    const t = await page.evaluate(() => document.querySelector('main').innerText); await h.shot(page, out, 'd37-insights-lop2');
    return { ok: /Tường lửa và phân đoạn mạng/.test(t) && !/Mật mã đối xứng \(AES, CBC\)/.test(t), detail: 'chủ đề C đứng đầu, không lẫn lớp 1' };
  },
});

export default async function run(browser, { base = 'http://localhost:3400', out, app } = {}) {
  const tab = await browser.open({ name: 'qc-demo-run', url: base + '/login', app, viewport: { width: 1440, height: 900 } });
  // chia hai lượt `tab.run` (mỗi lượt tối đa ~300 s); cùng một tab nên trạng thái demo đi tiếp
  const pass = (from, to, reset) => tab.run(async ({ page }, a) => {
    const { base, out, helpersSrc, stepsSrc, from, to, reset } = a;
    const fs = await import('node:fs'); if (out) fs.mkdirSync(out, { recursive: true });
    const h = new Function('return ' + helpersSrc)();
    const errs = [];
    page.on('console', (m) => { if (m.type() === 'error') errs.push(m.text().slice(0, 120)); });
    page.on('pageerror', (e) => errs.push('pageerror: ' + e.message.slice(0, 120)));
    if (reset) {
      // vào vai Admin từ /login (thẻ vai cần chuột thật) rồi đặt lại dữ liệu demo
      for (const e of await page.$$('button')) { const t = await page.evaluate((x) => (x.innerText || '').trim(), e); if (/^Admin/.test(t)) { await e.click(); break; } }
      await h.sleep(1500);
      await h.click(page, 'Tài khoản', 'i', 'header button', 600);
      await h.click(page, 'Đặt lại dữ liệu demo', '', 'button,a', 1500);
    }
    const steps = Object.entries(new Function('a', 'return (' + stepsSrc + ')(a)')({ h, base })).slice(from, to);
    const rows = [];
    for (const [id, fn] of steps) {
      const s = Date.now(); errs.length = 0;
      let r; try { r = await fn(page, out); } catch (e) { r = { ok: false, detail: 'lỗi: ' + e.message.slice(0, 120) }; }
      rows.push({ id, ok: r.ok, ms: Date.now() - s, detail: r.detail, console: errs.slice(0, 2) });
    }
    return rows;
  }, { timeout: 290000, args: [{ base, out, helpersSrc: HELPERS, stepsSrc: STEPS.toString(), from, to, reset }] });
  const t0 = Date.now();
  const rows = [...await pass(0, 8, true), ...await pass(8, 99, false)];
  rows.push({ id: 'TC-DEMO-38 tổng thời gian', ok: Date.now() - t0 < 13 * 60 * 1000 && rows.every((r) => r.ok), ms: Date.now() - t0, detail: 'ngưỡng kịch bản 13:45 · mọi bước đi được', console: [] });
  await tab.close();
  return rows;
}
