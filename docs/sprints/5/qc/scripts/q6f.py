from q6lib import *
qid = json.load(open('/tmp/q3/q6.json'))['qid']
o = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=62); c = o + dt.timedelta(seconds=3000)
s, j = mke(title='QC6b', opens_at=o.strftime('%Y-%m-%dT%H:%M:%SZ'), closes_at=c.strftime('%Y-%m-%dT%H:%M:%SZ'), duration_minutes=5); eid = j['id']; A = E + '/' + eid
call('PUT', E + '/' + eid + '/items', T, dict(items=[dict(question_id=qid, points='10')], version=ge(eid)['version'])); call('POST', E + '/' + eid + '/schedule', T, {})
time.sleep(max(0, (o - dt.datetime.now(dt.timezone.utc)).total_seconds()) + 3)
tok = login('sv.nguyco'); j = call('POST', A + '/attempts', tok, {}, idem())[1]; aid = j['attempt']['id']; iid = j['items'][0]['item_id']; H = {'X-Exam-Tab': str(uuid.uuid4())}
dl = dt.datetime.strptime(j['attempt']['deadline_at'][:26], '%Y-%m-%dT%H:%M:%S.%f').replace(tzinfo=dt.timezone.utc); now = lambda: dt.datetime.now(dt.timezone.utc)
call('POST', A + f'/attempts/{aid}/takeover', tok, {}, H); rev = 0; last_ok = None; n = 0
while now() < dl + dt.timedelta(seconds=8.5):
    n += 1; src = SUMSRC + '// v%d' % n; r = call('PUT', A + f'/attempts/{aid}/code/{iid}/draft', tok, dict(language='cpp17', source=src, base_rev=rev), H)
    if r[0] == 200: rev = r[1]['rev']; last_ok = (src, (now() - dl).total_seconds())
    time.sleep(1.2)
print('39 gõ liên tục tới deadline+8,5 s: bản cuối nhận', last_ok[1], 'rev', rev)
while now() < dl + dt.timedelta(seconds=9.3): time.sleep(0.1)
r9 = call('PUT', A + f'/attempts/{aid}/code/{iid}/draft', tok, dict(language='cpp17', source=SUMSRC + '// v+9', base_rev=rev), H); t9 = (now() - dl).total_seconds()
if r9[0] == 200: rev = r9[1]['rev']; last_ok = (SUMSRC + '// v+9', t9)
while now() < dl + dt.timedelta(seconds=11): time.sleep(0.1)
r11 = call('PUT', A + f'/attempts/{aid}/code/{iid}/draft', tok, dict(language='cpp17', source=SUMSRC + '// v+11', base_rev=rev), H); t11 = (now() - dl).total_seconds()
print('40 deadline+%.1f → %s ; deadline+%.1f → %s %s' % (t9, r9[0], t11, r11[0], codes(r11[1])))
for _ in range(40):
    row = db(f"select status,submit_reason from exam_attempts where id='{aid}'")
    if not row.startswith('IN_PROGRESS'): break
    time.sleep(1)
print('39 lượt:', row, '| SUBMIT auto:', db(f"select kind,auto,status,source=$q${last_ok[0]}$q$ from code_submissions where attempt_id='{aid}' and kind='SUBMIT'"), '(cột cuối = nguồn đúng bản nháp cuối nhận được)', '| draft DB cuối:', db(f"select rev,right(source,6) from code_drafts where attempt_id='{aid}'"))
