from q6lib import *
import urllib.request
st = json.load(open('/tmp/q3/q8.json')); plan = json.load(open('/tmp/q3/q8plan.json')); eid = st['eid']; A = E + '/' + eid
CAN = db("select explanation from question_bank where title='qc8-1'").split(' giải')[0]; print('canary đề:', CAN[:12])
tok = login('sv04'); aid = plan['sv04']['aid']; s, j, t = call('GET', A + f'/attempts/{aid}/result', tok); mc = [x for x in j['items'] if x['type'].startswith('MCQ')]
print('26 reveal bật: canary giải thích có (đối chứng dương):', CAN in t, '| đáp án đúng có:', any(x.get('correct') for x in mc), '| mine:', mc[0].get('mine'))
s, d, t = call('GET', A + f'/results/{aid}', T); print('detail GV', s, sorted(d.keys()), '| integrity' , 'integrity' in t); print('detail TA', 'integrity' in call('GET', A + f'/results/{aid}', TA)[2])
ver = d.get('version') or d['attempt'].get('version') if isinstance(d.get('attempt'), dict) else d.get('version'); print('version', ver)
def adj(sc, reason='QC chỉnh', v=None, tk=T): return call('PUT', A + f'/results/{aid}/score', tk, dict(score=sc, reason=reason, version=v if v is not None else ver))
print('46 ngoài khoảng', adj('10.01')[0], adj('-1')[0], '| sai bước', adj('8.505')[0], '| thiếu lý do', call('PUT', A + f'/results/{aid}/score', T, dict(score='8.5', version=ver))[0], '| 501 ký tự', adj('8.5', 'x' * 501)[0], '| sai version', adj('8.5', v=99)[0], codes(adj('8.5', v=99)[1]), '| TA', adj('8.5', tk=TA)[0])
r = adj('8.5'); print('46 ok', r[0], r[2][:260]); print('   DB', db(f"select auto_score, adjusted_score, adjusted_reason, version from exam_attempts where id='{aid}'"))
s, j2, _ = call('GET', A + f'/attempts/{aid}/result', tok); print('47 SV thấy', j2.get('score'), j2.get('score_adjusted'), '| EXAM_REGRADED', db(f"select count(*) from notifications where kind='EXAM_REGRADED'"))
adj('8.5', v=(r[1].get('version') if isinstance(r[1], dict) else None) or ver + 1); 
print('46 gỡ null', call('PUT', A + f'/results/{aid}/score', T, dict(score=None, reason='QC gỡ', version=(call('GET', A + f'/results/{aid}', T)[1].get('version'))))[0:1], db(f"select auto_score, adjusted_score from exam_attempts where id='{aid}'"))
# 43 CSV injection: đổi họ tên
for u, nm in [('sv09', '=HYPERLINK("http://x")'), ('sv10', '+84 123'), ('sv11', '-1'), ('sv12', '@SUM(1)')]: db(f"update users set full_name=$q${nm}$q$ where email='{u}@edupilot.local'")
raw = urllib.request.urlopen(urllib.request.Request(BASE + E + '/' + eid + '/results.csv', headers={'Authorization': 'Bearer ' + T}), context=CTX).read().decode('utf-8-sig')
print('43 dòng chứa công thức:', [l[:70] for l in raw.split('\n') if any(k in l for k in ('HYPERLINK', '+84', ';-1', 'SUM'))])
for u in ('sv09', 'sv10', 'sv11', 'sv12'): db(f"update users set full_name=(select full_name from users where email='sv04@edupilot.local') where false")
# 49 override void câu 0
its = {r.split('|')[1]: r.split('|')[0] for r in db(f"select i.id, q.title from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").split('\n')}
i0, i1, ic = its['qc8-0'], its['qc8-1'], [v for k, v in its.items() if k.startswith('qc8-') and k not in ('qc8-0', 'qc8-1', 'qc8-2')]
print('49 override mã (CODE) 422:', call('PUT', A + f'/items/{its["qc8-8"]}/override', T, dict(void=True, reason='x'))[0] if 'qc8-8' in its else '-', '| TA', call('PUT', A + f'/items/{i0}/override', TA, dict(void=True, reason='x'))[0])
before = {r.split('|')[0]: r.split('|')[1] for r in db(f"select u.email, a.auto_score from exam_attempts a join users u on u.id=a.student_id where a.exam_id='{eid}'").split('\n')}
r = call('PUT', A + f'/items/{i0}/override', T, dict(void=True, reason='QC: câu lỗi')); print('50 void', r[0], r[2][:150]); time.sleep(3)
after = {r.split('|')[0]: r.split('|')[1] for r in db(f"select u.email, a.auto_score from exam_attempts a join users u on u.id=a.student_id where a.exam_id='{eid}'").split('\n')}
# mọi lượt: q0 → trọn 1 điểm (void)
exp = {}
for u, p in plan.items(): q0 = float(p['parts'][0]); exp[u + '@edupilot.local'] = float(p['exp']) + (1 - q0)
bad = [(k, after[k], exp[k]) for k in after if abs(float(after[k]) - exp[k]) > 1e-9 and k != 'sv04@edupilot.local']; print('13/50 sau void: lệch', bad, '| mẫu: sv05', before['sv05@edupilot.local'], '→', after['sv05@edupilot.local'], 'kỳ vọng', exp['sv05@edupilot.local'])
print('50 SV sau void: override lộ?', json.dumps(call('GET', A + f'/attempts/{plan["sv05"]["aid"]}/result', login('sv05'))[1])[:0], [ (x.get('overridden'), x.get('earned'), x.get('max')) for x in call('GET', A + f'/attempts/{plan["sv05"]["aid"]}/result', login('sv05'))[1]['items']][:3])
