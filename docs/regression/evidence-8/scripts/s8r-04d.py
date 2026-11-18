import sys, re, json, copy; sys.path.insert(0, '.')
from s8r_chat import *
c1 = json.load(open('/tmp/qc2-s8r/c1.json')); C = c1['c0']; Q = f'/courses/{C}'
def cap_of(ev): return [e[1]['data']['bonus']['cap'] for e in ev if e[0] == 'block' and e[1].get('kind') == 'grade_scheme']
_, a0, ev0 = ask('sv.kha', C, 'Cách tính điểm môn này thế nào?'); time.sleep(8)
s, sc = L.api('gv.s8f', 'GET', Q + '/grade-scheme'); d = sc['draft']
b = {"version": d["version"], "components": [{k: x[k] for k in ("key", "name", "grp", "weight")} | {"items": x["items"]} for x in d["components"]], "rules": [{"kind": r["kind"], "applicable": r["applicable"], "params": copy.deepcopy(r["params"])} for r in d["rules"]]}
for x in b['components']:
    if x['key'] in ('midterm','weekly'): x['weight'] = {'midterm': '16', 'weekly': '24'}[x['key']]
sp, dp = L.api('gv.s8f', 'PUT', Q + '/grade-scheme', b); print('put', sp, [u['code'] for u in dp.get('unclear_items', [])] if isinstance(dp, dict) else dp)
sc2, rc = L.api('gv.s8f', 'POST', Q + '/grade-scheme/confirm', {'version': dp['version']}) if sp == 200 else (0, {})
print('confirm', sc2, str(rc)[:200])
_, a1, ev1 = ask('sv.kha', C, 'Cách tính điểm môn này thế nào?')
c0, c1_ = cap_of(ev0), cap_of(ev1)
T('TC-P6-09-33', 'P' if sc2 == 200 and c0 == ['1'] and c1_ == ['1.5'] and re.search(r'1[,.]5', a1) else 'F', f'QD8C1: trước xác nhận block cap {c0} (câu: «{re.findall(r"tối đa[^.-]*", a0)[:1]}»); PUT nháp {sp}, confirm {sc2}; hỏi lại → block cap {c1_} (câu: «{re.findall(r"tối đa[^.-]*", a1)[:1]}»); không dùng số cũ')
