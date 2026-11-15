import os; os.environ['SUITE']='PE-03'
from r5lib import *
import io, zipfile
def S(m, p, tk=T, b=None, **k): return call(m, p, tk, b, **k)
# TC01-05 MCQ
mc = dict(type='MCQ_SINGLE', title='t', topic='Mật mã', difficulty='EASY', stem='2+2=?', options=[{'body': '3'}, {'body': '4'}], correct=[1])
s, j, _ = S('POST', Q, b=mc); ids = [o['id'] for o in j.get('options', [])]
chk('TC-PE03-01', s == 201 and j.get('review_status') == 'DRAFT' and len(ids) == 2 and all(re.fullmatch(r'[0-9a-f-]{36}', x) for x in ids) and j['answer_key'].get('option_ids') == [ids[1]] or (s == 201 and ids and ids[1] in json.dumps(j['answer_key'])), f'{s} {j.get("review_status")} key={j.get("answer_key")}')
bad = {}
def ok422(name, body, exp):
    s, j, _ = S('POST', Q, b=body); bad[name] = (s, codes(j)); return s == 422 and exp in codes(j)
cases = [('1 đáp án', dict(mc, options=[{'body': 'a'}], correct=[0]), 'OPTION_COUNT'), ('9 đáp án', dict(mc, options=[{'body': str(i)} for i in range(9)]), 'OPTION_COUNT'), ('body rỗng', dict(mc, options=[{'body': ''}, {'body': 'x'}]), None), ('trùng', dict(mc, options=[{'body': 'a'}, {'body': ' a '}], correct=[0]), 'DUPLICATE_OPTION'), ('single 2 đúng', dict(mc, correct=[0, 1]), 'SINGLE_MULTIPLE_CORRECT')]
res = []
for n, b, e in cases:
    s, j, _ = S('POST', Q, b=b); res.append((n, s, codes(j)))
chk('TC-PE03-02', all(s == 422 for _, s, _ in res) and ('OPTION_COUNT' in res[0][2]) and ('DUPLICATE_OPTION' in res[3][2]) and ('SINGLE_MULTIPLE_CORRECT' in res[4][2]), str(res)[:250])
r3 = [S('POST', Q, b=dict(mc, options=[{'body': str(i)} for i in range(n)], correct=[0]))[0] for n in (2, 8)] + [S('POST', Q, b=dict(mc, stem='x' * 8000))[0], S('POST', Q, b=dict(mc, title='t' * 120))[0], S('POST', Q, b=dict(mc, stem='x' * 8001))[0]]
chk('TC-PE03-03', r3[:4] == [201] * 4 and r3[4] == 422, str(r3))
xs = '<script>alert(1)</script><img src=x onerror=alert(1)> đậm [x](javascript:alert(1))'
s, j, _ = S('POST', Q, b=dict(mc, stem=xs, options=[{'body': '<b>x</b>'}, {'body': 'y'}], correct=[0])); g = S('GET', f'{Q}/{j["id"]}')[1]
chk('TC-PE03-04', g['stem'] == xs and g['options'][0]['body'] == '<b>x</b>', 'API trả nguyên văn (phần hiển thị UI: S)')
w = [S('POST', Q, b=dict(mc, options=[{'body': 'a'}, {'body': t}], correct=[0]))[1] for t in ('Tất cả các đáp án trên', 'Cả hai đáp án', 'Không có đáp án nào', 'A và B')]
wn = S('POST', Q, b=dict(mc, options=[{'body': 'a'}, {'body': 'Tất cả các đáp án trên', 'pinned_last': True}], correct=[0]))[1]
chk('TC-PE03-05', all(x.get('details', {}).get('warnings') if isinstance(x.get('details'), dict) else x.get('warnings') for x in w) and not (wn.get('warnings') or (wn.get('details') or {}).get('warnings')), f'4 cảnh báo; pinned_last không cảnh báo. mẫu: {json.dumps(w[0])[-150:]}')
# TC07 TRUE_FALSE
tf = dict(type='TRUE_FALSE', title='tf', topic='x', stem='s', value=True)
a = S('POST', Q, b=tf); b2 = S('POST', Q, b=dict(tf, value=None)); b3 = S('POST', Q, b=dict(tf, options=[{'body': 'a'}])); b4 = S('POST', Q, b=dict(tf, type='SHORT')); b5 = S('POST', Q, b=dict(tf, type='ESSAY')); b6 = S('POST', Q, b=dict(tf, type='XYZ'))
chk('TC-PE03-07', a[0] == 201 and a[1]['answer_key'] == {'value': True} and b2[0] == 422 and b3[0] == 422 and b6[0] == 422, f'{a[0]} {a[1].get("answer_key")}; thiếu {b2[0]}; options thừa {b3[0]}; SHORT {b4[0]} {codes(b4[1])}; ESSAY {b5[0]}; XYZ {b6[0]}')
rec('TC-PE03-08', 'LT', 'SHORT/ESSAY nay được ngân hàng nhận (P-10-5, sprint 10) — ca gốc "SHORT/ESSAY → TYPE_NOT_SUPPORTED" lỗi thời' if b4[0] == 201 else f'SHORT {b4[0]}')
# code problem
s, j, _ = S('POST', Q, b=dict(type='CODE', title='ct', topic='x', difficulty='EASY', stem='s')); qid = j['id']; P = f'{Q}/{qid}'; v = [j['version']]
def code(**k):
    s, j, _ = S('PUT', P + '/code', b=dict(version=v[0], **k))
    if s == 200: v[0] = j.get("version", v[0])
    return s, codes(j), (j.get('code') or {})
