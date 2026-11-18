import sys, statistics, random, string, concurrent.futures as cf
sys.path.insert(0, '.'); from q4lib import *
pool = student_pool(80); P = iter(pool)
def tk(): u = next(P); return u, L(u)
def jn(t, code, pv=False): return call('POST', '/courses/join/preview' if pv else '/courses/join', t, {'code': code})
def zs(key): return rds('zrange', key, '0', '-1')
# ---------- P209-03 / 09: trạng thái và khác biệt hợp lệ ----------
cidF, ccF, codeF = new_course('f'); settings(cidF, capacity=1)
cidA, ccA, codeA = new_course('a'); settings(cidA, require_approval=True)
u1, t1 = tk(); u2, t2 = tk(); u3, t3 = tk()
resetjoin()
r_join1 = jn(t1, codeF)            # u1 vào lớp đầy chỗ (cap 1) → ACTIVE
pF = jn(t2, codeF, True); pA = jn(t3, codeA, True)
jA = jn(t3, codeA); pA2 = jn(t3, codeA, True); pM = jn(t1, codeF, True)
st = (pF[1].get('state'), pA[1].get('state'), pM[1].get('state'), pA2[1].get('state'))
C('TC-P209-03', st == ('FULL', 'REQUIRES_APPROVAL', 'ALREADY_MEMBER', 'PENDING'), f'preview theo trạng thái: lớp cap 1 đã có 1 SV → {st[0]}; lớp bật duyệt → {st[1]}; SV đã là thành viên → {st[2]}; sau join vào lớp bật duyệt (join → {jA[0]} {jA[1].get("status")}) → {st[3]}; preview không có mã/id: khoá {sorted(pF[1])}', f'state sai: {st}')
jF = jn(t2, codeF); bad = jn(t2, 'ZZZZZZZ'); nf = bad[1]
C('TC-P209-09', jF[0] == 409 and jF[1].get('code') == 'COURSE_FULL' and jF[1].get('message', '') .startswith('Lớp đã đủ sĩ số') and jA[0] == 200 and jA[1].get('status') == 'PENDING' and bad[0] == 404 and nf.get('code') == 'JOIN_CODE_INVALID',
  f'đầy → {jF[0]} {jF[1].get("code")} «{jF[1].get("message")}»; cần duyệt → {jA[0]} status {jA[1].get("status")}; mã sai → {bad[0]} {nf.get("code")} (chỉ hai trạng thái COURSE_FULL / PENDING khác biệt)')
# ---------- P209-22: tác dụng thật của từng cài đặt ----------
cidS, ccS, codeS = new_course('s'); uS, tS = tk(); resetjoin(); eff = {}
settings(cidS, enabled=False); eff['tắt'] = jn(tS, codeS)
settings(cidS, enabled=True); resetjoin()
settings(cidS, expires_at=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(time.time() + 3))); time.sleep(5); eff['hết hạn'] = jn(tS, codeS)
settings(cidS, expires_at=None, allowed_email_domain='ptit.edu.vn'); resetjoin(); eff['miền email'] = jn(tS, codeS)
settings(cidS, allowed_email_domain=None, capacity=1)
uX, tX = tk(); jn(tX, codeS); resetjoin(); uY, tY = tk(); eff['sĩ số'] = jn(tY, codeS)
settings(cidS, capacity=None, require_approval=True); resetjoin(); uZ, tZ = tk(); eff['duyệt'] = jn(tZ, codeS)
ok22 = all(eff[k][0] == 404 and eff[k][1].get('code') == 'JOIN_CODE_INVALID' for k in ['tắt', 'hết hạn', 'miền email']) and eff['sĩ số'][1].get('code') == 'COURSE_FULL' and eff['duyệt'][1].get('status') == 'PENDING'
C('TC-P209-22', ok22, 'tắt/hết hạn/miền email → 404 JOIN_CODE_INVALID (cùng một câu); capacity=số hiện có → 409 COURSE_FULL; require_approval → 200 PENDING; ' + '; '.join(f'{k}: {v[0]} {v[1].get("code") or v[1].get("status")}' for k, v in eff.items()))
# ---------- P209-19: audit regenerate ----------
tT = L('teacher'); old = codeS; s, j, _ = call('POST', f'/courses/{cidS}/join-code/regenerate', tT); newc = (j or {}).get('join_code')
det = psql("select coalesce(before::text,'')||' → '||coalesce(after::text,'') from audit_log where action='join_code_regenerated' and course_id='%s' order by created_at desc limit 1" % cidS); who = psql("select actor_id is not null from audit_log where action='join_code_regenerated' and course_id='%s' order by created_at desc limit 1" % cidS)
C('TC-P209-19', s == 200 and newc and newc != old and old not in det and newc not in det and who == 't' and old[:2] in det and newc[:2] in det, f'regenerate {s}; audit_log (entity course, action join_code_regenerated) có actor_id (người làm)={who}; before/after «{det}» — chỉ 2 ký tự đầu, không có mã cũ ({old}) hay mới ({newc}) dạng rõ (SQL: select before, after, actor_id from audit_log where action=join_code_regenerated; TC ghi cột «details» — schema thật là before/after)')
# ---------- P209-12: cửa sổ chỉ đếm thất bại ----------
cidW, ccW, codeW = new_course('w'); uW, tW = tk(); resetjoin(); settings(cidW)
cidFull, _, codeFull = new_course('v'); settings(cidFull, capacity=1); uq, tq = tk(); jn(tq, codeFull)   # lớp đầy để có COURSE_FULL
jn(tW, codeW)   # đã là thành viên
w4 = [jn(tW, 'AAAAAAA')[0] for _ in range(4)]; ok10 = [jn(tW, codeW) for _ in range(10)]; fl = jn(tW, codeFull)
r5 = jn(tW, 'AAAAAAA'); r6 = jn(tW, 'AAAAAAA')
uid_W = psql("select id from users where email='%s@edupilot.local'" % uW); key = 'ep:join:fail:user:' + uid_W
mem = rds('zrange', key, '0', '-1').split(); old_t = int((time.time() - 11 * 60) * 1000)
for i, m in enumerate(mem): rds('zadd', key, str(old_t + i), m)
r7 = jn(tW, 'AAAAAAA')
C('TC-P209-12', w4 == [404] * 4 and all(x[1].get('status') in ('ACTIVE', 'PENDING') or x[1].get('already_member') for x in ok10) and fl[1].get('code') == 'COURSE_FULL' and r5[0] == 404 and r6[0] == 429 and r7[0] == 404,
  f'4 sai {set(w4)}; 10 join thành công (already_member) {sorted({x[0] for x in ok10})} + COURSE_FULL {fl[0]} không đếm; sai lần 5 → {r5[0]}; lần 6 → {r6[0]} (đã 5 thất bại); lùi điểm đếm 11 phút (chỉnh score ZSET, rút gọn 10 phút — ghi rõ) rồi thử lại → {r7[0]} (cửa sổ trượt 10 phút)')
