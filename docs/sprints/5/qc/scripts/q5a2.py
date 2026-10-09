from q5lib import *
import threading, re
TAB1, TAB2 = str(uuid.uuid4()), str(uuid.uuid4())
st = json.load(open('/tmp/q3/q5.json')); eid, ids = st['eid'], st['ids']; A = E + '/' + eid
st2=json.load(open('/tmp/q3/q5.json')); A=E+'/'+eid
j=call('GET',A+'/attempts/mine',SV)[1]; aid=j['attempt']['id']; SVB2=login('sv.kha')
# 22 lưu
H = {'X-Exam-Tab': TAB1}
mc = [x for x in j['items'] if x['type'] == 'MCQ_SINGLE']; mm = [x for x in j['items'] if x['type'] == 'MCQ_MULTI']; tf = [x for x in j['items'] if x['type'] == 'TRUE_FALSE']
def conv(a): return dict(item_id=a['item_id'], answer={k: v for k, v in a.items() if k != 'item_id'})
def put(ans, tok=SV, h=H): return call('PUT', A + f'/attempts/{aid}/answers', tok, dict(items=[conv(a) for a in ans]), h)
s, r, t = put([dict(item_id=mc[0]['item_id'], option_ids=[mc[0]['options'][0]['id']])]); print('22 lưu 1 câu', s, t[:160])
s, r, t = put([dict(item_id=mc[1]['item_id'], option_ids=[mc[1]['options'][1]['id']]), dict(item_id=mm[0]['item_id'], option_ids=[o['id'] for o in mm[0]['options'][:2]]), dict(item_id=tf[0]['item_id'], value=True)]); print('22 lô 3 câu', s, codes(r))
print('22 id lạ', put([dict(item_id=mc[0]['item_id'], option_ids=[str(uuid.uuid4())])])[0:1], 'SINGLE 2 id', put([dict(item_id=mc[0]['item_id'], option_ids=[o['id'] for o in mc[0]['options'][:2]])])[0:1], 'rỗng (xoá)', put([dict(item_id=mc[0]['item_id'], option_ids=[])])[0:1])
print('22 DB exam_answers:', db(f"select count(*) from exam_answers where attempt_id='{aid}'"))
print('16 phản hồi lưu:', put([dict(item_id=mc[0]['item_id'], option_ids=[mc[0]['options'][2]['id']])])[2][:200])
print('23 SVB lưu vào lượt của SVA', call('PUT', A + f'/attempts/{aid}/answers', SVB2, dict(items=[]), H)[0])
# 35/36 một nơi ghi
s1 = put([dict(item_id=mc[0]['item_id'], option_ids=[mc[0]['options'][3]['id']])], SV, {'X-Exam-Tab': TAB1})
s2 = put([dict(item_id=mc[0]['item_id'], option_ids=[mc[0]['options'][0]['id']])], SV, {'X-Exam-Tab': TAB2}); print('35 T1', s1[0], 'T2 ngay sau', s2[0], codes(s2[1]))
tk = call('POST', A + f'/attempts/{aid}/takeover', SV, {}, {'X-Exam-Tab': TAB2}); print('35 takeover(T2)', tk[0])
s3 = put([dict(item_id=mc[0]['item_id'], option_ids=[mc[0]['options'][0]['id']])], SV, {'X-Exam-Tab': TAB2}); s4 = put([dict(item_id=mc[0]['item_id'], option_ids=[mc[0]['options'][3]['id']])], SV, {'X-Exam-Tab': TAB1}); print('36 T2 ghi', s3[0], 'T1 cũ ghi', s4[0], codes(s4[1]))
cur = db(f"select answer from exam_answers where attempt_id='{aid}' and item_id='{mc[0]['item_id']}'"); print('36 đáp án còn:', cur[:120], '(của T2 = option 0:', mc[0]['options'][0]['id'] in cur, ')')
# 59 rò trước công bố trên mọi GET của SV
for p in ['', '/attempts/mine', f'/attempts/{aid}/result']:
    s, r, t = call('GET', A + p, SV); print('59', p or '(exam)', s, 'canary:', CANARY in t, 'khoá cấm:', [kk for kk in ['answer_key', 'is_correct', 'explanation', 'correct', 'override', 'reference'] if '"%s"' % kk in t])
print('54 SVB đọc lượt của SVA', call('GET', A + f'/attempts/{aid}/result', SVB2)[0], 'TA/GV', call('GET', A + '/attempts/mine', TA)[0], call('GET', A + '/attempts/mine', T)[0], 'ADMIN', call('GET', A + '/attempts/mine', AD)[0])
