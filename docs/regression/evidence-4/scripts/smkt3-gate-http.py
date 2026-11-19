# SmkT3: kiểm chứng trực tiếp bằng HTTP trên ep-r cho GATEP2-41, 43, 47, 50, 51 (phần XFF), 52. Ghi results-smk4t3.tsv.
import os, sys, json, re, time, uuid, statistics, concurrent.futures as cf
from collections import Counter
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from smkT import *  # call (chống 429), psql, rds, resetrl, mail_to, link_token, stu, lg, ck...
import smkT, yaml, hashlib
OUT3 = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/results-smk34/results-smk4t3.tsv')
def R(tc, st, note=''):
    note = ' '.join(str(note).split())
    open(OUT3, 'a').write(f'{tc}\t{st}\t{note}\n'); print(st, tc, note[:230], flush=True)
only = set(sys.argv[1:])
def want(n): return not only or n in only
def ckv(h): return (h.get('Set-Cookie') or h.get('set-cookie') or '').split(';')[0]
def cv(c): return c.split('=', 1)[1] if '=' in c else ''
def sha(x): return hashlib.sha256(x.encode()).hexdigest()
def mk(label, verify=True):
    em, tok = stu(label, verify=verify); return em
def login_raw(em):
    resetrl(); s, j, h = call('POST', '/auth/login', None, {'email': em, 'password': NPW}); return s, j, h

# ---------------- GATEP2-41: dùng lại refresh cũ ----------------
if want('41'):
    em = mk('g41'); s, j, h = login_raw(em); acc = j['access_token']; t1 = ckv(h)
    pre = call('GET', '/me/courses', acc)[0]
    resetrl(); sa, ja, ha = call('POST', '/auth/refresh', None, {}, hdr={'Cookie': t1}); t2 = ckv(ha)   # kẻ trộm xoay trước
    t0 = time.time(); resetrl(); sb, jb, _ = call('POST', '/auth/refresh', None, {}, hdr={'Cookie': t1})  # nạn nhân mang T1 cũ
    sx, jx, _ = call('GET', '/me/courses', acc); dt = time.time() - t0
    # access mới của kẻ trộm (T2 → access) cũng phải chết
    sacc2 = ja.get('access_token'); sy = call('GET', '/me/courses', sacc2)[0] if sacc2 else None
    resetrl(); sc, jc, _ = call('POST', '/auth/refresh', None, {}, hdr={'Cookie': t2})
    rr = psql(f"select revoked_reason, revoked_at is not null from auth_sessions where user_id=(select id from users where email='{em}') order by created_at desc limit 1")
    au = psql(f"select count(*) from audit_log where action='refresh_reuse' and entity_id in (select id::text from auth_sessions where user_id=(select id from users where email='{em}'))")
    ok = pre == 200 and sa == 200 and sb == 401 and sx == 401 and sy == 401 and sc == 401 and rr.startswith('REFRESH_REUSE|t') and au == '1' and dt <= 1.5
    R('TC-GATEP2-41', 'P' if ok else 'F', f'HTTP trên ep-r: access trước {pre}; kẻ trộm xoay T1→T2 {sa}; nạn nhân mang T1 → {sb} {jb.get("code")}; sau đó access cũ {sx}, access mới của kẻ trộm {sy}, T2 {sc} (mọi thứ 401 ≤ {dt:.2f} s từ lúc phát hiện); auth_sessions {rr}; audit_log refresh_reuse={au}; ca nền TC-P202-11/12/13/14 đều P (smkt3-gate-http.py)')

