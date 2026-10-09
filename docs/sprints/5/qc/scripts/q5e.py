from q5lib import *
st = json.load(open('/tmp/q3/q5.json')); eid = st['eid']; A = E + '/' + eid
tokN = login('sv.nguyco'); rN = call('GET', A + '/attempts/mine', tokN)[1]; aidN = rN['attempt']['id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aidN}/takeover', tokN, {}, H)
its = [x for x in rN['items'] if x['type'] != 'TRUE_FALSE'][:3]
print('lưu 3 câu', call('PUT', A + f'/attempts/{aidN}/answers', tokN, dict(items=[dict(item_id=x['item_id'], answer=dict(option_ids=[x['options'][0]['id']])) for x in its]), H)[0])
k = idem(); s1 = call('POST', A + f'/attempts/{aidN}/submit', tokN, {}, {**H, **k}); s2 = call('POST', A + f'/attempts/{aidN}/submit', tokN, {}, {**H, **k}); s3 = call('POST', A + f'/attempts/{aidN}/submit', tokN, {}, {**H, **idem()})
print('32 nộp', s1[0], s1[2][:220]); print('32 cùng key', s2[0], 'thân giống:', s1[1].get('submitted_at') == s2[1].get('submitted_at'), '| khoá khác', s3[0], codes(s3[1]))
print('33 answered/total', s1[1].get('answered'), s1[1].get('total'), '| có khoá điểm/đúng sai:', [kk for kk in ['score', 'auto_score', 'correct', 'is_correct'] if '"%s"' % kk in s1[2]])
r = call('PUT', A + f'/attempts/{aidN}/answers', tokN, dict(items=[]), H); print('32 PUT sau nộp', r[0], codes(r[1]))
print('53 mine sau nộp:', json.dumps(call('GET', A + '/attempts/mine', tokN)[1])[:300], '| result trước công bố:', call('GET', A + f'/attempts/{aidN}/result', tokN)[0:1])
print('DB:', db(f"select status,submit_reason,auto_score from exam_attempts where id='{aidN}'"))
