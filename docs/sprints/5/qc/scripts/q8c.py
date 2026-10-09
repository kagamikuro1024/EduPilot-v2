from q6lib import *
st = json.load(open('/tmp/q3/q8.json')); plan = json.load(open('/tmp/q3/q8plan.json')); eid = st['eid']; A = E + '/' + eid
def keys(o, acc=None):
    acc = set() if acc is None else acc
    if isinstance(o, dict):
        for k, v in o.items(): acc.add(k); keys(v, acc)
    elif isinstance(o, list): [keys(x, acc) for x in o]
    return acc
tok = login('sv04'); aid = plan['sv04']['aid']
s, j, t = call('GET', A + f'/attempts/{aid}/result', tok); print('25', s, 'score', j.get('score'), 'adj', j.get('score_adjusted'), '| khoá:', sorted(keys(j)))
print('   canary giải thích (reveal bật):', CANARY in t, '| QC-HID:', HID in t, 'QC-REF:', REF in t)
print('   code item:', json.dumps([x for x in j.get('items', []) if x.get('type') == 'CODE'][0], ensure_ascii=False)[:500])
print('28 hidden', [x['code'].get('hidden') if x.get('code') else None for x in j.get('items', [])])
aidB = plan['sv05']['aid']; print('54 SV khác đọc lượt người ta', call('GET', A + f'/attempts/{aidB}/result', tok)[0], '| TA', call('GET', A + f'/attempts/{aid}/result', TA)[0], 'GV', call('GET', A + f'/attempts/{aid}/result', T)[0], 'AD', call('GET', A + f'/attempts/{aid}/result', AD)[0])
print('34 SV vắng (sv20) result', call('GET', A + f'/attempts/{uuid.uuid4()}/result', login('sv20'))[0:1])
s, j, t = call('GET', A + '/results?limit=5', T); print('36', s, {k: v for k, v in j.items() if k != 'items'}, '| item0', json.dumps(j['items'][0], ensure_ascii=False)[:400]); print('   flags GV:', [i.get('flags') for i in j['items']][:2])
s, jt, tt = call('GET', A + '/results?limit=5', TA); print('36 TA', s, 'flags' in tt, 'integrity' in tt)
s, j, t = call('GET', A + '/results?sort=score&limit=100', T); print('   sort=score', [i.get('score') for i in j['items']][:13])
print('   q=sv04', len(call('GET', A + '/results?q=sv04', T)[1]['items']), '| status=GRADED', len(call('GET', A + '/results?status=GRADED&limit=100', T)[1]['items']))
s, js, t = call('GET', A + '/stats', T); print('40 stats', s, json.dumps(js, ensure_ascii=False)[:900])
sc = sorted(float(plan[u]['exp']) for u in plan); import statistics; print('   QC mean %.4f median %.4f' % (statistics.mean(sc), statistics.median(sc)), 'phân bố (khoảng 1 điểm):', {k: sum(1 for x in sc if k <= x < k + 1) for k in range(0, 11) if any(k <= x < k + 1 for x in sc)})
r = call('GET', A + '/results.csv', T); print('42 CSV', r[0]); 
import ssl, urllib.request
req = urllib.request.Request(BASE + E + '/' + eid + '/results.csv', headers={'Authorization': 'Bearer ' + T}); raw = urllib.request.urlopen(req, context=CTX).read(); print('   3 byte đầu', raw[:3].hex(), '| dấu ;', raw.split(b'\n')[1][:120], '| dòng', raw.count(b'\n'))
print('   CSV TA', call('GET', A + '/results.csv', TA)[0], 'SV', call('GET', A + '/results.csv', tok)[0], 'AD', call('GET', A + '/results.csv', AD)[0])
print('   header:', raw.decode('utf-8-sig').split('\n')[0][:200])