# ---------- P209-15 / 16 : mô phỏng đoán, oracle thời gian ----------
alpha = '23456789ABCDEFGHJKMNPQRSTUVWXYZ'
cidG, ccG, codeG = new_course('g'); uG, tG = tk(); resetjoin(); random.seed(7)
codes = [''.join(random.choice(alpha) for _ in range(7)) for _ in range(100)]; res = [jn(tG, c)[0] for c in codes]
hit = [c for c, r in zip(codes, res) if r == 200]
# nhiều tài khoản, mỗi tài khoản 5 lần
resetjoin(); multi = []
for k in range(5):
    u, t = tk(); multi += [jn(t, ''.join(random.choice(alpha) for _ in range(7)))[0] for _ in range(5)]
    last = multi[-1]
C('TC-P209-15', not hit and res[:5] == [404] * 5 and set(res[5:]) == {429} and multi.count(404) == 20 and multi[20:].count(429) == 5, f'100 mã ngẫu nhiên (7 ký tự bảng 31) một tài khoản: trúng {len(hit)}; 5 lần đầu 404, 95 lần sau toàn 429; 5 tài khoản × 5 lần: {multi.count(404)} × 404 rồi {multi.count(429)} × 429 theo IP (20/10 phút). Xác suất trúng của kẻ có nhiều IP + tài khoản ≈ N/31^7 ≈ 2,7·10⁻¹¹ mỗi lần (≈ 1/{31**7:,})'.replace(',', '.'))
# 16 oracle: mã gần đúng vs sai hoàn toàn (30 mẫu mỗi loại), reset đếm mỗi lần để không dính 429
def timed(code, t):
    resetjoin(); a = time.perf_counter(); jn(t, code); return (time.perf_counter() - a) * 1000
near = [codeW[:6] + ('A' if codeW[6] != 'A' else 'B') for _ in range(30)]; far = [''.join(random.choice(alpha) for _ in range(7)) for _ in range(30)]
# xen kẽ để cùng điều kiện
tn, tf = [], []
for a, b in zip(near, far): tn.append(timed(a, tG)); tf.append(timed(b, tG))
mn, mf = statistics.median(tn), statistics.median(tf)
C('TC-P209-16', abs(mn - mf) / max(mn, mf) <= 0.35, f'trung vị 30 mẫu: mã đúng 6/7 ký tự {mn:.1f} ms vs sai hoàn toàn {mf:.1f} ms (chênh {abs(mn-mf)/max(mn,mf)*100:.0f} % ≤ 35 %)')
# 08 nhiều nguyên nhân: sai/tắt/hết hạn/miền email, 30 mẫu mỗi nguyên nhân
cidD, _, codeD = new_course('d'); settings(cidD, enabled=False)
cidE, _, codeE = new_course('e'); settings(cidE, expires_at=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(time.time() + 2))); time.sleep(4)
cidM, _, codeM = new_course('m'); settings(cidM, allowed_email_domain='ptit.edu.vn')
cidR, _, codeR = new_course('r'); settings(cidR); ar = call('POST', f'/admin/courses/{cidR}/archive', L('admin')); print('archive', ar[0])
causes = [('sai', 'ZZZZZZ2'), ('tắt', codeD), ('hết hạn', codeE), ('miền email', codeM), ('lớp lưu trữ', codeR), ('không tồn tại (lệch độ dài)', 'AB')]
samples = {k: [] for k, _ in causes}
for _ in range(30):            # xen kẽ vòng tròn để nhiễu hệ thống chia đều cho mọi nguyên nhân
    for k, c in causes: samples[k].append(timed(c, tG))
med = {k: statistics.median(v) for k, v in samples.items()}
mx, mi = max(med.values()), min(med.values())
C('TC-P209-08', (mx - mi) / mx <= 0.35, 'trung vị (ms) — 6 nguyên nhân: ' + ', '.join(f'{k} {v:.1f}' for k, v in med.items()) + f'; chênh {(mx-mi)/mx*100:.0f} % ≤ 35 % (không phân biệt được nguyên nhân bằng thời gian; 30 mẫu mỗi nguyên nhân, lấy xen kẽ vòng tròn)')