s, _, j = code(languages=['cpp17'])
chk('TC-PE03-09', (j.get('time_limit_ms'), j.get('memory_limit_mb'), j.get('output_limit_kb'), j.get('checker')) == (1000, 256, 1024, 'EXACT'), f'{s} {[j.get(k) for k in ["time_limit_ms","memory_limit_mb","output_limit_kb","checker","tests_version"]]}')
tl = [code(languages=['cpp17'], time_limit_ms=x)[0] for x in (99, 100, 10000, 10001)]; mm = [code(languages=['cpp17'], memory_limit_mb=x)[0] for x in (15, 16, 512, 513)]; ol = [code(languages=['cpp17'], output_limit_kb=x)[0] for x in (0, 1, 16384, 16385)]
fe = [code(languages=['cpp17'], checker='FLOAT_EPS')[:2], code(languages=['cpp17'], checker='EXACT', float_eps='0.1')[:2], code(languages=['java'])[:2], code(languages=[])[:2]]
chk('TC-PE03-10', tl == [422, 200, 200, 422] and mm == [422, 200, 200, 422] and ol == [422, 200, 200, 422] and all(x[0] == 422 for x in fe), f'tl {tl} mem(16..512 theo #5/v1.6; TC gốc ghi 1024 = lỗi thời một phần) {mm} out {ol} {fe}')
code(languages=['c11', 'cpp17'], time_limit_ms=1000, memory_limit_mb=256, output_limit_kb=1024, checker='EXACT', float_eps=None)
tv = []
for kw in [dict(languages=['c11', 'cpp17'], time_limit_ms=1500), dict(languages=['c11', 'cpp17'], time_limit_ms=1500, checker='TOKENS'), dict(languages=['cpp17'], time_limit_ms=1500, checker='TOKENS'), dict(languages=['cpp17'], time_limit_ms=1500, checker='TOKENS', starter_code={'cpp17': 'int main(){}'})]:
    tv.append(code(**kw)[2].get('tests_version'))
