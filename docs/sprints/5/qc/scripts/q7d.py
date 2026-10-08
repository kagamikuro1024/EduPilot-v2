from q6lib import *
st = json.load(open('/tmp/q3/q7.json')); eid = st['eid']; A = E + '/' + eid
s, j, _ = call('GET', A + '/similarity?limit=2', T); print('list limit=2', s, len(j['items']), 'next_cursor' in j, {k: v for k, v in j.items() if k != 'items'})
cur = j.get('next_cursor'); s2, j2, _ = call('GET', A + f'/similarity?limit=2&cursor={cur}', T); print('cursor trang 2', s2, [i['score'] for i in j2['items']], '| trùng id với trang 1:', {i['id'] for i in j['items']} & {i['id'] for i in j2['items']})
print('flagged=true', len(call('GET', A + '/similarity?flagged=true', T)[1]['items']), '| TA', call('GET', A + '/similarity', TA)[0], 'SV', call('GET', A + '/similarity', SV)[0], 'ADMIN', call('GET', A + '/similarity', AD)[0])
sid = j['items'][0]['id']; r = call('GET', A + f'/similarity/{sid}', T); print('#57 chi tiết', r[0], list(r[1].keys()), '| TA', call('GET', A + f'/similarity/{sid}', TA)[0], 'SV', call('GET', A + f'/similarity/{sid}', SV)[0]); d = r[1]
print('   khoá a:', {k: (v if k != 'source' else '...%d byte' % len(v)) for k, v in (d.get('a') or {}).items()} if isinstance(d.get('a'), dict) else d.get('a'))
R = A + f'/similarity/{sid}/review'
print('39 ghi chú 501', call('PUT', R, T, dict(state='FOLLOW_UP', note='x' * 501))[0:1], codes(call('PUT', R, T, dict(state='FOLLOW_UP', note='x' * 501))[1]), '| state lạ', call('PUT', R, T, dict(state='BAN'))[0], '| TA', call('PUT', R, TA, dict(state='CLEARED'))[0], 'SV', call('PUT', R, SV, dict(state='CLEARED'))[0])
print('39 FOLLOW_UP + ghi chú', call('PUT', R, T, dict(state='FOLLOW_UP', note='Cần trao đổi'))[0:1]); print('39 CLEARED', call('PUT', R, T, dict(state='CLEARED'))[0:1]); print('   DB', db(f"select review_state,note,reviewed_by is not null from similarity_reports where id='{sid}'"))
# Hôm nay: không có cặp flagged ⇒ không có việc
for nm, tok in [('GV', T), ('TA', TA)]:
    s, j, t = call('GET', '/me/today', tok); print('40 today', nm, s, 'EXAM_SIMILARITY' in t)
# chạy lại
r = call('POST', A + '/similarity/run', T, {}, idem()); print('34 chạy lại', r[0], r[2][:100]); time.sleep(8); print('34 số run_id', db(f"select count(distinct run_id) from similarity_reports where exam_id='{eid}'"), '| TA run', call('POST', A + '/similarity/run', TA, {}, idem())[0], 'SV', call('POST', A + '/similarity/run', SV, {}, idem())[0])
# bài MCQ không có code → 422
em = db(f"select id from exams where title like 'QC7 M5' limit 1"); print('34 bài không code', call('POST', E + '/' + em + '/similarity/run', T, {}, idem())[0:1] if em else '-')
# 23/42 DB
print('23 cột điểm trừ:', db("select table_name||'.'||column_name from information_schema.columns where table_name in ('exam_attempts','exam_answers','exam_events','similarity_reports') and (column_name ilike '%penal%' or column_name ilike '%cheat%' or column_name ilike '%deduct%' or column_name ilike '%violation%')") or '0')
