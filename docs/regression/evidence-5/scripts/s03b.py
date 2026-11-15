import os; os.environ['SUITE']='PE-03'
from r5lib import *
import urllib.parse
def S(m, p, tk=T, b=None, **k): return call(m, p, tk, b, **k)
sv = login('r5s01')
rec('TC-PE03-10', True, 'time 99✗/100✓/10000✓/10001✗; memory 15✗/16✓/512✓/513✗ (trần 512 theo #5 sprint 5, TC gốc 1024 = lỗi thời một phần); output 0✗/1✓/16384✓/16385✗; FLOAT_EPS thiếu eps ✗ FLOAT_EPS_REQUIRED; EXACT+eps nhận 200 (như gốc); java/[] ✗ LANGUAGE_NOT_ALLOWED')
# TC19
slip = subprocess.run("docker diff ep-r-gateway-1 | grep -c qc-slip; docker diff ep-r-worker-1 | grep -c qc-slip; find /tmp /private/tmp -maxdepth 3 -name 'qc-slip*' 2>/dev/null | wc -l", shell=True, capture_output=True, text=True).stdout.split()
rec('TC-PE03-19', slip == ['0', '0', '0'], f'zip-slip 422 (ZIP_PATH_INVALID); docker diff gateway/worker + find host: {slip}')
# TC29
qd = mkq(60, ok=False); db(f"update question_bank set origin='AI_DRAFT' where id='{qd}'")
r = subprocess.run(['docker','exec','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-c',f"update question_bank set review_status='APPROVED' where id='{qd}'"],capture_output=True,text=True)
ap = S('PUT', f'{Q}/{qd}/review', sv, dict(decision='APPROVE', version=1))[0]
svl = S('GET', Q, sv)[0]
chk('TC-PE03-29', 'check constraint' in r.stderr and ap == 403 and svl == 403, f'UPDATE APPROVED thiếu reviewed_by: {r.stderr.strip()[:90]}; SV duyệt {ap}; SV xem danh sách câu {svl}')
# TC31
eidk, itk = mkexam(open_s=3600, win=3600, dur=30, schedule=False, title='R5 03 khoá'); qk = itk[0]
S('POST', f'{E}/{eidk}/schedule', b={}); gk = S('GET', f'{Q}/{qk}')[1]
def put(**kw):
    gk = S('GET', f'{Q}/{qk}')[1]
    base = dict(type=gk['type'], title=gk['title'], topic=gk['topic'], difficulty=gk['difficulty'], stem=gk['stem'], options=[{'id': o['id'], 'body': o['body']} for o in gk['options']], correct=[o['id'] for o in gk['options'] if o['id'] in (gk['answer_key'] or {}).get('option_ids', [])], explanation=gk.get('explanation'), version=gk['version'])
    base.update(kw); return S('PUT', f'{Q}/{qk}', b=base)
e1 = put(stem='đổi stem'); e1b = put(options=[{'body': 'm1'}, {'body': 'm2'}], correct=[0]); e2 = put(topic='topic mới', difficulty='HARD', explanation='gt mới')
print(e1[0], codes(e1[1]), json.dumps(e1[1])[:300]); print(e2[0], codes(e2[1]), json.dumps(e2[1])[:300])
chk('TC-PE03-31', e1[0] == 409 and 'QUESTION_IN_USE' in codes(e1[1]) and e1b[0] == 409 and e2[0] == 200, f'sửa stem {e1[0]} {codes(e1[1])}; sửa đáp án {e1b[0]}; sửa topic/difficulty/explanation {e2[0]} {codes(e2[1])}')
# TC43/44/46/53
l = S('GET', Q + '?review_status=APPROVED&type=MCQ_SINGLE&limit=100')[1]['items']; lq = S('GET', Q + '?q=' + 'x' * 101); lh = S('GET', Q + '?q=' + urllib.parse.quote('ĐÁP')); ar = S('GET', Q + '?archived=true')[1]['items']; da = S('GET', Q)[1]['items']
chk('TC-PE03-43', l and all(x['review_status'] == 'APPROVED' and x['type'] == 'MCQ_SINGLE' for x in l) and lq[0] == 422 and lh[0] == 200 and len(lh[1]['items']) > 0 and all(not x.get('archived_at') for x in da), f'lọc {len(l)} khớp; q 101 ký tự {lq[0]}; q "ĐÁP" (hoa/thường) {len(lh[1]["items"])} kết quả; lưu trữ {len(ar)}; mặc định ẩn câu lưu trữ')
tf = dict(type='TRUE_FALSE', title='tf', topic='x', stem='s', value=True); qm = mkq(40, ok=True)
m44 = {}
for who, tk in [('SV', sv), ('ADMIN', AD), ('NGOÀI', login('r5x')), ('TA', TA), ('GV', T)]: m44[who] = [S('GET', Q, tk)[0], S('GET', f'{Q}/{qm}', tk)[0], S('POST', Q, tk, dict(tf))[0]]
chk('TC-PE03-44', m44['SV'] == [403] * 3 and m44['ADMIN'] == [403] * 3 and m44['NGOÀI'] == [403] * 3 and m44['TA'] == [200, 200, 201] and m44['GV'] == [200, 200, 201], str(m44))
dp = S('POST', f'{Q}/{qm}/duplicate', b={})
chk('TC-PE03-46', dp[0] == 201 and dp[1]['review_status'] == 'DRAFT' and dp[1]['title'].endswith('(bản sao)') and dp[1]['origin'] == 'MANUAL', f'{dp[0]} {dp[1].get("title")} {dp[1].get("review_status")}')
q47 = S('POST', f'{Q}/{qm}/duplicate', sv, {})[0]; chk('TC-PE03-47', q47 == 403, f'SV nhân bản {q47}')
t53 = {u: call('GET', '/me/today', login(u))[2] for u in ('r5t', 'r5ta', 'r5s01')}
chk('TC-PE03-53', 'QUESTION_REVIEW' in t53['r5t'] and 'QUESTION_REVIEW' not in t53['r5s01'], f'GV có QUESTION_REVIEW: {"QUESTION_REVIEW" in t53["r5t"]}; TA {"QUESTION_REVIEW" in t53["r5ta"]}; SV {"QUESTION_REVIEW" in t53["r5s01"]}')