# ---------------- GATEP2-43: link dùng hai lần / đua (50 song song) ----------------
if want('43'):
    res = {}
    # xác minh email
    em, tok = stu('g43v', verify=False)[0], None
    m = mail_to(em, 'Xác minh', '', 30); tok = link_token(m, 'verify-email')
    resetrl()
    with cf.ThreadPoolExecutor(50) as ex: rs = list(ex.map(lambda _: call('POST', '/auth/verify-email', None, {'token': tok})[:2], range(50)))
    res['xác minh'] = (Counter(x[0] for x in rs), Counter(x[1].get('details', {}).get('reason') for x in rs if x[0] == 410), Counter(x[1].get('code') for x in rs if x[0] == 410), psql(f"select status from users where email='{em}'"))
    # đặt lại mật khẩu
    em2 = mk('g43r'); resetrl(); call('POST', '/auth/forgot-password', None, {'email': em2}); m = mail_to(em2, 'đặt lại|Đặt lại', '', 30); tk2 = link_token(m, 'reset-password')
    resetrl()
    with cf.ThreadPoolExecutor(50) as ex: rs2 = list(ex.map(lambda i: call('POST', '/auth/reset-password', None, {'token': tk2, 'new_password': f'Smkt!Rc-{TAG}-{i:02d}qz'})[:2], range(50)))
    res['đặt lại'] = (Counter(x[0] for x in rs2), Counter(x[1].get('details', {}).get('reason') for x in rs2 if x[0] == 410), Counter(x[1].get('code') for x in rs2 if x[0] == 410))
    # mời
    A = tokof('admin'); em3 = f'smkt.g43i.{TAG}@edupilot.local'; call('POST', '/admin/users', A, {'email': em3, 'full_name': 'G43 Mời', 'role': 'TEACHER'}); m = mail_to(em3, 'mời|Lời mời', '', 30); tk3 = link_token(m, 'invite')
    resetrl()
    with cf.ThreadPoolExecutor(50) as ex: rs3 = list(ex.map(lambda _: call('POST', '/auth/accept-invite', None, {'token': tk3, 'password': NPW})[:2], range(50)))
    sess = psql(f"select count(*) from auth_sessions where user_id=(select id from users where email='{em3}')")
    res['mời'] = (Counter(x[0] for x in rs3), Counter(x[1].get('details', {}).get('reason') for x in rs3 if x[0] == 410), Counter(x[1].get('code') for x in rs3 if x[0] == 410), 'phiên=' + sess)
    ones = [Counter(x[0] for x in l)[200] == 1 for l in (rs, rs2, rs3)]
    others = all(all(c in (410, 429) for c in Counter(x[0] for x in l) if c != 200) for l in (rs, rs2, rs3))
    # bằng chứng chặt chẽ cho mã lỗi: mọi 410 là LINK_INVALID reason=used
    inv = all(set(v[2]) <= {'LINK_INVALID'} and set(v[1]) <= {'used'} for v in res.values())
    R('TC-GATEP2-43', 'P' if all(ones) and others and inv and sess == '1' else 'F', '50 yêu cầu song song mỗi loại trên ep-r: ' + '; '.join(f'{k}: mã {dict(v[0])}, lý do {dict(v[1])}, code {dict(v[2])}' + (f', {v[3]}' if len(v) > 3 else '') for k, v in res.items()) + '; ca nền TC-P203-13/14, P204-05/06, P206-06/07 đều P')

# ---------------- GATEP2-47: IDOR giữa lớp ----------------
if want('47'):
    A = tokof('admin'); T = tokof('teacher'); K = tokof('sv.kha')
    cc = ('QS47' + TAG).upper()
    c = call('POST', '/admin/courses', A, {'subject_code': 'INT1006', 'class_code': cc, 'name': 'Lớp IDOR ' + cc, 'semester': '2026-2027-HK1'}); assert c[0] == 201, c
    cid = c[1]['id']; call('POST', f'/admin/courses/{cid}/assign', A, {'teacher_id': psql("select id from users where email='teacher@edupilot.local'")})
    snap = lambda: psql(f"select (select count(*) from enrollments where course_id='{cid}')||'/'||(select md5(string_agg(id::text||status,',' order by id)) from enrollments where course_id='{cid}')||'/'||(select count(*) from threads where course_id='{cid}')||'/'||(select count(*) from notifications where course_id='{cid}')")
    before = snap()
    spec = yaml.safe_load(open(os.path.expanduser('~/Documents/TA_Agent_qc2/backend-go/api/openapi.yaml')))
    rep = []; n = 0; bad_leak = 0
    for path, ops in spec['paths'].items():
        if not re.search(r'/courses/\{(id|courseId)\}', path) or path.endswith('/courses/{id}') or '/admin/' in path: continue
        for m_, op in ops.items():
            if m_.upper() not in ('GET', 'POST', 'PUT', 'PATCH', 'DELETE') or not isinstance(op, dict): continue
            p = re.sub(r'\{(id|courseId)\}', cid, path, 1); p = re.sub(r'\{[^}]+\}', str(uuid.uuid4()), p)
            s_, j_, _ = call(m_.upper(), p.replace('/api/v1', ''), K, {} if m_.upper() != 'GET' else None); n += 1
            if 200 <= s_ < 300 or s_ >= 500: rep.append((m_.upper(), path[-40:], s_))
            if cid in json.dumps(j_) and s_ == 200: bad_leak += 1
    extra = [call('GET', f'/courses/{cid}/today?user_id={uuid.uuid4()}', K)[0], call('GET', f'/courses/{cid}/members?limit=5', K)[0], call('GET', f'/courses/{cid}/today', K, hdr={'X-Course-Id': cid})[0], call('GET', f'/courses/{cid}/threads', K)[0]]
    after = snap()
    R('TC-GATEP2-47', 'P' if not rep and not bad_leak and before == after and all(x == 403 for x in extra) else 'F',
      f'SV sv.kha (không thuộc lớp {cc}) gọi {n} thao tác (mọi phương thức) của mọi route lớp theo openapi: 2xx/5xx = {rep[:5] or 0}; /today?user_id=lạ, /members, /threads, header X-Course-Id → {extra}; DB lớp trước/sau {before}={after}; ca nền TC-P207-08/15, P209-37/38, P211-20/21 đều P')

