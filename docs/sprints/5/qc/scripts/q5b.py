from q5lib import *
st = json.load(open('/tmp/q3/q5.json')); eid = st['eid']; A = E + '/' + eid
j = call('GET', A + '/attempts/mine', SV)[1]; aid = j['attempt']['id']; TAB = str(uuid.uuid4()); H = {'X-Exam-Tab': TAB}
print('takeover', call('POST', A + f'/attempts/{aid}/takeover', SV, {}, H)[0])
mc = [x for x in j['items'] if x['type'] == 'MCQ_SINGLE']; mm = [x for x in j['items'] if x['type'] == 'MCQ_MULTI']; tf = [x for x in j['items'] if x['type'] == 'TRUE_FALSE']
def put(items, h=H, body=None, tok=SV): return call('PUT', A + f'/attempts/{aid}/answers', tok, body or dict(items=items), h)
a = lambda it, **k: dict(item_id=it['item_id'], answer=k)
r = put([a(mc[0], option_ids=[mc[0]['options'][0]['id']])]); print('22 lưu 1 câu', r[0], r[2][:120])
r = put([a(mc[1], option_ids=[mc[1]['options'][1]['id']]), a(mm[0], option_ids=[o['id'] for o in mm[0]['options'][:2]]), a(tf[0], value=True)]); print('22 lô 3 câu', r[0])
print('22 DB exam_answers (mong 4):', db(f"select count(*) from exam_answers where attempt_id='{aid}'"))
print('22 id lạ', put([a(mc[0], option_ids=[str(uuid.uuid4())])])[0:1], codes(put([a(mc[0], option_ids=[str(uuid.uuid4())])])[1]), '| SINGLE 2 id', put([a(mc[0], option_ids=[o['id'] for o in mc[0]['options'][:2]])])[0:1], '| rỗng', put([a(mc[0], option_ids=[])])[0:1])
print('22 DB sau xoá (mong 3):', db(f"select count(*) from exam_answers where attempt_id='{aid}'"))
big = 'x' * 70000; r = put([], body=dict(items=[a(tf[0], value=True)], pad=big)); print('22 thân > 64 KiB', r[0], codes(r[1]))
r = put([a(tf[0], value=True)]); print('16 phản hồi lưu:', r[2][:200])
print('22 item không thuộc bài', put([dict(item_id=str(uuid.uuid4()), answer=dict(value=True))])[0:1])
# 24 rate: 241 PUT / phút
t0 = time.time(); codes_ = []
for i in range(245):
    codes_.append(put([a(tf[0], value=bool(i % 2))])[0])
print('24 245 PUT trong %.0fs:' % (time.time() - t0), {c: codes_.count(c) for c in set(codes_)}, 'lần 429 đầu tiên ở yêu cầu số', (codes_.index(429) + 1) if 429 in codes_ else None)
