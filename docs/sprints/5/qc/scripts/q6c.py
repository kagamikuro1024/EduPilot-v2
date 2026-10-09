from q6lib import *
st = json.load(open('/tmp/q3/q6.json')); eid = st['eid']; A = E + '/' + eid
def setup(user):
    tok = login(user); j = call('POST', A + '/attempts', tok, {}, idem())[1]; aid = j['attempt']['id']; iid = [x for x in j['items'] if x['type'] == 'CODE'][0]['item_id']; H = {'X-Exam-Tab': str(uuid.uuid4())}; call('POST', A + f'/attempts/{aid}/takeover', tok, {}, H); return tok, aid, iid, H
# 17 rate
tok, aid, iid, H = setup('sv.kha'); CB = A + f'/attempts/{aid}/code/{iid}'
res = []; t0 = time.time()
for i in range(11):
    s, j, t = call('POST', CB + '/run', tok, dict(language='cpp17', source=SUMSRC), {**H, **idem()}); res.append((s, j.get('details'), j.get('code')))
print('17 mã:', [r[0] for r in res], '| lần 11:', res[10])
# lỗi đầu vào (không tính vào hạn mức vì 429 đang bị chặn → dùng SV khác)
tok2, aid2, iid2, H2 = setup('sv.nguyco'); CB2 = A + f'/attempts/{aid2}/code/{iid2}'
def r2(body, h=None, t=tok2, cb=CB2): return call('POST', cb + '/run', t, body, h if h is not None else {**H2, **idem()})
print('18 thiếu key', r2(dict(language='cpp17', source=SUMSRC), H2)[0:1], codes(r2(dict(language='cpp17', source=SUMSRC), H2)[1]))
print('18 rỗng', r2(dict(language='cpp17', source=''))[0:1], codes(r2(dict(language='cpp17', source=''))[1]), '| 64KiB+1', codes(r2(dict(language='cpp17', source='a' * 65537))[1]), '| lạ', codes(r2(dict(language='java', source='x'))[1]))
print('18 SV khác vào lượt người ta', call('POST', CB2 + '/run', tok, dict(language='cpp17', source=SUMSRC), {**H, **idem()})[0])
# 24..26 submit
tok3, aid3, iid3, H3 = setup('sv.gioi') if False else (SV, None, None, None)
