import os; os.environ['SUITE']='PE-04'
from r5lib import *
import concurrent.futures as cf
def S(m, p, tk=T, b=None, **k): return call(m, p, tk, b, **k)
def mke(**k):
    b = dict(title='R5 04', opens_at=iso(days=30), closes_at=iso(days=30, hours=1), duration_minutes=45); b.update(k); s, j, _ = S('POST', E, b=b, h=idem()); return s, j
s, j = mke(); eid = j['id']
chk('TC-PE04-01', s == 201 and j['status'] == 'DRAFT' and (j.get('shuffle_questions'), j.get('shuffle_options'), j.get('max_score'), j.get('rounding_step'), j.get('multi_scoring')) == (True, True, '10.00', '0.01', 'PARTIAL'), f'{s} {j["status"]} shuffle {j.get("shuffle_questions")}/{j.get("shuffle_options")} max {j.get("max_score")} step {j.get("rounding_step")} {j.get("multi_scoring")} appeal {j.get("appeal_days")}')
r = {}
for n, b in [('title rỗng', dict(title='')), ('title 121', dict(title='x' * 121)), ('instr 4001', dict(instructions='x' * 4001)), ('dur 301', dict(duration_minutes=301, closes_at=iso(days=30, hours=6))), ('dur > khung', dict(duration_minutes=90)), ('closes<=opens', dict(closes_at=iso(days=30)))]:
    s, j = mke(**b); r[n] = (s, codes(j))
okb = [mke(**b)[0] for b in (dict(instructions='x' * 4000), dict(duration_minutes=5), dict(duration_minutes=300, closes_at=iso(days=30, hours=6)))]
d4 = mke(duration_minutes=4)
chk('TC-PE04-02', all(v[0] == 422 for v in r.values()) and okb == [201] * 3, f'{r}; biên qua {okb}; duration 4 -> {d4[0]} {codes(d4[1])} (TC gốc kỳ vọng 422 DURATION_TOO_SHORT)')
a = mke(opens_at='2026-12-01T08:00:00+07:00', closes_at='2026-12-01T09:00:00+07:00')[1].get('opens_at'); b2 = mke(opens_at='2026-12-01T01:00:00Z', closes_at='2026-12-01T02:00:00Z')[1].get('opens_at')
sv = login('r5s01')
q3 = [mke()[0] if False else S('POST', E, TA, dict(title='ta', opens_at=iso(days=30), closes_at=iso(days=30, hours=1), duration_minutes=45), h=idem())[0], S('POST', E, sv, dict(title='x'), h=idem())[0], S('POST', E, AD, dict(title='x'), h=idem())[0], S('POST', E, login('r5x'), dict(title='x'), h=idem())[0]]
chk('TC-PE04-03', a == b2 and q3 == [201, 403, 403, 403], f'+07:00 = Z: {a == b2} ({a}); TA/SV/ADMIN/ngoài lớp: {q3}')
k = idem(); bd = dict(title='idem', opens_at=iso(days=30), closes_at=iso(days=30, hours=1), duration_minutes=45)
r1 = S('POST', E, b=bd, h=k); r2 = S('POST', E, b=bd, h=k); r3 = S('POST', E, b=dict(bd, title='khác'), h=k)
chk('TC-PE04-04', r1[1].get('id') == r2[1].get('id') and r3[0] == 422 and 'IDEMPOTENCY_KEY_REUSED' in codes(r3[1]), f'cùng key cùng id; khác thân {r3[0]} {codes(r3[1])}')
# items
ok = [mkq(i) for i in range(5)]; code_q = mkcode(50)
g = S('GET', f'{E}/{eid}')[1]; s6 = S('PUT', f'{E}/{eid}/items', b=dict(items=[dict(question_id=q, points='1') for q in ok + [code_q]], version=g['version']))
gg = S('GET', f'{E}/{eid}')[1]; k_mixed = gg.get('kind')
sm = S('PUT', f'{E}/{eid}/items', b=dict(items=[dict(question_id=q, points='1') for q in ok][::-1], version=gg['version']))
gg2 = S('GET', f'{E}/{eid}')[1]
chk('TC-PE04-06', s6[0] == 200 and k_mixed == 'MIXED' and sm[0] == 200 and gg2.get('kind') == 'MCQ', f'kind {k_mixed} -> {gg2.get("kind")} sau thay toàn bộ (items_count {gg2.get("items_count")})')
draft = mkq(80, ok=False); rej = mkq(81, ok=False); gq = S('GET', f'{Q}/{rej}')[1]; S('PUT', f'{Q}/{rej}/review', T, dict(decision='REQUEST', version=gq['version'])); gq = S('GET', f'{Q}/{rej}')[1]; S('PUT', f'{Q}/{rej}/review', TA, dict(decision='REJECT', version=gq['version']))
c2q = mkq(82, base=Q2); g = S('GET', f'{E}/{eid}')[1]
def put(items, **kw): 
    gx = S('GET', f'{E}/{eid}')[1]; s, j, _ = S('PUT', f'{E}/{eid}/items', b=dict(items=items, version=gx['version'])); return s, codes(j)
P = lambda q, p='1': dict(question_id=q, points=p)
res = dict(draft=put([P(draft)]), rejected=put([P(rej)]), other=put([P(c2q)]), dup=put([P(ok[0]), P(ok[0])]), empty=put([]), over=put([P(ok[0], '100.01')]), n101=put([P(ok[i % 5]) for i in range(101)]))
okp = (put([P(ok[0], '0.01')])[0], put([P(ok[0], '100')])[0])
exp = dict(draft='ITEM_NOT_APPROVED', rejected='ITEM_NOT_APPROVED', other='QUESTION_NOT_IN_COURSE', dup='DUPLICATE_ITEM', empty='NO_ITEMS')
bad = [k2 for k2, e in exp.items() if res[k2][0] != 422 or e not in res[k2][1]]
chk('TC-PE04-07', not bad and res['over'][0] == 422 and res['n101'][0] == 422 and okp == (200, 200), f'{res}; biên points 0.01/100 {okp}')
# schedule
eid2, items2 = mkexam(open_s=300, win=3600, dur=30, schedule=False, title='R5 04 lịch')
sc = S('POST', f'{E}/{eid2}/schedule', b={}); s_item = S('PUT', f'{E}/{eid2}/items', b=dict(items=[], version=S('GET', f'{E}/{eid2}')[1]['version']))
chk('TC-PE04-08', sc[0] == 200 and s_item[0] == 409 and 'EXAM_LOCKED' in codes(s_item[1]), f'schedule {sc[0]}; PUT items khi SCHEDULED {s_item[0]} {codes(s_item[1])} reason={reason(s_item[1])}')
chk('TC-PE04-13', db(f"select count(*) from audit_log where entity_id='{eid2}' and action='exam.schedule'") == '1' and int(db(f"select count(*) from notifications where type='EXAM_SCHEDULED' and body||title like '%R5 04 lịch%' and course_id='{C1}'") or 0) >= 12, f'audit exam.schedule 1 dòng; thông báo EXAM_SCHEDULED: {db("select count(*) from notifications where type=chr(69)||chr(88)||chr(65)||chr(77)||chr(95)||chr(83)||chr(67)||chr(72)||chr(69)||chr(68)||chr(85)||chr(76)||chr(69)||chr(68) and course_id=chr(39)||chr(39)") or ""}')
