from q5lib import *
import itertools
st = json.load(open('/tmp/q3/q5.json')); eid = st['eid']; A = E + '/' + eid
emails = [r.split('|')[0] for r in db(f"select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{C1}' and e.role_in_course='STUDENT' and e.status='ACTIVE' order by u.email limit 40").split('\n')]
print('SV ACTIVE lớp 1:', len(emails))
orders = {}; opts = {}; atts = {}
for em in emails[:30]:
    tok = login(em.split('@')[0]); r = call('POST', A + '/attempts', tok, {}, idem())
    if r[0] in (200, 201):
        orders[em] = tuple(x['item_id'] for x in r[1]['items']); opts[em] = tuple(tuple(o['id'] for o in x['options']) for x in r[1]['items']); atts[em] = (tok, r[1]['attempt']['id'], r[1])
print('18 lượt tạo được:', len(orders))
pairs = list(itertools.combinations(orders, 2)); diff = sum(1 for a, b in pairs if orders[a] != orders[b]); print('18 cặp SV thứ tự câu khác nhau: %.1f%%' % (100 * diff / len(pairs)))
# cùng SV hai lần
em0 = emails[0]; r2 = call('GET', A + '/attempts/mine', atts[em0][0])[1]; print('18 cùng SV hai lần cùng thứ tự:', tuple(x['item_id'] for x in r2['items']) == orders[em0], tuple(tuple(o['id'] for o in x['options']) for x in r2['items']) == opts[em0])
# mỗi câu/đáp án đúng 1 lần
g = atts[em0][2]; print('18 hoán vị đầy đủ:', len({x['item_id'] for x in g['items']}) == len(g['items']), all(len({o['id'] for o in x['options']}) == len(x['options']) for x in g['items']))
# 32/33 nộp tay (sv.nguyco đã có lượt): làm 4/7 rồi nộp
tokN = login('sv.nguyco'); rN = call('GET', A + '/attempts/mine', tokN)[1]; aidN = rN['attempt']['id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aidN}/takeover', tokN, {}, H)
its = [x for x in rN['items'] if x['type'] != 'TRUE_FALSE'][:3]
call('PUT', A + f'/attempts/{aidN}/answers', tokN, dict(items=[dict(item_id=x['item_id'], answer=dict(option_ids=[x['options'][0]['id']])) for x in its]), H)
k = idem(); s1 = call('POST', A + f'/attempts/{aidN}/submit', tokN, {}, k); s2 = call('POST', A + f'/attempts/{aidN}/submit', tokN, {}, k); s3 = call('POST', A + f'/attempts/{aidN}/submit', tokN, {}, idem())
print('32 nộp', s1[0], s1[2][:200]); print('32 cùng key', s2[0], s1[2] == s2[2].replace(s2[2], s1[2]) and 'thân giống' if s1[1].get('answered') == s2[1].get('answered') else 'khác', ' khoá khác', s3[0], codes(s3[1]))
print('33 answered/total', s1[1].get('answered'), s1[1].get('total'), '| không có điểm:', not any(kk in s1[2] for kk in ['score', 'correct']))
print('32 PUT answers sau nộp', call('PUT', A + f'/attempts/{aidN}/answers', tokN, dict(items=[]), H)[0:1], codes(call('PUT', A + f'/attempts/{aidN}/answers', tokN, dict(items=[]), H)[1]))
print('53 mine sau nộp:', json.dumps(call('GET', A + '/attempts/mine', tokN)[1])[:300], '| result trước công bố:', call('GET', A + f'/attempts/{aidN}/result', tokN)[0:1])
print('DB lượt đã chấm:', db(f"select status,submit_reason,auto_score from exam_attempts where id='{aidN}'"))
# 57 bị mời ra giữa giờ
em = emails[5]; tok, aid, _ = atts[em]; uid = db(f"select id from users where email='{em}'")
rm = call('DELETE', f'/courses/{C1}/members/{uid}', T); H5 = {'X-Exam-Tab': str(uuid.uuid4())}
print('57 GV mời ra', rm[0], '| SV lưu:', call('PUT', A + f'/attempts/{aid}/answers', tok, dict(items=[]), H5)[0], '| mine:', call('GET', A + '/attempts/mine', tok)[0], '| lượt còn trong DB:', db(f"select count(*) from exam_attempts where id='{aid}'"))
call('POST', f'/courses/{C1}/members/{uid}/undo', T, {})
# 03 không mở
d = mke(opens_at=now(days=30), closes_at=now(days=30, hours=1))[1]['id']
print('03 bài DRAFT', call('POST', E + '/' + d + '/attempts', SV, {}, idem())[0:1], codes(call('POST', E + '/' + d + '/attempts', SV, {}, idem())[1]), call('POST', E + '/' + d + '/attempts', SV, {}, idem())[1].get('details'))
q2 = [mkq5(70 + i) for i in range(2)]; call('PUT', E + '/' + d + '/items', T, dict(items=[dict(question_id=q, points='1') for q in q2], version=ge(d)['version'])); call('POST', E + '/' + d + '/schedule', T, {})
r = call('POST', E + '/' + d + '/attempts', SV, {}, idem()); print('03 SCHEDULED chưa tới giờ', r[0], codes(r[1]), r[1].get('details'))
# 56 PENDING/ngoài lớp
SVD = login('sv.moi'); print('04 sv.moi (không thuộc lớp 1)', call('POST', A + '/attempts', SVD, {}, idem())[0:1], 'chưa vào lớp' )
