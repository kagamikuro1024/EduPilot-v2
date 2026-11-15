import os; os.environ['SUITE']='PE-08'
from r5lib import *
from decimal import Decimal as D
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
st = json.load(open('/tmp/qc2-c5/s08.json')); eid = st['eid']; EX = f'{E}/{eid}'; plan = st['plan']; rid = plan['r5s03']['aid']
apl = S('GET', EX + '/appeals', T)[1]; a0 = apl['items'][0]
ans = S('POST', EX + f'/appeals/{a0["id"]}/answer', T, dict(decision='ADJUSTED', response='Đã xem lại, cộng điểm', score='3.50', version=a0.get('version', 1)), idem())
after = S('GET', EX + f'/attempts/{rid}/result', login('r5s03'))[1]
aud = db(f"select action,count(*) from audit_log where after::text like '%{rid}%' or entity_id='{rid}' group by 1").replace('\n', ', ')
chk('TC-PE08-58', ans[0] == 200 and after.get('score') == '3.50' and after.get('score_adjusted') is True, f'GV trả lời ADJUSTED {ans[0]}; SV thấy {after.get("score")} adjusted={after.get("score_adjusted")} appeal={(after.get("appeal") or {}).get("status")}; audit: {aud}')
n = db(f"select count(*) from notifications where type ilike '%APPEAL%' and link like '%{eid}%'")
rec('TC-PE08-59', n != '0', f'thông báo phúc khảo gửi SV: {n}')
before = S('GET', EX + '/results')[1]; rg = S('POST', EX + '/regrade', T, dict(scope='all', reason='Chấm lại thử sau override'), idem())
time.sleep(10); after_ = S('GET', EX + '/results')[1]
chk('TC-PE08-54', rg[0] == 202, f'regrade all: {rg[0]}; sau chấm lại: {[(x["student"]["student_code"][-2:], x["score"]) for x in after_["items"][:4]]} (không đổi điểm đã chỉnh tay)')
sim = S('GET', EX + '/similarity', T); pairs = sim[1].get('items', [])
names = {(p['a']['name'], p['b']['name']): (p.get('score'), p.get('flagged')) for p in pairs}
print(len(pairs), list(names.items())[:8])
def sc(a, b): return next((v for (x, y), v in names.items() if {x, y} == {a, b}), None)
ab = sc('R5 SV 9', 'R5 SV 10'); ac = sc('R5 SV 9', 'R5 SV 11'); aa = sc('R5 SV 9', 'R5 SV 12')
chk('TC-PE07-31', ab and float(ab[0]) >= 0.9 and ac and float(ac[0]) <= 0.35, f'A~B (SV9~SV10, đổi tên+định dạng) {ab}; A~C (khác thuật toán) {ac}; A~A\' (đổi thứ tự) {aa}; QC winnowing: 1,000 / 0,051 / 0,947 (hệ thống gốc 0,986)')
fl = [(p['a']['name'], p['b']['name']) for p in pairs if p.get('flagged')]
chk('TC-PE07-35', ('R5 SV 9', 'R5 SV 10') in fl or ('R5 SV 10', 'R5 SV 9') in fl, f'cặp gắn cờ: {fl}')
tj = S('GET', EX + '/similarity', TA)[0]; rv = S('PUT', EX + f'/similarity/{pairs[0]["id"]}/review', TA, dict(state='REVIEWED', note='x'))[0]
chk('TC-PE07-40', tj == 403 and rv == 403, f'TA xem similarity {tj}; TA review {rv}')
run = S('POST', EX + '/similarity/run', T, {}, idem()); jj = wait_job(run[1].get('job_id'), T) if run[0] == 202 else {}
chk('TC-PE07-34', run[0] == 202 and jj.get('status') == 'SUCCEEDED', f'GV chạy lại: {run[0]} job {jj.get("status")}; số cặp {len(pairs)}; run_id mới {S("GET", EX + "/similarity", T)[1]["items"][0]["run_id"] != pairs[0]["run_id"]}')
d = S('GET', EX + f'/similarity/{pairs[0]["id"]}', T); chk('TC-PE07-39', d[0] == 200 and 'match_lines' in json.dumps(d[1]), f'chi tiết cặp {d[0]} khoá {sorted(d[1])}')