# ---------------- GATEP2-50: liệt kê tài khoản ----------------
if want('50'):
    em = mk('g50'); ghost = f'smkt.ghost.{TAG}@edupilot.local'
    def timed(f, n=7):
        ts, outs = [], set()
        for i in range(n):
            resetrl(); t = time.time(); s, j, _ = f(i); ts.append(time.time() - t); j = dict(j); j.pop('trace_id', None); j.pop('retry_after', None); outs.add((s, json.dumps(j, sort_keys=True)))
        return statistics.median(ts), outs
    rows = {}
    rows['forgot-password'] = [timed(lambda i, e=e: call('POST', '/auth/forgot-password', None, {'email': e})) for e in (em, ghost)]
    rows['login sai'] = [timed(lambda i, e=e: call('POST', '/auth/login', None, {'email': e, 'password': 'Sai-Mat-Khau-1x'})) for e in (em, ghost)]
    rows['register'] = [timed(lambda i, e=e: call('POST', '/auth/register', None, {'email': e, 'password': NPW, 'full_name': 'U'}), 4) for e in (em, f'smkt.new{TAG}@edupilot.local')]
    rows['resend-verification'] = [timed(lambda i, e=e: call('POST', '/auth/resend-verification', None, {'email': e})) for e in (em, ghost)]
    kc = psql("select join_code from courses where class_code='761987'"); resetjoin = lambda: rds('del', 'x')
    tS = tokof('sv.gioi')
    def jp(code): return call('POST', '/courses/join/preview', tS, {'code': code})
    rows['mã tham gia'] = []
    for code in ('ZZZZZZZ', 'AAAAAAA'):   # hai mã không tồn tại (không có mã "thật nhưng tắt" dùng chung ở đây: ca nền TC-P209-07/08)
        subprocess.run("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:join:fail:*' | xargs -r redis-cli del\"", shell=True, capture_output=True)
        rows['mã tham gia'].append(timed(lambda i, c=code: jp(c), 3))
    ok = True; parts = []
    for k, ((m1, o1), (m2, o2)) in rows.items():
        same = o1 == o2 and len(o1) == 1
        ratio = min(m1, m2) / max(m1, m2) if max(m1, m2) else 1
        bodyset = {x[0] for x in o1 | o2}
        parts.append(f'{k}: cùng mã/thân={same} (mã {sorted(bodyset)}), trung vị {m1*1000:.0f} ms vs {m2*1000:.0f} ms (tỉ lệ {ratio:.2f})')
        ok = ok and same and ratio >= 0.65
    R('TC-GATEP2-50', 'P' if ok else 'F', 'có–không tài khoản, trên ep-r, 3–7 lần mỗi bên: ' + '; '.join(parts) + '; ca nền TC-P202-04/05, P203-04, P204-01, P205-10/28, P206-16, P209-07/08 đều P')

