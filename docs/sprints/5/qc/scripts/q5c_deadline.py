from q5lib import *
st = json.load(open('/tmp/q3/q5.json')); eid = st['eid']; A = E + '/' + eid
j = call('GET', A + '/attempts/mine', SV)[1]; aid = j['attempt']['id']; TAB = str(uuid.uuid4()); H = {'X-Exam-Tab': TAB}
call('POST', A + f'/attempts/{aid}/takeover', SV, {}, H)
dl = dt.datetime.strptime(j['attempt']['deadline_at'][:26], '%Y-%m-%dT%H:%M:%S.%f').replace(tzinfo=dt.timezone.utc)
# thứ tự theo item_id → chỉ số câu = số trong tiêu đề: lấy từ DB
rows = db(f"select i.id, q.title, q.type from exam_items i join question_bank q on q.id=i.question_id where i.exam_id='{eid}'").split('\n'); idx = {r.split('|')[0]: (r.split('|')[1], r.split('|')[2]) for r in rows}
def opt(it, k): return [o for o in it['options'] if o['body'].startswith('Đáp án %d ' % k)][0]['id']
ans = []; exp = 0
for it in j['items']:
    title, typ = idx[it['item_id']]; n = int(title.split('-')[1])
    if typ == 'TRUE_FALSE': ans.append(dict(item_id=it['item_id'], answer=dict(value=True))); exp += 1
    elif n == 1 or n == 2: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[opt(it, 1)]))); exp += 1
    elif n == 4: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[opt(it, 2)])))
    elif n == 0: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[opt(it, 0), opt(it, 2)]))); exp += 1
    elif n == 3: ans.append(dict(item_id=it['item_id'], answer=dict(option_ids=[opt(it, 0), opt(it, 1)])))
print('đặt đáp án:', call('PUT', A + f'/attempts/{aid}/answers', SV, dict(items=ans), H)[0], 'kỳ vọng earned', exp, '/ 7 → điểm', round(exp / 7 * 10, 2))
def now(): return dt.datetime.now(dt.timezone.utc)
while now() < dl + dt.timedelta(seconds=9): time.sleep(0.2)
r9 = call('PUT', A + f'/attempts/{aid}/answers', SV, dict(items=[ans[0]]), H); t9 = (now() - dl).total_seconds()
while now() < dl + dt.timedelta(seconds=11): time.sleep(0.2)
r11 = call('PUT', A + f'/attempts/{aid}/answers', SV, dict(items=[ans[0]]), H); t11 = (now() - dl).total_seconds()
print('28 lưu tại deadline+%.1fs → %s ; deadline+%.1fs → %s %s' % (t9, r9[0], t11, r11[0], codes(r11[1])))
t_sub = None
for _ in range(60):
    row = db(f"select status,submit_reason,submitted_at,auto_score,breakdown from exam_attempts where id='{aid}'")
    if row.startswith('GRADED') or row.startswith('GRADING'): t_sub = now(); print('27 DB:', row[:300]); break
    time.sleep(1)
print('27 tự nộp sau deadline: %.1fs' % ((t_sub - dl).total_seconds()) if t_sub else '27 chưa tự nộp sau 60 s', '| mine:', json.dumps(call('GET', A + '/attempts/mine', SV)[1])[:300])
print('32 nộp lại (idempotent):', call('POST', A + f'/attempts/{aid}/submit', SV, {}, idem())[0:1])
print('09 POST attempts sau nộp:', call('POST', A + '/attempts', SV, {}, idem())[0:1], codes(call('POST', A + '/attempts', SV, {}, idem())[1]))