chk('TC-PE03-11', tv[1] > tv[0] and tv[2] > tv[1] and tv[3] == tv[2], f'tests_version sau đổi: {tv} (starter_code không tăng)')
# test CRUD
TC = P + '/testcases'
def tcn(**k): return S('POST', TC, b=dict(name='t', input='1 2\n', expected='3\n', **k))
d = tcn(); wts = [(x, tcn(weight=x)[0]) for x in (-1, 1001, 1.5, 0, 1000)]
nm = [S('POST', TC, b=dict(name=n, input='1', expected='1'))[0] for n in ('', 'x' * 61, 'x' * 60)]
bg = S('POST', TC, b=dict(name='big', input='a' * 65537, expected='1'))
chk('TC-PE03-13', d[0] == 201 and d[1].get('weight') == 1 and [c for _, c in wts] == [422, 422, 422, 201, 201] and nm == [422, 422, 201] and bg[0] == 201, f'default weight {d[1].get("weight")}; weights {wts}; name {nm}; 64KiB+1 {bg[0]}')
v14 = S('GET', P)[1]['code']['tests_version']; r = tcn(); v14b = S('GET', P)[1]['code']['tests_version']
chk('TC-PE03-14', v14b == v14 + 1, f'POST test: tests_version {v14}->{v14b}')
sv = login('r5s01'); p15 = [S('GET', TC, sv)[0], S('GET', TC, AD)[0], S('GET', TC, login('r5x'))[0], S('GET', f'/courses/{C2}/questions/{qid}/testcases', T)[0], S('GET', TC, TA)[0], S('GET', TC, T)[0]]
chk('TC-PE03-15', p15[:3] == [403] * 3 and p15[3] in (403, 404) and p15[4:] == [200, 200], str(p15))
# zip
z = mkzip({'sample1.in': '1 2', 'sample1.out': '3', 't2.in': '2 2', 't2.out': '4', 't10.in': '3 3', 't10.ans': '6', 't3.in': '4 4', 't3.out': '8', 't4.in': '5 5', 't4.out': '10'})
n0 = len(S('GET', TC)[1]['items']); b, ct = multipart({}, 't.zip', z); sd, jd, td = S('POST', TC + '/import?dry_run=true', raw=b, ctype=ct); n1 = len(S('GET', TC)[1]['items'])
b, ct = multipart({}, 't.zip', z); si, ji, ti = S('POST', TC + '/import', raw=b, ctype=ct); l = S('GET', TC)[1]['items']
chk('TC-PE03-17', sd == 200 and n1 == n0 and si in (200, 201) and len(l) == n0 + 5 and any(x['is_sample'] for x in l), f'dry {sd} {td[:80]} (ghi {n1 - n0}); thật {si}, +{len(l) - n0}, sample {[x["name"] for x in l if x["is_sample"]][:2]}')
bad18 = []
for name, files in [('thiếu cặp', {'a.in': '1'}), ('trùng', {'a.in': '1', 'a.out': '1', 'a.ans': '1'}), ('lồng', {'d/a.in': '1', 'd/a.out': '1'}), ('../evil', {'../evil.in': '1', '../evil.out': '1'}), ('/abs', {'/abs.in': '1', '/abs.out': '1'}), ('không UTF-8', {'a.in': b'\xff\xfe', 'a.out': '1'})]:
    b, ct = multipart({}, 't.zip', mkzip(files)); s, j, t = S('POST', TC + '/import', raw=b, ctype=ct); bad18.append((name, s))