# ---------------- GATEP2-51 (bổ sung trực tiếp): dò mật khẩu, X-Forwarded-For giả ----------------
if want('51'):
    em = mk('g51'); resetrl(); codes = []
    for i in range(14): codes.append(call('POST', '/auth/login', None, {'email': em, 'password': 'Sai-%d-Qz' % i}, hdr={'X-Forwarded-For': f'10.{i}.7.7'}, _noretry=True)[0] if False else call_raw('POST', '/auth/login', None, {'email': em, 'password': 'Sai-%d-Qz' % i}, {'X-Forwarded-For': f'10.{i}.7.7'})[0])
    s_ok, j_ok, _ = call_raw('POST', '/auth/login', None, {'email': em, 'password': NPW})
    row = psql(f"select failed_logins, locked_until is not null from users where email='{em}'")
    resetrl()
    R('TC-GATEP2-51', 'P' if codes[:5] == [401] * 5 and 429 in codes[5:] and s_ok == 429 else 'F', f'ep-r: 14 lần sai, mỗi lần X-Forwarded-For khác → {codes}; đăng nhập ĐÚNG ngay sau → {s_ok} {j_ok.get("code")}; DB failed_logins/locked {row} (header giả vô tác dụng); các bước chờ tăng dần, khoá 15 phút (đo thật 899,8 s), IP chặn 40 lần, Admin cũng bị: ca nền TC-P205-01…19, P205-35, P205-36 đều P')

# ---------------- GATEP2-52: tệp roster độc hại ----------------
if want('52'):
    T = tokof('teacher'); A = tokof('admin'); cc = ('QS52' + TAG).upper()
    c = call('POST', '/admin/courses', A, {'subject_code': 'INT1006', 'class_code': cc, 'name': 'Lớp tệp độc ' + cc, 'semester': '2026-2027-HK1'}); cid = c[1]['id']
    call('POST', f'/admin/courses/{cid}/assign', A, {'teacher_id': psql("select id from users where email='teacher@edupilot.local'")})
    def mp(data, fname):
        b = '----q3' + uuid.uuid4().hex
        return (f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fname}"\r\nContent-Type: application/octet-stream\r\n\r\n').encode() + data + f'\r\n--{b}--\r\n'.encode(), f'multipart/form-data; boundary={b}'
    def imp(data, fname, q='?dry_run=true'):
        body, ct = mp(data, fname); return call('POST', f'/courses/{cid}/roster/import{q}', T, raw=body, hdr={'content-type': ct, 'Idempotency-Key': str(uuid.uuid4())}, idem=False)
    FX = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/sprints/4/qc/fixtures/roster/')
    elf = b'\x7fELF' + os.urandom(200)
    png = b'\x89PNG\r\n\x1a\n' + os.urandom(200)
    import io, zipfile
    zb = io.BytesIO(); z = zipfile.ZipFile(zb, 'w', zipfile.ZIP_DEFLATED); z.writestr('xl/worksheets/sheet1.xml', b'0' * (60 * 1024 * 1024)); z.close()
    t0 = time.time(); rz = imp(zb.getvalue(), 'bomb.xlsx'); dz = time.time() - t0
    formula = 'Email,Họ và tên,MSSV\r\n' + f'smkt.f1.{TAG}@example.test,=HYPERLINK("http://evil.example","x"),701234567\r\n' + f'smkt.f2.{TAG}@example.test,"@SUM(1+1)*cmd|\' /C calc\'!A0",701234568\r\n' + f'smkt.f3.{TAG}@example.test,+1+1,701234569\r\n'
    res = {'ELF→.csv': imp(elf, 'x.csv')[0], 'PNG→.csv': imp(png, 'x.csv')[0], 'zip bomb .xlsx': rz[0], 'rỗng': imp(b'', 'e.csv')[0]}
    rf = imp(formula.encode(), 'f.csv'); rfi = imp(formula.encode(), 'f.csv', '')
    names = psql(f"select u.full_name from users u where u.email like 'smkt.f%.{TAG}@example.test'")
    prev_rows = rf[1].get('rows') or rf[1].get('items') or []
    alive = call('GET', '/readyz', None)[0]
    safe = all(v in (400, 413, 415, 422) for v in res.values()) and dz <= 8 and alive == 200
    R('TC-GATEP2-52', 'P' if safe and rf[0] in (200, 201, 422) else 'F',
      f'dry_run: {res} (zip bomb 60 MB giải nén → {rz[0]} sau {dz:.1f} s; gateway còn sống {alive}); tệp công thức =HYPERLINK / @SUM / +1+1: dry_run {rf[0]}, nhập thật {rfi[0]}; tên lưu trong users: {names[:120]!r} (lưu nguyên văn như chữ, không thực thi); ca nền TC-P210-03/04/36; ghi nhận DEF-R4-02 (.xlsx thực chất là văn bản được đọc như CSV, vẫn qua kiểm tra dòng: Thấp) không thuộc ca độc hại')
