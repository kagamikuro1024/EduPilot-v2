import os; os.environ['SUITE']='PE-02'
from r5lib import *
def pc(txt): return '#include <stdio.h>\nint main(){ fputs(%s, stdout); return 0; }\n' % json.dumps(txt).replace('\\u00a0','\\xc2\\xa0').replace('\\u2003','\\xe2\\x80\\x83')
def refp(expected): return '#include <cstdio>\nint main(){ fputs(%s, stdout); }\n' % json.dumps(expected)
qa = mkcode(21, tests=[('s1','0\n','hello\n',True,1),('h1','1\n','hello\n',False,1)], checker='EXACT', ref=refp('hello\n'))
qb = mkcode(22, tests=[('s1','0\n','a b c\n',True,1),('h1','1\n','a b c\n',False,1)], checker='TOKENS', ref=refp('a b c\n'))
qc = mkcode(23, tests=[('s1','0\n','1000\n',True,1),('h1','1\n','1000\n',False,1)], checker='FLOAT_EPS', float_eps='0.000001', ref=refp('1000\n'))
# bài có 6 test trọng số [1,1,1,2,0,2] cho TC-20
LOOP = '#include <cstdio>\n#include <cstdlib>\nint main(){long long a,b;scanf("%lld %lld",&a,&b); if(a==3) for(;;); if(a==6) abort(); long long r=a+b; if(a==5) r+=1; printf("%lld\\n",r);}\n'
REF6 = SUM
qd = mkcode(24, tests=[(f't{i}', f'{i} {i+1}\n', f'{2*i+1}\n', i <= 2, w) for i, w in zip(range(1, 7), [1,1,1,2,0,2])], ref=REF6)
eid, items = mkexam(open_s=14, win=3000, dur=45, items=[qa, qb, qc, qd], title='R5 02b checker')
time.sleep(16)
users = ['r5s09', 'r5s10', 'r5s11']
ctxs = {}
def ctx(u):
    if u in ctxs: return ctxs[u]
    aid, j, H, s, _ = start(eid, u); its = {int(r.split('|')[1][-2:]) - 20: r.split('|')[0] for r in db(f"select i.id,q.title from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").split('\n')}
    ctxs[u] = (login(u), f'{E}/{eid}/attempts/{aid}', its, H, aid); return ctxs[u]
def runit(u, pos, lang, src):
    t, A, its, H, aid = ctx(u)
    for _ in range(5):
        s, rr, tt = call('POST', A + f'/code/{its[pos]}/run', t, dict(language=lang, source=src), {**H, **idem()})
        if s == 202: break
        time.sleep(6)
    else: return 'NA:%s' % tt[:80]
    for _ in range(60):
        s, r, tt = call('GET', A + '/runs/' + rr['run_id'], t)
        if r.get('status') in ('DONE', 'ERROR'): break
        time.sleep(1)
    if r.get('compile_ok') is False: return 'CE'
    return ','.join(x['verdict'] for x in r.get('samples', [])) or r.get('status')
t, A, its, H, aid = ctx('r5s09'); print('positions', its)
cases = [('r5s09', 1, 'hello\r\n', 'AC'), ('r5s09', 1, 'hello  \n', 'AC'), ('r5s09', 1, 'hello', 'AC'), ('r5s09', 1, 'hello\n\n\n', 'AC'), ('r5s09', 1, 'Hello\n', 'WA'),
         ('r5s10', 2, 'a   b\tc\n', 'AC'), ('r5s10', 2, 'a\u00a0b\u2003c\n', 'AC'), ('r5s10', 2, 'a b\n', 'WA'),
         ('r5s10', 3, '1000.0009\n', 'AC'), ('r5s10', 3, '1000.002\n', 'WA'), ('r5s10', 3, 'NaN\n', 'WA')]
ok = True; rows = []
for u, pos, out, exp in cases:
    v = runit(u, pos, 'c11', pc(out)); good = (v == exp); ok &= good; rows.append(f'{pos}:{out!r}->{v}({exp})'); print(good, rows[-1])
rec('TC-PE02-17', ok, '; '.join(rows)[:290])
# TC-20: nộp thật
t, A, its, H, aid = ctx('r5s11')
s, rr, tt = call('POST', A + f'/code/{its[4]}/submit', t, dict(language='cpp17', source=LOOP), {**H, **idem()}); sid = rr.get('submission_id'); print('submit', s, tt)
for _ in range(60):
    row = db(f"select status,verdict,passed_weight,total_weight,compile_ok,tests_version,jsonb_array_length(results) from code_submissions where id='{sid}'")
    if row.startswith('DONE') or row.startswith('ERROR'): break
    time.sleep(1)
res = json.loads(db(f"select results::text from code_submissions where id='{sid}'") or '[]')
print(row, [(x.get('position'), x.get('verdict'), x.get('weight')) for x in res])
vs = {x['position']: x['verdict'] for x in res}
ok20 = row.startswith('DONE|TLE|4|7|t|') and len(res) == 6 and vs.get(3) == 'TLE' and vs.get(5) == 'WA' and vs.get(6) == 'RE' and set(res[0]) >= {'test_id','position','is_sample','verdict','time_ms','memory_kb','weight'}
rec('TC-PE02-20', ok20, row + ' ' + str(vs))
# TC-23/24: hàng đợi
rec('TC-PE02-23', 'judge' in rds('xinfo','groups','judge.submit') and db(f"select count(*) from outbox where topic like 'judge%' and payload::text like '%{sid}%'") in ('1',), 'outbox topic/payload: %s; xinfo groups ok' % db(f"select topic,count(*) from outbox where payload::text like '%{sid}%' group by 1"))
before = db(f"select count(*), max(judged_at) from code_submissions where id='{sid}'")
rds('xadd', 'judge.submit', '*', 'submission_id', sid); rds('xadd', 'judge.submit', '*', 'submission_id', sid); time.sleep(8)
after = db(f"select count(*), max(judged_at) from code_submissions where id='{sid}'")
rec('TC-PE02-24', before == after and db(f"select count(*) from code_submissions where id='{sid}'") == '1', f'giao lại tin 2 lần: judged_at không đổi {before} -> {after}')
json.dump(dict(eid=eid, sid=sid), open('/tmp/qc2-c5/s02b.json', 'w'))
