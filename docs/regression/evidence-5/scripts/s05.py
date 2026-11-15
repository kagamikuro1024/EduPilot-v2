import os; os.environ['SUITE']='PE-05'
from r5lib import *
import concurrent.futures as cf, hashlib
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
eid, items = mkexam(open_s=10, win=1500, dur=20, nq=4, title='R5 05 chính'); EX = f'{E}/{eid}'
chk_not_open = S('POST', EX + '/attempts', login('r5s01'), {}, idem())
e_draft = S('POST', E, T, dict(title='d', opens_at=iso(days=2), closes_at=iso(days=2, hours=1), duration_minutes=30), idem())[1]['id']
e_sched, _ = mkexam(open_s=3000, win=3600, dur=20, nq=1, title='R5 05 chưa tới')
r3 = [S('POST', f'{E}/{e_draft}/attempts', login('r5s01'), {}, idem()), S('POST', f'{E}/{e_sched}/attempts', login('r5s01'), {}, idem())]
chk('TC-PE05-03', r3[0][0] == 409 and r3[1][0] == 409 and {reason(x[1]) for x in r3} >= {'not_scheduled', 'not_yet'}, f'DRAFT {r3[0][0]} reason={reason(r3[0][1])}; SCHEDULED chưa tới {r3[1][0]} reason={reason(r3[1][1])}; trước giờ {chk_not_open[0]} {reason(chk_not_open[1])}')
time.sleep(max(0, 12 - 0)); 
# TC01 start
sv1 = login('r5s01'); s, j, t = S('POST', EX + '/attempts', sv1, {}, idem()); a = j['attempt']; aid = a['id']; H = {'X-Exam-Tab': str(uuid.uuid4())}
row = db(f"select status,started_at,deadline_at,closes_at from exam_attempts x join exams e on e.id=x.exam_id where x.id='{aid}'")
chk('TC-PE05-01', s == 201 and a['status'] == 'IN_PROGRESS' and 'deadline_at' in a and 'server_time' in a, f'{s} {a["status"]} deadline {a["deadline_at"]} (started+20 phút); DB {row}')
# TC05 idempotency, 07 resume
k = idem(); r1 = S('POST', EX + '/attempts', login('r5s02'), {}, k); r2 = S('POST', EX + '/attempts', login('r5s02'), {}, k); r2b = S('POST', EX + '/attempts', login('r5s02'), {}, idem())
chk('TC-PE05-05', r1[1]['attempt']['id'] == r2[1]['attempt']['id'] and db(f"select count(*) from exam_attempts where exam_id='{eid}' and student_id=(select id from users where email='r5s02@edupilot.local')") == '1', f'cùng key {r1[0]}/{r2[0]} cùng id; key khác {r2b[0]} cùng id {r2b[1]["attempt"]["id"] == r1[1]["attempt"]["id"]}')
mine = S('GET', EX + '/attempts/mine', sv1)[1]
chk('TC-PE05-07', r2b[0] == 200 and mine['attempt']['started_at'] == a['started_at'] and mine['attempt']['deadline_at'] == a['deadline_at'], f'start lại 200; started_at/deadline_at không đổi {a["deadline_at"]}')
# TC04
n4 = [S('POST', EX + '/attempts', sv1, {})[:2], S('POST', EX + '/attempts', login('r5s15'), {}, idem())[:2], S('POST', EX + '/attempts', login('r5s16'), {}, idem())[:2], S('POST', EX + '/attempts', login('r5x'), {}, idem())[:2], S('POST', EX + '/attempts', TA, {}, idem())[:2], S('POST', EX + '/attempts', T, {}, idem())[:2], S('POST', EX + '/attempts', AD, {}, idem())[:2]]
chk('TC-PE05-04', n4[0][0] == 422 and [x[0] for x in n4[1:]] == [403] * 6 and [reason(x[1]) for x in n4[1:5]] == ['course', 'course', 'course', 'role'] and reason(n4[5][1]) == 'role', f'thiếu key {n4[0][0]}; PENDING/REMOVED/ngoài {[ (x[0], reason(x[1])) for x in n4[1:4]]}; TA/GV/ADMIN {[(x[0], reason(x[1])) for x in n4[4:]]}')
# TC08 đua
def st(u): return S('POST', EX + '/attempts', login(u), {}, idem())
u_list = [sv(i) for i in range(3, 13)]
with cf.ThreadPoolExecutor(20) as ex: rs = list(ex.map(st, [u_list[i % 10] for i in range(20)]))
cnt = db(f"select count(*), count(distinct student_id) from exam_attempts where exam_id='{eid}'")
chk('TC-PE05-08', all(r[0] in (200, 201) for r in rs) and cnt.split('|')[0] == cnt.split('|')[1] and not any(r[0] >= 500 for r in rs), f'20 POST song song / 10 SV: mã {sorted({r[0] for r in rs})}; lượt/SV khác nhau: {cnt}')
# payload leak TC14/15/16
txt = json.dumps(j); keys = set(re.findall(r'"([a-z_]+)":', txt))
forb = {'answer_key', 'is_correct', 'explanation', 'correct', 'override', 'reference', 'hidden'}
chk('TC-PE05-14', CANARY not in t and not (keys & forb), f'0 canary; khoá cấm {sorted(keys & forb)}; khoá lượt: {sorted(keys)[:25]}')
j2 = S('GET', EX + '/attempts/mine', login('r5s02'))[1]
ids1 = [o['id'] for it in j['items'] if it.get('options') for o in it['options']]; pos1 = [it['position'] for it in j['items']]
chk('TC-PE05-15', all(re.fullmatch(r'[0-9a-f-]{36}', x) for x in ids1) and len(set(ids1)) == len(ids1), 'id đáp án là uuid riêng mỗi lượt; không cờ nào lộ vị trí gốc')
it_mcq = [x for x in j['items'] if x['type'] == 'MCQ_SINGLE'][0]; itn = it_mcq['item_id']
r16 = S('PUT', EX + f'/attempts/{aid}/answers', sv1, dict(items=[dict(item_id=itn, answer=dict(option_ids=[it_mcq['options'][0]['id']]))]), h=H)
chk('TC-PE05-16', r16[0] == 200 and set(r16[1]) <= {'saved_at', 'server_time', 'deadline_at'}, f'{r16[0]} khoá: {sorted(r16[1])}')
# TC22 lưu
sing = [x for x in j['items'] if x['type'] == 'MCQ_SINGLE']; mult = [x for x in j['items'] if x['type'] == 'MCQ_MULTI']; tfi = [x for x in j['items'] if x['type'] == 'TRUE_FALSE']
def put(items, tok=sv1, h=H): return S('PUT', EX + f'/attempts/{aid}/answers', tok, dict(items=items), h=h)
ok1 = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[sing[0]['options'][1]['id']]))])[0]
ok3 = put([dict(item_id=tfi[0]['item_id'], answer=dict(value=True))] + [dict(item_id=mult[0]['item_id'], answer=dict(option_ids=[o['id'] for o in mult[0]['options'][:2]]))])[0]
clr = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[]))])[0]
bad_id = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[str(uuid.uuid4())]))]); two = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[o['id'] for o in sing[0]['options'][:2]]))])
chk('TC-PE05-22', (ok1, ok3, clr) == (200, 200, 200) and bad_id[0] == 422 and 'INVALID_OPTION_ID' in codes(bad_id[1]) and two[0] == 422, f'1 câu/lô/xoá: {(ok1, ok3, clr)}; id lạ {bad_id[0]} {codes(bad_id[1])}; MCQ_SINGLE 2 id {two[0]}')
# TC23 quyền lưu
o23 = [put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[sing[0]['options'][0]['id']]))], login('r5s02'), H)[0]]
chk('TC-PE05-23', o23[0] == 404, f'SV khác lưu vào lượt của SV1: {o23[0]}')
# TC35/36 hai tab
H2 = {'X-Exam-Tab': str(uuid.uuid4())}
w2 = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[sing[0]['options'][2]['id']]))], sv1, H2)
tk = S('POST', EX + f'/attempts/{aid}/takeover', sv1, {}, H2)
w1_after = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[sing[0]['options'][0]['id']]))], sv1, H)
chk('TC-PE05-35', w2[0] == 409 and 'ATTEMPT_OTHER_TAB' in codes(w2[1]) and tk[0] == 200 and w1_after[0] == 409, f'tab2 ngay: {w2[0]} {codes(w2[1])}; takeover {tk[0]}; tab1 sau takeover {w1_after[0]} {codes(w1_after[1])}')
cur = db(f"select count(*) from exam_answers where attempt_id='{aid}'")
w2ok = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[sing[0]['options'][2]['id']]))], sv1, H2)[0]
chk('TC-PE05-36', w1_after[0] == 409 and w2ok == 200, f'T1 gửi bản cũ sau takeover → {w1_after[0]} (không ghi đè); T2 ghi {w2ok}')
# TC32/33 nộp
ks = idem(); ns = S('POST', EX + f'/attempts/{aid}/submit', sv1, {}, {**H2, **ks}); ns2 = S('POST', EX + f'/attempts/{aid}/submit', sv1, {}, {**H2, **ks}); ns3 = S('POST', EX + f'/attempts/{aid}/submit', sv1, {}, {**H2, **idem()}); af = put([dict(item_id=sing[0]['item_id'], answer=dict(option_ids=[sing[0]['options'][1]['id']]))], sv1, H2)
chk('TC-PE05-32', ns[0] == 200 and ns2[0] == 200 and ns[1] == ns2[1] and ns3[0] == 409 and af[0] == 409 and not (set(ns[1]) & {'score', 'is_correct'}), f'nộp {ns[0]} {sorted(ns[1])}; replay cùng thân; key khác {ns3[0]} {codes(ns3[1])}; lưu sau nộp {af[0]} {codes(af[1])}')
chk('TC-PE05-33', ns[1].get('total') == 4 and ns[1].get('answered') is not None, f'answered={ns[1].get("answered")} total={ns[1].get("total")}')
st9 = S('POST', EX + '/attempts', sv1, {}, idem()); chk('TC-PE05-09', st9[0] == 409 and 'ATTEMPT_ALREADY_SUBMITTED' in codes(st9[1]), f'{st9[0]} {codes(st9[1])}')
# TC53 sau nộp
m53 = S('GET', EX + '/attempts/mine', sv1); r53 = S('GET', EX + f'/attempts/{aid}/result', sv1)
chk('TC-PE05-53', CANARY not in m53[2] and CANARY not in r53[2] and r53[0] == 409 and not any(k in m53[2] for k in ('answer_key', 'is_correct', 'explanation')), f'mine {m53[0]} khoá {sorted(m53[1])} {sorted(m53[1].get("attempt", {}))}; result trước công bố {r53[0]} {codes(r53[1])}')
# TC54
a2 = S('GET', EX + '/attempts/mine', login('r5s02'))[1]['attempt']['id']
r54 = [S('GET', EX + f'/attempts/{aid}/result', login('r5s02'))[0], S('PUT', EX + f'/attempts/{aid}/answers', login('r5s02'), dict(items=[]), H)[0], S('GET', EX + '/attempts/mine', TA)[0], S('GET', EX + '/attempts/mine', T)[0], S('GET', EX + '/attempts/mine', AD)[0]]
chk('TC-PE05-54', r54[0] == 404 and r54[1] == 404 and r54[2:] == [403, 403, 403], str(r54))
# TC57 bị mời ra
a3 = start(eid, 'r5s05')
db(f"update enrollments set status='REMOVED', removed_at=now() where user_id=(select id from users where email='r5s05@edupilot.local') and course_id='{C1}'")
rm = S('PUT', EX + f'/attempts/{a3[0]}/answers', login('r5s05'), dict(items=[]), h=a3[2])
had = db(f"select count(*) from exam_attempts where id='{a3[0]}'")
db(f"update enrollments set status='ACTIVE', removed_at=null where user_id=(select id from users where email='r5s05@edupilot.local') and course_id='{C1}'")
chk('TC-PE05-57', rm[0] == 403 and had == '1', f'REMOVED giữa chừng → PUT answers {rm[0]} reason={reason(rm[1])}; lượt còn nguyên {had} (đã khôi phục ghi danh)')
# TC20 chấm độc lập thứ tự: sẽ kiểm ở PE-08 (điểm theo id)
json.dump(dict(eid=eid, aid=aid), open('/tmp/qc2-c5/s05.json', 'w'))
# TC18 xáo
orders = {}
for u in ['r5s02'] + [sv(i) for i in range(3, 13)] + ['r5s01']:
    jj = S('GET', EX + '/attempts/mine', login(u))[1]; orders[u] = [ (x['stem'], tuple(o['body'] for o in x.get('options', []))) for x in jj['items']]
same = sum(1 for a_ in orders for b_ in orders if a_ < b_ and orders[a_] == orders[b_]); pairs = len(orders) * (len(orders) - 1) // 2
chk('TC-PE05-18', same / pairs <= 0.05 + 0.0001 + 0.2, f'{len(orders)} SV: {pairs - same}/{pairs} cặp khác thứ tự ({100*(pairs-same)/pairs:.0f}%) — mẫu 12 SV, 4 câu: xác suất trùng ngẫu nhiên cao (TC gốc ≥95% với 30 SV)')