big = mkzip({f't{i}.in': '1' for i in range(101)} | {f't{i}.out': '1' for i in range(101)}); b, ct = multipart({}, 't.zip', big); s101 = S('POST', TC + '/import', raw=b, ctype=ct)[0]
n2 = len(S('GET', TC)[1]['items'])
chk('TC-PE03-18', all(s == 422 for _, s in bad18) and s101 == 422 and n2 == len(l), f'{bad18} 101 test {s101}; số test không đổi {len(l)}->{n2}')
slip = sh = subprocess.run("find /tmp /var /Users/kuro/Documents -maxdepth 4 -name 'qc-slip*' 2>/dev/null | head", shell=True, capture_output=True, text=True).stdout.strip()
b, ct = multipart({}, 't.zip', mkzip({'../../../../tmp/qc-slip.in': '1', '../../../../tmp/qc-slip.out': '1'})); s, j, t = S('POST', TC + '/import', raw=b, ctype=ct)
slipc = subprocess.run("docker exec ep-r-gateway-1 sh -c 'find / -name \"qc-slip*\" 2>/dev/null | head'; docker exec ep-r-worker-1 sh -c 'find / -name \"qc-slip*\" 2>/dev/null | head'", shell=True, capture_output=True, text=True).stdout.strip()
chk('TC-PE03-19', s == 422 and not slip and not slipc, f'zip-slip {s}; tệp ngoài DB: host "{slip}" container "{slipc}"')
def bomb(sz):
    bio = io.BytesIO()
    with zipfile.ZipFile(bio, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as zz: zz.writestr('a.in', '0' * sz); zz.writestr('a.out', '1')
    return bio.getvalue()
t0 = time.time(); bz = bomb(60 << 20); b, ct = multipart({}, 't.zip', bz); s, j, t = S('POST', TC + '/import', raw=b, ctype=ct); dtm = time.time() - t0
b, ct = multipart({}, 't.zip', b'PK' + b'\x00' * (11 << 20)); s2, j2, _ = S('POST', TC + '/import', raw=b, ctype=ct)
chk('TC-PE03-20', s == 422 and dtm <= 5 and s2 in (413, 422) and call('GET', '/readyz')[0] == 200, f'bom 60MiB({len(bz)}B nén) {s} {codes(j)} {dtm:.1f}s; zip 11MiB {s2}')
b, ct = multipart({}, 't.zip', mkzip({'r1.in': '1 1', 'r1.out': '2', 'sample.in': '1 1', 'sample.out': '2'})); n3 = S('POST', TC + '/import?mode=replace', raw=b, ctype=ct); l3 = S('GET', TC)[1]['items']
chk('TC-PE03-21', n3[0] in (200, 201) and len(l3) == 2, f'replace {n3[0]}: còn {len(l3)} test (mong 2)')
# TC23-26 verify (mkcode)
qa = mkcode(31, approve_q=False)  # verify ok
g = S('GET', f'{Q}/{qa}')[1]['code']; chk('TC-PE03-23', g['reference_verified_version'] == g['tests_version'] and g['reference_verified_at'], f'verified_version {g["reference_verified_version"]} = tests_version {g["tests_version"]}')
la = db("select count(*) from llm_audit"); qb = mkcode(32, tests=[('s', '1 2\n', '3\n', True, 1), ('h', '1 1\n', '999\n', False, 1)], approve_q=False, verify=False)
sv_, jv, _ = S('POST', f'{Q}/{qb}/reference/verify', b={}, h=idem()); jj = wait_job(jv['job_id']) if sv_ == 202 else {}
gb = S('GET', f'{Q}/{qb}')[1]['code']
chk('TC-PE03-24', sv_ == 202 and 'ok' in json.dumps(jj.get('result', jj)) and gb['reference_verified_version'] is None and db("select count(*) from llm_audit") == la, f'verify lệch: {sv_} {json.dumps(jj.get("result"))[:200]}; cờ {gb["reference_verified_version"]}; llm_audit không đổi')
k = idem(); r1 = S('POST', f'{Q}/{qa}/reference/verify', b={}, h=k); r2 = S('POST', f'{Q}/{qa}/reference/verify', b={}, h=k); r3 = S('POST', f'{Q}/{qa}/reference/verify', b={})
chk('TC-PE03-25', r1[1].get('job_id') == r2[1].get('job_id') and r3[0] == 422 and 'IDEMPOTENCY_KEY_REQUIRED' in codes(r3[1]), f'cùng key cùng job; thiếu key {r3[0]} {codes(r3[1])} (nhánh judge dừng: xem PE-02 nhóm judge-down)')
# review
qm = mkq(40, ok=False); g = S('GET', f'{Q}/{qm}')[1]
r27 = [S('PUT', f'{Q}/{qm}/review', T, dict(decision='REQUEST', version=g['version']))]
g = S('GET', f'{Q}/{qm}')[1]; r27.append(S('PUT', f'{Q}/{qm}/review', TA, dict(decision='APPROVE', version=g['version'] + 3)))
r27.append(S('PUT', f'{Q}/{qm}/review', sv, dict(decision='APPROVE', version=g['version']))); r27.append(S('PUT', f'{Q}/{qm}/review', AD, dict(decision='APPROVE', version=g['version'])))
r27.append(S('PUT', f'{Q}/{qm}/review', TA, dict(decision='APPROVE', version=g['version']))); fin = S('GET', f'{Q}/{qm}')[1]
chk('TC-PE03-27', [x[0] for x in r27] == [200, 409, 403, 403, 200] and fin['review_status'] == 'APPROVED' and fin.get('reviewed_by'), str([x[0] for x in r27]) + ' ' + str(fin['review_status']))
# 28 điều kiện câu code
def newcode(): s, j, _ = S('POST', Q, b=dict(type='CODE', title='c28', topic='x', stem='s')); code_ = j['id']; vv = j['version']; S('PUT', f'{Q}/{code_}/code', b=dict(languages=['cpp17'], reference=dict(language='cpp17', source=SUM), version=vv)); return code_
def appr(qq): 
    g = S('GET', f'{Q}/{qq}')[1]; S('PUT', f'{Q}/{qq}/review', T, dict(decision='REQUEST', version=g['version'])); g = S('GET', f'{Q}/{qq}')[1]; r = S('PUT', f'{Q}/{qq}/review', TA, dict(decision='APPROVE', version=g['version'])); return r[0], codes(r[1])
c0 = newcode(); a0 = appr(c0)
c1 = newcode(); S('POST', f'{Q}/{c1}/testcases', b=dict(name='s', input='1', expected='1', is_sample=True)); a1 = appr(c1)
c2 = newcode(); S('POST', f'{Q}/{c2}/testcases', b=dict(name='h', input='1', expected='1', weight=0)); a2 = appr(c2)
chk('TC-PE03-28', a0[0] == 422 and a1[0] == 422 and a2[0] == 422, f'không test {a0}; chỉ mẫu {a1}; Σweight=0/chỉ ẩn {a2}')
# 29
dbr = subprocess.run(['docker','exec','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-c',"update question_bank set review_status='APPROVED' where origin='AI_DRAFT' and reviewed_by is null and id in (select id from question_bank where origin='AI_DRAFT' limit 1)"],capture_output=True,text=True)
have_ai = db("select count(*) from question_bank where origin='AI_DRAFT' and review_status<>'APPROVED'")
rec('TC-PE03-29', 'S' if have_ai == '0' else ('P' if '23514' in dbr.stderr or 'violates check' in dbr.stderr else 'F'), f'AI_DRAFT chưa duyệt: {have_ai}; UPDATE trực tiếp: {dbr.stderr.strip()[:120]}')
# 31/32 khoá khi đang dùng
eidk, itk = mkexam(open_s=3600, win=3600, dur=30, schedule=False, title='R5 03 khoá'); qk = itk[0]; gk = S('GET', f'{Q}/{qk}')[1]
sch = S('POST', f'{E}/{eidk}/schedule', b={}); gk = S('GET', f'{Q}/{qk}')[1]
e1 = S('PUT', f'{Q}/{qk}', b=dict(type='MCQ_SINGLE', title=gk['title'], topic=gk['topic'], stem='đổi stem', options=[{'body': 'a'}, {'body': 'b'}], correct=[0], version=gk['version']))
gk2 = S('GET', f'{Q}/{qk}')[1]
e2 = S('PUT', f'{Q}/{qk}', b=dict(type='MCQ_SINGLE', title=gk['title'], topic='đổi topic', difficulty='HARD', stem=gk['stem'], options=[{'body': o['body']} for o in gk['options']], correct=[1], explanation='mới', version=gk2['version']))
ar = S('POST', f'{Q}/{qk}/archive', b={})
chk('TC-PE03-31', e1[0] == 409 and 'QUESTION_IN_USE' in codes(e1[1]) and e2[0] in (200, 409), f'sửa stem/đáp án khi bài đã lên lịch: {e1[0]} {codes(e1[1])}; sửa topic/difficulty/explanation: {e2[0]} {codes(e2[1])}; archive {ar[0]}')
chk('TC-PE03-32', ar[0] == 200, f'archive câu đang dùng: {ar[0]}')
rec('TC-PE03-33', 'S', 'sửa test sau đóng + regrade: gộp PE-08 (regrade)')
# 43/44/46
l = S('GET', Q + '?review_status=APPROVED&type=MCQ_SINGLE&limit=100')[1]['items']; lq = S('GET', Q + '?q=' + 'x' * 101); lh = S('GET', Q + '?q=ĐÁP')
chk('TC-PE03-43', l and all(x['review_status'] == 'APPROVED' and x['type'] == 'MCQ_SINGLE' for x in l) and lq[0] == 422, f'lọc {len(l)} khớp; q 101 ký tự {lq[0]}; q hoa/thường {lh[0]}')
m44 = {}
for who, tk in [('SV', sv), ('ADMIN', AD), ('NGOÀI', login('r5x')), ('TA', TA), ('GV', T)]:
    m44[who] = [S('GET', Q, tk)[0], S('GET', f'{Q}/{qm}', tk)[0], S('POST', Q, tk, dict(tf))[0]]
chk('TC-PE03-44', m44['SV'] == [403] * 3 and m44['ADMIN'] == [403] * 3 and m44['NGOÀI'] == [403] * 3 and m44['TA'] == [200, 200, 201] and m44['GV'] == [200, 200, 201], str(m44))
dp = S('POST', f'{Q}/{qm}/duplicate', b={}); 
chk('TC-PE03-46', dp[0] == 201 and dp[1]['review_status'] == 'DRAFT' and dp[1]['title'].endswith('(bản sao)') and dp[1]['origin'] == 'MANUAL', f'{dp[0]} {dp[1].get("title")} {dp[1].get("review_status")}')
# 53 today
t53 = {u: call('GET', '/me/today', login(u))[2] for u in ('r5t', 'r5ta', 'r5s01')}
chk('TC-PE03-53', 'QUESTION_REVIEW' in t53['r5t'] and 'QUESTION_REVIEW' not in t53['r5s01'], f'GV có QUESTION_REVIEW: {"QUESTION_REVIEW" in t53["r5t"]}; TA: {"QUESTION_REVIEW" in t53["r5ta"]}; SV: {"QUESTION_REVIEW" in t53["r5s01"]}')
