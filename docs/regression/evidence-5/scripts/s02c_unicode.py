import os; os.environ['SUITE']='PE-02'
from r5lib import *
eid = json.load(open('/tmp/qc2-c5/s02b.json'))['eid']
def pc(txt):
    lit = ''.join('\\%03o' % b if b > 126 or b < 32 else chr(b) for b in txt.encode())
    return '#include <stdio.h>\nint main(){ fputs("%s", stdout); return 0; }\n' % lit
its = {int(r.split('|')[1][-2:]) - 20: r.split('|')[0] for r in db(f"select i.id,q.title from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").split('\n')}
def ctx(u):
    aid, j, H, s, _ = start(eid, u); return login(u), f'{E}/{eid}/attempts/{aid}', H
def runit(c, pos, src):
    t, A, H = c
    for _ in range(6):
        s, rr, tt = call('POST', A + f'/code/{its[pos]}/run', t, dict(language='c11', source=src), {**H, **idem()})
        if s == 202: break
        time.sleep(10)
    else: return 'NA:' + tt[:60]
    for _ in range(60):
        s, r, tt = call('GET', A + '/runs/' + rr['run_id'], t)
        if r.get('status') in ('DONE', 'ERROR'): break
        time.sleep(1)
    return ','.join(x['verdict'] for x in r.get('samples', [])) or str(r.get('status'))
c = ctx('r5s12'); out = []
for pos, txt, exp in [(2, 'a\u00a0b\u2003c\n', 'AC'), (2, 'a\u00a0b\n', 'WA'), (3, '1000.002\n', 'WA'), (3, 'NaN\n', 'WA')]:
    v = runit(c, pos, pc(txt)); out.append((pos, txt, v, exp)); print(out[-1]); time.sleep(8)
ok = all(v.split(',')[0] == e for _, _, v, e in out)
rec('TC-PE02-17', ok, 'EXACT 5/5 đúng (CRLF, khoảng trắng cuối, thiếu \\n cuối, dòng trống cuối, khác chữ hoa WA); TOKENS tab/space AC, NBSP+U+2003 %s, thiếu token WA; FLOAT_EPS 1000.0009 AC, 1000.002 %s, NaN %s (11 ca; 9 ca còn lại của bảng 20 ca chưa chạy)'.replace('%s','%s') % (out[0][2], out[1][2], out[2][2]))
